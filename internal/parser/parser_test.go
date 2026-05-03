package parser

import (
	"strings"
	"testing"

	"tya/internal/ast"
	"tya/internal/lexer"
)

func TestParseIndentedDictAssignment(t *testing.T) {
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
	obj, ok := assign.Values[0].(*ast.DictLit)
	if !ok {
		t.Fatalf("got %T", assign.Values[0])
	}
	if len(obj.Props) != 2 {
		t.Fatalf("got %d props", len(obj.Props))
	}
}

func TestParseInlineDict(t *testing.T) {
	toks, errs := lexer.Lex("user = { name: \"komagata\", age: 20 }\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	assign := prog.Stmts[0].(*ast.AssignStmt)
	obj, ok := assign.Values[0].(*ast.DictLit)
	if !ok {
		t.Fatalf("got %T", assign.Values[0])
	}
	if len(obj.Props) != 2 {
		t.Fatalf("got %d props", len(obj.Props))
	}
}

func TestParseSetLiteral(t *testing.T) {
	toks, errs := lexer.Lex("roles = { \"admin\", \"owner\" }\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	assign := prog.Stmts[0].(*ast.AssignStmt)
	set, ok := assign.Values[0].(*ast.SetLit)
	if !ok {
		t.Fatalf("got %T", assign.Values[0])
	}
	if len(set.Elems) != 2 {
		t.Fatalf("got %d elems", len(set.Elems))
	}
}

func TestParseEmptyCurlyLiteralAsDict(t *testing.T) {
	toks, errs := lexer.Lex("empty = {}\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	assign := prog.Stmts[0].(*ast.AssignStmt)
	if _, ok := assign.Values[0].(*ast.DictLit); !ok {
		t.Fatalf("got %T", assign.Values[0])
	}
}

func TestParseRejectsMixedDictAndSetLiteral(t *testing.T) {
	for _, src := range []string{
		"bad = { name: \"komagata\", \"admin\" }\n",
		"bad = { \"admin\", name: \"komagata\" }\n",
	} {
		toks, errs := lexer.Lex(src)
		if len(errs) != 0 {
			t.Fatalf("lex errors: %v", errs)
		}
		_, err := Parse(toks)
		if err == nil {
			t.Fatalf("expected mixed literal error for %q", src)
		}
		if !strings.Contains(err.Error(), "cannot mix dict entries and set entries in one literal") {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestParseCurrentBaselineTreatsClassAsExpression(t *testing.T) {
	toks, errs := lexer.Lex("class User\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	stmt, ok := prog.Stmts[0].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("got %T", prog.Stmts[0])
	}
	call, ok := stmt.Expr.(*ast.CallExpr)
	if !ok {
		t.Fatalf("got %T", stmt.Expr)
	}
	callee, ok := call.Callee.(*ast.Ident)
	if !ok || callee.Name != "class" {
		t.Fatalf("got callee %#v", call.Callee)
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

func TestParseIfElse(t *testing.T) {
	toks, errs := lexer.Lex("if true\n  print \"yes\"\nelse\n  print \"no\"\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := prog.Stmts[0].(*ast.IfStmt); !ok {
		t.Fatalf("got %T", prog.Stmts[0])
	}
}

func TestParseArrayLiteralAndIndex(t *testing.T) {
	toks, errs := lexer.Lex("items = [1, 2, 3]\nprint items[0]\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, err := Parse(toks); err != nil {
		t.Fatal(err)
	}
}

func TestParseIndexAssignment(t *testing.T) {
	toks, errs := lexer.Lex("items[1] = 20\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, err := Parse(toks); err != nil {
		t.Fatal(err)
	}
}

func TestParseGroupedExpression(t *testing.T) {
	toks, errs := lexer.Lex("print (2 + 3) * 4\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, err := Parse(toks); err != nil {
		t.Fatal(err)
	}
}

func TestParseNestedUnaryNoParenCall(t *testing.T) {
	toks, errs := lexer.Lex("print len keys user\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, err := Parse(toks); err != nil {
		t.Fatal(err)
	}
}

func TestParseWhile(t *testing.T) {
	toks, errs := lexer.Lex("while i < 5\n  i = i + 1\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := prog.Stmts[0].(*ast.WhileStmt); !ok {
		t.Fatalf("got %T", prog.Stmts[0])
	}
}

func TestParseReturnFunctionThenTopLevelPrint(t *testing.T) {
	src := "find_first_over = limit ->\n  i = 0\n  while true\n    if i > limit\n      return i\n    i = i + 1\n\nprint find_first_over 3\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	if len(prog.Stmts) != 2 {
		t.Fatalf("got %d top-level statements", len(prog.Stmts))
	}
	if _, ok := prog.Stmts[0].(*ast.AssignStmt); !ok {
		t.Fatalf("first stmt got %T", prog.Stmts[0])
	}
	if _, ok := prog.Stmts[1].(*ast.ExprStmt); !ok {
		t.Fatalf("second stmt got %T", prog.Stmts[1])
	}
}

func TestParseForIn(t *testing.T) {
	toks, errs := lexer.Lex("for item, index in items\n  print item\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := prog.Stmts[0].(*ast.ForInStmt); !ok {
		t.Fatalf("got %T", prog.Stmts[0])
	}
}
