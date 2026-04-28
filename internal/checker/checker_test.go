package checker

import (
	"strings"
	"testing"

	"tya/internal/ast"
	"tya/internal/lexer"
	"tya/internal/parser"
)

func TestCheckRejectsConstantReassignment(t *testing.T) {
	prog := parse(t, "MAX_RETRY = 3\nMAX_RETRY = 5\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected constant reassignment error")
	}
	if !strings.Contains(err.Error(), "cannot reassign constant MAX_RETRY") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckAllowsVariableReassignment(t *testing.T) {
	prog := parse(t, "retryCount = 3\nretryCount = 5\n")
	if err := Check(prog); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRejectsInvalidBindingName(t *testing.T) {
	prog := parse(t, "user_name = \"komagata\"\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected invalid binding name error")
	}
	if !strings.Contains(err.Error(), "invalid binding name user_name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRejectsDuplicateFunctionParameter(t *testing.T) {
	prog := parse(t, "add = a, a -> a\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected duplicate parameter error")
	}
	if !strings.Contains(err.Error(), "duplicate function parameter a") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRejectsDuplicateObjectProperty(t *testing.T) {
	prog := parse(t, "user =\n  name: \"a\"\n  name: \"b\"\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected duplicate property error")
	}
	if !strings.Contains(err.Error(), "duplicate object property name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRejectsUndefinedVariable(t *testing.T) {
	prog := parse(t, "print userNmae\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected undefined variable error")
	}
	if !strings.Contains(err.Error(), "undefined variable userNmae") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckAllowsFunctionParameterAndLoopVariable(t *testing.T) {
	prog := parse(t, "show = user -> user.name\nitems = [1]\nfor item in items\n  print item\n")
	if err := Check(prog); err != nil {
		t.Fatal(err)
	}
}

func TestCheckUnusedRejectsUnusedVariable(t *testing.T) {
	prog := parse(t, "name = \"Tya\"\n")
	err := CheckUnused(prog)
	if err == nil {
		t.Fatal("expected unused variable error")
	}
	if !strings.Contains(err.Error(), "unused variable name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckUnusedRejectsUnusedFunctionParameter(t *testing.T) {
	prog := parse(t, "first = value, unused -> value\nprint first 1, 2\n")
	err := CheckUnused(prog)
	if err == nil {
		t.Fatal("expected unused parameter error")
	}
	if !strings.Contains(err.Error(), "unused variable unused") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckUnusedAllowsUsedBindings(t *testing.T) {
	prog := parse(t, "items = [1]\nfor item in items\n  print item\nshow = value -> value\nprint show 1\n")
	if err := CheckUnused(prog); err != nil {
		t.Fatal(err)
	}
}

func parse(t *testing.T, src string) *ast.Program {
	t.Helper()
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	return prog
}
