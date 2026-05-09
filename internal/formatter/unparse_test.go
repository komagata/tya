package formatter

import (
	"strings"
	"testing"

	"tya/internal/lexer"
	"tya/internal/parser"
)

func unparseSource(t *testing.T, src string) (string, error) {
	t.Helper()
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errs: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	return Unparse(prog)
}

func TestUnparseAssignAndPrint(t *testing.T) {
	src := "x = 1\nprint x\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := "x = 1\nprint x\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestUnparseBinaryAndCall(t *testing.T) {
	src := "y = 2 + 3 * 4\nprint y\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "y = 2 + 3 * 4") {
		t.Errorf("got: %q", got)
	}
}

func TestUnparseIfElseifElse(t *testing.T) {
	src := "x = 1\nif x == 0\n  print \"a\"\nelseif x == 1\n  print \"b\"\nelse\n  print \"c\"\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"if x == 0",
		"  print \"a\"",
		"elseif x == 1",
		"  print \"b\"",
		"else",
		"  print \"c\"",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestUnparseLambdaSingleLine(t *testing.T) {
	src := "add = a, b -> a + b\nprint add(2, 3)\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "add = a, b -> a + b") {
		t.Errorf("got: %q", got)
	}
}

func TestUnparseLambdaBlock(t *testing.T) {
	src := "f = x ->\n  y = x + 1\n  return y\nprint f(2)\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"f = x ->", "  y = x + 1", "  return y", "print f(2)"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestUnparseImports(t *testing.T) {
	src := "import string\nimport file as f\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "import string") {
		t.Errorf("got: %q", got)
	}
	if !strings.Contains(got, "import file as f") {
		t.Errorf("got: %q", got)
	}
}

func TestUnparseUnsupportedReturnsError(t *testing.T) {
	src := "module m\n  helper = -> 1\n"
	_, err := unparseSource(t, src)
	if err == nil {
		t.Error("expected unsupported error for module decl in v0.37")
	}
}
