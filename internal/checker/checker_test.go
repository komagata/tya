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
	prog := parse(t, "items = [1]\nfor User in items\n  print(User)\n")
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

func TestCheckAllowsDictionaryIndexAccess(t *testing.T) {
	prog := parse(t, "user = { name: \"komagata\" }\nprint(user[\"name\"])\n")
	if err := Check(prog); err != nil {
		t.Fatal(err)
	}
}

func TestCheckAllowsDictionaryMemberAccess(t *testing.T) {
	prog := parse(t, "user = { name: \"komagata\" }\nprint(user.name)\n")
	if err := Check(prog); err != nil {
		t.Fatal(err)
	}
}

func TestCheckAllowsArrayMemberAccess(t *testing.T) {
	prog := parse(t, "items = [1]\nprint(items.len())\n")
	if err := Check(prog); err != nil {
		t.Fatal(err)
	}
}

func TestCheckAllowsKnownModuleMemberAccess(t *testing.T) {
	prog := parse(t, "greeting = { text: \"hello\" }\nprint(greeting.text)\n")
	if err := CheckWithModules(prog, []string{"greeting"}); err != nil {
		t.Fatal(err)
	}
}

func TestCheckNamespaceDictionary(t *testing.T) {
	src := "foo = \"foo\"\nbar = -> \"bar\"\nutil = { foo: foo, bar: bar }\nprint(util.foo)\nprint(util.bar())\n"
	if err := Check(parse(t, src)); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRejectsUndefinedVariable(t *testing.T) {
	prog := parse(t, "print(user_nmae)\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected undefined variable error")
	}
	if !strings.Contains(err.Error(), "undefined variable user_nmae") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRejectsExcludedBuiltins(t *testing.T) {
	for _, name := range []string{
		"each",
		"div",
	} {
		t.Run(name, func(t *testing.T) {
			err := Check(parse(t, name+"()\n"))
			if err == nil {
				t.Fatalf("expected %s to be rejected", name)
			}
			if !strings.Contains(err.Error(), "undefined variable "+name) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCheckRejectsRemovedPrimitiveHelpersWithReplacement(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "top_level_len", src: "items = [1]\nprint(len(items))\n", want: "use value.len()"},
		{name: "top_level_trim", src: "text = \" tya \"\nprint(trim(text))\n", want: "use text.trim()"},
		{name: "top_level_to_number", src: "print(to_number(\"12\"))\n", want: "use value.to_number()"},
		{name: "string_module", src: "print(string.trim(\" tya \"))\n", want: "use text.trim()"},
		{name: "array_module", src: "print(array.len([1]))\n", want: "use items.len()"},
		{name: "dict_module", src: "print(dict.has({ name: \"Tya\" }, \"name\"))\n", want: "use dict.has(key)"},
		{name: "value_module", src: "print(value.nil?(nil))\n", want: "use value == nil"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Check(parse(t, tt.src))
			if err == nil {
				t.Fatal("expected removed primitive helper error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCheckAllowsMemberAccessOnUnknownValue(t *testing.T) {
	prog := parse(t, "show = user -> user.name\n")
	if err := Check(prog); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckAllowsFunctionParameterAndLoopVariable(t *testing.T) {
	prog := parse(t, "show = user -> user\nitems = [1]\nfor item in items\n  print(item)\nprint(show(1))\n")
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
	prog := parse(t, "first = value, unused -> value\nprint(first(1, 2))\n")
	err := CheckUnused(prog)
	if err == nil {
		t.Fatal("expected unused parameter error")
	}
	if !strings.Contains(err.Error(), "unused variable unused") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckUnusedAllowsUsedBindings(t *testing.T) {
	prog := parse(t, "items = [1]\nfor item in items\n  print(item)\nshow = value -> value\nprint(show(1))\n")
	if err := CheckUnused(prog); err != nil {
		t.Fatal(err)
	}
}

func TestIsClassFileName(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"Request.tya", true},
		{"HttpClient.tya", true},
		{"foo/bar/Request.tya", true},
		{"request.tya", false},
		{"http_client.tya", false},
		{"_Hidden.tya", false},
		{"123.tya", false},
	}
	for _, c := range cases {
		if got := IsClassFileName(c.path); got != c.want {
			t.Errorf("IsClassFileName(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestIsScriptFileName(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"hello.tya", true},
		{"client.tya", true},
		{"http_client.tya", true},
		{"Hello.tya", false},
		{"_hidden.tya", false},
	}
	for _, c := range cases {
		if got := IsScriptFileName(c.path); got != c.want {
			t.Errorf("IsScriptFileName(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestCheckClassFileAcceptsMatchingClass(t *testing.T) {
	prog := parse(t, "class Request\n  initialize = url ->\n    self.url = url\n")
	if err := CheckClassFile(prog, "Request.tya"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckClassFileAcceptsImportsBeforeClass(t *testing.T) {
	prog := parse(t, "import string\n\nclass Request\n  initialize = url ->\n    self.url = url\n")
	if err := CheckClassFile(prog, "Request.tya"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckClassFileAcceptsPrivateClasses(t *testing.T) {
	src := "class Header\n  initialize = name, value ->\n    self.name = name\n    self.value = value\n\n" +
		"class Request\n  initialize = url ->\n    self.url = url\n"
	prog := parse(t, src)
	if err := CheckClassFile(prog, "Request.tya"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckClassFileAcceptsInterfaceCompanion(t *testing.T) {
	src := "interface Sendable\n  send = ->\n\nclass Request\n  initialize = ->\n    self.url = nil\n"
	prog := parse(t, src)
	if err := CheckClassFile(prog, "Request.tya"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckClassFileRejectsMissingPublicClass(t *testing.T) {
	prog := parse(t, "class Helper\n  initialize = ->\n    self.x = 1\n")
	err := CheckClassFile(prog, "Request.tya")
	if err == nil {
		t.Fatal("expected missing public class error")
	}
	if !strings.Contains(err.Error(), "must define class Request") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckClassFileRejectsNonPascalCaseFilename(t *testing.T) {
	prog := parse(t, "class Request\n  initialize = ->\n    self.x = 1\n")
	err := CheckClassFile(prog, "request.tya")
	if err == nil {
		t.Fatal("expected PascalCase filename error")
	}
	if !strings.Contains(err.Error(), "PascalCase name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckClassFileRejectsTopLevelStatement(t *testing.T) {
	src := "class Request\n  initialize = ->\n    self.x = 1\n\nx = 1\n"
	prog := parse(t, src)
	err := CheckClassFile(prog, "Request.tya")
	if err == nil {
		t.Fatal("expected top-level statement error")
	}
	if !strings.Contains(err.Error(), "may only contain import, class, and interface") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckClassFileRejectsImportAfterClass(t *testing.T) {
	src := "class Request\n  initialize = ->\n    self.x = 1\n\nimport string\n"
	prog := parse(t, src)
	err := CheckClassFile(prog, "Request.tya")
	if err == nil {
		t.Fatal("expected import-after-class error")
	}
	if !strings.Contains(err.Error(), "imports must precede") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckClassFileRejectsDuplicatePublicClass(t *testing.T) {
	// Duplicate top-level class names are caught earlier by the
	// structure checker, not by CheckClassFile itself, but we still
	// document the behavior.
	src := "class Request\n  initialize = ->\n    self.x = 1\n\nclass Request\n  initialize = ->\n    self.y = 1\n"
	prog := parse(t, src)
	err := CheckClassFile(prog, "Request.tya")
	if err == nil {
		t.Fatal("expected duplicate-class error")
	}
}

func parse(t *testing.T, src string) *ast.Program {
	t.Helper()
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	return prog
}
