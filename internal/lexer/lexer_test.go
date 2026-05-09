package lexer

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"tya/internal/token"
)

func TestLexIndentAndDedent(t *testing.T) {
	toks, errs := Lex("user =\n  name: \"komagata\"\nprint \"ok\"\n")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	var got []string
	for _, tok := range toks {
		got = append(got, string(tok.Type))
	}
	wantHas := map[string]bool{"INDENT": true, "DEDENT": true}
	gotHas := map[string]bool{"INDENT": false, "DEDENT": false}
	for _, typ := range got {
		if _, ok := gotHas[typ]; ok {
			gotHas[typ] = true
		}
	}
	if diff := cmp.Diff(wantHas, gotHas); diff != "" {
		t.Fatalf("indent token presence mismatch (-want +got):\n%s\nall tokens: %#v", diff, got)
	}
}

func TestLexV01TokenSequence(t *testing.T) {
	toks, errs := Lex("if score >= 90\n  print \"A\"\nelseif score >= 80\n  print \"B\"\n")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	got := tokenTypes(toks)
	want := []string{
		"IDENT", "IDENT", ">=", "INT", "NEWLINE",
		"INDENT", "IDENT", "STRING", "NEWLINE",
		"DEDENT", "IDENT", "IDENT", ">=", "INT", "NEWLINE",
		"INDENT", "IDENT", "STRING", "NEWLINE",
		"DEDENT", "EOF",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("token sequence mismatch (-want +got):\n%s", diff)
	}
}

func TestLexV01OperatorsAndDelimiters(t *testing.T) {
	toks, errs := Lex("= == != < <= > >= : , . + - * / % -> ( ) [ ] { }\n")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	got := tokenTypes(toks)
	want := []string{
		"=", "==", "!=", "<", "<=", ">", ">=", ":", ",", ".", "+", "-", "*", "/", "%", "->",
		"(", ")", "[", "]", "{", "}", "NEWLINE", "EOF",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("token sequence mismatch (-want +got):\n%s", diff)
	}
}

func TestLexCommentAfterCodeAllowsSpaceBeforeComment(t *testing.T) {
	toks, errs := Lex("name = 1 # comment\n")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	got := tokenTypes(toks)
	want := []string{"IDENT", "=", "INT", "NEWLINE", "EOF"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("token sequence mismatch (-want +got):\n%s", diff)
	}
}

func TestLexV01ModuleAndImportProgram(t *testing.T) {
	toks, errs := Lex("import greeting\n\nmodule greeting\n  hello = name -> \"Hello, {name}\"\n\nprint greeting.hello(\"komagata\")\n")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	got := tokenTypes(toks)
	want := []string{
		"IDENT", "IDENT", "NEWLINE",
		"IDENT", "IDENT", "NEWLINE",
		"INDENT", "IDENT", "=", "IDENT", "->", "STRING", "NEWLINE",
		"DEDENT", "IDENT", "IDENT", ".", "IDENT", "(", "STRING", ")", "NEWLINE",
		"EOF",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("token sequence mismatch (-want +got):\n%s", diff)
	}
}

func TestLexV01CollectionsAndIndexing(t *testing.T) {
	toks, errs := Lex("items = [1, 2, 3]\nuser = { name: \"komagata\", age: 20 }\nitems[0] = user[\"age\"]\n")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	got := tokenTypes(toks)
	want := []string{
		"IDENT", "=", "[", "INT", ",", "INT", ",", "INT", "]", "NEWLINE",
		"IDENT", "=", "{", "IDENT", ":", "STRING", ",", "IDENT", ":", "INT", "}", "NEWLINE",
		"IDENT", "[", "INT", "]", "=", "IDENT", "[", "STRING", "]", "NEWLINE",
		"EOF",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("token sequence mismatch (-want +got):\n%s", diff)
	}
}

func TestLexV01FunctionsReturnTryAndLoops(t *testing.T) {
	src := strings.Join([]string{
		"parse_user = text ->",
		"  if text == \"\"",
		"    return nil, error \"empty\"",
		"  return { name: text }, nil",
		"",
		"load_user = text ->",
		"  user = try parse_user(text)",
		"  user[\"name\"]",
		"",
		"for item, index in items",
		"  if index % 2 == 0 and not item == nil",
		"    continue",
		"  else",
		"    break",
		"",
		"for key, value of user",
		"  print \"{key}: {value}\"",
		"",
		"while count < 3",
		"  count = count + 1",
	}, "\n") + "\n"
	toks, errs := Lex(src)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	got := tokenTypes(toks)
	want := []string{
		"IDENT", "=", "IDENT", "->", "NEWLINE",
		"INDENT", "IDENT", "IDENT", "==", "STRING", "NEWLINE",
		"INDENT", "IDENT", "IDENT", ",", "IDENT", "STRING", "NEWLINE",
		"DEDENT", "IDENT", "{", "IDENT", ":", "IDENT", "}", ",", "IDENT", "NEWLINE",
		"DEDENT", "IDENT", "=", "IDENT", "->", "NEWLINE",
		"INDENT", "IDENT", "=", "IDENT", "IDENT", "(", "IDENT", ")", "NEWLINE",
		"IDENT", "[", "STRING", "]", "NEWLINE",
		"DEDENT", "IDENT", "IDENT", ",", "IDENT", "IDENT", "IDENT", "NEWLINE",
		"INDENT", "IDENT", "IDENT", "%", "INT", "==", "INT", "IDENT", "IDENT", "IDENT", "==", "IDENT", "NEWLINE",
		"INDENT", "IDENT", "NEWLINE",
		"DEDENT", "IDENT", "NEWLINE",
		"INDENT", "IDENT", "NEWLINE",
		"DEDENT", "DEDENT", "IDENT", "IDENT", ",", "IDENT", "IDENT", "IDENT", "NEWLINE",
		"INDENT", "IDENT", "STRING", "NEWLINE",
		"DEDENT", "IDENT", "IDENT", "<", "INT", "NEWLINE",
		"INDENT", "IDENT", "=", "IDENT", "+", "INT", "NEWLINE",
		"DEDENT", "EOF",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("token sequence mismatch (-want +got):\n%s", diff)
	}
}

func TestLexV01BlankLinesCommentsCRLFAndLocations(t *testing.T) {
	src := "# heading\r\nmodule util\r\n\r\n  # member\r\n  name = \"Tya\" # inline\r\n"
	toks, errs := Lex(src)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	got := tokenTypes(toks)
	want := []string{
		"IDENT", "IDENT", "NEWLINE",
		"INDENT", "IDENT", "=", "STRING", "NEWLINE",
		"DEDENT", "EOF",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("token sequence mismatch (-want +got):\n%s", diff)
	}
	name := toks[4]
	if name.Lexeme != "name" || name.Line != 5 || name.Col != 3 {
		t.Fatalf("got token %+v, want name at 5:3", name)
	}
}

func TestLexRejectsTabs(t *testing.T) {
	_, errs := Lex("user =\n\tname: \"komagata\"\n")
	if len(errs) == 0 {
		t.Fatal("expected tab error")
	}
}

func TestLexStringEscapes(t *testing.T) {
	toks, errs := Lex("print \"a\\n\\\"b\\\"\"\n")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if toks[1].Lexeme != "a\n\"b\"" {
		t.Fatalf("got %q", toks[1].Lexeme)
	}
}

func TestLexStringHashAfterEscapedQuote(t *testing.T) {
	toks, errs := Lex("print \"a \\\" # not comment\"\n")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if toks[1].Lexeme != "a \" # not comment" {
		t.Fatalf("got %q", toks[1].Lexeme)
	}
}

func TestLexTripleQuoteSingleLine(t *testing.T) {
	toks, errs := Lex("x = \"\"\"hello\"\"\"\n")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if toks[2].Type != "STRING" || toks[2].Lexeme != "hello" {
		t.Fatalf("got %v %q", toks[2].Type, toks[2].Lexeme)
	}
}

func TestLexTripleQuoteMultiLineNormalized(t *testing.T) {
	src := "x = \"\"\"\n  hello\n  Tya\n  \"\"\"\n"
	toks, errs := Lex(src)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if toks[2].Type != "STRING" || toks[2].Lexeme != "hello\nTya\n" {
		t.Fatalf("got %v %q", toks[2].Type, toks[2].Lexeme)
	}
}

func TestLexTripleQuoteInterpolationPreserved(t *testing.T) {
	src := "x = \"\"\"\n  hi {name}\n  \"\"\"\n"
	toks, errs := Lex(src)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if toks[2].Lexeme != "hi {name}\n" {
		t.Fatalf("got %q", toks[2].Lexeme)
	}
}

func TestLexRawSingleLine(t *testing.T) {
	src := "x = r\"\\d+ {name}\"\n"
	toks, errs := Lex(src)
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	if toks[2].Type != "STRING" {
		t.Fatalf("kind: %v", toks[2].Type)
	}
	// `{` and `}` are doubled so the runtime decodes them as
	// literal braces.
	if toks[2].Lexeme != "\\d+ {{name}}" {
		t.Fatalf("got %q", toks[2].Lexeme)
	}
}

func TestLexRawTriple(t *testing.T) {
	src := "x = r\"\"\"\n  hi {name}\n  \"\"\"\n"
	toks, errs := Lex(src)
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	if toks[2].Type != "STRING" {
		t.Fatalf("kind: %v", toks[2].Type)
	}
	if toks[2].Lexeme != "hi {{name}}\n" {
		t.Fatalf("got %q", toks[2].Lexeme)
	}
}

func TestLexBytesTriple(t *testing.T) {
	src := "x = b\"\"\"\n  AB\\n\n  \"\"\"\n"
	toks, errs := Lex(src)
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	if toks[2].Type != "BYTES" {
		t.Fatalf("kind: %v", toks[2].Type)
	}
	// Body line "AB\n" with escape interpretation: A B \n
	// (3 bytes). Indentation-normalization strips the 2-space
	// baseline, then a `\n` line-separator is appended after the
	// body line.
	if toks[2].Lexeme != "AB\n\n" {
		t.Fatalf("got %q", toks[2].Lexeme)
	}
}

func TestLexTripleQuoteUnterminated(t *testing.T) {
	_, errs := Lex("x = \"\"\"\n  hello\n")
	if len(errs) == 0 {
		t.Fatal("expected unterminated error")
	}
}

func TestLexTripleQuoteMixedIndent(t *testing.T) {
	_, errs := Lex("x = \"\"\"\n  hello\n there\n  \"\"\"\n")
	if len(errs) == 0 {
		t.Fatal("expected mixed-indent error")
	}
}

func TestLexPredicateQuestionToken(t *testing.T) {
	toks, errs := Lex("empty? = -> true\n")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	got := tokenTypes(toks)
	want := []string{"IDENT", "?", "=", "->", "IDENT", "NEWLINE", "EOF"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("token sequence mismatch (-want +got):\n%s", diff)
	}
}

func TestLexAtTokens(t *testing.T) {
	toks, errs := Lex("@name = 1\n@@count = 2\n")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	got := tokenTypes(toks)
	want := []string{"@", "IDENT", "=", "INT", "NEWLINE", "@", "@", "IDENT", "=", "INT", "NEWLINE", "EOF"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatal(diff)
	}
}

func TestLexFloatRequiresFractionDigits(t *testing.T) {
	toks, errs := Lex("value = 1.\n")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	got := tokenTypes(toks)
	want := []string{"IDENT", "=", "INT", ".", "NEWLINE", "EOF"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("token sequence mismatch (-want +got):\n%s", diff)
	}
}

func tokenTypes(toks []token.Token) []string {
	types := make([]string, len(toks))
	for i, tok := range toks {
		types[i] = string(tok.Type)
	}
	return types
}
