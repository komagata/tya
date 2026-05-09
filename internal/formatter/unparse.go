// AST-driven canonical serializer (Canonical Syntax §11.3).
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
	"sort"
	"strconv"
	"strings"

	"tya/internal/ast"
)

// Unparse renders prog as canonical Tya source. When prog was
// produced by parser.ParseWithComments, header and per-statement
// comments are emitted per docs/CANONICAL_SYNTAX.md §3.
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

// canonicalizeTopLevel reorders top-level statements per §8.4: any
// run of consecutive ImportStmts is sorted alphabetically with
// stdlib imports first, then user imports. Non-import statements
// keep their original order.
func canonicalizeTopLevel(stmts []ast.Stmt) []ast.Stmt {
	out := make([]ast.Stmt, 0, len(stmts))
	i := 0
	for i < len(stmts) {
		if _, ok := stmts[i].(*ast.ImportStmt); !ok {
			out = append(out, stmts[i])
			i++
			continue
		}
		j := i
		var imports []*ast.ImportStmt
		for j < len(stmts) {
			imp, ok := stmts[j].(*ast.ImportStmt)
			if !ok {
				break
			}
			imports = append(imports, imp)
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

// stdlibModules is the set of names recognized as stdlib for
// import grouping. Kept in sync with stdlib/*.tya.
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
// blank line goes between two top-level definitions, between an
// import and any following non-import statement, and at the
// stdlib/user import-group boundary. All other transitions get no
// blank line.
func requiresBlankBefore(cur, prev ast.Stmt) bool {
	pi, prevImport := prev.(*ast.ImportStmt)
	ci, curImport := cur.(*ast.ImportStmt)
	if prevImport && curImport {
		return isStdlibImport(pi.Name) != isStdlibImport(ci.Name)
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
		if n.Alias != "" {
			u.emitStmtLine(s, fmt.Sprintf("import %s as %s", n.Name, n.Alias))
		} else {
			u.emitStmtLine(s, fmt.Sprintf("import %s", n.Name))
		}
		return nil
	case *ast.AssignStmt:
		return u.assignStmt(n)
	case *ast.ExprStmt:
		ex, err := u.expr(n.Expr)
		if err != nil {
			return err
		}
		// Render `print foo` and `assert ...` keyword forms
		// faithfully. The parser accepts them; the AST stores them
		// as ExprStmt with a CallExpr whose callee is the keyword
		// ident. We emit the canonical keyword form.
		if call, ok := n.Expr.(*ast.CallExpr); ok {
			if id, ok := call.Callee.(*ast.Ident); ok {
				switch id.Name {
				case "print", "assert", "assert_equal":
					args := make([]string, 0, len(call.Args))
					for _, a := range call.Args {
						s, err := u.expr(a)
						if err != nil {
							return err
						}
						args = append(args, s)
					}
					u.emitStmtLine(s, id.Name+" "+strings.Join(args, ", "))
					return nil
				}
			}
		}
		// Wrap a top-level CallExpr if the single-line form
		// exceeds the column limit.
		if call, ok := n.Expr.(*ast.CallExpr); ok && !u.fitsInline(ex) {
			return u.emitWrappedCall(s, "", call)
		}
		u.emitStmtLine(s, ex)
		return nil
	case *ast.ReturnStmt:
		if len(n.Values) == 0 {
			u.emitStmtLine(s, "return")
			return nil
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
	case *ast.ModuleDecl:
		return u.moduleDecl(n)
	case *ast.ClassDecl:
		return u.classDecl(n)
	case *ast.InterfaceDecl:
		return u.interfaceDecl(n)
	}
	return fmt.Errorf("formatter.Unparse: stmt %T not supported", s)
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
	} else {
		u.line("catch _")
	}
	return u.block(n.Catch)
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
		head, err := u.funcHead(fn)
		if err != nil {
			return err
		}
		u.line(m.Name + " = " + head + " ->")
		return u.block(fn.Body)
	}
	val, err := u.expr(m.Value)
	if err != nil {
		return err
	}
	u.line(m.Name + " = " + val)
	return nil
}

func (u *unparser) classDecl(n *ast.ClassDecl) error {
	head := "class " + n.Name
	if n.Abstract {
		head = "abstract " + head
	}
	if n.Final {
		head = "final " + head
	}
	if n.Parent != nil {
		head += " < " + classRefString(*n.Parent)
	}
	if len(n.Implements) > 0 {
		impls := make([]string, 0, len(n.Implements))
		for _, r := range n.Implements {
			impls = append(impls, classRefString(r))
		}
		head += " : " + strings.Join(impls, ", ")
	}
	u.line(head)
	u.indent++
	defer func() { u.indent-- }()
	first := true
	for _, f := range n.Fields {
		if !first {
			u.b.WriteByte('\n')
		}
		first = false
		val, err := u.expr(f.Value)
		if err != nil {
			return err
		}
		u.line(f.Name + " = " + val)
	}
	for _, v := range n.Vars {
		if !first {
			u.b.WriteByte('\n')
		}
		first = false
		val, err := u.expr(v.Value)
		if err != nil {
			return err
		}
		u.line("@@" + v.Name + " = " + val)
	}
	for _, m := range n.Methods {
		if !first {
			u.b.WriteByte('\n')
		}
		first = false
		if err := u.classMethod(m); err != nil {
			return err
		}
	}
	return nil
}

func (u *unparser) classMethod(m ast.ClassMethod) error {
	prefix := ""
	if m.Class {
		prefix = "class."
	}
	if m.Abstract {
		head, err := u.funcHead(m.Func)
		if err != nil {
			return err
		}
		u.line("abstract " + prefix + m.Name + " = " + head)
		return nil
	}
	fn := m.Func
	head, err := u.funcHead(fn)
	if err != nil {
		return err
	}
	if fn.Expr != nil {
		body, err := u.expr(fn.Expr)
		if err != nil {
			return err
		}
		u.line(prefix + m.Name + " = " + head + " -> " + body)
		return nil
	}
	u.line(prefix + m.Name + " = " + head + " ->")
	return u.block(fn.Body)
}

func (u *unparser) interfaceDecl(n *ast.InterfaceDecl) error {
	head := "interface " + n.Name
	if len(n.Parents) > 0 {
		parents := make([]string, 0, len(n.Parents))
		for _, r := range n.Parents {
			parents = append(parents, classRefString(r))
		}
		head += " < " + strings.Join(parents, ", ")
	}
	u.line(head)
	u.indent++
	defer func() { u.indent-- }()
	for _, m := range n.Methods {
		params := strings.Join(m.Params, ", ")
		u.line(m.Name + " = " + params)
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
		if fn, ok := n.Values[0].(*ast.FuncLit); ok && fn.Expr == nil && len(fn.Body) > 0 {
			head, err := u.funcHead(fn)
			if err != nil {
				return err
			}
			u.line(strings.Join(targets, ", ") + " = " + head + " ->")
			return u.block(fn.Body)
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
			return u.emitWrappedCall(n, strings.Join(targets, ", ")+" = ", v)
		case *ast.ArrayLit:
			return u.emitWrappedArray(n, strings.Join(targets, ", ")+" = ", v)
		case *ast.FuncLit:
			// §5.3.8: a single-line `name -> expr` whose
			// rendering exceeds 80 columns switches to the
			// block-body form `name ->\n  expr`.
			if v.Expr != nil {
				head, err := u.funcHead(v)
				if err != nil {
					return err
				}
				bodyStr, err := u.expr(v.Expr)
				if err != nil {
					return err
				}
				u.line(strings.Join(targets, ", ") + " = " + head + " ->")
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
	for _, a := range call.Args {
		s, err := u.expr(a)
		if err != nil {
			u.indent--
			return err
		}
		u.line(s + ",")
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

// emitWrappedArray emits an array literal as the multi-line form
// per §5.3.2.
func (u *unparser) emitWrappedArray(stmt ast.Stmt, prefix string, arr *ast.ArrayLit) error {
	u.line(prefix + "[")
	u.indent++
	for _, el := range arr.Elems {
		s, err := u.expr(el)
		if err != nil {
			u.indent--
			return err
		}
		u.line(s + ",")
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
	cond, err := u.expr(n.Cond)
	if err != nil {
		return err
	}
	u.emitCondHeader("if", cond)
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

func (u *unparser) blockBody(stmts []ast.Stmt) error {
	return u.block(stmts)
}

func (u *unparser) funcHead(fn *ast.FuncLit) (string, error) {
	if len(fn.Params) == 0 {
		return "()", nil
	}
	if len(fn.Params) == 1 {
		return fn.Params[0], nil
	}
	return strings.Join(fn.Params, ", "), nil
}

func (u *unparser) expr(e ast.Expr) (string, error) {
	switch n := e.(type) {
	case *ast.Ident:
		return n.Name, nil
	case *ast.IntLit:
		return strconv.FormatInt(n.Value, 10), nil
	case *ast.FloatLit:
		return strconv.FormatFloat(n.Value, 'f', -1, 64), nil
	case *ast.StringLit:
		return strconv.Quote(n.Value), nil
	case *ast.BoolLit:
		if n.Value {
			return "true", nil
		}
		return "false", nil
	case *ast.NilLit:
		return "nil", nil
	case *ast.BinaryExpr:
		left, err := u.expr(n.Left)
		if err != nil {
			return "", err
		}
		right, err := u.expr(n.Right)
		if err != nil {
			return "", err
		}
		return left + " " + n.Op.Lexeme + " " + right, nil
	case *ast.UnaryExpr:
		inner, err := u.expr(n.Expr)
		if err != nil {
			return "", err
		}
		op := n.Op.Lexeme
		if op == "not" {
			return "not " + inner, nil
		}
		return op + inner, nil
	case *ast.CallExpr:
		callee, err := u.expr(n.Callee)
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
		target, err := u.expr(n.Target)
		if err != nil {
			return "", err
		}
		return target + "." + n.Name, nil
	case *ast.IndexExpr:
		target, err := u.expr(n.Target)
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
