package checker

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"tya/internal/ast"
)

var constNameRE = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
var valueNameRE = regexp.MustCompile(`^_?[a-z][a-z0-9]*(?:_[a-z0-9]+)*$|^_$`)
var classNameRE = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

func Check(prog *ast.Program) error {
	return CheckWithModules(prog, nil)
}

func CheckWithModules(prog *ast.Program, modules []string) error {
	constants := map[string]bool{}
	scope := newScope(nil)
	for _, name := range builtinNames {
		scope.define(name, kindUnknown)
	}
	for _, name := range modules {
		scope.define(name, kindModule)
	}
	return checkStmts(prog.Stmts, constants, scope)
}

func CheckModuleFile(prog *ast.Program, path string) error {
	want := strings.TrimSuffix(filepath.Base(path), ".tya")
	if !valueNameRE.MatchString(want) || strings.HasPrefix(want, "_") {
		return fmt.Errorf("invalid module file name %s", filepath.Base(path))
	}
	modules := []string{}
	for _, stmt := range prog.Stmts {
		switch n := stmt.(type) {
		case *ast.ImportStmt:
		case *ast.ModuleDecl:
			modules = append(modules, n.Name)
			if n.Name != want {
				return fmt.Errorf("%s must define module %s", filepath.Base(path), want)
			}
		default:
			return fmt.Errorf("%s may only contain imports and one module declaration", filepath.Base(path))
		}
	}
	if len(modules) != 1 {
		return fmt.Errorf("%s must define exactly one module", filepath.Base(path))
	}
	return Check(prog)
}

var builtinNames = []string{
	"all", "any", "args", "assert", "assert_equal", "contains", "delete",
	"ends_with", "env", "equal", "error", "exit", "file_exists", "filter",
	"find", "has", "join", "keys", "len", "map", "panic", "pop", "print",
	"push", "read_file", "read_line", "reduce", "replace", "split",
	"starts_with", "to_float", "to_int", "to_number", "to_string", "trim",
	"values", "write_file",
}

type scope struct {
	parent           *scope
	names            map[string]bool
	kinds            map[string]valueKind
	classes          map[string]classInfo
	inInstanceMethod bool
	inClassMethod    bool
	inClassBody      bool
	currentMethod    string
	currentClass     string
}

type classInfo struct {
	name      string
	parent    string
	hasInit   bool
	initArity int
	methods   map[string]int
}

func newScope(parent *scope) *scope {
	s := &scope{parent: parent, names: map[string]bool{}, kinds: map[string]valueKind{}}
	if parent != nil {
		s.classes = parent.classes
		s.inInstanceMethod = parent.inInstanceMethod
		s.inClassMethod = parent.inClassMethod
		s.inClassBody = parent.inClassBody
		s.currentMethod = parent.currentMethod
		s.currentClass = parent.currentClass
	} else {
		s.classes = map[string]classInfo{}
	}
	return s
}

type valueKind int

const (
	kindUnknown valueKind = iota
	kindArray
	kindDict
	kindModule
	kindClass
	kindObject
)

func (s *scope) define(name string, kind valueKind) {
	s.names[name] = true
	if s.kinds[name] == kindModule {
		return
	}
	s.kinds[name] = kind
}

func (s *scope) defined(name string) bool {
	if s.names[name] {
		return true
	}
	if s.parent != nil {
		return s.parent.defined(name)
	}
	return false
}

func (s *scope) kind(name string) valueKind {
	if kind, ok := s.kinds[name]; ok {
		return kind
	}
	if s.parent != nil {
		return s.parent.kind(name)
	}
	return kindUnknown
}

func checkStmts(stmts []ast.Stmt, constants map[string]bool, scope *scope) error {
	if err := predeclareFunctionBindings(stmts, scope); err != nil {
		return err
	}
	for _, stmt := range stmts {
		switch n := stmt.(type) {
		case *ast.ImportStmt:
			if !valueNameRE.MatchString(n.Name) || strings.HasPrefix(n.Name, "_") {
				return fmt.Errorf("%d:%d: invalid module name %s", n.NameTok.Line, n.NameTok.Col, n.Name)
			}
			scope.define(n.Name, kindModule)
		case *ast.AssignStmt:
			for _, value := range n.Values {
				if err := checkExpr(value, scope); err != nil {
					return err
				}
			}
			for _, target := range n.Targets {
				name, ok := target.(*ast.Ident)
				if !ok {
					if err := checkAssignmentTarget(target, scope); err != nil {
						return err
					}
					continue
				}
				if err := checkBindingName(name.Name, name.Tok.Line, name.Tok.Col); err != nil {
					return err
				}
				if constants[name.Name] {
					return fmt.Errorf("%d:%d: cannot reassign constant %s", n.Tok.Line, n.Tok.Col, name.Name)
				}
				if constNameRE.MatchString(name.Name) {
					constants[name.Name] = true
				}
				scope.define(name.Name, exprKind(n.Values, scope))
			}
		case *ast.ModuleDecl:
			if !valueNameRE.MatchString(n.Name) || strings.HasPrefix(n.Name, "_") {
				return fmt.Errorf("%d:%d: invalid module name %s", n.NameTok.Line, n.NameTok.Col, n.Name)
			}
			seen := map[string]bool{}
			scope.define(n.Name, kindModule)
			for _, class := range n.Classes {
				if !classNameRE.MatchString(class.Name) {
					return fmt.Errorf("%d:%d: invalid class name %s", class.NameTok.Line, class.NameTok.Col, class.Name)
				}
				if seen[class.Name] {
					return fmt.Errorf("%d:%d: duplicate module member %s", class.NameTok.Line, class.NameTok.Col, class.Name)
				}
				seen[class.Name] = true
				scope.define(n.Name+"."+class.Name, kindClass)
				if err := checkClass(class, scope, n.Name); err != nil {
					return err
				}
			}
			for _, member := range n.Members {
				if !valueNameRE.MatchString(member.Name) {
					return fmt.Errorf("%d:%d: invalid module member %s", member.Tok.Line, member.Tok.Col, member.Name)
				}
				if seen[member.Name] {
					return fmt.Errorf("%d:%d: duplicate module member %s", member.Tok.Line, member.Tok.Col, member.Name)
				}
				seen[member.Name] = true
				if err := checkExpr(member.Value, scope); err != nil {
					return err
				}
			}
		case *ast.ClassDecl:
			if err := checkClass(n, scope, ""); err != nil {
				return err
			}
		case *ast.IfStmt:
			if err := checkExpr(n.Cond, scope); err != nil {
				return err
			}
			if err := checkStmts(n.Then, constants, newScope(scope)); err != nil {
				return err
			}
			if err := checkStmts(n.Else, constants, newScope(scope)); err != nil {
				return err
			}
		case *ast.WhileStmt:
			if err := checkExpr(n.Cond, scope); err != nil {
				return err
			}
			if err := checkStmts(n.Body, constants, newScope(scope)); err != nil {
				return err
			}
		case *ast.ForInStmt:
			if err := checkBindingName(n.ValueName, n.ValueTok.Line, n.ValueTok.Col); err != nil {
				return err
			}
			if n.IndexName != "" {
				if err := checkBindingName(n.IndexName, n.IndexTok.Line, n.IndexTok.Col); err != nil {
					return err
				}
			}
			if err := checkExpr(n.Iterable, scope); err != nil {
				return err
			}
			child := newScope(scope)
			child.define(n.ValueName, kindUnknown)
			if n.IndexName != "" {
				child.define(n.IndexName, kindUnknown)
			}
			if err := checkStmts(n.Body, constants, child); err != nil {
				return err
			}
		case *ast.ExprStmt:
			if err := checkExpr(n.Expr, scope); err != nil {
				return err
			}
		case *ast.ReturnStmt:
			for _, value := range n.Values {
				if err := checkExpr(value, scope); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func predeclareFunctionBindings(stmts []ast.Stmt, scope *scope) error {
	for _, stmt := range stmts {
		switch n := stmt.(type) {
		case *ast.AssignStmt:
			if len(n.Targets) != 1 || len(n.Values) != 1 {
				continue
			}
			name, ok := n.Targets[0].(*ast.Ident)
			if !ok {
				continue
			}
			if _, ok := n.Values[0].(*ast.FuncLit); !ok {
				continue
			}
			if err := checkBindingName(name.Name, name.Tok.Line, name.Tok.Col); err != nil {
				return err
			}
			scope.define(name.Name, kindUnknown)
		case *ast.ClassDecl:
			if err := predeclareClass(n, scope); err != nil {
				return err
			}
		case *ast.ModuleDecl:
			for _, class := range n.Classes {
				if err := predeclareModuleClass(n.Name, class, scope); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func predeclareModuleClass(module string, class *ast.ClassDecl, scope *scope) error {
	if !classNameRE.MatchString(class.Name) {
		return fmt.Errorf("%d:%d: invalid class name %s", class.NameTok.Line, class.NameTok.Col, class.Name)
	}
	key := classKey(module, class)
	info := classInfo{name: key, methods: map[string]int{}}
	if class.Parent != nil {
		info.parent = refKey(class.Parent, module, scope)
	}
	for _, method := range class.Methods {
		if method.Class {
			continue
		}
		info.methods[method.Name] = len(method.Func.Params)
		if method.Name == "init" {
			info.hasInit = true
			info.initArity = len(method.Func.Params)
		}
	}
	scope.define(module+"."+class.Name, kindClass)
	scope.classes[key] = info
	return nil
}

func predeclareClass(class *ast.ClassDecl, scope *scope) error {
	if !classNameRE.MatchString(class.Name) {
		return fmt.Errorf("%d:%d: invalid class name %s", class.NameTok.Line, class.NameTok.Col, class.Name)
	}
	key := classKey("", class)
	info := classInfo{name: key, methods: map[string]int{}}
	if class.Parent != nil {
		info.parent = refKey(class.Parent, "", scope)
	}
	for _, method := range class.Methods {
		if method.Class {
			continue
		}
		info.methods[method.Name] = len(method.Func.Params)
		if method.Name == "init" {
			info.hasInit = true
			info.initArity = len(method.Func.Params)
		}
	}
	scope.define(class.Name, kindClass)
	scope.classes[key] = info
	return nil
}

func checkExpr(expr ast.Expr, scope *scope) error {
	switch n := expr.(type) {
	case *ast.Ident:
		if !scope.defined(n.Name) {
			return fmt.Errorf("%d:%d: undefined variable %s", n.Tok.Line, n.Tok.Col, n.Name)
		}
	case *ast.DictLit:
		seen := map[string]bool{}
		for _, prop := range n.Props {
			if !valueNameRE.MatchString(prop.Name) {
				return fmt.Errorf("%d:%d: invalid dictionary key %s", prop.Tok.Line, prop.Tok.Col, prop.Name)
			}
			if seen[prop.Name] {
				return fmt.Errorf("%d:%d: duplicate dictionary key %s", prop.Tok.Line, prop.Tok.Col, prop.Name)
			}
			seen[prop.Name] = true
			if err := checkExpr(prop.Value, scope); err != nil {
				return err
			}
		}
	case *ast.FuncLit:
		seen := map[string]bool{}
		child := newScope(scope)
		for i, param := range n.Params {
			line := 0
			col := 0
			if i < len(n.ParamToks) {
				line = n.ParamToks[i].Line
				col = n.ParamToks[i].Col
			}
			if err := checkBindingName(param, line, col); err != nil {
				return err
			}
			if seen[param] && param != "_" {
				if line > 0 {
					return fmt.Errorf("%d:%d: duplicate function parameter %s", line, col, param)
				}
				return fmt.Errorf("duplicate function parameter %s", param)
			}
			seen[param] = true
			child.define(param, kindUnknown)
		}
		if n.Expr != nil {
			if err := checkExpr(n.Expr, child); err != nil {
				return err
			}
		}
		if err := checkStmts(n.Body, map[string]bool{}, child); err != nil {
			return err
		}
	case *ast.ArrayLit:
		for _, elem := range n.Elems {
			if err := checkExpr(elem, scope); err != nil {
				return err
			}
		}
	case *ast.BinaryExpr:
		if err := checkExpr(n.Left, scope); err != nil {
			return err
		}
		return checkExpr(n.Right, scope)
	case *ast.UnaryExpr:
		return checkExpr(n.Expr, scope)
	case *ast.TryExpr:
		return checkExpr(n.Expr, scope)
	case *ast.SuperExpr:
		return fmt.Errorf("%d:%d: super must be called inside init or an instance method", n.Tok.Line, n.Tok.Col)
	case *ast.InstanceFieldExpr:
		if !scope.inInstanceMethod {
			return fmt.Errorf("%d:%d: @%s is only valid inside an instance method", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
		if !valueNameRE.MatchString(n.Name) {
			return fmt.Errorf("%d:%d: invalid field name %s", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
	case *ast.ClassVarExpr:
		if !scope.inClassBody && !scope.inInstanceMethod && !scope.inClassMethod {
			return fmt.Errorf("%d:%d: @@%s is only valid inside a class", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
		if !valueNameRE.MatchString(n.Name) {
			return fmt.Errorf("%d:%d: invalid class variable name %s", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
	case *ast.MemberExpr:
		if err := checkExpr(n.Target, scope); err != nil {
			return err
		}
		switch kindOf(n.Target, scope) {
		case kindModule:
			return nil
		case kindClass:
			return nil
		case kindObject:
			return nil
		case kindDict:
			return memberAccessError(n, "dictionary")
		case kindArray:
			return memberAccessError(n, "array")
		default:
			return nil
		}
	case *ast.IndexExpr:
		if err := checkExpr(n.Target, scope); err != nil {
			return err
		}
		return checkExpr(n.Index, scope)
	case *ast.CallExpr:
		if super, ok := n.Callee.(*ast.SuperExpr); ok {
			if scope.inClassMethod || !scope.inInstanceMethod || scope.currentMethod == "" || scope.currentClass == "" {
				return fmt.Errorf("%d:%d: super must be called inside init or an instance method", super.Tok.Line, super.Tok.Col)
			}
			parent := scope.classes[scope.currentClass].parent
			if parent == "" {
				return fmt.Errorf("%d:%d: super has no parent method", super.Tok.Line, super.Tok.Col)
			}
			if scope.currentMethod == "init" {
				arity, ok := inheritedInitArity(parent, scope)
				if !ok {
					return fmt.Errorf("%d:%d: super has no parent method", super.Tok.Line, super.Tok.Col)
				}
				if len(n.Args) != arity {
					return fmt.Errorf("%d:%d: super init expects %d arguments", super.Tok.Line, super.Tok.Col, arity)
				}
			} else {
				arity, ok := inheritedMethodArity(parent, scope.currentMethod, scope)
				if !ok {
					return fmt.Errorf("%d:%d: super has no parent method %s", super.Tok.Line, super.Tok.Col, scope.currentMethod)
				}
				if len(n.Args) != arity {
					return fmt.Errorf("%d:%d: super method %s expects %d arguments", super.Tok.Line, super.Tok.Col, scope.currentMethod, arity)
				}
			}
			for _, arg := range n.Args {
				if err := checkExpr(arg, scope); err != nil {
					return err
				}
			}
			return nil
		}
		if err := checkExpr(n.Callee, scope); err != nil {
			return err
		}
		if id, ok := n.Callee.(*ast.Ident); ok && scope.kind(id.Name) == kindClass {
			info := scope.classes[id.Name]
			hasInit, arity := effectiveInit(info, scope)
			if !hasInit && len(n.Args) > 0 {
				return fmt.Errorf("%d:%d: class %s has no init and takes no arguments", id.Tok.Line, id.Tok.Col, id.Name)
			}
			if hasInit && len(n.Args) != arity {
				return fmt.Errorf("%d:%d: class %s constructor expects %d arguments", id.Tok.Line, id.Tok.Col, id.Name, arity)
			}
		}
		for _, arg := range n.Args {
			if err := checkExpr(arg, scope); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkClass(class *ast.ClassDecl, scope *scope, module string) error {
	if !classNameRE.MatchString(class.Name) {
		return fmt.Errorf("%d:%d: invalid class name %s", class.NameTok.Line, class.NameTok.Col, class.Name)
	}
	key := classKey(module, class)
	if class.Parent != nil {
		parentKey := refKey(class.Parent, module, scope)
		if _, ok := scope.classes[parentKey]; !ok {
			return fmt.Errorf("%d:%d: unknown parent class %s", class.Parent.Tok.Line, class.Parent.Tok.Col, parentName(class.Parent))
		}
		if hasInheritanceCycle(key, scope) {
			return fmt.Errorf("%d:%d: inheritance cycle involving %s", class.NameTok.Line, class.NameTok.Col, class.Name)
		}
	}
	instanceMembers := map[string]bool{}
	classMembers := map[string]bool{}
	classBody := newScope(scope)
	classBody.inClassBody = true
	for _, field := range class.Fields {
		if !valueNameRE.MatchString(field.Name) {
			return fmt.Errorf("%d:%d: invalid field name %s", field.Tok.Line, field.Tok.Col, field.Name)
		}
		if instanceMembers[field.Name] {
			return fmt.Errorf("%d:%d: duplicate instance member %s", field.Tok.Line, field.Tok.Col, field.Name)
		}
		instanceMembers[field.Name] = true
		if err := checkExpr(field.Value, classBody); err != nil {
			return err
		}
	}
	for _, variable := range class.Vars {
		if !valueNameRE.MatchString(variable.Name) {
			return fmt.Errorf("%d:%d: invalid class variable name %s", variable.Tok.Line, variable.Tok.Col, variable.Name)
		}
		if classMembers[variable.Name] {
			return fmt.Errorf("%d:%d: duplicate class member %s", variable.Tok.Line, variable.Tok.Col, variable.Name)
		}
		classMembers[variable.Name] = true
		if err := checkExpr(variable.Value, classBody); err != nil {
			return err
		}
	}
	for _, method := range class.Methods {
		if !valueNameRE.MatchString(method.Name) {
			return fmt.Errorf("%d:%d: invalid method name %s", method.Tok.Line, method.Tok.Col, method.Name)
		}
		if method.Class {
			if classMembers[method.Name] {
				return fmt.Errorf("%d:%d: duplicate class member %s", method.Tok.Line, method.Tok.Col, method.Name)
			}
			classMembers[method.Name] = true
		} else {
			if instanceMembers[method.Name] {
				return fmt.Errorf("%d:%d: duplicate instance member %s", method.Tok.Line, method.Tok.Col, method.Name)
			}
			instanceMembers[method.Name] = true
		}
		child := newScope(scope)
		child.inClassBody = true
		child.currentClass = key
		child.currentMethod = method.Name
		if method.Class {
			child.inClassMethod = true
		} else {
			child.inInstanceMethod = true
		}
		if err := checkExpr(method.Func, child); err != nil {
			return err
		}
		if !method.Class {
			parent := scope.classes[key].parent
			if parent != "" {
				if method.Name == "init" {
					if _, ok := inheritedInitArity(parent, scope); ok && !funcCallsSuper(method.Func) {
						return fmt.Errorf("%d:%d: subclass init must call super", method.Tok.Line, method.Tok.Col)
					}
				} else if arity, ok := inheritedMethodArity(parent, method.Name, scope); ok && arity != len(method.Func.Params) {
					return fmt.Errorf("%d:%d: overriding method %s expects %d parameters", method.Tok.Line, method.Tok.Col, method.Name, arity)
				}
			}
		}
	}
	return nil
}

func classKey(module string, class *ast.ClassDecl) string {
	if module != "" {
		return module + "." + class.Name
	}
	return class.Name
}

func refKey(ref *ast.ClassRef, currentModule string, scope *scope) string {
	if ref.Module != "" {
		return ref.Module + "." + ref.Name
	}
	if currentModule != "" {
		if _, ok := scope.classes[currentModule+"."+ref.Name]; ok {
			return currentModule + "." + ref.Name
		}
	}
	return ref.Name
}

func parentName(ref *ast.ClassRef) string {
	if ref.Module != "" {
		return ref.Module + "." + ref.Name
	}
	return ref.Name
}

func hasInheritanceCycle(start string, scope *scope) bool {
	seen := map[string]bool{}
	current := start
	for current != "" {
		if seen[current] {
			return true
		}
		seen[current] = true
		current = scope.classes[current].parent
	}
	return false
}

func effectiveInit(info classInfo, scope *scope) (bool, int) {
	if info.hasInit {
		return true, info.initArity
	}
	if info.parent != "" {
		arity, ok := inheritedInitArity(info.parent, scope)
		return ok, arity
	}
	return false, 0
}

func inheritedInitArity(className string, scope *scope) (int, bool) {
	for className != "" {
		info, ok := scope.classes[className]
		if !ok {
			return 0, false
		}
		if info.hasInit {
			return info.initArity, true
		}
		className = info.parent
	}
	return 0, false
}

func inheritedMethodArity(className string, method string, scope *scope) (int, bool) {
	for className != "" {
		info, ok := scope.classes[className]
		if !ok {
			return 0, false
		}
		if arity, ok := info.methods[method]; ok {
			return arity, true
		}
		className = info.parent
	}
	return 0, false
}

func funcCallsSuper(fn *ast.FuncLit) bool {
	var exprHasSuper func(ast.Expr) bool
	exprHasSuper = func(expr ast.Expr) bool {
		switch n := expr.(type) {
		case *ast.CallExpr:
			if _, ok := n.Callee.(*ast.SuperExpr); ok {
				return true
			}
			if exprHasSuper(n.Callee) {
				return true
			}
			for _, arg := range n.Args {
				if exprHasSuper(arg) {
					return true
				}
			}
		case *ast.BinaryExpr:
			return exprHasSuper(n.Left) || exprHasSuper(n.Right)
		case *ast.UnaryExpr:
			return exprHasSuper(n.Expr)
		case *ast.TryExpr:
			return exprHasSuper(n.Expr)
		case *ast.MemberExpr:
			return exprHasSuper(n.Target)
		case *ast.IndexExpr:
			return exprHasSuper(n.Target) || exprHasSuper(n.Index)
		case *ast.ArrayLit:
			for _, elem := range n.Elems {
				if exprHasSuper(elem) {
					return true
				}
			}
		case *ast.DictLit:
			for _, prop := range n.Props {
				if exprHasSuper(prop.Value) {
					return true
				}
			}
		}
		return false
	}
	if fn.Expr != nil && exprHasSuper(fn.Expr) {
		return true
	}
	for _, stmt := range fn.Body {
		switch n := stmt.(type) {
		case *ast.ExprStmt:
			if exprHasSuper(n.Expr) {
				return true
			}
		case *ast.AssignStmt:
			for _, value := range n.Values {
				if exprHasSuper(value) {
					return true
				}
			}
		case *ast.ReturnStmt:
			for _, value := range n.Values {
				if exprHasSuper(value) {
					return true
				}
			}
		}
	}
	return false
}

func checkAssignmentTarget(target ast.Expr, scope *scope) error {
	switch n := target.(type) {
	case *ast.MemberExpr:
		if err := checkExpr(n.Target, scope); err != nil {
			return err
		}
		switch kindOf(n.Target, scope) {
		case kindDict:
			return memberAccessError(n, "dictionary")
		case kindArray:
			return memberAccessError(n, "array")
		}
		return nil
	case *ast.InstanceFieldExpr:
		if !scope.inInstanceMethod {
			return fmt.Errorf("%d:%d: @%s is only valid inside an instance method", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
		if !valueNameRE.MatchString(n.Name) {
			return fmt.Errorf("%d:%d: invalid field name %s", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
	case *ast.ClassVarExpr:
		if !scope.inClassBody && !scope.inInstanceMethod && !scope.inClassMethod {
			return fmt.Errorf("%d:%d: @@%s is only valid inside a class", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
		if !valueNameRE.MatchString(n.Name) {
			return fmt.Errorf("%d:%d: invalid class variable name %s", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
	case *ast.IndexExpr:
		if err := checkExpr(n.Target, scope); err != nil {
			return err
		}
		return checkExpr(n.Index, scope)
	}
	return nil
}

func exprKind(values []ast.Expr, scope *scope) valueKind {
	if len(values) != 1 {
		return kindUnknown
	}
	if call, ok := values[0].(*ast.CallExpr); ok {
		if id, ok := call.Callee.(*ast.Ident); ok && scope.kind(id.Name) == kindClass {
			return kindObject
		}
		if member, ok := call.Callee.(*ast.MemberExpr); ok && kindOf(member.Target, scope) == kindModule && classNameRE.MatchString(member.Name) {
			return kindObject
		}
	}
	return literalKind(values[0])
}

func kindOf(expr ast.Expr, scope *scope) valueKind {
	if id, ok := expr.(*ast.Ident); ok {
		return scope.kind(id.Name)
	}
	return literalKind(expr)
}

func literalKind(expr ast.Expr) valueKind {
	switch n := expr.(type) {
	case *ast.ArrayLit:
		return kindArray
	case *ast.DictLit:
		if hasFunctionMember(n) {
			return kindUnknown
		}
		return kindDict
	}
	return kindUnknown
}

func hasFunctionMember(dict *ast.DictLit) bool {
	for _, prop := range dict.Props {
		if _, ok := prop.Value.(*ast.FuncLit); ok {
			return true
		}
	}
	return false
}

func memberAccessError(expr *ast.MemberExpr, receiver string) error {
	line := expr.NameTok.Line
	col := expr.NameTok.Col
	if receiver == "dictionary" {
		if line > 0 {
			return fmt.Errorf("%d:%d: cannot use . access on dictionary; use index access", line, col)
		}
		return fmt.Errorf("cannot use . access on dictionary; use index access")
	}
	if receiver == "non-module value" {
		if line > 0 {
			return fmt.Errorf("%d:%d: cannot use . access on non-module value", line, col)
		}
		return fmt.Errorf("cannot use . access on non-module value")
	}
	if line > 0 {
		return fmt.Errorf("%d:%d: cannot use . access on %s", line, col, receiver)
	}
	return fmt.Errorf("cannot use . access on %s", receiver)
}

func checkBindingName(name string, line, col int) error {
	if constNameRE.MatchString(name) || valueNameRE.MatchString(name) {
		return nil
	}
	if line > 0 {
		return fmt.Errorf("%d:%d: invalid binding name %s", line, col, name)
	}
	return fmt.Errorf("invalid binding name %s", name)
}
