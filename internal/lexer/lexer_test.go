package lexer

import "testing"

func TestLexIndentAndDedent(t *testing.T) {
	toks, errs := Lex("user =\n  name: \"komagata\"\nprint \"ok\"\n")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	var got []string
	for _, tok := range toks {
		got = append(got, string(tok.Type))
	}
	wantHas := map[string]bool{"INDENT": false, "DEDENT": false}
	for _, typ := range got {
		if _, ok := wantHas[typ]; ok {
			wantHas[typ] = true
		}
	}
	for typ, ok := range wantHas {
		if !ok {
			t.Fatalf("missing %s in %#v", typ, got)
		}
	}
}

func TestLexRejectsTabs(t *testing.T) {
	_, errs := Lex("user =\n\tname: \"komagata\"\n")
	if len(errs) == 0 {
		t.Fatal("expected tab error")
	}
}
