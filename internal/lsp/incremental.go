package lsp

import (
	"fmt"
	"strings"
)

// ApplyChange returns text with `change` applied. When `change.Range`
// is nil the change is a Full-replace and `change.Text` becomes the
// new buffer; otherwise the byte span covered by Range is replaced
// with Text. Bad ranges return an error so the server can fall back
// to a full replace.
func ApplyChange(text string, change TextDocumentContentChangeEvent) (string, error) {
	if change.Range == nil {
		return change.Text, nil
	}
	startOff, err := positionToOffset(text, change.Range.Start)
	if err != nil {
		return "", err
	}
	endOff, err := positionToOffset(text, change.Range.End)
	if err != nil {
		return "", err
	}
	if startOff > endOff {
		return "", fmt.Errorf("range start past end: %d > %d", startOff, endOff)
	}
	return text[:startOff] + change.Text + text[endOff:], nil
}

// positionToOffset converts an LSP Position into a byte offset in
// text. v0.53 advertises UTF-8 as the position encoding, so we
// treat Character as a byte column. Callers running against
// UTF-16-only clients will still see correct behaviour for ASCII
// identifiers (the dominant case in tya).
func positionToOffset(text string, p Position) (int, error) {
	line := 0
	col := 0
	for i := 0; i < len(text); i++ {
		if line == p.Line && col == p.Character {
			return i, nil
		}
		if text[i] == '\n' {
			if line == p.Line {
				// Past the end of this line; clamp to its end.
				return i, nil
			}
			line++
			col = 0
			continue
		}
		col++
	}
	if line == p.Line && col == p.Character {
		return len(text), nil
	}
	// Clients sometimes send positions just past the buffer end
	// (open-end ranges); clamp instead of erroring.
	if line < p.Line || col < p.Character {
		return len(text), nil
	}
	return 0, fmt.Errorf("position %d:%d out of range", p.Line, p.Character)
}

// offsetToPosition is the inverse of positionToOffset; useful when
// servers need to report ranges produced from raw byte indices.
func offsetToPosition(text string, off int) Position {
	if off > len(text) {
		off = len(text)
	}
	line := 0
	col := 0
	for i := 0; i < off; i++ {
		if text[i] == '\n' {
			line++
			col = 0
			continue
		}
		col++
	}
	return Position{Line: line, Character: col}
}

// lineCount returns the number of newline-separated lines in text.
func lineCount(text string) int {
	if text == "" {
		return 1
	}
	n := strings.Count(text, "\n")
	if !strings.HasSuffix(text, "\n") {
		n++
	}
	return n
}
