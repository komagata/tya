package checker

import (
	"fmt"
	"os"
	"strings"

	"tya/internal/ast"
	"tya/internal/diag"
)

// v0.28 strict-lint checks: shadowing forbidden, unused imports, and unused
// function arguments. v0.29 routes them through the diag.Diagnostic pipeline.

type strictBinding struct {
	name string
	kind strictKind
	line int
	col  int
	used bool
}

type strictKind int

const (
	strictLocal strictKind = iota
	strictImport
	strictArg
	strictPredeclared
)

type strictScope struct {
	parent   *strictScope
	root     bool
	funcBody bool // marks the parameter scope of a function body
	bindings map[string]*strictBinding
	order    []*strictBinding
}

func newStrictScope(parent *strictScope) *strictScope {
	s := &strictScope{parent: parent, bindings: map[string]*strictBinding{}}
	if parent == nil {
		s.root = true
	}
	return s
}

// strictCtx accumulates diagnostics during a strict pass. file is the
// reporting path used for Diagnostic.Primary.File.
type strictCtx struct {
	file  string
	diags []diag.Diagnostic
	// stop is set after the first diagnostic when collectAll=false.
	collectAll bool
	stop       bool
	// v0.48 G6: name of the class currently being walked, used to
	// emit [TYA-E0413] when its own member is referenced via the
	// hardcoded class name instead of `Self`. Empty outside class
	// bodies.
	currentClass string
}

func (c *strictCtx) report(d diag.Diagnostic) {
	if c.stop {
		return
	}
	d.Primary.File = c.file
	d.Source = "checker"
	c.diags = append(c.diags, d)
	if !c.collectAll {
		c.stop = true
	}
}

func (c *strictCtx) halted() bool { return c.stop }

func (s *strictScope) define(name string, kind strictKind, line, col int, ctx *strictCtx, identLen int) {
	if name == "_" {
		return
	}
	if _, ok := s.bindings[name]; ok {
		return
	}
	if !s.root {
		for anc := s.parent; anc != nil; anc = anc.parent {
			if b, ok := anc.bindings[name]; ok {
				if b.kind == strictPredeclared {
					break
				}
				ctx.report(diag.Diagnostic{
					Severity: diag.Error,
					Code:     "TYA-E0301",
					Title:    "Shadowed binding",
					Message:  fmt.Sprintf("This binding shadows the outer name `%s`.", name),
					Primary:  region(line, col, identLen),
					Hints:    []string{"Rename the inner binding, or prefix it with `_` to mark it as intentional."},
				})
				return
			}
		}
	}
	b := &strictBinding{name: name, kind: kind, line: line, col: col}
	s.bindings[name] = b
	s.order = append(s.order, b)
}

// definePredeclared records a builtin/module name without running shadow
// checks (used during scope bootstrap).
func (s *strictScope) definePredeclared(name string) {
	if _, ok := s.bindings[name]; ok {
		return
	}
	b := &strictBinding{name: name, kind: strictPredeclared, used: true}
	s.bindings[name] = b
	s.order = append(s.order, b)
}

func (s *strictScope) use(name string) {
	for sc := s; sc != nil; sc = sc.parent {
		if b, ok := sc.bindings[name]; ok {
			b.used = true
			return
		}
	}
}

func (s *strictScope) resolves(name string) bool {
	for sc := s; sc != nil; sc = sc.parent {
		if _, ok := sc.bindings[name]; ok {
			return true
		}
	}
	return false
}

// resolvesAboveFunction reports whether name is bound in a scope that
// is *outside* the closest enclosing function body. Predeclared
// builtins and module names are excluded — they exist on every scope
// and are intentionally available to inner code without being treated
// as shadow targets. Returns false when the current scope is not
// inside a function body, or when the name only resolves at or below
// that body.
func (s *strictScope) resolvesAboveFunction(name string) bool {
	var fnRoot *strictScope
	for sc := s; sc != nil; sc = sc.parent {
		if sc.funcBody {
			fnRoot = sc
			break
		}
	}
	if fnRoot == nil {
		return false
	}
	for sc := s; sc != nil; sc = sc.parent {
		if _, ok := sc.bindings[name]; ok {
			return false
		}
		if sc == fnRoot {
			break
		}
	}
	for sc := fnRoot.parent; sc != nil; sc = sc.parent {
		if b, ok := sc.bindings[name]; ok {
			if b.kind == strictPredeclared {
				return false
			}
			return true
		}
	}
	return false
}

// bindLocal records a binding in the current scope without running the
// shadow check. Used by assignment statements which Tya treats as either
// reassignment of an outer name or creation of a local — never as
// shadowing.
func (s *strictScope) bindLocal(name string, kind strictKind, line, col int) {
	if name == "_" {
		return
	}
	if _, ok := s.bindings[name]; ok {
		return
	}
	b := &strictBinding{name: name, kind: kind, line: line, col: col, used: true}
	s.bindings[name] = b
	s.order = append(s.order, b)
}

func region(line, col, length int) diag.Region {
	if length < 1 {
		length = 1
	}
	return diag.Region{
		Start: diag.Pos{Line: line, Col: col},
		End:   diag.Pos{Line: line, Col: col + length},
	}
}

// CheckStrict runs the strict pass and stops at the first diagnostic.
// It returns an error wrapping a single Diagnostic, or nil.
func CheckStrict(prog *ast.Program, modules []string) error {
	diags := CheckStrictDiagnostics(prog, modules, "", false)
	if len(diags) == 0 {
		return nil
	}
	return &StrictError{Diags: diags}
}

// CheckStrictDiagnostics runs the strict pass. If collectAll is true,
// every diagnostic is collected; otherwise only the first.
// file is recorded as Diagnostic.Primary.File.
func CheckStrictDiagnostics(prog *ast.Program, modules []string, file string, collectAll bool) []diag.Diagnostic {
	ctx := &strictCtx{file: file, collectAll: collectAll}
	root := newStrictScope(nil)
	for _, name := range builtinNames {
		root.definePredeclared(name)
	}
	for _, name := range modules {
		root.definePredeclared(name)
	}
	strictCollectTopLevel(prog.Stmts, root, ctx)
	if !ctx.halted() {
		strictWalkStmts(prog.Stmts, root, true, ctx)
	}
	if !ctx.halted() {
		strictDiagnoseScope(root, ctx)
	}
	return ctx.diags
}

// StrictError carries strict diagnostics. The first diagnostic's
// line:col:msg is used for Error() so legacy callers keep working.
type StrictError struct {
	Diags []diag.Diagnostic
}

func (e *StrictError) Error() string {
	if len(e.Diags) == 0 {
		return "strict check failed"
	}
	d := e.Diags[0]
	return fmt.Sprintf("%d:%d: %s", d.Primary.Start.Line, d.Primary.Start.Col, d.Message)
}

func strictCollectTopLevel(stmts []ast.Stmt, scope *strictScope, ctx *strictCtx) {
	for _, stmt := range stmts {
		if ctx.halted() {
			return
		}
		switch n := stmt.(type) {
		case *ast.ImportStmt:
			binding := n.BindingName()
			tok := n.NameTok
			if n.Alias != "" {
				tok = n.AliasTok
			}
			scope.define(binding, strictImport, tok.Line, tok.Col, ctx, len(binding))
		case *ast.AssignStmt:
			for _, target := range n.Targets {
				if id, ok := target.(*ast.Ident); ok {
					scope.define(id.Name, strictLocal, id.Tok.Line, id.Tok.Col, ctx, len(id.Name))
				}
			}
		case *ast.ModuleDecl:
			scope.define(n.Name, strictLocal, n.NameTok.Line, n.NameTok.Col, ctx, len(n.Name))
			if b, ok := scope.bindings[n.Name]; ok {
				b.used = true
			}
		case *ast.ClassDecl:
			scope.define(n.Name, strictLocal, n.NameTok.Line, n.NameTok.Col, ctx, len(n.Name))
			if b, ok := scope.bindings[n.Name]; ok {
				b.used = true
			}
		case *ast.InterfaceDecl:
			scope.define(n.Name, strictLocal, n.NameTok.Line, n.NameTok.Col, ctx, len(n.Name))
			if b, ok := scope.bindings[n.Name]; ok {
				b.used = true
			}
		}
	}
}

func strictWalkStmts(stmts []ast.Stmt, scope *strictScope, atRoot bool, ctx *strictCtx) {
	for _, stmt := range stmts {
		if ctx.halted() {
			return
		}
		strictWalkStmt(stmt, scope, atRoot, ctx)
	}
}

func strictWalkStmt(stmt ast.Stmt, scope *strictScope, atRoot bool, ctx *strictCtx) {
	switch n := stmt.(type) {
	case *ast.ImportStmt:
		return
	case *ast.AssignStmt:
		for _, value := range n.Values {
			strictWalkExpr(value, scope, ctx)
			if ctx.halted() {
				return
			}
		}
		if !atRoot {
			for _, target := range n.Targets {
				if id, ok := target.(*ast.Ident); ok {
					if !scope.resolves(id.Name) {
						scope.bindLocal(id.Name, strictLocal, id.Tok.Line, id.Tok.Col)
					} else if scope.resolvesAboveFunction(id.Name) {
						// v0.44: assignment to a name that lives in
						// an enclosing scope outside this function
						// body. The runtime would silently create a
						// new local; surface that as an error so the
						// user can route through a dict / array
						// argument instead.
						ctx.report(diag.Diagnostic{
							Severity: diag.Error,
							Code:     "TYA-E0307",
							Title:    "Assignment to outer binding",
							Message:  fmt.Sprintf("Cannot reassign `%s` from inside a function body; it lives in an enclosing scope.", id.Name),
							Primary:  region(id.Tok.Line, id.Tok.Col, len(id.Name)),
							Hints: []string{
								"Tya function bodies cannot write to outer variables.",
								"Pass a dict or array argument and update its contents (e.g. `state[\"" + id.Name + "\"] = ...`).",
								"For a single integer counter, use sync.atomic_integer.",
								"To shadow intentionally, rename the inner binding.",
							},
						})
						if ctx.halted() {
							return
						}
					} else {
						scope.use(id.Name)
					}
				}
			}
		}
		for _, target := range n.Targets {
			strictWalkAssignTarget(target, scope, ctx)
			if ctx.halted() {
				return
			}
		}
	case *ast.ExprStmt:
		strictWalkExpr(n.Expr, scope, ctx)
	case *ast.IfStmt:
		strictWalkExpr(n.Cond, scope, ctx)
		if ctx.halted() {
			return
		}
		strictWalkStmts(n.Then, newStrictScope(scope), false, ctx)
		if ctx.halted() {
			return
		}
		strictWalkStmts(n.Else, newStrictScope(scope), false, ctx)
	case *ast.WhileStmt:
		strictWalkExpr(n.Cond, scope, ctx)
		if ctx.halted() {
			return
		}
		strictWalkStmts(n.Body, newStrictScope(scope), false, ctx)
	case *ast.ForInStmt:
		strictWalkExpr(n.Iterable, scope, ctx)
		if ctx.halted() {
			return
		}
		child := newStrictScope(scope)
		child.define(n.ValueName, strictLocal, n.ValueTok.Line, n.ValueTok.Col, ctx, len(n.ValueName))
		if n.IndexName != "" {
			child.define(n.IndexName, strictLocal, n.IndexTok.Line, n.IndexTok.Col, ctx, len(n.IndexName))
		}
		if name := n.ValueName; name != "" && name != "_" {
			if b, ok := child.bindings[name]; ok {
				b.used = true
			}
		}
		if name := n.IndexName; name != "" && name != "_" {
			if b, ok := child.bindings[name]; ok {
				b.used = true
			}
		}
		strictWalkStmts(n.Body, child, false, ctx)
	case *ast.ReturnStmt:
		for _, value := range n.Values {
			strictWalkExpr(value, scope, ctx)
			if ctx.halted() {
				return
			}
		}
	case *ast.RaiseStmt:
		strictWalkExpr(n.Value, scope, ctx)
	case *ast.TryCatchStmt:
		strictWalkStmts(n.Try, newStrictScope(scope), false, ctx)
		if ctx.halted() {
			return
		}
		child := newStrictScope(scope)
		if n.CatchName != "_" {
			child.define(n.CatchName, strictLocal, n.CatchTok.Line, n.CatchTok.Col, ctx, len(n.CatchName))
			if b, ok := child.bindings[n.CatchName]; ok {
				b.used = true
			}
		}
		strictWalkStmts(n.Catch, child, false, ctx)
	case *ast.MatchStmt:
		strictWalkExpr(n.Value, scope, ctx)
		if ctx.halted() {
			return
		}
		for _, c := range n.Cases {
			child := newStrictScope(scope)
			strictDefinePatternBindings(c.Pattern, child, ctx)
			if ctx.halted() {
				return
			}
			strictWalkStmts(c.Body, child, false, ctx)
			if ctx.halted() {
				return
			}
		}
	case *ast.SelectStmt:
		for _, arm := range n.Arms {
			if arm.Channel != nil {
				strictWalkExpr(arm.Channel, scope, ctx)
				if ctx.halted() {
					return
				}
			}
			if arm.Value != nil {
				strictWalkExpr(arm.Value, scope, ctx)
				if ctx.halted() {
					return
				}
			}
			if arm.Seconds != nil {
				strictWalkExpr(arm.Seconds, scope, ctx)
				if ctx.halted() {
					return
				}
			}
			child := newStrictScope(scope)
			if arm.BindName != "" {
				child.define(arm.BindName, strictLocal, arm.BindTok.Line, arm.BindTok.Col, ctx, len(arm.BindName))
				if b, ok := child.bindings[arm.BindName]; ok {
					b.used = true
				}
			}
			strictWalkStmts(arm.Body, child, false, ctx)
			if ctx.halted() {
				return
			}
		}
	case *ast.ModuleDecl:
		strictWalkModule(n, scope, ctx)
	case *ast.ClassDecl:
		strictWalkClass(n, scope, ctx)
	case *ast.InterfaceDecl:
		return
	case *ast.BreakStmt, *ast.ContinueStmt:
		return
	}
}

func strictWalkModule(m *ast.ModuleDecl, scope *strictScope, ctx *strictCtx) {
	body := newStrictScope(scope)
	body.root = true
	for _, member := range m.Members {
		strictWalkExpr(member.Value, body, ctx)
		if ctx.halted() {
			return
		}
	}
	for _, class := range m.Classes {
		strictWalkClass(class, body, ctx)
		if ctx.halted() {
			return
		}
	}
}

func strictWalkClass(c *ast.ClassDecl, scope *strictScope, ctx *strictCtx) {
	body := newStrictScope(scope)
	body.root = true
	prevClass := ctx.currentClass
	ctx.currentClass = c.Name
	defer func() { ctx.currentClass = prevClass }()
	for _, m := range c.Methods {
		if m.Abstract {
			continue
		}
		strictWalkExpr(m.Func, body, ctx)
		if ctx.halted() {
			return
		}
	}
	for _, f := range c.Fields {
		if f.Value != nil {
			strictWalkExpr(f.Value, body, ctx)
			if ctx.halted() {
				return
			}
		}
	}
	for _, v := range c.Vars {
		if v.Value != nil {
			strictWalkExpr(v.Value, body, ctx)
			if ctx.halted() {
				return
			}
		}
	}
}

func strictWalkExpr(expr ast.Expr, scope *strictScope, ctx *strictCtx) {
	switch n := expr.(type) {
	case *ast.Ident:
		scope.use(n.Name)
	case *ast.StringLit:
		strictUseInterpolations(n.Value, scope)
	case *ast.DictLit:
		for _, prop := range n.Props {
			strictWalkExpr(prop.Value, scope, ctx)
			if ctx.halted() {
				return
			}
		}
	case *ast.ArrayLit:
		for _, elem := range n.Elems {
			strictWalkExpr(elem, scope, ctx)
			if ctx.halted() {
				return
			}
		}
	case *ast.FuncLit:
		fnScope := newStrictScope(scope)
		fnScope.funcBody = true
		for _, param := range n.Params {
			if param == "_" {
				continue
			}
			if _, dup := fnScope.bindings[param]; dup {
				ctx.report(diag.Diagnostic{
					Severity: diag.Error,
					Code:     "TYA-E0305",
					Title:    "Duplicate parameter",
					Message:  fmt.Sprintf("The parameter `%s` appears more than once in this function.", param),
					Primary:  region(0, 0, len(param)),
					Hints:    []string{"Rename one of the parameters."},
				})
				return
			}
			b := &strictBinding{name: param, kind: strictArg}
			if strings.HasPrefix(param, "_") {
				b.used = true
			}
			fnScope.bindings[param] = b
			fnScope.order = append(fnScope.order, b)
		}
		if n.Expr != nil {
			strictWalkExpr(n.Expr, fnScope, ctx)
			if ctx.halted() {
				return
			}
		}
		strictWalkStmts(n.Body, fnScope, false, ctx)
		if ctx.halted() {
			return
		}
		strictDiagnoseScope(fnScope, ctx)
	case *ast.BinaryExpr:
		strictWalkExpr(n.Left, scope, ctx)
		if ctx.halted() {
			return
		}
		strictWalkExpr(n.Right, scope, ctx)
	case *ast.UnaryExpr:
		strictWalkExpr(n.Expr, scope, ctx)
	case *ast.TryExpr:
		strictWalkExpr(n.Expr, scope, ctx)
	case *ast.MemberExpr:
		strictCheckCanonicalSelf(n, ctx)
		strictWalkExpr(n.Target, scope, ctx)
	case *ast.IndexExpr:
		strictWalkExpr(n.Target, scope, ctx)
		if ctx.halted() {
			return
		}
		strictWalkExpr(n.Index, scope, ctx)
	case *ast.CallExpr:
		strictWalkExpr(n.Callee, scope, ctx)
		if ctx.halted() {
			return
		}
		for _, arg := range n.Args {
			strictWalkExpr(arg, scope, ctx)
			if ctx.halted() {
				return
			}
		}
	}
}

func strictWalkAssignTarget(target ast.Expr, scope *strictScope, ctx *strictCtx) {
	if id := assignedOuterRoot(target, scope); id != nil && !permissiveCapturedMutation() {
		ctx.report(diag.Diagnostic{
			Severity: diag.Error,
			Code:     "TYA-E0308",
			Title:    "Mutation through captured binding",
			Message:  fmt.Sprintf("Cannot mutate `%s` through a captured binding from inside a function body.", id.Name),
			Primary:  region(id.Tok.Line, id.Tok.Col, len(id.Name)),
			Hints: []string{
				"Tya closures may read captured values but cannot mutate through them.",
				"Pass the mutable value as an explicit parameter when mutation is intended.",
			},
		})
		return
	}
	switch n := target.(type) {
	case *ast.Ident:
	case *ast.MemberExpr:
		strictCheckCanonicalSelf(n, ctx)
		strictWalkExpr(n.Target, scope, ctx)
	case *ast.IndexExpr:
		strictWalkExpr(n.Target, scope, ctx)
		if ctx.halted() {
			return
		}
		strictWalkExpr(n.Index, scope, ctx)
	}
}

func permissiveCapturedMutation() bool {
	return permissiveLegacy || os.Getenv("TYA_LEGACY_MODULES") == "1"
}

func assignedOuterRoot(target ast.Expr, scope *strictScope) *ast.Ident {
	switch n := target.(type) {
	case *ast.MemberExpr:
		return assignedOuterRootIdent(n.Target, scope)
	case *ast.IndexExpr:
		return assignedOuterRootIdent(n.Target, scope)
	default:
		return nil
	}
}

func assignedOuterRootIdent(expr ast.Expr, scope *strictScope) *ast.Ident {
	switch n := expr.(type) {
	case *ast.Ident:
		if scope.resolvesAboveFunction(n.Name) {
			return n
		}
	case *ast.MemberExpr:
		return assignedOuterRootIdent(n.Target, scope)
	case *ast.IndexExpr:
		return assignedOuterRootIdent(n.Target, scope)
	}
	return nil
}

func strictDefinePatternBindings(pattern ast.Expr, scope *strictScope, ctx *strictCtx) {
	switch n := pattern.(type) {
	case *ast.Ident:
		if n.Name != "_" {
			if _, dup := scope.bindings[n.Name]; dup {
				ctx.report(diag.Diagnostic{
					Severity: diag.Error,
					Code:     "TYA-E0306",
					Title:    "Duplicate binding in pattern",
					Message:  fmt.Sprintf("The name `%s` is bound more than once in this pattern.", n.Name),
					Primary:  region(n.Tok.Line, n.Tok.Col, len(n.Name)),
					Hints:    []string{"Rename one of the bindings."},
				})
				return
			}
			scope.define(n.Name, strictLocal, n.Tok.Line, n.Tok.Col, ctx, len(n.Name))
			if b, ok := scope.bindings[n.Name]; ok {
				b.used = true
			}
		}
	case *ast.ArrayLit:
		for _, elem := range n.Elems {
			strictDefinePatternBindings(elem, scope, ctx)
			if ctx.halted() {
				return
			}
		}
	case *ast.DictLit:
		for _, prop := range n.Props {
			strictDefinePatternBindings(prop.Value, scope, ctx)
			if ctx.halted() {
				return
			}
		}
	}
}

func strictUseInterpolations(s string, scope *strictScope) {
	for i := 0; i < len(s); i++ {
		if s[i] != '{' {
			continue
		}
		if i+1 < len(s) && s[i+1] == '{' {
			i++
			continue
		}
		j := i + 1
		for j < len(s) && s[j] != '}' {
			j++
		}
		if j >= len(s) {
			return
		}
		expr := strings.TrimSpace(s[i+1 : j])
		if expr != "" {
			strictUseExprText(expr, scope)
		}
		i = j
	}
}

func strictUseExprText(text string, scope *strictScope) {
	cur := strings.Builder{}
	for i := 0; i < len(text); i++ {
		c := text[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || (c >= '0' && c <= '9' && cur.Len() > 0) {
			cur.WriteByte(c)
			continue
		}
		if cur.Len() > 0 {
			scope.use(cur.String())
			cur.Reset()
		}
	}
	if cur.Len() > 0 {
		scope.use(cur.String())
	}
}

func strictDiagnoseScope(scope *strictScope, ctx *strictCtx) {
	for _, b := range scope.order {
		if ctx.halted() {
			return
		}
		if b.used {
			continue
		}
		switch b.kind {
		case strictImport:
			ctx.report(diag.Diagnostic{
				Severity: diag.Error,
				Code:     "TYA-E0302",
				Title:    "Unused import",
				Message:  fmt.Sprintf("The module `%s` is imported but never used.", b.name),
				Primary:  region(b.line, b.col, len(b.name)),
				Hints:    []string{"Remove the import, or reference the module somewhere in this file."},
			})
		case strictArg:
			ctx.report(diag.Diagnostic{
				Severity: diag.Error,
				Code:     "TYA-E0303",
				Title:    "Unused argument",
				Message:  fmt.Sprintf("The argument `%s` is never used in the body of this function.", b.name),
				Primary:  region(b.line, b.col, len(b.name)),
				Hints:    []string{"Rename it to `_` or prefix it with `_` (e.g. `_" + b.name + "`) to mark it as intentional."},
			})
		}
	}
}

// strictCheckCanonicalSelf emits [TYA-E0413] when a MemberExpr
// inside a class body uses the hardcoded declaring-class name
// (e.g. `User.count` inside `class User`) instead of the canonical
// `Self.count`. The formatter pass at `tya format` rewrites the
// non-canonical form automatically; this check surfaces it under
// `tya check --check-unused` strict mode so CI can detect drift
// without invoking the formatter.
//
// Gated by ctx.collectAll (set true only by CheckAll / CheckUnused)
// so the warning does not break `tya run` on existing code that
// still uses the hardcoded form — for example the stdlib's own
// `Markdown.foo` calls inside `class Markdown`.
func strictCheckCanonicalSelf(m *ast.MemberExpr, ctx *strictCtx) {
	if !ctx.collectAll || ctx.currentClass == "" {
		return
	}
	ident, ok := m.Target.(*ast.Ident)
	if !ok || ident.Name != ctx.currentClass {
		return
	}
	ctx.report(diag.Diagnostic{
		Severity: diag.Warning,
		Code:     "TYA-E0413",
		Title:    "Non-canonical class member access",
		Message:  fmt.Sprintf("`%s.%s` inside its own class body is non-canonical; write `Self.%s`.", ident.Name, m.Name, m.Name),
		Primary:  region(ident.Tok.Line, ident.Tok.Col, len(ident.Name)),
		Hints:    []string{"Run `tya format` to rewrite to the canonical `Self.` form."},
	})
}
