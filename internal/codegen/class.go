package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"tya/internal/ast"
)

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
	g.line(fmt.Sprintf("tya_set_member(%s, %s, tya_bool(true));", target, strconv.Quote("__module_namespace")))
	for _, member := range functions {
		fn := member.Value.(*ast.FuncLit)
		funcName := module.Name + "_" + member.Name
		sym, err := g.emitFunc(funcName, fn)
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_set_member(%s, %s, tya_function_params(%s, %s));", target, strconv.Quote(member.Name), sym, cParamArray(fn.Params)))
	}
	// Pre-register every module class so the within-package fallback
	// in expr (Ident) and CallExpr can resolve unqualified references
	// between sibling class method bodies emitted in any order. The
	// constructor sym slot is filled in after emitClass returns;
	// before that, the entry serves as a marker that this name is a
	// known module class.
	for _, class := range classes {
		g.vars[module.Name+"_"+class.Name+"_class"] = true
		g.moduleClasses[module.Name+"_"+class.Name] = module.Name
	}
	for _, class := range classes {
		classTarget := cName(module.Name + "_" + class.Name + "_class")
		sym, err := g.emitClass(module.Name+"_"+class.Name, class, classTarget)
		if err != nil {
			return err
		}
		// Record the constructor symbol under the keyed-name form so
		// the within-package CallExpr fallback can resolve
		// unqualified PascalCase calls to the module-class
		// constructor.
		g.classes[module.Name+"_"+class.Name] = sym
		g.globalLine(fmt.Sprintf("TyaValue %s;", classTarget))
		g.line(fmt.Sprintf("%s = tya_class(%s, %s, %s);", classTarget, sym, strconv.Quote(displayClassName(class.Name)), g.parentExpr(module.Name+"_"+class.Name, class)))
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
		g.line(fmt.Sprintf("%s = tya_class(%s, %s, %s);", target, sym, strconv.Quote(displayClassName(name)), g.parentExpr(name, class)))
	} else {
		g.vars[name] = true
		g.globalLine(fmt.Sprintf("TyaValue %s;", target))
		g.line(fmt.Sprintf("%s = tya_class(%s, %s, %s);", target, sym, strconv.Quote(displayClassName(name)), g.parentExpr(name, class)))
	}
	return g.emitClassMembers(target, name, class)
}

func (g *cgen) emitClass(name string, class *ast.ClassDecl, classRef string) (string, error) {
	methodSyms := map[string]string{}
	collidingMethodNames := classMethodNameCollisions(class)
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
		emitName := name + "_" + method.Name
		if method.Class && collidingMethodNames[method.Name] {
			emitName = name + "_class_" + method.Name
		}
		sym, err := g.emitFuncWithContext(emitName, method.Func, classRef, methodKind)
		g.className, g.methodName, g.superClass = prevClass, prevMethod, prevSuper
		if err != nil {
			return "", err
		}
		methodSyms[methodSymbolKey(*method)] = sym
		if method.Class {
			g.classMethods[name+".class."+method.Name] = sym
			g.methodParams[name+".class."+method.Name] = append([]string(nil), method.Func.Params...)
		} else {
			g.classMethods[name+"."+method.Name] = sym
			g.methodParams[name+"."+method.Name] = append([]string(nil), method.Func.Params...)
		}
		if (method.Name == "init" || method.Name == "_init" || method.Name == "initialize") && !method.Class {
			initMethod = method
			g.classParams[name] = append([]string(nil), method.Func.Params...)
		}
	}
	prevClass, prevClassRef := g.className, g.classRef
	g.className, g.classRef = name, classRef
	defer func() {
		g.className, g.classRef = prevClass, prevClassRef
	}()
	sym := cFuncName(name+"_new", g.temp)
	g.temp++
	var out strings.Builder
	out.WriteString("TyaValue ")
	out.WriteString(sym)
	out.WriteString("(TyaValue __this, TyaValue __arg0, TyaValue __arg1, TyaValue __arg2, TyaValue __arg3, TyaValue __arg4, TyaValue __arg5) {\n")
	out.WriteString("  (void)__this;\n")
	out.WriteString("  TyaValue __obj = tya_object();\n")
	out.WriteString(fmt.Sprintf("  tya_set_member(__obj, \"class\", %s);\n", classRef))
	out.WriteString(fmt.Sprintf("  tya_set_member(__obj, \"class_name\", tya_string(%s));\n", strconv.Quote(displayClassName(class.Name))))
	if parentKey != "" {
		if err := g.emitParentDefaults(&out, parentKey); err != nil {
			return "", err
		}
	}
	if err := g.emitInterfaceFields(&out, name, class); err != nil {
		return "", err
	}
	for _, field := range class.Fields {
		value, _, err := g.expr(field.Value)
		if err != nil {
			return "", err
		}
		out.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, %s);\n", strconv.Quote("@"+field.Name), value))
		out.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, %s);\n", strconv.Quote(field.Name), value))
	}
	if parentKey != "" {
		g.emitParentMethods(&out, parentKey, class)
	}
	if err := g.emitInterfaceMethods(&out, name, class); err != nil {
		return "", err
	}
	for _, method := range class.Methods {
		if method.Abstract || method.Class || method.Name == "init" || method.Name == "_init" || method.Name == "initialize" {
			continue
		}
		out.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, tya_bind_method_params(__obj, %s, %s));\n", strconv.Quote(method.Name), methodSyms[methodSymbolKey(method)], cParamArray(method.Func.Params)))
	}
	if initMethod != nil {
		out.WriteString(fmt.Sprintf("  (void)%s(__obj, __arg0, __arg1, __arg2, __arg3, __arg4, __arg5);\n", methodSyms[methodSymbolKey(*initMethod)]))
	} else if parentKey != "" {
		if parentInit := g.inheritedMethodSym(parentKey, "init"); parentInit != "" {
			out.WriteString(fmt.Sprintf("  (void)%s(__obj, __arg0, __arg1, __arg2, __arg3, __arg4, __arg5);\n", parentInit))
		} else {
			if err := g.emitInterfaceInitializers(&out, name, class); err != nil {
				return "", err
			}
			out.WriteString("  (void)__arg0;\n  (void)__arg1;\n  (void)__arg2;\n  (void)__arg3;\n")
		}
	} else {
		if err := g.emitInterfaceInitializers(&out, name, class); err != nil {
			return "", err
		}
		out.WriteString("  (void)__arg0;\n  (void)__arg1;\n  (void)__arg2;\n  (void)__arg3;\n")
	}
	if parentKey != "" {
		g.emitParentMethods(&out, parentKey, class)
	}
	if err := g.emitInterfaceMethods(&out, name, class); err != nil {
		return "", err
	}
	for _, method := range class.Methods {
		if method.Abstract || method.Class || method.Name == "init" || method.Name == "_init" || method.Name == "initialize" {
			continue
		}
		out.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, tya_bind_method_params(__obj, %s, %s));\n", strconv.Quote(method.Name), methodSyms[methodSymbolKey(method)], cParamArray(method.Func.Params)))
	}
	out.WriteString("  return __obj;\n")
	out.WriteString("}\n\n")
	g.funcOut.WriteString(out.String())
	return sym, nil
}

func methodSymbolKey(method ast.ClassMethod) string {
	if method.Class {
		return "class:" + method.Name
	}
	return "instance:" + method.Name
}

func classMethodNameCollisions(class *ast.ClassDecl) map[string]bool {
	instance := map[string]bool{}
	classSide := map[string]bool{}
	for _, method := range class.Methods {
		if method.Abstract {
			continue
		}
		if method.Class {
			classSide[method.Name] = true
		} else {
			instance[method.Name] = true
		}
	}
	out := map[string]bool{}
	for name := range instance {
		if classSide[name] {
			out[name] = true
		}
	}
	return out
}

func (g *cgen) emitClassMembers(target string, name string, class *ast.ClassDecl) error {
	testMethods := []string{}
	hasSetup := false
	hasTeardown := false
	for _, method := range class.Methods {
		if method.Class || method.Abstract {
			continue
		}
		if strings.HasPrefix(method.Name, "test_") {
			testMethods = append(testMethods, method.Name)
		}
		if method.Name == "setup" {
			hasSetup = true
		}
		if method.Name == "teardown" {
			hasTeardown = true
		}
	}
	methodValues := make([]string, 0, len(testMethods))
	for _, method := range testMethods {
		methodValues = append(methodValues, fmt.Sprintf("tya_string(%s)", strconv.Quote(method)))
	}
	if len(methodValues) == 0 {
		g.line(fmt.Sprintf("tya_set_member(%s, %s, tya_array((TyaValue*)0, 0));", target, strconv.Quote("unittest_test_methods")))
	} else {
		g.line(fmt.Sprintf("tya_set_member(%s, %s, tya_array((TyaValue[]){%s}, %d));", target, strconv.Quote("unittest_test_methods"), strings.Join(methodValues, ", "), len(methodValues)))
	}
	setupValue := "tya_bool(0)"
	if hasSetup {
		setupValue = "tya_bool(1)"
	}
	teardownValue := "tya_bool(0)"
	if hasTeardown {
		teardownValue = "tya_bool(1)"
	}
	g.line(fmt.Sprintf("tya_set_member(%s, %s, %s);", target, strconv.Quote("unittest_has_setup"), setupValue))
	g.line(fmt.Sprintf("tya_set_member(%s, %s, %s);", target, strconv.Quote("unittest_has_teardown"), teardownValue))
	for _, constant := range class.Constants {
		value, _, err := g.expr(constant.Value)
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_set_member(%s, %s, %s);", target, strconv.Quote(constant.Name), value))
	}
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
		sym := g.classMethods[name+".class."+method.Name]
		g.line(fmt.Sprintf("tya_set_member(%s, %s, tya_bind_method_params(%s, %s, %s));", target, strconv.Quote(method.Name), target, sym, cParamArray(method.Func.Params)))
	}
	return nil
}

func (g *cgen) parentKey(name string, class *ast.ClassDecl) string {
	if class.Parent == nil {
		return ""
	}
	if class.Parent.Module != "" {
		if key := aliasedPackageSymbolKey(class.Parent.Module, class.Parent.Name); g.classDecls[key] != nil {
			return key
		}
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

func displayClassName(name string) string {
	if i := strings.Index(name, "TyaPkg"); i > 0 {
		return name[:i]
	}
	return name
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
		if method.Abstract || method.Class || method.Name == "init" || method.Name == "_init" || method.Name == "initialize" || overrides[method.Name] {
			continue
		}
		sym := g.classMethods[g.cClassName(parentKey)+"."+method.Name]
		if sym != "" {
			out.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, tya_bind_method_params(__obj, %s, %s));\n", strconv.Quote(method.Name), sym, cParamArray(method.Func.Params)))
		}
	}
}

func (g *cgen) interfaceKey(className string, ref ast.ClassRef) string {
	if ref.Module != "" {
		if key := aliasedPackageSymbolKey(ref.Module, ref.Name); g.interfaceDecls[key] != nil {
			return key
		}
		return ref.Module + "." + ref.Name
	}
	if strings.Contains(className, "_") {
		module := strings.SplitN(className, "_", 2)[0]
		if _, ok := g.interfaceDecls[module+"."+ref.Name]; ok {
			return module + "." + ref.Name
		}
	}
	return ref.Name
}

func aliasedPackageSymbolKey(module, name string) string {
	return name + "TyaPkg" + pascalIdentifier(module)
}

func pascalIdentifier(name string) string {
	var out strings.Builder
	capNext := true
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if capNext && r >= 'a' && r <= 'z' {
				r = r - 'a' + 'A'
			}
			out.WriteRune(r)
			capNext = false
			continue
		}
		capNext = true
	}
	if out.Len() == 0 {
		return "Package"
	}
	return out.String()
}

func (g *cgen) effectiveInterfaces(className string, class *ast.ClassDecl) []*ast.InterfaceDecl {
	var out []*ast.InterfaceDecl
	seen := map[string]bool{}
	var visit func(string)
	visit = func(key string) {
		if seen[key] {
			return
		}
		iface := g.interfaceDecls[key]
		if iface == nil {
			return
		}
		for _, parent := range iface.Parents {
			visit(g.interfaceKey(strings.ReplaceAll(key, ".", "_"), parent))
		}
		seen[key] = true
		out = append(out, iface)
	}
	for _, ref := range class.Implements {
		visit(g.interfaceKey(className, ref))
	}
	return out
}

func (g *cgen) emitInterfaceFields(out *strings.Builder, className string, class *ast.ClassDecl) error {
	classFields := map[string]bool{}
	for _, field := range class.Fields {
		classFields[field.Name] = true
	}
	for _, iface := range g.effectiveInterfaces(className, class) {
		for _, field := range iface.Fields {
			if classFields[field.Name] {
				continue
			}
			value, _, err := g.expr(field.Value)
			if err != nil {
				return err
			}
			out.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, %s);\n", strconv.Quote("@"+field.Name), value))
			out.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, %s);\n", strconv.Quote(field.Name), value))
		}
	}
	return nil
}

func (g *cgen) interfaceMethodSym(key string, iface *ast.InterfaceDecl, method ast.InterfaceMethod) (string, error) {
	return g.interfaceMethodSymWithNext(key, iface, method, "")
}

func (g *cgen) interfaceMethodSymWithNext(key string, iface *ast.InterfaceDecl, method ast.InterfaceMethod, nextSym string) (string, error) {
	cacheKey := "interface:" + key + "." + method.Name + ":" + nextSym
	if sym := g.classMethods[cacheKey]; sym != "" {
		return sym, nil
	}
	prevClass, prevMethod, prevSuper, prevInterfaceSuper := g.className, g.methodName, g.superClass, g.interfaceSuperSym
	g.className, g.methodName, g.superClass, g.interfaceSuperSym = key, method.Name, "", nextSym
	sym, err := g.emitFuncWithContext(strings.ReplaceAll(key, ".", "_")+"_"+method.Name, method.Func, "", "instance")
	g.className, g.methodName, g.superClass, g.interfaceSuperSym = prevClass, prevMethod, prevSuper, prevInterfaceSuper
	if err != nil {
		return "", err
	}
	g.classMethods[cacheKey] = sym
	return sym, nil
}

func (g *cgen) emitInterfaceMethods(out *strings.Builder, className string, class *ast.ClassDecl) error {
	overrides := map[string]bool{}
	for _, method := range class.Methods {
		if !method.Class {
			overrides[method.Name] = true
		}
	}
	ifaces := g.effectiveInterfaces(className, class)
	for i, iface := range ifaces {
		key := g.interfaceDeclKey(iface)
		for _, method := range iface.Methods {
			if method.Func == nil || method.Name == "initialize" || overrides[method.Name] {
				continue
			}
			nextSym, err := g.interfaceDefaultSymInStack(ifaces, i-1, method.Name)
			if err != nil {
				return err
			}
			sym, err := g.interfaceMethodSymWithNext(key, iface, method, nextSym)
			if err != nil {
				return err
			}
			out.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, tya_bind_method_params(__obj, %s, %s));\n", strconv.Quote(method.Name), sym, cParamArray(method.Func.Params)))
		}
	}
	return nil
}

func (g *cgen) interfaceDeclKey(iface *ast.InterfaceDecl) string {
	key := iface.Name
	for k, v := range g.interfaceDecls {
		if v == iface {
			key = k
			break
		}
	}
	return key
}

func (g *cgen) interfaceDefaultSymInStack(ifaces []*ast.InterfaceDecl, maxIndex int, methodName string) (string, error) {
	for i := maxIndex; i >= 0; i-- {
		iface := ifaces[i]
		key := g.interfaceDeclKey(iface)
		for _, method := range iface.Methods {
			if method.Name != methodName || method.Func == nil || method.Name == "initialize" {
				continue
			}
			nextSym, err := g.interfaceDefaultSymInStack(ifaces, i-1, methodName)
			if err != nil {
				return "", err
			}
			return g.interfaceMethodSymWithNext(key, iface, method, nextSym)
		}
	}
	return "", nil
}

func (g *cgen) emitInterfaceInitializers(out *strings.Builder, className string, class *ast.ClassDecl) error {
	for _, iface := range g.effectiveInterfaces(className, class) {
		key := g.interfaceDeclKey(iface)
		for _, method := range iface.Methods {
			if method.Name != "initialize" || method.Func == nil {
				continue
			}
			sym, err := g.interfaceMethodSym(key, iface, method)
			if err != nil {
				return err
			}
			out.WriteString(fmt.Sprintf("  (void)%s(__obj, tya_missing(), tya_missing(), tya_missing(), tya_missing(), tya_missing(), tya_missing());\n", sym))
		}
	}
	return nil
}

func (g *cgen) classHasInterfaceInitializers(className string, class *ast.ClassDecl) bool {
	for _, iface := range g.effectiveInterfaces(className, class) {
		for _, method := range iface.Methods {
			if method.Name == "initialize" && method.Func != nil {
				return true
			}
		}
	}
	return false
}

func (g *cgen) interfaceDefaultSymForCurrentClass(methodName string) (string, error) {
	class := g.classDecls[g.className]
	className := g.className
	if class == nil {
		if mod := g.currentModulePrefix(); mod != "" {
			key := mod + "." + strings.TrimPrefix(g.className, mod+"_")
			class = g.classDecls[key]
			className = key
		}
	}
	if class == nil {
		return "", nil
	}
	ifaces := g.effectiveInterfaces(className, class)
	return g.interfaceDefaultSymInStack(ifaces, len(ifaces)-1, methodName)
}

func (g *cgen) interfaceInitializerRunnerForCurrentClass() (string, error) {
	class := g.classDecls[g.className]
	className := g.className
	if class == nil {
		if mod := g.currentModulePrefix(); mod != "" {
			key := mod + "." + strings.TrimPrefix(g.className, mod+"_")
			class = g.classDecls[key]
			className = key
		}
	}
	if class == nil {
		return "", nil
	}
	cacheKey := "interface-init:" + className
	if sym := g.classMethods[cacheKey]; sym != "" {
		return sym, nil
	}
	sym := cFuncName(strings.ReplaceAll(className, ".", "_")+"_interface_initialize", g.temp)
	g.temp++
	var out strings.Builder
	out.WriteString("TyaValue ")
	out.WriteString(sym)
	out.WriteString("(TyaValue __this, TyaValue __arg0, TyaValue __arg1, TyaValue __arg2, TyaValue __arg3, TyaValue __arg4, TyaValue __arg5) {\n")
	out.WriteString("  (void)__arg0;\n  (void)__arg1;\n  (void)__arg2;\n  (void)__arg3;\n  (void)__arg4;\n  (void)__arg5;\n")
	for _, iface := range g.effectiveInterfaces(className, class) {
		key := g.interfaceDeclKey(iface)
		for _, method := range iface.Methods {
			if method.Name != "initialize" || method.Func == nil {
				continue
			}
			initSym, err := g.interfaceMethodSym(key, iface, method)
			if err != nil {
				return "", err
			}
			out.WriteString(fmt.Sprintf("  (void)%s(__this, tya_missing(), tya_missing(), tya_missing(), tya_missing(), tya_missing(), tya_missing());\n", initSym))
		}
	}
	out.WriteString("  return tya_nil();\n")
	out.WriteString("}\n\n")
	g.funcOut.WriteString(out.String())
	g.classMethods[cacheKey] = sym
	return sym, nil
}

func (g *cgen) constructorSuperRunnerForCurrentClass(parentSym string) (string, error) {
	class := g.classDecls[g.className]
	className := g.className
	if class == nil {
		if mod := g.currentModulePrefix(); mod != "" {
			key := mod + "." + strings.TrimPrefix(g.className, mod+"_")
			class = g.classDecls[key]
			className = key
		}
	}
	if class == nil || !g.classHasInterfaceInitializers(className, class) {
		return parentSym, nil
	}
	cacheKey := "constructor-super:" + className + ":" + parentSym
	if sym := g.classMethods[cacheKey]; sym != "" {
		return sym, nil
	}
	sym := cFuncName(strings.ReplaceAll(className, ".", "_")+"_constructor_super", g.temp)
	g.temp++
	var out strings.Builder
	out.WriteString("TyaValue ")
	out.WriteString(sym)
	out.WriteString("(TyaValue __this, TyaValue __arg0, TyaValue __arg1, TyaValue __arg2, TyaValue __arg3, TyaValue __arg4, TyaValue __arg5) {\n")
	if parentSym != "" {
		out.WriteString(fmt.Sprintf("  (void)%s(__this, __arg0, __arg1, __arg2, __arg3, __arg4, __arg5);\n", parentSym))
	} else {
		out.WriteString("  (void)__arg0;\n  (void)__arg1;\n  (void)__arg2;\n  (void)__arg3;\n  (void)__arg4;\n  (void)__arg5;\n")
	}
	for _, iface := range g.effectiveInterfaces(className, class) {
		key := g.interfaceDeclKey(iface)
		for _, method := range iface.Methods {
			if method.Name != "initialize" || method.Func == nil {
				continue
			}
			initSym, err := g.interfaceMethodSym(key, iface, method)
			if err != nil {
				return "", err
			}
			out.WriteString(fmt.Sprintf("  (void)%s(__this, tya_missing(), tya_missing(), tya_missing(), tya_missing(), tya_missing(), tya_missing());\n", initSym))
		}
	}
	out.WriteString("  return tya_nil();\n")
	out.WriteString("}\n\n")
	g.funcOut.WriteString(out.String())
	g.classMethods[cacheKey] = sym
	return sym, nil
}

func (g *cgen) inheritedMethodSym(parentKey string, method string) string {
	// v0.46 G5: "init" and "initialize" are constructor aliases.
	// When searching for the parent's constructor, accept either
	// spelling.
	wantAliases := []string{method}
	if method == "init" {
		wantAliases = []string{"init", "initialize", "_init"}
	} else if method == "initialize" {
		wantAliases = []string{"initialize", "init", "_init"}
	}
	for parentKey != "" {
		parent := g.classDecls[parentKey]
		if parent == nil {
			return ""
		}
		for _, parentMethod := range parent.Methods {
			if parentMethod.Class {
				continue
			}
			for _, want := range wantAliases {
				if parentMethod.Name == want {
					return g.classMethods[g.cClassName(parentKey)+"."+parentMethod.Name]
				}
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
				return g.classMethods[g.cClassName(parentKey)+".class."+method]
			}
		}
		parentKey = g.parentKey(g.cClassName(parentKey), parent)
	}
	return ""
}
