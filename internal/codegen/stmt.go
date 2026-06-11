package codegen

import (
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"tya/internal/ast"
	"tya/internal/token"
)

func (g *cgen) stmt(stmt ast.Stmt) error {
	g.instrument(stmt)
	switch n := stmt.(type) {
	case *ast.EmbedStmt:
		g.sourceLine(n.NameTok.Line)
		return g.emitEmbed(n)
	case *ast.ImportStmt:
		g.emitImportStmt(n)
		return nil
	case *ast.ImportBlockStmt:
		for _, imp := range n.Imports {
			g.emitImportStmt(imp)
		}
		return nil
	case *ast.ModuleDecl:
		g.sourceLine(n.NameTok.Line)
		return g.assignModuleDecl(n)
	case *ast.ClassDecl:
		g.sourceLine(n.NameTok.Line)
		return g.assignClassDecl(n.Name, n)
	case *ast.StructDecl:
		g.sourceLine(n.NameTok.Line)
		return g.assignStructDecl(n.Name, n)
	case *ast.InterfaceDecl:
		g.sourceLine(n.NameTok.Line)
		return g.assignInterfaceDecl(n.Name)
	case *ast.AssignStmt:
		g.sourceLine(n.Tok.Line)
		if n.Tok.Type == token.NIL_ASSIGN {
			return g.nilAssign(n)
		}
		if len(n.Targets) != 1 || len(n.Values) != 1 {
			return g.multiAssign(n)
		}
		if isDestructuringTarget(n.Targets[0]) {
			return g.assignDestructuring(n.Targets[0], n.Values[0])
		}
		if target, ok := n.Targets[0].(*ast.IndexExpr); ok {
			dict, _, err := g.expr(target.Target)
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
			g.line(fmt.Sprintf("tya_set_index(%s, %s, %s);", dict, index, value))
			return nil
		}
		if target, ok := n.Targets[0].(*ast.MemberExpr); ok {
			// v0.46 G2: route `self.x` and `Self.x` assignments
			// through the same lowering as `@x` / `@@x` so the v0.46
			// surface and the v0.45 surface share the same runtime
			// key layout (instance fields stored under both "@x" and
			// "x"; class members under "x" on the class object).
			if selfTarget, ok := target.Target.(*ast.SelfExpr); ok {
				if selfTarget.Class {
					// Self.x = ... → equivalent to ClassVarExpr assignment.
					value, _, err := g.expr(n.Values[0])
					if err != nil {
						return err
					}
					receiver := g.classTarget()
					g.line(fmt.Sprintf("tya_set_member(%s, %s, %s);", receiver, strconv.Quote(target.Name), value))
					return nil
				}
				// self.x = ... → equivalent to InstanceFieldExpr assignment.
				value, _, err := g.expr(n.Values[0])
				if err != nil {
					return err
				}
				tmp := fmt.Sprintf("__field%d", g.temp)
				g.temp++
				g.line(fmt.Sprintf("TyaValue %s = %s;", tmp, value))
				g.line(fmt.Sprintf("tya_set_member(__this, %s, %s);", strconv.Quote("@"+target.Name), tmp))
				g.line(fmt.Sprintf("tya_set_member(__this, %s, %s);", strconv.Quote(target.Name), tmp))
				return nil
			}
			receiver, _, err := g.expr(target.Target)
			if err != nil {
				return err
			}
			value, _, err := g.expr(n.Values[0])
			if err != nil {
				return err
			}
			g.line(fmt.Sprintf("tya_set_member(%s, %s, %s);", receiver, strconv.Quote(target.Name), value))
			return nil
		}
		if target, ok := n.Targets[0].(*ast.InstanceFieldExpr); ok {
			value, _, err := g.expr(n.Values[0])
			if err != nil {
				return err
			}
			tmp := fmt.Sprintf("__field%d", g.temp)
			g.temp++
			g.line(fmt.Sprintf("TyaValue %s = %s;", tmp, value))
			g.line(fmt.Sprintf("tya_set_member(__this, %s, %s);", strconv.Quote("@"+target.Name), tmp))
			g.line(fmt.Sprintf("tya_set_member(__this, %s, %s);", strconv.Quote(target.Name), tmp))
			return nil
		}
		if target, ok := n.Targets[0].(*ast.ClassVarExpr); ok {
			value, _, err := g.expr(n.Values[0])
			if err != nil {
				return err
			}
			g.line(fmt.Sprintf("tya_set_member(%s, %s, %s);", g.classTarget(), strconv.Quote(target.Name), value))
			return nil
		}
		id, ok := n.Targets[0].(*ast.Ident)
		if !ok {
			line, col, _ := stmtPos(n)
			return codegenError(codeAssignTargetUnsupported, "C emitter only supports variable assignment", line, col)
		}
		if tryExpr, ok := n.Values[0].(*ast.TryExpr); ok {
			return g.assignTry(id.Name, tryExpr)
		}
		if obj, ok := n.Values[0].(*ast.DictLit); ok {
			if err := g.assignDictLit(id.Name, obj); err != nil {
				return err
			}
			return nil
		}
		if fn, ok := n.Values[0].(*ast.FuncLit); ok {
			captures := g.freeVars(fn)
			sym, err := g.emitFuncWithCaptures(id.Name, fn, "", "", captures)
			if err != nil {
				return err
			}
			if len(captures) == 0 {
				g.funcs[id.Name] = sym
				g.funcParams[id.Name] = append([]string(nil), fn.Params...)
				g.line(fmt.Sprintf("%s = tya_function_params(%s, %s);", cName(id.Name), sym, cParamArray(fn.Params)))
				return nil
			}
			env := fmt.Sprintf("__env%d", g.temp)
			g.temp++
			g.line(fmt.Sprintf("TyaValue %s = tya_object();", env))
			for _, name := range sortedMapKeys(captures) {
				g.line(fmt.Sprintf("tya_set_member(%s, %s, %s);", env, strconv.Quote(name), g.captureValue(name)))
			}
			g.line(fmt.Sprintf("%s = tya_bind_method_params(%s, %s, %s);", cName(id.Name), env, sym, cParamArray(fn.Params)))
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
	case *ast.RaiseStmt:
		value, _, err := g.expr(n.Value)
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_raise_user(%s);", value))
	case *ast.TryCatchStmt:
		return g.tryCatchStmt(n)
	case *ast.MatchStmt:
		return g.matchStmt(n)
	case *ast.ScopeBlock:
		// scope wraps its body in setjmp so a synchronous raise from
		// the body still runs tya_scope_raise (which cancels siblings,
		// joins them, then re-raises the body's value).
		scopeName := fmt.Sprintf("__scope%d", g.temp)
		frameName := fmt.Sprintf("__scope_frame%d", g.temp)
		g.temp++
		g.line(fmt.Sprintf("{ TyaScope %s; tya_scope_enter(&%s);", scopeName, scopeName))
		g.line(fmt.Sprintf("TyaRaiseFrame %s;", frameName))
		g.line(fmt.Sprintf("tya_push_raise_frame(&%s);", frameName))
		g.line(fmt.Sprintf("if (setjmp(%s.env) == 0) {", frameName))
		g.indent++
		g.raiseDepth++
		for _, st := range n.Body {
			if err := g.stmt(st); err != nil {
				return err
			}
		}
		g.raiseDepth--
		g.line("tya_pop_raise_frame();")
		g.line(fmt.Sprintf("tya_scope_exit(&%s);", scopeName))
		g.indent--
		g.line("} else {")
		g.indent++
		g.line(fmt.Sprintf("TyaValue __scope_raised = %s.value;", frameName))
		g.line("tya_pop_raise_frame();")
		g.line(fmt.Sprintf("tya_scope_raise(&%s, __scope_raised);", scopeName))
		g.indent--
		g.line("} }")
		return nil
	case *ast.SelectStmt:
		return g.selectStmt(n)
	case *ast.ForInStmt:
		return g.forInStmt(n, "")
	case *ast.ReturnStmt:
		if !g.inFunc {
			g.line("/* return */")
			return nil
		}
		if len(n.Values) == 0 {
			g.returnLine("tya_nil()")
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
			g.returnLine(fmt.Sprintf("tya_array((TyaValue[]){%s}, %d)", strings.Join(values, ", "), len(values)))
			return nil
		}
		value, _, err := g.expr(n.Values[0])
		if err != nil {
			return err
		}
		g.returnLine(value)
	default:
		line, col, _ := stmtPos(stmt)
		return codegenError(codeStmtUnsupported, fmt.Sprintf("C emitter does not support %T", stmt), line, col)
	}
	return nil
}

func (g *cgen) emitImportStmt(n *ast.ImportStmt) {
	g.sourceLine(n.NameTok.Line)
	if n.Alias != "" {
		target := cName(n.Alias)
		source := cName(n.ModuleName())
		if g.vars[n.Alias] {
			g.line(fmt.Sprintf("%s = %s;", target, source))
		} else {
			g.vars[n.Alias] = true
			g.line(fmt.Sprintf("TyaValue %s = %s;", target, source))
		}
	} else {
		g.line("/* import resolved by runner */")
	}
}

func (g *cgen) forInStmt(n *ast.ForInStmt, result string) error {
	iterable, _, err := g.expr(n.Iterable)
	if err != nil {
		return err
	}
	iterName := fmt.Sprintf("__iter%d", g.temp)
	indexName := fmt.Sprintf("__i%d", g.temp)
	genericName := fmt.Sprintf("__generic_iter%d", g.temp)
	g.temp++
	g.line(fmt.Sprintf("TyaValue %s = %s;", iterName, iterable))
	g.declareForInBindings(n)
	g.line(fmt.Sprintf("if (%s.kind == TYA_ARRAY || %s.kind == TYA_STRING || %s.kind == TYA_BYTES || %s.kind == TYA_DICT) {", iterName, iterName, iterName, iterName))
	g.indent++
	g.line(fmt.Sprintf("for (int %s = 0; %s < (int)tya_len(%s).number; %s++) {", indexName, indexName, iterName, indexName))
	g.indent++
	if err := g.forInBody(n, result, iterName, indexName, fmt.Sprintf("(%s.kind == TYA_DICT ? tya_dict_entry_at(%s, tya_number(%s)) : tya_index(%s, tya_number(%s)))", iterName, iterName, indexName, iterName, indexName)); err != nil {
		return err
	}
	g.indent--
	g.line("}")
	g.indent--
	g.line("} else {")
	g.indent++
	g.line(fmt.Sprintf("TyaValue %s = tya_iter(%s);", genericName, iterName))
	g.line(fmt.Sprintf("for (int %s = 0; tya_truthy(tya_iterator_has_next(%s)); %s++) {", indexName, genericName, indexName))
	g.indent++
	if err := g.forInBody(n, result, iterName, indexName, fmt.Sprintf("tya_iterator_next(%s)", genericName)); err != nil {
		return err
	}
	g.indent--
	g.line("}")
	g.indent--
	g.line("}")
	return nil
}

func (g *cgen) declareForInBindings(n *ast.ForInStmt) {
	if n.IndexName != "" && !g.vars[n.IndexName] {
		g.vars[n.IndexName] = true
		g.line(fmt.Sprintf("TyaValue %s = tya_nil();", cName(n.IndexName)))
	}
	if !g.vars[n.ValueName] {
		g.vars[n.ValueName] = true
		g.line(fmt.Sprintf("TyaValue %s = tya_nil();", cName(n.ValueName)))
	}
}

func (g *cgen) forInBody(n *ast.ForInStmt, result, iterName, indexName, valueExpr string) error {
	if n.IndexName != "" {
		g.line(fmt.Sprintf("%s = tya_number(%s);", cName(n.IndexName), indexName))
	}
	_ = iterName
	g.line(fmt.Sprintf("%s = %s;", cName(n.ValueName), valueExpr))
	if result != "" {
		return g.assignControlBodyValue(result, n.Body)
	}
	for _, stmt := range n.Body {
		if err := g.stmt(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (g *cgen) assignInterfaceDecl(name string) error {
	target := cName(name)
	if g.vars[name] {
		g.line(fmt.Sprintf("%s = tya_string(%s);", target, strconv.Quote("interface "+name)))
		return nil
	}
	g.vars[name] = true
	g.globalLine(fmt.Sprintf("TyaValue %s;", target))
	g.line(fmt.Sprintf("%s = tya_string(%s);", target, strconv.Quote("interface "+name)))
	return nil
}

func (g *cgen) assignStructDecl(name string, decl *ast.StructDecl) error {
	withSym := ""
	params := make([]string, len(decl.Fields))
	for i, field := range decl.Fields {
		params[i] = field.Name
	}
	if decl.Record {
		withSym = cFuncName(name+"_record_with", g.temp)
		g.temp++
		g.structWithSyms[name] = withSym
		var with strings.Builder
		with.WriteString("TyaValue ")
		with.WriteString(withSym)
		with.WriteString("(TyaValue __this, TyaValue __arg0, TyaValue __arg1, TyaValue __arg2, TyaValue __arg3, TyaValue __arg4, TyaValue __arg5) {\n")
		with.WriteString("  TyaValue __obj = tya_object();\n")
		with.WriteString(fmt.Sprintf("  tya_set_member(__obj, \"__data_type\", tya_string(%s));\n", strconv.Quote(name)))
		with.WriteString("  tya_set_member(__obj, \"__record\", tya_bool(true));\n")
		for i, field := range decl.Fields {
			arg := fmt.Sprintf("__arg%d", i)
			if i >= 6 {
				continue
			}
			with.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, %s.kind == TYA_MISSING ? tya_member(__this, %s) : %s);\n", strconv.Quote(field.Name), arg, strconv.Quote(field.Name), arg))
		}
		with.WriteString(fmt.Sprintf("  tya_set_member(__obj, \"with\", tya_bind_method_params(__obj, %s, %s));\n", withSym, cParamArray(params)))
		with.WriteString("  return __obj;\n")
		with.WriteString("}\n")
		g.funcOut.WriteString(with.String())
	}
	sym := cFuncName(name+"_struct_ctor", g.temp)
	g.temp++
	required := 0
	for _, field := range decl.Fields {
		if !field.HasDefault {
			required++
		}
	}
	var out strings.Builder
	out.WriteString("TyaValue ")
	out.WriteString(sym)
	out.WriteString("(TyaValue __this, TyaValue __arg0, TyaValue __arg1, TyaValue __arg2, TyaValue __arg3, TyaValue __arg4, TyaValue __arg5) {\n")
	child := &cgen{vars: map[string]bool{}, funcs: g.funcs, funcParams: g.funcParams, classes: g.classes, classParams: g.classParams, classMethods: g.classMethods, methodParams: g.methodParams, classDecls: g.classDecls, structDecls: g.structDecls, structWithSyms: g.structWithSyms, interfaceDecls: g.interfaceDecls, moduleClasses: g.moduleClasses, sourcePath: g.sourcePath, temp: g.temp, indent: 1, inFunc: true}
	child.line("(void)__this;")
	child.line("TyaValue __obj = tya_object();")
	child.line(fmt.Sprintf("tya_set_member(__obj, \"__data_type\", tya_string(%s));", strconv.Quote(name)))
	if decl.Record {
		child.line("tya_set_member(__obj, \"__record\", tya_bool(true));")
	} else {
		child.line("tya_set_member(__obj, \"__struct\", tya_bool(true));")
	}
	for i, field := range decl.Fields {
		arg := fmt.Sprintf("__arg%d", i)
		if i >= 6 {
			continue
		}
		if field.HasDefault {
			def, _, err := child.expr(field.Value)
			if err != nil {
				return err
			}
			child.line(fmt.Sprintf("if (%s.kind == TYA_MISSING) { %s = %s; }", arg, arg, def))
		} else {
			child.line(fmt.Sprintf("if (%s.kind == TYA_MISSING) { tya_panic(tya_string(%s)); }", arg, strconv.Quote("missing required argument "+field.Name)))
		}
		child.line(fmt.Sprintf("tya_set_member(__obj, %s, %s);", strconv.Quote(field.Name), arg))
	}
	if decl.Record {
		child.line(fmt.Sprintf("tya_set_member(__obj, \"with\", tya_bind_method_params(__obj, %s, %s));", withSym, cParamArray(params)))
	}
	child.line("return __obj;")
	out.WriteString(child.out.String())
	out.WriteString("}\n")
	g.funcOut.WriteString(out.String())
	g.temp = child.temp
	target := cName(name)
	if !g.vars[name] {
		g.vars[name] = true
		g.globalLine(fmt.Sprintf("TyaValue %s;", target))
	}
	g.line(fmt.Sprintf("%s = tya_function_params(%s, %s);", target, sym, cParamArray(params)))
	g.funcs[name] = sym
	g.funcParams[name] = params
	g.classParams[name] = params
	_ = required
	return nil
}

func (g *cgen) structConstructorExpr(name string, decl *ast.StructDecl, call *ast.CallExpr) (string, error) {
	args, err := g.callArgExprs(call, structFieldNames(decl))
	if err != nil {
		return "", err
	}
	if len(args) > len(decl.Fields) {
		return "", fmt.Errorf("too many positional arguments")
	}
	for len(args) < len(decl.Fields) {
		args = append(args, "tya_missing()")
	}
	obj := fmt.Sprintf("__struct%d", g.temp)
	g.temp++
	lines := []string{
		fmt.Sprintf("TyaValue %s = tya_object();", obj),
		fmt.Sprintf("tya_set_member(%s, \"__data_type\", tya_string(%s));", obj, strconv.Quote(name)),
	}
	if decl.Record {
		lines = append(lines, fmt.Sprintf("tya_set_member(%s, \"__record\", tya_bool(true));", obj))
	} else {
		lines = append(lines, fmt.Sprintf("tya_set_member(%s, \"__struct\", tya_bool(true));", obj))
	}
	for i, field := range decl.Fields {
		arg := fmt.Sprintf("__field%d_%d", g.temp, i)
		lines = append(lines, fmt.Sprintf("TyaValue %s = %s;", arg, args[i]))
		if field.HasDefault {
			def, _, err := g.expr(field.Value)
			if err != nil {
				return "", err
			}
			lines = append(lines, fmt.Sprintf("if (%s.kind == TYA_MISSING) { %s = %s; }", arg, arg, def))
		} else {
			lines = append(lines, fmt.Sprintf("if (%s.kind == TYA_MISSING) { tya_panic(tya_string(%s)); }", arg, strconv.Quote("missing required argument "+field.Name)))
		}
		lines = append(lines, fmt.Sprintf("tya_set_member(%s, %s, %s);", obj, strconv.Quote(field.Name), arg))
	}
	if decl.Record {
		if withSym := g.structWithSyms[name]; withSym != "" {
			lines = append(lines, fmt.Sprintf("tya_set_member(%s, \"with\", tya_bind_method_params(%s, %s, %s));", obj, obj, withSym, cParamArray(structFieldNames(decl))))
		}
	}
	return fmt.Sprintf("({ %s %s; })", strings.Join(lines, " "), obj), nil
}

func structFieldNames(decl *ast.StructDecl) []string {
	names := make([]string, len(decl.Fields))
	for i, field := range decl.Fields {
		names[i] = field.Name
	}
	return names
}

func (g *cgen) sourceLine(line int) {
	if line > 0 {
		g.line(fmt.Sprintf("/* tya:%d */", line))
	}
}

// stmtPos returns the source position of stmt for coverage purposes.
// ok=false means the stmt is not instrumentable (no token, or no
// position information was preserved through parsing).
func stmtPos(stmt ast.Stmt) (line, col int, ok bool) {
	switch n := stmt.(type) {
	case *ast.AssignStmt:
		return n.Tok.Line, n.Tok.Col, n.Tok.Line > 0
	case *ast.ImportStmt:
		return n.NameTok.Line, n.NameTok.Col, n.NameTok.Line > 0
	case *ast.ImportBlockStmt:
		return n.Tok.Line, n.Tok.Col, n.Tok.Line > 0
	case *ast.ModuleDecl:
		return n.NameTok.Line, n.NameTok.Col, n.NameTok.Line > 0
	case *ast.ClassDecl:
		return n.NameTok.Line, n.NameTok.Col, n.NameTok.Line > 0
	case *ast.InterfaceDecl:
		return n.NameTok.Line, n.NameTok.Col, n.NameTok.Line > 0
	case *ast.ReturnStmt:
		return n.Tok.Line, n.Tok.Col, n.Tok.Line > 0
	case *ast.RaiseStmt:
		return n.Tok.Line, n.Tok.Col, n.Tok.Line > 0
	case *ast.BreakStmt:
		return n.Tok.Line, n.Tok.Col, n.Tok.Line > 0
	case *ast.ContinueStmt:
		return n.Tok.Line, n.Tok.Col, n.Tok.Line > 0
	case *ast.MatchStmt:
		return n.Tok.Line, n.Tok.Col, n.Tok.Line > 0
	case *ast.TryCatchStmt:
		return n.Tok.Line, n.Tok.Col, n.Tok.Line > 0
	case *ast.ForInStmt:
		return n.ValueTok.Line, n.ValueTok.Col, n.ValueTok.Line > 0
	case *ast.ExprStmt:
		return exprPos(n.Expr)
	}
	return 0, 0, false
}

func exprPos(expr ast.Expr) (line, col int, ok bool) {
	switch n := expr.(type) {
	case *ast.Ident:
		return n.Tok.Line, n.Tok.Col, n.Tok.Line > 0
	case *ast.CallExpr:
		return exprPos(n.Callee)
	case *ast.BinaryExpr:
		return exprPos(n.Left)
	case *ast.MemberExpr:
		return exprPos(n.Target)
	case *ast.IndexExpr:
		return exprPos(n.Target)
	case *ast.UnaryExpr:
		return exprPos(n.Expr)
	case *ast.TryExpr:
		return n.Tok.Line, n.Tok.Col, n.Tok.Line > 0
	case *ast.IfStmt:
		return exprPos(n.Cond)
	case *ast.WhileStmt:
		return exprPos(n.Cond)
	case *ast.ForInStmt:
		return n.ValueTok.Line, n.ValueTok.Col, n.ValueTok.Line > 0
	case *ast.MatchStmt:
		return n.Tok.Line, n.Tok.Col, n.Tok.Line > 0
	}
	return 0, 0, false
}

// instrument emits tya_cov_inc(<id>); when coverage is on and the stmt
// has a position in a non-excluded source file.
func (g *cgen) instrument(stmt ast.Stmt) {
	if !g.coverEnabled {
		return
	}
	line, col, ok := stmtPos(stmt)
	if !ok {
		return
	}
	file := g.sourcePath
	if g.excludedSource(file) {
		return
	}
	id := len(g.coverEntries)
	g.coverEntries = append(g.coverEntries, CoverageEntry{ID: id, File: file, Line: line, Col: col})
	g.line(fmt.Sprintf("tya_cov_inc(%d);", id))
}

func (g *cgen) excludedSource(path string) bool {
	if path == "" {
		return true
	}
	if g.coverOpt == nil {
		return false
	}
	if g.coverOpt.StdlibDir != "" && strings.HasPrefix(path, g.coverOpt.StdlibDir) {
		return true
	}
	if g.coverOpt.PackagesDir != "" && strings.HasPrefix(path, g.coverOpt.PackagesDir) {
		return true
	}
	for _, p := range g.coverOpt.ExcludePaths {
		if path == p || strings.HasPrefix(path, p) {
			return true
		}
	}
	if len(g.coverOpt.Include) > 0 {
		matched := false
		for _, p := range g.coverOpt.Include {
			if coverageGlobMatch(p, path) {
				matched = true
				break
			}
		}
		if !matched {
			return true
		}
	}
	for _, p := range g.coverOpt.Exclude {
		if coverageGlobMatch(p, path) {
			return true
		}
	}
	return false
}

func coverageGlobMatch(pattern, name string) bool {
	pattern = filepath.ToSlash(filepath.Clean(pattern))
	name = filepath.ToSlash(filepath.Clean(name))
	if ok, _ := path.Match(pattern, name); ok {
		return true
	}
	if !path.IsAbs(pattern) {
		for i := 0; i < len(name); i++ {
			if i > 0 && name[i-1] != '/' {
				continue
			}
			if ok, _ := path.Match(pattern, name[i:]); ok {
				return true
			}
		}
	}
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := strings.TrimSuffix(parts[0], "/")
			suffix := strings.TrimPrefix(parts[1], "/")
			return (strings.HasPrefix(name, prefix) || strings.Contains(name, "/"+prefix+"/")) && strings.HasSuffix(name, suffix)
		}
	}
	return false
}

func (g *cgen) tryCatchStmt(n *ast.TryCatchStmt) error {
	frame := fmt.Sprintf("__raise_frame%d", g.temp)
	g.temp++
	g.line(fmt.Sprintf("TyaRaiseFrame %s;", frame))
	g.line(fmt.Sprintf("tya_push_raise_frame(&%s);", frame))
	g.line(fmt.Sprintf("if (setjmp(%s.env) == 0) {", frame))
	g.indent++
	g.raiseDepth++
	for _, stmt := range n.Try {
		if err := g.stmt(stmt); err != nil {
			return err
		}
	}
	g.raiseDepth--
	if len(n.Finally) > 0 {
		for _, stmt := range n.Finally {
			if err := g.stmt(stmt); err != nil {
				return err
			}
		}
	}
	g.line("tya_pop_raise_frame();")
	g.indent--
	g.line("} else {")
	g.indent++
	raisedValue := fmt.Sprintf("__raised%d", g.temp)
	g.temp++
	g.line(fmt.Sprintf("TyaValue %s = tya_current_raise();", raisedValue))
	if n.Catch != nil && n.CatchName != "_" {
		name := cName(n.CatchName)
		g.vars[n.CatchName] = true
		g.line(fmt.Sprintf("TyaValue %s = %s;", name, raisedValue))
	}
	g.line("tya_pop_raise_frame();")
	if n.Catch != nil {
		for _, stmt := range n.Catch {
			if err := g.stmt(stmt); err != nil {
				return err
			}
		}
		if len(n.Finally) > 0 {
			for _, stmt := range n.Finally {
				if err := g.stmt(stmt); err != nil {
					return err
				}
			}
		}
	} else {
		if len(n.Finally) > 0 {
			for _, stmt := range n.Finally {
				if err := g.stmt(stmt); err != nil {
					return err
				}
			}
		}
		g.line(fmt.Sprintf("tya_raise(%s);", raisedValue))
	}
	g.indent--
	g.line("}")
	return nil
}

func (g *cgen) matchStmt(n *ast.MatchStmt) error {
	return g.matchStmtWithResult(n, "")
}

func (g *cgen) matchStmtWithResult(n *ast.MatchStmt, result string) error {
	value, _, err := g.expr(n.Value)
	if err != nil {
		return err
	}
	matchValue := fmt.Sprintf("__match%d", g.temp)
	matched := fmt.Sprintf("__matched%d", g.temp)
	g.temp++
	g.line(fmt.Sprintf("TyaValue %s = %s;", matchValue, value))
	g.line(fmt.Sprintf("bool %s = false;", matched))
	for _, c := range n.Cases {
		cond, err := g.matchCondition(c.Pattern, matchValue)
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("if (!%s && (%s)) {", matched, cond))
		g.indent++
		g.line(fmt.Sprintf("%s = true;", matched))
		bindings := patternBindings(c.Pattern)
		oldVars := map[string]bool{}
		for _, binding := range bindings {
			oldVars[binding] = g.vars[binding]
			g.vars[binding] = true
		}
		if err := g.bindPattern(c.Pattern, matchValue); err != nil {
			return err
		}
		if result != "" {
			if err := g.assignControlBodyValue(result, c.Body); err != nil {
				return err
			}
		} else {
			for _, stmt := range c.Body {
				if err := g.stmt(stmt); err != nil {
					return err
				}
			}
		}
		for _, binding := range bindings {
			if oldVars[binding] {
				g.vars[binding] = true
			} else {
				delete(g.vars, binding)
			}
		}
		g.indent--
		g.line("}")
	}
	return nil
}

func (g *cgen) matchCondition(pattern ast.Expr, value string) (string, error) {
	switch n := pattern.(type) {
	case *ast.Ident:
		return "true", nil
	case *ast.IntLit, *ast.FloatLit, *ast.StringLit, *ast.BoolLit, *ast.NilLit:
		lit, _, err := g.expr(pattern)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("tya_truthy(tya_deep_equal(%s, %s))", value, lit), nil
	case *ast.ArrayLit:
		parts := []string{fmt.Sprintf("%s.kind == TYA_ARRAY", value), fmt.Sprintf("(int)tya_len(%s).number == %d", value, len(n.Elems))}
		for i, elem := range n.Elems {
			cond, err := g.matchCondition(elem, fmt.Sprintf("tya_index(%s, tya_number(%d))", value, i))
			if err != nil {
				return "", err
			}
			parts = append(parts, cond)
		}
		return strings.Join(parts, " && "), nil
	case *ast.DictLit:
		parts := []string{fmt.Sprintf("%s.kind == TYA_DICT", value)}
		for _, prop := range n.Props {
			key := fmt.Sprintf("tya_string(%s)", strconv.Quote(prop.Name))
			parts = append(parts, fmt.Sprintf("tya_truthy(tya_has(%s, %s))", value, key))
			cond, err := g.matchCondition(prop.Value, fmt.Sprintf("tya_index(%s, %s)", value, key))
			if err != nil {
				return "", err
			}
			parts = append(parts, cond)
		}
		return strings.Join(parts, " && "), nil
	default:
		return "", codegenError(codePatternUnsupported, fmt.Sprintf("C emitter does not support pattern %T", pattern), 0, 0)
	}
}

func (g *cgen) bindPattern(pattern ast.Expr, value string) error {
	switch n := pattern.(type) {
	case *ast.Ident:
		if n.Name != "_" {
			g.line(fmt.Sprintf("TyaValue %s = %s;", cName(n.Name), value))
		}
	case *ast.ArrayLit:
		for i, elem := range n.Elems {
			if err := g.bindPattern(elem, fmt.Sprintf("tya_index(%s, tya_number(%d))", value, i)); err != nil {
				return err
			}
		}
	case *ast.DictLit:
		for _, prop := range n.Props {
			if err := g.bindPattern(prop.Value, fmt.Sprintf("tya_index(%s, tya_string(%s))", value, strconv.Quote(prop.Name))); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *cgen) assignTry(name string, tryExpr *ast.TryExpr) error {
	if !g.inFunc {
		line, col, _ := exprPos(tryExpr)
		return codegenError(codeTryOutsideFunc, "C emitter only supports try inside functions", line, col)
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
		line, col, _ := stmtPos(n)
		return codegenError(codeMultiAssignNonTuple, "C emitter only supports tuple-style multiple assignment", line, col)
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
			line, col, _ := exprPos(target)
			return codegenError(codeDestructureNonIdent, "C emitter only supports identifier multiple assignment targets", line, col)
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

func (g *cgen) nilAssign(n *ast.AssignStmt) error {
	if len(n.Targets) != 1 || len(n.Values) != 1 {
		line, col, _ := stmtPos(n)
		return codegenError(codeMultiAssignNonTuple, "??= assignment requires exactly one target and one value", line, col)
	}
	switch target := n.Targets[0].(type) {
	case *ast.Ident:
		name := cName(target.Name)
		if !g.vars[target.Name] {
			g.vars[target.Name] = true
			g.line(fmt.Sprintf("TyaValue %s = tya_nil();", name))
		}
		g.line(fmt.Sprintf("if (%s.kind == TYA_NIL) {", name))
		g.indent++
		value, _, err := g.expr(n.Values[0])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("%s = %s;", name, value))
		g.indent--
		g.line("}")
		return nil
	case *ast.IndexExpr:
		obj, _, err := g.expr(target.Target)
		if err != nil {
			return err
		}
		idx, _, err := g.expr(target.Index)
		if err != nil {
			return err
		}
		objTmp := fmt.Sprintf("__nil_assign_obj%d", g.temp)
		idxTmp := fmt.Sprintf("__nil_assign_idx%d", g.temp)
		g.temp++
		g.line(fmt.Sprintf("TyaValue %s = %s;", objTmp, obj))
		g.line(fmt.Sprintf("TyaValue %s = %s;", idxTmp, idx))
		g.line(fmt.Sprintf("if (tya_index(%s, %s).kind == TYA_NIL) {", objTmp, idxTmp))
		g.indent++
		value, _, err := g.expr(n.Values[0])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_set_index(%s, %s, %s);", objTmp, idxTmp, value))
		g.indent--
		g.line("}")
		return nil
	case *ast.MemberExpr:
		key := target.Name
		receiver := ""
		if selfTarget, ok := target.Target.(*ast.SelfExpr); ok {
			if selfTarget.Class {
				receiver = g.classTarget()
			} else {
				receiver = "__this"
				key = "@" + target.Name
			}
		} else {
			var err error
			receiver, _, err = g.expr(target.Target)
			if err != nil {
				return err
			}
		}
		recvTmp := fmt.Sprintf("__nil_assign_recv%d", g.temp)
		g.temp++
		g.line(fmt.Sprintf("TyaValue %s = %s;", recvTmp, receiver))
		g.line(fmt.Sprintf("if (tya_member(%s, %s).kind == TYA_NIL) {", recvTmp, strconv.Quote(key)))
		g.indent++
		value, _, err := g.expr(n.Values[0])
		if err != nil {
			return err
		}
		if selfTarget, ok := target.Target.(*ast.SelfExpr); ok && !selfTarget.Class {
			tmp := fmt.Sprintf("__field%d", g.temp)
			g.temp++
			g.line(fmt.Sprintf("TyaValue %s = %s;", tmp, value))
			g.line(fmt.Sprintf("tya_set_member(%s, %s, %s);", recvTmp, strconv.Quote("@"+target.Name), tmp))
			g.line(fmt.Sprintf("tya_set_member(%s, %s, %s);", recvTmp, strconv.Quote(target.Name), tmp))
		} else {
			g.line(fmt.Sprintf("tya_set_member(%s, %s, %s);", recvTmp, strconv.Quote(key), value))
		}
		g.indent--
		g.line("}")
		return nil
	case *ast.InstanceFieldExpr:
		g.line(fmt.Sprintf("if (tya_member(__this, %s).kind == TYA_NIL) {", strconv.Quote("@"+target.Name)))
		g.indent++
		value, _, err := g.expr(n.Values[0])
		if err != nil {
			return err
		}
		tmp := fmt.Sprintf("__field%d", g.temp)
		g.temp++
		g.line(fmt.Sprintf("TyaValue %s = %s;", tmp, value))
		g.line(fmt.Sprintf("tya_set_member(__this, %s, %s);", strconv.Quote("@"+target.Name), tmp))
		g.line(fmt.Sprintf("tya_set_member(__this, %s, %s);", strconv.Quote(target.Name), tmp))
		g.indent--
		g.line("}")
		return nil
	case *ast.ClassVarExpr:
		receiver := g.classTarget()
		g.line(fmt.Sprintf("if (tya_member(%s, %s).kind == TYA_NIL) {", receiver, strconv.Quote(target.Name)))
		g.indent++
		value, _, err := g.expr(n.Values[0])
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_set_member(%s, %s, %s);", receiver, strconv.Quote(target.Name), value))
		g.indent--
		g.line("}")
		return nil
	default:
		line, col, _ := exprPos(target)
		return codegenError(codeAssignTargetUnsupported, "C emitter only supports assignment targets for ??=", line, col)
	}
}

func (g *cgen) assignDestructuring(target ast.Expr, value ast.Expr) error {
	ex, _, err := g.expr(value)
	if err != nil {
		return err
	}
	temp := fmt.Sprintf("__destructure%d", g.temp)
	g.temp++
	g.line(fmt.Sprintf("TyaValue %s = %s;", temp, ex))
	return g.assignDestructuringValue(target, temp)
}

func (g *cgen) assignDestructuringValue(target ast.Expr, value string) error {
	switch n := target.(type) {
	case *ast.Ident:
		return g.assignIdentValue(n.Name, value)
	case *ast.ArrayLit:
		for i, elem := range n.Elems {
			item := fmt.Sprintf("tya_destructure_array(%s, %d, %d)", value, len(n.Elems), i)
			if err := g.assignDestructuringValue(elem, item); err != nil {
				return err
			}
		}
		return nil
	case *ast.DictLit:
		for _, prop := range n.Props {
			item := fmt.Sprintf("tya_destructure_dict(%s, %s)", value, strconv.Quote(prop.Name))
			if err := g.assignDestructuringValue(prop.Value, item); err != nil {
				return err
			}
		}
		return nil
	default:
		return codegenError(codeDestructureNonIdent, "C emitter only supports identifier destructuring targets", 0, 0)
	}
}

func (g *cgen) assignIdentValue(name string, value string) error {
	if name == "_" {
		return nil
	}
	if g.vars[name] {
		g.line(fmt.Sprintf("%s = %s;", cName(name), value))
	} else {
		g.vars[name] = true
		g.line(fmt.Sprintf("TyaValue %s = %s;", cName(name), value))
	}
	return nil
}

func isDestructuringTarget(target ast.Expr) bool {
	switch target.(type) {
	case *ast.ArrayLit, *ast.DictLit:
		return true
	default:
		return false
	}
}
