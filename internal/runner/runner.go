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
	return WithPrelude(path, src), nil
}

func LoadSourceWithModules(path string) (string, []string, error) {
	src, modules, err := LoadUserSourceWithModules(path)
	if err != nil {
		return "", nil, err
	}
	return WithPrelude(path, src), modules, nil
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
	body, imports, err := splitImports(string(src))
	if err != nil {
		return "", nil, err
	}
	if module {
		if err := validateModule(path, body); err != nil {
			return "", nil, err
		}
	}
	var out strings.Builder
	modules := []string{}
	for _, name := range imports {
		modPath := filepath.Join(filepath.Dir(path), name+".tya")
		modSrc, importedModules, err := loadSource(modPath, loading, true)
		if err != nil {
			return "", nil, err
		}
		modules = append(modules, importedModules...)
		modules = append(modules, name)
		out.WriteString(modSrc)
		if !strings.HasSuffix(modSrc, "\n") {
			out.WriteString("\n")
		}
	}
	out.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		out.WriteString("\n")
	}
	return out.String(), modules, nil
}

func splitImports(src string) (string, []string, error) {
	lines := strings.Split(src, "\n")
	imports := []string{}
	body := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			body = append(body, line)
			continue
		}
		if strings.HasPrefix(trimmed, "import ") {
			if strings.TrimLeft(line, " ") != line {
				return "", nil, fmt.Errorf("import must be top-level: %s", trimmed)
			}
			name := strings.TrimSpace(strings.TrimPrefix(trimmed, "import "))
			if !moduleNameRE.MatchString(name) {
				return "", nil, fmt.Errorf("invalid module name: %s", name)
			}
			imports = append(imports, name)
			continue
		}
		body = append(body, line)
	}
	return strings.Join(body, "\n"), imports, nil
}

func validateModule(path, src string) error {
	toks, errs := lexer.Lex(src)
	if len(errs) > 0 {
		return errs[0]
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		return err
	}
	want := strings.TrimSuffix(filepath.Base(path), ".tya")
	var public []string
	for _, stmt := range prog.Stmts {
		switch n := stmt.(type) {
		case *ast.ClassDecl:
			public = append(public, n.Name)
			if snakeCase(n.Name) != want {
				return fmt.Errorf("%s must define class %s", filepath.Base(path), pascalCase(want))
			}
		case *ast.ModuleDecl:
			public = append(public, n.Name)
			if n.Name != want {
				return fmt.Errorf("%s must define module %s", filepath.Base(path), want)
			}
		default:
			return fmt.Errorf("%s may only contain imports and one public class or module declaration", filepath.Base(path))
		}
	}
	if len(public) != 1 {
		return fmt.Errorf("%s must define exactly one public class or module", filepath.Base(path))
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

func WithPrelude(path, src string) string {
	candidates := []string{
		filepath.Join(filepath.Dir(path), "..", "stdlib", "prelude.tya"),
		filepath.Join("stdlib", "prelude.tya"),
	}
	for _, preludePath := range candidates {
		prelude, err := os.ReadFile(preludePath)
		if err != nil {
			continue
		}
		if strings.HasSuffix(string(prelude), "\n") {
			return string(prelude) + src
		}
		return string(prelude) + "\n" + src
	}
	return src
}
