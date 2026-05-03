package checker

import (
	"fmt"
	"regexp"
	"strings"

	"tya/internal/ast"
)

var constNameRE = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
var valueNameRE = regexp.MustCompile(`^_?[a-z][a-z0-9]*(?:_[a-z0-9]+)*$|^_$`)
var classNameRE = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)

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
	info := collectTypeInfo(prog)
	return checkStmts(prog.Stmts, constants, scope, info, checkContext{})
}

type methodSig struct{ arity int }
type classInfo struct {
	parent  string
	methods map[string]methodSig
}
type interfaceInfo struct{ methods map[string]methodSig }
type typeInfo struct {
	classes    map[string]classInfo
	interfaces map[string]interfaceInfo
}
type checkContext struct {
	className  string
	methodName string
	hasParent  bool
}

func collectTypeInfo(prog *ast.Program) typeInfo {
	info := typeInfo{classes: map[string]classInfo{}, interfaces: map[string]interfaceInfo{}}
	for _, stmt := range prog.Stmts {
		switch n := stmt.(type) {
		case *ast.ClassDecl:
			methods := map[string]methodSig{}
			for _, method := range n.Methods {
				methods[method.Name] = methodSig{arity: len(method.Func.Params)}
			}
			info.classes[n.Name] = classInfo{parent: n.Parent, methods: methods}
		case *ast.InterfaceDecl:
			methods := map[string]methodSig{}
			for _, method := range n.Methods {
				methods[method.Name] = methodSig{arity: len(method.ParamToks)}
			}
			info.interfaces[n.Name] = interfaceInfo{methods: methods}
		}
	}
	return info
}

var builtinNames = []string{
	"args", "byte_len", "byteLen", "char_len", "charLen", "contains", "delete", "div", "ends_with", "endsWith",
	"env", "equal", "error", "exit", "file_exists", "fileExists", "filter", "find", "all", "any", "each",
	"has", "join", "keys", "len", "map", "panic", "pop", "print", "push",
	"read_file", "read_line", "readFile", "readLine", "reduce", "replace", "set", "split", "starts_with", "startsWith",
	"to_float", "to_int", "to_number", "to_string", "toFloat", "toInt", "toNumber", "toString", "trim",
	"values", "write_file", "writeFile",
}

type scope struct {
	parent *scope
	names  map[string]bool
	kinds  map[string]valueKind
}

func newScope(parent *scope) *scope {
	return &scope{parent: parent, names: map[string]bool{}, kinds: map[string]valueKind{}}
}

type valueKind int

const (
	kindUnknown valueKind = iota
	kindArray
	kindDict
	kindClass
	kindModule
	kindSet
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

func checkStmts(stmts []ast.Stmt, constants map[string]bool, scope *scope, info typeInfo, ctx checkContext) error {
	for _, stmt := range stmts {
		switch n := stmt.(type) {
		case *ast.AssignStmt:
			for _, value := range n.Values {
				if err := checkExpr(value, scope, info, ctx); err != nil {
					return err
				}
			}
			for _, target := range n.Targets {
				name, ok := target.(*ast.Ident)
				if !ok {
					if err := checkAssignmentTarget(target, scope, info, ctx); err != nil {
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
				scope.define(name.Name, exprKind(n.Values))
			}
		case *ast.ClassDecl:
			if !classNameRE.MatchString(n.Name) {
				return fmt.Errorf("%d:%d: invalid class name %s", n.NameTok.Line, n.NameTok.Col, n.Name)
			}
			if n.Parent != "" {
				if !classNameRE.MatchString(n.Parent) {
					return fmt.Errorf("%d:%d: invalid parent class name %s", n.ParentTok.Line, n.ParentTok.Col, n.Parent)
				}
				if _, ok := info.classes[n.Parent]; !ok {
					return fmt.Errorf("%d:%d: undefined parent class %s", n.ParentTok.Line, n.ParentTok.Col, n.Parent)
				}
			}
			seen := map[string]bool{}
			scope.define(n.Name, kindClass)
			for _, method := range n.Methods {
				if !valueNameRE.MatchString(method.Name) {
					return fmt.Errorf("%d:%d: invalid method name %s", method.Tok.Line, method.Tok.Col, method.Name)
				}
				if seen[method.Name] {
					return fmt.Errorf("%d:%d: duplicate method %s", method.Tok.Line, method.Tok.Col, method.Name)
				}
				seen[method.Name] = true
				if parent := info.classes[n.Parent]; n.Parent != "" && method.Name != "init" {
					if inherited, ok := parent.methods[method.Name]; ok && inherited.arity != len(method.Func.Params) {
						return fmt.Errorf("%d:%d: override arity mismatch for %s", method.Tok.Line, method.Tok.Col, method.Name)
					}
				}
				if err := checkExpr(method.Func, scope, info, checkContext{className: n.Name, methodName: method.Name, hasParent: n.Parent != ""}); err != nil {
					return err
				}
			}
			for i, name := range n.Implements {
				tok := n.ImplToks[i]
				methods, ok := implementedMethods(name, info)
				if !ok {
					return fmt.Errorf("%d:%d: undefined interface %s", tok.Line, tok.Col, name)
				}
				for methodName, sig := range methods {
					own, ok := info.classes[n.Name].methods[methodName]
					if !ok {
						return fmt.Errorf("%d:%d: class %s does not implement %s.%s", tok.Line, tok.Col, n.Name, name, methodName)
					}
					if own.arity != sig.arity {
						return fmt.Errorf("%d:%d: class %s has wrong arity for %s.%s", tok.Line, tok.Col, n.Name, name, methodName)
					}
				}
			}
		case *ast.ModuleDecl:
			if !valueNameRE.MatchString(n.Name) || strings.HasPrefix(n.Name, "_") {
				return fmt.Errorf("%d:%d: invalid module name %s", n.NameTok.Line, n.NameTok.Col, n.Name)
			}
			seen := map[string]bool{}
			scope.define(n.Name, kindModule)
			for _, member := range n.Members {
				if !valueNameRE.MatchString(member.Name) {
					return fmt.Errorf("%d:%d: invalid module member %s", member.Tok.Line, member.Tok.Col, member.Name)
				}
				if seen[member.Name] {
					return fmt.Errorf("%d:%d: duplicate module member %s", member.Tok.Line, member.Tok.Col, member.Name)
				}
				seen[member.Name] = true
				if err := checkExpr(member.Value, scope, info, ctx); err != nil {
					return err
				}
			}
		case *ast.InterfaceDecl:
			if !classNameRE.MatchString(n.Name) {
				return fmt.Errorf("%d:%d: invalid interface name %s", n.NameTok.Line, n.NameTok.Col, n.Name)
			}
			seen := map[string]bool{}
			scope.define(n.Name, kindUnknown)
			for _, method := range n.Methods {
				if !valueNameRE.MatchString(method.Name) {
					return fmt.Errorf("%d:%d: invalid interface method %s", method.Tok.Line, method.Tok.Col, method.Name)
				}
				if seen[method.Name] {
					return fmt.Errorf("%d:%d: duplicate interface method %s", method.Tok.Line, method.Tok.Col, method.Name)
				}
				seen[method.Name] = true
			}
		case *ast.IfStmt:
			if err := checkExpr(n.Cond, scope, info, ctx); err != nil {
				return err
			}
			if err := checkStmts(n.Then, constants, newScope(scope), info, ctx); err != nil {
				return err
			}
			if err := checkStmts(n.Else, constants, newScope(scope), info, ctx); err != nil {
				return err
			}
		case *ast.WhileStmt:
			if err := checkExpr(n.Cond, scope, info, ctx); err != nil {
				return err
			}
			if err := checkStmts(n.Body, constants, newScope(scope), info, ctx); err != nil {
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
			if err := checkExpr(n.Iterable, scope, info, ctx); err != nil {
				return err
			}
			child := newScope(scope)
			child.define(n.ValueName, kindUnknown)
			if n.IndexName != "" {
				child.define(n.IndexName, kindUnknown)
			}
			if err := checkStmts(n.Body, constants, child, info, ctx); err != nil {
				return err
			}
		case *ast.ExprStmt:
			if err := checkExpr(n.Expr, scope, info, ctx); err != nil {
				return err
			}
		case *ast.ReturnStmt:
			for _, value := range n.Values {
				if err := checkExpr(value, scope, info, ctx); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func implementedMethods(name string, info typeInfo) (map[string]methodSig, bool) {
	if iface, ok := info.interfaces[name]; ok {
		return iface.methods, true
	}
	if class, ok := info.classes[name]; ok {
		return class.methods, true
	}
	return nil, false
}

func superArity(info typeInfo, ctx checkContext) int {
	class := info.classes[ctx.className]
	parent := info.classes[class.parent]
	if sig, ok := parent.methods[ctx.methodName]; ok {
		return sig.arity
	}
	return 0
}

func checkExpr(expr ast.Expr, scope *scope, info typeInfo, ctx checkContext) error {
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
			if err := checkExpr(prop.Value, scope, info, ctx); err != nil {
				return err
			}
		}
	case *ast.SetLit:
		for _, elem := range n.Elems {
			if err := checkExpr(elem, scope, info, ctx); err != nil {
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
			if err := checkExpr(n.Expr, child, info, ctx); err != nil {
				return err
			}
		}
		if err := checkStmts(n.Body, map[string]bool{}, child, info, ctx); err != nil {
			return err
		}
	case *ast.ArrayLit:
		for _, elem := range n.Elems {
			if err := checkExpr(elem, scope, info, ctx); err != nil {
				return err
			}
		}
	case *ast.BinaryExpr:
		if err := checkExpr(n.Left, scope, info, ctx); err != nil {
			return err
		}
		return checkExpr(n.Right, scope, info, ctx)
	case *ast.UnaryExpr:
		return checkExpr(n.Expr, scope, info, ctx)
	case *ast.TryExpr:
		return checkExpr(n.Expr, scope, info, ctx)
	case *ast.SuperExpr:
		if ctx.className == "" || !ctx.hasParent {
			return fmt.Errorf("%d:%d: super used outside subclass method", n.Tok.Line, n.Tok.Col)
		}
	case *ast.MemberExpr:
		if err := checkExpr(n.Object, scope, info, ctx); err != nil {
			return err
		}
		switch kindOf(n.Object, scope) {
		case kindDict:
			return memberAccessError(n, "dictionary")
		case kindSet:
			return memberAccessError(n, "set")
		case kindArray:
			return memberAccessError(n, "array")
		}
	case *ast.IndexExpr:
		if err := checkExpr(n.Object, scope, info, ctx); err != nil {
			return err
		}
		return checkExpr(n.Index, scope, info, ctx)
	case *ast.CallExpr:
		if err := checkExpr(n.Callee, scope, info, ctx); err != nil {
			return err
		}
		if _, ok := n.Callee.(*ast.SuperExpr); ok && len(n.Args) != superArity(info, ctx) {
			return fmt.Errorf("super expects %d arguments, got %d", superArity(info, ctx), len(n.Args))
		}
		for _, arg := range n.Args {
			if err := checkExpr(arg, scope, info, ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkAssignmentTarget(target ast.Expr, scope *scope, info typeInfo, ctx checkContext) error {
	switch n := target.(type) {
	case *ast.MemberExpr:
		if err := checkExpr(n.Object, scope, info, ctx); err != nil {
			return err
		}
		switch kindOf(n.Object, scope) {
		case kindDict:
			return memberAccessError(n, "dictionary")
		case kindSet:
			return memberAccessError(n, "set")
		case kindArray:
			return memberAccessError(n, "array")
		}
		return nil
	case *ast.IndexExpr:
		if err := checkExpr(n.Object, scope, info, ctx); err != nil {
			return err
		}
		return checkExpr(n.Index, scope, info, ctx)
	case *ast.ThisProp:
		return nil
	}
	return nil
}

func exprKind(values []ast.Expr) valueKind {
	if len(values) != 1 {
		return kindUnknown
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
	case *ast.SetLit:
		return kindSet
	case *ast.CallExpr:
		if id, ok := n.Callee.(*ast.Ident); ok && id.Name == "set" && len(n.Args) == 0 {
			return kindSet
		}
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
