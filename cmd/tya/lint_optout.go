package main

import (
	"strings"

	"tya/internal/lexer"
)

// optoutMap is line-number → set of suppressed codes. The empty
// string "" entry means "suppress every code".
type optoutMap map[int]map[string]bool

// buildOptouts scans the comment list extracted by
// lexer.LexWithComments and returns a map keyed by source line.
// `# tya-lint-ignore` (no code) suppresses everything on that line.
// `# tya-lint-ignore: CODE[, CODE...]` suppresses the listed codes.
//
// Full-line comments suppress the following statement: their
// associated line is `comment.Line + 1`. Inline (end-of-line)
// comments suppress findings reported on their own line.
func buildOptouts(comments []lexer.Comment) optoutMap {
	out := optoutMap{}
	for _, c := range comments {
		body := strings.TrimSpace(c.Text)
		const prefix = "tya-lint-ignore"
		if !strings.HasPrefix(body, prefix) {
			continue
		}
		rest := strings.TrimSpace(body[len(prefix):])
		codes := map[string]bool{}
		switch {
		case rest == "":
			codes[""] = true // wildcard
		case strings.HasPrefix(rest, ":"):
			for _, raw := range strings.Split(rest[1:], ",") {
				code := strings.TrimSpace(raw)
				if code != "" {
					codes[code] = true
				}
			}
		default:
			// Not the form we recognise; ignore.
			continue
		}
		target := c.Line
		if c.IsFullLine {
			target = c.Line + 1
		}
		if _, ok := out[target]; !ok {
			out[target] = map[string]bool{}
		}
		for code := range codes {
			out[target][code] = true
		}
	}
	return out
}

// suppressed reports whether the (line, code) pair is silenced by
// any opt-out directive collected from the source.
func (m optoutMap) suppressed(line int, code string) bool {
	entry, ok := m[line]
	if !ok {
		return false
	}
	if entry[""] {
		return true
	}
	return entry[code]
}
