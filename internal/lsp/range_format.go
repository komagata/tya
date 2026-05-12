package lsp

import (
	"strings"

	"tya/internal/formatter"
	"tya/internal/lexer"
	"tya/internal/parser"
)

// FormatRange implements the v0.53 heuristic A range formatter:
// widen the requested range to the smallest set of top-level
// statements that fully encloses it, run formatter.Unparse on the
// entire program, then replace exactly those lines in the buffer.
//
// Range with no top-level statements inside (e.g. a blank file) is
// a no-op (empty TextEdit list).
func FormatRange(src string, rng Range) ([]TextEdit, error) {
	toks, lcomments, lerrs := lexer.LexWithComments(src)
	if len(lerrs) > 0 {
		return nil, lerrs[0]
	}
	prog, err := parser.ParseWithComments(toks, toCommentInfos(lcomments))
	if err != nil {
		return nil, err
	}
	full, err := formatter.Unparse(prog)
	if err != nil {
		return nil, err
	}
	if len(prog.Stmts) == 0 {
		return []TextEdit{}, nil
	}

	// Identify the contiguous index range of top-level Stmts that
	// overlap `rng` (LSP 0-origin lines).
	startLSP := rng.Start.Line
	endLSP := rng.End.Line
	if endLSP < startLSP {
		endLSP = startLSP
	}
	startIdx := -1
	endIdx := -1
	for i, s := range prog.Stmts {
		l := stmtFirstLine(s) - 1
		e := stmtLastLine(s) - 1
		if e < startLSP || l > endLSP {
			continue
		}
		if startIdx == -1 {
			startIdx = i
		}
		endIdx = i
	}
	if startIdx == -1 {
		return []TextEdit{}, nil
	}

	// Convert top-level Stmt indices back to LSP line ranges.
	startLine := stmtFirstLine(prog.Stmts[startIdx]) - 1
	endLine := stmtLastLine(prog.Stmts[endIdx]) - 1

	// Slice the unparsed full document by line for the same block.
	fullLines := strings.Split(strings.TrimRight(full, "\n"), "\n")
	if startLine >= len(fullLines) {
		return []TextEdit{}, nil
	}
	if endLine >= len(fullLines) {
		endLine = len(fullLines) - 1
	}
	replacement := strings.Join(fullLines[startLine:endLine+1], "\n") + "\n"

	return []TextEdit{{
		Range: Range{
			Start: Position{Line: startLine, Character: 0},
			End:   Position{Line: endLine + 1, Character: 0},
		},
		NewText: replacement,
	}}, nil
}
