package eval

import (
	"bytes"
	"testing"

	"tya/internal/lexer"
	"tya/internal/parser"
)

func TestRunArithmeticAndLiterals(t *testing.T) {
	src := "add = a, b -> a + b\nprint add 2, 3\nprint 2 + 3 * 4\nprint 5 / 2\nprint true\nprint nil\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "5\n14\n2.5\ntrue\nnil\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}
