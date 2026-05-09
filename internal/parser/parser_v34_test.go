package parser

import (
	"testing"

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
