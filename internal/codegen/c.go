package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"tya/internal/ast"
)

func EmitC(prog *ast.Program) (string, error) {
	g := &cgen{vars: map[string]bool{}, funcs: map[string]string{}}
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
	funcs   map[string]string
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
			return g.multiAssign(n)
		}
		if target, ok := n.Targets[0].(*ast.IndexExpr); ok {
			object, _, err := g.expr(target.Object)
			if err != nil {
				return err
			}
			index, _, err := g.expr(target.Index)
			if err != nil {
				return err
			}
			value, _, err := g.expr(n.Values[0])
			if err != nil {
				return err
			}
			g.line(fmt.Sprintf("tya_set_index(%s, %s, %s);", object, index, value))
			return nil
		}
		if target, ok := n.Targets[0].(*ast.ThisProp); ok {
			value, _, err := g.expr(n.Values[0])
			if err != nil {
				return err
			}
			g.line(fmt.Sprintf("tya_set_member(__this, %s, %s);", strconv.Quote(target.Name), value))
			return nil
		}
		id, ok := n.Targets[0].(*ast.Ident)
		if !ok {
			return fmt.Errorf("C emitter only supports variable assignment")
		}
		if tryExpr, ok := n.Values[0].(*ast.TryExpr); ok {
			return g.assignTry(id.Name, tryExpr)
		}
		if obj, ok := n.Values[0].(*ast.ObjectLit); ok {
			if err := g.assignObjectLit(id.Name, obj); err != nil {
				return err
			}
			return nil
		}
		if fn, ok := n.Values[0].(*ast.FuncLit); ok {
			sym, err := g.emitFunc(id.Name, fn)
			if err != nil {
				return err
			}
			g.funcs[id.Name] = sym
			g.line(fmt.Sprintf("%s = tya_function(%s);", cName(id.Name), sym))
			return nil
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
		if n.Kind == "of" {
			if g.vars[n.ValueName] {
				g.line(fmt.Sprintf("%s = tya_object_key_at(%s, tya_number(%s));", cName(n.ValueName), iterName, indexName))
			} else {
				g.vars[n.ValueName] = true
				g.line(fmt.Sprintf("TyaValue %s = tya_object_key_at(%s, tya_number(%s));", cName(n.ValueName), iterName, indexName))
			}
			if n.IndexName != "" {
				if g.vars[n.IndexName] {
					g.line(fmt.Sprintf("%s = tya_object_value_at(%s, tya_number(%s));", cName(n.IndexName), iterName, indexName))
				} else {
					g.vars[n.IndexName] = true
					g.line(fmt.Sprintf("TyaValue %s = tya_object_value_at(%s, tya_number(%s));", cName(n.IndexName), iterName, indexName))
				}
			}
		} else {
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
		if len(n.Values) > 1 {
			values := make([]string, 0, len(n.Values))
			for _, expr := range n.Values {
				value, _, err := g.expr(expr)
				if err != nil {
					return err
				}
				values = append(values, value)
			}
			g.line(fmt.Sprintf("return tya_array((TyaValue[]){%s}, %d);", strings.Join(values, ", "), len(values)))
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

func (g *cgen) assignTry(name string, tryExpr *ast.TryExpr) error {
	if !g.inFunc {
		return fmt.Errorf("C emitter only supports try inside functions")
	}
	value, _, err := g.expr(tryExpr.Expr)
	if err != nil {
		return err
	}
	temp := fmt.Sprintf("__try%d", g.temp)
	errValue := fmt.Sprintf("__tryErr%d", g.temp)
	g.temp++
	g.line(fmt.Sprintf("TyaValue %s = %s;", temp, value))
	g.line(fmt.Sprintf("TyaValue %s = tya_index(%s, tya_number(1));", errValue, temp))
	g.line(fmt.Sprintf("if (tya_truthy(%s)) {", errValue))
	g.indent++
	g.line(fmt.Sprintf("return tya_array((TyaValue[]){tya_nil(), %s}, 2);", errValue))
	g.indent--
	g.line("}")
	item := fmt.Sprintf("tya_index(%s, tya_number(0))", temp)
	if g.vars[name] {
		g.line(fmt.Sprintf("%s = %s;", cName(name), item))
	} else {
		g.vars[name] = true
		g.line(fmt.Sprintf("TyaValue %s = %s;", cName(name), item))
	}
	return nil
}

func (g *cgen) multiAssign(n *ast.AssignStmt) error {
	if len(n.Values) != 1 {
		return fmt.Errorf("C emitter only supports tuple-style multiple assignment")
	}
	value, _, err := g.expr(n.Values[0])
	if err != nil {
		return err
	}
	temp := fmt.Sprintf("__tuple%d", g.temp)
	g.temp++
	g.line(fmt.Sprintf("TyaValue %s = %s;", temp, value))
	for i, target := range n.Targets {
		id, ok := target.(*ast.Ident)
		if !ok {
			return fmt.Errorf("C emitter only supports identifier multiple assignment targets")
		}
		item := fmt.Sprintf("tya_index(%s, tya_number(%d))", temp, i)
		if g.vars[id.Name] {
			g.line(fmt.Sprintf("%s = %s;", cName(id.Name), item))
		} else {
			g.vars[id.Name] = true
			g.line(fmt.Sprintf("TyaValue %s = %s;", cName(id.Name), item))
		}
	}
	return nil
}

func (g *cgen) emitFunc(name string, fn *ast.FuncLit) (string, error) {
	sym := cFuncName(name, g.temp)
	g.temp++
	var out strings.Builder
	out.WriteString("TyaValue ")
	out.WriteString(sym)
	out.WriteString("(TyaValue __this, TyaValue __arg0, TyaValue __arg1, TyaValue __arg2) {\n")
	child := &cgen{
		vars:   map[string]bool{},
		funcs:  g.funcs,
		temp:   g.temp,
		indent: 1,
		inFunc: true,
	}
	for i, param := range fn.Params {
		child.vars[param] = true
		if i < 3 {
			child.line(fmt.Sprintf("TyaValue %s = __arg%d;", cName(param), i))
		}
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
			return "", err
		}
		child.line(fmt.Sprintf("return %s;", value))
	} else {
		body := fn.Body
		if len(body) > 0 {
			if last, ok := body[len(body)-1].(*ast.ExprStmt); ok {
				body = body[:len(body)-1]
				for _, stmt := range body {
					if err := child.stmt(stmt); err != nil {
						return "", err
					}
				}
				value, _, err := child.expr(last.Expr)
				if err != nil {
					return "", err
				}
				child.line(fmt.Sprintf("return %s;", value))
			} else {
				for _, stmt := range body {
					if err := child.stmt(stmt); err != nil {
						return "", err
					}
				}
				child.line("return tya_nil();")
			}
		} else {
			child.line("return tya_nil();")
		}
	}
	g.temp = child.temp
	g.funcOut.WriteString(child.funcOut.String())
	out.WriteString(child.out.String())
	out.WriteString("}\n\n")
	g.funcOut.WriteString(out.String())
	return sym, nil
}

func (g *cgen) assignObjectLit(name string, obj *ast.ObjectLit) error {
	entries := []string{}
	methods := []ast.ObjectProp{}
	for _, prop := range obj.Props {
		if _, ok := prop.Value.(*ast.FuncLit); ok {
			methods = append(methods, prop)
			continue
		}
		value, _, err := g.expr(prop.Value)
		if err != nil {
			return err
		}
		entries = append(entries, fmt.Sprintf("{%s, %s}", strconv.Quote(prop.Name), value))
	}
	target := cName(name)
	g.line(fmt.Sprintf("%s = tya_object((TyaObjectEntry[]){%s}, %d);", target, strings.Join(entries, ", "), len(entries)))
	for _, method := range methods {
		fn := method.Value.(*ast.FuncLit)
		funcName := name + "_" + method.Name
		sym, err := g.emitFunc(funcName, fn)
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_set_member(%s, %s, tya_method(%s, %s));", target, strconv.Quote(method.Name), sym, target))
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
		_, _, err := g.expr(expr)
		return err
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
	if id.Name == "delete" && len(call.Args) == 2 {
		object, _, err := g.expr(call.Args[0])
		if err != nil {
			return err
		}
		key, _, err := g.expr(call.Args[1])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_delete(%s, %s);", object, key))
		return nil
	}
	if id.Name == "writeFile" && len(call.Args) == 2 {
		path, _, err := g.expr(call.Args[0])
		if err != nil {
			return err
		}
		text, _, err := g.expr(call.Args[1])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_write_file(%s, %s);", path, text))
		return nil
	}
	if id.Name == "exit" && len(call.Args) == 1 {
		code, _, err := g.expr(call.Args[0])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_exit(%s);", code))
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
		if strings.Contains(n.Value, "{") {
			return interpolateString(n.Value), "TyaValue", nil
		}
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
		if len(n.Props) == 0 {
			return "tya_object(0, 0)", "TyaValue", nil
		}
		entries := make([]string, 0, len(n.Props))
		for _, prop := range n.Props {
			value, _, err := g.expr(prop.Value)
			if err != nil {
				return "", "", err
			}
			entries = append(entries, fmt.Sprintf("{%s, %s}", strconv.Quote(prop.Name), value))
		}
		return fmt.Sprintf("tya_object((TyaObjectEntry[]){%s}, %d)", strings.Join(entries, ", "), len(entries)), "TyaValue", nil
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
			expr = fmt.Sprintf("tya_and(%s, %s)", left, right)
		case "||":
			expr = fmt.Sprintf("tya_or(%s, %s)", left, right)
		case "%":
			expr = fmt.Sprintf("tya_number((long)%s.number %% (long)%s.number)", left, right)
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
		if ok && id.Name == "startsWith" && len(n.Args) == 2 {
			text, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			prefix, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_starts_with(%s, %s)", text, prefix), "TyaValue", nil
		}
		if ok && id.Name == "endsWith" && len(n.Args) == 2 {
			text, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			suffix, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_ends_with(%s, %s)", text, suffix), "TyaValue", nil
		}
		if ok && id.Name == "trim" && len(n.Args) == 1 {
			text, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_trim(%s)", text), "TyaValue", nil
		}
		if ok && id.Name == "replace" && len(n.Args) == 3 {
			text, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			old, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			replacement, _, err := g.expr(n.Args[2])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_replace(%s, %s, %s)", text, old, replacement), "TyaValue", nil
		}
		if ok && id.Name == "byteLen" && len(n.Args) == 1 {
			text, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_byte_len(%s)", text), "TyaValue", nil
		}
		if ok && id.Name == "charLen" && len(n.Args) == 1 {
			text, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_char_len(%s)", text), "TyaValue", nil
		}
		if ok && id.Name == "args" && len(n.Args) == 0 {
			return "tya_args(argc, argv)", "TyaValue", nil
		}
		if ok && id.Name == "env" && len(n.Args) == 1 {
			name, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_env(%s)", name), "TyaValue", nil
		}
		if ok && id.Name == "readLine" && len(n.Args) == 0 {
			return "tya_read_line()", "TyaValue", nil
		}
		if ok && id.Name == "readFile" && len(n.Args) == 1 {
			path, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_read_file(%s)", path), "TyaValue", nil
		}
		if ok && id.Name == "error" && len(n.Args) == 1 {
			message, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_error(%s)", message), "TyaValue", nil
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
		if ok && id.Name == "join" && len(n.Args) == 2 {
			array, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			sep, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_join(%s, %s)", array, sep), "TyaValue", nil
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
		if ok && id.Name == "toFloat" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_to_float(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "toNumber" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_to_number(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "fileExists" && len(n.Args) == 1 {
			path, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_file_exists(%s)", path), "TyaValue", nil
		}
		if ok && id.Name == "equal" && len(n.Args) == 2 {
			left, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			right, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_deep_equal(%s, %s)", left, right), "TyaValue", nil
		}
		if ok && id.Name == "has" && len(n.Args) == 2 {
			object, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			key, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_has(%s, %s)", object, key), "TyaValue", nil
		}
		if ok && id.Name == "keys" && len(n.Args) == 1 {
			object, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_keys(%s)", object), "TyaValue", nil
		}
		if ok && id.Name == "values" && len(n.Args) == 1 {
			object, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_values(%s)", object), "TyaValue", nil
		}
		if ok && id.Name == "pop" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_pop(%s)", arg), "TyaValue", nil
		}
		if ok && (id.Name == "map" || id.Name == "filter" || id.Name == "find" || id.Name == "any" || id.Name == "all" || id.Name == "each") && len(n.Args) == 2 {
			array, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			fn, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_%s(%s, %s)", id.Name, array, fn), "TyaValue", nil
		}
		if ok && id.Name == "reduce" && len(n.Args) == 3 {
			array, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			initial, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			fn, _, err := g.expr(n.Args[2])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_reduce(%s, %s, %s)", array, initial, fn), "TyaValue", nil
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
			if sym, found := g.funcs[id.Name]; found {
				args := make([]string, 0, len(n.Args))
				for _, arg := range n.Args {
					ex, _, err := g.expr(arg)
					if err != nil {
						return "", "", err
					}
					args = append(args, ex)
				}
				for len(args) < 3 {
					args = append(args, "tya_nil()")
				}
				return fmt.Sprintf("%s(tya_nil(), %s)", sym, strings.Join(args[:3], ", ")), "TyaValue", nil
			}
		}
		if member, ok := n.Callee.(*ast.MemberExpr); ok {
			receiver, _, err := g.expr(member.Object)
			if err != nil {
				return "", "", err
			}
			args := make([]string, 0, len(n.Args))
			for _, arg := range n.Args {
				ex, _, err := g.expr(arg)
				if err != nil {
					return "", "", err
				}
				args = append(args, ex)
			}
			switch len(args) {
			case 0:
				return fmt.Sprintf("tya_call1(tya_member(%s, %s), tya_nil())", receiver, strconv.Quote(member.Name)), "TyaValue", nil
			case 1:
				return fmt.Sprintf("tya_call1(tya_member(%s, %s), %s)", receiver, strconv.Quote(member.Name), args[0]), "TyaValue", nil
			case 2:
				return fmt.Sprintf("tya_call2(tya_member(%s, %s), %s, %s)", receiver, strconv.Quote(member.Name), args[0], args[1]), "TyaValue", nil
			case 3:
				return fmt.Sprintf("tya_call3(tya_member(%s, %s), %s, %s, %s)", receiver, strconv.Quote(member.Name), args[0], args[1], args[2]), "TyaValue", nil
			}
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
		object, _, err := g.expr(n.Object)
		if err != nil {
			return "", "", err
		}
		return fmt.Sprintf("tya_member(%s, %s)", object, strconv.Quote(n.Name)), "TyaValue", nil
	case *ast.ThisProp:
		return fmt.Sprintf("tya_member(__this, %s)", strconv.Quote(n.Name)), "TyaValue", nil
	case *ast.TryExpr:
		return g.expr(n.Expr)
	}
	return "", "", fmt.Errorf("C emitter does not support expression %T", expr)
}

func cName(name string) string {
	switch name {
	case "auto", "break", "case", "char", "const", "continue", "default", "do", "double",
		"else", "enum", "extern", "float", "for", "goto", "if", "inline", "int",
		"long", "register", "restrict", "return", "short", "signed", "sizeof",
		"static", "struct", "switch", "typedef", "union", "unsigned", "void",
		"volatile", "while":
		return "tya_" + name
	}
	return name
}

func cFuncName(name string, serial int) string {
	return fmt.Sprintf("tya_fn_%s_%d", cName(name), serial)
}

func interpolateString(value string) string {
	parts := []string{}
	for len(value) > 0 {
		open := strings.IndexByte(value, '{')
		if open < 0 {
			if value != "" {
				parts = append(parts, "tya_string("+strconv.Quote(value)+")")
			}
			break
		}
		if open > 0 {
			parts = append(parts, "tya_string("+strconv.Quote(value[:open])+")")
		}
		close := strings.IndexByte(value[open+1:], '}')
		if close < 0 {
			parts = append(parts, "tya_string("+strconv.Quote(value[open:])+")")
			break
		}
		name := value[open+1 : open+1+close]
		if expr, ok := interpolationExpr(name); ok {
			parts = append(parts, "tya_to_string("+expr+")")
		} else {
			parts = append(parts, "tya_string("+strconv.Quote(value[open:open+close+2])+")")
		}
		value = value[open+close+2:]
	}
	if len(parts) == 0 {
		return "tya_string(\"\")"
	}
	expr := parts[0]
	for _, part := range parts[1:] {
		expr = "tya_add(" + expr + ", " + part + ")"
	}
	return expr
}

func isIdentName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func isPathName(name string) bool {
	if name == "" {
		return false
	}
	for _, part := range strings.Split(name, ".") {
		if !isIdentName(part) {
			return false
		}
	}
	return true
}

func pathExpr(name string) string {
	parts := strings.Split(name, ".")
	expr := cName(parts[0])
	for _, part := range parts[1:] {
		expr = "tya_member(" + expr + ", " + strconv.Quote(part) + ")"
	}
	return expr
}

func interpolationExpr(expr string) (string, bool) {
	expr = strings.TrimSpace(expr)
	if isPathName(expr) {
		return pathExpr(expr), true
	}
	for _, op := range []string{" + "} {
		if strings.Contains(expr, op) {
			parts := strings.Split(expr, op)
			if len(parts) != 2 {
				return "", false
			}
			left, ok := interpolationExpr(parts[0])
			if !ok {
				return "", false
			}
			right, ok := interpolationExpr(parts[1])
			if !ok {
				return "", false
			}
			return "tya_add(" + left + ", " + right + ")", true
		}
	}
	if n, err := strconv.ParseInt(expr, 10, 64); err == nil {
		return "tya_number(" + strconv.FormatInt(n, 10) + ")", true
	}
	if strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"") {
		unquoted, err := strconv.Unquote(expr)
		if err != nil {
			return "", false
		}
		return "tya_string(" + strconv.Quote(unquoted) + ")", true
	}
	return "", false
}

func assignedNames(stmts []ast.Stmt) []string {
	seen := map[string]bool{}
	var names []string
	var walk func([]ast.Stmt)
	walk = func(stmts []ast.Stmt) {
		for _, stmt := range stmts {
			switch n := stmt.(type) {
			case *ast.AssignStmt:
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
				for _, name := range []string{n.ValueName, n.IndexName} {
					if name == "" || seen[name] {
						continue
					}
					seen[name] = true
					names = append(names, name)
				}
				walk(n.Body)
			}
		}
	}
	walk(stmts)
	return names
}
