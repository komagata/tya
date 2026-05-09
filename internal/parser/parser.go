package parser

import (
	"fmt"
	"strconv"
	"strings"

	"tya/internal/ast"
	"tya/internal/token"
)

type Parser struct {
	toks          []token.Token
	pos           int
	blockDepth    int
	loopDepth     int
	functionDepth int
	classDepth    int
}

func Parse(toks []token.Token) (*ast.Program, error) {
	p := &Parser{toks: toks}
	return p.program()
}

// ParseWithComments runs Parse and additionally fills
// Program.HeaderComments and Program.Comments from the supplied
// comments slice. Per docs/CANONICAL_SYNTAX.md §3.1–§3.3:
//
//   - Leading comments: contiguous `#` lines immediately before a
//     statement, at the same indent as that statement.
//   - Line-end comments: a single non-full-line `#` comment on the
//     same source line as the statement's start.
//   - Header comments: contiguous full-line `#` lines starting at
//     line 1 at indent 0, separated from the body by exactly one
//     blank line.
func ParseWithComments(toks []token.Token, comments []CommentInfo) (*ast.Program, error) {
	prog, err := Parse(toks)
	if err != nil {
		return nil, err
	}
	if len(comments) == 0 {
		return prog, nil
	}
	attachStmtComments(prog, comments)
	firstStmtLine := 0
	for _, t := range toks {
		if t.Type == token.NEWLINE || t.Type == token.INDENT || t.Type == token.DEDENT || t.Type == token.EOF {
			continue
		}
		firstStmtLine = t.Line
		break
	}
	header := []string{}
	lastHeaderLine := 0
	for _, c := range comments {
		if !c.IsFullLine {
			continue
		}
		if c.Indent != 0 {
			continue
		}
		if firstStmtLine > 0 && c.Line >= firstStmtLine {
			break
		}
		// Contiguous block from line 1: each comment's line must
		// follow the previous one with no gap, or start at line 1.
		if len(header) == 0 {
			if c.Line != 1 {
				continue
			}
		} else if c.Line != lastHeaderLine+1 {
			break
		}
		header = append(header, c.Text)
		lastHeaderLine = c.Line
	}
	if len(header) > 0 && firstStmtLine > 0 {
		// Require a blank line between the header block and the body.
		if firstStmtLine == lastHeaderLine+1 {
			return prog, nil
		}
	}
	prog.HeaderComments = header
	return prog, nil
}

// CommentInfo mirrors lexer.Comment to avoid an import cycle. The
// CLI converts before calling.
type CommentInfo struct {
	Line       int
	Col        int
	Indent     int
	Text       string
	IsFullLine bool
}

// attachStmtComments walks top-level statements and attaches leading
// and line-end comments per §3.1 / §3.2.
func attachStmtComments(prog *ast.Program, comments []CommentInfo) {
	if prog == nil || len(prog.Stmts) == 0 {
		return
	}
	prog.Comments = map[ast.Stmt]ast.StmtComments{}
	used := make([]bool, len(comments))
	for i := 0; i < len(prog.Stmts); i++ {
		stmt := prog.Stmts[i]
		startLine, _ := stmtPos(stmt)
		if startLine == 0 {
			continue
		}
		// Top-level statements always have indent 0; CANONICAL §3.1
		// requires leading comments at the same indent. Inner stmts
		// (inside if/while/match/module bodies) are handled in a
		// later release.
		indent := 0
		// Leading: walk backward from comments preceding startLine.
		var leading []string
		expectedLine := startLine - 1
		for j := len(comments) - 1; j >= 0; j-- {
			if used[j] {
				continue
			}
			c := comments[j]
			if c.Line >= startLine {
				continue
			}
			if c.Line < expectedLine {
				break
			}
			if !c.IsFullLine {
				break
			}
			if c.Indent != indent {
				break
			}
			leading = append([]string{c.Text}, leading...)
			used[j] = true
			expectedLine = c.Line - 1
		}
		// Line-end: a non-full-line comment whose Line == startLine.
		var lineEnd string
		for j := range comments {
			if used[j] {
				continue
			}
			c := comments[j]
			if c.Line != startLine {
				continue
			}
			if c.IsFullLine {
				continue
			}
			lineEnd = c.Text
			used[j] = true
			break
		}
		if len(leading) == 0 && lineEnd == "" {
			continue
		}
		prog.Comments[stmt] = ast.StmtComments{Leading: leading, LineEnd: lineEnd}
	}
}

// stmtPos returns the source line and indent (column - 1) of stmt's
// first significant token, or (0, 0) when stmt does not carry a
// position.
func stmtPos(stmt ast.Stmt) (line, indent int) {
	switch n := stmt.(type) {
	case *ast.AssignStmt:
		return n.Tok.Line, n.Tok.Col - 1
	case *ast.ImportStmt:
		return n.NameTok.Line, n.NameTok.Col - 1
	case *ast.ModuleDecl:
		return n.NameTok.Line, n.NameTok.Col - 1
	case *ast.ClassDecl:
		return n.NameTok.Line, n.NameTok.Col - 1
	case *ast.InterfaceDecl:
		return n.NameTok.Line, n.NameTok.Col - 1
	case *ast.ReturnStmt:
		return n.Tok.Line, n.Tok.Col - 1
	case *ast.RaiseStmt:
		return n.Tok.Line, n.Tok.Col - 1
	case *ast.MatchStmt:
		return n.Tok.Line, n.Tok.Col - 1
	case *ast.TryCatchStmt:
		return n.Tok.Line, n.Tok.Col - 1
	case *ast.ForInStmt:
		return n.ValueTok.Line, n.ValueTok.Col - 1
	}
	return 0, 0
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
	if p.at(token.IDENT) && p.peek().Lexeme == "module" {
		if p.blockDepth != 0 {
			return nil, p.err("module must be top-level")
		}
		return p.moduleDecl()
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "interface" {
		if p.blockDepth != 0 {
			return nil, p.err("interface must be top-level")
		}
		return p.interfaceDecl()
	}
	if p.startsClassDecl() {
		if p.blockDepth != 0 {
			return nil, p.err("class must be top-level")
		}
		return p.classDecl()
	}
	if err := p.rejectV01ExcludedIdent(); err != nil {
		return nil, err
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "import" {
		if p.blockDepth != 0 {
			return nil, p.err("import must be top-level")
		}
		return p.importStmt()
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "if" {
		return p.ifStmt()
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "while" {
		return p.whileStmt()
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "for" {
		return p.forStmt()
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "match" {
		return p.matchStmt()
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "case" {
		return nil, p.err("case outside match")
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "try" && p.peekN(1).Type == token.NEWLINE {
		return p.tryCatchStmt()
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "catch" {
		return nil, p.err("catch outside block try")
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "raise" {
		return p.raiseStmt()
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "break" {
		if p.loopDepth == 0 {
			return nil, p.err("break must be inside a loop")
		}
		p.next()
		return &ast.BreakStmt{}, nil
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "continue" {
		if p.loopDepth == 0 {
			return nil, p.err("continue must be inside a loop")
		}
		p.next()
		return &ast.ContinueStmt{}, nil
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "return" {
		if p.functionDepth == 0 {
			return nil, p.err("return must be inside a function")
		}
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
	ex, err := p.stmtExprLine()
	if err != nil {
		return nil, err
	}
	return &ast.ExprStmt{Expr: ex}, nil
}

func (p *Parser) importStmt() (ast.Stmt, error) {
	p.next()
	name, err := p.expectName("expected module name after import")
	if err != nil {
		return nil, err
	}
	parts := []string{name.Lexeme}
	for p.match(token.SLASH) {
		seg, err := p.expectName("expected module path segment after '/'")
		if err != nil {
			return nil, err
		}
		parts = append(parts, seg.Lexeme)
	}
	var alias string
	var aliasTok token.Token
	if p.at(token.IDENT) && p.peek().Lexeme == "as" {
		p.next()
		tok, err := p.expectName("expected import alias after as")
		if err != nil {
			return nil, err
		}
		alias = tok.Lexeme
		aliasTok = tok
	}
	if !p.at(token.NEWLINE) && !p.at(token.DEDENT) && !p.at(token.EOF) {
		return nil, p.err("expected newline after import")
	}
	return &ast.ImportStmt{Name: strings.Join(parts, "/"), NameTok: name, Alias: alias, AliasTok: aliasTok}, nil
}

func (p *Parser) moduleDecl() (ast.Stmt, error) {
	p.next()
	name, err := p.expectName("expected module name")
	if err != nil {
		return nil, err
	}
	if !p.match(token.NEWLINE) || !p.match(token.INDENT) {
		return nil, p.err("expected indented block after module")
	}
	decl := &ast.ModuleDecl{Name: name.Lexeme, NameTok: name}
	p.skipNewlines()
	for !p.at(token.DEDENT) && !p.at(token.EOF) {
		if p.at(token.IDENT) && p.peek().Lexeme == "interface" {
			iface, err := p.interfaceDecl()
			if err != nil {
				return nil, err
			}
			decl.Interfaces = append(decl.Interfaces, iface.(*ast.InterfaceDecl))
			p.skipNewlines()
			continue
		}
		if p.startsClassDecl() {
			cls, err := p.classDecl()
			if err != nil {
				return nil, err
			}
			decl.Classes = append(decl.Classes, cls.(*ast.ClassDecl))
			p.skipNewlines()
			continue
		}
		memberName, err := p.expectCallableName("expected module member name")
		if err != nil {
			return nil, err
		}
		if !p.match(token.ASSIGN) {
			return nil, p.err("expected '=' after module member name")
		}
		value, err := p.exprLine()
		if err != nil {
			return nil, err
		}
		decl.Members = append(decl.Members, ast.ModuleMember{Name: memberName.Lexeme, Tok: memberName, Value: value})
		p.skipNewlines()
	}
	if !p.match(token.DEDENT) {
		return nil, p.err("expected dedent after module")
	}
	return decl, nil
}

func (p *Parser) classDecl() (ast.Stmt, error) {
	abstract := false
	final := false
	for p.at(token.IDENT) && (p.peek().Lexeme == "abstract" || p.peek().Lexeme == "final") {
		if p.peek().Lexeme == "abstract" {
			abstract = true
		} else {
			final = true
		}
		p.next()
	}
	if !p.matchWord("class") {
		return nil, p.err("expected class")
	}
	name, err := p.expectName("expected class name")
	if err != nil {
		return nil, err
	}
	var parent *ast.ClassRef
	if p.at(token.IDENT) && p.peek().Lexeme == "extends" {
		p.next()
		parent, err = p.classRef()
		if err != nil {
			return nil, err
		}
		if p.match(token.COMMA) {
			return nil, p.err("multiple inheritance is not in Tya v0.7")
		}
	}
	var implements []ast.ClassRef
	if p.at(token.IDENT) && p.peek().Lexeme == "implements" {
		p.next()
		for {
			ref, err := p.classRef()
			if err != nil {
				return nil, err
			}
			implements = append(implements, *ref)
			if !p.match(token.COMMA) {
				break
			}
		}
	}
	if !p.match(token.NEWLINE) || !p.match(token.INDENT) {
		return nil, p.err("expected indented block after class")
	}
	p.blockDepth++
	p.classDepth++
	defer func() {
		p.classDepth--
		p.blockDepth--
	}()
	decl := &ast.ClassDecl{Name: name.Lexeme, NameTok: name, Parent: parent, Implements: implements, Abstract: abstract, Final: final}
	p.skipNewlines()
	for !p.at(token.DEDENT) && !p.at(token.EOF) {
		isAbstractMethod := false
		isOverrideMethod := false
		if p.at(token.IDENT) && p.peek().Lexeme == "abstract" {
			isAbstractMethod = true
			p.next()
		}
		if p.at(token.IDENT) && p.peek().Lexeme == "override" {
			isOverrideMethod = true
			p.next()
		}
		if isAbstractMethod && isOverrideMethod {
			return nil, p.err("method cannot be both abstract and override")
		}
		isClassMember := p.match(token.AT)
		if isClassMember {
			if !p.match(token.AT) {
				return nil, p.err("expected '@' for class member")
			}
		}
		memberName, err := p.expectCallableName("expected class member name")
		if err != nil {
			return nil, err
		}
		if !p.match(token.ASSIGN) {
			return nil, p.err("expected '=' after class member name")
		}
		if isAbstractMethod {
			params, paramToks, err := p.abstractMethodParams()
			if err != nil {
				return nil, err
			}
			decl.Methods = append(decl.Methods, ast.ClassMethod{Name: memberName.Lexeme, Tok: memberName, Func: &ast.FuncLit{Params: params, ParamToks: paramToks}, Class: isClassMember, Abstract: true})
			p.skipNewlines()
			continue
		}
		value, err := p.exprLine()
		if err != nil {
			return nil, err
		}
		if funcLit, ok := value.(*ast.FuncLit); ok {
			decl.Methods = append(decl.Methods, ast.ClassMethod{Name: memberName.Lexeme, Tok: memberName, Func: funcLit, Class: isClassMember, Override: isOverrideMethod})
		} else if isOverrideMethod {
			return nil, p.err("override can only be used on methods")
		} else if isClassMember {
			decl.Vars = append(decl.Vars, ast.ClassVar{Name: memberName.Lexeme, Tok: memberName, Value: value})
		} else {
			decl.Fields = append(decl.Fields, ast.ClassField{Name: memberName.Lexeme, Tok: memberName, Value: value})
		}
		p.skipNewlines()
	}
	if !p.match(token.DEDENT) {
		return nil, p.err("expected dedent after class")
	}
	return decl, nil
}

func (p *Parser) interfaceDecl() (ast.Stmt, error) {
	p.next()
	name, err := p.expectName("expected interface name")
	if err != nil {
		return nil, err
	}
	var parents []ast.ClassRef
	if p.at(token.IDENT) && p.peek().Lexeme == "extends" {
		p.next()
		for {
			ref, err := p.classRef()
			if err != nil {
				return nil, err
			}
			parents = append(parents, *ref)
			if !p.match(token.COMMA) {
				break
			}
		}
	}
	if !p.match(token.NEWLINE) {
		return nil, p.err("expected indented block after interface")
	}
	decl := &ast.InterfaceDecl{Name: name.Lexeme, NameTok: name, Parents: parents}
	if !p.match(token.INDENT) {
		if len(parents) > 0 {
			return decl, nil
		}
		return nil, p.err("expected indented block after interface")
	}
	p.blockDepth++
	defer func() { p.blockDepth-- }()
	p.skipNewlines()
	for !p.at(token.DEDENT) && !p.at(token.EOF) {
		if p.match(token.AT) {
			return nil, p.err("interface bodies may only contain instance method requirements")
		}
		methodName, err := p.expectName("expected interface method name")
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(methodName.Lexeme, "_") {
			return nil, p.errAt(methodName, "private interface methods are not in Tya v0.13")
		}
		if !p.match(token.ASSIGN) {
			return nil, p.err("expected '=' after interface method name")
		}
		if !p.at(token.IDENT) && !p.at(token.ARROW) {
			return nil, p.err("interface bodies may only contain instance method requirements")
		}
		params, paramToks, err := p.abstractMethodParams()
		if err != nil {
			return nil, err
		}
		decl.Methods = append(decl.Methods, ast.InterfaceMethod{Name: methodName.Lexeme, Tok: methodName, Params: params, ParamToks: paramToks})
		p.skipNewlines()
	}
	if !p.match(token.DEDENT) {
		return nil, p.err("expected dedent after interface")
	}
	return decl, nil
}

func (p *Parser) startsClassDecl() bool {
	if !p.at(token.IDENT) {
		return false
	}
	if p.peek().Lexeme == "class" {
		return true
	}
	if (p.peek().Lexeme == "abstract" || p.peek().Lexeme == "final") && p.peekN(1).Lexeme == "class" {
		return true
	}
	if p.peek().Lexeme == "abstract" && p.peekN(1).Lexeme == "final" && p.peekN(2).Lexeme == "class" {
		return true
	}
	if p.peek().Lexeme == "final" && p.peekN(1).Lexeme == "abstract" && p.peekN(2).Lexeme == "class" {
		return true
	}
	return false
}

func (p *Parser) abstractMethodParams() ([]string, []token.Token, error) {
	var params []string
	var paramToks []token.Token
	if p.match(token.ARROW) {
		if p.startsExpr() {
			return nil, nil, p.err("abstract methods cannot have bodies")
		}
		if p.at(token.NEWLINE) && p.peekN(1).Type == token.INDENT {
			return nil, nil, p.err("abstract methods cannot have bodies")
		}
		return params, paramToks, nil
	}
	for {
		param, err := p.expectName("expected abstract method parameter")
		if err != nil {
			return nil, nil, err
		}
		params = append(params, param.Lexeme)
		paramToks = append(paramToks, param)
		if !p.match(token.COMMA) {
			break
		}
	}
	if !p.match(token.ARROW) {
		return nil, nil, p.err("expected '->' after abstract method parameters")
	}
	if p.startsExpr() {
		return nil, nil, p.err("abstract methods cannot have bodies")
	}
	if p.at(token.NEWLINE) && p.peekN(1).Type == token.INDENT {
		return nil, nil, p.err("abstract methods cannot have bodies")
	}
	return params, paramToks, nil
}

func (p *Parser) classRef() (*ast.ClassRef, error) {
	first, err := p.expectName("expected parent class name")
	if err != nil {
		return nil, err
	}
	ref := &ast.ClassRef{Name: first.Lexeme, Tok: first}
	if p.match(token.DOT) {
		second, err := p.expectName("expected parent class name")
		if err != nil {
			return nil, err
		}
		ref.Module = first.Lexeme
		ref.Name = second.Lexeme
		ref.Tok = second
	}
	return ref, nil
}

func (p *Parser) ifStmt() (ast.Stmt, error) {
	p.next()
	cond, err := p.exprLine()
	if err != nil {
		return nil, err
	}
	thenBlock, err := p.block("if")
	if err != nil {
		return nil, err
	}
	var elseBlock []ast.Stmt
	p.skipNewlines()
	if p.at(token.IDENT) && p.peek().Lexeme == "elseif" {
		elseif, err := p.ifStmt()
		if err != nil {
			return nil, err
		}
		elseBlock = []ast.Stmt{elseif}
	} else if p.at(token.IDENT) && p.peek().Lexeme == "else" {
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
	cond, err := p.exprLine()
	if err != nil {
		return nil, err
	}
	p.loopDepth++
	body, err := p.block("while")
	p.loopDepth--
	if err != nil {
		return nil, err
	}
	return &ast.WhileStmt{Cond: cond, Body: body}, nil
}

func (p *Parser) forStmt() (ast.Stmt, error) {
	p.next()
	valueTok, err := p.expectName("expected loop variable")
	if err != nil {
		return nil, err
	}
	valueName := valueTok.Lexeme
	var indexName string
	var indexTok token.Token
	if p.match(token.COMMA) {
		idx, err := p.expectName("expected loop index variable")
		if err != nil {
			return nil, err
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
	iterable, err := p.exprLine()
	if err != nil {
		return nil, err
	}
	p.loopDepth++
	body, err := p.block("for")
	p.loopDepth--
	if err != nil {
		return nil, err
	}
	return &ast.ForInStmt{ValueName: valueName, IndexName: indexName, ValueTok: valueTok, IndexTok: indexTok, Kind: kind, Iterable: iterable, Body: body}, nil
}

func (p *Parser) returnStmt() (ast.Stmt, error) {
	tok := p.next()
	if p.at(token.NEWLINE) || p.at(token.DEDENT) || p.at(token.EOF) {
		return &ast.ReturnStmt{Tok: tok}, nil
	}
	values, err := p.exprListLine()
	if err != nil {
		return nil, err
	}
	return &ast.ReturnStmt{Values: values, Tok: tok}, nil
}

func (p *Parser) raiseStmt() (ast.Stmt, error) {
	tok := p.next()
	if p.at(token.NEWLINE) || p.at(token.DEDENT) || p.at(token.EOF) {
		return nil, p.errAt(tok, "raise requires an expression")
	}
	value, err := p.exprLine()
	if err != nil {
		return nil, err
	}
	return &ast.RaiseStmt{Value: value, Tok: tok}, nil
}

func (p *Parser) tryCatchStmt() (ast.Stmt, error) {
	tok := p.next()
	body, err := p.block("try")
	if err != nil {
		return nil, err
	}
	p.skipNewlines()
	if !(p.at(token.IDENT) && p.peek().Lexeme == "catch") {
		return nil, p.errAt(tok, "block try requires catch")
	}
	catchTok := p.next()
	if !p.at(token.IDENT) {
		return nil, p.errAt(catchTok, "catch requires a binding name")
	}
	name := p.next()
	if p.at(token.COMMA) || p.at(token.LBRACKET) || p.at(token.LBRACE) {
		return nil, p.errAt(name, "catch binding must be a name")
	}
	catchBody, err := p.block("catch")
	if err != nil {
		return nil, err
	}
	return &ast.TryCatchStmt{Try: body, CatchName: name.Lexeme, CatchTok: name, Catch: catchBody, Tok: tok}, nil
}

func (p *Parser) matchStmt() (ast.Stmt, error) {
	tok := p.next()
	if p.at(token.NEWLINE) || p.at(token.DEDENT) || p.at(token.EOF) {
		return nil, p.errAt(tok, "match requires a value expression")
	}
	value, err := p.exprLine()
	if err != nil {
		return nil, err
	}
	if !p.match(token.NEWLINE) || !p.match(token.INDENT) {
		return nil, p.err("expected indented block after match")
	}
	p.blockDepth++
	defer func() { p.blockDepth-- }()
	var cases []ast.MatchCase
	p.skipNewlines()
	for !p.at(token.DEDENT) && !p.at(token.EOF) {
		if !(p.at(token.IDENT) && p.peek().Lexeme == "case") {
			return nil, p.err("expected case in match")
		}
		caseTok := p.next()
		pattern, err := p.pattern()
		if err != nil {
			return nil, err
		}
		body, err := p.block("case")
		if err != nil {
			return nil, err
		}
		cases = append(cases, ast.MatchCase{Pattern: pattern, Tok: caseTok, Body: body})
		p.skipNewlines()
	}
	if !p.match(token.DEDENT) {
		return nil, p.err("expected dedent after match")
	}
	return &ast.MatchStmt{Value: value, Cases: cases, Tok: tok}, nil
}

func (p *Parser) block(owner string) ([]ast.Stmt, error) {
	if !p.match(token.NEWLINE) || !p.match(token.INDENT) {
		return nil, p.err("expected indented block after " + owner)
	}
	p.blockDepth++
	defer func() { p.blockDepth-- }()
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
		obj, err := p.dictBody()
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

func (p *Parser) dictBody() (*ast.DictLit, error) {
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

func (p *Parser) stmtExprLine() (ast.Expr, error) {
	if p.at(token.IDENT) && p.peek().Lexeme == "assert_equal" && p.peekN(1).Type != token.LPAREN && p.startsExprAt(p.pos+1) {
		callee := &ast.Ident{Name: p.peek().Lexeme, Tok: p.next()}
		args, err := p.exprListLine()
		if err != nil {
			return nil, err
		}
		if len(args) != 2 {
			return nil, p.errAt(callee.Tok, "assert_equal expects two expressions")
		}
		return &ast.CallExpr{Callee: callee, Args: args}, nil
	}
	if p.at(token.IDENT) && (p.peek().Lexeme == "print" || p.peek().Lexeme == "assert") && p.peekN(1).Type != token.LPAREN && p.startsExprAt(p.pos+1) {
		callee := &ast.Ident{Name: p.peek().Lexeme, Tok: p.next()}
		arg, err := p.expr()
		if err != nil {
			return nil, err
		}
		if !p.at(token.NEWLINE) && !p.at(token.DEDENT) && !p.at(token.EOF) {
			return nil, p.err(callee.Name + " expects one expression")
		}
		return &ast.CallExpr{Callee: callee, Args: []ast.Expr{arg}}, nil
	}
	return p.exprLineWithPrintSugar(true)
}

func (p *Parser) exprLine() (ast.Expr, error) {
	return p.exprLineWithPrintSugar(false)
}

func (p *Parser) exprLineWithPrintSugar(allowPrintSugar bool) (ast.Expr, error) {
	ex, err := p.expr()
	if err != nil {
		return nil, err
	}
	if _, ok := ex.(*ast.FuncLit); ok {
		return ex, nil
	}
	if allowPrintSugar && p.isSingleArgStatementSugarIdent(ex) && p.startsExpr() {
		arg, err := p.expr()
		if err != nil {
			return nil, err
		}
		if !p.at(token.NEWLINE) && !p.at(token.DEDENT) && !p.at(token.EOF) {
			id := ex.(*ast.Ident)
			return nil, p.err(id.Name + " expects one expression")
		}
		return &ast.CallExpr{Callee: ex, Args: []ast.Expr{arg}}, nil
	}
	if !allowPrintSugar && p.isSingleArgStatementSugarIdent(ex) && p.startsExpr() {
		return nil, p.err("no-paren calls are not in Tya v0.1; use parentheses")
	}
	if p.startsExpr() {
		return nil, p.err("no-paren calls are not in Tya v0.1; use parentheses")
	}
	return ex, nil
}

func (p *Parser) isSingleArgStatementSugarIdent(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	return ok && (id.Name == "print" || id.Name == "assert")
}

func (p *Parser) expr() (ast.Expr, error) {
	return p.exprWithCommaFunc(true)
}

func (p *Parser) exprNoCommaFunc() (ast.Expr, error) {
	return p.exprWithCommaFunc(false)
}

func (p *Parser) exprWithCommaFunc(allowCommaFunc bool) (ast.Expr, error) {
	if p.startsFunctionParams(allowCommaFunc) {
		params, paramToks, err := p.functionParams()
		if err != nil {
			return nil, err
		}
		return p.finishFunc(params, paramToks)
	}
	if p.startsParenFunctionParams() {
		params, paramToks, err := p.parenFunctionParams()
		if err != nil {
			return nil, err
		}
		return p.finishFunc(params, paramToks)
	}
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

func (p *Parser) startsFunctionParams(allowCommaFunc bool) bool {
	if !p.at(token.IDENT) {
		return false
	}
	if p.peekN(1).Type == token.ARROW {
		return true
	}
	if !allowCommaFunc || p.peekN(1).Type != token.COMMA {
		return false
	}
	i := p.pos + 2
	if p.toks[i].Type != token.IDENT {
		return false
	}
	for {
		i++
		switch p.toks[i].Type {
		case token.ARROW:
			return true
		case token.COMMA:
			i++
			if p.toks[i].Type != token.IDENT {
				return false
			}
		default:
			return false
		}
	}
}

// startsParenFunctionParams returns true when the current token starts
// a parenthesized function-parameter list of the form `(a, b, c) ->`.
// The form `() -> body` is already handled by primary(), so we only
// match LPAREN immediately followed by an IDENT here.
func (p *Parser) startsParenFunctionParams() bool {
	if !p.at(token.LPAREN) {
		return false
	}
	i := p.pos + 1
	if p.toks[i].Type != token.IDENT {
		return false
	}
	for {
		i++
		switch p.toks[i].Type {
		case token.RPAREN:
			if p.toks[i+1].Type == token.ARROW {
				return true
			}
			return false
		case token.COMMA:
			i++
			if p.toks[i].Type != token.IDENT {
				return false
			}
		default:
			return false
		}
	}
}

func (p *Parser) parenFunctionParams() ([]string, []token.Token, error) {
	if !p.match(token.LPAREN) {
		return nil, nil, p.err("expected '('")
	}
	first, err := p.expectName("expected function parameter")
	if err != nil {
		return nil, nil, err
	}
	params := []string{first.Lexeme}
	paramToks := []token.Token{first}
	for p.match(token.COMMA) {
		next, err := p.expectName("expected function parameter")
		if err != nil {
			return nil, nil, err
		}
		params = append(params, next.Lexeme)
		paramToks = append(paramToks, next)
	}
	if !p.match(token.RPAREN) {
		return nil, nil, p.err("expected ')'")
	}
	p.match(token.ARROW)
	return params, paramToks, nil
}

func (p *Parser) functionParams() ([]string, []token.Token, error) {
	first, err := p.expectName("expected function parameter")
	if err != nil {
		return nil, nil, err
	}
	params := []string{first.Lexeme}
	paramToks := []token.Token{first}
	for p.match(token.COMMA) {
		next, err := p.expectName("expected function parameter")
		if err != nil {
			return nil, nil, err
		}
		params = append(params, next.Lexeme)
		paramToks = append(paramToks, next)
	}
	p.match(token.ARROW)
	return params, paramToks, nil
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
	left, err := p.bitOr()
	if err != nil {
		return nil, err
	}
	for p.match(token.LT) || p.match(token.LTE) || p.match(token.GT) || p.match(token.GTE) {
		op := p.prev()
		right, err := p.bitOr()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) bitOr() (ast.Expr, error) {
	left, err := p.bitXor()
	if err != nil {
		return nil, err
	}
	for p.match(token.PIPE) {
		op := p.prev()
		right, err := p.bitXor()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) bitXor() (ast.Expr, error) {
	left, err := p.bitAnd()
	if err != nil {
		return nil, err
	}
	for p.match(token.CARET) {
		op := p.prev()
		right, err := p.bitAnd()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) bitAnd() (ast.Expr, error) {
	left, err := p.term()
	if err != nil {
		return nil, err
	}
	for p.match(token.AMP) {
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
	for p.match(token.STAR) || p.match(token.SLASH) || p.match(token.PERCENT) || p.match(token.SHL) || p.match(token.SHR) {
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
		if p.functionDepth == 0 {
			return nil, p.err("try must be inside a function")
		}
		tok := p.next()
		ex, err := p.unary()
		if err != nil {
			return nil, err
		}
		return &ast.TryExpr{Expr: ex, Tok: tok}, nil
	}
	if p.matchWord("not") || p.match(token.MINUS) || p.match(token.TILDE) {
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
			name, err := p.expectCallableIdent("expected property name")
			if err != nil {
				return nil, err
			}
			ex = &ast.MemberExpr{Target: ex, Name: name.Lexeme, NameTok: name}
			continue
		}
		if p.match(token.LPAREN) {
			var args []ast.Expr
			if !p.at(token.RPAREN) {
				for {
					arg, err := p.exprNoCommaFunc()
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
			ex = &ast.IndexExpr{Target: ex, Index: index}
			continue
		}
		break
	}
	return ex, nil
}

func (p *Parser) primary() (ast.Expr, error) {
	tok := p.next()
	switch tok.Type {
	case token.AT:
		if p.at(token.AT) {
			p.next()
			name, err := p.expectName("expected class variable name")
			if err != nil {
				return nil, err
			}
			return &ast.ClassVarExpr{Name: name.Lexeme, NameTok: name}, nil
		}
		name, err := p.expectName("expected instance field name")
		if err != nil {
			return nil, err
		}
		return &ast.InstanceFieldExpr{Name: name.Lexeme, NameTok: name}, nil
	case token.IDENT:
		if p.peek().Type == token.QUESTION {
			p.next()
			tok.Lexeme += "?"
			return &ast.Ident{Name: tok.Lexeme, Tok: tok}, nil
		}
		if tok.Lexeme == "true" {
			return &ast.BoolLit{Value: true}, nil
		}
		if tok.Lexeme == "false" {
			return &ast.BoolLit{Value: false}, nil
		}
		if tok.Lexeme == "nil" {
			return &ast.NilLit{}, nil
		}
		if tok.Lexeme == "super" {
			return &ast.SuperExpr{Tok: tok}, nil
		}
		if tok.Lexeme == "self" {
			return &ast.SelfExpr{Tok: tok}, nil
		}
		if err := p.rejectReservedName(tok); err != nil {
			return nil, err
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
	case token.BYTES:
		return &ast.BytesLit{Value: tok.Lexeme}, nil
	case token.LPAREN:
		if p.match(token.RPAREN) {
			if p.match(token.ARROW) {
				return p.finishFunc(nil, nil)
			}
			return nil, p.err("expected expression")
		}
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
	case token.ARROW:
		return p.finishFunc(nil, nil)
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
	return nil, p.err("set literals are not in Tya v0.1")
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

func (p *Parser) finishFunc(params []string, paramToks []token.Token) (*ast.FuncLit, error) {
	p.functionDepth++
	defer func() { p.functionDepth-- }()

	if p.at(token.NEWLINE) && p.peekN(1).Type == token.INDENT {
		p.next()
		p.next()
		p.blockDepth++
		prevLoopDepth := p.loopDepth
		p.loopDepth = 0
		defer func() {
			p.loopDepth = prevLoopDepth
			p.blockDepth--
		}()
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
		if isDestructuringTarget(first) {
			return nil, p.err("mixed destructuring and multiple assignment is not in Tya v0.14")
		}
		if _, ok := first.(*ast.Ident); !ok {
			return nil, p.err("multiple assignment targets must be identifiers")
		}
		next, err := p.assignTarget()
		if err != nil {
			return nil, err
		}
		if _, ok := next.(*ast.Ident); !ok {
			return nil, p.err("multiple assignment targets must be identifiers")
		}
		targets = append(targets, next)
	}
	return targets, nil
}

func (p *Parser) assignTarget() (ast.Expr, error) {
	var ex ast.Expr
	if p.at(token.LBRACKET) {
		return p.arrayPattern()
	}
	if p.at(token.LBRACE) {
		return p.dictPattern()
	}
	if p.match(token.AT) {
		if p.at(token.AT) {
			p.next()
			name, err := p.expectName("expected class variable name")
			if err != nil {
				return nil, err
			}
			ex = &ast.ClassVarExpr{Name: name.Lexeme, NameTok: name}
		} else {
			name, err := p.expectName("expected instance field name")
			if err != nil {
				return nil, err
			}
			ex = &ast.InstanceFieldExpr{Name: name.Lexeme, NameTok: name}
		}
	} else {
		tok, err := p.expectCallableName("expected assignment target")
		if err != nil {
			return nil, err
		}
		ex = ast.Expr(&ast.Ident{Name: tok.Lexeme, Tok: tok})
	}
	for {
		if p.match(token.DOT) {
			name, err := p.expectCallableIdent("expected property name")
			if err != nil {
				return nil, err
			}
			ex = &ast.MemberExpr{Target: ex, Name: name.Lexeme, NameTok: name}
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
			ex = &ast.IndexExpr{Target: ex, Index: index}
			continue
		}
		break
	}
	return ex, nil
}

func (p *Parser) arrayPattern() (ast.Expr, error) {
	p.next()
	var elems []ast.Expr
	if !p.at(token.RBRACKET) {
		for {
			elem, err := p.patternTarget()
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
}

func (p *Parser) dictPattern() (ast.Expr, error) {
	p.next()
	dict := &ast.DictLit{}
	if !p.at(token.RBRACE) {
		for {
			if !p.at(token.STRING) {
				return nil, p.err("destructuring dictionary keys must be string literals")
			}
			key := p.next()
			if !p.match(token.COLON) {
				return nil, p.err("expected ':' after destructuring dictionary key")
			}
			value, err := p.patternTarget()
			if err != nil {
				return nil, err
			}
			dict.Props = append(dict.Props, ast.DictProp{Name: key.Lexeme, Tok: key, Value: value})
			if !p.match(token.COMMA) {
				break
			}
		}
	}
	if !p.match(token.RBRACE) {
		return nil, p.err("expected '}'")
	}
	return dict, nil
}

func (p *Parser) patternTarget() (ast.Expr, error) {
	switch p.peek().Type {
	case token.IDENT:
		tok := p.next()
		return &ast.Ident{Name: tok.Lexeme, Tok: tok}, nil
	case token.LBRACKET:
		return p.arrayPattern()
	case token.LBRACE:
		return p.dictPattern()
	default:
		return nil, p.err("expected destructuring target")
	}
}

func (p *Parser) pattern() (ast.Expr, error) {
	return p.patternValue(map[string]bool{})
}

func (p *Parser) patternValue(bindings map[string]bool) (ast.Expr, error) {
	switch p.peek().Type {
	case token.IDENT:
		tok := p.next()
		switch tok.Lexeme {
		case "nil":
			return &ast.NilLit{}, nil
		case "true":
			return &ast.BoolLit{Value: true}, nil
		case "false":
			return &ast.BoolLit{Value: false}, nil
		case "_":
			return &ast.Ident{Name: tok.Lexeme, Tok: tok}, nil
		default:
			if bindings[tok.Lexeme] {
				return nil, p.errAt(tok, "duplicate binding name in pattern")
			}
			bindings[tok.Lexeme] = true
			return &ast.Ident{Name: tok.Lexeme, Tok: tok}, nil
		}
	case token.INT:
		tok := p.next()
		v, _ := strconv.ParseInt(tok.Lexeme, 10, 64)
		return &ast.IntLit{Value: v}, nil
	case token.FLOAT:
		tok := p.next()
		v, _ := strconv.ParseFloat(tok.Lexeme, 64)
		return &ast.FloatLit{Value: v}, nil
	case token.STRING:
		tok := p.next()
		return &ast.StringLit{Value: tok.Lexeme}, nil
	case token.LBRACKET:
		return p.arrayPatternValue(bindings)
	case token.LBRACE:
		return p.dictPatternValue(bindings)
	default:
		return nil, p.err("invalid pattern syntax")
	}
}

func (p *Parser) arrayPatternValue(bindings map[string]bool) (ast.Expr, error) {
	p.next()
	var elems []ast.Expr
	if !p.at(token.RBRACKET) {
		for {
			elem, err := p.patternValue(bindings)
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
}

func (p *Parser) dictPatternValue(bindings map[string]bool) (ast.Expr, error) {
	p.next()
	dict := &ast.DictLit{}
	if !p.at(token.RBRACE) {
		for {
			if !p.at(token.STRING) {
				return nil, p.err("pattern dictionary keys must be string literals")
			}
			key := p.next()
			if !p.match(token.COLON) {
				return nil, p.err("expected ':' after pattern dictionary key")
			}
			value, err := p.patternValue(bindings)
			if err != nil {
				return nil, err
			}
			dict.Props = append(dict.Props, ast.DictProp{Name: key.Lexeme, Tok: key, Value: value})
			if !p.match(token.COMMA) {
				break
			}
		}
	}
	if !p.match(token.RBRACE) {
		return nil, p.err("expected '}'")
	}
	return dict, nil
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
	if p.toks[*i].Type == token.LBRACKET || p.toks[*i].Type == token.LBRACE {
		return p.scanBalancedPattern(i)
	}
	if p.toks[*i].Type == token.AT {
		*i++
		if p.toks[*i].Type == token.AT {
			*i++
		}
		if p.toks[*i].Type != token.IDENT {
			return false
		}
		*i++
		if p.toks[*i].Type == token.QUESTION {
			*i++
		}
		return true
	}
	if p.toks[*i].Type != token.IDENT {
		return false
	}
	*i++
	if p.toks[*i].Type == token.QUESTION {
		*i++
	}
	for {
		if p.toks[*i].Type == token.DOT {
			*i += 2
			if p.toks[*i].Type == token.QUESTION {
				*i++
			}
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

func (p *Parser) scanBalancedPattern(i *int) bool {
	stack := []token.Type{}
	for *i < len(p.toks) {
		switch p.toks[*i].Type {
		case token.LBRACKET:
			stack = append(stack, token.RBRACKET)
		case token.LBRACE:
			stack = append(stack, token.RBRACE)
		case token.RBRACKET, token.RBRACE:
			if len(stack) == 0 || stack[len(stack)-1] != p.toks[*i].Type {
				return false
			}
			stack = stack[:len(stack)-1]
			*i++
			if len(stack) == 0 {
				return true
			}
			continue
		case token.NEWLINE, token.EOF:
			return false
		}
		*i++
	}
	return false
}

func isDestructuringTarget(target ast.Expr) bool {
	switch target.(type) {
	case *ast.ArrayLit, *ast.DictLit:
		return true
	default:
		return false
	}
}

func (p *Parser) skipNewlines() {
	for p.match(token.NEWLINE) {
	}
}
func (p *Parser) startsExpr() bool {
	return p.startsExprAt(p.pos)
}
func (p *Parser) startsExprAt(pos int) bool {
	if pos >= len(p.toks) {
		return false
	}
	switch p.toks[pos].Type {
	case token.IDENT, token.STRING, token.BYTES, token.INT, token.FLOAT, token.MINUS, token.TILDE, token.LPAREN, token.LBRACKET, token.LBRACE, token.ARROW, token.AT:
		return true
	default:
		return false
	}
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
func (p *Parser) expect(t token.Type) token.Token {
	if p.at(t) {
		return p.next()
	}
	return token.Token{}
}
func (p *Parser) expectName(msg string) (token.Token, error) {
	if !p.at(token.IDENT) {
		return token.Token{}, p.err(msg)
	}
	tok := p.next()
	if err := p.rejectReservedName(tok); err != nil {
		return token.Token{}, err
	}
	return tok, nil
}
func (p *Parser) expectCallableName(msg string) (token.Token, error) {
	if !p.at(token.IDENT) {
		return token.Token{}, p.err(msg)
	}
	tok := p.next()
	if p.match(token.QUESTION) {
		tok.Lexeme += "?"
		return tok, nil
	}
	if err := p.rejectReservedName(tok); err != nil {
		return token.Token{}, err
	}
	return tok, nil
}
func (p *Parser) expectIdent(msg string) (token.Token, error) {
	if !p.at(token.IDENT) {
		return token.Token{}, p.err(msg)
	}
	return p.next(), nil
}
func (p *Parser) expectCallableIdent(msg string) (token.Token, error) {
	tok, err := p.expectIdent(msg)
	if err != nil {
		return token.Token{}, err
	}
	if p.match(token.QUESTION) {
		tok.Lexeme += "?"
	}
	return tok, nil
}
func (p *Parser) rejectV01ExcludedIdent() error {
	if !p.at(token.IDENT) {
		return nil
	}
	return p.rejectV01ExcludedWord(p.peek().Lexeme)
}
func (p *Parser) rejectV01ExcludedWord(word string) error {
	switch word {
	case "object", "set":
		return p.err(word + " is not in Tya v0.1")
	default:
		return nil
	}
}
func (p *Parser) rejectReservedName(tok token.Token) error {
	switch tok.Lexeme {
	case "object", "set":
		return p.errAt(tok, tok.Lexeme+" is not in Tya v0.1")
	case "interface", "class", "self", "true", "false", "nil", "if", "elseif", "else", "while", "for", "in", "of", "break", "continue", "return", "try", "module", "import", "and", "or", "not":
		return p.errAt(tok, tok.Lexeme+" cannot be used as a name")
	default:
		return nil
	}
}
func (p *Parser) err(msg string) error {
	tok := p.peek()
	return p.errAt(tok, msg)
}
func (p *Parser) errAt(tok token.Token, msg string) error {
	return fmt.Errorf("%d:%d: %s near %q", tok.Line, tok.Col, msg, tok.Lexeme)
}
