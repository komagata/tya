package checker

import (
	"tya/internal/ast"
)

// LintAutofixHint describes one autofixable lint finding that the
// LSP `textDocument/codeAction` handler (or another consumer) can
// convert into a concrete TextEdit.
//
// Kind values:
//   - "delete-line"  remove the line at Line (1-origin) entirely.
//   - "unwrap-if"    replace the `if` statement starting at Line
//                    with its Then body (when Code == TYAL0003 and
//                    the literal is `true`) or with its Else body
//                    (when the literal is `false`).
type LintAutofixHint struct {
	Code     string
	Kind     string
	Name     string // populated for "delete-line" (TYAL0001), empty otherwise
	Line     int    // 1-origin source line of the head of the construct
	Col      int    // 1-origin column
	EndLine  int    // inclusive end line of the construct
	BodyOnly bool   // true when "unwrap-if" should drop the construct's keyword line and de-indent the body
}

// LintAutofixHints returns the hints applicable to prog. It is a
// shallow merge of CollectUnused (TYAL0001) and the subset of
// CollectLintFindings that maps to a known autofix template.
//
// The hint list is intentionally a thin facade over the existing
// checker passes — its consumer is responsible for turning each
// hint into a TextEdit (whose computation needs the source bytes).
func LintAutofixHints(prog *ast.Program) []LintAutofixHint {
	if prog == nil {
		return nil
	}
	out := []LintAutofixHint{}
	for _, u := range CollectUnused(prog) {
		out = append(out, LintAutofixHint{
			Code: "TYAL0001",
			Kind: "delete-line",
			Name: u.Name,
			Line: u.Line,
			Col:  u.Col,
		})
	}
	for _, stmt := range prog.Stmts {
		out = append(out, walkAutofixStmt(stmt)...)
	}
	return out
}

func walkAutofixStmt(stmt ast.Stmt) []LintAutofixHint {
	var out []LintAutofixHint
	switch n := stmt.(type) {
	case *ast.IfStmt:
		if lit, ok := n.Cond.(*ast.BoolLit); ok {
			line, col := stmtFirstPos(stmt)
			endLine := line
			body := n.Then
			if !lit.Value {
				body = n.Else
			}
			endLine = autofixLastLine(body, line)
			out = append(out, LintAutofixHint{
				Code:     "TYAL0003",
				Kind:     "unwrap-if",
				Line:     line,
				Col:      col,
				EndLine:  endLine,
				BodyOnly: true,
			})
		}
		for _, s := range n.Then {
			out = append(out, walkAutofixStmt(s)...)
		}
		for _, s := range n.Else {
			out = append(out, walkAutofixStmt(s)...)
		}
	case *ast.WhileStmt:
		for _, s := range n.Body {
			out = append(out, walkAutofixStmt(s)...)
		}
	case *ast.ForInStmt:
		for _, s := range n.Body {
			out = append(out, walkAutofixStmt(s)...)
		}
	case *ast.TryCatchStmt:
		for _, s := range n.Try {
			out = append(out, walkAutofixStmt(s)...)
		}
		for _, s := range n.Catch {
			out = append(out, walkAutofixStmt(s)...)
		}
	case *ast.MatchStmt:
		for _, c := range n.Cases {
			for _, s := range c.Body {
				out = append(out, walkAutofixStmt(s)...)
			}
		}
	}
	return out
}

// autofixLastLine returns the 1-origin source line of the last
// statement in body, or fallback when the body is empty.
func autofixLastLine(body []ast.Stmt, fallback int) int {
	if len(body) == 0 {
		return fallback
	}
	last := body[len(body)-1]
	l, _ := stmtFirstPos(last)
	if l < fallback {
		return fallback
	}
	return l
}
