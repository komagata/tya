package lexer

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"tya/internal/token"
)

func prefixName(c byte) string {
	if c == 'x' || c == 'X' {
		return "hex"
	}
	return "binary"
}

type Lexer struct {
	src    string
	tokens []token.Token
	errs   []error
	ind    []int
}

func Lex(src string) ([]token.Token, []error) {
	l := &Lexer{src: strings.ReplaceAll(src, "\r\n", "\n"), ind: []int{0}}
	l.lex()
	return l.tokens, l.errs
}

func (l *Lexer) add(t token.Type, s string, line, col int) {
	l.tokens = append(l.tokens, token.Token{Type: t, Lexeme: s, Line: line, Col: col})
}

func (l *Lexer) lex() {
	lines := strings.Split(l.src, "\n")
	for i, raw := range lines {
		lineNo := i + 1
		line := strings.TrimRight(stripComment(raw), " ")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.Contains(line, "\t") {
			l.errs = append(l.errs, fmt.Errorf("%d: tabs are forbidden", lineNo))
			continue
		}
		if strings.TrimRight(line, " ") != line {
			l.errs = append(l.errs, fmt.Errorf("%d: trailing whitespace is forbidden", lineNo))
			continue
		}
		spaces := len(line) - len(strings.TrimLeft(line, " "))
		if spaces%2 != 0 {
			l.errs = append(l.errs, fmt.Errorf("%d: indentation must use exactly 2 spaces", lineNo))
			continue
		}
		l.handleIndent(spaces, lineNo)
		l.lexLine(strings.TrimLeft(line, " "), lineNo, spaces+1)
		l.add(token.NEWLINE, "", lineNo, len(raw)+1)
	}
	for len(l.ind) > 1 {
		l.ind = l.ind[:len(l.ind)-1]
		l.add(token.DEDENT, "", len(lines), 1)
	}
	l.add(token.EOF, "", len(lines), 1)
}

func (l *Lexer) handleIndent(n, line int) {
	cur := l.ind[len(l.ind)-1]
	if n > cur {
		if n != cur+2 {
			l.errs = append(l.errs, fmt.Errorf("%d: indentation may only increase by 2 spaces", line))
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
		l.errs = append(l.errs, fmt.Errorf("%d: inconsistent indentation", line))
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
		if ch == 'b' && i+1 < len(s) && s[i+1] == '"' {
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
					case '"':
						b.WriteByte('"')
					case '\\':
						b.WriteByte('\\')
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
				l.errs = append(l.errs, fmt.Errorf("%d:%d: unterminated string", line, col))
				return
			}
			i++
			l.add(token.STRING, b.String(), line, col)
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
			l.errs = append(l.errs, fmt.Errorf("%d:%d: unexpected character %q", line, col, ch))
		}
		i++
	}
}

func stripComment(s string) string {
	inString := false
	escaped := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if inString && ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
		}
		if ch == '#' && !inString {
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
