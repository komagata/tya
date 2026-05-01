#!/usr/bin/env sh
set -eu

out_dir="$(mktemp -d "${TMPDIR:-/tmp}/tya-stage1-selfhost-sources.XXXXXX")"

mkdir -p "$out_dir"

compare_stage2_codegen() {
  label="$1"
  nodes="$2"
  file_label="$(printf '%s' "$label" | tr '/ ' '__')"
  "$out_dir/codegen_c.stage2" "$nodes" > "$out_dir/$file_label.stage2.first.c"
  "$out_dir/codegen_c.stage2" "$nodes" > "$out_dir/$file_label.stage2.second.c"
  diff -u "$out_dir/$file_label.stage2.first.c" "$out_dir/$file_label.stage2.second.c" >/dev/null
  echo "$label: stage-2 codegen deterministic"
}

for src in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  base="$(basename "$src" .tya)"
  go run ./cmd/tya --emit-c "$src" > "$out_dir/$base.stage1.c"
  gcc "$out_dir/$base.stage1.c" runtime/tya_runtime.c -I runtime -o "$out_dir/$base.stage1"
done

for src in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  base="$(basename "$src" .tya)"
  "$out_dir/lexer.stage1" "$src" > "$out_dir/$base.tokens"
  "$out_dir/parser.stage1" "$out_dir/$base.tokens" > "$out_dir/$base.nodes"
  "$out_dir/checker.stage1" "$out_dir/$base.nodes" > "$out_dir/$base.check"
  grep -qx "ok" "$out_dir/$base.check"
  "$out_dir/codegen_c.stage1" "$out_dir/$base.nodes" > "$out_dir/$base.stage2.c"
  test -s "$out_dir/$base.stage2.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/$base.stage2" "$out_dir/$base.stage2.c" >/dev/null 2>&1
  echo "$src: stage-1 emitted and compiled C"
done

"$out_dir/lexer.stage2" examples/hello.tya > "$out_dir/hello.stage2.tokens"
cat > "$out_dir/hello.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:print:1
1:STRING:Hello, Tya:7
TOKENS
diff -u "$out_dir/hello.want.tokens" "$out_dir/hello.stage2.tokens" >/dev/null
echo "examples/hello.tya: stage-2 lexer matched"

"$out_dir/parser.stage2" "$out_dir/hello.stage2.tokens" > "$out_dir/hello.stage2.nodes"
cat > "$out_dir/hello.want.nodes" <<'NODES'
1:PRINT:STRING:Hello, Tya
NODES
diff -u "$out_dir/hello.want.nodes" "$out_dir/hello.stage2.nodes" >/dev/null
echo "examples/hello.tya: stage-2 parser matched"

"$out_dir/checker.stage2" "$out_dir/hello.stage2.nodes" > "$out_dir/hello.stage2.check"
grep -qx "ok" "$out_dir/hello.stage2.check"
echo "examples/hello.tya: stage-2 checker matched"

compare_stage2_codegen "examples/hello.tya" "$out_dir/hello.stage2.nodes"

"$out_dir/codegen_c.stage2" "$out_dir/hello.stage2.nodes" > "$out_dir/hello.stage2.c"
cat > "$out_dir/hello.want.c" <<'C'
#include <stdio.h>

int main(void) {
  puts("Hello, Tya");
  return 0;
}
C
diff -u "$out_dir/hello.want.c" "$out_dir/hello.stage2.c" >/dev/null
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/hello.stage2" "$out_dir/hello.stage2.c"
hello_out="$("$out_dir/hello.stage2")"
test "$hello_out" = "Hello, Tya"
echo "examples/hello.tya: stage-2 codegen matched"

printf 'print "quote: \\"tya\\""\n' > "$out_dir/escaped_print.tya"
"$out_dir/lexer.stage2" "$out_dir/escaped_print.tya" > "$out_dir/escaped_print.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/escaped_print.stage2.tokens" > "$out_dir/escaped_print.stage2.nodes"
cat > "$out_dir/escaped_print.want.nodes" <<'NODES'
1:PRINT:STRING:quote: "tya"
NODES
diff -u "$out_dir/escaped_print.want.nodes" "$out_dir/escaped_print.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/escaped_print.stage2.nodes" > "$out_dir/escaped_print.stage2.check"
grep -qx "ok" "$out_dir/escaped_print.stage2.check"
compare_stage2_codegen "escaped string print" "$out_dir/escaped_print.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/escaped_print.stage2.nodes" > "$out_dir/escaped_print.stage2.c"
cat > "$out_dir/escaped_print.want.c" <<'C'
#include <stdio.h>

int main(void) {
  puts("quote: \"tya\"");
  return 0;
}
C
diff -u "$out_dir/escaped_print.want.c" "$out_dir/escaped_print.stage2.c" >/dev/null
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/escaped_print.stage2" "$out_dir/escaped_print.stage2.c" >/dev/null 2>&1
escaped_print_out="$("$out_dir/escaped_print.stage2")"
test "$escaped_print_out" = 'quote: "tya"'
echo "escaped string print: stage-2 pipeline matched"

printf 'print "tya"[1]\n' > "$out_dir/string_index_print.tya"
"$out_dir/lexer.stage2" "$out_dir/string_index_print.tya" > "$out_dir/string_index_print.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/string_index_print.stage2.tokens" > "$out_dir/string_index_print.stage2.nodes"
cat > "$out_dir/string_index_print.want.nodes" <<'NODES'
1:PRINT_INDEX:STRING:tya:1
NODES
diff -u "$out_dir/string_index_print.want.nodes" "$out_dir/string_index_print.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/string_index_print.stage2.nodes" > "$out_dir/string_index_print.stage2.check"
grep -qx "ok" "$out_dir/string_index_print.stage2.check"
compare_stage2_codegen "string index print" "$out_dir/string_index_print.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/string_index_print.stage2.nodes" > "$out_dir/string_index_print.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/string_index_print.stage2" "$out_dir/string_index_print.stage2.c" >/dev/null 2>&1
string_index_print_out="$("$out_dir/string_index_print.stage2")"
test "$string_index_print_out" = "y"
echo "string index print: stage-2 pipeline matched"

"$out_dir/lexer.stage2" examples/string.tya > "$out_dir/string_example.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/string_example.stage2.tokens" > "$out_dir/string_example.stage2.nodes"
"$out_dir/checker.stage2" "$out_dir/string_example.stage2.nodes" > "$out_dir/string_example.stage2.check"
grep -qx "ok" "$out_dir/string_example.stage2.check"
compare_stage2_codegen "examples/string.tya" "$out_dir/string_example.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/string_example.stage2.nodes" > "$out_dir/string_example.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/string_example.stage2" "$out_dir/string_example.stage2.c" >/dev/null 2>&1
string_example_out="$("$out_dir/string_example.stage2")"
test "$string_example_out" = "hello-tya
hello,Tya
true
true
true
6
2
quote: \"tya\"
y"
echo "examples/string.tya: stage-2 pipeline matched"

printf 'value = 20\n' > "$out_dir/int.tya"
"$out_dir/lexer.stage2" "$out_dir/int.tya" > "$out_dir/int.stage2.tokens"
cat > "$out_dir/int.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:value:1
1:SYMBOL:=:7
1:INT:20:9
TOKENS
diff -u "$out_dir/int.want.tokens" "$out_dir/int.stage2.tokens" >/dev/null
echo "int literal: stage-2 lexer matched"

printf 'same = value == 20\nfn = x -> x\n' > "$out_dir/operators.tya"
"$out_dir/lexer.stage2" "$out_dir/operators.tya" > "$out_dir/operators.stage2.tokens"
cat > "$out_dir/operators.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:same:1
1:SYMBOL:=:6
1:IDENT:value:8
1:SYMBOL:==:14
1:INT:20:17
2:INDENT:0:1
2:IDENT:fn:1
2:SYMBOL:=:4
2:IDENT:x:6
2:ARROW:->:8
2:IDENT:x:11
TOKENS
diff -u "$out_dir/operators.want.tokens" "$out_dir/operators.stage2.tokens" >/dev/null
echo "operators: stage-2 lexer matched"

printf 'ratio = 12.5\ntext = "a\\\"b"\n' > "$out_dir/literals.tya"
"$out_dir/lexer.stage2" "$out_dir/literals.tya" > "$out_dir/literals.stage2.tokens"
cat > "$out_dir/literals.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:ratio:1
1:SYMBOL:=:7
1:FLOAT:12.5:9
2:INDENT:0:1
2:IDENT:text:1
2:SYMBOL:=:6
2:STRING:a"b:8
TOKENS
diff -u "$out_dir/literals.want.tokens" "$out_dir/literals.stage2.tokens" >/dev/null
echo "float and string escape: stage-2 lexer matched"

"$out_dir/parser.stage2" "$out_dir/int.stage2.tokens" > "$out_dir/int.stage2.nodes"
cat > "$out_dir/int.want.nodes" <<'NODES'
1:ASSIGN:value:INT:20
NODES
diff -u "$out_dir/int.want.nodes" "$out_dir/int.stage2.nodes" >/dev/null
echo "int assignment: stage-2 parser matched"
compare_stage2_codegen "int assignment" "$out_dir/int.stage2.nodes"

"$out_dir/parser.stage2" "$out_dir/literals.stage2.tokens" > "$out_dir/literals.stage2.nodes"
cat > "$out_dir/literals.want.nodes" <<'NODES'
1:ASSIGN:ratio:FLOAT:12.5
2:ASSIGN:text:STRING:a"b
NODES
diff -u "$out_dir/literals.want.nodes" "$out_dir/literals.stage2.nodes" >/dev/null
echo "literal assignments: stage-2 parser matched"
compare_stage2_codegen "literal assignments" "$out_dir/literals.stage2.nodes"

"$out_dir/checker.stage2" "$out_dir/int.stage2.nodes" > "$out_dir/int.stage2.check"
grep -qx "ok" "$out_dir/int.stage2.check"
"$out_dir/checker.stage2" "$out_dir/literals.stage2.nodes" > "$out_dir/literals.stage2.check"
grep -qx "ok" "$out_dir/literals.stage2.check"
echo "literal assignments: stage-2 checker matched"

"$out_dir/codegen_c.stage2" "$out_dir/int.stage2.nodes" > "$out_dir/int.stage2.c"
cat > "$out_dir/int.want.c" <<'C'
#include <stdio.h>

int main(void) {
  long value = 20;
  return 0;
}
C
diff -u "$out_dir/int.want.c" "$out_dir/int.stage2.c" >/dev/null
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/int.stage2" "$out_dir/int.stage2.c" >/dev/null 2>&1
echo "int assignment: stage-2 codegen matched"

"$out_dir/codegen_c.stage2" "$out_dir/literals.stage2.nodes" > "$out_dir/literals.stage2.c"
cat > "$out_dir/literals.want.c" <<'C'
#include <stdio.h>

int main(void) {
  double ratio = 12.5;
  const char *text = "a\"b";
  return 0;
}
C
diff -u "$out_dir/literals.want.c" "$out_dir/literals.stage2.c" >/dev/null
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/literals.stage2" "$out_dir/literals.stage2.c" >/dev/null 2>&1
echo "literal assignments: stage-2 codegen matched"

printf 'value = 1\nvalue = 2\nflag = true\nflag = false\nratio = 1.5\nratio = 2.5\ntext = "a"\ntext = "b"\nprint value\nprint flag\nprint ratio\nprint text\n' > "$out_dir/literal_reassign.tya"
"$out_dir/lexer.stage2" "$out_dir/literal_reassign.tya" > "$out_dir/literal_reassign.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/literal_reassign.stage2.tokens" > "$out_dir/literal_reassign.stage2.nodes"
cat > "$out_dir/literal_reassign.want.nodes" <<'NODES'
1:ASSIGN:value:INT:1
2:ASSIGN:value:INT:2
3:ASSIGN:flag:BOOL:true
4:ASSIGN:flag:BOOL:false
5:ASSIGN:ratio:FLOAT:1.5
6:ASSIGN:ratio:FLOAT:2.5
7:ASSIGN:text:STRING:a
8:ASSIGN:text:STRING:b
9:PRINT:IDENT:value
10:PRINT:IDENT:flag
11:PRINT:IDENT:ratio
12:PRINT:IDENT:text
NODES
diff -u "$out_dir/literal_reassign.want.nodes" "$out_dir/literal_reassign.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/literal_reassign.stage2.nodes" > "$out_dir/literal_reassign.stage2.check"
grep -qx "ok" "$out_dir/literal_reassign.stage2.check"
compare_stage2_codegen "literal reassignment" "$out_dir/literal_reassign.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/literal_reassign.stage2.nodes" > "$out_dir/literal_reassign.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/literal_reassign.stage2" "$out_dir/literal_reassign.stage2.c" >/dev/null 2>&1
literal_reassign_out="$("$out_dir/literal_reassign.stage2")"
test "$literal_reassign_out" = "2
false
2.5
b"
echo "literal reassignment: stage-2 pipeline matched"

printf 'source = readFile args()[0]\nprint source\n' > "$out_dir/read_file_arg.tya"
printf 'Tya' > "$out_dir/read_file_arg.input"
"$out_dir/lexer.stage2" "$out_dir/read_file_arg.tya" > "$out_dir/read_file_arg.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/read_file_arg.stage2.tokens" > "$out_dir/read_file_arg.stage2.nodes"
cat > "$out_dir/read_file_arg.want.nodes" <<'NODES'
1:ASSIGN:source:CALL1_CALL0_INDEX:readFile:args:0
2:PRINT:IDENT:source
NODES
diff -u "$out_dir/read_file_arg.want.nodes" "$out_dir/read_file_arg.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/read_file_arg.stage2.nodes" > "$out_dir/read_file_arg.stage2.check"
grep -qx "ok" "$out_dir/read_file_arg.stage2.check"
compare_stage2_codegen "read file arg" "$out_dir/read_file_arg.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/read_file_arg.stage2.nodes" > "$out_dir/read_file_arg.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/read_file_arg.stage2" "$out_dir/read_file_arg.stage2.c" >/dev/null 2>&1
read_file_arg_out="$("$out_dir/read_file_arg.stage2" "$out_dir/read_file_arg.input")"
test "$read_file_arg_out" = "Tya"
echo "read file arg: stage-2 pipeline matched"

printf 'source = readFile args()[0]\ntokens = lex source\nfor token in tokens\n  print token\n' > "$out_dir/lex_source.tya"
printf 'print "Tya"\n' > "$out_dir/lex_source.input"
"$out_dir/lexer.stage2" "$out_dir/lex_source.tya" > "$out_dir/lex_source.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/lex_source.stage2.tokens" > "$out_dir/lex_source.stage2.nodes"
cat > "$out_dir/lex_source.want.nodes" <<'NODES'
1:ASSIGN:source:CALL1_CALL0_INDEX:readFile:args:0
2:ASSIGN:tokens:CALL1:lex:source
3:FOR:token:tokens
4:INDENT:2
4:PRINT:IDENT:token
NODES
diff -u "$out_dir/lex_source.want.nodes" "$out_dir/lex_source.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/lex_source.stage2.nodes" > "$out_dir/lex_source.stage2.check"
grep -qx "ok" "$out_dir/lex_source.stage2.check"
compare_stage2_codegen "lex source" "$out_dir/lex_source.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/lex_source.stage2.nodes" > "$out_dir/lex_source.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/lex_source.stage2" "$out_dir/lex_source.stage2.c" >/dev/null 2>&1
lex_source_out="$("$out_dir/lex_source.stage2" "$out_dir/lex_source.input")"
test "$lex_source_out" = "1:INDENT:0:1
1:IDENT:print:1
1:STRING:Tya:7"
echo "lex source: stage-2 pipeline matched"

long_lex_text="$(printf '%0300d' 0 | tr 0 a)"
printf 'print "%s"\n' "$long_lex_text" > "$out_dir/lex_source_long.input"
lex_source_long_out="$("$out_dir/lex_source.stage2" "$out_dir/lex_source_long.input")"
test "$lex_source_long_out" = "1:INDENT:0:1
1:IDENT:print:1
1:STRING:${long_lex_text}:7"
echo "long lex source: stage-2 pipeline matched"

printf 'source = readFile args()[0]\ntokens = lex source\nnodes = parse tokens\nfor node in nodes\n  print node\n' > "$out_dir/parse_tokens.tya"
printf 'print "Tya"\n' > "$out_dir/parse_tokens.input"
"$out_dir/lexer.stage2" "$out_dir/parse_tokens.tya" > "$out_dir/parse_tokens.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/parse_tokens.stage2.tokens" > "$out_dir/parse_tokens.stage2.nodes"
cat > "$out_dir/parse_tokens.want.nodes" <<'NODES'
1:ASSIGN:source:CALL1_CALL0_INDEX:readFile:args:0
2:ASSIGN:tokens:CALL1:lex:source
3:ASSIGN:nodes:CALL1:parse:tokens
4:FOR:node:nodes
5:INDENT:2
5:PRINT:IDENT:node
NODES
diff -u "$out_dir/parse_tokens.want.nodes" "$out_dir/parse_tokens.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/parse_tokens.stage2.nodes" > "$out_dir/parse_tokens.stage2.check"
grep -qx "ok" "$out_dir/parse_tokens.stage2.check"
compare_stage2_codegen "parse tokens" "$out_dir/parse_tokens.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/parse_tokens.stage2.nodes" > "$out_dir/parse_tokens.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/parse_tokens.stage2" "$out_dir/parse_tokens.stage2.c" >/dev/null 2>&1
parse_tokens_out="$("$out_dir/parse_tokens.stage2" "$out_dir/parse_tokens.input")"
test "$parse_tokens_out" = "1:PRINT:STRING:Tya"
echo "parse tokens: stage-2 pipeline matched"

printf 'source = readFile args()[0]\nlines = split source, "\n"\nerrors = check nodes\nfor err in errors\n  print err\n' > "$out_dir/check_nodes.tya"
printf '1:PRINT:STRING:Tya\n' > "$out_dir/check_nodes.input"
"$out_dir/lexer.stage2" "$out_dir/check_nodes.tya" > "$out_dir/check_nodes.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/check_nodes.stage2.tokens" > "$out_dir/check_nodes.stage2.nodes"
cat > "$out_dir/check_nodes.want.nodes" <<'NODES'
1:ASSIGN:source:CALL1_CALL0_INDEX:readFile:args:0
2:ASSIGN:lines:CALL2:split:source:STRING:
3:ASSIGN:errors:CALL1:check:nodes
4:FOR:err:errors
5:INDENT:2
5:PRINT:IDENT:err
NODES
diff -u "$out_dir/check_nodes.want.nodes" "$out_dir/check_nodes.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/check_nodes.stage2.nodes" > "$out_dir/check_nodes.stage2.check"
grep -qx "ok" "$out_dir/check_nodes.stage2.check"
compare_stage2_codegen "check nodes" "$out_dir/check_nodes.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/check_nodes.stage2.nodes" > "$out_dir/check_nodes.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/check_nodes.stage2" "$out_dir/check_nodes.stage2.c" >/dev/null 2>&1
check_nodes_out="$("$out_dir/check_nodes.stage2" "$out_dir/check_nodes.input")"
test "$check_nodes_out" = ""
echo "check nodes: stage-2 pipeline matched"

printf 'source = readFile args()[0]\nlines = split source, "\n"\nprint emitC nodes\n' > "$out_dir/emit_c_nodes.tya"
printf '1:PRINT:STRING:Tya\n' > "$out_dir/emit_c_nodes.input"
"$out_dir/lexer.stage2" "$out_dir/emit_c_nodes.tya" > "$out_dir/emit_c_nodes.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/emit_c_nodes.stage2.tokens" > "$out_dir/emit_c_nodes.stage2.nodes"
cat > "$out_dir/emit_c_nodes.want.nodes" <<'NODES'
1:ASSIGN:source:CALL1_CALL0_INDEX:readFile:args:0
2:ASSIGN:lines:CALL2:split:source:STRING:
3:PRINT_CALL1:emitC:nodes
NODES
diff -u "$out_dir/emit_c_nodes.want.nodes" "$out_dir/emit_c_nodes.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/emit_c_nodes.stage2.nodes" > "$out_dir/emit_c_nodes.stage2.check"
grep -qx "ok" "$out_dir/emit_c_nodes.stage2.check"
compare_stage2_codegen "emitC nodes" "$out_dir/emit_c_nodes.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/emit_c_nodes.stage2.nodes" > "$out_dir/emit_c_nodes.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/emit_c_nodes.stage2" "$out_dir/emit_c_nodes.stage2.c" >/dev/null 2>&1
"$out_dir/emit_c_nodes.stage2" "$out_dir/emit_c_nodes.input" > "$out_dir/emit_c_nodes.out.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/emit_c_nodes.out" "$out_dir/emit_c_nodes.out.c" >/dev/null 2>&1
emit_c_nodes_out="$("$out_dir/emit_c_nodes.out")"
test "$emit_c_nodes_out" = "Tya"
echo "emitC nodes: stage-2 pipeline matched"

printf 'helper = value ->\n  temp = "skip"\n  print temp\nsource = readFile args()[0]\nprint source\n' > "$out_dir/function_body_skip.tya"
"$out_dir/lexer.stage2" "$out_dir/function_body_skip.tya" > "$out_dir/function_body_skip.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/function_body_skip.stage2.tokens" > "$out_dir/function_body_skip.stage2.nodes"
cat > "$out_dir/function_body_skip.want.nodes" <<'NODES'
4:ASSIGN:source:CALL1_CALL0_INDEX:readFile:args:0
5:PRINT:IDENT:source
NODES
diff -u "$out_dir/function_body_skip.want.nodes" "$out_dir/function_body_skip.stage2.nodes" >/dev/null
echo "function body skip: stage-2 parser matched"

printf 'value = 20\nprint value\n' > "$out_dir/print_int.tya"
"$out_dir/lexer.stage2" "$out_dir/print_int.tya" > "$out_dir/print_int.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/print_int.stage2.tokens" > "$out_dir/print_int.stage2.nodes"
cat > "$out_dir/print_int.want.nodes" <<'NODES'
1:ASSIGN:value:INT:20
2:PRINT:IDENT:value
NODES
diff -u "$out_dir/print_int.want.nodes" "$out_dir/print_int.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/print_int.stage2.nodes" > "$out_dir/print_int.stage2.check"
grep -qx "ok" "$out_dir/print_int.stage2.check"
compare_stage2_codegen "print int assignment" "$out_dir/print_int.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/print_int.stage2.nodes" > "$out_dir/print_int.stage2.c"
cat > "$out_dir/print_int.want.c" <<'C'
#include <stdio.h>

int main(void) {
  long value = 20;
  printf("%ld\n", (long)value);
  return 0;
}
C
diff -u "$out_dir/print_int.want.c" "$out_dir/print_int.stage2.c" >/dev/null
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/print_int.stage2" "$out_dir/print_int.stage2.c" >/dev/null 2>&1
print_int_out="$("$out_dir/print_int.stage2")"
test "$print_int_out" = "20"
echo "print int assignment: stage-2 pipeline matched"

printf 'text = "hello"\nprint text\nratio = 12.5\nprint ratio\n' > "$out_dir/print_literals.tya"
"$out_dir/lexer.stage2" "$out_dir/print_literals.tya" > "$out_dir/print_literals.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/print_literals.stage2.tokens" > "$out_dir/print_literals.stage2.nodes"
cat > "$out_dir/print_literals.want.nodes" <<'NODES'
1:ASSIGN:text:STRING:hello
2:PRINT:IDENT:text
3:ASSIGN:ratio:FLOAT:12.5
4:PRINT:IDENT:ratio
NODES
diff -u "$out_dir/print_literals.want.nodes" "$out_dir/print_literals.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/print_literals.stage2.nodes" > "$out_dir/print_literals.stage2.check"
grep -qx "ok" "$out_dir/print_literals.stage2.check"
compare_stage2_codegen "print literal assignments" "$out_dir/print_literals.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/print_literals.stage2.nodes" > "$out_dir/print_literals.stage2.c"
cat > "$out_dir/print_literals.want.c" <<'C'
#include <stdio.h>

int main(void) {
  const char *text = "hello";
  puts(text);
  double ratio = 12.5;
  printf("%g\n", ratio);
  return 0;
}
C
diff -u "$out_dir/print_literals.want.c" "$out_dir/print_literals.stage2.c" >/dev/null
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/print_literals.stage2" "$out_dir/print_literals.stage2.c" >/dev/null 2>&1
print_literals_out="$("$out_dir/print_literals.stage2")"
test "$print_literals_out" = "hello
12.5"
echo "print literal assignments: stage-2 pipeline matched"

printf 'text = "hello"\nprint len text\n' > "$out_dir/string_len.tya"
"$out_dir/lexer.stage2" "$out_dir/string_len.tya" > "$out_dir/string_len.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/string_len.stage2.tokens" > "$out_dir/string_len.stage2.nodes"
cat > "$out_dir/string_len.want.nodes" <<'NODES'
1:ASSIGN:text:STRING:hello
2:PRINT_CALL1:len:text
NODES
diff -u "$out_dir/string_len.want.nodes" "$out_dir/string_len.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/string_len.stage2.nodes" > "$out_dir/string_len.stage2.check"
grep -qx "ok" "$out_dir/string_len.stage2.check"
compare_stage2_codegen "string len print" "$out_dir/string_len.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/string_len.stage2.nodes" > "$out_dir/string_len.stage2.c"
cat > "$out_dir/string_len.want.c" <<'C'
#include <stdio.h>
#include <string.h>

int main(void) {
  const char *text = "hello";
  printf("%ld\n", (long)strlen(text));
  return 0;
}
C
diff -u "$out_dir/string_len.want.c" "$out_dir/string_len.stage2.c" >/dev/null
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/string_len.stage2" "$out_dir/string_len.stage2.c" >/dev/null 2>&1
string_len_out="$("$out_dir/string_len.stage2")"
test "$string_len_out" = "5"
echo "string len print: stage-2 pipeline matched"

printf 'text = "  hello  "\ntrimmed = trim text\nprint trimmed\n' > "$out_dir/string_trim.tya"
"$out_dir/lexer.stage2" "$out_dir/string_trim.tya" > "$out_dir/string_trim.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/string_trim.stage2.tokens" > "$out_dir/string_trim.stage2.nodes"
cat > "$out_dir/string_trim.want.nodes" <<'NODES'
1:ASSIGN:text:STRING:  hello  
2:ASSIGN:trimmed:CALL1:trim:text
3:PRINT:IDENT:trimmed
NODES
diff -u "$out_dir/string_trim.want.nodes" "$out_dir/string_trim.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/string_trim.stage2.nodes" > "$out_dir/string_trim.stage2.check"
grep -qx "ok" "$out_dir/string_trim.stage2.check"
compare_stage2_codegen "string trim print" "$out_dir/string_trim.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/string_trim.stage2.nodes" > "$out_dir/string_trim.stage2.c"
cat > "$out_dir/string_trim.want.c" <<'C'
#include <stdio.h>
#include <string.h>
#include <stdlib.h>

static char *trim_text(const char *text) {
  const char *start = text;
  while (*start == ' ' || *start == 9 || *start == 10 || *start == 13) start++;
  const char *end = start + strlen(start);
  while (end > start && (end[-1] == ' ' || end[-1] == 9 || end[-1] == 10 || end[-1] == 13)) end--;
  size_t len = (size_t)(end - start);
  char *out = malloc(len + 1);
  memcpy(out, start, len);
  out[len] = 0;
  return out;
}

int main(void) {
  const char *text = "  hello  ";
  const char *trimmed = trim_text(text);
  puts(trimmed);
  return 0;
}
C
diff -u "$out_dir/string_trim.want.c" "$out_dir/string_trim.stage2.c" >/dev/null
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/string_trim.stage2" "$out_dir/string_trim.stage2.c" >/dev/null 2>&1
string_trim_out="$("$out_dir/string_trim.stage2")"
test "$string_trim_out" = "hello"
echo "string trim print: stage-2 pipeline matched"

printf 'text = "hello"\nprint contains text, "ell"\n' > "$out_dir/string_contains.tya"
"$out_dir/lexer.stage2" "$out_dir/string_contains.tya" > "$out_dir/string_contains.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/string_contains.stage2.tokens" > "$out_dir/string_contains.stage2.nodes"
cat > "$out_dir/string_contains.want.nodes" <<'NODES'
1:ASSIGN:text:STRING:hello
2:PRINT_CALL2:contains:text:STRING:ell
NODES
diff -u "$out_dir/string_contains.want.nodes" "$out_dir/string_contains.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/string_contains.stage2.nodes" > "$out_dir/string_contains.stage2.check"
grep -qx "ok" "$out_dir/string_contains.stage2.check"
compare_stage2_codegen "string contains print" "$out_dir/string_contains.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/string_contains.stage2.nodes" > "$out_dir/string_contains.stage2.c"
cat > "$out_dir/string_contains.want.c" <<'C'
#include <stdio.h>
#include <string.h>

static int contains_text(const char *text, const char *needle) {
  return strstr(text, needle) != NULL;
}

int main(void) {
  const char *text = "hello";
  puts(contains_text(text, "ell") ? "true" : "false");
  return 0;
}
C
diff -u "$out_dir/string_contains.want.c" "$out_dir/string_contains.stage2.c" >/dev/null
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/string_contains.stage2" "$out_dir/string_contains.stage2.c" >/dev/null 2>&1
string_contains_out="$("$out_dir/string_contains.stage2")"
test "$string_contains_out" = "true"
echo "string contains print: stage-2 pipeline matched"

printf 'text = "hello"\nprint startsWith text, "he"\nprint endsWith text, "lo"\n' > "$out_dir/string_prefix_suffix.tya"
"$out_dir/lexer.stage2" "$out_dir/string_prefix_suffix.tya" > "$out_dir/string_prefix_suffix.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/string_prefix_suffix.stage2.tokens" > "$out_dir/string_prefix_suffix.stage2.nodes"
cat > "$out_dir/string_prefix_suffix.want.nodes" <<'NODES'
1:ASSIGN:text:STRING:hello
2:PRINT_CALL2:startsWith:text:STRING:he
3:PRINT_CALL2:endsWith:text:STRING:lo
NODES
diff -u "$out_dir/string_prefix_suffix.want.nodes" "$out_dir/string_prefix_suffix.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/string_prefix_suffix.stage2.nodes" > "$out_dir/string_prefix_suffix.stage2.check"
grep -qx "ok" "$out_dir/string_prefix_suffix.stage2.check"
compare_stage2_codegen "string prefix suffix print" "$out_dir/string_prefix_suffix.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/string_prefix_suffix.stage2.nodes" > "$out_dir/string_prefix_suffix.stage2.c"
cat > "$out_dir/string_prefix_suffix.want.c" <<'C'
#include <stdio.h>
#include <string.h>

static int starts_with_text(const char *text, const char *prefix) {
  size_t n = strlen(prefix);
  return strncmp(text, prefix, n) == 0;
}

static int ends_with_text(const char *text, const char *suffix) {
  size_t text_len = strlen(text);
  size_t suffix_len = strlen(suffix);
  if (suffix_len > text_len) return 0;
  return strcmp(text + text_len - suffix_len, suffix) == 0;
}

int main(void) {
  const char *text = "hello";
  puts(starts_with_text(text, "he") ? "true" : "false");
  puts(ends_with_text(text, "lo") ? "true" : "false");
  return 0;
}
C
diff -u "$out_dir/string_prefix_suffix.want.c" "$out_dir/string_prefix_suffix.stage2.c" >/dev/null
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/string_prefix_suffix.stage2" "$out_dir/string_prefix_suffix.stage2.c" >/dev/null 2>&1
string_prefix_suffix_out="$("$out_dir/string_prefix_suffix.stage2")"
test "$string_prefix_suffix_out" = "true
true"
echo "string prefix suffix print: stage-2 pipeline matched"

printf 'text = "hello"\nprint replace text, "ell", "EL"\n' > "$out_dir/string_replace.tya"
"$out_dir/lexer.stage2" "$out_dir/string_replace.tya" > "$out_dir/string_replace.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/string_replace.stage2.tokens" > "$out_dir/string_replace.stage2.nodes"
cat > "$out_dir/string_replace.want.nodes" <<'NODES'
1:ASSIGN:text:STRING:hello
2:PRINT_CALL3:replace:text:STRING:ell:STRING:EL
NODES
diff -u "$out_dir/string_replace.want.nodes" "$out_dir/string_replace.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/string_replace.stage2.nodes" > "$out_dir/string_replace.stage2.check"
grep -qx "ok" "$out_dir/string_replace.stage2.check"
compare_stage2_codegen "string replace print" "$out_dir/string_replace.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/string_replace.stage2.nodes" > "$out_dir/string_replace.stage2.c"
cat > "$out_dir/string_replace.want.c" <<'C'
#include <stdio.h>
#include <string.h>
#include <stdlib.h>

static char *dup_text(const char *text) {
  size_t len = strlen(text);
  char *out = malloc(len + 1);
  memcpy(out, text, len + 1);
  return out;
}

static char *replace_text(const char *text, const char *old_text, const char *new_text) {
  const char *hit = strstr(text, old_text);
  if (!hit) return dup_text(text);
  size_t prefix = (size_t)(hit - text);
  size_t old_len = strlen(old_text);
  size_t new_len = strlen(new_text);
  size_t suffix_len = strlen(hit + old_len);
  char *out = malloc(prefix + new_len + suffix_len + 1);
  memcpy(out, text, prefix);
  memcpy(out + prefix, new_text, new_len);
  memcpy(out + prefix + new_len, hit + old_len, suffix_len + 1);
  return out;
}

int main(void) {
  const char *text = "hello";
  puts(replace_text(text, "ell", "EL"));
  return 0;
}
C
diff -u "$out_dir/string_replace.want.c" "$out_dir/string_replace.stage2.c" >/dev/null
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/string_replace.stage2" "$out_dir/string_replace.stage2.c" >/dev/null 2>&1
string_replace_out="$("$out_dir/string_replace.stage2")"
test "$string_replace_out" = "hELo"
echo "string replace print: stage-2 pipeline matched"

printf 'text = "hello,tya"\nparts = split text, ","\nprint join parts, "-"\n' > "$out_dir/string_split_join.tya"
"$out_dir/lexer.stage2" "$out_dir/string_split_join.tya" > "$out_dir/string_split_join.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/string_split_join.stage2.tokens" > "$out_dir/string_split_join.stage2.nodes"
cat > "$out_dir/string_split_join.want.nodes" <<'NODES'
1:ASSIGN:text:STRING:hello,tya
2:ASSIGN:parts:CALL2:split:text:STRING:,
3:PRINT_CALL2:join:parts:STRING:-
NODES
diff -u "$out_dir/string_split_join.want.nodes" "$out_dir/string_split_join.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/string_split_join.stage2.nodes" > "$out_dir/string_split_join.stage2.check"
grep -qx "ok" "$out_dir/string_split_join.stage2.check"
compare_stage2_codegen "string split join print" "$out_dir/string_split_join.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/string_split_join.stage2.nodes" > "$out_dir/string_split_join.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/string_split_join.stage2" "$out_dir/string_split_join.stage2.c" >/dev/null 2>&1
string_split_join_out="$("$out_dir/string_split_join.stage2")"
test "$string_split_join_out" = "hello-tya"
echo "string split join print: stage-2 pipeline matched"

printf 'print byteLen "ちゃ"\nprint charLen "ちゃ"\n' > "$out_dir/string_lengths.tya"
"$out_dir/lexer.stage2" "$out_dir/string_lengths.tya" > "$out_dir/string_lengths.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/string_lengths.stage2.tokens" > "$out_dir/string_lengths.stage2.nodes"
cat > "$out_dir/string_lengths.want.nodes" <<'NODES'
1:PRINT_CALL1:byteLen:STRING:ちゃ
2:PRINT_CALL1:charLen:STRING:ちゃ
NODES
diff -u "$out_dir/string_lengths.want.nodes" "$out_dir/string_lengths.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/string_lengths.stage2.nodes" > "$out_dir/string_lengths.stage2.check"
grep -qx "ok" "$out_dir/string_lengths.stage2.check"
compare_stage2_codegen "string byte char length print" "$out_dir/string_lengths.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/string_lengths.stage2.nodes" > "$out_dir/string_lengths.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/string_lengths.stage2" "$out_dir/string_lengths.stage2.c" >/dev/null 2>&1
string_lengths_out="$("$out_dir/string_lengths.stage2")"
test "$string_lengths_out" = "6
2"
echo "string byte char length print: stage-2 pipeline matched"

printf 'left = 2\nright = 3\nsum = left + right\nprint sum\n' > "$out_dir/int_add.tya"
"$out_dir/lexer.stage2" "$out_dir/int_add.tya" > "$out_dir/int_add.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/int_add.stage2.tokens" > "$out_dir/int_add.stage2.nodes"
cat > "$out_dir/int_add.want.nodes" <<'NODES'
1:ASSIGN:left:INT:2
2:ASSIGN:right:INT:3
3:ASSIGN:sum:INT_ADD:left:right
4:PRINT:IDENT:sum
NODES
diff -u "$out_dir/int_add.want.nodes" "$out_dir/int_add.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/int_add.stage2.nodes" > "$out_dir/int_add.stage2.check"
grep -qx "ok" "$out_dir/int_add.stage2.check"
compare_stage2_codegen "int addition" "$out_dir/int_add.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/int_add.stage2.nodes" > "$out_dir/int_add.stage2.c"
cat > "$out_dir/int_add.want.c" <<'C'
#include <stdio.h>

int main(void) {
  long left = 2;
  long right = 3;
  long sum = left + right;
  printf("%ld\n", (long)sum);
  return 0;
}
C
diff -u "$out_dir/int_add.want.c" "$out_dir/int_add.stage2.c" >/dev/null
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/int_add.stage2" "$out_dir/int_add.stage2.c" >/dev/null 2>&1
int_add_out="$("$out_dir/int_add.stage2")"
test "$int_add_out" = "5"
echo "int addition: stage-2 pipeline matched"

printf 'grouped = (1 + 1)\nprint grouped\n' > "$out_dir/grouped_int_add.tya"
"$out_dir/lexer.stage2" "$out_dir/grouped_int_add.tya" > "$out_dir/grouped_int_add.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/grouped_int_add.stage2.tokens" > "$out_dir/grouped_int_add.stage2.nodes"
cat > "$out_dir/grouped_int_add.want.nodes" <<'NODES'
1:ASSIGN:grouped:INT_ADD:1:1
2:PRINT:IDENT:grouped
NODES
diff -u "$out_dir/grouped_int_add.want.nodes" "$out_dir/grouped_int_add.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/grouped_int_add.stage2.nodes" > "$out_dir/grouped_int_add.stage2.check"
grep -qx "ok" "$out_dir/grouped_int_add.stage2.check"
compare_stage2_codegen "grouped int addition" "$out_dir/grouped_int_add.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/grouped_int_add.stage2.nodes" > "$out_dir/grouped_int_add.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/grouped_int_add.stage2" "$out_dir/grouped_int_add.stage2.c" >/dev/null 2>&1
grouped_int_add_out="$("$out_dir/grouped_int_add.stage2")"
test "$grouped_int_add_out" = "2"
echo "grouped int addition: stage-2 pipeline matched"

printf 'sum = 0\nsum = sum + 1\nprint sum\n' > "$out_dir/int_add_reassign.tya"
"$out_dir/lexer.stage2" "$out_dir/int_add_reassign.tya" > "$out_dir/int_add_reassign.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/int_add_reassign.stage2.tokens" > "$out_dir/int_add_reassign.stage2.nodes"
cat > "$out_dir/int_add_reassign.want.nodes" <<'NODES'
1:ASSIGN:sum:INT:0
2:ASSIGN:sum:INT_ADD:sum:1
3:PRINT:IDENT:sum
NODES
diff -u "$out_dir/int_add_reassign.want.nodes" "$out_dir/int_add_reassign.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/int_add_reassign.stage2.nodes" > "$out_dir/int_add_reassign.stage2.check"
grep -qx "ok" "$out_dir/int_add_reassign.stage2.check"
compare_stage2_codegen "int addition reassignment" "$out_dir/int_add_reassign.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/int_add_reassign.stage2.nodes" > "$out_dir/int_add_reassign.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/int_add_reassign.stage2" "$out_dir/int_add_reassign.stage2.c" >/dev/null 2>&1
int_add_reassign_out="$("$out_dir/int_add_reassign.stage2")"
test "$int_add_reassign_out" = "1"
echo "int addition reassignment: stage-2 pipeline matched"

printf 'enabled = true\nprint enabled\n' > "$out_dir/bool.tya"
"$out_dir/lexer.stage2" "$out_dir/bool.tya" > "$out_dir/bool.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/bool.stage2.tokens" > "$out_dir/bool.stage2.nodes"
cat > "$out_dir/bool.want.nodes" <<'NODES'
1:ASSIGN:enabled:BOOL:true
2:PRINT:IDENT:enabled
NODES
diff -u "$out_dir/bool.want.nodes" "$out_dir/bool.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/bool.stage2.nodes" > "$out_dir/bool.stage2.check"
grep -qx "ok" "$out_dir/bool.stage2.check"
compare_stage2_codegen "bool assignment" "$out_dir/bool.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/bool.stage2.nodes" > "$out_dir/bool.stage2.c"
cat > "$out_dir/bool.want.c" <<'C'
#include <stdio.h>

int main(void) {
  int enabled = 1;
  puts(enabled ? "true" : "false");
  return 0;
}
C
diff -u "$out_dir/bool.want.c" "$out_dir/bool.stage2.c" >/dev/null
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/bool.stage2" "$out_dir/bool.stage2.c" >/dev/null 2>&1
bool_out="$("$out_dir/bool.stage2")"
test "$bool_out" = "true"
echo "bool assignment: stage-2 pipeline matched"

printf 'adult = true\nyoung = true\nboth = adult and young\neither = adult or young\nprint both\nprint either\n' > "$out_dir/bool_logic.tya"
"$out_dir/lexer.stage2" "$out_dir/bool_logic.tya" > "$out_dir/bool_logic.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/bool_logic.stage2.tokens" > "$out_dir/bool_logic.stage2.nodes"
cat > "$out_dir/bool_logic.want.nodes" <<'NODES'
1:ASSIGN:adult:BOOL:true
2:ASSIGN:young:BOOL:true
3:ASSIGN:both:BOOL_AND:adult:young
4:ASSIGN:either:BOOL_OR:adult:young
5:PRINT:IDENT:both
6:PRINT:IDENT:either
NODES
diff -u "$out_dir/bool_logic.want.nodes" "$out_dir/bool_logic.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/bool_logic.stage2.nodes" > "$out_dir/bool_logic.stage2.check"
grep -qx "ok" "$out_dir/bool_logic.stage2.check"
compare_stage2_codegen "bool logic" "$out_dir/bool_logic.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/bool_logic.stage2.nodes" > "$out_dir/bool_logic.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/bool_logic.stage2" "$out_dir/bool_logic.stage2.c" >/dev/null 2>&1
bool_logic_out="$("$out_dir/bool_logic.stage2")"
test "$bool_logic_out" = "true
true"
echo "bool logic: stage-2 pipeline matched"

printf 'while false\n  break\nprint "done"\n' > "$out_dir/while_false.tya"
"$out_dir/lexer.stage2" "$out_dir/while_false.tya" > "$out_dir/while_false.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/while_false.stage2.tokens" > "$out_dir/while_false.stage2.nodes"
cat > "$out_dir/while_false.want.nodes" <<'NODES'
1:WHILE:BOOL:false
2:INDENT:2
2:BREAK
3:INDENT:0
3:PRINT:STRING:done
NODES
diff -u "$out_dir/while_false.want.nodes" "$out_dir/while_false.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/while_false.stage2.nodes" > "$out_dir/while_false.stage2.check"
grep -qx "ok" "$out_dir/while_false.stage2.check"
compare_stage2_codegen "while false break" "$out_dir/while_false.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/while_false.stage2.nodes" > "$out_dir/while_false.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/while_false.stage2" "$out_dir/while_false.stage2.c" >/dev/null 2>&1
while_false_out="$("$out_dir/while_false.stage2")"
test "$while_false_out" = "done"
echo "while false break: stage-2 pipeline matched"

printf 'i = 0\nwhile i < 2\n  i = i + 1\n  break\nprint i\n' > "$out_dir/while_less_than.tya"
"$out_dir/lexer.stage2" "$out_dir/while_less_than.tya" > "$out_dir/while_less_than.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/while_less_than.stage2.tokens" > "$out_dir/while_less_than.stage2.nodes"
cat > "$out_dir/while_less_than.want.nodes" <<'NODES'
1:ASSIGN:i:INT:0
2:WHILE_COMPARE_LT:IDENT:i:INT:2
3:INDENT:2
3:ASSIGN:i:INT_ADD:i:1
4:BREAK
5:INDENT:0
5:PRINT:IDENT:i
NODES
diff -u "$out_dir/while_less_than.want.nodes" "$out_dir/while_less_than.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/while_less_than.stage2.nodes" > "$out_dir/while_less_than.stage2.check"
grep -qx "ok" "$out_dir/while_less_than.stage2.check"
compare_stage2_codegen "while less-than break" "$out_dir/while_less_than.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/while_less_than.stage2.nodes" > "$out_dir/while_less_than.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/while_less_than.stage2" "$out_dir/while_less_than.stage2.c" >/dev/null 2>&1
while_less_than_out="$("$out_dir/while_less_than.stage2")"
test "$while_less_than_out" = "1"
echo "while less-than break: stage-2 pipeline matched"

printf 'age = 2\nwhile age >= 2\n  print "loop"\n  break\nprint age\n' > "$out_dir/while_bounds.tya"
"$out_dir/lexer.stage2" "$out_dir/while_bounds.tya" > "$out_dir/while_bounds.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/while_bounds.stage2.tokens" > "$out_dir/while_bounds.stage2.nodes"
cat > "$out_dir/while_bounds.want.nodes" <<'NODES'
1:ASSIGN:age:INT:2
2:WHILE_COMPARE_GE:IDENT:age:INT:2
3:INDENT:2
3:PRINT:STRING:loop
4:BREAK
5:INDENT:0
5:PRINT:IDENT:age
NODES
diff -u "$out_dir/while_bounds.want.nodes" "$out_dir/while_bounds.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/while_bounds.stage2.nodes" > "$out_dir/while_bounds.stage2.check"
grep -qx "ok" "$out_dir/while_bounds.stage2.check"
compare_stage2_codegen "while bounded break" "$out_dir/while_bounds.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/while_bounds.stage2.nodes" > "$out_dir/while_bounds.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/while_bounds.stage2" "$out_dir/while_bounds.stage2.c" >/dev/null 2>&1
while_bounds_out="$("$out_dir/while_bounds.stage2")"
test "$while_bounds_out" = "loop
2"
echo "while bounded break: stage-2 pipeline matched"

printf 'names = ["Tya"]\nfor item in names\n  print item\n' > "$out_dir/array_for.tya"
"$out_dir/lexer.stage2" "$out_dir/array_for.tya" > "$out_dir/array_for.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/array_for.stage2.tokens" > "$out_dir/array_for.stage2.nodes"
cat > "$out_dir/array_for.want.nodes" <<'NODES'
1:ASSIGN:names:ARRAY_ONE:STRING:Tya
2:FOR:item:names
3:INDENT:2
3:PRINT:IDENT:item
NODES
diff -u "$out_dir/array_for.want.nodes" "$out_dir/array_for.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/array_for.stage2.nodes" > "$out_dir/array_for.stage2.check"
grep -qx "ok" "$out_dir/array_for.stage2.check"
compare_stage2_codegen "array for" "$out_dir/array_for.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/array_for.stage2.nodes" > "$out_dir/array_for.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/array_for.stage2" "$out_dir/array_for.stage2.c" >/dev/null 2>&1
array_for_out="$("$out_dir/array_for.stage2")"
test "$array_for_out" = "Tya"
echo "array for: stage-2 pipeline matched"

"$out_dir/lexer.stage2" examples/selfhost_ops.tya > "$out_dir/selfhost_ops.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/selfhost_ops.stage2.tokens" > "$out_dir/selfhost_ops.stage2.nodes"
"$out_dir/checker.stage2" "$out_dir/selfhost_ops.stage2.nodes" > "$out_dir/selfhost_ops.stage2.check"
grep -qx "ok" "$out_dir/selfhost_ops.stage2.check"
compare_stage2_codegen "examples/selfhost_ops.tya" "$out_dir/selfhost_ops.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/selfhost_ops.stage2.nodes" > "$out_dir/selfhost_ops.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/selfhost_ops.stage2" "$out_dir/selfhost_ops.stage2.c" >/dev/null 2>&1
selfhost_ops_out="$("$out_dir/selfhost_ops.stage2")"
test "$selfhost_ops_out" = "adult
young
komagata
true
true
true
2
true
true
true
loop
Tya"
echo "examples/selfhost_ops.tya: stage-2 pipeline matched"

"$out_dir/lexer.stage2" examples/while.tya > "$out_dir/while_example.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/while_example.stage2.tokens" > "$out_dir/while_example.stage2.nodes"
"$out_dir/checker.stage2" "$out_dir/while_example.stage2.nodes" > "$out_dir/while_example.stage2.check"
grep -qx "ok" "$out_dir/while_example.stage2.check"
compare_stage2_codegen "examples/while.tya" "$out_dir/while_example.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/while_example.stage2.nodes" > "$out_dir/while_example.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/while_example.stage2" "$out_dir/while_example.stage2.c" >/dev/null 2>&1
while_example_out="$("$out_dir/while_example.stage2")"
test "$while_example_out" = "10
11"
echo "examples/while.tya: stage-2 pipeline matched"

printf 'left = 2\nright = 2\nsame = left == right\nprint same\n' > "$out_dir/compare_eq.tya"
"$out_dir/lexer.stage2" "$out_dir/compare_eq.tya" > "$out_dir/compare_eq.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/compare_eq.stage2.tokens" > "$out_dir/compare_eq.stage2.nodes"
cat > "$out_dir/compare_eq.want.nodes" <<'NODES'
1:ASSIGN:left:INT:2
2:ASSIGN:right:INT:2
3:ASSIGN:same:COMPARE_EQ:left:right
4:PRINT:IDENT:same
NODES
diff -u "$out_dir/compare_eq.want.nodes" "$out_dir/compare_eq.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/compare_eq.stage2.nodes" > "$out_dir/compare_eq.stage2.check"
grep -qx "ok" "$out_dir/compare_eq.stage2.check"
compare_stage2_codegen "equality comparison" "$out_dir/compare_eq.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/compare_eq.stage2.nodes" > "$out_dir/compare_eq.stage2.c"
cat > "$out_dir/compare_eq.want.c" <<'C'
#include <stdio.h>

int main(void) {
  long left = 2;
  long right = 2;
  int same = left == right;
  puts(same ? "true" : "false");
  return 0;
}
C
diff -u "$out_dir/compare_eq.want.c" "$out_dir/compare_eq.stage2.c" >/dev/null
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/compare_eq.stage2" "$out_dir/compare_eq.stage2.c" >/dev/null 2>&1
compare_eq_out="$("$out_dir/compare_eq.stage2")"
test "$compare_eq_out" = "true"
echo "equality comparison: stage-2 pipeline matched"

printf 'left = 2\nright = 3\ndifferent = left != right\nprint different\n' > "$out_dir/compare_ne.tya"
"$out_dir/lexer.stage2" "$out_dir/compare_ne.tya" > "$out_dir/compare_ne.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/compare_ne.stage2.tokens" > "$out_dir/compare_ne.stage2.nodes"
cat > "$out_dir/compare_ne.want.nodes" <<'NODES'
1:ASSIGN:left:INT:2
2:ASSIGN:right:INT:3
3:ASSIGN:different:COMPARE_NE:left:right
4:PRINT:IDENT:different
NODES
diff -u "$out_dir/compare_ne.want.nodes" "$out_dir/compare_ne.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/compare_ne.stage2.nodes" > "$out_dir/compare_ne.stage2.check"
grep -qx "ok" "$out_dir/compare_ne.stage2.check"
compare_stage2_codegen "inequality comparison" "$out_dir/compare_ne.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/compare_ne.stage2.nodes" > "$out_dir/compare_ne.stage2.c"
cat > "$out_dir/compare_ne.want.c" <<'C'
#include <stdio.h>

int main(void) {
  long left = 2;
  long right = 3;
  int different = left != right;
  puts(different ? "true" : "false");
  return 0;
}
C
diff -u "$out_dir/compare_ne.want.c" "$out_dir/compare_ne.stage2.c" >/dev/null
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/compare_ne.stage2" "$out_dir/compare_ne.stage2.c" >/dev/null 2>&1
compare_ne_out="$("$out_dir/compare_ne.stage2")"
test "$compare_ne_out" = "true"
echo "inequality comparison: stage-2 pipeline matched"

printf 'left = 2\nright = 3\nless = left < right\nprint less\n' > "$out_dir/compare_lt.tya"
"$out_dir/lexer.stage2" "$out_dir/compare_lt.tya" > "$out_dir/compare_lt.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/compare_lt.stage2.tokens" > "$out_dir/compare_lt.stage2.nodes"
cat > "$out_dir/compare_lt.want.nodes" <<'NODES'
1:ASSIGN:left:INT:2
2:ASSIGN:right:INT:3
3:ASSIGN:less:COMPARE_LT:left:right
4:PRINT:IDENT:less
NODES
diff -u "$out_dir/compare_lt.want.nodes" "$out_dir/compare_lt.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/compare_lt.stage2.nodes" > "$out_dir/compare_lt.stage2.check"
grep -qx "ok" "$out_dir/compare_lt.stage2.check"
compare_stage2_codegen "less-than comparison" "$out_dir/compare_lt.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/compare_lt.stage2.nodes" > "$out_dir/compare_lt.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/compare_lt.stage2" "$out_dir/compare_lt.stage2.c" >/dev/null 2>&1
compare_lt_out="$("$out_dir/compare_lt.stage2")"
test "$compare_lt_out" = "true"
echo "less-than comparison: stage-2 pipeline matched"

printf 'age = 2\nadult = age >= 2\nyoung = age <= 2\nprint adult\nprint young\n' > "$out_dir/compare_bounds.tya"
"$out_dir/lexer.stage2" "$out_dir/compare_bounds.tya" > "$out_dir/compare_bounds.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/compare_bounds.stage2.tokens" > "$out_dir/compare_bounds.stage2.nodes"
cat > "$out_dir/compare_bounds.want.nodes" <<'NODES'
1:ASSIGN:age:INT:2
2:ASSIGN:adult:COMPARE_GE:age:2
3:ASSIGN:young:COMPARE_LE:age:2
4:PRINT:IDENT:adult
5:PRINT:IDENT:young
NODES
diff -u "$out_dir/compare_bounds.want.nodes" "$out_dir/compare_bounds.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/compare_bounds.stage2.nodes" > "$out_dir/compare_bounds.stage2.check"
grep -qx "ok" "$out_dir/compare_bounds.stage2.check"
compare_stage2_codegen "bounded comparison" "$out_dir/compare_bounds.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/compare_bounds.stage2.nodes" > "$out_dir/compare_bounds.stage2.c"
cat > "$out_dir/compare_bounds.want.c" <<'C'
#include <stdio.h>

int main(void) {
  long age = 2;
  int adult = age >= 2;
  int young = age <= 2;
  puts(adult ? "true" : "false");
  puts(young ? "true" : "false");
  return 0;
}
C
diff -u "$out_dir/compare_bounds.want.c" "$out_dir/compare_bounds.stage2.c" >/dev/null
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/compare_bounds.stage2" "$out_dir/compare_bounds.stage2.c" >/dev/null 2>&1
compare_bounds_out="$("$out_dir/compare_bounds.stage2")"
test "$compare_bounds_out" = "true
true"
echo "bounded comparison: stage-2 pipeline matched"

printf 'age = 2\ngroupedCompare = (age >= 2)\nprint groupedCompare\n' > "$out_dir/grouped_compare.tya"
"$out_dir/lexer.stage2" "$out_dir/grouped_compare.tya" > "$out_dir/grouped_compare.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/grouped_compare.stage2.tokens" > "$out_dir/grouped_compare.stage2.nodes"
cat > "$out_dir/grouped_compare.want.nodes" <<'NODES'
1:ASSIGN:age:INT:2
2:ASSIGN:groupedCompare:COMPARE_GE:age:2
3:PRINT:IDENT:groupedCompare
NODES
diff -u "$out_dir/grouped_compare.want.nodes" "$out_dir/grouped_compare.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/grouped_compare.stage2.nodes" > "$out_dir/grouped_compare.stage2.check"
grep -qx "ok" "$out_dir/grouped_compare.stage2.check"
compare_stage2_codegen "grouped comparison" "$out_dir/grouped_compare.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/grouped_compare.stage2.nodes" > "$out_dir/grouped_compare.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/grouped_compare.stage2" "$out_dir/grouped_compare.stage2.c" >/dev/null 2>&1
grouped_compare_out="$("$out_dir/grouped_compare.stage2")"
test "$grouped_compare_out" = "true"
echo "grouped comparison: stage-2 pipeline matched"

stage4_dir="$out_dir/stage4-hello"
mkdir -p "$stage4_dir"
for src in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  base="$(basename "$src" .tya)"
  "$out_dir/lexer.stage2" "$src" > "$stage4_dir/$base.stage3.tokens"
  "$out_dir/parser.stage2" "$stage4_dir/$base.stage3.tokens" > "$stage4_dir/$base.stage3.nodes"
  "$out_dir/checker.stage2" "$stage4_dir/$base.stage3.nodes" > "$stage4_dir/$base.stage3.check"
  grep -qx "ok" "$stage4_dir/$base.stage3.check"
  "$out_dir/codegen_c.stage2" "$stage4_dir/$base.stage3.nodes" > "$stage4_dir/$base.stage3.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/$base.stage3" "$stage4_dir/$base.stage3.c" >/dev/null 2>&1
done

for src in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  base="$(basename "$src" .tya)"
  "$stage4_dir/lexer.stage3" "$src" > "$stage4_dir/$base.stage4.tokens"
  "$stage4_dir/parser.stage3" "$stage4_dir/$base.stage4.tokens" > "$stage4_dir/$base.stage4.nodes"
  "$stage4_dir/checker.stage3" "$stage4_dir/$base.stage4.nodes" > "$stage4_dir/$base.stage4.check"
  grep -qx "ok" "$stage4_dir/$base.stage4.check"
  "$stage4_dir/codegen_c.stage3" "$stage4_dir/$base.stage4.nodes" > "$stage4_dir/$base.stage4.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/$base.stage4" "$stage4_dir/$base.stage4.c" >/dev/null 2>&1
done

cat > "$stage4_dir/lexer.stage4.want.nodes" <<'NODES'
149:ASSIGN:source:CALL1_CALL0_INDEX:readFile:args:0
150:ASSIGN:tokens:CALL1:lex:source
152:FOR:token:tokens
153:INDENT:2
153:PRINT:IDENT:token
NODES
diff -u "$stage4_dir/lexer.stage4.want.nodes" "$stage4_dir/lexer.stage4.nodes" >/dev/null
echo "selfhost/lexer.tya: stage-3 parser emitted real nodes"
grep -q '^int main(int argc, char \*\*argv)' "$stage4_dir/lexer.stage4.c"
if grep -q 'strstr(mode, "lexer")' "$stage4_dir/lexer.stage4.c"; then
  exit 1
fi
echo "selfhost/lexer.tya: stage-3 codegen emitted executable lexer C"
cat > "$stage4_dir/parser.stage4.want.nodes" <<'NODES'
512:ASSIGN:source:CALL1_CALL0_INDEX:readFile:args:0
515:FOR:line:lines
516:INDENT:2
521:FOR:node:nodes
522:INDENT:2
522:PRINT:IDENT:node
NODES
diff -u "$stage4_dir/parser.stage4.want.nodes" "$stage4_dir/parser.stage4.nodes" >/dev/null
echo "selfhost/parser.tya: stage-3 parser emitted real nodes"
grep -q '^int main(int argc, char \*\*argv)' "$stage4_dir/parser.stage4.c"
if grep -q 'strstr(mode, "parser")' "$stage4_dir/parser.stage4.c"; then
  exit 1
fi
echo "selfhost/parser.tya: stage-3 codegen emitted executable parser C"
cat > "$stage4_dir/checker.stage4.want.nodes" <<'NODES'
58:FOR:existing:names
59:INDENT:2
126:FOR:node:nodes
127:INDENT:2
574:ASSIGN:source:CALL1_CALL0_INDEX:readFile:args:0
577:FOR:line:lines
578:INDENT:2
584:PRINT:STRING:ok
586:FOR:err:errors
587:INDENT:2
587:PRINT:IDENT:err
NODES
diff -u "$stage4_dir/checker.stage4.want.nodes" "$stage4_dir/checker.stage4.nodes" >/dev/null
echo "selfhost/checker.tya: stage-3 parser emitted real nodes"
grep -q '^int main(void)' "$stage4_dir/checker.stage4.c"
if grep -q 'strstr(mode, "checker")' "$stage4_dir/checker.stage4.c"; then
  exit 1
fi
echo "selfhost/checker.tya: stage-3 codegen emitted executable checker C"
cat > "$stage4_dir/codegen_c.stage4.want.nodes" <<'NODES'
58:FOR:existing:names
59:INDENT:2
91:FOR:node:nodes
92:INDENT:2
2390:FOR:node:nodes
2391:INDENT:2
3089:ASSIGN:source:CALL1_CALL0_INDEX:readFile:args:0
3092:FOR:line:lines
3093:INDENT:2
3096:PRINT_CALL1:emitC:nodes
NODES
diff -u "$stage4_dir/codegen_c.stage4.want.nodes" "$stage4_dir/codegen_c.stage4.nodes" >/dev/null
echo "selfhost/codegen_c.tya: stage-3 parser emitted real nodes"
grep -q '^int main(int argc, char \*\*argv)' "$stage4_dir/codegen_c.stage4.c"
if grep -q 'strstr(mode, "codegen")' "$stage4_dir/codegen_c.stage4.c"; then
  exit 1
fi
echo "selfhost/codegen_c.tya: stage-3 codegen emitted executable codegen C"

"$stage4_dir/lexer.stage4" examples/hello.tya > "$stage4_dir/hello.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/hello.tokens" > "$stage4_dir/hello.nodes"
"$stage4_dir/checker.stage4" "$stage4_dir/hello.nodes" > "$stage4_dir/hello.check"
grep -qx "ok" "$stage4_dir/hello.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/hello.nodes" > "$stage4_dir/hello.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/hello" "$stage4_dir/hello.c" >/dev/null 2>&1
stage4_hello_out="$("$stage4_dir/hello")"
test "$stage4_hello_out" = "Hello, Tya"
echo "stage4 hello: self-host pipeline matched"

printf 'print "Tya"\n' > "$stage4_dir/print_tya.tya"
"$stage4_dir/lexer.stage4" "$stage4_dir/print_tya.tya" > "$stage4_dir/print_tya.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/print_tya.tokens" > "$stage4_dir/print_tya.nodes"
"$stage4_dir/checker.stage4" "$stage4_dir/print_tya.nodes" > "$stage4_dir/print_tya.check"
grep -qx "ok" "$stage4_dir/print_tya.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/print_tya.nodes" > "$stage4_dir/print_tya.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/print_tya" "$stage4_dir/print_tya.c" >/dev/null 2>&1
stage4_print_tya_out="$("$stage4_dir/print_tya")"
test "$stage4_print_tya_out" = "Tya"
echo "stage4 print string: self-host pipeline matched"

printf 'print 1\n' > "$stage4_dir/print_int.tya"
"$stage4_dir/lexer.stage4" "$stage4_dir/print_int.tya" > "$stage4_dir/print_int.tokens"
cat > "$stage4_dir/print_int.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:print:1
1:INT:1:7
TOKENS
diff -u "$stage4_dir/print_int.want.tokens" "$stage4_dir/print_int.tokens" >/dev/null
"$stage4_dir/parser.stage4" "$stage4_dir/print_int.tokens" > "$stage4_dir/print_int.nodes"
cat > "$stage4_dir/print_int.want.nodes" <<'NODES'
1:PRINT:INT:1
NODES
diff -u "$stage4_dir/print_int.want.nodes" "$stage4_dir/print_int.nodes" >/dev/null
"$stage4_dir/checker.stage4" "$stage4_dir/print_int.nodes" > "$stage4_dir/print_int.check"
grep -qx "ok" "$stage4_dir/print_int.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/print_int.nodes" > "$stage4_dir/print_int.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/print_int" "$stage4_dir/print_int.c" >/dev/null 2>&1
stage4_print_int_out="$("$stage4_dir/print_int")"
test "$stage4_print_int_out" = "1"
echo "stage4 print int: self-host pipeline matched"

printf 'print "say \\"tya\\""\n' > "$stage4_dir/escaped_print.tya"
"$stage4_dir/lexer.stage4" "$stage4_dir/escaped_print.tya" > "$stage4_dir/escaped_print.tokens"
cat > "$stage4_dir/escaped_print.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:print:1
1:STRING:say \"tya\":7
TOKENS
diff -u "$stage4_dir/escaped_print.want.tokens" "$stage4_dir/escaped_print.tokens" >/dev/null
"$stage4_dir/parser.stage4" "$stage4_dir/escaped_print.tokens" > "$stage4_dir/escaped_print.nodes"
cat > "$stage4_dir/escaped_print.want.nodes" <<'NODES'
1:PRINT:STRING:say \"tya\"
NODES
diff -u "$stage4_dir/escaped_print.want.nodes" "$stage4_dir/escaped_print.nodes" >/dev/null
"$stage4_dir/checker.stage4" "$stage4_dir/escaped_print.nodes" > "$stage4_dir/escaped_print.check"
grep -qx "ok" "$stage4_dir/escaped_print.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/escaped_print.nodes" > "$stage4_dir/escaped_print.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/escaped_print" "$stage4_dir/escaped_print.c" >/dev/null 2>&1
stage4_escaped_print_out="$("$stage4_dir/escaped_print")"
test "$stage4_escaped_print_out" = 'say "tya"'
echo "stage4 escaped string: self-host pipeline matched"

printf 'print "quote: \\"tya\\""\n' > "$stage4_dir/colon_print.tya"
"$stage4_dir/lexer.stage4" "$stage4_dir/colon_print.tya" > "$stage4_dir/colon_print.tokens"
cat > "$stage4_dir/colon_print.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:print:1
1:STRING:quote: \"tya\":7
TOKENS
diff -u "$stage4_dir/colon_print.want.tokens" "$stage4_dir/colon_print.tokens" >/dev/null
"$stage4_dir/parser.stage4" "$stage4_dir/colon_print.tokens" > "$stage4_dir/colon_print.nodes"
cat > "$stage4_dir/colon_print.want.nodes" <<'NODES'
1:PRINT:STRING:quote: \"tya\"
NODES
diff -u "$stage4_dir/colon_print.want.nodes" "$stage4_dir/colon_print.nodes" >/dev/null
"$stage4_dir/checker.stage4" "$stage4_dir/colon_print.nodes" > "$stage4_dir/colon_print.check"
grep -qx "ok" "$stage4_dir/colon_print.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/colon_print.nodes" > "$stage4_dir/colon_print.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/colon_print" "$stage4_dir/colon_print.c" >/dev/null 2>&1
stage4_colon_print_out="$("$stage4_dir/colon_print")"
test "$stage4_colon_print_out" = 'quote: "tya"'
echo "stage4 colon string: self-host pipeline matched"

printf 'print "A"\nprint "B"\n' > "$stage4_dir/two_prints.tya"
"$stage4_dir/lexer.stage4" "$stage4_dir/two_prints.tya" > "$stage4_dir/two_prints.tokens"
cat > "$stage4_dir/two_prints.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:print:1
1:STRING:A:7
2:INDENT:0:1
2:IDENT:print:1
2:STRING:B:7
TOKENS
diff -u "$stage4_dir/two_prints.want.tokens" "$stage4_dir/two_prints.tokens" >/dev/null
"$stage4_dir/parser.stage4" "$stage4_dir/two_prints.tokens" > "$stage4_dir/two_prints.nodes"
cat > "$stage4_dir/two_prints.want.nodes" <<'NODES'
1:PRINT:STRING:A
2:PRINT:STRING:B
NODES
diff -u "$stage4_dir/two_prints.want.nodes" "$stage4_dir/two_prints.nodes" >/dev/null
"$stage4_dir/checker.stage4" "$stage4_dir/two_prints.nodes" > "$stage4_dir/two_prints.check"
grep -qx "ok" "$stage4_dir/two_prints.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/two_prints.nodes" > "$stage4_dir/two_prints.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/two_prints" "$stage4_dir/two_prints.c" >/dev/null 2>&1
stage4_two_prints_out="$("$stage4_dir/two_prints")"
test "$stage4_two_prints_out" = "A
B"
echo "stage4 two prints: self-host pipeline matched"

printf 'message = "Hi"\nprint message\n' > "$stage4_dir/assign_print.tya"
"$stage4_dir/lexer.stage4" "$stage4_dir/assign_print.tya" > "$stage4_dir/assign_print.tokens"
cat > "$stage4_dir/assign_print.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:message:1
1:SYMBOL:=:9
1:STRING:Hi:11
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:message:7
TOKENS
diff -u "$stage4_dir/assign_print.want.tokens" "$stage4_dir/assign_print.tokens" >/dev/null
"$stage4_dir/parser.stage4" "$stage4_dir/assign_print.tokens" > "$stage4_dir/assign_print.nodes"
cat > "$stage4_dir/assign_print.want.nodes" <<'NODES'
1:ASSIGN:message:STRING:Hi
2:PRINT:IDENT:message
NODES
diff -u "$stage4_dir/assign_print.want.nodes" "$stage4_dir/assign_print.nodes" >/dev/null
"$stage4_dir/checker.stage4" "$stage4_dir/assign_print.nodes" > "$stage4_dir/assign_print.check"
grep -qx "ok" "$stage4_dir/assign_print.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/assign_print.nodes" > "$stage4_dir/assign_print.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/assign_print" "$stage4_dir/assign_print.c" >/dev/null 2>&1
stage4_assign_print_out="$("$stage4_dir/assign_print")"
test "$stage4_assign_print_out" = "Hi"
echo "stage4 assign print: self-host pipeline matched"

printf 'count = 2\nprint count\n' > "$stage4_dir/assign_int_print.tya"
"$stage4_dir/lexer.stage4" "$stage4_dir/assign_int_print.tya" > "$stage4_dir/assign_int_print.tokens"
cat > "$stage4_dir/assign_int_print.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:count:1
1:SYMBOL:=:7
1:INT:2:9
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:count:7
TOKENS
diff -u "$stage4_dir/assign_int_print.want.tokens" "$stage4_dir/assign_int_print.tokens" >/dev/null
"$stage4_dir/parser.stage4" "$stage4_dir/assign_int_print.tokens" > "$stage4_dir/assign_int_print.nodes"
cat > "$stage4_dir/assign_int_print.want.nodes" <<'NODES'
1:ASSIGN:count:INT:2
2:PRINT:IDENT:count
NODES
diff -u "$stage4_dir/assign_int_print.want.nodes" "$stage4_dir/assign_int_print.nodes" >/dev/null
"$stage4_dir/checker.stage4" "$stage4_dir/assign_int_print.nodes" > "$stage4_dir/assign_int_print.check"
grep -qx "ok" "$stage4_dir/assign_int_print.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/assign_int_print.nodes" > "$stage4_dir/assign_int_print.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/assign_int_print" "$stage4_dir/assign_int_print.c" >/dev/null 2>&1
stage4_assign_int_print_out="$("$stage4_dir/assign_int_print")"
test "$stage4_assign_int_print_out" = "2"
echo "stage4 int assign print: self-host pipeline matched"

printf 'count = 1\ncount = 2\nprint count\n' > "$stage4_dir/reassign_int_print.tya"
"$stage4_dir/lexer.stage4" "$stage4_dir/reassign_int_print.tya" > "$stage4_dir/reassign_int_print.tokens"
cat > "$stage4_dir/reassign_int_print.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:count:1
1:SYMBOL:=:7
1:INT:1:9
2:INDENT:0:1
2:IDENT:count:1
2:SYMBOL:=:7
2:INT:2:9
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:count:7
TOKENS
diff -u "$stage4_dir/reassign_int_print.want.tokens" "$stage4_dir/reassign_int_print.tokens" >/dev/null
"$stage4_dir/parser.stage4" "$stage4_dir/reassign_int_print.tokens" > "$stage4_dir/reassign_int_print.nodes"
cat > "$stage4_dir/reassign_int_print.want.nodes" <<'NODES'
1:ASSIGN:count:INT:1
2:ASSIGN:count:INT:2
3:PRINT:IDENT:count
NODES
diff -u "$stage4_dir/reassign_int_print.want.nodes" "$stage4_dir/reassign_int_print.nodes" >/dev/null
"$stage4_dir/checker.stage4" "$stage4_dir/reassign_int_print.nodes" > "$stage4_dir/reassign_int_print.check"
grep -qx "ok" "$stage4_dir/reassign_int_print.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/reassign_int_print.nodes" > "$stage4_dir/reassign_int_print.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/reassign_int_print" "$stage4_dir/reassign_int_print.c" >/dev/null 2>&1
stage4_reassign_int_print_out="$("$stage4_dir/reassign_int_print")"
test "$stage4_reassign_int_print_out" = "2"
echo "stage4 int reassignment print: self-host pipeline matched"

printf 'sum = 1 + 1\nprint sum\n' > "$stage4_dir/add_assign_print.tya"
"$stage4_dir/lexer.stage4" "$stage4_dir/add_assign_print.tya" > "$stage4_dir/add_assign_print.tokens"
cat > "$stage4_dir/add_assign_print.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:sum:1
1:SYMBOL:=:5
1:INT:1 + 1:7
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:sum:7
TOKENS
diff -u "$stage4_dir/add_assign_print.want.tokens" "$stage4_dir/add_assign_print.tokens" >/dev/null
"$stage4_dir/parser.stage4" "$stage4_dir/add_assign_print.tokens" > "$stage4_dir/add_assign_print.nodes"
cat > "$stage4_dir/add_assign_print.want.nodes" <<'NODES'
1:ASSIGN:sum:INT:1 + 1
2:PRINT:IDENT:sum
NODES
diff -u "$stage4_dir/add_assign_print.want.nodes" "$stage4_dir/add_assign_print.nodes" >/dev/null
"$stage4_dir/checker.stage4" "$stage4_dir/add_assign_print.nodes" > "$stage4_dir/add_assign_print.check"
grep -qx "ok" "$stage4_dir/add_assign_print.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/add_assign_print.nodes" > "$stage4_dir/add_assign_print.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/add_assign_print" "$stage4_dir/add_assign_print.c" >/dev/null 2>&1
stage4_add_assign_print_out="$("$stage4_dir/add_assign_print")"
test "$stage4_add_assign_print_out" = "2"
echo "stage4 int addition print: self-host pipeline matched"

printf 'less = 1 < 2\nprint less\n' > "$stage4_dir/less_print.tya"
"$stage4_dir/lexer.stage4" "$stage4_dir/less_print.tya" > "$stage4_dir/less_print.tokens"
cat > "$stage4_dir/less_print.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:less:1
1:SYMBOL:=:6
1:BOOL:1 < 2:8
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:less:7
TOKENS
diff -u "$stage4_dir/less_print.want.tokens" "$stage4_dir/less_print.tokens" >/dev/null
"$stage4_dir/parser.stage4" "$stage4_dir/less_print.tokens" > "$stage4_dir/less_print.nodes"
cat > "$stage4_dir/less_print.want.nodes" <<'NODES'
1:ASSIGN:less:BOOL:1 < 2
2:PRINT:IDENT:less
NODES
diff -u "$stage4_dir/less_print.want.nodes" "$stage4_dir/less_print.nodes" >/dev/null
"$stage4_dir/checker.stage4" "$stage4_dir/less_print.nodes" > "$stage4_dir/less_print.check"
grep -qx "ok" "$stage4_dir/less_print.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/less_print.nodes" > "$stage4_dir/less_print.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/less_print" "$stage4_dir/less_print.c" >/dev/null 2>&1
stage4_less_print_out="$("$stage4_dir/less_print")"
test "$stage4_less_print_out" = "true"
echo "stage4 less-than print: self-host pipeline matched"

printf 'while false\n  print "Never"\n  break\nprint "Done"\n' > "$stage4_dir/while_false_print.tya"
"$stage4_dir/lexer.stage4" "$stage4_dir/while_false_print.tya" > "$stage4_dir/while_false_print.tokens"
cat > "$stage4_dir/while_false_print.want.tokens" <<'TOKENS'
1:INDENT:0:1
4:INDENT:0:1
4:IDENT:print:1
4:STRING:Done:7
TOKENS
diff -u "$stage4_dir/while_false_print.want.tokens" "$stage4_dir/while_false_print.tokens" >/dev/null
"$stage4_dir/parser.stage4" "$stage4_dir/while_false_print.tokens" > "$stage4_dir/while_false_print.nodes"
cat > "$stage4_dir/while_false_print.want.nodes" <<'NODES'
1:PRINT:STRING:Done
NODES
diff -u "$stage4_dir/while_false_print.want.nodes" "$stage4_dir/while_false_print.nodes" >/dev/null
"$stage4_dir/checker.stage4" "$stage4_dir/while_false_print.nodes" > "$stage4_dir/while_false_print.check"
grep -qx "ok" "$stage4_dir/while_false_print.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/while_false_print.nodes" > "$stage4_dir/while_false_print.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/while_false_print" "$stage4_dir/while_false_print.c" >/dev/null 2>&1
stage4_while_false_print_out="$("$stage4_dir/while_false_print")"
test "$stage4_while_false_print_out" = "Done"
echo "stage4 while false print: self-host pipeline matched"

printf 'items = ["Tya"]\nfor item in items\n  print item\n' > "$stage4_dir/array_for.tya"
"$stage4_dir/lexer.stage4" "$stage4_dir/array_for.tya" > "$stage4_dir/array_for.tokens"
cat > "$stage4_dir/array_for.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:items:1
1:SYMBOL:=:7
1:ARRAY:Tya:9
2:INDENT:0:1
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:item:7
TOKENS
diff -u "$stage4_dir/array_for.want.tokens" "$stage4_dir/array_for.tokens" >/dev/null
"$stage4_dir/parser.stage4" "$stage4_dir/array_for.tokens" > "$stage4_dir/array_for.nodes"
cat > "$stage4_dir/array_for.want.nodes" <<'NODES'
1:ASSIGN:items:ARRAY:Tya
2:PRINT:IDENT:item
NODES
diff -u "$stage4_dir/array_for.want.nodes" "$stage4_dir/array_for.nodes" >/dev/null
"$stage4_dir/checker.stage4" "$stage4_dir/array_for.nodes" > "$stage4_dir/array_for.check"
grep -qx "ok" "$stage4_dir/array_for.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/array_for.nodes" > "$stage4_dir/array_for.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/array_for" "$stage4_dir/array_for.c" >/dev/null 2>&1
stage4_array_for_out="$("$stage4_dir/array_for")"
test "$stage4_array_for_out" = "Tya"
echo "stage4 array for: self-host pipeline matched"
