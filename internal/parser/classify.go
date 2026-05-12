package parser

import "strings"

// classifyParserCode infers the TYA-E01xx code from a raw parser
// error message. The mapping is heuristic — it groups the existing
// 75+ message strings into a handful of buckets so we can land
// stable codes without renaming every call site.
//
// The classification is intentionally conservative: when nothing
// matches, the generic "unexpected token" code (TYA-E0102) is
// returned. Future refactors can subdivide once the call sites
// have been updated to pass an explicit code.
func classifyParserCode(msg string) string {
	m := strings.ToLower(msg)
	switch {
	case strings.Contains(m, "expected indented block"):
		return CodeExpectedBlock
	case strings.Contains(m, "expected"):
		return CodeExpectedToken
	case strings.HasPrefix(m, "unexpected"):
		return CodeUnexpectedToken
	case strings.Contains(m, "outside") ||
		strings.Contains(m, "must be inside") ||
		strings.Contains(m, "not allowed at top level") ||
		strings.Contains(m, "only allowed"):
		return CodePositionConstraint
	case strings.Contains(m, "cannot be used as a name") ||
		strings.Contains(m, "reserved"):
		return CodeReservedName
	case strings.Contains(m, "not in tya") ||
		strings.Contains(m, "no longer supported") ||
		strings.Contains(m, "is not supported"):
		return CodeUnsupportedSyntax
	case strings.Contains(m, "pattern") ||
		strings.Contains(m, "destructuring") ||
		strings.Contains(m, "multiple-assignment"):
		return CodePatternSyntax
	case strings.Contains(m, "expression"):
		return CodeExpectedExpression
	}
	return CodeUnexpectedToken
}
