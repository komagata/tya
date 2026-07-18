package codegen

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"tya/internal/ast"
	"tya/internal/diag"
	"tya/internal/pkg"
)

var classNameRE = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

var primitiveClassNames = map[string]bool{
	"Number":  true,
	"String":  true,
	"Array":   true,
	"Dict":    true,
	"Boolean": true,
	"Nil":     true,
}

var removedPrimitiveHelperNames = map[string]bool{
	"kind": true, "len": true, "byte_len": true, "char_len": true,
	"trim": true, "contains": true, "index_of": true, "starts_with": true, "ends_with": true,
	"replace": true, "split": true, "join": true, "keys": true,
	"values": true, "has": true, "push": true, "pop": true,
	"map": true, "filter": true, "find": true, "any": true,
	"all": true, "reduce": true, "to_string": true, "to_int": true,
	"to_float": true, "to_number": true,
}

var removedPrimitiveModuleNames = map[string]bool{
	"string": true,
	"array":  true,
	"dict":   true,
	"value":  true,
}

var nativeFunctions map[string]pkg.NativeFunction

func SetNativeFunctions(functions map[string]pkg.NativeFunction) func() {
	prev := nativeFunctions
	if len(functions) == 0 {
		nativeFunctions = nil
	} else {
		nativeFunctions = map[string]pkg.NativeFunction{}
		for name, fn := range functions {
			nativeFunctions[name] = fn
		}
	}
	return func() { nativeFunctions = prev }
}

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

func (g *cgen) currentClassHasConstant(name string) bool {
	if g.className == "" {
		return false
	}
	class := g.classDecls[g.className]
	if class == nil {
		return false
	}
	for _, constant := range class.Constants {
		if constant.Name == name {
			return true
		}
	}
	return false
}

// EmitC compiles prog to a self-contained C translation unit.
// Returns (csource, diags, err). Codegen is fail-fast in v0.56:
// diags is nil on a clean emit, or carries a single entry copied
// from err.(*CodegenError).Diags on failure. err continues to
// satisfy errors.As(&CodegenError) for callers that prefer the
// wrapper.
func EmitC(prog *ast.Program) (string, []diag.Diagnostic, error) {
	return EmitCWithPath(prog, "")
}

func EmitCWithPath(prog *ast.Program, sourcePath string) (string, []diag.Diagnostic, error) {
	src, _, diags, err := EmitCWithCoverage(prog, sourcePath, nil)
	return src, diags, err
}

// CoverageOptions enables v0.30 coverage instrumentation. Stdlib,
// .tya/packages/, synthesized test-suite source, and empty paths are
// always excluded; ExcludePaths additions extend that set.
type CoverageOptions struct {
	StdlibDir    string
	PackagesDir  string
	ExcludePaths []string
	Include      []string
	Exclude      []string
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
//
// Returns (csource, registry, diags, err). diags is nil on success,
// or the same payload as err.(*CodegenError).Diags on failure.
func EmitCWithCoverage(prog *ast.Program, sourcePath string, opt *CoverageOptions) (string, *CoverageRegistry, []diag.Diagnostic, error) {
	g := &cgen{vars: map[string]bool{}, funcs: map[string]string{}, funcParams: map[string][]string{}, classes: map[string]string{}, classParams: map[string][]string{}, classMethods: map[string]string{}, methodParams: map[string][]string{}, classDecls: map[string]*ast.ClassDecl{}, structDecls: map[string]*ast.StructDecl{}, structWithSyms: map[string]string{}, interfaceDecls: map[string]*ast.InterfaceDecl{}, moduleClasses: map[string]string{}, sourcePath: sourcePath, gcSafeLoops: true}
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
	var collected []diag.Diagnostic
	for i, stmt := range prog.Stmts {
		if err := g.stmt(stmt); err != nil {
			if ce, ok := AsCodegenError(err); ok {
				collected = append(collected, ce.Diags...)
			} else {
				return "", nil, diagsFromErr(err), err
			}
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
	if len(collected) > 0 {
		diags := append([]diag.Diagnostic(nil), collected...)
		return "", nil, diags, &CodegenError{Diags: diags}
	}
	var out strings.Builder
	out.WriteString("#include \"tya_runtime.h\"\n\n")
	if g.coverEnabled {
		out.WriteString("extern void tya_cov_init(int n);\n")
		out.WriteString("extern void tya_cov_inc(int id);\n")
		fmt.Fprintf(&out, "static int tya_cov_n = %d;\n", len(g.coverEntries))
	}
	for _, name := range sortedNativeFunctionNames() {
		fn := nativeFunctions[name]
		fmt.Fprintf(&out, "extern TyaValue %s(TyaValue __this, TyaValue a0, TyaValue a1, TyaValue a2, TyaValue a3);\n", fn.Symbol)
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
	return out.String(), reg, nil, nil
}

func sortedNativeFunctionNames() []string {
	names := make([]string, 0, len(nativeFunctions))
	for name := range nativeFunctions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// diagsFromErr lifts the []diag.Diagnostic payload out of err
// when err is a *CodegenError, falling back to a synthetic
// single-entry slice for plain errors so callers still get a
// non-nil diags slice on failure.
func diagsFromErr(err error) []diag.Diagnostic {
	if ce, ok := AsCodegenError(err); ok {
		return append([]diag.Diagnostic{}, ce.Diags...)
	}
	if err == nil {
		return nil
	}
	return []diag.Diagnostic{{
		Severity: diag.Error,
		Message:  err.Error(),
		Source:   "tya",
	}}
}

type cgen struct {
	out            strings.Builder
	globalOut      strings.Builder
	funcOut        strings.Builder
	sourcePath     string
	indent         int
	vars           map[string]bool
	funcs          map[string]string
	funcParams     map[string][]string
	classes        map[string]string
	classParams    map[string][]string
	classMethods   map[string]string
	methodParams   map[string][]string
	classDecls     map[string]*ast.ClassDecl
	structDecls    map[string]*ast.StructDecl
	structWithSyms map[string]string
	interfaceDecls map[string]*ast.InterfaceDecl
	// moduleClasses maps a module-class key ("module_ClassName") to
	// its module name. Populated for every v0.44 module class so the
	// within-package fallback in currentModulePrefix and the call
	// emission can resolve unqualified PascalCase references inside
	// sibling method bodies before the constructor sym is filled in
	// to g.classes. Shared across child cgens.
	moduleClasses     map[string]string
	temp              int
	inFunc            bool
	inClassMethod     bool
	inInstanceMethod  bool
	classRef          string
	className         string
	methodName        string
	superClass        string
	interfaceSuperSym string
	predicateName     string
	closureVars       map[string]bool
	raiseDepth        int
	coverEnabled      bool
	coverOpt          *CoverageOptions
	coverEntries      []CoverageEntry
	gcSafeLoops       bool
	gcFrameName       string
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
		case *ast.StructDecl:
			g.structDecls[n.Name] = n
		case *ast.InterfaceDecl:
			g.interfaceDecls[n.Name] = n
		case *ast.ModuleDecl:
			for _, iface := range n.Interfaces {
				g.interfaceDecls[n.Name+"."+iface.Name] = iface
			}
			for _, class := range n.Classes {
				g.classDecls[n.Name+"."+class.Name] = class
			}
		}
	}
}
