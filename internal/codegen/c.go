package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"tya/internal/ast"
)

func EmitC(prog *ast.Program) (string, error) {
	return EmitCWithPath(prog, "")
}

func EmitCWithPath(prog *ast.Program, sourcePath string) (string, error) {
	g := &cgen{vars: map[string]bool{}, funcs: map[string]string{}, classes: map[string]string{}, classMethods: map[string]string{}, classDecls: map[string]*ast.ClassDecl{}, sourcePath: sourcePath}
	g.collectClasses(prog.Stmts)
	for _, name := range assignedNames(prog.Stmts) {
		g.vars[name] = true
		g.globalLine(fmt.Sprintf("TyaValue %s;", cName(name)))
	}
	for _, stmt := range prog.Stmts {
		if err := g.stmt(stmt); err != nil {
			return "", err
		}
	}
	var out strings.Builder
	out.WriteString("#include \"tya_runtime.h\"\n\n")
	out.WriteString(g.globalOut.String())
	if g.globalOut.Len() > 0 {
		out.WriteByte('\n')
	}
	out.WriteString(g.funcOut.String())
	out.WriteString("int main(int argc, char **argv) {\n")
	out.WriteString(g.out.String())
	out.WriteString("  return 0;\n")
	out.WriteString("}\n")
	return out.String(), nil
}

type cgen struct {
	out              strings.Builder
	globalOut        strings.Builder
	funcOut          strings.Builder
	sourcePath       string
	indent           int
	vars             map[string]bool
	funcs            map[string]string
	classes          map[string]string
	classMethods     map[string]string
	classDecls       map[string]*ast.ClassDecl
	temp             int
	inFunc           bool
	inClassMethod    bool
	inInstanceMethod bool
	classRef         string
	className        string
	methodName       string
	superClass       string
	predicateName    string
}

func (g *cgen) globalLine(s string) {
	g.globalOut.WriteString(s)
	g.globalOut.WriteByte('\n')
}

func (g *cgen) line(s string) {
	g.out.WriteString(strings.Repeat("  ", g.indent))
	g.out.WriteString(s)
	g.out.WriteByte('\n')
}

func (g *cgen) classTarget() string {
	if g.inClassMethod {
		return "__this"
	}
	if g.inInstanceMethod {
		return "tya_member(__this, \"class\")"
	}
	if g.classRef != "" {
		return g.classRef
	}
	return "__this"
}

func (g *cgen) collectClasses(stmts []ast.Stmt) {
	for _, stmt := range stmts {
		switch n := stmt.(type) {
		case *ast.ClassDecl:
			g.classDecls[n.Name] = n
		case *ast.ModuleDecl:
			for _, class := range n.Classes {
				g.classDecls[n.Name+"."+class.Name] = class
			}
		}
	}
}

func (g *cgen) stmt(stmt ast.Stmt) error {
	switch n := stmt.(type) {
	case *ast.ImportStmt:
		g.sourceLine(n.NameTok.Line)
		g.line("/* import resolved by runner */")
		return nil
	case *ast.ModuleDecl:
		g.sourceLine(n.NameTok.Line)
		return g.assignModuleDecl(n)
	case *ast.ClassDecl:
		g.sourceLine(n.NameTok.Line)
		return g.assignClassDecl(n.Name, n)
	case *ast.AssignStmt:
		g.sourceLine(n.Tok.Line)
		if len(n.Targets) != 1 || len(n.Values) != 1 {
			return g.multiAssign(n)
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
			receiver := g.classTarget()
			if strings.HasPrefix(target.Name, "_") && g.classRef != "" {
				receiver = g.classRef
			}
			g.line(fmt.Sprintf("tya_set_member(%s, %s, %s);", receiver, strconv.Quote(target.Name), value))
			return nil
		}
		id, ok := n.Targets[0].(*ast.Ident)
		if !ok {
			return fmt.Errorf("C emitter only supports variable assignment")
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
				g.line(fmt.Sprintf("%s = tya_dict_key_at(%s, tya_number(%s));", cName(n.ValueName), iterName, indexName))
			} else {
				g.vars[n.ValueName] = true
				g.line(fmt.Sprintf("TyaValue %s = tya_dict_key_at(%s, tya_number(%s));", cName(n.ValueName), iterName, indexName))
			}
			if n.IndexName != "" {
				if g.vars[n.IndexName] {
					g.line(fmt.Sprintf("%s = tya_dict_value_at(%s, tya_number(%s));", cName(n.IndexName), iterName, indexName))
				} else {
					g.vars[n.IndexName] = true
					g.line(fmt.Sprintf("TyaValue %s = tya_dict_value_at(%s, tya_number(%s));", cName(n.IndexName), iterName, indexName))
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
		return fmt.Errorf("C emitter does not support %T", stmt)
	}
	return nil
}

func (g *cgen) sourceLine(line int) {
	if line > 0 {
		g.line(fmt.Sprintf("/* tya:%d */", line))
	}
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
	return g.emitFuncWithContext(name, fn, "", "")
}

func (g *cgen) emitFuncWithContext(name string, fn *ast.FuncLit, classRef string, methodKind string) (string, error) {
	sym := cFuncName(name, g.temp)
	g.temp++
	var out strings.Builder
	out.WriteString("TyaValue ")
	out.WriteString(sym)
	out.WriteString("(TyaValue __this, TyaValue __arg0, TyaValue __arg1, TyaValue __arg2, TyaValue __arg3) {\n")
	child := &cgen{
		vars:             map[string]bool{},
		funcs:            g.funcs,
		classes:          g.classes,
		classMethods:     g.classMethods,
		classDecls:       g.classDecls,
		sourcePath:       g.sourcePath,
		temp:             g.temp,
		indent:           1,
		inFunc:           true,
		inClassMethod:    methodKind == "class",
		inInstanceMethod: methodKind == "instance",
		classRef:         classRef,
		className:        g.className,
		methodName:       g.methodName,
		superClass:       g.superClass,
		predicateName:    predicateName(name),
	}
	for i, param := range fn.Params {
		child.vars[param] = true
		if i < 4 {
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
	g.temp = child.temp
	g.funcOut.WriteString(child.funcOut.String())
	out.WriteString(child.out.String())
	out.WriteString("}\n\n")
	g.funcOut.WriteString(out.String())
	return sym, nil
}

func (g *cgen) returnLine(value string) {
	if g.predicateName == "" {
		g.line(fmt.Sprintf("return %s;", value))
		return
	}
	result := fmt.Sprintf("__predicate_result_%d", g.temp)
	g.temp++
	g.line(fmt.Sprintf("TyaValue %s = %s;", result, value))
	g.line(fmt.Sprintf("if (%s.kind != TYA_BOOL) {", result))
	g.indent++
	g.line(fmt.Sprintf("tya_panic(tya_string(%s));", strconv.Quote(g.predicateName+" must return boolean")))
	g.indent--
	g.line("}")
	g.line(fmt.Sprintf("return %s;", result))
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
	case "push", "delete", "write_file", "exit", "panic", "print":
		return true
	default:
		return false
	}
}

func (g *cgen) assignDictLit(name string, obj *ast.DictLit) error {
	entries := []string{}
	for _, prop := range obj.Props {
		value, _, err := g.expr(prop.Value)
		if err != nil {
			return err
		}
		entries = append(entries, fmt.Sprintf("{%s, %s}", strconv.Quote(prop.Name), value))
	}
	target := cName(name)
	g.line(fmt.Sprintf("%s = tya_dict((TyaDictEntry[]){%s}, %d);", target, strings.Join(entries, ", "), len(entries)))
	return nil
}

func (g *cgen) assignModuleDecl(module *ast.ModuleDecl) error {
	entries := []string{}
	functions := []ast.ModuleMember{}
	classes := module.Classes
	for _, member := range module.Members {
		if _, ok := member.Value.(*ast.FuncLit); ok {
			functions = append(functions, member)
			continue
		}
		value, _, err := g.expr(member.Value)
		if err != nil {
			return err
		}
		entries = append(entries, fmt.Sprintf("{%s, %s}", strconv.Quote(member.Name), value))
	}
	target := cName(module.Name)
	if !g.vars[module.Name] {
		g.vars[module.Name] = true
		g.line(fmt.Sprintf("TyaValue %s = tya_nil();", target))
	}
	g.line(fmt.Sprintf("%s = tya_dict((TyaDictEntry[]){%s}, %d);", target, strings.Join(entries, ", "), len(entries)))
	for _, member := range functions {
		fn := member.Value.(*ast.FuncLit)
		funcName := module.Name + "_" + member.Name
		sym, err := g.emitFunc(funcName, fn)
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_set_member(%s, %s, tya_function(%s));", target, strconv.Quote(member.Name), sym))
	}
	for _, class := range classes {
		classTarget := cName(module.Name + "_" + class.Name + "_class")
		sym, err := g.emitClass(module.Name+"_"+class.Name, class, classTarget)
		if err != nil {
			return err
		}
		g.globalLine(fmt.Sprintf("TyaValue %s;", classTarget))
		g.line(fmt.Sprintf("%s = tya_class(%s, %s, %s);", classTarget, sym, strconv.Quote(class.Name), g.parentExpr(module.Name+"_"+class.Name, class)))
		g.line(fmt.Sprintf("tya_set_member(%s, %s, %s);", target, strconv.Quote(class.Name), classTarget))
		if err := g.emitClassMembers(classTarget, module.Name+"_"+class.Name, class); err != nil {
			return err
		}
	}
	return nil
}

func (g *cgen) assignClassDecl(name string, class *ast.ClassDecl) error {
	target := cName(name)
	sym, err := g.emitClass(name, class, target)
	if err != nil {
		return err
	}
	g.classes[name] = sym
	if g.vars[name] {
		g.line(fmt.Sprintf("%s = tya_class(%s, %s, %s);", target, sym, strconv.Quote(name), g.parentExpr(name, class)))
	} else {
		g.vars[name] = true
		g.globalLine(fmt.Sprintf("TyaValue %s;", target))
		g.line(fmt.Sprintf("%s = tya_class(%s, %s, %s);", target, sym, strconv.Quote(name), g.parentExpr(name, class)))
	}
	return g.emitClassMembers(target, name, class)
}

func (g *cgen) emitClass(name string, class *ast.ClassDecl, classRef string) (string, error) {
	methodSyms := map[string]string{}
	var initMethod *ast.ClassMethod
	parentKey := g.parentKey(name, class)
	for i := range class.Methods {
		method := &class.Methods[i]
		if method.Abstract {
			continue
		}
		prevClass, prevMethod, prevSuper := g.className, g.methodName, g.superClass
		g.className, g.methodName, g.superClass = name, method.Name, parentKey
		methodKind := "instance"
		if method.Class {
			methodKind = "class"
		}
		sym, err := g.emitFuncWithContext(name+"_"+method.Name, method.Func, classRef, methodKind)
		g.className, g.methodName, g.superClass = prevClass, prevMethod, prevSuper
		if err != nil {
			return "", err
		}
		methodSyms[method.Name] = sym
		g.classMethods[name+"."+method.Name] = sym
		if (method.Name == "init" || method.Name == "_init") && !method.Class {
			initMethod = method
		}
	}
	sym := cFuncName(name+"_new", g.temp)
	g.temp++
	var out strings.Builder
	out.WriteString("TyaValue ")
	out.WriteString(sym)
	out.WriteString("(TyaValue __this, TyaValue __arg0, TyaValue __arg1, TyaValue __arg2, TyaValue __arg3) {\n")
	out.WriteString("  (void)__this;\n")
	out.WriteString("  TyaValue __obj = tya_object();\n")
	out.WriteString(fmt.Sprintf("  tya_set_member(__obj, \"class\", %s);\n", classRef))
	out.WriteString(fmt.Sprintf("  tya_set_member(__obj, \"class_name\", tya_string(%s));\n", strconv.Quote(class.Name)))
	if parentKey != "" {
		if err := g.emitParentDefaults(&out, parentKey); err != nil {
			return "", err
		}
	}
	for _, field := range class.Fields {
		value, _, err := g.expr(field.Value)
		if err != nil {
			return "", err
		}
		out.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, %s);\n", strconv.Quote("@"+field.Name), value))
		out.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, %s);\n", strconv.Quote(field.Name), value))
	}
	for _, method := range class.Methods {
		if method.Abstract || method.Class || method.Name == "init" || method.Name == "_init" || !strings.HasPrefix(method.Name, "_") {
			continue
		}
		out.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, tya_bind_method(__obj, %s));\n", strconv.Quote(method.Name), methodSyms[method.Name]))
	}
	if initMethod != nil {
		out.WriteString(fmt.Sprintf("  (void)%s(__obj, __arg0, __arg1, __arg2, __arg3);\n", methodSyms[initMethod.Name]))
	} else if parentKey != "" {
		if parentInit := g.inheritedMethodSym(parentKey, "init"); parentInit != "" {
			out.WriteString(fmt.Sprintf("  (void)%s(__obj, __arg0, __arg1, __arg2, __arg3);\n", parentInit))
		} else {
			out.WriteString("  (void)__arg0;\n  (void)__arg1;\n  (void)__arg2;\n  (void)__arg3;\n")
		}
	} else {
		out.WriteString("  (void)__arg0;\n  (void)__arg1;\n  (void)__arg2;\n  (void)__arg3;\n")
	}
	if parentKey != "" {
		g.emitParentMethods(&out, parentKey, class)
	}
	for _, method := range class.Methods {
		if method.Abstract || method.Class || method.Name == "init" || method.Name == "_init" {
			continue
		}
		out.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, tya_bind_method(__obj, %s));\n", strconv.Quote(method.Name), methodSyms[method.Name]))
	}
	out.WriteString("  return __obj;\n")
	out.WriteString("}\n\n")
	g.funcOut.WriteString(out.String())
	return sym, nil
}

func (g *cgen) emitClassMembers(target string, name string, class *ast.ClassDecl) error {
	for _, variable := range class.Vars {
		value, _, err := g.expr(variable.Value)
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_set_member(%s, %s, %s);", target, strconv.Quote(variable.Name), value))
	}
	for _, method := range class.Methods {
		if method.Abstract || !method.Class {
			continue
		}
		sym := g.classMethods[name+"."+method.Name]
		g.line(fmt.Sprintf("tya_set_member(%s, %s, tya_bind_method(%s, %s));", target, strconv.Quote(method.Name), target, sym))
	}
	return nil
}

func (g *cgen) parentKey(name string, class *ast.ClassDecl) string {
	if class.Parent == nil {
		return ""
	}
	if class.Parent.Module != "" {
		return class.Parent.Module + "." + class.Parent.Name
	}
	if strings.Contains(name, "_") {
		module := strings.SplitN(name, "_", 2)[0]
		if _, ok := g.classDecls[module+"."+class.Parent.Name]; ok {
			return module + "." + class.Parent.Name
		}
	}
	return class.Parent.Name
}

func (g *cgen) parentExpr(name string, class *ast.ClassDecl) string {
	parent := g.parentKey(name, class)
	if parent == "" {
		return "tya_nil()"
	}
	if strings.Contains(parent, ".") {
		return cName(g.cClassName(parent) + "_class")
	}
	return cName(parent)
}

func (g *cgen) cClassName(key string) string {
	return strings.ReplaceAll(key, ".", "_")
}

func (g *cgen) emitParentDefaults(out *strings.Builder, parentKey string) error {
	parent := g.classDecls[parentKey]
	if parent == nil {
		return nil
	}
	grand := g.parentKey(g.cClassName(parentKey), parent)
	if grand != "" {
		if err := g.emitParentDefaults(out, grand); err != nil {
			return err
		}
	}
	for _, field := range parent.Fields {
		value, _, err := g.expr(field.Value)
		if err != nil {
			return err
		}
		out.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, %s);\n", strconv.Quote("@"+field.Name), value))
		out.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, %s);\n", strconv.Quote(field.Name), value))
	}
	return nil
}

func (g *cgen) emitParentMethods(out *strings.Builder, parentKey string, class *ast.ClassDecl) {
	parent := g.classDecls[parentKey]
	if parent == nil {
		return
	}
	grand := g.parentKey(g.cClassName(parentKey), parent)
	if grand != "" {
		g.emitParentMethods(out, grand, class)
	}
	overrides := map[string]bool{}
	for _, method := range class.Methods {
		if !method.Class {
			overrides[method.Name] = true
		}
	}
	for _, method := range parent.Methods {
		if method.Abstract || method.Class || method.Name == "init" || method.Name == "_init" || strings.HasPrefix(method.Name, "_") || overrides[method.Name] {
			continue
		}
		sym := g.classMethods[g.cClassName(parentKey)+"."+method.Name]
		if sym != "" {
			out.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, tya_bind_method(__obj, %s));\n", strconv.Quote(method.Name), sym))
		}
	}
}

func (g *cgen) inheritedMethodSym(parentKey string, method string) string {
	for parentKey != "" {
		parent := g.classDecls[parentKey]
		if parent == nil {
			return ""
		}
		for _, parentMethod := range parent.Methods {
			if !parentMethod.Class && parentMethod.Name == method {
				return g.classMethods[g.cClassName(parentKey)+"."+method]
			}
		}
		parentKey = g.parentKey(g.cClassName(parentKey), parent)
	}
	return ""
}

func (g *cgen) inheritedClassMethodSym(parentKey string, method string) string {
	for parentKey != "" {
		parent := g.classDecls[parentKey]
		if parent == nil {
			return ""
		}
		for _, parentMethod := range parent.Methods {
			if parentMethod.Class && parentMethod.Name == method {
				return g.classMethods[g.cClassName(parentKey)+"."+method]
			}
		}
		parentKey = g.parentKey(g.cClassName(parentKey), parent)
	}
	return ""
}

func (g *cgen) exprStmt(expr ast.Expr) error {
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
	if id.Name != "print" || len(call.Args) != 1 {
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

func (g *cgen) expr(expr ast.Expr) (string, string, error) {
	switch n := expr.(type) {
	case *ast.IntLit:
		return "tya_number(" + strconv.FormatInt(n.Value, 10) + ")", "TyaValue", nil
	case *ast.FloatLit:
		return "tya_number(" + strconv.FormatFloat(n.Value, 'f', -1, 64) + ")", "TyaValue", nil
	case *ast.StringLit:
		if strings.Contains(n.Value, "{") {
			return g.interpolateString(n.Value), "TyaValue", nil
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
		sym, err := g.emitFunc(name, n)
		if err != nil {
			return "", "", err
		}
		return fmt.Sprintf("tya_function(%s)", sym), "TyaValue", nil
	case *ast.Ident:
		return cName(n.Name), "TyaValue", nil
	case *ast.SelfExpr:
		return "__this", "TyaValue", nil
	case *ast.InstanceFieldExpr:
		return fmt.Sprintf("tya_member(__this, %s)", strconv.Quote("@"+n.Name)), "TyaValue", nil
	case *ast.ClassVarExpr:
		target := g.classTarget()
		if strings.HasPrefix(n.Name, "_") && g.classRef != "" {
			target = g.classRef
		}
		return fmt.Sprintf("tya_member(%s, %s)", target, strconv.Quote(n.Name)), "TyaValue", nil
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
		if _, ok := n.Callee.(*ast.SuperExpr); ok {
			sym := g.inheritedMethodSym(g.superClass, g.methodName)
			if g.inClassMethod {
				sym = g.inheritedClassMethodSym(g.superClass, g.methodName)
			}
			if sym == "" {
				return "tya_nil()", "TyaValue", nil
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
			return fmt.Sprintf("%s(__this, %s)", sym, strings.Join(args[:4], ", ")), "TyaValue", nil
		}
		id, ok := n.Callee.(*ast.Ident)
		if ok {
			if sym, found := g.classes[id.Name]; found {
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
				return fmt.Sprintf("%s(tya_nil(), %s)", sym, strings.Join(args[:4], ", ")), "TyaValue", nil
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
			return "tya_args(argc, argv)", "TyaValue", nil
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
		if ok && id.Name == "panic" && len(n.Args) == 1 {
			message, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("(tya_panic(%s), tya_nil())", message), "TyaValue", nil
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
		if ok && id.Name == "to_int" && len(n.Args) == 1 {
			arg, _, err := g.expr(n.Args[0])
			if err != nil {
				return "", "", err
			}
			return fmt.Sprintf("tya_to_int(%s)", arg), "TyaValue", nil
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
			if strings.HasPrefix(id.Name, "_") && g.inInstanceMethod {
				args := make([]string, 0, len(n.Args))
				for _, arg := range n.Args {
					ex, _, err := g.expr(arg)
					if err != nil {
						return "", "", err
					}
					args = append(args, ex)
				}
				return g.emitDynamicCall(fmt.Sprintf("tya_member(__this, %s)", strconv.Quote(id.Name)), args), "TyaValue", nil
			}
			if sym, found := g.funcs[id.Name]; found {
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
				return fmt.Sprintf("%s(tya_nil(), %s)", sym, strings.Join(args[:4], ", ")), "TyaValue", nil
			}
		}
		if member, ok := n.Callee.(*ast.MemberExpr); ok {
			receiver, _, err := g.expr(member.Target)
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
			case 4:
				return fmt.Sprintf("tya_call4(tya_member(%s, %s), %s, %s, %s, %s)", receiver, strconv.Quote(member.Name), args[0], args[1], args[2], args[3]), "TyaValue", nil
			}
		}
		callee, _, err := g.expr(n.Callee)
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
			return fmt.Sprintf("tya_call1(%s, tya_nil())", callee), "TyaValue", nil
		case 1:
			return fmt.Sprintf("tya_call1(%s, %s)", callee, args[0]), "TyaValue", nil
		case 2:
			return fmt.Sprintf("tya_call2(%s, %s, %s)", callee, args[0], args[1]), "TyaValue", nil
		case 3:
			return fmt.Sprintf("tya_call3(%s, %s, %s, %s)", callee, args[0], args[1], args[2]), "TyaValue", nil
		case 4:
			return fmt.Sprintf("tya_call4(%s, %s, %s, %s, %s)", callee, args[0], args[1], args[2], args[3]), "TyaValue", nil
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
		dict, _, err := g.expr(n.Target)
		if err != nil {
			return "", "", err
		}
		return fmt.Sprintf("tya_member(%s, %s)", dict, strconv.Quote(n.Name)), "TyaValue", nil
	case *ast.TryExpr:
		return g.expr(n.Expr)
	}
	return "", "", fmt.Errorf("C emitter does not support expression %T", expr)
}

func (g *cgen) emitDynamicCall(callee string, args []string) string {
	switch len(args) {
	case 0:
		return fmt.Sprintf("tya_call1(%s, tya_nil())", callee)
	case 1:
		return fmt.Sprintf("tya_call1(%s, %s)", callee, args[0])
	case 2:
		return fmt.Sprintf("tya_call2(%s, %s, %s)", callee, args[0], args[1])
	case 3:
		return fmt.Sprintf("tya_call3(%s, %s, %s, %s)", callee, args[0], args[1], args[2])
	default:
		return fmt.Sprintf("tya_call4(%s, %s, %s, %s, %s)", callee, args[0], args[1], args[2], args[3])
	}
}

func cName(name string) string {
	name = strings.ReplaceAll(name, "?", "_p")
	switch name {
	case "auto", "break", "case", "char", "const", "continue", "default", "do", "double",
		"else", "enum", "extern", "float", "for", "goto", "if", "index", "inline", "int",
		"long", "register", "restrict", "return", "short", "signed", "sizeof",
		"static", "struct", "switch", "typedef", "union", "unsigned", "void",
		"volatile", "while":
		return "tya_var_" + name
	}
	return name
}

func cFuncName(name string, serial int) string {
	return fmt.Sprintf("tya_fn_%s_%d", cName(name), serial)
}

func (g *cgen) interpolateString(value string) string {
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
		if expr, ok := g.interpolationExpr(name); ok {
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

func (g *cgen) interpolationExpr(expr string) (string, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "super()" {
		sym := g.inheritedMethodSym(g.superClass, g.methodName)
		if g.inClassMethod {
			sym = g.inheritedClassMethodSym(g.superClass, g.methodName)
		}
		if sym == "" {
			return "tya_nil()", true
		}
		return fmt.Sprintf("%s(__this, tya_nil(), tya_nil(), tya_nil(), tya_nil())", sym), true
	}
	if strings.HasPrefix(expr, "@@") && isIdentName(strings.TrimPrefix(expr, "@@")) {
		name := strings.TrimPrefix(expr, "@@")
		target := g.classTarget()
		if strings.HasPrefix(name, "_") && g.classRef != "" {
			target = g.classRef
		}
		return "tya_member(" + target + ", " + strconv.Quote(name) + ")", true
	}
	return interpolationExpr(expr)
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
	if strings.HasPrefix(name, "@") && isIdentName(strings.TrimPrefix(name, "@")) {
		return "tya_member(__this, " + strconv.Quote(name) + ")"
	}
	parts := strings.Split(name, ".")
	expr := cName(parts[0])
	for _, part := range parts[1:] {
		expr = "tya_member(" + expr + ", " + strconv.Quote(part) + ")"
	}
	return expr
}

func interpolationExpr(expr string) (string, bool) {
	expr = strings.TrimSpace(expr)
	if strings.HasPrefix(expr, "@") && isIdentName(strings.TrimPrefix(expr, "@")) {
		return pathExpr(expr), true
	}
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
