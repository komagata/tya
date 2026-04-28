package parser

import (
	"testing"

	"tya/internal/ast"
	"tya/internal/lexer"
)

func TestParseObjectAssignment(t *testing.T) {
	toks, errs := lexer.Lex("user =\n  name: \"komagata\"\n  age: 20\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	assign, ok := prog.Stmts[0].(*ast.AssignStmt)
	if !ok {
		t.Fatalf("got %T", prog.Stmts[0])
	}
	obj, ok := assign.Value.(*ast.ObjectLit)
	if !ok {
		t.Fatalf("got %T", assign.Value)
	}
	if len(obj.Props) != 2 {
		t.Fatalf("got %d props", len(obj.Props))
	}
}

func TestParseMultipleFunctionParams(t *testing.T) {
	toks, errs := lexer.Lex("add = a, b -> a + b\nprint add 2, 3\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, err := Parse(toks); err != nil {
		t.Fatal(err)
	}
}
