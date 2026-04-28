package checker

import (
	"fmt"
	"regexp"

	"tya/internal/ast"
)

var constNameRE = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

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
			if constants[name.Name] {
				return fmt.Errorf("%d:%d: cannot reassign constant %s", n.Tok.Line, n.Tok.Col, name.Name)
			}
			if constNameRE.MatchString(name.Name) {
				constants[name.Name] = true
			}
		case *ast.IfStmt:
			if err := checkStmts(n.Then, constants); err != nil {
				return err
			}
			if err := checkStmts(n.Else, constants); err != nil {
				return err
			}
		case *ast.WhileStmt:
			if err := checkStmts(n.Body, constants); err != nil {
				return err
			}
		case *ast.ForInStmt:
			if err := checkStmts(n.Body, constants); err != nil {
				return err
			}
		}
	}
	return nil
}
