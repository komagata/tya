package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveZigToolchainUsesTYAZig(t *testing.T) {
	dir := t.TempDir()
	zig := filepath.Join(dir, zigExecutableName())
	script := "#!/bin/sh\nprintf '0.16.0\\n'\n"
	if runtime.GOOS == "windows" {
		t.Skip("shell fake is POSIX-only")
	}
	if err := os.WriteFile(zig, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TYA_ZIG", zig)
	t.Setenv("TYA_ZIG_DIR", "")
	t.Setenv("PATH", "")

	got, err := resolveZigToolchain()
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != zig {
		t.Fatalf("path = %q, want %q", got.Path, zig)
	}
	if got.Source != "TYA_ZIG" {
		t.Fatalf("source = %q, want TYA_ZIG", got.Source)
	}
	if got.Version != managedZigVersion {
		t.Fatalf("version = %q, want %q", got.Version, managedZigVersion)
	}
}

func TestResolveZigToolchainMissingReportsRepair(t *testing.T) {
	t.Setenv("TYA_ZIG", "")
	t.Setenv("TYA_ZIG_DIR", "")
	t.Setenv("PATH", "")

	_, err := resolveZigToolchain()
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{
		"reinstall or repair Tya",
		"https://tya-lang.org/install.sh",
		"https://tya-lang.org/install.ps1",
		"TYA_ZIG=/path/to/zig",
		"TYA_ZIG_DIR=/path/to/zig-dir",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want substring %q", err.Error(), want)
		}
	}
}

func TestHasCompileFlagAcceptsSplitAndCompactForms(t *testing.T) {
	flags := []string{"-I", "/opt/homebrew/opt/openssl@3/include", "-L/usr/local/lib"}

	if !hasCompileFlag(flags, "-I/opt/homebrew/opt/openssl@3/include") {
		t.Fatal("expected split -I flag to match compact form")
	}
	if !hasCompileFlag(flags, "-L/usr/local/lib") {
		t.Fatal("expected compact -L flag to match compact form")
	}
	if hasCompileFlag(flags, "-L/opt/homebrew/opt/openssl@3/lib") {
		t.Fatal("unexpected missing -L flag match")
	}
}
