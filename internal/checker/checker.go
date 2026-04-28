package checker

import (
	"fmt"
	"regexp"

	"tya/internal/ast"
)

var constNameRE = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
var valueNameRE = regexp.MustCompile(`^_?[a-z][A-Za-z0-9]*$|^_$`)

func Check(prog *ast.Program) error {
	constants := map[string]bool{}
	return checkStmts(prog.Stmts, constants)
}

func checkStmts(stmts []ast.Stmt, constants map[string]bool) error {
	for _, stmt := range stmts {
		switch n := stmt.(type) {
		case *ast.AssignStmt:
			name, ok := n.Target.(*ast.Ident)
			if !ok {
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
			if err := checkExpr(n.Value); err != nil {
				return err
			}
		case *ast.IfStmt:
			if err := checkExpr(n.Cond); err != nil {
				return err
			}
			if err := checkStmts(n.Then, constants); err != nil {
				return err
			}
			if err := checkStmts(n.Else, constants); err != nil {
				return err
			}
		case *ast.WhileStmt:
			if err := checkExpr(n.Cond); err != nil {
				return err
			}
			if err := checkStmts(n.Body, constants); err != nil {
				return err
			}
		case *ast.ForInStmt:
			if err := checkBindingName(n.ValueName, 0, 0); err != nil {
				return err
			}
			if n.IndexName != "" {
				if err := checkBindingName(n.IndexName, 0, 0); err != nil {
					return err
				}
			}
			if err := checkExpr(n.Iterable); err != nil {
				return err
			}
			if err := checkStmts(n.Body, constants); err != nil {
				return err
			}
		case *ast.ExprStmt:
			if err := checkExpr(n.Expr); err != nil {
				return err
			}
		case *ast.ReturnStmt:
			if n.Value != nil {
				if err := checkExpr(n.Value); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func checkExpr(expr ast.Expr) error {
	switch n := expr.(type) {
	case *ast.ObjectLit:
		seen := map[string]bool{}
		for _, prop := range n.Props {
			if !valueNameRE.MatchString(prop.Name) {
				return fmt.Errorf("invalid object property name %s", prop.Name)
			}
			if seen[prop.Name] {
				return fmt.Errorf("duplicate object property %s", prop.Name)
			}
			seen[prop.Name] = true
			if err := checkExpr(prop.Value); err != nil {
				return err
			}
		}
	case *ast.FuncLit:
		seen := map[string]bool{}
		for _, param := range n.Params {
			if err := checkBindingName(param, 0, 0); err != nil {
				return err
			}
			if seen[param] && param != "_" {
				return fmt.Errorf("duplicate function parameter %s", param)
			}
			seen[param] = true
		}
		if n.Expr != nil {
			if err := checkExpr(n.Expr); err != nil {
				return err
			}
		}
		if err := checkStmts(n.Body, map[string]bool{}); err != nil {
			return err
		}
	case *ast.ArrayLit:
		for _, elem := range n.Elems {
			if err := checkExpr(elem); err != nil {
				return err
			}
		}
	case *ast.BinaryExpr:
		if err := checkExpr(n.Left); err != nil {
			return err
		}
		return checkExpr(n.Right)
	case *ast.UnaryExpr:
		return checkExpr(n.Expr)
	case *ast.MemberExpr:
		return checkExpr(n.Object)
	case *ast.IndexExpr:
		if err := checkExpr(n.Object); err != nil {
			return err
		}
		return checkExpr(n.Index)
	case *ast.CallExpr:
		if err := checkExpr(n.Callee); err != nil {
			return err
		}
		for _, arg := range n.Args {
			if err := checkExpr(arg); err != nil {
				return err
			}
		}
	}
	return nil
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
