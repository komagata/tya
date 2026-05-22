package lexer

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"tya/internal/diag"
	"tya/internal/interp"
	"tya/internal/token"
)

// Diagnostic is the v0.32 lexer-error wrapper. It implements `error`
// so the existing []error API works, and exposes the underlying
// diag.Diagnostic for the CLI's renderer.
type Diagnostic struct {
	Diag diag.Diagnostic
}

func (e *Diagnostic) Error() string {
	d := e.Diag
	return fmt.Sprintf("%d:%d: %s", d.Primary.Start.Line, d.Primary.Start.Col, d.Message)
}

func (l *Lexer) diagErr(code, title, msg, hint string, line, col, length int) {
	if length < 1 {
		length = 1
	}
	d := diag.Diagnostic{
		Severity: diag.Error,
		Code:     code,
		Title:    title,
		Message:  msg,
		Primary: diag.Region{
			Start: diag.Pos{Line: line, Col: col},
			End:   diag.Pos{Line: line, Col: col + length},
		},
		Source: "lexer",
	}
	if hint != "" {
		d.Hints = []string{hint}
	}
	l.errs = append(l.errs, &Diagnostic{Diag: d})
}

func prefixName(c byte) string {
	if c == 'x' || c == 'X' {
		return "hex"
	}
	return "binary"
}

type Lexer struct {
	src      string
	tokens   []token.Token
	errs     []error
	ind      []int
	comments []Comment
}

// Comment carries one captured `# …` comment from the source.
// IsFullLine is true when the source line contains nothing but the
// comment (after leading whitespace); otherwise the comment was at
// end-of-line after a statement.
type Comment struct {
	Line       int
	Col        int
	Indent     int
	Text       string
	IsFullLine bool
}

func Lex(src string) ([]token.Token, []error) {
	l := &Lexer{src: strings.ReplaceAll(src, "\r\n", "\n"), ind: []int{0}}
	l.lex()
	return suppressBracketNewlines(l.tokens), l.errs
}

// LexWithComments runs the lexer and returns the captured comments
// alongside the token stream.
func LexWithComments(src string) ([]token.Token, []Comment, []error) {
	l := &Lexer{src: strings.ReplaceAll(src, "\r\n", "\n"), ind: []int{0}}
	l.lex()
	return suppressBracketNewlines(l.tokens), l.comments, l.errs
}

// suppressBracketNewlines drops NEWLINE / INDENT / DEDENT tokens
// that fall inside `(` / `[` brackets so the parser can read
// multi-line call argument lists and array literals as a single
// logical line. It also recognizes binary-operator continuation
// lines per CANONICAL §5.3.5: when a line starts with a binary
// operator at deeper indent, the NEWLINE+INDENT before the
// operator and the matching DEDENT after the continued
// expression are dropped, so the leading-operator multi-line
// form parses as a single binary expression.
//
// Brace literals (`{`) stay single-line per §5.3.3 (the dict
// block form has no braces) and are not affected.
func suppressBracketNewlines(toks []token.Token) []token.Token {
	out := make([]token.Token, 0, len(toks))
	depth := 0
	pendingDedent := 0
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		switch t.Type {
		case token.LPAREN, token.LBRACKET:
			depth++
		case token.RPAREN, token.RBRACKET:
			if depth > 0 {
				depth--
			}
		}
		if depth > 0 && (t.Type == token.NEWLINE || t.Type == token.INDENT || t.Type == token.DEDENT) {
			continue
		}
		// Detect `NEWLINE INDENT <binary-op>` and drop
		// the NEWLINE and INDENT, marking that we owe a
		// DEDENT-drop later.
		// `NEWLINE INDENT <binary-op>` opens a continuation
		// indented block — drop both, owe a DEDENT-drop.
		if t.Type == token.NEWLINE && i+2 < len(toks) &&
			toks[i+1].Type == token.INDENT &&
			isContinuationOp(toks[i+2]) {
			pendingDedent++
			i++ // skip INDENT
			continue
		}
		// `NEWLINE <binary-op>` at the same depth (subsequent
		// continuation lines after the first) — drop the
		// NEWLINE only.
		if t.Type == token.NEWLINE && i+1 < len(toks) && isContinuationOp(toks[i+1]) {
			continue
		}
		// `INDENT <binary-op>` (no preceding NEWLINE in the
		// stream — defensive).
		if t.Type == token.INDENT && i+1 < len(toks) && isContinuationOp(toks[i+1]) {
			pendingDedent++
			continue
		}
		if t.Type == token.DEDENT && pendingDedent > 0 {
			pendingDedent--
			continue
		}
		out = append(out, t)
	}
	return out
}

// isContinuationOp reports whether t can start a leading-operator
// continuation line per CANONICAL §5.3.5.
func isContinuationOp(t token.Token) bool {
	switch t.Type {
	case token.PLUS, token.MINUS, token.STAR, token.SLASH, token.PERCENT,
		token.EQ, token.NEQ, token.LT, token.LTE, token.GT, token.GTE,
		token.AMP, token.PIPE, token.CARET, token.SHL, token.SHR:
		return true
	case token.IDENT:
		return t.Lexeme == "and" || t.Lexeme == "or"
	}
	return false
}

func (l *Lexer) add(t token.Type, s string, line, col int) {
	l.tokens = append(l.tokens, token.Token{Type: t, Lexeme: s, Line: line, Col: col})
}

func (l *Lexer) addString(t token.Type, s string, line, col int, form, lang, marker string) {
	l.tokens = append(l.tokens, token.Token{Type: t, Lexeme: s, Line: line, Col: col, StringForm: form, Lang: lang, Marker: marker})
}

func (l *Lexer) lex() {
	lines := strings.Split(l.src, "\n")
	for i := 0; i < len(lines); i++ {
		raw := lines[i]
		lineNo := i + 1
		stripped, commentText, commentCol, hasComment := splitComment(raw)
		if hasComment {
			indent := len(raw) - len(strings.TrimLeft(raw, " "))
			fullLine := strings.TrimSpace(stripped) == ""
			l.comments = append(l.comments, Comment{
				Line:       lineNo,
				Col:        commentCol + 1,
				Indent:     indent,
				Text:       commentText,
				IsFullLine: fullLine,
			})
		}
		line := strings.TrimRight(stripped, " ")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.Contains(line, "\t") {
			l.diagErr("TYA-E0001", "Tabs are forbidden",
				"This line contains a tab character; Tya source must use spaces only.",
				"Replace the tab with two spaces.",
				lineNo, 1, 1)
			continue
		}
		if strings.TrimRight(line, " ") != line {
			l.diagErr("TYA-E0002", "Trailing whitespace",
				"This line has trailing whitespace.",
				"Remove the trailing spaces.",
				lineNo, 1, 1)
			continue
		}
		spaces := len(line) - len(strings.TrimLeft(line, " "))
		if spaces%2 != 0 {
			l.diagErr("TYA-E0003", "Indentation step",
				"Indentation must use exactly 2 spaces.",
				"Round the leading-space count to a multiple of 2.",
				lineNo, 1, 1)
			continue
		}
		l.handleIndent(spaces, lineNo)
		consumed := l.lexLineWithLines(strings.TrimLeft(line, " "), lineNo, spaces+1, lines, i)
		if consumed > 0 {
			i += consumed
			l.add(token.NEWLINE, "", i+1, 1)
			continue
		}
		l.add(token.NEWLINE, "", lineNo, len(raw)+1)
	}
	for len(l.ind) > 1 {
		l.ind = l.ind[:len(l.ind)-1]
		l.add(token.DEDENT, "", len(lines), 1)
	}
	l.add(token.EOF, "", len(lines), 1)
}

// lexLineWithLines lexes a single logical line, but if a multi-line
// triple-quoted string is encountered, it consumes additional lines
// from `lines` starting after `lineIdx`. Returns the number of extra
// lines consumed (0 if none).
func (l *Lexer) lexLineWithLines(s string, line, baseCol int, lines []string, lineIdx int) int {
	tq := findStringOpen(s)
	if tq.kind == "" {
		l.lexLine(s, line, baseCol)
		return 0
	}
	if tq.invalidTag != "" {
		l.diagErr("TYA-E0018", "Invalid string tag", "Language tags must match [a-z][a-z0-9_]*.", "Use a lowercase tag such as sql or html.", line, baseCol+tq.start, len(tq.invalidTag))
		return 0
	}
	if tq.kind == "heredoc" {
		return l.lexHeredoc(s, line, baseCol, lines, lineIdx, tq)
	}
	// Lex everything before the (optional prefix +) triple-quote normally.
	preEnd := tq.start
	if preEnd > 0 {
		l.lexLine(s[:preEnd], line, baseCol)
	}
	// Single-line triple quote? Look for a closing """ on the same line.
	body := s[tq.delim+3:]
	if close := strings.Index(body, `"""`); close >= 0 {
		raw := body[:close]
		switch tq.prefix {
		case 'r':
			l.addString(token.STRING, encodeRawString(raw), line, baseCol+preEnd, "raw_triple", "", "")
		case 'b':
			value, err := interpretBytesEscapes(raw, line, baseCol+tq.delim)
			if err != nil {
				l.errs = append(l.errs, err)
				return 0
			}
			l.addString(token.BYTES, value, line, baseCol+preEnd, "bytes_triple", "", "")
		default:
			value, err := interpretEscapes(raw, line, baseCol+tq.delim)
			if err != nil {
				l.errs = append(l.errs, err)
				return 0
			}
			l.addString(token.STRING, value, line, baseCol+preEnd, "triple", tq.lang, "")
		}
		// Lex the remainder after the closing """.
		rest := body[close+3:]
		if rest != "" {
			l.lexLine(rest, line, baseCol+tq.delim+3+close+3)
		}
		return 0
	}
	// Multi-line triple-quoted string. Collect raw body across lines,
	// strip the closing line's indent baseline, interpret escapes,
	// emit STRING.
	openingLineRest := body
	closingFound := false
	closingIdx := -1
	closingIndent := 0
	for j := lineIdx + 1; j < len(lines); j++ {
		ln := lines[j]
		trimmed := strings.TrimLeft(ln, " ")
		if strings.HasPrefix(trimmed, `"""`) {
			closingFound = true
			closingIdx = j
			closingIndent = len(ln) - len(trimmed)
			break
		}
	}
	if !closingFound {
		l.diagErr("TYA-E0016", "Unterminated triple-quoted string",
			`This """ literal is not closed.`,
			`Add a matching """ on a later line.`,
			line, baseCol+tq.delim, 3)
		return 0
	}
	var b strings.Builder
	openingHasContent := openingLineRest != ""
	if openingHasContent {
		b.WriteString(openingLineRest)
		b.WriteByte('\n')
	} else {
		// Skip the immediate newline after `"""` only when nothing
		// followed on the opening line.
	}
	for j := lineIdx + 1; j < closingIdx; j++ {
		bodyLine := lines[j]
		if strings.Contains(bodyLine, "\t") {
			l.diagErr("TYA-E0001", "Tabs are forbidden",
				"This line of the triple-quoted string contains a tab character.",
				"Replace the tab with spaces.",
				j+1, 1, 1)
			return closingIdx - lineIdx
		}
		if bodyLine == "" {
			b.WriteByte('\n')
			continue
		}
		if len(bodyLine) < closingIndent || bodyLine[:closingIndent] != strings.Repeat(" ", closingIndent) {
			leading := len(bodyLine) - len(strings.TrimLeft(bodyLine, " "))
			if leading == len(bodyLine) {
				// Whitespace-only line: treat as empty.
				b.WriteByte('\n')
				continue
			}
			l.diagErr("TYA-E0017", "Mixed indentation in triple-string",
				"This line is shallower than the closing \"\"\" indent baseline.",
				"Indent every body line at least as far as the closing \"\"\".",
				j+1, 1, 1)
			return closingIdx - lineIdx
		}
		b.WriteString(bodyLine[closingIndent:])
		b.WriteByte('\n')
	}
	switch tq.prefix {
	case 'r':
		l.addString(token.STRING, encodeRawString(b.String()), line, baseCol+preEnd, "raw_triple", "", "")
	case 'b':
		value, err := interpretBytesEscapes(b.String(), line, baseCol+tq.delim)
		if err != nil {
			l.errs = append(l.errs, err)
			return closingIdx - lineIdx
		}
		l.addString(token.BYTES, value, line, baseCol+preEnd, "bytes_triple", "", "")
	default:
		value, err := interpretEscapes(b.String(), line, baseCol+tq.delim)
		if err != nil {
			l.errs = append(l.errs, err)
			return closingIdx - lineIdx
		}
		l.addString(token.STRING, value, line, baseCol+preEnd, "triple", tq.lang, "")
	}
	// Lex any content after the closing """ on its line.
	closingLine := lines[closingIdx]
	closingTrimmed := strings.TrimLeft(closingLine, " ")
	tail := closingTrimmed[3:]
	if tail != "" {
		l.lexLine(tail, closingIdx+1, closingIndent+4)
	}
	return closingIdx - lineIdx
}

type stringOpen struct {
	kind       string
	start      int
	delim      int
	prefix     byte
	lang       string
	marker     string
	invalidTag string
}

func findStringOpen(s string) stringOpen {
	best := stringOpen{}
	for i := 0; i < len(s); i++ {
		if i+2 < len(s) && s[i] == '"' && s[i+1] == '"' && s[i+2] == '"' {
			open := classifyStringPrefix(s, i)
			open.kind = "triple"
			open.delim = i
			if best.kind == "" || open.start < best.start {
				best = open
			}
		}
		if i+2 < len(s) && s[i] == '<' && s[i+1] == '<' && s[i+2] == '<' {
			open := classifyStringPrefix(s, i)
			open.kind = "heredoc"
			open.delim = i
			rest := s[i+3:]
			markerEnd := 0
			for markerEnd < len(rest) && isIdentChar(rest[markerEnd]) {
				markerEnd++
			}
			open.marker = rest[:markerEnd]
			if best.kind == "" || open.start < best.start {
				best = open
			}
		}
	}
	return best
}

func classifyStringPrefix(s string, delim int) stringOpen {
	open := stringOpen{start: delim}
	if delim == 0 {
		return open
	}
	j := delim - 1
	for j >= 0 && isIdentChar(s[j]) {
		j--
	}
	if j == delim-1 {
		return open
	}
	word := s[j+1 : delim]
	if j >= 0 && isIdentChar(s[j]) {
		return open
	}
	if word == "r" || word == "b" {
		open.start = delim - 1
		open.prefix = word[0]
		return open
	}
	if isLanguageTag(word) {
		open.start = j + 1
		open.lang = word
	} else if word != "" {
		open.start = j + 1
		open.invalidTag = word
	}
	return open
}

func isLanguageTag(s string) bool {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= '0' && s[i] <= '9') || s[i] == '_') {
			return false
		}
	}
	return true
}

func isHeredocMarker(s string) bool {
	if s == "" || s[0] < 'A' || s[0] > 'Z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !((s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= '0' && s[i] <= '9') || s[i] == '_') {
			return false
		}
	}
	return true
}

func (l *Lexer) lexHeredoc(s string, line, baseCol int, lines []string, lineIdx int, open stringOpen) int {
	if open.start > 0 {
		l.lexLine(s[:open.start], line, baseCol)
	}
	if open.prefix == 'r' && open.lang != "" {
		l.diagErr("TYA-E0018", "Invalid string tag", "Raw language-tagged strings are not supported.", "Use r<<<MARKER without a language tag.", line, baseCol+open.start, open.delim-open.start+3)
		return 0
	}
	rest := s[open.delim+3:]
	if open.marker == "" || !isHeredocMarker(open.marker) {
		l.diagErr("TYA-E0019", "Invalid heredoc marker", "Heredoc markers must match [A-Z][A-Z0-9_]*.", "Use an uppercase marker such as SQL or HTML.", line, baseCol+open.delim, len(rest)+3)
		return 0
	}
	if strings.TrimSpace(rest[len(open.marker):]) != "" {
		l.diagErr("TYA-E0020", "Invalid heredoc opening", "Only the heredoc marker may follow <<< on the opening line.", "Move content to the next line.", line, baseCol+open.delim, len(rest)+3)
		return 0
	}
	closingFound := false
	closingIdx := -1
	closingIndent := 0
	for j := lineIdx + 1; j < len(lines); j++ {
		ln := lines[j]
		trimmed := strings.TrimLeft(ln, " ")
		if trimmed == open.marker {
			closingFound = true
			closingIdx = j
			closingIndent = len(ln) - len(trimmed)
			break
		}
	}
	if !closingFound {
		l.diagErr("TYA-E0021", "Unterminated heredoc string", "This heredoc literal is not closed.", "Add a closing marker line that matches "+open.marker+".", line, baseCol+open.delim, len(open.marker)+3)
		return 0
	}
	var b strings.Builder
	for j := lineIdx + 1; j < closingIdx; j++ {
		bodyLine := lines[j]
		if strings.Contains(bodyLine, "\t") {
			l.diagErr("TYA-E0001", "Tabs are forbidden", "This line of the heredoc string contains a tab character.", "Replace the tab with spaces.", j+1, 1, 1)
			return closingIdx - lineIdx
		}
		if bodyLine == "" {
			b.WriteByte('\n')
			continue
		}
		if len(bodyLine) < closingIndent || bodyLine[:closingIndent] != strings.Repeat(" ", closingIndent) {
			leading := len(bodyLine) - len(strings.TrimLeft(bodyLine, " "))
			if leading == len(bodyLine) {
				b.WriteByte('\n')
				continue
			}
			l.diagErr("TYA-E0017", "Mixed indentation in heredoc string", "This line is shallower than the closing marker indent baseline.", "Indent every body line at least as far as the closing marker.", j+1, 1, 1)
			return closingIdx - lineIdx
		}
		b.WriteString(bodyLine[closingIndent:])
		b.WriteByte('\n')
	}
	switch open.prefix {
	case 'r':
		l.addString(token.STRING, encodeRawString(b.String()), line, baseCol+open.start, "raw_heredoc", "", open.marker)
	case 'b':
		value, err := interpretBytesEscapes(b.String(), line, baseCol+open.delim)
		if err != nil {
			l.errs = append(l.errs, err)
			return closingIdx - lineIdx
		}
		l.addString(token.BYTES, value, line, baseCol+open.start, "bytes_heredoc", "", open.marker)
	default:
		value, err := interpretEscapes(b.String(), line, baseCol+open.delim)
		if err != nil {
			l.errs = append(l.errs, err)
			return closingIdx - lineIdx
		}
		l.addString(token.STRING, value, line, baseCol+open.start, "heredoc", open.lang, open.marker)
	}
	return closingIdx - lineIdx
}

// findTripleQuote scans s for the earliest triple-quote, optionally
// preceded by a single-character prefix (`r` or `b`). It returns the
// byte offset of the first `"` of the `"""` sequence and the prefix
// byte, or (-1, 0) when no triple-quote occurs. The prefix is
// recognized only when it sits immediately before `"""` and is not
// part of a longer identifier (the byte before, if any, is not an
// ident-continuation character).
func findTripleQuote(s string) (int, byte) {
	for i := 0; i+2 < len(s); i++ {
		if s[i] != '"' || s[i+1] != '"' || s[i+2] != '"' {
			continue
		}
		// Plain triple-quote at i. Check for r/b prefix.
		if i > 0 {
			prev := s[i-1]
			if prev == 'r' || prev == 'b' {
				if i-2 < 0 || !isIdentChar(s[i-2]) {
					return i, prev
				}
			}
		}
		return i, 0
	}
	return -1, 0
}

func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

// encodeRawString turns a raw-string body into a STRING.Lexeme that
// the existing interpolation runtime decodes back to the verbatim
// body. Brace characters are doubled so `{name}` is treated as
// literal text.
func encodeRawString(body string) string {
	var out strings.Builder
	for i := 0; i < len(body); i++ {
		switch body[i] {
		case '{':
			out.WriteString("{{")
		case '}':
			out.WriteString("}}")
		default:
			out.WriteByte(body[i])
		}
	}
	return out.String()
}

// interpretBytesEscapes processes \n, \t, \r, \", \\, and \xHH
// escapes for a bytes-literal body. Mirrors the v0.25 single-line
// `b"..."` escape rules but is reused for `b"""..."""`.
func interpretBytesEscapes(s string, line, col int) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		if i+1 >= len(s) {
			return "", fmt.Errorf("%d:%d: unterminated escape", line, col+i)
		}
		switch s[i+1] {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case '"':
			b.WriteByte('"')
		case '\\':
			b.WriteByte('\\')
		case 'x':
			if i+3 >= len(s) {
				return "", fmt.Errorf("%d:%d: truncated \\x escape", line, col+i)
			}
			hi := hexDigit(s[i+2])
			lo := hexDigit(s[i+3])
			if hi < 0 || lo < 0 {
				return "", fmt.Errorf("%d:%d: invalid \\x escape", line, col+i)
			}
			b.WriteByte(byte(hi*16 + lo))
			i += 3
			continue
		default:
			return "", fmt.Errorf("%d:%d: unknown escape \\%c", line, col+i, s[i+1])
		}
		i++
	}
	return b.String(), nil
}

// interpretEscapes processes \n, \t, \r, \", \\, \{ inside a string body
// the same way the single-line "..." path does. {{ and }} pass
// through unchanged for the interpolation pipeline.
func interpretEscapes(s string, line, col int) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '{' && i+1 < len(s) && s[i+1] == '{' && !interp.LooksLikeDictExprStart(s, i) {
			b.WriteString("{{")
			i++
			continue
		}
		if c == '{' {
			if close := interp.FindExprEnd(s, i); close >= 0 {
				if interp.LooksLikeDictExprStart(s, i) {
					b.WriteString("{(")
					b.WriteString(s[i+1 : close])
					b.WriteString(")}")
				} else {
					b.WriteString(s[i : close+1])
				}
				i = close
				continue
			}
		}
		if c == '\\' {
			if i+1 >= len(s) {
				return "", &Diagnostic{Diag: diag.Diagnostic{
					Severity: diag.Error,
					Code:     "TYA-E0007",
					Title:    "Unterminated escape",
					Message:  "Backslash at end of string body has no escape character.",
					Primary: diag.Region{
						Start: diag.Pos{Line: line, Col: col + i},
						End:   diag.Pos{Line: line, Col: col + i + 1},
					},
					Hints:  []string{"Add the escape character (e.g. \\n, \\t, \\\\)."},
					Source: "lexer",
				}}
			}
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			case '{':
				b.WriteString(`\{`)
			default:
				return "", &Diagnostic{Diag: diag.Diagnostic{
					Severity: diag.Error,
					Code:     "TYA-E0008",
					Title:    "Unknown escape",
					Message:  fmt.Sprintf("Unknown escape sequence \\%c.", s[i+1]),
					Primary: diag.Region{
						Start: diag.Pos{Line: line, Col: col + i},
						End:   diag.Pos{Line: line, Col: col + i + 2},
					},
					Hints:  []string{"Supported escapes: \\n, \\t, \\r, \\\", \\\\, \\{."},
					Source: "lexer",
				}}
			}
			i++
			continue
		}
		b.WriteByte(c)
	}
	return b.String(), nil
}

func interpretSingleQuotedEscapes(s string, line, col int) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '{':
			b.WriteString("{{")
			continue
		case '}':
			b.WriteString("}}")
			continue
		case '\\':
			if i+1 >= len(s) {
				return "", &Diagnostic{Diag: diag.Diagnostic{
					Severity: diag.Error,
					Code:     "TYA-E0007",
					Title:    "Unterminated escape",
					Message:  "Backslash at end of string body has no escape character.",
					Primary: diag.Region{
						Start: diag.Pos{Line: line, Col: col + i},
						End:   diag.Pos{Line: line, Col: col + i + 1},
					},
					Hints:  []string{"Add the escape character (e.g. \\n, \\t, \\\\, \\')."},
					Source: "lexer",
				}}
			}
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '"':
				b.WriteByte('"')
			case '\'':
				b.WriteByte('\'')
			case '\\':
				b.WriteByte('\\')
			case '{':
				b.WriteString(`\{`)
			default:
				return "", &Diagnostic{Diag: diag.Diagnostic{
					Severity: diag.Error,
					Code:     "TYA-E0008",
					Title:    "Unknown escape",
					Message:  fmt.Sprintf("Unknown escape sequence \\%c.", s[i+1]),
					Primary: diag.Region{
						Start: diag.Pos{Line: line, Col: col + i},
						End:   diag.Pos{Line: line, Col: col + i + 2},
					},
					Hints:  []string{"Supported escapes: \\n, \\t, \\r, \\\", \\', \\\\, \\{."},
					Source: "lexer",
				}}
			}
			i++
			continue
		}
		b.WriteByte(c)
	}
	return b.String(), nil
}

func (l *Lexer) handleIndent(n, line int) {
	cur := l.ind[len(l.ind)-1]
	if n > cur {
		if n != cur+2 {
			l.diagErr("TYA-E0005", "Indentation step",
				"Indentation may only increase by 2 spaces.",
				"Match the previous indent and add exactly 2 spaces.",
				line, 1, 1)
		}
		l.ind = append(l.ind, n)
		l.add(token.INDENT, "", line, 1)
		return
	}
	for n < cur {
		l.ind = l.ind[:len(l.ind)-1]
		l.add(token.DEDENT, "", line, 1)
		cur = l.ind[len(l.ind)-1]
	}
	if n != cur {
		l.diagErr("TYA-E0004", "Inconsistent indentation",
			"This line's indentation does not match any enclosing block.",
			"Align this line with the surrounding block.",
			line, 1, 1)
	}
}

func (l *Lexer) lexLine(s string, line, baseCol int) {
	for i := 0; i < len(s); {
		ch := s[i]
		col := baseCol + i
		if ch == ' ' {
			i++
			continue
		}
		if ch == 'r' && i+1 < len(s) && s[i+1] == '"' && !(i+3 < len(s) && s[i+2] == '"' && s[i+3] == '"') {
			// Raw single-line string. Body is verbatim until the
			// next `"`. Brace characters are doubled so the
			// existing interpolation pipeline treats them as
			// literal.
			start := baseCol + i
			i += 2
			var b strings.Builder
			for i < len(s) && s[i] != '"' {
				switch s[i] {
				case '{':
					b.WriteString("{{")
				case '}':
					b.WriteString("}}")
				default:
					b.WriteByte(s[i])
				}
				i++
			}
			if i >= len(s) {
				l.diagErr("TYA-E0006", "Unterminated string",
					"This raw string literal has no closing quote.",
					`Add a closing " on the same line, or use r"""...""" for a multi-line raw string.`,
					line, start, 1)
				return
			}
			i++
			l.add(token.STRING, b.String(), line, start)
			continue
		}
		if ch == 'b' && i+1 < len(s) && s[i+1] == '"' && !(i+3 < len(s) && s[i+2] == '"' && s[i+3] == '"') {
			var b strings.Builder
			start := baseCol + i
			i += 2
			for i < len(s) && s[i] != '"' {
				if s[i] == '\\' {
					if i+1 >= len(s) {
						l.errs = append(l.errs, fmt.Errorf("%d:%d: unterminated escape", line, baseCol+i))
						return
					}
					switch s[i+1] {
					case 'n':
						b.WriteByte('\n')
					case 't':
						b.WriteByte('\t')
					case 'r':
						b.WriteByte('\r')
					case '"':
						b.WriteByte('"')
					case '\\':
						b.WriteByte('\\')
					case 'x':
						if i+3 >= len(s) {
							l.errs = append(l.errs, fmt.Errorf("%d:%d: truncated \\x escape", line, baseCol+i))
							return
						}
						hi := hexDigit(s[i+2])
						lo := hexDigit(s[i+3])
						if hi < 0 || lo < 0 {
							l.errs = append(l.errs, fmt.Errorf("%d:%d: invalid \\x escape", line, baseCol+i))
							return
						}
						b.WriteByte(byte(hi*16 + lo))
						i += 4
						continue
					default:
						l.errs = append(l.errs, fmt.Errorf("%d:%d: unknown escape \\%c", line, baseCol+i, s[i+1]))
						return
					}
					i += 2
					continue
				}
				b.WriteByte(s[i])
				i++
			}
			if i >= len(s) {
				l.errs = append(l.errs, fmt.Errorf("%d:%d: unterminated bytes literal", line, start))
				return
			}
			i++
			l.add(token.BYTES, b.String(), line, start)
			continue
		}
		if isAlpha(ch) || ch == '_' {
			start := i
			for i < len(s) && (isAlpha(s[i]) || isDigit(s[i]) || s[i] == '_') {
				i++
			}
			l.add(token.IDENT, s[start:i], line, baseCol+start)
			continue
		}
		if isDigit(ch) {
			start := i
			// Hex / binary literal: 0x... or 0b...
			if ch == '0' && i+1 < len(s) && (s[i+1] == 'x' || s[i+1] == 'X' || s[i+1] == 'b' || s[i+1] == 'B') {
				prefix := s[i+1]
				i += 2
				digitStart := i
				digits := strings.Builder{}
				isHex := prefix == 'x' || prefix == 'X'
				for i < len(s) {
					c := s[i]
					if c == '_' {
						i++
						continue
					}
					if isHex && hexDigit(c) >= 0 {
						digits.WriteByte(c)
						i++
						continue
					}
					if !isHex && (c == '0' || c == '1') {
						digits.WriteByte(c)
						i++
						continue
					}
					break
				}
				if digits.Len() == 0 {
					l.errs = append(l.errs, fmt.Errorf("%d:%d: %s literal needs at least one digit", line, baseCol+start, prefixName(prefix)))
					return
				}
				if i < len(s) && (isAlpha(s[i]) || isDigit(s[i])) {
					l.errs = append(l.errs, fmt.Errorf("%d:%d: invalid digit %q in %s literal", line, baseCol+i, s[i], prefixName(prefix)))
					return
				}
				_ = digitStart
				base := 16
				if !isHex {
					base = 2
				}
				n, err := strconv.ParseInt(digits.String(), base, 64)
				if err != nil {
					l.errs = append(l.errs, fmt.Errorf("%d:%d: invalid %s literal", line, baseCol+start, prefixName(prefix)))
					return
				}
				l.add(token.INT, strconv.FormatInt(n, 10), line, baseCol+start)
				continue
			}
			// Decimal literal with optional underscore separators.
			intDigits := strings.Builder{}
			intDigits.WriteByte(ch)
			i++
			for i < len(s) {
				c := s[i]
				if c == '_' {
					if i+1 >= len(s) || !isDigit(s[i+1]) {
						break
					}
					i++
					continue
				}
				if !isDigit(c) {
					break
				}
				intDigits.WriteByte(c)
				i++
			}
			typ := token.INT
			lexeme := intDigits.String()
			if i+1 < len(s) && s[i] == '.' && isDigit(s[i+1]) {
				typ = token.FLOAT
				lexeme += "."
				i++
				for i < len(s) {
					c := s[i]
					if c == '_' {
						if i+1 >= len(s) || !isDigit(s[i+1]) {
							break
						}
						i++
						continue
					}
					if !isDigit(c) {
						break
					}
					lexeme += string(c)
					i++
				}
			}
			l.add(typ, lexeme, line, baseCol+start)
			continue
		}
		if ch == '"' {
			var b strings.Builder
			i++
			for i < len(s) && s[i] != '"' {
				if s[i] == '{' && i+1 < len(s) && s[i+1] == '{' && !interp.LooksLikeDictExprStart(s, i) {
					b.WriteString("{{")
					i += 2
					continue
				}
				if s[i] == '{' {
					if close := interp.FindExprEnd(s, i); close >= 0 {
						if interp.LooksLikeDictExprStart(s, i) {
							b.WriteString("{(")
							b.WriteString(s[i+1 : close])
							b.WriteString(")}")
						} else {
							b.WriteString(s[i : close+1])
						}
						i = close + 1
						continue
					}
				}
				if s[i] == '\\' {
					if i+1 >= len(s) {
						l.diagErr("TYA-E0007", "Unterminated escape",
							"Backslash at end of string has no escape character.",
							"Add the escape character (e.g. \\n, \\t, \\\\).",
							line, baseCol+i, 1)
						return
					}
					switch s[i+1] {
					case 'n':
						b.WriteByte('\n')
					case 't':
						b.WriteByte('\t')
					case 'r':
						b.WriteByte('\r')
					case '"':
						b.WriteByte('"')
					case '\\':
						b.WriteByte('\\')
					default:
						l.diagErr("TYA-E0008", "Unknown escape",
							fmt.Sprintf("Unknown escape sequence \\%c.", s[i+1]),
							"Supported escapes: \\n, \\t, \\r, \\\", \\\\.",
							line, baseCol+i, 2)
						return
					}
					i += 2
					continue
				}
				b.WriteByte(s[i])
				i++
			}
			if i >= len(s) {
				l.diagErr("TYA-E0006", "Unterminated string",
					"This string literal has no closing quote.",
					`Add a closing " on the same line, or use """...""" for a multi-line string.`,
					line, col, 1)
				return
			}
			i++
			l.add(token.STRING, b.String(), line, col)
			continue
		}
		if ch == '\'' {
			start := col
			i++
			var raw strings.Builder
			for i < len(s) && s[i] != '\'' {
				if s[i] == '\\' {
					if i+1 >= len(s) {
						l.diagErr("TYA-E0007", "Unterminated escape",
							"Backslash at end of string has no escape character.",
							"Add the escape character (e.g. \\n, \\t, \\\\, \\').",
							line, baseCol+i, 1)
						return
					}
					raw.WriteByte(s[i])
					raw.WriteByte(s[i+1])
					i += 2
					continue
				}
				raw.WriteByte(s[i])
				i++
			}
			if i >= len(s) {
				l.diagErr("TYA-E0006", "Unterminated string",
					"This single-quoted string literal has no closing quote.",
					"Add a closing ' on the same line.",
					line, start, 1)
				return
			}
			i++
			value, err := interpretSingleQuotedEscapes(raw.String(), line, start+1)
			if err != nil {
				l.errs = append(l.errs, err)
				return
			}
			l.addString(token.STRING, value, line, start, "single", "", "")
			continue
		}
		if ch == '-' && i+1 < len(s) && s[i+1] == '>' {
			l.add(token.ARROW, "->", line, col)
			i += 2
			continue
		}
		if i+1 < len(s) {
			two := s[i : i+2]
			switch two {
			case "==":
				l.add(token.EQ, two, line, col)
				i += 2
				continue
			case "!=":
				l.add(token.NEQ, two, line, col)
				i += 2
				continue
			case "<=":
				l.add(token.LTE, two, line, col)
				i += 2
				continue
			case ">=":
				l.add(token.GTE, two, line, col)
				i += 2
				continue
			case "<<":
				l.add(token.SHL, two, line, col)
				i += 2
				continue
			case ">>":
				l.add(token.SHR, two, line, col)
				i += 2
				continue
			}
		}
		switch ch {
		case '=':
			l.add(token.ASSIGN, "=", line, col)
		case '<':
			l.add(token.LT, "<", line, col)
		case '>':
			l.add(token.GT, ">", line, col)
		case ':':
			l.add(token.COLON, ":", line, col)
		case ',':
			l.add(token.COMMA, ",", line, col)
		case '.':
			l.add(token.DOT, ".", line, col)
		case '?':
			l.add(token.QUESTION, "?", line, col)
		case '@':
			l.add(token.AT, "@", line, col)
		case '+':
			l.add(token.PLUS, "+", line, col)
		case '-':
			l.add(token.MINUS, "-", line, col)
		case '*':
			l.add(token.STAR, "*", line, col)
		case '/':
			l.add(token.SLASH, "/", line, col)
		case '%':
			l.add(token.PERCENT, "%", line, col)
		case '(':
			l.add(token.LPAREN, "(", line, col)
		case ')':
			l.add(token.RPAREN, ")", line, col)
		case '[':
			l.add(token.LBRACKET, "[", line, col)
		case ']':
			l.add(token.RBRACKET, "]", line, col)
		case '{':
			l.add(token.LBRACE, "{", line, col)
		case '}':
			l.add(token.RBRACE, "}", line, col)
		case '&':
			l.add(token.AMP, "&", line, col)
		case '|':
			l.add(token.PIPE, "|", line, col)
		case '^':
			l.add(token.CARET, "^", line, col)
		case '~':
			l.add(token.TILDE, "~", line, col)
		default:
			l.diagErr("TYA-E0015", "Unexpected character",
				fmt.Sprintf("Unexpected character %q.", ch),
				"Remove or replace this character.",
				line, col, 1)
		}
		i++
	}
}

// splitComment scans s for an unescaped `#` outside strings. When
// found, it returns the part before `#`, the comment text (without
// the `#` itself), the byte offset of the `#`, and true. When no
// comment is found, hasComment is false and stripped == s.
func splitComment(s string) (stripped, comment string, pos int, hasComment bool) {
	var quote byte
	escaped := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if quote != 0 && ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' || ch == '\'' {
			if quote == 0 {
				quote = ch
			} else if quote == ch {
				quote = 0
			}
		}
		if ch == '#' && quote == 0 {
			return s[:i], strings.TrimRight(s[i+1:], " \t"), i, true
		}
	}
	return s, "", 0, false
}

func stripComment(s string) string {
	var quote byte
	escaped := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if quote != 0 && ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' || ch == '\'' {
			if quote == 0 {
				quote = ch
			} else if quote == ch {
				quote = 0
			}
		}
		if ch == '#' && quote == 0 {
			return s[:i]
		}
	}
	return s
}

func isAlpha(b byte) bool { return b < unicode.MaxASCII && (unicode.IsLetter(rune(b))) }
func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func hexDigit(b byte) int {
	if b >= '0' && b <= '9' {
		return int(b - '0')
	}
	if b >= 'a' && b <= 'f' {
		return int(b-'a') + 10
	}
	if b >= 'A' && b <= 'F' {
		return int(b-'A') + 10
	}
	return -1
}
