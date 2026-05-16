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

func TestExtractReportFollowsReexportsAndReportsDiagnostics(t *testing.T) {
	dir := t.TempDir()
	api := filepath.Join(dir, "api.tya")
	helpers := filepath.Join(dir, "helpers.tya")
	if err := os.WriteFile(api, []byte("import helpers\n\n# dangling\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(helpers, []byte("# Helper docs\nhelper = -> 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := ExtractReport([]string{api})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Items) != 1 {
		t.Fatalf("expected one re-exported item, got %#v", report.Items)
	}
	if report.Items[0].Name != "helper" || report.Items[0].ReexportedFrom != api {
		t.Fatalf("unexpected re-exported item: %#v", report.Items[0])
	}
	if len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != "TYADOC0001" {
		t.Fatalf("expected orphan diagnostic, got %#v", report.Diagnostics)
	}
}

func TestExtractReportDuplicateAndMarkdownDiagnostics(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.tya")
	b := filepath.Join(dir, "b.tya")
	if err := os.WriteFile(a, []byte("# A docs\nsame = -> 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("# Broken\n# ```\nsame = -> 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := ExtractReport([]string{a, b})
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]bool{}
	for _, diag := range report.Diagnostics {
		codes[diag.Code] = true
	}
	if !codes["TYADOC0002"] || !codes["TYADOC0003"] {
		t.Fatalf("expected duplicate and markdown diagnostics, got %#v", report.Diagnostics)
	}
}

func TestExtractReportImportCycleDiagnostic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.tya")
	b := filepath.Join(dir, "b.tya")
	if err := os.WriteFile(a, []byte("import b\n\n# A docs\na = -> 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("import a\n\n# B docs\nb = -> 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := ExtractReport([]string{a})
	if err != nil {
		t.Fatal(err)
	}
	for _, diag := range report.Diagnostics {
		if diag.Code == "TYADOC0004" {
			return
		}
	}
	t.Fatalf("expected import cycle diagnostic, got %#v", report.Diagnostics)
}
