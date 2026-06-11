package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"tya/internal/ast"
)

func (g *cgen) exprStmt(expr ast.Expr) error {
	// v0.42: spawn / await have side effects (start / join a worker
	// thread). When used at statement position the value is dropped
	// but the call still has to be emitted, so route through the same
	// (void) path as CallExpr.
	switch expr.(type) {
	case *ast.SpawnExpr, *ast.AwaitExpr:
		ex, _, err := g.expr(expr)
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("(void)%s;", ex))
		return nil
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		_, _, err := g.expr(expr)
		return err
	}
	id, ok := call.Callee.(*ast.Ident)
	if !ok {
		ex, _, err := g.expr(expr)
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("(void)%s;", ex))
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
		dict, _, err := g.expr(call.Args[0])
		if err != nil {
			return err
		}
		key, _, err := g.expr(call.Args[1])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_delete(%s, %s);", dict, key))
		return nil
	}
	if id.Name == "write_file" && len(call.Args) == 2 {
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
	if id.Name == "panic" && len(call.Args) == 1 {
		message, _, err := g.expr(call.Args[0])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_panic(%s);", message))
		return nil
	}
	if id.Name == "dir_mkdir" && len(call.Args) == 1 {
		arg, _, err := g.expr(call.Args[0])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_dir_mkdir(%s);", arg))
		return nil
	}
	if id.Name == "dir_rmdir" && len(call.Args) == 1 {
		arg, _, err := g.expr(call.Args[0])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_dir_rmdir(%s);", arg))
		return nil
	}
	if id.Name == "file_remove" && len(call.Args) == 1 {
		arg, _, err := g.expr(call.Args[0])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_file_remove(%s);", arg))
		return nil
	}
	if id.Name == "file_rename" && len(call.Args) == 2 {
		oldArg, _, err := g.expr(call.Args[0])
		if err != nil {
			return err
		}
		newArg, _, err := g.expr(call.Args[1])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_file_rename(%s, %s);", oldArg, newArg))
		return nil
	}
	if id.Name == "chdir" && len(call.Args) == 1 {
		arg, _, err := g.expr(call.Args[0])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_chdir(%s);", arg))
		return nil
	}
	line := 1
	if ident, ok := call.Callee.(*ast.Ident); ok {
		line = ident.Tok.Line
	}
	if id.Name == "assert" && len(call.Args) == 1 {
		arg, _, err := g.expr(call.Args[0])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_assert(%s, %s, %d);", arg, strconv.Quote(g.sourcePath), line))
		return nil
	}
	if id.Name == "assert_equal" && len(call.Args) == 2 {
		expected, _, err := g.expr(call.Args[0])
		if err != nil {
			return err
		}
		actual, _, err := g.expr(call.Args[1])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_assert_equal(%s, %s, %s, %d);", expected, actual, strconv.Quote(g.sourcePath), line))
		return nil
	}
	if (id.Name != "print" && id.Name != "println") || len(call.Args) != 1 {
		ex, _, err := g.expr(call)
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("(void)%s;", ex))
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

func (g *cgen) sideEffectCallExpr(name string, args []ast.Expr) (string, bool, error) {
	switch name {
	case "print", "println":
		if len(args) != 1 {
			return "", false, nil
		}
		arg, _, err := g.expr(args[0])
		if err != nil {
			return "", true, err
		}
		g.line(fmt.Sprintf("tya_print(%s);", arg))
		return "tya_nil()", true, nil
	case "assert":
		if len(args) != 1 && len(args) != 2 {
			return "", false, nil
		}
		arg, _, err := g.expr(args[0])
		if err != nil {
			return "", true, err
		}
		g.line(fmt.Sprintf("tya_assert(%s, %s, %d);", arg, strconv.Quote(g.sourcePath), 1))
		return "tya_nil()", true, nil
	case "assert_equal":
		if len(args) < 2 || len(args) > 3 {
			return "", false, nil
		}
		expected, _, err := g.expr(args[0])
		if err != nil {
			return "", true, err
		}
		actual, _, err := g.expr(args[1])
		if err != nil {
			return "", true, err
		}
		g.line(fmt.Sprintf("tya_assert_equal(%s, %s, %s, %d);", expected, actual, strconv.Quote(g.sourcePath), 1))
		return "tya_nil()", true, nil
	default:
		return "", false, nil
	}
}

func (g *cgen) assignControlBodyValue(result string, body []ast.Stmt) error {
	if len(body) == 0 {
		return nil
	}
	for _, stmt := range body[:len(body)-1] {
		if err := g.stmt(stmt); err != nil {
			return err
		}
	}
	last := body[len(body)-1]
	if exprStmt, ok := last.(*ast.ExprStmt); ok {
		if isSideEffectCall(exprStmt.Expr) {
			if err := g.stmt(last); err != nil {
				return err
			}
			g.line(fmt.Sprintf("%s = tya_nil();", result))
			return nil
		}
		value, _, err := g.expr(exprStmt.Expr)
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("%s = %s;", result, value))
		return nil
	}
	return g.stmt(last)
}

func (g *cgen) ifExpr(n *ast.IfStmt) (string, string, error) {
	result := fmt.Sprintf("__ifexpr%d", g.temp)
	g.temp++
	g.line(fmt.Sprintf("TyaValue %s = tya_nil();", result))
	cond, _, err := g.expr(n.Cond)
	if err != nil {
		return "", "", err
	}
	g.line(fmt.Sprintf("if (tya_truthy(%s)) {", cond))
	g.indent++
	if err := g.assignControlBodyValue(result, n.Then); err != nil {
		return "", "", err
	}
	g.indent--
	if err := g.ifExprElse(result, n.Else); err != nil {
		return "", "", err
	}
	return result, "TyaValue", nil
}

func (g *cgen) ifExprElse(result string, body []ast.Stmt) error {
	if len(body) == 0 {
		g.line("}")
		return nil
	}
	if len(body) == 1 {
		if inner, ok := body[0].(*ast.IfStmt); ok {
			cond, _, err := g.expr(inner.Cond)
			if err != nil {
				return err
			}
			g.line(fmt.Sprintf("} else if (tya_truthy(%s)) {", cond))
			g.indent++
			if err := g.assignControlBodyValue(result, inner.Then); err != nil {
				return err
			}
			g.indent--
			return g.ifExprElse(result, inner.Else)
		}
	}
	g.line("} else {")
	g.indent++
	if err := g.assignControlBodyValue(result, body); err != nil {
		return err
	}
	g.indent--
	g.line("}")
	return nil
}

func (g *cgen) whileExpr(n *ast.WhileStmt) (string, string, error) {
	result := fmt.Sprintf("__loopexpr%d", g.temp)
	g.temp++
	g.line(fmt.Sprintf("TyaValue %s = tya_nil();", result))
	cond, _, err := g.expr(n.Cond)
	if err != nil {
		return "", "", err
	}
	g.line(fmt.Sprintf("while (tya_truthy(%s)) {", cond))
	g.indent++
	if err := g.assignControlBodyValue(result, n.Body); err != nil {
		return "", "", err
	}
	g.indent--
	g.line("}")
	return result, "TyaValue", nil
}

func (g *cgen) forExpr(n *ast.ForInStmt) (string, string, error) {
	result := fmt.Sprintf("__loopexpr%d", g.temp)
	g.temp++
	g.line(fmt.Sprintf("TyaValue %s = tya_nil();", result))
	if err := g.forInStmt(n, result); err != nil {
		return "", "", err
	}
	return result, "TyaValue", nil
}

func (g *cgen) matchExpr(n *ast.MatchStmt) (string, string, error) {
	result := fmt.Sprintf("__matchexpr%d", g.temp)
	g.temp++
	g.line(fmt.Sprintf("TyaValue %s = tya_nil();", result))
	if err := g.matchStmtWithResult(n, result); err != nil {
		return "", "", err
	}
	return result, "TyaValue", nil
}

func (g *cgen) expr(expr ast.Expr) (string, string, error) {
	switch n := expr.(type) {
	case *ast.IntLit:
		return "tya_number(" + strconv.FormatInt(n.Value, 10) + ")", "TyaValue", nil
	case *ast.FloatLit:
		return "tya_float(" + strconv.FormatFloat(n.Value, 'f', -1, 64) + ")", "TyaValue", nil
	case *ast.StringLit:
		if strings.ContainsAny(n.Value, "{}") {
			value, err := g.interpolateString(n.Value)
			return value, "TyaValue", err
		}
		return "tya_string(" + strconv.Quote(n.Value) + ")", "TyaValue", nil
	case *ast.BytesLit:
		if n.Value == "" {
			return "tya_bytes_lit((const char *)0, 0)", "TyaValue", nil
		}
		var b strings.Builder
		b.WriteString("tya_bytes_lit((const char *)(unsigned char[]){")
		for i := 0; i < len(n.Value); i++ {
			if i > 0 {
				b.WriteByte(',')
			}
			fmt.Fprintf(&b, "0x%02x", n.Value[i])
		}
		fmt.Fprintf(&b, "}, %d)", len(n.Value))
		return b.String(), "TyaValue", nil
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
	case *ast.DictLit:
		if len(n.Props) == 0 {
			return "tya_dict(0, 0)", "TyaValue", nil
		}
		entries := make([]string, 0, len(n.Props))
		for _, prop := range n.Props {
			value, _, err := g.expr(prop.Value)
			if err != nil {
				return "", "", err
			}
			entries = append(entries, fmt.Sprintf("{%s, %s}", strconv.Quote(prop.Name), value))
		}
		return fmt.Sprintf("tya_dict((TyaDictEntry[]){%s}, %d)", strings.Join(entries, ", "), len(entries)), "TyaValue", nil
	case *ast.FuncLit:
		name := fmt.Sprintf("__anon%d", g.temp)
		captures := g.freeVars(n)
		sym, err := g.emitFuncWithCaptures(name, n, "", "", captures)
		if err != nil {
			return "", "", err
		}
		if len(captures) == 0 {
			return fmt.Sprintf("tya_function_params(%s, %s)", sym, cParamArray(n.Params)), "TyaValue", nil
		}
		env := fmt.Sprintf("__env%d", g.temp)
		g.temp++
		lines := []string{fmt.Sprintf("TyaValue %s = tya_object();", env)}
		for _, name := range sortedMapKeys(captures) {
			lines = append(lines, fmt.Sprintf("tya_set_member(%s, %s, %s);", env, strconv.Quote(name), g.captureValue(name)))
		}
		lines = append(lines, fmt.Sprintf("tya_bind_method_params(%s, %s, %s)", env, sym, cParamArray(n.Params)))
		return fmt.Sprintf("({ %s; })", strings.Join(lines, " ")), "TyaValue", nil
	case *ast.Ident:
		if n.ImplicitField {
			return fmt.Sprintf("tya_member(__this, %s)", strconv.Quote("@"+n.Name)), "TyaValue", nil
		}
		if g.closureVars != nil && g.closureVars[n.Name] {
			return fmt.Sprintf("tya_member(__this, %s)", strconv.Quote(n.Name)), "TyaValue", nil
		}
		if g.currentClassHasConstant(n.Name) {
			return fmt.Sprintf("tya_member(%s, %s)", g.classTarget(), strconv.Quote(n.Name)), "TyaValue", nil
		}
		if primitiveClassNames[n.Name] {
			return fmt.Sprintf("tya_primitive_class(%s)", strconv.Quote(n.Name)), "TyaValue", nil
		}
		// v0.44 within-package fallback: inside a module class
		// method, an unqualified PascalCase identifier that names a
		// sibling module class resolves to the module-prefixed C
		// symbol. See checker.currentModulePrefix for the matching
		// rule on the type-check side.
		if mod := g.currentModulePrefix(); mod != "" && classNameRE.MatchString(n.Name) {
			key := mod + "_" + n.Name
			if _, found := g.moduleClasses[key]; found {
				return cName(key + "_class"), "TyaValue", nil
			}
		}
		return cName(n.Name), "TyaValue", nil
	case *ast.SelfExpr:
		if n.Class {
			// v0.46 G2: `Self` resolves to the lexically enclosing
			// class via classTarget(), which already handles both
			// instance-method (`tya_member(__this, "class")`) and
			// class-method (`__this`) contexts.
			return g.classTarget(), "TyaValue", nil
		}
		return "__this", "TyaValue", nil
	case *ast.InstanceFieldExpr:
		return fmt.Sprintf("tya_member(__this, %s)", strconv.Quote("@"+n.Name)), "TyaValue", nil
	case *ast.ClassVarExpr:
		return fmt.Sprintf("tya_member(%s, %s)", g.classTarget(), strconv.Quote(n.Name)), "TyaValue", nil
	case *ast.BinaryExpr:
		left, _, err := g.expr(n.Left)
		if err != nil {
			return "", "", err
		}
		op := n.Op.Lexeme
		if op == "and" {
			right, _, err := g.expr(n.Right)
			if err != nil {
				return "", "", err
			}
			tmp := fmt.Sprintf("__logic%d", g.temp)
			g.temp++
			return fmt.Sprintf("({ TyaValue %s = %s; tya_truthy(%s) ? tya_bool(tya_truthy(%s)) : tya_bool(false); })", tmp, left, tmp, right), "TyaValue", nil
		}
		if op == "or" {
			right, _, err := g.expr(n.Right)
			if err != nil {
				return "", "", err
			}
			tmp := fmt.Sprintf("__logic%d", g.temp)
			g.temp++
			return fmt.Sprintf("({ TyaValue %s = %s; tya_truthy(%s) ? tya_bool(true) : tya_bool(tya_truthy(%s)); })", tmp, left, tmp, right), "TyaValue", nil
		}
		if op == "??" {
			right, _, err := g.expr(n.Right)
			if err != nil {
				return "", "", err
			}
			tmp := fmt.Sprintf("__nil_coalesce%d", g.temp)
			g.temp++
			return fmt.Sprintf("({ TyaValue %s = %s; %s.kind == TYA_NIL ? %s : %s; })", tmp, left, tmp, right, tmp), "TyaValue", nil
		}
		right, _, err := g.expr(n.Right)
		if err != nil {
			return "", "", err
		}
		typ := "TyaValue"
		expr := fmt.Sprintf("(%s.number %s %s.number)", left, op, right)
		switch op {
		case "+":
			expr = fmt.Sprintf("tya_add(%s, %s)", left, right)
		case "-":
			expr = fmt.Sprintf("tya_sub(%s, %s)", left, right)
		case "*":
			expr = fmt.Sprintf("tya_mul(%s, %s)", left, right)
		case "==":
			expr = fmt.Sprintf("tya_bool(tya_equal(%s, %s))", left, right)
		case "!=":
			expr = fmt.Sprintf("tya_bool(!tya_equal(%s, %s))", left, right)
		case "&&":
			expr = fmt.Sprintf("tya_bool(tya_truthy(%s) && tya_truthy(%s))", left, right)
		case "||":
			expr = fmt.Sprintf("tya_bool(tya_truthy(%s) || tya_truthy(%s))", left, right)
		case "%":
			expr = fmt.Sprintf("tya_mod(%s, %s)", left, right)
		case "/":
			expr = fmt.Sprintf("tya_div(%s, %s)", left, right)
		case "&":
			expr = fmt.Sprintf("tya_number((long)%s.number & (long)%s.number)", left, right)
		case "|":
			expr = fmt.Sprintf("tya_number((long)%s.number | (long)%s.number)", left, right)
		case "^":
			expr = fmt.Sprintf("tya_number((long)%s.number ^ (long)%s.number)", left, right)
		case "<<":
			expr = fmt.Sprintf("tya_number((long)%s.number << (long)%s.number)", left, right)
		case ">>":
			expr = fmt.Sprintf("tya_number((long)%s.number >> (long)%s.number)", left, right)
		case "<", "<=", ">", ">=":
			expr = fmt.Sprintf("tya_bool(tya_order_compare(%s, %s).number %s 0)", left, right, op)
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
		if n.Op.Lexeme == "~" {
			return "tya_number(~(long)" + ex + ".number)", "TyaValue", nil
		}
		return "({ TyaValue __neg = " + ex + "; TyaValue __out = tya_number(-__neg.number); __out.number_is_int = __neg.number_is_int; __out; })", typ, nil
	case *ast.CallExpr:
		if _, ok := n.Callee.(*ast.SuperExpr); ok {
			sym := g.inheritedMethodSym(g.superClass, g.methodName)
			if g.inClassMethod {
				sym = g.inheritedClassMethodSym(g.superClass, g.methodName)
			}
			if !g.inClassMethod && (g.methodName == "init" || g.methodName == "_init" || g.methodName == "initialize") && g.superClass != "" {
				if initSym, err := g.constructorSuperRunnerForCurrentClass(sym); err != nil {
					return "", "", err
				} else {
					sym = initSym
				}
			}
			if sym == "" && !g.inClassMethod {
				if (g.methodName == "init" || g.methodName == "_init" || g.methodName == "initialize") && g.superClass == "" {
					if initSym, err := g.interfaceInitializerRunnerForCurrentClass(); err != nil {
						return "", "", err
					} else {
						sym = initSym
					}
				}
			}
			if sym == "" && !g.inClassMethod {
				if ifaceSym, err := g.interfaceDefaultSymForCurrentClass(g.methodName); err != nil {
					return "", "", err
				} else {
					sym = ifaceSym
				}
			}
			if sym == "" && g.interfaceSuperSym != "" {
				sym = g.interfaceSuperSym
			}
			if sym == "" {
				return "tya_nil()", "TyaValue", nil
			}
			params := []string(nil)
			if g.inClassMethod {
				params = g.methodParams[g.superClass+".class."+g.methodName]
			} else if g.methodName == "init" || g.methodName == "_init" || g.methodName == "initialize" {
				params = g.classParams[g.superClass]
			} else {
				params = g.methodParams[g.superClass+"."+g.methodName]
			}
			args, err := g.callArgExprs(n, params)
			if err != nil {
				return "", "", err
			}
			for len(args) < 6 {
				args = append(args, "tya_missing()")
			}
			return fmt.Sprintf("%s(__this, %s)", sym, strings.Join(args[:6], ", ")), "TyaValue", nil
		}
		if id, ok := n.Callee.(*ast.Ident); ok && (n.ImplicitSelf || n.ImplicitClass) {
			receiver := "__this"
			if n.ImplicitClass {
				receiver = g.classTarget()
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
				return fmt.Sprintf("tya_call0(tya_member(%s, %s))", receiver, strconv.Quote(id.Name)), "TyaValue", nil
			case 1:
				return fmt.Sprintf("tya_call1(tya_member(%s, %s), %s)", receiver, strconv.Quote(id.Name), args[0]), "TyaValue", nil
			case 2:
				return fmt.Sprintf("tya_call2(tya_member(%s, %s), %s, %s)", receiver, strconv.Quote(id.Name), args[0], args[1]), "TyaValue", nil
			case 3:
				return fmt.Sprintf("tya_call3(tya_member(%s, %s), %s, %s, %s)", receiver, strconv.Quote(id.Name), args[0], args[1], args[2]), "TyaValue", nil
			case 4:
				return fmt.Sprintf("tya_call4(tya_member(%s, %s), %s, %s, %s, %s)", receiver, strconv.Quote(id.Name), args[0], args[1], args[2], args[3]), "TyaValue", nil
			case 5:
				return fmt.Sprintf("tya_call5(tya_member(%s, %s), %s, %s, %s, %s, %s)", receiver, strconv.Quote(id.Name), args[0], args[1], args[2], args[3], args[4]), "TyaValue", nil
			case 6:
				return fmt.Sprintf("tya_call6(tya_member(%s, %s), %s, %s, %s, %s, %s, %s)", receiver, strconv.Quote(id.Name), args[0], args[1], args[2], args[3], args[4], args[5]), "TyaValue", nil
			}
			return "tya_nil()", "TyaValue", nil
		}
		id, ok := n.Callee.(*ast.Ident)
		if ok {
			if expr, handled, err := g.sideEffectCallExpr(id.Name, n.Args); handled {
				return expr, "TyaValue", err
			}
			if removedPrimitiveHelperNames[id.Name] {
				return "", "", fmt.Errorf("top-level builtin %s was removed; use receiver method syntax", id.Name)
			}
			classKey := id.Name
			sym, found := g.classes[classKey]
			if !found {
				// v0.44 within-package fallback: inside a module
				// class, resolve unqualified PascalCase calls to
				// `<currentModule>_<Name>` constructor.
				if mod := g.currentModulePrefix(); mod != "" && classNameRE.MatchString(id.Name) {
					classKey = mod + "_" + id.Name
					sym, found = g.classes[classKey]
				}
			}
			if found {
				args, err := g.callArgExprs(n, g.classParams[classKey])
				if err != nil {
					return "", "", err
				}
				for len(args) < 6 {
					args = append(args, "tya_missing()")
				}
				return fmt.Sprintf("%s(tya_nil(), %s)", sym, strings.Join(args[:6], ", ")), "TyaValue", nil
			}
			if decl, found := g.structDecls[id.Name]; found {
				ex, err := g.structConstructorExpr(id.Name, decl, n)
				return ex, "TyaValue", err
			}
		}
		if ok && id.Name == "len" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_len(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "map" && len(n.Args) == 2 {
			array, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			fn, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_map(%s, %s)", array, fn), "TyaValue", nil
		}
		if ok && id.Name == "filter" && len(n.Args) == 2 {
			array, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			fn, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_filter(%s, %s)", array, fn), "TyaValue", nil
		}
		if ok && id.Name == "find" && len(n.Args) == 2 {
			array, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			fn, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_find(%s, %s)", array, fn), "TyaValue", nil
		}
		if ok && id.Name == "any" && len(n.Args) == 2 {
			array, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			fn, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_any(%s, %s)", array, fn), "TyaValue", nil
		}
		if ok && id.Name == "all" && len(n.Args) == 2 {
			array, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			fn, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_all(%s, %s)", array, fn), "TyaValue", nil
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
		if ok && id.Name == "read_line" && len(n.Args) == 0 {
			return "tya_read_line()", "TyaValue", nil
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
		if ok && id.Name == "index_of" && (len(n.Args) == 2 || len(n.Args) == 3) {
			text, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			needle, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			start := "tya_number(0)"
			if len(n.Args) == 3 {
				start, _, err = g.expr(n.Args[2])
				if err != nil {
					return "", "", err
				}
			}
			return fmt.Sprintf("tya_string_index_of(%s, %s, %s)", text, needle, start), "TyaValue", nil
		}
		if ok && id.Name == "starts_with" && len(n.Args) == 2 {
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
		if ok && id.Name == "ends_with" && len(n.Args) == 2 {
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
		if ok && id.Name == "args" && len(n.Args) == 0 {
			return "tya_args(g_tya_argc, g_tya_argv)", "TyaValue", nil
		}
		if ok && id.Name == "env" && len(n.Args) == 1 {
			name, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_env(%s)", name), "TyaValue", nil
		}
		if ok && id.Name == "read_file" && len(n.Args) == 1 {
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
		if ok && id.Name == "error" && len(n.Args) == 2 {
			message, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			options, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_error2(%s, %s)", message, options), "TyaValue", nil
		}
		if ok && id.Name == "panic" && len(n.Args) == 1 {
			message, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("(tya_panic(%s), tya_nil())", message), "TyaValue", nil
		}
		if ok && id.Name == "exit" && len(n.Args) == 1 {
			code, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("(tya_exit(%s), tya_nil())", code), "TyaValue", nil
		}
		if ok && id.Name == "write_file" && len(n.Args) == 2 {
			path, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			text, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("(tya_write_file(%s, %s), tya_nil())", path, text), "TyaValue", nil
		}
		if ok && id.Name == "dir_list" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_dir_list(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "dir_mkdir" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_dir_mkdir(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "dir_rmdir" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_dir_rmdir(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "file_remove" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_file_remove(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "file_rename" && len(n.Args) == 2 {
			oldArg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			newArg, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_file_rename(%s, %s)", oldArg, newArg), "TyaValue", nil
		}
		if ok && id.Name == "file_stat" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_file_stat(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "path_expand_user" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_path_expand_user(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "cwd" && len(n.Args) == 0 {
			return "tya_cwd()", "TyaValue", nil
		}
		if ok && id.Name == "chdir" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_chdir(%s)", arg), "TyaValue", nil
		}
		if ok {
			if call := v24Codegen(g, id.Name, n.Args); call != "" {
				return call, "TyaValue", nil
			}
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
		if ok && id.Name == "to_string" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_to_string(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "inspect" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_inspect(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "to_int" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_to_int(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "byte_len" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_byte_len(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "char_len" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_len(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "ord" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_ord(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "kind" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_kind(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "chr" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_chr(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "to_float" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_to_float(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "to_number" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_to_number(%s)", arg), "TyaValue", nil
		}
		if ok && id.Name == "file_exists" && len(n.Args) == 1 {
			path, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_file_exists(%s)", path), "TyaValue", nil
		}
		if ok && id.Name == "has" && len(n.Args) == 2 {
			dict, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			key, _, err := g.expr(n.Args[1])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_has(%s, %s)", dict, key), "TyaValue", nil
		}
		if ok && id.Name == "keys" && len(n.Args) == 1 {
			dict, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_keys(%s)", dict), "TyaValue", nil
		}
		if ok && id.Name == "values" && len(n.Args) == 1 {
			dict, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_values(%s)", dict), "TyaValue", nil
		}
		if ok && id.Name == "pop" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_pop(%s)", arg), "TyaValue", nil
		}
		if ok {
			if sym, found := g.funcs[id.Name]; found {
				args, err := g.callArgExprs(n, g.funcParams[id.Name])
				if err != nil {
					return "", "", err
				}
				for len(args) < 6 {
					args = append(args, "tya_missing()")
				}
				return fmt.Sprintf("%s(tya_nil(), %s)", sym, strings.Join(args[:6], ", ")), "TyaValue", nil
			}
			if fn, found := nativeFunctions[id.Name]; found {
				if len(n.Args) != fn.Arity {
					return "", "", fmt.Errorf("native function %s expects %d arguments", id.Name, fn.Arity)
				}
				args := make([]string, 0, len(n.Args))
				for _, arg := range n.Args {
					ex, _, err := g.expr(arg)
					if err != nil {
						return "", "", err
					}
					args = append(args, ex)
				}
				for len(args) < 4 {
					args = append(args, "tya_nil()")
				}
				return fmt.Sprintf("%s(tya_nil(), %s)", fn.Symbol, strings.Join(args[:4], ", ")), "TyaValue", nil
			}
		}
		if member, ok := n.Callee.(*ast.MemberExpr); ok {
			if call, err := g.concurrencyConstructorCall(member, n.Args); call != "" || err != nil {
				return call, "TyaValue", err
			}
			if key := packageMemberStructKey(member); key != "" {
				if decl, found := g.structDecls[key]; found {
					ex, err := g.structConstructorExpr(key, decl, n)
					return ex, "TyaValue", err
				}
			}
			if target, ok := member.Target.(*ast.Ident); ok {
				if removedPrimitiveModuleNames[target.Name] && !g.vars[target.Name] && g.funcs[target.Name] == "" && g.classes[target.Name] == "" {
					return "", "", fmt.Errorf("primitive helper %s.%s was removed; use receiver method syntax", target.Name, member.Name)
				}
			}
			receiver, _, err := g.expr(member.Target)
			if err != nil {
				return "", "", err
			}
			if !n.PositionalArgsOnly() {
				positional, keywords, err := g.dynamicKeywordArgs(n)
				if err != nil {
					return "", "", err
				}
				return fmt.Sprintf("tya_call_keywords(tya_member(%s, %s), (TyaValue[]){%s}, %d, %s)", receiver, strconv.Quote(member.Name), strings.Join(positional, ", "), len(positional), keywords), "TyaValue", nil
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
				return fmt.Sprintf("tya_call0(tya_member(%s, %s))", receiver, strconv.Quote(member.Name)), "TyaValue", nil
			case 1:
				return fmt.Sprintf("tya_call1(tya_member(%s, %s), %s)", receiver, strconv.Quote(member.Name), args[0]), "TyaValue", nil
			case 2:
				return fmt.Sprintf("tya_call2(tya_member(%s, %s), %s, %s)", receiver, strconv.Quote(member.Name), args[0], args[1]), "TyaValue", nil
			case 3:
				return fmt.Sprintf("tya_call3(tya_member(%s, %s), %s, %s, %s)", receiver, strconv.Quote(member.Name), args[0], args[1], args[2]), "TyaValue", nil
			case 4:
				return fmt.Sprintf("tya_call4(tya_member(%s, %s), %s, %s, %s, %s)", receiver, strconv.Quote(member.Name), args[0], args[1], args[2], args[3]), "TyaValue", nil
			case 5:
				return fmt.Sprintf("tya_call5(tya_member(%s, %s), %s, %s, %s, %s, %s)", receiver, strconv.Quote(member.Name), args[0], args[1], args[2], args[3], args[4]), "TyaValue", nil
			case 6:
				return fmt.Sprintf("tya_call6(tya_member(%s, %s), %s, %s, %s, %s, %s, %s)", receiver, strconv.Quote(member.Name), args[0], args[1], args[2], args[3], args[4], args[5]), "TyaValue", nil
			}
		}
		callee, _, err := g.expr(n.Callee)
		if err != nil {
			return "", "", err
		}
		if !n.PositionalArgsOnly() {
			positional, keywords, err := g.dynamicKeywordArgs(n)
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_call_keywords(%s, (TyaValue[]){%s}, %d, %s)", callee, strings.Join(positional, ", "), len(positional), keywords), "TyaValue", nil
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
			return fmt.Sprintf("tya_call0(%s)", callee), "TyaValue", nil
		case 1:
			return fmt.Sprintf("tya_call1(%s, %s)", callee, args[0]), "TyaValue", nil
		case 2:
			return fmt.Sprintf("tya_call2(%s, %s, %s)", callee, args[0], args[1]), "TyaValue", nil
		case 3:
			return fmt.Sprintf("tya_call3(%s, %s, %s, %s)", callee, args[0], args[1], args[2]), "TyaValue", nil
		case 4:
			return fmt.Sprintf("tya_call4(%s, %s, %s, %s, %s)", callee, args[0], args[1], args[2], args[3]), "TyaValue", nil
		case 5:
			return fmt.Sprintf("tya_call5(%s, %s, %s, %s, %s, %s)", callee, args[0], args[1], args[2], args[3], args[4]), "TyaValue", nil
		case 6:
			return fmt.Sprintf("tya_call6(%s, %s, %s, %s, %s, %s, %s)", callee, args[0], args[1], args[2], args[3], args[4], args[5]), "TyaValue", nil
		}
		return "tya_nil()", "TyaValue", nil
	case *ast.IndexExpr:
		dict, _, err := g.expr(n.Target)
		if err != nil {
			return "", "", err
		}
		index, _, err := g.expr(n.Index)
		if err != nil {
			return "", "", err
		}
		return fmt.Sprintf("tya_index(%s, %s)", dict, index), "TyaValue", nil
	case *ast.MemberExpr:
		// v0.46/47 G2: `self.x` reads the "@x" key (where instance
		// fields canonical-live) so that an instance field is not
		// shadowed by a method of the same name on the bare key.
		// This mirrors the InstanceFieldExpr read at line 1566 and
		// keeps `self.x` and `@x` byte-identical at the codegen level.
		if selfTarget, ok := n.Target.(*ast.SelfExpr); ok && !selfTarget.Class {
			return fmt.Sprintf("tya_member(__this, %s)", strconv.Quote("@"+n.Name)), "TyaValue", nil
		}
		dict, _, err := g.expr(n.Target)
		if err != nil {
			return "", "", err
		}
		if n.Name == "class" {
			return fmt.Sprintf("tya_class_of(%s)", dict), "TyaValue", nil
		}
		return fmt.Sprintf("tya_member(%s, %s)", dict, strconv.Quote(n.Name)), "TyaValue", nil
	case *ast.TryExpr:
		return g.expr(n.Expr)
	case *ast.IfStmt:
		return g.ifExpr(n)
	case *ast.WhileStmt:
		return g.whileExpr(n)
	case *ast.ForInStmt:
		return g.forExpr(n)
	case *ast.MatchStmt:
		return g.matchExpr(n)
	case *ast.SpawnExpr:
		// spawn fn(args) takes the args evaluated in this thread, then
		// runs fn(args...) on a new pthread. Decompose the CallExpr to
		// pass callee and args separately to tya_task_new.
		if call, ok := n.Callee.(*ast.CallExpr); ok {
			callee, _, err := g.expr(call.Callee)
			if err != nil {
				return "", "", err
			}
			if len(call.Args) > 4 {
				return "", "", fmt.Errorf("spawn: at most 4 arguments are supported, got %d", len(call.Args))
			}
			argv := make([]string, 4)
			for i := range argv {
				argv[i] = "tya_nil()"
			}
			for i, arg := range call.Args {
				s, _, err := g.expr(arg)
				if err != nil {
					return "", "", err
				}
				argv[i] = s
			}
			return fmt.Sprintf("tya_task_new(%s, %d, %s, %s, %s, %s)",
				callee, len(call.Args), argv[0], argv[1], argv[2], argv[3]), "TyaValue", nil
		}
		callee, _, err := g.expr(n.Callee)
		if err != nil {
			return "", "", err
		}
		return fmt.Sprintf("tya_task_new(%s, 0, tya_nil(), tya_nil(), tya_nil(), tya_nil())", callee), "TyaValue", nil
	case *ast.AwaitExpr:
		target, _, err := g.expr(n.Target)
		if err != nil {
			return "", "", err
		}
		return fmt.Sprintf("tya_task_await(%s)", target), "TyaValue", nil
	}
	line, col, _ := exprPos(expr)
	return "", "", codegenError(codeStmtUnsupported, fmt.Sprintf("C emitter does not support expression %T", expr), line, col)
}

func packageMemberStructKey(member *ast.MemberExpr) string {
	parts := []string{member.Name}
	for target := member.Target; ; {
		switch n := target.(type) {
		case *ast.Ident:
			if len(parts) == 1 {
				return aliasedPackageSymbolKey(n.Name, parts[0])
			}
			return strings.Join(append([]string{n.Name}, parts...), ".")
		case *ast.MemberExpr:
			parts = append([]string{n.Name}, parts...)
			target = n.Target
		default:
			return ""
		}
	}
}

func (g *cgen) concurrencyConstructorCall(member *ast.MemberExpr, argExprs []ast.Expr) (string, error) {
	target, ok := member.Target.(*ast.Ident)
	if !ok {
		return "", nil
	}
	args := make([]string, 0, len(argExprs))
	for _, arg := range argExprs {
		ex, _, err := g.expr(arg)
		if err != nil {
			return "", err
		}
		args = append(args, ex)
	}
	switch target.Name + "." + member.Name {
	case "channel.Channel":
		if len(args) == 1 {
			return fmt.Sprintf("tya_channel_new(%s)", args[0]), nil
		}
	case "sync.Mutex":
		if len(args) == 0 {
			return "tya_sync_mutex_new()", nil
		}
	case "sync.AtomicInteger":
		if len(args) == 1 {
			return fmt.Sprintf("tya_sync_atomic_integer_new(%s)", args[0]), nil
		}
	case "sync.WaitGroup":
		if len(args) == 0 {
			return "tya_sync_wait_group_new()", nil
		}
	}
	return "", nil
}

func (g *cgen) emitDynamicCall(callee string, args []string) string {
	switch len(args) {
	case 0:
		return fmt.Sprintf("tya_call0(%s)", callee)
	case 1:
		return fmt.Sprintf("tya_call1(%s, %s)", callee, args[0])
	case 2:
		return fmt.Sprintf("tya_call2(%s, %s, %s)", callee, args[0], args[1])
	case 3:
		return fmt.Sprintf("tya_call3(%s, %s, %s, %s)", callee, args[0], args[1], args[2])
	case 4:
		return fmt.Sprintf("tya_call4(%s, %s, %s, %s, %s)", callee, args[0], args[1], args[2], args[3])
	case 5:
		return fmt.Sprintf("tya_call5(%s, %s, %s, %s, %s, %s)", callee, args[0], args[1], args[2], args[3], args[4])
	default:
		return fmt.Sprintf("tya_call6(%s, %s, %s, %s, %s, %s, %s)", callee, args[0], args[1], args[2], args[3], args[4], args[5])
	}
}

func (g *cgen) callArgExprs(call *ast.CallExpr, params []string) ([]string, error) {
	if call.PositionalArgsOnly() || len(params) == 0 {
		args := make([]string, 0, len(call.Args))
		for _, arg := range call.Args {
			ex, _, err := g.expr(arg)
			if err != nil {
				return nil, err
			}
			args = append(args, ex)
		}
		return args, nil
	}
	paramIndex := map[string]int{}
	for i, name := range params {
		paramIndex[name] = i
	}
	values := make([]string, len(params))
	filled := make([]bool, len(params))
	pos := 0
	expansions := []string{}
	for _, arg := range call.EffectiveArgs() {
		if arg.Expand {
			ex, _, err := g.expr(arg.Value)
			if err != nil {
				return nil, err
			}
			tmp := fmt.Sprintf("__kw%d", g.temp)
			g.temp++
			ex = fmt.Sprintf("({ TyaValue %s = %s; if (%s.kind != TYA_DICT) { tya_panic(tya_string(\"keyword expansion expects dictionary\")); } %s; })", tmp, ex, tmp, tmp)
			expansions = append(expansions, ex)
			continue
		}
		ex, _, err := g.expr(arg.Value)
		if err != nil {
			return nil, err
		}
		if arg.Name == "" {
			if pos >= len(params) {
				return nil, fmt.Errorf("too many positional arguments")
			}
			values[pos] = ex
			filled[pos] = true
			pos++
			continue
		}
		index, ok := paramIndex[arg.Name]
		if !ok {
			return nil, fmt.Errorf("unknown keyword %s", arg.Name)
		}
		if filled[index] {
			return nil, fmt.Errorf("argument %s supplied multiple times", arg.Name)
		}
		values[index] = ex
		filled[index] = true
	}
	last := -1
	for i := range params {
		if filled[i] || len(expansions) > 0 {
			last = i
		}
	}
	if last < 0 {
		return nil, nil
	}
	out := make([]string, last+1)
	for i := 0; i <= last; i++ {
		if filled[i] {
			out[i] = values[i]
			continue
		}
		ex := "tya_missing()"
		for j := len(expansions) - 1; j >= 0; j-- {
			ex = fmt.Sprintf("tya_dict_get(%s, tya_string(%s), %s, true)", expansions[j], strconv.Quote(params[i]), ex)
		}
		out[i] = ex
	}
	return out, nil
}

func (g *cgen) dynamicKeywordArgs(call *ast.CallExpr) ([]string, string, error) {
	positional := []string{}
	lines := []string{}
	kw := fmt.Sprintf("__kw%d", g.temp)
	g.temp++
	lines = append(lines, fmt.Sprintf("TyaValue %s = tya_dict(0, 0);", kw))
	for _, arg := range call.EffectiveArgs() {
		ex, _, err := g.expr(arg.Value)
		if err != nil {
			return nil, "", err
		}
		if arg.Expand {
			tmp := fmt.Sprintf("__kw_exp%d", g.temp)
			g.temp++
			lines = append(lines, fmt.Sprintf("TyaValue %s = %s;", tmp, ex))
			lines = append(lines, fmt.Sprintf("tya_keywords_merge(%s, %s);", kw, tmp))
			continue
		}
		if arg.Name == "" {
			positional = append(positional, ex)
			continue
		}
		lines = append(lines, fmt.Sprintf("tya_set_index(%s, tya_string(%s), %s);", kw, strconv.Quote(arg.Name), ex))
	}
	lines = append(lines, kw)
	return positional, "({ " + strings.Join(lines, " ") + "; })", nil
}

func (g *cgen) selectStmt(n *ast.SelectStmt) error {
	ops := make([]string, 0, len(n.Arms))
	for _, arm := range n.Arms {
		switch arm.Kind {
		case "receive":
			ch, _, err := g.expr(arm.Channel)
			if err != nil {
				return err
			}
			ops = append(ops, fmt.Sprintf("tya_array((TyaValue[]){%s, tya_string(\"receive\")}, 2)", ch))
		case "send":
			ch, _, err := g.expr(arm.Channel)
			if err != nil {
				return err
			}
			value, _, err := g.expr(arm.Value)
			if err != nil {
				return err
			}
			ops = append(ops, fmt.Sprintf("tya_array((TyaValue[]){%s, tya_string(\"send\"), %s}, 3)", ch, value))
		case "timeout":
			seconds, _, err := g.expr(arm.Seconds)
			if err != nil {
				return err
			}
			ops = append(ops, fmt.Sprintf("tya_array((TyaValue[]){tya_nil(), tya_string(\"timeout\"), %s}, 3)", seconds))
		case "default":
			ops = append(ops, "tya_array((TyaValue[]){tya_nil(), tya_string(\"default\")}, 2)")
		}
	}
	id := g.temp
	g.temp++
	opsName := fmt.Sprintf("__select_ops%d", id)
	resultName := fmt.Sprintf("__select_result%d", id)
	indexName := fmt.Sprintf("__select_index%d", id)
	g.line("{")
	g.indent++
	g.line(fmt.Sprintf("TyaValue %s = tya_array((TyaValue[]){%s}, %d);", opsName, strings.Join(ops, ", "), len(ops)))
	g.line(fmt.Sprintf("TyaValue %s = tya_channel_select(%s);", resultName, opsName))
	g.line(fmt.Sprintf("TyaValue %s = tya_index(%s, tya_string(\"index\"));", indexName, resultName))
	for i, arm := range n.Arms {
		if i == 0 {
			g.line(fmt.Sprintf("if ((int)%s.number == %d) {", indexName, i))
		} else {
			g.line(fmt.Sprintf("else if ((int)%s.number == %d) {", indexName, i))
		}
		g.indent++
		if arm.Kind == "receive" && arm.BindName != "" {
			g.line(fmt.Sprintf("TyaValue %s = tya_index(%s, tya_string(\"value\"));", cName(arm.BindName), resultName))
		}
		for _, st := range arm.Body {
			if err := g.stmt(st); err != nil {
				return err
			}
		}
		g.indent--
		g.line("}")
	}
	g.indent--
	g.line("}")
	return nil
}
