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
	if out, err := exec.Command("gcc", cfile, "-o", bin).CombinedOutput(); err != nil {
		t.Fatalf("gcc: %v\n%s\n%s", err, out, csrc)
	}
	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "14\n" {
		t.Fatalf("got %q", out)
	}
}
