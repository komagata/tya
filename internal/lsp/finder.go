package lsp

import (
	"tya/internal/ast"
	"tya/internal/token"
)

// FindIdentAt returns the innermost identifier-like node that
// covers (line, col), or nil if no identifier was found at that
// position. Inputs are 1-origin to match tya token positions; LSP
// boundary code converts before calling.
//
// A MemberExpr's member side is reported as a synthesised Ident
// (so `obj.field` resolves on the `field` half), while the
// receiver side falls through to the underlying Expr walker.
func FindIdentAt(prog *ast.Program, line, col int) *ast.Ident {
	if prog == nil {
		return nil
	}
	var best *ast.Ident
	consider := func(name string, tok token.Token) {
		if !tokenCovers(tok, name, line, col) {
			return
		}
		best = &ast.Ident{Name: name, Tok: tok}
	}
	walkStmts(prog.Stmts, func(node any) {
		switch n := node.(type) {
		case *ast.Ident:
			consider(n.Name, n.Tok)
		case *ast.MemberExpr:
			consider(n.Name, n.NameTok)
		case *ast.AssignStmt:
			// Targets that are Idents may be top-level definitions; walked separately.
		case *ast.ClassDecl:
			consider(n.Name, n.NameTok)
			for _, m := range n.Methods {
				consider(m.Name, m.Tok)
			}
			for _, f := range n.Fields {
				consider(f.Name, f.Tok)
			}
			for _, v := range n.Vars {
				consider(v.Name, v.Tok)
			}
			for _, c := range n.Constants {
				consider(c.Name, c.Tok)
			}
		case *ast.ModuleDecl:
			consider(n.Name, n.NameTok)
			for _, m := range n.Members {
				consider(m.Name, m.Tok)
			}
		case *ast.InterfaceDecl:
			consider(n.Name, n.NameTok)
			for _, m := range n.Methods {
				consider(m.Name, m.Tok)
			}
			for _, f := range n.Fields {
				consider(f.Name, f.Tok)
			}
		}
	})
	return best
}

func tokenCovers(t token.Token, name string, line, col int) bool {
	if t.Line != line {
		return false
	}
	end := t.Col + len(name)
	return col >= t.Col && col < end
}

// walkStmts is a depth-first visitor that yields every Stmt and
// Expr (including nested ones) to the visitor func.
func walkStmts(stmts []ast.Stmt, visit func(node any)) {
	for _, s := range stmts {
		walkStmt(s, visit)
	}
}

func walkStmt(stmt ast.Stmt, visit func(node any)) {
	visit(stmt)
	switch n := stmt.(type) {
	case *ast.AssignStmt:
		for _, t := range n.Targets {
			walkExpr(t, visit)
		}
		for _, v := range n.Values {
			walkExpr(v, visit)
		}
	case *ast.ExprStmt:
		walkExpr(n.Expr, visit)
	case *ast.IfStmt:
		walkExpr(n.Cond, visit)
		walkStmts(n.Then, visit)
		walkStmts(n.Else, visit)
	case *ast.WhileStmt:
		walkExpr(n.Cond, visit)
		walkStmts(n.Body, visit)
	case *ast.ForInStmt:
		walkExpr(n.Iterable, visit)
		walkStmts(n.Body, visit)
	case *ast.ReturnStmt:
		for _, v := range n.Values {
			walkExpr(v, visit)
		}
	case *ast.RaiseStmt:
		if n.Value != nil {
			walkExpr(n.Value, visit)
		}
	case *ast.TryCatchStmt:
		walkStmts(n.Try, visit)
		walkStmts(n.Catch, visit)
		walkStmts(n.Finally, visit)
	case *ast.MatchStmt:
		walkExpr(n.Value, visit)
		for _, c := range n.Cases {
			walkStmts(c.Body, visit)
		}
	case *ast.ClassDecl:
		if n.Parent != nil {
			visit(n.Parent)
		}
		for i := range n.Implements {
			visit(&n.Implements[i])
		}
		for _, m := range n.Methods {
			visit(&m)
			if m.Func != nil {
				visit(m.Func)
				walkStmts(m.Func.Body, visit)
				if m.Func.Expr != nil {
					walkExpr(m.Func.Expr, visit)
				}
			}
		}
		for _, v := range n.Vars {
			visit(&v)
			if v.Value != nil {
				walkExpr(v.Value, visit)
			}
		}
		for _, c := range n.Constants {
			visit(&c)
			if c.Value != nil {
				walkExpr(c.Value, visit)
			}
		}
		for _, f := range n.Fields {
			visit(&f)
			if f.Value != nil {
				walkExpr(f.Value, visit)
			}
		}
	case *ast.ModuleDecl:
		for _, m := range n.Members {
			visit(&m)
			if m.Value != nil {
				walkExpr(m.Value, visit)
			}
		}
		for _, c := range n.Classes {
			walkStmt(c, visit)
		}
		for _, i := range n.Interfaces {
			walkStmt(i, visit)
		}
	case *ast.InterfaceDecl:
		for _, p := range n.Parents {
			visit(&p)
		}
		for _, m := range n.Methods {
			visit(&m)
			if m.Func != nil {
				visit(m.Func)
				walkStmts(m.Func.Body, visit)
				if m.Func.Expr != nil {
					walkExpr(m.Func.Expr, visit)
				}
			}
		}
		for _, f := range n.Fields {
			visit(&f)
			if f.Value != nil {
				walkExpr(f.Value, visit)
			}
		}
	}
}

func walkExpr(expr ast.Expr, visit func(node any)) {
	if expr == nil {
		return
	}
	visit(expr)
	switch n := expr.(type) {
	case *ast.BinaryExpr:
		walkExpr(n.Left, visit)
		walkExpr(n.Right, visit)
	case *ast.UnaryExpr:
		walkExpr(n.Expr, visit)
	case *ast.MemberExpr:
		walkExpr(n.Target, visit)
	case *ast.FuncLit:
		walkStmts(n.Body, visit)
		if n.Expr != nil {
			walkExpr(n.Expr, visit)
		}
	case *ast.CallExpr:
		walkExpr(n.Callee, visit)
		for _, a := range n.Args {
			walkExpr(a, visit)
		}
	case *ast.IndexExpr:
		walkExpr(n.Target, visit)
		walkExpr(n.Index, visit)
	case *ast.ArrayLit:
		for _, el := range n.Elems {
			walkExpr(el, visit)
		}
	case *ast.DictLit:
		for _, p := range n.Props {
			walkExpr(p.Value, visit)
		}
	case *ast.TryExpr:
		walkExpr(n.Expr, visit)
	}
}
