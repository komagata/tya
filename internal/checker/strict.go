package checker

import (
	"fmt"
	"strings"

	"tya/internal/ast"
)

// v0.28 strict-lint checks: shadowing forbidden, unused imports, unused
// function arguments, unused private top-level definitions.
//
// This file implements a separate AST walk that runs after the structural
// checker. Violations are reported as compile errors.

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
	strictPrivateTop
	strictPredeclared
)

type strictScope struct {
	parent   *strictScope
	root     bool
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

func (s *strictScope) define(name string, kind strictKind, line, col int) error {
	if name == "_" {
		return nil
	}
	if existing, ok := s.bindings[name]; ok {
		// Same-scope rebinding is not shadowing; just reuse the slot.
		_ = existing
		return nil
	}
	if !s.root {
		// Look up the chain (excluding current) for shadowing.
		for anc := s.parent; anc != nil; anc = anc.parent {
			if b, ok := anc.bindings[name]; ok {
				if b.kind == strictPredeclared {
					break
				}
				return fmt.Errorf("%d:%d: %s shadows outer binding %s", line, col, name, name)
			}
		}
	}
	b := &strictBinding{name: name, kind: kind, line: line, col: col}
	s.bindings[name] = b
	s.order = append(s.order, b)
	return nil
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

// CheckStrict performs the v0.28 strict checks on prog. modules lists the
// pre-imported module names that the runner has already woven into the
// source; they are treated as predeclared so user code can reference
// `string`, `array`, etc. without triggering shadow errors.
func CheckStrict(prog *ast.Program, modules []string) error {
	root := newStrictScope(nil)
	for _, name := range builtinNames {
		_ = root.define(name, strictPredeclared, 0, 0)
		root.bindings[name].used = true
	}
	for _, name := range modules {
		_ = root.define(name, strictPredeclared, 0, 0)
		root.bindings[name].used = true
	}
	if err := strictCollectTopLevel(prog.Stmts, root); err != nil {
		return err
	}
	if err := strictWalkStmts(prog.Stmts, root, true); err != nil {
		return err
	}
	return strictDiagnoseScope(root)
}

// strictCollectTopLevel pre-binds top-level names so within-file mutual
// references work. Imports are recorded with kind=import. Names starting
// with `_` are recorded as kind=privateTop. Other top-level assignments,
// modules, classes, interfaces are kind=local.
func strictCollectTopLevel(stmts []ast.Stmt, scope *strictScope) error {
	for _, stmt := range stmts {
		switch n := stmt.(type) {
		case *ast.ImportStmt:
			binding := n.BindingName()
			tok := n.NameTok
			if n.Alias != "" {
				tok = n.AliasTok
			}
			if err := scope.define(binding, strictImport, tok.Line, tok.Col); err != nil {
				return err
			}
		case *ast.AssignStmt:
			for _, target := range n.Targets {
				if id, ok := target.(*ast.Ident); ok {
					kind := strictLocal
					if strings.HasPrefix(id.Name, "_") && id.Name != "_" {
						kind = strictPrivateTop
					}
					if err := scope.define(id.Name, kind, id.Tok.Line, id.Tok.Col); err != nil {
						return err
					}
				}
			}
		case *ast.ModuleDecl:
			if err := scope.define(n.Name, strictLocal, n.NameTok.Line, n.NameTok.Col); err != nil {
				return err
			}
			scope.bindings[n.Name].used = true // module decl is a public artifact
		case *ast.ClassDecl:
			if err := scope.define(n.Name, strictLocal, n.NameTok.Line, n.NameTok.Col); err != nil {
				return err
			}
			scope.bindings[n.Name].used = true
		case *ast.InterfaceDecl:
			if err := scope.define(n.Name, strictLocal, n.NameTok.Line, n.NameTok.Col); err != nil {
				return err
			}
			scope.bindings[n.Name].used = true
		}
	}
	return nil
}

// strictWalkStmts walks statement bodies. atRoot=true skips re-defining
// top-level names (they were collected by strictCollectTopLevel) but still
// walks their values.
func strictWalkStmts(stmts []ast.Stmt, scope *strictScope, atRoot bool) error {
	for _, stmt := range stmts {
		if err := strictWalkStmt(stmt, scope, atRoot); err != nil {
			return err
		}
	}
	return nil
}

func strictWalkStmt(stmt ast.Stmt, scope *strictScope, atRoot bool) error {
	switch n := stmt.(type) {
	case *ast.ImportStmt:
		// Already defined in collect; nothing more to do.
		return nil
	case *ast.AssignStmt:
		for _, value := range n.Values {
			if err := strictWalkExpr(value, scope); err != nil {
				return err
			}
		}
		if !atRoot {
			// `=` in Tya reassigns an outer-scope binding when one exists,
			// otherwise creates a local. Either way, this is not the kind
			// of "new binding" form that triggers shadow checks; the
			// shadow rule applies only to for-loop variables, catch
			// bindings, and parameters (which are exempted separately).
			for _, target := range n.Targets {
				if id, ok := target.(*ast.Ident); ok {
					if !scope.resolves(id.Name) {
						scope.bindLocal(id.Name, strictLocal, id.Tok.Line, id.Tok.Col)
					} else {
						scope.use(id.Name)
					}
				}
			}
		}
		for _, target := range n.Targets {
			strictWalkAssignTarget(target, scope)
		}
		return nil
	case *ast.ExprStmt:
		return strictWalkExpr(n.Expr, scope)
	case *ast.IfStmt:
		if err := strictWalkExpr(n.Cond, scope); err != nil {
			return err
		}
		if err := strictWalkStmts(n.Then, newStrictScope(scope), false); err != nil {
			return err
		}
		return strictWalkStmts(n.Else, newStrictScope(scope), false)
	case *ast.WhileStmt:
		if err := strictWalkExpr(n.Cond, scope); err != nil {
			return err
		}
		return strictWalkStmts(n.Body, newStrictScope(scope), false)
	case *ast.ForInStmt:
		if err := strictWalkExpr(n.Iterable, scope); err != nil {
			return err
		}
		child := newStrictScope(scope)
		if err := child.define(n.ValueName, strictLocal, 0, 0); err != nil {
			return err
		}
		if n.IndexName != "" {
			if err := child.define(n.IndexName, strictLocal, 0, 0); err != nil {
				return err
			}
		}
		if name := n.ValueName; name != "" && name != "_" {
			child.bindings[name].used = true
		}
		if name := n.IndexName; name != "" && name != "_" {
			child.bindings[name].used = true
		}
		return strictWalkStmts(n.Body, child, false)
	case *ast.ReturnStmt:
		for _, value := range n.Values {
			if err := strictWalkExpr(value, scope); err != nil {
				return err
			}
		}
		return nil
	case *ast.RaiseStmt:
		return strictWalkExpr(n.Value, scope)
	case *ast.TryCatchStmt:
		if err := strictWalkStmts(n.Try, newStrictScope(scope), false); err != nil {
			return err
		}
		child := newStrictScope(scope)
		if n.CatchName != "_" {
			if err := child.define(n.CatchName, strictLocal, n.CatchTok.Line, n.CatchTok.Col); err != nil {
				return err
			}
			child.bindings[n.CatchName].used = true
		}
		return strictWalkStmts(n.Catch, child, false)
	case *ast.MatchStmt:
		if err := strictWalkExpr(n.Value, scope); err != nil {
			return err
		}
		for _, c := range n.Cases {
			child := newStrictScope(scope)
			if err := strictDefinePatternBindings(c.Pattern, child); err != nil {
				return err
			}
			if err := strictWalkStmts(c.Body, child, false); err != nil {
				return err
			}
		}
		return nil
	case *ast.ModuleDecl:
		// Module body is a separate scope; classes, methods, interfaces
		// inside don't shadow outer for v0.28's purposes.
		return strictWalkModule(n, scope)
	case *ast.ClassDecl:
		return strictWalkClass(n, scope)
	case *ast.InterfaceDecl:
		return nil
	case *ast.BreakStmt, *ast.ContinueStmt:
		return nil
	}
	return nil
}

func strictWalkModule(m *ast.ModuleDecl, scope *strictScope) error {
	body := newStrictScope(scope)
	body.root = true // suppress shadow check inside module member declarations
	for _, member := range m.Members {
		if err := strictWalkExpr(member.Value, body); err != nil {
			return err
		}
	}
	for _, class := range m.Classes {
		if err := strictWalkClass(class, body); err != nil {
			return err
		}
	}
	return nil
}

func strictWalkClass(c *ast.ClassDecl, scope *strictScope) error {
	body := newStrictScope(scope)
	body.root = true
	for _, m := range c.Methods {
		if m.Abstract {
			continue
		}
		if err := strictWalkExpr(m.Func, body); err != nil {
			return err
		}
	}
	for _, f := range c.Fields {
		if f.Value != nil {
			if err := strictWalkExpr(f.Value, body); err != nil {
				return err
			}
		}
	}
	for _, v := range c.Vars {
		if v.Value != nil {
			if err := strictWalkExpr(v.Value, body); err != nil {
				return err
			}
		}
	}
	return nil
}

func strictWalkExpr(expr ast.Expr, scope *strictScope) error {
	switch n := expr.(type) {
	case *ast.Ident:
		scope.use(n.Name)
		strictUseInterpolation(n.Name, scope)
	case *ast.StringLit:
		strictUseInterpolations(n.Value, scope)
	case *ast.DictLit:
		for _, prop := range n.Props {
			if err := strictWalkExpr(prop.Value, scope); err != nil {
				return err
			}
		}
	case *ast.ArrayLit:
		for _, elem := range n.Elems {
			if err := strictWalkExpr(elem, scope); err != nil {
				return err
			}
		}
	case *ast.FuncLit:
		fnScope := newStrictScope(scope)
		// Parameters do not participate in the shadow check against the
		// enclosing scope: a function arg may share a name with a
		// top-level binding without error. Body-local bindings still walk
		// from fnScope outward and so participate in shadow checks.
		for _, param := range n.Params {
			if param == "_" {
				continue
			}
			if _, dup := fnScope.bindings[param]; dup {
				return fmt.Errorf("duplicate parameter %s", param)
			}
			b := &strictBinding{name: param, kind: strictArg, line: 0, col: 0}
			if strings.HasPrefix(param, "_") {
				b.used = true
			}
			fnScope.bindings[param] = b
			fnScope.order = append(fnScope.order, b)
		}
		if n.Expr != nil {
			if err := strictWalkExpr(n.Expr, fnScope); err != nil {
				return err
			}
		}
		if err := strictWalkStmts(n.Body, fnScope, false); err != nil {
			return err
		}
		return strictDiagnoseScope(fnScope)
	case *ast.BinaryExpr:
		if err := strictWalkExpr(n.Left, scope); err != nil {
			return err
		}
		return strictWalkExpr(n.Right, scope)
	case *ast.UnaryExpr:
		return strictWalkExpr(n.Expr, scope)
	case *ast.TryExpr:
		return strictWalkExpr(n.Expr, scope)
	case *ast.MemberExpr:
		return strictWalkExpr(n.Target, scope)
	case *ast.IndexExpr:
		if err := strictWalkExpr(n.Target, scope); err != nil {
			return err
		}
		return strictWalkExpr(n.Index, scope)
	case *ast.CallExpr:
		if err := strictWalkExpr(n.Callee, scope); err != nil {
			return err
		}
		for _, arg := range n.Args {
			if err := strictWalkExpr(arg, scope); err != nil {
				return err
			}
		}
	}
	return nil
}

func strictWalkAssignTarget(target ast.Expr, scope *strictScope) {
	switch n := target.(type) {
	case *ast.Ident:
		// Assignment to a name; not a use.
	case *ast.MemberExpr:
		_ = strictWalkExpr(n.Target, scope)
	case *ast.IndexExpr:
		_ = strictWalkExpr(n.Target, scope)
		_ = strictWalkExpr(n.Index, scope)
	}
}

func strictDefinePatternBindings(pattern ast.Expr, scope *strictScope) error {
	switch n := pattern.(type) {
	case *ast.Ident:
		if n.Name != "_" {
			if err := scope.define(n.Name, strictLocal, n.Tok.Line, n.Tok.Col); err != nil {
				return err
			}
			if b, ok := scope.bindings[n.Name]; ok {
				b.used = true
			}
		}
	case *ast.ArrayLit:
		for _, elem := range n.Elems {
			if err := strictDefinePatternBindings(elem, scope); err != nil {
				return err
			}
		}
	case *ast.DictLit:
		for _, prop := range n.Props {
			if err := strictDefinePatternBindings(prop.Value, scope); err != nil {
				return err
			}
		}
	}
	return nil
}

// strictUseInterpolations scans a string literal value for `{name}`
// interpolations and marks the names used. The full lexer/parser already
// handled interpolation at compile time; here we just scan for identifier
// references.
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

func strictUseInterpolation(_ string, _ *strictScope) {
	// Identifier itself was marked used in caller.
}

// strictUseExprText takes interpolated expression text and marks the
// outermost identifier references used. It's intentionally a simple scan;
// false positives only relax checks, not tighten them.
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

func strictDiagnoseScope(scope *strictScope) error {
	for _, b := range scope.order {
		if b.used {
			continue
		}
		switch b.kind {
		case strictImport:
			return fmt.Errorf("%d:%d: unused import %s", b.line, b.col, b.name)
		case strictArg:
			return fmt.Errorf("%d:%d: unused argument %s", b.line, b.col, b.name)
		case strictPrivateTop:
			return fmt.Errorf("%d:%d: unused private top-level definition %s", b.line, b.col, b.name)
		}
	}
	return nil
}
