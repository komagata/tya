package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"tya/internal/ast"
	"tya/internal/interp"
	"tya/internal/lexer"
	"tya/internal/parser"
)

func cName(name string) string {
	name = strings.ReplaceAll(name, "?", "_p")
	name = strings.ReplaceAll(name, "!", "_bang")
	switch name {
	case "auto", "break", "case", "char", "const", "continue", "default", "do", "double",
		"else", "enum", "extern", "float", "for", "goto", "if", "index", "inline", "int",
		"log", "long", "register", "restrict", "return", "short", "signed", "sizeof",
		"static", "struct", "switch", "typedef", "union", "unsigned", "void",
		"volatile", "while":
		return "tya_var_" + name
	}
	return name
}

func cFuncName(name string, serial int) string {
	return fmt.Sprintf("tya_fn_%s_%d", cName(name), serial)
}

func (g *cgen) interpolateString(value string) (string, error) {
	parts := []string{}
	var text strings.Builder
	flushText := func() {
		if text.Len() > 0 {
			parts = append(parts, "tya_string("+strconv.Quote(text.String())+")")
			text.Reset()
		}
	}
	for i := 0; i < len(value); {
		switch value[i] {
		case '{':
			if i+1 < len(value) && value[i+1] == '{' {
				text.WriteByte('{')
				i += 2
				continue
			}
			close := interp.FindExprEnd(value, i)
			if close < 0 {
				return "", fmt.Errorf("unclosed interpolation")
			}
			expr := strings.TrimSpace(value[i+1 : close])
			if expr == "" {
				return "", fmt.Errorf("empty interpolation")
			}
			compiled, err := g.interpolationExpr(expr)
			if err != nil {
				return "", err
			}
			flushText()
			parts = append(parts, "tya_to_string("+compiled+")")
			i = close + 1
		case '}':
			if i+1 < len(value) && value[i+1] == '}' {
				text.WriteByte('}')
				i += 2
				continue
			}
			return "", fmt.Errorf("unmatched '}' in string interpolation")
		default:
			text.WriteByte(value[i])
			i++
		}
	}
	flushText()
	if len(parts) == 0 {
		return "tya_string(\"\")", nil
	}
	expr := parts[0]
	for _, part := range parts[1:] {
		expr = "tya_add(" + expr + ", " + part + ")"
	}
	return expr, nil
}

func (g *cgen) interpolationExpr(expr string) (string, error) {
	expr = strings.TrimSpace(expr)
	if expr == "super()" {
		sym := g.inheritedMethodSym(g.superClass, g.methodName)
		if g.inClassMethod {
			sym = g.inheritedClassMethodSym(g.superClass, g.methodName)
		}
		if !g.inClassMethod && (g.methodName == "init" || g.methodName == "_init" || g.methodName == "initialize") && g.superClass != "" {
			if initSym, err := g.constructorSuperRunnerForCurrentClass(sym); err != nil {
				return "", err
			} else {
				sym = initSym
			}
		}
		if sym == "" && !g.inClassMethod {
			if (g.methodName == "init" || g.methodName == "_init" || g.methodName == "initialize") && g.superClass == "" {
				if initSym, err := g.interfaceInitializerRunnerForCurrentClass(); err != nil {
					return "", err
				} else {
					sym = initSym
				}
			}
		}
		if sym == "" && !g.inClassMethod {
			if ifaceSym, err := g.interfaceDefaultSymForCurrentClass(g.methodName); err != nil {
				return "", err
			} else {
				sym = ifaceSym
			}
		}
		if sym == "" && g.interfaceSuperSym != "" {
			sym = g.interfaceSuperSym
		}
		if sym == "" {
			return "tya_nil()", nil
		}
		return fmt.Sprintf("%s(__this, tya_missing(), tya_missing(), tya_missing(), tya_missing(), tya_missing(), tya_missing())", sym), nil
	}
	toks, errs := lexer.Lex(expr)
	if len(errs) > 0 {
		return "", fmt.Errorf("invalid interpolation expression: %w", errs[0])
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		return "", fmt.Errorf("invalid interpolation expression: %w", err)
	}
	if len(prog.Stmts) != 1 {
		return "", fmt.Errorf("interpolation must contain one expression")
	}
	stmt, ok := prog.Stmts[0].(*ast.ExprStmt)
	if !ok {
		return "", fmt.Errorf("interpolation must contain an expression")
	}
	value, _, err := g.expr(stmt.Expr)
	return value, err
}

func assignedNames(stmts []ast.Stmt) []string {
	seen := map[string]bool{}
	var names []string
	var walk func([]ast.Stmt)
	walk = func(stmts []ast.Stmt) {
		for _, stmt := range stmts {
			switch n := stmt.(type) {
			case *ast.ImportStmt:
				if n.Alias != "" && !seen[n.Alias] {
					seen[n.Alias] = true
					names = append(names, n.Alias)
				}
			case *ast.ImportBlockStmt:
				for _, imp := range n.Imports {
					if imp.Alias != "" && !seen[imp.Alias] {
						seen[imp.Alias] = true
						names = append(names, imp.Alias)
					}
				}
			case *ast.EmbedStmt:
				if !seen[n.Name] {
					seen[n.Name] = true
					names = append(names, n.Name)
				}
			case *ast.ModuleDecl:
				if !seen[n.Name] {
					seen[n.Name] = true
					names = append(names, n.Name)
				}
			case *ast.ClassDecl:
				if !seen[n.Name] {
					seen[n.Name] = true
					names = append(names, n.Name)
				}
			case *ast.AssignStmt:
				for _, target := range n.Targets {
					id, ok := target.(*ast.Ident)
					if !ok || seen[id.Name] {
						continue
					}
					seen[id.Name] = true
					names = append(names, id.Name)
				}
			case *ast.IfStmt:
				walk(n.Then)
				walk(n.Else)
			case *ast.WhileStmt:
				walk(n.Body)
			case *ast.ForInStmt:
				for _, name := range []string{n.ValueName, n.IndexName} {
					if name == "" || seen[name] {
						continue
					}
					seen[name] = true
					names = append(names, name)
				}
				walk(n.Body)
			case *ast.TryCatchStmt:
				walk(n.Try)
				walk(n.Catch)
				walk(n.Finally)
			case *ast.MatchStmt:
				for _, c := range n.Cases {
					walk(c.Body)
				}
			}
		}
	}
	walk(stmts)
	return names
}

func (g *cgen) standardModuleCall(module string, name string, argExprs []ast.Expr) (string, error) {
	args := make([]string, 0, len(argExprs))
	for _, arg := range argExprs {
		ex, _, err := g.expr(arg)
		if err != nil {
			return "", err
		}
		args = append(args, ex)
	}
	switch module {
	case "string":
		if call := standardStringCall(name, args); call != "" {
			return call, nil
		}
		return "", nil
	case "array":
		if call := standardArrayCall(name, args); call != "" {
			return call, nil
		}
		return "", nil
	case "dict":
		if call := standardDictCall(name, args); call != "" {
			return call, nil
		}
		return "", nil
	case "value":
		if name == "nil?" && len(args) == 1 {
			return fmt.Sprintf("tya_bool(%s.kind == TYA_NIL)", args[0]), nil
		}
		return "", nil
	default:
		return "", nil
	}
}

func standardStringCall(name string, args []string) string {
	switch name {
	case "len", "char_len":
		if len(args) == 1 {
			return fmt.Sprintf("tya_len(%s)", args[0])
		}
	case "byte_len":
		if len(args) == 1 {
			return fmt.Sprintf("tya_byte_len(%s)", args[0])
		}
	case "slice":
		if len(args) == 3 {
			return fmt.Sprintf("tya_string_slice(%s, %s, %s)", args[0], args[1], args[2])
		}
	case "trim":
		if len(args) == 1 {
			return fmt.Sprintf("tya_trim(%s)", args[0])
		}
	case "contains":
		if len(args) == 2 {
			return fmt.Sprintf("tya_contains(%s, %s)", args[0], args[1])
		}
	case "index_of":
		if len(args) == 2 {
			return fmt.Sprintf("tya_string_index_of(%s, %s, tya_number(0))", args[0], args[1])
		}
		if len(args) == 3 {
			return fmt.Sprintf("tya_string_index_of(%s, %s, %s)", args[0], args[1], args[2])
		}
	case "starts_with":
		if len(args) == 2 {
			return fmt.Sprintf("tya_starts_with(%s, %s)", args[0], args[1])
		}
	case "ends_with":
		if len(args) == 2 {
			return fmt.Sprintf("tya_ends_with(%s, %s)", args[0], args[1])
		}
	case "replace":
		if len(args) == 3 {
			return fmt.Sprintf("tya_replace(%s, %s, %s)", args[0], args[1], args[2])
		}
	case "split":
		if len(args) == 2 {
			return fmt.Sprintf("tya_split(%s, %s)", args[0], args[1])
		}
	case "join":
		if len(args) == 2 {
			return fmt.Sprintf("tya_join(%s, %s)", args[0], args[1])
		}
	case "lines":
		if len(args) == 1 {
			return fmt.Sprintf("tya_lines(%s)", args[0])
		}
	case "chars":
		if len(args) == 1 {
			return fmt.Sprintf("tya_chars(%s)", args[0])
		}
	case "bytes":
		if len(args) == 1 {
			return fmt.Sprintf("tya_bytes_of(%s)", args[0])
		}
	case "upcase":
		if len(args) == 1 {
			return fmt.Sprintf("tya_upcase(%s)", args[0])
		}
	case "downcase":
		if len(args) == 1 {
			return fmt.Sprintf("tya_downcase(%s)", args[0])
		}
	}
	return ""
}

func standardArrayCall(name string, args []string) string {
	switch name {
	case "len":
		if len(args) == 1 {
			return fmt.Sprintf("tya_len(%s)", args[0])
		}
	case "empty?":
		if len(args) == 1 {
			return fmt.Sprintf("tya_bool((int)tya_len(%s).number == 0)", args[0])
		}
	case "first":
		if len(args) == 1 {
			return fmt.Sprintf("tya_first(%s)", args[0])
		}
	case "last":
		if len(args) == 1 {
			return fmt.Sprintf("tya_last(%s)", args[0])
		}
	case "push":
		if len(args) == 2 {
			return fmt.Sprintf("tya_array_push(%s, %s)", args[0], args[1])
		}
	case "pop":
		if len(args) == 1 {
			return fmt.Sprintf("tya_pop(%s)", args[0])
		}
	case "slice":
		if len(args) == 3 {
			return fmt.Sprintf("tya_slice(%s, %s, %s)", args[0], args[1], args[2])
		}
	case "reverse":
		if len(args) == 1 {
			return fmt.Sprintf("tya_reverse(%s)", args[0])
		}
	case "contains":
		if len(args) == 2 {
			return fmt.Sprintf("tya_array_contains(%s, %s)", args[0], args[1])
		}
	case "join":
		if len(args) == 2 {
			return fmt.Sprintf("tya_join(%s, %s)", args[0], args[1])
		}
	case "map":
		if len(args) == 2 {
			return fmt.Sprintf("tya_map(%s, %s)", args[0], args[1])
		}
	case "filter":
		if len(args) == 2 {
			return fmt.Sprintf("tya_filter(%s, %s)", args[0], args[1])
		}
	case "find":
		if len(args) == 2 {
			return fmt.Sprintf("tya_find(%s, %s)", args[0], args[1])
		}
	case "any":
		if len(args) == 2 {
			return fmt.Sprintf("tya_any(%s, %s)", args[0], args[1])
		}
	case "all":
		if len(args) == 2 {
			return fmt.Sprintf("tya_all(%s, %s)", args[0], args[1])
		}
	case "each":
		if len(args) == 2 {
			return fmt.Sprintf("tya_each(%s, %s)", args[0], args[1])
		}
	case "reduce":
		if len(args) == 3 {
			return fmt.Sprintf("tya_reduce(%s, %s, %s)", args[0], args[1], args[2])
		}
	}
	return ""
}

func standardDictCall(name string, args []string) string {
	switch name {
	case "len":
		if len(args) == 1 {
			return fmt.Sprintf("tya_len(%s)", args[0])
		}
	case "has", "has?":
		if len(args) == 2 {
			return fmt.Sprintf("tya_has(%s, %s)", args[0], args[1])
		}
	case "get":
		if len(args) == 2 {
			return fmt.Sprintf("tya_dict_get(%s, %s, tya_nil(), false)", args[0], args[1])
		}
		if len(args) == 3 {
			return fmt.Sprintf("tya_dict_get(%s, %s, %s, true)", args[0], args[1], args[2])
		}
	case "set":
		if len(args) == 3 {
			return fmt.Sprintf("tya_dict_set(%s, %s, %s)", args[0], args[1], args[2])
		}
	case "delete":
		if len(args) == 2 {
			return fmt.Sprintf("tya_dict_delete(%s, %s)", args[0], args[1])
		}
	case "keys":
		if len(args) == 1 {
			return fmt.Sprintf("tya_keys(%s)", args[0])
		}
	case "values":
		if len(args) == 1 {
			return fmt.Sprintf("tya_values(%s)", args[0])
		}
	case "merge":
		if len(args) == 2 {
			return fmt.Sprintf("tya_dict_merge(%s, %s)", args[0], args[1])
		}
	case "merge!":
		if len(args) == 2 {
			return fmt.Sprintf("tya_dict_merge_bang(%s, %s)", args[0], args[1])
		}
	case "entries":
		if len(args) == 1 {
			return fmt.Sprintf("tya_dict_entries(%s)", args[0])
		}
	}
	return ""
}

func patternBindings(pattern ast.Expr) []string {
	seen := map[string]bool{}
	var names []string
	var walk func(ast.Expr)
	walk = func(pattern ast.Expr) {
		switch n := pattern.(type) {
		case *ast.Ident:
			if n.Name != "_" && !seen[n.Name] {
				seen[n.Name] = true
				names = append(names, n.Name)
			}
		case *ast.ArrayLit:
			for _, elem := range n.Elems {
				walk(elem)
			}
		case *ast.DictLit:
			for _, prop := range n.Props {
				walk(prop.Value)
			}
		}
	}
	walk(pattern)
	return names
}

// v0.24 native builtin codegen (returns C expression or "" if not v0.24).
func v24Codegen(g *cgen, name string, args []ast.Expr) string {
	emit := func(format string, n int) string {
		if len(args) != n {
			return ""
		}
		out := make([]string, n)
		for i, a := range args {
			expr, _, err := g.expr(a)
			if err != nil {
				return ""
			}
			out[i] = expr
		}
		switch n {
		case 0:
			return format
		case 1:
			return fmt.Sprintf(format, out[0])
		case 2:
			return fmt.Sprintf(format, out[0], out[1])
		case 3:
			return fmt.Sprintf(format, out[0], out[1], out[2])
		case 4:
			return fmt.Sprintf(format, out[0], out[1], out[2], out[3])
		case 5:
			return fmt.Sprintf(format, out[0], out[1], out[2], out[3], out[4])
		}
		return ""
	}
	switch name {
	case "time_now":
		return emit("tya_time_now()", 0)
	case "time_monotonic":
		return emit("tya_time_monotonic()", 0)
	case "time_unix":
		return emit("tya_time_unix(%s, %s)", 2)
	case "time_duration":
		return emit("tya_time_duration(%s, %s)", 2)
	case "time_sleep":
		return emit("tya_time_sleep(%s)", 1)
	case "time_format":
		if len(args) == 1 {
			return emit("tya_time_format(%s, tya_nil(), false)", 1)
		}
		if len(args) == 2 {
			a, _, err := g.expr(args[0])
			if err != nil {
				return ""
			}
			b, _, err := g.expr(args[1])
			if err != nil {
				return ""
			}
			return fmt.Sprintf("tya_time_format(%s, %s, true)", a, b)
		}
	case "time_parse":
		if len(args) == 1 {
			return emit("tya_time_parse(%s, tya_nil(), false)", 1)
		}
		if len(args) == 2 {
			a, _, err := g.expr(args[0])
			if err != nil {
				return ""
			}
			b, _, err := g.expr(args[1])
			if err != nil {
				return ""
			}
			return fmt.Sprintf("tya_time_parse(%s, %s, true)", a, b)
		}
	case "time_since":
		return emit("tya_time_since(%s)", 1)
	case "random_seed":
		return emit("tya_random_seed(%s)", 1)
	case "random_int":
		return emit("tya_random_int(%s, %s)", 2)
	case "random_float":
		return emit("tya_random_float()", 0)
	case "serialization_kind":
		return emit("tya_serialization_kind(%s)", 1)
	case "serialization_id":
		return emit("tya_serialization_id(%s)", 1)
	case "serialization_public_fields":
		return emit("tya_serialization_public_fields(%s)", 1)
	case "serialization_has_member":
		return emit("tya_serialization_has_member(%s, %s)", 2)
	case "compiler_lexer_lex":
		return emit("tya_compiler_lexer_lex(%s)", 1)
	case "compiler_lexer_lex_with_comments":
		return emit("tya_compiler_lexer_lex_with_comments(%s)", 1)
	case "compiler_parser_parse":
		return emit("tya_compiler_parser_parse(%s)", 1)
	case "compiler_parser_parse_tokens":
		return emit("tya_compiler_parser_parse_tokens(%s)", 1)
	case "compiler_ast_children":
		return emit("tya_compiler_ast_children(%s)", 1)
	case "compiler_ast_kind":
		return emit("tya_compiler_ast_kind(%s)", 1)
	case "compiler_ast_span":
		return emit("tya_compiler_ast_span(%s)", 1)
	case "compiler_checker_check":
		return emit("tya_compiler_checker_check(%s)", 1)
	case "compiler_checker_check_ast":
		return emit("tya_compiler_checker_check_ast(%s)", 1)
	case "compiler_format_format":
		return emit("tya_compiler_format_format(%s)", 1)
	case "compiler_format_unparse":
		return emit("tya_compiler_format_unparse(%s)", 1)
	case "math_sqrt":
		return emit("tya_math_sqrt(%s)", 1)
	case "math_pow":
		return emit("tya_math_pow(%s, %s)", 2)
	case "math_floor":
		return emit("tya_math_floor(%s)", 1)
	case "math_ceil":
		return emit("tya_math_ceil(%s)", 1)
	case "math_round":
		return emit("tya_math_round(%s)", 1)
	case "math_trunc":
		return emit("tya_math_trunc(%s)", 1)
	case "math_log":
		return emit("tya_math_log(%s)", 1)
	case "math_log2":
		return emit("tya_math_log2(%s)", 1)
	case "math_log10":
		return emit("tya_math_log10(%s)", 1)
	case "math_exp":
		return emit("tya_math_exp(%s)", 1)
	case "math_sin":
		return emit("tya_math_sin(%s)", 1)
	case "math_cos":
		return emit("tya_math_cos(%s)", 1)
	case "math_tan":
		return emit("tya_math_tan(%s)", 1)
	case "math_asin":
		return emit("tya_math_asin(%s)", 1)
	case "math_acos":
		return emit("tya_math_acos(%s)", 1)
	case "math_atan":
		return emit("tya_math_atan(%s)", 1)
	case "math_atan2":
		return emit("tya_math_atan2(%s, %s)", 2)
	case "process_run":
		if len(args) == 1 {
			a, _, err := g.expr(args[0])
			if err != nil {
				return ""
			}
			return fmt.Sprintf("tya_process_run(%s, tya_nil())", a)
		}
		if len(args) == 2 {
			return emit("tya_process_run(%s, %s)", 2)
		}
	case "process_exec":
		if len(args) == 1 {
			a, _, err := g.expr(args[0])
			if err != nil {
				return ""
			}
			return fmt.Sprintf("tya_process_exec(%s, tya_nil())", a)
		}
		if len(args) == 2 {
			return emit("tya_process_exec(%s, %s)", 2)
		}
	case "environ":
		return emit("tya_environ()", 0)
	case "setenv":
		return emit("tya_setenv(%s, %s)", 2)
	case "unsetenv":
		return emit("tya_unsetenv(%s)", 1)
	case "digest_md5":
		return emit("tya_digest_md5(%s)", 1)
	case "digest_sha1":
		return emit("tya_digest_sha1(%s)", 1)
	case "digest_sha256":
		return emit("tya_digest_sha256(%s)", 1)
	case "digest_sha384":
		return emit("tya_digest_sha384(%s)", 1)
	case "digest_sha512":
		return emit("tya_digest_sha512(%s)", 1)
	case "regex_compile":
		return emit("tya_regex_compile(%s, %s)", 2)
	case "secure_random_bytes":
		return emit("tya_secure_random_bytes(%s)", 1)
	case "secure_random_int":
		return emit("tya_secure_random_int(%s, %s)", 2)
	case "runtime_gc_stats":
		return emit("tya_gc_stats()", 0)
	case "runtime_gc_collect":
		return "(tya_gc_collect(), tya_nil())"
	case "channel_new":
		return emit("tya_channel_new(%s)", 1)
	case "channel_send":
		return emit("tya_channel_send(%s, %s)", 2)
	case "channel_receive":
		return emit("tya_channel_receive(%s)", 1)
	case "channel_receive_timeout":
		return emit("tya_channel_receive_timeout(%s, %s)", 2)
	case "channel_close":
		return emit("tya_channel_close(%s)", 1)
	case "channel_closed_p":
		return emit("tya_channel_closed(%s)", 1)
	case "channel_select":
		return emit("tya_channel_select(%s)", 1)
	case "task_cancel":
		return emit("tya_task_cancel(%s)", 1)
	case "task_is_cancelled_p":
		return emit("tya_task_is_cancelled(%s)", 1)
	case "task_current":
		return emit("tya_current_task()", 0)
	case "sync_mutex_new":
		return emit("tya_sync_mutex_new()", 0)
	case "sync_lock":
		return emit("tya_sync_lock(%s)", 1)
	case "sync_unlock":
		return emit("tya_sync_unlock(%s)", 1)
	case "sync_atomic_integer_new":
		return emit("tya_sync_atomic_integer_new(%s)", 1)
	case "sync_atomic_integer_add":
		return emit("tya_sync_atomic_integer_add(%s, %s)", 2)
	case "sync_atomic_integer_load":
		return emit("tya_sync_atomic_integer_load(%s)", 1)
	case "sync_atomic_integer_store":
		return emit("tya_sync_atomic_integer_store(%s, %s)", 2)
	case "sync_atomic_integer_cas":
		return emit("tya_sync_atomic_integer_cas(%s, %s, %s)", 3)
	case "sync_wait_group_new":
		return emit("tya_sync_wait_group_new()", 0)
	case "sync_wait_group_add":
		return emit("tya_sync_wait_group_add(%s, %s)", 2)
	case "sync_wait_group_done":
		return emit("tya_sync_wait_group_done(%s)", 1)
	case "sync_wait_group_wait":
		return emit("tya_sync_wait_group_wait(%s)", 1)
	case "bytes":
		return emit("tya_bytes_from_array(%s)", 1)
	case "bytes_of":
		return emit("tya_bytes_of(%s)", 1)
	case "bytes_text":
		return emit("tya_bytes_text(%s)", 1)
	case "bytes_array":
		return emit("tya_bytes_array(%s)", 1)
	case "bytes_concat":
		return emit("tya_bytes_concat(%s, %s)", 2)
	case "bytes_slice":
		return emit("tya_bytes_slice(%s, %s, %s)", 3)
	case "file_read_bytes":
		return emit("tya_file_read_bytes(%s)", 1)
	case "file_write_bytes":
		return emit("tya_file_write_bytes(%s, %s)", 2)
	case "file_copy":
		if len(args) == 2 {
			a, _, err := g.expr(args[0])
			if err != nil {
				return ""
			}
			b, _, err := g.expr(args[1])
			if err != nil {
				return ""
			}
			return fmt.Sprintf("tya_file_copy(%s, %s, tya_nil())", a, b)
		}
		if len(args) == 3 {
			return emit("tya_file_copy(%s, %s, %s)", 3)
		}
	case "file_chmod":
		return emit("tya_file_chmod(%s, %s)", 2)
	case "file_temp":
		if len(args) == 0 {
			return "tya_file_temp(tya_string(\"tya\"), tya_string(\"\"))"
		}
		if len(args) == 1 {
			a, _, err := g.expr(args[0])
			if err != nil {
				return ""
			}
			return fmt.Sprintf("tya_file_temp(%s, tya_string(\"\"))", a)
		}
		if len(args) == 2 {
			return emit("tya_file_temp(%s, %s)", 2)
		}
	case "dir_mkdir_all":
		return emit("tya_dir_mkdir_all(%s)", 1)
	case "dir_remove_all":
		return emit("tya_dir_remove_all(%s)", 1)
	case "dir_temp_dir":
		if len(args) == 0 {
			return "tya_dir_temp_dir(tya_string(\"tya\"))"
		}
		if len(args) == 1 {
			return emit("tya_dir_temp_dir(%s)", 1)
		}
	case "dir_walk":
		if len(args) == 2 {
			a, _, err := g.expr(args[0])
			if err != nil {
				return ""
			}
			b, _, err := g.expr(args[1])
			if err != nil {
				return ""
			}
			return fmt.Sprintf("tya_dir_walk(%s, %s, tya_nil())", a, b)
		}
		if len(args) == 3 {
			return emit("tya_dir_walk(%s, %s, %s)", 3)
		}
	case "binary_read_f32":
		return emit("tya_binary_read_f32(%s, %s, %s)", 3)
	case "binary_read_f64":
		return emit("tya_binary_read_f64(%s, %s, %s)", 3)
	case "binary_write_f32":
		return emit("tya_binary_write_f32(%s, %s)", 2)
	case "binary_write_f64":
		return emit("tya_binary_write_f64(%s, %s)", 2)
	case "stderr_write":
		return emit("tya_stderr_write(%s)", 1)
	case "file_append":
		return emit("tya_file_append(%s, %s)", 2)
	case "compress_gzip":
		return emit("tya_compress_gzip(%s)", 1)
	case "compress_gunzip":
		return emit("tya_compress_gunzip(%s)", 1)
	case "compress_zlib":
		return emit("tya_compress_zlib(%s)", 1)
	case "compress_unzlib":
		return emit("tya_compress_unzlib(%s)", 1)
	case "io_stdin":
		return emit("tya_io_stdin()", 0)
	case "io_stdout":
		return emit("tya_io_stdout()", 0)
	case "io_stderr":
		return emit("tya_io_stderr()", 0)
	case "io_open":
		return emit("tya_io_open(%s, %s)", 2)
	case "io_stream_read":
		return emit("tya_io_stream_read(%s, %s)", 2)
	case "io_stream_read_line":
		return emit("tya_io_stream_read_line(%s)", 1)
	case "io_stream_eof":
		return emit("tya_io_stream_eof(%s)", 1)
	case "io_stream_write":
		return emit("tya_io_stream_write(%s, %s)", 2)
	case "io_stream_flush":
		return emit("tya_io_stream_flush(%s)", 1)
	case "io_stream_close":
		return emit("tya_io_stream_close(%s)", 1)
	case "socket_connect":
		return emit("tya_socket_connect(%s, %s, %s)", 3)
	case "tls_connect":
		return emit("tya_tls_connect(%s, %s, %s)", 3)
	case "socket_server_listen":
		return emit("tya_socket_server_listen(%s, %s, %s)", 3)
	case "socket_server_accept":
		return emit("tya_socket_server_accept(%s)", 1)
	case "socket_read":
		return emit("tya_socket_read(%s, %s)", 2)
	case "socket_read_line":
		return emit("tya_socket_read_line(%s)", 1)
	case "socket_write":
		return emit("tya_socket_write(%s, %s)", 2)
	case "socket_close":
		return emit("tya_socket_close(%s)", 1)
	case "socket_closed":
		return emit("tya_socket_closed(%s)", 1)
	case "socket_local_address":
		return emit("tya_socket_local_address(%s)", 1)
	case "socket_remote_address":
		return emit("tya_socket_remote_address(%s)", 1)
	case "socket_server_close":
		return emit("tya_socket_server_close(%s)", 1)
	case "socket_server_local_address":
		return emit("tya_socket_server_local_address(%s)", 1)
	case "http_server_run":
		return emit("tya_http_server_run(%s, %s)", 2)
	case "http_server_run_tls":
		return emit("tya_http_server_run_tls(%s, %s, %s, %s, %s)", 5)
	}
	return ""
}
