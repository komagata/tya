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
	"all", "any", "args", "contains", "delete", "ends_with", "env", "equal",
	"error", "exit", "file_exists", "filter", "find", "has", "join", "keys",
	"len", "map", "panic", "pop", "print", "push", "read_file", "read_line",
	"reduce", "replace", "split", "starts_with", "to_float", "to_int",
	"to_number", "to_string", "trim", "values", "write_file",
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
	kindModule
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
				scope.define(name.Name, exprKind(n.Values))
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
				if err := checkExpr(member.Value, scope); err != nil {
					return err
				}
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
		assign, ok := stmt.(*ast.AssignStmt)
		if !ok || len(assign.Targets) != 1 || len(assign.Values) != 1 {
			continue
		}
		name, ok := assign.Targets[0].(*ast.Ident)
		if !ok {
			continue
		}
		if _, ok := assign.Values[0].(*ast.FuncLit); !ok {
			continue
		}
		if err := checkBindingName(name.Name, name.Tok.Line, name.Tok.Col); err != nil {
			return err
		}
		scope.define(name.Name, kindUnknown)
	}
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
	case *ast.MemberExpr:
		if err := checkExpr(n.Target, scope); err != nil {
			return err
		}
		switch kindOf(n.Target, scope) {
		case kindModule:
			return nil
		case kindDict:
			return memberAccessError(n, "dictionary")
		case kindArray:
			return memberAccessError(n, "array")
		default:
			return memberAccessError(n, "non-module value")
		}
	case *ast.IndexExpr:
		if err := checkExpr(n.Target, scope); err != nil {
			return err
		}
		return checkExpr(n.Index, scope)
	case *ast.CallExpr:
		if err := checkExpr(n.Callee, scope); err != nil {
			return err
		}
		for _, arg := range n.Args {
			if err := checkExpr(arg, scope); err != nil {
				return err
			}
		}
	}
	return nil
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
	case *ast.IndexExpr:
		if err := checkExpr(n.Target, scope); err != nil {
			return err
		}
		return checkExpr(n.Index, scope)
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
