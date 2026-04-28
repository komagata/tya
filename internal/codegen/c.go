package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"tya/internal/ast"
)

func EmitC(prog *ast.Program) (string, error) {
	g := &cgen{vars: map[string]bool{}}
	g.line("#include <stdbool.h>")
	g.line("#include <stdio.h>")
	g.line("")
	g.line("int main(void) {")
	g.indent++
	for _, stmt := range prog.Stmts {
		if err := g.stmt(stmt); err != nil {
			return "", err
		}
	}
	g.line("return 0;")
	g.indent--
	g.line("}")
	return g.out.String(), nil
}

type cgen struct {
	out    strings.Builder
	indent int
	vars   map[string]bool
}

func (g *cgen) line(s string) {
	g.out.WriteString(strings.Repeat("  ", g.indent))
	g.out.WriteString(s)
	g.out.WriteByte('\n')
}

func (g *cgen) stmt(stmt ast.Stmt) error {
	switch n := stmt.(type) {
	case *ast.AssignStmt:
		if len(n.Targets) != 1 || len(n.Values) != 1 {
			return fmt.Errorf("C emitter does not support multiple assignment yet")
		}
		id, ok := n.Targets[0].(*ast.Ident)
		if !ok {
			return fmt.Errorf("C emitter only supports variable assignment")
		}
		ex, typ, err := g.expr(n.Values[0])
		if err != nil {
			return err
		}
		if g.vars[id.Name] {
			g.line(fmt.Sprintf("%s = %s;", id.Name, ex))
		} else {
			g.vars[id.Name] = true
			g.line(fmt.Sprintf("%s %s = %s;", typ, id.Name, ex))
		}
	case *ast.ExprStmt:
		return g.exprStmt(n.Expr)
	case *ast.IfStmt:
		cond, _, err := g.expr(n.Cond)
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("if (%s) {", cond))
		g.indent++
		for _, stmt := range n.Then {
			if err := g.stmt(stmt); err != nil {
				return err
			}
		}
		g.indent--
		if len(n.Else) == 0 {
			g.line("}")
			return nil
		}
		g.line("} else {")
		g.indent++
		for _, stmt := range n.Else {
			if err := g.stmt(stmt); err != nil {
				return err
			}
		}
		g.indent--
		g.line("}")
	case *ast.WhileStmt:
		cond, _, err := g.expr(n.Cond)
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("while (%s) {", cond))
		g.indent++
		for _, stmt := range n.Body {
			if err := g.stmt(stmt); err != nil {
				return err
			}
		}
		g.indent--
		g.line("}")
	case *ast.BreakStmt:
		g.line("break;")
	case *ast.ContinueStmt:
		g.line("continue;")
	default:
		return fmt.Errorf("C emitter does not support %T", stmt)
	}
	return nil
}

func (g *cgen) exprStmt(expr ast.Expr) error {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		_, _, err := g.expr(expr)
		return err
	}
	id, ok := call.Callee.(*ast.Ident)
	if !ok || id.Name != "print" || len(call.Args) != 1 {
		return fmt.Errorf("C emitter only supports print expression statements")
	}
	arg, typ, err := g.expr(call.Args[0])
	if err != nil {
		return err
	}
	switch typ {
	case "const char*":
		g.line(fmt.Sprintf("printf(\"%%s\\n\", %s);", arg))
	case "bool":
		g.line(fmt.Sprintf("printf(\"%%s\\n\", %s ? \"true\" : \"false\");", arg))
	default:
		g.line(fmt.Sprintf("printf(\"%%g\\n\", (double)(%s));", arg))
	}
	return nil
}

func (g *cgen) expr(expr ast.Expr) (string, string, error) {
	switch n := expr.(type) {
	case *ast.IntLit:
		return strconv.FormatInt(n.Value, 10), "long", nil
	case *ast.FloatLit:
		return strconv.FormatFloat(n.Value, 'f', -1, 64), "double", nil
	case *ast.StringLit:
		return strconv.Quote(n.Value), "const char*", nil
	case *ast.BoolLit:
		if n.Value {
			return "true", "bool", nil
		}
		return "false", "bool", nil
	case *ast.Ident:
		return n.Name, "double", nil
	case *ast.BinaryExpr:
		left, _, err := g.expr(n.Left)
		if err != nil {
			return "", "", err
		}
		right, _, err := g.expr(n.Right)
		if err != nil {
			return "", "", err
		}
		op := n.Op.Lexeme
		if op == "and" {
			op = "&&"
		}
		if op == "or" {
			op = "||"
		}
		typ := "double"
		switch op {
		case "==", "!=", "<", "<=", ">", ">=", "&&", "||":
			typ = "bool"
		}
		return fmt.Sprintf("(%s %s %s)", left, op, right), typ, nil
	case *ast.UnaryExpr:
		ex, typ, err := g.expr(n.Expr)
		if err != nil {
			return "", "", err
		}
		if n.Op.Lexeme == "not" {
			return "(!" + ex + ")", "bool", nil
		}
		return "(-" + ex + ")", typ, nil
	case *ast.CallExpr:
		id, ok := n.Callee.(*ast.Ident)
		if ok && id.Name == "div" && len(n.Args) == 2 {
			left, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			right, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("(%s / %s)", left, right), "long", nil
		}
	}
	return "", "", fmt.Errorf("C emitter does not support expression %T", expr)
}
