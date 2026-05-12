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
