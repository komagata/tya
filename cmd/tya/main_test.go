package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestPortableNativeCFlagsUsesAMD64Baseline(t *testing.T) {
	// Pin the backend so the expected flag spelling doesn't depend on
	// whether Zig is resolvable in the environment running this test:
	// plain cc/gcc wants "-march=x86-64", Zig's clang frontend wants
	// "-mcpu=x86_64" (see portableNativeCFlags).
	t.Setenv("CC", "cc")

	got := portableNativeCFlags("amd64")
	want := []string{"-march=x86-64"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	if got := portableNativeCFlags("arm64"); got != nil {
		t.Fatalf("got %v, want nil", got)
	}
}

func TestPortableNativeCFlagsUsesZigCPUSpelling(t *testing.T) {
	t.Setenv("CC", "")
	t.Setenv("TYA_ZIG", "/nonexistent/zig")

	got := portableNativeCFlags("amd64")
	want := []string{"-mcpu=x86_64"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNativeCFlagsPlaceEnvOverridesLast(t *testing.T) {
	t.Setenv("CC", "cc")
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
