package tests

import (
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
	if err := os.WriteFile(path, []byte("name = \"Tya\"\nprint name\n"), 0644); err != nil {
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

func TestCLIAllowsCombinedOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "used.tya")
	if err := os.WriteFile(path, []byte("name = \"Tya\"\nprint name\n"), 0644); err != nil {
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
	if err := os.WriteFile(path, []byte("name = \"Tya\"\nprint name\n"), 0644); err != nil {
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
	if err := os.WriteFile(path, []byte("print \"Hello from run\"\n"), 0644); err != nil {
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

func TestCLIDirectFileDoesNotUseInterpreterPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hello.tya")
	if err := os.WriteFile(path, []byte("print \"direct\"\n"), 0644); err != nil {
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
	if err := os.WriteFile(path, []byte("items = args()\nprint len(items)\nprint items[0]\n"), 0644); err != nil {
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
	if err := os.WriteFile(module, []byte("module greeting\n  hello = name -> \"Hello, {name}\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "main.tya")
	if err := os.WriteFile(path, []byte("import greeting\nprint greeting.hello(\"komagata\")\n"), 0644); err != nil {
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

func TestCLIBuildWritesExecutable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.tya")
	if err := os.WriteFile(path, []byte("print \"Hello from build\"\n"), 0644); err != nil {
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
}

func TestCLIBuildUsesDefaultOutputPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "default_out.tya")
	if err := os.WriteFile(path, []byte("print \"default output\"\n"), 0644); err != nil {
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

func TestCLIVersionCommand(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/tya", "version")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "0.35.0\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}
