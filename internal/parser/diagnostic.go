package parser

import (
	"strings"

	"tya/internal/diag"
)

// ParserError is the error returned by Parse / ParseWithComments
// when one or more parser-stage diagnostics fire. It implements
// `error` so callers can keep using a single `error` return value;
// `errors.As` lifts the structured list back out.
type ParserError struct {
	Diags []diag.Diagnostic
}

// Error returns a backwards-compatible flat string matching the
// pre-v0.54 `line:col: msg near 'tok'` format that existing parser
// tests grep against. The structured payload is retrieved via
// `errors.As(err, &perr)` and consumed by CLI / LSP.
func (e *ParserError) Error() string {
	if len(e.Diags) == 0 {
		return "parser error"
	}
	parts := make([]string, 0, len(e.Diags))
	for _, d := range e.Diags {
		parts = append(parts, renderLegacy(d))
	}
	return strings.Join(parts, "\n")
}

func renderLegacy(d diag.Diagnostic) string {
	out := ""
	if d.Primary.Start.Line > 0 {
		out += itoa(d.Primary.Start.Line) + ":" + itoa(d.Primary.Start.Col) + ": "
	}
	if d.Message != "" {
		out += d.Message
	} else {
		out += d.Title
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := false
	if n < 0 {
		negative = true
		n = -n
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// newDiag constructs a parser-stage diagnostic and appends it to
// the parser's accumulator. Returning the resulting ParserError
// lets the caller `return p.err(...)` while keeping the existing
// `error` return shape.
func (p *Parser) newDiag(code, title, message string, line, col int, hints ...string) *ParserError {
	d := diag.Diagnostic{
		Severity: diag.Error,
		Code:     code,
		Title:    title,
		Message:  message,
		Primary: diag.Region{
			Start: diag.Pos{Line: line, Col: col},
			End:   diag.Pos{Line: line, Col: col + 1},
		},
		Hints:  append([]string{}, hints...),
		Source: "tya",
	}
	p.errs = append(p.errs, d)
	return &ParserError{Diags: []diag.Diagnostic{d}}
}

// flushErrors wraps the accumulated diagnostics into a ParserError
// (or returns nil when there are none). Used by Parse() at exit.
func (p *Parser) flushErrors() error {
	if len(p.errs) == 0 {
		return nil
	}
	return &ParserError{Diags: append([]diag.Diagnostic{}, p.errs...)}
}
