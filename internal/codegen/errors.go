package codegen

import (
	"errors"
	"fmt"

	"tya/internal/diag"
)

// Codegen diagnostic codes occupy the TYA-E0601-0606 band.
const (
	codeAssignTargetUnsupported = "TYA-E0601"
	codeStmtUnsupported         = "TYA-E0602"
	codePatternUnsupported      = "TYA-E0603"
	codeTryOutsideFunc          = "TYA-E0604"
	codeMultiAssignNonTuple     = "TYA-E0605"
	codeDestructureNonIdent     = "TYA-E0606"
)

// CodegenError wraps a list of structured diagnostics produced
// during C emission. It implements error so callers can keep
// returning a single error value.
type CodegenError struct {
	Diags []diag.Diagnostic
}

func (e *CodegenError) Error() string {
	if len(e.Diags) == 0 {
		return "codegen error"
	}
	if len(e.Diags) == 1 {
		return formatLegacy(e.Diags[0])
	}
	out := ""
	for i, d := range e.Diags {
		if i > 0 {
			out += "\n"
		}
		out += formatLegacy(d)
	}
	return out
}

func formatLegacy(d diag.Diagnostic) string {
	if d.Message != "" {
		return d.Message
	}
	return d.Title
}

// codegenError builds a single-diagnostic CodegenError. Position
// is supplied as 1-origin (line, col) and may be zero when no AST
// node is in scope.
func codegenError(code, message string, line, col int) error {
	return &CodegenError{Diags: []diag.Diagnostic{{
		Severity: diag.Error,
		Code:     code,
		Title:    "Codegen",
		Message:  fmt.Sprintf("[%s] %s", code, message),
		Primary: diag.Region{
			Start: diag.Pos{Line: line, Col: col},
			End:   diag.Pos{Line: line, Col: col + 1},
		},
		Source: "tya",
	}}}
}

// AsCodegenError unwraps err to *CodegenError if present.
func AsCodegenError(err error) (*CodegenError, bool) {
	var ce *CodegenError
	if errors.As(err, &ce) {
		return ce, true
	}
	return nil, false
}
