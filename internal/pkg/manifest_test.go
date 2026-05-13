package pkg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadManifest(t *testing.T) {
	dir := t.TempDir()
	src := `name = "myapp"
version = "0.1.0"
description = "An app"
authors = ["Alice", "Bob"]

[dependencies]
foo = "^1.0.0"
bar = { path = "../bar" }
baz = { git = "https://example.com/baz", tag = "v2.0.0" }

[dev-dependencies]
mock = "^0.3.0"
`
	path := filepath.Join(dir, "tya.toml")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	m, err := ReadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "myapp" {
		t.Errorf("name: got %q", m.Name)
	}
	if m.Version.String() != "0.1.0" {
		t.Errorf("version: got %s", m.Version)
	}
	if m.Description != "An app" {
		t.Errorf("description: got %q", m.Description)
	}
	if len(m.Authors) != 2 {
		t.Errorf("authors: got %v", m.Authors)
	}
	if len(m.Deps) != 3 {
		t.Errorf("deps: got %d", len(m.Deps))
	}
	if d, ok := m.Deps["foo"]; !ok || d.Source != "version" || d.Constraint.Raw != "^1.0.0" {
		t.Errorf("foo: %+v", d)
	}
	if d, ok := m.Deps["bar"]; !ok || d.Source != "path" || d.PathRef != "../bar" {
		t.Errorf("bar: %+v", d)
	}
	if d, ok := m.Deps["baz"]; !ok || d.Source != "git" || d.Tag != "v2.0.0" {
		t.Errorf("baz: %+v", d)
	}
	if _, ok := m.DevDeps["mock"]; !ok {
		t.Errorf("mock missing")
	}
}

func TestReadManifestTasks(t *testing.T) {
	dir := t.TempDir()
	src := `name = "tasks-app"
version = "0.1.0"

[tasks]
ci = "tya format && tya test"
release = ["tya build", "git tag v1.0.0", "git push --tags"]
greet = "echo hello"
`
	path := filepath.Join(dir, "tya.toml")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	m, err := ReadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Tasks) != 3 {
		t.Fatalf("tasks: got %d, want 3", len(m.Tasks))
	}
	if got := m.TaskOrder; len(got) != 3 || got[0] != "ci" || got[1] != "release" || got[2] != "greet" {
		t.Errorf("task order: got %v", got)
	}
	if got, ok := m.Tasks["ci"]; !ok || got.Kind != "string" || got.String != "tya format && tya test" {
		t.Errorf("ci: %+v", got)
	}
	if got, ok := m.Tasks["release"]; !ok || got.Kind != "array" || len(got.Array) != 3 || got.Array[1] != "git tag v1.0.0" {
		t.Errorf("release: %+v", got)
	}
	if got, ok := m.Tasks["greet"]; !ok || got.Kind != "string" || got.String != "echo hello" {
		t.Errorf("greet: %+v", got)
	}
}

func TestReadManifestNative(t *testing.T) {
	dir := t.TempDir()
	src := `name = "native-demo"
version = "0.1.0"

[native]
sources = ["native/demo.c"]
headers = ["include/demo.h"]
include_dirs = ["include"]
pkg_config = ["demo"]
cflags = ["-DDEMO=1"]
ldflags = ["-ldemo"]

[native.functions]
demo_init = { symbol = "tya_demo_init", arity = 0 }
demo_poll = { symbol = "tya_demo_poll", arity = 1 }
`
	path := filepath.Join(dir, "tya.toml")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	m, err := ReadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Native.Sources; len(got) != 1 || got[0] != "native/demo.c" {
		t.Errorf("sources: %+v", got)
	}
	if got := m.Native.IncludeDirs; len(got) != 1 || got[0] != "include" {
		t.Errorf("include dirs: %+v", got)
	}
	if got := m.Native.PkgConfig; len(got) != 1 || got[0] != "demo" {
		t.Errorf("pkg_config: %+v", got)
	}
	fn := m.Native.Functions["demo_poll"]
	if fn.Symbol != "tya_demo_poll" || fn.Arity != 1 {
		t.Errorf("demo_poll: %+v", fn)
	}
	if got := m.Native.FuncOrder; len(got) != 2 || got[0] != "demo_init" || got[1] != "demo_poll" {
		t.Errorf("function order: %+v", got)
	}
}

func TestReadManifestTasksInvalidType(t *testing.T) {
	dir := t.TempDir()
	src := `name = "x"
version = "0.1.0"

[tasks]
broken = 42
`
	path := filepath.Join(dir, "tya.toml")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadManifest(path); err == nil {
		t.Fatal("expected error for non-string/array task, got nil")
	}
}

func TestReadManifestTasksParallelTable(t *testing.T) {
	dir := t.TempDir()
	src := `name = "x"
version = "0.1.0"

[tasks.watch]
cmds = ["tya format --watch", "tya lsp"]
parallel = true
`
	path := filepath.Join(dir, "tya.toml")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	m, err := ReadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := m.Tasks["watch"]
	if !ok {
		t.Fatalf("watch task missing: %+v", m.Tasks)
	}
	if got.Kind != "parallel" {
		t.Errorf("kind: got %q, want parallel", got.Kind)
	}
	if len(got.Array) != 2 || got.Array[0] != "tya format --watch" || got.Array[1] != "tya lsp" {
		t.Errorf("cmds: %+v", got.Array)
	}
}

func TestReadManifestTasksTableWithoutParallel(t *testing.T) {
	dir := t.TempDir()
	src := `name = "x"
version = "0.1.0"

[tasks.build]
cmds = ["echo step1", "echo step2"]
`
	path := filepath.Join(dir, "tya.toml")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	m, err := ReadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	got := m.Tasks["build"]
	if got.Kind != "array" {
		t.Errorf("kind: got %q, want array", got.Kind)
	}
	if len(got.Array) != 2 {
		t.Errorf("cmds: %+v", got.Array)
	}
}

func TestReadManifestTasksArrayWithNonString(t *testing.T) {
	dir := t.TempDir()
	src := `name = "x"
version = "0.1.0"

[tasks]
mixed = ["ok", 42]
`
	path := filepath.Join(dir, "tya.toml")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadManifest(path); err == nil {
		t.Fatal("expected error for array with non-string element, got nil")
	}
}

func TestWriteManifestTasksRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tya.toml")
	m := &Manifest{
		Path:    path,
		Name:    "x",
		Version: Version{Major: 1, Minor: 0, Patch: 0, Raw: "1.0.0"},
		Tasks: map[string]Task{
			"ci":      {Name: "ci", Kind: "string", String: "tya test"},
			"release": {Name: "release", Kind: "array", Array: []string{"tya build", "git tag v1.0.0"}},
		},
		TaskOrder: []string{"ci", "release"},
	}
	if err := WriteManifest(m); err != nil {
		t.Fatal(err)
	}
	got, err := ReadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tasks["ci"].String != "tya test" {
		t.Errorf("ci: %+v", got.Tasks["ci"])
	}
	rel := got.Tasks["release"]
	if rel.Kind != "array" || len(rel.Array) != 2 || rel.Array[0] != "tya build" {
		t.Errorf("release: %+v", rel)
	}
	if order := got.TaskOrder; len(order) != 2 || order[0] != "ci" || order[1] != "release" {
		t.Errorf("task order: %v", order)
	}
}

func TestWriteManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tya.toml")
	m := &Manifest{
		Path:    path,
		Name:    "x",
		Version: Version{Major: 1, Minor: 0, Patch: 0, Raw: "1.0.0"},
		Deps: map[string]Dependency{
			"foo": {Name: "foo", Source: "path", PathRef: "../foo"},
		},
		DepOrder: []string{"foo"},
	}
	if err := WriteManifest(m); err != nil {
		t.Fatal(err)
	}
	got, err := ReadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "x" || got.Deps["foo"].PathRef != "../foo" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}
