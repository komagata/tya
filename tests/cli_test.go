package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLICheckUnusedRejectsUnusedBinding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unused.tya")
	if err := os.WriteFile(path, []byte("name = \"Tya\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "--check-unused", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected --check-unused to fail")
	}
	if !strings.Contains(string(out), "unused variable name") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLICheckUnusedAllowsUsedBinding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "used.tya")
	if err := os.WriteFile(path, []byte("name = \"Tya\"\nprint(name)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "--check-unused", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func buildTyaCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "tya")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/tya")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build tya CLI failed: %v\n%s", err, out)
	}
	return bin
}

func setRuntimeEnv(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	cmd.Env = append(os.Environ(), "TYA_RUNTIME_DIR="+filepath.Join(repo, "runtime"))
}

func TestCLIAllowsCombinedOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "used.tya")
	if err := os.WriteFile(path, []byte("name = \"Tya\"\nprint(name)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "--check-unused", "--emit-c", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "tya_print") {
		t.Fatalf("expected emitted C, got: %s", out)
	}
}

func TestCLITokensCanBeCombinedWithChecks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "used.tya")
	if err := os.WriteFile(path, []byte("name = \"Tya\"\nprint(name)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "--check-unused", "--tokens", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "IDENT") {
		t.Fatalf("expected tokens, got: %s", out)
	}
}

func TestCLIRunCompilesAndRunsProgram(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hello.tya")
	if err := os.WriteFile(path, []byte("print(\"Hello from run\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "run", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "Hello from run\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIRunAcceptedButUnformattedSyntax(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accepted.tya")
	src := "add = (a, b,) -> a + b\nname = 'Tya'\nitems = [1, 2,]\nuser = { name: name, age: 20, }\nprint(add(items[0], add(1, 2,)))\nprint(user[\"name\"])\n"
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "run", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "4\nTya\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIDirectFileDoesNotUseInterpreterPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hello.tya")
	if err := os.WriteFile(path, []byte("print(\"direct\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected direct file invocation to fail")
	}
	if !strings.Contains(string(out), "usage: tya run") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIRunPassesArgs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "args.tya")
	if err := os.WriteFile(path, []byte("items = args()\nprint(items.len())\nprint(items[0])\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "run", path, "first")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "1\nfirst\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIRunLoadsImportedModule(t *testing.T) {
	dir := t.TempDir()
	module := filepath.Join(dir, "greeting.tya")
	if err := os.WriteFile(module, []byte("hello = name -> \"Hello, {name}\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "main.tya")
	if err := os.WriteFile(path, []byte("import greeting\nprint(greeting.hello(\"komagata\"))\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "run", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "Hello, komagata\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIFormatDoesNotRewriteInvalidSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.tya")
	original := "x=1\nif\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "format", "-w", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected invalid format input to fail")
	}
	if !strings.Contains(string(out), "expected expression") {
		t.Fatalf("unexpected output: %s", out)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != original {
		t.Fatalf("format rewrote invalid source: %q", got)
	}
}

func TestCLIFormatAndCheckAcceptedSyntax(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accepted.tya")
	original := "add = (a, b,) -> a + b\nname = 'Tya'\nprint(add(1, 2,))\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	check := exec.Command("go", "run", "./cmd/tya", "format", "--check", path)
	check.Dir = ".."
	out, err := check.CombinedOutput()
	if err == nil {
		t.Fatal("expected format --check to report drift")
	}
	if !strings.Contains(string(out), "not in formatted syntax") {
		t.Fatalf("unexpected output: %s", out)
	}

	format := exec.Command("go", "run", "./cmd/tya", "format", "-w", path)
	format.Dir = ".."
	out, err = format.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected format error: %v\n%s", err, out)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	want := "add = a, b -> a + b\nname = \"Tya\"\nprint(add(1, 2))\n"
	if string(got) != want {
		t.Fatalf("got %q want %q", got, want)
	}
	check = exec.Command("go", "run", "./cmd/tya", "format", "--check", path)
	check.Dir = ".."
	out, err = check.CombinedOutput()
	if err != nil {
		t.Fatalf("formatted source should pass --check: %v\n%s", err, out)
	}
}

func TestCLIFormatAndCheckCollapsesImportBlankLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "imports.tya")
	original := strings.Join([]string{
		"import os",
		"",
		"import cli",
		"",
		"class Cli",
		"  initialize: ->",
		"    self.value = 1",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	check := exec.Command("go", "run", "./cmd/tya", "format", "--check", path)
	check.Dir = ".."
	out, err := check.CombinedOutput()
	if err == nil {
		t.Fatal("expected format --check to report import blank-line drift")
	}
	if !strings.Contains(string(out), "not in formatted syntax") {
		t.Fatalf("unexpected output: %s", out)
	}

	format := exec.Command("go", "run", "./cmd/tya", "format", "-w", path)
	format.Dir = ".."
	out, err = format.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected format error: %v\n%s", err, out)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	want := strings.Join([]string{
		"import os",
		"import cli",
		"",
		"class Cli",
		"  initialize: ->",
		"    self.value = 1",
		"",
	}, "\n")
	if string(got) != want {
		t.Fatalf("got %q want %q", got, want)
	}
	check = exec.Command("go", "run", "./cmd/tya", "format", "--check", path)
	check.Dir = ".."
	out, err = check.CombinedOutput()
	if err != nil {
		t.Fatalf("expected formatted imports to pass --check: %v\n%s", err, out)
	}
}

func TestCLIFormatAndCheckClassMemberOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "class_order.tya")
	original := strings.Join([]string{
		"class Sample",
		"  private helper: -> 1",
		"",
		"  static make: -> Self()",
		"",
		"  ALPHA: 1",
		"",
		"  ready: false",
		"",
		"  initialize: ->",
		"    self.ready = true",
		"",
		"  static build: -> Self()",
		"",
		"  name: \"\"",
		"",
		"print(\"ok\")",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	checkSyntax := exec.Command("go", "run", "./cmd/tya", "check", path)
	checkSyntax.Dir = ".."
	if out, err := checkSyntax.CombinedOutput(); err != nil {
		t.Fatalf("unordered class members should remain accepted syntax: %v\n%s", err, out)
	}
	checkFormat := exec.Command("go", "run", "./cmd/tya", "format", "--check", path)
	checkFormat.Dir = ".."
	out, err := checkFormat.CombinedOutput()
	if err == nil {
		t.Fatal("expected format --check to report class member order drift")
	}
	if !strings.Contains(string(out), "not in formatted syntax") {
		t.Fatalf("unexpected output: %s", out)
	}
	format := exec.Command("go", "run", "./cmd/tya", "format", "-w", path)
	format.Dir = ".."
	out, err = format.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected format error: %v\n%s", err, out)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	want := strings.Join([]string{
		"class Sample",
		"  ALPHA: 1",
		"",
		"  name: \"\"",
		"",
		"  ready: false",
		"",
		"  static build: -> Self()",
		"",
		"  static make: -> Self()",
		"",
		"  initialize: ->",
		"    self.ready = true",
		"",
		"  private helper: -> 1",
		"print(\"ok\")",
		"",
	}, "\n")
	if string(got) != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCLILintParsesAcceptedButUnformattedSyntax(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accepted.tya")
	src := "add = (a, b,) -> a + b\nprint(add(1, 2,))\n"
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "lint", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected lint parse error: %v\n%s", err, out)
	}
}

func TestFormatRejectsExcludedV1Syntax(t *testing.T) {
	path := filepath.Join(t.TempDir(), "excluded.tya")
	original := "items = [1, 2, 3]\nprint(items[1:3])\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "format", "-w", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected excluded syntax format input to fail")
	}
	if !strings.Contains(string(out), "slice syntax is not part of Tya") {
		t.Fatalf("unexpected output: %s", out)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != original {
		t.Fatalf("format rewrote excluded source: %q", got)
	}
}

func TestTyaTestDiscoversOnlyTestFilesAndOrdersDeterministically(t *testing.T) {
	dir := t.TempDir()
	writeTestFile := func(name, className, methodName string) {
		t.Helper()
		src := fmt.Sprintf("import unittest/*\n\nclass %s extends TestCase\n  %s: ->\n    self.assert(true, \"%s\")\n", className, methodName, methodName)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0644); err != nil {
			t.Fatal(err)
		}
	}
	writeTestFile("b_test.tya", "BTest", "test_second")
	writeTestFile("a_test.tya", "ATest", "test_first")
	if err := os.WriteFile(filepath.Join(dir, "helper.tya"), []byte("print(\"must not run\")\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "run", "./cmd/tya", "test", dir)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	text := string(out)
	first := strings.Index(text, "PASS  ATest.test_first")
	second := strings.Index(text, "PASS  BTest.test_second")
	if first < 0 || second < 0 || first > second {
		t.Fatalf("expected path and definition order, got:\n%s", text)
	}
	if strings.Contains(text, "must not run") {
		t.Fatalf("ordinary .tya file was discovered:\n%s", text)
	}
	if !strings.Contains(text, "2 tests, 2 passed, 0 failed") {
		t.Fatalf("unexpected summary:\n%s", text)
	}
}

func TestFormatClassInheritanceOutputRunsInTyaTest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample_test.tya")
	src := "import unittest/*\n\nclass SampleTest extends TestCase\n  test_example: () ->\n    self.assert(true, \"example\")\n"
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	format := exec.Command("go", "run", "./cmd/tya", "format", "-w", path)
	format.Dir = ".."
	if out, err := format.CombinedOutput(); err != nil {
		t.Fatalf("format failed: %v\n%s", err, out)
	}
	formatted, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(formatted), "class SampleTest extends TestCase") {
		t.Fatalf("formatted source missing canonical inheritance: %s", formatted)
	}
	testCmd := exec.Command("go", "run", "./cmd/tya", "test", path)
	testCmd.Dir = ".."
	if out, err := testCmd.CombinedOutput(); err != nil {
		t.Fatalf("tya test failed after format: %v\n%s", err, out)
	}
}

func TestCLIEmitCStableAndNoNondeterministicMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.tya")
	if err := os.WriteFile(path, []byte("print(\"stable\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run := func() string {
		t.Helper()
		cmd := exec.Command("go", "run", "./cmd/tya", "emit-c", path)
		cmd.Dir = ".."
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("emit-c failed: %v\n%s", err, out)
		}
		return string(out)
	}
	first := run()
	second := run()
	if first != second {
		t.Fatalf("emit-c output is not stable")
	}
	if strings.Contains(first, filepath.Dir(path)) {
		t.Fatalf("emit-c output contains absolute temp path:\n%s", first)
	}
}

func TestTyaCheckReportsMultipleRecoverableErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "strict.tya")
	src := strings.Join([]string{
		"outer = 1",
		"handler = value ->",
		"  outer = value",
		"  value",
		"other = unused ->",
		"  1",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "--format=json", "check", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected check to fail")
	}
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %v", err)
	}
	text := string(out)
	if !strings.Contains(text, `"code":"TYA-E0307"`) || !strings.Contains(text, `"code":"TYA-E0303"`) {
		t.Fatalf("expected multiple recoverable diagnostics, got:\n%s", text)
	}
	if !strings.Contains(text, `"errors":2`) {
		t.Fatalf("expected error summary, got:\n%s", text)
	}
}

func TestNoExperimentalFeatureWithoutSpec(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.tya")
	if err := os.WriteFile(path, []byte("print(\"ok\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "format", "--experimental-layout", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected unknown experimental option to fail")
	}
	if !strings.Contains(string(out), "unknown format option: --experimental-layout") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestFormatDoesNotRewriteTyaToml(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tya.toml")
	original := "name = \"demo\"\nversion = \"0.1.0\"\nlicense = \"MIT\"\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "format", "-w", path)
	cmd.Dir = ".."
	_, _ = cmd.CombinedOutput()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Fatalf("format rewrote tya.toml: %q", got)
	}
}

func TestRunReadsStdin(t *testing.T) {
	tya := buildTyaCLI(t)
	cmd := exec.Command(tya, "run", "-")
	cmd.Dir = t.TempDir()
	cmd.Stdin = strings.NewReader("print(\"stdin\")\n")
	setRuntimeEnv(t, cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "stdin\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCheckReadsStdin(t *testing.T) {
	tya := buildTyaCLI(t)
	cmd := exec.Command(tya, "check", "-")
	cmd.Dir = t.TempDir()
	cmd.Stdin = strings.NewReader("print(\"ok\")\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestStdinDiagnosticsUseStdinPath(t *testing.T) {
	tya := buildTyaCLI(t)
	cmd := exec.Command(tya, "check", "-")
	cmd.Dir = t.TempDir()
	cmd.Stdin = strings.NewReader("if\n")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected check to fail")
	}
	if !strings.Contains(string(out), "<stdin>") {
		t.Fatalf("expected <stdin> diagnostic path, got:\n%s", out)
	}
}

func TestStdinRelativeImportsUseCwd(t *testing.T) {
	dir := t.TempDir()
	tya := buildTyaCLI(t)
	if err := os.WriteFile(filepath.Join(dir, "helper.tya"), []byte("message = \"cwd import\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(tya, "run", "-")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader("import helper\nprint(helper.message)\n")
	setRuntimeEnv(t, cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "cwd import\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestRunArgsAfterSeparator(t *testing.T) {
	tya := buildTyaCLI(t)
	path := filepath.Join(t.TempDir(), "args.tya")
	if err := os.WriteFile(path, []byte("print(args()[0])\nprint(args()[1])\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(tya, "run", path, "--", "one", "two")
	setRuntimeEnv(t, cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "one\ntwo\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestRunArgsWithoutSeparatorStillSupported(t *testing.T) {
	tya := buildTyaCLI(t)
	path := filepath.Join(t.TempDir(), "args.tya")
	if err := os.WriteFile(path, []byte("print(args()[0])\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(tya, "run", path, "one")
	setRuntimeEnv(t, cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "one\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestBuildRejectsProgramArgs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.tya")
	if err := os.WriteFile(path, []byte("print(\"ok\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "build", path, "--", "one")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected build program args to fail")
	}
	if !strings.Contains(string(out), "unknown build option: --") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestNoCommonTimeoutFlag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.tya")
	if err := os.WriteFile(path, []byte("print(\"ok\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "check", "--timeout", "1s", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected timeout flag to fail")
	}
	if !strings.Contains(string(out), "usage: tya") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestJsonDiagnosticAliasSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.tya")
	if err := os.WriteFile(path, []byte("outer = 1\nhandler = value ->\n  outer = value\n  value\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "--json", "check", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected check to fail")
	}
	text := string(out)
	for _, key := range []string{`"code"`, `"severity"`, `"message"`, `"primary"`, `"file"`, `"line"`, `"col"`, `"hints"`, `"source"`, `"summary"`} {
		if !strings.Contains(text, key) {
			t.Fatalf("json diagnostic missing %s:\n%s", key, text)
		}
	}
}

func TestNoColorFlag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.tya")
	if err := os.WriteFile(path, []byte("print(missing)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "--no-color", "check", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected check to fail")
	}
	if strings.Contains(string(out), "\x1b[") {
		t.Fatalf("expected no ANSI color, got:\n%s", out)
	}
}

func TestVersionJsonIncludesVersions(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/tya", "version", "--json")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	text := string(out)
	for _, key := range []string{`"compiler"`, `"runtime"`, `"spec"`, `"selfhost"`} {
		if !strings.Contains(text, key) {
			t.Fatalf("version json missing %s: %s", key, text)
		}
	}
}

func TestPublishCommandUnavailable(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/tya", "publish")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected publish to fail")
	}
	if !strings.Contains(string(out), "invalid Tya file name: publish") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCleanRemovesBuildOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tya.toml"), []byte("name = \"demo\"\nversion = \"0.1.0\"\nlicense = \"MIT\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	buildDir := filepath.Join(dir, ".tya", "build")
	pkgDir := filepath.Join(dir, ".tya", "packages")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	tya := buildTyaCLI(t)
	cmd := exec.Command(tya, "clean")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if _, err := os.Stat(buildDir); !os.IsNotExist(err) {
		t.Fatalf("build dir still exists or stat failed: %v", err)
	}
	if _, err := os.Stat(pkgDir); err != nil {
		t.Fatalf("packages dir should remain: %v", err)
	}
}

func TestCleanPackagesRemovesDependencyCache(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tya.toml"), []byte("name = \"demo\"\nversion = \"0.1.0\"\nlicense = \"MIT\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(dir, ".tya", "packages")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	tya := buildTyaCLI(t)
	cmd := exec.Command(tya, "clean", "--packages")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if _, err := os.Stat(pkgDir); !os.IsNotExist(err) {
		t.Fatalf("packages dir still exists or stat failed: %v", err)
	}
}

func TestCLIBuildWritesExecutable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.tya")
	if err := os.WriteFile(path, []byte("print(\"Hello from build\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "hello-bin")
	cmd := exec.Command("go", "run", "./cmd/tya", "build", path, "-o", bin)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	run := exec.Command(bin)
	out, err = run.CombinedOutput()
	if err != nil {
		t.Fatalf("built executable failed: %v\n%s", err, out)
	}
	if string(out) != "Hello from build\n" {
		t.Fatalf("unexpected output: %s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".tya", "build", "main.c")); err != nil {
		t.Fatalf("missing intermediate C artifact under .tya/build: %v", err)
	}
}

func TestCLIBuildAcceptedButUnformattedSyntax(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "accepted.tya")
	if err := os.WriteFile(path, []byte("add = (a, b,) -> a + b\nprint(add(1, 2,))\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "accepted-bin")
	cmd := exec.Command("go", "run", "./cmd/tya", "build", path, "-o", bin)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("missing executable: %v", err)
	}
}

func TestCLIBuildUsesDefaultOutputPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "default_out.tya")
	if err := os.WriteFile(path, []byte("print(\"default output\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "build", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	bin := filepath.Join("..", "default_out")
	t.Cleanup(func() {
		_ = os.Remove(bin)
	})
	run := exec.Command(bin)
	out, err = run.CombinedOutput()
	if err != nil {
		t.Fatalf("built executable failed: %v\n%s", err, out)
	}
	if string(out) != "default output\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIBuildNativeTargetWritesExecutable(t *testing.T) {
	skipShort(t, "native build integration is covered by the release gate")

	dir := t.TempDir()
	path := filepath.Join(dir, "hello.tya")
	if err := os.WriteFile(path, []byte("print(\"native target\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "hello-native")
	cmd := exec.Command("go", "run", "./cmd/tya", "build", "--target", "native", path, "-o", bin)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	run := exec.Command(bin)
	out, err = run.CombinedOutput()
	if err != nil {
		t.Fatalf("built executable failed: %v\n%s", err, out)
	}
	if string(out) != "native target\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIBuildWasmTargets(t *testing.T) {
	skipShort(t, "WASM build integration is covered by the release gate")

	if _, err := exec.LookPath("zig"); err != nil {
		t.Skip("zig not installed")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.tya")
	if err := os.WriteFile(path, []byte("print(\"hello wasm\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	wasiOut := filepath.Join(dir, "hello.wasm")
	cmd := exec.Command("go", "run", "./cmd/tya", "build", "--target", "wasm32-wasi", path, "-o", wasiOut)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wasi build failed: %v\n%s", err, out)
	}
	if info, err := os.Stat(wasiOut); err != nil || info.Size() == 0 {
		t.Fatalf("missing wasi wasm: %v", err)
	}
	browserOut := filepath.Join(dir, "browser", "hello.wasm")
	cmd = exec.Command("go", "run", "./cmd/tya", "build", "--target", "wasm32-browser", path, "-o", browserOut)
	cmd.Dir = ".."
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("browser build failed: %v\n%s", err, out)
	}
	if info, err := os.Stat(browserOut); err != nil || info.Size() == 0 {
		t.Fatalf("missing browser wasm: %v", err)
	}
	if info, err := os.Stat(filepath.Join(dir, "browser", "hello.js")); err != nil || info.Size() == 0 {
		t.Fatalf("missing browser loader: %v", err)
	}
}

func TestCLIBuildBrowserWasmSupportsBasicRuntimeValues(t *testing.T) {
	skipShort(t, "WASM runtime integration is covered by the release gate")

	if _, err := exec.LookPath("zig"); err != nil {
		t.Skip("zig not installed")
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "basic.tya")
	source := strings.Join([]string{
		"class Box",
		"  value: nil",
		"",
		"  initialize : box_value ->",
		"    self.value = box_value",
		"",
		"items = [1, 2]",
		"data = { name: \"tya\" }",
		"box = Box(\"ok\")",
		"print(items.len())",
		"print(data[\"name\"])",
		"print(box.value)",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	wasm := filepath.Join(dir, "basic.wasm")
	cmd := exec.Command("go", "run", "./cmd/tya", "build", "--target", "wasm32-browser", path, "-o", wasm)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("browser build failed: %v\n%s", err, out)
	}
	script := fmt.Sprintf(`
const fs = require('fs');
(async () => {
  const { instance } = await WebAssembly.instantiate(fs.readFileSync(%q), {});
  instance.exports.tya_output_reset();
  instance.exports.main(0, 0);
  const ptr = instance.exports.tya_output_ptr();
  const len = instance.exports.tya_output_len();
  process.stdout.write(new TextDecoder().decode(new Uint8Array(instance.exports.memory.buffer, ptr, len)));
})();
`, wasm)
	cmd = exec.Command("node", "-e", script)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node wasm failed: %v\n%s", err, out)
	}
	if string(out) != "2\ntya\nok\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIBuildWasiSupportsArgsAndBasicFiles(t *testing.T) {
	skipShort(t, "WASI runtime integration is covered by the release gate")

	if _, err := exec.LookPath("zig"); err != nil {
		t.Skip("zig not installed")
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "wasi_file.tya")
	source := strings.Join([]string{
		"import file",
		"print(args().len())",
		"file.File().write(\"tya-wasi-check.txt\", \"ok\")",
		"print(file.File().read(\"tya-wasi-check.txt\"))",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	wasm := filepath.Join(dir, "wasi_file.wasm")
	cmd := exec.Command("go", "run", "./cmd/tya", "build", "--target", "wasm32-wasi", path, "-o", wasm)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wasi build failed: %v\n%s", err, out)
	}
	script := fmt.Sprintf(`
const { WASI } = require('wasi');
const fs = require('fs');
(async () => {
  const wasi = new WASI({ version: 'preview1', args: ['wasi_file.wasm', 'arg1'], preopens: {'.': %q} });
  const { instance } = await WebAssembly.instantiate(fs.readFileSync(%q), wasi.getImportObject());
  wasi.start(instance);
})();
`, dir, wasm)
	cmd = exec.Command("node", "-e", script)
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node wasi failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "2\nok\n") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIBuildRejectsBadWasmInputs(t *testing.T) {
	skipShort(t, "WASM target validation is covered by the release gate")

	dir := t.TempDir()
	path := filepath.Join(dir, "hello.tya")
	if err := os.WriteFile(path, []byte("print(\"bad\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "build", "--target", "plan9", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected unsupported target to fail")
	}
	if !strings.Contains(string(out), "unsupported build target: plan9") {
		t.Fatalf("unexpected output: %s", out)
	}
	fileImport := filepath.Join(dir, "file_import.tya")
	if err := os.WriteFile(fileImport, []byte("import file\nprint(\"x\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("go", "run", "./cmd/tya", "build", "--target", "wasm32-browser", fileImport, "-o", filepath.Join(dir, "bad.wasm"))
	cmd.Dir = ".."
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected browser file import to fail")
	}
	if !strings.Contains(string(out), "not supported for target wasm32-browser") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIDoctorWasm(t *testing.T) {
	skipShort(t, "WASM doctor integration is covered by the release gate")

	if _, err := exec.LookPath("zig"); err != nil {
		t.Skip("zig not installed")
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "doctor", "wasm")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("doctor wasm failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "wasm32-wasi") || !strings.Contains(string(out), "wasm32-browser") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIVersionCommand(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/tya", "version")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "0.71.2\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}
