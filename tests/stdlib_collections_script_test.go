package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestStdlibCollectionsScript(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(root, "tests", "stdlib_collections_spec.tya"))
	if err != nil {
		t.Fatal(err)
	}
	tmp := filepath.Join(t.TempDir(), "stdlib_collections_test.tya")
	if err := os.WriteFile(tmp, src, 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "test", tmp)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stdlib collections script failed: %v\n%s", err, out)
	}
}
