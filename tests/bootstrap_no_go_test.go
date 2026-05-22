package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestBootstrapNoGoRequiresBootstrapBinary(t *testing.T) {
	out, err := runBootstrapNoGo(t, nil)
	if err == nil {
		t.Fatalf("expected failure without TYA_BOOTSTRAP_TYA")
	}
	if !strings.Contains(out, "TYA_BOOTSTRAP_TYA") {
		t.Fatalf("stderr = %q, want TYA_BOOTSTRAP_TYA", out)
	}
}

func TestBootstrapNoGoRejectsNonExecutableBootstrapBinary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tya")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runBootstrapNoGo(t, []string{"TYA_BOOTSTRAP_TYA=" + path})
	if err == nil {
		t.Fatalf("expected failure for non-executable bootstrap")
	}
	if !strings.Contains(out, "not an executable file") || !strings.Contains(out, path) {
		t.Fatalf("stderr = %q, want unusable bootstrap path", out)
	}
}

func TestBootstrapNoGoInstallsGoShim(t *testing.T) {
	bootstrap := writeExecutable(t, "fake-tya", `#!/bin/sh
go version
`)
	out, err := runBootstrapNoGo(t, []string{"TYA_BOOTSTRAP_TYA=" + bootstrap})
	if err == nil {
		t.Fatalf("expected no-Go shim failure")
	}
	if !strings.Contains(out, "no-Go violation") {
		t.Fatalf("stderr = %q, want no-Go violation", out)
	}
}

func TestBootstrapNoGoKeepsTempOnFailureWhenRequested(t *testing.T) {
	bootstrap := writeExecutable(t, "fake-tya", `#!/bin/sh
echo "fake bootstrap failure" >&2
exit 43
`)
	out, err := runBootstrapNoGo(t, []string{
		"TYA_BOOTSTRAP_TYA=" + bootstrap,
		"TYA_KEEP_BOOTSTRAP_TMP=1",
	})
	if err == nil {
		t.Fatalf("expected bootstrap failure")
	}
	dir := bootstrapWorkDir(t, out)
	if _, statErr := os.Stat(dir); statErr != nil {
		t.Fatalf("retained work dir %q: %v\nstderr:\n%s", dir, statErr, out)
	}
}

func TestBootstrapNoGoRemovesTempOnFailureByDefault(t *testing.T) {
	bootstrap := writeExecutable(t, "fake-tya", `#!/bin/sh
echo "fake bootstrap failure" >&2
exit 43
`)
	out, err := runBootstrapNoGo(t, []string{"TYA_BOOTSTRAP_TYA=" + bootstrap})
	if err == nil {
		t.Fatalf("expected bootstrap failure")
	}
	dir := bootstrapWorkDir(t, out)
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Fatalf("work dir %q should be removed, stat err = %v\nstderr:\n%s", dir, statErr, out)
	}
}

func TestBootstrapNoGoSelfhostV02FixedPoint(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the v02 self-host fixed-point proof")
	}
	if os.Getenv("TYA_RUN_LONG_SELFHOST") != "1" {
		t.Skip("set TYA_RUN_LONG_SELFHOST=1 to run the v02 no-Go fixed-point proof")
	}
	bin := filepath.Join(t.TempDir(), "tya")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/tya")
	cmd.Dir = ".."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/tya: %v\n%s", err, out)
	}

	out, err := runBootstrapNoGo(t, []string{"TYA_BOOTSTRAP_TYA=" + bin})
	if err != nil {
		t.Fatalf("bootstrap_no_go failed: %v\n%s", err, out)
	}
	for _, want := range []string{"stage-2 emit", "stage-3 emit", "fixed-point compare passed"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stderr = %q, want %q", out, want)
		}
	}
	dir := bootstrapWorkDir(t, out)
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Fatalf("successful work dir %q should be removed, stat err = %v", dir, statErr)
	}
}

func runBootstrapNoGo(t *testing.T, env []string) (string, error) {
	t.Helper()
	cmd := exec.Command("sh", "scripts/bootstrap_no_go.sh")
	cmd.Dir = ".."
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func writeExecutable(t *testing.T, name string, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func bootstrapWorkDir(t *testing.T, output string) string {
	t.Helper()
	re := regexp.MustCompile(`work directory: ([^\s]+)`)
	match := re.FindStringSubmatch(output)
	if match == nil {
		t.Fatalf("missing work directory in output:\n%s", output)
	}
	return match[1]
}
