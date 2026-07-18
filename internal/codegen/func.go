package codegen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tya/internal/ast"
)

func (g *cgen) emitFunc(name string, fn *ast.FuncLit) (string, error) {
	return g.emitFuncWithContext(name, fn, "", "")
}

func (g *cgen) emitFuncWithContext(name string, fn *ast.FuncLit, classRef string, methodKind string) (string, error) {
	return g.emitFuncWithCaptures(name, fn, classRef, methodKind, nil)
}

func (g *cgen) emitFuncWithCaptures(name string, fn *ast.FuncLit, classRef string, methodKind string, captures map[string]bool) (string, error) {
	sym := cFuncName(name, g.temp)
	g.temp++
	var out strings.Builder
	out.WriteString("TyaValue ")
	out.WriteString(sym)
	out.WriteString("(TyaValue __this, TyaValue __arg0, TyaValue __arg1, TyaValue __arg2, TyaValue __arg3, TyaValue __arg4, TyaValue __arg5) {\n")
	child := &cgen{
		vars:              map[string]bool{},
		funcs:             g.funcs,
		funcParams:        g.funcParams,
		classes:           g.classes,
		classParams:       g.classParams,
		classMethods:      g.classMethods,
		methodParams:      g.methodParams,
		classDecls:        g.classDecls,
		structDecls:       g.structDecls,
		structWithSyms:    g.structWithSyms,
		interfaceDecls:    g.interfaceDecls,
		moduleClasses:     g.moduleClasses,
		sourcePath:        g.sourcePath,
		temp:              g.temp,
		indent:            1,
		inFunc:            true,
		inClassMethod:     methodKind == "class",
		inInstanceMethod:  methodKind == "instance",
		classRef:          classRef,
		className:         g.className,
		methodName:        g.methodName,
		superClass:        g.superClass,
		interfaceSuperSym: g.interfaceSuperSym,
		predicateName:     predicateName(name),
		closureVars:       captures,
		gcSafeLoops:       true,
		gcFrameName:       "__gc_frame",
	}
	hasDefaults := false
	for _, def := range fn.Defaults {
		if def != nil {
			hasDefaults = true
			break
		}
	}
	for i, param := range fn.Params {
		child.vars[param] = true
		if i < 6 {
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
	rootNames := []string{"__this"}
	for i, param := range fn.Params {
		if i < 6 {
			rootNames = append(rootNames, cName(param))
		}
	}
	for _, local := range assignedNames(fn.Body) {
		if !contains(rootNames, cName(local)) {
			rootNames = append(rootNames, cName(local))
		}
	}
	rootPointers := make([]string, 0, len(rootNames))
	for _, root := range rootNames {
		rootPointers = append(rootPointers, "&"+root)
	}
	child.line(fmt.Sprintf("TyaValue *__gc_roots[] = {%s};", strings.Join(rootPointers, ", ")))
	child.line("TyaGcRootFrame __gc_frame;")
	child.line(fmt.Sprintf("tya_gc_enter_frame(&__gc_frame, __gc_roots, %d);", len(rootPointers)))
	for i, param := range fn.Params {
		if i >= 6 {
			continue
		}
		if i < len(fn.Defaults) && fn.Defaults[i] != nil {
			def, _, err := child.expr(fn.Defaults[i])
			if err != nil {
				return "", err
			}
			child.line(fmt.Sprintf("if (%s.kind == TYA_MISSING) { %s = %s; }", cName(param), cName(param), def))
		} else if hasDefaults {
			child.line(fmt.Sprintf("if (%s.kind == TYA_MISSING) { %s = tya_nil(); }", cName(param), cName(param)))
		}
	}
	if fn.Expr != nil {
		value, _, err := child.expr(fn.Expr)
		if err != nil {
			return "", err
		}
		child.returnLine(value)
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
				if isSideEffectCall(last.Expr) {
					if err := child.stmt(last); err != nil {
						return "", err
					}
					child.returnLine("tya_nil()")
				} else {
					value, _, err := child.expr(last.Expr)
					if err != nil {
						return "", err
					}
					child.returnLine(value)
				}
			} else {
				if control, ok := body[len(body)-1].(ast.Expr); ok && isControlFlowExpr(control) {
					body = body[:len(body)-1]
					for _, stmt := range body {
						if err := child.stmt(stmt); err != nil {
							return "", err
						}
					}
					value, _, err := child.expr(control)
					if err != nil {
						return "", err
					}
					child.returnLine(value)
					goto doneBody
				}
				for _, stmt := range body {
					if err := child.stmt(stmt); err != nil {
						return "", err
					}
				}
				child.returnLine("tya_nil()")
			}
		} else {
			child.returnLine("tya_nil()")
		}
	}
doneBody:
	g.temp = child.temp
	g.funcOut.WriteString(child.funcOut.String())
	out.WriteString(child.out.String())
	out.WriteString("}\n\n")
	g.funcOut.WriteString(out.String())
	return sym, nil
}

func cParamArray(params []string) string {
	if len(params) == 0 {
		return "(const char**)0, 0"
	}
	values := make([]string, 0, len(params))
	for _, param := range params {
		values = append(values, strconv.Quote(param))
	}
	return fmt.Sprintf("((const char*[]){%s}), %d", strings.Join(values, ", "), len(params))
}

func (g *cgen) returnLine(value string) {
	for i := 0; i < g.raiseDepth; i++ {
		g.line("tya_pop_raise_frame();")
	}
	result := fmt.Sprintf("__predicate_result_%d", g.temp)
	g.temp++
	g.line(fmt.Sprintf("TyaValue %s = %s;", result, value))
	if g.predicateName != "" {
		g.line(fmt.Sprintf("if (%s.kind != TYA_BOOL) {", result))
		g.indent++
		g.line(fmt.Sprintf("tya_panic(tya_string(%s));", strconv.Quote(g.predicateName+" must return boolean")))
		g.indent--
		g.line("}")
	}
	if g.gcFrameName != "" {
		g.line(fmt.Sprintf("tya_gc_leave_frame(&%s);", g.gcFrameName))
	}
	g.line(fmt.Sprintf("return %s;", result))
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func predicateName(name string) string {
	parts := strings.Split(name, "_")
	name = parts[len(parts)-1]
	if strings.HasSuffix(name, "?") {
		return name
	}
	return ""
}

func isSideEffectCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	id, ok := call.Callee.(*ast.Ident)
	if !ok {
		return false
	}
	switch id.Name {
	case "push", "delete", "write_file", "exit", "panic", "print", "println", "assert", "assert_equal":
		return true
	default:
		return false
	}
}

func isControlFlowExpr(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.IfStmt, *ast.WhileStmt, *ast.ForInStmt, *ast.MatchStmt:
		return true
	default:
		return false
	}
}

func sortedMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (g *cgen) freeVars(fn *ast.FuncLit) map[string]bool {
	defined := map[string]bool{}
	for _, param := range fn.Params {
		defined[param] = true
	}
	for _, local := range assignedNames(fn.Body) {
		defined[local] = true
	}
	used := map[string]bool{}
	for _, def := range fn.Defaults {
		collectExprIdents(def, used)
	}
	if fn.Expr != nil {
		collectExprIdents(fn.Expr, used)
	}
	for _, stmt := range fn.Body {
		collectStmtIdents(stmt, used)
	}
	captures := map[string]bool{}
	for name := range used {
		if defined[name] || primitiveClassNames[name] || g.classes[name] != "" || g.funcs[name] != "" {
			continue
		}
		if (g.inFunc && g.vars[name]) || (g.closureVars != nil && g.closureVars[name]) {
			captures[name] = true
		}
	}
	return captures
}

func (g *cgen) captureValue(name string) string {
	if g.closureVars != nil && g.closureVars[name] {
		return fmt.Sprintf("tya_member(__this, %s)", strconv.Quote(name))
	}
	return cName(name)
}

func collectStmtIdents(stmt ast.Stmt, out map[string]bool) {
	switch n := stmt.(type) {
	case *ast.AssignStmt:
		for _, value := range n.Values {
			collectExprIdents(value, out)
		}
	case *ast.ExprStmt:
		collectExprIdents(n.Expr, out)
	case *ast.IfStmt:
		collectExprIdents(n.Cond, out)
		for _, stmt := range n.Then {
			collectStmtIdents(stmt, out)
		}
		for _, stmt := range n.Else {
			collectStmtIdents(stmt, out)
		}
	case *ast.WhileStmt:
		collectExprIdents(n.Cond, out)
		for _, stmt := range n.Body {
			collectStmtIdents(stmt, out)
		}
	case *ast.ForInStmt:
		collectExprIdents(n.Iterable, out)
		for _, stmt := range n.Body {
			collectStmtIdents(stmt, out)
		}
	case *ast.ReturnStmt:
		for _, value := range n.Values {
			collectExprIdents(value, out)
		}
	case *ast.RaiseStmt:
		collectExprIdents(n.Value, out)
	case *ast.TryCatchStmt:
		for _, stmt := range n.Try {
			collectStmtIdents(stmt, out)
		}
		for _, stmt := range n.Catch {
			collectStmtIdents(stmt, out)
		}
		for _, stmt := range n.Finally {
			collectStmtIdents(stmt, out)
		}
	case *ast.MatchStmt:
		collectExprIdents(n.Value, out)
		for _, c := range n.Cases {
			for _, stmt := range c.Body {
				collectStmtIdents(stmt, out)
			}
		}
	case *ast.ScopeBlock:
		for _, stmt := range n.Body {
			collectStmtIdents(stmt, out)
		}
	case *ast.SelectStmt:
		for _, arm := range n.Arms {
			collectExprIdents(arm.Channel, out)
			collectExprIdents(arm.Value, out)
			collectExprIdents(arm.Seconds, out)
			for _, stmt := range arm.Body {
				collectStmtIdents(stmt, out)
			}
		}
	}
}

func collectExprIdents(expr ast.Expr, out map[string]bool) {
	switch n := expr.(type) {
	case *ast.Ident:
		if n.ImplicitField {
			return
		}
		out[n.Name] = true
	case *ast.ArrayLit:
		for _, elem := range n.Elems {
			collectExprIdents(elem, out)
		}
	case *ast.DictLit:
		for _, prop := range n.Props {
			collectExprIdents(prop.Value, out)
		}
	case *ast.BinaryExpr:
		collectExprIdents(n.Left, out)
		collectExprIdents(n.Right, out)
	case *ast.UnaryExpr:
		collectExprIdents(n.Expr, out)
	case *ast.TryExpr:
		collectExprIdents(n.Expr, out)
	case *ast.IfStmt:
		collectExprIdents(n.Cond, out)
		for _, stmt := range n.Then {
			collectStmtIdents(stmt, out)
		}
		for _, stmt := range n.Else {
			collectStmtIdents(stmt, out)
		}
	case *ast.WhileStmt:
		collectExprIdents(n.Cond, out)
		for _, stmt := range n.Body {
			collectStmtIdents(stmt, out)
		}
	case *ast.ForInStmt:
		collectExprIdents(n.Iterable, out)
		for _, stmt := range n.Body {
			collectStmtIdents(stmt, out)
		}
	case *ast.MatchStmt:
		collectExprIdents(n.Value, out)
		for _, c := range n.Cases {
			collectExprIdents(c.Pattern, out)
			for _, stmt := range c.Body {
				collectStmtIdents(stmt, out)
			}
		}
	case *ast.MemberExpr:
		collectExprIdents(n.Target, out)
	case *ast.IndexExpr:
		collectExprIdents(n.Target, out)
		collectExprIdents(n.Index, out)
	case *ast.CallExpr:
		collectExprIdents(n.Callee, out)
		for _, arg := range n.Args {
			collectExprIdents(arg, out)
		}
	case *ast.SpawnExpr:
		collectExprIdents(n.Callee, out)
	case *ast.AwaitExpr:
		collectExprIdents(n.Target, out)
	case *ast.FuncLit:
		collectFuncFreeIdentUses(n, out)
	}
}

func collectFuncFreeIdentUses(fn *ast.FuncLit, out map[string]bool) {
	defined := map[string]bool{}
	for _, param := range fn.Params {
		defined[param] = true
	}
	for _, local := range assignedNames(fn.Body) {
		defined[local] = true
	}
	used := map[string]bool{}
	for _, def := range fn.Defaults {
		collectExprIdents(def, used)
	}
	if fn.Expr != nil {
		collectExprIdents(fn.Expr, used)
	}
	for _, stmt := range fn.Body {
		collectStmtIdents(stmt, used)
	}
	for name := range used {
		if !defined[name] {
			out[name] = true
		}
	}
}
