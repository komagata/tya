package tests

import (
	"os"
	"os/exec"
	"testing"
)

func TestSelfhostPrototypePipeline(t *testing.T) {
	out := run(t, "sh", "scripts/selfhost.sh")
	if string(out) != "ok\nTya\nTya\n3\ntrue\nfalse\ntrue\ntrue\nIndented\nCompared\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostCheckerRejectsUndefinedConditionNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:IF:IDENT:missingIf\n2:WHILE:IDENT:missingWhile\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingIf\n2: undefined variable: missingWhile\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedAssignmentNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:ASSIGN:alias:IDENT:missing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missing\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedNotNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:ASSIGN:negated:BOOL_NOT:missing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missing\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedPushNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:PUSH:missingTarget:IDENT:missingValue\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingTarget\n1: undefined variable: missingValue\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedReturnNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:RETURN:IDENT:missing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missing\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedCallNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:ASSIGN:result:CALL1:missingFunc:missingArg\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingFunc\n1: undefined variable: missingArg\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedForCollections(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:FOR:item:missingItems\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingItems\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
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
