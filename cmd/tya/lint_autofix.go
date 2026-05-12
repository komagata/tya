package main

import (
	"strings"

	"tya/internal/checker"
)

// applyUnwrapIf returns source with the unwrap-if hints applied:
// each "unwrap-if" hint at line L through endLine E drops line L
// (the `if true/false` header) and de-indents lines L+1..E by two
// spaces. Returns the rewritten source plus the number of hints
// applied.
//
// Hints are processed in descending order of Line so earlier line
// numbers remain valid as we mutate the buffer.
func applyUnwrapIf(source string, hints []checker.LintAutofixHint) (string, int) {
	relevant := []checker.LintAutofixHint{}
	for _, h := range hints {
		if h.Kind == "unwrap-if" {
			relevant = append(relevant, h)
		}
	}
	if len(relevant) == 0 {
		return source, 0
	}
	// Sort in descending Line order so positional mutations stay
	// stable; the input list from LintAutofixHints is already
	// source-order ascending, so reverse it.
	for i, j := 0, len(relevant)-1; i < j; i, j = i+1, j-1 {
		relevant[i], relevant[j] = relevant[j], relevant[i]
	}

	lines := strings.Split(source, "\n")
	applied := 0
	for _, h := range relevant {
		// h.Line is 1-origin and points at the FIRST BODY stmt of
		// the `if`, so the header keyword sits one line above at
		// 1-origin h.Line-1 (0-origin h.Line-2). h.EndLine is the
		// last body stmt's line.
		if h.Line < 2 || h.Line > len(lines) {
			continue
		}
		endLine := h.EndLine
		if endLine < h.Line {
			endLine = h.Line
		}
		if endLine > len(lines) {
			endLine = len(lines)
		}
		headerIdx := h.Line - 2 // 0-origin
		newLines := make([]string, 0, len(lines))
		newLines = append(newLines, lines[:headerIdx]...)
		for i := h.Line - 1; i < endLine && i < len(lines); i++ {
			body := lines[i]
			strip := 0
			for strip < 2 && strip < len(body) && body[strip] == ' ' {
				strip++
			}
			newLines = append(newLines, body[strip:])
		}
		newLines = append(newLines, lines[endLine:]...)
		lines = newLines
		applied++
	}
	return strings.Join(lines, "\n"), applied
}
