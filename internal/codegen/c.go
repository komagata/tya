package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"tya/internal/ast"
)

func EmitC(prog *ast.Program) (string, error) {
	g := &cgen{vars: map[string]bool{}}
	g.line("#include \"tya_runtime.h\"")
	g.line("")
	g.line("int main(int argc, char **argv) {")
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
			_ = typ
			g.line(fmt.Sprintf("TyaValue %s = %s;", id.Name, ex))
		}
	case *ast.ExprStmt:
		return g.exprStmt(n.Expr)
	case *ast.IfStmt:
		cond, _, err := g.expr(n.Cond)
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("if (tya_truthy(%s)) {", cond))
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
		g.line(fmt.Sprintf("while (tya_truthy(%s)) {", cond))
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
	case *ast.ForInStmt:
		iterable, _, err := g.expr(n.Iterable)
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("if (false) { /* for %s in %s */", n.ValueName, iterable))
		g.indent++
		if n.IndexName != "" && !g.vars[n.IndexName] {
			g.vars[n.IndexName] = true
			g.line(fmt.Sprintf("TyaValue %s = tya_nil();", n.IndexName))
		}
		if !g.vars[n.ValueName] {
			g.vars[n.ValueName] = true
			g.line(fmt.Sprintf("TyaValue %s = tya_nil();", n.ValueName))
		}
		for _, stmt := range n.Body {
			if err := g.stmt(stmt); err != nil {
				return err
			}
		}
		g.indent--
		g.line("}")
	case *ast.ReturnStmt:
		g.line("/* return */")
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
	if !ok {
		return fmt.Errorf("C emitter only supports simple call expression statements")
	}
	if id.Name == "push" && len(call.Args) == 2 {
		array, _, err := g.expr(call.Args[0])
		if err != nil {
			return err
		}
		value, _, err := g.expr(call.Args[1])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_push(%s, %s);", array, value))
		return nil
	}
	if id.Name != "print" || len(call.Args) != 1 {
		g.line(fmt.Sprintf("/* call %s */", id.Name))
		return nil
	}
	arg, typ, err := g.expr(call.Args[0])
	if err != nil {
		return err
	}
	_ = typ
	g.line(fmt.Sprintf("tya_print(%s);", arg))
	return nil
}

func (g *cgen) expr(expr ast.Expr) (string, string, error) {
	switch n := expr.(type) {
	case *ast.IntLit:
		return "tya_number(" + strconv.FormatInt(n.Value, 10) + ")", "TyaValue", nil
	case *ast.FloatLit:
		return "tya_number(" + strconv.FormatFloat(n.Value, 'f', -1, 64) + ")", "TyaValue", nil
	case *ast.StringLit:
		return "tya_string(" + strconv.Quote(n.Value) + ")", "TyaValue", nil
	case *ast.BoolLit:
		if n.Value {
			return "tya_bool(true)", "TyaValue", nil
		}
		return "tya_bool(false)", "TyaValue", nil
	case *ast.NilLit:
		return "tya_nil()", "TyaValue", nil
	case *ast.ArrayLit:
		if len(n.Elems) == 0 {
			return "tya_array(0, 0)", "TyaValue", nil
		}
		elems := make([]string, 0, len(n.Elems))
		for _, elem := range n.Elems {
			ex, _, err := g.expr(elem)
			if err != nil {
				return "", "", err
			}
			elems = append(elems, ex)
		}
		return fmt.Sprintf("tya_array((TyaValue[]){%s}, %d)", strings.Join(elems, ", "), len(elems)), "TyaValue", nil
	case *ast.ObjectLit:
		return "tya_nil() /* object */", "TyaValue", nil
	case *ast.FuncLit:
		return "tya_nil() /* function */", "TyaValue", nil
	case *ast.Ident:
		return n.Name, "TyaValue", nil
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
		typ := "TyaValue"
		expr := fmt.Sprintf("(%s.number %s %s.number)", left, op, right)
		switch op {
		case "+":
			expr = fmt.Sprintf("tya_add(%s, %s)", left, right)
		case "==":
			expr = fmt.Sprintf("tya_bool(tya_equal(%s, %s))", left, right)
		case "!=":
			expr = fmt.Sprintf("tya_bool(!tya_equal(%s, %s))", left, right)
		case "&&":
			expr = fmt.Sprintf("tya_bool(tya_truthy(%s) && tya_truthy(%s))", left, right)
		case "||":
			expr = fmt.Sprintf("tya_bool(tya_truthy(%s) || tya_truthy(%s))", left, right)
		case "<", "<=", ">", ">=":
			expr = fmt.Sprintf("tya_bool(%s.number %s %s.number)", left, op, right)
		default:
			expr = fmt.Sprintf("tya_number(%s)", expr)
		}
		return expr, typ, nil
	case *ast.UnaryExpr:
		ex, typ, err := g.expr(n.Expr)
		if err != nil {
			return "", "", err
		}
		if n.Op.Lexeme == "not" {
			return "tya_bool(!tya_truthy(" + ex + "))", "TyaValue", nil
		}
		return "tya_number(-" + ex + ".number)", typ, nil
	case *ast.CallExpr:
		id, ok := n.Callee.(*ast.Ident)
		if ok && id.Name == "len" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_len(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "contains" && len(n.Args) == 2 {
			text, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			part, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_contains(%s, %s)", text, part), "TyaValue", nil
		}
		if ok && id.Name == "args" && len(n.Args) == 0 {
			return "tya_args(argc, argv)", "TyaValue", nil
		}
		if ok && id.Name == "readFile" && len(n.Args) == 1 {
			path, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_read_file(%s)", path), "TyaValue", nil
		}
		if ok && id.Name == "split" && len(n.Args) == 2 {
			text, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			sep, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_split(%s, %s)", text, sep), "TyaValue", nil
		}
		if ok && id.Name == "toString" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_to_string(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "toInt" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_to_int(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "div" && len(n.Args) == 2 {
			left, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			right, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_number((long)%s.number / (long)%s.number)", left, right), "TyaValue", nil
		}
		if ok {
			return fmt.Sprintf("tya_nil() /* call %s */", id.Name), "TyaValue", nil
		}
		return "tya_nil() /* call */", "TyaValue", nil
	case *ast.IndexExpr:
		object, _, err := g.expr(n.Object)
		if err != nil {
			return "", "", err
		}
		index, _, err := g.expr(n.Index)
		if err != nil {
			return "", "", err
		}
		return fmt.Sprintf("tya_index(%s, %s)", object, index), "TyaValue", nil
	case *ast.MemberExpr:
		return "tya_nil() /* member */", "TyaValue", nil
	case *ast.TryExpr:
		return g.expr(n.Expr)
	}
	return "", "", fmt.Errorf("C emitter does not support expression %T", expr)
}
