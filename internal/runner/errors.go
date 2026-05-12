package runner

import (
	"errors"
	"fmt"

	"tya/internal/checker"
	"tya/internal/diag"
	"tya/internal/util"
)

// Runner diagnostic codes occupy the TYA-E0800-0899 band. The
// constants below add to the pre-v0.54 set (E0850-E0855 were
// already in use). New codes target the file / module / import
// edges that previously returned bare fmt.Errorf strings.
const (
	codeInvalidFileName       = "TYA-E0840"
	codeEntryRedefinesModule  = "TYA-E0856"
	codeImportNameConflict    = "TYA-E0857"
	codeUndefinedVariable     = "TYA-E0858"
)

// RunnerError is the structured wrapper around the diag.Diagnostic
// list produced inside the runner. v0.56 widens the previous
// single-Diag shape to []diag.Diagnostic for symmetry with
// *ParserError and *CodegenError. Runner is still fail-fast — Diags
// typically carries exactly one entry. The Diag convenience getter
// returns the first diagnostic for callers that pre-v0.56 read
// `.Diag` directly.
type RunnerError struct {
	Diags []diag.Diagnostic
}

func (e *RunnerError) Error() string {
	if e == nil || len(e.Diags) == 0 {
		return ""
	}
	d := e.Diags[0]
	if d.Primary.Start.Line > 0 {
		return fmt.Sprintf("[%s] %d:%d: %s",
			d.Code, d.Primary.Start.Line, d.Primary.Start.Col, d.Message)
	}
	return fmt.Sprintf("[%s] %s", d.Code, d.Message)
}

// Diag returns the first diagnostic (or a zero value when Diags is
// empty). Provided for compatibility with pre-v0.56 call sites that
// read `.Diag` as a single value.
func (e *RunnerError) Diag() diag.Diagnostic {
	if e == nil || len(e.Diags) == 0 {
		return diag.Diagnostic{}
	}
	return e.Diags[0]
}

// AsRunnerError unwraps err to *RunnerError when present.
func AsRunnerError(err error) (*RunnerError, bool) {
	var re *RunnerError
	if errors.As(err, &re) {
		return re, true
	}
	return nil, false
}

// runnerError builds a single-diagnostic RunnerError. Position is
// 1-origin; pass 0 when no source location is in scope.
func runnerError(code, message string, line, col int, hints ...string) error {
	return &RunnerError{Diags: []diag.Diagnostic{{
		Severity: diag.Error,
		Code:     code,
		Title:    "Runner",
		Message:  message,
		Primary: diag.Region{
			Start: diag.Pos{Line: line, Col: col},
			End:   diag.Pos{Line: line, Col: col + 1},
		},
		Hints:  append([]string{}, hints...),
		Source: "tya",
	}}}
}

// undefinedNameHint computes a "did you mean ..." hint string for
// name, drawing candidates from the builtin function list and the
// supplied scope names. Returns the empty string when no close
// candidate exists.
func undefinedNameHint(name string, locals []string) string {
	pool := append([]string{}, checker.BuiltinNames()...)
	pool = append(pool, locals...)
	suggestions := util.Suggest(name, pool, 2, 3)
	if len(suggestions) == 0 {
		return ""
	}
	if len(suggestions) == 1 {
		return fmt.Sprintf("did you mean %q?", suggestions[0])
	}
	out := "did you mean "
	for i, s := range suggestions {
		if i > 0 {
			out += " / "
		}
		out += fmt.Sprintf("%q", s)
	}
	out += "?"
	return out
}
