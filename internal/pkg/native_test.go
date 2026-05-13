package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectNativeMissingPkgConfigBinary(t *testing.T) {
	dir := nativeProject(t, `pkg_config = ["missing-host"]`)
	t.Setenv("PATH", filepath.Join(dir, "empty-bin"))
	if err := os.Mkdir(filepath.Join(dir, "empty-bin"), 0755); err != nil {
		t.Fatal(err)
	}
	_, err := CollectNative(dir)
	if err == nil || !strings.Contains(err.Error(), "pkg-config was not found") {
		t.Fatalf("expected missing pkg-config diagnostic, got %v", err)
	}
}

func TestCollectNativeMissingPkgConfigPackage(t *testing.T) {
	dir := nativeProject(t, `pkg_config = ["missing-host"]`)
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "pkg-config"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	_, err := CollectNative(dir)
	if err == nil || !strings.Contains(err.Error(), "missing pkg-config dependency missing-host") {
		t.Fatalf("expected missing dependency diagnostic, got %v", err)
	}
}

func nativeProject(t *testing.T, nativeLine string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "native"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "native", "demo.c"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	src := `name = "demo"
version = "0.1.0"

[native]
sources = ["native/demo.c"]
` + nativeLine + `

[native.functions]
demo_value = { symbol = "tya_demo_value", arity = 0 }
`
	if err := os.WriteFile(filepath.Join(dir, "tya.toml"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}
