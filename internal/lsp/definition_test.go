package lsp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefinitionAtBarePackageClassImport(t *testing.T) {
	dir := t.TempDir()
	writeLSPFile(t, filepath.Join(dir, "tya.toml"), "name = \"demo\"\nversion = \"0.1.0\"\nlicense = \"MIT\"\n")
	writeLSPFile(t, filepath.Join(dir, "src", "net", "http", "Request.tya"), "class Request\n  initialize: ->\n    self.path = \"\"\n")
	mainPath := filepath.Join(dir, "src", "main.tya")
	mainSrc := "import net/http\n\nrequest = Request()\n"
	writeLSPFile(t, mainPath, mainSrc)
	mainURI, _ := PathToURI(mainPath)
	wantURI, _ := PathToURI(filepath.Join(dir, "src", "net", "http", "Request.tya"))
	ws := NewWorkspace()
	ws.Root = dir

	locs, err := DefinitionAt(DefinitionContext{
		Doc:       &Document{URI: mainURI, Path: mainPath, Text: mainSrc},
		Workspace: ws,
	}, 2, 11)
	if err != nil {
		t.Fatal(err)
	}
	if len(locs) != 1 {
		t.Fatalf("expected one definition, got %d", len(locs))
	}
	if locs[0].URI != wantURI {
		t.Fatalf("URI = %q, want %q", locs[0].URI, wantURI)
	}
}

func TestDefinitionAtAliasedPackageClassImport(t *testing.T) {
	dir := t.TempDir()
	writeLSPFile(t, filepath.Join(dir, "tya.toml"), "name = \"demo\"\nversion = \"0.1.0\"\nlicense = \"MIT\"\n")
	writeLSPFile(t, filepath.Join(dir, "src", "net", "http", "Request.tya"), "class Request\n  initialize: ->\n    self.path = \"\"\n")
	mainPath := filepath.Join(dir, "src", "main.tya")
	mainSrc := "import net/http as http\n\nrequest = http.Request()\n"
	writeLSPFile(t, mainPath, mainSrc)
	mainURI, _ := PathToURI(mainPath)
	wantURI, _ := PathToURI(filepath.Join(dir, "src", "net", "http", "Request.tya"))
	ws := NewWorkspace()
	ws.Root = dir

	locs, err := DefinitionAt(DefinitionContext{
		Doc:       &Document{URI: mainURI, Path: mainPath, Text: mainSrc},
		Workspace: ws,
	}, 2, 15)
	if err != nil {
		t.Fatal(err)
	}
	if len(locs) != 1 {
		t.Fatalf("expected one definition, got %d", len(locs))
	}
	if locs[0].URI != wantURI {
		t.Fatalf("URI = %q, want %q", locs[0].URI, wantURI)
	}
}

func writeLSPFile(t *testing.T, path string, src string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}
