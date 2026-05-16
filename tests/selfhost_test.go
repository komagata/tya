//go:build selfhost_legacy && pre_v01_legacy_ast

package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"tya/internal/ast"
	"tya/internal/checker"
	"tya/internal/lexer"
	"tya/internal/parser"
	"tya/internal/token"
)

var selfhostAstGeneratedToolsOnce sync.Once
var selfhostAstGeneratedTools map[string]string
var selfhostAstGeneratedToolsErr string

func selfhostNodeShapes(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	shapes := make([]string, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			shapes = append(shapes, parts[1])
		} else if line != "" {
			shapes = append(shapes, line)
		}
	}
	return strings.Join(shapes, "\n")
}

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

func TestSelfhostClassicArraySumExample(t *testing.T) {
	out := run(t, "sh", "scripts/selfhost.sh", "examples/classic/array_sum.tya")
	if string(out) != "ok\n6\n" {
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
	src := "message = \" Tya \"\ntrimmed = trim message\ncount = 1 + 1\nremaining = count - 1\ndoubled = count * 2\nhalved = doubled / 2\nmodded = doubled % 2\ngrouped = (doubled * 3)\nlarge = count > 0\nresult = identity(trimmed)\ntried = try identity(trimmed)\nleft, right = result\ncallLeft, callRight = identity(trimmed)\nbareLeft, bareRight = identity \"value\"\nparts = split(trimmed, \"\\n\")\nreplaced = replace(trimmed, \"T\", trimmed)\nprint replace trimmed, \"T\", trimmed\nprint contains trimmed, \"T\"\nprint starts_with trimmed, \"T\"\nprint ends_with trimmed, \"a\"\nprint len trimmed\nif count > 0\n  print trimmed\nif count >= 2\n  print trimmed\nelse\n  print \"small\"\nwhile count > 0\n  break\nwhile count <= 2\n  break\nqueue = [trimmed, \"Other\"]\nuser = { name: trimmed }\nprint trimmed\npush queue, trimmed\nfor entry in queue\n  print entry\nfor entry, index in queue\n  print entry\nfor entry in user\n  print entry[\"key\"]\nreturn trimmed, \"ok\"\nreturn nil, error \"bad\"\nreturn { name: trimmed }, nil\n"
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

func TestSelfhostParserBuildsAstBeforeLegacyAdapter(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "selfhost", "parser.tya"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	required := []string{
		"parse_ast = tokens ->",
		"program_ast = stmts ->",
		"body_of = nodes ->",
		"scalar_name_of = token ->",
		"expr_kind = expr ->",
		"expr_value = expr ->",
		"expr_name = expr ->",
		"expr_legacy_kind = expr ->",
		"expr_line = expr ->",
		"binary_op_of = legacy_kind ->",
		"binary_kind_of = op ->",
		"is_binary_op_text = op ->",
		"is_binary_op_token = token ->",
		"is_grouped_binary_op_text = op ->",
		"is_grouped_binary_op_token = token ->",
		"binary_op_expr = op, left, right ->",
		"binary_result_expr = op, left, right ->",
		"parse_binary_expr = left, op, right ->",
		"parse_binary_at = tokens, start ->",
		"parse_grouped_binary_at = tokens, start ->",
		"expr_parse_result = expr, next ->",
		"result_expr = result ->",
		"result_next = result ->",
		"binary_precedence_of = op ->",
		"is_scalar_ast_expr = expr ->",
		"parse_primary_result_at = tokens, start, not_kind ->",
		"parse_precedence_result_at = tokens, start, min_prec, not_kind ->",
		"result = parse_primary_result_at tokens, next + 1, not_kind",
		"right = result_expr result",
		"right_next = result_next result",
		"while right_next + 1 < len(tokens) and line_of(tokens[right_next]) == expr_line(right) and is_binary_op_token(tokens[right_next]) and binary_precedence_of(text_of(tokens[right_next])) > prec",
		"right = binary_result_expr text_of(right_op), right, result_expr(result)",
		"left = binary_result_expr text_of(op), left, right",
		"next = right_next",
		"parse_expr_result_at = tokens, start ->",
		`parse_precedence_result_at tokens, start, 1, "BOOL_NOT"`,
		"parse_expr_with_unary_at = tokens, start, not_kind ->",
		"result = parse_precedence_result_at tokens, start, 1, not_kind",
		"result_expr result",
		"parse_expr_at = tokens, start ->",
		`op: binary_op_of(legacy_kind)`,
		"binary_expr binary_kind_of(op), left, right",
		"binary_op_expr text_of(op), left, right",
		"parse_binary_expr left, op, right",
		"parse_postfix_result_at = tokens, start ->",
		"while next < len(tokens) and line_of(tokens[next]) == expr_line(expr)",
		"expr = member_expr_from_expr expr, tokens[next + 1]",
		"expr = index_expr_from_expr expr, tokens[next + 1]",
		"arg = scalar_expr(tokens[next])",
		"arg = member_expr_from_expr arg, tokens[next + 2]",
		"arg = index_expr_from_expr arg, tokens[next + 2]",
		"args = [arg]",
		"expr = call_expr_from_expr expr, args",
		"parse_postfix_result_at tokens, start",
		"result = parse_expr_result_at tokens, i + 1",
		"i = result_next result",
		"result = parse_expr_result_at tokens, start",
		"parse_try_expr_result_at = tokens, start ->",
		"inner = result_expr result",
		"expr = try_expr inner",
		"parse_try_expr_at = tokens, start ->",
		"legacy_stmt = stmt ->",
		"legacy_program = nodes ->",
		"parse = tokens ->",
		"stmts = parse_ast tokens",
		"legacy_program stmts",
		`{ kind: "program", body: stmts }`,
		"body = body_of nodes",
		`targets: [name]`,
		`values: [expr]`,
		`values: exprs`,
		`value_name: item`,
		`index_name: index`,
		`iterable: collection`,
		"stmt_values = stmt ->",
		"stmt_value_name = stmt ->",
		"stmt_index_name = stmt ->",
		"stmt_iterable = stmt ->",
		"exprs = stmt_values stmt",
		"with_indent = stmt, indent ->",
		"push_stmt_ast = stmts, stmt, indent ->",
		`indent: indent`,
		"push_stmt_ast stmts, stmt, current_indent",
		"scalar_kind_of = token ->",
		`return "ident"`,
		`return "int"`,
		`return "string"`,
		`return "bool"`,
		`return "nil"`,
		`name: scalar_name_of(token)`,
		"legacy_call1_arg_kind = arg ->",
		"legacy_call1_arg_payload = arg ->",
		"legacy_call1 = callee, arg ->",
		"legacy_call2 = callee, args ->",
		"legacy_call2_kinded = callee, args ->",
		"legacy_call3 = callee, args ->",
		`return legacy_call1(expr["callee"], args[0])`,
		`return legacy_call2 expr["callee"], args`,
		`return legacy_call3 expr["callee"], args`,
		`return stmt["line"] + ":PRINT_" + legacy_call1(stmt["expr"]["callee"], args[0])`,
		`return stmt["line"] + ":PRINT_" + legacy_call2_kinded(stmt["expr"]["callee"], args)`,
		`return stmt["line"] + ":PRINT_" + legacy_call3(stmt["expr"]["callee"], args)`,
	}
	for _, marker := range required {
		if !strings.Contains(source, marker) {
			t.Fatalf("selfhost parser is missing AST adapter marker %q", marker)
		}
	}
	forbidden := []string{
		"push nodes, legacy_assign",
		"push nodes, legacy_print",
		"push nodes, legacy_condition",
		"push nodes, legacy_return",
		"push nodes, legacy_for",
		"push nodes, legacy_func",
	}
	for _, marker := range forbidden {
		if strings.Contains(source, marker) {
			t.Fatalf("selfhost parser still writes legacy node strings directly during parsing with %q", marker)
		}
	}
	if strings.Contains(source, "parse_expr_at(") {
		t.Fatal("selfhost parser still uses parse_expr_at instead of expression result parsing")
	}
	if strings.Contains(source, `{ kind: "scalar", legacy_kind:`) {
		t.Fatal("selfhost parser still collapses scalar AST expressions to kind scalar")
	}
}

func TestSelfhostAstParserCoversSelfhostStatementKinds(t *testing.T) {
	for _, srcPath := range []string{
		"selfhost/lexer.tya",
		"selfhost/parser.tya",
		"selfhost/checker.tya",
		"selfhost/codegen_c.tya",
	} {
		t.Run(srcPath, func(t *testing.T) {
			dir := t.TempDir()
			tokensPath := filepath.Join(dir, "tokens.txt")
			astTokensPath := filepath.Join(dir, "ast_tokens.txt")
			nodesPath := filepath.Join(dir, "nodes.txt")
			runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
			tokens, err := os.ReadFile(tokensPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
				t.Fatal(err)
			}
			runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
			nodes, err := os.ReadFile(nodesPath)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(nodes), ":AST_STMT:") {
				t.Fatalf("%s still has unsupported AST statement nodes:\n%s", srcPath, nodes)
			}
		})
	}
}

func TestSelfhostAstParserCoversSupportedManifestStatementKinds(t *testing.T) {
	manifestRaw, err := os.ReadFile(filepath.Join("..", "scripts", "selfhost_examples_manifest.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(manifestRaw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 2 || parts[1] != "supported" {
			continue
		}
		srcPath := parts[0]
		t.Run(srcPath, func(t *testing.T) {
			dir := t.TempDir()
			tokensPath := filepath.Join(dir, "tokens.txt")
			astTokensPath := filepath.Join(dir, "ast_tokens.txt")
			nodesPath := filepath.Join(dir, "nodes.txt")
			runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
			tokens, err := os.ReadFile(tokensPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
				t.Fatal(err)
			}
			runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
			nodes, err := os.ReadFile(nodesPath)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(nodes), ":AST_STMT:") {
				t.Fatalf("%s still has unsupported AST statement nodes:\n%s", srcPath, nodes)
			}
		})
	}
}

func TestSelfhostAstCheckerAcceptsSelfhostSources(t *testing.T) {
	for _, srcPath := range []string{
		"selfhost/lexer.tya",
		"selfhost/parser.tya",
		"selfhost/checker.tya",
		"selfhost/codegen_c.tya",
	} {
		t.Run(srcPath, func(t *testing.T) {
			dir := t.TempDir()
			tokensPath := filepath.Join(dir, "tokens.txt")
			astTokensPath := filepath.Join(dir, "ast_tokens.txt")
			nodesPath := filepath.Join(dir, "nodes.txt")
			runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
			tokens, err := os.ReadFile(tokensPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
				t.Fatal(err)
			}
			runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
			out := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
			if out != "ok\n" {
				t.Fatalf("%s AST checker got %q", srcPath, out)
			}
		})
	}
}

func TestSelfhostAstCodegenCompilesSelfhostSources(t *testing.T) {
	for _, srcPath := range []string{
		"selfhost/lexer.tya",
		"selfhost/parser.tya",
		"selfhost/checker.tya",
		"selfhost/codegen_c.tya",
	} {
		t.Run(srcPath, func(t *testing.T) {
			dir := t.TempDir()
			tokensPath := filepath.Join(dir, "tokens.txt")
			astTokensPath := filepath.Join(dir, "ast_tokens.txt")
			nodesPath := filepath.Join(dir, "nodes.txt")
			cPath := filepath.Join(dir, "out.c")
			binPath := filepath.Join(dir, "out")
			runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
			tokens, err := os.ReadFile(tokensPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
				t.Fatal(err)
			}
			runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
			runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
			run(t, "cc", cPath, "-o", binPath)
		})
	}
}

func compileSelfhostAstGeneratedTools(t *testing.T, dir string) map[string]string {
	t.Helper()
	selfhostAstGeneratedToolsOnce.Do(func() {
		cacheDir, err := os.MkdirTemp("", "tya-selfhost-ast-tools-*")
		if err != nil {
			selfhostAstGeneratedToolsErr = err.Error()
			return
		}
		bins := map[string]string{}
		for _, tool := range []string{"lexer", "parser", "checker", "codegen_c"} {
			tokensPath := filepath.Join(cacheDir, tool+".tokens")
			astTokensPath := filepath.Join(cacheDir, tool+".ast_tokens")
			nodesPath := filepath.Join(cacheDir, tool+".nodes")
			cPath := filepath.Join(cacheDir, tool+".c")
			binPath := filepath.Join(cacheDir, tool)
			runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/"+tool+".tya")
			tokens, err := os.ReadFile(tokensPath)
			if err != nil {
				selfhostAstGeneratedToolsErr = err.Error()
				return
			}
			if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
				selfhostAstGeneratedToolsErr = err.Error()
				return
			}
			runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
			runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
			run(t, "cc", cPath, "-o", binPath)
			bins[tool] = binPath
		}
		selfhostAstGeneratedTools = bins
	})
	if selfhostAstGeneratedToolsErr != "" {
		t.Fatal(selfhostAstGeneratedToolsErr)
	}
	bins := map[string]string{}
	for tool, path := range selfhostAstGeneratedTools {
		bins[tool] = path
	}
	return bins
}

func runSelfhostAstGeneratedPipeline(t *testing.T, dir string, bins map[string]string, srcPath string) string {
	t.Helper()
	out, errOut, status := runSelfhostAstGeneratedPipelineResult(t, dir, bins, srcPath)
	if status != 0 {
		t.Fatalf("%s exited with status %d\nstdout:\n%s\nstderr:\n%s", srcPath, status, out, errOut)
	}
	if errOut != "" {
		t.Fatalf("%s wrote unexpected stderr:\n%s", srcPath, errOut)
	}
	return out
}

func runSelfhostAstGeneratedPipelineResult(t *testing.T, dir string, bins map[string]string, srcPath string) (string, string, int) {
	t.Helper()
	base := strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
	inputTokensPath := filepath.Join(dir, base+".tokens")
	inputAstTokensPath := filepath.Join(dir, base+".ast_tokens")
	inputNodesPath := filepath.Join(dir, base+".nodes")
	checkPath := filepath.Join(dir, base+".check")
	outCPath := filepath.Join(dir, base+".c")
	outBinPath := filepath.Join(dir, base)
	runToFile(t, inputTokensPath, bins["lexer"], srcPath)
	tokens, err := os.ReadFile(inputTokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputAstTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, inputNodesPath, bins["parser"], inputAstTokensPath)
	runToFile(t, checkPath, bins["checker"], inputNodesPath)
	checkOut, err := os.ReadFile(checkPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(checkOut) != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, outCPath, bins["codegen_c"], inputNodesPath)
	run(t, "cc", outCPath, "-o", outBinPath)
	cmd := exec.Command(outBinPath)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}
	status, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("%s: %v\nstdout:\n%s\nstderr:\n%s", outBinPath, err, stdout.String(), stderr.String())
	}
	return stdout.String(), stderr.String(), status.ExitCode()
}

func TestSelfhostAstGeneratedLexerRunsOnSimpleProgram(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "lexer_ast.c")
	binPath := filepath.Join(dir, "lexer_ast")
	srcPath := filepath.Join(dir, "hello.tya")
	if err := os.WriteFile(srcPath, []byte("print \"Hello\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/lexer.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath, srcPath))
	want := "1:INDENT:0:1\n1:IDENT:print:1\n1:STRING:Hello:7\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedParserRunsOnSimpleTokenStream(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "parser_ast.c")
	binPath := filepath.Join(dir, "parser_ast")
	srcPath := filepath.Join(dir, "hello.tya")
	inputTokensPath := filepath.Join(dir, "hello.tokens")
	if err := os.WriteFile(srcPath, []byte("print \"Hello\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/parser.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	runToFile(t, inputTokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := string(run(t, binPath, inputTokensPath))
	want := "1:PRINT:STRING:Hello\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedParserRunsAstModeOnSimpleTokenStream(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "parser_ast.c")
	binPath := filepath.Join(dir, "parser_ast")
	srcPath := filepath.Join(dir, "hello.tya")
	inputTokensPath := filepath.Join(dir, "hello.tokens")
	inputAstTokensPath := filepath.Join(dir, "hello.ast_tokens")
	if err := os.WriteFile(srcPath, []byte("print \"Hello\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/parser.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	runToFile(t, inputTokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	inputTokens, err := os.ReadFile(inputTokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputAstTokensPath, append([]byte("0:IDENT:ASTMODE\n"), inputTokens...), 0644); err != nil {
		t.Fatal(err)
	}
	out := selfhostNodeShapes(string(run(t, binPath, inputAstTokensPath)))
	want := "INDENT:0\nAST_PRINT:string(Hello)"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedParserRunsAstModeOnAssignPrintTokenStream(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "parser_ast.c")
	binPath := filepath.Join(dir, "parser_ast")
	srcPath := filepath.Join(dir, "assign_print.tya")
	inputTokensPath := filepath.Join(dir, "assign_print.tokens")
	inputAstTokensPath := filepath.Join(dir, "assign_print.ast_tokens")
	if err := os.WriteFile(srcPath, []byte("message = \"Hello\"\nready = true\nmissing = nil\nitems = []\none = [\"A\"]\ntwo = [\"A\", \"B\"]\nclean = trim(message)\nhas = contains message, \"H\"\nchanged = replace message, \"H\", \"T\"\nchanged_alias = replace message, \"H\", changed\nalias = message\nsame = message == alias\nsmall = 1 < 2\ndifferent = message != \"Other\"\nlarge = 3 >= 2\nbigger = 3 > 2\nbounded = 2 <= 3\nparen = (3 > 2)\nprint message\nprint trim(message)\nprint trim message\nprint contains message, \"H\"\nprint replace message, \"H\", \"T\"\nprint replace message, \"H\", alias\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/parser.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	runToFile(t, inputTokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	inputTokens, err := os.ReadFile(inputTokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputAstTokensPath, append([]byte("0:IDENT:ASTMODE\n"), inputTokens...), 0644); err != nil {
		t.Fatal(err)
	}
	out := selfhostNodeShapes(string(run(t, binPath, inputAstTokensPath)))
	want := "INDENT:0\nAST_ASSIGN:message:string(Hello)\nINDENT:0\nAST_ASSIGN:ready:bool(true)\nINDENT:0\nAST_ASSIGN:missing:nil(nil)\nINDENT:0\nAST_ASSIGN:items:array0()\nINDENT:0\nAST_ASSIGN:one:array1(string(A))\nINDENT:0\nAST_ASSIGN:two:array2(string(A) string(B))\nINDENT:0\nAST_ASSIGN:clean:call(trim ident(message))\nINDENT:0\nAST_ASSIGN:has:call(contains ident(message) string(H))\nINDENT:0\nAST_ASSIGN:changed:call(replace ident(message) string(H) string(T))\nINDENT:0\nAST_ASSIGN:changed_alias:call(replace ident(message) string(H) ident(changed))\nINDENT:0\nAST_ASSIGN:alias:ident(message)\nINDENT:0\nAST_ASSIGN:same:binary(== ident(message) ident(alias))\nINDENT:0\nAST_ASSIGN:small:binary(< int(1) int(2))\nINDENT:0\nAST_ASSIGN:different:binary(!= ident(message) string(Other))\nINDENT:0\nAST_ASSIGN:large:binary(>= int(3) int(2))\nINDENT:0\nAST_ASSIGN:bigger:binary(> int(3) int(2))\nINDENT:0\nAST_ASSIGN:bounded:binary(<= int(2) int(3))\nINDENT:0\nAST_ASSIGN:paren:binary(> int(3) int(2))\nINDENT:0\nAST_PRINT:ident(message)\nINDENT:0\nAST_PRINT:call(trim ident(message))\nINDENT:0\nAST_PRINT:call(trim ident(message))\nINDENT:0\nAST_PRINT:call(contains ident(message) string(H))\nINDENT:0\nAST_PRINT:call(replace ident(message) string(H) string(T))\nINDENT:0\nAST_PRINT:call(replace ident(message) string(H) ident(alias))"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedParserRunsAstModeOnIfElseTokenStream(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "parser_ast.c")
	binPath := filepath.Join(dir, "parser_ast")
	srcPath := filepath.Join(dir, "if_else.tya")
	inputTokensPath := filepath.Join(dir, "if_else.tokens")
	inputAstTokensPath := filepath.Join(dir, "if_else.ast_tokens")
	src := "flag = \"off\"\nif flag == \"on\"\n  print \"yes\"\nelse\n  print \"no\"\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/parser.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	runToFile(t, inputTokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	inputTokens, err := os.ReadFile(inputTokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputAstTokensPath, append([]byte("0:IDENT:ASTMODE\n"), inputTokens...), 0644); err != nil {
		t.Fatal(err)
	}
	out := string(run(t, binPath, inputAstTokensPath))
	for _, want := range []string{
		"2:AST_IF:binary(== ident(flag) string(on))",
		"4:AST_ELSE",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("got:\n%s\nmissing %q", out, want)
		}
	}
}

func TestSelfhostGeneratedParserDoesNotTreatStringTokenTextAsIfSyntax(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "parser_ast.c")
	binPath := filepath.Join(dir, "parser_ast")
	srcPath := filepath.Join(dir, "string_if_marker.tya")
	inputTokensPath := filepath.Join(dir, "string_if_marker.tokens")
	src := "push lines, \"    if (strstr(tokens[i], \\\":IDENT:if:\\\")) {\"\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/parser.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	runToFile(t, inputTokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := string(run(t, binPath, inputTokensPath))
	if strings.Contains(out, ":IF") || strings.Contains(out, ":AST_IF") {
		t.Fatalf("generated parser treated string token text as if syntax:\n%s", out)
	}
	if count := strings.Count(out, ":PUSH:lines:"); count != 1 {
		t.Fatalf("got %d push nodes, want 1:\n%s", count, out)
	}
}

func TestSelfhostGeneratedParserDoesNotTreatStringTokenTextAsPrintSyntax(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "parser_ast.c")
	binPath := filepath.Join(dir, "parser_ast")
	srcPath := filepath.Join(dir, "string_print_marker.tya")
	inputTokensPath := filepath.Join(dir, "string_print_marker.tokens")
	src := "push lines, \"    if (strstr(tokens[i], \\\":IDENT:print:\\\")) {\"\n" +
		"push lines, \"    if (strstr(tokens[i + 3], \\\":SYMBOL:,:\\\") && strstr(tokens[i + 4], \\\":STRING:\\\")) {\"\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/parser.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	runToFile(t, inputTokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := string(run(t, binPath, inputTokensPath))
	if strings.Contains(out, ":PRINT") || strings.Contains(out, ":AST_PRINT") {
		t.Fatalf("generated parser treated string token text as print syntax:\n%s", out)
	}
	if count := strings.Count(out, ":PUSH:lines:"); count != 2 {
		t.Fatalf("got %d push nodes, want 2:\n%s", count, out)
	}
}

func TestSelfhostGeneratedParserDoesNotTreatStringTokenTextAsReturnSyntax(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "parser_ast.c")
	binPath := filepath.Join(dir, "parser_ast")
	srcPath := filepath.Join(dir, "string_return_marker.tya")
	inputTokensPath := filepath.Join(dir, "string_return_marker.tokens")
	src := "push lines, \"    if (strstr(tokens[i], \\\":IDENT:return:\\\")) {\"\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/parser.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	runToFile(t, inputTokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := string(run(t, binPath, inputTokensPath))
	if strings.Contains(out, ":RETURN") || strings.Contains(out, ":AST_RETURN") {
		t.Fatalf("generated parser treated string token text as return syntax:\n%s", out)
	}
	if count := strings.Count(out, ":PUSH:lines:"); count != 1 {
		t.Fatalf("got %d push nodes, want 1:\n%s", count, out)
	}
}

func TestSelfhostGeneratedParserDoesNotTreatStringTokenTextAsWhileSyntax(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "parser_ast.c")
	binPath := filepath.Join(dir, "parser_ast")
	srcPath := filepath.Join(dir, "string_while_marker.tya")
	inputTokensPath := filepath.Join(dir, "string_while_marker.tokens")
	src := "push lines, \"    if (strstr(tokens[i], \\\":IDENT:while:\\\")) {\"\n" +
		"push lines, \"    if (strstr(tokens[i], \\\":IDENT:for:\\\") && strstr(tokens[i + 2], \\\":IDENT:in:\\\")) {\"\n" +
		"push lines, \"    if (strstr(tokens[i], \\\":IDENT:break:\\\") || strstr(tokens[i], \\\":IDENT:continue:\\\")) {\"\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/parser.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	runToFile(t, inputTokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := string(run(t, binPath, inputTokensPath))
	if strings.Contains(out, ":WHILE") || strings.Contains(out, ":AST_WHILE") {
		t.Fatalf("generated parser treated string token text as while syntax:\n%s", out)
	}
	if count := strings.Count(out, ":PUSH:lines:"); count != 3 {
		t.Fatalf("got %d push nodes, want 3:\n%s", count, out)
	}
}

func TestSelfhostGeneratedParserDoesNotTreatStringTokenTextAsAssignmentSyntax(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "parser_ast.c")
	binPath := filepath.Join(dir, "parser_ast")
	srcPath := filepath.Join(dir, "string_assignment_marker.tya")
	inputTokensPath := filepath.Join(dir, "string_assignment_marker.tokens")
	src := "push lines, \"    if (strstr(tokens[i + 1], \\\":SYMBOL:=:\\\") && strstr(tokens[i + 3], \\\":SYMBOL:==:\\\")) {\"\n" +
		"push lines, \"    if (strstr(tokens[i + 2], \\\":SYMBOL:(:\\\") && strstr(tokens[i + 4], \\\":SYMBOL:>=:\\\") && strstr(tokens[i + 6], \\\":SYMBOL:):\\\")) {\"\n" +
		"push lines, \"    if (strstr(tokens[i + 2], \\\":SYMBOL:(:\\\") && strstr(tokens[i + 4], \\\":SYMBOL:+:\\\") && strstr(tokens[i + 7], \\\":SYMBOL:*:\\\")) {\"\n" +
		"push lines, \"    if (strstr(tokens[i + 3], \\\":SYMBOL:-:\\\") || strstr(tokens[i + 3], \\\":SYMBOL:+:\\\")) {\"\n" +
		"push lines, \"    if (strstr(tokens[i + 3], \\\":SYMBOL:+:\\\") && strstr(tokens[i + 5], \\\":SYMBOL:[:\\\") && strstr(tokens[i + 7], \\\":SYMBOL:]:\\\")) {\"\n" +
		"push lines, \"    if (strstr(tokens[i + 2], \\\":IDENT:true:\\\") || strstr(tokens[i + 2], \\\":IDENT:false:\\\")) {\"\n" +
		"push lines, \"    if (strstr(tokens[i + 2], \\\":SYMBOL:[:\\\") && strstr(tokens[i + 3], \\\":SYMBOL:]:\\\")) {\"\n" +
		"push lines, \"    if (strstr(tokens[i + 2], \\\":SYMBOL:[:\\\") && strstr(tokens[i + 3], \\\":STRING:\\\") && strstr(tokens[i + 6], \\\":SYMBOL:]:\\\")) {\"\n" +
		"push lines, \"    if (strstr(tokens[i + 3], \\\":ARROW:->:\\\") || strstr(tokens[i + 5], \\\":ARROW:->:\\\")) {\"\n" +
		"push lines, \"    if (strstr(tokens[i + 4], \\\":SYMBOL:(:\\\") && strstr(tokens[i + 6], \\\":SYMBOL:[:\\\") && strstr(tokens[i + 8], \\\":SYMBOL:]:\\\")) {\"\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/parser.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	runToFile(t, inputTokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := string(run(t, binPath, inputTokensPath))
	if strings.Contains(out, ":ASSIGN:") || strings.Contains(out, ":AST_ASSIGN:") {
		t.Fatalf("generated parser treated string token text as assignment syntax:\n%s", out)
	}
	if count := strings.Count(out, ":PUSH:lines:"); count != 10 {
		t.Fatalf("got %d push nodes, want 10:\n%s", count, out)
	}
}

func TestSelfhostAstGeneratedParserRunsAstModeOnWhileTokenStream(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "parser_ast.c")
	binPath := filepath.Join(dir, "parser_ast")
	srcPath := filepath.Join(dir, "while.tya")
	inputTokensPath := filepath.Join(dir, "while.tokens")
	inputAstTokensPath := filepath.Join(dir, "while.ast_tokens")
	src := "i = 0\nwhile i < 3\n  print i\n  i = i + 1\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/parser.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	runToFile(t, inputTokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	inputTokens, err := os.ReadFile(inputTokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputAstTokensPath, append([]byte("0:IDENT:ASTMODE\n"), inputTokens...), 0644); err != nil {
		t.Fatal(err)
	}
	out := string(run(t, binPath, inputAstTokensPath))
	want := "2:AST_WHILE:binary(< ident(i) int(3))"
	if !strings.Contains(out, want) {
		t.Fatalf("got:\n%s\nmissing %q", out, want)
	}
}

func TestSelfhostAstGeneratedParserRunsAstModeOnArrayForTokenStream(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "parser_ast.c")
	binPath := filepath.Join(dir, "parser_ast")
	srcPath := filepath.Join(dir, "array_for.tya")
	inputTokensPath := filepath.Join(dir, "array_for.tokens")
	inputAstTokensPath := filepath.Join(dir, "array_for.ast_tokens")
	src := "items = [\"A\", \"B\"]\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/parser.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	runToFile(t, inputTokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	inputTokens, err := os.ReadFile(inputTokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputAstTokensPath, append([]byte("0:IDENT:ASTMODE\n"), inputTokens...), 0644); err != nil {
		t.Fatal(err)
	}
	out := string(run(t, binPath, inputAstTokensPath))
	for _, want := range []string{
		"1:AST_ASSIGN:items:array2(string(A) string(B))",
		"2:AST_FOR:item:items",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("got:\n%s\nmissing %q", out, want)
		}
	}
}

func TestSelfhostAstGeneratedParserRunsAstModeOnFunctionCallTokenStream(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "parser_ast.c")
	binPath := filepath.Join(dir, "parser_ast")
	srcPath := filepath.Join(dir, "function_call.tya")
	inputTokensPath := filepath.Join(dir, "function_call.tokens")
	inputAstTokensPath := filepath.Join(dir, "function_call.ast_tokens")
	src := "identity = value ->\n  return value\nmessage = \"Hello\"\nprint identity(message)\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/parser.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	runToFile(t, inputTokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	inputTokens, err := os.ReadFile(inputTokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputAstTokensPath, append([]byte("0:IDENT:ASTMODE\n"), inputTokens...), 0644); err != nil {
		t.Fatal(err)
	}
	out := string(run(t, binPath, inputAstTokensPath))
	for _, want := range []string{
		"1:AST_FUNC:identity:value",
		"2:AST_RETURN:ident(value)",
		"3:AST_ASSIGN:message:string(Hello)",
		"4:AST_PRINT:call(identity ident(message))",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("got:\n%s\nmissing %q", out, want)
		}
	}
}

func TestSelfhostAstGeneratedParserRunsAstModeOnArrayReturnFunctionTokenStream(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "parser_ast.c")
	binPath := filepath.Join(dir, "parser_ast")
	srcPath := filepath.Join(dir, "array_return.tya")
	inputTokensPath := filepath.Join(dir, "array_return.tokens")
	inputAstTokensPath := filepath.Join(dir, "array_return.ast_tokens")
	src := "collect = value ->\n  items = []\n  item = value\n  push items, item\n  push items, \"done\"\n  return items\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/parser.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	runToFile(t, inputTokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	inputTokens, err := os.ReadFile(inputTokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputAstTokensPath, append([]byte("0:IDENT:ASTMODE\n"), inputTokens...), 0644); err != nil {
		t.Fatal(err)
	}
	out := string(run(t, binPath, inputAstTokensPath))
	for _, want := range []string{
		"1:AST_FUNC:collect:value",
		"2:AST_ASSIGN:items:array0()",
		"3:AST_ASSIGN:item:ident(value)",
		"4:AST_PUSH:items:ident(item)",
		"5:AST_PUSH:items:string(done)",
		"6:AST_RETURN:ident(items)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("got:\n%s\nmissing %q", out, want)
		}
	}
}

func TestSelfhostAstGeneratedParserRunsAstModeOnExitPanicTokenStream(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "parser_ast.c")
	binPath := filepath.Join(dir, "parser_ast")
	srcPath := filepath.Join(dir, "exit_panic.tya")
	inputTokensPath := filepath.Join(dir, "exit_panic.tokens")
	inputAstTokensPath := filepath.Join(dir, "exit_panic.ast_tokens")
	src := "code = 7\nexit code\npanic \"bad\"\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/parser.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	runToFile(t, inputTokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	inputTokens, err := os.ReadFile(inputTokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputAstTokensPath, append([]byte("0:IDENT:ASTMODE\n"), inputTokens...), 0644); err != nil {
		t.Fatal(err)
	}
	out := string(run(t, binPath, inputAstTokensPath))
	for _, want := range []string{
		"2:AST_EXIT:ident(code)",
		"3:AST_PANIC:string(bad)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("got:\n%s\nmissing %q", out, want)
		}
	}
}

func TestSelfhostAstGeneratedCheckerRunsOnSimpleNodeStream(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "checker_ast.c")
	binPath := filepath.Join(dir, "checker_ast")
	srcPath := filepath.Join(dir, "hello.tya")
	inputTokensPath := filepath.Join(dir, "hello.tokens")
	inputNodesPath := filepath.Join(dir, "hello.nodes")
	if err := os.WriteFile(srcPath, []byte("print \"Hello\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/checker.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	runToFile(t, inputTokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	runToFile(t, inputNodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", inputTokensPath)
	out := string(run(t, binPath, inputNodesPath))
	if out != "ok\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsOnSimpleNodeStream(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "tokens.txt")
	astTokensPath := filepath.Join(dir, "ast_tokens.txt")
	nodesPath := filepath.Join(dir, "nodes.txt")
	cPath := filepath.Join(dir, "codegen_ast.c")
	binPath := filepath.Join(dir, "codegen_ast")
	srcPath := filepath.Join(dir, "hello.tya")
	inputTokensPath := filepath.Join(dir, "hello.tokens")
	inputNodesPath := filepath.Join(dir, "hello.nodes")
	outCPath := filepath.Join(dir, "hello.c")
	outBinPath := filepath.Join(dir, "hello")
	if err := os.WriteFile(srcPath, []byte("print \"Hello\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/codegen_c.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	runToFile(t, inputTokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	runToFile(t, inputNodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", inputTokensPath)
	runToFile(t, outCPath, binPath, inputNodesPath)
	run(t, "cc", outCPath, "-o", outBinPath)
	out := string(run(t, outBinPath))
	if out != "Hello\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsSimpleProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "hello.tya")
	if err := os.WriteFile(srcPath, []byte("print \"Hello\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bins := map[string]string{}
	for _, tool := range []string{"lexer", "parser", "checker", "codegen_c"} {
		tokensPath := filepath.Join(dir, tool+".tokens")
		astTokensPath := filepath.Join(dir, tool+".ast_tokens")
		nodesPath := filepath.Join(dir, tool+".nodes")
		cPath := filepath.Join(dir, tool+".c")
		binPath := filepath.Join(dir, tool)
		runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/"+tool+".tya")
		tokens, err := os.ReadFile(tokensPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
			t.Fatal(err)
		}
		runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
		runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
		run(t, "cc", cPath, "-o", binPath)
		bins[tool] = binPath
	}
	inputTokensPath := filepath.Join(dir, "hello.tokens")
	inputNodesPath := filepath.Join(dir, "hello.nodes")
	checkPath := filepath.Join(dir, "hello.check")
	outCPath := filepath.Join(dir, "hello.c")
	outBinPath := filepath.Join(dir, "hello")
	runToFile(t, inputTokensPath, bins["lexer"], srcPath)
	runToFile(t, inputNodesPath, bins["parser"], inputTokensPath)
	runToFile(t, checkPath, bins["checker"], inputNodesPath)
	checkOut, err := os.ReadFile(checkPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(checkOut) != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, outCPath, bins["codegen_c"], inputNodesPath)
	run(t, "cc", outCPath, "-o", outBinPath)
	out := string(run(t, outBinPath))
	if out != "Hello\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsArrayForProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "array_for.tya")
	if err := os.WriteFile(srcPath, []byte("items = [\"A\", \"B\"]\nfor item in items\n  print item\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bins := map[string]string{}
	for _, tool := range []string{"lexer", "parser", "checker", "codegen_c"} {
		tokensPath := filepath.Join(dir, tool+".tokens")
		astTokensPath := filepath.Join(dir, tool+".ast_tokens")
		nodesPath := filepath.Join(dir, tool+".nodes")
		cPath := filepath.Join(dir, tool+".c")
		binPath := filepath.Join(dir, tool)
		runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/"+tool+".tya")
		tokens, err := os.ReadFile(tokensPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
			t.Fatal(err)
		}
		runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
		runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
		run(t, "cc", cPath, "-o", binPath)
		bins[tool] = binPath
	}
	inputTokensPath := filepath.Join(dir, "array_for.tokens")
	inputNodesPath := filepath.Join(dir, "array_for.nodes")
	checkPath := filepath.Join(dir, "array_for.check")
	outCPath := filepath.Join(dir, "array_for.c")
	outBinPath := filepath.Join(dir, "array_for")
	runToFile(t, inputTokensPath, bins["lexer"], srcPath)
	runToFile(t, inputNodesPath, bins["parser"], inputTokensPath)
	nodes, err := os.ReadFile(inputNodesPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(nodes), "1:ASSIGN:items:ARRAY_TWO:STRING:A:STRING:B\n") {
		t.Fatalf("nodes:\n%s", nodes)
	}
	runToFile(t, checkPath, bins["checker"], inputNodesPath)
	checkOut, err := os.ReadFile(checkPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(checkOut) != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, outCPath, bins["codegen_c"], inputNodesPath)
	run(t, "cc", outCPath, "-o", outBinPath)
	out := string(run(t, outBinPath))
	if out != "A\nB\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeArrayForProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "array_for_ast.tya")
	if err := os.WriteFile(srcPath, []byte("items = [\"A\", \"B\"]\nfor item in items\n  print item\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "A\nB\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsIntWhileProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "int_while.tya")
	if err := os.WriteFile(srcPath, []byte("i = 0\nwhile i < 3\n  print i\n  i = i + 1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bins := map[string]string{}
	for _, tool := range []string{"lexer", "parser", "checker", "codegen_c"} {
		tokensPath := filepath.Join(dir, tool+".tokens")
		astTokensPath := filepath.Join(dir, tool+".ast_tokens")
		nodesPath := filepath.Join(dir, tool+".nodes")
		cPath := filepath.Join(dir, tool+".c")
		binPath := filepath.Join(dir, tool)
		runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/"+tool+".tya")
		tokens, err := os.ReadFile(tokensPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
			t.Fatal(err)
		}
		runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
		runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
		run(t, "cc", cPath, "-o", binPath)
		bins[tool] = binPath
	}
	inputTokensPath := filepath.Join(dir, "int_while.tokens")
	inputNodesPath := filepath.Join(dir, "int_while.nodes")
	checkPath := filepath.Join(dir, "int_while.check")
	outCPath := filepath.Join(dir, "int_while.c")
	outBinPath := filepath.Join(dir, "int_while")
	runToFile(t, inputTokensPath, bins["lexer"], srcPath)
	runToFile(t, inputNodesPath, bins["parser"], inputTokensPath)
	runToFile(t, checkPath, bins["checker"], inputNodesPath)
	checkOut, err := os.ReadFile(checkPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(checkOut) != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, outCPath, bins["codegen_c"], inputNodesPath)
	run(t, "cc", outCPath, "-o", outBinPath)
	out := string(run(t, outBinPath))
	if out != "0\n1\n2\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeIntWhileProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "int_while_ast.tya")
	if err := os.WriteFile(srcPath, []byte("i = 0\nwhile i < 3\n  print i\n  i = i + 1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "0\n1\n2\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsFunctionCallProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "function_call.tya")
	if err := os.WriteFile(srcPath, []byte("identity = value ->\n  return value\nmessage = \"Hello\"\nprint identity(message)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bins := map[string]string{}
	for _, tool := range []string{"lexer", "parser", "checker", "codegen_c"} {
		tokensPath := filepath.Join(dir, tool+".tokens")
		astTokensPath := filepath.Join(dir, tool+".ast_tokens")
		nodesPath := filepath.Join(dir, tool+".nodes")
		cPath := filepath.Join(dir, tool+".c")
		binPath := filepath.Join(dir, tool)
		runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "selfhost/"+tool+".tya")
		tokens, err := os.ReadFile(tokensPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
			t.Fatal(err)
		}
		runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
		runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
		run(t, "cc", cPath, "-o", binPath)
		bins[tool] = binPath
	}
	inputTokensPath := filepath.Join(dir, "function_call.tokens")
	inputNodesPath := filepath.Join(dir, "function_call.nodes")
	checkPath := filepath.Join(dir, "function_call.check")
	outCPath := filepath.Join(dir, "function_call.c")
	outBinPath := filepath.Join(dir, "function_call")
	runToFile(t, inputTokensPath, bins["lexer"], srcPath)
	runToFile(t, inputNodesPath, bins["parser"], inputTokensPath)
	runToFile(t, checkPath, bins["checker"], inputNodesPath)
	checkOut, err := os.ReadFile(checkPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(checkOut) != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, outCPath, bins["codegen_c"], inputNodesPath)
	run(t, "cc", outCPath, "-o", outBinPath)
	out := string(run(t, outBinPath))
	if out != "Hello\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeFunctionCallProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "function_call_ast.tya")
	if err := os.WriteFile(srcPath, []byte("identity = value ->\n  return value\nmessage = \"Hello\"\nprint identity(message)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "Hello\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeConditionalStringFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "conditional_string_function_ast.tya")
	src := "escape = char ->\n  if char == \"n\"\n    return \"\\n\"\n  if char == \"t\"\n    return \"\\t\"\n  return char\nprint escape(\"n\")\nprint escape(\"x\")\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "\n\nx\n" {
		t.Fatalf("got %q, want escaped newline then x", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeImplicitBoolFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "implicit_bool_function_ast.tya")
	src := "is_space = char ->\n  char == \" \"\nprint is_space(\" \")\nprint is_space(\"x\")\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "true\nfalse\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeImplicitContainsBoolFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "implicit_contains_bool_function_ast.tya")
	src := "is_digit = char ->\n  contains \"012\", char\nprint is_digit(\"1\")\nprint is_digit(\"x\")\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "true\nfalse\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeReplaceCallAssignmentProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "replace_call_assign_ast.tya")
	src := "message = \"Tya\"\nchanged = replace message, \"T\", \"M\"\nchanged_again = replace message, \"a\", changed\nprint changed\nprint changed_again\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "Mya\nTyMya\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeArgsEnvProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "args_env_ast.tya")
	src := "items = args()\nprint len items\nprint env \"TYA_EXAMPLE\"\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "0\nnil\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeArithmeticProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "arithmetic_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "arithmetic.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/arithmetic.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeArrayProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "array_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "array.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/array.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeArrayFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "array_function_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "archive", "pre-v0.1", "array_function.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/archive/pre-v0.1/array_function.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeClassProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "class_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "class.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/class.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeClassicArraySumProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "classic_array_sum_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "classic", "array_sum.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/classic/array_sum.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeClassicFibProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "classic_fib_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "classic", "fib.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/classic/fib.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeClassicFizzBuzzProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "classic_fizzbuzz_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "classic", "fizzbuzz.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/classic/fizzbuzz.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeClassicGcdProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "classic_gcd_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "classic", "gcd.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/classic/gcd.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeClassicPrimeProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "classic_prime_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "classic", "prime.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/classic/prime.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeConvertProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "convert_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "convert.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/convert.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeDictSetProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "dict_set_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "dict_set.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/dict_set.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeEqualProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "equal_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "equal.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/equal.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeErrorProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "error_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "error.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/error.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeExitProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "exit_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "exit.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/exit.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModePanicProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "panic_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "panic.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out, errOut, status := runSelfhostAstGeneratedPipelineResult(t, dir, bins, srcPath)
	if status != 1 {
		t.Fatalf("got status %d stdout %q stderr %q, want status 1", status, out, errOut)
	}
	if out != "" {
		t.Fatalf("got stdout %q, want empty stdout", out)
	}
	if errOut != "panic: bad state\n" {
		t.Fatalf("got stderr %q, want panic stderr", errOut)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeFileProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "file_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "file.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/file.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeForProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "for_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "for.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/for.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeForObjectProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "for_object_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "for_object.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/for_object.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeFunctionProgramExample(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "function_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "function.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/function.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeIfProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "if_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "if.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/if.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeInheritanceProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "inheritance_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "inheritance.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/inheritance.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeInterfaceProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "interface_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "interface.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/interface.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeLogicProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "logic_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "logic.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/logic.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeMethodProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "method_ast.tya")
	raw, err := os.ReadFile(filepath.Join("..", "examples", "method.tya"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	want := string(run(t, "go", "run", "./cmd/tya", "examples/method.tya"))
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeImplicitComposedBoolFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "implicit_composed_bool_function_ast.tya")
	src := "is_digit = char ->\n  contains \"012\", char\nis_lower = char ->\n  contains \"abc\", char\nis_upper = char ->\n  contains \"ABC\", char\nis_alpha = char ->\n  is_lower(char) or is_upper(char) or char == \"_\"\nis_alpha_num = char ->\n  is_alpha(char) or is_digit(char)\nprint is_alpha_num(\"a\")\nprint is_alpha_num(\"A\")\nprint is_alpha_num(\"_\")\nprint is_alpha_num(\"1\")\nprint is_alpha_num(\"-\")\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "true\ntrue\ntrue\ntrue\nfalse\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeImplicitComparisonChainBoolFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "implicit_comparison_chain_bool_function_ast.tya")
	src := "is_space = char ->\n  char == \" \" or char == \"\\n\" or char == \"\\t\"\nprint is_space(\" \")\nprint is_space(\"\\n\")\nprint is_space(\"\\t\")\nprint is_space(\"x\")\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "true\ntrue\ntrue\nfalse\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "array_return_function_ast.tya")
	src := "collect = value ->\n  items = []\n  item = value\n  push items, item\n  push items, \"done\"\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "Tya\ndone\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeCallPushArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "call_push_array_return_function_ast.tya")
	src := "collect = value ->\n  items = []\n  push items, trim value\n  return items\nmessage = \" Tya \"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "Tya\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeToStringPushArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "to_string_push_array_return_function_ast.tya")
	src := "collect = value ->\n  items = []\n  count = 7\n  push items, to_string count\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "7\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeStringIndexPushArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "string_index_push_array_return_function_ast.tya")
	src := "collect = value ->\n  items = []\n  i = 1\n  char = value[i]\n  push items, char\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "y\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeDirectStringIndexPushArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "direct_string_index_push_array_return_function_ast.tya")
	src := "collect = value ->\n  items = []\n  i = 1\n  push items, value[i]\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "y\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeStringIndexConditionArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "string_index_condition_array_return_function_ast.tya")
	src := "collect = value ->\n  items = []\n  i = 1\n  if value[i] == \"y\"\n    push items, value[i]\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "y\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineSkipsAstModeFalseStringIndexConditionArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "false_string_index_condition_array_return_function_ast.tya")
	src := "collect = value ->\n  items = []\n  i = 0\n  if value[i] == \"y\"\n    push items, value[i]\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeCompoundWhileStringIndexConditionArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "compound_while_string_index_condition_array_return_function_ast.tya")
	src := "collect = value ->\n  items = []\n  i = 0\n  while i < 2 and value[i] != \"a\"\n    push items, value[i]\n    i = i + 1\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "T\ny\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineStopsAstModeCompoundWhileStringIndexConditionArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "stopping_compound_while_string_index_condition_array_return_function_ast.tya")
	src := "collect = value ->\n  items = []\n  i = 0\n  while i < 3 and value[i] != \"a\"\n    push items, value[i]\n    i = i + 1\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "T\ny\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeLenCompoundWhileStringIndexConditionArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "len_compound_while_string_index_condition_array_return_function_ast.tya")
	src := "collect = value ->\n  items = []\n  i = 0\n  while i < len(value) and value[i] != \"a\"\n    push items, value[i]\n    i = i + 1\n  return items\nmessage = \"Ty\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "T\ny\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineStopsAstModeLenCompoundWhileStringIndexConditionArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "stopping_len_compound_while_string_index_condition_array_return_function_ast.tya")
	src := "collect = value ->\n  items = []\n  i = 0\n  while i < len(value) and value[i] != \"a\"\n    push items, value[i]\n    i = i + 1\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "T\ny\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeStringIndexAccumulationArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "string_index_accumulation_array_return_function_ast.tya")
	src := "collect = value ->\n  items = []\n  text = \"\"\n  i = 0\n  while i < len(value) and value[i] != \"a\"\n    text = text + value[i]\n    i = i + 1\n  push items, text\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "Ty\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeLoopingArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "looping_array_return_function_ast.tya")
	src := "collect = value ->\n  items = []\n  i = 0\n  while i < 2\n    push items, value\n    i = i + 1\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "Tya\nTya\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeLoopingArrayReturnWithContinueProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "looping_array_return_continue_ast.tya")
	src := "collect = value ->\n  items = []\n  i = 0\n  while i < 2\n    if i == 0\n      i = i + 1\n      continue\n    push items, value\n    i = i + 1\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "Tya\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelinePreservesArrayFunctionLoopStatementOrder(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "loop_stmt_order_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  text = \"\"\n  i = 0\n  while i < 2\n    push items, text\n    text = text + value\n    i = i + 1\n  return items\nmessage = \"Ty\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "\nTy\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeStringAccumulatingArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "string_accum_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  text = \"\"\n  i = 0\n  while i < 2\n    text = text + value\n    i = i + 1\n  push items, text\n  return items\nmessage = \"Ty\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "TyTy\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeCallStringAccumulationArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "call_string_accum_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  text = \"\"\n  i = 0\n  while i < 2\n    text = text + trim value\n    i = i + 1\n  push items, text\n  return items\nmessage = \" Ty \"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "TyTy\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeToStringAccumulationArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "to_string_accum_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  text = \"\"\n  count = 1\n  while count <= 2\n    text = text + to_string count\n    count = count + 1\n  push items, text\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "12\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeAliasedStringAccumulatingArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "alias_string_accum_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  text = \"\"\n  part = value\n  i = 0\n  while i < 2\n    text = text + part\n    i = i + 1\n  push items, text\n  return items\nmessage = \"Ty\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "TyTy\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeNamedCounterStepArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "named_counter_step_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  count = 1\n  while count < 5\n    push items, value\n    count = count + 2\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "Tya\nTya\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeLessEqualLoopArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "less_equal_loop_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  count = 1\n  while count <= 5\n    push items, value\n    count = count + 2\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "Tya\nTya\nTya\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeDecrementingLoopArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "decrementing_loop_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  count = 3\n  while count > 0\n    push items, value\n    count = count - 1\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "Tya\nTya\nTya\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeNamedCounterStepContinueArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "named_counter_step_continue_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  count = 1\n  while count < 5\n    if count == 1\n      count = count + 2\n      continue\n    push items, value\n    count = count + 2\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "Tya\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeNotEqualContinueArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "not_equal_continue_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  count = 1\n  while count < 5\n    if count != 3\n      count = count + 2\n      continue\n    push items, value\n    count = count + 2\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "Tya\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeConditionalPushArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "conditional_push_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  count = 1\n  while count < 5\n    if count == 3\n      push items, value\n    push items, \"all\"\n    count = count + 2\n  return items\nmessage = \"hit\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "all\nhit\nall\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeConditionalStringAccumulationArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "conditional_string_accum_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  text = \"\"\n  count = 1\n  while count < 5\n    if count == 3\n      text = text + value\n    text = text + \"!\"\n    count = count + 2\n  push items, text\n  return items\nmessage = \"hit\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "!hit!\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeConditionalCallStringAccumulationArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "conditional_call_string_accum_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  text = \"\"\n  count = 1\n  while count < 5\n    if count == 3\n      text = text + trim value\n    text = text + \"!\"\n    count = count + 2\n  push items, text\n  return items\nmessage = \" hit \"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "!hit!\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeConditionalNotEqualPushArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "conditional_not_equal_push_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  count = 1\n  while count < 5\n    if count != 3\n      push items, value\n    push items, \"all\"\n    count = count + 2\n  return items\nmessage = \"hit\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "hit\nall\nall\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeConditionalGreaterThanPushArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "conditional_greater_than_push_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  count = 1\n  while count < 5\n    if count > 1\n      push items, value\n    push items, \"all\"\n    count = count + 2\n  return items\nmessage = \"hit\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "all\nhit\nall\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeConditionalGreaterEqualPushArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "conditional_greater_equal_push_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  count = 1\n  while count < 5\n    if count >= 3\n      push items, value\n    push items, \"all\"\n    count = count + 2\n  return items\nmessage = \"hit\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "all\nhit\nall\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeConditionalLessEqualPushArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "conditional_less_equal_push_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  count = 1\n  while count < 5\n    if count <= 1\n      push items, value\n    push items, \"all\"\n    count = count + 2\n  return items\nmessage = \"hit\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "hit\nall\nall\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeConditionalMultiStatementArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "conditional_multi_stmt_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  text = \"\"\n  count = 1\n  while count < 5\n    if count >= 3\n      text = text + value\n      push items, text\n    push items, \"all\"\n    count = count + 2\n  return items\nmessage = \"hit\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "all\nhit\nall\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeIfElseArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "if_else_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  count = 1\n  while count < 5\n    if count >= 3\n      push items, value\n    else\n      push items, \"low\"\n    count = count + 2\n  return items\nmessage = \"hit\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "low\nhit\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeIfElseMultiStatementArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "if_else_multi_stmt_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  text = \"\"\n  count = 1\n  while count < 5\n    if count >= 3\n      text = text + value\n      push items, text\n    else\n      text = text + \"low\"\n      push items, text\n    count = count + 2\n  return items\nmessage = \"hit\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "low\nlowhit\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModePostLoopIfElseArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "post_loop_if_else_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  text = \"\"\n  count = 1\n  while count < 5\n    text = text + value\n    count = count + 2\n  if count >= 5\n    push items, text\n  else\n    push items, \"low\"\n  return items\nmessage = \"hi\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "hihi\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModePostLoopIfElseCallPushArrayReturnFunctionProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "post_loop_if_else_call_push_array_return_ast.tya")
	src := "collect = value ->\n  items = []\n  count = 1\n  while count < 5\n    count = count + 2\n  if count >= 5\n    push items, trim value\n  else\n    push items, \"low\"\n  return items\nmessage = \" hi \"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "hi\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsIfElseProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "if_else.tya")
	if err := os.WriteFile(srcPath, []byte("flag = \"off\"\nif flag == \"on\"\n  print \"yes\"\nelse\n  print \"no\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "no\n" {
		t.Fatalf("got %q, want no", out)
	}
}

func TestSelfhostParserLegacyAdapterForScalarAssignAndPrint(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/scalar.tya"
	tokensPath := dir + "/tokens.txt"
	src := "name = \"Tya\"\nage = 20\nready = true\nmissing = nil\nprint name\nprint \"ok\"\nprint 20\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:ASSIGN:name:STRING:Tya
2:INDENT:0
2:ASSIGN:age:INT:20
3:INDENT:0
3:ASSIGN:ready:BOOL:true
4:INDENT:0
4:ASSIGN:missing:NIL:nil
5:INDENT:0
5:PRINT:IDENT:name
6:INDENT:0
6:PRINT:STRING:ok
7:INDENT:0
7:PRINT:INT:20`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserLegacyAdapterForBinaryAndIndexAssign(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/binary_index.tya"
	tokensPath := dir + "/tokens.txt"
	src := "items = [1, 2]\nsum = 1 + 2\ndiff = sum - 1\nlarge = sum > diff\nsame = sum == diff\nfirst = items[0]\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:ASSIGN:items:ARRAY_TWO:INT:1:INT:2
2:INDENT:0
2:ASSIGN:sum:INT_ADD:1:2
3:INDENT:0
3:ASSIGN:diff:INT_SUB:sum:1
4:INDENT:0
4:ASSIGN:large:COMPARE_GT:sum:diff
5:INDENT:0
5:ASSIGN:same:COMPARE_EQ:sum:diff
6:INDENT:0
6:ASSIGN:first:INDEX:items:0`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserAstModeKeepsNestedBinaryPrecedence(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/nested_binary.tya"
	tokensPath := dir + "/nested_binary.ast.tokens"
	src := "sum = 1 + 2 * 3\nprint sum\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	out := string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath))
	want := "AST_ASSIGN:sum:binary(+ int(1) binary(* int(2) int(3)))"
	if !strings.Contains(out, want) {
		t.Fatalf("got:\n%s\nwant AST shape containing:\n%s", out, want)
	}
}

func TestSelfhostCheckerAcceptsParserAstAssignAndPrintStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_stream.tya"
	tokensPath := dir + "/ast_stream.tokens"
	nodesPath := dir + "/ast_stream.nodes"
	src := "sum = 1 + 2 * 3\nprint sum\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)
	out := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if out != "ok\n" {
		t.Fatalf("got %q, want ok", out)
	}
}

func TestSelfhostCheckerRejectsUndefinedAstPrintName(t *testing.T) {
	path := t.TempDir() + "/ast_nodes.txt"
	if err := os.WriteFile(path, []byte("1:AST_PRINT:ident(missing)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path))
	want := "1: undefined variable: missing\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerChecksAstBinaryExpressionNames(t *testing.T) {
	path := t.TempDir() + "/ast_nodes.txt"
	nodes := "1:AST_ASSIGN:known:int(1)\n2:AST_ASSIGN:sum:binary(+ ident(known) ident(missing_add))\n3:AST_PRINT:binary(* ident(sum) ident(missing_print))\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path))
	want := "2: undefined variable: missing_add\n3: undefined variable: missing_print\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerChecksAstMemberAndIndexExpressionNames(t *testing.T) {
	path := t.TempDir() + "/ast_nodes.txt"
	nodes := "1:AST_ASSIGN:user:int(1)\n2:AST_ASSIGN:items:int(1)\n3:AST_ASSIGN:i:int(0)\n4:AST_PRINT:member(user.name)\n5:AST_PRINT:index(items i)\n6:AST_PRINT:member(missing_user.name)\n7:AST_PRINT:index(missing_items missing_index)\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path))
	want := "6: undefined variable: missing_user\n7: undefined variable: missing_items\n7: undefined variable: missing_index\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerChecksAstCallCalleeNames(t *testing.T) {
	path := t.TempDir() + "/ast_nodes.txt"
	nodes := "1:AST_ASSIGN:callee:int(1)\n2:AST_ASSIGN:arg:int(1)\n3:AST_ASSIGN:result:call(callee ident(arg))\n4:AST_PRINT:call(missing_callee ident(arg))\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path))
	want := "4: undefined variable: missing_callee\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostParserAndCheckerHandleAstFuncReturn(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/return.tya"
	tokensPath := dir + "/return.tokens"
	nodesPath := dir + "/return.nodes"
	src := "identity = value ->\n  return value\nreturn missing\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_FUNC:identity:value",
		"AST_RETURN:ident(value)",
		"AST_RETURN:ident(missing)",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	out := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	want := "3: return outside function\n3: undefined variable: missing\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostParserAndCheckerHandleAstMultiReturn(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/return2.tya"
	tokensPath := dir + "/return2.tokens"
	nodesPath := dir + "/return2.nodes"
	src := "pair = left, right ->\n  return left, right\nreturn missing_left, missing\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_FUNC:pair:left:right",
		"AST_RETURN2:ident(left):ident(right)",
		"AST_RETURN2:ident(missing_left):ident(missing)",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	if strings.Contains(nodes, "AST_STMT:return") {
		t.Fatalf("nodes:\n%s\ncontains AST_STMT:return", nodes)
	}
	out := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	want := "3: return outside function\n3: undefined variable: missing_left\n3: undefined variable: missing\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostParserAndCheckerHandleAstMultiAssign(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/multi_assign.tya"
	tokensPath := dir + "/multi_assign.tokens"
	nodesPath := dir + "/multi_assign.nodes"
	src := "pair = \"value\"\nleft, right = pair\nbad, alsoBad = missing\nprint left\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_ASSIGN:pair:string(value)",
		"AST_MULTI_ASSIGN2:left:right:ident(pair)",
		"AST_MULTI_ASSIGN2:bad:alsoBad:ident(missing)",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	out := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	want := "3: invalid binding name: alsoBad\n3: undefined variable: missing\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsCallStmtStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "call_stmt.tya")
	tokensPath := filepath.Join(dir, "call_stmt.tokens")
	astTokensPath := filepath.Join(dir, "call_stmt.ast.tokens")
	nodesPath := filepath.Join(dir, "call_stmt.nodes")
	cPath := filepath.Join(dir, "call_stmt.c")
	binPath := filepath.Join(dir, "call_stmt")
	outPath := filepath.Join(dir, "out.txt")
	src := "path = \"" + outPath + "\"\nname = \" Tya \"\nwrite_file path, trim name\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	wantNode := "AST_CALL_STMT2:write_file:ident(path):call(trim ident(name))"
	if !strings.Contains(nodes, wantNode) {
		t.Fatalf("nodes:\n%s\nmissing %q", nodes, wantNode)
	}
	if strings.Contains(nodes, "AST_STMT:call_stmt") {
		t.Fatalf("nodes:\n%s\ncontains AST_STMT:call_stmt", nodes)
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	run(t, binPath)
	out, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "Tya" {
		t.Fatalf("got %q, want Tya", out)
	}
}

func TestSelfhostAstGeneratedPipelineRunsAstModeCallStmtProgram(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "call_stmt_generated.tya")
	outPath := filepath.Join(dir, "out.txt")
	src := "path = \"" + outPath + "\"\nname = \" Tya \"\nwrite_file path, trim name\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	out := runSelfhostAstGeneratedPipeline(t, dir, bins, srcPath)
	if out != "" {
		t.Fatalf("got stdout %q, want empty", out)
	}
	written, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != "Tya" {
		t.Fatalf("got file %q, want Tya", written)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstDeleteNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "delete.nodes")
	cPath := filepath.Join(dir, "delete.c")
	binPath := filepath.Join(dir, "delete")
	nodes := "1:AST_ASSIGN:user:object1(name string(Tya))\n2:AST_DELETE:user:string(name)\n3:AST_PRINT:member(user.name)\n"
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "nil\n" {
		t.Fatalf("got %q, want nil", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstExitAndPanicNodes(t *testing.T) {
	dir := t.TempDir()
	bins := compileSelfhostAstGeneratedTools(t, dir)

	exitNodesPath := filepath.Join(dir, "exit.nodes")
	exitCPath := filepath.Join(dir, "exit.c")
	exitBinPath := filepath.Join(dir, "exit")
	if err := os.WriteFile(exitNodesPath, []byte("1:AST_EXIT:int(7)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, exitCPath, bins["codegen_c"], exitNodesPath)
	run(t, "cc", exitCPath, "-o", exitBinPath)
	exitCmd := exec.Command(exitBinPath)
	exitOut, exitErr := exitCmd.CombinedOutput()
	if exitErr == nil {
		t.Fatalf("exit command succeeded unexpectedly: %q", exitOut)
	}
	if status, ok := exitErr.(*exec.ExitError); !ok || status.ExitCode() != 7 {
		t.Fatalf("got exit err %v output %q, want status 7", exitErr, exitOut)
	}

	panicNodesPath := filepath.Join(dir, "panic.nodes")
	panicCPath := filepath.Join(dir, "panic.c")
	panicBinPath := filepath.Join(dir, "panic")
	if err := os.WriteFile(panicNodesPath, []byte("1:AST_PANIC:string(bad)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, panicCPath, bins["codegen_c"], panicNodesPath)
	run(t, "cc", panicCPath, "-o", panicBinPath)
	panicCmd := exec.Command(panicBinPath)
	panicOut, panicErr := panicCmd.CombinedOutput()
	if panicErr == nil {
		t.Fatalf("panic command succeeded unexpectedly: %q", panicOut)
	}
	if status, ok := panicErr.(*exec.ExitError); !ok || status.ExitCode() != 1 {
		t.Fatalf("got panic err %v output %q, want status 1", panicErr, panicOut)
	}
	if string(panicOut) != "panic: bad\n" {
		t.Fatalf("got panic output %q", panicOut)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstBreakContinueNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "break_continue.nodes")
	cPath := filepath.Join(dir, "break_continue.c")
	binPath := filepath.Join(dir, "break_continue")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:i:int(0)",
		"2:AST_WHILE:binary(< ident(i) int(1))",
		"3:INDENT:2",
		"3:AST_ASSIGN:i:binary(+ ident(i) int(1))",
		"4:INDENT:2",
		"4:AST_CONTINUE",
		"5:INDENT:2",
		"5:AST_PRINT:string(bad)",
		"6:INDENT:0",
		"7:AST_WHILE:binary(< ident(i) int(2))",
		"8:INDENT:2",
		"8:AST_BREAK",
		"9:INDENT:2",
		"9:AST_PRINT:string(bad)",
		"10:INDENT:0",
		"11:AST_PRINT:string(done)",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "done\n" {
		t.Fatalf("got %q, want done", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstForIndexNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "for_index.nodes")
	cPath := filepath.Join(dir, "for_index.c")
	binPath := filepath.Join(dir, "for_index")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:items:array2(string(A) string(B))",
		"2:AST_FOR_INDEX:item:index:items",
		"3:INDENT:2",
		"3:AST_PRINT:ident(index)",
		"4:INDENT:2",
		"4:AST_PRINT:ident(item)",
		"5:INDENT:0",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "0\nA\n1\nB\n" {
		t.Fatalf("got %q, want indexed loop output", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstIfIntCompareNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "if_int.nodes")
	cPath := filepath.Join(dir, "if_int.c")
	binPath := filepath.Join(dir, "if_int")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:count:int(2)",
		"2:AST_IF:binary(>= ident(count) int(2))",
		"3:INDENT:2",
		"3:AST_PRINT:string(hit)",
		"4:INDENT:0",
		"5:AST_IF:binary(!= ident(count) int(2))",
		"6:INDENT:2",
		"6:AST_PRINT:string(bad)",
		"7:INDENT:0",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "hit\n" {
		t.Fatalf("got %q, want hit", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstAssignIntStepNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "assign_int_step.nodes")
	cPath := filepath.Join(dir, "assign_int_step.c")
	binPath := filepath.Join(dir, "assign_int_step")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:count:int(5)",
		"2:AST_ASSIGN:count:binary(- ident(count) int(2))",
		"3:AST_PRINT:ident(count)",
		"4:AST_ASSIGN:count:binary(+ ident(count) int(4))",
		"5:AST_PRINT:ident(count)",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "3\n7\n" {
		t.Fatalf("got %q, want int step output", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstAssignBinaryIntNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "assign_binary_int.nodes")
	cPath := filepath.Join(dir, "assign_binary_int.c")
	binPath := filepath.Join(dir, "assign_binary_int")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:left:int(5)",
		"2:AST_ASSIGN:sum:binary(+ int(1) int(2))",
		"3:AST_ASSIGN:total:binary(* ident(left) int(3))",
		"4:AST_PRINT:ident(sum)",
		"5:AST_PRINT:ident(total)",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "3\n15\n" {
		t.Fatalf("got %q, want binary int assignment output", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstAssignIdentBinaryIntNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "assign_ident_binary_int.nodes")
	cPath := filepath.Join(dir, "assign_ident_binary_int.c")
	binPath := filepath.Join(dir, "assign_ident_binary_int")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:left:int(5)",
		"2:AST_ASSIGN:right:int(4)",
		"3:AST_ASSIGN:sum:binary(+ ident(left) ident(right))",
		"4:AST_ASSIGN:product:binary(* ident(sum) ident(right))",
		"5:AST_PRINT:ident(sum)",
		"6:AST_PRINT:ident(product)",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "9\n36\n" {
		t.Fatalf("got %q, want ident binary int assignment output", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstAssignStringConcatNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "assign_string_concat.nodes")
	cPath := filepath.Join(dir, "assign_string_concat.c")
	binPath := filepath.Join(dir, "assign_string_concat")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:text:string(T)",
		"2:AST_ASSIGN:value:string(ya)",
		"3:AST_ASSIGN:text:binary(+ ident(text) ident(value))",
		"4:AST_ASSIGN:again:binary(+ ident(text) ident(value))",
		"5:AST_PRINT:ident(text)",
		"6:AST_PRINT:ident(again)",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\nTyaya\n" {
		t.Fatalf("got %q, want string concat assignment output", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstAssignStringLiteralConcatNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "assign_string_literal_concat.nodes")
	cPath := filepath.Join(dir, "assign_string_literal_concat.c")
	binPath := filepath.Join(dir, "assign_string_literal_concat")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:text:string(T)",
		"2:AST_ASSIGN:text:binary(+ ident(text) string(ya))",
		"3:AST_ASSIGN:again:binary(+ string(Go ) ident(text))",
		"4:AST_PRINT:ident(text)",
		"5:AST_PRINT:ident(again)",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\nGo Tya\n" {
		t.Fatalf("got %q, want string literal concat assignment output", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstAssignTrimConcatNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "assign_trim_concat.nodes")
	cPath := filepath.Join(dir, "assign_trim_concat.c")
	binPath := filepath.Join(dir, "assign_trim_concat")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:text:string(T)",
		"2:AST_ASSIGN:raw:string( ya )",
		"3:AST_ASSIGN:text:binary(+ ident(text) call(trim ident(raw)))",
		"4:AST_ASSIGN:again:binary(+ ident(text) call(trim ident(raw)))",
		"5:AST_PRINT:ident(text)",
		"6:AST_PRINT:ident(again)",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\nTyaya\n" {
		t.Fatalf("got %q, want trim concat assignment output", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstAssignToStringConcatNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "assign_to_string_concat.nodes")
	cPath := filepath.Join(dir, "assign_to_string_concat.c")
	binPath := filepath.Join(dir, "assign_to_string_concat")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:text:string(line )",
		"2:AST_ASSIGN:num:int(7)",
		"3:AST_ASSIGN:text:binary(+ ident(text) call(to_string ident(num)))",
		"4:AST_ASSIGN:again:binary(+ ident(text) call(to_string ident(num)))",
		"5:AST_PRINT:ident(text)",
		"6:AST_PRINT:ident(again)",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "line 7\nline 77\n" {
		t.Fatalf("got %q, want to_string concat assignment output", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstAssignNestedBinaryIntNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "assign_nested_binary_int.nodes")
	cPath := filepath.Join(dir, "assign_nested_binary_int.c")
	binPath := filepath.Join(dir, "assign_nested_binary_int")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:sum:binary(+ int(1) binary(* int(2) int(3)))",
		"2:AST_PRINT:ident(sum)",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "7\n" {
		t.Fatalf("got %q, want nested binary int assignment output", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstAssignLiteralAndIdentNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "assign_literal_ident.nodes")
	cPath := filepath.Join(dir, "assign_literal_ident.c")
	binPath := filepath.Join(dir, "assign_literal_ident")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:count:int(3)",
		"2:AST_ASSIGN:label:string(Tya)",
		"3:AST_ASSIGN:ready:bool(true)",
		"4:AST_ASSIGN:missing:nil(nil)",
		"5:AST_ASSIGN:alias_count:ident(count)",
		"6:AST_ASSIGN:alias_label:ident(label)",
		"7:AST_ASSIGN:alias_ready:ident(ready)",
		"8:AST_ASSIGN:alias_missing:ident(missing)",
		"9:AST_PRINT:ident(alias_count)",
		"10:AST_PRINT:ident(alias_label)",
		"11:AST_PRINT:ident(alias_ready)",
		"12:AST_PRINT:ident(alias_missing)",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "3\nTya\ntrue\nnil\n" {
		t.Fatalf("got %q, want literal and ident assignment output", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstPrintExpressionNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "print_expr.nodes")
	cPath := filepath.Join(dir, "print_expr.c")
	binPath := filepath.Join(dir, "print_expr")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:count:int(3)",
		"2:AST_PRINT:int(7)",
		"3:AST_PRINT:bool(false)",
		"4:AST_PRINT:nil(nil)",
		"5:AST_PRINT:binary(+ ident(count) int(4))",
		"6:AST_PRINT:binary(>= ident(count) int(3))",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "7\nfalse\nnil\n7\ntrue\n" {
		t.Fatalf("got %q, want print expression output", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstPrintStringConcatNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "print_string_concat.nodes")
	cPath := filepath.Join(dir, "print_string_concat.c")
	binPath := filepath.Join(dir, "print_string_concat")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:left:string(T)",
		"2:AST_ASSIGN:right:string(ya)",
		"3:AST_PRINT:binary(+ ident(left) ident(right))",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\n" {
		t.Fatalf("got %q, want string concat print output", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstPrintStringLiteralConcatNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "print_string_literal_concat.nodes")
	cPath := filepath.Join(dir, "print_string_literal_concat.c")
	binPath := filepath.Join(dir, "print_string_literal_concat")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:text:string(ya)",
		"2:AST_PRINT:binary(+ string(T) ident(text))",
		"3:AST_PRINT:binary(+ ident(text) string(!))",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\nya!\n" {
		t.Fatalf("got %q, want string literal concat print output", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstPrintTrimConcatNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "print_trim_concat.nodes")
	cPath := filepath.Join(dir, "print_trim_concat.c")
	binPath := filepath.Join(dir, "print_trim_concat")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:text:string(T)",
		"2:AST_ASSIGN:raw:string( ya )",
		"3:AST_PRINT:binary(+ ident(text) call(trim ident(raw)))",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\n" {
		t.Fatalf("got %q, want trim concat print output", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstPrintToStringConcatNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "print_to_string_concat.nodes")
	cPath := filepath.Join(dir, "print_to_string_concat.c")
	binPath := filepath.Join(dir, "print_to_string_concat")
	nodes := strings.Join([]string{
		"1:AST_ASSIGN:text:string(line )",
		"2:AST_ASSIGN:num:int(7)",
		"3:AST_PRINT:binary(+ ident(text) call(to_string ident(num)))",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "line 7\n" {
		t.Fatalf("got %q, want to_string concat print output", out)
	}
}

func TestSelfhostAstGeneratedCodegenRunsAstPrintNestedBinaryNodes(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "print_nested_binary.nodes")
	cPath := filepath.Join(dir, "print_nested_binary.c")
	binPath := filepath.Join(dir, "print_nested_binary")
	nodes := strings.Join([]string{
		"1:AST_PRINT:binary(+ int(1) binary(* int(2) int(3)))",
		"2:AST_PRINT:binary(== int(7) binary(+ int(3) int(4)))",
		"",
	}, "\n")
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	bins := compileSelfhostAstGeneratedTools(t, dir)
	runToFile(t, cPath, bins["codegen_c"], nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "7\ntrue\n" {
		t.Fatalf("got %q, want nested binary print output", out)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsMultiReturnCallStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "multi_return.tya")
	tokensPath := filepath.Join(dir, "multi_return.tokens")
	astTokensPath := filepath.Join(dir, "multi_return.ast.tokens")
	nodesPath := filepath.Join(dir, "multi_return.nodes")
	cPath := filepath.Join(dir, "multi_return.c")
	binPath := filepath.Join(dir, "multi_return")
	src := "parse_user = text ->\n  return nil, error \"empty user\"\n  return { name: text }, nil\nuser, err = parse_user \"komagata\"\nprint user\nprint err\nmissing, err = parse_user \"\"\nprint missing\nprint err\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_RETURN2:nil(nil):call(error string(empty user))",
		"AST_RETURN2:object1(name ident(text)):nil(nil)",
		"AST_MULTI_ASSIGN2:user:err:call(parse_user string(komagata))",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	want := "komagata\n\n\nempty user\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsMultipleReturnExampleStream(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "multiple_return.tokens")
	astTokensPath := filepath.Join(dir, "multiple_return.ast.tokens")
	nodesPath := filepath.Join(dir, "multiple_return.nodes")
	cPath := filepath.Join(dir, "multiple_return.c")
	binPath := filepath.Join(dir, "multiple_return")
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "examples/multiple_return.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_PRINT:index(user string(name))",
		"AST_PRINT:index(missing string(name))",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	want := "komagata\nempty user\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsLogicExampleStream(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "logic.tokens")
	astTokensPath := filepath.Join(dir, "logic.ast.tokens")
	nodesPath := filepath.Join(dir, "logic.nodes")
	cPath := filepath.Join(dir, "logic.c")
	binPath := filepath.Join(dir, "logic")
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "examples/logic.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_IF:binary(and binary(>= ident(age) int(20)) binary(== ident(name) string(komagata)))",
		"AST_PRINT:binary(OR nil(nil) string(anonymous))",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	generated, err := os.ReadFile(cPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(generated), " and ") || strings.Contains(string(generated), " OR ") {
		t.Fatalf("generated C still contains Tya boolean operator:\n%s", generated)
	}
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if !strings.Contains(out, "match\n") {
		t.Fatalf("got %q, missing match", out)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsClassicArraySumStream(t *testing.T) {
	dir := t.TempDir()
	tokensPath := filepath.Join(dir, "array_sum.tokens")
	astTokensPath := filepath.Join(dir, "array_sum.ast.tokens")
	nodesPath := filepath.Join(dir, "array_sum.nodes")
	cPath := filepath.Join(dir, "array_sum.c")
	binPath := filepath.Join(dir, "array_sum")
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", "examples/classic/array_sum.tya")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_ASSIGN:items:array3(int(1) int(2) int(3))",
		"AST_FOR:item:items",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	generated, err := os.ReadFile(cPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "long items_0 = 1;") {
		t.Fatalf("generated C does not use int array slots:\n%s", generated)
	}
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "6\n" {
		t.Fatalf("got %q, want 6", out)
	}
}

func TestSelfhostParserAstCallArgumentsKeepMemberAndIndexShape(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/call_args.tya"
	tokensPath := dir + "/call_args.tokens"
	src := "result = show(user.name)\nresult2 = show(items[0])\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	out := string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath))
	required := []string{
		"AST_ASSIGN:result:call(show member(user.name))",
		"AST_ASSIGN:result2:call(show index(items 0))",
	}
	for _, want := range required {
		if !strings.Contains(out, want) {
			t.Fatalf("got:\n%s\nwant AST shape containing:\n%s", out, want)
		}
	}
}

func TestSelfhostParserLegacyAdapterForPrintIndex(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/print_index.tya"
	tokensPath := dir + "/tokens.txt"
	src := "items = [\"A\", \"B\"]\nprint items[1]\nprint \"tya\"[1]\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:ASSIGN:items:ARRAY_TWO:STRING:A:STRING:B
2:INDENT:0
2:PRINT_INDEX:IDENT:items:1
3:INDENT:0
3:PRINT_INDEX:STRING:tya:1`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserPostfixAdapterForMemberIndexChain(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/postfix_chain.tya"
	tokensPath := dir + "/tokens.txt"
	src := "print user.name[0]\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:PRINT_INDEX:MEMBER:user.name:0`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserAssignmentUsesExpressionResult(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/assign_postfix_chain.tya"
	tokensPath := dir + "/tokens.txt"
	src := "value = user.name[0]\nprint value\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:ASSIGN:value:INDEX:user.name:0
2:INDENT:0
2:PRINT:IDENT:value`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserBareCallArgumentUsesPostfixExpression(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/bare_call_postfix_arg.tya"
	tokensPath := dir + "/tokens.txt"
	src := "print show user.name\nprint show users[0]\nresult = show user.name\nresult2 = show users[0]\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:PRINT_CALL1:show:user.name
2:INDENT:0
2:PRINT_CALL1_INDEX:show:users:0
3:INDENT:0
3:ASSIGN:result:CALL1:show:user.name
4:INDENT:0
4:ASSIGN:result2:CALL1_INDEX:show:users:0`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserParenCallArgumentUsesPostfixExpression(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/paren_call_postfix_arg.tya"
	tokensPath := dir + "/tokens.txt"
	src := "print show(user.name)\nprint show(users[0])\nresult = show(user.name)\nresult2 = show(users[0])\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:PRINT_CALL1:show:user.name
2:INDENT:0
2:PRINT_CALL1_INDEX:show:users:0
3:INDENT:0
3:ASSIGN:result:CALL1:show:user.name
4:INDENT:0
4:ASSIGN:result2:CALL1_INDEX:show:users:0`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserPushAndDeleteUseExpressionResult(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/push_delete_postfix.tya"
	tokensPath := dir + "/tokens.txt"
	src := "items = []\npush items, user.name[0]\ndelete cache, user.name\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:ASSIGN:items:ARRAY_EMPTY:
2:INDENT:0
2:PUSH:items:INDEX:user.name:0
3:INDENT:0
3:DELETE:cache:MEMBER:user.name`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserReturnExitAndPanicUseExpressionResult(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/return_exit_panic_postfix.tya"
	tokensPath := dir + "/tokens.txt"
	src := "return user.name[0]\nexit user.code\npanic errors[0]\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:RETURN:INDEX:user.name:0
2:INDENT:0
2:EXIT:MEMBER:user.code
3:INDENT:0
3:PANIC:INDEX:errors:0`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserConditionsUseExpressionResultForPostfix(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/condition_postfix.tya"
	tokensPath := dir + "/tokens.txt"
	src := "if user.active\n  print user.name\nwhile items[0]\n  break\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:IF:MEMBER:user.active
2:INDENT:2
2:PRINT_MEMBER:user:name
3:INDENT:0
3:WHILE:INDEX:items:0
4:INDENT:2
4:BREAK`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserForCollectionUsesExpressionResultForMember(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/for_member_collection.tya"
	tokensPath := dir + "/tokens.txt"
	src := "for user in groups.active\n  print user\nfor user, index in groups.active\n  print index\nfor user in groups[0]\n  print user\nfor user, index in groups[0]\n  print index\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:FOR:user:groups.active
2:INDENT:2
2:PRINT:IDENT:user
3:INDENT:0
3:FOR_INDEX:user:index:groups.active
4:INDENT:2
4:PRINT:IDENT:index
5:INDENT:0
5:FOR:user:groups[0]
6:INDENT:2
6:PRINT:IDENT:user
7:INDENT:0
7:FOR_INDEX:user:index:groups[0]
8:INDENT:2
8:PRINT:IDENT:index`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserLegacyAdapterForPrintCalls(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/print_calls.tya"
	tokensPath := dir + "/tokens.txt"
	src := "message = \"Tya\"\nprint len message\nprint contains message, \"T\"\nprint replace message, \"T\", message\nprint text_of(tokens[i])\nprint text_of(tokens[i + 3])\nprint to_string(1 + 2)\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:ASSIGN:message:STRING:Tya
2:INDENT:0
2:PRINT_CALL1:len:message
3:INDENT:0
3:PRINT_CALL2:contains:message:STRING:T
4:INDENT:0
4:PRINT_CALL3:replace:message:STRING:T:message
5:INDENT:0
5:PRINT_CALL1_INDEX:text_of:tokens:i
6:INDENT:0
6:PRINT_CALL1_INDEX_BINARY:text_of:tokens:i:+:3
7:INDENT:0
7:PRINT_CALL1_EXPR:to_string:INT_ADD:1:2`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserLegacyAdapterForPrintMemberAndOperators(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/print_member_ops.tya"
	tokensPath := dir + "/tokens.txt"
	src := "user = { name: \"Tya\" }\nprint user.name\nprint 1 == 1\nprint nil or \"anon\"\nprint not false\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:ASSIGN:user:OBJECT_ONE:name:STRING:Tya
2:INDENT:0
2:PRINT_MEMBER:user:name
3:INDENT:0
3:PRINT_COMPARE_EQ:INT:1:INT:1
4:INDENT:0
4:PRINT_OR:NIL:nil:STRING:anon
5:INDENT:0
5:PRINT_NOT:BOOL:false`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserLegacyAdapterForAssignCalls(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/assign_calls.tya"
	tokensPath := dir + "/tokens.txt"
	src := "message = trim(\" Tya \")\nparts = split(message, \"y\")\nreplaced = replace(message, \"T\", message)\nechoed = echo message\ntried = try parse(message)\nkind = kind_of(tokens[i])\ntext = text_of(tokens[i + 3])\nadded = to_string(1 + 2)\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:ASSIGN:message:CALL1:trim: Tya 
2:INDENT:0
2:ASSIGN:parts:CALL2:split:message:y
3:INDENT:0
3:ASSIGN:replaced:CALL3:replace:message:STRING:T:message
4:INDENT:0
4:ASSIGN:echoed:CALL1:echo:message
5:INDENT:0
5:ASSIGN:tried:TRY_CALL1:parse:message
6:INDENT:0
6:ASSIGN:kind:CALL1_INDEX:kind_of:tokens:i
7:INDENT:0
7:ASSIGN:text:CALL1_INDEX_BINARY:text_of:tokens:i:+:3
8:INDENT:0
8:ASSIGN:added:CALL1_EXPR:to_string:INT_ADD:1:2`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserLegacyAdapterKeepsBareCallLimit(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/bare_call_limit.tya"
	tokensPath := dir + "/tokens.txt"
	src := "emit = line, kind, text, col ->\n  return line\ntoken = emit line, \"INDENT\", to_string(line_spaces), 1\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:FUNC4:emit:line:kind:text:col
2:INDENT:2
2:RETURN:IDENT:line
3:INDENT:0
3:ASSIGN:token:CALL3:emit:line:STRING:INDENT:to_string`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserLegacyAdapterForAssignBoolOperators(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/assign_bool_ops.tya"
	tokensPath := dir + "/tokens.txt"
	src := "enabled = true\ninverse = not enabled\nboth = enabled and inverse\neither = enabled or inverse\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:ASSIGN:enabled:BOOL:true
2:INDENT:0
2:ASSIGN:inverse:BOOL_NOT:enabled
3:INDENT:0
3:ASSIGN:both:BOOL_AND:enabled:inverse
4:INDENT:0
4:ASSIGN:either:BOOL_OR:enabled:inverse`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserLegacyAdapterForGroupedBinaryAssign(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/grouped_binary.tya"
	tokensPath := dir + "/tokens.txt"
	src := "sum = (1 + 2)\ndiff = (sum - 1)\nmodded = (sum % 2)\nlarge = (sum > diff)\nsame = (sum == diff)\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:ASSIGN:sum:INT_ADD:1:2
2:INDENT:0
2:ASSIGN:diff:INT_SUB:sum:1
3:INDENT:0
3:ASSIGN:modded:INT_MOD:sum:2
4:INDENT:0
4:ASSIGN:large:COMPARE_GT:sum:diff
5:INDENT:0
5:ASSIGN:same:COMPARE_EQ:sum:diff`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserLegacyAdapterForArrayAndObjectAssign(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/array_object.tya"
	tokensPath := dir + "/tokens.txt"
	src := "empty = []\none = [1]\ntwo = [1, \"Tya\"]\nthree = [1, 2, 3]\nfour = [1, 2, 3, 4]\nuser = { name: \"Tya\" }\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:ASSIGN:empty:ARRAY_EMPTY:
2:INDENT:0
2:ASSIGN:one:ARRAY_ONE:INT:1
3:INDENT:0
3:ASSIGN:two:ARRAY_TWO:INT:1:STRING:Tya
4:INDENT:0
4:ASSIGN:three:ARRAY_THREE:INT:1:INT:2:INT:3
5:INDENT:0
5:ASSIGN:four:ARRAY_FOUR:INT:1:INT:2:INT:3:INT:4
6:INDENT:0
6:ASSIGN:user:OBJECT_ONE:name:STRING:Tya`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserLegacyAdapterForSimpleConditions(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/conditions.tya"
	tokensPath := dir + "/tokens.txt"
	src := "if ready\n  print \"yes\"\nif count == 1\n  print count\nif count != 2\n  print count\nif count >= 3\n  print count\nif count > 4\n  print count\nif count <= 5\n  print count\nwhile running\n  break\nwhile count < 10\n  break\nwhile count != 11\n  break\nwhile count >= 12\n  break\nwhile count > 13\n  break\nwhile count <= 14\n  break\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:IF:IDENT:ready
2:INDENT:2
2:PRINT:STRING:yes
3:INDENT:0
3:IF_COMPARE_EQ:IDENT:count:INT:1
4:INDENT:2
4:PRINT:IDENT:count
5:INDENT:0
5:IF_COMPARE_NE:IDENT:count:INT:2
6:INDENT:2
6:PRINT:IDENT:count
7:INDENT:0
7:IF_COMPARE_GE:IDENT:count:INT:3
8:INDENT:2
8:PRINT:IDENT:count
9:INDENT:0
9:IF_COMPARE_GT:IDENT:count:INT:4
10:INDENT:2
10:PRINT:IDENT:count
11:INDENT:0
11:IF_COMPARE_LE:IDENT:count:INT:5
12:INDENT:2
12:PRINT:IDENT:count
13:INDENT:0
13:WHILE:IDENT:running
14:INDENT:2
14:BREAK
15:INDENT:0
15:WHILE_COMPARE_LT:IDENT:count:INT:10
16:INDENT:2
16:BREAK
17:INDENT:0
17:WHILE_COMPARE_NE:IDENT:count:INT:11
18:INDENT:2
18:BREAK
19:INDENT:0
19:WHILE_COMPARE_GE:IDENT:count:INT:12
20:INDENT:2
20:BREAK
21:INDENT:0
21:WHILE_COMPARE_GT:IDENT:count:INT:13
22:INDENT:2
22:BREAK
23:INDENT:0
23:WHILE_COMPARE_LE:IDENT:count:INT:14
24:INDENT:2
24:BREAK`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserLegacyAdapterForSpecialConditions(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/special_conditions.tya"
	tokensPath := dir + "/tokens.txt"
	src := "if not ready\n  print ready\nif not contains(name, \"T\")\n  print name\nif not contains(name) and not starts_with(name, \"T\")\n  print name\nif count % 2 == 0\n  print count\nif text_of(token) == \"if\" and kind_of(token) == \"IDENT\"\n  print token\nif text_of(token) == \"if\" and kind_of(token) != \"IDENT\"\n  print token\nif len(items) < limit\n  print limit\nif len(items) > limit\n  print limit\nif contains(name)\n  print name\nif left == right or left2 == right2\n  print left\nwhile i < len(items)\n  print i\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:IF_NOT:IDENT:ready
2:INDENT:2
2:PRINT:IDENT:ready
3:INDENT:0
3:IF_NOT_CALL2:contains:name:T
4:INDENT:2
4:PRINT:IDENT:name
5:INDENT:0
5:IF_NOT_CALL_AND_NOT_CALL:contains:name:starts_with:name:T
6:INDENT:2
6:PRINT:IDENT:name
7:INDENT:0
7:IF_INT_MOD_EQ:IDENT:count:INT:2:INT:0
8:INDENT:2
8:PRINT:IDENT:count
9:INDENT:0
9:IF_CALL_EQ_AND_CALL_EQ:text_of:token:STRING:if:kind_of:token:STRING:IDENT
10:INDENT:2
10:PRINT:IDENT:token
11:INDENT:0
11:IF_CALL_EQ_AND_CALL_NE:text_of:token:STRING:if:kind_of:token:STRING:IDENT
12:INDENT:2
12:PRINT:IDENT:token
13:INDENT:0
13:IF_CALL_LT:len:items:limit
14:INDENT:2
14:PRINT:IDENT:limit
15:INDENT:0
15:IF_CALL_GT:len:items:limit
16:INDENT:2
16:PRINT:IDENT:limit
17:INDENT:0
17:IF_CALL1:contains:name
18:INDENT:2
18:PRINT:IDENT:name
19:INDENT:0
19:IF_COMPARE_OR:IDENT:left:IDENT:right:IDENT:left2:IDENT:right2
20:INDENT:2
20:PRINT:IDENT:left
21:INDENT:0
21:WHILE_LT_CALL:i:len:items
22:INDENT:2
22:PRINT:IDENT:i`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserLegacyAdapterForSimpleControlStatements(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/control_statements.tya"
	tokensPath := dir + "/tokens.txt"
	src := "items = []\nname = \" Tya \"\npush items, \"Tya\"\npush items, trim name\ndelete user, \"name\"\nexit 1\nexit to_int code\npanic \"bad\"\npanic error \"bad\"\nwrite_file path, trim name\nidentity = value ->\n  return value\npair = left, right ->\n  return left, right\nfirst = items ->\n  return items[0]\nclean = value ->\n  return trim value\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:ASSIGN:items:ARRAY_EMPTY:
2:INDENT:0
2:ASSIGN:name:STRING: Tya 
3:INDENT:0
3:PUSH:items:STRING:Tya
4:INDENT:0
4:PUSH:items:CALL1:trim:name
5:INDENT:0
5:DELETE:user:STRING:name
6:INDENT:0
6:EXIT:INT:1
7:INDENT:0
7:EXIT:CALL1:to_int:code
8:INDENT:0
8:PANIC:STRING:bad
9:INDENT:0
9:PANIC:CALL1:error:bad
10:INDENT:0
10:CALL_STMT2:write_file:path:CALL1:trim:name
11:INDENT:0
11:FUNC:identity:value
12:INDENT:2
12:RETURN:IDENT:value
13:INDENT:0
13:FUNC2:pair:left:right
14:INDENT:2
14:RETURN2:IDENT:left:IDENT:right
15:INDENT:0
15:FUNC:first:items
16:INDENT:2
16:RETURN:INDEX:items:0
17:INDENT:0
17:FUNC:clean:value
18:INDENT:2
18:RETURN_CALL2:trim:IDENT:value`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserLegacyAdapterForSpecialReturns(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/special_returns.tya"
	tokensPath := dir + "/tokens.txt"
	src := "parse_user = text ->\n  return nil, error \"empty user\"\n  return { name: text }, nil\npair = right -> result, right\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:FUNC:parse_user:text
2:INDENT:2
2:RETURN2_CALL1:NIL:nil:error:STRING:empty user
3:INDENT:2
3:RETURN2_OBJECT_NIL:name:IDENT:text
4:INDENT:0
4:FUNC:pair:right
4:RETURN_CALL2:result:IDENT:right`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserLegacyAdapterForForCallAndMultiAssign(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/for_call_multi.tya"
	tokensPath := dir + "/tokens.txt"
	src := "for item in items\n  print item\nfor item, index in items\n  print index\nfor key in user\n  print key\nfor role, key in user\n  print role\nwrite_file path, \"ok\"\nleft, right = pair\nuser, err = parse_user \"komagata\"\nvalue, err = parse_user(\"komagata\")\ntrimmed, err = parse_user message\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:FOR:item:items
2:INDENT:2
2:PRINT:IDENT:item
3:INDENT:0
3:FOR_INDEX:item:index:items
4:INDENT:2
4:PRINT:IDENT:index
5:INDENT:0
5:FOR:key:user
6:INDENT:2
6:PRINT:IDENT:key
7:INDENT:0
7:FOR_INDEX:role:key:user
8:INDENT:2
8:PRINT:IDENT:role
9:INDENT:0
9:CALL_STMT2:write_file:path:STRING:ok
10:INDENT:0
10:MULTI_ASSIGN2:left:right:IDENT:pair
11:INDENT:0
11:MULTI_ASSIGN2_CALL1:user:err:parse_user:STRING:komagata
12:INDENT:0
12:MULTI_ASSIGN2_CALL1:value:err:parse_user:STRING:komagata
13:INDENT:0
13:MULTI_ASSIGN2_CALL1:trimmed:err:parse_user:IDENT:message`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserLegacyAdapterForFunctions(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/functions.tya"
	tokensPath := dir + "/tokens.txt"
	src := "one = a ->\n  return a\ntwo = a, b ->\n  return a\nthree = a, b, c ->\n  return a\nfour = a, b, c, d ->\n  return a\ninline = value -> value\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:FUNC:one:a
2:INDENT:2
2:RETURN:IDENT:a
3:INDENT:0
3:FUNC2:two:a:b
4:INDENT:2
4:RETURN:IDENT:a
5:INDENT:0
5:FUNC3:three:a:b:c
6:INDENT:2
6:RETURN:IDENT:a
7:INDENT:0
7:FUNC4:four:a:b:c:d
8:INDENT:2
8:RETURN:IDENT:a
9:INDENT:0
9:FUNC:inline:value
9:RETURN:IDENT:value`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserLegacyAdapterForCallIndexAssign(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/call_index_assign.tya"
	tokensPath := dir + "/tokens.txt"
	src := "first = args()[0]\nsource = read_file args()[0]\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:ASSIGN:first:CALL0_INDEX:args:0
2:INDENT:0
2:ASSIGN:source:CALL1_CALL0_INDEX:read_file:args:0`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostParserLegacyAdapterForSpecialAssigns(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/special_assigns.tya"
	tokensPath := dir + "/tokens.txt"
	src := "score = (left + right) * factor\ndoubled = map items, item -> item * 2\nevens = filter items, item -> item % 2 == 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	out := strings.TrimSpace(string(run(t, "go", "run", "./cmd/tya", "selfhost/parser.tya", tokensPath)))
	want := strings.TrimSpace(`1:INDENT:0
1:ASSIGN:score:INT_MUL_ADD:left:right:factor
2:INDENT:0
2:ASSIGN:doubled:MAP_MUL:items:item:2
3:INDENT:0
3:ASSIGN:evens:CALL2_FUNC_MOD_EQ:filter:items:item:2:0`)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSelfhostCheckerUsesCallHelpers(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "selfhost", "checker.tya"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	required := []string{
		"check_assign_call1 = errors, names, node ->",
		"check_assign_call2 = errors, names, node ->",
		"check_assign_call3 = errors, names, node ->",
		"check_print_call1 = errors, names, node ->",
		"check_print_call2 = errors, names, node ->",
		"check_print_call3 = errors, names, node ->",
		"check_assign_call1 errors, names, node",
		"check_assign_call2 errors, names, node",
		"check_assign_call3 errors, names, node",
		"check_print_call1 errors, names, node",
		"check_print_call2 errors, names, node",
		"check_print_call3 errors, names, node",
	}
	for _, marker := range required {
		if !strings.Contains(source, marker) {
			t.Fatalf("selfhost checker is missing call1 helper marker %q", marker)
		}
	}
}

func TestSelfhostGeneratedParserRecognizesTwoArgFunctions(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "selfhost", "codegen_c.tya"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	required := []string{
		`token_is(tokens[i + 3], \"SYMBOL\", \",\")`,
		`token_is(tokens[i + 5], \"ARROW\", \"->\")`,
		`snprintf(node, sizeof(node), \"%s:FUNC2:%s:%s:%s\"`,
		"skip_function_body = 1;",
	}
	for _, marker := range required {
		if !strings.Contains(source, marker) {
			t.Fatalf("selfhost generated parser is missing two-arg function marker %q", marker)
		}
	}
}

func TestSelfhostGeneratedParserUsesAstAtomHelper(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "selfhost", "codegen_c.tya"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	required := []string{
		"static int ast_atom_from_token(const char *token, char *out, size_t out_size)",
		"static int ast_index_from_tokens(char **tokens, long tokens_len, long start, char *out, size_t out_size)",
		"static int ast_call_from_tokens(char **tokens, long tokens_len, long start, char *out, size_t out_size)",
		"static int ast_expr_from_tokens(char **tokens, long tokens_len, long start, char *out, size_t out_size)",
		"static void ast_binary_condition(char *out, size_t out_size",
		"static void ast_binary_chain3(char *out, size_t out_size",
		"static int ast_bool_chain3_from_tokens(char **tokens, long tokens_len, long start",
		"static int ast_bool_expr_from_tokens(char **tokens, long tokens_len, long start",
		"static int ast_binary_compare_from_tokens(char **tokens, long tokens_len, long start",
		"static int ast_bool_expr_candidate_at(char **tokens, long tokens_len, long start)",
		"static void ast_compound_while_condition(char *out, size_t out_size",
		"static int ast_compound_while_from_tokens(char **tokens, long tokens_len, long start",
		"static int ast_return_from_tokens(char **tokens, long tokens_len, long start",
		"static int ast_assign_from_tokens(char **tokens, long tokens_len, long start",
		"static int ast_push_from_tokens(char **tokens, long tokens_len, long start",
		"static int ast_call_stmt2_from_tokens(char **tokens, long tokens_len, long start",
		"static int ast_for_from_tokens(char **tokens, long tokens_len, long start",
		"static int ast_control_from_tokens(char **tokens, long tokens_len, long start",
		"static int ast_effect_from_tokens(char **tokens, long tokens_len, long start",
		"static int ast_simple_if_from_tokens(char **tokens, long tokens_len, long start",
		"static int ast_simple_while_from_tokens(char **tokens, long tokens_len, long start",
		"static int append_ast_assign_from_tokens(char ***nodes, long *len, char **tokens, long tokens_len, long start)",
		"static int ast_print_from_tokens(char **tokens, long tokens_len, long start",
		`snprintf(out, out_size, \"call(%s %s %s)\", callee, left_expr, right_expr)`,
		`ast_bool_expr_from_tokens(tokens, tokens_len, i, expr, sizeof(expr))`,
		`ast_bool_expr_candidate_at(tokens, tokens_len, i)`,
		`ast_expr_from_tokens(tokens, tokens_len, i + 2, left_ast, sizeof(left_ast))`,
		`ast_expr_from_tokens(tokens, tokens_len, i + 4, right_ast, sizeof(right_ast))`,
		`ast_binary_condition(out, out_size, op, left_expr, right_expr)`,
		`ast_expr_from_tokens(tokens, tokens_len, start + 1, left_expr, sizeof(left_expr))`,
		`ast_expr_from_tokens(tokens, tokens_len, start + 3, right_expr, sizeof(right_expr))`,
		`ast_expr_from_tokens(tokens, tokens_len, start + 8, index_expr, sizeof(index_expr))`,
		`ast_expr_from_tokens(tokens, tokens_len, start + 13, value_expr, sizeof(value_expr))`,
		`ast_expr_from_tokens(tokens, tokens_len, start + 5, index_expr, sizeof(index_expr))`,
		`ast_expr_from_tokens(tokens, tokens_len, start + 10, value_expr, sizeof(value_expr))`,
		`ast_return_from_tokens(tokens, tokens_len, i, expr, sizeof(expr))`,
		`ast_assign_from_tokens(tokens, tokens_len, i, ast_assign, sizeof(ast_assign))`,
		`append_ast_assign_from_tokens(&nodes, &len, tokens, tokens_len, i)`,
		`ast_print_from_tokens(tokens, tokens_len, i, ast_print, sizeof(ast_print))`,
		`ast_push_from_tokens(tokens, tokens_len, i, ast_push, sizeof(ast_push))`,
		`ast_call_stmt2_from_tokens(tokens, tokens_len, i, ast_call, sizeof(ast_call))`,
		`ast_for_from_tokens(tokens, tokens_len, i, ast_kind, sizeof(ast_kind), ast_for, sizeof(ast_for))`,
		`ast_control_from_tokens(tokens, tokens_len, i, ast_control, sizeof(ast_control))`,
		`ast_effect_from_tokens(tokens, tokens_len, i, ast_kind, sizeof(ast_kind), ast_effect, sizeof(ast_effect))`,
		`ast_compound_while_from_tokens(tokens, tokens_len, i, condition, sizeof(condition))`,
		`ast_simple_if_from_tokens(tokens, tokens_len, i, condition, sizeof(condition))`,
		`ast_simple_while_from_tokens(tokens, tokens_len, i, condition, sizeof(condition))`,
		`snprintf(node, sizeof(node), \"%s:AST_ASSIGN:%s:binary(%s %s %s)\"`,
	}
	for _, marker := range required {
		if !strings.Contains(source, marker) {
			t.Fatalf("selfhost generated parser is missing AST atom helper marker %q", marker)
		}
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
	nodes := "1:FUNC:identity:value\n2:INDENT:2\n2:RETURN:IDENT:value\n3:INDENT:0\n3:ASSIGN:message:STRING: Tya \n4:ASSIGN:trimmed:CALL1:trim:message\n5:ASSIGN:result:CALL1:identity:trimmed\n6:PRINT_CALL1:identity:trimmed\n7:ASSIGN:replaced:CALL3:replace:trimmed:STRING:T:trimmed\n8:PRINT_CALL3:replace:trimmed:STRING:T:trimmed\n9:PRINT_CALL2:contains:trimmed:STRING:T\n10:PRINT_CALL2:starts_with:trimmed:STRING:T\n11:PRINT_CALL2:ends_with:trimmed:STRING:a\n12:PRINT_CALL1:len:trimmed\n13:ASSIGN:user:OBJECT_ONE:name:IDENT:trimmed\n14:PRINT_MEMBER:user:name\n15:ASSIGN:source:CALL1_CALL0_INDEX:read_file:args:0\n16:ASSIGN:tokens:CALL1:lex:source\n17:ASSIGN:lines:CALL2:split:source:\\n\n18:ASSIGN:nodes:CALL1:parse:tokens\n19:ASSIGN:items:ARRAY_EMPTY:\n20:PUSH:items:IDENT:trimmed\n21:ASSIGN:first:INDEX:items:0\n22:ASSIGN:names:ARRAY_TWO:STRING:Ada:STRING:Tya\n23:ASSIGN:selected:STRING:default\n24:ASSIGN:selected:INDEX:names:1\n25:FOR:token:tokens\n26:INDENT:2\n26:PRINT:IDENT:token\n"
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
		t.Fatalf("generated C missing print starts_with lowering:\n%s", out)
	}
	if !strings.Contains(out, "puts(ends_with_text(trimmed, \"a\") ? \"true\" : \"false\");") {
		t.Fatalf("generated C missing print ends_with lowering:\n%s", out)
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
		t.Fatalf("generated C missing read_file args()[0] lowering:\n%s", out)
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
	if !strings.Contains(out, "const char *first = items[0];") {
		t.Fatalf("generated C missing dynamic array index lowering:\n%s", out)
	}
	if !strings.Contains(out, "const char *selected = \"default\";") || !strings.Contains(out, "selected = names_1;") {
		t.Fatalf("generated C missing static array index reassignment lowering:\n%s", out)
	}
	if strings.Contains(out, "const char *selected = names_1;") {
		t.Fatalf("generated C redeclared static array index reassignment:\n%s", out)
	}
	if strings.Contains(out, "/* func identity") {
		t.Fatalf("generated C kept function comment:\n%s", out)
	}
}

func TestSelfhostCodegenRunsAstAssignAndPrintStream(t *testing.T) {
	dir := t.TempDir()
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_stream.c"
	binPath := dir + "/ast_stream"
	nodes := "1:AST_ASSIGN:count:int(3)\n2:AST_ASSIGN:label:string(Tya)\n3:AST_ASSIGN:ready:bool(true)\n4:AST_ASSIGN:missing:nil(nil)\n5:AST_ASSIGN:alias:ident(count)\n6:AST_PRINT:ident(alias)\n7:AST_PRINT:ident(label)\n8:AST_PRINT:ident(ready)\n9:AST_PRINT:ident(missing)\n10:AST_PRINT:int(7)\n11:AST_PRINT:string(done)\n12:AST_PRINT:bool(false)\n"
	if err := os.WriteFile(nodesPath, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	want := "3\nTya\ntrue\nnil\n7\ndone\nfalse\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsFunctionCallStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_func_call.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_func_call.c"
	binPath := dir + "/ast_func_call"
	src := "identity = value ->\n  return value\nmessage = \"Tya\"\nresult = identity(message)\nprint result\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_FUNC:identity:value",
		"AST_RETURN:ident(value)",
		"AST_ASSIGN:result:call(identity ident(message))",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\n" {
		t.Fatalf("got %q, want Tya", out)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsConditionalStringFunctionStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_func_case.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_func_case.c"
	binPath := dir + "/ast_func_case"
	src := "escape = char ->\n  if char == \"n\"\n    return \"\\n\"\n  if char == \"t\"\n    return \"\\t\"\n  return char\nprint escape(\"n\")\nprint escape(\"x\")\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_FUNC:escape:char",
		"AST_IF:binary(== ident(char) string(n))",
		"AST_RETURN:string(\\n)",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "\n\nx\n" {
		t.Fatalf("got %q, want escaped newline then x", out)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsArrayReturnFunctionStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_array_func.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_array_func.c"
	binPath := dir + "/ast_array_func"
	src := "collect = value ->\n  items = []\n  push items, value\n  push items, \"done\"\n  return items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_FUNC:collect:value",
		"AST_ASSIGN:items:array0()",
		"AST_RETURN:ident(items)",
		"AST_ASSIGN:items:call(collect ident(message))",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\ndone\n" {
		t.Fatalf("got %q, want array return output", out)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsImplicitArrayReturnFunctionStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_implicit_array_func.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_implicit_array_func.c"
	binPath := dir + "/ast_implicit_array_func"
	src := "collect = value ->\n  items = []\n  push items, value\n  push items, \"done\"\n  items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_FUNC:collect:value",
		"AST_ASSIGN:items:array0()",
		"AST_EXPR:ident(items)",
		"AST_ASSIGN:items:call(collect ident(message))",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\ndone\n" {
		t.Fatalf("got %q, want implicit array return output", out)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsAliasedArrayReturnFunctionStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_aliased_array_func.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_aliased_array_func.c"
	binPath := dir + "/ast_aliased_array_func"
	src := "collect = value ->\n  items = []\n  item = value\n  push items, item\n  items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_ASSIGN:item:ident(value)",
		"AST_PUSH:items:ident(item)",
		"AST_EXPR:ident(items)",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	generated := string(run(t, "cat", cPath))
	if !strings.Contains(generated, "const char *item = value;") {
		t.Fatalf("generated C missing local alias assignment in function body:\n%s", generated)
	}
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\n" {
		t.Fatalf("got %q, want aliased array return output", out)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsStringLiteralAliasedArrayReturnFunctionStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_string_literal_alias_array_func.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_string_literal_alias_array_func.c"
	binPath := dir + "/ast_string_literal_alias_array_func"
	src := "collect = value ->\n  items = []\n  item = \"done\"\n  push items, value\n  push items, item\n  items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_ASSIGN:item:string(done)",
		"AST_PUSH:items:ident(item)",
		"AST_EXPR:ident(items)",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	generated := string(run(t, "cat", cPath))
	if !strings.Contains(generated, "const char *item = \"done\";") || !strings.Contains(generated, "items[items_len] = dup_text(item);") {
		t.Fatalf("generated C missing string literal alias assignment in function body:\n%s", generated)
	}
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\ndone\n" {
		t.Fatalf("got %q, want string literal aliased array return output", out)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsLoopingAliasedArrayReturnFunctionStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_loop_alias_array_func.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_loop_alias_array_func.c"
	binPath := dir + "/ast_loop_alias_array_func"
	src := "collect = value ->\n  items = []\n  item = value\n  count = 0\n  while count < 2\n    push items, item\n    count = count + 1\n  items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_ASSIGN:item:ident(value)",
		"AST_WHILE:binary(< ident(count) int(2))",
		"AST_PUSH:items:ident(item)",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	generated := string(run(t, "cat", cPath))
	if !strings.Contains(generated, "const char *item = value;") || !strings.Contains(generated, "items[items_len] = dup_text(item);") {
		t.Fatalf("generated C missing loop alias push in function body:\n%s", generated)
	}
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\nTya\n" {
		t.Fatalf("got %q, want looping aliased array return output", out)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsLoopingArrayReturnFunctionStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_loop_array_func.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_loop_array_func.c"
	binPath := dir + "/ast_loop_array_func"
	src := "collect = value ->\n  items = []\n  i = 0\n  while i < 2\n    push items, value\n    i = i + 1\n  items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_FUNC:collect:value",
		"AST_WHILE:binary(< ident(i) int(2))",
		"AST_PUSH:items:ident(value)",
		"AST_EXPR:ident(items)",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\nTya\n" {
		t.Fatalf("got %q, want looping array return output", out)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsNamedCounterArrayReturnFunctionStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_named_counter_array_func.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_named_counter_array_func.c"
	binPath := dir + "/ast_named_counter_array_func"
	src := "collect = value ->\n  items = []\n  count = 0\n  while count < 2\n    push items, value\n    count = count + 1\n  items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_FUNC:collect:value",
		"AST_WHILE:binary(< ident(count) int(2))",
		"AST_PUSH:items:ident(value)",
		"AST_EXPR:ident(items)",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	generated := string(run(t, "cat", cPath))
	if !strings.Contains(generated, "long count = 0;") || !strings.Contains(generated, "while ((count < 2))") || !strings.Contains(generated, "count = count + 1;") {
		t.Fatalf("generated C did not use the loop counter from the AST condition:\n%s", generated)
	}
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\nTya\n" {
		t.Fatalf("got %q, want named counter array return output", out)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsNamedCounterInitialValueArrayReturnFunctionStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_named_counter_initial_array_func.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_named_counter_initial_array_func.c"
	binPath := dir + "/ast_named_counter_initial_array_func"
	src := "collect = value ->\n  items = []\n  count = 1\n  while count < 3\n    push items, value\n    count = count + 1\n  items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_ASSIGN:count:int(1)",
		"AST_WHILE:binary(< ident(count) int(3))",
		"AST_PUSH:items:ident(value)",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	generated := string(run(t, "cat", cPath))
	if !strings.Contains(generated, "long count = 1;") || !strings.Contains(generated, "while ((count < 3))") {
		t.Fatalf("generated C did not use the loop counter initial value from the AST assignment:\n%s", generated)
	}
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\nTya\n" {
		t.Fatalf("got %q, want named counter initial value array return output", out)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsNamedCounterStepArrayReturnFunctionStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_named_counter_step_array_func.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_named_counter_step_array_func.c"
	binPath := dir + "/ast_named_counter_step_array_func"
	src := "collect = value ->\n  items = []\n  count = 1\n  while count < 5\n    push items, value\n    count = count + 2\n  items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_ASSIGN:count:int(1)",
		"AST_WHILE:binary(< ident(count) int(5))",
		"AST_ASSIGN:count:binary(+ ident(count) int(2))",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	generated := string(run(t, "cat", cPath))
	if !strings.Contains(generated, "long count = 1;") || !strings.Contains(generated, "count = count + 2;") {
		t.Fatalf("generated C did not use the loop counter step from the AST assignment:\n%s", generated)
	}
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\nTya\n" {
		t.Fatalf("got %q, want named counter step array return output", out)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsLoopingArrayReturnWithContinueStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_loop_array_continue_func.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_loop_array_continue_func.c"
	binPath := dir + "/ast_loop_array_continue_func"
	src := "collect = value ->\n  items = []\n  i = 0\n  while i < 2\n    if i == 0\n      i = i + 1\n      continue\n    push items, value\n    i = i + 1\n  items\nmessage = \"Tya\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_FUNC:collect:value",
		"AST_IF:binary(== ident(i) int(0))",
		"AST_CONTINUE",
		"AST_PUSH:items:ident(value)",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "Tya\n" {
		t.Fatalf("got %q, want looping array return with continue output", out)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsStringAccumulatingArrayReturnFunctionStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_string_accum_array_func.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_string_accum_array_func.c"
	binPath := dir + "/ast_string_accum_array_func"
	src := "collect = value ->\n  items = []\n  text = \"\"\n  i = 0\n  while i < 2\n    text = text + value\n    i = i + 1\n  push items, text\n  items\nmessage = \"Ty\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_ASSIGN:text:string()",
		"AST_ASSIGN:text:binary(+ ident(text) ident(value))",
		"AST_PUSH:items:ident(text)",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "TyTy\n" {
		t.Fatalf("got %q, want string accumulating array return output", out)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsAliasedStringAccumulatingArrayReturnFunctionStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_alias_string_accum_array_func.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_alias_string_accum_array_func.c"
	binPath := dir + "/ast_alias_string_accum_array_func"
	src := "collect = value ->\n  items = []\n  text = \"\"\n  part = value\n  count = 0\n  while count < 2\n    text = text + part\n    count = count + 1\n  push items, text\n  items\nmessage = \"Ty\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_ASSIGN:part:ident(value)",
		"AST_ASSIGN:text:binary(+ ident(text) ident(part))",
		"AST_PUSH:items:ident(text)",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	generated := string(run(t, "cat", cPath))
	if !strings.Contains(generated, "const char *part = value;") || !strings.Contains(generated, "text = concat_text(text, part);") {
		t.Fatalf("generated C missing alias use in string accumulation:\n%s", generated)
	}
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "TyTy\n" {
		t.Fatalf("got %q, want aliased string accumulating array return output", out)
	}
}

func TestSelfhostAstParserCheckerCodegenPreservesArrayFunctionLoopStatementOrder(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_loop_stmt_order_array_func.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_loop_stmt_order_array_func.c"
	binPath := dir + "/ast_loop_stmt_order_array_func"
	src := "collect = value ->\n  items = []\n  text = \"\"\n  i = 0\n  while i < 2\n    push items, text\n    text = text + value\n    i = i + 1\n  items\nmessage = \"Ty\"\nitems = collect(message)\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_PUSH:items:ident(text)",
		"AST_ASSIGN:text:binary(+ ident(text) ident(value))",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	generated := string(run(t, "cat", cPath))
	pushIndex := strings.Index(generated, "items[items_len] = dup_text(text);")
	concatIndex := strings.Index(generated, "text = concat_text(text, value);")
	if pushIndex < 0 || concatIndex < 0 || pushIndex > concatIndex {
		t.Fatalf("generated C did not preserve loop statement order:\n%s", generated)
	}
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "\nTy\n" {
		t.Fatalf("got %q, want loop statement order output", out)
	}
}

func TestSelfhostAstParserKeepsFourArgumentCallsWhole(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_call4.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	src := "token = emit(line, \"INDENT\", text, 1)\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	want := "AST_ASSIGN:token:call(emit ident(line) string(INDENT) ident(text) int(1))"
	if !strings.Contains(nodes, want) {
		t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
	}
	if strings.Contains(nodes, "AST_EXPR:int(1)") {
		t.Fatalf("call argument was split into an expression statement:\n%s", nodes)
	}
}

func TestSelfhostAstParserKeepsNestedCallArgumentsWhole(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_nested_call_arg.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	src := "token = emit(line, \"INDENT\", to_string(line_spaces), 1)\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	want := "AST_ASSIGN:token:call(emit ident(line) string(INDENT) call(to_string ident(line_spaces)) int(1))"
	if !strings.Contains(nodes, want) {
		t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
	}
	if strings.Contains(nodes, "AST_EXPR:int(1)") {
		t.Fatalf("nested call argument was split into an expression statement:\n%s", nodes)
	}
}

func TestSelfhostAstParserKeepsBareNestedCallArgumentsWhole(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_bare_nested_call_arg.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	src := "token = emit line, \"INDENT\", to_string(line_spaces), 1\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	want := "AST_ASSIGN:token:call(emit ident(line) string(INDENT) call(to_string ident(line_spaces)) int(1))"
	if !strings.Contains(nodes, want) {
		t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
	}
	if strings.Contains(nodes, "AST_EXPR:int(1)") {
		t.Fatalf("bare nested call argument was split into an expression statement:\n%s", nodes)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsBinaryStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_binary.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_binary.c"
	binPath := dir + "/ast_binary"
	src := "sum = 1 + 2 * 3\nprint sum\nprint sum + 4\nproduct = (1 + 2) * 3\nprint product\nlarge = product > sum\nprint large\nprint product == 9\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	want := "7\n11\n9\ntrue\ntrue\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsIfWhileStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_if_while.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_if_while.c"
	binPath := dir + "/ast_if_while"
	src := "count = 2\nif count > 0\n  print count\nwhile count > 0\n  print count\n  count = count - 1\nprint count\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_IF:binary(> ident(count) int(0))",
		"AST_WHILE:binary(> ident(count) int(0))",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	want := "2\n2\n1\n0\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsElseStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_else.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_else.c"
	binPath := dir + "/ast_else"
	src := "flag = \"off\"\nif flag == \"on\"\n  print \"yes\"\nelse\n  print \"no\"\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	if !strings.Contains(nodes, "AST_ELSE") {
		t.Fatalf("nodes:\n%s\nmissing AST_ELSE", nodes)
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "no\n" {
		t.Fatalf("got %q, want no", out)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsPushStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_push.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_push.c"
	binPath := dir + "/ast_push"
	src := "items = []\nmessage = \"Tya\"\npush items, message\npush items, \"Lang\"\nprint items[0]\nprint items[1]\nprint len(items)\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_PUSH:items:ident(message)",
		"AST_PUSH:items:string(Lang)",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	want := "Tya\nLang\n2\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsForStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_for.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_for.c"
	binPath := dir + "/ast_for"
	src := "items = []\npush items, \"A\"\npush items, \"B\"\nfor item in items\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	if !strings.Contains(nodes, "AST_FOR:item:items") {
		t.Fatalf("nodes:\n%s\nmissing AST_FOR", nodes)
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	want := "A\nB\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsForIndexStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_for_index.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_for_index.c"
	binPath := dir + "/ast_for_index"
	src := "items = []\npush items, \"A\"\npush items, \"B\"\nfor item, index in items\n  print index\n  print item\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	if !strings.Contains(nodes, "AST_FOR_INDEX:item:index:items") {
		t.Fatalf("nodes:\n%s\nmissing AST_FOR_INDEX", nodes)
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	want := "0\nA\n1\nB\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsBreakContinueStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_break_continue.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_break_continue.c"
	binPath := dir + "/ast_break_continue"
	src := "i = 0\nwhile i < 4\n  i = i + 1\n  if i == 2\n    continue\n  print i\n  if i == 3\n    break\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{"AST_CONTINUE", "AST_BREAK"} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	want := "1\n3\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsDeleteExitPanicStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_side_effects.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_side_effects.c"
	binPath := dir + "/ast_side_effects"
	src := "user = { name: \"Tya\" }\ndelete user, \"name\"\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	if !strings.Contains(nodes, "AST_DELETE:user:string(name)") {
		t.Fatalf("nodes:\n%s\nmissing AST_DELETE", nodes)
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	deleteRunNodesPath := dir + "/delete_run.nodes"
	deleteRunNodes := "1:AST_ASSIGN:user:object1(name string(Tya))\n2:AST_DELETE:user:string(name)\n3:PRINT_INDEX:IDENT:user:name\n"
	if err := os.WriteFile(deleteRunNodesPath, []byte(deleteRunNodes), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", deleteRunNodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	if out != "nil\n" {
		t.Fatalf("got %q, want nil", out)
	}

	exitNodesPath := dir + "/exit.nodes"
	exitCPath := dir + "/exit.c"
	exitBinPath := dir + "/exit"
	if err := os.WriteFile(exitNodesPath, []byte("1:AST_EXIT:int(7)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, exitCPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", exitNodesPath)
	run(t, "cc", exitCPath, "-o", exitBinPath)
	exitCmd := exec.Command(exitBinPath)
	exitOut, exitErr := exitCmd.CombinedOutput()
	if exitErr == nil {
		t.Fatalf("exit command succeeded unexpectedly: %q", exitOut)
	}
	if status, ok := exitErr.(*exec.ExitError); !ok || status.ExitCode() != 7 {
		t.Fatalf("got exit err %v output %q, want status 7", exitErr, exitOut)
	}

	panicNodesPath := dir + "/panic.nodes"
	panicCPath := dir + "/panic.c"
	panicBinPath := dir + "/panic"
	if err := os.WriteFile(panicNodesPath, []byte("1:AST_PANIC:string(bad)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, panicCPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", panicNodesPath)
	run(t, "cc", panicCPath, "-o", panicBinPath)
	panicCmd := exec.Command(panicBinPath)
	panicOut, panicErr := panicCmd.CombinedOutput()
	if panicErr == nil {
		t.Fatalf("panic command succeeded unexpectedly: %q", panicOut)
	}
	if status, ok := panicErr.(*exec.ExitError); !ok || status.ExitCode() != 1 {
		t.Fatalf("got panic err %v output %q, want status 1", panicErr, panicOut)
	}
	if string(panicOut) != "panic: bad\n" {
		t.Fatalf("got panic output %q", panicOut)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsMemberIndexAndCallStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_member_index_call.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_member_index_call.c"
	binPath := dir + "/ast_member_index_call"
	src := "items = [\"A\", \"B\", \"C\"]\nprint items[1]\nprint items[2]\nsize = len(items)\nprint size\nuser = { name: \"Tya\" }\nprint user.name\nprint len(items)\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_ASSIGN:items:array3(string(A) string(B) string(C))",
		"AST_ASSIGN:size:call(len ident(items))",
		"AST_ASSIGN:user:object1(name string(Tya))",
		"AST_PRINT:index(items 1)",
		"AST_PRINT:index(items 2)",
		"AST_PRINT:ident(size)",
		"AST_PRINT:member(user.name)",
		"AST_PRINT:call(len ident(items))",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	want := "B\nC\n3\nTya\n3\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsStdlibCallStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_stdlib_call.tya"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_stdlib_call.c"
	binPath := dir + "/ast_stdlib_call"
	src := "raw = \" Tya \"\nmessage = trim(raw)\nprint message\nhas_t = contains(message, \"T\")\nprint has_t\nprint contains(message, \"x\")\nhas_prefix = starts_with(message, \"T\")\nprint has_prefix\nhas_suffix = ends_with(message, \"a\")\nprint has_suffix\nchanged = replace(message, \"T\", \"M\")\nprint changed\nprint replace(message, \"y\", \"i\")\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	for _, want := range []string{
		"AST_ASSIGN:message:call(trim ident(raw))",
		"AST_ASSIGN:has_t:call(contains ident(message) string(T))",
		"AST_PRINT:call(contains ident(message) string(x))",
		"AST_ASSIGN:has_prefix:call(starts_with ident(message) string(T))",
		"AST_ASSIGN:has_suffix:call(ends_with ident(message) string(a))",
		"AST_ASSIGN:changed:call(replace ident(message) string(T) string(M))",
		"AST_PRINT:call(replace ident(message) string(y) string(i))",
	} {
		if !strings.Contains(nodes, want) {
			t.Fatalf("nodes:\n%s\nmissing %q", nodes, want)
		}
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath))
	want := "Tya\ntrue\nfalse\ntrue\ntrue\nMya\nTia\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostAstParserCheckerCodegenRunsReadFileArgsIndexStream(t *testing.T) {
	dir := t.TempDir()
	srcPath := dir + "/ast_read_file.tya"
	inputPath := dir + "/input.txt"
	tokensPath := dir + "/tokens.txt"
	astTokensPath := dir + "/ast_tokens.txt"
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/ast_read_file.c"
	binPath := dir + "/ast_read_file"
	src := "source = read_file args()[0]\nprint source\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputPath, []byte("Tya"), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, tokensPath, "go", "run", "./cmd/tya", "selfhost/lexer.tya", srcPath)
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(astTokensPath, append([]byte("0:IDENT:ASTMODE\n"), tokens...), 0644); err != nil {
		t.Fatal(err)
	}
	runToFile(t, nodesPath, "go", "run", "./cmd/tya", "selfhost/parser.tya", astTokensPath)
	nodes := string(run(t, "cat", nodesPath))
	if !strings.Contains(nodes, "AST_ASSIGN:source:call_index(read_file args 0)") {
		t.Fatalf("nodes:\n%s\nmissing call_index AST assignment", nodes)
	}
	checkOut := string(run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", nodesPath))
	if checkOut != "ok\n" {
		t.Fatalf("checker got %q", checkOut)
	}
	runToFile(t, cPath, "go", "run", "./cmd/tya", "selfhost/codegen_c.tya", nodesPath)
	run(t, "cc", cPath, "-o", binPath)
	out := string(run(t, binPath, inputPath))
	if out != "Tya\n" {
		t.Fatalf("got %q, want Tya", out)
	}
}

func TestSelfhostCodegenUsesCallEnvAdapter(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "selfhost", "codegen_c.tya"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	required := []string{
		"codegen_env = indent, deleted_object, deleted_key ->",
		"[prefix, deleted_object, deleted_key]",
		"emit_print_call2 = lines, env, node ->",
		"names = env[3]",
		"types = env[4]",
		"emit_print_call3 = lines, env, node ->",
		"emit_assign_call3 = lines, env, node ->",
		"emit_assign_call2_split = lines, env, node ->",
		"emit_assign_call2_admin = lines, env, node ->",
		"emit_assign_call2_collection = lines, env, node ->",
		"prefix = env[0]",
		"env = codegen_env indent, deleted_object, deleted_key",
		"emit_print_call2 lines, env, node",
		"emit_assign_call3 lines, env, node",
		"emit_assign_call2_split lines, env, node",
		"emit_assign_call2_admin lines, env, node",
		"emit_assign_call2_collection lines, env, node",
		"emit_print_call3 lines, codegen_env(indent, deleted_object, deleted_key), node",
	}
	for _, marker := range required {
		if !strings.Contains(source, marker) {
			t.Fatalf("selfhost codegen is missing call env adapter marker %q", marker)
		}
	}
}

func TestSelfhostCodegenUsesIterableTypeAdapter(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "selfhost", "codegen_c.tya"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	required := []string{
		"iterable_kind = name ->",
		`parts = split name, "."`,
		"if len(parts) > 1",
		`index_parts = split name, "["`,
		"if len(index_parts) > 1",
		"collection_type = type_of(names, types, value_of(node))",
		`if iterable_kind(value_of(node)) == "MEMBER"`,
		`if iterable_kind(value_of(node)) == "INDEX"`,
		`if collection_type == "ARRAY"`,
		`if collection_type == "DYNARRAY"`,
		`if collection_type == "INTARRAY"`,
		"collection_type = type_of(names, types, collection_name)",
		`if iterable_kind(collection_name) == "MEMBER"`,
		`if iterable_kind(collection_name) == "INDEX"`,
	}
	for _, marker := range required {
		if !strings.Contains(source, marker) {
			t.Fatalf("selfhost codegen is missing iterable type adapter marker %q", marker)
		}
	}
}

func TestSelfhostGeneratedCodegenUsesAstBinaryCompareHelper(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "selfhost", "codegen_c.tya"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	required := []string{
		"static int ast_generated_is_compare_op(const char *op)",
		"ast_generated_is_compare_op(op)",
		"ast_generated_is_compare_op(outer_op)",
	}
	for _, marker := range required {
		if !strings.Contains(source, marker) {
			t.Fatalf("selfhost generated codegen is missing AST binary compare helper marker %q", marker)
		}
	}
}

func TestSelfhostGeneratedCodegenUsesAstConcatNeedHelper(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "selfhost", "codegen_c.tya"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	required := []string{
		"static int ast_generated_needs_concat(const char *expr)",
		"ast_generated_needs_concat(b)",
		"ast_generated_needs_concat(a)",
		"binary(+ ident(",
		"binary(+ string(",
	}
	for _, marker := range required {
		if !strings.Contains(source, marker) {
			t.Fatalf("selfhost generated codegen is missing AST concat helper marker %q", marker)
		}
	}
}

func TestSelfhostGeneratedCodegenUsesAstKnownTypeHelper(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "selfhost", "codegen_c.tya"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	required := []string{
		"static int ast_generated_known_type(char names[][256], char types[][32], int len, const char *name, const char *want)",
		`ast_generated_known_type(known_names, known_types, known_len, left, \"INT\")`,
		`ast_generated_known_type(known_names, known_types, known_len, right_ident, \"STRING\")`,
	}
	for _, marker := range required {
		if !strings.Contains(source, marker) {
			t.Fatalf("selfhost generated codegen is missing AST known type helper marker %q", marker)
		}
	}
}

func TestSelfhostGeneratedCodegenUsesAstStringAssignmentHelper(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "selfhost", "codegen_c.tya"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	required := []string{
		"static void ast_generated_emit_string_assignment(const char *target, const char *value, int declare_target)",
		"ast_generated_emit_string_assignment(a, concat_expr, 0)",
		"ast_generated_emit_string_assignment(a, concat_expr, 1)",
	}
	for _, marker := range required {
		if !strings.Contains(source, marker) {
			t.Fatalf("selfhost generated codegen is missing AST string assignment helper marker %q", marker)
		}
	}
}

func TestSelfhostGeneratedCodegenUsesAstStringPrintHelper(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "selfhost", "codegen_c.tya"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	required := []string{
		"static void ast_generated_emit_string_print(const char *value)",
		"ast_generated_emit_string_print(concat_expr)",
	}
	for _, marker := range required {
		if !strings.Contains(source, marker) {
			t.Fatalf("selfhost generated codegen is missing AST string print helper marker %q", marker)
		}
	}
}

func TestSelfhostCodegenRunsMultipleReturnSubset(t *testing.T) {
	dir := t.TempDir()
	nodesPath := dir + "/nodes.txt"
	cPath := dir + "/multiple_return.c"
	binPath := dir + "/multiple_return"
	nodes := "1:FUNC:parse_user:text\n2:INDENT:2\n2:IF_COMPARE_EQ:IDENT:text:STRING:\n3:INDENT:4\n3:RETURN2_CALL1:NIL:nil:error:STRING:empty user\n4:INDENT:2\n4:RETURN2_OBJECT_NIL:name:IDENT:text\n6:INDENT:0\n6:MULTI_ASSIGN2_CALL1:user:err:parse_user:STRING:komagata\n8:INDENT:0\n8:IF:IDENT:err\n9:INDENT:2\n9:PRINT_MEMBER:err:message\n10:INDENT:0\n10:ELSE\n11:INDENT:2\n11:PRINT_MEMBER:user:name\n13:INDENT:0\n13:MULTI_ASSIGN2_CALL1:missing:err:parse_user:STRING:\n15:INDENT:0\n15:IF:IDENT:err\n16:INDENT:2\n16:PRINT_MEMBER:err:message\n17:INDENT:0\n17:ELSE\n18:INDENT:2\n18:PRINT_MEMBER:missing:name\n"
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
	want := "selfhost/lexer.tya: stage-1 emitted and compiled C\nselfhost/parser.tya: stage-1 emitted and compiled C\nselfhost/checker.tya: stage-1 emitted and compiled C\nselfhost/codegen_c.tya: stage-1 emitted and compiled C\nexamples/hello.tya: stage-2 lexer matched\nexamples/hello.tya: stage-2 parser matched\nexamples/hello.tya: stage-2 checker matched\nexamples/hello.tya: stage-2 codegen deterministic\nexamples/hello.tya: stage-2 codegen matched\nescaped string print: stage-2 codegen deterministic\nescaped string print: stage-2 pipeline matched\nstring index print: stage-2 codegen deterministic\nstring index print: stage-2 pipeline matched\nexamples/string.tya: stage-2 codegen deterministic\nexamples/string.tya: stage-2 pipeline matched\nint literal: stage-2 lexer matched\noperators: stage-2 lexer matched\nfloat and string escape: stage-2 lexer matched\nint assignment: stage-2 parser matched\nint assignment: stage-2 codegen deterministic\nliteral assignments: stage-2 parser matched\nliteral assignments: stage-2 codegen deterministic\nliteral assignments: stage-2 checker matched\nint assignment: stage-2 codegen matched\nliteral assignments: stage-2 codegen matched\nliteral reassignment: stage-2 codegen deterministic\nliteral reassignment: stage-2 pipeline matched\nread file arg: stage-2 codegen deterministic\nread file arg: stage-2 pipeline matched\nlex source: stage-2 codegen deterministic\nlex source: stage-2 pipeline matched\nlong lex source: stage-2 pipeline matched\nparse tokens: stage-2 codegen deterministic\nparse tokens: stage-2 pipeline matched\ncheck nodes: stage-2 codegen deterministic\ncheck nodes: stage-2 pipeline matched\nconstant reassignment: stage-1 checker matched\nemit_c nodes: stage-2 codegen deterministic\nemit_c nodes: stage-2 pipeline matched\nfunction body skip: stage-2 parser matched\nprint int assignment: stage-2 codegen deterministic\nprint int assignment: stage-2 pipeline matched\nprint literal assignments: stage-2 codegen deterministic\nprint literal assignments: stage-2 pipeline matched\nstring len print: stage-2 codegen deterministic\nstring len print: stage-2 pipeline matched\nstring trim print: stage-2 codegen deterministic\nstring trim print: stage-2 pipeline matched\nstring contains print: stage-2 codegen deterministic\nstring contains print: stage-2 pipeline matched\nstring prefix suffix print: stage-2 codegen deterministic\nstring prefix suffix print: stage-2 pipeline matched\nstring replace print: stage-2 codegen deterministic\nstring replace print: stage-2 pipeline matched\nstring split join print: stage-2 codegen deterministic\nstring split join print: stage-2 pipeline matched\nstring byte char length print: stage-2 codegen deterministic\nstring byte char length print: stage-2 pipeline matched\nint addition: stage-2 codegen deterministic\nint addition: stage-2 pipeline matched\ngrouped int addition: stage-2 codegen deterministic\ngrouped int addition: stage-2 pipeline matched\nint addition reassignment: stage-2 codegen deterministic\nint addition reassignment: stage-2 pipeline matched\nbool assignment: stage-2 codegen deterministic\nbool assignment: stage-2 pipeline matched\nbool logic: stage-2 codegen deterministic\nbool logic: stage-2 pipeline matched\nwhile false break: stage-2 codegen deterministic\nwhile false break: stage-2 pipeline matched\nwhile less-than break: stage-2 codegen deterministic\nwhile less-than break: stage-2 pipeline matched\nwhile bounded break: stage-2 codegen deterministic\nwhile bounded break: stage-2 pipeline matched\narray for: stage-2 codegen deterministic\narray for: stage-2 pipeline matched\nexamples/selfhost_ops.tya: stage-2 codegen deterministic\nexamples/selfhost_ops.tya: stage-2 pipeline matched\nexamples/multiple_return.tya: stage-2 parser matched\nexamples/multiple_return.tya: stage-2 checker matched\nexamples/multiple_return.tya: stage-2 codegen deterministic\nexamples/multiple_return.tya: stage-2 pipeline matched\nexamples/while.tya: stage-2 codegen deterministic\nexamples/while.tya: stage-2 pipeline matched\nequality comparison: stage-2 codegen deterministic\nequality comparison: stage-2 pipeline matched\ninequality comparison: stage-2 codegen deterministic\ninequality comparison: stage-2 pipeline matched\nless-than comparison: stage-2 codegen deterministic\nless-than comparison: stage-2 pipeline matched\nbounded comparison: stage-2 codegen deterministic\nbounded comparison: stage-2 pipeline matched\ngrouped comparison: stage-2 codegen deterministic\ngrouped comparison: stage-2 pipeline matched\nselfhost/lexer.tya: stage-3 parser emitted real nodes\nselfhost/lexer.tya: stage-3 codegen emitted executable lexer C\nselfhost/parser.tya: stage-3 parser emitted real nodes\nselfhost/parser.tya: stage-3 codegen emitted executable parser C\nselfhost/checker.tya: stage-3 parser emitted real nodes\nselfhost/checker.tya: stage-3 codegen emitted executable checker C\nselfhost/codegen_c.tya: stage-3 parser emitted real nodes\nselfhost/codegen_c.tya: stage-3 codegen emitted executable codegen C\nstage4 hello: self-host pipeline matched\nstage4 print string: self-host pipeline matched\nstage4 print int: self-host pipeline matched\nstage4 escaped string: self-host pipeline matched\nstage4 colon string: self-host pipeline matched\nstage4 two prints: self-host pipeline matched\nstage4 assign print: self-host pipeline matched\nstage4 int assign print: self-host pipeline matched\nstage4 int reassignment print: self-host pipeline matched\nstage4 int addition print: self-host pipeline matched\nstage4 less-than print: self-host pipeline matched\nstage4 while false print: self-host pipeline matched\nstage4 array for: self-host pipeline matched\nstage4 multiple return: self-host pipeline matched\nstage4 while example: self-host pipeline matched\nstage4 string example: self-host pipeline matched\nstage4 selfhost ops: self-host pipeline matched\nstage4 arithmetic: self-host pipeline matched\nstage4 function: self-host pipeline matched\nstage4 return: self-host pipeline matched\nstage4 object: self-host pipeline matched\nstage4 object inline: self-host pipeline matched\nstage4 if example: self-host pipeline matched\nstage4 logic: self-host pipeline matched\nstage4 error: self-host pipeline matched\nstage4 convert: self-host pipeline matched\nstage4 file: self-host pipeline matched\nstage4 args: self-host pipeline matched\nstage4 equal: self-host pipeline matched\nselfhost/lexer.tya: stage-4 emitted and compiled stage-5 C\nselfhost/parser.tya: stage-4 emitted and compiled stage-5 C\nselfhost/checker.tya: stage-4 emitted and compiled stage-5 C\nselfhost/codegen_c.tya: stage-4 emitted and compiled stage-5 C\n"
	want = strings.Replace(want, "stage4 equal: self-host pipeline matched\nselfhost/lexer.tya", "stage4 equal: self-host pipeline matched\nstage4 array: self-host pipeline matched\nselfhost/lexer.tya", 1)
	want = strings.Replace(want, "stage4 array: self-host pipeline matched\nselfhost/lexer.tya", "stage4 array: self-host pipeline matched\nstage4 for: self-host pipeline matched\nselfhost/lexer.tya", 1)
	want = strings.Replace(want, "array for: stage-2 pipeline matched\nexamples/selfhost_ops.tya", "array for: stage-2 pipeline matched\narray index assignment: stage-2 codegen deterministic\narray index assignment: stage-2 pipeline matched\nexamples/selfhost_ops.tya", 1)
	want = strings.Replace(want, "array index assignment: stage-2 pipeline matched\nexamples/selfhost_ops.tya", "array index assignment: stage-2 pipeline matched\ninline filter function literal: stage-2 codegen deterministic\ninline filter function literal: stage-2 pipeline matched\nexamples/selfhost_ops.tya", 1)
	want = strings.Replace(want, "check nodes: stage-2 pipeline matched\nconstant reassignment", "check nodes: stage-2 pipeline matched\ncheck nodes undefined print: stage-2 pipeline matched\ncheck nodes undefined assign/for: stage-2 pipeline matched\ncheck nodes constant reassignment: stage-2 pipeline matched\ncheck nodes constant first assignment: stage-2 pipeline matched\ncheck nodes bool-not undefined: stage-2 pipeline matched\ncheck nodes bool-binary undefined: stage-2 pipeline matched\ncheck nodes arithmetic undefined: stage-2 pipeline matched\ncheck nodes compare undefined: stage-2 pipeline matched\nconstant reassignment", 1)
	want = strings.Replace(want, "while less-than break: stage-2 pipeline matched\nwhile bounded break", "while less-than break: stage-2 pipeline matched\nwhile greater-than break: stage-2 codegen deterministic\nwhile greater-than break: stage-2 pipeline matched\nwhile bounded break", 1)
	want = strings.Replace(want, "less-than comparison: stage-2 pipeline matched\nbounded comparison", "less-than comparison: stage-2 pipeline matched\ngreater-than comparison: stage-2 codegen deterministic\ngreater-than comparison: stage-2 pipeline matched\nbounded comparison", 1)
	want = strings.ReplaceAll(want, "examples/multiple_return.tya: stage-2 parser matched\nexamples/multiple_return.tya: stage-2 checker matched\nexamples/multiple_return.tya: stage-2 codegen deterministic\nexamples/multiple_return.tya: stage-2 pipeline matched\n", "")
	want = strings.ReplaceAll(want, "stage4 multiple return: self-host pipeline matched\n", "")
	want = strings.ReplaceAll(want, "stage4 function: self-host pipeline matched\n", "")
	want = strings.ReplaceAll(want, "stage4 object: self-host pipeline matched\nstage4 object inline: self-host pipeline matched\nstage4 if example: self-host pipeline matched\n", "")
	want = strings.Replace(want, "selfhost/codegen_c.tya: stage-4 emitted and compiled stage-5 C\n", "selfhost/codegen_c.tya: stage-4 emitted and compiled stage-5 C\nstage5 hello: self-host pipeline matched\n", 1)
	want = strings.Replace(want, "stage5 hello: self-host pipeline matched\n", "stage5 hello: self-host pipeline matched\nstage5 print string: self-host pipeline matched\n", 1)
	want = strings.Replace(want, "stage5 print string: self-host pipeline matched\n", "stage5 print string: self-host pipeline matched\nstage5 print int: self-host pipeline matched\n", 1)
	want = strings.Replace(want, "stage5 print int: self-host pipeline matched\n", "stage5 print int: self-host pipeline matched\nstage5 two prints: self-host pipeline matched\n", 1)
	want = strings.Replace(want, "stage5 two prints: self-host pipeline matched\n", "stage5 two prints: self-host pipeline matched\nstage5 constant reassignment: self-host checker matched\n", 1)
	want = strings.Replace(want, "stage5 constant reassignment: self-host checker matched\n", "stage5 constant reassignment: self-host checker matched\nstage5 undefined print: self-host checker matched\n", 1)
	want = strings.Replace(want, "stage5 undefined print: self-host checker matched\n", "stage5 undefined print: self-host checker matched\nstage5 undefined assignment: self-host checker matched\n", 1)
	want = strings.Replace(want, "stage5 undefined assignment: self-host checker matched\n", "stage5 undefined assignment: self-host checker matched\nstage5 undefined for collection: self-host checker matched\n", 1)
	want = strings.Replace(want, "stage5 undefined for collection: self-host checker matched\n", "stage5 undefined for collection: self-host checker matched\nselfhost/lexer.tya: stage-5 emitted and compiled stage-6 C\nselfhost/parser.tya: stage-5 emitted and compiled stage-6 C\nselfhost/checker.tya: stage-5 emitted and compiled stage-6 C\nselfhost/codegen_c.tya: stage-5 emitted and compiled stage-6 C\nstage6 print string: self-host pipeline matched\n", 1)
	want = strings.Replace(want, "stage6 print string: self-host pipeline matched\n", "stage6 print string: self-host pipeline matched\nstage6 print int: self-host pipeline matched\nstage6 two prints: self-host pipeline matched\n", 1)
	want = strings.Replace(want, "stage6 two prints: self-host pipeline matched\n", "stage6 two prints: self-host pipeline matched\nselfhost/lexer.tya: stage-6 emitted stable stage-7 C\nselfhost/parser.tya: stage-6 emitted stable stage-7 C\nselfhost/checker.tya: stage-6 emitted stable stage-7 C\nselfhost/codegen_c.tya: stage-6 emitted stable stage-7 C\nstage7 self-host fixed point: self-host pipeline matched\n", 1)
	manifestRaw, err := os.ReadFile(filepath.Join("..", "scripts", "selfhost_examples_manifest.txt"))
	if err != nil {
		t.Fatal(err)
	}
	var manifestLines strings.Builder
	for _, line := range strings.Split(string(manifestRaw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) == 4 && parts[1] == "supported" {
			manifestLines.WriteString(parts[0])
			if parts[0] == "examples/panic.tya" {
				manifestLines.WriteString(": stage-4 manifest panic status matched\n")
			} else {
				manifestLines.WriteString(": stage-4 manifest pipeline matched\n")
			}
		}
	}
	want = strings.Replace(want, "stage4 for: self-host pipeline matched\nselfhost/lexer.tya", "stage4 for: self-host pipeline matched\n"+manifestLines.String()+"selfhost/lexer.tya", 1)
	want = strings.Replace(want, "examples/while.tya: stage-4 manifest pipeline matched\nselfhost/lexer.tya", "examples/while.tya: stage-4 manifest pipeline matched\nexamples/exit.tya: stage-4 manifest exit status matched\nselfhost/lexer.tya", 1)
	want = strings.Replace(want, "selfhost/lexer.tya: stage-4 emitted and compiled stage-5 C\n", "selfhost/lexer.tya: stage-4 fixed-point generated C stable\nselfhost/lexer.tya: stage-4 emitted and compiled stage-5 C\n", 1)
	want = strings.Replace(want, "selfhost/parser.tya: stage-4 emitted and compiled stage-5 C\n", "selfhost/parser.tya: stage-4 fixed-point generated C stable\nselfhost/parser.tya: stage-4 emitted and compiled stage-5 C\n", 1)
	want = strings.Replace(want, "selfhost/checker.tya: stage-4 emitted and compiled stage-5 C\n", "selfhost/checker.tya: stage-4 fixed-point generated C stable\nselfhost/checker.tya: stage-4 emitted and compiled stage-5 C\n", 1)
	want = strings.Replace(want, "selfhost/codegen_c.tya: stage-4 emitted and compiled stage-5 C\n", "selfhost/codegen_c.tya: stage-4 fixed-point generated C stable\nselfhost/codegen_c.tya: stage-4 emitted and compiled stage-5 C\n", 1)
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostFixedPointCheck(t *testing.T) {
	if os.Getenv("TYA_RUN_SLOW_FIXED_POINT_TEST") != "1" {
		t.Skip("covered by standalone scripts/selfhost_fixed_point_check.sh and selfhost_bootstrap_check.sh verification")
	}
	out := run(t, "sh", "scripts/selfhost_fixed_point_check.sh")
	want := "selfhost/lexer.tya: stage-4 fixed-point generated C stable\nselfhost/parser.tya: stage-4 fixed-point generated C stable\nselfhost/checker.tya: stage-4 fixed-point generated C stable\nselfhost/codegen_c.tya: stage-4 fixed-point generated C stable\n"
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

func TestSelfhostBootstrapGateDocumentation(t *testing.T) {
	bootstrapRaw, err := os.ReadFile(filepath.Join("..", "scripts", "selfhost_bootstrap_check.sh"))
	if err != nil {
		t.Fatal(err)
	}
	bootstrap := string(bootstrapRaw)
	requiredScripts := []string{
		"scripts/selfhost_check.sh",
		"scripts/selfhost_compile_check.sh",
		"scripts/go_emit_selfhost_compile_check.sh",
		"scripts/go_emit_selfhost_ops_check.sh",
		"scripts/stage1_selfhost_sources_check.sh",
		"scripts/go_emit_selfhost_run_check.sh",
	}
	for _, script := range requiredScripts {
		if !strings.Contains(bootstrap, script) {
			t.Fatalf("self-host bootstrap gate does not run %s", script)
		}
	}

	stageRaw, err := os.ReadFile(filepath.Join("..", "scripts", "stage1_selfhost_sources_check.sh"))
	if err != nil {
		t.Fatal(err)
	}
	stage := string(stageRaw)
	requiredStageCoverage := []string{
		"scripts/selfhost_examples_manifest.txt",
		"assert_check_ok()",
		`test "$(cat "$check_file")" = "ok"`,
		"raw parser adapter remains",
		"argv[0] mode fallback remains",
		"generated checker source classifier remains",
		"stage-4 fixed-point generated C stable",
		"stage7 self-host fixed point",
	}
	for _, marker := range requiredStageCoverage {
		if !strings.Contains(stage, marker) {
			t.Fatalf("stage-generated self-host gate is missing %q", marker)
		}
	}
	if strings.Contains(stage, `grep -qx "ok"`) {
		t.Fatal("stage-generated self-host gate still accepts check files with extra errors after ok")
	}

	requiredDocs := []string{
		filepath.Join("..", "README.md"),
		filepath.Join("..", "ROADMAP.md"),
		filepath.Join("..", "SELFHOST_WORK.md"),
	}
	for _, doc := range requiredDocs {
		raw, err := os.ReadFile(doc)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "sh scripts/selfhost_bootstrap_check.sh") {
			t.Fatalf("%s does not document the self-host bootstrap gate", doc)
		}
	}
}

func TestGoEmitterMatchesSelectedExamples(t *testing.T) {
	out := run(t, "sh", "scripts/go_emit_examples_check.sh")
	want := "examples/hello.tya: matched\nexamples/arithmetic.tya: matched\nexamples/function.tya: matched\nexamples/return.tya: matched\nexamples/multiple_return.tya: matched\nexamples/try.tya: matched\nexamples/while.tya: matched\nexamples/if.tya: matched\nexamples/logic.tya: matched\nexamples/array.tya: matched\nexamples/archive/pre-v0.1/array_function.tya: matched\nexamples/classic/array_sum.tya: matched\nexamples/classic/factorial.tya: matched\nexamples/classic/fib.tya: matched\nexamples/classic/fizzbuzz.tya: matched\nexamples/classic/gcd.tya: matched\nexamples/classic/prime.tya: matched\nexamples/string.tya: matched\nexamples/object.tya: matched\nexamples/object_inline.tya: matched\nexamples/object_builtin.tya: matched\nexamples/method.tya: matched\nexamples/prelude.tya: matched\nexamples/convert.tya: matched\nexamples/error.tya: matched\nexamples/file.tya: matched\nexamples/equal.tya: matched\nexamples/for.tya: matched\nexamples/for_object.tya: matched\nexamples/read_line.tya: matched\nexamples/exit.tya: matched\nexamples/use_module.tya: matched\n"
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

func TestSelfhostExampleParityManifest(t *testing.T) {
	manifestPath := filepath.Join("..", "scripts", "selfhost_examples_manifest.txt")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	stageCheck, err := os.ReadFile(filepath.Join("..", "scripts", "stage1_selfhost_sources_check.sh"))
	if err != nil {
		t.Fatal(err)
	}
	exampleFiles, err := filepath.Glob(filepath.Join("..", "examples", "**", "*.tya"))
	if err != nil {
		t.Fatal(err)
	}
	rootExamples, err := filepath.Glob(filepath.Join("..", "examples", "*.tya"))
	if err != nil {
		t.Fatal(err)
	}
	exampleFiles = append(exampleFiles, rootExamples...)

	examples := map[string]bool{}
	for _, path := range exampleFiles {
		rel, err := filepath.Rel("..", path)
		if err != nil {
			t.Fatal(err)
		}
		examples[filepath.ToSlash(rel)] = true
	}

	classified := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 4 {
			t.Fatalf("invalid manifest line %q", line)
		}
		path, status, gate, reason := parts[0], parts[1], parts[2], parts[3]
		if !strings.HasPrefix(path, "examples/") || !strings.HasSuffix(path, ".tya") {
			t.Fatalf("invalid example path %q", path)
		}
		if reason == "" {
			t.Fatalf("missing reason for %s", path)
		}
		if !examples[path] {
			t.Fatalf("manifest references missing example %s", path)
		}
		if classified[path] {
			t.Fatalf("duplicate manifest entry for %s", path)
		}
		classified[path] = true
		switch status {
		case "supported":
			if gate != "scripts/stage1_selfhost_sources_check.sh" {
				t.Fatalf("supported example %s has non-bootstrap gate %q", path, gate)
			}
			if !strings.Contains(string(stageCheck), path) {
				t.Fatalf("supported example %s is not referenced by %s", path, gate)
			}
		case "expected-failing", "out-of-scope":
			if gate == "" {
				t.Fatalf("%s example %s needs a feature or scope reason", status, path)
			}
		default:
			t.Fatalf("invalid status %q for %s", status, path)
		}
	}

	var missing []string
	for path := range examples {
		if !classified[path] {
			missing = append(missing, path)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("examples missing self-host parity classification: %s", strings.Join(missing, ", "))
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
	if err := os.WriteFile(path, []byte("1:ASSIGN:alias:IDENT:missing\n2:ASSIGN:ok:COMPARE_GE:missing:1\n3:ASSIGN:ok2:COMPARE_GT:missing:1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missing\n2: undefined variable: missing\n3: undefined variable: missing\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsConstantReassignment(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:ASSIGN:MAX_RETRY:INT:3\n2:ASSIGN:MAX_RETRY:INT:5\n3:ASSIGN:retry_count:INT:3\n4:ASSIGN:retry_count:INT:5\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "2: cannot reassign constant MAX_RETRY\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerChecksMultiAssign2Names(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:MULTI_ASSIGN2:left:right:IDENT:missing\n2:PRINT:IDENT:left\n3:MULTI_ASSIGN2:valid:alsoBad:STRING:value\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missing\n3: invalid binding name: alsoBad\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerChecksMultiAssign2CallNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:MULTI_ASSIGN2_CALL1:left:right:missingFunc:IDENT:missingArg\n2:PRINT:IDENT:left\n3:MULTI_ASSIGN2_CALL1:valid:alsoBad:missingFunc:IDENT:missingArg\n4:MULTI_ASSIGN2_CALL1:ok:err:missingFunc:STRING:value\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingFunc\n1: undefined variable: missingArg\n3: invalid binding name: alsoBad\n3: undefined variable: missingFunc\n3: undefined variable: missingArg\n4: undefined variable: missingFunc\n"
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
	if err := os.WriteFile(path, []byte("1:PRINT_MEMBER:missingTarget:name\n"), 0644); err != nil {
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

func TestSelfhostCheckerRejectsUndefinedBoolBinaryNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:ASSIGN:known:BOOL:true\n2:ASSIGN:both:BOOL_AND:known:missing_and\n3:ASSIGN:either:BOOL_OR:missing_or:known\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "2: undefined variable: missing_and\n3: undefined variable: missing_or\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedArithmeticNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:ASSIGN:known:INT:1\n2:ASSIGN:sum:INT_ADD:known:missing_add\n3:ASSIGN:typed:INT_SUB:IDENT:missing_sub:INT:1\n4:ASSIGN:literal:INT_MUL:INT:2:INT:3\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "2: undefined variable: missing_add\n3: undefined variable: missing_sub\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedCompareAssignNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:ASSIGN:known:INT:1\n2:ASSIGN:greater:COMPARE_GT:known:missing_gt\n3:ASSIGN:equal:COMPARE_EQ:missing_eq:1\n4:ASSIGN:literal:COMPARE_LE:1:2\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "2: undefined variable: missing_gt\n3: undefined variable: missing_eq\n"
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
	nodes := "1:ASSIGN:result:CALL1:missingFunc:missingArg\n2:ASSIGN:indexed:CALL1_INDEX:missingIndexed:missingItems:i\n3:ASSIGN:indexed_binary:CALL1_INDEX_BINARY:missingIndexedBinary:missingTokens:i:+:missingOffset\n4:ASSIGN:expr:CALL1_EXPR:missingExprFunc:INT_ADD:missingLeft:missingRight\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingFunc\n1: undefined variable: missingArg\n2: undefined variable: missingIndexed\n2: undefined variable: missingItems\n2: undefined variable: i\n3: undefined variable: missingIndexedBinary\n3: undefined variable: missingTokens\n3: undefined variable: i\n3: undefined variable: missingOffset\n4: undefined variable: missingExprFunc\n4: undefined variable: missingLeft\n4: undefined variable: missingRight\n"
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
	nodes := "1:ASSIGN:message:STRING:Tya\n2:PRINT_CALL2:contains:message:STRING:T\n3:PRINT_CALL2:starts_with:message:STRING:T\n4:PRINT_CALL2:ends_with:message:STRING:a\n"
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
	nodes := "1:IF_CALL1:missingFunc:missingArg\n2:PRINT_CALL1_INDEX:missingPrint:missingItems:i\n3:PRINT_CALL1_INDEX_BINARY:missingPrintBinary:missingTokens:i:+:missingOffset\n4:PRINT_CALL1_EXPR:missingPrintExpr:INT_ADD:missingLeft:missingRight\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingFunc\n1: undefined variable: missingArg\n2: undefined variable: missingPrint\n2: undefined variable: missingItems\n2: undefined variable: i\n3: undefined variable: missingPrintBinary\n3: undefined variable: missingTokens\n3: undefined variable: i\n3: undefined variable: missingOffset\n4: undefined variable: missingPrintExpr\n4: undefined variable: missingLeft\n4: undefined variable: missingRight\n"
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
	if err := os.WriteFile(path, []byte("1:WHILE_COMPARE_LT:IDENT:left:IDENT:right\n2:WHILE_COMPARE_NE:IDENT:left:IDENT:right\n3:WHILE_COMPARE_GE:IDENT:left:IDENT:right\n4:WHILE_COMPARE_GT:IDENT:left:IDENT:right\n5:WHILE_COMPARE_LE:IDENT:left:IDENT:right\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: left\n1: undefined variable: right\n2: undefined variable: left\n2: undefined variable: right\n3: undefined variable: left\n3: undefined variable: right\n4: undefined variable: left\n4: undefined variable: right\n5: undefined variable: left\n5: undefined variable: right\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerRejectsUndefinedCompareConditionNames(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	if err := os.WriteFile(path, []byte("1:IF_COMPARE_EQ:IDENT:left:IDENT:right\n2:IF_COMPARE_NE:IDENT:left:IDENT:right\n3:IF_COMPARE_GE:IDENT:left:IDENT:right\n4:IF_COMPARE_GT:IDENT:left:IDENT:right\n5:IF_COMPARE_LE:IDENT:left:IDENT:right\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: left\n1: undefined variable: right\n2: undefined variable: left\n2: undefined variable: right\n3: undefined variable: left\n3: undefined variable: right\n4: undefined variable: left\n4: undefined variable: right\n5: undefined variable: left\n5: undefined variable: right\n"
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
	if err := os.WriteFile(path, []byte("1:FOR:item:missingItems\n2:FOR_INDEX:item:index:missingIndexedItems\n3:FOR:item:missingGroups.active\n4:FOR_INDEX:item:index:missingIndexedGroups.active\n5:FOR:item:missingGroups[0]\n6:FOR_INDEX:item:index:missingIndexedGroups[0]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: undefined variable: missingItems\n2: undefined variable: missingIndexedItems\n3: undefined variable: missingGroups\n4: undefined variable: missingIndexedGroups\n5: undefined variable: missingGroups\n6: undefined variable: missingIndexedGroups\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSelfhostCheckerKeepsBlockLocalNamesScoped(t *testing.T) {
	path := t.TempDir() + "/nodes.txt"
	nodes := "1:INDENT:0\n1:IF:BOOL:true\n2:INDENT:2\n2:ASSIGN:local_if:INT:1\n3:INDENT:0\n3:PRINT:IDENT:local_if\n4:WHILE:BOOL:true\n5:INDENT:2\n5:ASSIGN:local_while:INT:1\n6:INDENT:0\n6:PRINT:IDENT:local_while\n7:ASSIGN:items:ARRAY_EMPTY:\n8:FOR:item:items\n9:INDENT:2\n9:PRINT:IDENT:item\n10:INDENT:0\n10:PRINT:IDENT:item\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "3: undefined variable: local_if\n6: undefined variable: local_while\n10: undefined variable: item\n"
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
	nodes := "1:FUNC:show:User\n2:FUNC2:show2:userName:ok\n3:ASSIGN:badName:INT:1\n4:ASSIGN:items:ARRAY_EMPTY:\n5:FOR:Item:items\n"
	if err := os.WriteFile(path, []byte(nodes), 0644); err != nil {
		t.Fatal(err)
	}
	out := run(t, "go", "run", "./cmd/tya", "selfhost/checker.tya", path)
	want := "1: invalid binding name: User\n2: invalid binding name: userName\n3: invalid binding name: badName\n5: invalid binding name: Item\n"
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
	if name == "cc" {
		// glibc needs explicit -lpthread / -lm for the runtime's
		// thread + math intrinsics. macOS rolls both into libSystem
		// so the flags are harmless there.
		switch goruntime.GOOS {
		case "linux":
			args = append(args, "-lpthread", "-lm")
		case "windows":
			// no extra flags
		default:
			args = append(args, "-lm")
		}
	}
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
	prog, _, err := parser.Parse(toks)
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
	prog, _, err := parser.Parse(toks)
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
					if obj, ok := member.Target.(*ast.Ident); ok {
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
		if n.IndexName != "" {
			*out = append(*out, "FOR_INDEX:"+n.ValueName+":"+n.IndexName+":"+summarizeGoScalar(n.Iterable))
		} else {
			*out = append(*out, "FOR:"+n.ValueName+":"+summarizeGoScalar(n.Iterable))
		}
		for _, child := range n.Body {
			summarizeGoStmt(out, child)
		}
	case *ast.ReturnStmt:
		if len(n.Values) == 2 {
			if obj, ok := n.Values[0].(*ast.TargetLit); ok {
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
	case *ast.TargetLit:
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
		case "-":
			return "INT_SUB:" + left + ":" + right
		case "*":
			return "INT_MUL:" + left + ":" + right
		case "/":
			return "INT_DIV:" + left + ":" + right
		case "%":
			return "INT_MOD:" + left + ":" + right
		case ">=":
			return "COMPARE_GE:" + left + ":" + right
		case ">":
			return "COMPARE_GT:" + left + ":" + right
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
		case ">":
			return "COMPARE_GT:" + summarizeGoKindedScalar(bin.Left) + ":" + summarizeGoKindedScalar(bin.Right)
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
