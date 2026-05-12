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

// RunnerError is the structured wrapper around one diag.Diagnostic
// produced inside the runner. Existing callers continue to return
// `error`; LSP and CLI lift the structured payload back out via
// `errors.As`.
type RunnerError struct {
	Diag diag.Diagnostic
}

func (e *RunnerError) Error() string {
	if e == nil {
		return ""
	}
	if e.Diag.Primary.Start.Line > 0 {
		return fmt.Sprintf("[%s] %d:%d: %s",
			e.Diag.Code, e.Diag.Primary.Start.Line, e.Diag.Primary.Start.Col, e.Diag.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Diag.Code, e.Diag.Message)
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
	return &RunnerError{Diag: diag.Diagnostic{
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
	}}
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
