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

func TestLexStringEscapes(t *testing.T) {
	toks, errs := Lex("print \"a\\n\\\"b\\\"\"\n")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if toks[1].Lexeme != "a\n\"b\"" {
		t.Fatalf("got %q", toks[1].Lexeme)
	}
}

func TestLexPredicateIdentifier(t *testing.T) {
	toks, errs := Lex("empty? = -> true\nprint empty?()\n")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if toks[0].Lexeme != "empty?" {
		t.Fatalf("got %q", toks[0].Lexeme)
	}
	if toks[6].Lexeme != "empty?" {
		t.Fatalf("got %q", toks[6].Lexeme)
	}
}
