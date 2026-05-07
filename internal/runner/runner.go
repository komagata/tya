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
	src, modules, err := loadSource(path, map[string]bool{}, false)
	if err != nil {
		return "", nil, err
	}
	return src, modules, nil
}

type publicDef struct {
	name string
	kind string
}

func loadSource(path string, loading map[string]bool, module bool) (string, []string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", nil, err
	}
	if loading[abs] {
		return "", nil, fmt.Errorf("cyclic module import: %s", filepath.Base(path))
	}
	loading[abs] = true
	defer delete(loading, abs)

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
		modPath, err := resolveModulePath(path, imp)
		if err != nil {
			return "", nil, err
		}
		importDef, err := publicDefForFile(modPath)
		if err != nil {
			return "", nil, err
		}
		modSrc, importedModules, err := loadSource(modPath, loading, true)
		if err != nil {
			return "", nil, err
		}
		modules = append(modules, importedModules...)
		visible := importDef.name
		if visibleImports[visible] {
			return "", nil, fmt.Errorf("import name conflict: %s", visible)
		}
		visibleImports[visible] = true
		modules = append(modules, visible)
		out.WriteString(modSrc)
		if !strings.HasSuffix(modSrc, "\n") {
			out.WriteString("\n")
		}
	}
	if !module {
		if err := validateEntry(path, prog, visibleImports); err != nil {
			return "", nil, err
		}
	}
	_ = def
	out.WriteString(source)
	if !strings.HasSuffix(source, "\n") {
		out.WriteString("\n")
	}
	return out.String(), modules, nil
}

func resolveModulePath(importerPath string, name string) (string, error) {
	fileName := name + ".tya"
	candidates := []string{filepath.Join(filepath.Dir(importerPath), fileName)}
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
			return candidate, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("module not found: %s", name)
}

func stdlibDirs() []string {
	dirs := []string{filepath.Join("stdlib")}
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

func collectImports(prog *ast.Program) ([]string, error) {
	imports := []string{}
	for _, stmt := range prog.Stmts {
		imp, ok := stmt.(*ast.ImportStmt)
		if !ok {
			continue
		}
		if !moduleNameRE.MatchString(imp.Name) {
			return nil, fmt.Errorf("invalid module name: %s", imp.Name)
		}
		imports = append(imports, imp.Name)
	}
	return imports, nil
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
