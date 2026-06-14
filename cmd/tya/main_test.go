package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestPortableNativeCFlagsUsesAMD64Baseline(t *testing.T) {
	got := portableNativeCFlags("amd64")
	want := []string{"-march=x86-64"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	if got := portableNativeCFlags("arm64"); got != nil {
		t.Fatalf("got %v, want nil", got)
	}
}

func TestNativeCFlagsPlaceEnvOverridesLast(t *testing.T) {
	t.Setenv("TYA_CFLAGS", "-march=native -funroll-loops")

	got := nativeCFlagsForArch("build", "amd64")
	want := []string{"-O2", "-march=x86-64", "-march=native", "-funroll-loops"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

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
