package checker

import (
	"strconv"

	"tya/internal/ast"
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
	return out
}

func walkLintStmts(stmts []ast.Stmt, depth int, out *[]LintFinding) {
	for _, stmt := range stmts {
		walkLintStmt(stmt, depth, out)
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
	case *ast.MatchStmt:
		walkLintExpr(n.Value, out)
		for _, c := range n.Cases {
			walkBlock(c.Body, depth+1, out)
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

// stmtFirstPos returns a best-effort 1-origin (line, col) for stmt.
// Mirrors parser.stmtPos but covers a few more cases used by lint.
func stmtFirstPos(stmt ast.Stmt) (line, col int) {
	switch n := stmt.(type) {
	case *ast.AssignStmt:
		return n.Tok.Line, n.Tok.Col
	case *ast.ImportStmt:
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

