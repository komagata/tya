package lsp

import (
	"tya/internal/lexer"
	"tya/internal/parser"
)

// Definition answers textDocument/definition. Returns the
// defining location for the identifier under the cursor when it
// resolves to a same-file top-level binding. Cross-file resolution
// is v0.53+ work.
func Definition(doc *Document, line, character int) ([]Location, error) {
	if doc == nil {
		return nil, nil
	}
	toks, lcomments, lerrs := lexer.LexWithComments(doc.Text)
	if len(lerrs) > 0 {
		return nil, nil
	}
	prog, err := parser.ParseWithComments(toks, toCommentInfos(lcomments))
	if err != nil || prog == nil {
		return nil, nil
	}
	id := FindIdentAt(prog, line+1, character+1)
	if id == nil {
		return nil, nil
	}
	idx := BuildSymbols(prog)
	sym, ok := idx.Lookup(id.Name)
	if !ok {
		return nil, nil
	}
	return []Location{{
		URI:   doc.URI,
		Range: rangeAt(sym.NameTok.Line, sym.NameTok.Col, len(sym.Name)),
	}}, nil
}
