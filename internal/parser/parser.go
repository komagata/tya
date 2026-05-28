package parser

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"tya/internal/ast"
	"tya/internal/diag"
	"tya/internal/token"
)

type Parser struct {
	toks          []token.Token
	pos           int
	blockDepth    int
	loopDepth     int
	functionDepth int
	classDepth    int
	continuedLine bool

	// errs accumulates structured diagnostics during a multi-error
	// parse run. Populated by newDiag (see diagnostic.go) and
	// flushed into a *ParserError when Parse returns.
	errs []diag.Diagnostic
}

// Parse runs the multi-error parser over toks. The convention
// (since v0.56) is:
//
//   - diags is the structured list of every recoverable parser
//     diagnostic collected via statement- and expression-level
//     recovery. It is nil on a clean parse, non-nil otherwise.
//   - err is a *ParserError wrapping the same diags. It is nil
//     when diags is nil and non-nil otherwise. Existing callers
//     using `errors.As(err, &perr)` continue to work; new
//     callers can iterate diags directly.
func Parse(toks []token.Token) (*ast.Program, []diag.Diagnostic, error) {
	p := &Parser{toks: toks}
	prog, _ := p.program()
	if flushed := p.flushErrors(); flushed != nil {
		perr := flushed.(*ParserError)
		return prog, perr.Diags, perr
	}
	return prog, nil, nil
}

// ParseWithComments runs Parse and additionally fills
// Program.HeaderComments and Program.Comments from the supplied
// comments slice. Per docs/SPEC.md Formatted Syntax:
//
//   - Leading comments: contiguous `#` lines immediately before a
//     statement, at the same indent as that statement.
//   - Line-end comments: a single non-full-line `#` comment on the
//     same source line as the statement's start.
//   - Header comments: contiguous full-line `#` lines starting at
//     line 1 at indent 0, separated from the body by exactly one
//     blank line.
func ParseWithComments(toks []token.Token, comments []CommentInfo) (*ast.Program, []diag.Diagnostic, error) {
	prog, diags, err := Parse(toks)
	if err != nil {
		return nil, diags, err
	}
	if len(comments) == 0 {
		return prog, nil, nil
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
			return prog, diags, nil
		}
	}
	prog.HeaderComments = header
	return prog, diags, nil
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

// attachStmtComments walks every Stmt body in prog and attaches
// leading and line-end comments per §3.1 / §3.2. Top-level
// statements use indent 0; nested bodies inherit indent + 2.
// ModuleDecl.Members and ClassDecl.Methods are not Stmt slices and
// are not visited in v0.36.
func attachStmtComments(prog *ast.Program, comments []CommentInfo) {
	attachStmtCommentsTracked(prog, comments)
}

// attachStmtCommentsTracked is attachStmtComments plus a returned
// `used` slice marking which comments were attached as
// header / leading / line-end.  Comments with used[i] == false are
// at forbidden positions per CANONICAL §3.4.
func attachStmtCommentsTracked(prog *ast.Program, comments []CommentInfo) []bool {
	used := make([]bool, len(comments))
	if prog == nil {
		return used
	}
	// Header comments were claimed already in ParseWithComments;
	// mark their indices as used.
	if len(prog.HeaderComments) > 0 {
		count := len(prog.HeaderComments)
		consumed := 0
		for i := 0; i < len(comments) && consumed < count; i++ {
			c := comments[i]
			if !c.IsFullLine || c.Indent != 0 {
				continue
			}
			used[i] = true
			consumed++
		}
	}
	if len(prog.Stmts) == 0 {
		return used
	}
	prog.Comments = map[ast.Stmt]ast.StmtComments{}
	attachStmtBlock(prog.Comments, prog.Stmts, 0, comments, used)
	markMemberComments(prog.Stmts, comments, used)
	return used
}

func markMemberComments(stmts []ast.Stmt, comments []CommentInfo, used []bool) {
	for _, stmt := range stmts {
		switch d := stmt.(type) {
		case *ast.ModuleDecl:
			for _, class := range d.Classes {
				markClassMemberComments(class, comments, used)
			}
			for _, iface := range d.Interfaces {
				markInterfaceMemberComments(iface, comments, used)
			}
		case *ast.ClassDecl:
			markClassMemberComments(d, comments, used)
		case *ast.InterfaceDecl:
			markInterfaceMemberComments(d, comments, used)
		}
	}
}

func markClassMemberComments(d *ast.ClassDecl, comments []CommentInfo, used []bool) {
	for i := range d.Fields {
		d.Fields[i].Comments.Leading = takeLeadingComments(d.Fields[i].Tok.Line, 2, comments, used)
	}
	for i := range d.Constants {
		d.Constants[i].Comments.Leading = takeLeadingComments(d.Constants[i].Tok.Line, 2, comments, used)
	}
	for i := range d.Vars {
		d.Vars[i].Comments.Leading = takeLeadingComments(d.Vars[i].Tok.Line, 2, comments, used)
	}
	for i := range d.Methods {
		d.Methods[i].Comments.Leading = takeLeadingComments(d.Methods[i].Tok.Line, 2, comments, used)
	}
}

func markInterfaceMemberComments(d *ast.InterfaceDecl, comments []CommentInfo, used []bool) {
	for i := range d.Fields {
		d.Fields[i].Comments.Leading = takeLeadingComments(d.Fields[i].Tok.Line, 2, comments, used)
	}
	for i := range d.Methods {
		d.Methods[i].Comments.Leading = takeLeadingComments(d.Methods[i].Tok.Line, 2, comments, used)
	}
}

func takeLeadingComments(startLine int, indent int, comments []CommentInfo, used []bool) []string {
	expectedLine := startLine - 1
	var leading []string
	for j := len(comments) - 1; j >= 0; j-- {
		if used[j] {
			continue
		}
		c := comments[j]
		if c.Line > expectedLine {
			continue
		}
		if c.Line < expectedLine {
			break
		}
		if !c.IsFullLine || c.Indent != indent {
			break
		}
		used[j] = true
		leading = append(leading, c.Text)
		expectedLine--
	}
	for i, j := 0, len(leading)-1; i < j; i, j = i+1, j-1 {
		leading[i], leading[j] = leading[j], leading[i]
	}
	return leading
}

// OrphanComments reports comments that ParseWithComments could not
// attach to a header, leading, or line-end position. Per CANONICAL
// §3.4 these are forbidden: block-trailing, file-trailing, and
// (when the lexer surfaces them) bracket-internal comments. The
// caller (typically `tya check`) renders them as structured
// diagnostics.
func OrphanComments(prog *ast.Program, comments []CommentInfo) []CommentInfo {
	used := attachStmtCommentsTracked(prog, comments)
	var orphans []CommentInfo
	for i, c := range comments {
		if !used[i] {
			orphans = append(orphans, c)
		}
	}
	return orphans
}

func attachStmtBlock(out map[ast.Stmt]ast.StmtComments, stmts []ast.Stmt, indent int, comments []CommentInfo, used []bool) {
	for i := 0; i < len(stmts); i++ {
		stmt := stmts[i]
		startLine, _ := stmtPos(stmt)
		if startLine == 0 {
			recurseStmtBodies(out, stmt, indent, comments, used)
			continue
		}
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
		if len(leading) > 0 || lineEnd != "" {
			out[stmt] = ast.StmtComments{Leading: leading, LineEnd: lineEnd}
		}
		recurseStmtBodies(out, stmt, indent, comments, used)
	}
}

func recurseStmtBodies(out map[ast.Stmt]ast.StmtComments, stmt ast.Stmt, indent int, comments []CommentInfo, used []bool) {
	inner := indent + 2
	switch n := stmt.(type) {
	case *ast.IfStmt:
		attachStmtBlock(out, n.Then, inner, comments, used)
		attachStmtBlock(out, n.Else, inner, comments, used)
	case *ast.WhileStmt:
		attachStmtBlock(out, n.Body, inner, comments, used)
	case *ast.ForInStmt:
		attachStmtBlock(out, n.Body, inner, comments, used)
	case *ast.TryCatchStmt:
		attachStmtBlock(out, n.Try, inner, comments, used)
		attachStmtBlock(out, n.Catch, inner, comments, used)
		attachStmtBlock(out, n.Finally, inner, comments, used)
	case *ast.MatchStmt:
		for _, c := range n.Cases {
			attachStmtBlock(out, c.Body, inner, comments, used)
		}
	case *ast.SelectStmt:
		for _, arm := range n.Arms {
			attachStmtBlock(out, arm.Body, inner, comments, used)
		}
	case *ast.AssignStmt:
		// Walk function-literal bodies on the rhs so nested stmts
		// inside `f = x -> ...` are visited.
		for _, v := range n.Values {
			recurseExprBodies(out, v, inner, comments, used)
		}
	case *ast.ExprStmt:
		recurseExprBodies(out, n.Expr, inner, comments, used)
	}
}

func recurseExprBodies(out map[ast.Stmt]ast.StmtComments, expr ast.Expr, indent int, comments []CommentInfo, used []bool) {
	switch n := expr.(type) {
	case *ast.FuncLit:
		attachStmtBlock(out, n.Body, indent, comments, used)
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
	case *ast.ImportBlockStmt:
		return n.Tok.Line, n.Tok.Col - 1
	case *ast.EmbedStmt:
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
	case *ast.SelectStmt:
		return n.Tok.Line, n.Tok.Col - 1
	}
	return 0, 0
}

func (p *Parser) program() (*ast.Program, error) {
	var stmts []ast.Stmt
	p.skipNewlines()
	for !p.at(token.EOF) {
		s, err := p.stmt()
		if err != nil {
			// v0.54 multi-error parsing: the diagnostic is in
			// p.errs already; skip to the next top-level
			// statement so we can surface additional issues in
			// the same pass.
			p.skipToNextStmt()
			p.skipNewlines()
			continue
		}
		stmts = append(stmts, s)
		p.skipNewlines()
	}
	return &ast.Program{Stmts: stmts}, nil
}

func (p *Parser) stmt() (ast.Stmt, error) {
	if p.at(token.IDENT) && p.peek().Lexeme == "module" {
		if os.Getenv("TYA_LEGACY_MODULES") == "1" {
			if p.blockDepth != 0 {
				return nil, p.err("module must be top-level")
			}
			return p.moduleDecl()
		}
		tok := p.peek()
		p.skipRemovedModuleDecl()
		return nil, p.newDiag(
			CodeModuleRemoved,
			"module keyword removed",
			"module declarations were removed; use top-level bindings or class package files instead",
			tok.Line,
			tok.Col,
			"delete the module line and move its members to top level",
		)
	}
	if err := p.rejectV1ExcludedStatementSyntax(); err != nil {
		return nil, err
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
	if p.at(token.IDENT) && p.peek().Lexeme == "embed" {
		if p.blockDepth != 0 {
			return nil, p.err("embed must be top-level")
		}
		return p.embedStmt()
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
	if p.at(token.IDENT) && p.peek().Lexeme == "scope" && p.peekN(1).Type == token.NEWLINE {
		return p.scopeBlock()
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "select" && p.peekN(1).Type == token.NEWLINE {
		return p.selectStmt()
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
	if p.at(token.IDENT) && p.peek().Lexeme == "finally" {
		return nil, p.err("finally outside block try")
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "raise" {
		return p.raiseStmt()
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "break" {
		if p.loopDepth == 0 {
			return nil, p.err("break must be inside a loop")
		}
		tok := p.next()
		return &ast.BreakStmt{Tok: tok}, nil
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "continue" {
		if p.loopDepth == 0 {
			return nil, p.err("continue must be inside a loop")
		}
		tok := p.next()
		return &ast.ContinueStmt{Tok: tok}, nil
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

func (p *Parser) skipRemovedModuleDecl() {
	p.next() // module
	for !p.at(token.NEWLINE) && !p.at(token.EOF) {
		p.next()
	}
	if !p.match(token.NEWLINE) || !p.match(token.INDENT) {
		return
	}
	depth := 1
	for depth > 0 && !p.at(token.EOF) {
		if p.match(token.INDENT) {
			depth++
			continue
		}
		if p.match(token.DEDENT) {
			depth--
			continue
		}
		p.next()
	}
}

func (p *Parser) importStmt() (ast.Stmt, error) {
	tok := p.next()
	if p.match(token.NEWLINE) {
		if !p.match(token.INDENT) {
			return nil, p.err("expected indented import block")
		}
		imports := []*ast.ImportStmt{}
		for !p.at(token.DEDENT) && !p.at(token.EOF) {
			if p.at(token.NEWLINE) {
				p.next()
				continue
			}
			imp, err := p.importEntry()
			if err != nil {
				return nil, err
			}
			imports = append(imports, imp)
			if !p.at(token.NEWLINE) && !p.at(token.DEDENT) && !p.at(token.EOF) {
				return nil, p.err("expected newline after import")
			}
			p.match(token.NEWLINE)
		}
		if len(imports) == 0 {
			return nil, p.err("import block requires at least one entry")
		}
		if !p.match(token.DEDENT) {
			return nil, p.err("expected dedent after import block")
		}
		return &ast.ImportBlockStmt{Imports: imports, Tok: tok}, nil
	}
	imp, err := p.importEntry()
	if err != nil {
		return nil, err
	}
	if !p.at(token.NEWLINE) && !p.at(token.DEDENT) && !p.at(token.EOF) {
		return nil, p.err("expected newline after import")
	}
	return imp, nil
}

func (p *Parser) importEntry() (*ast.ImportStmt, error) {
	name, err := p.expectName("expected module name after import")
	if err != nil {
		return nil, err
	}
	parts := []string{name.Lexeme}
	wildcard := false
	for p.match(token.SLASH) {
		if p.match(token.STAR) {
			wildcard = true
			break
		}
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
	return &ast.ImportStmt{Name: strings.Join(parts, "/"), NameTok: name, Alias: alias, AliasTok: aliasTok, Wildcard: wildcard}, nil
}

func (p *Parser) embedStmt() (ast.Stmt, error) {
	p.next() // consume `embed`
	if !p.at(token.STRING) {
		return nil, p.err("expected string path after embed")
	}
	pathTok := p.next()
	if !(p.at(token.IDENT) && p.peek().Lexeme == "as") {
		return nil, p.err("expected 'as' after embed path")
	}
	p.next() // consume `as`
	nameTok, err := p.expectName("expected binding name after embed ... as")
	if err != nil {
		return nil, err
	}
	transforms := map[string]bool{}
	if p.at(token.IDENT) && p.peek().Lexeme == "with" {
		p.next()
		if !p.match(token.LBRACE) {
			return nil, p.err("expected transform dictionary after embed with")
		}
		for !p.at(token.RBRACE) && !p.at(token.EOF) {
			keyTok, err := p.expectName("expected embed transform key")
			if err != nil {
				return nil, err
			}
			if !p.match(token.COLON) {
				return nil, p.err("expected ':' after embed transform key")
			}
			if !p.at(token.IDENT) || (p.peek().Lexeme != "true" && p.peek().Lexeme != "false") {
				return nil, p.err("embed transform values must be boolean literals")
			}
			valueTok := p.next()
			transforms[keyTok.Lexeme] = valueTok.Lexeme == "true"
			if !p.match(token.COMMA) {
				break
			}
		}
		if !p.match(token.RBRACE) {
			return nil, p.err("expected '}' after embed transform dictionary")
		}
	}
	if !p.at(token.NEWLINE) && !p.at(token.DEDENT) && !p.at(token.EOF) {
		return nil, p.err("expected newline after embed")
	}
	return &ast.EmbedStmt{Path: pathTok.Lexeme, PathTok: pathTok, Name: nameTok.Lexeme, NameTok: nameTok, Transforms: transforms}, nil
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
	if (p.at(token.IDENT) && p.peek().Lexeme == "extends") || p.at(token.LT) {
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
		// v0.68 canonical modifier order:
		//   [private] [static] [abstract|override] [@@] <name>: <body>
		isPrivateMember := false
		if p.at(token.IDENT) && p.peek().Lexeme == "private" && p.peekN(1).Type != token.QUESTION {
			isPrivateMember = true
			p.next()
		}
		// v0.46 G3: optional `static` keyword. Sets the same Class flag
		// as the legacy `@@` prefix; both forms coexist during the
		// transition.
		isStaticMember := false
		if p.at(token.IDENT) && p.peek().Lexeme == "static" && p.peekN(1).Type != token.ASSIGN {
			isStaticMember = true
			p.next()
		}
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
		isClassMember := isStaticMember || p.match(token.AT)
		if isClassMember && !isStaticMember {
			if !p.match(token.AT) {
				return nil, p.err("expected '@' for class member")
			}
		}
		memberName, err := p.expectCallableName("expected class member name")
		if err != nil {
			return nil, err
		}
		if p.match(token.ASSIGN) {
			return nil, p.err("class members use ':' declarations; replace '=' with ':'")
		}
		if !p.match(token.COLON) {
			return nil, p.err("expected ':' after class member name")
		}
		memberPrivate := isPrivateMember
		if isAbstractMethod {
			params, paramToks, err := p.abstractMethodParams()
			if err != nil {
				return nil, err
			}
			decl.Methods = append(decl.Methods, ast.ClassMethod{Name: memberName.Lexeme, Tok: memberName, Func: &ast.FuncLit{Params: params, ParamToks: paramToks}, Class: isClassMember, Abstract: true, Private: memberPrivate})
			p.skipNewlines()
			continue
		}
		value, err := p.exprLine()
		if err != nil {
			return nil, err
		}
		if funcLit, ok := value.(*ast.FuncLit); ok {
			decl.Methods = append(decl.Methods, ast.ClassMethod{Name: memberName.Lexeme, Tok: memberName, Func: funcLit, Class: isClassMember, Override: isOverrideMethod, Private: memberPrivate})
		} else if isOverrideMethod {
			return nil, p.err("override can only be used on methods")
		} else if !isClassMember && isConstName(memberName.Lexeme) {
			decl.Constants = append(decl.Constants, ast.ClassConst{Name: memberName.Lexeme, Tok: memberName, Value: value, Private: memberPrivate})
		} else if isClassMember {
			decl.Vars = append(decl.Vars, ast.ClassVar{Name: memberName.Lexeme, Tok: memberName, Value: value, Private: memberPrivate})
		} else {
			decl.Fields = append(decl.Fields, ast.ClassField{Name: memberName.Lexeme, Tok: memberName, Value: value, Private: memberPrivate})
		}
		p.skipNewlines()
	}
	if !p.match(token.DEDENT) {
		return nil, p.err("expected dedent after class")
	}
	return decl, nil
}

func isConstName(name string) bool {
	if name == "" || name[0] < 'A' || name[0] > 'Z' {
		return false
	}
	for i := 1; i < len(name); i++ {
		ch := name[i]
		if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			continue
		}
		return false
	}
	return true
}

func (p *Parser) interfaceDecl() (ast.Stmt, error) {
	p.next()
	name, err := p.expectName("expected interface name")
	if err != nil {
		return nil, err
	}
	var parents []ast.ClassRef
	if p.at(token.IDENT) && (p.peek().Lexeme == "extends" || p.peek().Lexeme == "implements") {
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
		// v0.46/47: `static` modifier in an interface body is rejected
		// with the canonical message (was: `@@name` via the AT path).
		if p.at(token.IDENT) && p.peek().Lexeme == "static" {
			return nil, p.err("interface bodies may only contain instance method requirements")
		}
		methodName, err := p.expectCallableName("expected interface method name")
		if err != nil {
			return nil, err
		}
		if p.match(token.ASSIGN) {
			return nil, p.err("interface members use ':' declarations; replace '=' with ':'")
		}
		if !p.match(token.COLON) {
			return nil, p.err("expected ':' after interface member name")
		}
		if p.startsInterfaceMethodValue() {
			params, paramToks, fn, err := p.interfaceMethodValue()
			if err != nil {
				return nil, err
			}
			decl.Methods = append(decl.Methods, ast.InterfaceMethod{Name: methodName.Lexeme, Tok: methodName, Params: params, ParamToks: paramToks, Func: fn})
		} else {
			value, err := p.exprLine()
			if err != nil {
				return nil, err
			}
			decl.Fields = append(decl.Fields, ast.ClassField{Name: methodName.Lexeme, Tok: methodName, Value: value})
		}
		p.skipNewlines()
	}
	if !p.match(token.DEDENT) {
		return nil, p.err("expected dedent after interface")
	}
	return decl, nil
}

func (p *Parser) startsInterfaceMethodValue() bool {
	if p.at(token.ARROW) {
		return true
	}
	if p.startsParenFunctionParams() || p.startsFunctionParams(true) {
		return true
	}
	return p.at(token.LPAREN) && p.peekN(1).Type == token.RPAREN && p.peekN(2).Type == token.ARROW
}

func (p *Parser) interfaceMethodValue() ([]string, []token.Token, *ast.FuncLit, error) {
	var params []string
	var paramToks []token.Token
	var defaults []ast.Expr
	if p.match(token.ARROW) {
		// no parameters
	} else if p.at(token.LPAREN) && p.peekN(1).Type == token.RPAREN && p.peekN(2).Type == token.ARROW {
		p.next()
		p.next()
		p.next()
	} else if p.startsParenFunctionParams() {
		var err error
		params, paramToks, defaults, err = p.parenFunctionParams()
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		var err error
		params, paramToks, defaults, err = p.functionParams()
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if !p.startsExpr() && !(p.at(token.NEWLINE) && p.peekN(1).Type == token.INDENT) {
		return params, paramToks, nil, nil
	}
	fn, err := p.finishFunc(params, paramToks, defaults)
	return params, paramToks, fn, err
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
	checkNoBody := func() ([]string, []token.Token, error) {
		if p.startsExpr() {
			return nil, nil, p.err("abstract methods cannot have bodies")
		}
		if p.at(token.NEWLINE) && p.peekN(1).Type == token.INDENT {
			return nil, nil, p.err("abstract methods cannot have bodies")
		}
		return params, paramToks, nil
	}
	if p.match(token.ARROW) {
		return checkNoBody()
	}
	if p.at(token.LPAREN) && p.peekN(1).Type == token.RPAREN && p.peekN(2).Type == token.ARROW {
		p.next()
		p.next()
		p.next()
		return checkNoBody()
	}
	if p.startsParenFunctionParams() {
		parsedParams, parsedToks, _, err := p.parenFunctionParams()
		if err != nil {
			return nil, nil, err
		}
		params = parsedParams
		paramToks = parsedToks
		return checkNoBody()
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
	return checkNoBody()
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
	return p.ifAfterKeyword()
}

func (p *Parser) ifAfterKeyword() (*ast.IfStmt, error) {
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
	return p.whileAfterKeyword()
}

func (p *Parser) whileAfterKeyword() (*ast.WhileStmt, error) {
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

func (p *Parser) scopeBlock() (ast.Stmt, error) {
	tok := p.next()
	body, err := p.block("scope")
	if err != nil {
		return nil, err
	}
	return &ast.ScopeBlock{Body: body, Tok: tok}, nil
}

func (p *Parser) selectStmt() (ast.Stmt, error) {
	tok := p.next()
	if !p.match(token.NEWLINE) || !p.match(token.INDENT) {
		return nil, p.err("expected indented block after select")
	}
	p.blockDepth++
	defer func() { p.blockDepth-- }()
	var arms []ast.SelectArm
	seenDefault := false
	p.skipNewlines()
	for !p.at(token.DEDENT) && !p.at(token.EOF) {
		arm, err := p.selectArm()
		if err != nil {
			return nil, err
		}
		if arm.Kind == "default" {
			if seenDefault {
				return nil, p.err("select may have at most one default arm")
			}
			seenDefault = true
		}
		arms = append(arms, arm)
		p.skipNewlines()
	}
	if !p.match(token.DEDENT) {
		return nil, p.err("expected dedent after select")
	}
	if len(arms) == 0 {
		return nil, p.err("select requires at least one arm")
	}
	return &ast.SelectStmt{Arms: arms, Tok: tok}, nil
}

func (p *Parser) selectArm() (ast.SelectArm, error) {
	tok := p.peek()
	if p.at(token.IDENT) && p.peek().Lexeme == "default" {
		tok = p.next()
		body, err := p.block("default")
		return ast.SelectArm{Kind: "default", Tok: tok, Body: body}, err
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "timeout" {
		tok = p.next()
		seconds, err := p.exprLine()
		if err != nil {
			return ast.SelectArm{}, err
		}
		body, err := p.block("timeout")
		return ast.SelectArm{Kind: "timeout", Tok: tok, Seconds: seconds, Body: body}, err
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "send" {
		tok = p.next()
		ch, err := p.exprNoCommaFunc()
		if err != nil {
			return ast.SelectArm{}, err
		}
		if !p.match(token.COMMA) {
			return ast.SelectArm{}, p.err("expected ',' after select send channel")
		}
		value, err := p.exprLine()
		if err != nil {
			return ast.SelectArm{}, err
		}
		body, err := p.block("send")
		return ast.SelectArm{Kind: "send", Tok: tok, Channel: ch, Value: value, Body: body}, err
	}
	if p.at(token.IDENT) && p.peekN(1).Type == token.ASSIGN && p.peekN(2).Type == token.IDENT && p.peekN(2).Lexeme == "receive" {
		bind := p.next()
		p.next()
		tok = p.next()
		ch, err := p.exprLine()
		if err != nil {
			return ast.SelectArm{}, err
		}
		body, err := p.block("receive")
		return ast.SelectArm{Kind: "receive", Tok: tok, BindName: bind.Lexeme, BindTok: bind, Channel: ch, Body: body}, err
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "receive" {
		tok = p.next()
		ch, err := p.exprLine()
		if err != nil {
			return ast.SelectArm{}, err
		}
		body, err := p.block("receive")
		return ast.SelectArm{Kind: "receive", Tok: tok, Channel: ch, Body: body}, err
	}
	return ast.SelectArm{}, p.err("expected select arm")
}

func (p *Parser) forStmt() (ast.Stmt, error) {
	p.next()
	return p.forAfterKeyword()
}

func (p *Parser) forAfterKeyword() (*ast.ForInStmt, error) {
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
	} else {
		return nil, p.err("expected 'in' in for loop")
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
	catchName := ""
	var catchTok token.Token
	var catchBody []ast.Stmt
	var finallyBody []ast.Stmt
	if p.at(token.IDENT) && p.peek().Lexeme == "catch" {
		ctok := p.next()
		if !p.at(token.IDENT) {
			return nil, p.errAt(ctok, "catch requires a binding name")
		}
		name := p.next()
		if p.at(token.COMMA) || p.at(token.LBRACKET) || p.at(token.LBRACE) {
			return nil, p.errAt(name, "catch binding must be a name")
		}
		if p.at(token.IDENT) && p.peek().Lexeme == "if" {
			return nil, p.err("catch filters are not part of Tya v1.0.0")
		}
		if !p.at(token.NEWLINE) && !p.at(token.DEDENT) && !p.at(token.EOF) {
			return nil, p.err("catch syntax is only catch err in Tya v1.0.0")
		}
		catchName = name.Lexeme
		catchTok = name
		catchBody, err = p.block("catch")
		if err != nil {
			return nil, err
		}
		p.skipNewlines()
		if p.at(token.IDENT) && p.peek().Lexeme == "catch" {
			return nil, p.err("multiple catch clauses are not part of Tya v1.0.0")
		}
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "finally" {
		p.next()
		finallyBody, err = p.block("finally")
		if err != nil {
			return nil, err
		}
	}
	if catchBody == nil && finallyBody == nil {
		return nil, p.errAt(tok, "block try requires catch or finally")
	}
	return &ast.TryCatchStmt{Try: body, CatchName: catchName, CatchTok: catchTok, Catch: catchBody, Finally: finallyBody, Tok: tok}, nil
}

func (p *Parser) matchStmt() (ast.Stmt, error) {
	tok := p.next()
	return p.matchAfterKeyword(tok)
}

func (p *Parser) matchAfterKeyword(tok token.Token) (*ast.MatchStmt, error) {
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
		if p.at(token.IDENT) && p.peek().Lexeme == "if" {
			return nil, p.err("match guards are not part of Tya v1.0.0")
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
			// Statement-level recovery (v0.54 multi-error parsing):
			// the diagnostic is already recorded in p.errs by
			// newDiag; skip to the next plausible statement and
			// keep parsing so we surface more diagnostics in one
			// pass.
			p.skipToNextStmt()
			p.skipNewlines()
			continue
		}
		stmts = append(stmts, s)
		p.skipNewlines()
	}
	if !p.match(token.DEDENT) {
		// Don't compound — record the error and let the caller
		// resume. If the loop already accumulated diagnostics this
		// keeps them all visible.
		_ = p.err("expected dedent after " + owner)
	}
	if len(stmts) == 0 {
		if os.Getenv("TYA_LEGACY_MODULES") != "1" {
			return nil, p.err("empty blocks are prohibited")
		}
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
		name := p.next()
		if name.Type != token.IDENT && name.Type != token.STRING {
			return nil, p.err("expected dictionary key")
		}
		if !p.match(token.COLON) {
			return nil, p.err("expected ':' after dictionary key")
		}
		var val ast.Expr
		var err error
		if p.match(token.ARROW) {
			val, err = p.finishFunc(nil, nil, nil)
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
	return p.exprLine()
}

func (p *Parser) exprLine() (ast.Expr, error) {
	ex, err := p.expr()
	if err != nil {
		return nil, err
	}
	continuedLine := p.continuedLine
	p.continuedLine = false
	if _, ok := ex.(*ast.FuncLit); ok {
		return ex, nil
	}
	switch ex.(type) {
	case *ast.IfStmt, *ast.WhileStmt, *ast.ForInStmt, *ast.MatchStmt:
		return ex, nil
	}
	if !continuedLine && p.startsExpr() {
		return nil, p.err("no-paren calls are not in Tya; use parentheses")
	}
	return ex, nil
}

func (p *Parser) expr() (ast.Expr, error) {
	return p.exprWithCommaFunc(true)
}

func (p *Parser) exprNoCommaFunc() (ast.Expr, error) {
	return p.exprWithCommaFunc(false)
}

func (p *Parser) exprWithCommaFunc(allowCommaFunc bool) (ast.Expr, error) {
	if p.startsFunctionParams(allowCommaFunc) {
		params, paramToks, defaults, err := p.functionParams()
		if err != nil {
			return nil, err
		}
		return p.finishFunc(params, paramToks, defaults)
	}
	if p.startsParenFunctionParams() {
		params, paramToks, defaults, err := p.parenFunctionParams()
		if err != nil {
			return nil, err
		}
		return p.finishFunc(params, paramToks, defaults)
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
		return p.finishFunc(params, paramToks, nil)
	}
	return left, nil
}

func (p *Parser) startsFunctionParams(allowCommaFunc bool) bool {
	if !p.at(token.IDENT) {
		return false
	}
	if p.peekN(1).Type == token.ARROW || p.peekN(1).Type == token.ASSIGN {
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
		case token.ASSIGN:
			i++
			depth := 0
			for i < len(p.toks) {
				switch p.toks[i].Type {
				case token.LPAREN, token.LBRACKET, token.LBRACE:
					depth++
				case token.RPAREN, token.RBRACKET, token.RBRACE:
					if depth == 0 {
						return false
					}
					depth--
				case token.COMMA:
					if depth == 0 {
						if p.toks[i+1].Type == token.ARROW {
							return true
						}
						if p.toks[i+1].Type != token.IDENT {
							return false
						}
						i++
						goto nextParam
					}
				case token.ARROW:
					if depth == 0 {
						return true
					}
				case token.NEWLINE, token.EOF:
					return false
				}
				i++
			}
			return false
		case token.COMMA:
			i++
			if p.toks[i].Type == token.ARROW {
				return true
			}
			if p.toks[i].Type != token.IDENT {
				return false
			}
		default:
			return false
		}
	nextParam:
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
		case token.ASSIGN:
			i++
			depth := 0
			for i < len(p.toks) {
				switch p.toks[i].Type {
				case token.LPAREN, token.LBRACKET, token.LBRACE:
					depth++
				case token.RPAREN:
					if depth == 0 {
						return p.toks[i+1].Type == token.ARROW
					}
					depth--
				case token.RBRACKET, token.RBRACE:
					if depth == 0 {
						return false
					}
					depth--
				case token.COMMA:
					if depth == 0 {
						if p.toks[i+1].Type == token.RPAREN {
							return p.toks[i+2].Type == token.ARROW
						}
						if p.toks[i+1].Type != token.IDENT {
							return false
						}
						i++
						goto nextParam
					}
				case token.NEWLINE, token.EOF:
					return false
				}
				i++
			}
			return false
		case token.COMMA:
			i++
			if p.toks[i].Type == token.RPAREN {
				return p.toks[i+1].Type == token.ARROW
			}
			if p.toks[i].Type != token.IDENT {
				return false
			}
		default:
			return false
		}
	nextParam:
	}
}

func (p *Parser) parenFunctionParams() ([]string, []token.Token, []ast.Expr, error) {
	if !p.match(token.LPAREN) {
		return nil, nil, nil, p.err("expected '('")
	}
	first, def, err := p.functionParam()
	if err != nil {
		return nil, nil, nil, err
	}
	params := []string{first.Lexeme}
	paramToks := []token.Token{first}
	defaults := []ast.Expr{def}
	for p.match(token.COMMA) {
		if p.at(token.RPAREN) {
			break
		}
		next, def, err := p.functionParam()
		if err != nil {
			return nil, nil, nil, err
		}
		params = append(params, next.Lexeme)
		paramToks = append(paramToks, next)
		defaults = append(defaults, def)
	}
	if !p.match(token.RPAREN) {
		return nil, nil, nil, p.err("expected ')'")
	}
	p.match(token.ARROW)
	return params, paramToks, defaults, nil
}

func (p *Parser) functionParams() ([]string, []token.Token, []ast.Expr, error) {
	first, def, err := p.functionParam()
	if err != nil {
		return nil, nil, nil, err
	}
	params := []string{first.Lexeme}
	paramToks := []token.Token{first}
	defaults := []ast.Expr{def}
	for p.match(token.COMMA) {
		if p.at(token.ARROW) {
			break
		}
		next, def, err := p.functionParam()
		if err != nil {
			return nil, nil, nil, err
		}
		params = append(params, next.Lexeme)
		paramToks = append(paramToks, next)
		defaults = append(defaults, def)
	}
	p.match(token.ARROW)
	return params, paramToks, defaults, nil
}

func (p *Parser) functionParam() (token.Token, ast.Expr, error) {
	name, err := p.expectName("expected function parameter")
	if err != nil {
		return token.Token{}, nil, err
	}
	if !p.match(token.ASSIGN) {
		return name, nil, nil
	}
	def, err := p.logicOr()
	if err != nil {
		return token.Token{}, nil, err
	}
	return name, def, nil
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
		return nil, p.err("try expressions are not part of Tya v1.0.0; use try as a statement")
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "spawn" {
		tok := p.next()
		ex, err := p.unary()
		if err != nil {
			return nil, err
		}
		return &ast.SpawnExpr{Callee: ex, Tok: tok}, nil
	}
	if p.at(token.IDENT) && p.peek().Lexeme == "await" {
		tok := p.next()
		ex, err := p.unary()
		if err != nil {
			return nil, err
		}
		return &ast.AwaitExpr{Target: ex, Tok: tok}, nil
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
	if p.at(token.IDENT) &&
		p.peekN(1).Type == token.LT &&
		p.peekN(2).Type == token.IDENT &&
		p.peekN(3).Type == token.GT &&
		(p.peekN(4).Type == token.LPAREN || p.peekN(4).Type == token.NEWLINE || p.peekN(4).Type == token.EOF) {
		return nil, p.err("generic type-parameter syntax is not part of Tya v1.0.0")
	}
	ex, err := p.primary()
	if err != nil {
		return nil, err
	}
	continuationIndents := 0
	for {
		if p.at(token.NEWLINE) && (p.peekN(1).Type == token.DOT || (p.peekN(1).Type == token.INDENT && p.peekN(2).Type == token.DOT)) {
			p.next()
			if p.at(token.INDENT) {
				p.next()
				continuationIndents++
			}
		}
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
			var callArgs []ast.CallArg
			seenKeyword := false
			if !p.at(token.RPAREN) {
				for {
					if p.at(token.IDENT) && p.peekN(1).Type == token.COLON {
						seenKeyword = true
						name := p.next()
						p.next()
						arg, err := p.exprNoCommaFunc()
						if err != nil {
							p.skipToCommaOrClose(token.RPAREN)
						} else {
							args = append(args, arg)
							callArgs = append(callArgs, ast.CallArg{Name: name.Lexeme, NameTok: name, Value: arg})
						}
					} else if p.at(token.STAR) && p.peekN(1).Type == token.STAR {
						seenKeyword = true
						p.next()
						p.next()
						arg, err := p.exprNoCommaFunc()
						if err != nil {
							p.skipToCommaOrClose(token.RPAREN)
						} else {
							args = append(args, arg)
							callArgs = append(callArgs, ast.CallArg{Value: arg, Expand: true})
						}
					} else {
						if seenKeyword {
							return nil, p.err("positional argument after keyword argument")
						}
						if p.at(token.STAR) {
							return nil, p.err("splat call syntax is not part of Tya v1.0.0")
						}
						arg, err := p.exprNoCommaFunc()
						if err != nil {
							// v0.56 expression-level recovery: the
							// diagnostic is already recorded in
							// p.errs by p.err/newDiag — skip to the
							// next ',' or ')' and continue with the
							// remaining args.
							p.skipToCommaOrClose(token.RPAREN)
						} else {
							args = append(args, arg)
							callArgs = append(callArgs, ast.CallArg{Value: arg})
						}
					}
					if !p.match(token.COMMA) {
						break
					}
					if p.at(token.RPAREN) {
						break
					}
				}
			}
			if !p.match(token.RPAREN) {
				return nil, p.err("expected ')'")
			}
			ex = &ast.CallExpr{Callee: ex, Args: args, CallArgs: callArgs}
			continue
		}
		if p.match(token.LBRACKET) {
			index, err := p.expr()
			if err != nil {
				return nil, err
			}
			if p.at(token.COLON) {
				return nil, p.err("slice syntax is not part of Tya v1.0.0; use .slice(...)")
			}
			if !p.match(token.RBRACKET) {
				return nil, p.err("expected ']'")
			}
			ex = &ast.IndexExpr{Target: ex, Index: index}
			continue
		}
		break
	}
	for continuationIndents > 0 && p.at(token.NEWLINE) && p.peekN(1).Type == token.DEDENT {
		p.next()
		p.next()
		continuationIndents--
		p.continuedLine = true
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
		switch tok.Lexeme {
		case "if":
			return p.ifAfterKeyword()
		case "while":
			return p.whileAfterKeyword()
		case "for":
			return p.forAfterKeyword()
		case "match":
			return p.matchAfterKeyword(tok)
		}
		if p.peek().Type == token.QUESTION || p.peek().Type == token.BANG {
			p.next()
			tok.Lexeme += p.prev().Lexeme
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
		// v0.46 G2: `Self` (capital S) refers to the declaring class
		// in any class-body context. Statically resolved.
		if tok.Lexeme == "Self" {
			return &ast.SelfExpr{Tok: tok, Class: true}, nil
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
		return &ast.StringLit{Value: tok.Lexeme, Form: tok.StringForm, Lang: tok.Lang, Marker: tok.Marker}, nil
	case token.BYTES:
		return &ast.BytesLit{Value: tok.Lexeme, Form: tok.StringForm, Marker: tok.Marker}, nil
	case token.LPAREN:
		if p.match(token.RPAREN) {
			if p.match(token.ARROW) {
				return p.finishFunc(nil, nil, nil)
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
					// v0.56 expression-level recovery: diag already
					// recorded; skip to next ',' or ']'.
					p.skipToCommaOrClose(token.RBRACKET)
				} else {
					elems = append(elems, elem)
				}
				if !p.match(token.COMMA) {
					break
				}
				if p.at(token.RBRACKET) {
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
		return p.finishFunc(nil, nil, nil)
	}
	return nil, p.err("expected expression")
}

func (p *Parser) curlyLiteral() (ast.Expr, error) {
	if p.match(token.RBRACE) {
		return &ast.DictLit{}, nil
	}
	if p.dictLiteralKeyAhead() {
		return p.dictLiteral()
	}
	return nil, p.err("set literals are not in Tya v0.1")
}

func (p *Parser) dictLiteral() (ast.Expr, error) {
	dict := &ast.DictLit{}
	for {
		if !p.dictLiteralKeyAhead() {
			return nil, p.err("cannot mix dict entries and set entries in one literal")
		}
		name := p.next()
		p.next()
		value, err := p.expr()
		if err != nil {
			// v0.56 expression-level recovery: diag already
			// recorded; skip to next ',' or '}' and continue.
			p.skipToCommaOrClose(token.RBRACE)
		} else {
			dict.Props = append(dict.Props, ast.DictProp{Name: name.Lexeme, Tok: name, Value: value})
		}
		if !p.match(token.COMMA) {
			break
		}
		if p.at(token.RBRACE) {
			break
		}
	}
	if !p.match(token.RBRACE) {
		return nil, p.err("expected '}'")
	}
	return dict, nil
}

func (p *Parser) dictLiteralKeyAhead() bool {
	return (p.at(token.IDENT) || p.at(token.STRING)) && p.peekN(1).Type == token.COLON
}

func (p *Parser) finishFunc(params []string, paramToks []token.Token, defaults []ast.Expr) (*ast.FuncLit, error) {
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
		if len(body) == 0 {
			if os.Getenv("TYA_LEGACY_MODULES") != "1" {
				return nil, p.err("empty blocks are prohibited")
			}
		}
		return &ast.FuncLit{Params: params, ParamToks: paramToks, Defaults: defaults, Body: body}, nil
	}
	ex, err := p.exprLine()
	if err != nil {
		return nil, err
	}
	return &ast.FuncLit{Params: params, ParamToks: paramToks, Defaults: defaults, Expr: ex}, nil
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
		next, err := p.assignTarget()
		if err != nil {
			return nil, err
		}
		targets = append(targets, next)
	}
	return targets, nil
}

func (p *Parser) assignTarget() (ast.Expr, error) {
	var ex ast.Expr
	if p.at(token.LBRACKET) {
		return nil, p.err("destructuring assignment is not part of Tya v1.0.0; use multi-return assignment")
	}
	if p.at(token.LBRACE) {
		return nil, p.err("destructuring assignment is not part of Tya v1.0.0; use multi-return assignment")
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
	} else if p.at(token.IDENT) && p.peek().Lexeme == "Self" {
		// v0.46 G2: `Self.foo = ...` writes to a class-level member.
		tok := p.next()
		ex = &ast.SelfExpr{Tok: tok, Class: true}
	} else if p.at(token.IDENT) && p.peek().Lexeme == "self" {
		// v0.46 G2: `self.foo = ...` writes through the receiver.
		tok := p.next()
		ex = &ast.SelfExpr{Tok: tok}
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
			if p.at(token.COLON) {
				return nil, p.err("slice syntax is not part of Tya v1.0.0; use .slice(...)")
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
			return nil, p.errAt(tok, "binding patterns are not part of Tya v1.0.0")
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
		return &ast.StringLit{Value: tok.Lexeme, Form: tok.StringForm, Lang: tok.Lang, Marker: tok.Marker}, nil
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
	for i < len(p.toks) && p.toks[i].Type == token.COMMA {
		i++
		if !p.scanAssignTarget(&i) {
			return false
		}
	}
	return i < len(p.toks) && p.toks[i].Type == token.ASSIGN
}

func (p *Parser) scanAssignTarget(i *int) bool {
	if *i >= len(p.toks) {
		return false
	}
	if p.toks[*i].Type == token.LBRACKET || p.toks[*i].Type == token.LBRACE {
		return p.scanBalancedPattern(i)
	}
	if p.toks[*i].Type == token.AT {
		*i++
		if *i >= len(p.toks) {
			return false
		}
		if p.toks[*i].Type == token.AT {
			*i++
		}
		if *i >= len(p.toks) {
			return false
		}
		if p.toks[*i].Type != token.IDENT {
			return false
		}
		*i++
		if *i < len(p.toks) && (p.toks[*i].Type == token.QUESTION || p.toks[*i].Type == token.BANG) {
			*i++
		}
		return true
	}
	if p.toks[*i].Type != token.IDENT {
		return false
	}
	*i++
	if *i < len(p.toks) && (p.toks[*i].Type == token.QUESTION || p.toks[*i].Type == token.BANG) {
		*i++
	}
	for {
		if *i >= len(p.toks) {
			return false
		}
		if p.toks[*i].Type == token.DOT {
			*i += 2
			if *i >= len(p.toks) {
				return false
			}
			if p.toks[*i].Type == token.QUESTION || p.toks[*i].Type == token.BANG {
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
	if p.at(token.QUESTION) || p.at(token.BANG) {
		tok.Lexeme += p.next().Lexeme
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
	if p.at(token.QUESTION) || p.at(token.BANG) {
		tok.Lexeme += p.next().Lexeme
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
func (p *Parser) rejectV1ExcludedStatementSyntax() error {
	if !p.at(token.IDENT) {
		return nil
	}
	switch p.peek().Lexeme {
	case "enum", "record", "struct", "macro", "async", "protected", "friend", "defer", "assert", "cancel":
		if p.peekN(1).Type != token.IDENT {
			return nil
		}
		return p.err(p.peek().Lexeme + " syntax is not part of Tya v1.0.0")
	default:
		return nil
	}
}
func (p *Parser) rejectReservedName(tok token.Token) error {
	switch tok.Lexeme {
	case "object", "set":
		return p.errAt(tok, tok.Lexeme+" is not in Tya v0.1")
	case "interface", "class", "self", "Self", "true", "false", "nil", "if", "elseif", "else", "while", "for", "in", "break", "continue", "return", "try", "catch", "finally", "module", "import", "embed", "and", "or", "not", "spawn", "await", "scope":
		return p.errAt(tok, tok.Lexeme+" cannot be used as a name")
	default:
		return nil
	}
}
func (p *Parser) err(msg string) error {
	tok := p.peek()
	return p.errAt(tok, msg)
}

// errAt is the v0.54 structured-diagnostic shim. It still returns
// an `error` (so call sites stay unchanged) but the underlying
// value is a *ParserError carrying a diag.Diagnostic, with the
// flat `line:col: msg near "tok"` formatting preserved for
// legacy callers via ParserError.Error().
func (p *Parser) errAt(tok token.Token, msg string) error {
	full := fmt.Sprintf("%s near %q", msg, tok.Lexeme)
	code := classifyParserCode(msg)
	return p.newDiag(code, msg, full, tok.Line, tok.Col)
}
