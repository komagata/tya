package codegen

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"tya/internal/ast"
	"tya/internal/lexer"
	"tya/internal/parser"
)

var classNameRE = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

// currentModulePrefix returns the module prefix for the class
// currently being emitted, or "" when at top level. cgen.className
// is set to "module_ClassName" for module classes via emitClass; we
// recover the module by checking for an underscore separator and
// confirming the suffix matches a known module class entry in
// g.classes (which is shared across child cgens spawned for class
// method bodies). For top-level classes, className has no
// underscore-separated module prefix so this returns "".
func (g *cgen) currentModulePrefix() string {
	if g.className == "" {
		return ""
	}
	if mod, found := g.moduleClasses[g.className]; found {
		return mod
	}
	return ""
}

func EmitC(prog *ast.Program) (string, error) {
	return EmitCWithPath(prog, "")
}

func EmitCWithPath(prog *ast.Program, sourcePath string) (string, error) {
	src, _, err := EmitCWithCoverage(prog, sourcePath, nil)
	return src, err
}

// CoverageOptions enables v0.30 coverage instrumentation. Stdlib,
// .tya/packages/, synthesized test-suite source, and empty paths are
// always excluded; ExcludePaths additions extend that set.
type CoverageOptions struct {
	StdlibDir    string
	PackagesDir  string
	ExcludePaths []string
}

// CoverageEntry describes one registered statement counter.
type CoverageEntry struct {
	ID   int
	File string
	Line int
	Col  int
}

// CoverageRegistry is returned by EmitCWithCoverage when coverage is
// enabled. The runner uses it to emit F/S records into fragment files.
type CoverageRegistry struct {
	Entries []CoverageEntry
}

// EmitCWithCoverage is like EmitCWithPath but emits per-statement
// counter increments when opt is non-nil. When opt is nil, it is
// identical to EmitCWithPath and returns a nil registry.
func EmitCWithCoverage(prog *ast.Program, sourcePath string, opt *CoverageOptions) (string, *CoverageRegistry, error) {
	g := &cgen{vars: map[string]bool{}, funcs: map[string]string{}, classes: map[string]string{}, classMethods: map[string]string{}, classDecls: map[string]*ast.ClassDecl{}, moduleClasses: map[string]string{}, sourcePath: sourcePath}
	if opt != nil {
		g.coverEnabled = true
		g.coverOpt = opt
	}
	g.collectClasses(prog.Stmts)
	globalNames := assignedNames(prog.Stmts)
	for _, name := range globalNames {
		g.vars[name] = true
		g.globalLine(fmt.Sprintf("TyaValue %s;", cName(name)))
	}
	for i, stmt := range prog.Stmts {
		if err := g.stmt(stmt); err != nil {
			return "", nil, err
		}
		// v0.41: emit a safe-point GC trigger between top-level
		// statements. After a top-level statement finishes, any heap
		// values it produced are either stored into a registered global
		// or are no longer referenced; locals used during evaluation are
		// gone. The trigger only collects when allocations have grown
		// past the threshold, so it is cheap when nothing has happened.
		if i < len(prog.Stmts)-1 {
			g.line("tya_gc_maybe_collect();")
		}
	}
	var out strings.Builder
	out.WriteString("#include \"tya_runtime.h\"\n\n")
	if g.coverEnabled {
		out.WriteString("extern void tya_cov_init(int n);\n")
		out.WriteString("extern void tya_cov_inc(int id);\n")
		fmt.Fprintf(&out, "static int tya_cov_n = %d;\n", len(g.coverEntries))
	}
	out.WriteString("static int g_tya_argc = 0;\nstatic char **g_tya_argv = (char **)0;\n\n")
	out.WriteString(g.globalOut.String())
	if g.globalOut.Len() > 0 {
		out.WriteByte('\n')
	}
	out.WriteString(g.funcOut.String())
	out.WriteString("int main(int argc, char **argv) {\n")
	out.WriteString("  g_tya_argc = argc; g_tya_argv = argv;\n")
	if g.coverEnabled {
		out.WriteString("  tya_cov_init(tya_cov_n);\n")
	}
	for _, name := range globalNames {
		fmt.Fprintf(&out, "  tya_gc_register_root(&%s);\n", cName(name))
	}
	out.WriteString(g.out.String())
	out.WriteString("  return 0;\n")
	out.WriteString("}\n")
	var reg *CoverageRegistry
	if g.coverEnabled {
		reg = &CoverageRegistry{Entries: append([]CoverageEntry(nil), g.coverEntries...)}
	}
	return out.String(), reg, nil
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
	// moduleClasses maps a module-class key ("module_ClassName") to
	// its module name. Populated for every v0.44 module class so the
	// within-package fallback in currentModulePrefix and the call
	// emission can resolve unqualified PascalCase references inside
	// sibling method bodies before the constructor sym is filled in
	// to g.classes. Shared across child cgens.
	moduleClasses map[string]string
	temp             int
	inFunc           bool
	inClassMethod    bool
	inInstanceMethod bool
	classRef         string
	className        string
	methodName       string
	superClass       string
	predicateName    string
	raiseDepth       int
	coverEnabled     bool
	coverOpt         *CoverageOptions
	coverEntries     []CoverageEntry
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
	g.instrument(stmt)
	switch n := stmt.(type) {
	case *ast.ImportStmt:
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
		return nil
	case *ast.ModuleDecl:
		g.sourceLine(n.NameTok.Line)
		return g.assignModuleDecl(n)
	case *ast.ClassDecl:
		g.sourceLine(n.NameTok.Line)
		return g.assignClassDecl(n.Name, n)
	case *ast.InterfaceDecl:
		g.sourceLine(n.NameTok.Line)
		g.line("/* interface declaration checked at compile time */")
		return nil
	case *ast.AssignStmt:
		g.sourceLine(n.Tok.Line)
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
			receiver := g.classTarget()
			if strings.HasPrefix(target.Name, "_") && g.classRef != "" {
				receiver = g.classRef
			}
			g.line(fmt.Sprintf("tya_set_member(%s, %s, %s);", receiver, strconv.Quote(target.Name), value))
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
	case *ast.RaiseStmt:
		value, _, err := g.expr(n.Value)
		if err != nil {
			return err
		}
		g.line(fmt.Sprintf("tya_raise(%s);", value))
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
		line, col, _ := stmtPos(stmt)
		return codegenError(codeStmtUnsupported, fmt.Sprintf("C emitter does not support %T", stmt), line, col)
	}
	return nil
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
	g.line("tya_pop_raise_frame();")
	g.indent--
	g.line("} else {")
	g.indent++
	if n.CatchName != "_" {
		name := cName(n.CatchName)
		g.vars[n.CatchName] = true
		g.line(fmt.Sprintf("TyaValue %s = tya_current_raise();", name))
	}
	g.line("tya_pop_raise_frame();")
	for _, stmt := range n.Catch {
		if err := g.stmt(stmt); err != nil {
			return err
		}
	}
	g.indent--
	g.line("}")
	return nil
}

func (g *cgen) matchStmt(n *ast.MatchStmt) error {
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
		for _, stmt := range c.Body {
			if err := g.stmt(stmt); err != nil {
				return err
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
		moduleClasses:    g.moduleClasses,
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
	for i := 0; i < g.raiseDepth; i++ {
		g.line("tya_pop_raise_frame();")
	}
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
	case "push", "delete", "write_file", "exit", "panic", "print", "println":
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
		if (method.Name == "init" || method.Name == "_init" || method.Name == "initialize") && !method.Class {
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
		if method.Abstract || method.Class || method.Name == "init" || method.Name == "_init" || method.Name == "initialize" || !strings.HasPrefix(method.Name, "_") {
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
		if method.Abstract || method.Class || method.Name == "init" || method.Name == "_init" || method.Name == "initialize" {
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
		if method.Abstract || method.Class || method.Name == "init" || method.Name == "_init" || method.Name == "initialize" || strings.HasPrefix(method.Name, "_") || overrides[method.Name] {
			continue
		}
		sym := g.classMethods[g.cClassName(parentKey)+"."+method.Name]
		if sym != "" {
			out.WriteString(fmt.Sprintf("  tya_set_member(__obj, %s, tya_bind_method(__obj, %s));\n", strconv.Quote(method.Name), sym))
		}
	}
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
				return g.classMethods[g.cClassName(parentKey)+"."+method]
			}
		}
		parentKey = g.parentKey(g.cClassName(parentKey), parent)
	}
	return ""
}

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

func (g *cgen) expr(expr ast.Expr) (string, string, error) {
	switch n := expr.(type) {
	case *ast.IntLit:
		return "tya_number(" + strconv.FormatInt(n.Value, 10) + ")", "TyaValue", nil
	case *ast.FloatLit:
		return "tya_number(" + strconv.FormatFloat(n.Value, 'f', -1, 64) + ")", "TyaValue", nil
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
		sym, err := g.emitFunc(name, n)
		if err != nil {
			return "", "", err
		}
		return fmt.Sprintf("tya_function(%s)", sym), "TyaValue", nil
	case *ast.Ident:
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
			expr = fmt.Sprintf("(tya_truthy(%s) ? (%s) : tya_bool(false))", left, right)
		case "||":
			expr = fmt.Sprintf("(tya_truthy(%s) ? tya_bool(true) : (%s))", left, right)
		case "%":
			expr = fmt.Sprintf("tya_number((long)%s.number %% (long)%s.number)", left, right)
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
		if n.Op.Lexeme == "~" {
			return "tya_number(~(long)" + ex + ".number)", "TyaValue", nil
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
			if target, ok := member.Target.(*ast.Ident); ok {
				if call, err := g.standardModuleCall(target.Name, member.Name, n.Args); call != "" || err != nil {
					return call, "TyaValue", err
				}
			}
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
		return fmt.Sprintf("tya_member(%s, %s)", dict, strconv.Quote(n.Name)), "TyaValue", nil
	case *ast.TryExpr:
		return g.expr(n.Expr)
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

func (g *cgen) interpolateString(value string) (string, error) {
	parts := []string{}
	var text strings.Builder
	flushText := func() {
		if text.Len() > 0 {
			parts = append(parts, "tya_string("+strconv.Quote(text.String())+")")
			text.Reset()
		}
	}
	for i := 0; i < len(value); {
		switch value[i] {
		case '{':
			if i+1 < len(value) && value[i+1] == '{' {
				text.WriteByte('{')
				i += 2
				continue
			}
			close := strings.IndexByte(value[i+1:], '}')
			if close < 0 {
				return "", fmt.Errorf("unclosed interpolation")
			}
			expr := strings.TrimSpace(value[i+1 : i+1+close])
			if expr == "" {
				return "", fmt.Errorf("empty interpolation")
			}
			compiled, err := g.interpolationExpr(expr)
			if err != nil {
				return "", err
			}
			flushText()
			parts = append(parts, "tya_to_string("+compiled+")")
			i += close + 2
		case '}':
			if i+1 < len(value) && value[i+1] == '}' {
				text.WriteByte('}')
				i += 2
				continue
			}
			return "", fmt.Errorf("unmatched '}' in string interpolation")
		default:
			text.WriteByte(value[i])
			i++
		}
	}
	flushText()
	if len(parts) == 0 {
		return "tya_string(\"\")", nil
	}
	expr := parts[0]
	for _, part := range parts[1:] {
		expr = "tya_add(" + expr + ", " + part + ")"
	}
	return expr, nil
}

func (g *cgen) interpolationExpr(expr string) (string, error) {
	expr = strings.TrimSpace(expr)
	if expr == "super()" {
		sym := g.inheritedMethodSym(g.superClass, g.methodName)
		if g.inClassMethod {
			sym = g.inheritedClassMethodSym(g.superClass, g.methodName)
		}
		if sym == "" {
			return "tya_nil()", nil
		}
		return fmt.Sprintf("%s(__this, tya_nil(), tya_nil(), tya_nil(), tya_nil())", sym), nil
	}
	toks, errs := lexer.Lex(expr)
	if len(errs) > 0 {
		return "", fmt.Errorf("invalid interpolation expression: %w", errs[0])
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		return "", fmt.Errorf("invalid interpolation expression: %w", err)
	}
	if len(prog.Stmts) != 1 {
		return "", fmt.Errorf("interpolation must contain one expression")
	}
	stmt, ok := prog.Stmts[0].(*ast.ExprStmt)
	if !ok {
		return "", fmt.Errorf("interpolation must contain an expression")
	}
	value, _, err := g.expr(stmt.Expr)
	return value, err
}

func assignedNames(stmts []ast.Stmt) []string {
	seen := map[string]bool{}
	var names []string
	var walk func([]ast.Stmt)
	walk = func(stmts []ast.Stmt) {
		for _, stmt := range stmts {
			switch n := stmt.(type) {
			case *ast.ImportStmt:
				if n.Alias != "" && !seen[n.Alias] {
					seen[n.Alias] = true
					names = append(names, n.Alias)
				}
			case *ast.ModuleDecl:
				if !seen[n.Name] {
					seen[n.Name] = true
					names = append(names, n.Name)
				}
			case *ast.ClassDecl:
				if !seen[n.Name] {
					seen[n.Name] = true
					names = append(names, n.Name)
				}
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
			case *ast.TryCatchStmt:
				walk(n.Try)
				walk(n.Catch)
			case *ast.MatchStmt:
				for _, c := range n.Cases {
					walk(c.Body)
				}
			}
		}
	}
	walk(stmts)
	return names
}

func (g *cgen) standardModuleCall(module string, name string, argExprs []ast.Expr) (string, error) {
	args := make([]string, 0, len(argExprs))
	for _, arg := range argExprs {
		ex, _, err := g.expr(arg)
		if err != nil {
			return "", err
		}
		args = append(args, ex)
	}
	switch module {
	case "string":
		if call := standardStringCall(name, args); call != "" {
			return call, nil
		}
		return "", nil
	case "array":
		if call := standardArrayCall(name, args); call != "" {
			return call, nil
		}
		return "", nil
	case "dict":
		if call := standardDictCall(name, args); call != "" {
			return call, nil
		}
		return "", nil
	case "value":
		if name == "nil?" && len(args) == 1 {
			return fmt.Sprintf("tya_bool(%s.kind == TYA_NIL)", args[0]), nil
		}
		return "", nil
	default:
		return "", nil
	}
}

func standardStringCall(name string, args []string) string {
	switch name {
	case "len", "char_len":
		if len(args) == 1 {
			return fmt.Sprintf("tya_len(%s)", args[0])
		}
	case "byte_len":
		if len(args) == 1 {
			return fmt.Sprintf("tya_byte_len(%s)", args[0])
		}
	case "trim":
		if len(args) == 1 {
			return fmt.Sprintf("tya_trim(%s)", args[0])
		}
	case "contains":
		if len(args) == 2 {
			return fmt.Sprintf("tya_contains(%s, %s)", args[0], args[1])
		}
	case "starts_with":
		if len(args) == 2 {
			return fmt.Sprintf("tya_starts_with(%s, %s)", args[0], args[1])
		}
	case "ends_with":
		if len(args) == 2 {
			return fmt.Sprintf("tya_ends_with(%s, %s)", args[0], args[1])
		}
	case "replace":
		if len(args) == 3 {
			return fmt.Sprintf("tya_replace(%s, %s, %s)", args[0], args[1], args[2])
		}
	case "split":
		if len(args) == 2 {
			return fmt.Sprintf("tya_split(%s, %s)", args[0], args[1])
		}
	case "join":
		if len(args) == 2 {
			return fmt.Sprintf("tya_join(%s, %s)", args[0], args[1])
		}
	case "lines":
		if len(args) == 1 {
			return fmt.Sprintf("tya_lines(%s)", args[0])
		}
	case "upcase":
		if len(args) == 1 {
			return fmt.Sprintf("tya_upcase(%s)", args[0])
		}
	case "downcase":
		if len(args) == 1 {
			return fmt.Sprintf("tya_downcase(%s)", args[0])
		}
	}
	return ""
}

func standardArrayCall(name string, args []string) string {
	switch name {
	case "len":
		if len(args) == 1 {
			return fmt.Sprintf("tya_len(%s)", args[0])
		}
	case "empty?":
		if len(args) == 1 {
			return fmt.Sprintf("tya_bool((int)tya_len(%s).number == 0)", args[0])
		}
	case "first":
		if len(args) == 1 {
			return fmt.Sprintf("tya_first(%s)", args[0])
		}
	case "last":
		if len(args) == 1 {
			return fmt.Sprintf("tya_last(%s)", args[0])
		}
	case "push":
		if len(args) == 2 {
			return fmt.Sprintf("tya_array_push(%s, %s)", args[0], args[1])
		}
	case "pop":
		if len(args) == 1 {
			return fmt.Sprintf("tya_pop(%s)", args[0])
		}
	case "slice":
		if len(args) == 3 {
			return fmt.Sprintf("tya_slice(%s, %s, %s)", args[0], args[1], args[2])
		}
	case "reverse":
		if len(args) == 1 {
			return fmt.Sprintf("tya_reverse(%s)", args[0])
		}
	case "join":
		if len(args) == 2 {
			return fmt.Sprintf("tya_join(%s, %s)", args[0], args[1])
		}
	case "map":
		if len(args) == 2 {
			return fmt.Sprintf("tya_map(%s, %s)", args[0], args[1])
		}
	case "filter":
		if len(args) == 2 {
			return fmt.Sprintf("tya_filter(%s, %s)", args[0], args[1])
		}
	case "find":
		if len(args) == 2 {
			return fmt.Sprintf("tya_find(%s, %s)", args[0], args[1])
		}
	case "any":
		if len(args) == 2 {
			return fmt.Sprintf("tya_any(%s, %s)", args[0], args[1])
		}
	case "all":
		if len(args) == 2 {
			return fmt.Sprintf("tya_all(%s, %s)", args[0], args[1])
		}
	case "each":
		if len(args) == 2 {
			return fmt.Sprintf("tya_each(%s, %s)", args[0], args[1])
		}
	case "reduce":
		if len(args) == 3 {
			return fmt.Sprintf("tya_reduce(%s, %s, %s)", args[0], args[1], args[2])
		}
	}
	return ""
}

func standardDictCall(name string, args []string) string {
	switch name {
	case "len":
		if len(args) == 1 {
			return fmt.Sprintf("tya_len(%s)", args[0])
		}
	case "has", "has?":
		if len(args) == 2 {
			return fmt.Sprintf("tya_has(%s, %s)", args[0], args[1])
		}
	case "get":
		if len(args) == 2 {
			return fmt.Sprintf("tya_dict_get(%s, %s, tya_nil(), false)", args[0], args[1])
		}
		if len(args) == 3 {
			return fmt.Sprintf("tya_dict_get(%s, %s, %s, true)", args[0], args[1], args[2])
		}
	case "set":
		if len(args) == 3 {
			return fmt.Sprintf("tya_dict_set(%s, %s, %s)", args[0], args[1], args[2])
		}
	case "delete":
		if len(args) == 2 {
			return fmt.Sprintf("tya_dict_delete(%s, %s)", args[0], args[1])
		}
	case "keys":
		if len(args) == 1 {
			return fmt.Sprintf("tya_keys(%s)", args[0])
		}
	case "values":
		if len(args) == 1 {
			return fmt.Sprintf("tya_values(%s)", args[0])
		}
	case "merge":
		if len(args) == 2 {
			return fmt.Sprintf("tya_dict_merge(%s, %s)", args[0], args[1])
		}
	}
	return ""
}

func patternBindings(pattern ast.Expr) []string {
	seen := map[string]bool{}
	var names []string
	var walk func(ast.Expr)
	walk = func(pattern ast.Expr) {
		switch n := pattern.(type) {
		case *ast.Ident:
			if n.Name != "_" && !seen[n.Name] {
				seen[n.Name] = true
				names = append(names, n.Name)
			}
		case *ast.ArrayLit:
			for _, elem := range n.Elems {
				walk(elem)
			}
		case *ast.DictLit:
			for _, prop := range n.Props {
				walk(prop.Value)
			}
		}
	}
	walk(pattern)
	return names
}

// v0.24 native builtin codegen (returns C expression or "" if not v0.24).
func v24Codegen(g *cgen, name string, args []ast.Expr) string {
	emit := func(format string, n int) string {
		if len(args) != n {
			return ""
		}
		out := make([]string, n)
		for i, a := range args {
			expr, _, err := g.expr(a)
			if err != nil {
				return ""
			}
			out[i] = expr
		}
		switch n {
		case 0:
			return format
		case 1:
			return fmt.Sprintf(format, out[0])
		case 2:
			return fmt.Sprintf(format, out[0], out[1])
		case 3:
			return fmt.Sprintf(format, out[0], out[1], out[2])
		}
		return ""
	}
	switch name {
	case "time_now":
		return emit("tya_time_now()", 0)
	case "time_sleep":
		return emit("tya_time_sleep(%s)", 1)
	case "time_format":
		if len(args) == 1 {
			return emit("tya_time_format(%s, tya_nil(), false)", 1)
		}
		if len(args) == 2 {
			a, _, err := g.expr(args[0])
			if err != nil {
				return ""
			}
			b, _, err := g.expr(args[1])
			if err != nil {
				return ""
			}
			return fmt.Sprintf("tya_time_format(%s, %s, true)", a, b)
		}
	case "time_parse":
		return emit("tya_time_parse(%s)", 1)
	case "time_since":
		return emit("tya_time_since(%s)", 1)
	case "random_seed":
		return emit("tya_random_seed(%s)", 1)
	case "random_int":
		return emit("tya_random_int(%s, %s)", 2)
	case "random_float":
		return emit("tya_random_float()", 0)
	case "math_sqrt":
		return emit("tya_math_sqrt(%s)", 1)
	case "math_pow":
		return emit("tya_math_pow(%s, %s)", 2)
	case "math_floor":
		return emit("tya_math_floor(%s)", 1)
	case "math_ceil":
		return emit("tya_math_ceil(%s)", 1)
	case "math_round":
		return emit("tya_math_round(%s)", 1)
	case "math_trunc":
		return emit("tya_math_trunc(%s)", 1)
	case "math_log":
		return emit("tya_math_log(%s)", 1)
	case "math_log2":
		return emit("tya_math_log2(%s)", 1)
	case "math_log10":
		return emit("tya_math_log10(%s)", 1)
	case "math_exp":
		return emit("tya_math_exp(%s)", 1)
	case "math_sin":
		return emit("tya_math_sin(%s)", 1)
	case "math_cos":
		return emit("tya_math_cos(%s)", 1)
	case "math_tan":
		return emit("tya_math_tan(%s)", 1)
	case "math_asin":
		return emit("tya_math_asin(%s)", 1)
	case "math_acos":
		return emit("tya_math_acos(%s)", 1)
	case "math_atan":
		return emit("tya_math_atan(%s)", 1)
	case "math_atan2":
		return emit("tya_math_atan2(%s, %s)", 2)
	case "process_run":
		if len(args) == 1 {
			a, _, err := g.expr(args[0])
			if err != nil {
				return ""
			}
			return fmt.Sprintf("tya_process_run(%s, tya_nil())", a)
		}
		if len(args) == 2 {
			return emit("tya_process_run(%s, %s)", 2)
		}
	case "digest_md5":
		return emit("tya_digest_md5(%s)", 1)
	case "digest_sha1":
		return emit("tya_digest_sha1(%s)", 1)
	case "digest_sha256":
		return emit("tya_digest_sha256(%s)", 1)
	case "digest_sha384":
		return emit("tya_digest_sha384(%s)", 1)
	case "digest_sha512":
		return emit("tya_digest_sha512(%s)", 1)
	case "secure_random_bytes":
		return emit("tya_secure_random_bytes(%s)", 1)
	case "secure_random_int":
		return emit("tya_secure_random_int(%s, %s)", 2)
	case "runtime_gc_stats":
		return emit("tya_gc_stats()", 0)
	case "runtime_gc_collect":
		return "(tya_gc_collect(), tya_nil())"
	case "channel_new":
		return emit("tya_channel_new(%s)", 1)
	case "channel_send":
		return emit("tya_channel_send(%s, %s)", 2)
	case "channel_receive":
		return emit("tya_channel_receive(%s)", 1)
	case "channel_receive_timeout":
		return emit("tya_channel_receive_timeout(%s, %s)", 2)
	case "channel_close":
		return emit("tya_channel_close(%s)", 1)
	case "channel_closed_p":
		return emit("tya_channel_closed(%s)", 1)
	case "channel_select":
		return emit("tya_channel_select(%s)", 1)
	case "task_cancel":
		return emit("tya_task_cancel(%s)", 1)
	case "task_is_cancelled_p":
		return emit("tya_task_is_cancelled(%s)", 1)
	case "task_current":
		return emit("tya_current_task()", 0)
	case "sync_mutex_new":
		return emit("tya_sync_mutex_new()", 0)
	case "sync_lock":
		return emit("tya_sync_lock(%s)", 1)
	case "sync_unlock":
		return emit("tya_sync_unlock(%s)", 1)
	case "sync_atomic_integer_new":
		return emit("tya_sync_atomic_integer_new(%s)", 1)
	case "sync_atomic_integer_add":
		return emit("tya_sync_atomic_integer_add(%s, %s)", 2)
	case "sync_atomic_integer_load":
		return emit("tya_sync_atomic_integer_load(%s)", 1)
	case "sync_atomic_integer_store":
		return emit("tya_sync_atomic_integer_store(%s, %s)", 2)
	case "sync_atomic_integer_cas":
		return emit("tya_sync_atomic_integer_cas(%s, %s, %s)", 3)
	case "sync_wait_group_new":
		return emit("tya_sync_wait_group_new()", 0)
	case "sync_wait_group_add":
		return emit("tya_sync_wait_group_add(%s, %s)", 2)
	case "sync_wait_group_done":
		return emit("tya_sync_wait_group_done(%s)", 1)
	case "sync_wait_group_wait":
		return emit("tya_sync_wait_group_wait(%s)", 1)
	case "bytes":
		return emit("tya_bytes_from_array(%s)", 1)
	case "bytes_of":
		return emit("tya_bytes_of(%s)", 1)
	case "bytes_text":
		return emit("tya_bytes_text(%s)", 1)
	case "bytes_array":
		return emit("tya_bytes_array(%s)", 1)
	case "bytes_concat":
		return emit("tya_bytes_concat(%s, %s)", 2)
	case "bytes_slice":
		return emit("tya_bytes_slice(%s, %s, %s)", 3)
	case "file_read_bytes":
		return emit("tya_file_read_bytes(%s)", 1)
	case "file_write_bytes":
		return emit("tya_file_write_bytes(%s, %s)", 2)
	}
	return ""
}
