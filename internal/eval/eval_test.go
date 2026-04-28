package eval

import (
	"bytes"
	"path/filepath"
	"testing"

	"tya/internal/lexer"
	"tya/internal/parser"
)

func TestRunArithmeticAndLiterals(t *testing.T) {
	src := "add = a, b -> a + b\nprint add 2, 3\nprint 2 + 3 * 4\nprint 5 / 2\nprint true\nprint nil\nage = 20\nprint \"next year: {age + 1}\"\n"
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
	want := "5\n14\n2.5\ntrue\nnil\nnext year: 21\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunIfElseTruthiness(t *testing.T) {
	src := "if nil\n  print \"bad\"\nelse\n  print \"nil\"\n\nif 0\n  print \"zero\"\n"
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
	want := "nil\nzero\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunComparisonAndLogic(t *testing.T) {
	src := "age = 20\nname = \"komagata\"\nif age >= 20 and name == \"komagata\"\n  print \"match\"\nprint nil or \"anonymous\"\nprint not false\n"
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
	want := "match\nanonymous\ntrue\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunArrays(t *testing.T) {
	src := "items = [1, 2, 3]\nprint len items\nprint items[0]\nprint items[9]\npush items, 4\nprint len items\nprint pop items\nprint len items\n"
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
	want := "3\n1\nnil\n4\n4\n3\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunWhile(t *testing.T) {
	src := "i = 0\nsum = 0\nwhile i < 5\n  i = i + 1\n  if i == 3\n    continue\n  sum = sum + i\n  if sum > 7\n    break\nprint sum\n"
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
	if out.String() != "12\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunRejectsBreakOutsideLoop(t *testing.T) {
	toks, errs := lexer.Lex("break\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunReturn(t *testing.T) {
	src := "findFirstOver = limit ->\n  i = 0\n  while true\n    if i > limit\n      return i\n    i = i + 1\nprint findFirstOver 3\n"
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
	if out.String() != "4\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunConversions(t *testing.T) {
	src := "print toString 20\nprint toInt \"42\"\nprint toFloat \"2.5\"\nprint toNumber \"12\"\nprint toNumber \"12.5\"\n"
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
	want := "20\n42\n2.5\n12\n12.5\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunStringBuiltins(t *testing.T) {
	src := "text = \"  hello,tya  \"\ntrimmed = trim text\nparts = split trimmed, \",\"\nprint join parts, \"-\"\nprint replace trimmed, \"tya\", \"Tya\"\nprint contains trimmed, \"hello\"\nprint startsWith trimmed, \"hello\"\nprint endsWith trimmed, \"tya\"\nprint byteLen \"ちゃ\"\nprint charLen \"ちゃ\"\n"
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
	want := "hello-tya\nhello,Tya\ntrue\ntrue\ntrue\n6\n2\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunObjectBuiltins(t *testing.T) {
	src := "user =\n  name: \"komagata\"\n  age: 20\n\nprint has user, \"name\"\nuserKeys = keys user\nuserValues = values user\nprint len userKeys\nprint len userValues\ndelete user, \"age\"\nprint has user, \"age\"\nprint user.age\n"
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
	want := "true\n2\n2\nfalse\nnil\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunForIn(t *testing.T) {
	src := "items = [2, 4, 6]\nsum = 0\nfor item in items\n  sum = sum + item\nprint sum\nfor item, index in items\n  print \"{index}:{item}\"\n"
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
	want := "12\n0:2\n1:4\n2:6\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunForOfObject(t *testing.T) {
	src := "user =\n  name: \"komagata\"\n  age: 20\ncount = 0\nfor key, value of user\n  if key == \"name\"\n    print value\n  count = count + 1\nprint count\n"
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
	want := "komagata\n2\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunFileBuiltins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memo.txt")
	src := "writeFile \"" + path + "\", \"hello\"\nprint fileExists \"" + path + "\"\nprint readFile \"" + path + "\"\n"
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
	want := "true\nhello\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunRejectsReturnOutsideFunction(t *testing.T) {
	toks, errs := lexer.Lex("return 1\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err == nil {
		t.Fatal("expected error")
	}
}
