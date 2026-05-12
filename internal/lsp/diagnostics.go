package lsp

import (
	"errors"
	"strings"

	"tya/internal/checker"
	"tya/internal/diag"
	"tya/internal/lexer"
	"tya/internal/parser"
)

// DiagnosticsFor lexes, parses, and checks src and returns the
// resulting LSP-shaped findings for `path`. Lex / parse failures
// surface as a single-line diagnostic at their reported location;
// successful parses fan out to checker.CheckAll plus the lint and
// orphan-comment passes.
func DiagnosticsFor(path, src string) []Diagnostic {
	out := []Diagnostic{}
	toks, lcomments, lerrs := lexer.LexWithComments(src)
	for _, e := range lerrs {
		out = append(out, lexErrorToLSP(e))
	}
	infos := toCommentInfos(lcomments)
	prog, pdiags, err := parser.ParseWithComments(toks, infos)
	for _, d := range pdiags {
		out = append(out, diagToLSP(d))
	}
	if err != nil {
		if len(pdiags) == 0 {
			out = append(out, parseErrorToLSP(err))
		}
		if prog == nil {
			return out
		}
	}
	for _, oc := range parser.OrphanComments(prog, infos) {
		out = append(out, orphanCommentToLSP(oc))
	}
	if diags, _ := checker.CheckAll(prog, nil, path, true); len(diags) > 0 {
		for _, d := range diags {
			out = append(out, diagToLSP(d))
		}
	}
	for _, u := range checker.CollectUnused(prog) {
		out = append(out, Diagnostic{
			Range:    rangeAt(u.Line, u.Col, len(u.Name)),
			Severity: DiagSeverityWarning,
			Code:     "TYAL0001",
			Source:   "tya",
			Message:  "unused local \"" + u.Name + "\"",
		})
	}
	for _, f := range checker.CollectLintFindings(prog) {
		out = append(out, Diagnostic{
			Range:    rangeAt(f.Line, f.Col, 1),
			Severity: DiagSeverityWarning,
			Code:     f.Code,
			Source:   "tya",
			Message:  f.Message,
		})
	}
	return out
}

func toCommentInfos(lcomments []lexer.Comment) []parser.CommentInfo {
	out := make([]parser.CommentInfo, len(lcomments))
	for i, c := range lcomments {
		out[i] = parser.CommentInfo{
			Line:       c.Line,
			Col:        c.Col,
			Indent:     c.Indent,
			Text:       c.Text,
			IsFullLine: c.IsFullLine,
		}
	}
	return out
}

func diagToLSP(d diag.Diagnostic) Diagnostic {
	sev := DiagSeverityError
	if d.Severity == diag.Warning {
		sev = DiagSeverityWarning
	}
	return Diagnostic{
		Range: Range{
			Start: Position{Line: maxZero(d.Primary.Start.Line - 1), Character: maxZero(d.Primary.Start.Col - 1)},
			End:   Position{Line: maxZero(d.Primary.End.Line - 1), Character: maxZero(d.Primary.End.Col - 1)},
		},
		Severity: sev,
		Code:     d.Code,
		Source:   "tya",
		Message:  joinMessage(d),
	}
}

func joinMessage(d diag.Diagnostic) string {
	parts := []string{}
	if d.Title != "" && d.Title != d.Message {
		parts = append(parts, d.Title)
	}
	if d.Message != "" {
		parts = append(parts, d.Message)
	}
	body := strings.Join(parts, ": ")
	if len(d.Hints) > 0 {
		body += "\n\nhint: " + strings.Join(d.Hints, " ")
	}
	return body
}

func lexErrorToLSP(e error) Diagnostic {
	var ld *lexer.Diagnostic
	if errors.As(e, &ld) {
		return diagToLSP(ld.Diag)
	}
	line, col := extractLineCol(e.Error())
	return Diagnostic{
		Range:    rangeAt(line, col, 1),
		Severity: DiagSeverityError,
		Source:   "tya",
		Message:  e.Error(),
	}
}

func parseErrorToLSP(err error) Diagnostic {
	var serr *checker.StrictError
	if errors.As(err, &serr) && len(serr.Diags) > 0 {
		return diagToLSP(serr.Diags[0])
	}
	line, col := extractLineCol(err.Error())
	return Diagnostic{
		Range:    rangeAt(line, col, 1),
		Severity: DiagSeverityError,
		Source:   "tya",
		Message:  err.Error(),
	}
}

func orphanCommentToLSP(c parser.CommentInfo) Diagnostic {
	return Diagnostic{
		Range:    rangeAt(c.Line, c.Col, 1),
		Severity: DiagSeverityWarning,
		Code:     "TYA-E0150",
		Source:   "tya",
		Message:  "comment at forbidden position; move above the next statement or to the file header",
	}
}

// rangeAt builds a 1-character-wide LSP range starting at the
// given 1-origin (line, col).
func rangeAt(line, col, length int) Range {
	if length < 1 {
		length = 1
	}
	return Range{
		Start: Position{Line: maxZero(line - 1), Character: maxZero(col - 1)},
		End:   Position{Line: maxZero(line - 1), Character: maxZero(col - 1 + length)},
	}
}

func maxZero(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// extractLineCol scans "<file>:<line>:<col>" prefixes embedded in
// many tya error strings and returns (line, col) when found.
func extractLineCol(s string) (int, int) {
	parts := strings.Split(s, ":")
	if len(parts) < 3 {
		return 1, 1
	}
	for i := 0; i+1 < len(parts); i++ {
		line, ok1 := atoi(strings.TrimSpace(parts[i]))
		col, ok2 := atoi(strings.TrimSpace(parts[i+1]))
		if ok1 && ok2 {
			return line, col
		}
	}
	return 1, 1
}

func atoi(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}
