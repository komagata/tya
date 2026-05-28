// AST-driven formatted syntax serializer.
//
// v0.37 ships the foundation for this serializer. It handles the
// common subset of the AST (imports, simple assignments, expression
// statements, returns/raises/break/continue, single-line `if`,
// `while`, `for`, plus single-line function expressions and the
// usual literal / operator / call / member / index expressions).
//
// Constructs that are not yet handled — modules, classes, match,
// try/catch, multi-line function bodies, dict / array literals
// long enough to exceed 80 columns, the `"""..."""` rewrite rule —
// return an "unsupported" error so callers can fall back to the
// existing text formatter. v0.38 extends this coverage and
// introduces the wrap rules.
//
// The serializer is intentionally not yet wired into `tya fmt`. It
// is exported for tests and for opt-in tooling.

package formatter

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"tya/internal/ast"
)

var constNameRE = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// Unparse renders prog as formatted Tya source. When prog was
// produced by parser.ParseWithComments, header and per-statement
// comments are emitted per docs/SPEC.md Formatted Syntax.
//
// Top-level statements follow the §3.5 / §8.4 normalization rules:
// imports are sorted alphabetically and grouped (stdlib first, then
// user); a blank line separates each top-level definition (module,
// class, interface, function-typed assignment) from the previous
// statement; otherwise no blank lines are emitted.
func Unparse(prog *ast.Program) (string, error) {
	u := &unparser{comments: prog.Comments}
	if len(prog.HeaderComments) > 0 {
		for _, c := range prog.HeaderComments {
			u.line("#" + c)
		}
		u.b.WriteByte('\n')
	}
	stmts := canonicalizeTopLevel(prog.Stmts)
	for i, stmt := range stmts {
		if i > 0 && requiresBlankBefore(stmt, stmts[i-1]) {
			u.b.WriteByte('\n')
		}
		if err := u.stmt(stmt); err != nil {
			return "", err
		}
	}
	return u.b.String(), nil
}

// canonicalizeTopLevel reorders top-level import runs per §8.4:
// imports are sorted alphabetically with stdlib imports first, then
// user imports. Non-import statements keep their original order.
func canonicalizeTopLevel(stmts []ast.Stmt) []ast.Stmt {
	out := make([]ast.Stmt, 0, len(stmts))
	i := 0
	for i < len(stmts) {
		if !isImportStmt(stmts[i]) {
			out = append(out, stmts[i])
			i++
			continue
		}
		j := i
		var imports []*ast.ImportStmt
		for j < len(stmts) {
			if !isImportStmt(stmts[j]) {
				break
			}
			imports = append(imports, flattenImportStmt(stmts[j])...)
			j++
		}
		sortImports(imports)
		for _, imp := range imports {
			out = append(out, imp)
		}
		i = j
	}
	return out
}

func isImportStmt(stmt ast.Stmt) bool {
	switch stmt.(type) {
	case *ast.ImportStmt, *ast.ImportBlockStmt:
		return true
	default:
		return false
	}
}

func flattenImportStmt(stmt ast.Stmt) []*ast.ImportStmt {
	switch n := stmt.(type) {
	case *ast.ImportStmt:
		return []*ast.ImportStmt{n}
	case *ast.ImportBlockStmt:
		return n.Imports
	default:
		return nil
	}
}

func sortImports(imports []*ast.ImportStmt) {
	sort.SliceStable(imports, func(a, b int) bool {
		ia, ib := imports[a], imports[b]
		sa, sb := isStdlibImport(ia.Name), isStdlibImport(ib.Name)
		if sa != sb {
			return sa
		}
		return ia.BindingName() < ib.BindingName()
	})
}

func importEntryString(imp *ast.ImportStmt) string {
	name := imp.Name
	if imp.Wildcard {
		name += "/*"
	}
	if imp.Alias != "" {
		return fmt.Sprintf("%s as %s", name, imp.Alias)
	}
	return name
}

// stdlibModules is the set of names recognized as stdlib for
// import grouping. Kept in sync with lib/*.tya.
var stdlibModules = map[string]bool{
	"array":         true,
	"base64":        true,
	"csv":           true,
	"dict":          true,
	"digest":        true,
	"dir":           true,
	"file":          true,
	"hex":           true,
	"json":          true,
	"markdown":      true,
	"math":          true,
	"os":            true,
	"path":          true,
	"process":       true,
	"random":        true,
	"secure_random": true,
	"string":        true,
	"time":          true,
	"unittest":      true,
	"url":           true,
	"matrix":        true,
}

func isStdlibImport(name string) bool {
	// Strip any leading path components — stdlib names are always
	// bare module names (no slashes).
	if strings.ContainsAny(name, "/.") {
		return false
	}
	return stdlibModules[name]
}

// requiresBlankBefore reports whether canonical layout puts a blank
// line between prev and cur at the top level. Per §3.5 / §8.4: a
// blank line goes between two top-level definitions and between an
// import and any following non-import statement. Consecutive imports
// always stay in one import block. All other transitions get no blank
// line.
func requiresBlankBefore(cur, prev ast.Stmt) bool {
	prevImport := isImportStmt(prev)
	curImport := isImportStmt(cur)
	if prevImport && curImport {
		return false
	}
	if prevImport && !curImport {
		return true
	}
	if isTopDefinition(prev) && isTopDefinition(cur) {
		return true
	}
	return false
}

func isTopDefinition(s ast.Stmt) bool {
	switch n := s.(type) {
	case *ast.ModuleDecl, *ast.ClassDecl, *ast.InterfaceDecl:
		return true
	case *ast.AssignStmt:
		if len(n.Values) == 1 {
			if _, ok := n.Values[0].(*ast.FuncLit); ok {
				return true
			}
		}
	}
	return false
}

type unparser struct {
	b        strings.Builder
	indent   int
	comments map[ast.Stmt]ast.StmtComments
	// v0.48 G6: name of the class currently being unparsed, used to
	// canonicalize same-class receiver method calls inside its own body.
	// Empty outside class bodies.
	currentClass       string
	currentClassMethod bool
}

// columnLimit is the canonical wrap target (CANONICAL §5.1).
const columnLimit = 80

func (u *unparser) currentIndent() int { return u.indent * 2 }

// fitsInline reports whether body fits on a single line at the
// current indent (no trailing comment).
func (u *unparser) fitsInline(body string) bool {
	return u.currentIndent()+len(body) <= columnLimit
}

// preStmt writes leading comments for stmt, when any are attached.
func (u *unparser) preStmt(stmt ast.Stmt) {
	if u.comments == nil {
		return
	}
	sc, ok := u.comments[stmt]
	if !ok {
		return
	}
	for _, c := range sc.Leading {
		u.line("#" + c)
	}
}

func (u *unparser) leadingComments(comments ast.StmtComments) {
	for _, c := range comments.Leading {
		u.line("#" + c)
	}
}

// trailingFor returns the line-end comment text for stmt, or "" when
// none. The caller is responsible for concatenating it onto the
// emitted line.
func (u *unparser) trailingFor(stmt ast.Stmt) string {
	if u.comments == nil {
		return ""
	}
	if sc, ok := u.comments[stmt]; ok {
		return sc.LineEnd
	}
	return ""
}

// emitStmtLine writes a one-line statement, appending its line-end
// comment when set.
func (u *unparser) emitStmtLine(stmt ast.Stmt, body string) {
	if t := u.trailingFor(stmt); t != "" {
		u.line(body + "  #" + t)
		return
	}
	u.line(body)
}

func (u *unparser) line(s string) {
	u.b.WriteString(strings.Repeat("  ", u.indent))
	u.b.WriteString(s)
	u.b.WriteByte('\n')
}

func (u *unparser) stmt(s ast.Stmt) error {
	u.preStmt(s)
	switch n := s.(type) {
	case *ast.ImportStmt:
		u.emitStmtLine(s, "import "+importEntryString(n))
		return nil
	case *ast.ImportBlockStmt:
		u.line("import")
		u.indent++
		for _, imp := range n.Imports {
			u.line(importEntryString(imp))
		}
		u.indent--
		return nil
	case *ast.EmbedStmt:
		body := fmt.Sprintf("embed %q as %s", n.Path, n.Name)
		if len(n.Transforms) > 0 {
			keys := make([]string, 0, len(n.Transforms))
			for k := range n.Transforms {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			parts := make([]string, 0, len(keys))
			for _, k := range keys {
				parts = append(parts, fmt.Sprintf("%s: %v", k, n.Transforms[k]))
			}
			body += " with { " + strings.Join(parts, ", ") + " }"
		}
		u.emitStmtLine(s, body)
		return nil
	case *ast.AssignStmt:
		return u.assignStmt(n)
	case *ast.ExprStmt:
		ex, err := u.expr(n.Expr)
		if err != nil {
			return err
		}
		// Wrap a top-level CallExpr if the single-line form
		// exceeds the column limit.
		if call, ok := n.Expr.(*ast.CallExpr); ok && !u.fitsInline(ex) {
			if u.isWrappedCallChain(call) {
				return u.emitWrappedCallChain(s, "", call)
			}
			return u.emitWrappedCall(s, "", call)
		}
		u.emitStmtLine(s, ex)
		return nil
	case *ast.ReturnStmt:
		if len(n.Values) == 0 {
			u.emitStmtLine(s, "return")
			return nil
		}
		if len(n.Values) == 1 && u.isControlExpr(n.Values[0]) {
			return u.controlExpr("return ", n.Values[0])
		}
		parts := make([]string, 0, len(n.Values))
		for _, v := range n.Values {
			s, err := u.expr(v)
			if err != nil {
				return err
			}
			parts = append(parts, s)
		}
		u.emitStmtLine(s, "return "+strings.Join(parts, ", "))
		return nil
	case *ast.RaiseStmt:
		ex, err := u.expr(n.Value)
		if err != nil {
			return err
		}
		u.emitStmtLine(s, "raise "+ex)
		return nil
	case *ast.BreakStmt:
		u.emitStmtLine(s, "break")
		return nil
	case *ast.ContinueStmt:
		u.emitStmtLine(s, "continue")
		return nil
	case *ast.IfStmt:
		return u.ifStmt(n)
	case *ast.WhileStmt:
		cond, err := u.expr(n.Cond)
		if err != nil {
			return err
		}
		u.emitCondHeader("while", cond)
		return u.block(n.Body)
	case *ast.ForInStmt:
		iter, err := u.expr(n.Iterable)
		if err != nil {
			return err
		}
		head := "for " + n.ValueName
		if n.IndexName != "" {
			head += ", " + n.IndexName
		}
		kind := n.Kind
		if kind == "" {
			kind = "in"
		}
		u.line(head + " " + kind + " " + iter)
		return u.block(n.Body)
	case *ast.MatchStmt:
		return u.matchStmt(n)
	case *ast.TryCatchStmt:
		return u.tryStmt(n)
	case *ast.ScopeBlock:
		u.line("scope")
		return u.block(n.Body)
	case *ast.SelectStmt:
		u.line("select")
		u.indent++
		for _, arm := range n.Arms {
			if err := u.selectArm(arm); err != nil {
				return err
			}
		}
		u.indent--
		return nil
	case *ast.ModuleDecl:
		return u.moduleDecl(n)
	case *ast.ClassDecl:
		return u.classDecl(n)
	case *ast.InterfaceDecl:
		return u.interfaceDecl(n)
	}
	return fmt.Errorf("formatter.Unparse: stmt %T not supported", s)
}

func (u *unparser) selectArm(arm ast.SelectArm) error {
	switch arm.Kind {
	case "receive":
		ch, err := u.expr(arm.Channel)
		if err != nil {
			return err
		}
		if arm.BindName != "" {
			u.line(arm.BindName + " = receive " + ch)
		} else {
			u.line("receive " + ch)
		}
	case "send":
		ch, err := u.expr(arm.Channel)
		if err != nil {
			return err
		}
		value, err := u.expr(arm.Value)
		if err != nil {
			return err
		}
		u.line("send " + ch + ", " + value)
	case "timeout":
		seconds, err := u.expr(arm.Seconds)
		if err != nil {
			return err
		}
		u.line("timeout " + seconds)
	case "default":
		u.line("default")
	}
	return u.block(arm.Body)
}

func (u *unparser) matchStmt(n *ast.MatchStmt) error {
	val, err := u.expr(n.Value)
	if err != nil {
		return err
	}
	u.line("match " + val)
	u.indent++
	defer func() { u.indent-- }()
	for _, c := range n.Cases {
		pat, err := u.expr(c.Pattern)
		if err != nil {
			return err
		}
		u.line("case " + pat)
		if err := u.block(c.Body); err != nil {
			return err
		}
	}
	return nil
}

func (u *unparser) tryStmt(n *ast.TryCatchStmt) error {
	u.line("try")
	if err := u.block(n.Try); err != nil {
		return err
	}
	if n.CatchName != "" {
		u.line("catch " + n.CatchName)
		if err := u.block(n.Catch); err != nil {
			return err
		}
	} else if len(n.Catch) > 0 {
		u.line("catch _")
		if err := u.block(n.Catch); err != nil {
			return err
		}
	}
	if len(n.Finally) > 0 {
		u.line("finally")
		return u.block(n.Finally)
	}
	return nil
}

func (u *unparser) moduleDecl(n *ast.ModuleDecl) error {
	u.line("module " + n.Name)
	u.indent++
	defer func() { u.indent-- }()
	first := true
	for _, m := range n.Members {
		if !first {
			u.b.WriteByte('\n')
		}
		first = false
		if err := u.moduleMember(m); err != nil {
			return err
		}
	}
	for _, c := range n.Classes {
		if !first {
			u.b.WriteByte('\n')
		}
		first = false
		if err := u.classDecl(c); err != nil {
			return err
		}
	}
	for _, i := range n.Interfaces {
		if !first {
			u.b.WriteByte('\n')
		}
		first = false
		if err := u.interfaceDecl(i); err != nil {
			return err
		}
	}
	return nil
}

func (u *unparser) moduleMember(m ast.ModuleMember) error {
	if fn, ok := m.Value.(*ast.FuncLit); ok && fn.Expr == nil && len(fn.Body) > 0 {
		head, err := u.funcDefHead(fn)
		if err != nil {
			return err
		}
		prefix := m.Name + " = " + defArrow(head)
		if expr, ok, err := u.singleExprBody(fn.Body); err != nil {
			return err
		} else if ok {
			body, err := u.expr(expr)
			if err != nil {
				return err
			}
			line := prefix + " " + body
			if u.fitsInline(line) {
				u.line(line)
				return nil
			}
		}
		u.line(prefix)
		return u.funcBlock(fn.Body)
	}
	val, err := u.expr(m.Value)
	if err != nil {
		return err
	}
	u.line(m.Name + " = " + val)
	return nil
}

func (u *unparser) classDecl(n *ast.ClassDecl) error {
	// v0.48 G6: track the declaring class so same-class receiver calls
	// can be canonicalized inside their own body.
	prevClass := u.currentClass
	u.currentClass = n.Name
	defer func() { u.currentClass = prevClass }()
	head := "class " + n.Name
	if n.Abstract {
		head = "abstract " + head
	}
	if n.Final {
		head = "final " + head
	}
	if n.Parent != nil {
		head += " extends " + classRefString(*n.Parent)
	}
	if len(n.Implements) > 0 {
		impls := make([]string, 0, len(n.Implements))
		for _, r := range n.Implements {
			impls = append(impls, classRefString(r))
		}
		head += " implements " + strings.Join(impls, ", ")
	}
	u.line(head)
	u.indent++
	defer func() { u.indent-- }()
	members := canonicalClassMembers(n)
	for i, member := range members {
		if i > 0 {
			u.b.WriteByte('\n')
		}
		if err := member.emit(u); err != nil {
			return err
		}
	}
	return nil
}

type classMember struct {
	name string
	rank int
	emit func(*unparser) error
}

func canonicalClassMembers(n *ast.ClassDecl) []classMember {
	members := make([]classMember, 0, len(n.Constants)+len(n.Vars)+len(n.Fields)+len(n.Methods))
	for _, c := range n.Constants {
		c := c
		rank := 0
		if c.Private {
			rank = 1
		}
		members = append(members, classMember{name: c.Name, rank: rank, emit: func(u *unparser) error {
			return u.classConst(c)
		}})
	}
	for _, v := range n.Vars {
		v := v
		rank := 2
		if v.Private {
			rank = 3
		}
		members = append(members, classMember{name: v.Name, rank: rank, emit: func(u *unparser) error {
			return u.classVar(v)
		}})
	}
	for _, f := range n.Fields {
		f := f
		rank := 4
		if f.Private {
			rank = 5
		}
		members = append(members, classMember{name: f.Name, rank: rank, emit: func(u *unparser) error {
			return u.classField(f)
		}})
	}
	for _, m := range n.Methods {
		m := m
		rank := classMethodRank(m)
		members = append(members, classMember{name: m.Name, rank: rank, emit: func(u *unparser) error {
			return u.classMethod(m)
		}})
	}
	sort.SliceStable(members, func(i, j int) bool {
		if members[i].rank != members[j].rank {
			return members[i].rank < members[j].rank
		}
		return members[i].name < members[j].name
	})
	return members
}

func classMethodRank(m ast.ClassMethod) int {
	if m.Class {
		if m.Private {
			return 7
		}
		return 6
	}
	if m.Name == "initialize" {
		return 8
	}
	if m.Private {
		return 10
	}
	return 9
}

func (u *unparser) classField(f ast.ClassField) error {
	u.leadingComments(f.Comments)
	val, err := u.expr(f.Value)
	if err != nil {
		return err
	}
	// v0.46+ canonical: `private` keyword instead of `_`-prefix.
	fieldPrefix := ""
	if f.Private {
		fieldPrefix = "private "
	}
	u.line(fieldPrefix + f.Name + ": " + val)
	return nil
}

func (u *unparser) classVar(v ast.ClassVar) error {
	u.leadingComments(v.Comments)
	val, err := u.expr(v.Value)
	if err != nil {
		return err
	}
	// v0.46+ canonical: `static` keyword instead of `@@`; `private`
	// keyword instead of `_`-prefix.
	varPrefix := "static "
	if v.Private {
		varPrefix = "private static "
	}
	u.line(varPrefix + v.Name + ": " + val)
	return nil
}

func (u *unparser) classConst(c ast.ClassConst) error {
	u.leadingComments(c.Comments)
	val, err := u.expr(c.Value)
	if err != nil {
		return err
	}
	prefix := ""
	if c.Private {
		prefix = "private "
	}
	u.line(prefix + c.Name + ": " + val)
	return nil
}

func (u *unparser) classMethod(m ast.ClassMethod) error {
	u.leadingComments(m.Comments)
	prevClassMethod := u.currentClassMethod
	u.currentClassMethod = m.Class
	defer func() { u.currentClassMethod = prevClassMethod }()

	// v0.46+ canonical: keyword modifiers in canonical order
	// (`private static abstract|override`). Legacy `@@` is removed.
	parts := []string{}
	if m.Private {
		parts = append(parts, "private")
	}
	if m.Class {
		parts = append(parts, "static")
	}
	if m.Abstract {
		parts = append(parts, "abstract")
	}
	if m.Override {
		parts = append(parts, "override")
	}
	prefix := strings.Join(parts, " ")
	if prefix != "" {
		prefix += " "
	}
	if m.Abstract {
		head, err := u.funcDefHead(m.Func)
		if err != nil {
			return err
		}
		u.line(prefix + m.Name + ": " + defArrow(head))
		return nil
	}
	fn := m.Func
	head, err := u.funcDefHead(fn)
	if err != nil {
		return err
	}
	if fn.Expr != nil {
		body, err := u.expr(fn.Expr)
		if err != nil {
			return err
		}
		line := prefix + m.Name + ": " + defArrow(head) + " " + body
		if u.fitsInline(line) {
			u.line(line)
			return nil
		}
		u.line(prefix + m.Name + ": " + defArrow(head))
		u.indent++
		u.line(body)
		u.indent--
		return nil
	}
	linePrefix := prefix + m.Name + ": " + defArrow(head)
	if fn.Expr == nil {
		if expr, ok, err := u.singleExprBody(fn.Body); err != nil {
			return err
		} else if ok {
			body, err := u.expr(expr)
			if err != nil {
				return err
			}
			line := linePrefix + " " + body
			if u.fitsInline(line) {
				u.line(line)
				return nil
			}
		}
	}
	u.line(linePrefix)
	return u.funcBlock(fn.Body)
}

func (u *unparser) interfaceDecl(n *ast.InterfaceDecl) error {
	head := "interface " + n.Name
	if len(n.Parents) > 0 {
		parents := make([]string, 0, len(n.Parents))
		for _, r := range n.Parents {
			parents = append(parents, classRefString(r))
		}
		head += " extends " + strings.Join(parents, ", ")
	}
	u.line(head)
	u.indent++
	defer func() { u.indent-- }()
	for _, f := range n.Fields {
		// Keep interface members adjacent. A blank line after an empty
		// abstract signature can be parsed as an empty default body
		// boundary and change checker semantics.
		u.leadingComments(f.Comments)
		value, err := u.expr(f.Value)
		if err != nil {
			return err
		}
		u.line(f.Name + ": " + value)
	}
	for _, m := range n.Methods {
		// Keep interface members adjacent for the same reason as
		// fields above.
		u.leadingComments(m.Comments)
		head := m.Name + ": " + defArrow(strings.Join(m.Params, ", "))
		if m.Func == nil {
			u.line(head)
			continue
		}
		if m.Func.Expr != nil {
			body, err := u.expr(m.Func.Expr)
			if err != nil {
				return err
			}
			line := head + " " + body
			if u.fitsInline(line) {
				u.line(line)
				continue
			}
			u.line(head)
			u.indent++
			u.line(body)
			u.indent--
			continue
		}
		if m.Func.Expr == nil {
			if expr, ok, err := u.singleExprBody(m.Func.Body); err != nil {
				return err
			} else if ok {
				body, err := u.expr(expr)
				if err != nil {
					return err
				}
				line := head + " " + body
				if u.fitsInline(line) {
					u.line(line)
					continue
				}
			}
		}
		u.line(head)
		if err := u.block(m.Func.Body); err != nil {
			return err
		}
	}
	return nil
}

func classRefString(r ast.ClassRef) string {
	if r.Module != "" {
		return r.Module + "." + r.Name
	}
	return r.Name
}

func (u *unparser) assignStmt(n *ast.AssignStmt) error {
	if len(n.Targets) == 0 {
		return fmt.Errorf("formatter.Unparse: empty AssignStmt")
	}
	targets := make([]string, 0, len(n.Targets))
	for _, t := range n.Targets {
		s, err := u.expr(t)
		if err != nil {
			return err
		}
		targets = append(targets, s)
	}
	// FuncLit value with a block body needs special handling: we
	// emit `name = params -> ...` and recurse into the block.
	if len(n.Values) == 1 {
		if u.isControlExpr(n.Values[0]) {
			return u.controlExpr(strings.Join(targets, ", ")+" = ", n.Values[0])
		}
		if fn, ok := n.Values[0].(*ast.FuncLit); ok {
			head, err := u.funcDefHead(fn)
			if err != nil {
				return err
			}
			prefix := strings.Join(targets, ", ") + " = " + defArrow(head)
			if fn.Expr != nil {
				body, err := u.expr(fn.Expr)
				if err != nil {
					return err
				}
				line := prefix + " " + body
				if u.fitsInline(line) {
					u.emitStmtLine(n, line)
					return nil
				}
			} else if len(fn.Body) > 0 {
				if expr, ok, err := u.singleExprBody(fn.Body); err != nil {
					return err
				} else if ok {
					body, err := u.expr(expr)
					if err != nil {
						return err
					}
					line := prefix + " " + body
					if u.fitsInline(line) {
						u.emitStmtLine(n, line)
						return nil
					}
				}
				u.line(prefix)
				return u.funcBlock(fn.Body)
			}
		}
		if lit, ok := n.Values[0].(*ast.StringLit); ok && strings.Contains(lit.Form, "heredoc") {
			return u.emitHeredoc(n, strings.Join(targets, ", ")+" = ", lit)
		}
		if lit, ok := n.Values[0].(*ast.BytesLit); ok && strings.Contains(lit.Form, "heredoc") {
			return u.emitBytesHeredoc(n, strings.Join(targets, ", ")+" = ", lit)
		}
		if lit, ok := n.Values[0].(*ast.StringLit); ok && lit.Lang != "" && lit.Form == "triple" {
			return u.emitTripleQuoted(n, strings.Join(targets, ", ")+" = ", lit)
		}
	}
	values := make([]string, 0, len(n.Values))
	for _, v := range n.Values {
		s, err := u.expr(v)
		if err != nil {
			return err
		}
		values = append(values, s)
	}
	body := strings.Join(targets, ", ") + " = " + strings.Join(values, ", ")
	if u.fitsInline(body) {
		u.emitStmtLine(n, body)
		return nil
	}
	// Single-line too long. Wrap the rhs based on its shape.
	if len(n.Values) == 1 {
		switch v := n.Values[0].(type) {
		case *ast.CallExpr:
			if u.isWrappedCallChain(v) {
				return u.emitWrappedCallChain(n, strings.Join(targets, ", ")+" = ", v)
			}
			return u.emitWrappedCall(n, strings.Join(targets, ", ")+" = ", v)
		case *ast.ArrayLit:
			return u.emitWrappedArray(n, strings.Join(targets, ", ")+" = ", v)
		case *ast.BinaryExpr:
			return u.emitWrappedBinary(n, strings.Join(targets, ", ")+" = ", v)
		case *ast.DictLit:
			return u.emitDictBlock(n, strings.Join(targets, ", "), v)
		case *ast.StringLit:
			// §6.3: long single-line string with embedded
			// newlines rewrites to """ ... """. Strings
			// without \n or containing literal `"""` fall
			// back to the atomic-token exception (long
			// single-line is OK).
			if strings.Contains(v.Value, "\n") && !strings.Contains(v.Value, `"""`) {
				return u.emitTripleQuoted(n, strings.Join(targets, ", ")+" = ", v)
			}
		case *ast.FuncLit:
			// §5.3.8: a single-line `name -> expr` whose
			// rendering exceeds 80 columns switches to the
			// block-body form `name ->\n  expr`.
			if v.Expr != nil {
				head, err := u.funcDefHead(v)
				if err != nil {
					return err
				}
				bodyStr, err := u.expr(v.Expr)
				if err != nil {
					return err
				}
				u.line(strings.Join(targets, ", ") + " = " + defArrow(head))
				u.indent++
				u.line(bodyStr)
				u.indent--
				return nil
			}
		}
	}
	u.emitStmtLine(n, body)
	return nil
}

type callChainStep struct {
	name string
	args []ast.Expr
}

func (u *unparser) isWrappedCallChain(call *ast.CallExpr) bool {
	_, steps, ok := u.callChain(call)
	return ok && len(steps) >= 2
}

func (u *unparser) callChain(call *ast.CallExpr) (ast.Expr, []callChainStep, bool) {
	var reversed []callChainStep
	var current ast.Expr = call
	for {
		c, ok := current.(*ast.CallExpr)
		if !ok {
			break
		}
		member, ok := c.Callee.(*ast.MemberExpr)
		if !ok {
			break
		}
		reversed = append(reversed, callChainStep{name: member.Name, args: c.Args})
		current = member.Target
	}
	if len(reversed) == 0 {
		return nil, nil, false
	}
	steps := make([]callChainStep, len(reversed))
	for i := range reversed {
		steps[len(reversed)-1-i] = reversed[i]
	}
	return current, steps, true
}

func (u *unparser) emitWrappedCallChain(stmt ast.Stmt, prefix string, call *ast.CallExpr) error {
	base, steps, ok := u.callChain(call)
	if !ok {
		return u.emitWrappedCall(stmt, prefix, call)
	}
	baseText, err := u.expr(base)
	if err != nil {
		return err
	}
	u.line(prefix + baseText)
	u.indent++
	for _, step := range steps {
		args := make([]string, 0, len(step.args))
		for _, arg := range step.args {
			text, err := u.expr(arg)
			if err != nil {
				u.indent--
				return err
			}
			args = append(args, text)
		}
		u.line("." + step.name + "(" + strings.Join(args, ", ") + ")")
	}
	u.indent--
	if t := u.trailingFor(stmt); t != "" {
		u.line("#" + t)
	}
	return nil
}

// emitWrappedCall emits a call as the multi-line form per §5.3.1:
//
//	prefix callee(
//	  arg1,
//	  arg2,
//	  ...
//	)
//
// prefix is "" for an ExprStmt or "name = " for an AssignStmt rhs.
func (u *unparser) emitWrappedCall(stmt ast.Stmt, prefix string, call *ast.CallExpr) error {
	callee, err := u.expr(call.Callee)
	if err != nil {
		return err
	}
	u.line(prefix + callee + "(")
	u.indent++
	for i, a := range call.Args {
		s, err := u.expr(a)
		if err != nil {
			u.indent--
			return err
		}
		if i < len(call.Args)-1 {
			s += ","
		}
		u.line(s)
	}
	u.indent--
	closing := ")"
	if t := u.trailingFor(stmt); t != "" {
		u.line(closing + "  #" + t)
	} else {
		u.line(closing)
	}
	return nil
}

// emitWrappedBinary emits a left-associative binary chain in the
// leading-operator style per CANONICAL §5.3.5:
//
//	prefix a
//	  + b
//	  + c
//
// The chain is flattened by repeated left descent. Operators that
// would change associativity (mixed precedence) fall back to the
// long single-line under the atomic-token exception.
func (u *unparser) emitWrappedBinary(stmt ast.Stmt, prefix string, expr *ast.BinaryExpr) error {
	// Flatten left-associative same-operator chain. We collect
	// operands and the operators between them.
	rootOp := expr.Op.Lexeme
	var operands []ast.Expr
	var ops []string
	var walk func(e ast.Expr)
	walk = func(e ast.Expr) {
		if be, ok := e.(*ast.BinaryExpr); ok && be.Op.Lexeme == rootOp {
			walk(be.Left)
			ops = append(ops, be.Op.Lexeme)
			operands = append(operands, be.Right)
			return
		}
		operands = append(operands, e)
	}
	walk(expr)
	if len(operands) < 2 {
		// Mixed or unsupported chain — fall through to inline.
		body, err := u.expr(expr)
		if err != nil {
			return err
		}
		u.emitStmtLine(stmt, prefix+body)
		return nil
	}
	first, err := u.expr(operands[0])
	if err != nil {
		return err
	}
	u.line(prefix + first)
	u.indent++
	for i := 1; i < len(operands); i++ {
		s, err := u.expr(operands[i])
		if err != nil {
			u.indent--
			return err
		}
		u.line(ops[i-1] + " " + s)
	}
	u.indent--
	return nil
}

// binaryOperand renders an operand of a binary expression and
// wraps it in parentheses when its precedence is lower than the
// parent operator's. left=true means the operand is the left side
// (which can stay un-parenthesized for same-precedence
// left-associative operators).
func (u *unparser) binaryOperand(e ast.Expr, parentOp string, left bool) (string, error) {
	s, err := u.expr(e)
	if err != nil {
		return "", err
	}
	be, ok := e.(*ast.BinaryExpr)
	if !ok {
		return s, nil
	}
	pp := opPrecedence(parentOp)
	cp := opPrecedence(be.Op.Lexeme)
	if cp < pp {
		return "(" + s + ")", nil
	}
	if cp == pp && !left {
		// Right operand of left-associative operator at same
		// precedence needs parens to preserve grouping.
		return "(" + s + ")", nil
	}
	return s, nil
}

// opPrecedence returns the precedence of a binary operator. Higher
// numbers bind tighter. Mirrors the parser's precedence ordering.
func opPrecedence(op string) int {
	switch op {
	case "or":
		return 1
	case "and":
		return 2
	case "==", "!=":
		return 3
	case "<", "<=", ">", ">=":
		return 4
	case "|":
		return 5
	case "^":
		return 6
	case "&":
		return 7
	case "<<", ">>":
		return 8
	case "+", "-":
		return 9
	case "*", "/", "%":
		return 10
	}
	return 0
}

// emitTripleQuoted emits a string literal as a triple-quoted
// multi-line literal per CANONICAL §6.3. The closing """ sits at
// the current indent + 2; each body line is also at +2 indent and
// the lexer's normalization step strips that baseline back out.
//
//	target = """
//	  first line
//	  second line
//	  """
func (u *unparser) emitTripleQuoted(stmt ast.Stmt, prefix string, lit *ast.StringLit) error {
	u.line(prefix + lit.Lang + `"""`)
	u.indent++
	defer func() { u.indent-- }()
	// The triple-quoted body ends with a newline by virtue of
	// the closing `"""` being on its own line. Drop a single
	// trailing empty segment from the split so re-parsing the
	// emitted form (which adds that trailing newline back to
	// StringLit.Value) is idempotent.
	body := lit.Value
	if strings.HasSuffix(body, "\n") {
		body = strings.TrimSuffix(body, "\n")
	}
	for _, line := range strings.Split(body, "\n") {
		encoded := strings.ReplaceAll(line, `\`, `\\`)
		u.line(encoded)
	}
	u.line(`"""`)
	return nil
}

func (u *unparser) emitHeredoc(stmt ast.Stmt, prefix string, lit *ast.StringLit) error {
	_ = stmt
	marker := lit.Marker
	if marker == "" {
		marker = "TXT"
	}
	head := prefix
	if lit.Form == "raw_heredoc" {
		head += "r"
	} else {
		head += lit.Lang
	}
	u.line(head + "<<<" + marker)
	u.indent++
	body := lit.Value
	if strings.HasSuffix(body, "\n") {
		body = strings.TrimSuffix(body, "\n")
	}
	for _, line := range strings.Split(body, "\n") {
		u.line(line)
	}
	u.line(marker)
	u.indent--
	return nil
}

func (u *unparser) emitBytesHeredoc(stmt ast.Stmt, prefix string, lit *ast.BytesLit) error {
	_ = stmt
	marker := lit.Marker
	if marker == "" {
		marker = "BYTES"
	}
	u.line(prefix + "b<<<" + marker)
	u.indent++
	body := lit.Value
	if strings.HasSuffix(body, "\n") {
		body = strings.TrimSuffix(body, "\n")
	}
	for _, line := range strings.Split(body, "\n") {
		u.line(line)
	}
	u.line(marker)
	u.indent--
	return nil
}

// emitDictBlock emits a dict literal as the §5.3.3 block form:
//
//	target =
//	  key: value
//	  key: value
//
// Used when the inline `target = { k: v, ... }` form would
// exceed the column limit. The parser already accepts this form
// (parser.valuesAfterAssign treats `=` followed by NEWLINE INDENT
// as a dictBody).
func (u *unparser) emitDictBlock(stmt ast.Stmt, target string, dict *ast.DictLit) error {
	u.line(target + " =")
	u.indent++
	defer func() { u.indent-- }()
	for _, p := range dict.Props {
		v, err := u.expr(p.Value)
		if err != nil {
			return err
		}
		u.line(p.Name + ": " + v)
	}
	return nil
}

// emitWrappedArray emits an array literal as the multi-line form
// per §5.3.2.
func (u *unparser) emitWrappedArray(stmt ast.Stmt, prefix string, arr *ast.ArrayLit) error {
	u.line(prefix + "[")
	u.indent++
	for i, el := range arr.Elems {
		s, err := u.expr(el)
		if err != nil {
			u.indent--
			return err
		}
		if i < len(arr.Elems)-1 {
			s += ","
		}
		u.line(s)
	}
	u.indent--
	closing := "]"
	if t := u.trailingFor(stmt); t != "" {
		u.line(closing + "  #" + t)
	} else {
		u.line(closing)
	}
	return nil
}

func (u *unparser) ifStmt(n *ast.IfStmt) error {
	return u.ifStmtWithPrefix("", n)
}

func (u *unparser) ifStmtWithPrefix(prefix string, n *ast.IfStmt) error {
	cond, err := u.expr(n.Cond)
	if err != nil {
		return err
	}
	u.emitCondHeader(prefix+"if", cond)
	if err := u.block(n.Then); err != nil {
		return err
	}
	if len(n.Else) == 0 {
		return nil
	}
	// If else is a single IfStmt, emit `elseif ...`.
	if len(n.Else) == 1 {
		if inner, ok := n.Else[0].(*ast.IfStmt); ok {
			cond2, err := u.expr(inner.Cond)
			if err != nil {
				return err
			}
			u.emitCondHeader("elseif", cond2)
			if err := u.block(inner.Then); err != nil {
				return err
			}
			if len(inner.Else) > 0 {
				u.line("else")
				return u.blockBody(inner.Else)
			}
			return nil
		}
	}
	u.line("else")
	return u.blockBody(n.Else)
}

func (u *unparser) isControlExpr(e ast.Expr) bool {
	switch e.(type) {
	case *ast.IfStmt, *ast.WhileStmt, *ast.ForInStmt, *ast.MatchStmt:
		return true
	default:
		return false
	}
}

func (u *unparser) controlExpr(prefix string, e ast.Expr) error {
	switch n := e.(type) {
	case *ast.IfStmt:
		return u.ifStmtWithPrefix(prefix, n)
	case *ast.WhileStmt:
		cond, err := u.expr(n.Cond)
		if err != nil {
			return err
		}
		u.emitCondHeader(prefix+"while", cond)
		return u.block(n.Body)
	case *ast.ForInStmt:
		iter, err := u.expr(n.Iterable)
		if err != nil {
			return err
		}
		head := prefix + "for " + n.ValueName
		if n.IndexName != "" {
			head += ", " + n.IndexName
		}
		kind := n.Kind
		if kind == "" {
			kind = "in"
		}
		u.emitCondHeader(head+" "+kind, iter)
		return u.block(n.Body)
	case *ast.MatchStmt:
		value, err := u.expr(n.Value)
		if err != nil {
			return err
		}
		u.line(prefix + "match " + value)
		u.indent++
		for _, c := range n.Cases {
			pat, err := u.expr(c.Pattern)
			if err != nil {
				return err
			}
			u.line("case " + pat)
			if err := u.block(c.Body); err != nil {
				u.indent--
				return err
			}
		}
		u.indent--
		return nil
	default:
		return fmt.Errorf("formatter.Unparse: unsupported control expression %T", e)
	}
}

// emitCondHeader writes `keyword cond` on a single line, falling
// back to the parenthesized multi-line form (CANONICAL §5.3.6) when
// the inline form exceeds the column limit. The wrapped form is
//
//	keyword (
//	  cond
//	)
//
// In v0.38 this is a single-condition wrap; binary-chain
// leading-operator wrap of the inner condition arrives in §5.3.5.
func (u *unparser) emitCondHeader(keyword, cond string) {
	inline := keyword + " " + cond
	if u.fitsInline(inline) {
		u.line(inline)
		return
	}
	u.line(keyword + " (")
	u.indent++
	u.line(cond)
	u.indent--
	u.line(")")
}

func (u *unparser) block(stmts []ast.Stmt) error {
	u.indent++
	defer func() { u.indent-- }()
	for i, s := range stmts {
		if i > 0 && u.hasLeadingComments(s) {
			u.b.WriteByte('\n')
		}
		if err := u.stmt(s); err != nil {
			return err
		}
	}
	return nil
}

func (u *unparser) hasLeadingComments(s ast.Stmt) bool {
	if u.comments == nil {
		return false
	}
	sc, ok := u.comments[s]
	return ok && len(sc.Leading) > 0
}

func (u *unparser) hasAnyComments(s ast.Stmt) bool {
	if u.comments == nil {
		return false
	}
	sc, ok := u.comments[s]
	return ok && (len(sc.Leading) > 0 || sc.LineEnd != "")
}

func (u *unparser) singleExprBody(stmts []ast.Stmt) (ast.Expr, bool, error) {
	if len(stmts) != 1 || u.hasAnyComments(stmts[0]) {
		return nil, false, nil
	}
	switch n := stmts[0].(type) {
	case *ast.ExprStmt:
		return n.Expr, true, nil
	case *ast.ReturnStmt:
		if len(n.Values) != 1 {
			return nil, false, nil
		}
		return n.Values[0], true, nil
	default:
		return nil, false, nil
	}
}

func (u *unparser) blockBody(stmts []ast.Stmt) error {
	return u.block(stmts)
}

func (u *unparser) funcBlock(stmts []ast.Stmt) error {
	if len(stmts) == 0 {
		return u.block(stmts)
	}
	u.indent++
	for _, st := range stmts[:len(stmts)-1] {
		if err := u.stmt(st); err != nil {
			u.indent--
			return err
		}
	}
	err := u.funcTailStmt(stmts[len(stmts)-1])
	u.indent--
	return err
}

func (u *unparser) funcTailStmt(st ast.Stmt) error {
	switch n := st.(type) {
	case *ast.ReturnStmt:
		if len(n.Values) != 1 {
			return u.stmt(st)
		}
		if u.hasAnyComments(st) {
			return u.stmt(st)
		}
		ex, err := u.expr(n.Values[0])
		if err != nil {
			return err
		}
		u.emitStmtLine(&ast.ExprStmt{Expr: n.Values[0]}, ex)
		return nil
	case *ast.IfStmt:
		return u.ifStmtTail(n)
	default:
		return u.stmt(st)
	}
}

func (u *unparser) tailBlock(stmts []ast.Stmt) error {
	if len(stmts) == 0 {
		return nil
	}
	u.indent++
	for _, st := range stmts[:len(stmts)-1] {
		if err := u.stmt(st); err != nil {
			u.indent--
			return err
		}
	}
	err := u.funcTailStmt(stmts[len(stmts)-1])
	u.indent--
	return err
}

func (u *unparser) ifStmtTail(n *ast.IfStmt) error {
	cond, err := u.expr(n.Cond)
	if err != nil {
		return err
	}
	u.emitCondHeader("if", cond)
	if err := u.tailBlock(n.Then); err != nil {
		return err
	}
	if len(n.Else) == 0 {
		return nil
	}
	if len(n.Else) == 1 {
		if inner, ok := n.Else[0].(*ast.IfStmt); ok {
			cond2, err := u.expr(inner.Cond)
			if err != nil {
				return err
			}
			u.emitCondHeader("elseif", cond2)
			if err := u.tailBlock(inner.Then); err != nil {
				return err
			}
			if len(inner.Else) > 0 {
				u.line("else")
				return u.tailBlock(inner.Else)
			}
			return nil
		}
	}
	u.line("else")
	return u.tailBlock(n.Else)
}

func (u *unparser) funcHead(fn *ast.FuncLit) (string, error) {
	if len(fn.Params) == 0 {
		return "()", nil
	}
	parts := make([]string, len(fn.Params))
	for i, param := range fn.Params {
		parts[i] = param
		if i < len(fn.Defaults) && fn.Defaults[i] != nil {
			def, err := u.expr(fn.Defaults[i])
			if err != nil {
				return "", err
			}
			parts[i] += " = " + def
		}
	}
	if len(parts) == 1 {
		return parts[0], nil
	}
	return strings.Join(parts, ", "), nil
}

func (u *unparser) funcDefHead(fn *ast.FuncLit) (string, error) {
	if len(fn.Params) == 0 {
		return "", nil
	}
	return u.funcHead(fn)
}

func defArrow(head string) string {
	if head == "" {
		return "->"
	}
	return head + " ->"
}

func (u *unparser) expr(e ast.Expr) (string, error) {
	switch n := e.(type) {
	case *ast.Ident:
		return n.Name, nil
	case *ast.IntLit:
		return strconv.FormatInt(n.Value, 10), nil
	case *ast.FloatLit:
		s := strconv.FormatFloat(n.Value, 'f', -1, 64)
		if !strings.ContainsAny(s, ".eE") {
			s += ".0"
		}
		return s, nil
	case *ast.StringLit:
		if n.Lang != "" && n.Form == "triple" && !strings.Contains(n.Value, `"""`) {
			return n.Lang + `"""` + n.Value + `"""`, nil
		}
		return strconv.Quote(n.Value), nil
	case *ast.BoolLit:
		if n.Value {
			return "true", nil
		}
		return "false", nil
	case *ast.NilLit:
		return "nil", nil
	case *ast.SelfExpr:
		if n.Class {
			return "Self", nil
		}
		return "self", nil
	case *ast.SuperExpr:
		return "super", nil
	case *ast.BinaryExpr:
		left, err := u.binaryOperand(n.Left, n.Op.Lexeme, true)
		if err != nil {
			return "", err
		}
		right, err := u.binaryOperand(n.Right, n.Op.Lexeme, false)
		if err != nil {
			return "", err
		}
		return left + " " + n.Op.Lexeme + " " + right, nil
	case *ast.TryExpr:
		inner, err := u.expr(n.Expr)
		if err != nil {
			return "", err
		}
		return "try " + inner, nil
	case *ast.SpawnExpr:
		inner, err := u.expr(n.Callee)
		if err != nil {
			return "", err
		}
		return "spawn " + inner, nil
	case *ast.AwaitExpr:
		inner, err := u.expr(n.Target)
		if err != nil {
			return "", err
		}
		return "await " + inner, nil
	case *ast.UnaryExpr:
		inner, err := u.expr(n.Expr)
		if err != nil {
			return "", err
		}
		// Parenthesize binary operands inside unary so
		// `not (a or b)` does not flatten to `not a or b`.
		if _, ok := n.Expr.(*ast.BinaryExpr); ok {
			inner = "(" + inner + ")"
		}
		op := n.Op.Lexeme
		if op == "not" {
			return "not " + inner, nil
		}
		return op + inner, nil
	case *ast.CallExpr:
		if name, ok := u.bareSameClassCallName(n.Callee); ok {
			args := make([]string, 0, len(n.Args))
			for _, a := range n.Args {
				s, err := u.expr(a)
				if err != nil {
					return "", err
				}
				args = append(args, s)
			}
			return name + "(" + strings.Join(args, ", ") + ")", nil
		}
		callee, err := u.postfixTarget(n.Callee)
		if err != nil {
			return "", err
		}
		args := make([]string, 0, len(n.Args))
		for _, a := range n.Args {
			s, err := u.expr(a)
			if err != nil {
				return "", err
			}
			args = append(args, s)
		}
		return callee + "(" + strings.Join(args, ", ") + ")", nil
	case *ast.MemberExpr:
		if name, ok := u.bareSameClassConstantName(n); ok {
			return name, nil
		}
		// v0.48 G6 canonical receiver: `<DeclaringClass>.foo` inside
		// its own class body is rewritten to `Self.foo` for non-constant
		// non-call member expressions.
		if u.currentClass != "" {
			if ident, ok := n.Target.(*ast.Ident); ok && ident.Name == u.currentClass {
				return "Self." + n.Name, nil
			}
		}
		target, err := u.postfixTarget(n.Target)
		if err != nil {
			return "", err
		}
		return target + "." + n.Name, nil
	case *ast.IndexExpr:
		target, err := u.postfixTarget(n.Target)
		if err != nil {
			return "", err
		}
		index, err := u.expr(n.Index)
		if err != nil {
			return "", err
		}
		return target + "[" + index + "]", nil
	case *ast.ArrayLit:
		parts := make([]string, 0, len(n.Elems))
		for _, el := range n.Elems {
			s, err := u.expr(el)
			if err != nil {
				return "", err
			}
			parts = append(parts, s)
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	case *ast.DictLit:
		if len(n.Props) == 0 {
			return "{}", nil
		}
		parts := make([]string, 0, len(n.Props))
		for _, p := range n.Props {
			s, err := u.expr(p.Value)
			if err != nil {
				return "", err
			}
			parts = append(parts, p.Name+": "+s)
		}
		return "{ " + strings.Join(parts, ", ") + " }", nil
	case *ast.FuncLit:
		head, err := u.funcHead(n)
		if err != nil {
			return "", err
		}
		// Single-line lambda only at expression level. Block-bodied
		// FuncLits at expression level (e.g. inside a call) are
		// out of v0.37 scope.
		if n.Expr != nil {
			body, err := u.expr(n.Expr)
			if err != nil {
				return "", err
			}
			return head + " -> " + body, nil
		}
		if len(n.Body) == 0 {
			return head + " -> nil", nil
		}
		return "", fmt.Errorf("formatter.Unparse: block-bodied function expression at expr position not supported in v0.37")
	}
	return "", fmt.Errorf("formatter.Unparse: expr %T not supported in v0.37", e)
}

func (u *unparser) bareSameClassCallName(e ast.Expr) (string, bool) {
	member, ok := e.(*ast.MemberExpr)
	if !ok || u.currentClass == "" {
		return "", false
	}
	switch target := member.Target.(type) {
	case *ast.SelfExpr:
		if target.Class {
			return member.Name, true
		}
		return member.Name, !u.currentClassMethod
	case *ast.Ident:
		if target.Name == "self" {
			return member.Name, !u.currentClassMethod
		}
		if target.Name == "Self" || target.Name == u.currentClass {
			return member.Name, true
		}
	}
	return "", false
}

func (u *unparser) bareSameClassConstantName(e *ast.MemberExpr) (string, bool) {
	if u.currentClass == "" || !constNameRE.MatchString(e.Name) {
		return "", false
	}
	switch target := e.Target.(type) {
	case *ast.SelfExpr:
		if target.Class {
			return e.Name, true
		}
	case *ast.Ident:
		if target.Name == "Self" || target.Name == u.currentClass {
			return e.Name, true
		}
	}
	return "", false
}

func (u *unparser) postfixTarget(e ast.Expr) (string, error) {
	s, err := u.expr(e)
	if err != nil {
		return "", err
	}
	switch e.(type) {
	case *ast.BinaryExpr, *ast.UnaryExpr, *ast.TryExpr, *ast.SpawnExpr, *ast.AwaitExpr, *ast.FuncLit:
		return "(" + s + ")", nil
	default:
		return s, nil
	}
}
