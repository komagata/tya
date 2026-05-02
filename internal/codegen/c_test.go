package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tya/internal/ast"
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

func TestEmitCIncludesSourceLineComments(t *testing.T) {
	prog := checkedProgram(t, "x = 1\nprint x\n")
	csrc, err := EmitC(prog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(csrc, "/* tya:1 */") {
		t.Fatalf("missing source line comment:\n%s", csrc)
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
	src := "text = \"  hello,tya  \"\ntrimmed = trim text\nparts = split trimmed, \",\"\nprint join parts, \"-\"\nprint replace trimmed, \"tya\", \"Tya\"\nprint contains trimmed, \"hello\"\nprint starts_with trimmed, \"hello\"\nprint ends_with trimmed, \"tya\"\nprint byte_len \"ちゃ\"\nprint char_len \"ちゃ\"\nprint \"quote: \\\"tya\\\"\"\nprint \"tya\"[1]\n"
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
	src := "items = [1, 2, 3, 4]\ndouble = item -> item * 2\nis_even = item -> item % 2 == 0\nadd = total, item -> total + item\ndoubled = map items, double\nevens = filter items, is_even\nfirst_even = find items, is_even\nhas_even = any items, is_even\nall_even = all items, is_even\nsum = reduce items, 0, add\neach items, double\nprint doubled[2]\nprint len evens\nprint first_even\nprint has_even\nprint all_even\nprint sum\n"
	out := compileAndRun(t, src)
	if string(out) != "6\n2\n2\ntrue\nfalse\n10\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesFunctionValueCalls(t *testing.T) {
	src := "double = item -> item * 2\nalias = double\ncallbacks = [double]\nprint alias 3\nprint callbacks[0](4)\n"
	out := compileAndRun(t, src)
	if string(out) != "6\n8\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesUserFunctionCallStatements(t *testing.T) {
	src := "append = items, item ->\n  push items, item\nitems = []\nappend items, \"Tya\"\nprint len items\n"
	out := compileAndRun(t, src)
	if string(out) != "1\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesFunctionLiteralsAsValues(t *testing.T) {
	src := "callbacks = [item -> item * 2]\nprint callbacks[0](5)\n"
	out := compileAndRun(t, src)
	if string(out) != "10\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesNoParenCallWithFunctionLiteralArgument(t *testing.T) {
	src := "items = [1, 2, 3]\ndoubled = map items, item -> item * 2\nprint doubled[2]\n"
	out := compileAndRun(t, src)
	if string(out) != "6\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesFileAndConversionProgram(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(path, []byte("12\na\n"), 0644); err != nil {
		t.Fatal(err)
	}
	src := "path = args()[0]\ntext = read_file path\nparts = split text, \"\\n\"\nfirst = to_int parts[0]\nprint first + 8\nprint to_string true\nprint join parts, \":\"\n"
	out := compileAndRunArgs(t, src, path)
	if string(out) != "20\ntrue\n12:a:\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesWriteFileProgram(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output.txt")
	src := "path = args()[0]\nwrite_file path, \"Hello\"\nprint file_exists path\nprint read_file path\n"
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
	src := "name = read_line()\nprint \"Hello, {name}\"\n"
	out := compileAndRunWithInput(t, src, "komagata\n")
	if string(out) != "Hello, komagata\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesExitProgram(t *testing.T) {
	src := "items = args()\nif len(items) > 0\n  exit to_int items[0]\nprint \"no exit\"\n"
	out, code := compileAndRunArgsAllowExit(t, src, "7")
	if string(out) != "" || code != 7 {
		t.Fatalf("got output %q and code %d", out, code)
	}
}

func TestEmitCCompilesPanicProgram(t *testing.T) {
	out, code := compileAndRunArgsAllowExit(t, "panic \"bad state\"\n")
	if string(out) != "panic: bad state\n" || code != 1 {
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
	src := "add = a, b -> a + b\nprint add 2, 3\nfind_first_over = limit ->\n  i = 0\n  while true\n    if i > limit\n      return i\n    i = i + 1\nprint find_first_over 3\n"
	out := compileAndRun(t, src)
	if string(out) != "5\n4\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesMultipleReturnProgram(t *testing.T) {
	src := "parse_user = text ->\n  if text == \"\"\n    return nil, error \"empty user\"\n  return { name: text }, nil\n\nuser, err = parse_user \"komagata\"\nif err\n  print err.message\nelse\n  print user.name\n\nmissing, err = parse_user \"\"\nif err\n  print err.message\nelse\n  print missing.name\n"
	out := compileAndRun(t, src)
	if string(out) != "komagata\nempty user\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesTryProgram(t *testing.T) {
	src := "parse_user = text ->\n  if text == \"\"\n    return nil, error \"empty user\"\n  return { name: text }, nil\n\nread_user = text ->\n  user = try parse_user(text)\n  return user.name, nil\n\nname, err = read_user \"komagata\"\nif err\n  print err.message\nelse\n  print name\n\nname, err = read_user \"\"\nif err\n  print err.message\nelse\n  print name\n"
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
	prog := checkedProgram(t, src)
	csrc, err := EmitC(prog)
	if err != nil {
		t.Fatal(err)
	}
	return compileAndRunC(t, csrc, input, args...)
}

func checkedProgram(t *testing.T, src string) *ast.Program {
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
	return prog
}

func compileAndRunC(t *testing.T, csrc string, input string, args ...string) ([]byte, int) {
	t.Helper()
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
