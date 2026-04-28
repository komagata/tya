package tests

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"tya/internal/lexer"
	"tya/internal/token"
)

func TestSelfhostPrototypePipeline(t *testing.T) {
	out := run(t, "sh", "scripts/selfhost.sh")
	if string(out) != "ok\nshort\nsame text\neither\nhas t\nboth\nTya\nTya\nTya\n3\ntrue\nfalse\ntrue\ntrue\nIndented\nCompared\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostElseExample(t *testing.T) {
	path := t.TempDir() + "/else.tya"
	src := "flag = false\nif flag\n  print \"yes\"\nelse\n  print \"no\"\n"
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "sh", "scripts/selfhost.sh", path)
	if string(out) != "ok\nno\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostOpsExample(t *testing.T) {
	out := run(t, "sh", "scripts/selfhost.sh", "examples/selfhost_ops.tya")
	if string(out) != "ok\nadult\nyoung\nkomagata\ntrue\ntrue\ntrue\nloop\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostWhileExample(t *testing.T) {
	out := run(t, "sh", "scripts/selfhost.sh", "examples/while.tya")
	if string(out) != "ok\n10\n11\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostIdentityCallExample(t *testing.T) {
	path := t.TempDir() + "/identity.tya"
	src := "message = \"Tya\"\nidentity = value ->\n  return value\necho = value -> value\nresult = identity message\nprint result\nprint echo message\n"
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "sh", "scripts/selfhost.sh", path)
	if string(out) != "ok\nTya\nTya\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostLexerSourceChecks(t *testing.T) {
	out := run(t, "sh", "scripts/selfhost_check.sh")
	want := "selfhost/lexer.tya: ok\nselfhost/parser.tya: ok\nselfhost/checker.tya: ok\nselfhost/codegen_c.tya: ok\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostLexerMatchesGoLexerSubset(t *testing.T) {
	path := t.TempDir() + "/tokens.tya"
	src := "name = \"Ty\\\"a\"\nratio = 12.5\nitems = [1, 2]\nuser = { name: name }\n@count = @count + 1\nif ratio >= 10 and name != \"\"\n  print name\n"
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/lexer.tya", path)
	got := strings.TrimSpace(string(out))
	want := strings.Join(goLexerSelfhostTokens(t, src), "\n")
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestSelfhostSourcesCompileToC(t *testing.T) {
	out := run(t, "sh", "scripts/selfhost_compile_check.sh")
	want := "selfhost/lexer.tya: compiled\nselfhost/parser.tya: compiled\nselfhost/checker.tya: compiled\nselfhost/codegen_c.tya: compiled\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestGoEmitterCompilesSelfhostSourcesToC(t *testing.T) {
	out := run(t, "sh", "scripts/go_emit_selfhost_compile_check.sh")
	want := "selfhost/lexer.tya: go-emit compiled\nselfhost/parser.tya: go-emit compiled\nselfhost/checker.tya: go-emit compiled\nselfhost/codegen_c.tya: go-emit compiled\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestGoEmittedSelfhostPipelineRuns(t *testing.T) {
	out := run(t, "sh", "scripts/go_emit_selfhost_run_check.sh")
	want := "Hello, Tya\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestGoEmitterMatchesSelectedExamples(t *testing.T) {
	out := run(t, "sh", "scripts/go_emit_examples_check.sh")
	want := "examples/hello.tya: matched\nexamples/arithmetic.tya: matched\nexamples/function.tya: matched\nexamples/return.tya: matched\nexamples/multiple_return.tya: matched\nexamples/try.tya: matched\nexamples/while.tya: matched\nexamples/if.tya: matched\nexamples/logic.tya: matched\nexamples/array.tya: matched\nexamples/array_function.tya: matched\nexamples/string.tya: matched\nexamples/object.tya: matched\nexamples/object_inline.tya: matched\nexamples/object_builtin.tya: matched\nexamples/method.tya: matched\nexamples/prelude.tya: matched\nexamples/convert.tya: matched\nexamples/error.tya: matched\nexamples/file.tya: matched\nexamples/equal.tya: matched\nexamples/for.tya: matched\nexamples/for_object.tya: matched\nexamples/read_line.tya: matched\nexamples/exit.tya: matched\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestGoEmitterMatchesArgsExample(t *testing.T) {
	out := run(t, "sh", "scripts/go_emit_args_check.sh")
	if string(out) != "examples/args.tya: matched\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostCheckerRejectsUndefinedConditionNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:IF:IDENT:missingIf\n2:WHILE:IDENT:missingWhile\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingIf\n2: undefined variable: missingWhile\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedAssignmentNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:ASSIGN:alias:IDENT:missing\n2:ASSIGN:ok:COMPARE_GE:missing:1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missing\n2: undefined variable: missing\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedPrintCallNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:PRINT_CALL1:missingFunc:missingArg\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingFunc\n1: undefined variable: missingArg\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedNotNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:ASSIGN:negated:BOOL_NOT:missing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missing\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedPushNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:PUSH:missingTarget:IDENT:missingValue\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingTarget\n1: undefined variable: missingValue\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedReturnNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:RETURN:IDENT:missing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missing\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedReturnCallNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:RETURN_CALL2:missingFunc:IDENT:missingArg\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingFunc\n1: undefined variable: missingArg\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedCallNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:ASSIGN:result:CALL1:missingFunc:missingArg\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingFunc\n1: undefined variable: missingArg\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedTwoArgCallNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:ASSIGN:result:CALL2:missingFunc:left:right\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingFunc\n1: undefined variable: left\n1: undefined variable: right\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedThreeArgCallNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:ASSIGN:result:CALL3:missingFunc:left:STRING:literal:right\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingFunc\n1: undefined variable: left\n1: undefined variable: right\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedIndexNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:ASSIGN:first:INDEX:missingItems:i\n2:RETURN:INDEX:missingItems:i\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingItems\n1: undefined variable: i\n2: undefined variable: missingItems\n2: undefined variable: i\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedCallConditionNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:IF_CALL_LT:missingFunc:missingArg:limit\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingFunc\n1: undefined variable: missingArg\n1: undefined variable: limit\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedOneArgCallConditionNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:IF_CALL1:missingFunc:missingArg\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingFunc\n1: undefined variable: missingArg\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedCallAndCallConditionNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:IF_CALL_EQ_AND_CALL_EQ:missingFunc:left:STRING:x:missingFunc2:right:STRING:y\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingFunc\n1: undefined variable: left\n1: undefined variable: missingFunc2\n1: undefined variable: right\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedNotCallConditionNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:IF_NOT_CALL2:missingFunc:left:right\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingFunc\n1: undefined variable: left\n1: undefined variable: right\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedWhileCallConditionNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:WHILE_LT_CALL:left:missingFunc:missingArg\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: left\n1: undefined variable: missingFunc\n1: undefined variable: missingArg\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedWhileCompareNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:WHILE_COMPARE_LT:IDENT:left:IDENT:right\n2:WHILE_COMPARE_NE:IDENT:left:IDENT:right\n3:WHILE_COMPARE_GE:IDENT:left:IDENT:right\n4:WHILE_COMPARE_LE:IDENT:left:IDENT:right\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: left\n1: undefined variable: right\n2: undefined variable: left\n2: undefined variable: right\n3: undefined variable: left\n3: undefined variable: right\n4: undefined variable: left\n4: undefined variable: right\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedCompareConditionNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:IF_COMPARE_EQ:IDENT:left:IDENT:right\n2:IF_COMPARE_NE:IDENT:left:IDENT:right\n3:IF_COMPARE_GE:IDENT:left:IDENT:right\n4:IF_COMPARE_LE:IDENT:left:IDENT:right\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: left\n1: undefined variable: right\n2: undefined variable: left\n2: undefined variable: right\n3: undefined variable: left\n3: undefined variable: right\n4: undefined variable: left\n4: undefined variable: right\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedOrCompareConditionNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:IF_COMPARE_OR:IDENT:left:IDENT:right:IDENT:left2:IDENT:right2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: left\n1: undefined variable: right\n1: undefined variable: left2\n1: undefined variable: right2\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedCallIndexNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:ASSIGN:input:CALL0_INDEX:missingArgs:i\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingArgs\n1: undefined variable: i\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedCallWithCallIndexNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:ASSIGN:source:CALL1_CALL0_INDEX:missingRead:missingArgs:i\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingRead\n1: undefined variable: missingArgs\n1: undefined variable: i\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedForCollections(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:FOR:item:missingItems\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingItems\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func runToFile(t *testing.T, path string, name string, args ...string) {
	t.Helper()
	out := run(t, name, args...)
	if err := os.WriteFile(path, out, 0644); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, name string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
	return out
}

func goLexerSelfhostTokens(t *testing.T, src string) []string {
	t.Helper()
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	lines := strings.Split(src, "\n")
	out := []string{}
	seenLine := map[int]bool{}
	for _, tok := range toks {
		if tok.Type != token.NEWLINE && tok.Type != token.DEDENT && tok.Type != token.EOF && !seenLine[tok.Line] {
			out = append(out, selfhostToken(tok.Line, "INDENT", strconv.Itoa(leadingSpaces(lines[tok.Line-1]))))
			seenLine[tok.Line] = true
		}
		switch tok.Type {
		case token.NEWLINE, token.DEDENT, token.EOF:
			continue
		case token.INDENT:
			continue
		case token.ARROW:
			out = append(out, selfhostToken(tok.Line, "ARROW", tok.Lexeme))
		case token.IDENT, token.INT, token.FLOAT, token.STRING:
			out = append(out, selfhostToken(tok.Line, string(tok.Type), escapeSelfhostLexeme(tok.Lexeme)))
		default:
			out = append(out, selfhostToken(tok.Line, "SYMBOL", tok.Lexeme))
		}
	}
	return out
}

func selfhostToken(line int, kind string, text string) string {
	return strconv.Itoa(line) + ":" + kind + ":" + text
}

func leadingSpaces(s string) int {
	n := 0
	for n < len(s) && s[n] == ' ' {
		n++
	}
	return n
}

func escapeSelfhostLexeme(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}
