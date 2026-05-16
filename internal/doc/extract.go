// Package doc implements the v0.51 `tya doc` source documentation
// generator. It walks top-level declarations in tya source files,
// pulls their leading `#` comment block as Markdown body, and
// renders either plain text or a multi-page HTML static site.
package doc

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tya/internal/ast"
	"tya/internal/lexer"
	"tya/internal/parser"
)

// DocItem is one documented top-level binding extracted from a
// source file.
type DocItem struct {
	Name           string
	Kind           string // "function" | "class" | "module" | "interface"
	Signature      string
	RawDoc         string // Leading comments joined with "\n" (raw Markdown)
	FilePath       string
	Line           int
	ReexportedFrom string
}

// Diagnostic reports a documentation-quality problem that does not
// necessarily prevent output from being produced.
type Diagnostic struct {
	Code     string
	Severity string
	Message  string
	FilePath string
	Line     int
	Col      int
}

// Report is the machine-readable extraction result used by JSON
// output and by text / HTML modes for diagnostics.
type Report struct {
	Version     string
	Items       []DocItem
	Diagnostics []Diagnostic
}

// ExtractFile parses one tya source file and returns its documented
// top-level bindings. Files that fail to lex or parse return an
// error; callers may choose to surface or skip them.
func ExtractFile(path string) ([]DocItem, error) {
	parsed, err := parseFile(path)
	if err != nil {
		return nil, err
	}

	return docItemsForProgram(parsed.prog, parsed.comments, path), nil
}

// ExtractFiles processes multiple files and returns the merged
// item list sorted by Kind+Name. Per-file errors are returned to
// the caller as a joined error string but the successfully parsed
// items are still returned so callers can render partial output.
func ExtractFiles(paths []string) ([]DocItem, error) {
	report, err := ExtractReport(paths)
	if err != nil {
		return report.Items, err
	}
	return report.Items, nil
}

// ExtractReport processes multiple files and returns documented items,
// re-exported items reachable through imports among those files, and
// stable diagnostics.
func ExtractReport(paths []string) (Report, error) {
	parsed := map[string]*parsedFile{}
	root := map[string]bool{}
	var fatal []string
	for _, p := range paths {
		root[p] = true
		file, err := parseFile(p)
		if err != nil {
			fatal = append(fatal, fmt.Sprintf("%s: %s", p, err.Error()))
			continue
		}
		parsed[p] = file
	}
	report := Report{Version: "1"}
	for _, p := range paths {
		file := parsed[p]
		if file == nil {
			continue
		}
		report.Items = append(report.Items, file.items...)
		for _, c := range parser.OrphanComments(file.prog, file.comments) {
			if !c.IsFullLine || c.Indent > 0 {
				continue
			}
			report.Diagnostics = append(report.Diagnostics, Diagnostic{
				Code:     "TYADOC0001",
				Severity: "warning",
				Message:  "orphan doc comment is not attached to a public item",
				FilePath: p,
				Line:     c.Line,
				Col:      c.Col,
			})
		}
		for _, item := range file.items {
			if diags := ValidateMarkdown(item.RawDoc, item.FilePath, item.Line); len(diags) > 0 {
				report.Diagnostics = append(report.Diagnostics, diags...)
			}
		}
	}
	report.Items = append(report.Items, collectReexports(paths, parsed, root, &report)...)
	sortItems(report.Items)
	report.Diagnostics = append(report.Diagnostics, duplicateDiagnostics(report.Items)...)
	sortDiagnostics(report.Diagnostics)
	if len(fatal) > 0 {
		return report, fmt.Errorf("%s", strings.Join(fatal, "\n"))
	}
	return report, nil
}

func kindOrder(kind string) int {
	switch kind {
	case "module":
		return 0
	case "class":
		return 1
	case "interface":
		return 2
	case "function":
		return 3
	case "method":
		return 4
	default:
		return 9
	}
}

type parsedFile struct {
	path     string
	prog     *ast.Program
	comments []parser.CommentInfo
	items    []DocItem
	imports  []importInfo
}

type importInfo struct {
	name string
	line int
	col  int
}

func parseFile(path string) (*parsedFile, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	toks, lcomments, lerrs := lexer.LexWithComments(string(src))
	if len(lerrs) > 0 {
		return nil, lerrs[0]
	}
	infos := make([]parser.CommentInfo, 0, len(lcomments))
	for _, c := range lcomments {
		infos = append(infos, parser.CommentInfo{
			Line:       c.Line,
			Col:        c.Col,
			Indent:     c.Indent,
			Text:       c.Text,
			IsFullLine: c.IsFullLine,
		})
	}
	prog, _, err := parser.ParseWithComments(toks, infos)
	if err != nil {
		return nil, err
	}
	file := &parsedFile{path: path, prog: prog, comments: infos}
	file.items = append(file.items, docItemsForProgram(prog, infos, path)...)
	for _, stmt := range prog.Stmts {
		if imp, ok := stmt.(*ast.ImportStmt); ok {
			file.imports = append(file.imports, importInfo{
				name: imp.Name,
				line: imp.NameTok.Line,
				col:  imp.NameTok.Col,
			})
		}
	}
	return file, nil
}

func docItemsForProgram(prog *ast.Program, comments []parser.CommentInfo, path string) []DocItem {
	var items []DocItem
	for _, stmt := range prog.Stmts {
		if item, ok := stmtToDocItem(stmt, prog, path); ok {
			items = append(items, item)
		}
		switch d := stmt.(type) {
		case *ast.ClassDecl:
			for _, method := range d.Methods {
				if method.Private {
					continue
				}
				items = append(items, DocItem{
					Name:      d.Name + "." + method.Name,
					Kind:      "method",
					Signature: MethodSignature(d.Name, method),
					RawDoc:    leadingBeforeLine(comments, method.Tok.Line, 2),
					FilePath:  path,
					Line:      method.Tok.Line,
				})
			}
		case *ast.InterfaceDecl:
			for _, method := range d.Methods {
				items = append(items, DocItem{
					Name:      d.Name + "." + method.Name,
					Kind:      "method",
					Signature: InterfaceMethodSignature(d.Name, method),
					RawDoc:    leadingBeforeLine(comments, method.Tok.Line, 2),
					FilePath:  path,
					Line:      method.Tok.Line,
				})
			}
		}
	}
	return items
}

func collectReexports(paths []string, parsed map[string]*parsedFile, root map[string]bool, report *Report) []DocItem {
	moduleIndex := buildModuleIndex(paths)
	var out []DocItem
	seen := map[string]bool{}
	for _, p := range paths {
		file := parsed[p]
		if file == nil {
			continue
		}
		stack := map[string]bool{p: true}
		for _, imp := range file.imports {
			target := resolveImportPath(imp.name, p, moduleIndex)
			if target == "" {
				continue
			}
			out = append(out, reexportsFrom(target, p, parsed, moduleIndex, root, report, stack, seen)...)
		}
	}
	return out
}

func reexportsFrom(path, reexporter string, parsed map[string]*parsedFile, moduleIndex map[string]string, root map[string]bool, report *Report, stack map[string]bool, seen map[string]bool) []DocItem {
	if stack[path] {
		report.Diagnostics = append(report.Diagnostics, Diagnostic{
			Code:     "TYADOC0004",
			Severity: "error",
			Message:  "import cycle encountered while following documentation re-exports",
			FilePath: path,
			Line:     1,
			Col:      1,
		})
		return nil
	}
	file := parsed[path]
	if file == nil {
		parsedFile, err := parseFile(path)
		if err != nil {
			return nil
		}
		file = parsedFile
		parsed[path] = file
	}
	stack[path] = true
	defer delete(stack, path)

	var out []DocItem
	if !root[path] {
		for _, item := range file.items {
			clone := item
			clone.ReexportedFrom = reexporter
			key := clone.Kind + "\x00" + clone.Name + "\x00" + clone.FilePath + "\x00" + reexporter
			if !seen[key] {
				seen[key] = true
				out = append(out, clone)
			}
		}
	}
	for _, imp := range file.imports {
		target := resolveImportPath(imp.name, path, moduleIndex)
		if target == "" {
			continue
		}
		out = append(out, reexportsFrom(target, reexporter, parsed, moduleIndex, root, report, stack, seen)...)
	}
	return out
}

func resolveImportPath(name, importer string, moduleIndex map[string]string) string {
	if path := moduleIndex[name]; path != "" {
		return path
	}
	candidates := []string{
		filepath.Join(filepath.Dir(importer), filepath.FromSlash(name)+".tya"),
		filepath.Join(filepath.Dir(importer), filepath.FromSlash(name), "Package.tya"),
		filepath.Join(filepath.Dir(importer), filepath.FromSlash(name), "package.tya"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			moduleIndex[name] = candidate
			return candidate
		}
	}
	return ""
}

func buildModuleIndex(paths []string) map[string]string {
	index := map[string]string{}
	for _, p := range paths {
		noExt := strings.TrimSuffix(filepath.ToSlash(p), ".tya")
		parts := strings.Split(noExt, "/")
		for i := 0; i < len(parts); i++ {
			name := strings.Join(parts[i:], "/")
			if _, exists := index[name]; !exists {
				index[name] = p
			}
		}
	}
	return index
}

func duplicateDiagnostics(items []DocItem) []Diagnostic {
	first := map[string]DocItem{}
	var out []Diagnostic
	for _, item := range items {
		key := filepath.Dir(item.FilePath) + "\x00" + item.Kind + "\x00" + item.Name
		if prev, ok := first[key]; ok {
			out = append(out, Diagnostic{
				Code:     "TYADOC0002",
				Severity: "error",
				Message:  fmt.Sprintf("duplicate public documentation name %s %s; first defined at %s:%d", item.Kind, item.Name, prev.FilePath, prev.Line),
				FilePath: item.FilePath,
				Line:     item.Line,
				Col:      1,
			})
			continue
		}
		first[key] = item
	}
	return out
}

func sortItems(items []DocItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			return kindOrder(items[i].Kind) < kindOrder(items[j].Kind)
		}
		if items[i].Name != items[j].Name {
			return items[i].Name < items[j].Name
		}
		if items[i].FilePath != items[j].FilePath {
			return items[i].FilePath < items[j].FilePath
		}
		return items[i].ReexportedFrom < items[j].ReexportedFrom
	})
}

func sortDiagnostics(diags []Diagnostic) {
	sort.SliceStable(diags, func(i, j int) bool {
		if diags[i].FilePath != diags[j].FilePath {
			return diags[i].FilePath < diags[j].FilePath
		}
		if diags[i].Line != diags[j].Line {
			return diags[i].Line < diags[j].Line
		}
		if diags[i].Col != diags[j].Col {
			return diags[i].Col < diags[j].Col
		}
		return diags[i].Code < diags[j].Code
	})
}

func stmtToDocItem(stmt ast.Stmt, prog *ast.Program, path string) (DocItem, bool) {
	leading := ""
	if prog.Comments != nil {
		if sc, ok := prog.Comments[stmt]; ok && len(sc.Leading) > 0 {
			leading = stripLeading(sc.Leading)
		}
	}

	switch d := stmt.(type) {
	case *ast.ClassDecl:
		return DocItem{
			Name:      d.Name,
			Kind:      "class",
			Signature: "class " + d.Name,
			RawDoc:    leading,
			FilePath:  path,
			Line:      d.NameTok.Line,
		}, true
	case *ast.ModuleDecl:
		return DocItem{
			Name:      d.Name,
			Kind:      "module",
			Signature: "module " + d.Name,
			RawDoc:    leading,
			FilePath:  path,
			Line:      d.NameTok.Line,
		}, true
	case *ast.InterfaceDecl:
		return DocItem{
			Name:      d.Name,
			Kind:      "interface",
			Signature: "interface " + d.Name,
			RawDoc:    leading,
			FilePath:  path,
			Line:      d.NameTok.Line,
		}, true
	case *ast.AssignStmt:
		if len(d.Targets) != 1 || len(d.Values) != 1 {
			return DocItem{}, false
		}
		id, ok := d.Targets[0].(*ast.Ident)
		if !ok {
			return DocItem{}, false
		}
		fn, ok := d.Values[0].(*ast.FuncLit)
		if !ok {
			return DocItem{}, false
		}
		return DocItem{
			Name:      id.Name,
			Kind:      "function",
			Signature: FuncSignature(id.Name, fn),
			RawDoc:    leading,
			FilePath:  path,
			Line:      d.Tok.Line,
		}, true
	}
	return DocItem{}, false
}

// stripLeading takes the slice of raw leading-comment text returned
// by parser.ParseWithComments (each item is the body after the
// initial `#`, including a possibly leading space) and joins it into
// a single Markdown blob. It trims one optional leading space per
// line so Markdown indentation behaves as expected.
func stripLeading(lines []string) string {
	out := make([]string, len(lines))
	for i, line := range lines {
		if strings.HasPrefix(line, " ") {
			out[i] = line[1:]
		} else {
			out[i] = line
		}
	}
	return strings.Join(out, "\n")
}

func leadingBeforeLine(comments []parser.CommentInfo, line int, indent int) string {
	var lines []string
	expected := line - 1
	for i := len(comments) - 1; i >= 0; i-- {
		c := comments[i]
		if c.Line > expected {
			continue
		}
		if c.Line < expected {
			if len(lines) == 0 {
				continue
			}
			break
		}
		if !c.IsFullLine || c.Indent != indent {
			if len(lines) == 0 {
				continue
			}
			break
		}
		lines = append([]string{c.Text}, lines...)
		expected--
	}
	return stripLeading(lines)
}

// FuncSignature renders a "name(p1, p2)" string from a FuncLit's
// parameter names. Type information is not available in the AST so
// this is the maximum precision v0.51 ships.
func FuncSignature(name string, fn *ast.FuncLit) string {
	return name + "(" + strings.Join(fn.Params, ", ") + ")"
}

func MethodSignature(className string, method ast.ClassMethod) string {
	return className + "." + FuncSignature(method.Name, method.Func)
}

func InterfaceMethodSignature(interfaceName string, method ast.InterfaceMethod) string {
	return interfaceName + "." + method.Name + "(" + strings.Join(method.Params, ", ") + ")"
}
