package lsp

import (
	"fmt"
	"strings"

	"tya/internal/checker"
)

// CodeActions returns the quickfix proposals that apply to the
// document at `params.TextDocument.URI`. v0.53 generates fixes
// from checker.LintAutofixHints (TYAL0001, TYAL0003).
//
// The set is filtered by `params.Context.Diagnostics` when the
// client supplies one — clients normally include the diagnostics
// that prompted the code-action request, and we hide fixes whose
// code is not currently visible.
func CodeActions(doc *Document, params CodeActionParams) []CodeAction {
	if doc == nil {
		return nil
	}
	prog := parseOrNil(doc.Text)
	if prog == nil {
		return nil
	}
	wantCode := map[string]bool{}
	for _, d := range params.Context.Diagnostics {
		if d.Code != "" {
			wantCode[d.Code] = true
		}
	}
	out := []CodeAction{}
	for _, hint := range checker.LintAutofixHints(prog) {
		if len(wantCode) > 0 && !wantCode[hint.Code] {
			continue
		}
		if !rangesOverlap(params.Range, rangeAt(hint.Line, 1, 1)) {
			continue
		}
		switch hint.Kind {
		case "delete-line":
			out = append(out, CodeAction{
				Title:       fmt.Sprintf("Remove unused %q", hint.Name),
				Kind:        CodeActionKindQuickFix,
				Diagnostics: relatedDiagnostics(params.Context.Diagnostics, hint.Code),
				Edit:        deleteLineEdit(doc, hint.Line),
				IsPreferred: true,
			})
		case "unwrap-if":
			edit, ok := unwrapIfEdit(doc, hint)
			if !ok {
				continue
			}
			out = append(out, CodeAction{
				Title:       "Unwrap `if true/false` block",
				Kind:        CodeActionKindQuickFix,
				Diagnostics: relatedDiagnostics(params.Context.Diagnostics, hint.Code),
				Edit:        edit,
				IsPreferred: true,
			})
		}
	}
	return out
}

func relatedDiagnostics(all []Diagnostic, code string) []Diagnostic {
	out := []Diagnostic{}
	for _, d := range all {
		if d.Code == code {
			out = append(out, d)
		}
	}
	return out
}

func rangesOverlap(a, b Range) bool {
	if a.Start.Line > b.End.Line {
		return false
	}
	if b.Start.Line > a.End.Line {
		return false
	}
	return true
}

func deleteLineEdit(doc *Document, line int) WorkspaceEdit {
	if line < 1 {
		line = 1
	}
	return WorkspaceEdit{Changes: map[string][]TextEdit{
		doc.URI: {{
			Range: Range{
				Start: Position{Line: line - 1, Character: 0},
				End:   Position{Line: line, Character: 0},
			},
			NewText: "",
		}},
	}}
}

func unwrapIfEdit(doc *Document, hint checker.LintAutofixHint) (WorkspaceEdit, bool) {
	lines := strings.Split(doc.Text, "\n")
	if hint.Line < 1 || hint.Line > len(lines) {
		return WorkspaceEdit{}, false
	}
	endLine := hint.EndLine
	if endLine < hint.Line {
		endLine = hint.Line
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	// Strip the `if true/false` header line and de-indent the body
	// by the number of leading spaces on the header.
	header := lines[hint.Line-1]
	headerIndent := len(header) - len(strings.TrimLeft(header, " "))
	body := []string{}
	for i := hint.Line; i < endLine && i < len(lines); i++ {
		line := lines[i]
		trimmed := line
		stripped := 0
		for stripped < 2 && stripped < len(trimmed) && trimmed[stripped] == ' ' {
			stripped++
		}
		_ = headerIndent
		body = append(body, trimmed[stripped:])
	}
	replacement := strings.Join(body, "\n") + "\n"
	return WorkspaceEdit{Changes: map[string][]TextEdit{
		doc.URI: {{
			Range: Range{
				Start: Position{Line: hint.Line - 1, Character: 0},
				End:   Position{Line: endLine, Character: 0},
			},
			NewText: replacement,
		}},
	}}, true
}
