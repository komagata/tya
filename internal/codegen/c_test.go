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
	src := "items = [1, 2]\npush items, 3\nprint len items\nprint items[2]\nitems[1] = 20\nprint items[1]\nprint pop items\nprint len items\n"
	out := compileAndRun(t, src)
	if string(out) != "3\n3\n20\n3\n2\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesStringProgram(t *testing.T) {
	src := "text = \"  hello,tya  \"\ntrimmed = trim text\nparts = split trimmed, \",\"\nprint join parts, \"-\"\nprint replace trimmed, \"tya\", \"Tya\"\nprint contains trimmed, \"hello\"\nprint startsWith trimmed, \"hello\"\nprint endsWith trimmed, \"tya\"\nprint byteLen \"ちゃ\"\nprint charLen \"ちゃ\"\nprint \"quote: \\\"tya\\\"\"\nprint \"tya\"[1]\n"
	out := compileAndRun(t, src)
	if string(out) != "hello-tya\nhello,Tya\ntrue\ntrue\ntrue\n6\n2\nquote: \"tya\"\ny\n" {
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

func TestEmitCCompilesFileAndConversionProgram(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(path, []byte("12\na\n"), 0644); err != nil {
		t.Fatal(err)
	}
	src := "path = args()[0]\ntext = readFile path\nparts = split text, \"\\n\"\nfirst = toInt parts[0]\nprint first + 8\nprint toString true\nprint join parts, \":\"\n"
	out := compileAndRunArgs(t, src, path)
	if string(out) != "20\ntrue\n12:a:\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesWriteFileProgram(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output.txt")
	src := "path = args()[0]\nwriteFile path, \"Hello\"\nprint fileExists path\nprint readFile path\n"
	out := compileAndRunArgs(t, src, path)
	if string(out) != "true\nHello\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesForInProgram(t *testing.T) {
	src := "items = [1, 2, 3]\ntotal = 0\nfor item in items\n  total = total + item\nprint total\n"
	out := compileAndRun(t, src)
	if string(out) != "6\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesFunctionProgram(t *testing.T) {
	src := "add = a, b -> a + b\nprint add 2, 3\nfindFirstOver = limit ->\n  i = 0\n  while true\n    if i > limit\n      return i\n    i = i + 1\nprint findFirstOver 3\n"
	out := compileAndRun(t, src)
	if string(out) != "5\n4\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesStringInterpolationProgram(t *testing.T) {
	src := "name = \"Tya\"\nline = 3\nprint \"{line}:IDENT:{name}\"\nprint \"next: {line + 1}\"\n"
	out := compileAndRun(t, src)
	if string(out) != "3:IDENT:Tya\nnext: 4\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesObjectProgram(t *testing.T) {
	src := "user =\n  name: \"komagata\"\n  age: 20\nprint user.name\nprint len user\n"
	out := compileAndRun(t, src)
	if string(out) != "komagata\n2\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesMemberInterpolationProgram(t *testing.T) {
	src := "greet = user -> \"Hello, {user.name}\"\nuser =\n  name: \"komagata\"\nprint greet user\n"
	out := compileAndRun(t, src)
	if string(out) != "Hello, komagata\n" {
		t.Fatalf("got %q", out)
	}
}

func compileAndRun(t *testing.T, src string) []byte {
	t.Helper()
	return compileAndRunArgs(t, src)
}

func compileAndRunArgs(t *testing.T, src string, args ...string) []byte {
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
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	return out
}
