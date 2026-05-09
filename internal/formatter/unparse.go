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
	"strconv"
	"strings"

	"tya/internal/ast"
)

// Unparse renders prog as canonical Tya source. v0.37 covers a
// useful subset of programs; unsupported AST shapes return a
// non-nil error.
func Unparse(prog *ast.Program) (string, error) {
	u := &unparser{}
	for _, stmt := range prog.Stmts {
		if err := u.stmt(stmt); err != nil {
			return "", err
		}
	}
	return u.b.String(), nil
}

type unparser struct {
	b      strings.Builder
	indent int
}

func (u *unparser) line(s string) {
	u.b.WriteString(strings.Repeat("  ", u.indent))
	u.b.WriteString(s)
	u.b.WriteByte('\n')
}

func (u *unparser) stmt(s ast.Stmt) error {
	switch n := s.(type) {
	case *ast.ImportStmt:
		if n.Alias != "" {
			u.line(fmt.Sprintf("import %s as %s", n.Name, n.Alias))
		} else {
			u.line(fmt.Sprintf("import %s", n.Name))
		}
		return nil
	case *ast.AssignStmt:
		return u.assignStmt(n)
	case *ast.ExprStmt:
		ex, err := u.expr(n.Expr)
		if err != nil {
			return err
		}
		// v0.37: render `print foo` and `assert ...` keyword forms
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
					u.line(id.Name + " " + strings.Join(args, ", "))
					return nil
				}
			}
		}
		u.line(ex)
		return nil
	case *ast.ReturnStmt:
		if len(n.Values) == 0 {
			u.line("return")
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
		u.line("return " + strings.Join(parts, ", "))
		return nil
	case *ast.RaiseStmt:
		s, err := u.expr(n.Value)
		if err != nil {
			return err
		}
		u.line("raise " + s)
		return nil
	case *ast.BreakStmt:
		u.line("break")
		return nil
	case *ast.ContinueStmt:
		u.line("continue")
		return nil
	case *ast.IfStmt:
		return u.ifStmt(n)
	case *ast.WhileStmt:
		cond, err := u.expr(n.Cond)
		if err != nil {
			return err
		}
		u.line("while " + cond)
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
	}
	return fmt.Errorf("formatter.Unparse: stmt %T not supported in v0.37", s)
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
	u.line(strings.Join(targets, ", ") + " = " + strings.Join(values, ", "))
	return nil
}

func (u *unparser) ifStmt(n *ast.IfStmt) error {
	cond, err := u.expr(n.Cond)
	if err != nil {
		return err
	}
	u.line("if " + cond)
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
			u.line("elseif " + cond2)
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

func (u *unparser) block(stmts []ast.Stmt) error {
	u.indent++
	defer func() { u.indent-- }()
	for _, s := range stmts {
		if err := u.stmt(s); err != nil {
			return err
		}
	}
	return nil
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
