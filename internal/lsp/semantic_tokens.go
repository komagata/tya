package lsp

import (
	"sort"
	"strings"

	"tya/internal/lexer"
	"tya/internal/token"
)

// Semantic token types advertised in the server capabilities. The
// indices in this slice are the legend values written into
// SemanticTokens.Data.
var semanticTokenTypes = []string{
	"keyword",   // 0
	"variable",  // 1
	"string",    // 2
	"number",    // 3
	"comment",   // 4
	"operator",  // 5
	"function",  // 6
	"class",     // 7
	"namespace", // 8 (used for module / interface names)
}

// SemanticTokensLegendValue is the constant legend returned in
// initialize results.
func SemanticTokensLegendValue() SemanticTokensLegend {
	return SemanticTokensLegend{TokenTypes: semanticTokenTypes, TokenModifiers: []string{"readonly", "deprecated", "definition", "defaultLibrary"}}
}

const (
	stKeyword    = 0
	stVariable   = 1
	stString     = 2
	stNumber     = 3
	stComment    = 4
	stOperator   = 5
	stFunction   = 6
	stClass      = 7
	stNamespace  = 8
	stUnassigned = -1
)

// keywordSet is the lookup table used by semantic-token classification.
var keywordSet = func() map[string]bool {
	m := map[string]bool{}
	for _, k := range Keywords() {
		m[k] = true
	}
	return m
}()

type semanticEvent struct {
	line   int // 0-origin
	col    int // 0-origin
	length int
	kind   int
	mods   uint32
}

// SemanticTokensFor returns the v0.53 semantic token payload for
// src. tokens that have unknown classification are dropped.
func SemanticTokensFor(src string) SemanticTokens {
	toks, comments, _ := lexer.LexWithComments(src)
	events := []semanticEvent{}
	prog := parseOrNil(src)
	idx := BuildSymbols(prog)
	for i, t := range toks {
		kind := classifyToken(t, toks, i, idx)
		if kind == stUnassigned {
			continue
		}
		length := lexemeLength(t)
		if length == 0 {
			continue
		}
		events = append(events, semanticEvent{
			line:   t.Line - 1,
			col:    t.Col - 1,
			length: length,
			kind:   kind,
			mods:   semanticModifiers(t, idx),
		})
	}
	for _, c := range comments {
		events = append(events, semanticEvent{
			line:   c.Line - 1,
			col:    c.Col - 1,
			length: len(c.Text) + 1, // include the leading "#"
			kind:   stComment,
		})
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].line != events[j].line {
			return events[i].line < events[j].line
		}
		return events[i].col < events[j].col
	})
	return SemanticTokens{Data: deltaEncode(events)}
}

func classifyToken(t token.Token, all []token.Token, i int, idx *SymbolIndex) int {
	switch t.Type {
	case token.STRING:
		return stString
	case token.INT, token.FLOAT:
		return stNumber
	case token.IDENT:
		if keywordSet[t.Lexeme] {
			return stKeyword
		}
		if idx != nil {
			if sym, ok := idx.Lookup(t.Lexeme); ok {
				switch sym.Kind {
				case "class":
					return stClass
				case "struct":
					return stClass
				case "record":
					return stClass
				case "module":
					return stNamespace
				case "interface":
					return stNamespace
				case "function":
					return stFunction
				}
			}
		}
		// Heuristic: ident immediately followed by `(` is a call.
		for j := i + 1; j < len(all); j++ {
			next := all[j]
			if next.Type == token.NEWLINE || next.Type == token.INDENT || next.Type == token.DEDENT {
				continue
			}
			if next.Type == token.LPAREN {
				return stFunction
			}
			break
		}
		return stVariable
	case token.NEWLINE, token.INDENT, token.DEDENT, token.EOF, token.ILLEGAL:
		return stUnassigned
	}
	if isOperatorToken(t.Type) {
		return stOperator
	}
	return stUnassigned
}

func isOperatorToken(t token.Type) bool {
	switch t {
	case token.ASSIGN, token.NIL_ASSIGN, token.EQ, token.NEQ, token.LT, token.LTE, token.GT, token.GTE,
		token.PLUS, token.MINUS, token.STAR, token.SLASH, token.PERCENT, token.ARROW,
		token.AMP, token.PIPE, token.CARET, token.TILDE, token.SHL, token.SHR,
		token.COLON, token.COMMA, token.DOT, token.QUESTION:
		return true
	}
	return false
}

func lexemeLength(t token.Token) int {
	if t.Lexeme != "" {
		return len(t.Lexeme)
	}
	// Punctuation lexemes are the same as their literal Type.
	if s := string(t.Type); !strings.HasPrefix(s, "IDENT") && len(s) < 4 {
		return len(s)
	}
	return 0
}

// deltaEncode renders the LSP-required 5-tuple-per-token format:
// [deltaLine, deltaStart, length, tokenType, tokenModifiers].
func deltaEncode(events []semanticEvent) []uint32 {
	out := make([]uint32, 0, len(events)*5)
	prevLine, prevCol := 0, 0
	for _, e := range events {
		dLine := e.line - prevLine
		dCol := e.col
		if dLine == 0 {
			dCol = e.col - prevCol
		}
		out = append(out, uint32(dLine), uint32(dCol), uint32(e.length), uint32(e.kind), e.mods)
		prevLine = e.line
		prevCol = e.col
	}
	return out
}

func semanticModifiers(t token.Token, idx *SymbolIndex) uint32 {
	if idx != nil {
		if sym, ok := idx.Lookup(t.Lexeme); ok && sym.NameTok.Line == t.Line && sym.NameTok.Col == t.Col {
			return 1 << 2 // definition
		}
	}
	return 0
}
