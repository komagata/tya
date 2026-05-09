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
