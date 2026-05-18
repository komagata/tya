package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpecDocumentsSingleErrorModel(t *testing.T) {
	spec := readRepoFile(t, "docs", "SPEC.md")
	for _, forbidden := range []string{
		"try` may be used as an expression",
		"try can also be an expression",
		"return a documented `value, err`",
		"return nil, error(",
	} {
		if strings.Contains(spec, forbidden) {
			t.Fatalf("SPEC.md still documents forbidden error model text %q", forbidden)
		}
	}
	for _, required := range []string{
		"`try` is a statement only",
		"`catch err` is the only catch syntax",
		"raise structured error values",
		"error(message, options = {})",
	} {
		if !strings.Contains(spec, required) {
			t.Fatalf("SPEC.md missing %q", required)
		}
	}
}

func TestStdlibDocsUseStructuredRaisedErrors(t *testing.T) {
	spec := readRepoFile(t, "docs", "SPEC.md")
	for _, required := range []string{
		"kind",
		"code",
		"data",
		"cause",
		"raise structured error values",
	} {
		if !strings.Contains(spec, required) {
			t.Fatalf("structured error docs missing %q", required)
		}
	}
}

func readRepoFile(t *testing.T, elems ...string) string {
	t.Helper()
	parts := append([]string{".."}, elems...)
	data, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
