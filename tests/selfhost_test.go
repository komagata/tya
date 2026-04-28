package tests

import (
	"os"
	"os/exec"
	"testing"
)

func TestSelfhostPrototypePipeline(t *testing.T) {
	out := run(t, "sh", "scripts/selfhost.sh")
	if string(out) != "ok\nTya\n2\nIndented\n" {
		t.Fatalf("got %q", out)
	}
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
