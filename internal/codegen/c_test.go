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

func TestEmitCFieldAssignmentDoesNotOverwriteSameNamedMethod(t *testing.T) {
	src := "class Response\n  initialize: ->\n    self.status = 200\n    self.bump()\n\n  bump: ->\n    self.status = self.status + 1\n\n  status: ->\n    self.status\n\nresponse = Response()\nprint(response.status())\nresponse.bump()\nprint(response.status())\n"
	out := compileAndRun(t, src)
	if string(out) != "201\n202\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCAllowsInstanceAndClassPrivatePredicateMethods(t *testing.T) {
	src := "class Address\n  private?: -> true\n\n  static private?: -> false\n\naddr = Address()\nprint(addr.private?())\nprint(Address.private?())\n"
	out := compileAndRun(t, src)
	if string(out) != "true\nfalse\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCAllowsSelfClassConstantInFieldInitializer(t *testing.T) {
	src := "class Csv\n  SEPARATOR: \",\"\n\n  options: { separator: Self.SEPARATOR, header: false }\n\ncsv = Csv()\nprint(csv.options[\"separator\"])\n"
	out := compileAndRun(t, src)
	if string(out) != ",\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCEnvironmentAndProcessProgram(t *testing.T) {
	src := "setenv(\"TYA_CODEGEN_ENV\", \"ok\")\nprint(env(\"TYA_CODEGEN_ENV\"))\nr = process_run([\"sh\", \"-c\", \"printf $TYA_CODEGEN_CHILD\"], { env: { \"TYA_CODEGEN_CHILD\": \"child\" } })\nprint(r[\"status\"])\nprint(r[\"success\"])\nprint(r[\"stdout\"])\n"
	out := compileAndRun(t, src)
	want := "ok\n0\ntrue\nchild\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestEmitCFilesystemUtilitiesProgram(t *testing.T) {
	t.Setenv("TYA_PATH", filepath.Clean(filepath.Join("..", "..", "stdlib")))
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	src := filepath.Join(root, "src.bin")
	dst := filepath.Join(root, "dst.bin")
	code := "import file/* as file\nimport dir/* as dir\nimport bytes/* as bytes\ndir.Dir(\"" + root + "\").mkdir_all()\nfile.File(\"" + src + "\").write_bytes(bytes.Bytes([0, 255, 65]).from_array())\nfile.File(\"" + src + "\").copy(\"" + dst + "\", {})\nprint(bytes.Bytes(file.File(\"" + dst + "\").read_bytes()).to_array()[1])\nseen = []\nrecord = entry -> seen.push(entry[\"name\"])\ndir.Dir(\"" + root + "\").walk(record, {})\nprint(seen.len() >= 0)\ntmp = file.File().temp(\"tya-cg\", \".tmp\")\nprint(file.File(tmp).exists?())\nfile.File(tmp).remove()\ndir.Dir(\"" + root + "\").remove_all()\nprint(file.File(\"" + root + "\").exists?())\n"
	main := filepath.Join(dir, "main.tya")
	if err := os.WriteFile(main, []byte(code), 0644); err != nil {
		t.Fatal(err)
	}
	out := compileAndRunFile(t, main)
	want := "255\ntrue\ntrue\nfalse\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestEmitCHmacProgram(t *testing.T) {
	t.Setenv("TYA_PATH", filepath.Clean(filepath.Join("..", "..", "stdlib")))
	dir := t.TempDir()
	main := filepath.Join(dir, "main.tya")
	code := "import hmac/* as hmac\nprint(hmac.Hmac(\"sha256\", \"key\").hexdigest(\"The quick brown fox jumps over the lazy dog\"))\nprint(hmac.Hmac(\"sha256\", \"key\").verify(\"The quick brown fox jumps over the lazy dog\", \"f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8\"))\n"
	if err := os.WriteFile(main, []byte(code), 0644); err != nil {
		t.Fatal(err)
	}
	out := compileAndRunFile(t, main)
	want := "f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8\ntrue\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestEmitCRegexProgram(t *testing.T) {
	t.Setenv("TYA_PATH", filepath.Clean(filepath.Join("..", "..", "stdlib")))
	dir := t.TempDir()
	main := filepath.Join(dir, "main.tya")
	code := "import regex/* as regex\nrx = regex.Regex(\"([a-z]+)=([0-9]+)\").compile()\nfound = rx.find(\"a=1 b=22\")\nprint(found[\"text\"])\nprint(found[\"groups\"][1])\nprint(rx.find_all(\"a=1 b=22\").len())\nprint(regex.Regex(\", *\").compile().split(\"a, b, c\").join(\"|\"))\nprint(rx.replace(\"a=1 b=22\", r\"${2}\", 1))\n"
	if err := os.WriteFile(main, []byte(code), 0644); err != nil {
		t.Fatal(err)
	}
	out := compileAndRunFile(t, main)
	want := "a=1\n1\n2\na|b|c\n1 b=22\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestEmitCTimeContractProgram(t *testing.T) {
	t.Setenv("TYA_PATH", filepath.Clean(filepath.Join("..", "..", "stdlib")))
	dir := t.TempDir()
	main := filepath.Join(dir, "main.tya")
	code := "import time/* as time\nt = time.Time().unix(1704067200)\nprint(t.format(\"rfc3339\"))\nprint(time.Time(\"2024-01-01T00:00:00Z\").parse().format(\"unix\"))\nd = time.Time().duration(1, { milliseconds: 500 })\nprint(d.milliseconds())\nprint(t.add(d).sub(t).nanoseconds() == 1500000000)\ntime.Time().sleep(time.Time().duration(0))\n"
	if err := os.WriteFile(main, []byte(code), 0644); err != nil {
		t.Fatal(err)
	}
	out := compileAndRunFile(t, main)
	want := "2024-01-01T00:00:00Z\n1704067200\n1500\ntrue\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
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
	src := "text = \"  hello,tya  \"\ntrimmed = text.trim()\nparts = trimmed.split(\",\")\nprint(parts.join(\"-\"))\nprint(trimmed.replace(\"tya\", \"Tya\"))\nprint(trimmed.contains(\"hello\"))\nprint(trimmed.starts_with(\"hello\"))\nprint(trimmed.ends_with(\"tya\"))\nprint(trimmed.slice(1, 4))\nprint(\"あいう\".slice(1, 3))\nprint(\"quote: \\\"tya\\\"\")\nprint(\"tya\"[1])\n"
	out := compileAndRun(t, src)
	if string(out) != "hello-tya\nhello,Tya\ntrue\ntrue\ntrue\nell\nいう\nquote: \"tya\"\ny\n" {
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
	src := "import file/* as file\npath = args()[0]\ntext = file.File(path).read()\nparts = text.split(\"\\n\")\nfirst = parts[0].to_i()\nprint(first + 8)\nprint(true.to_s())\nprint(parts.join(\":\"))\n"
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
	src := "import file/* as file\npath = args()[0]\nfile.File(path).write(\"Hello\")\nprint(file.File(path).exists?())\nprint(file.File(path).read())\n"
	if err := os.WriteFile(main, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out := compileAndRunFile(t, main, path)
	if string(out) != "true\nHello\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesErrorProgram(t *testing.T) {
	src := "err = error(\"file not found\")\nprint(err)\nprint(err[\"message\"])\ncause = error(\"root\")\ntagged = error(\"missing\", { kind: \"io\", code: \"file_not_found\", data: { path: \"missing.txt\" }, cause: cause })\nprint(tagged[\"kind\"])\nprint(tagged[\"code\"])\nprint(tagged[\"data\"][\"path\"])\nprint(tagged[\"cause\"][\"message\"])\n"
	out := compileAndRun(t, src)
	if string(out) != "file not found\nfile not found\nio\nfile_not_found\nmissing.txt\nroot\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesTryCatchFinally(t *testing.T) {
	src := "try\n  print(\"try\")\ncatch err\n  print(err)\nfinally\n  print(\"finally\")\ntry\n  raise error(\"boom\")\ncatch err\n  print(err[\"message\"])\nfinally\n  print(\"cleanup\")\n"
	out := compileAndRun(t, src)
	if string(out) != "try\nfinally\nboom\ncleanup\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesTryFinallyReraises(t *testing.T) {
	src := "try\n  raise error(\"boom\")\nfinally\n  print(\"cleanup\")\n"
	out, code := compileAndRunArgsAllowExit(t, src)
	if code == 0 {
		t.Fatalf("expected non-zero exit, got %q", out)
	}
	if !strings.Contains(string(out), "cleanup") || !strings.Contains(string(out), "boom") {
		t.Fatalf("unexpected output: %q", out)
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
	src := "bounds = items ->\n  return items[0], items[items.len() - 1]\n\nmin, max = bounds([1, 2, 3])\nprint(min)\nprint(max)\n"
	out := compileAndRun(t, src)
	if string(out) != "1\n3\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesStructuredErrorProgram(t *testing.T) {
	src := "try\n  raise error(\"missing\", { kind: \"io\", code: \"file_not_found\", data: { path: \"missing.txt\" } })\ncatch err\n  print(err[\"message\"])\n  print(err[\"kind\"])\n  print(err[\"code\"])\n  print(err[\"data\"][\"path\"])\n"
	out := compileAndRun(t, src)
	if string(out) != "missing\nio\nfile_not_found\nmissing.txt\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCCompilesRaiseCatchErrorProgram(t *testing.T) {
	src := "raise_and_catch = ->\n  try\n    raise error(\"failed\")\n  catch err\n    print(err[\"message\"])\nraise_and_catch()\n"
	out := compileAndRun(t, src)
	if string(out) != "failed\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEmitCRejectsRaiseNonErrorProgram(t *testing.T) {
	_, code := compileAndRunArgsAllowExit(t, "raise \"failed\"\n")
	if code == 0 {
		t.Fatal("expected raise non-error program to fail")
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
	src := "user =\n  name: \"komagata\"\n  age: 20\nprint(user[\"name\"])\nprint(user.len())\nuser[\"city\"] = \"Tokyo\"\nprint(user[\"city\"])\nprint(user.update({ name: \"Tya\", lang: \"tya\" }))\nprint(user[\"name\"])\nprint(user[\"lang\"])\n"
	out := compileAndRun(t, src)
	if string(out) != "komagata\n2\nTokyo\nnil\nTya\ntya\n" {
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
	src := "foo = \"foo\"\nbar = -> \"bar\"\nutil = { foo: foo, bar: bar }\nprint(util[\"foo\"])\nprint(util[\"bar\"]())\n"
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

func TestEmitCCompilesAliasedPackageClassWithSameNamedLocalClass(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "cli"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "cli", "cli.tya"), "class Cli\n  static parse_or_exit: args, spec ->\n    return Cli.parse(args, spec)\n\n  static parse: args, spec ->\n    return { args: args, spec: spec }\n")
	writeFile(t, filepath.Join(dir, "cli.tya"), "import cli/* as cli\n\nclass Cli\n  static main: argv ->\n    parsed = cli.Cli.parse_or_exit(argv, Self.option_spec())\n    println(\"ok\")\n\n  static option_spec: ->\n    return { command: \"demo\", options: {} }\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "Cli.main(args())\n")
	out := compileAndRunFile(t, main)
	if string(out) != "ok\n" {
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
