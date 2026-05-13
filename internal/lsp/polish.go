package lsp

import (
	"path/filepath"
	"regexp"
	"strings"

	"tya/internal/ast"
	"tya/internal/lexer"
	"tya/internal/parser"
)

func PrepareRename(doc *Document, line, character int) (*PrepareRenameResult, error) {
	if doc == nil {
		return nil, &RenameError{msg: "[TYA-E0933] no open document for rename"}
	}
	prog := parseOrNil(doc.Text)
	if prog == nil {
		return nil, &RenameError{msg: "[TYA-E0933] cannot rename: file has parse errors"}
	}
	kind, name := ScopeKindAt(prog, line+1, character+1)
	if kind == ScopeKindUnknown || name == "" {
		return nil, &RenameError{msg: "[TYA-E0933] cursor is not on a known binding"}
	}
	id := FindIdentAt(prog, line+1, character+1)
	if id == nil {
		return nil, &RenameError{msg: "[TYA-E0933] cursor is not on a known binding"}
	}
	return &PrepareRenameResult{Range: rangeAt(id.Tok.Line, id.Tok.Col, len(id.Name)), Placeholder: id.Name}, nil
}

func SelectionRangesFor(doc *Document, positions []Position) []SelectionRange {
	out := make([]SelectionRange, 0, len(positions))
	lines := strings.Split(doc.Text, "\n")
	for _, pos := range positions {
		word := wordRangeAt(lines, pos)
		lineRange := Range{Start: Position{Line: pos.Line, Character: 0}, End: Position{Line: pos.Line, Character: len(lineAt(lines, pos.Line))}}
		full := Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: len(lines) - 1, Character: len(lineAt(lines, len(lines)-1))}}
		out = append(out, SelectionRange{Range: word, Parent: &SelectionRange{Range: lineRange, Parent: &SelectionRange{Range: full}}})
	}
	return out
}

func FoldingRangesFor(src string) []FoldingRange {
	lines := strings.Split(src, "\n")
	out := []FoldingRange{}
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if !(strings.HasPrefix(trim, "class ") || strings.HasPrefix(trim, "module ") || strings.HasPrefix(trim, "interface ") || strings.HasSuffix(trim, "->") || strings.HasSuffix(trim, ":")) {
			continue
		}
		indent := leadingSpaces(line)
		end := i
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "" {
				continue
			}
			if leadingSpaces(lines[j]) <= indent {
				break
			}
			end = j
		}
		if end > i {
			out = append(out, FoldingRange{StartLine: i, EndLine: end})
		}
	}
	return out
}

func DocumentLinksFor(doc *Document) []DocumentLink {
	toks, errs := lexer.Lex(doc.Text)
	if len(errs) > 0 {
		return nil
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		return nil
	}
	out := []DocumentLink{}
	for _, stmt := range prog.Stmts {
		imp, ok := stmt.(*ast.ImportStmt)
		if !ok {
			continue
		}
		target := resolveImportLink(doc.Path, imp.Name)
		if target == "" {
			continue
		}
		out = append(out, DocumentLink{Range: rangeAt(imp.NameTok.Line, imp.NameTok.Col, len(imp.Name)), Target: target})
	}
	for _, link := range markdownLinks(doc.Text) {
		out = append(out, link)
	}
	return out
}

func CodeLensesFor(doc *Document) []CodeLens {
	prog := parseOrNil(doc.Text)
	if prog == nil {
		return nil
	}
	out := []CodeLens{}
	for _, stmt := range prog.Stmts {
		class, ok := stmt.(*ast.ClassDecl)
		if !ok || !strings.HasSuffix(class.Name, "Test") {
			continue
		}
		out = append(out, CodeLens{
			Range: rangeAt(class.NameTok.Line, class.NameTok.Col, len(class.Name)),
			Command: Command{
				Title:     "Run tya test",
				Command:   "tya.test",
				Arguments: []any{doc.URI, class.Name},
			},
		})
	}
	return out
}

func InlayHintsFor(doc *Document, rng Range) []InlayHint {
	return []InlayHint{}
}

func PrepareCallHierarchy(doc *Document, pos Position) []CallHierarchyItem {
	prog := parseOrNil(doc.Text)
	if prog == nil {
		return []CallHierarchyItem{}
	}
	id := FindIdentAt(prog, pos.Line+1, pos.Character+1)
	if id == nil {
		return []CallHierarchyItem{}
	}
	return []CallHierarchyItem{{
		Name:           id.Name,
		Kind:           SymbolKindFunction,
		URI:            doc.URI,
		Range:          rangeAt(id.Tok.Line, id.Tok.Col, len(id.Name)),
		SelectionRange: rangeAt(id.Tok.Line, id.Tok.Col, len(id.Name)),
	}}
}

func EmptyIncomingCalls() []CallHierarchyIncomingCall { return []CallHierarchyIncomingCall{} }
func EmptyOutgoingCalls() []CallHierarchyOutgoingCall { return []CallHierarchyOutgoingCall{} }

func wordRangeAt(lines []string, pos Position) Range {
	line := lineAt(lines, pos.Line)
	start := pos.Character
	if start > len(line) {
		start = len(line)
	}
	end := start
	for start > 0 && isIdentByte(line[start-1]) {
		start--
	}
	for end < len(line) && isIdentByte(line[end]) {
		end++
	}
	if start == end {
		end = start + 1
		if end > len(line) {
			end = len(line)
		}
	}
	return Range{Start: Position{Line: pos.Line, Character: start}, End: Position{Line: pos.Line, Character: end}}
}

func lineAt(lines []string, line int) string {
	if line < 0 || line >= len(lines) {
		return ""
	}
	return lines[line]
}

func isIdentByte(b byte) bool {
	return b == '_' || b == '?' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func leadingSpaces(s string) int {
	n := 0
	for n < len(s) && s[n] == ' ' {
		n++
	}
	return n
}

func resolveImportLink(importerPath, name string) string {
	if importerPath == "" || name == "" {
		return ""
	}
	candidates := []string{
		filepath.Join(filepath.Dir(importerPath), name+".tya"),
		filepath.Join(filepath.Dir(importerPath), name),
	}
	for _, c := range candidates {
		if uri, err := PathToURI(c); err == nil {
			return uri
		}
	}
	return ""
}

var markdownLinkRE = regexp.MustCompile(`\[[^\]]+\]\((https?://[^)]+)\)`)

func markdownLinks(src string) []DocumentLink {
	out := []DocumentLink{}
	lines := strings.Split(src, "\n")
	for lineNo, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		for _, m := range markdownLinkRE.FindAllStringSubmatchIndex(line, -1) {
			out = append(out, DocumentLink{
				Range:  Range{Start: Position{Line: lineNo, Character: m[2]}, End: Position{Line: lineNo, Character: m[3]}},
				Target: line[m[2]:m[3]],
			})
		}
	}
	return out
}
