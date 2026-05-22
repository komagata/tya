package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"tya/internal/ast"
	"tya/internal/lexer"
)

func TestParseIndentedDictAssignment(t *testing.T) {
	toks, errs := lexer.Lex("user =\n  name: \"komagata\"\n  age: 20\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
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
	prog, _, err := Parse(toks)
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

func TestParseDictionaryStringKeys(t *testing.T) {
	tests := []string{
		"headers = { \"Content-Type\": \"text/plain\", \"$schema\": \"x\", \"\": \"empty\" }\n",
		"headers =\n  \"Content-Type\": \"text/plain\"\n  \"$schema\": \"x\"\n  \"\": \"empty\"\n",
	}
	for _, src := range tests {
		toks, errs := lexer.Lex(src)
		if len(errs) != 0 {
			t.Fatalf("lex errors: %v", errs)
		}
		prog, _, err := Parse(toks)
		if err != nil {
			t.Fatal(err)
		}
		assign := prog.Stmts[0].(*ast.AssignStmt)
		obj, ok := assign.Values[0].(*ast.DictLit)
		if !ok {
			t.Fatalf("got %T", assign.Values[0])
		}
		if len(obj.Props) != 3 {
			t.Fatalf("got %d props", len(obj.Props))
		}
	}
}

func TestParseFunctionDefaultParameters(t *testing.T) {
	toks, errs := lexer.Lex("greet = name, suffix = \"!\" -> name + suffix\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	assign := prog.Stmts[0].(*ast.AssignStmt)
	fn := assign.Values[0].(*ast.FuncLit)
	if len(fn.Params) != 2 || fn.Params[0] != "name" || fn.Params[1] != "suffix" {
		t.Fatalf("got params %v", fn.Params)
	}
	if len(fn.Defaults) != 2 || fn.Defaults[0] != nil || fn.Defaults[1] == nil {
		t.Fatalf("got defaults %#v", fn.Defaults)
	}
}

func TestParseTryCatchFinally(t *testing.T) {
	toks, errs := lexer.Lex("try\n  print(\"try\")\ncatch err\n  print(err)\nfinally\n  print(\"done\")\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	stmt := prog.Stmts[0].(*ast.TryCatchStmt)
	if len(stmt.Try) != 1 || stmt.CatchName != "err" || len(stmt.Catch) != 1 || len(stmt.Finally) != 1 {
		t.Fatalf("unexpected try stmt: %#v", stmt)
	}
}

func TestParseTryFinally(t *testing.T) {
	toks, errs := lexer.Lex("try\n  print(\"try\")\nfinally\n  print(\"done\")\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	stmt := prog.Stmts[0].(*ast.TryCatchStmt)
	if stmt.Catch != nil || len(stmt.Finally) != 1 {
		t.Fatalf("unexpected try/finally stmt: %#v", stmt)
	}
}

func TestParseRejectsBareTry(t *testing.T) {
	toks, errs := lexer.Lex("try\n  print(\"try\")\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, _, err := Parse(toks); err == nil {
		t.Fatal("expected bare try error")
	}
}

func TestParseRejectsCatchWithoutTry(t *testing.T) {
	toks, errs := lexer.Lex("catch err\n  print(err)\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, _, err := Parse(toks); err == nil {
		t.Fatal("expected catch without try error")
	}
}

func TestParseAcceptsFormattedSyntaxTrailingCommas(t *testing.T) {
	tests := []string{
		"items = [1,]\n",
		"user = { name: \"Tya\", }\n",
		"print(1,)\n",
		"f = (a,) -> a\n",
		"f = a, b, -> a + b\n",
	}
	for _, src := range tests {
		toks, errs := lexer.Lex(src)
		if len(errs) != 0 {
			t.Fatalf("lex errors: %v", errs)
		}
		if _, _, err := Parse(toks); err != nil {
			t.Fatalf("unexpected trailing comma error for %q: %v", src, err)
		}
	}
}

func TestParseRejectsSliceSyntax(t *testing.T) {
	tests := []string{
		"items = [1, 2, 3]\nprint(items[1:3])\n",
		"items = [1, 2, 3]\nprint(items[:3])\n",
		"items = [1, 2, 3]\nprint(items[1:])\n",
	}
	for _, src := range tests {
		toks, errs := lexer.Lex(src)
		if len(errs) != 0 {
			t.Fatalf("lex errors: %v", errs)
		}
		if _, _, err := Parse(toks); err == nil {
			t.Fatalf("expected slice syntax error for %q", src)
		}
	}
}

func TestParseRejectsNamedArguments(t *testing.T) {
	toks, errs := lexer.Lex("request(url, timeout: 10)\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, _, err := Parse(toks); err == nil {
		t.Fatal("expected named argument syntax error")
	}
}

func TestParseRejectsEmptyBlocks(t *testing.T) {
	toks, errs := lexer.Lex("if true\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, _, err := Parse(toks); err == nil {
		t.Fatal("expected empty block error")
	}
}

func TestParseAllowsExplicitNilNoOpBlock(t *testing.T) {
	toks, errs := lexer.Lex("if true\n  nil\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, _, err := Parse(toks); err != nil {
		t.Fatal(err)
	}
}

func TestParseRejectsVariadicParameters(t *testing.T) {
	tests := []string{
		"f = values... -> values\n",
		"f = *args -> args\n",
	}
	for _, src := range tests {
		toks, errs := lexer.Lex(src)
		if len(errs) != 0 {
			continue
		}
		if _, _, err := Parse(toks); err == nil {
			t.Fatalf("expected variadic syntax error for %q", src)
		}
	}
}

func TestParseRejectsVariadicAndSplatSyntax(t *testing.T) {
	tests := []string{
		"fn = *args -> args\n",
		"fn = items -> items\nfn(*items)\n",
	}
	for _, src := range tests {
		toks, errs := lexer.Lex(src)
		if len(errs) != 0 {
			t.Fatalf("lex errors: %v", errs)
		}
		if _, _, err := Parse(toks); err == nil {
			t.Fatalf("expected variadic or splat syntax error for %q", src)
		}
	}
}

func TestParseRejectsDestructuringAssignment(t *testing.T) {
	valid := "pair = -> 1, 2\na, b = pair()\n"
	toks, errs := lexer.Lex(valid)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, _, err := Parse(toks); err != nil {
		t.Fatalf("multi-return assignment should remain valid: %v", err)
	}

	tests := []string{
		"[a, b] = items\n",
		"{ name } = user\n",
		"{ \"name\": name } = user\n",
	}
	for _, src := range tests {
		toks, errs := lexer.Lex(src)
		if len(errs) != 0 {
			t.Fatalf("lex errors: %v", errs)
		}
		_, _, err := Parse(toks)
		if err == nil {
			t.Fatalf("expected destructuring assignment error for %q", src)
		}
		if !strings.Contains(err.Error(), "destructuring assignment is not part of Tya v1.0.0") {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestParseRejectsMatchGuardsAndBindingPatterns(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{
			src:  "ready = true\nmatch 1\n  case _ if ready\n    print(\"ready\")\n",
			want: "match guards are not part of Tya v1.0.0",
		},
		{
			src:  "match [1, 2]\n  case [head, tail]\n    print(head)\n",
			want: "binding patterns are not part of Tya v1.0.0",
		},
		{
			src:  "match 1\n  case value\n    print(value)\n",
			want: "binding patterns are not part of Tya v1.0.0",
		},
	}
	for _, tt := range tests {
		toks, errs := lexer.Lex(tt.src)
		if len(errs) != 0 {
			t.Fatalf("lex errors: %v", errs)
		}
		_, _, err := Parse(toks)
		if err == nil {
			t.Fatalf("expected %q error", tt.want)
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestParseRejectsSetLiteral(t *testing.T) {
	toks, errs := lexer.Lex("roles = { \"admin\", \"owner\" }\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	_, _, err := Parse(toks)
	if err == nil {
		t.Fatal("expected set literal error")
	}
	if !strings.Contains(err.Error(), "set literals are not in Tya v0.1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseEmptyCurlyLiteralAsDict(t *testing.T) {
	toks, errs := lexer.Lex("empty = {}\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
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
		_, _, err := Parse(toks)
		if err == nil {
			t.Fatalf("expected mixed literal error for %q", src)
		}
		if !strings.Contains(err.Error(), "cannot mix dict entries and set entries in one literal") &&
			!strings.Contains(err.Error(), "set literals are not in Tya v0.1") {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestParseClassDeclaration(t *testing.T) {
	src := "class User\n  initialize = name ->\n    self.name = name\n\n  greeting = ->\n    \"Hello, {@name}\"\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatalf("parse class: %v", err)
	}
	if len(prog.Stmts) != 1 {
		t.Fatalf("stmt count = %d", len(prog.Stmts))
	}
	class, ok := prog.Stmts[0].(*ast.ClassDecl)
	if !ok {
		t.Fatalf("stmt type = %T", prog.Stmts[0])
	}
	if class.Name != "User" || len(class.Methods) != 2 || len(class.Fields) != 0 || len(class.Vars) != 0 {
		t.Fatalf("class = %#v", class)
	}
}

func TestParseInterfaceDeclaration(t *testing.T) {
	src := "interface Reader extends io.Source, Seekable\n  read = ->\n  write = text ->\ninterface Named extends Reader\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatalf("parse interface: %v", err)
	}
	iface, ok := prog.Stmts[0].(*ast.InterfaceDecl)
	if !ok {
		t.Fatalf("stmt type = %T", prog.Stmts[0])
	}
	if iface.Name != "Reader" || len(iface.Parents) != 2 || iface.Parents[0].Module != "io" || len(iface.Methods) != 2 || len(iface.Methods[1].Params) != 1 {
		t.Fatalf("interface = %#v", iface)
	}
	child, ok := prog.Stmts[1].(*ast.InterfaceDecl)
	if !ok {
		t.Fatalf("second stmt type = %T", prog.Stmts[1])
	}
	if child.Name != "Named" || len(child.Parents) != 1 || len(child.Methods) != 0 {
		t.Fatalf("child interface = %#v", child)
	}
}

func TestParseRejectsV01ExcludedSyntaxVariants(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "interface inheritance",
			src:  "interface Writer < Reader\n  write: text -> text\n",
			want: "expected indented block after interface",
		},
		{
			name: "nested import in function",
			src:  "load = ->\n  import greeting\n",
			want: "import must be top-level",
		},
		{
			name: "nested module in function",
			src:  "load = ->\n  module greeting\n    hello = -> \"hello\"\n",
			want: "module declarations were removed",
		},
		{
			name: "identifier set literal",
			src:  "roles = { admin }\n",
			want: "set literals are not in Tya v0.1",
		},
		{
			name: "set constructor",
			src:  "roles = set()\n",
			want: "set is not in Tya v0.1",
		},
		{
			name: "object keyword",
			src:  "value = object\n",
			want: "object is not in Tya v0.1",
		},
		{
			name: "class expression",
			src:  "value = class\n",
			want: "class cannot be used as a name",
		},
		{
			name: "interface expression",
			src:  "value = interface\n",
			want: "interface cannot be used as a name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toks, errs := lexer.Lex(tt.src)
			if len(errs) != 0 {
				t.Fatalf("lex errors: %v", errs)
			}
			_, _, err := Parse(toks)
			if err == nil {
				t.Fatalf("expected %q error", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseRejectsGenericSyntax(t *testing.T) {
	tests := []string{
		"items: Array<Int> = []\n",
		"value = Box<T>\n",
		"fn<T>(1)\n",
	}
	for _, src := range tests {
		toks, errs := lexer.Lex(src)
		if len(errs) != 0 {
			t.Fatalf("lex errors: %v", errs)
		}
		if _, _, err := Parse(toks); err == nil {
			t.Fatalf("expected generic syntax error for %q", src)
		}
	}
}

func TestParseRejectsModuleEnumRecordStructMacroAsyncSyntax(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{
			src:  "module util\n  value = 1\n",
			want: "module declarations were removed",
		},
		{
			src:  "enum Role\n  Admin\n",
			want: "enum syntax is not part of Tya v1.0.0",
		},
		{
			src:  "record User\n  name = \"\"\n",
			want: "record syntax is not part of Tya v1.0.0",
		},
		{
			src:  "struct Point\n  x = 0\n",
			want: "struct syntax is not part of Tya v1.0.0",
		},
		{
			src:  "macro debug(value)\n  value\n",
			want: "macro syntax is not part of Tya v1.0.0",
		},
		{
			src:  "async fetch = -> nil\n",
			want: "async syntax is not part of Tya v1.0.0",
		},
	}
	for _, tt := range tests {
		toks, errs := lexer.Lex(tt.src)
		if len(errs) != 0 {
			t.Fatalf("lex errors: %v", errs)
		}
		_, _, err := Parse(toks)
		if err == nil {
			t.Fatalf("expected %q error", tt.want)
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestParseRejectsUnsupportedVisibilityModifiers(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{
			src:  "protected name = 1\n",
			want: "protected syntax is not part of Tya v1.0.0",
		},
		{
			src:  "friend class Helper\n",
			want: "friend syntax is not part of Tya v1.0.0",
		},
	}
	for _, tt := range tests {
		toks, errs := lexer.Lex(tt.src)
		if len(errs) != 0 {
			t.Fatalf("lex errors: %v", errs)
		}
		_, _, err := Parse(toks)
		if err == nil {
			t.Fatalf("expected %q error", tt.want)
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestParseRejectsDefer(t *testing.T) {
	toks, errs := lexer.Lex("defer cleanup()\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	_, _, err := Parse(toks)
	if err == nil {
		t.Fatal("expected defer syntax error")
	}
	if !strings.Contains(err.Error(), "defer syntax is not part of Tya v1.0.0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseRejectsLanguageAssert(t *testing.T) {
	toks, errs := lexer.Lex("assert ready\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	_, _, err := Parse(toks)
	if err == nil {
		t.Fatal("expected assert syntax error")
	}
	if !strings.Contains(err.Error(), "assert syntax is not part of Tya v1.0.0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseRejectsCancellationSyntax(t *testing.T) {
	toks, errs := lexer.Lex("cancel task\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	_, _, err := Parse(toks)
	if err == nil {
		t.Fatal("expected cancellation syntax error")
	}
	if !strings.Contains(err.Error(), "cancel syntax is not part of Tya v1.0.0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseRejectsNonFiniteNumericLiterals(t *testing.T) {
	for _, name := range []string{"NaN", "Infinity", "nan", "infinity"} {
		t.Run(name, func(t *testing.T) {
			toks, errs := lexer.Lex("value = " + name + "\n")
			if len(errs) != 0 {
				t.Fatalf("lex errors: %v", errs)
			}
			prog, _, err := Parse(toks)
			if err != nil {
				t.Fatal(err)
			}
			assign, ok := prog.Stmts[0].(*ast.AssignStmt)
			if !ok || len(assign.Values) != 1 {
				t.Fatalf("unexpected assignment: %#v", prog.Stmts[0])
			}
			ident, ok := assign.Values[0].(*ast.Ident)
			if !ok || ident.Name != name {
				t.Fatalf("%s parsed as %#v, want identifier", name, assign.Values[0])
			}
		})
	}
}

func TestParseInstanceFieldSyntax(t *testing.T) {
	toks, errs := lexer.Lex("class User\n  initialize = name ->\n    self.name = name\n  name = ->\n    @name\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, _, err := Parse(toks); err != nil {
		t.Fatalf("parse instance fields: %v", err)
	}

	toks, errs = lexer.Lex("class User\n  name = \"\"\n  @@count = 0\n  initialize = ->\n    @@count = @@count + 1\n  @@count_users = ->\n    @@count\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatalf("parse class variables: %v", err)
	}
	class := prog.Stmts[0].(*ast.ClassDecl)
	if len(class.Fields) != 1 || len(class.Vars) != 1 || len(class.Methods) != 2 || !class.Methods[1].Class {
		t.Fatalf("class = %#v", class)
	}
}

func TestParseRejectsReservedNamesInBindingPositions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "assignment target",
			src:  "true = 1\n",
			want: "true cannot be used as a name",
		},
		{
			name: "function parameter",
			src:  "value = self -> self\n",
			want: "self cannot be used as a name",
		},
		{
			name: "multi function parameter",
			src:  "value = left, class -> left\n",
			want: "class cannot be used as a name",
		},
		{
			name: "loop value",
			src:  "for self in items\n  print self\n",
			want: "self cannot be used as a name",
		},
		{
			name: "loop index",
			src:  "for item, class in items\n  print item\n",
			want: "class cannot be used as a name",
		},
		{
			name: "import name",
			src:  "import object\n",
			want: "object is not in Tya v0.1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toks, errs := lexer.Lex(tt.src)
			if len(errs) != 0 {
				t.Fatalf("lex errors: %v", errs)
			}
			_, _, err := Parse(toks)
			if err == nil {
				t.Fatalf("expected %q error", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseRejectsModuleDeclaration(t *testing.T) {
	src := "module util\n  foo = \"foo\"\n\n  bar = name ->\n    name\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	_, diags, err := Parse(toks)
	if err == nil {
		t.Fatal("expected module removal error")
	}
	if len(diags) == 0 || diags[0].Code != CodeModuleRemoved {
		t.Fatalf("expected %s, got %#v / %v", CodeModuleRemoved, diags, err)
	}
}

func TestParseRejectsLegacyModuleColonMemberAsRemovedModule(t *testing.T) {
	src := "module util\n  foo: \"foo\"\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	_, _, err := Parse(toks)
	if err == nil {
		t.Fatal("expected legacy module member error")
	}
	if !strings.Contains(err.Error(), "module declarations were removed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseImportStatement(t *testing.T) {
	toks, errs := lexer.Lex("import http/server as http_server\nprint(http_server.listen(8080))\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	imp, ok := prog.Stmts[0].(*ast.ImportStmt)
	if !ok {
		t.Fatalf("got %T", prog.Stmts[0])
	}
	if imp.Name != "http/server" {
		t.Fatalf("got import %q", imp.Name)
	}
	if imp.Alias != "http_server" {
		t.Fatalf("got alias %q", imp.Alias)
	}
}

func TestParseRejectsImportInsideBlock(t *testing.T) {
	toks, errs := lexer.Lex("if true\n  import greeting\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	_, _, err := Parse(toks)
	if err == nil {
		t.Fatal("expected nested import error")
	}
	if !strings.Contains(err.Error(), "import must be top-level") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseRejectsModuleInsideBlock(t *testing.T) {
	toks, errs := lexer.Lex("if true\n  module greeting\n    hello = -> \"hello\"\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	_, _, err := Parse(toks)
	if err == nil {
		t.Fatal("expected nested module error")
	}
	if !strings.Contains(err.Error(), "module declarations were removed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseImportAlias(t *testing.T) {
	toks, errs := lexer.Lex("import greeting as g\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	imp, ok := prog.Stmts[0].(*ast.ImportStmt)
	if !ok {
		t.Fatalf("got %T", prog.Stmts[0])
	}
	if imp.Name != "greeting" || imp.Alias != "g" {
		t.Fatalf("got import %#v", imp)
	}
}

func TestParseSuperCall(t *testing.T) {
	toks, errs := lexer.Lex("class Admin extends User\n  initialize = name ->\n    super(name)\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatalf("parse super: %v", err)
	}
	class := prog.Stmts[0].(*ast.ClassDecl)
	if class.Parent == nil || class.Parent.Name != "User" {
		t.Fatalf("parent = %#v", class.Parent)
	}
}

func TestParseFormattedClassInheritance(t *testing.T) {
	toks, errs := lexer.Lex("class Admin < User\n  initialize = name ->\n    super(name)\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatalf("parse formatted inheritance: %v", err)
	}
	class := prog.Stmts[0].(*ast.ClassDecl)
	if class.Parent == nil || class.Parent.Name != "User" {
		t.Fatalf("parent = %#v", class.Parent)
	}
}

func TestParseMultipleFunctionParams(t *testing.T) {
	toks, errs := lexer.Lex("add = a, b -> a + b\nprint(add(2, 3))\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, _, err := Parse(toks); err != nil {
		t.Fatal(err)
	}
}

func TestParseFunctionLiteralInCallArgument(t *testing.T) {
	toks, errs := lexer.Lex("doubled = map(items, item -> item * 2)\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	assign := prog.Stmts[0].(*ast.AssignStmt)
	call := assign.Values[0].(*ast.CallExpr)
	if len(call.Args) != 2 {
		t.Fatalf("got %d args", len(call.Args))
	}
	fn, ok := call.Args[1].(*ast.FuncLit)
	if !ok {
		t.Fatalf("second arg got %T", call.Args[1])
	}
	if len(fn.Params) != 1 || fn.Params[0] != "item" {
		t.Fatalf("got params %v", fn.Params)
	}
}

func TestParseCallArgsDoNotBecomeFunctionParams(t *testing.T) {
	toks, errs := lexer.Lex("value = pair(left, right)\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	assign := prog.Stmts[0].(*ast.AssignStmt)
	call := assign.Values[0].(*ast.CallExpr)
	if len(call.Args) != 2 {
		t.Fatalf("got %d args", len(call.Args))
	}
}

func TestParseIfElse(t *testing.T) {
	toks, errs := lexer.Lex("if true\n  print(\"yes\")\nelse\n  print(\"no\")\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := prog.Stmts[0].(*ast.IfStmt); !ok {
		t.Fatalf("got %T", prog.Stmts[0])
	}
}

func TestParseIfElseifElse(t *testing.T) {
	toks, errs := lexer.Lex("if score >= 90\n  print(\"A\")\nelseif score >= 80\n  print(\"B\")\nelse\n  print(\"C\")\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	stmt, ok := prog.Stmts[0].(*ast.IfStmt)
	if !ok {
		t.Fatalf("got %T", prog.Stmts[0])
	}
	if len(stmt.Else) != 1 {
		t.Fatalf("got else block %#v", stmt.Else)
	}
	if _, ok := stmt.Else[0].(*ast.IfStmt); !ok {
		t.Fatalf("elseif got %T", stmt.Else[0])
	}
}

func TestParseZeroParamFunction(t *testing.T) {
	toks, errs := lexer.Lex("value = -> \"ok\"\nother = () -> \"ok\"\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	if len(prog.Stmts) != 2 {
		t.Fatalf("got %d statements", len(prog.Stmts))
	}
	for _, stmt := range prog.Stmts {
		assign := stmt.(*ast.AssignStmt)
		fn, ok := assign.Values[0].(*ast.FuncLit)
		if !ok {
			t.Fatalf("got %T", assign.Values[0])
		}
		if len(fn.Params) != 0 {
			t.Fatalf("got params %v", fn.Params)
		}
	}
}

func TestParseArrayLiteralAndIndex(t *testing.T) {
	toks, errs := lexer.Lex("items = [1, 2, 3]\nprint(items[0])\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, _, err := Parse(toks); err != nil {
		t.Fatal(err)
	}
}

func TestParseIndexAssignment(t *testing.T) {
	toks, errs := lexer.Lex("items[1] = 20\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, _, err := Parse(toks); err != nil {
		t.Fatal(err)
	}
}

func TestParseMemberAssignment(t *testing.T) {
	tests := []string{
		"user.name = \"komagata\"\n",
		"greeting.hello = -> \"hello\"\n",
	}
	for _, src := range tests {
		t.Run(src, func(t *testing.T) {
			toks, errs := lexer.Lex(src)
			if len(errs) != 0 {
				t.Fatalf("lex errors: %v", errs)
			}
			if _, _, err := Parse(toks); err != nil {
				t.Fatalf("parse member assignment: %v", err)
			}
		})
	}
}

func TestParseAllowsMultipleAssignmentToIndexTarget(t *testing.T) {
	toks, errs := lexer.Lex("items[0], right = pair()\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, _, err := Parse(toks); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseGroupedExpression(t *testing.T) {
	toks, errs := lexer.Lex("print (2 + 3) * 4\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, _, err := Parse(toks); err != nil {
		t.Fatal(err)
	}
}

func TestParseRejectsNestedNoParenCall(t *testing.T) {
	toks, errs := lexer.Lex("print len keys user\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	_, _, err := Parse(toks)
	if err == nil {
		t.Fatal("expected no-paren call error")
	}
	if !strings.Contains(err.Error(), "no-paren calls are not in Tya") && !strings.Contains(err.Error(), "print expects one expression") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParsePrintWithParenCalls(t *testing.T) {
	toks, errs := lexer.Lex("print(len(keys(user)))\nprint(add(2, 3))\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, _, err := Parse(toks); err != nil {
		t.Fatal(err)
	}
}

func TestParsePrintWithUnaryMinusExpression(t *testing.T) {
	toks, errs := lexer.Lex("print(-1)\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	if len(prog.Stmts) != 1 {
		t.Fatalf("got %d statements", len(prog.Stmts))
	}
	stmt := prog.Stmts[0].(*ast.ExprStmt)
	call, ok := stmt.Expr.(*ast.CallExpr)
	if !ok {
		t.Fatalf("got %T", stmt.Expr)
	}
	if len(call.Args) != 1 {
		t.Fatalf("got %d args", len(call.Args))
	}
	if _, ok := call.Args[0].(*ast.UnaryExpr); !ok {
		t.Fatalf("arg got %T", call.Args[0])
	}
}

func TestParseRejectsPrintSugarOutsideStatement(t *testing.T) {
	tests := []string{
		"value = print \"hello\"\n",
		"message = -> print \"hello\"\n",
		"user =\n  name: print \"hello\"\n",
	}
	for _, src := range tests {
		t.Run(src, func(t *testing.T) {
			toks, errs := lexer.Lex(src)
			if len(errs) != 0 {
				t.Fatalf("lex errors: %v", errs)
			}
			_, _, err := Parse(toks)
			if err == nil {
				t.Fatal("expected print sugar error")
			}
			if !strings.Contains(err.Error(), "no-paren calls are not in Tya") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseRejectsNoParenCall(t *testing.T) {
	toks, errs := lexer.Lex("add 2, 3\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	_, _, err := Parse(toks)
	if err == nil {
		t.Fatal("expected no-paren call error")
	}
	if !strings.Contains(err.Error(), "no-paren calls are not in Tya") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseWhile(t *testing.T) {
	toks, errs := lexer.Lex("while i < 5\n  i = i + 1\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := prog.Stmts[0].(*ast.WhileStmt); !ok {
		t.Fatalf("got %T", prog.Stmts[0])
	}
}

func TestParseReturnFunctionThenTopLevelPrint(t *testing.T) {
	src := "find_first_over = limit ->\n  i = 0\n  while true\n    if i > limit\n      return i\n    i = i + 1\n\nprint(find_first_over(3))\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
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
	toks, errs := lexer.Lex("for item, index in items\n  print(item)\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := prog.Stmts[0].(*ast.ForInStmt); !ok {
		t.Fatalf("got %T", prog.Stmts[0])
	}
}

func TestParseMultipleAssignment(t *testing.T) {
	toks, errs := lexer.Lex("left, right = pair\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	assign, ok := prog.Stmts[0].(*ast.AssignStmt)
	if !ok {
		t.Fatalf("got %T", prog.Stmts[0])
	}
	if len(assign.Targets) != 2 {
		t.Fatalf("got %d targets", len(assign.Targets))
	}
	if len(assign.Values) != 1 {
		t.Fatalf("got %d values", len(assign.Values))
	}
}

func TestParseReturnMultipleValues(t *testing.T) {
	toks, errs := lexer.Lex("parse_user = text ->\n  return { name: text }, nil\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	assign := prog.Stmts[0].(*ast.AssignStmt)
	fn := assign.Values[0].(*ast.FuncLit)
	ret, ok := fn.Body[0].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf("got %T", fn.Body[0])
	}
	if len(ret.Values) != 2 {
		t.Fatalf("got %d return values", len(ret.Values))
	}
}

func TestParseRejectsForOfDictionaryIteration(t *testing.T) {
	toks, errs := lexer.Lex("for key, value of user\n  print(value)\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, _, err := Parse(toks); err == nil {
		t.Fatal("expected for of parse error")
	}
}

func TestParseBreakAndContinue(t *testing.T) {
	toks, errs := lexer.Lex("while true\n  continue\n  break\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	loop := prog.Stmts[0].(*ast.WhileStmt)
	if _, ok := loop.Body[0].(*ast.ContinueStmt); !ok {
		t.Fatalf("first loop stmt got %T", loop.Body[0])
	}
	if _, ok := loop.Body[1].(*ast.BreakStmt); !ok {
		t.Fatalf("second loop stmt got %T", loop.Body[1])
	}
}

func TestParseRejectsControlFlowOutsideAllowedContext(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "top-level return",
			src:  "return 1\n",
			want: "return must be inside a function",
		},
		{
			name: "top-level break",
			src:  "break\n",
			want: "break must be inside a loop",
		},
		{
			name: "top-level continue",
			src:  "continue\n",
			want: "continue must be inside a loop",
		},
		{
			name: "break inside function but outside loop",
			src:  "stop = ->\n  break\n",
			want: "break must be inside a loop",
		},
		{
			name: "continue inside if but outside loop",
			src:  "if true\n  continue\n",
			want: "continue must be inside a loop",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toks, errs := lexer.Lex(tt.src)
			if len(errs) != 0 {
				t.Fatalf("lex errors: %v", errs)
			}
			_, _, err := Parse(toks)
			if err == nil {
				t.Fatalf("expected %q error", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseAllowsReturnAndLoopControlInFunctionLoop(t *testing.T) {
	src := "find = items ->\n  for item in items\n    if item == nil\n      continue\n    if item == \"stop\"\n      break\n    return item\n  return nil\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	if _, _, err := Parse(toks); err != nil {
		t.Fatal(err)
	}
}

func TestParseRejectsOuterLoopControlInsideNestedFunction(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "break",
			src:  "while true\n  stop = ->\n    break\n",
			want: "break must be inside a loop",
		},
		{
			name: "continue",
			src:  "while true\n  skip = ->\n    continue\n",
			want: "continue must be inside a loop",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toks, errs := lexer.Lex(tt.src)
			if len(errs) != 0 {
				t.Fatalf("lex errors: %v", errs)
			}
			_, _, err := Parse(toks)
			if err == nil {
				t.Fatalf("expected %q error", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseRejectsTrailingTokensInBlockHeaderExpressions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "if",
			src:  "if ready true\n  print \"bad\"\n",
			want: "no-paren calls are not in Tya",
		},
		{
			name: "elseif",
			src:  "if false\n  print \"no\"\nelseif ready true\n  print \"bad\"\n",
			want: "no-paren calls are not in Tya",
		},
		{
			name: "while",
			src:  "while ready true\n  print \"bad\"\n",
			want: "no-paren calls are not in Tya",
		},
		{
			name: "for",
			src:  "for item in items extra\n  print item\n",
			want: "no-paren calls are not in Tya",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toks, errs := lexer.Lex(tt.src)
			if len(errs) != 0 {
				t.Fatalf("lex errors: %v", errs)
			}
			_, _, err := Parse(toks)
			if err == nil {
				t.Fatalf("expected %q error", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseTryExpression(t *testing.T) {
	tests := []string{
		"load_user = text ->\n  user = try parse_user(text)\n  user[\"name\"]\n",
		"load_user = text -> try parse_user(text)\n",
	}
	for _, src := range tests {
		toks, errs := lexer.Lex(src)
		if len(errs) != 0 {
			t.Fatalf("lex errors: %v", errs)
		}
		_, _, err := Parse(toks)
		if err == nil {
			t.Fatal("expected try expression error")
		}
		if !strings.Contains(err.Error(), "try expressions are not part of Tya v1.0.0") {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestParseRejectsTryOutsideFunction(t *testing.T) {
	tests := []string{
		"value = try parse_user(text)\n",
		"print(try parse_user(text))\n",
		"if try ready()\n  print(\"ready\")\n",
	}
	for _, src := range tests {
		t.Run(src, func(t *testing.T) {
			toks, errs := lexer.Lex(src)
			if len(errs) != 0 {
				t.Fatalf("lex errors: %v", errs)
			}
			_, _, err := Parse(toks)
			if err == nil {
				t.Fatal("expected try context error")
			}
			if !strings.Contains(err.Error(), "try expressions are not part of Tya v1.0.0") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseRejectsUnsupportedCatchForms(t *testing.T) {
	tests := []string{
		"try\n  raise error(\"bad\")\ncatch Error err\n  print(err)\n",
		"try\n  raise error(\"bad\")\ncatch { \"kind\": \"io\" }\n  print(\"io\")\n",
		"try\n  raise error(\"bad\")\ncatch err if err[\"kind\"] == \"io\"\n  print(err)\n",
		"try\n  raise error(\"bad\")\ncatch first\n  print(first)\ncatch second\n  print(second)\n",
	}
	for _, src := range tests {
		toks, errs := lexer.Lex(src)
		if len(errs) != 0 {
			t.Fatalf("lex errors: %v", errs)
		}
		if _, _, err := Parse(toks); err == nil {
			t.Fatalf("expected unsupported catch form error for %q", src)
		}
	}
}

func TestParseRejectsTypedPatternAndMultipleCatch(t *testing.T) {
	tests := []string{
		"try\n  raise error(\"bad\")\ncatch Error err\n  print(err)\n",
		"try\n  raise error(\"bad\")\ncatch { \"kind\": \"io\" }\n  print(\"io\")\n",
		"try\n  raise error(\"bad\")\ncatch first\n  print(first)\ncatch second\n  print(second)\n",
	}
	for _, src := range tests {
		t.Run(src, func(t *testing.T) {
			toks, errs := lexer.Lex(src)
			if len(errs) != 0 {
				t.Fatalf("lex errors: %v", errs)
			}
			if _, _, err := Parse(toks); err == nil {
				t.Fatal("expected unsupported catch form error")
			}
		})
	}
}

func TestParseModuleMemberCall(t *testing.T) {
	toks, errs := lexer.Lex("print(greeting.hello(\"komagata\"))\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	stmt := prog.Stmts[0].(*ast.ExprStmt)
	printCall := stmt.Expr.(*ast.CallExpr)
	memberCall, ok := printCall.Args[0].(*ast.CallExpr)
	if !ok {
		t.Fatalf("got %T", printCall.Args[0])
	}
	if _, ok := memberCall.Callee.(*ast.MemberExpr); !ok {
		t.Fatalf("callee got %T", memberCall.Callee)
	}
}

func TestParseV01Examples(t *testing.T) {
	root := filepath.Join("..", "..", "examples")
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "archive" {
			return filepath.SkipDir
		}
		if d.IsDir() || filepath.Ext(path) != ".tya" {
			return nil
		}
		t.Run(strings.TrimPrefix(path, root+string(os.PathSeparator)), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			toks, errs := lexer.Lex(string(raw))
			if len(errs) != 0 {
				t.Fatalf("lex errors: %v", errs)
			}
			if _, _, err := Parse(toks); err != nil {
				t.Fatal(err)
			}
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseASTGoldenV01CoreProgram(t *testing.T) {
	src := strings.Join([]string{
		"import greeting",
		"",
		"user = { name: \"komagata\", age: 20 }",
		"",
		"score_label = score ->",
		"  if score >= 90",
		"    return \"A\"",
		"  elseif score >= 80",
		"    return \"B\"",
		"  else",
		"    return \"C\"",
		"",
		"for entry in user",
		"  print(entry[\"value\"])",
		"",
		"print(greeting.hello(user[\"name\"]))",
	}, "\n") + "\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	got := dumpProgram(prog)
	want := strings.Join([]string{
		"import greeting",
		"assign ident(user) = dict{name: string(\"komagata\"), age: int(20)}",
		"assign ident(score_label) = func(score) block[if binary(ident(score) >= int(90)) then[return string(\"A\")] else[if binary(ident(score) >= int(80)) then[return string(\"B\")] else[return string(\"C\")]]]",
		"for entry in ident(user) [expr call(ident(print), index(ident(entry), string(\"value\")))]",
		"expr call(ident(print), call(member(ident(greeting).hello), index(ident(user), string(\"name\"))))",
	}, "\n")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("AST golden mismatch (-want +got):\n%s", diff)
	}
}

func TestParseASTGoldenFunctionLiteralInCall(t *testing.T) {
	toks, errs := lexer.Lex("doubled = map(items, item -> item * 2)\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	got := dumpProgram(prog)
	want := "assign ident(doubled) = call(ident(map), ident(items), func(item) expr(binary(ident(item) * int(2))))"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("AST golden mismatch (-want +got):\n%s", diff)
	}
}

func TestParseASTGoldenPostfixPrecedence(t *testing.T) {
	src := strings.Join([]string{
		"value = modules[0].factory(\"x\")[1]",
		"call_result = api.client().send(payload[0])",
	}, "\n") + "\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	got := dumpProgram(prog)
	want := strings.Join([]string{
		"assign ident(value) = index(call(member(index(ident(modules), int(0)).factory), string(\"x\")), int(1))",
		"assign ident(call_result) = call(member(call(member(ident(api).client)).send), index(ident(payload), int(0)))",
	}, "\n")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("AST golden mismatch (-want +got):\n%s", diff)
	}
}

func dumpProgram(prog *ast.Program) string {
	lines := make([]string, 0, len(prog.Stmts))
	for _, stmt := range prog.Stmts {
		lines = append(lines, dumpStmt(stmt))
	}
	return strings.Join(lines, "\n")
}

func dumpStmt(stmt ast.Stmt) string {
	switch n := stmt.(type) {
	case *ast.ImportStmt:
		if n.Alias != "" {
			return "import " + n.Name + " as " + n.Alias
		}
		return "import " + n.Name
	case *ast.ModuleDecl:
		parts := make([]string, 0, len(n.Members))
		for _, member := range n.Members {
			parts = append(parts, member.Name+": "+dumpExpr(member.Value))
		}
		return fmt.Sprintf("module %s {%s}", n.Name, strings.Join(parts, ", "))
	case *ast.AssignStmt:
		targets := make([]string, 0, len(n.Targets))
		for _, target := range n.Targets {
			targets = append(targets, dumpExpr(target))
		}
		values := make([]string, 0, len(n.Values))
		for _, value := range n.Values {
			values = append(values, dumpExpr(value))
		}
		return fmt.Sprintf("assign %s = %s", strings.Join(targets, ", "), strings.Join(values, ", "))
	case *ast.ExprStmt:
		return "expr " + dumpExpr(n.Expr)
	case *ast.IfStmt:
		return fmt.Sprintf("if %s then[%s] else[%s]", dumpExpr(n.Cond), dumpStmts(n.Then), dumpStmts(n.Else))
	case *ast.WhileStmt:
		return fmt.Sprintf("while %s [%s]", dumpExpr(n.Cond), dumpStmts(n.Body))
	case *ast.ForInStmt:
		names := n.ValueName
		if n.IndexName != "" {
			names += "," + n.IndexName
		}
		return fmt.Sprintf("for %s %s %s [%s]", names, n.Kind, dumpExpr(n.Iterable), dumpStmts(n.Body))
	case *ast.BreakStmt:
		return "break"
	case *ast.ContinueStmt:
		return "continue"
	case *ast.ReturnStmt:
		values := make([]string, 0, len(n.Values))
		for _, value := range n.Values {
			values = append(values, dumpExpr(value))
		}
		return "return " + strings.Join(values, ", ")
	default:
		return fmt.Sprintf("%T", stmt)
	}
}

func dumpStmts(stmts []ast.Stmt) string {
	parts := make([]string, 0, len(stmts))
	for _, stmt := range stmts {
		parts = append(parts, dumpStmt(stmt))
	}
	return strings.Join(parts, "; ")
}

func dumpExpr(expr ast.Expr) string {
	switch n := expr.(type) {
	case *ast.Ident:
		return "ident(" + n.Name + ")"
	case *ast.IntLit:
		return fmt.Sprintf("int(%d)", n.Value)
	case *ast.FloatLit:
		return fmt.Sprintf("float(%g)", n.Value)
	case *ast.StringLit:
		return fmt.Sprintf("string(%q)", n.Value)
	case *ast.BoolLit:
		return fmt.Sprintf("bool(%t)", n.Value)
	case *ast.NilLit:
		return "nil"
	case *ast.DictLit:
		props := make([]string, 0, len(n.Props))
		for _, prop := range n.Props {
			props = append(props, prop.Name+": "+dumpExpr(prop.Value))
		}
		return "dict{" + strings.Join(props, ", ") + "}"
	case *ast.ArrayLit:
		elems := make([]string, 0, len(n.Elems))
		for _, elem := range n.Elems {
			elems = append(elems, dumpExpr(elem))
		}
		return "array[" + strings.Join(elems, ", ") + "]"
	case *ast.FuncLit:
		if len(n.Body) != 0 {
			return fmt.Sprintf("func(%s) block[%s]", strings.Join(n.Params, ", "), dumpStmts(n.Body))
		}
		return fmt.Sprintf("func(%s) expr(%s)", strings.Join(n.Params, ", "), dumpExpr(n.Expr))
	case *ast.BinaryExpr:
		return fmt.Sprintf("binary(%s %s %s)", dumpExpr(n.Left), n.Op.Lexeme, dumpExpr(n.Right))
	case *ast.UnaryExpr:
		return fmt.Sprintf("unary(%s %s)", n.Op.Lexeme, dumpExpr(n.Expr))
	case *ast.TryExpr:
		return "try " + dumpExpr(n.Expr)
	case *ast.MemberExpr:
		return fmt.Sprintf("member(%s.%s)", dumpExpr(n.Target), n.Name)
	case *ast.IndexExpr:
		return fmt.Sprintf("index(%s, %s)", dumpExpr(n.Target), dumpExpr(n.Index))
	case *ast.CallExpr:
		parts := []string{dumpExpr(n.Callee)}
		for _, arg := range n.Args {
			parts = append(parts, dumpExpr(arg))
		}
		return "call(" + strings.Join(parts, ", ") + ")"
	default:
		return fmt.Sprintf("%T", expr)
	}
}
