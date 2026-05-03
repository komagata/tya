package parser

import (
	"fmt"
	"strconv"

	"tya/internal/ast"
	"tya/internal/token"
)

type Parser struct {
	toks              []token.Token
	pos               int
	suppressCommaFunc bool
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
	if p.at(token.IDENT) && p.peek().Lexeme == "for" {
		return p.forStmt()
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "break" {
		p.next()
		return &ast.BreakStmt{}, nil
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "continue" {
		p.next()
		return &ast.ContinueStmt{}, nil
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "return" {
		return p.returnStmt()
	}
	if p.isAssignStart() {
		tok := p.peek()
		targets, err := p.assignTargets()
		if err != nil {
			return nil, err
		}
		p.next()
		values, err := p.valuesAfterAssign()
		if err != nil {
			return nil, err
		}
		return &ast.AssignStmt{Targets: targets, Values: values, Tok: tok}, nil
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

func (p *Parser) forStmt() (ast.Stmt, error) {
	p.next()
	valueTok := p.expect(token.IDENT)
	if valueTok.Type != token.IDENT {
		return nil, p.err("expected loop variable")
	}
	valueName := valueTok.Lexeme
	var indexName string
	var indexTok token.Token
	if p.match(token.COMMA) {
		idx := p.expect(token.IDENT)
		if idx.Type != token.IDENT {
			return nil, p.err("expected loop index variable")
		}
		indexName = idx.Lexeme
		indexTok = idx
	}
	kind := ""
	if p.matchWord("in") {
		kind = "in"
	} else if p.matchWord("of") {
		kind = "of"
	} else {
		return nil, p.err("expected 'in' or 'of' in for loop")
	}
	iterable, err := p.expr()
	if err != nil {
		return nil, err
	}
	body, err := p.block("for")
	if err != nil {
		return nil, err
	}
	return &ast.ForInStmt{ValueName: valueName, IndexName: indexName, ValueTok: valueTok, IndexTok: indexTok, Kind: kind, Iterable: iterable, Body: body}, nil
}

func (p *Parser) returnStmt() (ast.Stmt, error) {
	p.next()
	if p.at(token.NEWLINE) || p.at(token.DEDENT) || p.at(token.EOF) {
		return &ast.ReturnStmt{}, nil
	}
	values, err := p.exprListLine()
	if err != nil {
		return nil, err
	}
	return &ast.ReturnStmt{Values: values}, nil
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

func (p *Parser) valuesAfterAssign() ([]ast.Expr, error) {
	if p.at(token.NEWLINE) && p.peekN(1).Type == token.INDENT {
		p.next()
		p.next()
		obj, err := p.objectBody()
		if err != nil {
			return nil, err
		}
		return []ast.Expr{obj}, nil
	}
	return p.exprListLine()
}

func (p *Parser) exprListLine() ([]ast.Expr, error) {
	first, err := p.exprLine()
	if err != nil {
		return nil, err
	}
	values := []ast.Expr{first}
	for p.match(token.COMMA) {
		next, err := p.exprLine()
		if err != nil {
			return nil, err
		}
		values = append(values, next)
	}
	return values, nil
}

func (p *Parser) objectBody() (*ast.DictLit, error) {
	dict := &ast.DictLit{}
	p.skipNewlines()
	for !p.at(token.DEDENT) && !p.at(token.EOF) {
		name := p.expect(token.IDENT)
		if name.Type != token.IDENT {
			return nil, p.err("expected dictionary key")
		}
		if !p.match(token.COLON) {
			return nil, p.err("expected ':' after dictionary key")
		}
		var val ast.Expr
		var err error
		if p.match(token.ARROW) {
			val, err = p.finishFunc(nil, nil)
		} else {
			val, err = p.exprLine()
		}
		if err != nil {
			return nil, err
		}
		dict.Props = append(dict.Props, ast.DictProp{Name: name.Lexeme, Tok: name, Value: val})
		p.skipNewlines()
	}
	if !p.match(token.DEDENT) {
		return nil, p.err("expected dedent after dictionary")
	}
	return dict, nil
}

func (p *Parser) exprLine() (ast.Expr, error) {
	ex, err := p.expr()
	if err != nil {
		return nil, err
	}
	if _, ok := ex.(*ast.FuncLit); ok {
		return ex, nil
	}
	var args []ast.Expr
	for p.startsExpr() {
		p.suppressCommaFunc = true
		arg, err := p.expr()
		p.suppressCommaFunc = false
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		p.match(token.COMMA)
	}
	if len(args) == 0 {
		return ex, nil
	}
	if id, ok := ex.(*ast.Ident); ok && isUnaryNoParenCall(id.Name) && len(args) > 1 {
		return &ast.CallExpr{
			Callee: ex,
			Args:   []ast.Expr{nestUnaryNoParen(args)},
		}, nil
	}
	if id, ok := ex.(*ast.Ident); ok && id.Name == "print" && len(args) > 1 {
		if first, ok := args[0].(*ast.Ident); ok && isUnaryNoParenCall(first.Name) {
			return &ast.CallExpr{
				Callee: ex,
				Args:   []ast.Expr{nestUnaryNoParen(args)},
			}, nil
		}
		return &ast.CallExpr{
			Callee: ex,
			Args:   []ast.Expr{&ast.CallExpr{Callee: args[0], Args: args[1:]}},
		}, nil
	}
	ex = &ast.CallExpr{Callee: ex, Args: args}
	return ex, nil
}

func isUnaryNoParenCall(name string) bool {
	switch name {
	case "len", "keys", "values", "trim", "to_string", "toString", "to_int", "toInt", "to_float", "toFloat", "to_number", "toNumber", "byte_len", "byteLen", "char_len", "charLen", "read_file", "readFile", "read_line", "readLine", "file_exists", "fileExists", "env", "error", "panic", "exit":
		return true
	default:
		return false
	}
}

func nestUnaryNoParen(exprs []ast.Expr) ast.Expr {
	if len(exprs) == 1 {
		return exprs[0]
	}
	return &ast.CallExpr{
		Callee: exprs[0],
		Args:   []ast.Expr{nestUnaryNoParen(exprs[1:])},
	}
}

func (p *Parser) expr() (ast.Expr, error) {
	left, err := p.logicOr()
	if err != nil {
		return nil, err
	}
	if p.match(token.ARROW) {
		var params []string
		var paramToks []token.Token
		if id, ok := left.(*ast.Ident); ok {
			params = []string{id.Name}
			paramToks = []token.Token{id.Tok}
		} else {
			return nil, p.err("function parameters must be identifiers")
		}
		return p.finishFunc(params, paramToks)
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
	if p.at(token.IDENT) && p.peek().Lexeme == "try" {
		tok := p.next()
		ex, err := p.unary()
		if err != nil {
			return nil, err
		}
		return &ast.TryExpr{Expr: ex, Tok: tok}, nil
	}
	if p.matchWord("not") || p.match(token.MINUS) {
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
			ex = &ast.MemberExpr{Object: ex, Name: name.Lexeme, NameTok: name}
			continue
		}
		if p.match(token.LPAREN) {
			var args []ast.Expr
			if !p.at(token.RPAREN) {
				for {
					p.suppressCommaFunc = true
					arg, err := p.expr()
					p.suppressCommaFunc = false
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
			return p.finishFunc([]string{tok.Lexeme}, []token.Token{tok})
		}
		switch tok.Lexeme {
		case "true":
			return &ast.BoolLit{Value: true}, nil
		case "false":
			return &ast.BoolLit{Value: false}, nil
		case "nil":
			return &ast.NilLit{}, nil
		}
		if !p.suppressCommaFunc && p.at(token.COMMA) && p.hasArrowBeforeLine() {
			p.next()
			params := []string{tok.Lexeme}
			paramToks := []token.Token{tok}
			for {
				next := p.expect(token.IDENT)
				if next.Type != token.IDENT {
					return nil, p.err("expected function parameter")
				}
				params = append(params, next.Lexeme)
				paramToks = append(paramToks, next)
				if !p.match(token.COMMA) {
					break
				}
			}
			if !p.match(token.ARROW) {
				return nil, p.err("expected '->' after function parameters")
			}
			return p.finishFunc(params, paramToks)
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
	case token.LPAREN:
		ex, err := p.expr()
		if err != nil {
			return nil, err
		}
		if !p.match(token.RPAREN) {
			return nil, p.err("expected ')'")
		}
		return ex, nil
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
	case token.LBRACE:
		return p.curlyLiteral()
	case token.AT:
		name := p.expect(token.IDENT)
		if name.Type != token.IDENT {
			return nil, p.err("expected property name after @")
		}
		return &ast.ThisProp{Name: name.Lexeme, Tok: tok}, nil
	}
	return nil, p.err("expected expression")
}

func (p *Parser) curlyLiteral() (ast.Expr, error) {
	if p.match(token.RBRACE) {
		return &ast.DictLit{}, nil
	}
	if p.at(token.IDENT) && p.peekN(1).Type == token.COLON {
		return p.dictLiteral()
	}
	return p.setLiteral()
}

func (p *Parser) dictLiteral() (ast.Expr, error) {
	dict := &ast.DictLit{}
	for {
		if !(p.at(token.IDENT) && p.peekN(1).Type == token.COLON) {
			return nil, p.err("cannot mix dict entries and set entries in one literal")
		}
		name := p.next()
		p.next()
		value, err := p.expr()
		if err != nil {
			return nil, err
		}
		dict.Props = append(dict.Props, ast.DictProp{Name: name.Lexeme, Tok: name, Value: value})
		if !p.match(token.COMMA) {
			break
		}
	}
	if !p.match(token.RBRACE) {
		return nil, p.err("expected '}'")
	}
	return dict, nil
}

func (p *Parser) setLiteral() (ast.Expr, error) {
	set := &ast.SetLit{}
	for {
		elem, err := p.expr()
		if err != nil {
			return nil, err
		}
		if p.at(token.COLON) {
			return nil, p.err("cannot mix dict entries and set entries in one literal")
		}
		set.Elems = append(set.Elems, elem)
		if !p.match(token.COMMA) {
			break
		}
	}
	if !p.match(token.RBRACE) {
		return nil, p.err("expected '}'")
	}
	return set, nil
}

func (p *Parser) finishFunc(params []string, paramToks []token.Token) (*ast.FuncLit, error) {
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
		return &ast.FuncLit{Params: params, ParamToks: paramToks, Body: body}, nil
	}
	ex, err := p.exprLine()
	if err != nil {
		return nil, err
	}
	return &ast.FuncLit{Params: params, ParamToks: paramToks, Expr: ex}, nil
}

func (p *Parser) assignTargets() ([]ast.Expr, error) {
	first, err := p.assignTarget()
	if err != nil {
		return nil, err
	}
	targets := []ast.Expr{first}
	for p.match(token.COMMA) {
		next, err := p.assignTarget()
		if err != nil {
			return nil, err
		}
		targets = append(targets, next)
	}
	return targets, nil
}

func (p *Parser) assignTarget() (ast.Expr, error) {
	if p.match(token.AT) {
		name := p.expect(token.IDENT)
		return &ast.ThisProp{Name: name.Lexeme, Tok: name}, nil
	}
	tok := p.expect(token.IDENT)
	ex := ast.Expr(&ast.Ident{Name: tok.Lexeme, Tok: tok})
	for {
		if p.match(token.DOT) {
			name := p.expect(token.IDENT)
			ex = &ast.MemberExpr{Object: ex, Name: name.Lexeme, NameTok: name}
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

func (p *Parser) isAssignStart() bool {
	i := p.pos
	if !p.scanAssignTarget(&i) {
		return false
	}
	for p.toks[i].Type == token.COMMA {
		i++
		if !p.scanAssignTarget(&i) {
			return false
		}
	}
	return p.toks[i].Type == token.ASSIGN
}

func (p *Parser) scanAssignTarget(i *int) bool {
	if p.toks[*i].Type == token.AT {
		*i += 2
		return true
	}
	if p.toks[*i].Type != token.IDENT {
		return false
	}
	*i++
	for {
		if p.toks[*i].Type == token.DOT {
			*i += 2
			continue
		}
		if p.toks[*i].Type == token.LBRACKET {
			depth := 1
			*i++
			for *i < len(p.toks) && depth > 0 {
				if p.toks[*i].Type == token.LBRACKET {
					depth++
				}
				if p.toks[*i].Type == token.RBRACKET {
					depth--
				}
				*i++
			}
			continue
		}
		break
	}
	return true
}

func (p *Parser) skipNewlines() {
	for p.match(token.NEWLINE) {
	}
}
func (p *Parser) startsExpr() bool {
	return p.at(token.IDENT) || p.at(token.STRING) || p.at(token.INT) || p.at(token.FLOAT) || p.at(token.AT) || p.at(token.LPAREN) || p.at(token.LBRACKET) || p.at(token.LBRACE)
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
