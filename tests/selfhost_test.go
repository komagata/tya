package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSelfhostPrototypePipeline(t *testing.T) {
	root := ".."
	dir := t.TempDir()
	tokens := filepath.Join(dir, "tokens.txt")
	nodes := filepath.Join(dir, "nodes.txt")
	cfile := filepath.Join(dir, "main.c")
	bin := filepath.Join(dir, "main")

	runToFile(t, tokens, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "examples/selfhost_input.tya")
	runToFile(t, nodes, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokens)
	run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodes)
	runToFile(t, cfile, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodes)
	run(t, "gcc", cfile, "-o", bin)
	out := run(t, bin)
	if string(out) != "Tya\n" {
		t.Fatalf("got %q", out)
	}
	_ = root
}

func runToFile(t *testing.T, path string, name string, args ...string) {
	t.Helper()
	out := run(t, name, args...)
	if err := os.WriteFile(path, out, 0644); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, name string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
	return out
}
