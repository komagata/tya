package lsp

import (
	"strings"
	"testing"
)

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

func TestDiagnosticsForStrictCheckerErrors(t *testing.T) {
	src := "outer = 1\nhandler = value ->\n  outer = value\n  value\n"
	diags := DiagnosticsFor("strict.tya", src)
	for _, d := range diags {
		if d.Code == "TYA-E0307" {
			if d.Severity != DiagSeverityError {
				t.Fatalf("TYA-E0307 severity = %d, want error", d.Severity)
			}
			if !strings.Contains(d.Message, "Assignment to outer binding") {
				t.Fatalf("message = %q", d.Message)
			}
			return
		}
	}
	t.Fatalf("missing TYA-E0307 in %#v", diags)
}
