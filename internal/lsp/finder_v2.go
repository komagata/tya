package lsp

import (
	"tya/internal/ast"
	"tya/internal/token"
)

// ScopeKind names where a binding lives. It is the central
// concept behind v0.53's scope-aware rename and references.
type ScopeKind string

const (
	ScopeKindTopLevel ScopeKind = "top"
	ScopeKindLocal    ScopeKind = "local"
	ScopeKindParam    ScopeKind = "param"
	ScopeKindUnknown  ScopeKind = "unknown"
)

// FindAllIdentRefs returns the token positions of every Ident or
// MemberExpr name that matches `name` within prog. Used by
// References and Rename when the scope is global to the file.
func FindAllIdentRefs(prog *ast.Program, name string) []token.Token {
	var out []token.Token
	walkStmts(prog.Stmts, func(node any) {
		switch n := node.(type) {
		case *ast.Ident:
			if n.Name == name {
				out = append(out, n.Tok)
			}
		case *ast.MemberExpr:
			if n.Name == name {
				out = append(out, n.NameTok)
			}
		}
	})
	return out
}

// FindAllIdentRefsIn walks a subset of statements (e.g. a FuncLit
// body) and returns matching Ident references. Used by Rename for
// local / param scope.
func FindAllIdentRefsIn(stmts []ast.Stmt, name string) []token.Token {
	var out []token.Token
	walkStmts(stmts, func(node any) {
		switch n := node.(type) {
		case *ast.Ident:
			if n.Name == name {
				out = append(out, n.Tok)
			}
		case *ast.MemberExpr:
			if n.Name == name {
				out = append(out, n.NameTok)
			}
		}
	})
	return out
}

// FindAllParamRefs walks a FuncLit body for a parameter name. The
// FuncLit's Params slice itself is the declaration site, but tokens
// for each parameter are in ParamToks (1:1 index aligned).
func FindAllParamRefs(fn *ast.FuncLit, name string) []token.Token {
	var out []token.Token
	for i, p := range fn.Params {
		if p == name && i < len(fn.ParamToks) {
			out = append(out, fn.ParamToks[i])
		}
	}
	out = append(out, FindAllIdentRefsIn(fn.Body, name)...)
	if fn.Expr != nil {
		walkExpr(fn.Expr, func(node any) {
			switch n := node.(type) {
			case *ast.Ident:
				if n.Name == name {
					out = append(out, n.Tok)
				}
			case *ast.MemberExpr:
				if n.Name == name {
					out = append(out, n.NameTok)
				}
			}
		})
	}
	return out
}

// EnclosingFunc returns the *ast.FuncLit that lexically contains
// (line, col), or nil for top-level positions. The search picks
// the *innermost* enclosing FuncLit.
func EnclosingFunc(prog *ast.Program, line, col int) *ast.FuncLit {
	var best *ast.FuncLit
	var bestStart int
	walkStmts(prog.Stmts, func(node any) {
		fn, ok := node.(*ast.FuncLit)
		if !ok {
			return
		}
		// Determine the FuncLit's span from its first parameter
		// token (or its first body statement) through the last
		// body statement.
		startLine, _ := funcLitStart(fn)
		endLine := funcLitEnd(fn)
		if startLine == 0 || endLine == 0 {
			return
		}
		if line < startLine || line > endLine {
			return
		}
		if best == nil || startLine >= bestStart {
			best = fn
			bestStart = startLine
		}
	})
	return best
}

func funcLitStart(fn *ast.FuncLit) (line, col int) {
	if len(fn.ParamToks) > 0 {
		return fn.ParamToks[0].Line, fn.ParamToks[0].Col
	}
	if len(fn.Body) > 0 {
		return stmtFirstLine(fn.Body[0]), 0
	}
	return 0, 0
}

func funcLitEnd(fn *ast.FuncLit) int {
	if len(fn.Body) > 0 {
		return stmtLastLine(fn.Body[len(fn.Body)-1])
	}
	if fn.Expr != nil {
		if id, ok := fn.Expr.(*ast.Ident); ok {
			return id.Tok.Line
		}
	}
	if len(fn.ParamToks) > 0 {
		return fn.ParamToks[0].Line
	}
	return 0
}

func stmtFirstLine(s ast.Stmt) int {
	switch n := s.(type) {
	case *ast.AssignStmt:
		return n.Tok.Line
	case *ast.ExprStmt:
		return exprFirstLine(n.Expr)
	case *ast.IfStmt:
		return exprFirstLine(n.Cond)
	case *ast.WhileStmt:
		return exprFirstLine(n.Cond)
	case *ast.ForInStmt:
		return exprFirstLine(n.Iterable)
	case *ast.ReturnStmt:
		return n.Tok.Line
	case *ast.RaiseStmt:
		return n.Tok.Line
	case *ast.ClassDecl:
		return n.NameTok.Line
	case *ast.ModuleDecl:
		return n.NameTok.Line
	case *ast.InterfaceDecl:
		return n.NameTok.Line
	}
	return 0
}

func stmtLastLine(s ast.Stmt) int {
	switch n := s.(type) {
	case *ast.IfStmt:
		last := stmtFirstLine(s)
		for _, b := range n.Then {
			if l := stmtLastLine(b); l > last {
				last = l
			}
		}
		for _, b := range n.Else {
			if l := stmtLastLine(b); l > last {
				last = l
			}
		}
		return last
	case *ast.WhileStmt:
		last := stmtFirstLine(s)
		for _, b := range n.Body {
			if l := stmtLastLine(b); l > last {
				last = l
			}
		}
		return last
	case *ast.ForInStmt:
		last := stmtFirstLine(s)
		for _, b := range n.Body {
			if l := stmtLastLine(b); l > last {
				last = l
			}
		}
		return last
	case *ast.TryCatchStmt:
		last := stmtFirstLine(s)
		body := append(append([]ast.Stmt{}, n.Try...), n.Catch...)
		body = append(body, n.Finally...)
		for _, b := range body {
			if l := stmtLastLine(b); l > last {
				last = l
			}
		}
		return last
	case *ast.MatchStmt:
		last := stmtFirstLine(s)
		for _, c := range n.Cases {
			for _, b := range c.Body {
				if l := stmtLastLine(b); l > last {
					last = l
				}
			}
		}
		return last
	case *ast.ClassDecl:
		last := n.NameTok.Line
		for _, m := range n.Methods {
			if m.Func != nil {
				if l := funcLitEnd(m.Func); l > last {
					last = l
				}
			}
		}
		return last
	case *ast.ModuleDecl:
		return n.NameTok.Line
	case *ast.InterfaceDecl:
		return n.NameTok.Line
	}
	return stmtFirstLine(s)
}

func exprFirstLine(e ast.Expr) int {
	switch n := e.(type) {
	case *ast.Ident:
		return n.Tok.Line
	case *ast.MemberExpr:
		return n.NameTok.Line
	case *ast.BinaryExpr:
		return exprFirstLine(n.Left)
	case *ast.UnaryExpr:
		return n.Op.Line
	case *ast.CallExpr:
		return exprFirstLine(n.Callee)
	case *ast.IndexExpr:
		return exprFirstLine(n.Target)
	case *ast.FuncLit:
		l, _ := funcLitStart(n)
		return l
	case *ast.IntLit, *ast.FloatLit, *ast.StringLit, *ast.BoolLit, *ast.NilLit:
		return 0 // literals carry no Tok in the AST surface
	}
	return 0
}

// ScopeKindAt classifies the binding referenced at (line, col):
// - "top"   = the identifier resolves to a top-level Stmt name
// - "param" = a parameter of the innermost enclosing FuncLit
// - "local" = a local binding inside the innermost enclosing FuncLit
// - "unknown" = no resolution (builtin, stdlib, or undefined name)
//
// The returned target name is the resolved binding's name (always
// the same as the identifier under the cursor when found).
func ScopeKindAt(prog *ast.Program, line, col int) (ScopeKind, string) {
	id := FindIdentAt(prog, line, col)
	if id == nil {
		return ScopeKindUnknown, ""
	}
	idx := BuildSymbols(prog)
	if _, ok := idx.Lookup(id.Name); ok {
		return ScopeKindTopLevel, id.Name
	}
	fn := EnclosingFunc(prog, line, col)
	if fn != nil {
		for _, p := range fn.Params {
			if p == id.Name {
				return ScopeKindParam, id.Name
			}
		}
		for _, s := range fn.Body {
			if a, ok := s.(*ast.AssignStmt); ok {
				for _, t := range a.Targets {
					if ti, ok := t.(*ast.Ident); ok && ti.Name == id.Name {
						return ScopeKindLocal, id.Name
					}
				}
			}
		}
	}
	return ScopeKindUnknown, id.Name
}
