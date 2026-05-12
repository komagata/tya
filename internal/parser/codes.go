package parser

// Parser diagnostic codes occupy the TYA-E0100-0299 band.
// The constants below name the bucket each lexical issue belongs
// to. Many surface messages share the same code — the Title and
// Message fields of the resulting diag.Diagnostic carry the
// specifics.
const (
	// 0100-0119: expected token / syntax (open-of-block, bracket, keyword)
	CodeExpectedToken   = "TYA-E0100"
	CodeExpectedBlock   = "TYA-E0101"
	CodeUnexpectedToken = "TYA-E0102"

	// 0120-0139: positioning constraints (top-level, loop, function, class)
	CodePositionConstraint = "TYA-E0120"

	// 0140-0159: reserved / deprecated names
	CodeReservedName     = "TYA-E0140"
	CodeUnsupportedSyntax = "TYA-E0141"

	// 0160-0179: pattern / assignment syntax
	CodePatternSyntax = "TYA-E0160"

	// 0180-0199: expression-level errors
	CodeExpectedExpression = "TYA-E0180"
)
