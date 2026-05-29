package formatter

import (
	"strings"
	"testing"

	"tya/internal/lexer"
	"tya/internal/parser"
)

func unparseSourceWithComments(t *testing.T, src string) (string, error) {
	t.Helper()
	toks, lcomments, errs := lexer.LexWithComments(src)
	if len(errs) != 0 {
		t.Fatalf("lex errs: %v", errs)
	}
	comments := make([]parser.CommentInfo, 0, len(lcomments))
	for _, c := range lcomments {
		comments = append(comments, parser.CommentInfo{Line: c.Line, Col: c.Col, Indent: c.Indent, Text: c.Text, IsFullLine: c.IsFullLine})
	}
	prog, _, err := parser.ParseWithComments(toks, comments)
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	return Unparse(prog)
}

func TestUnparseEmitsHeaderAndStmtComments(t *testing.T) {
	src := "# header line one\n# header line two\n\n# greet a user\ngreet = name -> name\nx = 1  # initial value\n"
	got, err := unparseSourceWithComments(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"# header line one",
		"# header line two",
		"# greet a user",
		"greet = name -> name",
		"x = 1  # initial value",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestUnparseKeepsCommentedFunctionBodyBlockBodied(t *testing.T) {
	src := strings.Join([]string{
		"foo = ->",
		"  # value docs.",
		"  return \"aaa\"",
		"",
	}, "\n")
	got, err := unparseSourceWithComments(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"foo = ->",
		"  # value docs.",
		"  return \"aaa\"",
		"",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%swant:\n%s", got, want)
	}
	again, err := unparseSourceWithComments(t, got)
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Fatalf("not idempotent:\nfirst:\n%s\nsecond:\n%s", got, again)
	}
}

func TestUnparseWrapsLongClassArrayLiteral(t *testing.T) {
	src := strings.Join([]string{
		"class FlakewatchCommand",
		"  options: [cli.Spec.option(\"junit\", \"JUnit XML glob or file path\", { default: [], array: true }), cli.Spec.option(\"history\", \"JSONL history glob or file path\", { default: [], array: true })]",
		"",
	}, "\n")
	got, err := unparseSourceWithComments(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"class FlakewatchCommand",
		"  options: [",
		"    cli.Spec.option(\"junit\", \"JUnit XML glob or file path\", { default: [], array: true }),",
		"    cli.Spec.option(\"history\", \"JSONL history glob or file path\", { default: [], array: true })",
		"  ]",
		"",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%swant:\n%s", got, want)
	}
	again, err := unparseSourceWithComments(t, got)
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Fatalf("not idempotent:\nfirst:\n%s\nsecond:\n%s", got, again)
	}
}

func TestUnparseSeparatesClassAndInterfaceMembersBeforeComments(t *testing.T) {
	src := strings.Join([]string{
		"class Foo",
		"  a: 1",
		"  # b docs.",
		"  b: 2",
		"",
		"interface Iterator",
		"  # has_next docs.",
		"  has_next: ->",
		"  # next docs.",
		"  next: ->",
		"",
	}, "\n")
	got, err := unparseSourceWithComments(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"class Foo",
		"  a: 1",
		"",
		"  # b docs.",
		"  b: 2",
		"",
		"interface Iterator",
		"  # has_next docs.",
		"  has_next: ->",
		"  # next docs.",
		"  next: ->",
		"",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%swant:\n%s", got, want)
	}
	again, err := unparseSourceWithComments(t, got)
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Fatalf("not idempotent:\nfirst:\n%s\nsecond:\n%s", got, again)
	}
}

func TestUnparseKeepsRecordDeclarationAndFieldComments(t *testing.T) {
	src := strings.Join([]string{
		"# Option docs.",
		"record Option",
		"  # kind docs.",
		"  kind",
		"",
		"  # name docs.",
		"  name",
		"",
	}, "\n")
	got, err := unparseSourceWithComments(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if got != src {
		t.Fatalf("got:\n%swant:\n%s", got, src)
	}
	again, err := unparseSourceWithComments(t, got)
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Fatalf("not idempotent:\nfirst:\n%s\nsecond:\n%s", got, again)
	}
}

func TestUnparseSeparatesStructAndRecordFields(t *testing.T) {
	src := strings.Join([]string{
		"struct User",
		"  name",
		"  age: 0",
		"",
		"record Point",
		"  x",
		"  y",
		"",
	}, "\n")
	want := strings.Join([]string{
		"struct User",
		"  name",
		"",
		"  age: 0",
		"",
		"record Point",
		"  x",
		"",
		"  y",
		"",
	}, "\n")
	got, err := unparseSourceWithComments(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got:\n%swant:\n%s", got, want)
	}
	again, err := unparseSourceWithComments(t, got)
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Fatalf("not idempotent:\nfirst:\n%s\nsecond:\n%s", got, again)
	}
}

func TestUnparseSeparatesTypeBodyFromFollowingStatement(t *testing.T) {
	src := strings.Join([]string{
		"class Foo",
		"  name: \"\"",
		"print(\"class\")",
		"",
		"interface Named",
		"  name: ->",
		"print(\"interface\")",
		"",
		"struct User",
		"  name",
		"print(\"struct\")",
		"",
		"record Point",
		"  x",
		"print(\"record\")",
		"",
	}, "\n")
	want := strings.Join([]string{
		"class Foo",
		"  name: \"\"",
		"",
		"print(\"class\")",
		"interface Named",
		"  name: ->",
		"",
		"print(\"interface\")",
		"struct User",
		"  name",
		"",
		"print(\"struct\")",
		"record Point",
		"  x",
		"",
		"print(\"record\")",
		"",
	}, "\n")
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got:\n%swant:\n%s", got, want)
	}
	again, err := unparseSource(t, got)
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Fatalf("not idempotent:\nfirst:\n%s\nsecond:\n%s", got, again)
	}
}

func unparseSource(t *testing.T, src string) (string, error) {
	t.Helper()
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errs: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	return Unparse(prog)
}

func TestUnparseAssignAndPrint(t *testing.T) {
	src := "x = 1\nprint(x)\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := "x = 1\nprint(x)\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestUnparseNilCoalescingAssignment(t *testing.T) {
	src := "value??=\"fallback\"\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := "value ??= \"fallback\"\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestUnparseBinaryAndCall(t *testing.T) {
	src := "y = 2 + 3 * 4\nprint(y)\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "y = 2 + 3 * 4") {
		t.Errorf("got: %q", got)
	}
}

func TestUnparseIfElseifElse(t *testing.T) {
	src := "x = 1\nif x == 0\n  print(\"a\")\nelseif x == 1\n  print(\"b\")\nelse\n  print(\"c\")\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"if x == 0",
		"  print(\"a\")",
		"elseif x == 1",
		"  print(\"b\")",
		"else",
		"  print(\"c\")",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestUnparseControlFlowAssignmentExpressions(t *testing.T) {
	src := "label = if score >= 90\n  \"A\"\nelseif score >= 80\n  \"B\"\nelseif score >= 70\n  \"C\"\nelse\n  \"D\"\nlast = for item in items\n  item\nresult = match status\n  case \"ok\"\n    \"success\"\n  case _\n    \"fallback\"\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "else\n  if") {
		t.Fatalf("elseif chain was nested:\n%s", got)
	}
	for _, want := range []string{
		"label = if score >= 90",
		"elseif score >= 80",
		"elseif score >= 70",
		"last = for item in items",
		"result = match status",
		"case _",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestFormatTryFinally(t *testing.T) {
	src := "try\n  print(\"try\")\ncatch err\n  print(err)\nfinally\n  print(\"done\")\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if got != src {
		t.Fatalf("got %q want %q", got, src)
	}
}

func TestUnparseLambdaSingleLine(t *testing.T) {
	src := "add = a, b -> a + b\nprint(add(2, 3))\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "add = a, b -> a + b") {
		t.Errorf("got: %q", got)
	}
}

func TestUnparseAcceptedSyntaxToFormattedSyntax(t *testing.T) {
	src := strings.Join([]string{
		"items = [1, 2,]",
		"user = { name: 'Ada', age: 20, }",
		"add = (a, b,) -> a + b",
		"print(add(1, 2,))",
		"",
	}, "\n")
	want := strings.Join([]string{
		"items = [1, 2]",
		"user = { name: \"Ada\", age: 20 }",
		"add = a, b -> a + b",
		"print(add(1, 2))",
		"",
	}, "\n")
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got:\n%swant:\n%s", got, want)
	}
	again, err := unparseSource(t, got)
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Fatalf("not idempotent:\n%s", again)
	}
}

func TestUnparseSingleQuotedStringPreservesLiteralValue(t *testing.T) {
	src := "value = '{name} } \" \\\\ \\' \\n \\t \\r'\n"
	want := "value = \"{{name}} }} \\\" \\\\ ' \\n \\t \\r\"\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestUnparseSelfAndSuperExpressions(t *testing.T) {
	src := "class Box\n  static get: ->\n    return Self.wrap(self.value + super.value)\n"
	want := "class Box\n  static get: -> wrap(self.value + super.value)\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got:\n%swant:\n%s", got, want)
	}
}

func TestUnparseZeroArgFunctionDefinitionsUseShortArrow(t *testing.T) {
	src := strings.Join([]string{
		"helper = () ->",
		"  return true",
		"",
		"inline = () -> true",
		"",
		"with_args = name -> name",
		"",
		"class Box",
		"  initialize: () ->",
		"    self.value = 1",
		"",
		"  static build: () -> Box.new()",
		"",
	}, "\n")
	want := strings.Join([]string{
		"helper = -> true",
		"",
		"inline = -> true",
		"",
		"with_args = name -> name",
		"",
		"class Box",
		"  static build: -> new()",
		"",
		"  initialize: ->",
		"    self.value = 1",
		"",
	}, "\n")
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got:\n%swant:\n%s", got, want)
	}
}

func TestUnparseLambdaBlock(t *testing.T) {
	src := "f = x ->\n  y = x + 1\n  return y\nprint(f(2))\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"f = x ->", "  y = x + 1", "  y", "print(f(2))"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestUnparseImports(t *testing.T) {
	src := "import string\nimport file as f\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := "import file as f\nimport string\n"
	if got != want {
		t.Errorf("got: %q want %q", got, want)
	}
}

func TestUnparseIdempotent(t *testing.T) {
	cases := []string{
		"x = 1\n",
		"x = 1\nprint(x)\n",
		"x = 1\ny = x + 2 * 3\nprint(y)\n",
		"add = (a, b) -> a + b\nprint(add(2, 3))\n",
		"f = x ->\n  y = x + 1\n  return y\nprint(f(2))\n",
		"if x == 0\n  print(\"a\")\nelseif x == 1\n  print(\"b\")\nelse\n  print(\"c\")\n",
		"items = [1, 2, 3]\nfor item in items\n  print(item)\n",
		// Wrap forms now round-trip after lexer/parser learned
		// to ignore newlines inside (...) and [...].
		"result = compute_filtered_items(source_alpha, source_beta, source_gamma, source_delta)\n",
		"items = [first_item_name, second_item_name, third_item_name, fourth_item_name, fifth_item_name]\n",
	}
	for _, src := range cases {
		first, err := unparseSource(t, src)
		if err != nil {
			t.Fatalf("unparse(parse(%q)): %v", src, err)
		}
		second, err := unparseSource(t, first)
		if err != nil {
			t.Fatalf("unparse(parse(unparse(parse(%q)))): %v", src, err)
		}
		if first != second {
			t.Errorf("not idempotent for %q\nfirst:\n%s\nsecond:\n%s", src, first, second)
		}
	}
}

func TestUnparseRewritesLongStringWithNewlines(t *testing.T) {
	src := "msg = \"line one of message\\nline two of message\\nline three of message and a bit more here\"\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"msg = \"\"\"",
		"  line one of message",
		"  line two of message",
		"  line three of message and a bit more here",
		"  \"\"\"",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// Idempotent.
	second, err := unparseSource(t, got)
	if err != nil {
		t.Fatal(err)
	}
	if got != second {
		t.Errorf("not idempotent\nfirst:\n%s\nsecond:\n%s", got, second)
	}
}

func TestUnparseLongStringWithoutNewlinesStaysSingleLine(t *testing.T) {
	src := "url = \"https://example.com/path/to/very/long/resource/that/does/not/contain/whitespace\"\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "url = \"https://example.com") || strings.Contains(got, `"""`) {
		t.Errorf("expected single-line atomic-token exception, got:\n%s", got)
	}
}

func TestUnparseWrapsLongDictToBlock(t *testing.T) {
	src := "user = { full_name: \"Alice Example\", role: \"administrator-level\", region: \"asia\" }\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"user =",
		"  full_name: \"Alice Example\"",
		"  role: \"administrator-level\"",
		"  region: \"asia\"",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// Idempotent.
	second, err := unparseSource(t, got)
	if err != nil {
		t.Fatal(err)
	}
	if got != second {
		t.Errorf("not idempotent\nfirst:\n%s\nsecond:\n%s", got, second)
	}
}

func TestUnparseWrapsBinaryChain(t *testing.T) {
	src := "total = first_part_value + second_part_value + third_part_value + fourth_part_value\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"total = first_part_value",
		"  + second_part_value",
		"  + third_part_value",
		"  + fourth_part_value",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestUnparseBinaryChainIdempotent(t *testing.T) {
	src := "total = first_part_value + second_part_value + third_part_value + fourth_part_value\n"
	first, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	second, err := unparseSource(t, first)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Errorf("not idempotent\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestUnparseWrapsLongIfCondition(t *testing.T) {
	src := "if some_condition_value + another_part_value > threshold_limit and not exceptional_case_was_seen\n  process(a, b)\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"if (",
		"  some_condition_value + another_part_value > threshold_limit and not exceptional_case_was_seen",
		")",
		"  process(a, b)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestUnparseWrapsLongLambdaBody(t *testing.T) {
	src := "greet = recipient_name -> \"Hello, \" + recipient_name + \"! Welcome to the service.\"\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"greet = recipient_name ->",
		"  \"Hello, \" + recipient_name + \"! Welcome to the service.\"",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestUnparseWrapsLongCall(t *testing.T) {
	src := "result = compute_filtered_items(source_alpha, source_beta, source_gamma, source_delta)\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := "result = compute_filtered_items(\n  source_alpha,\n  source_beta,\n  source_gamma,\n  source_delta\n)\n"
	if got != want {
		t.Errorf("wrap mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestUnparseWrapsLongReturnCall(t *testing.T) {
	src := "build = ->\n  if has_alpha\n    return rgba(color_parts[0].trim().to_i(), color_parts[1].trim().to_i(), color_parts[2].trim().to_i(), color_parts[3].trim().to_i())\n  rgb(0, 0, 0)\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := "build = ->\n  if has_alpha\n    return rgba(\n      color_parts[0].trim().to_i(),\n      color_parts[1].trim().to_i(),\n      color_parts[2].trim().to_i(),\n      color_parts[3].trim().to_i()\n    )\n  rgb(0, 0, 0)\n"
	if got != want {
		t.Errorf("wrap mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
	again, err := unparseSource(t, got)
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Fatalf("not idempotent:\n%s", again)
	}
}

func TestUnparseWrapsLongCallChain(t *testing.T) {
	src := "value = validation.Validation.number(value, name, \"color.channel\").integer().between(0, 255).validate!()\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := "value = validation.Validation\n  .number(value, name, \"color.channel\")\n  .integer()\n  .between(0, 255)\n  .validate!()\n"
	if got != want {
		t.Errorf("wrap mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
	again, err := unparseSource(t, got)
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Fatalf("not idempotent:\n%s", again)
	}
}

func TestUnparseWrapsLongBinaryExprStmt(t *testing.T) {
	src := "abs(self.r - other.r) <= tolerance and abs(self.g - other.g) <= tolerance and abs(self.b - other.b) <= tolerance and abs(self.a - other.a) <= tolerance\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := "abs(self.r - other.r) <= tolerance\n  and abs(self.g - other.g) <= tolerance\n  and abs(self.b - other.b) <= tolerance\n  and abs(self.a - other.a) <= tolerance\n"
	if got != want {
		t.Errorf("wrap mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
	again, err := unparseSource(t, got)
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Fatalf("not idempotent:\n%s", again)
	}
}

func TestUnparseWrapsLongArray(t *testing.T) {
	src := "items = [first_item_name, second_item_name, third_item_name, fourth_item_name, fifth_item_name]\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"items = [\n  first_item_name,\n", "  fifth_item_name\n]\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestUnparseImportSortAndBlankLines(t *testing.T) {
	src := "import zmod\nimport string\nimport file\nimport mylib\n\ngreet = name -> name\nx = 1\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := "import file\nimport string\nimport mylib\nimport zmod\n\ngreet = name -> name\nx = 1\n"
	if got != want {
		t.Errorf("import sort/blank-line layout mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
	again, err := unparseSource(t, got)
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Fatalf("not idempotent:\n%s", again)
	}
}

func TestUnparsePreservesPostfixReceiverGrouping(t *testing.T) {
	src := strings.Join([]string{
		"new_w = (self.width * scale).to_i()",
		"sx = ((x * self.width) / width.to_f()).to_i()",
		"r = (top.r * a + under.r * inv).to_i()",
		"",
	}, "\n")
	want := strings.Join([]string{
		"new_w = (self.width * scale).to_i()",
		"sx = (x * self.width / width.to_f()).to_i()",
		"r = (top.r * a + under.r * inv).to_i()",
		"",
	}, "\n")
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("postfix receiver grouping changed\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestUnparsePreservesFloatLiteralKind(t *testing.T) {
	src := strings.Join([]string{
		"a = top.a / 255.0",
		"b = 1.5",
		"",
	}, "\n")
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if got != src {
		t.Fatalf("float literal kind changed\nwant:\n%s\ngot:\n%s", src, got)
	}
}

func TestUnparseClass(t *testing.T) {
	src := "class Dog\n  bark: ->\n    return \"woof\"\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"class Dog", "  bark: -> \"woof\""} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestUnparsePrivatePredicateMethodName(t *testing.T) {
	src := "class Address\n  private?: -> true\n  static private?: -> false\n  private helper: -> true\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := "class Address\n  static private?: -> false\n\n  private?: -> true\n\n  private helper: -> true\n"
	if got != want {
		t.Fatalf("got:\n%swant:\n%s", got, want)
	}
}

func TestUnparseOmitsFinalReturnInFunctionBodies(t *testing.T) {
	src := strings.Join([]string{
		"double = x ->",
		"  y = x * 2",
		"  return y",
		"",
		"class Box",
		"  value: ->",
		"    return 1",
		"",
		"  branch: ok ->",
		"    if ok",
		"      return 1",
		"    else",
		"      return 2",
		"",
		"  choose: a, b ->",
		"    if a",
		"      return 1",
		"    elseif b",
		"      return 2",
		"    else",
		"      return 3",
		"",
	}, "\n")
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"double = x ->",
		"  y = x * 2",
		"  y",
		"",
		"class Box",
		"  branch: ok ->",
		"    if ok",
		"      1",
		"    else",
		"      2",
		"",
		"  choose: a, b ->",
		"    if a",
		"      1",
		"    elseif b",
		"      2",
		"    else",
		"      3",
		"",
		"  value: -> 1",
	}, "\n") + "\n"
	if got != want {
		t.Fatalf("got:\n%swant:\n%s", got, want)
	}
}

func TestUnparseSortsClassMembers(t *testing.T) {
	src := strings.Join([]string{
		"class Sample",
		"  private helper: -> 1",
		"",
		"  zed: 0",
		"",
		"  static make: -> Self()",
		"",
		"  # public constant",
		"  ALPHA: 1",
		"",
		"  private static hidden_count: 0",
		"",
		"  beta: -> 2",
		"",
		"  static build: -> Self()",
		"",
		"  protected static label: -> \"label\"",
		"",
		"  private id: 0",
		"",
		"  # private static method",
		"  private static normalize: -> 3",
		"",
		"  private SECRET: 9",
		"",
		"  ready: false",
		"",
		"  initialize: ->",
		"    self.ready = true",
		"",
		"  name: \"\"",
		"",
		"  ZETA: 2",
		"",
		"  static alpha_count: 0",
		"",
		"  zeta: -> 4",
		"",
		"  protected normalize: value -> value",
		"",
	}, "\n")
	want := strings.Join([]string{
		"class Sample",
		"  # public constant",
		"  ALPHA: 1",
		"",
		"  ZETA: 2",
		"",
		"  private SECRET: 9",
		"",
		"  static alpha_count: 0",
		"",
		"  private static hidden_count: 0",
		"",
		"  name: \"\"",
		"",
		"  ready: false",
		"",
		"  zed: 0",
		"",
		"  private id: 0",
		"",
		"  static build: -> Self()",
		"",
		"  static make: -> Self()",
		"",
		"  protected static label: -> \"label\"",
		"",
		"  # private static method",
		"  private static normalize: -> 3",
		"",
		"  initialize: ->",
		"    self.ready = true",
		"",
		"  beta: -> 2",
		"",
		"  zeta: -> 4",
		"",
		"  protected normalize: value -> value",
		"",
		"  private helper: -> 1",
		"",
	}, "\n")
	got, err := unparseSourceWithComments(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("class member order mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
	again, err := unparseSourceWithComments(t, got)
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Fatalf("class member order is not idempotent\nfirst:\n%s\nsecond:\n%s", got, again)
	}
}

func TestUnparseBareInstanceFieldReads(t *testing.T) {
	src := strings.Join([]string{
		"class Command",
		"  arguments: []",
		"  count: 0",
		"",
		"  argument: spec ->",
		"    self.arguments.push(spec)",
		"    self.count",
		"",
		"  replace_arguments: arguments ->",
		"    self.arguments = arguments",
		"    self.arguments",
		"",
		"  assign_local: ->",
		"    count = 1",
		"    self.count",
		"",
	}, "\n")
	want := strings.Join([]string{
		"class Command",
		"  arguments: []",
		"",
		"  count: 0",
		"",
		"  argument: spec ->",
		"    arguments.push(spec)",
		"    count",
		"",
		"  assign_local: ->",
		"    count = 1",
		"    self.count",
		"",
		"  replace_arguments: arguments ->",
		"    self.arguments = arguments",
		"    self.arguments",
		"",
	}, "\n")
	got, err := unparseSourceWithComments(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got:\n%swant:\n%s", got, want)
	}
}

func TestUnparseInterfaceMethodComments(t *testing.T) {
	src := strings.Join([]string{
		"# Iterator docs.",
		"interface Iterator",
		"  # has_next docs.",
		"  # @return Boolean whether more values exist.",
		"  has_next: ->",
		"  # next docs.",
		"  # @return Any next value.",
		"  next: ->",
		"",
	}, "\n")
	got, err := unparseSourceWithComments(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"# Iterator docs.",
		"interface Iterator",
		"  # has_next docs.",
		"  # @return Boolean whether more values exist.",
		"  has_next: ->",
		"  # next docs.",
		"  # @return Any next value.",
		"  next: ->",
		"",
	}, "\n")
	if got != want {
		t.Fatalf("interface method comments were not preserved\nwant:\n%s\ngot:\n%s", want, got)
	}
	again, err := unparseSourceWithComments(t, got)
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Fatalf("interface method comment formatting is not idempotent\nfirst:\n%s\nsecond:\n%s", got, again)
	}
}

func TestUnparseMatch(t *testing.T) {
	src := "match x\n  case 1\n    print(\"one\")\n  case _\n    print(\"other\")\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"match x", "  case 1", "    print(\"one\")", "  case _"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}
