package lsp

import (
	"testing"

	"tya/internal/lexer"
	"tya/internal/parser"
)

func TestBuildSymbolsIncludesUnderscoreTopLevelFunction(t *testing.T) {
	src := "_helper = value -> value\n"
	toks, comments, errs := lexer.LexWithComments(src)
	if len(errs) > 0 {
		t.Fatal(errs[0])
	}
	prog, _, err := parser.ParseWithComments(toks, toCommentInfos(comments))
	if err != nil {
		t.Fatal(err)
	}

	idx := BuildSymbols(prog)
	if _, ok := idx.Lookup("_helper"); !ok {
		t.Fatalf("expected _helper in symbol index, got %#v", idx.All())
	}
}
