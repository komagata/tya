package doc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractFileIncludesUnderscoreTopLevelFunction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "helpers.tya")
	write := "# Helper docs\n_helper = value -> value\n"
	if err := os.WriteFile(path, []byte(write), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := ExtractFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "_helper" {
		t.Fatalf("expected _helper doc item, got %#v", items)
	}
}
