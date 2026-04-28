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
