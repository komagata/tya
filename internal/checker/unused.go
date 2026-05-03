package checker

import (
	"fmt"

	"tya/internal/ast"
)

type bindingUse struct {
	name string
	line int
	col  int
	used bool
}

type useScope struct {
	parent   *useScope
	bindings map[string]*bindingUse
	order    []*bindingUse
	children []*useScope
}

func CheckUnused(prog *ast.Program) error {
	scope := newUseScope(nil)
	if err := checkUnusedStmts(prog.Stmts, scope); err != nil {
		return err
	}
	return firstUnused(scope)
}

func newUseScope(parent *useScope) *useScope {
	scope := &useScope{parent: parent, bindings: map[string]*bindingUse{}}
	if parent != nil {
		parent.children = append(parent.children, scope)
	}
	return scope
}

func (s *useScope) define(name string, line int, col int) {
	if name == "_" || s.bindings[name] != nil {
		return
	}
	binding := &bindingUse{name: name, line: line, col: col}
	s.bindings[name] = binding
	s.order = append(s.order, binding)
}

func (s *useScope) use(name string) {
	for scope := s; scope != nil; scope = scope.parent {
		if binding := scope.bindings[name]; binding != nil {
			binding.used = true
			return
		}
	}
}

func checkUnusedStmts(stmts []ast.Stmt, scope *useScope) error {
	for _, stmt := range stmts {
		switch n := stmt.(type) {
		case *ast.AssignStmt:
			for _, value := range n.Values {
				checkUnusedExpr(value, scope)
			}
			for _, target := range n.Targets {
				if id, ok := target.(*ast.Ident); ok {
					scope.define(id.Name, id.Tok.Line, id.Tok.Col)
					continue
				}
				checkUnusedAssignmentTarget(target, scope)
			}
		case *ast.IfStmt:
			checkUnusedExpr(n.Cond, scope)
			if err := checkUnusedStmts(n.Then, newUseScope(scope)); err != nil {
				return err
			}
			if err := checkUnusedStmts(n.Else, newUseScope(scope)); err != nil {
				return err
			}
		case *ast.WhileStmt:
			checkUnusedExpr(n.Cond, scope)
			if err := checkUnusedStmts(n.Body, newUseScope(scope)); err != nil {
				return err
			}
		case *ast.ForInStmt:
			checkUnusedExpr(n.Iterable, scope)
			child := newUseScope(scope)
			child.define(n.ValueName, 0, 0)
			if n.IndexName != "" {
				child.define(n.IndexName, 0, 0)
			}
			if err := checkUnusedStmts(n.Body, child); err != nil {
				return err
			}
		case *ast.ExprStmt:
			checkUnusedExpr(n.Expr, scope)
		case *ast.ReturnStmt:
			for _, value := range n.Values {
				checkUnusedExpr(value, scope)
			}
		}
	}
	return nil
}

func checkUnusedExpr(expr ast.Expr, scope *useScope) {
	switch n := expr.(type) {
	case *ast.Ident:
		scope.use(n.Name)
	case *ast.DictLit:
		for _, prop := range n.Props {
			checkUnusedExpr(prop.Value, scope)
		}
	case *ast.SetLit:
		for _, elem := range n.Elems {
			checkUnusedExpr(elem, scope)
		}
	case *ast.FuncLit:
		child := newUseScope(scope)
		for _, param := range n.Params {
			child.define(param, 0, 0)
		}
		if n.Expr != nil {
			checkUnusedExpr(n.Expr, child)
		}
		_ = checkUnusedStmts(n.Body, child)
	case *ast.ArrayLit:
		for _, elem := range n.Elems {
			checkUnusedExpr(elem, scope)
		}
	case *ast.BinaryExpr:
		checkUnusedExpr(n.Left, scope)
		checkUnusedExpr(n.Right, scope)
	case *ast.UnaryExpr:
		checkUnusedExpr(n.Expr, scope)
	case *ast.TryExpr:
		checkUnusedExpr(n.Expr, scope)
	case *ast.MemberExpr:
		checkUnusedExpr(n.Object, scope)
	case *ast.IndexExpr:
		checkUnusedExpr(n.Object, scope)
		checkUnusedExpr(n.Index, scope)
	case *ast.CallExpr:
		checkUnusedExpr(n.Callee, scope)
		for _, arg := range n.Args {
			checkUnusedExpr(arg, scope)
		}
	}
}

func checkUnusedAssignmentTarget(target ast.Expr, scope *useScope) {
	switch n := target.(type) {
	case *ast.MemberExpr:
		checkUnusedExpr(n.Object, scope)
	case *ast.IndexExpr:
		checkUnusedExpr(n.Object, scope)
		checkUnusedExpr(n.Index, scope)
	}
}

func firstUnused(scope *useScope) error {
	for _, binding := range scope.order {
		if !binding.used {
			if binding.line > 0 {
				return fmt.Errorf("%d:%d: unused variable %s", binding.line, binding.col, binding.name)
			}
			return fmt.Errorf("unused variable %s", binding.name)
		}
	}
	for _, child := range scope.children {
		if err := firstUnused(child); err != nil {
			return err
		}
	}
	return nil
}
