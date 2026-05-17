package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tya/internal/ast"
	"tya/internal/checker"
	"tya/internal/lexer"
	"tya/internal/parser"
	"tya/internal/runner"
)

var runtimeOS = runtime.GOOS

func TestEmitCCompilesSimpleProgram(t *testing.T) {
	src := "x = 2 + 3 * 4\nprint(x)\n"
	out := compileAndRun(t, src)
	if string(out) != "14\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCIncludesSourceLineComments(t *testing.T) {
	prog := checkedProgram(t, "x = 1\nprint(x)\n")
	csrc, _, err := EmitC(prog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(csrc, "/* tya:1 */") {
		t.Fatalf("missing source line comment:\n%s", csrc)
	}
}

func TestEmitCWithCoverageCollectsCodegenDiagnostics(t *testing.T) {
	prog := checkedProgram(t, "embed \"missing-one.txt\" as one\nprint(\"ok\")\nembed \"missing-two.txt\" as two\n")
	sourcePath := filepath.Join(t.TempDir(), "main.tya")
	csrc, reg, diags, err := EmitCWithCoverage(prog, sourcePath, &CoverageOptions{})
	if err == nil {
		t.Fatal("expected codegen error")
	}
	if csrc != "" {
		t.Fatalf("expected empty C source, got %q", csrc)
	}
	if reg != nil {
		t.Fatalf("expected nil coverage registry, got %#v", reg)
	}
	if len(diags) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d: %#v", len(diags), diags)
	}
	ce, ok := AsCodegenError(err)
	if !ok {
		t.Fatalf("expected CodegenError, got %T", err)
	}
	if len(ce.Diags) != len(diags) {
		t.Fatalf("error diagnostics length mismatch: %d != %d", len(ce.Diags), len(diags))
	}
	if ce.Diags[0].Code != codeEmbedNotFound || ce.Diags[1].Code != codeEmbedNotFound {
		t.Fatalf("unexpected codes: %#v", ce.Diags)
	}
	if !strings.Contains(err.Error(), "embed source not found: missing-one.txt") {
		t.Fatalf("legacy error string missing first diagnostic: %q", err.Error())
	}
}

func TestEmitCWithCoverageIncludesBreakAndContinue(t *testing.T) {
	prog := checkedProgram(t, "i = 0\nwhile i < 3\n  i = i + 1\n  if i == 1\n    continue\n  break\n")
	sourcePath := filepath.Join(t.TempDir(), "main.tya")
	_, reg, diags, err := EmitCWithCoverage(prog, sourcePath, &CoverageOptions{})
	if err != nil {
		t.Fatalf("EmitCWithCoverage: %v %#v", err, diags)
	}
	lines := map[int]bool{}
	for _, e := range reg.Entries {
		lines[e.Line] = true
	}
	if !lines[5] || !lines[6] {
		t.Fatalf("missing break/continue coverage entries: %+v", reg.Entries)
	}
}

func TestEmitCCompilesArrayProgram(t *testing.T) {
	src := "items = [1, 2]\nitems.push(3)\nprint(items.len())\nprint(items[2])\nitems[1] = 20\nprint(items[1])\nprint(items.pop())\nprint(items.len())\n"
	out := compileAndRun(t, src)
	if string(out) != "3\n3\n20\n3\n2\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesStringProgram(t *testing.T) {
	src := "text = \"  hello,tya  \"\ntrimmed = text.trim()\nparts = trimmed.split(\",\")\nprint(parts.join(\"-\"))\nprint(trimmed.replace(\"tya\", \"Tya\"))\nprint(trimmed.contains(\"hello\"))\nprint(trimmed.starts_with(\"hello\"))\nprint(trimmed.ends_with(\"tya\"))\nprint(\"quote: \\\"tya\\\"\")\nprint(\"tya\"[1])\n"
	out := compileAndRun(t, src)
	if string(out) != "hello-tya\nhello,Tya\ntrue\ntrue\ntrue\nquote: \"tya\"\ny\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesEqualityProgram(t *testing.T) {
	src := "print(\"tya\" == \"tya\")\nprint(\"tya\" == \"Tya\")\nprint(2 != 3)\nprint(true == true)\nprint(true and not false)\n"
	out := compileAndRun(t, src)
	if string(out) != "true\nfalse\ntrue\ntrue\ntrue\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesAdditionProgram(t *testing.T) {
	src := "print(2 + 3)\nprint(\"Ty\" + \"a\")\nprint(b\"Ty\" + b\"a\")\ncount = 3\nprint(\"count: {count}\")\n"
	out := compileAndRun(t, src)
	if string(out) != "5\nTya\n<bytes:3>\ncount: 3\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCRejectsMixedStringPlus(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{name: "string number", src: "value = -> 3\nprint(\"count: \" + value())\n"},
		{name: "number string", src: "value = -> 3\nprint(value() + \" items\")\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, code := compileAndRunArgsAllowExit(t, tt.src)
			if code == 0 {
				t.Fatalf("expected non-zero exit, got output %q", out)
			}
			if !strings.Contains(string(out), "+ expects numbers, strings, or bytes of the same kind") {
				t.Fatalf("unexpected output: %q", out)
			}
		})
	}
}

func TestEmitCCompilesModuloAndCKeywordNames(t *testing.T) {
	src := "double = item -> item % 2\nprint(double(5))\n"
	out := compileAndRun(t, src)
	if string(out) != "1\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesFunctionValueCalls(t *testing.T) {
	src := "double = item -> item * 2\nalias = double\ncallbacks = [double]\nprint(alias(3))\nprint(callbacks[0](4))\n"
	out := compileAndRun(t, src)
	if string(out) != "6\n8\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesUserFunctionCallStatements(t *testing.T) {
	src := "append = items, item ->\n  items.push(item)\nitems = []\nappend(items, \"Tya\")\nprint(items.len())\n"
	out := compileAndRun(t, src)
	if string(out) != "1\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesFunctionLiteralsAsValues(t *testing.T) {
	src := "callbacks = [item -> item * 2]\nprint(callbacks[0](5))\n"
	out := compileAndRun(t, src)
	if string(out) != "10\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesFileAndConversionProgram(t *testing.T) {
	t.Setenv("TYA_PATH", filepath.Clean(filepath.Join("..", "..", "stdlib")))
	dir := t.TempDir()
	path := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(path, []byte("12\na\n"), 0644); err != nil {
		t.Fatal(err)
	}
	main := filepath.Join(dir, "main.tya")
	src := "import file as file\npath = args()[0]\ntext = file.File.read(path)\nparts = text.split(\"\\n\")\nfirst = parts[0].to_i()\nprint(first + 8)\nprint(true.to_s())\nprint(parts.join(\":\"))\n"
	if err := os.WriteFile(main, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out := compileAndRunFile(t, main, path)
	if string(out) != "20\ntrue\n12:a:\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesWriteFileProgram(t *testing.T) {
	t.Setenv("TYA_PATH", filepath.Clean(filepath.Join("..", "..", "stdlib")))
	dir := t.TempDir()
	path := filepath.Join(dir, "output.txt")
	main := filepath.Join(dir, "main.tya")
	src := "import file as file\npath = args()[0]\nfile.File.write(path, \"Hello\")\nprint(file.File.exists?(path))\nprint(file.File.read(path))\n"
	if err := os.WriteFile(main, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out := compileAndRunFile(t, main, path)
	if string(out) != "true\nHello\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesErrorProgram(t *testing.T) {
	src := "err = error(\"file not found\")\nprint(err)\nprint(err[\"message\"])\n"
	out := compileAndRun(t, src)
	if string(out) != "error: file not found\nfile not found\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesArgsAndEnvProgram(t *testing.T) {
	t.Setenv("TYA_EXAMPLE", "hello")
	src := "items = args()\nprint(items.len())\nprint(env(\"TYA_EXAMPLE\"))\n"
	out := compileAndRunArgs(t, src, "foo")
	if string(out) != "1\nhello\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesExitProgram(t *testing.T) {
	src := "items = args()\nif items.len() > 0\n  exit(items[0].to_i())\nprint(\"no exit\")\n"
	out, code := compileAndRunArgsAllowExit(t, src, "7")
	if string(out) != "" || code != 7 {
		t.Fatalf("got output %q and code %d", out, code)
	}
}

func TestEmitCCompilesPanicProgram(t *testing.T) {
	out, code := compileAndRunArgsAllowExit(t, "panic(\"bad state\")\n")
	if string(out) != "panic: bad state\n" || code != 1 {
		t.Fatalf("got output %q and code %d", out, code)
	}
}

func TestEmitCCompilesForInProgram(t *testing.T) {
	src := "items = [1, 2, 3]\ntotal = 0\nfor item in items\n  total = total + item\nprint(total)\n"
	out := compileAndRun(t, src)
	if string(out) != "6\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesFunctionProgram(t *testing.T) {
	src := "add = a, b -> a + b\nprint(add(2, 3))\nfind_first_over = limit ->\n  i = 0\n  while true\n    if i > limit\n      return i\n    i = i + 1\nprint(find_first_over(3))\n"
	out := compileAndRun(t, src)
	if string(out) != "5\n4\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesMultipleReturnProgram(t *testing.T) {
	src := "parse_user = text ->\n  if text == \"\"\n    return nil, error(\"empty user\")\n  return { name: text }, nil\n\nuser, err = parse_user(\"komagata\")\nif err\n  print(err[\"message\"])\nelse\n  print(user[\"name\"])\n\nmissing, err = parse_user(\"\")\nif err\n  print(err[\"message\"])\nelse\n  print(missing[\"name\"])\n"
	out := compileAndRun(t, src)
	if string(out) != "komagata\nempty user\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesTryProgram(t *testing.T) {
	src := "parse_user = text ->\n  if text == \"\"\n    return nil, error(\"empty user\")\n  return { name: text }, nil\n\nread_user = text ->\n  user = try parse_user(text)\n  return user[\"name\"], nil\n\nname, err = read_user(\"komagata\")\nif err\n  print(err[\"message\"])\nelse\n  print(name)\n\nname, err = read_user(\"\")\nif err\n  print(err[\"message\"])\nelse\n  print(name)\n"
	out := compileAndRun(t, src)
	if string(out) != "komagata\nempty user\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesStringInterpolationProgram(t *testing.T) {
	src := "name = \"Tya\"\nline = 3\nprint(\"{line}:IDENT:{name}\")\nprint(\"next: {line + 1}\")\n"
	out := compileAndRun(t, src)
	if string(out) != "3:IDENT:Tya\nnext: 4\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesDictProgram(t *testing.T) {
	src := "user =\n  name: \"komagata\"\n  age: 20\nprint(user[\"name\"])\nprint(user.len())\nuser[\"city\"] = \"Tokyo\"\nprint(user[\"city\"])\n"
	out := compileAndRun(t, src)
	if string(out) != "komagata\n2\nTokyo\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesMemberInterpolationProgram(t *testing.T) {
	src := "greet = user -> \"Hello, \" + user[\"name\"]\nuser =\n  name: \"komagata\"\nprint(greet(user))\n"
	out := compileAndRun(t, src)
	if string(out) != "Hello, komagata\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesNamespaceDictionary(t *testing.T) {
	src := "foo = \"foo\"\nbar = -> \"bar\"\nutil = { foo: foo, bar: bar }\nprint(util.foo)\nprint(util.bar())\n"
	out := compileAndRun(t, src)
	if string(out) != "foo\nbar\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesImportedModuleMemberCalls(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greeting.tya"), "hello = name -> \"Hello, {name}\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting\nprint(greeting.hello(\"komagata\"))\n")
	out := compileAndRunFile(t, main)
	if string(out) != "Hello, komagata\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesImportedModuleAlias(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greeting.tya"), "hello = name -> \"Hello, {name}\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting as g\nsay = name -> g.hello(name)\nprint(say(\"komagata\"))\n")
	out := compileAndRunFile(t, main)
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
		t.Fatalf("exit(code) %d\n%s", code, out)
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
	csrc, _, err := EmitC(prog)
	if err != nil {
		t.Fatal(err)
	}
	return compileAndRunC(t, csrc, input, args...)
}

func compileAndRunFile(t *testing.T, path string, args ...string) []byte {
	t.Helper()
	source, modules, err := runner.LoadSourceWithModules(path)
	if err != nil {
		t.Fatal(err)
	}
	toks, errs := lexer.Lex(source)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	if err := checker.CheckWithModules(prog, modules); err != nil {
		t.Fatal(err)
	}
	csrc, _, err := EmitC(prog)
	if err != nil {
		t.Fatal(err)
	}
	out, code := compileAndRunC(t, csrc, "", args...)
	if code != 0 {
		t.Fatalf("exit(code) %d\n%s", code, out)
	}
	return out
}

func checkedProgram(t *testing.T, src string) *ast.Program {
	t.Helper()
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
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
	gccArgs := []string{cfile, runtime, "-I", include, "-o", bin}
	if runtimeOS == "linux" {
		gccArgs = append(gccArgs, "-lpthread", "-lm", "-lz")
	} else if runtimeOS == "windows" {
		gccArgs = append(gccArgs, "-lws2_32", "-lz")
	} else if runtimeOS != "windows" {
		gccArgs = append(gccArgs, "-lm", "-lz")
	}
	if out, err := exec.Command("gcc", gccArgs...).CombinedOutput(); err != nil {
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

func writeFile(t *testing.T, path string, src string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
}
