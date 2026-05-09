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
	prog, err := parser.ParseWithComments(toks, comments)
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

func unparseSource(t *testing.T, src string) (string, error) {
	t.Helper()
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errs: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	return Unparse(prog)
}

func TestUnparseAssignAndPrint(t *testing.T) {
	src := "x = 1\nprint x\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := "x = 1\nprint x\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestUnparseBinaryAndCall(t *testing.T) {
	src := "y = 2 + 3 * 4\nprint y\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "y = 2 + 3 * 4") {
		t.Errorf("got: %q", got)
	}
}

func TestUnparseIfElseifElse(t *testing.T) {
	src := "x = 1\nif x == 0\n  print \"a\"\nelseif x == 1\n  print \"b\"\nelse\n  print \"c\"\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"if x == 0",
		"  print \"a\"",
		"elseif x == 1",
		"  print \"b\"",
		"else",
		"  print \"c\"",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestUnparseLambdaSingleLine(t *testing.T) {
	src := "add = a, b -> a + b\nprint add(2, 3)\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "add = a, b -> a + b") {
		t.Errorf("got: %q", got)
	}
}

func TestUnparseLambdaBlock(t *testing.T) {
	src := "f = x ->\n  y = x + 1\n  return y\nprint f(2)\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"f = x ->", "  y = x + 1", "  return y", "print f(2)"} {
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
	if !strings.Contains(got, "import string") {
		t.Errorf("got: %q", got)
	}
	if !strings.Contains(got, "import file as f") {
		t.Errorf("got: %q", got)
	}
}

func TestUnparseIdempotent(t *testing.T) {
	cases := []string{
		"x = 1\n",
		"x = 1\nprint x\n",
		"x = 1\ny = x + 2 * 3\nprint y\n",
		"add = (a, b) -> a + b\nprint add(2, 3)\n",
		"f = x ->\n  y = x + 1\n  return y\nprint f(2)\n",
		"if x == 0\n  print \"a\"\nelseif x == 1\n  print \"b\"\nelse\n  print \"c\"\n",
		"items = [1, 2, 3]\nfor item in items\n  print item\n",
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
	want := "result = compute_filtered_items(\n  source_alpha,\n  source_beta,\n  source_gamma,\n  source_delta,\n)\n"
	if got != want {
		t.Errorf("wrap mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestUnparseWrapsLongArray(t *testing.T) {
	src := "items = [first_item_name, second_item_name, third_item_name, fourth_item_name, fifth_item_name]\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"items = [\n  first_item_name,\n", "  fifth_item_name,\n]\n"} {
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
	want := "import file\nimport string\n\nimport mylib\nimport zmod\n\ngreet = name -> name\nx = 1\n"
	if got != want {
		t.Errorf("import sort/blank-line layout mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestUnparseModule(t *testing.T) {
	src := "module greet\n  hello = name -> \"Hello, \" + name\n  bye = -> \"bye\"\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"module greet",
		"  hello = name -> \"Hello, \" + name",
		"  bye = () -> \"bye\"",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestUnparseClass(t *testing.T) {
	src := "class Dog\n  bark = ->\n    return \"woof\"\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"class Dog", "  bark = () ->", "    return \"woof\""} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestUnparseMatch(t *testing.T) {
	src := "match x\n  case 1\n    print \"one\"\n  case _\n    print \"other\"\n"
	got, err := unparseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"match x", "  case 1", "    print \"one\"", "  case _"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}
