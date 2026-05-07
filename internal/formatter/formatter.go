package formatter

import "strings"

// FormatSource applies the conservative v0.2 formatting pass without changing
// tokens or expression structure.
func FormatSource(src string) string {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	lines := strings.Split(src, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	for i, line := range lines {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			lines[i] = ""
			continue
		}
		width := 0
		pos := 0
		for pos < len(line) {
			switch line[pos] {
			case ' ':
				width++
				pos++
			case '\t':
				width += 2
				pos++
			default:
				goto doneIndent
			}
		}
	doneIndent:
		lines[i] = strings.Repeat(" ", width) + line[pos:]
	}
	return strings.Join(lines, "\n") + "\n"
}
