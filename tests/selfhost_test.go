package tests

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"tya/internal/ast"
	"tya/internal/checker"
	"tya/internal/lexer"
	"tya/internal/parser"
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
	if string(out) != "ok\nadult\nyoung\nkomagata\ntrue\ntrue\ntrue\n2\ntrue\ntrue\ntrue\nloop\nTya\n" {
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

func TestSelfhostParserMatchesGoParserSubset(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/parser_subset.tya"
	tokensPath := dir + "/tokens.txt"
	src := "message = \" Tya \"\ntrimmed = trim message\ncount = 1 + 1\nresult = identity(trimmed)\ntried = try identity(trimmed)\nleft, right = result\ncallLeft, callRight = identity(trimmed)\nbareLeft, bareRight = identity \"value\"\nparts = split(trimmed, \"\\n\")\nreplaced = replace(trimmed, \"T\", trimmed)\nprint replace trimmed, \"T\", trimmed\nprint contains trimmed, \"T\"\nprint startsWith trimmed, \"T\"\nprint endsWith trimmed, \"a\"\nprint len trimmed\nif count >= 2\n  print trimmed\nelse\n  print \"small\"\nwhile count <= 2\n  break\nqueue = [trimmed, \"Other\"]\nuser = { name: trimmed }\nprint user.name\npush queue, trimmed\nfor entry in queue\n  print entry\nfor entry, index in queue\n  print entry\nfor key, value of user\n  print key\nreturn trimmed, \"ok\"\nreturn nil, error \"bad\"\nreturn { name: trimmed }, nil\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)
	got := summarizeSelfhostNodes(string(out))
	want := summarizeGoProgram(t, src)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestSelfhostCheckerMatchesGoCheckerUndefinedSubset(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/checker_subset.tya"
	tokensPath := dir + "/tokens.txt"
	nodesPath := dir + "/nodes.txt"
	src := "message = missing\nprint message\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)
	got := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath)))
	want := normalizeGoCheckerError(t, src)
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSelfhostCodegenMatchesInterpreterSubset(t *testing.T) {
	selfhostOut := string(run(t, "sh", "scripts/selfhost.sh", "examples/selfhost_ops.tya"))
	selfhostOut = strings.TrimPrefix(selfhostOut, "ok\n")
	interpOut := string(run(t, "go", "run", "./cmd/tya", "examples/selfhost_ops.tya"))
	if selfhostOut != interpOut {
		t.Fatalf("selfhost output %q, interpreter output %q", selfhostOut, interpOut)
	}
}

func TestSelfhostCodegenEmitsSimpleReturnFunctions(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:FUNC:identity:value\n2:INDENT:2\n2:RETURN:IDENT:value\n3:INDENT:0\n3:ASSIGN:message:STRING: Tya \n4:ASSIGN:trimmed:CALL1:trim:message\n5:ASSIGN:result:CALL1:identity:trimmed\n6:PRINT_CALL1:identity:trimmed\n7:ASSIGN:replaced:CALL3:replace:trimmed:STRING:T:trimmed\n8:PRINT_CALL3:replace:trimmed:STRING:T:trimmed\n9:PRINT_CALL2:contains:trimmed:STRING:T\n10:PRINT_CALL2:startsWith:trimmed:STRING:T\n11:PRINT_CALL2:endsWith:trimmed:STRING:a\n12:PRINT_CALL1:len:trimmed\n13:ASSIGN:user:OBJECT_ONE:name:IDENT:trimmed\n14:PRINT_MEMBER:user:name\n15:ASSIGN:source:CALL1_CALL0_INDEX:readFile:args:0\n16:ASSIGN:tokens:CALL1:lex:source\n17:ASSIGN:lines:CALL2:split:source:\\n\n18:ASSIGN:nodes:CALL1:parse:tokens\n19:ASSIGN:items:ARRAY_EMPTY:\n20:PUSH:items:IDENT:trimmed\n21:FOR:token:tokens\n22:INDENT:2\n22:PRINT:IDENT:token\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := string(run(t, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", path))
	if !strings.Contains(out, "const char *identity(const char *value)") {
		t.Fatalf("generated C missing function body:\n%s", out)
	}
	if !strings.Contains(out, "const char *trimmed = trim_text(message);") {
		t.Fatalf("generated C missing trim lowering:\n%s", out)
	}
	if !strings.Contains(out, "const char *result = identity(trimmed);") {
		t.Fatalf("generated C missing function call assignment:\n%s", out)
	}
	if !strings.Contains(out, "puts(identity(trimmed));") {
		t.Fatalf("generated C missing function call print:\n%s", out)
	}
	if !strings.Contains(out, "static char *replace_text(const char *text, const char *old_text, const char *new_text)") {
		t.Fatalf("generated C missing replace helper:\n%s", out)
	}
	if !strings.Contains(out, "const char *replaced = replace_text(trimmed, \"T\", trimmed);") {
		t.Fatalf("generated C missing replace lowering:\n%s", out)
	}
	if !strings.Contains(out, "puts(replace_text(trimmed, \"T\", trimmed));") {
		t.Fatalf("generated C missing print replace lowering:\n%s", out)
	}
	if !strings.Contains(out, "static int contains_text(const char *text, const char *needle)") {
		t.Fatalf("generated C missing contains helper:\n%s", out)
	}
	if !strings.Contains(out, "puts(contains_text(trimmed, \"T\") ? \"true\" : \"false\");") {
		t.Fatalf("generated C missing print contains lowering:\n%s", out)
	}
	if !strings.Contains(out, "puts(starts_with_text(trimmed, \"T\") ? \"true\" : \"false\");") {
		t.Fatalf("generated C missing print startsWith lowering:\n%s", out)
	}
	if !strings.Contains(out, "puts(ends_with_text(trimmed, \"a\") ? \"true\" : \"false\");") {
		t.Fatalf("generated C missing print endsWith lowering:\n%s", out)
	}
	if !strings.Contains(out, "printf(\"%ld\\n\", (long)strlen(trimmed));") {
		t.Fatalf("generated C missing print len lowering:\n%s", out)
	}
	if !strings.Contains(out, "const char *user = trimmed; /* object name */") {
		t.Fatalf("generated C missing object value placeholder:\n%s", out)
	}
	if !strings.Contains(out, "puts(user);") {
		t.Fatalf("generated C missing print member lowering:\n%s", out)
	}
	if !strings.Contains(out, "int main(int argc, char **argv)") {
		t.Fatalf("generated C missing argv-capable main:\n%s", out)
	}
	if !strings.Contains(out, "const char *source = argc > 1 ? read_file(argv[1]) : \"\";") {
		t.Fatalf("generated C missing readFile args()[0] lowering:\n%s", out)
	}
	if !strings.Contains(out, "static char **lex_source(const char *source, long *out_len)") {
		t.Fatalf("generated C missing lexer helper:\n%s", out)
	}
	if !strings.Contains(out, "char **tokens = lex_source(source, &tokens_len);") {
		t.Fatalf("generated C missing lex(source) lowering:\n%s", out)
	}
	if !strings.Contains(out, "for (long token_i = 0; token_i < tokens_len; token_i++)") {
		t.Fatalf("generated C missing dynamic array for loop:\n%s", out)
	}
	if !strings.Contains(out, "char **lines = split_lines(source, &lines_len);") {
		t.Fatalf("generated C missing split(source, newline) lowering:\n%s", out)
	}
	if !strings.Contains(out, "char **nodes = parse_tokens(tokens, tokens_len, &nodes_len);") {
		t.Fatalf("generated C missing parse(tokens) lowering:\n%s", out)
	}
	if !strings.Contains(out, "char **items = NULL;") || !strings.Contains(out, "items[items_len] = dup_text(trimmed);") {
		t.Fatalf("generated C missing dynamic array push lowering:\n%s", out)
	}
	if strings.Contains(out, "/* func identity") {
		t.Fatalf("generated C kept function comment:\n%s", out)
	}
}

func TestSelfhostCodegenRunsMultipleReturnSubset(t *testing.T) {
	dir := t.TempDir()
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/multiple_return.c"
	binPath := dir + "/multiple_return"
	nodes := "1:FUNC:parseUser:text\n2:INDENT:2\n2:IF_COMPARE_EQ:IDENT:text:STRING:\n3:INDENT:4\n3:RETURN2_CALL1:NIL:nil:error:STRING:empty user\n4:INDENT:2\n4:RETURN2_OBJECT_NIL:name:IDENT:text\n6:INDENT:0\n6:MULTI_ASSIGN2_CALL1:user:err:parseUser:STRING:komagata\n8:INDENT:0\n8:IF:IDENT:err\n9:INDENT:2\n9:PRINT_MEMBER:err:message\n10:INDENT:0\n10:ELSE\n11:INDENT:2\n11:PRINT_MEMBER:user:name\n13:INDENT:0\n13:MULTI_ASSIGN2_CALL1:missing:err:parseUser:STRING:\n15:INDENT:0\n15:IF:IDENT:err\n16:INDENT:2\n16:PRINT_MEMBER:err:message\n17:INDENT:0\n17:ELSE\n18:INDENT:2\n18:PRINT_MEMBER:missing:name\n"
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", "-std=c99", "-Wall", "-Wextra", "-pedantic", "-o", binPath, cPath)
	out := run(t, binPath)
	if string(out) != "komagata\nempty user\n" {
		t.Fatalf("got %q", out)
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

func TestStage1SelfhostSourcesEmitC(t *testing.T) {
	out := run(t, "sh", "scripts/stage1_selfhost_sources_check.sh")
	want := "selfhost/lexer.tya: stage-1 emitted and compiled C\nselfhost/parser.tya: stage-1 emitted and compiled C\nselfhost/checker.tya: stage-1 emitted and compiled C\nselfhost/codegen_c.tya: stage-1 emitted and compiled C\nexamples/hello.tya: stage-2 lexer matched\nexamples/hello.tya: stage-2 parser matched\nexamples/hello.tya: stage-2 checker matched\nexamples/hello.tya: stage-2 codegen deterministic\nexamples/hello.tya: stage-2 codegen matched\nescaped string print: stage-2 codegen deterministic\nescaped string print: stage-2 pipeline matched\nstring index print: stage-2 codegen deterministic\nstring index print: stage-2 pipeline matched\nexamples/string.tya: stage-2 codegen deterministic\nexamples/string.tya: stage-2 pipeline matched\nint literal: stage-2 lexer matched\noperators: stage-2 lexer matched\nfloat and string escape: stage-2 lexer matched\nint assignment: stage-2 parser matched\nint assignment: stage-2 codegen deterministic\nliteral assignments: stage-2 parser matched\nliteral assignments: stage-2 codegen deterministic\nliteral assignments: stage-2 checker matched\nint assignment: stage-2 codegen matched\nliteral assignments: stage-2 codegen matched\nliteral reassignment: stage-2 codegen deterministic\nliteral reassignment: stage-2 pipeline matched\nread file arg: stage-2 codegen deterministic\nread file arg: stage-2 pipeline matched\nlex source: stage-2 codegen deterministic\nlex source: stage-2 pipeline matched\nlong lex source: stage-2 pipeline matched\nparse tokens: stage-2 codegen deterministic\nparse tokens: stage-2 pipeline matched\ncheck nodes: stage-2 codegen deterministic\ncheck nodes: stage-2 pipeline matched\nemitC nodes: stage-2 codegen deterministic\nemitC nodes: stage-2 pipeline matched\nfunction body skip: stage-2 parser matched\nprint int assignment: stage-2 codegen deterministic\nprint int assignment: stage-2 pipeline matched\nprint literal assignments: stage-2 codegen deterministic\nprint literal assignments: stage-2 pipeline matched\nstring len print: stage-2 codegen deterministic\nstring len print: stage-2 pipeline matched\nstring trim print: stage-2 codegen deterministic\nstring trim print: stage-2 pipeline matched\nstring contains print: stage-2 codegen deterministic\nstring contains print: stage-2 pipeline matched\nstring prefix suffix print: stage-2 codegen deterministic\nstring prefix suffix print: stage-2 pipeline matched\nstring replace print: stage-2 codegen deterministic\nstring replace print: stage-2 pipeline matched\nstring split join print: stage-2 codegen deterministic\nstring split join print: stage-2 pipeline matched\nstring byte char length print: stage-2 codegen deterministic\nstring byte char length print: stage-2 pipeline matched\nint addition: stage-2 codegen deterministic\nint addition: stage-2 pipeline matched\ngrouped int addition: stage-2 codegen deterministic\ngrouped int addition: stage-2 pipeline matched\nint addition reassignment: stage-2 codegen deterministic\nint addition reassignment: stage-2 pipeline matched\nbool assignment: stage-2 codegen deterministic\nbool assignment: stage-2 pipeline matched\nbool logic: stage-2 codegen deterministic\nbool logic: stage-2 pipeline matched\nwhile false break: stage-2 codegen deterministic\nwhile false break: stage-2 pipeline matched\nwhile less-than break: stage-2 codegen deterministic\nwhile less-than break: stage-2 pipeline matched\nwhile bounded break: stage-2 codegen deterministic\nwhile bounded break: stage-2 pipeline matched\narray for: stage-2 codegen deterministic\narray for: stage-2 pipeline matched\nexamples/selfhost_ops.tya: stage-2 codegen deterministic\nexamples/selfhost_ops.tya: stage-2 pipeline matched\nexamples/multiple_return.tya: stage-2 parser matched\nexamples/multiple_return.tya: stage-2 checker matched\nexamples/while.tya: stage-2 codegen deterministic\nexamples/while.tya: stage-2 pipeline matched\nequality comparison: stage-2 codegen deterministic\nequality comparison: stage-2 pipeline matched\ninequality comparison: stage-2 codegen deterministic\ninequality comparison: stage-2 pipeline matched\nless-than comparison: stage-2 codegen deterministic\nless-than comparison: stage-2 pipeline matched\nbounded comparison: stage-2 codegen deterministic\nbounded comparison: stage-2 pipeline matched\ngrouped comparison: stage-2 codegen deterministic\ngrouped comparison: stage-2 pipeline matched\nselfhost/lexer.tya: stage-3 parser emitted real nodes\nselfhost/lexer.tya: stage-3 codegen emitted executable lexer C\nselfhost/parser.tya: stage-3 parser emitted real nodes\nselfhost/parser.tya: stage-3 codegen emitted executable parser C\nselfhost/checker.tya: stage-3 parser emitted real nodes\nselfhost/checker.tya: stage-3 codegen emitted executable checker C\nselfhost/codegen_c.tya: stage-3 parser emitted real nodes\nselfhost/codegen_c.tya: stage-3 codegen emitted executable codegen C\nstage4 hello: self-host pipeline matched\nstage4 print string: self-host pipeline matched\nstage4 print int: self-host pipeline matched\nstage4 escaped string: self-host pipeline matched\nstage4 colon string: self-host pipeline matched\nstage4 two prints: self-host pipeline matched\nstage4 assign print: self-host pipeline matched\nstage4 int assign print: self-host pipeline matched\nstage4 int reassignment print: self-host pipeline matched\nstage4 int addition print: self-host pipeline matched\nstage4 less-than print: self-host pipeline matched\nstage4 while false print: self-host pipeline matched\nstage4 array for: self-host pipeline matched\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostBootstrapCheck(t *testing.T) {
	if os.Getenv("TYA_RUN_SLOW_BOOTSTRAP_TEST") != "1" {
		t.Skip("covered by standalone scripts/selfhost_bootstrap_check.sh verification")
	}
	cmd := exec.Command("sh", "scripts/selfhost_bootstrap_check.sh")
	cmd.Dir = ".."
	cmd.Env = append(os.Environ(), "TYA_SKIP_STAGE1_SELFHOST_SOURCES=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sh [scripts/selfhost_bootstrap_check.sh]: %v\n%s", err, out)
	}
	if string(out) != "selfhost bootstrap: ok\n" {
		t.Fatalf("got %q", out)
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

func TestSelfhostCheckerRejectsBreakContinueOutsideLoop(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:BREAK\n2:CONTINUE\n3:WHILE:BOOL:true\n4:INDENT:2\n4:BREAK\n5:CONTINUE\n6:INDENT:0\n6:BREAK\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: break outside loop\n2: continue outside loop\n6: break outside loop\n"
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

func TestSelfhostCheckerChecksMultiAssign2Names(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:MULTI_ASSIGN2:left:right:IDENT:missing\n2:PRINT:IDENT:left\n3:MULTI_ASSIGN2:valid:also_bad:STRING:value\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missing\n3: invalid binding name: also_bad\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerChecksMultiAssign2CallNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:MULTI_ASSIGN2_CALL1:left:right:missingFunc:IDENT:missingArg\n2:PRINT:IDENT:left\n3:MULTI_ASSIGN2_CALL1:valid:also_bad:missingFunc:IDENT:missingArg\n4:MULTI_ASSIGN2_CALL1:ok:err:missingFunc:STRING:value\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingFunc\n1: undefined variable: missingArg\n3: invalid binding name: also_bad\n3: undefined variable: missingFunc\n3: undefined variable: missingArg\n4: undefined variable: missingFunc\n"
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

func TestSelfhostCheckerRejectsUndefinedPrintMemberNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:PRINT_MEMBER:missingObject:name\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingObject\n"
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
	if err := os.WriteFile(path, []byte("1:FUNC:f:\n2:INDENT:2\n2:RETURN:IDENT:missing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "2: undefined variable: missing\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedReturn2Names(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:FUNC:f:\n2:INDENT:2\n2:RETURN2:IDENT:missingLeft:IDENT:missingRight\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "2: undefined variable: missingLeft\n2: undefined variable: missingRight\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerChecksReturn2CallNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:FUNC:f:\n2:INDENT:2\n2:RETURN2_CALL1:IDENT:missingLeft:missingFunc:IDENT:missingArg\n3:RETURN2_CALL1:NIL:nil:error:STRING:bad\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "2: undefined variable: missingLeft\n2: undefined variable: missingFunc\n2: undefined variable: missingArg\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerChecksReturn2ObjectNilNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:FUNC:f:\n2:INDENT:2\n2:RETURN2_OBJECT_NIL:name:IDENT:missing\n3:RETURN2_OBJECT_NIL:name:STRING:value\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "2: undefined variable: missing\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedReturnCallNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:FUNC:f:\n2:INDENT:2\n2:RETURN_CALL2:missingFunc:IDENT:missingArg\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "2: undefined variable: missingFunc\n2: undefined variable: missingArg\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsReturnOutsideFunction(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:RETURN:INT:1\n2:RETURN2:INT:1:INT:2\n3:FUNC2:known:left:right\n4:ASSIGN:arg:STRING:value\n5:INDENT:0\n5:RETURN_CALL2:known:IDENT:arg\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: return outside function\n2: return outside function\n5: return outside function\n"
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

func TestSelfhostCheckerChecksTryCallNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:ASSIGN:top:TRY_CALL1:missingFunc:missingArg\n2:FUNC:read:arg\n3:INDENT:2\n3:ASSIGN:inside:TRY_CALL1:missingFunc:missingArg\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: try used outside function\n1: undefined variable: missingFunc\n1: undefined variable: missingArg\n3: undefined variable: missingFunc\n3: undefined variable: missingArg\n"
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

func TestSelfhostCheckerAllowsReplaceBuiltin(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:ASSIGN:message:STRING:Tya\n2:ASSIGN:trimmed:CALL1:trim:message\n3:ASSIGN:result:CALL3:replace:trimmed:STRING:T:trimmed\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	if string(out) != "ok\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostCheckerAllowsPrintReplaceBuiltin(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:ASSIGN:message:STRING:Tya\n2:PRINT_CALL3:replace:message:STRING:T:message\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	if string(out) != "ok\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostCheckerAllowsPrintContainsBuiltin(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:ASSIGN:message:STRING:Tya\n2:PRINT_CALL2:contains:message:STRING:T\n3:PRINT_CALL2:startsWith:message:STRING:T\n4:PRINT_CALL2:endsWith:message:STRING:a\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	if string(out) != "ok\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostCheckerRejectsUndefinedIndexNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:ASSIGN:first:INDEX:missingItems:i\n2:FUNC:f:\n3:INDENT:2\n3:RETURN:INDEX:missingItems:i\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingItems\n1: undefined variable: i\n3: undefined variable: missingItems\n3: undefined variable: i\n"
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

func TestSelfhostCheckerKeepsBlockLocalNamesScoped(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:INDENT:0\n1:IF:BOOL:true\n2:INDENT:2\n2:ASSIGN:localIf:INT:1\n3:INDENT:0\n3:PRINT:IDENT:localIf\n4:WHILE:BOOL:true\n5:INDENT:2\n5:ASSIGN:localWhile:INT:1\n6:INDENT:0\n6:PRINT:IDENT:localWhile\n7:ASSIGN:items:ARRAY_EMPTY:\n8:FOR:item:items\n9:INDENT:2\n9:PRINT:IDENT:item\n10:INDENT:0\n10:PRINT:IDENT:item\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "3: undefined variable: localIf\n6: undefined variable: localWhile\n10: undefined variable: item\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsDuplicateFunctionParams(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:FUNC2:same:a:a\n2:FUNC3:same3:a:b:a\n3:FUNC4:same4:a:b:c:b\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: duplicate function parameter: a\n2: duplicate function parameter: a\n3: duplicate function parameter: b\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsInvalidBindingNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:FUNC:show:User\n2:FUNC2:show2:user_name:ok\n3:ASSIGN:items:ARRAY_EMPTY:\n4:FOR:Item:items\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: invalid binding name: User\n2: invalid binding name: user_name\n4: invalid binding name: Item\n"
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
			out = append(out, selfhostToken(tok.Line, "INDENT", strconv.Itoa(leadingSpaces(lines[tok.Line-1])), 1))
			seenLine[tok.Line] = true
		}
		switch tok.Type {
		case token.NEWLINE, token.DEDENT, token.EOF:
			continue
		case token.INDENT:
			continue
		case token.ARROW:
			out = append(out, selfhostToken(tok.Line, "ARROW", tok.Lexeme, tok.Col))
		case token.IDENT, token.INT, token.FLOAT, token.STRING:
			out = append(out, selfhostToken(tok.Line, string(tok.Type), escapeSelfhostLexeme(tok.Lexeme), tok.Col))
		default:
			out = append(out, selfhostToken(tok.Line, "SYMBOL", tok.Lexeme, tok.Col))
		}
	}
	return out
}

func selfhostToken(line int, kind string, text string, col int) string {
	return strconv.Itoa(line) + ":" + kind + ":" + text + ":" + strconv.Itoa(col)
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

func summarizeSelfhostNodes(nodes string) []string {
	out := []string{}
	for _, line := range strings.Split(strings.TrimSpace(nodes), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) < 2 || parts[1] == "INDENT" {
			continue
		}
		out = append(out, strings.Join(parts[1:], ":"))
	}
	return out
}

func summarizeGoProgram(t *testing.T, src string) []string {
	t.Helper()
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	out := []string{}
	for _, stmt := range prog.Stmts {
		summarizeGoStmt(&out, stmt)
	}
	return out
}

func normalizeGoCheckerError(t *testing.T, src string) string {
	t.Helper()
	toks, errs := lexer.Lex(src)
	if len(errs) != 0 {
		t.Fatalf("lex errors: %v", errs)
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	err = checker.Check(prog)
	if err == nil {
		t.Fatal("expected checker error")
	}
	parts := strings.Split(err.Error(), ": ")
	if len(parts) != 2 {
		t.Fatalf("unexpected checker error: %v", err)
	}
	lineCol := strings.Split(parts[0], ":")
	return strings.Replace(lineCol[0]+": "+parts[1], "undefined variable ", "undefined variable: ", 1)
}

func summarizeGoStmt(out *[]string, stmt ast.Stmt) {
	switch n := stmt.(type) {
	case *ast.AssignStmt:
		if len(n.Targets) == 2 && len(n.Values) == 1 {
			if left, ok := n.Targets[0].(*ast.Ident); ok {
				if right, ok := n.Targets[1].(*ast.Ident); ok {
					if call, ok := n.Values[0].(*ast.CallExpr); ok {
						if id, ok := call.Callee.(*ast.Ident); ok && len(call.Args) == 1 {
							*out = append(*out, "MULTI_ASSIGN2_CALL1:"+left.Name+":"+right.Name+":"+id.Name+":"+summarizeGoKindedScalar(call.Args[0]))
							return
						}
					}
					*out = append(*out, "MULTI_ASSIGN2:"+left.Name+":"+right.Name+":"+summarizeGoExpr(n.Values[0]))
				}
			}
		}
		if len(n.Targets) == 1 && len(n.Values) == 1 {
			if id, ok := n.Targets[0].(*ast.Ident); ok {
				*out = append(*out, "ASSIGN:"+id.Name+":"+summarizeGoExpr(n.Values[0]))
			}
		}
	case *ast.ExprStmt:
		if call, ok := n.Expr.(*ast.CallExpr); ok {
			if id, ok := call.Callee.(*ast.Ident); ok && id.Name == "print" && len(call.Args) == 1 {
				if inner, ok := call.Args[0].(*ast.CallExpr); ok {
					if callee, ok := inner.Callee.(*ast.Ident); ok && len(inner.Args) == 1 {
						if arg, ok := inner.Args[0].(*ast.Ident); ok {
							*out = append(*out, "PRINT_CALL1:"+callee.Name+":"+arg.Name)
							return
						}
					}
					if callee, ok := inner.Callee.(*ast.Ident); ok && len(inner.Args) == 2 {
						if left, ok := inner.Args[0].(*ast.Ident); ok {
							*out = append(*out, "PRINT_CALL2:"+callee.Name+":"+left.Name+":"+summarizeGoKindedScalar(inner.Args[1]))
							return
						}
					}
					if callee, ok := inner.Callee.(*ast.Ident); ok && len(inner.Args) == 3 {
						if left, ok := inner.Args[0].(*ast.Ident); ok {
							*out = append(*out, "PRINT_CALL3:"+callee.Name+":"+left.Name+":"+summarizeGoKindedScalar(inner.Args[1])+":"+summarizeGoScalar(inner.Args[2]))
							return
						}
					}
				}
				if member, ok := call.Args[0].(*ast.MemberExpr); ok {
					if obj, ok := member.Object.(*ast.Ident); ok {
						*out = append(*out, "PRINT_MEMBER:"+obj.Name+":"+member.Name)
						return
					}
				}
				*out = append(*out, "PRINT:"+summarizeGoExpr(call.Args[0]))
			}
			if id, ok := call.Callee.(*ast.Ident); ok && id.Name == "push" && len(call.Args) == 2 {
				if target, ok := call.Args[0].(*ast.Ident); ok {
					*out = append(*out, "PUSH:"+target.Name+":"+summarizeGoExpr(call.Args[1]))
				}
			}
		}
	case *ast.IfStmt:
		*out = append(*out, "IF_"+summarizeGoConditionExpr(n.Cond))
		for _, child := range n.Then {
			summarizeGoStmt(out, child)
		}
		if len(n.Else) > 0 {
			*out = append(*out, "ELSE")
			for _, child := range n.Else {
				summarizeGoStmt(out, child)
			}
		}
	case *ast.WhileStmt:
		*out = append(*out, "WHILE_"+summarizeGoConditionExpr(n.Cond))
		for _, child := range n.Body {
			summarizeGoStmt(out, child)
		}
	case *ast.ForInStmt:
		*out = append(*out, "FOR:"+n.ValueName+":"+summarizeGoScalar(n.Iterable))
		for _, child := range n.Body {
			summarizeGoStmt(out, child)
		}
	case *ast.ReturnStmt:
		if len(n.Values) == 2 {
			if obj, ok := n.Values[0].(*ast.ObjectLit); ok {
				if _, ok := n.Values[1].(*ast.NilLit); ok && len(obj.Props) == 1 {
					*out = append(*out, "RETURN2_OBJECT_NIL:"+obj.Props[0].Name+":"+summarizeGoExpr(obj.Props[0].Value))
					return
				}
			}
		}
		if len(n.Values) == 2 {
			if call, ok := n.Values[1].(*ast.CallExpr); ok {
				if id, ok := call.Callee.(*ast.Ident); ok && len(call.Args) == 1 {
					*out = append(*out, "RETURN2_CALL1:"+summarizeGoExpr(n.Values[0])+":"+id.Name+":"+summarizeGoKindedScalar(call.Args[0]))
					return
				}
			}
		}
		if len(n.Values) == 2 {
			*out = append(*out, "RETURN2:"+summarizeGoExpr(n.Values[0])+":"+summarizeGoExpr(n.Values[1]))
		}
	case *ast.BreakStmt:
		*out = append(*out, "BREAK")
	case *ast.ContinueStmt:
		*out = append(*out, "CONTINUE")
	}
}

func summarizeGoExpr(expr ast.Expr) string {
	switch n := expr.(type) {
	case *ast.Ident:
		return "IDENT:" + n.Name
	case *ast.IntLit:
		return "INT:" + strconv.FormatInt(n.Value, 10)
	case *ast.StringLit:
		return "STRING:" + n.Value
	case *ast.BoolLit:
		if n.Value {
			return "BOOL:true"
		}
		return "BOOL:false"
	case *ast.NilLit:
		return "NIL:nil"
	case *ast.ArrayLit:
		if len(n.Elems) == 0 {
			return "ARRAY_EMPTY:"
		}
		if len(n.Elems) == 1 {
			return "ARRAY_ONE:" + summarizeGoExpr(n.Elems[0])
		}
		if len(n.Elems) == 2 {
			return "ARRAY_TWO:" + summarizeGoExpr(n.Elems[0]) + ":" + summarizeGoExpr(n.Elems[1])
		}
	case *ast.ObjectLit:
		if len(n.Props) == 1 {
			return "OBJECT_ONE:" + n.Props[0].Name + ":" + summarizeGoExpr(n.Props[0].Value)
		}
	case *ast.CallExpr:
		if id, ok := n.Callee.(*ast.Ident); ok {
			if len(n.Args) == 1 {
				if arg, ok := n.Args[0].(*ast.Ident); ok {
					return "CALL1:" + id.Name + ":" + arg.Name
				}
			}
			if len(n.Args) == 2 {
				if left, ok := n.Args[0].(*ast.Ident); ok {
					return "CALL2:" + id.Name + ":" + left.Name + ":" + summarizeGoScalar(n.Args[1])
				}
			}
			if len(n.Args) == 3 {
				if left, ok := n.Args[0].(*ast.Ident); ok {
					return "CALL3:" + id.Name + ":" + left.Name + ":" + summarizeGoKindedScalar(n.Args[1]) + ":" + summarizeGoScalar(n.Args[2])
				}
			}
		}
	case *ast.TryExpr:
		return "TRY_" + summarizeGoExpr(n.Expr)
	case *ast.BinaryExpr:
		left := summarizeGoScalar(n.Left)
		right := summarizeGoScalar(n.Right)
		switch n.Op.Lexeme {
		case "+":
			return "INT_ADD:" + left + ":" + right
		case ">=":
			return "COMPARE_GE:" + left + ":" + right
		case "<=":
			return "COMPARE_LE:" + left + ":" + right
		}
	}
	return "UNKNOWN"
}

func summarizeGoConditionExpr(expr ast.Expr) string {
	if bin, ok := expr.(*ast.BinaryExpr); ok {
		switch bin.Op.Lexeme {
		case ">=":
			return "COMPARE_GE:" + summarizeGoKindedScalar(bin.Left) + ":" + summarizeGoKindedScalar(bin.Right)
		case "<=":
			return "COMPARE_LE:" + summarizeGoKindedScalar(bin.Left) + ":" + summarizeGoKindedScalar(bin.Right)
		case "!=":
			return "COMPARE_NE:" + summarizeGoKindedScalar(bin.Left) + ":" + summarizeGoKindedScalar(bin.Right)
		case "<":
			return "COMPARE_LT:" + summarizeGoKindedScalar(bin.Left) + ":" + summarizeGoKindedScalar(bin.Right)
		case "==":
			return "COMPARE_EQ:" + summarizeGoKindedScalar(bin.Left) + ":" + summarizeGoKindedScalar(bin.Right)
		}
	}
	return summarizeGoExpr(expr)
}

func summarizeGoKindedScalar(expr ast.Expr) string {
	switch n := expr.(type) {
	case *ast.Ident:
		return "IDENT:" + n.Name
	case *ast.IntLit:
		return "INT:" + strconv.FormatInt(n.Value, 10)
	case *ast.StringLit:
		return "STRING:" + n.Value
	case *ast.BoolLit:
		if n.Value {
			return "BOOL:true"
		}
		return "BOOL:false"
	}
	return summarizeGoExpr(expr)
}

func summarizeGoScalar(expr ast.Expr) string {
	switch n := expr.(type) {
	case *ast.Ident:
		return n.Name
	case *ast.IntLit:
		return strconv.FormatInt(n.Value, 10)
	case *ast.StringLit:
		return strings.NewReplacer("\n", "\\n", "\t", "\\t").Replace(n.Value)
	}
	return summarizeGoExpr(expr)
}
