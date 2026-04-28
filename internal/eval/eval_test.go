package eval

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"tya/internal/lexer"
	"tya/internal/parser"
)

func TestRunArithmeticAndLiterals(t *testing.T) {
	src := "add = a, b -> a + b\nprint add 2, 3\nprint 2 + 3 * 4\ngrouped = (2 + 3) * 4\nprint grouped\nprint 5 / 2\nprint div 5, 2\nnegative = -5 + 2\nprint negative\nprint true\nprint nil\nage = 20\nprint \"next year: {age + 1}\"\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "5\n14\n20\n2.5\n2\n-3\ntrue\nnil\nnext year: 21\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunIfElseTruthiness(t *testing.T) {
	src := "if nil\n  print \"bad\"\nelse\n  print \"nil\"\n\nif 0\n  print \"zero\"\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "nil\nzero\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunComparisonAndLogic(t *testing.T) {
	src := "age = 20\nname = \"komagata\"\nif age >= 20 and name == \"komagata\"\n  print \"match\"\nprint nil or \"anonymous\"\nprint not false\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "match\nanonymous\ntrue\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunArrays(t *testing.T) {
	src := "items = [1, 2, 3]\nprint len items\nprint items[0]\nprint items[9]\npush items, 4\nprint len items\nprint pop items\nprint len items\nitems[1] = 20\nprint items[1]\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "3\n1\nnil\n4\n4\n3\n20\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunArrayFunctionBuiltins(t *testing.T) {
	src := "items = [1, 2, 3, 4]\ndouble = item -> item * 2\nisEven = item -> item % 2 == 0\nadd = total, item -> total + item\ndoubled = map items, double\nevens = filter items, isEven\nfirstEven = find items, isEven\nhasEven = any items, isEven\nallEven = all items, isEven\nsum = reduce items, 0, add\nprint doubled[2]\nprint len evens\nprint firstEven\nprint hasEven\nprint allEven\nprint sum\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "6\n2\n2\ntrue\nfalse\n10\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunWhile(t *testing.T) {
	src := "i = 0\nsum = 0\nwhile i < 5\n  i = i + 1\n  if i == 3\n    continue\n  sum = sum + i\n  if sum > 7\n    break\nprint sum\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "12\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunRejectsBreakOutsideLoop(t *testing.T) {
	toks, errs := lexer.Lex("break\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunReturn(t *testing.T) {
	src := "findFirstOver = limit ->\n  i = 0\n  while true\n    if i > limit\n      return i\n    i = i + 1\nprint findFirstOver 3\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "4\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunConversions(t *testing.T) {
	src := "print toString 20\nprint toInt \"42\"\nprint toFloat \"2.5\"\nprint toNumber \"12\"\nprint toNumber \"12.5\"\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "20\n42\n2.5\n12\n12.5\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunStringBuiltins(t *testing.T) {
	src := "text = \"  hello,tya  \"\ntrimmed = trim text\nparts = split trimmed, \",\"\nprint join parts, \"-\"\nprint replace trimmed, \"tya\", \"Tya\"\nprint contains trimmed, \"hello\"\nprint startsWith trimmed, \"hello\"\nprint endsWith trimmed, \"tya\"\nprint byteLen \"ちゃ\"\nprint charLen \"ちゃ\"\nprint \"quote: \\\"tya\\\"\"\nprint \"tya\"[1]\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "hello-tya\nhello,Tya\ntrue\ntrue\ntrue\n6\n2\nquote: \"tya\"\ny\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunObjectBuiltins(t *testing.T) {
	src := "user =\n  name: \"komagata\"\n  age: 20\n\nprint has user, \"name\"\nprint len keys user\nprint len values user\ndelete user, \"age\"\nprint has user, \"age\"\nprint user.age\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "true\n2\n2\nfalse\nnil\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunInlineObject(t *testing.T) {
	src := "user = { name: \"komagata\", age: 20 }\nprint \"Hello, {user.name}\"\nprint user.age\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "Hello, komagata\n20\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunDeepEqualBuiltin(t *testing.T) {
	src := "left =\n  name: \"komagata\"\n  nums: [1, 2]\nright =\n  name: \"komagata\"\n  nums: [1, 2]\nprint left == right\nsame = equal left, right\nnumsA = [1, 2]\nnumsB = [1, 3]\ndifferent = equal numsA, numsB\nprint same\nprint different\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "false\ntrue\nfalse\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunForIn(t *testing.T) {
	src := "items = [2, 4, 6]\nsum = 0\nfor item in items\n  sum = sum + item\nprint sum\nfor item, index in items\n  print \"{index}:{item}\"\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "12\n0:2\n1:4\n2:6\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunForOfObject(t *testing.T) {
	src := "user =\n  name: \"komagata\"\n  age: 20\ncount = 0\nfor key, value of user\n  if key == \"name\"\n    print value\n  count = count + 1\nprint count\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "komagata\n2\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunFileBuiltins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memo.txt")
	src := "writeFile \"" + path + "\", \"hello\"\nprint fileExists \"" + path + "\"\nprint readFile \"" + path + "\"\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "true\nhello\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunArgsAndEnvBuiltins(t *testing.T) {
	t.Setenv("TYA_TEST_ENV", "ok")
	src := "items = args()\nprint len items\nprint items[0]\nprint env \"TYA_TEST_ENV\"\nprint env \"TYA_MISSING_ENV\"\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := RunWithArgs(prog, &out, []string{"one", "two"}); err != nil {
		t.Fatal(err)
	}
	want := "2\none\nok\nnil\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunReadLineBuiltin(t *testing.T) {
	toks, errs := lexer.Lex("name = readLine()\nprint \"Hello, {name}\"\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := RunWithIO(prog, strings.NewReader("komagata\n"), &out, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Hello, komagata\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunExitBuiltin(t *testing.T) {
	toks, errs := lexer.Lex("exit 7\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = Run(prog, &out)
	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("got %T %v", err, err)
	}
	if exitErr.Code != 7 {
		t.Fatalf("got exit code %d", exitErr.Code)
	}
}

func TestRunPanicBuiltin(t *testing.T) {
	toks, errs := lexer.Lex("panic \"bad\"\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err == nil {
		t.Fatal("expected panic error")
	}
}

func TestRunErrorBuiltin(t *testing.T) {
	toks, errs := lexer.Lex("err = error \"file not found\"\nprint err\nprint err.message\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "error: file not found\nfile not found\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunMultipleReturnAndAssignment(t *testing.T) {
	src := "parseUser = text ->\n  if text == \"\"\n    return nil, error \"empty user\"\n  return { name: text }, nil\nuser, err = parseUser \"komagata\"\nif err\n  print err.message\nelse\n  print user.name\nmissing, err = parseUser \"\"\nif err\n  print err.message\nelse\n  print missing.name\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "komagata\nempty user\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunTryPropagation(t *testing.T) {
	src := "parseUser = text ->\n  if text == \"\"\n    return nil, error \"empty user\"\n  return { name: text }, nil\nreadUser = text ->\n  user = try parseUser(text)\n  return user.name, nil\nname, err = readUser \"komagata\"\nif err\n  print err.message\nelse\n  print name\nname, err = readUser \"\"\nif err\n  print err.message\nelse\n  print name\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "komagata\nempty user\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunRejectsReturnOutsideFunction(t *testing.T) {
	toks, errs := lexer.Lex("return 1\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err == nil {
		t.Fatal("expected error")
	}
}
