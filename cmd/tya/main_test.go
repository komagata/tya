package main

import (
	"path/filepath"
	"testing"
)

func TestFirstStdlibDirUsesLibEnv(t *testing.T) {
	libDir := filepath.Join(t.TempDir(), "lib")
	t.Setenv("TYA_LIB_DIR", libDir)
	t.Setenv("TYA_STDLIB_DIR", filepath.Join(t.TempDir(), "stdlib"))

	if got := firstStdlibDir(); got != libDir {
		t.Fatalf("got %q, want %q", got, libDir)
	}
}

func TestFirstStdlibDirFallsBackToDeprecatedStdlibEnv(t *testing.T) {
	stdlibDir := filepath.Join(t.TempDir(), "stdlib")
	t.Setenv("TYA_LIB_DIR", "")
	t.Setenv("TYA_STDLIB_DIR", stdlibDir)

	if got := firstStdlibDir(); got != stdlibDir {
		t.Fatalf("got %q, want %q", got, stdlibDir)
	}
}
