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
	prog := parse(t, "retry_count = 3\nretry_count = 5\n")
	if err := Check(prog); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRejectsInvalidBindingName(t *testing.T) {
	prog := parse(t, "userName = \"komagata\"\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected invalid binding name error")
	}
	if !strings.Contains(err.Error(), "invalid binding name userName") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRejectsInvalidLoopBindingNameWithLocation(t *testing.T) {
	prog := parse(t, "items = [1]\nfor User in items\n  print User\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected invalid loop binding name error")
	}
	if !strings.Contains(err.Error(), "2:5: invalid binding name User") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRejectsDuplicateFunctionParameter(t *testing.T) {
	prog := parse(t, "add = a, a -> a\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected duplicate parameter error")
	}
	if !strings.Contains(err.Error(), "1:10: duplicate function parameter a") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRejectsInvalidFunctionParameterWithLocation(t *testing.T) {
	prog := parse(t, "show = User -> User\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected invalid parameter error")
	}
	if !strings.Contains(err.Error(), "1:8: invalid binding name User") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRejectsDuplicateDictKey(t *testing.T) {
	prog := parse(t, "user =\n  name: \"a\"\n  name: \"b\"\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected duplicate key error")
	}
	if !strings.Contains(err.Error(), "3:3: duplicate dictionary key name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRejectsInvalidDictKeyWithLocation(t *testing.T) {
	prog := parse(t, "user = { Name: \"a\" }\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected invalid dictionary key error")
	}
	if !strings.Contains(err.Error(), "1:10: invalid dictionary key Name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckAllowsSetLiteralAndSetBuiltin(t *testing.T) {
	prog := parse(t, "roles = { \"admin\", \"owner\" }\nempty_roles = set()\nprint has roles, \"admin\"\nprint len empty_roles\n")
	if err := Check(prog); err != nil {
		t.Fatal(err)
	}
}

func TestCheckAllowsDictionaryIndexAccess(t *testing.T) {
	prog := parse(t, "user = { name: \"komagata\" }\nprint user[\"name\"]\n")
	if err := Check(prog); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRejectsDictionaryMemberAccess(t *testing.T) {
	prog := parse(t, "user = { name: \"komagata\" }\nprint user.name\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected dictionary member access error")
	}
	if !strings.Contains(err.Error(), "cannot use . access on dictionary; use index access") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRejectsArrayAndSetMemberAccess(t *testing.T) {
	cases := map[string]string{
		"items = [1]\nprint items.count\n":          "cannot use . access on array",
		"roles = { \"admin\" }\nprint roles.name\n": "cannot use . access on set",
		"roles = set()\nprint roles.name\n":         "cannot use . access on set",
	}
	for src, want := range cases {
		t.Run(want, func(t *testing.T) {
			err := Check(parse(t, src))
			if err == nil {
				t.Fatal("expected member access error")
			}
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCheckAllowsKnownModuleMemberAccess(t *testing.T) {
	prog := parse(t, "greeting = { text: \"hello\" }\nprint greeting.text\n")
	if err := CheckWithModules(prog, []string{"greeting"}); err != nil {
		t.Fatal(err)
	}
}

func TestCheckClassDeclaration(t *testing.T) {
	src := "class User\n  init: name ->\n    @name = name\n\n  greet: ->\n    \"Hello, {@name}\"\n\nuser = User(\"komagata\")\nprint user.greet()\n"
	if err := Check(parse(t, src)); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRejectsInvalidClassName(t *testing.T) {
	err := Check(parse(t, "class user\n  init: -> nil\n"))
	if err == nil {
		t.Fatal("expected invalid class name error")
	}
	if !strings.Contains(err.Error(), "invalid class name user") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRejectsDuplicateClassMethod(t *testing.T) {
	err := Check(parse(t, "class User\n  greet: -> \"a\"\n  greet: -> \"b\"\n"))
	if err == nil {
		t.Fatal("expected duplicate method error")
	}
	if !strings.Contains(err.Error(), "duplicate method greet") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRejectsUndefinedVariable(t *testing.T) {
	prog := parse(t, "print user_nmae\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected undefined variable error")
	}
	if !strings.Contains(err.Error(), "undefined variable user_nmae") {
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
