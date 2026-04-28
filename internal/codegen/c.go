package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"tya/internal/ast"
)

func EmitC(prog *ast.Program) (string, error) {
	g := &cgen{vars: map[string]bool{}, funcs: map[string]bool{}}
	for _, name := range assignedNames(prog.Stmts) {
		g.vars[name] = true
		g.line(fmt.Sprintf("TyaValue %s = tya_nil();", cName(name)))
	}
	for _, stmt := range prog.Stmts {
		if err := g.stmt(stmt); err != nil {
			return "", err
		}
	}
	var out strings.Builder
	out.WriteString("#include \"tya_runtime.h\"\n\n")
	out.WriteString(g.funcOut.String())
	out.WriteString("int main(int argc, char **argv) {\n")
	out.WriteString(g.out.String())
	out.WriteString("  return 0;\n")
	out.WriteString("}\n")
	return out.String(), nil
}

type cgen struct {
	out     strings.Builder
	funcOut strings.Builder
	indent  int
	vars    map[string]bool
	funcs   map[string]bool
	temp    int
	inFunc  bool
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
		if fn, ok := n.Values[0].(*ast.FuncLit); ok {
			return g.emitFunc(id.Name, fn)
		}
		ex, typ, err := g.expr(n.Values[0])
		if err != nil {
			return err
		}
		if g.vars[id.Name] {
			g.line(fmt.Sprintf("%s = %s;", cName(id.Name), ex))
		} else {
			g.vars[id.Name] = true
			_ = typ
			g.line(fmt.Sprintf("TyaValue %s = %s;", cName(id.Name), ex))
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
		iterName := fmt.Sprintf("__iter%d", g.temp)
		indexName := fmt.Sprintf("__i%d", g.temp)
		g.temp++
		g.line(fmt.Sprintf("TyaValue %s = %s;", iterName, iterable))
		g.line(fmt.Sprintf("for (int %s = 0; %s < (int)tya_len(%s).number; %s++) {", indexName, indexName, iterName, indexName))
		g.indent++
		if n.IndexName != "" {
			if g.vars[n.IndexName] {
				g.line(fmt.Sprintf("%s = tya_number(%s);", cName(n.IndexName), indexName))
			} else {
				g.vars[n.IndexName] = true
				g.line(fmt.Sprintf("TyaValue %s = tya_number(%s);", cName(n.IndexName), indexName))
			}
		}
		if g.vars[n.ValueName] {
			g.line(fmt.Sprintf("%s = tya_index(%s, tya_number(%s));", cName(n.ValueName), iterName, indexName))
		} else {
			g.vars[n.ValueName] = true
			g.line(fmt.Sprintf("TyaValue %s = tya_index(%s, tya_number(%s));", cName(n.ValueName), iterName, indexName))
		}
		for _, stmt := range n.Body {
			if err := g.stmt(stmt); err != nil {
				return err
			}
		}
		g.indent--
		g.line("}")
	case *ast.ReturnStmt:
		if !g.inFunc {
			g.line("/* return */")
			return nil
		}
		if len(n.Values) == 0 {
			g.line("return tya_nil();")
			return nil
		}
		value, _, err := g.expr(n.Values[0])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("return %s;", value))
	default:
		return fmt.Errorf("C emitter does not support %T", stmt)
	}
	return nil
}

func (g *cgen) emitFunc(name string, fn *ast.FuncLit) error {
	g.funcs[name] = true
	var out strings.Builder
	out.WriteString("TyaValue ")
	out.WriteString(cName(name))
	out.WriteByte('(')
	for i, param := range fn.Params {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString("TyaValue ")
		out.WriteString(cName(param))
	}
	out.WriteString(") {\n")
	child := &cgen{
		vars:   map[string]bool{},
		funcs:  g.funcs,
		temp:   g.temp,
		indent: 1,
		inFunc: true,
	}
	for _, param := range fn.Params {
		child.vars[param] = true
	}
	for _, local := range assignedNames(fn.Body) {
		if child.vars[local] {
			continue
		}
		child.vars[local] = true
		child.line(fmt.Sprintf("TyaValue %s = tya_nil();", cName(local)))
	}
	if fn.Expr != nil {
		value, _, err := child.expr(fn.Expr)
		if err != nil {
			return err
		}
		child.line(fmt.Sprintf("return %s;", value))
	} else {
		for _, stmt := range fn.Body {
			if err := child.stmt(stmt); err != nil {
				return err
			}
		}
		child.line("return tya_nil();")
	}
	g.temp = child.temp
	out.WriteString(child.out.String())
	out.WriteString("}\n\n")
	g.funcOut.WriteString(out.String())
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
		return cName(n.Name), "TyaValue", nil
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
		if ok && g.funcs[id.Name] {
			args := make([]string, 0, len(n.Args))
			for _, arg := range n.Args {
				ex, _, err := g.expr(arg)
				if err != nil {
					return "", "", err
				}
				args = append(args, ex)
			}
			return fmt.Sprintf("%s(%s)", cName(id.Name), strings.Join(args, ", ")), "TyaValue", nil
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

func cName(name string) string {
	if name == "char" {
		return "tya_char"
	}
	return name
}

func assignedNames(stmts []ast.Stmt) []string {
	seen := map[string]bool{}
	var names []string
	var walk func([]ast.Stmt)
	walk = func(stmts []ast.Stmt) {
		for _, stmt := range stmts {
			switch n := stmt.(type) {
			case *ast.AssignStmt:
				if len(n.Values) == 1 {
					if _, ok := n.Values[0].(*ast.FuncLit); ok {
						continue
					}
				}
				for _, target := range n.Targets {
					id, ok := target.(*ast.Ident)
					if !ok || seen[id.Name] {
						continue
					}
					seen[id.Name] = true
					names = append(names, id.Name)
				}
			case *ast.IfStmt:
				walk(n.Then)
				walk(n.Else)
			case *ast.WhileStmt:
				walk(n.Body)
			case *ast.ForInStmt:
				walk(n.Body)
			}
		}
	}
	walk(stmts)
	return names
}
