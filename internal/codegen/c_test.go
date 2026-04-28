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

func TestEmitCCompilesModuloAndCKeywordNames(t *testing.T) {
	src := "double = item -> item % 2\nprint double 5\n"
	out := compileAndRun(t, src)
	if string(out) != "1\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesArrayFunctionBuiltins(t *testing.T) {
	src := "items = [1, 2, 3, 4]\ndouble = item -> item * 2\nisEven = item -> item % 2 == 0\nadd = total, item -> total + item\ndoubled = map items, double\nevens = filter items, isEven\nfirstEven = find items, isEven\nhasEven = any items, isEven\nallEven = all items, isEven\nsum = reduce items, 0, add\neach items, double\nprint doubled[2]\nprint len evens\nprint firstEven\nprint hasEven\nprint allEven\nprint sum\n"
	out := compileAndRun(t, src)
	if string(out) != "6\n2\n2\ntrue\nfalse\n10\n" {
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

func TestEmitCCompilesErrorProgram(t *testing.T) {
	src := "err = error \"file not found\"\nprint err\nprint err.message\n"
	out := compileAndRun(t, src)
	if string(out) != "error: file not found\nfile not found\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesArgsAndEnvProgram(t *testing.T) {
	t.Setenv("TYA_EXAMPLE", "hello")
	src := "items = args()\nprint len items\nprint env \"TYA_EXAMPLE\"\n"
	out := compileAndRunArgs(t, src, "foo")
	if string(out) != "1\nhello\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesReadLineProgram(t *testing.T) {
	src := "name = readLine()\nprint \"Hello, {name}\"\n"
	out := compileAndRunWithInput(t, src, "komagata\n")
	if string(out) != "Hello, komagata\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesExitProgram(t *testing.T) {
	src := "items = args()\nif len(items) > 0\n  exit toInt items[0]\nprint \"no exit\"\n"
	out, code := compileAndRunArgsAllowExit(t, src, "7")
	if string(out) != "" || code != 7 {
		t.Fatalf("got output %q and code %d", out, code)
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

func TestEmitCCompilesMultipleReturnProgram(t *testing.T) {
	src := "parseUser = text ->\n  if text == \"\"\n    return nil, error \"empty user\"\n  return { name: text }, nil\n\nuser, err = parseUser \"komagata\"\nif err\n  print err.message\nelse\n  print user.name\n\nmissing, err = parseUser \"\"\nif err\n  print err.message\nelse\n  print missing.name\n"
	out := compileAndRun(t, src)
	if string(out) != "komagata\nempty user\n" {
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

func TestEmitCCompilesMethodProgram(t *testing.T) {
	src := "counter =\n  count: 0\n\n  inc: ->\n    @count = @count + 1\n    @count\n\nprint counter.inc()\nprint counter.inc()\n"
	out := compileAndRun(t, src)
	if string(out) != "1\n2\n" {
		t.Fatalf("got %q", out)
	}
}

func compileAndRun(t *testing.T, src string) []byte {
	t.Helper()
	return compileAndRunArgs(t, src)
}

func compileAndRunArgs(t *testing.T, src string, args ...string) []byte {
	t.Helper()
	return compileAndRunArgsWithInput(t, src, "", args...)
}

func compileAndRunWithInput(t *testing.T, src string, input string, args ...string) []byte {
	t.Helper()
	return compileAndRunArgsWithInput(t, src, input, args...)
}

func compileAndRunArgsWithInput(t *testing.T, src string, input string, args ...string) []byte {
	t.Helper()
	out, code := compileAndRunArgsWithInputAllowExit(t, src, input, args...)
	if code != 0 {
		t.Fatalf("exit code %d\n%s", code, out)
	}
	return out
}

func compileAndRunArgsAllowExit(t *testing.T, src string, args ...string) ([]byte, int) {
	t.Helper()
	return compileAndRunArgsWithInputAllowExit(t, src, "", args...)
}

func compileAndRunArgsWithInputAllowExit(t *testing.T, src string, input string, args ...string) ([]byte, int) {
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
	if input != "" {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			defer stdin.Close()
			_, _ = stdin.Write([]byte(input))
		}()
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return out, exitErr.ExitCode()
		}
		t.Fatal(err)
	}
	return out, 0
}
