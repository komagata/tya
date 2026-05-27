package formatter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tya/internal/lexer"
	"tya/internal/parser"
)

// TestUnparseHandlesCorpus walks every .tya source in the
// repository's lib/, examples/, and selfhost/ trees and asserts
// that Unparse either:
//
//   - returns a string that re-parses successfully, or
//   - returns an "unsupported" error (graceful degradation, the
//     CLI falls back to the text pass for these).
//
// The goal is to catch regressions where Unparse emits unparseable
// output (a panic on the user). Files where Unparse returns an
// unsupported error are recorded and reported; that surface is
// expected to shrink over time as Unparse coverage grows.
func TestUnparseHandlesCorpus(t *testing.T) {
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	roots := []string{
		filepath.Join(repo, "lib"),
		filepath.Join(repo, "examples"),
		filepath.Join(repo, "selfhost"),
	}
	var unsupported []string
	var checked int
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".tya") {
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			toks, errs := lexer.Lex(string(src))
			if len(errs) > 0 {
				return nil
			}
			prog, _, err := parser.Parse(toks)
			if err != nil {
				return nil
			}
			checked++
			out, err := Unparse(prog)
			if err != nil {
				rel, _ := filepath.Rel(repo, path)
				unsupported = append(unsupported, rel)
				return nil
			}
			toks2, errs2 := lexer.Lex(out)
			if len(errs2) > 0 {
				rel, _ := filepath.Rel(repo, path)
				t.Errorf("Unparse output for %s does not re-lex: %v", rel, errs2[0])
				return nil
			}
			if _, _, err := parser.Parse(toks2); err != nil {
				rel, _ := filepath.Rel(repo, path)
				t.Errorf("Unparse output for %s does not re-parse: %v", rel, err)
			}
			return nil
		})
	}
	t.Logf("Unparse: handled %d files, %d unsupported (fall back to text pass)", checked-len(unsupported), len(unsupported))
	if len(unsupported) > 0 {
		t.Logf("unsupported files (will use text-pass fallback):")
		for _, p := range unsupported {
			t.Logf("  %s", p)
		}
	}
}
