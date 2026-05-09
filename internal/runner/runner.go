package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"tya/internal/ast"
	"tya/internal/checker"
	"tya/internal/eval"
	"tya/internal/lexer"
	"tya/internal/parser"
	"tya/internal/pkg"
)

var fileNameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*\.tya$`)
var moduleNameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func ValidateFileName(path string) error {
	if filepath.Ext(path) != ".tya" || !fileNameRE.MatchString(filepath.Base(path)) {
		return fmt.Errorf("invalid Tya file name: %s", filepath.Base(path))
	}
	return nil
}

func RunFile(path string, in io.Reader, out io.Writer, args []string) error {
	if err := ValidateFileName(path); err != nil {
		return err
	}
	source, err := LoadSource(path)
	if err != nil {
		return err
	}
	toks, errs := lexer.Lex(source)
	if len(errs) > 0 {
		return errs[0]
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		return err
	}
	_, modules, err := LoadUserSourceWithModules(path)
	if err != nil {
		return err
	}
	if err := checker.CheckWithModules(prog, modules); err != nil {
		return err
	}
	return eval.RunWithIO(prog, in, out, args)
}

func LoadSource(path string) (string, error) {
	src, _, err := LoadUserSourceWithModules(path)
	if err != nil {
		return "", err
	}
	return src, nil
}

func LoadSourceWithModules(path string) (string, []string, error) {
	src, modules, err := LoadUserSourceWithModules(path)
	if err != nil {
		return "", nil, err
	}
	return src, modules, nil
}

func LoadUserSource(path string) (string, error) {
	src, _, err := LoadUserSourceWithModules(path)
	return src, err
}

func LoadUserSourceWithModules(path string) (string, []string, error) {
	if err := ValidateFileName(path); err != nil {
		return "", nil, err
	}
	state := &loadState{loading: map[string]bool{}, loaded: map[string]bool{}}
	src, modules, err := loadSource(path, state, false)
	if err != nil {
		return "", nil, err
	}
	return src, modules, nil
}

type publicDef struct {
	name string
	kind string
}

type importSpec struct {
	stmt    *ast.ImportStmt
	path    string
	binding string
}

type loadState struct {
	loading map[string]bool
	loaded  map[string]bool
	stack   []loadFrame
}

type loadFrame struct {
	path string
	name string
}

func (s *loadState) cyclePath(abs string, name string) string {
	for i, frame := range s.stack {
		if frame.path == abs {
			parts := []string{}
			for _, f := range s.stack[i:] {
				parts = append(parts, f.name)
			}
			parts = append(parts, name)
			return strings.Join(parts, " -> ")
		}
	}
	parts := []string{}
	for _, f := range s.stack {
		parts = append(parts, f.name)
	}
	parts = append(parts, name)
	return strings.Join(parts, " -> ")
}

func displayModuleName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".tya")
}

func loadSource(path string, state *loadState, module bool) (string, []string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", nil, err
	}
	abs = filepath.Clean(abs)
	if state.loading[abs] {
		return "", nil, fmt.Errorf("import cycle: %s", state.cyclePath(abs, displayModuleName(path)))
	}
	if module && state.loaded[abs] {
		return "", nil, nil
	}
	state.loading[abs] = true
	state.stack = append(state.stack, loadFrame{path: abs, name: displayModuleName(path)})
	defer func() {
		delete(state.loading, abs)
		state.stack = state.stack[:len(state.stack)-1]
	}()

	src, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	source := string(src)
	prog, err := parseSource(source)
	if err != nil {
		return "", nil, err
	}
	imports, err := collectImports(prog)
	if err != nil {
		return "", nil, err
	}
	var def publicDef
	if module {
		def, err = validateModule(path, prog)
		if err != nil {
			return "", nil, err
		}
	}
	var out strings.Builder
	modules := []string{}
	visibleImports := map[string]bool{}
	for _, imp := range imports {
		modPath, err := resolveModulePath(path, imp.path)
		if err != nil {
			return "", nil, err
		}
		importDef, err := publicDefForFile(modPath)
		if err != nil {
			return "", nil, err
		}
		modSrc, importedModules, err := loadSource(modPath, state, true)
		if err != nil {
			return "", nil, err
		}
		modules = append(modules, importedModules...)
		if importDef.name != imp.stmt.ModuleName() {
			return "", nil, fmt.Errorf("%s must define module %s", filepath.Base(modPath), imp.stmt.ModuleName())
		}
		if visibleImports[imp.binding] {
			return "", nil, fmt.Errorf("import name conflict: %s", imp.binding)
		}
		visibleImports[imp.binding] = true
		modules = append(modules, imp.binding)
		if modSrc != "" {
			out.WriteString(modSrc)
			if !strings.HasSuffix(modSrc, "\n") {
				out.WriteString("\n")
			}
		}
	}
	if !module {
		if err := validateEntry(path, prog, visibleImports); err != nil {
			return "", nil, err
		}
		if err := validateAliasedImportOriginals(prog, imports); err != nil {
			return "", nil, err
		}
	}
	_ = def
	out.WriteString(source)
	if !strings.HasSuffix(source, "\n") {
		out.WriteString("\n")
	}
	if module {
		state.loaded[abs] = true
	}
	return out.String(), modules, nil
}

func resolveModulePath(importerPath string, name string) (string, error) {
	parts := strings.Split(name, "/")
	leading := parts[0]
	pathParts := append([]string{}, parts...)
	pathParts[len(pathParts)-1] = pathParts[len(pathParts)-1] + ".tya"
	fileName := filepath.Join(pathParts...)
	candidates := []string{filepath.Join(filepath.Dir(importerPath), fileName)}
	// v0.26: manifest-declared packages live under .tya/packages/<name>-<version>/src/
	for _, dir := range packageSrcDirs(importerPath, leading) {
		candidates = append(candidates, filepath.Join(dir, fileName))
	}
	for _, dir := range filepath.SplitList(os.Getenv("TYA_PATH")) {
		if dir == "" {
			continue
		}
		candidates = append(candidates, filepath.Join(dir, fileName))
	}
	for _, dir := range stdlibDirs() {
		candidates = append(candidates, filepath.Join(dir, fileName))
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			abs, err := filepath.Abs(candidate)
			if err != nil {
				return "", err
			}
			return filepath.Clean(abs), nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("module not found: %s", name)
}

// packageSrcDirs walks up from importerPath looking for a project root that
// contains a tya.toml, then returns candidate src/ directories. It consults
// tya.lock for package locations: git-sourced packages live under
// .tya/packages/<name>-<version>/, while path-sourced packages are read
// directly from the path recorded in the lockfile.
func packageSrcDirs(importerPath, leadingName string) []string {
	dir := filepath.Dir(importerPath)
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "tya.toml")); err == nil {
			out := []string{}
			lockPath := filepath.Join(dir, "tya.lock")
			if lf, err := pkg.ReadLockfile(lockPath); err == nil {
				for i := range lf.Packages {
					p := &lf.Packages[i]
					if p.Name != leadingName {
						continue
					}
					out = append(out, filepath.Join(pkg.PackageDir(dir, p), "src"))
				}
			}
			pkgs := filepath.Join(dir, ".tya", "packages")
			entries, err := os.ReadDir(pkgs)
			if err == nil {
				prefix := leadingName + "-"
				for _, e := range entries {
					if !e.IsDir() {
						continue
					}
					if e.Name() == leadingName || strings.HasPrefix(e.Name(), prefix) {
						out = append(out, filepath.Join(pkgs, e.Name(), "src"))
					}
				}
			}
			return out
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil
}

func stdlibDirs() []string {
	dirs := []string{}
	if dir := os.Getenv("TYA_STDLIB_DIR"); dir != "" {
		dirs = append(dirs, dir)
	}
	dirs = append(dirs, filepath.Join("stdlib"))
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		dirs = append(dirs,
			filepath.Join(exeDir, "stdlib"),
			filepath.Clean(filepath.Join(exeDir, "..", "share", "tya", "stdlib")),
		)
	}
	seen := map[string]bool{}
	out := []string{}
	for _, dir := range dirs {
		if seen[dir] {
			continue
		}
		seen[dir] = true
		out = append(out, dir)
	}
	return out
}

func publicDefForFile(path string) (publicDef, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return publicDef{}, err
	}
	prog, err := parseSource(string(src))
	if err != nil {
		return publicDef{}, err
	}
	return validateModule(path, prog)
}

func parseSource(src string) (*ast.Program, error) {
	toks, errs := lexer.Lex(src)
	if len(errs) > 0 {
		return nil, errs[0]
	}
	return parser.Parse(toks)
}

func collectImports(prog *ast.Program) ([]importSpec, error) {
	imports := []importSpec{}
	for _, stmt := range prog.Stmts {
		imp, ok := stmt.(*ast.ImportStmt)
		if !ok {
			continue
		}
		if err := validateImportPath(imp.Name); err != nil {
			return nil, err
		}
		binding := imp.BindingName()
		if !moduleNameRE.MatchString(binding) {
			return nil, fmt.Errorf("invalid import binding: %s", binding)
		}
		if strings.HasPrefix(binding, "_") {
			return nil, fmt.Errorf("invalid import binding: %s", binding)
		}
		imports = append(imports, importSpec{stmt: imp, path: imp.Name, binding: binding})
	}
	return imports, nil
}

func validateImportPath(name string) error {
	if name == "" || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "./") || strings.HasPrefix(name, "../") {
		return fmt.Errorf("invalid module name: %s", name)
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == "" || !moduleNameRE.MatchString(segment) {
			return fmt.Errorf("invalid module name: %s", name)
		}
	}
	return nil
}

func validateModule(path string, prog *ast.Program) (publicDef, error) {
	if err := checker.CheckModuleFile(prog, path); err != nil {
		return publicDef{}, err
	}
	var def publicDef
	for _, stmt := range prog.Stmts {
		if n, ok := stmt.(*ast.ModuleDecl); ok {
			def = publicDef{name: n.Name, kind: "module"}
		}
	}
	return def, nil
}

func validateEntry(path string, prog *ast.Program, imports map[string]bool) error {
	for _, stmt := range prog.Stmts {
		switch n := stmt.(type) {
		case *ast.ModuleDecl:
			return fmt.Errorf("%s entry file cannot define module %s directly", filepath.Base(path), n.Name)
		case *ast.ClassDecl:
			if imports[n.Name] {
				return fmt.Errorf("import name conflict: %s", n.Name)
			}
		case *ast.InterfaceDecl:
			if imports[n.Name] {
				return fmt.Errorf("import name conflict: %s", n.Name)
			}
		case *ast.AssignStmt:
			for _, target := range n.Targets {
				if id, ok := target.(*ast.Ident); ok && imports[id.Name] {
					return fmt.Errorf("import name conflict: %s", id.Name)
				}
			}
		}
	}
	return nil
}

func validateAliasedImportOriginals(prog *ast.Program, imports []importSpec) error {
	hidden := map[string]bool{}
	topLevel := map[string]bool{}
	for _, imp := range imports {
		topLevel[imp.binding] = true
		if imp.stmt.Alias != "" {
			hidden[imp.stmt.ModuleName()] = true
		}
	}
	if len(hidden) == 0 {
		return nil
	}
	for _, stmt := range prog.Stmts {
		switch n := stmt.(type) {
		case *ast.AssignStmt:
			for _, target := range n.Targets {
				if id, ok := target.(*ast.Ident); ok {
					topLevel[id.Name] = true
				}
			}
		case *ast.ClassDecl:
			topLevel[n.Name] = true
		case *ast.InterfaceDecl:
			topLevel[n.Name] = true
		}
	}
	for name := range topLevel {
		delete(hidden, name)
	}
	if len(hidden) == 0 {
		return nil
	}
	for _, stmt := range prog.Stmts {
		if err := rejectHiddenImportUseStmt(stmt, hidden, topLevel); err != nil {
			return err
		}
	}
	return nil
}

func rejectHiddenImportUseStmt(stmt ast.Stmt, hidden map[string]bool, bound map[string]bool) error {
	switch n := stmt.(type) {
	case *ast.ImportStmt:
		return nil
	case *ast.AssignStmt:
		for _, value := range n.Values {
			if err := rejectHiddenImportUseExpr(value, hidden, bound); err != nil {
				return err
			}
		}
		for _, target := range n.Targets {
			if _, ok := target.(*ast.Ident); ok {
				continue
			}
			if err := rejectHiddenImportUseExpr(target, hidden, bound); err != nil {
				return err
			}
		}
	case *ast.ExprStmt:
		return rejectHiddenImportUseExpr(n.Expr, hidden, bound)
	case *ast.IfStmt:
		if err := rejectHiddenImportUseExpr(n.Cond, hidden, bound); err != nil {
			return err
		}
		if err := rejectHiddenImportUseStmts(n.Then, hidden, bound); err != nil {
			return err
		}
		return rejectHiddenImportUseStmts(n.Else, hidden, bound)
	case *ast.WhileStmt:
		if err := rejectHiddenImportUseExpr(n.Cond, hidden, bound); err != nil {
			return err
		}
		return rejectHiddenImportUseStmts(n.Body, hidden, bound)
	case *ast.ForInStmt:
		if err := rejectHiddenImportUseExpr(n.Iterable, hidden, bound); err != nil {
			return err
		}
		child := cloneBoolMap(bound)
		child[n.ValueName] = true
		if n.IndexName != "" {
			child[n.IndexName] = true
		}
		return rejectHiddenImportUseStmts(n.Body, hidden, child)
	case *ast.ReturnStmt:
		for _, value := range n.Values {
			if err := rejectHiddenImportUseExpr(value, hidden, bound); err != nil {
				return err
			}
		}
	case *ast.RaiseStmt:
		return rejectHiddenImportUseExpr(n.Value, hidden, bound)
	case *ast.TryCatchStmt:
		if err := rejectHiddenImportUseStmts(n.Try, hidden, bound); err != nil {
			return err
		}
		child := cloneBoolMap(bound)
		child[n.CatchName] = true
		return rejectHiddenImportUseStmts(n.Catch, hidden, child)
	case *ast.MatchStmt:
		if err := rejectHiddenImportUseExpr(n.Value, hidden, bound); err != nil {
			return err
		}
		for _, c := range n.Cases {
			if err := rejectHiddenImportUseExpr(c.Pattern, hidden, bound); err != nil {
				return err
			}
			if err := rejectHiddenImportUseStmts(c.Body, hidden, bound); err != nil {
				return err
			}
		}
	case *ast.ModuleDecl:
		for _, member := range n.Members {
			if err := rejectHiddenImportUseExpr(member.Value, hidden, bound); err != nil {
				return err
			}
		}
	case *ast.ClassDecl:
		for _, field := range n.Fields {
			if err := rejectHiddenImportUseExpr(field.Value, hidden, bound); err != nil {
				return err
			}
		}
		for _, classVar := range n.Vars {
			if err := rejectHiddenImportUseExpr(classVar.Value, hidden, bound); err != nil {
				return err
			}
		}
		for _, method := range n.Methods {
			if err := rejectHiddenImportUseExpr(method.Func, hidden, bound); err != nil {
				return err
			}
		}
	}
	return nil
}

func rejectHiddenImportUseStmts(stmts []ast.Stmt, hidden map[string]bool, bound map[string]bool) error {
	for _, stmt := range stmts {
		if err := rejectHiddenImportUseStmt(stmt, hidden, bound); err != nil {
			return err
		}
	}
	return nil
}

func rejectHiddenImportUseExpr(expr ast.Expr, hidden map[string]bool, bound map[string]bool) error {
	switch n := expr.(type) {
	case *ast.Ident:
		if hidden[n.Name] && !bound[n.Name] {
			return fmt.Errorf("%d:%d: undefined variable %s", n.Tok.Line, n.Tok.Col, n.Name)
		}
	case *ast.DictLit:
		for _, prop := range n.Props {
			if err := rejectHiddenImportUseExpr(prop.Value, hidden, bound); err != nil {
				return err
			}
		}
	case *ast.ArrayLit:
		for _, elem := range n.Elems {
			if err := rejectHiddenImportUseExpr(elem, hidden, bound); err != nil {
				return err
			}
		}
	case *ast.FuncLit:
		child := cloneBoolMap(bound)
		for _, param := range n.Params {
			child[param] = true
		}
		if n.Expr != nil {
			if err := rejectHiddenImportUseExpr(n.Expr, hidden, child); err != nil {
				return err
			}
		}
		return rejectHiddenImportUseStmts(n.Body, hidden, child)
	case *ast.BinaryExpr:
		if err := rejectHiddenImportUseExpr(n.Left, hidden, bound); err != nil {
			return err
		}
		return rejectHiddenImportUseExpr(n.Right, hidden, bound)
	case *ast.UnaryExpr:
		return rejectHiddenImportUseExpr(n.Expr, hidden, bound)
	case *ast.TryExpr:
		return rejectHiddenImportUseExpr(n.Expr, hidden, bound)
	case *ast.MemberExpr:
		return rejectHiddenImportUseExpr(n.Target, hidden, bound)
	case *ast.IndexExpr:
		if err := rejectHiddenImportUseExpr(n.Target, hidden, bound); err != nil {
			return err
		}
		return rejectHiddenImportUseExpr(n.Index, hidden, bound)
	case *ast.CallExpr:
		if err := rejectHiddenImportUseExpr(n.Callee, hidden, bound); err != nil {
			return err
		}
		for _, arg := range n.Args {
			if err := rejectHiddenImportUseExpr(arg, hidden, bound); err != nil {
				return err
			}
		}
	}
	return nil
}

func cloneBoolMap(in map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func snakeCase(name string) string {
	var out strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out.WriteByte('_')
		}
		out.WriteRune(lowerASCII(r))
	}
	return out.String()
}

func pascalCase(name string) string {
	var out strings.Builder
	upper := true
	for _, r := range name {
		if r == '_' {
			upper = true
			continue
		}
		if upper {
			out.WriteRune(upperASCII(r))
			upper = false
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func lowerASCII(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}

func upperASCII(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - ('a' - 'A')
	}
	return r
}
