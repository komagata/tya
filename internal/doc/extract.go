// Package doc implements the v0.51 `tya doc` source documentation
// generator. It walks top-level declarations in tya source files,
// pulls their leading `#` comment block as Markdown body, and
// renders either plain text or a multi-page HTML static site.
package doc

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"tya/internal/ast"
	"tya/internal/lexer"
	"tya/internal/parser"
)

// DocItem is one documented top-level binding extracted from a
// source file.
type DocItem struct {
	Name      string
	Kind      string // "function" | "class" | "module" | "interface"
	Signature string
	RawDoc    string // Leading comments joined with "\n" (raw Markdown)
	FilePath  string
	Line      int
}

// ExtractFile parses one tya source file and returns its documented
// top-level bindings. Files that fail to lex or parse return an
// error; callers may choose to surface or skip them.
func ExtractFile(path string) ([]DocItem, error) {
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
	prog, err := parser.ParseWithComments(toks, infos)
	if err != nil {
		return nil, err
	}

	var items []DocItem
	for _, stmt := range prog.Stmts {
		item, ok := stmtToDocItem(stmt, prog, path)
		if !ok {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// ExtractFiles processes multiple files and returns the merged
// item list sorted by Kind+Name. Per-file errors are returned to
// the caller as a joined error string but the successfully parsed
// items are still returned so callers can render partial output.
func ExtractFiles(paths []string) ([]DocItem, error) {
	var all []DocItem
	var errs []string
	for _, p := range paths {
		items, err := ExtractFile(p)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %s", p, err.Error()))
			continue
		}
		all = append(all, items...)
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].Kind != all[j].Kind {
			return kindOrder(all[i].Kind) < kindOrder(all[j].Kind)
		}
		return all[i].Name < all[j].Name
	})
	if len(errs) > 0 {
		return all, fmt.Errorf("%s", strings.Join(errs, "\n"))
	}
	return all, nil
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
	default:
		return 9
	}
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
		if isPrivateName(d.Name) {
			return DocItem{}, false
		}
		return DocItem{
			Name:      d.Name,
			Kind:      "class",
			Signature: "class " + d.Name,
			RawDoc:    leading,
			FilePath:  path,
			Line:      d.NameTok.Line,
		}, true
	case *ast.ModuleDecl:
		if isPrivateName(d.Name) {
			return DocItem{}, false
		}
		return DocItem{
			Name:      d.Name,
			Kind:      "module",
			Signature: "module " + d.Name,
			RawDoc:    leading,
			FilePath:  path,
			Line:      d.NameTok.Line,
		}, true
	case *ast.InterfaceDecl:
		if isPrivateName(d.Name) {
			return DocItem{}, false
		}
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
		if isPrivateName(id.Name) {
			return DocItem{}, false
		}
		fn, ok := d.Values[0].(*ast.FuncLit)
		if !ok {
			return DocItem{}, false
		}
		return DocItem{
			Name:      id.Name,
			Kind:      "function",
			Signature: funcSignature(id.Name, fn),
			RawDoc:    leading,
			FilePath:  path,
			Line:      d.Tok.Line,
		}, true
	}
	return DocItem{}, false
}

// isPrivateName treats top-level bindings whose name starts with
// "_" as private, per the v0.51 SPEC.
func isPrivateName(name string) bool {
	return strings.HasPrefix(name, "_")
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

// funcSignature renders a "name(p1, p2)" string from a FuncLit's
// parameter names. Type information is not available in the AST so
// this is the maximum precision v0.51 ships.
func funcSignature(name string, fn *ast.FuncLit) string {
	return name + "(" + strings.Join(fn.Params, ", ") + ")"
}
