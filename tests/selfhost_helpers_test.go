//go:build selfhost_legacy && pre_v01_legacy_ast

package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"testing"

	"tya/internal/ast"
	"tya/internal/checker"
	"tya/internal/lexer"
	"tya/internal/parser"
	"tya/internal/token"
)

func runToFile(t *testing.T, path string, name string, args ...string) {
	t.Helper()
	out := run(t, name, args...)
	if err := os.WriteFile(path, out, 0644); err != nil {
		t.Fatal(err)
	}
}

func readSelfhostTestdata(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "selfhost", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
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
			if obj, ok := n.Values[0].(*ast.DictLit); ok {
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
	case *ast.DictLit:
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
