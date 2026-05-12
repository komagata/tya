package lsp

import (
	"fmt"
	"strings"

	"tya/internal/lexer"
	"tya/internal/parser"
)

// Hover answers textDocument/hover. The position is LSP 0-origin;
// the returned Hover (if non-nil) uses an LSP Range.
func HoverAt(doc *Document, line, character int) (*Hover, error) {
	if doc == nil {
		return nil, nil
	}
	toks, lcomments, lerrs := lexer.LexWithComments(doc.Text)
	if len(lerrs) > 0 {
		return nil, nil
	}
	prog, _, err := parser.ParseWithComments(toks, toCommentInfos(lcomments))
	if err != nil || prog == nil {
		return nil, nil
	}
	id := FindIdentAt(prog, line+1, character+1)
	if id == nil {
		return nil, nil
	}
	idx := BuildSymbols(prog)
	if sym, ok := idx.Lookup(id.Name); ok {
		return &Hover{
			Contents: MarkupContent{Kind: MarkupMarkdown, Value: renderHoverMarkdown(sym)},
			Range:    rangePtr(rangeAt(id.Tok.Line, id.Tok.Col, len(id.Name))),
		}, nil
	}
	if isStdlibModule(id.Name) {
		return &Hover{
			Contents: MarkupContent{Kind: MarkupMarkdown, Value: fmt.Sprintf("```tya\nimport %s\n```\n\nStandard attached module.", id.Name)},
			Range:    rangePtr(rangeAt(id.Tok.Line, id.Tok.Col, len(id.Name))),
		}, nil
	}
	if isBuiltinName(id.Name) {
		return &Hover{
			Contents: MarkupContent{Kind: MarkupMarkdown, Value: fmt.Sprintf("```tya\n%s(…)\n```\n\nBuilt-in function. See [API](https://tya-lang.org/api.html).", id.Name)},
			Range:    rangePtr(rangeAt(id.Tok.Line, id.Tok.Col, len(id.Name))),
		}, nil
	}
	return nil, nil
}

func renderHoverMarkdown(sym Symbol) string {
	var b strings.Builder
	fmt.Fprintf(&b, "```tya\n%s\n```\n", sym.Signature)
	if strings.TrimSpace(sym.DocBody) != "" {
		b.WriteString("\n")
		b.WriteString(sym.DocBody)
		if !strings.HasSuffix(sym.DocBody, "\n") {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func rangePtr(r Range) *Range { return &r }
