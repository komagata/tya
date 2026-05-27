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

func TestCheckAllowsSelfClassConstantInFieldInitializer(t *testing.T) {
	prog := parse(t, "class Csv\n  SEPARATOR: \",\"\n\n  options: { separator: Self.SEPARATOR, header: false }\n")
	if err := Check(prog); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRejectsKindChangingReassignment(t *testing.T) {
	prog := parse(t, "retry_count = 3\nretry_count = \"three\"\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected kind-changing reassignment error")
	}
	if !strings.Contains(err.Error(), "cannot reassign retry_count from number to string") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRejectsMixedStringPlus(t *testing.T) {
	tests := []string{
		"count = 3\nprint(\"count: \" + count)\n",
		"count = 3\nprint(count + \" items\")\n",
		"data = b\"a\"\nprint(data + \"b\")\n",
	}
	for _, src := range tests {
		prog := parse(t, src)
		err := Check(prog)
		if err == nil {
			t.Fatalf("expected mixed plus error for %q", src)
		}
		if !strings.Contains(err.Error(), "+ expects numbers, strings, or bytes of the same kind") {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestCheckRejectsDictionaryKeyMemberAccess(t *testing.T) {
	prog := parse(t, "user = { name: \"komagata\" }\nprint(user.name)\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected dictionary member access error")
	}
	if !strings.Contains(err.Error(), "cannot use . access on dictionary; use index access") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckAllowsDictionaryIndexAndMethodAccess(t *testing.T) {
	prog := parse(t, "user = { name: \"komagata\" }\nprint(user[\"name\"])\nprint(user.keys())\nprint(user.has?(\"name\"))\nuser.delete(\"age\")\n")
	if err := Check(prog); err != nil {
		t.Fatal(err)
	}
}

func TestCheckAllowsDictionaryStringLiteralKeys(t *testing.T) {
	prog := parse(t, "headers = { \"Content-Type\": \"text/plain\", \"$schema\": \"x\", \"1\": \"one\", \"\": \"empty\" }\nprint(headers[\"Content-Type\"])\n")
	if err := Check(prog); err != nil {
		t.Fatal(err)
	}
}

func TestCheckNumberKindAllowsIntFloatEquality(t *testing.T) {
	prog := parse(t, "print(1 == 1.0)\n")
	if err := Check(prog); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRejectsFloatModulo(t *testing.T) {
	tests := []string{
		"print(5.5 % 2)\n",
		"print(5 % 2.0)\n",
	}
	for _, src := range tests {
		prog := parse(t, src)
		err := Check(prog)
		if err == nil {
			t.Fatalf("expected float modulo error for %q", src)
		}
		if !strings.Contains(err.Error(), "% expects integers") {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestCheckRejectsNegativeLiteralIndexes(t *testing.T) {
	tests := []string{
		"items = [1]\nprint(items[-1])\n",
		"print(\"abc\"[-1])\n",
		"print(b\"abc\"[-1])\n",
	}
	for _, src := range tests {
		prog := parse(t, src)
		err := Check(prog)
		if err == nil {
			t.Fatalf("expected negative index error for %q", src)
		}
		if !strings.Contains(err.Error(), "negative indexes are invalid") {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestCheckRejectsStringOrdering(t *testing.T) {
	prog := parse(t, "print(\"a\" < \"b\")\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected string ordering error")
	}
	if !strings.Contains(err.Error(), "< expects numbers") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckDefaultParameterRules(t *testing.T) {
	t.Run("earlier param reference", func(t *testing.T) {
		prog := parse(t, "label = name, text = name -> text\nprint(label(\"Tya\"))\n")
		if err := Check(prog); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("required after default", func(t *testing.T) {
		prog := parse(t, "f = a = 1, b -> b\n")
		err := Check(prog)
		if err == nil {
			t.Fatal("expected required-after-default error")
		}
		if !strings.Contains(err.Error(), "required parameter b after default parameter") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("later param reference", func(t *testing.T) {
		prog := parse(t, "f = a = b, b = 1 -> a\n")
		err := Check(prog)
		if err == nil {
			t.Fatal("expected later-reference error")
		}
		if !strings.Contains(err.Error(), "undefined variable b") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestCheckRejectsDuplicateImports(t *testing.T) {
	tests := []string{
		"import string as s\nimport string as s\n",
		"import string as first\nimport string as second\n",
	}
	for _, src := range tests {
		prog := parse(t, src)
		err := Check(prog)
		if err == nil {
			t.Fatalf("expected duplicate import error for %q", src)
		}
		if !strings.Contains(err.Error(), "duplicate import string") {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestCheckRejectsDuplicateNormalizedDictionaryKeys(t *testing.T) {
	prog := parse(t, "user = { name: \"Tya\", \"name\": \"duplicate\" }\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected duplicate dictionary key error")
	}
	if !strings.Contains(err.Error(), "duplicate dictionary key name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRejectsKindChangingReassignmentThroughNil(t *testing.T) {
	prog := parse(t, "value = 1\nvalue = nil\nvalue = \"one\"\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected kind-changing reassignment through nil error")
	}
	if !strings.Contains(err.Error(), "cannot reassign value from number to string") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckAllowsNilToConcreteWhenNoPriorConcreteKind(t *testing.T) {
	prog := parse(t, "value = nil\nvalue = \"one\"\n")
	if err := Check(prog); err != nil {
		t.Fatal(err)
	}
}

func TestCheckKeepsBlockLocalBindingsLocal(t *testing.T) {
	prog := parse(t, "if true\n  local = 1\nprint(local)\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected block local binding error")
	}
	if !strings.Contains(err.Error(), "undefined variable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckAllowsBlockAssignmentToOuterBinding(t *testing.T) {
	prog := parse(t, "count = 1\nif true\n  count = 2\nprint(count)\n")
	if err := Check(prog); err != nil {
		t.Fatal(err)
	}
}

func TestCheckKeepsForBindingsLocal(t *testing.T) {
	prog := parse(t, "items = [1]\nfor item, index in items\n  print(item)\nprint(item)\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected for binding scope error")
	}
	if !strings.Contains(err.Error(), "undefined variable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckKeepsCatchBindingLocal(t *testing.T) {
	prog := parse(t, "try\n  raise \"bad\"\ncatch err\n  print(err)\nprint(err)\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected catch binding scope error")
	}
	if !strings.Contains(err.Error(), "undefined variable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckAllowsFunctionReassignment(t *testing.T) {
	prog := parse(t, "handler = -> 1\nhandler = -> 2\nprint(handler())\n")
	if err := Check(prog); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRejectsConstantMutation(t *testing.T) {
	prog := parse(t, "ITEMS = [1]\nITEMS[0] = 2\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected constant mutation error")
	}
	if !strings.Contains(err.Error(), "cannot mutate constant ITEMS") {
		t.Fatalf("unexpected error: %v", err)
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

func TestCheckRejectsOperatorOverloadDeclarations(t *testing.T) {
	toks, errs := lexer.Lex("class Number\n  + : other ->\n    self\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, _, err := parser.Parse(toks); err == nil {
		t.Fatal("expected operator method declaration to fail before checking")
	}
}

func TestCheckRejectsFunctionAndMethodOverloads(t *testing.T) {
	src := "class Finder\n  find: id ->\n    id\n  find: first, last ->\n    first\n"
	prog := parse(t, src)
	err := Check(prog)
	if err == nil {
		t.Fatal("expected duplicate method overload error")
	}
	if !strings.Contains(err.Error(), "duplicate instance member find") {
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

func TestCheckRejectsDictionaryMemberAccess(t *testing.T) {
	prog := parse(t, "user = { name: \"komagata\" }\nprint(user.name)\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected dictionary member access error")
	}
	if !strings.Contains(err.Error(), "cannot use . access on dictionary; use index access") {
		t.Fatalf("unexpected error: %v", err)
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
	src := "foo = \"foo\"\nbar = -> \"bar\"\nutil = { foo: foo, bar: bar }\nprint(util[\"foo\"])\nprint(util[\"bar\"]())\n"
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
		{name: "top_level_file_exists", src: "print(file_exists(\"memo.txt\"))\n", want: "use File().exists?(path)"},
		{name: "top_level_random_int", src: "print(random_int(1, 10))\n", want: "use Random().int(min, max)"},
		{name: "top_level_compiler_lexer", src: "print(compiler_lexer_lex(\"print(1)\"))\n", want: "use Lexer().lex(source)"},
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
		{"request.tya", true},
		{"http_client.tya", true},
		{"foo/bar/request.tya", true},
		{"Request.tya", false},
		{"HttpClient.tya", false},
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
	prog := parse(t, "class Request\n  initialize: url ->\n    self.url = url\n")
	if err := CheckClassFile(prog, "request.tya"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckClassFileAcceptsAcronymSnakeCaseClass(t *testing.T) {
	prog := parse(t, "class HTTPServer\n  initialize: ->\n    self.path = \"/\"\n")
	if err := CheckClassFile(prog, "http_server.tya"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckClassFileAcceptsMatchingInterface(t *testing.T) {
	prog := parse(t, "interface Request\n  send: ->\n")
	if err := CheckClassFile(prog, "request.tya"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckClassFileAcceptsImportsBeforeClass(t *testing.T) {
	prog := parse(t, "import string\n\nclass Request\n  initialize: url ->\n    self.url = url\n")
	if err := CheckClassFile(prog, "request.tya"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckClassFileAcceptsPrivateClasses(t *testing.T) {
	src := "class Header\n  initialize: name, value ->\n    self.name = name\n    self.value = value\n\n" +
		"class Request\n  initialize: url ->\n    self.url = url\n"
	prog := parse(t, src)
	if err := CheckClassFile(prog, "request.tya"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckClassFileAcceptsInterfaceCompanion(t *testing.T) {
	src := "interface Sendable\n  send: ->\n\nclass Request\n  initialize: ->\n    self.url = nil\n"
	prog := parse(t, src)
	if err := CheckClassFile(prog, "request.tya"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckAllowsSelfFieldCreationInInstanceMethods(t *testing.T) {
	src := "class Request\n  initialize: url ->\n    self.url = url\n  reset: ->\n    self.url = nil\n"
	prog := parse(t, src)
	if err := Check(prog); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRejectsTopLevelSelfFieldAssignment(t *testing.T) {
	prog := parse(t, "self.url = \"https://example.com\"\n")
	err := Check(prog)
	if err == nil {
		t.Fatal("expected top-level self error")
	}
	if !strings.Contains(err.Error(), "self is only valid inside a class method or instance method") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckInterfaceArityExactMatch(t *testing.T) {
	src := "interface Drawable\n  draw: x, y ->\n\nclass Point implements Drawable\n  draw: x ->\n    nil\n"
	prog := parse(t, src)
	err := Check(prog)
	if err == nil {
		t.Fatal("expected interface arity error")
	}
	if !strings.Contains(err.Error(), "implementing interface method draw expects 2 parameters") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckInterfaceArityIgnoresDefaultParameters(t *testing.T) {
	src := "interface Drawable\n  draw: x ->\n\nclass Point implements Drawable\n  draw: x, y = 0 ->\n    nil\n"
	prog := parse(t, src)
	err := Check(prog)
	if err == nil {
		t.Fatal("expected interface arity error")
	}
	if !strings.Contains(err.Error(), "implementing interface method draw expects 1 parameters") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckClassFileRejectsMissingPublicClass(t *testing.T) {
	prog := parse(t, "class Helper\n  initialize: ->\n    self.x = 1\n")
	err := CheckClassFile(prog, "request.tya")
	if err == nil {
		t.Fatal("expected missing public class error")
	}
	if !strings.Contains(err.Error(), "must define a class or interface that maps to request") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckClassFileRejectsNonSnakeCaseFilename(t *testing.T) {
	prog := parse(t, "class Request\n  initialize: ->\n    self.x = 1\n")
	err := CheckClassFile(prog, "Request.tya")
	if err == nil {
		t.Fatal("expected snake_case filename error")
	}
	if !strings.Contains(err.Error(), "snake_case name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckClassFileRejectsTopLevelStatement(t *testing.T) {
	src := "class Request\n  initialize: ->\n    self.x = 1\n\nx = 1\n"
	prog := parse(t, src)
	err := CheckClassFile(prog, "request.tya")
	if err == nil {
		t.Fatal("expected top-level statement error")
	}
	if !strings.Contains(err.Error(), "may only contain import, class, and interface") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckClassFileRejectsImportAfterClass(t *testing.T) {
	src := "class Request\n  initialize: ->\n    self.x = 1\n\nimport string\n"
	prog := parse(t, src)
	err := CheckClassFile(prog, "request.tya")
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
	src := "class Request\n  initialize: ->\n    self.x = 1\n\nclass Request\n  initialize: ->\n    self.y = 1\n"
	prog := parse(t, src)
	err := CheckClassFile(prog, "request.tya")
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
