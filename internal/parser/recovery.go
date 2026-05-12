package parser

import "tya/internal/token"

// skipToNextStmt advances past tokens until a plausible statement
// boundary is reached. Used by block-level recovery to keep
// parsing after a per-statement failure.
//
// Boundaries: NEWLINE / DEDENT / EOF. The function consumes the
// boundary token itself when it is NEWLINE so the next stmt()
// call sees a clean indent.
func (p *Parser) skipToNextStmt() {
	steps := 0
	for !p.at(token.EOF) {
		if steps > 2048 {
			// Defensive: never spin forever on pathological input.
			return
		}
		steps++
		if p.at(token.DEDENT) {
			return
		}
		if p.at(token.NEWLINE) {
			p.next()
			return
		}
		p.next()
	}
}

// skipToCommaOrClose advances past tokens until a `,` or one of
// the supplied close-bracket types is reached. It does NOT
// consume the boundary token — callers (CallExpr / ArrayLit /
// DictLit arg loops) inspect it to decide whether to continue
// the next element or finish the list.
//
// Nested `()` / `[]` / `{}` are skipped over so an inner stray
// `,` or `)` cannot fool the recovery into stopping early.
// NEWLINE / DEDENT / EOF also halt — a half-typed expression
// terminated by a statement boundary should not eat the next
// statement.
func (p *Parser) skipToCommaOrClose(closes ...token.Type) {
	steps := 0
	for !p.at(token.EOF) {
		if steps > 2048 {
			return
		}
		steps++
		if p.at(token.COMMA) || p.at(token.NEWLINE) || p.at(token.DEDENT) {
			return
		}
		for _, c := range closes {
			if p.at(c) {
				return
			}
		}
		switch p.peek().Type {
		case token.LPAREN:
			p.next()
			p.skipBalanced(token.LPAREN, token.RPAREN)
		case token.LBRACKET:
			p.next()
			p.skipBalanced(token.LBRACKET, token.RBRACKET)
		case token.LBRACE:
			p.next()
			p.skipBalanced(token.LBRACE, token.RBRACE)
		default:
			p.next()
		}
	}
}

// skipBalanced eats tokens up to and including the matching close
// bracket. Used by skipToCommaOrClose to step over nested
// `( ... )`, `[ ... ]`, and `{ ... }` chunks so a comma inside
// one of them does not stop outer-list recovery.
func (p *Parser) skipBalanced(open, close token.Type) {
	depth := 1
	steps := 0
	for !p.at(token.EOF) && depth > 0 {
		if steps > 4096 {
			return
		}
		steps++
		switch p.peek().Type {
		case open:
			depth++
		case close:
			depth--
		}
		p.next()
	}
}
