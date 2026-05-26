package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"tya/internal/ast"
	"tya/internal/checker"
	"tya/internal/diag"
	"tya/internal/eval"
	"tya/internal/lexer"
	"tya/internal/parser"
	"tya/internal/pkg"
	"tya/internal/token"
)

var fileNameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*\.tya$`)
var moduleNameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// ValidateFileName enforces the strict file-name rules for entry
// points: the file must be a `.tya` lowercase script file. PascalCase
// class files are rejected with the v0.44 [TYA-E0850] diagnostic
// because class files are library-only.
//
// Use ValidateAnyTyaFileName for read-only commands (tya format,
// tya check, tya emit-c) that should accept both script and class
// files.
func ValidateFileName(path string) error {
	base := filepath.Base(path)
	if filepath.Ext(path) != ".tya" {
		return runnerError(codeInvalidFileName, fmt.Sprintf("invalid Tya file name: %s", base), 0, 0)
	}
	if fileNameRE.MatchString(base) {
		return nil
	}
	// v0.44: a PascalCase filename identifies a class file. Class files
	// are library-only; tya run accepts only script files (lowercase).
	if checker.IsClassFileName(base) {
		return fmt.Errorf("[TYA-E0850] %s is a class file; tya run accepts only script files (lowercase filename)", base)
	}
	return runnerError(codeInvalidFileName, fmt.Sprintf("invalid Tya file name: %s", base), 0, 0)
}

// ValidateAnyTyaFileName accepts any well-formed v0.44 .tya file:
// either a script file (lowercase) or a class file (PascalCase). It
// is intended for read-only developer commands like `tya format`,
// `tya check`, and `tya emit-c` that operate on individual files
// without running them as entry points.
func ValidateAnyTyaFileName(path string) error {
	base := filepath.Base(path)
	if filepath.Ext(path) != ".tya" {
		return runnerError(codeInvalidFileName, fmt.Sprintf("invalid Tya file name: %s", base), 0, 0)
	}
	if fileNameRE.MatchString(base) {
		return nil
	}
	if checker.IsClassFileName(base) {
		return nil
	}
	return runnerError(codeInvalidFileName, fmt.Sprintf("invalid Tya file name: %s", base), 0, 0)
}

// IsLegacyV01Path reports whether the given path identifies a
// source file under the selfhost/v01/ directory (which keeps the
// v0.43 class-member surface per the v0.46 / v0.47 SPECs' Self-Host
// Constraint). The runner enables permissive legacy mode on the
// checker before processing such inputs.
func IsLegacyV01Path(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	cleaned := filepath.ToSlash(filepath.Clean(abs))
	return strings.Contains(cleaned, "/selfhost/v01/")
}

// RunFile loads and executes the program at path. Returns
// (diags, err). v0.56: diags carries the recoverable parser
// diagnostics collected during the run (typically empty); err is
// the first fatal error encountered. Runner stays fail-fast for
// downstream stages — the diags slice has 0..N entries from the
// parser, but checker / eval stop on first error.
func RunFile(path string, in io.Reader, out io.Writer, args []string) ([]diag.Diagnostic, error) {
	if err := ValidateFileName(path); err != nil {
		return nil, err
	}
	defer checker.SetPermissiveLegacy(IsLegacyV01Path(path))()
	source, modules, origins, err := LoadUserSourceWithOrigins(path)
	if err != nil {
		return nil, err
	}
	toks, errs := lexer.Lex(source)
	if len(errs) > 0 {
		return nil, errs[0]
	}
	prog, pdiags, err := parser.Parse(toks)
	if err != nil {
		return pdiags, err
	}
	StampOriginFiles(prog, origins)
	if err := checker.CheckWithModules(prog, modules); err != nil {
		return pdiags, err
	}
	return pdiags, eval.RunWithIO(prog, in, out, args)
}

// StampOriginFiles walks package-expanded programs and stamps the
// per-class origin file path onto each top-level ClassDecl whose
// package origin was recorded during directory-package synthesis.
func StampOriginFiles(prog *ast.Program, origins map[string]map[string]string) {
	if prog == nil || len(origins) == 0 {
		return
	}
	classOrigins := map[string]string{}
	for _, pkgOrigins := range origins {
		for className, origin := range pkgOrigins {
			classOrigins[className] = origin
		}
	}
	for _, stmt := range prog.Stmts {
		class, ok := stmt.(*ast.ClassDecl)
		if !ok {
			continue
		}
		if origin, ok := classOrigins[class.Name]; ok {
			class.OriginFile = origin
		}
	}
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

// LoadSourceWithOrigins is the v0.45 variant of LoadSourceWithModules
// that also returns the package class/interface origin map. Callers
// that drive the checker should prefer this entry point and pass
// origins through StampOriginFiles so the [TYA-E0406] cross-file
// private check sees correct per-class origin metadata.
func LoadSourceWithOrigins(path string) (string, []string, map[string]map[string]string, error) {
	return LoadUserSourceWithOrigins(path)
}

// LoadClassFileWithSiblingOrigins loads a PascalCase class/interface file
// together with the other PascalCase files in the same directory. It is used
// by read-only commands such as `tya check Foo.tya`, where the checked file is
// a package member rather than an entry script.
func LoadClassFileWithSiblingOrigins(path string) (string, []string, map[string]map[string]string, error) {
	if err := ValidateAnyTyaFileName(path); err != nil {
		return "", nil, nil, err
	}
	if !checker.IsClassFileName(path) {
		return "", nil, nil, fmt.Errorf("%s is not a class file", filepath.Base(path))
	}
	classFiles, err := findEntrySiblings(path)
	if err != nil {
		return "", nil, nil, err
	}
	pkgName := "classfile"
	source, origins, err := synthesizePackageSource(classFiles, pkgName, false)
	if err != nil {
		return "", nil, nil, err
	}
	return source, nil, map[string]map[string]string{pkgName: origins}, nil
}

func LoadUserSource(path string) (string, error) {
	src, _, err := LoadUserSourceWithModules(path)
	return src, err
}

// LoadUserSourceWithOrigins is the v0.45 entry point that also returns
// the per-package class/interface origin map collected during package
// synthesis. The outer key is the synthesized package module name; the
// inner key is the class or interface name; the value is the source
// file base name (e.g. "Util.tya") that declared it. The checker uses
// this map to enforce cross-file private class visibility
// ([TYA-E0406]).
func LoadUserSourceWithOrigins(path string) (string, []string, map[string]map[string]string, error) {
	if err := ValidateFileName(path); err != nil {
		return "", nil, nil, err
	}
	state := &loadState{loading: map[string]bool{}, loaded: map[string]bool{}, synthModules: map[string]string{}, classOrigins: map[string]map[string]string{}}
	src, modules, err := loadSource(path, state, false, false)
	if err != nil {
		return "", nil, nil, err
	}
	return src, modules, state.classOrigins, nil
}

func LoadUserSourceWithModules(path string) (string, []string, error) {
	if err := ValidateFileName(path); err != nil {
		return "", nil, err
	}
	state := &loadState{loading: map[string]bool{}, loaded: map[string]bool{}, synthModules: map[string]string{}, classOrigins: map[string]map[string]string{}}
	src, modules, err := loadSource(path, state, false, false)
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
	stmt        *ast.ImportStmt
	path        string
	binding     string
	packageDir  bool
	publicNames []string
}

type loadState struct {
	loading map[string]bool
	loaded  map[string]bool
	stack   []loadFrame
	// synthModules tracks synthesized v0.44 package module names
	// → the absolute path of the directory that produced them. If
	// two different package directories synthesize the same module
	// name (terminal segment of their paths matches), the second
	// load fails with a clear collision error rather than silently
	// overwriting the first.
	synthModules map[string]string
	// classOrigins tracks per-package class/interface origin files.
	// Outer key is the synthesized package module name; inner key is
	// the class or interface name; value is the source file path
	// (relative to the package root, e.g. "Util.tya") that declared
	// the class. v0.45 [TYA-E0406] cross-file private enforcement
	// reads this map after the merged AST is parsed.
	classOrigins map[string]map[string]string
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

func samePath(left string, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr != nil || rightErr != nil {
		return filepath.Clean(left) == filepath.Clean(right)
	}
	return filepath.Clean(leftAbs) == filepath.Clean(rightAbs)
}

func loadSource(path string, state *loadState, module bool, packageNamespace bool) (string, []string, error) {
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

	info, statErr := os.Stat(path)
	if statErr != nil {
		return "", nil, statErr
	}
	var source string
	if info.IsDir() {
		// v0.44 directory-as-package: synthesize a virtual module
		// source that wraps every class file in the directory under
		// `module <last-segment>`.
		entries, err := os.ReadDir(path)
		if err != nil {
			return "", nil, err
		}
		classFiles := []string{}
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".tya" {
				continue
			}
			if checker.IsScriptFileName(e.Name()) {
				return "", nil, fmt.Errorf("[TYA-E0852] package %s contains script file %s; packages may not include lowercase .tya files", filepath.Base(path), e.Name())
			}
			if !checker.IsClassFileName(e.Name()) {
				continue
			}
			classFiles = append(classFiles, filepath.Join(path, e.Name()))
		}
		if len(classFiles) == 0 {
			return "", nil, fmt.Errorf("[TYA-E0853] package %s contains no class files", filepath.Base(path))
		}
		pkgName := filepath.Base(path)
		// v0.44: detect aliased same-segment package collision. If
		// another path already synthesized a module with this name,
		// fail clearly rather than silently overwriting the first
		// in the merged source.
		if prev, taken := state.synthModules[pkgName]; taken && prev != abs {
			return "", nil, fmt.Errorf("[TYA-E0855] package name conflict: both %s and %s would synthesize module %s; use distinct directory names or rename one", prev, abs, pkgName)
		}
		state.synthModules[pkgName] = abs
		synth, origins, err := synthesizePackageSource(classFiles, pkgName, packageNamespace)
		if err != nil {
			return "", nil, err
		}
		source = synth
		state.classOrigins[pkgName] = origins
	} else {
		src, err := os.ReadFile(path)
		if err != nil {
			return "", nil, err
		}
		source = string(src)
	}
	prog, err := parseSource(source)
	if err != nil {
		return "", nil, err
	}
	imports, err := collectImports(prog)
	if err != nil {
		return "", nil, err
	}
	var out strings.Builder
	modules := []string{}
	visibleImports := map[string]bool{}
	for i := range imports {
		imp := &imports[i]
		modPath, err := resolveModulePath(path, imp.path)
		isPackage := false
		if err != nil {
			// v0.44 fallback: try directory-as-package resolution.
			pkgDir, classFiles, perr := resolvePackageDir(path, imp.path)
			if perr != nil {
				return "", nil, perr
			}
			if pkgDir == "" {
				return "", nil, err
			}
			modPath = pkgDir
			isPackage = true
			imp.packageDir = true
			names, err := publicPackageNames(classFiles)
			if err != nil {
				return "", nil, err
			}
			imp.publicNames = names
		}
		modSrc, importedModules, err := loadSource(modPath, state, true, isPackage && imp.stmt.Alias != "")
		if err != nil {
			return "", nil, err
		}
		modules = append(modules, importedModules...)
		visibleNames := []string{imp.binding}
		if imp.packageDir && imp.stmt.Alias == "" {
			visibleNames = imp.publicNames
		}
		for _, name := range visibleNames {
			if visibleImports[name] {
				return "", nil, fmt.Errorf("import name conflict: %s", name)
			}
			visibleImports[name] = true
			if !imp.packageDir || imp.stmt.Alias != "" {
				modules = append(modules, name)
			}
		}
		if modSrc != "" {
			out.WriteString(modSrc)
			if !strings.HasSuffix(modSrc, "\n") {
				out.WriteString("\n")
			}
		}
	}
	if err := validateBarePackageImportNamespaces(prog, imports); err != nil {
		return "", nil, err
	}
	if err := validateAliasedPackageBareNames(prog, imports); err != nil {
		return "", nil, err
	}
	if !module {
		if err := validateEntry(path, prog, visibleImports); err != nil {
			return "", nil, err
		}
		if err := validateAliasedImportOriginals(prog, imports); err != nil {
			return "", nil, err
		}
	}

	// v0.44 same-directory sibling auto-visibility: when loading an
	// entry script, every PascalCase class file in the entry's
	// directory is auto-loaded so its public class is in scope without
	// an explicit import. Sibling class files' imports are resolved
	// alongside the entry's own imports and deduplicated by binding.
	if !module && info.Mode().IsRegular() {
		siblings, err := findEntrySiblings(path)
		if err != nil {
			return "", nil, err
		}
		for _, sib := range siblings {
			sibBytes, err := os.ReadFile(sib)
			if err != nil {
				return "", nil, err
			}
			sibSrc := string(sibBytes)
			sibProg, err := parseSource(sibSrc)
			if err != nil {
				return "", nil, fmt.Errorf("%s: %w", sib, err)
			}
			if err := checker.CheckClassFileStructure(sibProg, sib); err != nil {
				return "", nil, err
			}
			sibImports, err := collectImports(sibProg)
			if err != nil {
				return "", nil, err
			}
			for _, imp := range sibImports {
				if visibleImports[imp.binding] {
					continue
				}
				modPath, err := resolveModulePath(sib, imp.path)
				isPackage := false
				if err != nil {
					pkgDir, classFiles, perr := resolvePackageDir(sib, imp.path)
					if perr != nil {
						return "", nil, perr
					}
					if pkgDir == "" {
						return "", nil, err
					}
					modPath = pkgDir
					isPackage = true
					imp.packageDir = true
					names, err := publicPackageNames(classFiles)
					if err != nil {
						return "", nil, err
					}
					imp.publicNames = names
				}
				modSrc, importedModules, err := loadSource(modPath, state, true, isPackage && imp.stmt.Alias != "")
				if err != nil {
					return "", nil, err
				}
				modules = append(modules, importedModules...)
				visibleNames := []string{imp.binding}
				if imp.packageDir && imp.stmt.Alias == "" {
					visibleNames = imp.publicNames
				}
				for _, name := range visibleNames {
					visibleImports[name] = true
					if !imp.packageDir || imp.stmt.Alias != "" {
						modules = append(modules, name)
					}
				}
				if modSrc != "" {
					out.WriteString(modSrc)
					if !strings.HasSuffix(modSrc, "\n") {
						out.WriteString("\n")
					}
				}
			}
			body := stripTopLevelImports(sibSrc)
			out.WriteString(body)
			if !strings.HasSuffix(body, "\n") {
				out.WriteString("\n")
			}
		}
	}
	finalSource := stripBarePackageImports(source, imports)
	out.WriteString(finalSource)
	if !strings.HasSuffix(finalSource, "\n") {
		out.WriteString("\n")
	}
	if module && !info.IsDir() && !legacyModuleSource(prog) {
		binding := strings.TrimSuffix(filepath.Base(path), ".tya")
		namespace, err := synthesizeScriptNamespace(source, binding)
		if err != nil {
			return "", nil, err
		}
		out.WriteString(namespace)
	}
	if module {
		state.loaded[abs] = true
	}
	return dedupeTopLevelImports(out.String()), modules, nil
}

func legacyModuleSource(prog *ast.Program) bool {
	if os.Getenv("TYA_LEGACY_MODULES") != "1" || prog == nil {
		return false
	}
	for _, stmt := range prog.Stmts {
		if _, ok := stmt.(*ast.ModuleDecl); ok {
			return true
		}
	}
	return false
}

func resolveModulePath(importerPath string, name string) (string, error) {
	parts := strings.Split(name, "/")
	leading := parts[0]
	pathParts := append([]string{}, parts...)
	pathParts[len(pathParts)-1] = pathParts[len(pathParts)-1] + ".tya"
	fileName := filepath.Join(pathParts...)
	candidates := []string{filepath.Join(filepath.Dir(importerPath), fileName)}
	// v0.26: manifest-declared packages live under .tya/packages/<name>-<version>/src/
	pkgDirs, err := packageSrcDirs(importerPath, leading)
	if err != nil {
		return "", err
	}
	for _, dir := range pkgDirs {
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
		if samePath(candidate, importerPath) {
			continue
		}
		if ok, err := exactFileExists(candidate); err != nil {
			return "", err
		} else if ok {
			abs, err := filepath.Abs(candidate)
			if err != nil {
				return "", err
			}
			return filepath.Clean(abs), nil
		}
	}
	return "", fmt.Errorf("module not found: %s", name)
}

func exactFileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.IsDir() {
		return false, nil
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		return false, err
	}
	base := filepath.Base(path)
	for _, entry := range entries {
		if entry.Name() == base {
			return true, nil
		}
	}
	return false, nil
}

// packageSrcDirs walks up from importerPath looking for a project root that
// contains a tya.toml, then returns candidate src/ directories. It consults
// tya.lock for package locations: git-sourced packages live under
// .tya/packages/<name>-<version>/, while path-sourced packages are read
// directly from the path recorded in the lockfile.
func packageSrcDirs(importerPath, leadingName string) ([]string, error) {
	dir := filepath.Dir(importerPath)
	if root := os.Getenv("TYA_PROJECT_ROOT"); root != "" {
		out, err := packageSrcDirsFromRoot(root, leadingName)
		if err != nil {
			return nil, err
		}
		if len(out) > 0 {
			return out, nil
		}
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "tya.toml")); err == nil {
			return packageSrcDirsFromRoot(dir, leadingName)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil, nil
}

func packageSrcDirsFromRoot(root, leadingName string) ([]string, error) {
	out := []string{filepath.Join(root, "src")}
	lockPath := filepath.Join(root, "tya.lock")
	if lf, err := pkg.ReadLockfile(lockPath); err == nil {
		if m, manifestErr := pkg.ReadManifest(filepath.Join(root, "tya.toml")); manifestErr == nil && !lf.SatisfiesManifest(m) {
			return nil, fmt.Errorf("[TYA-E0941] tya.lock is stale; run `tya install`")
		}
		for i := range lf.Packages {
			p := &lf.Packages[i]
			if p.Name != leadingName {
				continue
			}
			pkgRoot := pkg.PackageDir(root, p)
			if p.Source != "path" && p.Checksum != "" {
				got, err := pkg.TreeChecksum(pkgRoot)
				if err != nil {
					return nil, fmt.Errorf("[TYA-E0942] locked dependency %s is unavailable locally; run `tya install`", p.Name)
				}
				if got != p.Checksum {
					return nil, fmt.Errorf("[TYA-E0943] dependency %s content hash mismatch; run `tya install`", p.Name)
				}
			}
			out = append(out, filepath.Join(pkgRoot, "src"))
		}
	}
	pkgs := filepath.Join(root, ".tya", "packages")
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
	return out, nil
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
	info, err := os.Stat(path)
	if err != nil {
		return publicDef{}, err
	}
	if info.IsDir() {
		// v0.44 package directory: the public binding is the
		// directory's leaf name, treated as a synthesized module.
		name := filepath.Base(path)
		if !moduleNameRE.MatchString(name) {
			return publicDef{}, fmt.Errorf("[TYA-E0854] invalid package directory name: %s", name)
		}
		return publicDef{name: name, kind: "module"}, nil
	}
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

func synthesizeScriptNamespace(source string, binding string) (string, error) {
	if !moduleNameRE.MatchString(binding) {
		return "", fmt.Errorf("invalid import binding: %s", binding)
	}
	prog, err := parseSource(source)
	if err != nil {
		return "", err
	}
	names := publicTopLevelNames(prog)
	var out strings.Builder
	out.WriteString(binding)
	out.WriteString(" = {}\n")
	out.WriteString(binding)
	out.WriteString("[\"__module_namespace\"] = true\n")
	for _, name := range names {
		out.WriteString(binding)
		out.WriteString("[")
		out.WriteString(strconv.Quote(name))
		out.WriteString("] = ")
		out.WriteString(name)
		out.WriteString("\n")
	}
	return out.String(), nil
}

func publicTopLevelNames(prog *ast.Program) []string {
	seen := map[string]bool{}
	names := []string{}
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}
	for _, stmt := range prog.Stmts {
		switch n := stmt.(type) {
		case *ast.AssignStmt:
			for _, target := range n.Targets {
				if id, ok := target.(*ast.Ident); ok {
					add(id.Name)
				}
			}
		case *ast.ClassDecl:
			add(n.Name)
		case *ast.EmbedStmt:
			add(n.Name)
		}
	}
	return names
}

func parseSource(src string) (*ast.Program, error) {
	toks, errs := lexer.Lex(src)
	if len(errs) > 0 {
		return nil, errs[0]
	}
	prog, _, err := parser.Parse(toks)
	return prog, err
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
		if isRemovedPrimitiveModule(imp.Name) {
			return nil, runnerError(codeRemovedPrimitiveModule, fmt.Sprintf("module %s was removed in v0.59; methods now live on the wrapper class", imp.Name), imp.NameTok.Line, imp.NameTok.Col)
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

func isRemovedPrimitiveModule(name string) bool {
	return name == "string" || name == "array" || name == "dict"
}

func validateImportPath(name string) error {
	if name == "" || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "./") || strings.HasPrefix(name, "../") {
		return fmt.Errorf("[TYA-E0851] invalid module name: %s", name)
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == "" || !moduleNameRE.MatchString(segment) {
			return fmt.Errorf("[TYA-E0851] invalid module name: %s", name)
		}
	}
	return nil
}

// findEntrySiblings returns the absolute paths of every PascalCase
// class file that shares the entry's directory. It excludes the
// entry itself, hidden files, subdirectories, and non-class .tya
// files. Used by entry script loading to implement v0.44
// same-directory sibling auto-visibility.
func findEntrySiblings(entryPath string) ([]string, error) {
	dir := filepath.Dir(entryPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".tya" {
			continue
		}
		if !checker.IsClassFileName(e.Name()) {
			continue
		}
		abs, err := filepath.Abs(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, filepath.Clean(abs))
	}
	return out, nil
}

// stripTopLevelImports removes top-level (column-zero) `import`
// lines from a class file source. Used when injecting sibling class
// file content into an entry script's merged source so duplicate
// imports do not appear in the final program.
func stripTopLevelImports(src string) string {
	var out strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if strings.HasPrefix(line, "import ") || line == "import" {
			continue
		}
		out.WriteString(line)
		out.WriteString("\n")
	}
	return strings.TrimRight(out.String(), "\n") + "\n"
}

func stripBarePackageImports(src string, imports []importSpec) string {
	strip := map[string]bool{}
	for _, imp := range imports {
		if imp.packageDir && imp.stmt.Alias == "" {
			strip[imp.path] = true
		}
	}
	if len(strip) == 0 {
		return src
	}
	var out strings.Builder
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, "import ") && strip[strings.TrimPrefix(trimmed, "import ")] {
			continue
		}
		out.WriteString(line)
		out.WriteString("\n")
	}
	return strings.TrimRight(out.String(), "\n") + "\n"
}

func dedupeTopLevelImports(src string) string {
	seenNoAlias := map[string]bool{}
	seenAlias := map[string]map[string]bool{}
	var out strings.Builder
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if (strings.HasPrefix(trimmed, "import ") || trimmed == "import") && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			body := strings.TrimSpace(strings.TrimPrefix(trimmed, "import "))
			path, alias, hasAlias := strings.Cut(body, " as ")
			path = strings.TrimSpace(path)
			alias = strings.TrimSpace(alias)
			effectiveNoAlias := !hasAlias || alias == filepath.Base(path)
			if hasAlias && alias != "" {
				if aliases := seenAlias[path]; aliases != nil && aliases[alias] {
					continue
				}
				if seenAlias[path] == nil {
					seenAlias[path] = map[string]bool{}
				}
				seenAlias[path][alias] = true
			}
			if seenNoAlias[path] {
				if !effectiveNoAlias && alias != "" {
					out.WriteString(alias)
					out.WriteString(" = ")
					out.WriteString(filepath.Base(path))
					out.WriteString("\n")
				}
				continue
			}
			if effectiveNoAlias {
				seenNoAlias[path] = true
			}
		}
		out.WriteString(line)
		out.WriteString("\n")
	}
	return strings.TrimRight(out.String(), "\n") + "\n"
}

// synthesizePackageSource takes a list of class file paths and a
// package name and produces a single Tya source string. Bare package
// imports expose every class as a top-level binding. Aliased package
// imports rewrite the package-internal class names and expose a
// namespace dict with the original public names so generated symbols
// stay distinct from same-named local classes.
//
// Each class file is validated via checker.CheckClassFile before
// inclusion; its imports are extracted from the AST, and the file's
// source text is included with top-level import lines removed.
func synthesizePackageSource(classFiles []string, pkgName string, includeNamespace bool) (string, map[string]string, error) {
	if !moduleNameRE.MatchString(pkgName) {
		return "", nil, fmt.Errorf("invalid package name: %s", pkgName)
	}
	type importKey struct {
		path  string
		alias string
	}
	var orderedImports []*ast.ImportStmt
	seenImports := map[importKey]bool{}
	var bodies []string
	origins := map[string]string{}
	publicSymbols := map[string]bool{}
	packageSymbols := map[string]bool{}
	publicInterfaces := map[string]bool{}

	for _, file := range classFiles {
		raw, err := os.ReadFile(file)
		if err != nil {
			return "", nil, err
		}
		text := string(raw)
		prog, err := parseSource(text)
		if err != nil {
			return "", nil, fmt.Errorf("%s: %w", file, err)
		}
		if err := checker.CheckClassFileStructure(prog, file); err != nil {
			return "", nil, err
		}
		relName := filepath.Base(file)
		for _, stmt := range prog.Stmts {
			switch n := stmt.(type) {
			case *ast.ImportStmt:
				key := importKey{path: n.Name, alias: n.Alias}
				if seenImports[key] {
					continue
				}
				seenImports[key] = true
				orderedImports = append(orderedImports, n)
			case *ast.ClassDecl:
				packageSymbols[n.Name] = true
				if _, dup := origins[n.Name]; !dup {
					origins[n.Name] = relName
				}
				if relName == n.Name+".tya" {
					publicSymbols[n.Name] = true
				}
			case *ast.InterfaceDecl:
				packageSymbols[n.Name] = true
				publicSymbols[n.Name] = true
				publicInterfaces[n.Name] = true
			}
		}
		// Strip top-level import lines from the source text.
		var body strings.Builder
		for _, line := range strings.Split(text, "\n") {
			trimmed := strings.TrimLeft(line, " \t")
			if strings.HasPrefix(trimmed, "import ") || trimmed == "import" {
				// Only strip when at column zero (top-level import).
				if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
					continue
				}
			}
			if strings.TrimSpace(line) == "" {
				body.WriteString("\n")
				continue
			}
			body.WriteString(line)
			body.WriteString("\n")
		}
		bodies = append(bodies, body.String())
	}

	var out strings.Builder
	for _, imp := range orderedImports {
		out.WriteString("import ")
		out.WriteString(imp.Name)
		if imp.Alias != "" {
			out.WriteString(" as ")
			out.WriteString(imp.Alias)
		}
		out.WriteString("\n")
	}
	names := make([]string, 0, len(publicSymbols))
	for name := range publicSymbols {
		names = append(names, name)
	}
	sort.Strings(names)
	if includeNamespace {
		renames := packageSymbolRenames(packageSymbols, pkgName)
		for _, b := range bodies {
			out.WriteString(rewritePackageClassNames(b, renames))
		}
		out.WriteString(pkgName)
		out.WriteString(" = {}\n")
		out.WriteString(pkgName)
		out.WriteString("[\"__module_namespace\"] = true\n")
		for _, name := range names {
			out.WriteString(pkgName)
			out.WriteString("[")
			out.WriteString(strconv.Quote(name))
			out.WriteString("] = ")
			if publicInterfaces[name] {
				out.WriteString(strconv.Quote("interface " + pkgName + "." + name))
			} else {
				out.WriteString(renames[name])
			}
			out.WriteString("\n")
		}
		return out.String(), origins, nil
	}
	for _, b := range bodies {
		out.WriteString(b)
	}
	return out.String(), origins, nil
}

func packageSymbolRenames(symbols map[string]bool, pkgName string) map[string]string {
	names := make([]string, 0, len(symbols))
	for name := range symbols {
		names = append(names, name)
	}
	sort.Strings(names)
	suffix := "TyaPkg" + pascalIdentifier(pkgName)
	renames := map[string]string{}
	for _, name := range names {
		renames[name] = name + suffix
	}
	return renames
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

func rewritePackageClassNames(src string, renames map[string]string) string {
	if len(renames) == 0 {
		return src
	}
	toks, errs := lexer.Lex(src)
	if len(errs) > 0 {
		return src
	}
	lineStarts := []int{0}
	for i, r := range src {
		if r == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}
	type replacement struct {
		start int
		end   int
		value string
	}
	replacements := []replacement{}
	var prev token.Token
	for _, tok := range toks {
		name, ok := renames[tok.Lexeme]
		if !ok || tok.Type != token.IDENT {
			if tok.Type != token.NEWLINE && tok.Type != token.INDENT && tok.Type != token.DEDENT {
				prev = tok
			}
			continue
		}
		if prev.Type == token.DOT {
			prev = tok
			continue
		}
		if tok.Line < 1 || tok.Line > len(lineStarts) {
			prev = tok
			continue
		}
		start := lineStarts[tok.Line-1] + tok.Col - 1
		end := start + len(tok.Lexeme)
		if start >= 0 && end <= len(src) {
			replacements = append(replacements, replacement{start: start, end: end, value: name})
		}
		prev = tok
	}
	for i := len(replacements) - 1; i >= 0; i-- {
		r := replacements[i]
		src = src[:r.start] + r.value + src[r.end:]
	}
	return src
}

func publicPackageNames(classFiles []string) ([]string, error) {
	seen := map[string]bool{}
	names := []string{}
	for _, file := range classFiles {
		raw, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		prog, err := parseSource(string(raw))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		if err := checker.CheckClassFileStructure(prog, file); err != nil {
			return nil, err
		}
		for _, stmt := range prog.Stmts {
			switch n := stmt.(type) {
			case *ast.ClassDecl:
				if seen[n.Name] {
					continue
				}
				seen[n.Name] = true
				names = append(names, n.Name)
			case *ast.InterfaceDecl:
				if seen[n.Name] {
					continue
				}
				seen[n.Name] = true
				names = append(names, n.Name)
			}
		}
	}
	sort.Strings(names)
	return names, nil
}

// resolvePackageDir attempts to resolve an import path to a v0.44
// package directory (a directory containing one or more PascalCase
// class files). It searches the same roots as resolveModulePath:
// importer's directory, manifest packages, TYA_PATH, and stdlib.
//
// On success it returns the absolute directory path and a sorted list
// of the absolute paths of class files (PascalCase .tya files) inside
// it. On failure it returns ("", nil, nil) without an error so callers
// can fall back to file-based module resolution.
//
// Directories are rejected when they contain a script file at the
// leaf (lowercase .tya), to forbid script-file imports.
//
// This helper is part of v0.44 STEP M3 foundation; the runner does
// not consume it yet. It exists so the import flow can be wired in a
// follow-up STEP without further resolver changes.
func resolvePackageDir(importerPath string, name string) (string, []string, error) {
	parts := strings.Split(name, "/")
	leading := parts[0]
	relDir := filepath.Join(parts...)
	candidates := []string{filepath.Join(filepath.Dir(importerPath), relDir)}
	pkgDirs, err := packageSrcDirs(importerPath, leading)
	if err != nil {
		return "", nil, err
	}
	for _, dir := range pkgDirs {
		candidates = append(candidates, filepath.Join(dir, relDir))
	}
	for _, dir := range filepath.SplitList(os.Getenv("TYA_PATH")) {
		if dir == "" {
			continue
		}
		candidates = append(candidates, filepath.Join(dir, relDir))
	}
	for _, dir := range stdlibDirs() {
		candidates = append(candidates, filepath.Join(dir, relDir))
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", nil, err
		}
		if !info.IsDir() {
			continue
		}
		entries, err := os.ReadDir(candidate)
		if err != nil {
			return "", nil, err
		}
		classFiles := []string{}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".tya" {
				continue
			}
			if checker.IsScriptFileName(entry.Name()) {
				return "", nil, fmt.Errorf("package %s contains script file %s; packages may not include lowercase .tya files", name, entry.Name())
			}
			if !checker.IsClassFileName(entry.Name()) {
				continue
			}
			abs, err := filepath.Abs(filepath.Join(candidate, entry.Name()))
			if err != nil {
				return "", nil, err
			}
			classFiles = append(classFiles, filepath.Clean(abs))
		}
		if len(classFiles) == 0 {
			// Directory exists but contains no class files. Surface
			// a clear error rather than letting the resolver fall
			// back to "module not found", which would suggest the
			// package itself is missing.
			return "", nil, fmt.Errorf("package %s contains no class files", name)
		}
		absDir, err := filepath.Abs(candidate)
		if err != nil {
			return "", nil, err
		}
		return filepath.Clean(absDir), classFiles, nil
	}
	return "", nil, nil
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
			return runnerError(codeEntryRedefinesModule, fmt.Sprintf("%s entry file cannot define module %s directly", filepath.Base(path), n.Name), 0, 0)
		case *ast.ClassDecl:
			if imports[n.Name] {
				return runnerError(codeImportNameConflict, fmt.Sprintf("import name conflict: %s", n.Name), 0, 0)
			}
		case *ast.InterfaceDecl:
			if imports[n.Name] {
				return runnerError(codeImportNameConflict, fmt.Sprintf("import name conflict: %s", n.Name), 0, 0)
			}
		case *ast.AssignStmt:
			for _, target := range n.Targets {
				if id, ok := target.(*ast.Ident); ok && imports[id.Name] {
					return runnerError(codeImportNameConflict, fmt.Sprintf("import name conflict: %s", id.Name), 0, 0)
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

func validateBarePackageImportNamespaces(prog *ast.Program, imports []importSpec) error {
	hidden := map[string]bool{}
	topLevel := map[string]bool{}
	for _, imp := range imports {
		if !imp.packageDir || imp.stmt.Alias != "" {
			continue
		}
		hidden[imp.stmt.ModuleName()] = true
		for _, name := range imp.publicNames {
			topLevel[name] = true
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
		if err := rejectHiddenImportUseClassRefs(stmt, hidden); err != nil {
			return err
		}
	}
	return nil
}

func validateAliasedPackageBareNames(prog *ast.Program, imports []importSpec) error {
	hidden := map[string]bool{}
	topLevel := map[string]bool{}
	for _, imp := range imports {
		if !imp.packageDir || imp.stmt.Alias == "" {
			continue
		}
		topLevel[imp.binding] = true
		for _, name := range imp.publicNames {
			hidden[name] = true
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
		if err := rejectHiddenImportUseClassRefs(stmt, hidden); err != nil {
			return err
		}
	}
	return nil
}

func rejectHiddenImportUseClassRefs(stmt ast.Stmt, hidden map[string]bool) error {
	switch n := stmt.(type) {
	case *ast.ClassDecl:
		if n.Parent != nil && hidden[n.Parent.Module] {
			return runnerError(codeUndefinedVariable, fmt.Sprintf("undefined variable %s", n.Parent.Module), n.Parent.Tok.Line, n.Parent.Tok.Col)
		}
		for _, ref := range n.Implements {
			if hidden[ref.Module] {
				return runnerError(codeUndefinedVariable, fmt.Sprintf("undefined variable %s", ref.Module), ref.Tok.Line, ref.Tok.Col)
			}
		}
	case *ast.InterfaceDecl:
		for _, ref := range n.Parents {
			if hidden[ref.Module] {
				return runnerError(codeUndefinedVariable, fmt.Sprintf("undefined variable %s", ref.Module), ref.Tok.Line, ref.Tok.Col)
			}
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
		if n.Catch != nil {
			child := cloneBoolMap(bound)
			if n.CatchName != "" {
				child[n.CatchName] = true
			}
			if err := rejectHiddenImportUseStmts(n.Catch, hidden, child); err != nil {
				return err
			}
		}
		return rejectHiddenImportUseStmts(n.Finally, hidden, bound)
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
			locals := make([]string, 0, len(bound))
			for k := range bound {
				locals = append(locals, k)
			}
			hint := undefinedNameHint(n.Name, locals)
			msg := fmt.Sprintf("undefined variable %s", n.Name)
			if hint != "" {
				return runnerError(codeUndefinedVariable, msg, n.Tok.Line, n.Tok.Col, hint)
			}
			return runnerError(codeUndefinedVariable, msg, n.Tok.Line, n.Tok.Col)
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
