package parser

import (
	"testing"

	"tya/internal/ast"
	"tya/internal/lexer"
)

func TestParseWithCommentsHeader(t *testing.T) {
	src := `# This file is a probe.
# It has two header lines.

x = 1
`
	toks, lcomments, errs := lexer.LexWithComments(src)
	if len(errs) != 0 {
		t.Fatalf("lex errs: %v", errs)
	}
	comments := make([]CommentInfo, 0, len(lcomments))
	for _, c := range lcomments {
		comments = append(comments, CommentInfo{
			Line: c.Line, Col: c.Col, Indent: c.Indent,
			Text: c.Text, IsFullLine: c.IsFullLine,
		})
	}
	prog, err := ParseWithComments(toks, comments)
	if err != nil {
		t.Fatal(err)
	}
	if len(prog.HeaderComments) != 2 {
		t.Fatalf("want 2 header comments, got %d: %v", len(prog.HeaderComments), prog.HeaderComments)
	}
}

func TestParseWithCommentsNoBlankLine(t *testing.T) {
	src := `# leading comment, not a header
x = 1
`
	toks, lcomments, _ := lexer.LexWithComments(src)
	comments := make([]CommentInfo, 0, len(lcomments))
	for _, c := range lcomments {
		comments = append(comments, CommentInfo{Line: c.Line, Col: c.Col, Indent: c.Indent, Text: c.Text, IsFullLine: c.IsFullLine})
	}
	prog, err := ParseWithComments(toks, comments)
	if err != nil {
		t.Fatal(err)
	}
	if len(prog.HeaderComments) != 0 {
		t.Errorf("expected no header comments without blank line, got %v", prog.HeaderComments)
	}
}

func TestParseWithCommentsLeadingAndLineEnd(t *testing.T) {
	src := `# greet a user
greet = name -> "hi"
x = 1  # initial value
`
	toks, lcomments, _ := lexer.LexWithComments(src)
	comments := make([]CommentInfo, 0, len(lcomments))
	for _, c := range lcomments {
		comments = append(comments, CommentInfo{Line: c.Line, Col: c.Col, Indent: c.Indent, Text: c.Text, IsFullLine: c.IsFullLine})
	}
	prog, err := ParseWithComments(toks, comments)
	if err != nil {
		t.Fatal(err)
	}
	if prog.Comments == nil {
		t.Fatal("Comments map is nil")
	}
	greetStmt := prog.Stmts[0]
	gc, ok := prog.Comments[greetStmt]
	if !ok {
		t.Fatalf("greet stmt missing comments: %+v", prog.Comments)
	}
	if len(gc.Leading) != 1 || gc.Leading[0] != " greet a user" {
		t.Errorf("greet leading: %v", gc.Leading)
	}
	xStmt := prog.Stmts[1]
	xc, ok := prog.Comments[xStmt]
	if !ok {
		t.Fatalf("x stmt missing comments: %+v", prog.Comments)
	}
	if xc.LineEnd != " initial value" {
		t.Errorf("x line-end: %q", xc.LineEnd)
	}
}

func TestParseWithCommentsNested(t *testing.T) {
	src := `outer = ->
  # inner leading
  x = 1
  if x > 0
    # then leading
    y = 2  # then end
`
	toks, lcomments, _ := lexer.LexWithComments(src)
	comments := make([]CommentInfo, 0, len(lcomments))
	for _, c := range lcomments {
		comments = append(comments, CommentInfo{Line: c.Line, Col: c.Col, Indent: c.Indent, Text: c.Text, IsFullLine: c.IsFullLine})
	}
	prog, err := ParseWithComments(toks, comments)
	if err != nil {
		t.Fatal(err)
	}
	if prog.Comments == nil {
		t.Fatal("Comments map is nil")
	}
	// Find x=1 and y=2 inside the function body and the if-body.
	var found []string
	for stmt, sc := range prog.Comments {
		switch s := stmt.(type) {
		case *ast.AssignStmt:
			id, ok := s.Targets[0].(*ast.Ident)
			if !ok {
				continue
			}
			if id.Name == "x" && len(sc.Leading) == 1 && sc.Leading[0] == " inner leading" {
				found = append(found, "x")
			}
			if id.Name == "y" && len(sc.Leading) == 1 && sc.Leading[0] == " then leading" && sc.LineEnd == " then end" {
				found = append(found, "y")
			}
		}
	}
	if len(found) != 2 {
		t.Errorf("expected nested attachments for x and y, got %v (map %v)", found, prog.Comments)
	}
}

func TestParseWithCommentsLineEndIgnored(t *testing.T) {
	src := `x = 1  # inline
y = 2
`
	toks, lcomments, _ := lexer.LexWithComments(src)
	comments := make([]CommentInfo, 0, len(lcomments))
	for _, c := range lcomments {
		comments = append(comments, CommentInfo{Line: c.Line, Col: c.Col, Indent: c.Indent, Text: c.Text, IsFullLine: c.IsFullLine})
	}
	prog, err := ParseWithComments(toks, comments)
	if err != nil {
		t.Fatal(err)
	}
	if len(prog.HeaderComments) != 0 {
		t.Errorf("inline-only comment should not become header, got %v", prog.HeaderComments)
	}
}
