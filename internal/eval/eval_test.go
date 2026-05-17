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
	src := "add = a, b -> a + b\nprint(add(2, 3))\nprint(2 + 3 * 4)\ngrouped = (2 + 3) * 4\nprint(grouped)\nprint(5 / 2)\nnegative = -5 + 2\nprint(negative)\nprint(true)\nprint(nil)\nage = 20\nprint(\"next year: {age + 1}\")\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "5\n14\n20\n2.5\n-3\ntrue\nnil\nnext year: 21\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunStrictStringPlus(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "string string", src: "print(\"Ty\" + \"a\")\n", want: "Tya\n"},
		{name: "number number", src: "print(2 + 3)\n", want: "5\n"},
		{name: "bytes bytes", src: "print(b\"Ty\" + b\"a\")\n", want: "&{[84 121 97]}\n"},
		{name: "interpolation formats number", src: "count = 3\nprint(\"count: {count}\")\n", want: "count: 3\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toks, errs := lexer.Lex(tt.src)
			if len(errs) != 0 {
				t.Fatalf("lex errors: %v", errs)
			}
			prog, _, err := parser.Parse(toks)
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := Run(prog, &out); err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Fatalf("got %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestRunRejectsMixedStringPlus(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{name: "string number", src: "count = 3\nprint(\"count: \" + count)\n"},
		{name: "number string", src: "count = 3\nprint(count + \" items\")\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toks, errs := lexer.Lex(tt.src)
			if len(errs) != 0 {
				t.Fatalf("lex errors: %v", errs)
			}
			prog, _, err := parser.Parse(toks)
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			err = Run(prog, &out)
			if err == nil {
				t.Fatal("expected mixed string plus error")
			}
			if !strings.Contains(err.Error(), "+ expects numbers, strings, or bytes of the same kind") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRunIfElseTruthiness(t *testing.T) {
	src := "if nil\n  print(\"bad\")\nelse\n  print(\"nil\")\n\nif 0\n  print(\"zero\")\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
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
	src := "age = 20\nname = \"komagata\"\nif age >= 20 and name == \"komagata\"\n  print(\"match\")\nprint(nil or \"anonymous\")\nprint(\"fallback\" or false)\nprint(not false)\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "match\ntrue\ntrue\ntrue\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunArrays(t *testing.T) {
	src := "items = [1, 2, 3]\nprint(items.len())\nprint(items[0])\nprint(items[9])\nitems.push(4)\nprint(items.len())\nprint(items.pop())\nprint(items.len())\nitems[1] = 20\nprint(items[1])\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
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

func TestRunWhile(t *testing.T) {
	src := "i = 0\nsum = 0\nwhile i < 5\n  i = i + 1\n  if i == 3\n    continue\n  sum = sum + i\n  if sum > 7\n    break\nprint(sum)\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
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
	if _, _, err := parser.Parse(toks); err == nil {
		t.Fatal("expected parser error")
	}
}

func TestRunReturn(t *testing.T) {
	src := "find_first_over = limit ->\n  i = 0\n  while true\n    if i > limit\n      return i\n    i = i + 1\nprint(find_first_over(3))\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
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
	src := "print(20.to_s())\nprint(\"42\".to_i())\nprint(\"2.5\".to_f())\nprint(\"12\".to_number())\nprint(\"12.5\".to_number())\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
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
	src := "text = \"  hello,tya  \"\ntrimmed = text.trim()\nparts = trimmed.split(\",\")\nprint(parts.join(\"-\"))\nprint(trimmed.replace(\"tya\", \"Tya\"))\nprint(trimmed.contains(\"hello\"))\nprint(trimmed.starts_with(\"hello\"))\nprint(trimmed.ends_with(\"tya\"))\nprint(\"quote: \\\"tya\\\"\")\nprint(\"tya\"[1])\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "hello-tya\nhello,Tya\ntrue\ntrue\ntrue\nquote: \"tya\"\ny\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunDictBuiltins(t *testing.T) {
	src := "user =\n  name: \"komagata\"\n  age: 20\n\nprint(user.has(\"name\"))\nprint(user.keys().len())\nprint(user.values().len())\nuser.delete(\"age\")\nprint(user.has(\"age\"))\nprint(user[\"age\"])\nuser[\"city\"] = \"Tokyo\"\nprint(user[\"city\"])\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "true\n2\n2\nfalse\nnil\nTokyo\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunInlineDict(t *testing.T) {
	src := "user = { name: \"komagata\", age: 20 }\nprint(\"Hello, \" + user[\"name\"])\nprint(user[\"age\"])\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
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

func TestRunNamespaceDictionary(t *testing.T) {
	src := "foo = \"foo\"\nbar = ->\n  \"bar\"\nutil = {}\nutil[\"foo\"] = foo\nutil[\"bar\"] = bar\nprint(util.foo)\nprint(util.bar())\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "foo\nbar\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunDictionaryEquality(t *testing.T) {
	src := "left =\n  name: \"komagata\"\n  nums: [1, 2]\nright =\n  name: \"komagata\"\n  nums: [1, 2]\nprint(left == right)\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "true\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunBytesOutOfRangeIndexReturnsNil(t *testing.T) {
	src := "data = b\"abc\"\nprint(data[99])\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err != nil {
		t.Fatal(err)
	}
	want := "nil\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunForIn(t *testing.T) {
	src := "items = [2, 4, 6]\nsum = 0\nfor item in items\n  sum = sum + item\nprint(sum)\nfor item, index in items\n  print(\"{index}:{item}\")\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
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

func TestRunForInDictEntries(t *testing.T) {
	src := "user =\n  name: \"komagata\"\n  age: 20\ncount = 0\nfor entry in user\n  if entry[\"key\"] == \"name\"\n    print(entry[\"value\"])\n  count = count + 1\nprint(count)\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
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
	src := "write_file(\"" + path + "\", \"hello\")\nprint(file_exists(\"" + path + "\"))\nprint(read_file(\"" + path + "\"))\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
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
	src := "items = args()\nprint(items.len())\nprint(items[0])\nprint(env(\"TYA_TEST_ENV\"))\nprint(env(\"TYA_MISSING_ENV\"))\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
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

func TestRunExitBuiltin(t *testing.T) {
	toks, errs := lexer.Lex("exit(7)\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
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
		t.Fatalf("got exit(code) %d", exitErr.Code)
	}
}

func TestRunPanicBuiltin(t *testing.T) {
	toks, errs := lexer.Lex("panic(\"bad\")\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, &out); err == nil {
		t.Fatal("expected panic error")
	}
}

func TestRunErrorBuiltin(t *testing.T) {
	toks, errs := lexer.Lex("err = error(\"file not found\")\nprint(err)\nprint(err[\"message\"])\n")
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
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
	src := "parse_user = text ->\n  if text == \"\"\n    return nil, error(\"empty user\")\n  return { name: text }, nil\nuser, err = parse_user(\"komagata\")\nif err\n  print(err[\"message\"])\nelse\n  print(user[\"name\"])\nmissing, err = parse_user(\"\")\nif err\n  print(err[\"message\"])\nelse\n  print(missing[\"name\"])\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
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
	src := "parse_user = text ->\n  if text == \"\"\n    return nil, error(\"empty user\")\n  return { name: text }, nil\nread_user = text ->\n  user = try parse_user(text)\n  return user[\"name\"], nil\nname, err = read_user(\"komagata\")\nif err\n  print(err[\"message\"])\nelse\n  print(name)\nname, err = read_user(\"\")\nif err\n  print(err[\"message\"])\nelse\n  print(name)\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
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
	if _, _, err := parser.Parse(toks); err == nil {
		t.Fatal("expected parser error")
	}
}

func TestRunRejectsStrictDynamicErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "nil arithmetic", src: "print(nil + 1)\n", want: "+ expects numbers, strings, or bytes of the same kind"},
		{name: "nil ordering", src: "print(nil < 1)\n", want: "< expects numbers"},
		{name: "nil indexing", src: "print(nil[0])\n", want: "index target is not array or string"},
		{name: "wrong array index", src: "print([1][\"0\"])\n", want: "array index must be int"},
		{name: "wrong dict index", src: "print({ name: \"tya\" }[0])\n", want: "dictionary index must be string"},
		{name: "nil call", src: "fn = nil\nprint(fn())\n", want: "value is not callable"},
		{name: "nil member", src: "print(nil.name)\n", want: "cannot read property name on non-dictionary"},
		{name: "assignment arity", src: "first, second = [1]\n", want: "assignment expects 2 values, got 1"},
		{name: "function arity", src: "add = a, b -> a + b\nprint(add(1))\n", want: "function expects 2 arguments, got 1"},
		{name: "kind changing reassignment", src: "value = 1\nvalue = \"one\"\n", want: "cannot reassign value from number to string"},
		{name: "raise nil", src: "raise nil\n", want: "raise expects non-nil value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toks, errs := lexer.Lex(tt.src)
			if len(errs) != 0 {
				t.Fatalf("lex errors: %v", errs)
			}
			prog, _, err := parser.Parse(toks)
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			err = Run(prog, &out)
			if err == nil {
				t.Fatal("expected runtime error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got error %q, want %q", err.Error(), tt.want)
			}
		})
	}
}
