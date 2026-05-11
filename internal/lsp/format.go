package lsp

import (
	"strings"

	"tya/internal/formatter"
	"tya/internal/lexer"
	"tya/internal/parser"
)

// Format runs the canonical formatter on src and returns a single
// full-replace TextEdit. If src is already canonical, an empty
// slice is returned so clients short-circuit applying the edit.
// Lex / parse failures bubble up as errors so the caller can decide
// whether to suppress them; the LSP handler converts them to a
// JSON-RPC error response.
func Format(src string) ([]TextEdit, error) {
	toks, lcomments, lerrs := lexer.LexWithComments(src)
	if len(lerrs) > 0 {
		return nil, lerrs[0]
	}
	prog, err := parser.ParseWithComments(toks, toCommentInfos(lcomments))
	if err != nil {
		return nil, err
	}
	out, err := formatter.Unparse(prog)
	if err != nil {
		return nil, err
	}
	if out == src {
		return []TextEdit{}, nil
	}
	endLine := strings.Count(src, "\n")
	if !strings.HasSuffix(src, "\n") && src != "" {
		endLine++
	}
	return []TextEdit{{
		Range: Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: endLine, Character: 0},
		},
		NewText: out,
	}}, nil
}
