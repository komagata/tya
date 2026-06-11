//go:build selfhost_legacy && pre_v01_legacy_ast

package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
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
