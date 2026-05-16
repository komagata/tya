package lsp

import "testing"

func TestDiagnosticsForLintWarningsUseStableCodes(t *testing.T) {
	src := "outer = 1\nhandler = used, unused ->\n  outer = 2\n  print(used)\n"
	diags := DiagnosticsFor("sample.tya", src)
	want := map[string]string{
		"TYAL0007": "unused function parameter \"unused\"",
		"TYAL0008": "shadowed binding \"outer\" (previous binding at 1:1)",
	}
	for code, message := range want {
		found := false
		for _, d := range diags {
			if d.Code == code {
				found = true
				if d.Severity != DiagSeverityWarning {
					t.Fatalf("%s severity = %d, want warning", code, d.Severity)
				}
				if d.Message != message {
					t.Fatalf("%s message = %q, want %q", code, d.Message, message)
				}
			}
		}
		if !found {
			t.Fatalf("missing %s in %#v", code, diags)
		}
	}
}
