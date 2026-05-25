package eval

import (
	"bytes"
	"os"
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
		{name: "bytes bytes", src: "print(b\"Ty\" + b\"a\")\n", want: "<bytes:3>\n"},
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

func TestRunDefaultParameters(t *testing.T) {
	src := strings.Join([]string{
		"greet = name, suffix = \"!\" -> name + suffix",
		"label = name, text = name -> text",
		"make = items = [] ->",
		"  items.push(1)",
		"  items.len()",
		"print(greet(\"Tya\"))",
		"print(greet(\"Tya\", \"?\"))",
		"print(label(\"copy\"))",
		"maybe = value = \"fallback\" -> value",
		"print(maybe())",
		"print(maybe(nil))",
		"print(make())",
		"print(make())",
		"",
	}, "\n")
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
	want := "Tya!\nTya?\ncopy\nfallback\nnil\n1\n1\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunTryCatchFinallyValue(t *testing.T) {
	src := strings.Join([]string{
		"try",
		"  print(\"try\")",
		"catch err",
		"  print(err)",
		"finally",
		"  print(\"finally\")",
		"try",
		"  raise error(\"boom\")",
		"catch err",
		"  print(err[\"message\"])",
		"finally",
		"  print(\"cleanup\")",
		"",
	}, "\n")
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
	want := "try\nfinally\nboom\ncleanup\n"
	if out.String() != want {
		t.Fatalf("got %q want %q", out.String(), want)
	}
}

func TestRunTryFinallyReraises(t *testing.T) {
	src := "try\n  raise error(\"boom\")\nfinally\n  print(\"cleanup\")\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = Run(prog, &out)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected boom raise, got %v", err)
	}
	if out.String() != "cleanup\n" {
		t.Fatalf("got %q want cleanup", out.String())
	}
}

func TestRunFinallyRunsBeforeReturn(t *testing.T) {
	src := strings.Join([]string{
		"f = ->",
		"  try",
		"    return \"value\"",
		"  finally",
		"    print(\"cleanup\")",
		"print(f())",
		"",
	}, "\n")
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
	if out.String() != "cleanup\nvalue\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunFinallyRunsBeforeBreakAndContinue(t *testing.T) {
	src := strings.Join([]string{
		"i = 0",
		"while i < 3",
		"  i = i + 1",
		"  try",
		"    if i == 1",
		"      continue",
		"    break",
		"  finally",
		"    print(i)",
		"",
	}, "\n")
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
	if out.String() != "1\n2\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunFinallyControlFlowOverridesPendingFlow(t *testing.T) {
	src := strings.Join([]string{
		"f = ->",
		"  try",
		"    return \"value\"",
		"  finally",
		"    raise error(\"override\")",
		"print(f())",
		"",
	}, "\n")
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = Run(prog, &out)
	if err == nil || !strings.Contains(err.Error(), "override") {
		t.Fatalf("expected override raise, got %v", err)
	}
}

func TestRunMethodReceiverEvaluatedOnce(t *testing.T) {
	src := strings.Join([]string{
		"state = { count: 0 }",
		"get_text = ->",
		"  state[\"count\"] = state[\"count\"] + 1",
		"  \"tya\"",
		"print(get_text().upper())",
		"print(state[\"count\"])",
		"",
	}, "\n")
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
	if out.String() != "TYA\n1\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunAssignmentTargetEvaluationOrder(t *testing.T) {
	src := strings.Join([]string{
		"events = []",
		"items = [0, 0]",
		"record = name, value ->",
		"  events.push(name)",
		"  value",
		"items[record(\"i\", 0)], items[record(\"j\", 1)] = record(\"a\", 10), record(\"b\", 20)",
		"print(events.join(\",\"))",
		"print(items[0])",
		"print(items[1])",
		"",
	}, "\n")
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
	if out.String() != "a,b,i,j\n10\n20\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunLogicalOperatorsReturnBoolAndShortCircuit(t *testing.T) {
	src := strings.Join([]string{
		"count = 0",
		"bump = ->",
		"  count = count + 1",
		"  true",
		"print(\"x\" and \"y\")",
		"print(nil or \"y\")",
		"print(false and bump())",
		"print(true or bump())",
		"print(count)",
		"",
	}, "\n")
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
	if out.String() != "true\ntrue\nfalse\ntrue\n0\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunNotReturnsBool(t *testing.T) {
	src := "print(not nil)\nprint(not false)\nprint(not true)\nprint(not 0)\nprint(not \"x\")\n"
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
	if out.String() != "true\ntrue\nfalse\nfalse\nfalse\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunFunctionDebugNameDisplay(t *testing.T) {
	src := "named = -> nil\nprint(named)\nprint(-> nil)\n"
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
	if out.String() != "<function named>\n<function>\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunStableRuntimeDisplays(t *testing.T) {
	src := strings.Join([]string{
		"class User",
		"  static name = -> \"User\"",
		"print(User)",
		"print(42.class)",
		"print(b\"abc\")",
		"",
	}, "\n")
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
	if out.String() != "<class User>\n<class Number>\n<bytes:3>\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunCyclicDisplayTerminates(t *testing.T) {
	src := "items = []\nitems.push(items)\nprint(items)\n"
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
	if !strings.Contains(out.String(), "<cycle>") {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunCyclicDeepEqualityErrors(t *testing.T) {
	src := "items = []\nitems.push(items)\nprint(items == items)\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = Run(prog, &out)
	if err == nil || !strings.Contains(err.Error(), "cyclic equality") {
		t.Fatalf("expected cyclic equality error, got %v", err)
	}
}

func TestRunUnknownMemberErrorIncludesReceiver(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{src: "print(1.missing())\n", want: "unknown method missing on number"},
		{src: "print(nil.name)\n", want: "unknown member name on nil"},
	}
	for _, tt := range tests {
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
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("got %v, want %q", err, tt.want)
		}
	}
}

func TestRunInvalidUtf8TextReadErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.txt")
	if err := os.WriteFile(path, []byte{0xff}, 0o644); err != nil {
		t.Fatal(err)
	}
	src := "print(read_file(\"" + strings.ReplaceAll(path, "\\", "\\\\") + "\"))\n"
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = Run(prog, &out)
	if err == nil || !strings.Contains(err.Error(), "invalid UTF-8") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBinaryReadReturnsBytesForInvalidUtf8(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.bin")
	if err := os.WriteFile(path, []byte{0xff}, 0o644); err != nil {
		t.Fatal(err)
	}
	src := "data = file_read_bytes(\"" + strings.ReplaceAll(path, "\\", "\\\\") + "\")\nprint(bytes_array(data)[0])\n"
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
	if out.String() != "255\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunNumericEdgeSemantics(t *testing.T) {
	src := "print(1 == 1.0)\nprint(5 / 2)\nprint(5 % 2)\n"
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
	want := "true\n2.5\n1\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunNegativeIndexesError(t *testing.T) {
	tests := []string{
		"i = -1\nprint([1][i])\n",
		"i = -1\nprint(\"abc\"[i])\n",
		"i = -1\nprint(b\"abc\"[i])\n",
	}
	for _, src := range tests {
		toks, errs := lexer.Lex(src)
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
			t.Fatalf("expected negative index error for %q", src)
		}
		if !strings.Contains(err.Error(), "negative indexes are invalid") {
			t.Fatalf("unexpected error: %v", err)
		}
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

func TestRunLogicShortCircuitsAndReturnsBool(t *testing.T) {
	src := "boom = ->\n  raise \"boom\"\nprint(false and boom())\nprint(true or boom())\nprint(\"value\" and \"right\")\nprint(nil or \"fallback\")\n"
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
	want := "false\ntrue\ntrue\ntrue\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunRejectsDictionaryKeyMemberAccess(t *testing.T) {
	src := "make_user = -> { name: \"komagata\" }\nuser = make_user()\nprint(user.name)\n"
	toks, errs := lexer.Lex(src)
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
		t.Fatal("expected dictionary member access error")
	}
	if !strings.Contains(err.Error(), "unknown member name on dictionary") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRejectsKindChangingReassignmentThroughNil(t *testing.T) {
	src := "value = 1\nvalue = nil\nvalue = \"one\"\n"
	toks, errs := lexer.Lex(src)
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
		t.Fatal("expected kind-changing reassignment error")
	}
	if !strings.Contains(err.Error(), "cannot reassign value from number to string") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBlockScopeAndOuterAssignment(t *testing.T) {
	src := "count = 1\nif true\n  count = 2\n  local = 9\nprint(count)\nshow = ->\n  if true\n    inner = 1\n  return inner\nprint(show())\n"
	toks, errs := lexer.Lex(src)
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
		t.Fatal("expected block local lookup error")
	}
	if out.String() != "2\n" {
		t.Fatalf("got output %q, want %q", out.String(), "2\n")
	}
	if !strings.Contains(err.Error(), "undefined variable inner") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunFieldAssignmentDoesNotOverwriteSameNamedMethod(t *testing.T) {
	src := "class Response\n  initialize = ->\n    self.status = 200\n    self.bump()\n\n  bump = ->\n    self.status = self.status + 1\n\n  status = ->\n    self.status\n\nresponse = Response()\nprint(response.status())\nresponse.bump()\nprint(response.status())\n"
	if got := runEval(t, src); got != "201\n202\n" {
		t.Fatalf("got %q", got)
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
	src := "text = \"  hello,tya  \"\ntrimmed = text.trim()\nparts = trimmed.split(\",\")\nprint(parts.join(\"-\"))\nprint(trimmed.replace(\"tya\", \"Tya\"))\nprint(trimmed.contains(\"hello\"))\nprint(trimmed.starts_with(\"hello\"))\nprint(trimmed.ends_with(\"tya\"))\nprint(trimmed.slice(1, 4))\nprint(\"あいう\".slice(1, 3))\nprint(\"quote: \\\"tya\\\"\")\nprint(\"tya\"[1])\n"
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
	want := "hello-tya\nhello,Tya\ntrue\ntrue\ntrue\nell\nいう\nquote: \"tya\"\ny\n"
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
	src := "foo = \"foo\"\nbar = ->\n  \"bar\"\nutil = {}\nutil[\"foo\"] = foo\nutil[\"bar\"] = bar\nprint(util[\"foo\"])\nprint(util[\"bar\"]())\n"
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
	toks, errs := lexer.Lex("err = error(\"file not found\")\nprint(err)\nprint(err[\"message\"])\nprint(err[\"kind\"])\nprint(err[\"code\"])\nprint(err[\"data\"].len())\nprint(err[\"cause\"])\ncause = error(\"root\")\ntagged = error(\"missing\", { kind: \"io\", code: \"file_not_found\", data: { path: \"missing.txt\" }, cause: cause })\nprint(tagged[\"kind\"])\nprint(tagged[\"code\"])\nprint(tagged[\"data\"][\"path\"])\nprint(tagged[\"cause\"][\"message\"])\n")
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
	want := "file not found\nfile not found\nerror\n\n0\nnil\nio\nfile_not_found\nmissing.txt\nroot\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunMultipleReturnAndAssignment(t *testing.T) {
	src := "bounds = items ->\n  return items[0], items[items.len() - 1]\nmin, max = bounds([1, 2, 3])\nprint(min)\nprint(max)\n"
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
	want := "1\n3\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunRaiseCatchStructuredError(t *testing.T) {
	src := "try\n  raise error(\"missing\", { kind: \"io\", code: \"file_not_found\", data: { path: \"missing.txt\" } })\ncatch err\n  print(err[\"message\"])\n  print(err[\"kind\"])\n  print(err[\"code\"])\n  print(err[\"data\"][\"path\"])\n"
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
	want := "missing\nio\nfile_not_found\nmissing.txt\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestRunDictionaryStringLiteralKeys(t *testing.T) {
	src := "headers = { \"Content-Type\": \"text/plain\", \"$schema\": \"x\", \"1\": \"one\", \"\": \"empty\" }\nprint(headers[\"Content-Type\"])\nprint(headers[\"$schema\"])\nprint(headers[\"1\"])\nprint(headers[\"\"])\nprint(headers[\"missing\"])\n"
	out := runEval(t, src)
	want := "text/plain\nx\none\nempty\nnil\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunCollectionMutationReturnValues(t *testing.T) {
	src := "items = []\nprint(items.push(1))\nprint(items.pop())\nprint(items.pop())\ndict = {}\nprint(dict.set(\"name\", \"Tya\"))\nprint(dict[\"name\"])\nprint(dict.delete(\"name\"))\nprint(dict[\"name\"])\n"
	out := runEval(t, src)
	want := "nil\n1\nnil\nnil\nTya\nnil\nnil\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunEmptyControlFlowValues(t *testing.T) {
	src := "empty_for = ->\n  for item in []\n    item\nunmatched_if = ->\n  if false\n    \"bad\"\nunmatched_match = ->\n  match \"x\"\n    case \"y\"\n      \"bad\"\nprint(empty_for())\nprint(unmatched_if())\nprint(unmatched_match())\n"
	out := runEval(t, src)
	want := "nil\nnil\nnil\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunBreakContinueLoopValues(t *testing.T) {
	src := "break_before = ->\n  for item in [1]\n    break\nbreak_after = ->\n  for item in [1, 2]\n    item\n    break\ncontinue_discards = ->\n  for item in [1, 2]\n    item\n    continue\nprint(break_before())\nprint(break_after())\nprint(continue_discards())\n"
	out := runEval(t, src)
	want := "nil\n1\nnil\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunAssignmentEvaluationOrderAndSwap(t *testing.T) {
	src := "items = []\nrecord = value ->\n  items.push(value)\n  return value\na = 1\nb = 2\na, b = record(b), record(a)\nprint(a)\nprint(b)\nprint(items.join(\",\"))\n"
	out := runEval(t, src)
	want := "2\n1\n2,1\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunStringIndexingUsesRunes(t *testing.T) {
	src := "print(\"abc\"[1])\nprint(\"あい\"[0])\nprint(\"あい\"[99])\n"
	out := runEval(t, src)
	want := "b\nあ\nnil\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunNumberKindAndDivisionBoundaries(t *testing.T) {
	src := "print(1 == 1.0)\nprint(5 / 2)\nprint(1.0 & 3)\n"
	out := runEval(t, src)
	want := "true\n2.5\n1\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}

	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "float modulo", src: "print(5.5 % 2)\n", want: "% expects integers"},
		{name: "float bitwise", src: "print(1.5 & 1)\n", want: "& expects integers"},
		{name: "float shift", src: "print(1 << 1.5)\n", want: "<< expects integers"},
		{name: "float index", src: "print([10][0.5])\n", want: "array index must be int"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runEvalError(t, tt.src)
			if err == nil {
				t.Fatal("expected runtime error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got error %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestRunStringBytesConversionBoundaries(t *testing.T) {
	src := "data = bytes([255, 65])\nprint(data)\nprint(bytes_array(data)[0])\n"
	out := runEval(t, src)
	want := "<bytes:2>\n255\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}

	err := runEvalError(t, "print(bytes_text(bytes([255])))\n")
	if err == nil {
		t.Fatal("expected invalid UTF-8 error")
	}
	if !strings.Contains(err.Error(), "invalid UTF-8") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunErrorValuesAreNotStrings(t *testing.T) {
	src := "err = error(\"failed\")\nprint(err)\nprint(err == \"failed\")\nprint(err[\"message\"])\n"
	out := runEval(t, src)
	want := "failed\nfalse\nfailed\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunCatchCatchesErrorValue(t *testing.T) {
	src := "raise_and_catch = value ->\n  try\n    raise value\n  catch err\n    return err\nprint(raise_and_catch(error(\"bad\"))[\"message\"])\n"
	out := runEval(t, src)
	want := "bad\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunCatchCatchesStructuredErrorsOnly(t *testing.T) {
	src := "try\n  raise error(\"bad\", { kind: \"test\", code: \"failed\" })\ncatch err\n  print(err[\"kind\"])\n  print(err[\"code\"])\n"
	out := runEval(t, src)
	want := "test\nfailed\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}

	tests := []string{
		"raise \"failed\"\n",
		"raise 1\n",
		"raise { message: \"failed\" }\n",
		"raise nil\n",
	}
	for _, src := range tests {
		t.Run(src, func(t *testing.T) {
			err := runEvalError(t, src)
			if err == nil {
				t.Fatal("expected non-error raise error")
			}
			if !strings.Contains(err.Error(), "raise expects error value") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRunResourceCleanupUsesFinally(t *testing.T) {
	src := "state = { closed: false }\ntry\n  raise error(\"boom\")\ncatch err\n  print(err[\"message\"])\nfinally\n  state[\"closed\"] = true\nprint(state[\"closed\"])\n"
	out := runEval(t, src)
	want := "boom\ntrue\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunOsEnvironmentMutation(t *testing.T) {
	key := "TYA_TEST_ENV_MUTATION"
	t.Setenv(key, "")
	src := "print(env(\"" + key + "\"))\nsetenv(\"" + key + "\", \"one\")\nprint(env(\"" + key + "\"))\nprint(environ()[\"" + key + "\"])\nunsetenv(\"" + key + "\")\nprint(env(\"" + key + "\"))\n"
	out := runEval(t, src)
	want := "\none\none\nnil\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunProcessRunDirectCommand(t *testing.T) {
	src := "r = process_run([\"sh\", \"-c\", \"printf out; printf err 1>&2; exit 7\"], {})\nprint(r[\"status\"])\nprint(r[\"exit_code\"])\nprint(r[\"success\"])\nprint(r[\"stdout\"])\nprint(r[\"stderr\"])\n"
	out := runEval(t, src)
	want := "7\n7\nfalse\nout\nerr\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunProcessRunShellOptIn(t *testing.T) {
	src := "r = process_run(\"printf shell\", { shell: true })\nprint(r[\"stdout\"])\n"
	out := runEval(t, src)
	if out != "shell\n" {
		t.Fatalf("got %q", out)
	}
	err := runEvalError(t, "process_run(\"printf no\", {})\n")
	if err == nil || !strings.Contains(err.Error(), "string command requires shell option") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunProcessRunCwdEnvStdinTimeout(t *testing.T) {
	dir := t.TempDir()
	src := "r = process_run([\"sh\", \"-c\", \"pwd; printf $TYA_CHILD; cat\"], { cwd: \"" + strings.ReplaceAll(dir, "\\", "\\\\") + "\", env: { TYA_CHILD: \"env\" }, stdin: \"input\" })\nprint(r[\"status\"])\nprint(r[\"stdout\"].contains(\"envinput\"))\nslow = process_run([\"sh\", \"-c\", \"sleep 1\"], { timeout: 0.01 })\nprint(slow[\"timed_out\"])\n"
	out := runEval(t, src)
	want := "0\ntrue\ntrue\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunProcessStructuredErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "unknown option", src: "process_run([\"sh\"], { bad: true })\n", want: "unknown option bad"},
		{name: "env value", src: "process_run([\"sh\"], { env: { BAD: 1 } })\n", want: "env values must be strings"},
		{name: "missing executable", src: "process_run([\"/no/such/tya-command\"], {})\n", want: "process.run:"},
		{name: "exec unsupported", src: "process_exec([\"sh\"], {})\n", want: "process.exec: unsupported"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runEvalError(t, tt.src)
			if err == nil {
				t.Fatal("expected process error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got error %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestRunFileCopyAndChmod(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	if err := os.WriteFile(src, []byte{0, 255, 65}, 0600); err != nil {
		t.Fatal(err)
	}
	code := "file_copy(\"" + src + "\", \"" + dst + "\", { overwrite: true })\nfile_chmod(\"" + dst + "\", 420)\nprint(bytes_array(file_read_bytes(\"" + dst + "\"))[1])\nprint(file_stat(\"" + dst + "\")[\"mode\"])\n"
	out := runEval(t, code)
	if out != "255\n420\n" {
		t.Fatalf("got %q", out)
	}
	err := runEvalError(t, "file_copy(\""+src+"\", \""+dst+"\", { overwrite: false })\n")
	if err == nil || !strings.Contains(err.Error(), "destination exists") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDirMkdirAllAndRemoveAll(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "a", "b")
	code := "dir_mkdir_all(\"" + dir + "\")\nprint(file_exists(\"" + dir + "\"))\ndir_remove_all(\"" + filepath.Dir(filepath.Dir(dir)) + "\")\nprint(file_exists(\"" + dir + "\"))\ndir_remove_all(\"" + filepath.Join(t.TempDir(), "missing") + "\")\n"
	out := runEval(t, code)
	if out != "true\nfalse\n" {
		t.Fatalf("got %q", out)
	}
	err := runEvalError(t, "dir_remove_all(\".\")\n")
	if err == nil || !strings.Contains(err.Error(), "dangerous path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDirWalkDeterministicOrder(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	code := "names = []\nrecord = entry -> names.push(entry[\"name\"])\ndir_walk(\"" + dir + "\", record, {})\nprint(names.join(\",\"))\n"
	out := runEval(t, code)
	if out != "b.txt,sub,a.txt\n" {
		t.Fatalf("got %q", out)
	}
}

func TestRunTempFileAndDir(t *testing.T) {
	code := "f1 = file_temp(\"tya-test\", \".tmp\")\nf2 = file_temp(\"tya-test\", \".tmp\")\nd = dir_temp_dir(\"tya-test\")\nprint(f1 != f2)\nprint(file_exists(f1))\nprint(file_exists(d))\ndir_remove_all(d)\nfile_remove(f1)\nfile_remove(f2)\n"
	out := runEval(t, code)
	if out != "true\ntrue\ntrue\n" {
		t.Fatalf("got %q", out)
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

func runEval(t *testing.T, src string) string {
	t.Helper()
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
	return out.String()
}

func runEvalError(t *testing.T, src string) error {
	t.Helper()
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	return Run(prog, &out)
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
		{name: "nil member", src: "print(nil.name)\n", want: "unknown member name on nil"},
		{name: "assignment arity", src: "first, second = [1]\n", want: "assignment expects 2 values, got 1"},
		{name: "function arity", src: "add = a, b -> a + b\nprint(add(1))\n", want: "function expects 2 arguments, got 1"},
		{name: "kind changing reassignment", src: "value = 1\nvalue = \"one\"\n", want: "cannot reassign value from number to string"},
		{name: "raise nil", src: "raise nil\n", want: "raise expects error value"},
		{name: "raise string", src: "raise \"failed\"\n", want: "raise expects error value"},
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
