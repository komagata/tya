package parser

import (
	"fmt"
	"strconv"

	"tya/internal/ast"
	"tya/internal/token"
)

type Parser struct {
	toks []token.Token
	pos  int
}

func Parse(toks []token.Token) (*ast.Program, error) {
	p := &Parser{toks: toks}
	return p.program()
}

func (p *Parser) program() (*ast.Program, error) {
	var stmts []ast.Stmt
	p.skipNewlines()
	for !p.at(token.EOF) {
		s, err := p.stmt()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, s)
		p.skipNewlines()
	}
	return &ast.Program{Stmts: stmts}, nil
}

func (p *Parser) stmt() (ast.Stmt, error) {
	if p.at(token.IDENT) && p.peek().Lexeme == "if" {
		return p.ifStmt()
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "while" {
		return p.whileStmt()
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "break" {
		p.next()
		return &ast.BreakStmt{}, nil
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "continue" {
		p.next()
		return &ast.ContinueStmt{}, nil
	}
	if p.isAssignStart() {
		tok := p.peek()
		target, err := p.assignTarget()
		if err != nil {
			return nil, err
		}
		p.next()
		value, err := p.valueAfterAssign()
		if err != nil {
			return nil, err
		}
		return &ast.AssignStmt{Target: target, Value: value, Tok: tok}, nil
	}
	ex, err := p.exprLine()
	if err != nil {
		return nil, err
	}
	return &ast.ExprStmt{Expr: ex}, nil
}

func (p *Parser) ifStmt() (ast.Stmt, error) {
	p.next()
	cond, err := p.expr()
	if err != nil {
		return nil, err
	}
	thenBlock, err := p.block("if")
	if err != nil {
		return nil, err
	}
	var elseBlock []ast.Stmt
	p.skipNewlines()
	if p.at(token.IDENT) && p.peek().Lexeme == "else" {
		p.next()
		elseBlock, err = p.block("else")
		if err != nil {
			return nil, err
		}
	}
	return &ast.IfStmt{Cond: cond, Then: thenBlock, Else: elseBlock}, nil
}

func (p *Parser) whileStmt() (ast.Stmt, error) {
	p.next()
	cond, err := p.expr()
	if err != nil {
		return nil, err
	}
	body, err := p.block("while")
	if err != nil {
		return nil, err
	}
	return &ast.WhileStmt{Cond: cond, Body: body}, nil
}

func (p *Parser) block(owner string) ([]ast.Stmt, error) {
	if !p.match(token.NEWLINE) || !p.match(token.INDENT) {
		return nil, p.err("expected indented block after " + owner)
	}
	var stmts []ast.Stmt
	p.skipNewlines()
	for !p.at(token.DEDENT) && !p.at(token.EOF) {
		s, err := p.stmt()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, s)
		p.skipNewlines()
	}
	if !p.match(token.DEDENT) {
		return nil, p.err("expected dedent after " + owner)
	}
	return stmts, nil
}

func (p *Parser) valueAfterAssign() (ast.Expr, error) {
	if p.at(token.NEWLINE) && p.peekN(1).Type == token.INDENT {
		p.next()
		p.next()
		obj, err := p.objectBody()
		if err != nil {
			return nil, err
		}
		return obj, nil
	}
	return p.exprLine()
}

func (p *Parser) objectBody() (*ast.ObjectLit, error) {
	obj := &ast.ObjectLit{}
	p.skipNewlines()
	for !p.at(token.DEDENT) && !p.at(token.EOF) {
		name := p.expect(token.IDENT)
		if name.Type != token.IDENT {
			return nil, p.err("expected object property name")
		}
		if !p.match(token.COLON) {
			return nil, p.err("expected ':' after object property")
		}
		var val ast.Expr
		var err error
		if p.match(token.ARROW) {
			val, err = p.finishFunc(nil)
		} else {
			val, err = p.exprLine()
		}
		if err != nil {
			return nil, err
		}
		obj.Props = append(obj.Props, ast.ObjectProp{Name: name.Lexeme, Value: val})
		p.skipNewlines()
	}
	if !p.match(token.DEDENT) {
		return nil, p.err("expected dedent after object")
	}
	return obj, nil
}

func (p *Parser) exprLine() (ast.Expr, error) {
	ex, err := p.expr()
	if err != nil {
		return nil, err
	}
	var args []ast.Expr
	for p.startsExpr() {
		arg, err := p.expr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		p.match(token.COMMA)
	}
	if len(args) == 0 {
		return ex, nil
	}
	if id, ok := ex.(*ast.Ident); ok && id.Name == "print" && len(args) > 1 {
		return &ast.CallExpr{
			Callee: ex,
			Args:   []ast.Expr{&ast.CallExpr{Callee: args[0], Args: args[1:]}},
		}, nil
	}
	ex = &ast.CallExpr{Callee: ex, Args: args}
	return ex, nil
}

func (p *Parser) expr() (ast.Expr, error) {
	left, err := p.logicOr()
	if err != nil {
		return nil, err
	}
	if p.match(token.ARROW) {
		var params []string
		if id, ok := left.(*ast.Ident); ok {
			params = []string{id.Name}
		} else {
			return nil, p.err("function parameters must be identifiers")
		}
		return p.finishFunc(params)
	}
	return left, nil
}

func (p *Parser) logicOr() (ast.Expr, error) {
	left, err := p.logicAnd()
	if err != nil {
		return nil, err
	}
	for p.matchWord("or") {
		op := p.prev()
		right, err := p.logicAnd()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) logicAnd() (ast.Expr, error) {
	left, err := p.equality()
	if err != nil {
		return nil, err
	}
	for p.matchWord("and") {
		op := p.prev()
		right, err := p.equality()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) equality() (ast.Expr, error) {
	left, err := p.comparison()
	if err != nil {
		return nil, err
	}
	for p.match(token.EQ) || p.match(token.NEQ) {
		op := p.prev()
		right, err := p.comparison()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) comparison() (ast.Expr, error) {
	left, err := p.term()
	if err != nil {
		return nil, err
	}
	for p.match(token.LT) || p.match(token.LTE) || p.match(token.GT) || p.match(token.GTE) {
		op := p.prev()
		right, err := p.term()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) term() (ast.Expr, error) {
	left, err := p.factor()
	if err != nil {
		return nil, err
	}
	for p.match(token.PLUS) || p.match(token.MINUS) {
		op := p.prev()
		right, err := p.factor()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) factor() (ast.Expr, error) {
	left, err := p.unary()
	if err != nil {
		return nil, err
	}
	for p.match(token.STAR) || p.match(token.SLASH) || p.match(token.PERCENT) {
		op := p.prev()
		right, err := p.unary()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) unary() (ast.Expr, error) {
	if p.matchWord("not") {
		op := p.prev()
		ex, err := p.unary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Op: op, Expr: ex}, nil
	}
	return p.call()
}

func (p *Parser) call() (ast.Expr, error) {
	ex, err := p.primary()
	if err != nil {
		return nil, err
	}
	for {
		if p.match(token.DOT) {
			name := p.expect(token.IDENT)
			if name.Type != token.IDENT {
				return nil, p.err("expected property name")
			}
			ex = &ast.MemberExpr{Object: ex, Name: name.Lexeme}
			continue
		}
		if p.match(token.LPAREN) {
			var args []ast.Expr
			if !p.at(token.RPAREN) {
				for {
					arg, err := p.expr()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
					if !p.match(token.COMMA) {
						break
					}
				}
			}
			if !p.match(token.RPAREN) {
				return nil, p.err("expected ')'")
			}
			ex = &ast.CallExpr{Callee: ex, Args: args}
			continue
		}
		if p.match(token.LBRACKET) {
			index, err := p.expr()
			if err != nil {
				return nil, err
			}
			if !p.match(token.RBRACKET) {
				return nil, p.err("expected ']'")
			}
			ex = &ast.IndexExpr{Object: ex, Index: index}
			continue
		}
		break
	}
	return ex, nil
}

func (p *Parser) primary() (ast.Expr, error) {
	tok := p.next()
	switch tok.Type {
	case token.IDENT:
		if p.match(token.ARROW) {
			return p.finishFunc([]string{tok.Lexeme})
		}
		switch tok.Lexeme {
		case "true":
			return &ast.BoolLit{Value: true}, nil
		case "false":
			return &ast.BoolLit{Value: false}, nil
		case "nil":
			return &ast.NilLit{}, nil
		}
		if p.at(token.COMMA) && p.hasArrowBeforeLine() {
			p.next()
			params := []string{tok.Lexeme}
			for {
				next := p.expect(token.IDENT)
				if next.Type != token.IDENT {
					return nil, p.err("expected function parameter")
				}
				params = append(params, next.Lexeme)
				if !p.match(token.COMMA) {
					break
				}
			}
			if !p.match(token.ARROW) {
				return nil, p.err("expected '->' after function parameters")
			}
			return p.finishFunc(params)
		}
		return &ast.Ident{Name: tok.Lexeme, Tok: tok}, nil
	case token.INT:
		v, _ := strconv.ParseInt(tok.Lexeme, 10, 64)
		return &ast.IntLit{Value: v}, nil
	case token.FLOAT:
		v, _ := strconv.ParseFloat(tok.Lexeme, 64)
		return &ast.FloatLit{Value: v}, nil
	case token.STRING:
		return &ast.StringLit{Value: tok.Lexeme}, nil
	case token.LBRACKET:
		var elems []ast.Expr
		if !p.at(token.RBRACKET) {
			for {
				elem, err := p.expr()
				if err != nil {
					return nil, err
				}
				elems = append(elems, elem)
				if !p.match(token.COMMA) {
					break
				}
			}
		}
		if !p.match(token.RBRACKET) {
			return nil, p.err("expected ']'")
		}
		return &ast.ArrayLit{Elems: elems}, nil
	case token.AT:
		name := p.expect(token.IDENT)
		if name.Type != token.IDENT {
			return nil, p.err("expected property name after @")
		}
		return &ast.ThisProp{Name: name.Lexeme, Tok: tok}, nil
	}
	return nil, p.err("expected expression")
}

func (p *Parser) finishFunc(params []string) (*ast.FuncLit, error) {
	if p.at(token.NEWLINE) && p.peekN(1).Type == token.INDENT {
		p.next()
		p.next()
		var body []ast.Stmt
		p.skipNewlines()
		for !p.at(token.DEDENT) && !p.at(token.EOF) {
			s, err := p.stmt()
			if err != nil {
				return nil, err
			}
			body = append(body, s)
			p.skipNewlines()
		}
		if !p.match(token.DEDENT) {
			return nil, p.err("expected dedent after function")
		}
		return &ast.FuncLit{Params: params, Body: body}, nil
	}
	ex, err := p.exprLine()
	if err != nil {
		return nil, err
	}
	return &ast.FuncLit{Params: params, Expr: ex}, nil
}

func (p *Parser) assignTarget() (ast.Expr, error) {
	if p.match(token.AT) {
		name := p.expect(token.IDENT)
		return &ast.ThisProp{Name: name.Lexeme, Tok: name}, nil
	}
	tok := p.expect(token.IDENT)
	ex := ast.Expr(&ast.Ident{Name: tok.Lexeme, Tok: tok})
	for p.match(token.DOT) {
		name := p.expect(token.IDENT)
		ex = &ast.MemberExpr{Object: ex, Name: name.Lexeme}
	}
	return ex, nil
}

func (p *Parser) isAssignStart() bool {
	i := p.pos
	if p.toks[i].Type == token.AT {
		i += 2
	} else if p.toks[i].Type == token.IDENT {
		i++
		for p.toks[i].Type == token.DOT {
			i += 2
		}
	} else {
		return false
	}
	return p.toks[i].Type == token.ASSIGN
}

func (p *Parser) skipNewlines() {
	for p.match(token.NEWLINE) {
	}
}
func (p *Parser) startsExpr() bool {
	return p.at(token.IDENT) || p.at(token.STRING) || p.at(token.INT) || p.at(token.FLOAT) || p.at(token.AT) || p.at(token.LBRACKET)
}
func (p *Parser) at(t token.Type) bool { return p.peek().Type == t }
func (p *Parser) match(t token.Type) bool {
	if p.at(t) {
		p.next()
		return true
	}
	return false
}
func (p *Parser) matchWord(word string) bool {
	if p.at(token.IDENT) && p.peek().Lexeme == word {
		p.next()
		return true
	}
	return false
}
func (p *Parser) next() token.Token {
	tok := p.peek()
	if p.pos < len(p.toks)-1 {
		p.pos++
	}
	return tok
}
func (p *Parser) prev() token.Token { return p.toks[p.pos-1] }
func (p *Parser) peek() token.Token { return p.toks[p.pos] }
func (p *Parser) peekN(n int) token.Token {
	if p.pos+n >= len(p.toks) {
		return p.toks[len(p.toks)-1]
	}
	return p.toks[p.pos+n]
}
func (p *Parser) hasArrowBeforeLine() bool {
	for i := p.pos; i < len(p.toks); i++ {
		switch p.toks[i].Type {
		case token.ARROW:
			return true
		case token.NEWLINE, token.EOF, token.RPAREN, token.RBRACKET:
			return false
		}
	}
	return false
}
func (p *Parser) expect(t token.Type) token.Token {
	if p.at(t) {
		return p.next()
	}
	return token.Token{}
}
func (p *Parser) err(msg string) error {
	tok := p.peek()
	return fmt.Errorf("%d:%d: %s near %q", tok.Line, tok.Col, msg, tok.Lexeme)
}
