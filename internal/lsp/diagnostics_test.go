package lsp

import (
	"strings"
	"testing"

	"tya/internal/checker"
	"tya/internal/lexer"
	"tya/internal/parser"
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
			if !strings.Contains(d.Message, "Cannot reassign `outer`") {
				t.Fatalf("message = %q", d.Message)
			}
			return
		}
	}
	t.Fatalf("missing TYA-E0307 in %#v", diags)
}

func TestLspDiagnosticsMatchTyaCheck(t *testing.T) {
	path := "strict.tya"
	src := "outer = 1\nhandler = value ->\n  outer = value\n  value\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	checkDiags, err := checker.CheckAll(prog, nil, path, true)
	if err != nil {
		t.Fatal(err)
	}
	lspDiags := DiagnosticsFor(path, src)
	for _, checkDiag := range checkDiags {
		found := false
		for _, lspDiag := range lspDiags {
			if lspDiag.Code == checkDiag.Code && lspDiag.Message == checkDiag.Message {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing check diagnostic %s %q in %#v", checkDiag.Code, checkDiag.Message, lspDiags)
		}
	}
}

func TestLspAcceptsAcceptedButUnformattedSyntax(t *testing.T) {
	src := "add = (a, b,) -> a + b\nname = 'Tya'\nprint(add(1, 2,))\n"
	diags := DiagnosticsFor("accepted.tya", src)
	for _, d := range diags {
		if d.Severity == DiagSeverityError {
			t.Fatalf("unexpected syntax/check error: %#v", d)
		}
	}
}

func TestDiagnosticsAllowRecordDocComments(t *testing.T) {
	src := "# Option stores one parsed option token.\nrecord Option\n  # Option.kind stores option kind.\n  kind\n  # Option.name stores option name.\n  name\n"
	diags := DiagnosticsFor("option_parser.tya", src)
	for _, d := range diags {
		if d.Severity == DiagSeverityError {
			t.Fatalf("unexpected diagnostic: %#v", d)
		}
	}
}
