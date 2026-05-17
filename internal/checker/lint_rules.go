package checker

import (
	"strconv"

	"tya/internal/ast"
	"tya/internal/token"
)

// LintFinding describes one issue reported by the lint rule pipeline.
// Code is the stable diagnostic identifier (e.g. "TYAL0003"). Message
// is the human-readable explanation already formatted for stdout.
// Line / Col are 1-origin source positions; both are 0 when the
// AST does not carry enough position data (rare, best-effort).
type LintFinding struct {
	Code    string
	Message string
	Line    int
	Col     int
}

// nestingThreshold is the depth at which TYAL0004 fires. A block
// counts as 1 of nesting; the outermost program statements are
// depth 0.
const nestingThreshold = 5

// longFunctionThreshold is the maximum number of body statements a
// function literal may carry before TYAL0005 fires.
const longFunctionThreshold = 50

// CollectLintFindings walks prog and returns every TYAL0003 /
// TYAL0004 / TYAL0005 finding. TYAL0001 unused locals are produced
// separately by CollectUnused.
func CollectLintFindings(prog *ast.Program) []LintFinding {
	var out []LintFinding
	walkLintStmts(prog.Stmts, 0, &out)
	walkLintExtras(prog.Stmts, newLintScope(nil), &out)
	return out
}

type lintScope struct {
	parent *lintScope
	names  map[string]token.Token
}

func newLintScope(parent *lintScope) *lintScope {
	return &lintScope{parent: parent, names: map[string]token.Token{}}
}

func (s *lintScope) define(name string, tok token.Token, out *[]LintFinding) {
	if name == "_" || name == "" {
		return
	}
	for parent := s; parent != nil; parent = parent.parent {
		if prev, ok := parent.names[name]; ok {
			*out = append(*out, LintFinding{
				Code:    "TYAL0008",
				Message: "shadowed binding " + strconv.Quote(name) + " (previous binding at " + strconv.Itoa(prev.Line) + ":" + strconv.Itoa(prev.Col) + ")",
				Line:    tok.Line,
				Col:     tok.Col,
			})
			break
		}
	}
	if _, exists := s.names[name]; !exists {
		s.names[name] = tok
	}
}

func walkLintStmts(stmts []ast.Stmt, depth int, out *[]LintFinding) {
	deadAfter := -1
	deadKind := ""
	for i, stmt := range stmts {
		if deadAfter >= 0 {
			line, col := stmtFirstPos(stmt)
			*out = append(*out, LintFinding{
				Code:    "TYAL0002",
				Message: "dead code after " + deadKind,
				Line:    line,
				Col:     col,
			})
			_ = i
		}
		walkLintStmt(stmt, depth, out)
		if deadAfter < 0 {
			switch stmt.(type) {
			case *ast.ReturnStmt:
				deadAfter = i
				deadKind = "return"
			case *ast.RaiseStmt:
				deadAfter = i
				deadKind = "raise"
			}
		}
	}
}

func walkLintStmt(stmt ast.Stmt, depth int, out *[]LintFinding) {
	switch n := stmt.(type) {
	case *ast.IfStmt:
		if lit, ok := n.Cond.(*ast.BoolLit); ok {
			line, col := stmtFirstPos(stmt)
			val := "true"
			if !lit.Value {
				val = "false"
			}
			*out = append(*out, LintFinding{
				Code:    "TYAL0003",
				Message: "redundant `if " + val + "`",
				Line:    line,
				Col:     col,
			})
		}
		walkBlock(n.Then, depth+1, out)
		walkBlock(n.Else, depth+1, out)
	case *ast.WhileStmt:
		walkLintExpr(n.Cond, out)
		walkBlock(n.Body, depth+1, out)
	case *ast.ForInStmt:
		walkLintExpr(n.Iterable, out)
		walkBlock(n.Body, depth+1, out)
	case *ast.TryCatchStmt:
		walkBlock(n.Try, depth+1, out)
		walkBlock(n.Catch, depth+1, out)
		walkBlock(n.Finally, depth+1, out)
	case *ast.MatchStmt:
		walkLintExpr(n.Value, out)
		for _, c := range n.Cases {
			walkBlock(c.Body, depth+1, out)
		}
	case *ast.SelectStmt:
		for _, arm := range n.Arms {
			if arm.Channel != nil {
				walkLintExpr(arm.Channel, out)
			}
			if arm.Value != nil {
				walkLintExpr(arm.Value, out)
			}
			if arm.Seconds != nil {
				walkLintExpr(arm.Seconds, out)
			}
			walkBlock(arm.Body, depth+1, out)
		}
	case *ast.AssignStmt:
		for _, v := range n.Values {
			walkLintExpr(v, out)
		}
	case *ast.ExprStmt:
		walkLintExpr(n.Expr, out)
	case *ast.ReturnStmt:
		for _, v := range n.Values {
			walkLintExpr(v, out)
		}
	case *ast.RaiseStmt:
		walkLintExpr(n.Value, out)
	}
}

// walkBlock dives into a nested block, warning once if it crosses
// the nesting threshold.
func walkBlock(body []ast.Stmt, depth int, out *[]LintFinding) {
	if depth >= nestingThreshold && len(body) > 0 {
		line, col := stmtFirstPos(body[0])
		*out = append(*out, LintFinding{
			Code:    "TYAL0004",
			Message: "deeply nested block (depth >= " + strconv.Itoa(nestingThreshold) + ")",
			Line:    line,
			Col:     col,
		})
	}
	walkLintStmts(body, depth, out)
}

func walkLintExpr(expr ast.Expr, out *[]LintFinding) {
	switch n := expr.(type) {
	case *ast.FuncLit:
		if len(n.Body) > longFunctionThreshold {
			line, col := 0, 0
			if len(n.Body) > 0 {
				line, col = stmtFirstPos(n.Body[0])
			}
			*out = append(*out, LintFinding{
				Code:    "TYAL0005",
				Message: "function body has " + strconv.Itoa(len(n.Body)) + " statements (> " + strconv.Itoa(longFunctionThreshold) + ")",
				Line:    line,
				Col:     col,
			})
		}
		walkLintStmts(n.Body, 0, out)
		if n.Expr != nil {
			walkLintExpr(n.Expr, out)
		}
	case *ast.BinaryExpr:
		walkLintExpr(n.Left, out)
		walkLintExpr(n.Right, out)
	case *ast.UnaryExpr:
		walkLintExpr(n.Expr, out)
	case *ast.CallExpr:
		walkLintExpr(n.Callee, out)
		for _, a := range n.Args {
			walkLintExpr(a, out)
		}
	case *ast.MemberExpr:
		walkLintExpr(n.Target, out)
	case *ast.IndexExpr:
		walkLintExpr(n.Target, out)
		walkLintExpr(n.Index, out)
	case *ast.ArrayLit:
		for _, e := range n.Elems {
			walkLintExpr(e, out)
		}
	case *ast.DictLit:
		for _, p := range n.Props {
			walkLintExpr(p.Value, out)
		}
	case *ast.TryExpr:
		walkLintExpr(n.Expr, out)
	}
}

func walkLintExtras(stmts []ast.Stmt, scope *lintScope, out *[]LintFinding) {
	for _, stmt := range stmts {
		switch n := stmt.(type) {
		case *ast.ImportStmt:
			tok := n.NameTok
			if n.Alias != "" {
				tok = n.AliasTok
			}
			scope.define(n.BindingName(), tok, out)
		case *ast.AssignStmt:
			for _, target := range n.Targets {
				if id, ok := target.(*ast.Ident); ok {
					scope.define(id.Name, id.Tok, out)
				}
			}
			for _, value := range n.Values {
				walkLintExprExtras(value, scope, out)
			}
		case *ast.IfStmt:
			walkLintExprExtras(n.Cond, scope, out)
			walkLintExtras(n.Then, newLintScope(scope), out)
			walkLintExtras(n.Else, newLintScope(scope), out)
		case *ast.WhileStmt:
			walkLintExprExtras(n.Cond, scope, out)
			walkLintExtras(n.Body, newLintScope(scope), out)
		case *ast.ForInStmt:
			walkLintExprExtras(n.Iterable, scope, out)
			if suspiciousForIndexNames(n.ValueName, n.IndexName) {
				*out = append(*out, LintFinding{
					Code:    "TYAL0006",
					Message: "suspicious for index pattern: value binding comes before index binding",
					Line:    n.ValueTok.Line,
					Col:     n.ValueTok.Col,
				})
			}
			child := newLintScope(scope)
			child.define(n.ValueName, n.ValueTok, out)
			if n.IndexName != "" {
				child.define(n.IndexName, n.IndexTok, out)
			}
			walkLintExtras(n.Body, child, out)
		case *ast.ExprStmt:
			walkLintExprExtras(n.Expr, scope, out)
		case *ast.ReturnStmt:
			for _, value := range n.Values {
				walkLintExprExtras(value, scope, out)
			}
		case *ast.RaiseStmt:
			walkLintExprExtras(n.Value, scope, out)
		case *ast.TryCatchStmt:
			walkLintExtras(n.Try, newLintScope(scope), out)
			if n.Catch != nil {
				child := newLintScope(scope)
				if n.CatchName != "" {
					child.define(n.CatchName, n.CatchTok, out)
				}
				walkLintExtras(n.Catch, child, out)
			}
			walkLintExtras(n.Finally, newLintScope(scope), out)
		case *ast.MatchStmt:
			walkLintExprExtras(n.Value, scope, out)
			for _, c := range n.Cases {
				child := newLintScope(scope)
				defineLintPatternBindings(c.Pattern, child, out)
				walkLintExtras(c.Body, child, out)
			}
		case *ast.SelectStmt:
			for _, arm := range n.Arms {
				if arm.Channel != nil {
					walkLintExprExtras(arm.Channel, scope, out)
				}
				if arm.Value != nil {
					walkLintExprExtras(arm.Value, scope, out)
				}
				if arm.Seconds != nil {
					walkLintExprExtras(arm.Seconds, scope, out)
				}
				child := newLintScope(scope)
				child.define(arm.BindName, arm.BindTok, out)
				walkLintExtras(arm.Body, child, out)
			}
		}
	}
}

func walkLintExprExtras(expr ast.Expr, scope *lintScope, out *[]LintFinding) {
	switch n := expr.(type) {
	case *ast.FuncLit:
		reportUnusedParams(n, out)
		child := newLintScope(scope)
		for i, param := range n.Params {
			if i < len(n.ParamToks) {
				child.define(param, n.ParamToks[i], out)
			}
		}
		for _, def := range n.Defaults {
			walkLintExprExtras(def, child, out)
		}
		if n.Expr != nil {
			walkLintExprExtras(n.Expr, child, out)
		}
		walkLintExtras(n.Body, child, out)
	case *ast.BinaryExpr:
		walkLintExprExtras(n.Left, scope, out)
		walkLintExprExtras(n.Right, scope, out)
	case *ast.UnaryExpr:
		walkLintExprExtras(n.Expr, scope, out)
	case *ast.CallExpr:
		walkLintExprExtras(n.Callee, scope, out)
		for _, a := range n.Args {
			walkLintExprExtras(a, scope, out)
		}
	case *ast.MemberExpr:
		walkLintExprExtras(n.Target, scope, out)
	case *ast.IndexExpr:
		walkLintExprExtras(n.Target, scope, out)
		walkLintExprExtras(n.Index, scope, out)
	case *ast.ArrayLit:
		for _, e := range n.Elems {
			walkLintExprExtras(e, scope, out)
		}
	case *ast.DictLit:
		for _, p := range n.Props {
			walkLintExprExtras(p.Value, scope, out)
		}
	case *ast.TryExpr:
		walkLintExprExtras(n.Expr, scope, out)
	case *ast.SpawnExpr:
		walkLintExprExtras(n.Callee, scope, out)
	case *ast.AwaitExpr:
		walkLintExprExtras(n.Target, scope, out)
	}
}

func reportUnusedParams(fn *ast.FuncLit, out *[]LintFinding) {
	if len(fn.Params) == 0 || len(fn.ParamToks) == 0 {
		return
	}
	used := map[string]bool{}
	for _, def := range fn.Defaults {
		collectIdentUses(def, used)
	}
	if fn.Expr != nil {
		collectIdentUses(fn.Expr, used)
	}
	for _, stmt := range fn.Body {
		collectStmtIdentUses(stmt, used)
	}
	for i, param := range fn.Params {
		if param == "_" || i >= len(fn.ParamToks) || used[param] {
			continue
		}
		tok := fn.ParamToks[i]
		*out = append(*out, LintFinding{
			Code:    "TYAL0007",
			Message: "unused function parameter " + strconv.Quote(param),
			Line:    tok.Line,
			Col:     tok.Col,
		})
	}
}

func suspiciousForIndexNames(valueName, indexName string) bool {
	if indexName == "" {
		return false
	}
	return isIndexLikeName(valueName) && !isIndexLikeName(indexName)
}

func isIndexLikeName(name string) bool {
	switch name {
	case "i", "j", "k", "idx", "index":
		return true
	default:
		return false
	}
}

func defineLintPatternBindings(pattern ast.Expr, scope *lintScope, out *[]LintFinding) {
	switch n := pattern.(type) {
	case *ast.Ident:
		scope.define(n.Name, n.Tok, out)
	case *ast.ArrayLit:
		for _, elem := range n.Elems {
			defineLintPatternBindings(elem, scope, out)
		}
	case *ast.DictLit:
		for _, prop := range n.Props {
			defineLintPatternBindings(prop.Value, scope, out)
		}
	}
}

func collectStmtIdentUses(stmt ast.Stmt, used map[string]bool) {
	switch n := stmt.(type) {
	case *ast.AssignStmt:
		for _, value := range n.Values {
			collectIdentUses(value, used)
		}
		for _, target := range n.Targets {
			if _, ok := target.(*ast.Ident); !ok {
				collectIdentUses(target, used)
			}
		}
	case *ast.IfStmt:
		collectIdentUses(n.Cond, used)
		for _, s := range n.Then {
			collectStmtIdentUses(s, used)
		}
		for _, s := range n.Else {
			collectStmtIdentUses(s, used)
		}
	case *ast.WhileStmt:
		collectIdentUses(n.Cond, used)
		for _, s := range n.Body {
			collectStmtIdentUses(s, used)
		}
	case *ast.ForInStmt:
		collectIdentUses(n.Iterable, used)
		for _, s := range n.Body {
			collectStmtIdentUses(s, used)
		}
	case *ast.ExprStmt:
		collectIdentUses(n.Expr, used)
	case *ast.ReturnStmt:
		for _, value := range n.Values {
			collectIdentUses(value, used)
		}
	case *ast.RaiseStmt:
		collectIdentUses(n.Value, used)
	case *ast.TryCatchStmt:
		for _, s := range n.Try {
			collectStmtIdentUses(s, used)
		}
		for _, s := range n.Catch {
			collectStmtIdentUses(s, used)
		}
		for _, s := range n.Finally {
			collectStmtIdentUses(s, used)
		}
	case *ast.MatchStmt:
		collectIdentUses(n.Value, used)
		for _, c := range n.Cases {
			for _, s := range c.Body {
				collectStmtIdentUses(s, used)
			}
		}
	case *ast.SelectStmt:
		for _, arm := range n.Arms {
			if arm.Channel != nil {
				collectIdentUses(arm.Channel, used)
			}
			if arm.Value != nil {
				collectIdentUses(arm.Value, used)
			}
			if arm.Seconds != nil {
				collectIdentUses(arm.Seconds, used)
			}
			for _, s := range arm.Body {
				collectStmtIdentUses(s, used)
			}
		}
	}
}

func collectIdentUses(expr ast.Expr, used map[string]bool) {
	switch n := expr.(type) {
	case *ast.Ident:
		used[n.Name] = true
	case *ast.BinaryExpr:
		collectIdentUses(n.Left, used)
		collectIdentUses(n.Right, used)
	case *ast.UnaryExpr:
		collectIdentUses(n.Expr, used)
	case *ast.CallExpr:
		collectIdentUses(n.Callee, used)
		for _, a := range n.Args {
			collectIdentUses(a, used)
		}
	case *ast.MemberExpr:
		collectIdentUses(n.Target, used)
	case *ast.IndexExpr:
		collectIdentUses(n.Target, used)
		collectIdentUses(n.Index, used)
	case *ast.ArrayLit:
		for _, e := range n.Elems {
			collectIdentUses(e, used)
		}
	case *ast.DictLit:
		for _, p := range n.Props {
			collectIdentUses(p.Value, used)
		}
	case *ast.TryExpr:
		collectIdentUses(n.Expr, used)
	case *ast.FuncLit:
		// Nested functions own their parameters; do not count their
		// body as usage for the outer function's parameter check.
	case *ast.SpawnExpr:
		collectIdentUses(n.Callee, used)
	case *ast.AwaitExpr:
		collectIdentUses(n.Target, used)
	}
}

// stmtFirstPos returns a best-effort 1-origin (line, col) for stmt.
// Mirrors parser.stmtPos but covers a few more cases used by lint.
func stmtFirstPos(stmt ast.Stmt) (line, col int) {
	switch n := stmt.(type) {
	case *ast.AssignStmt:
		return n.Tok.Line, n.Tok.Col
	case *ast.ImportStmt:
		return n.NameTok.Line, n.NameTok.Col
	case *ast.EmbedStmt:
		return n.NameTok.Line, n.NameTok.Col
	case *ast.ReturnStmt:
		return n.Tok.Line, n.Tok.Col
	case *ast.RaiseStmt:
		return n.Tok.Line, n.Tok.Col
	case *ast.MatchStmt:
		return n.Tok.Line, n.Tok.Col
	case *ast.TryCatchStmt:
		return n.Tok.Line, n.Tok.Col
	case *ast.ForInStmt:
		return n.ValueTok.Line, n.ValueTok.Col
	case *ast.SelectStmt:
		return n.Tok.Line, n.Tok.Col
	case *ast.ExprStmt:
		return exprFirstPos(n.Expr)
	case *ast.IfStmt:
		if len(n.Then) > 0 {
			l, c := stmtFirstPos(n.Then[0])
			if l > 0 {
				return l, c
			}
		}
		return 0, 0
	case *ast.WhileStmt:
		if len(n.Body) > 0 {
			l, c := stmtFirstPos(n.Body[0])
			if l > 0 {
				return l, c
			}
		}
		return 0, 0
	}
	return 0, 0
}

// exprFirstPos returns a best-effort 1-origin (line, col) of the
// leftmost token reachable from expr. Many literals carry no token
// at all, in which case we return (0, 0) — callers must tolerate
// that, as did the old behavior.
func exprFirstPos(expr ast.Expr) (line, col int) {
	switch n := expr.(type) {
	case *ast.Ident:
		return n.Tok.Line, n.Tok.Col
	case *ast.CallExpr:
		return exprFirstPos(n.Callee)
	case *ast.MemberExpr:
		return exprFirstPos(n.Target)
	case *ast.IndexExpr:
		return exprFirstPos(n.Target)
	case *ast.BinaryExpr:
		return exprFirstPos(n.Left)
	case *ast.UnaryExpr:
		return exprFirstPos(n.Expr)
	}
	return 0, 0
}
