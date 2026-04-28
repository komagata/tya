package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"tya/internal/checker"
	"tya/internal/lexer"
	"tya/internal/parser"
)

func TestEmitCCompilesSimpleProgram(t *testing.T) {
	src := "x = 2 + 3 * 4\nprint x\n"
	out := compileAndRun(t, src)
	if string(out) != "14\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesArrayProgram(t *testing.T) {
	src := "items = [1, 2]\npush items, 3\nprint len items\nprint items[2]\n"
	out := compileAndRun(t, src)
	if string(out) != "3\n3\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesStringProgram(t *testing.T) {
	src := "text = \"hello\"\nprint len text\nprint contains text, \"ell\"\nprint text[1]\n"
	out := compileAndRun(t, src)
	if string(out) != "5\ntrue\ne\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesEqualityProgram(t *testing.T) {
	src := "print \"tya\" == \"tya\"\nprint \"tya\" == \"Tya\"\nprint 2 != 3\nprint true == true\nprint true and not false\n"
	out := compileAndRun(t, src)
	if string(out) != "true\nfalse\ntrue\ntrue\ntrue\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesAdditionProgram(t *testing.T) {
	src := "print 2 + 3\nprint \"Ty\" + \"a\"\n"
	out := compileAndRun(t, src)
	if string(out) != "5\nTya\n" {
		t.Fatalf("got %q", out)
	}
}

func compileAndRun(t *testing.T, src string) []byte {
	t.Helper()
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	if err := checker.Check(prog); err != nil {
		t.Fatal(err)
	}
	csrc, err := EmitC(prog)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cfile := filepath.Join(dir, "main.c")
	bin := filepath.Join(dir, "main")
	if err := os.WriteFile(cfile, []byte(csrc), 0644); err != nil {
		t.Fatal(err)
	}
	runtime := filepath.Join("..", "..", "runtime", "tya_runtime.c")
	include := filepath.Join("..", "..", "runtime")
	if out, err := exec.Command("gcc", cfile, runtime, "-I", include, "-o", bin).CombinedOutput(); err != nil {
		t.Fatalf("gcc: %v\n%s\n%s", err, out, csrc)
	}
	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	return out
}
