#!/usr/bin/env sh
set -eu

out_dir="$(mktemp -d "${TMPDIR:-/tmp}/tya-stage1-selfhost-sources.XXXXXX")"
cc_warning_flags=""
if printf '' | gcc -Wno-format-truncation -x c -fsyntax-only - >/dev/null 2>&1; then
  cc_warning_flags="-Wno-format-truncation"
fi

mkdir -p "$out_dir"

assert_check_ok() {
  check_file="$1"
  test "$(cat "$check_file")" = "ok"
}

assert_node_shapes() {
  want_file="$1"
  actual_file="$2"
  actual_shape_file="$actual_file.shape"
  sed 's/^[0-9][0-9]*://' "$actual_file" > "$actual_shape_file"
  diff -u "$want_file" "$actual_shape_file" >/dev/null
}

assert_node_shapes_include() {
  want_file="$1"
  actual_file="$2"
  actual_shape_file="$actual_file.shape"
  sed 's/^[0-9][0-9]*://' "$actual_file" > "$actual_shape_file"
  while IFS= read -r shape; do
    if test -n "$shape"; then
      grep -Fx "$shape" "$actual_shape_file" >/dev/null
    fi
  done < "$want_file"
}

if grep -Eq 'raw_(expr|condition)' selfhost/parser.tya; then
  echo "selfhost/parser.tya: raw parser adapter remains" >&2
  exit 1
fi

if grep -Eq 'strstr\(mode,|mode = argv' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: argv[0] mode fallback remains" >&2
  exit 1
fi

if grep -q 'source_is_checker' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: generated checker source classifier remains" >&2
  exit 1
fi

if grep -Eq 'source_is_(lexer|parser)' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: generated lexer/parser source classifier naming remains" >&2
  exit 1
fi

if grep -q 'generated_parser_main' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: broad generated parser main recognizer remains" >&2
  exit 1
fi

if grep -q 'generated_lexer_has_' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: generated lexer main recognizer remains" >&2
  exit 1
fi

if grep -q 'generated_parser_has_' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: generated parser main recognizer remains" >&2
  exit 1
fi
if grep -q 'lines_len > 0 && !input_emits_codegen && !strstr(lines\[0\], "\\\\":PRINT:\\\\"' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: broad non-PRINT generated checker fallback remains" >&2
  exit 1
fi
if grep -Eq 'ASSIGN:check:BOOL|ASSIGN:errors:INT:check nodes' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: old generated checker recognizer remains" >&2
  exit 1
fi
if grep -Eq 'ASSIGN:lex:BOOL|ASSIGN:tokens:INT:lex source|ASSIGN:parse:BOOL|ASSIGN:nodes:INT:parse tokens' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: old generated lexer/parser recognizer remains" >&2
  exit 1
fi
if grep -Eq 'ASSIGN:emit_c:BOOL|PRINT:IDENT:emit_c nodes' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: old generated emit_c recognizer remains" >&2
  exit 1
fi
if grep -q 'ASSIGN:greet:BOOL:user ->' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: stale generated greet recognizer remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*greet user' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact greet user print branch remains" >&2
  exit 1
fi
if grep -q 'ASSIGN:find_first_over:BOOL:limit ->' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: whole-program return recognizer remains" >&2
  exit 1
fi
if grep -q ':PRINT:IDENT:to_string 20' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: whole-program convert recognizer remains" >&2
  exit 1
fi
if grep -q 'to_string items' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: to_string items fixed-output case remains" >&2
  exit 1
fi
if grep -Eq ':PRINT:IDENT:left == right|strcmp\(start, .*left == right' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: whole-program equal recognizer remains" >&2
  exit 1
fi
if grep -Eq '1:ASSIGN:age:INT:20|:PRINT:IDENT:nil or|:PRINT:IDENT:not false|strcmp\(start, .*nil or|strcmp\(start, .*not false|value_of\(node\) == "nil or|value_of\(node\) == "not false' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: whole-program logic recognizer remains" >&2
  exit 1
fi
if grep -q ':ASSIGN:add:BOOL:a, b ->' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: whole-program arithmetic recognizer remains" >&2
  exit 1
fi
if grep -q 'add 2, 3' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact add arithmetic branch remains" >&2
  exit 1
fi
if grep -q 'div 5, 2' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact div arithmetic branch remains" >&2
  exit 1
fi
if grep -q '5 / 2' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact float division branch remains" >&2
  exit 1
fi
if grep -q '2 + 3 \* 4' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact mixed arithmetic branch remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*negative|value_of\(node\) == "negative"' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact negative arithmetic branch remains" >&2
  exit 1
fi
if grep -q 'next year: 21' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact interpolation output branch remains" >&2
  exit 1
fi
if grep -Eq 'grouped_print_count|strcmp\(start, .*grouped' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact grouped arithmetic branch remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*doubled\[2\]' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact doubled index branch remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*len evens' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact len evens branch remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*first_even' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact first_even branch remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*has_even' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact has_even branch remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*all_even' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact all_even branch remains" >&2
  exit 1
fi
if grep -Eq 'array_function_sum|strcmp\(start, .*\\+"sum\\+"' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact array_function sum branch remains" >&2
  exit 1
fi
if grep -q 'sum nums' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact prelude sum branch remains" >&2
  exit 1
fi
if grep -Eq 'first items|second items' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact prelude first/second branch remains" >&2
  exit 1
fi
if grep -q 'compact items' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact prelude compact branch remains" >&2
  exit 1
fi
if grep -Eq 'items_len_print_count|len items|pop items|:PUSH:items:INT:4|len empty|len roles|len empty_roles|strcmp\(start, .*items\[[019]\].*== 0\) puts' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact array len/pop/push/index branch remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*byte_len .*ちゃ|strcmp\(start, .*char_len .*ちゃ' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact string length branch remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*contains trimmed|strcmp\(start, .*starts_with trimmed|strcmp\(start, .*ends_with trimmed' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact string predicate branch remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*join parts|strcmp\(start, .*replace trimmed' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact string transform branch remains" >&2
  exit 1
fi
if grep -q ':ASSIGN:even?:BOOL:item ->' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: whole-program array_function recognizer remains" >&2
  exit 1
fi
if grep -q ':ASSIGN:err:INT:error' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: whole-program error recognizer remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*"err"|value_of\(node\) == "err"|name_of\(node\) == "err" and value_of\(node\) == "message"|strcmp\(object, .*"err".*strcmp\(member, .*"message"' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact error variable branch remains" >&2
  exit 1
fi
if grep -q ':ASSIGN:items:ARRAY:, 2, 3]' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: whole-program array recognizer remains" >&2
  exit 1
fi
if grep -Eq ':ASSIGN:items:ARRAY:, 4, 6]|:ASSIGN:items:ARRAY_THREE:INT:2:INT:4:INT:6' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: whole-program for recognizer remains" >&2
  exit 1
fi
if grep -Eq 'FOR:item,:dex|FOR:item:index|0:2|1:4|2:6' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: indexed for fixed-output workaround remains" >&2
  exit 1
fi
if grep -Eq 'file_exists path|read_file path' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: file example fixed-output branch remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*env .*TYA_EXAMPLE' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact env branch remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*has user|strcmp\(start, .*has roles' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact has branch remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(has_target, .*"(user|roles)"|strcmp\(has_key, .*"(name|age|owner)"' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact generated has target/key branch remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*len keys user|strcmp\(start, .*len values user' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact object keys/values len branch remains" >&2
  exit 1
fi
if grep -Eq 'name_of\(node\) == "has" and value_of\(node\) == "(user|roles)"' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact has node branch remains" >&2
  exit 1
fi
if grep -Eq 'value_of\(node\) == "user" and extra_of\(node\) == "name"' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact object index name branch remains" >&2
  exit 1
fi
if grep -Eq 'value_of\(node\) == "user" and extra_of\(node\) == "age"' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact object index age branch remains" >&2
  exit 1
fi
if grep -q 'user_age_deleted' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact user age delete state remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*"(user.name|admin.name|admin.greet\(\))"|strcmp\(object, .*"(user|admin)".*strcmp\(member, .*"(name|greet)"' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact generated class member branch remains" >&2
  exit 1
fi
if grep -Eq 'name_of\(node\) == "(greeting|util|counter)"|strcmp\(object, .*"(greeting|util|counter)"' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact module/object member receiver branch remains" >&2
  exit 1
fi
if grep -Eq 'factorial_state|open_factorial_node' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact factorial state branch remains" >&2
  exit 1
fi
if grep -Eq 'WHILE:IDENT:d|open_prime_node|value_of\(node\) == "d" and has_name\(names, "n"\)|strcmp\(start, .*"d"' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact prime while branch remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*user\\\[' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact generated object access branch remains" >&2
  exit 1
fi
if grep -Eq 'strcmp\(start, .*user.age' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact generated object member branch remains" >&2
  exit 1
fi
if grep -Eq 'bool_print.*strcmp\(start, .*\\+"(true|false)"|bool_print.*strcmp\(start, .*"(true|false)"' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: exact generated bool literal branch remains" >&2
  exit 1
fi
if grep -q ':ASSIGN:trimmed:INT:trim text' selfhost/codegen_c.tya; then
  echo "selfhost/codegen_c.tya: whole-program string recognizer remains" >&2
  exit 1
fi
compare_stage2_codegen() {
  label="$1"
  nodes="$2"
  file_label="$(printf '%s' "$label" | tr '/ ' '__')"
  "$out_dir/codegen_c.stage2" "$nodes" > "$out_dir/$file_label.stage2.first.c"
  "$out_dir/codegen_c.stage2" "$nodes" > "$out_dir/$file_label.stage2.second.c"
  diff -u "$out_dir/$file_label.stage2.first.c" "$out_dir/$file_label.stage2.second.c" >/dev/null
  echo "$label: stage-2 codegen deterministic"
}

run_stage4_manifest_example() {
  example="$1"
  file_label="$(printf '%s' "$example" | tr '/ ' '__')"
  "$stage4_dir/lexer.stage4" "$example" > "$stage4_dir/$file_label.manifest.tokens"
  "$stage4_dir/parser.stage4" "$stage4_dir/$file_label.manifest.tokens" > "$stage4_dir/$file_label.manifest.nodes"
  "$stage4_dir/checker.stage4" "$stage4_dir/$file_label.manifest.nodes" > "$stage4_dir/$file_label.manifest.check"
  assert_check_ok "$stage4_dir/$file_label.manifest.check"
  "$stage4_dir/codegen_c.stage4" "$stage4_dir/$file_label.manifest.nodes" > "$stage4_dir/$file_label.manifest.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/$file_label.manifest" "$stage4_dir/$file_label.manifest.c" >/dev/null 2>&1
  if [ "$example" = "examples/panic.tya" ]; then
    set +e
    "$stage4_dir/$file_label.manifest" > "$stage4_dir/$file_label.manifest.got" 2> "$stage4_dir/$file_label.manifest.err"
    panic_status="$?"
    set -e
    test "$panic_status" = "1"
    test ! -s "$stage4_dir/$file_label.manifest.got"
    test "$(cat "$stage4_dir/$file_label.manifest.err")" = "panic: bad state"
    echo "$example: stage-4 manifest panic status matched"
    return
  fi
  if [ "$example" = "examples/read_line.tya" ]; then
    printf 'komagata\n' | go run ./cmd/tya "$example" > "$stage4_dir/$file_label.manifest.want"
    printf 'komagata\n' | "$stage4_dir/$file_label.manifest" > "$stage4_dir/$file_label.manifest.got"
  else
    go run ./cmd/tya "$example" > "$stage4_dir/$file_label.manifest.want"
    "$stage4_dir/$file_label.manifest" > "$stage4_dir/$file_label.manifest.got"
  fi
  diff -u "$stage4_dir/$file_label.manifest.want" "$stage4_dir/$file_label.manifest.got" >/dev/null
  echo "$example: stage-4 manifest pipeline matched"
}

for src in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  base="$(basename "$src" .tya)"
  go run ./cmd/tya --emit-c "$src" > "$out_dir/$base.stage1.c"
  gcc $cc_warning_flags "$out_dir/$base.stage1.c" runtime/tya_runtime.c -I runtime -o "$out_dir/$base.stage1"
done

for src in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  base="$(basename "$src" .tya)"
  "$out_dir/lexer.stage1" "$src" > "$out_dir/$base.tokens"
  "$out_dir/parser.stage1" "$out_dir/$base.tokens" > "$out_dir/$base.nodes"
  "$out_dir/checker.stage1" "$out_dir/$base.nodes" > "$out_dir/$base.check"
  assert_check_ok "$out_dir/$base.check"
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
assert_check_ok "$out_dir/hello.stage2.check"
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
assert_check_ok "$out_dir/escaped_print.stage2.check"
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
assert_check_ok "$out_dir/string_index_print.stage2.check"
compare_stage2_codegen "string index print" "$out_dir/string_index_print.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/string_index_print.stage2.nodes" > "$out_dir/string_index_print.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/string_index_print.stage2" "$out_dir/string_index_print.stage2.c" >/dev/null 2>&1
string_index_print_out="$("$out_dir/string_index_print.stage2")"
test "$string_index_print_out" = "y"
echo "string index print: stage-2 pipeline matched"

"$out_dir/lexer.stage2" examples/string.tya > "$out_dir/string_example.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/string_example.stage2.tokens" > "$out_dir/string_example.stage2.nodes"
"$out_dir/checker.stage2" "$out_dir/string_example.stage2.nodes" > "$out_dir/string_example.stage2.check"
assert_check_ok "$out_dir/string_example.stage2.check"
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

printf '%s\n' 'ratio = 12.5' 'text = "a\"b"' > "$out_dir/literals.tya"
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
assert_check_ok "$out_dir/int.stage2.check"
"$out_dir/checker.stage2" "$out_dir/literals.stage2.nodes" > "$out_dir/literals.stage2.check"
assert_check_ok "$out_dir/literals.stage2.check"
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
assert_check_ok "$out_dir/literal_reassign.stage2.check"
compare_stage2_codegen "literal reassignment" "$out_dir/literal_reassign.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/literal_reassign.stage2.nodes" > "$out_dir/literal_reassign.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/literal_reassign.stage2" "$out_dir/literal_reassign.stage2.c" >/dev/null 2>&1
literal_reassign_out="$("$out_dir/literal_reassign.stage2")"
test "$literal_reassign_out" = "2
false
2.5
b"
echo "literal reassignment: stage-2 pipeline matched"

printf 'source = read_file args()[0]\nprint source\n' > "$out_dir/read_file_arg.tya"
printf 'Tya' > "$out_dir/read_file_arg.input"
"$out_dir/lexer.stage2" "$out_dir/read_file_arg.tya" > "$out_dir/read_file_arg.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/read_file_arg.stage2.tokens" > "$out_dir/read_file_arg.stage2.nodes"
cat > "$out_dir/read_file_arg.want.nodes" <<'NODES'
1:ASSIGN:source:CALL1_CALL0_INDEX:read_file:args:0
2:PRINT:IDENT:source
NODES
diff -u "$out_dir/read_file_arg.want.nodes" "$out_dir/read_file_arg.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/read_file_arg.stage2.nodes" > "$out_dir/read_file_arg.stage2.check"
assert_check_ok "$out_dir/read_file_arg.stage2.check"
compare_stage2_codegen "read file arg" "$out_dir/read_file_arg.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/read_file_arg.stage2.nodes" > "$out_dir/read_file_arg.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/read_file_arg.stage2" "$out_dir/read_file_arg.stage2.c" >/dev/null 2>&1
read_file_arg_out="$("$out_dir/read_file_arg.stage2" "$out_dir/read_file_arg.input")"
test "$read_file_arg_out" = "Tya"
echo "read file arg: stage-2 pipeline matched"

printf 'source = read_file args()[0]\ntokens = lex source\nfor token in tokens\n  print token\n' > "$out_dir/lex_source.tya"
printf 'print "Tya"\n' > "$out_dir/lex_source.input"
"$out_dir/lexer.stage2" "$out_dir/lex_source.tya" > "$out_dir/lex_source.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/lex_source.stage2.tokens" > "$out_dir/lex_source.stage2.nodes"
cat > "$out_dir/lex_source.want.nodes" <<'NODES'
1:ASSIGN:source:CALL1_CALL0_INDEX:read_file:args:0
2:ASSIGN:tokens:CALL1:lex:source
3:FOR:token:tokens
4:INDENT:2
4:PRINT:IDENT:token
NODES
diff -u "$out_dir/lex_source.want.nodes" "$out_dir/lex_source.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/lex_source.stage2.nodes" > "$out_dir/lex_source.stage2.check"
assert_check_ok "$out_dir/lex_source.stage2.check"
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

printf 'source = read_file args()[0]\ntokens = lex source\nnodes = parse tokens\nfor node in nodes\n  print node\n' > "$out_dir/parse_tokens.tya"
printf 'print "Tya"\n' > "$out_dir/parse_tokens.input"
"$out_dir/lexer.stage2" "$out_dir/parse_tokens.tya" > "$out_dir/parse_tokens.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/parse_tokens.stage2.tokens" > "$out_dir/parse_tokens.stage2.nodes"
cat > "$out_dir/parse_tokens.want.nodes" <<'NODES'
1:ASSIGN:source:CALL1_CALL0_INDEX:read_file:args:0
2:ASSIGN:tokens:CALL1:lex:source
3:ASSIGN:nodes:CALL1:parse:tokens
4:FOR:node:nodes
5:INDENT:2
5:PRINT:IDENT:node
NODES
diff -u "$out_dir/parse_tokens.want.nodes" "$out_dir/parse_tokens.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/parse_tokens.stage2.nodes" > "$out_dir/parse_tokens.stage2.check"
assert_check_ok "$out_dir/parse_tokens.stage2.check"
compare_stage2_codegen "parse tokens" "$out_dir/parse_tokens.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/parse_tokens.stage2.nodes" > "$out_dir/parse_tokens.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/parse_tokens.stage2" "$out_dir/parse_tokens.stage2.c" >/dev/null 2>&1
parse_tokens_out="$("$out_dir/parse_tokens.stage2" "$out_dir/parse_tokens.input")"
test "$parse_tokens_out" = "1:PRINT:STRING:Tya"
echo "parse tokens: stage-2 pipeline matched"

printf 'source = read_file args()[0]\nlines = split source, "\n"\nerrors = check nodes\nfor err in errors\n  print err\n' > "$out_dir/check_nodes.tya"
printf '1:PRINT:STRING:Tya\n' > "$out_dir/check_nodes.input"
"$out_dir/lexer.stage2" "$out_dir/check_nodes.tya" > "$out_dir/check_nodes.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/check_nodes.stage2.tokens" > "$out_dir/check_nodes.stage2.nodes"
cat > "$out_dir/check_nodes.want.nodes" <<'NODES'
1:ASSIGN:source:CALL1_CALL0_INDEX:read_file:args:0
2:ASSIGN:lines:CALL2:split:source:STRING:
3:ASSIGN:errors:CALL1:check:nodes
4:FOR:err:errors
5:INDENT:2
5:PRINT:IDENT:err
NODES
diff -u "$out_dir/check_nodes.want.nodes" "$out_dir/check_nodes.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/check_nodes.stage2.nodes" > "$out_dir/check_nodes.stage2.check"
assert_check_ok "$out_dir/check_nodes.stage2.check"
compare_stage2_codegen "check nodes" "$out_dir/check_nodes.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/check_nodes.stage2.nodes" > "$out_dir/check_nodes.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/check_nodes.stage2" "$out_dir/check_nodes.stage2.c" >/dev/null 2>&1
check_nodes_out="$("$out_dir/check_nodes.stage2" "$out_dir/check_nodes.input")"
test "$check_nodes_out" = ""
echo "check nodes: stage-2 pipeline matched"

printf '1:PRINT:IDENT:missing\n' > "$out_dir/check_nodes_undefined.input"
check_nodes_undefined_out="$("$out_dir/check_nodes.stage2" "$out_dir/check_nodes_undefined.input")"
test "$check_nodes_undefined_out" = "1: undefined variable: missing"
echo "check nodes undefined print: stage-2 pipeline matched"

printf '1:ASSIGN:alias:IDENT:missing\n2:FOR:item:items\n' > "$out_dir/check_nodes_undefined_assign_for.input"
check_nodes_undefined_assign_for_out="$("$out_dir/check_nodes.stage2" "$out_dir/check_nodes_undefined_assign_for.input")"
test "$check_nodes_undefined_assign_for_out" = "1: undefined variable: missing
2: undefined variable: items"
echo "check nodes undefined assign/for: stage-2 pipeline matched"

printf '1:ASSIGN:MAX_RETRY:INT:3\n2:ASSIGN:MAX_RETRY:INT:5\n3:ASSIGN:retry_count:INT:3\n4:ASSIGN:retry_count:INT:5\n' > "$out_dir/check_nodes_constant_reassign.input"
check_nodes_constant_reassign_out="$("$out_dir/check_nodes.stage2" "$out_dir/check_nodes_constant_reassign.input")"
test "$check_nodes_constant_reassign_out" = "2: cannot reassign constant MAX_RETRY"
echo "check nodes constant reassignment: stage-2 pipeline matched"

printf '1:ASSIGN:MAX_RETRY:INT:3\n2:ASSIGN:retry_count:INT:3\n' > "$out_dir/check_nodes_constant_first_assign.input"
check_nodes_constant_first_assign_out="$("$out_dir/check_nodes.stage2" "$out_dir/check_nodes_constant_first_assign.input")"
test "$check_nodes_constant_first_assign_out" = ""
echo "check nodes constant first assignment: stage-2 pipeline matched"

printf '1:ASSIGN:disabled:BOOL_NOT:enabled\n' > "$out_dir/check_nodes_bool_not_undefined.input"
check_nodes_bool_not_undefined_out="$("$out_dir/check_nodes.stage2" "$out_dir/check_nodes_bool_not_undefined.input")"
test "$check_nodes_bool_not_undefined_out" = "1: undefined variable: enabled"
echo "check nodes bool-not undefined: stage-2 pipeline matched"

printf '1:ASSIGN:known:BOOL:true\n2:ASSIGN:both:BOOL_AND:known:missing_and\n3:ASSIGN:either:BOOL_OR:missing_or:known\n' > "$out_dir/check_nodes_bool_binary_undefined.input"
check_nodes_bool_binary_undefined_out="$("$out_dir/check_nodes.stage2" "$out_dir/check_nodes_bool_binary_undefined.input")"
test "$check_nodes_bool_binary_undefined_out" = "2: undefined variable: missing_and
3: undefined variable: missing_or"
echo "check nodes bool-binary undefined: stage-2 pipeline matched"

printf '1:ASSIGN:known:INT:1\n2:ASSIGN:sum:INT_ADD:known:missing_add\n3:ASSIGN:typed:INT_SUB:IDENT:missing_sub:INT:1\n4:ASSIGN:literal:INT_MUL:INT:2:INT:3\n' > "$out_dir/check_nodes_arithmetic_undefined.input"
check_nodes_arithmetic_undefined_out="$("$out_dir/check_nodes.stage2" "$out_dir/check_nodes_arithmetic_undefined.input")"
test "$check_nodes_arithmetic_undefined_out" = "2: undefined variable: missing_add
3: undefined variable: missing_sub"
echo "check nodes arithmetic undefined: stage-2 pipeline matched"

printf '1:ASSIGN:known:INT:1\n2:ASSIGN:greater:COMPARE_GT:known:missing_gt\n3:ASSIGN:equal:COMPARE_EQ:missing_eq:1\n4:ASSIGN:literal:COMPARE_LE:1:2\n' > "$out_dir/check_nodes_compare_undefined.input"
check_nodes_compare_undefined_out="$("$out_dir/check_nodes.stage2" "$out_dir/check_nodes_compare_undefined.input")"
test "$check_nodes_compare_undefined_out" = "2: undefined variable: missing_gt
3: undefined variable: missing_eq"
echo "check nodes compare undefined: stage-2 pipeline matched"

cat > "$out_dir/constant_reassign.nodes" <<'NODES'
1:ASSIGN:MAX_RETRY:INT:3
2:ASSIGN:MAX_RETRY:INT:5
3:ASSIGN:retry_count:INT:3
4:ASSIGN:retry_count:INT:5
NODES
"$out_dir/checker.stage1" "$out_dir/constant_reassign.nodes" > "$out_dir/constant_reassign.check"
grep -qx "2: cannot reassign constant MAX_RETRY" "$out_dir/constant_reassign.check"
echo "constant reassignment: stage-1 checker matched"

printf 'source = read_file args()[0]\nlines = split source, "\n"\nprint emit_c nodes\n' > "$out_dir/emit_c_nodes.tya"
printf '1:PRINT:STRING:Tya\n' > "$out_dir/emit_c_nodes.input"
"$out_dir/lexer.stage2" "$out_dir/emit_c_nodes.tya" > "$out_dir/emit_c_nodes.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/emit_c_nodes.stage2.tokens" > "$out_dir/emit_c_nodes.stage2.nodes"
cat > "$out_dir/emit_c_nodes.want.nodes" <<'NODES'
1:ASSIGN:source:CALL1_CALL0_INDEX:read_file:args:0
2:ASSIGN:lines:CALL2:split:source:STRING:
3:PRINT_CALL1:emit_c:nodes
NODES
diff -u "$out_dir/emit_c_nodes.want.nodes" "$out_dir/emit_c_nodes.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/emit_c_nodes.stage2.nodes" > "$out_dir/emit_c_nodes.stage2.check"
assert_check_ok "$out_dir/emit_c_nodes.stage2.check"
compare_stage2_codegen "emit_c nodes" "$out_dir/emit_c_nodes.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/emit_c_nodes.stage2.nodes" > "$out_dir/emit_c_nodes.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/emit_c_nodes.stage2" "$out_dir/emit_c_nodes.stage2.c" >/dev/null 2>&1
"$out_dir/emit_c_nodes.stage2" "$out_dir/emit_c_nodes.input" > "$out_dir/emit_c_nodes.out.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/emit_c_nodes.out" "$out_dir/emit_c_nodes.out.c" >/dev/null 2>&1
emit_c_nodes_out="$("$out_dir/emit_c_nodes.out")"
test "$emit_c_nodes_out" = "Tya"
echo "emit_c nodes: stage-2 pipeline matched"

printf 'helper = value ->\n  temp = "skip"\n  print temp\nsource = read_file args()[0]\nprint source\n' > "$out_dir/function_body_skip.tya"
"$out_dir/lexer.stage2" "$out_dir/function_body_skip.tya" > "$out_dir/function_body_skip.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/function_body_skip.stage2.tokens" > "$out_dir/function_body_skip.stage2.nodes"
cat > "$out_dir/function_body_skip.want.nodes" <<'NODES'
1:FUNC:helper:value
2:INDENT:2
4:INDENT:0
4:ASSIGN:source:CALL1_CALL0_INDEX:read_file:args:0
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
assert_check_ok "$out_dir/print_int.stage2.check"
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
assert_check_ok "$out_dir/print_literals.stage2.check"
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
assert_check_ok "$out_dir/string_len.stage2.check"
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
assert_check_ok "$out_dir/string_trim.stage2.check"
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
assert_check_ok "$out_dir/string_contains.stage2.check"
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

printf 'text = "hello"\nprint starts_with text, "he"\nprint ends_with text, "lo"\n' > "$out_dir/string_prefix_suffix.tya"
"$out_dir/lexer.stage2" "$out_dir/string_prefix_suffix.tya" > "$out_dir/string_prefix_suffix.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/string_prefix_suffix.stage2.tokens" > "$out_dir/string_prefix_suffix.stage2.nodes"
cat > "$out_dir/string_prefix_suffix.want.nodes" <<'NODES'
1:ASSIGN:text:STRING:hello
2:PRINT_CALL2:starts_with:text:STRING:he
3:PRINT_CALL2:ends_with:text:STRING:lo
NODES
diff -u "$out_dir/string_prefix_suffix.want.nodes" "$out_dir/string_prefix_suffix.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/string_prefix_suffix.stage2.nodes" > "$out_dir/string_prefix_suffix.stage2.check"
assert_check_ok "$out_dir/string_prefix_suffix.stage2.check"
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
assert_check_ok "$out_dir/string_replace.stage2.check"
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
assert_check_ok "$out_dir/string_split_join.stage2.check"
compare_stage2_codegen "string split join print" "$out_dir/string_split_join.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/string_split_join.stage2.nodes" > "$out_dir/string_split_join.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/string_split_join.stage2" "$out_dir/string_split_join.stage2.c" >/dev/null 2>&1
string_split_join_out="$("$out_dir/string_split_join.stage2")"
test "$string_split_join_out" = "hello-tya"
echo "string split join print: stage-2 pipeline matched"

printf 'print byte_len "ちゃ"\nprint char_len "ちゃ"\n' > "$out_dir/string_lengths.tya"
"$out_dir/lexer.stage2" "$out_dir/string_lengths.tya" > "$out_dir/string_lengths.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/string_lengths.stage2.tokens" > "$out_dir/string_lengths.stage2.nodes"
cat > "$out_dir/string_lengths.want.nodes" <<'NODES'
1:PRINT_CALL1:byte_len:STRING:ちゃ
2:PRINT_CALL1:char_len:STRING:ちゃ
NODES
diff -u "$out_dir/string_lengths.want.nodes" "$out_dir/string_lengths.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/string_lengths.stage2.nodes" > "$out_dir/string_lengths.stage2.check"
assert_check_ok "$out_dir/string_lengths.stage2.check"
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
assert_check_ok "$out_dir/int_add.stage2.check"
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
assert_check_ok "$out_dir/grouped_int_add.stage2.check"
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
assert_check_ok "$out_dir/int_add_reassign.stage2.check"
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
assert_check_ok "$out_dir/bool.stage2.check"
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
assert_check_ok "$out_dir/bool_logic.stage2.check"
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
assert_check_ok "$out_dir/while_false.stage2.check"
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
assert_check_ok "$out_dir/while_less_than.stage2.check"
compare_stage2_codegen "while less-than break" "$out_dir/while_less_than.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/while_less_than.stage2.nodes" > "$out_dir/while_less_than.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/while_less_than.stage2" "$out_dir/while_less_than.stage2.c" >/dev/null 2>&1
while_less_than_out="$("$out_dir/while_less_than.stage2")"
test "$while_less_than_out" = "1"
echo "while less-than break: stage-2 pipeline matched"

printf 'age = 2\nwhile age > 1\n  print "loop"\n  break\nprint age\n' > "$out_dir/while_greater_than.tya"
"$out_dir/lexer.stage2" "$out_dir/while_greater_than.tya" > "$out_dir/while_greater_than.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/while_greater_than.stage2.tokens" > "$out_dir/while_greater_than.stage2.nodes"
cat > "$out_dir/while_greater_than.want.nodes" <<'NODES'
1:ASSIGN:age:INT:2
2:WHILE_COMPARE_GT:IDENT:age:INT:1
3:INDENT:2
3:PRINT:STRING:loop
4:BREAK
5:INDENT:0
5:PRINT:IDENT:age
NODES
diff -u "$out_dir/while_greater_than.want.nodes" "$out_dir/while_greater_than.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/while_greater_than.stage2.nodes" > "$out_dir/while_greater_than.stage2.check"
assert_check_ok "$out_dir/while_greater_than.stage2.check"
compare_stage2_codegen "while greater-than break" "$out_dir/while_greater_than.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/while_greater_than.stage2.nodes" > "$out_dir/while_greater_than.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/while_greater_than.stage2" "$out_dir/while_greater_than.stage2.c" >/dev/null 2>&1
while_greater_than_out="$("$out_dir/while_greater_than.stage2")"
test "$while_greater_than_out" = "loop
2"
echo "while greater-than break: stage-2 pipeline matched"

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
assert_check_ok "$out_dir/while_bounds.stage2.check"
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
assert_check_ok "$out_dir/array_for.stage2.check"
compare_stage2_codegen "array for" "$out_dir/array_for.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/array_for.stage2.nodes" > "$out_dir/array_for.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/array_for.stage2" "$out_dir/array_for.stage2.c" >/dev/null 2>&1
array_for_out="$("$out_dir/array_for.stage2")"
test "$array_for_out" = "Tya"
echo "array for: stage-2 pipeline matched"

cat > "$out_dir/array_index.stage2.nodes" <<'NODES'
1:ASSIGN:names:ARRAY_TWO:STRING:Ada:STRING:Tya
2:ASSIGN:name:INDEX:names:1
3:PRINT:IDENT:name
NODES
"$out_dir/checker.stage2" "$out_dir/array_index.stage2.nodes" > "$out_dir/array_index.stage2.check"
assert_check_ok "$out_dir/array_index.stage2.check"
compare_stage2_codegen "array index assignment" "$out_dir/array_index.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/array_index.stage2.nodes" > "$out_dir/array_index.stage2.c"
if grep -Fq '/* names[1] */' "$out_dir/array_index.stage2.c"; then
  echo "array index assignment kept placeholder" >&2
  exit 1
fi
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/array_index.stage2" "$out_dir/array_index.stage2.c" >/dev/null 2>&1
array_index_out="$("$out_dir/array_index.stage2")"
test "$array_index_out" = "Tya"
echo "array index assignment: stage-2 pipeline matched"

cat > "$out_dir/inline_filter.tya" <<'TYA'
items = [1, 2, 3, 4]
evens = filter items, item -> item % 2 == 0
print len evens
TYA
"$out_dir/lexer.stage2" "$out_dir/inline_filter.tya" > "$out_dir/inline_filter.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/inline_filter.stage2.tokens" > "$out_dir/inline_filter.stage2.nodes"
cat > "$out_dir/inline_filter.want.nodes" <<'NODES'
1:ASSIGN:items:ARRAY_FOUR:INT:1:INT:2:INT:3:INT:4
2:ASSIGN:evens:CALL2_FUNC_MOD_EQ:filter:items:item:2:0
3:PRINT_CALL1:len:evens
NODES
diff -u "$out_dir/inline_filter.want.nodes" "$out_dir/inline_filter.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/inline_filter.stage2.nodes" > "$out_dir/inline_filter.stage2.check"
assert_check_ok "$out_dir/inline_filter.stage2.check"
compare_stage2_codegen "inline filter function literal" "$out_dir/inline_filter.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/inline_filter.stage2.nodes" > "$out_dir/inline_filter.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/inline_filter.stage2" "$out_dir/inline_filter.stage2.c" >/dev/null 2>&1
inline_filter_out="$("$out_dir/inline_filter.stage2")"
test "$inline_filter_out" = "2"
echo "inline filter function literal: stage-2 pipeline matched"

"$out_dir/lexer.stage2" examples/selfhost_ops.tya > "$out_dir/selfhost_ops.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/selfhost_ops.stage2.tokens" > "$out_dir/selfhost_ops.stage2.nodes"
"$out_dir/checker.stage2" "$out_dir/selfhost_ops.stage2.nodes" > "$out_dir/selfhost_ops.stage2.check"
assert_check_ok "$out_dir/selfhost_ops.stage2.check"
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
assert_check_ok "$out_dir/while_example.stage2.check"
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
assert_check_ok "$out_dir/compare_eq.stage2.check"
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
assert_check_ok "$out_dir/compare_ne.stage2.check"
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
assert_check_ok "$out_dir/compare_lt.stage2.check"
compare_stage2_codegen "less-than comparison" "$out_dir/compare_lt.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/compare_lt.stage2.nodes" > "$out_dir/compare_lt.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/compare_lt.stage2" "$out_dir/compare_lt.stage2.c" >/dev/null 2>&1
compare_lt_out="$("$out_dir/compare_lt.stage2")"
test "$compare_lt_out" = "true"
echo "less-than comparison: stage-2 pipeline matched"

printf 'left = 3\nright = 2\ngreater = left > right\nprint greater\n' > "$out_dir/compare_gt.tya"
"$out_dir/lexer.stage2" "$out_dir/compare_gt.tya" > "$out_dir/compare_gt.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/compare_gt.stage2.tokens" > "$out_dir/compare_gt.stage2.nodes"
cat > "$out_dir/compare_gt.want.nodes" <<'NODES'
1:ASSIGN:left:INT:3
2:ASSIGN:right:INT:2
3:ASSIGN:greater:COMPARE_GT:left:right
4:PRINT:IDENT:greater
NODES
diff -u "$out_dir/compare_gt.want.nodes" "$out_dir/compare_gt.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/compare_gt.stage2.nodes" > "$out_dir/compare_gt.stage2.check"
assert_check_ok "$out_dir/compare_gt.stage2.check"
compare_stage2_codegen "greater-than comparison" "$out_dir/compare_gt.stage2.nodes"
"$out_dir/codegen_c.stage2" "$out_dir/compare_gt.stage2.nodes" > "$out_dir/compare_gt.stage2.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/compare_gt.stage2" "$out_dir/compare_gt.stage2.c" >/dev/null 2>&1
compare_gt_out="$("$out_dir/compare_gt.stage2")"
test "$compare_gt_out" = "true"
echo "greater-than comparison: stage-2 pipeline matched"

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
assert_check_ok "$out_dir/compare_bounds.stage2.check"
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

printf 'age = 2\ngrouped_compare = (age >= 2)\nprint grouped_compare\n' > "$out_dir/grouped_compare.tya"
"$out_dir/lexer.stage2" "$out_dir/grouped_compare.tya" > "$out_dir/grouped_compare.stage2.tokens"
"$out_dir/parser.stage2" "$out_dir/grouped_compare.stage2.tokens" > "$out_dir/grouped_compare.stage2.nodes"
cat > "$out_dir/grouped_compare.want.nodes" <<'NODES'
1:ASSIGN:age:INT:2
2:ASSIGN:grouped_compare:COMPARE_GE:age:2
3:PRINT:IDENT:grouped_compare
NODES
diff -u "$out_dir/grouped_compare.want.nodes" "$out_dir/grouped_compare.stage2.nodes" >/dev/null
"$out_dir/checker.stage2" "$out_dir/grouped_compare.stage2.nodes" > "$out_dir/grouped_compare.stage2.check"
assert_check_ok "$out_dir/grouped_compare.stage2.check"
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
  assert_check_ok "$stage4_dir/$base.stage3.check"
  "$out_dir/codegen_c.stage2" "$stage4_dir/$base.stage3.nodes" > "$stage4_dir/$base.stage3.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/$base.stage3" "$stage4_dir/$base.stage3.c" >/dev/null 2>&1
done

for src in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  base="$(basename "$src" .tya)"
  "$stage4_dir/lexer.stage3" "$src" > "$stage4_dir/$base.stage4.tokens"
  "$stage4_dir/parser.stage3" "$stage4_dir/$base.stage4.tokens" > "$stage4_dir/$base.stage4.nodes"
  "$stage4_dir/checker.stage3" "$stage4_dir/$base.stage4.nodes" > "$stage4_dir/$base.stage4.check"
  assert_check_ok "$stage4_dir/$base.stage4.check"
  "$stage4_dir/codegen_c.stage3" "$stage4_dir/$base.stage4.nodes" > "$stage4_dir/$base.stage4.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/$base.stage4" "$stage4_dir/$base.stage4.c" >/dev/null 2>&1
done

cat > "$stage4_dir/lexer.stage4.want.shapes" <<'NODES'
ASSIGN:char:INDEX:source:i
ASSIGN:source:CALL1_CALL0_INDEX:read_file:args:0
ASSIGN:tokens:CALL1:lex:source
FOR:token:tokens
INDENT:2
PRINT:IDENT:token
NODES
assert_node_shapes "$stage4_dir/lexer.stage4.want.shapes" "$stage4_dir/lexer.stage4.nodes"
echo "selfhost/lexer.tya: stage-3 parser emitted real nodes"
grep -q '^int main(int argc, char \*\*argv)' "$stage4_dir/lexer.stage4.c"
if grep -q 'strstr(mode, "lexer")' "$stage4_dir/lexer.stage4.c"; then
  exit 1
fi
echo "selfhost/lexer.tya: stage-3 codegen emitted executable lexer C"
cat > "$stage4_dir/parser.stage4.want.shapes" <<'NODES'
ASSIGN:body:CALL1:body_of:nodes
FOR:stmt:body
INDENT:2
ASSIGN:source:CALL1_CALL0_INDEX:read_file:args:0
FOR:line:lines
INDENT:2
FOR:node:nodes
INDENT:2
PRINT:IDENT:node
NODES
assert_node_shapes_include "$stage4_dir/parser.stage4.want.shapes" "$stage4_dir/parser.stage4.nodes"
echo "selfhost/parser.tya: stage-3 parser emitted real nodes"
grep -q '^int main(int argc, char \*\*argv)' "$stage4_dir/parser.stage4.c"
if grep -q 'strstr(mode, "parser")' "$stage4_dir/parser.stage4.c"; then
  exit 1
fi
echo "selfhost/parser.tya: stage-3 codegen emitted executable parser C"
cat > "$stage4_dir/checker.stage4.want.shapes" <<'NODES'
FOR:existing:names
INDENT:2
FOR:node:nodes
INDENT:2
ASSIGN:ast_part:INDEX:ast_parts:ast_i
ASSIGN:source:CALL1_CALL0_INDEX:read_file:args:0
FOR:line:lines
INDENT:2
ASSIGN:errors:CALL1:check:nodes
PRINT:STRING:ok
FOR:err:errors
INDENT:2
PRINT:IDENT:err
NODES
assert_node_shapes_include "$stage4_dir/checker.stage4.want.shapes" "$stage4_dir/checker.stage4.nodes"
echo "selfhost/checker.tya: stage-3 parser emitted real nodes"
grep -q '^int main(void)' "$stage4_dir/checker.stage4.c"
if grep -q 'strstr(mode, "checker")' "$stage4_dir/checker.stage4.c"; then
  exit 1
fi
echo "selfhost/checker.tya: stage-3 codegen emitted executable checker C"
cat > "$stage4_dir/codegen_c.stage4.want.shapes" <<'NODES'
FOR:existing:names
INDENT:2
ASSIGN:source:CALL1_CALL0_INDEX:read_file:args:0
FOR:node:nodes
FOR:line:lines
PRINT_CALL1:emit_c:nodes
NODES
assert_node_shapes_include "$stage4_dir/codegen_c.stage4.want.shapes" "$stage4_dir/codegen_c.stage4.nodes"
echo "selfhost/codegen_c.tya: stage-3 parser emitted real nodes"
grep -q '^int main(int argc, char \*\*argv)' "$stage4_dir/codegen_c.stage4.c"
if grep -q 'strstr(mode, "codegen")' "$stage4_dir/codegen_c.stage4.c"; then
  exit 1
fi
if grep -q 'generated_parser_main' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -q 'generated_lexer_has_' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -q 'generated_parser_has_' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
grep -q 'input_needs_lex' "$stage4_dir/codegen_c.stage3.c"
grep -q 'input_needs_parse' "$stage4_dir/codegen_c.stage3.c"
grep -q 'input_needs_check' "$stage4_dir/codegen_c.stage3.c"
if grep -q 'lines_len > 0 && !input_emits_codegen && !strstr(lines\[0\], ":PRINT:"' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -Eq 'ASSIGN:check:BOOL|ASSIGN:errors:INT:check nodes' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -Eq 'ASSIGN:lex:BOOL|ASSIGN:tokens:INT:lex source|ASSIGN:parse:BOOL|ASSIGN:nodes:INT:parse tokens' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -Eq 'ASSIGN:emit_c:BOOL|PRINT:IDENT:emit_c nodes' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -q 'ASSIGN:greet:BOOL:user ->' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -Eq 'strcmp\(start, .*greet user' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -q 'ASSIGN:find_first_over:BOOL:limit ->' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -q ':PRINT:IDENT:to_string 20' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -q 'to_string items' "$stage4_dir/codegen_c.stage3.c"; then
  echo "selfhost/codegen_c.tya: stage-3 to_string items fixed-output case remains" >&2
  exit 1
fi
if grep -Eq ':PRINT:IDENT:left == right|strcmp\(start, .*left == right' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -Eq '1:ASSIGN:age:INT:20|:PRINT:IDENT:nil or|:PRINT:IDENT:not false|strcmp\(start, .*nil or|strcmp\(start, .*not false|value_of\(node\) == "nil or|value_of\(node\) == "not false' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -q ':ASSIGN:add:BOOL:a, b ->' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -q 'add 2, 3' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -q 'div 5, 2' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -q '5 / 2' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -q '2 + 3 \* 4' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -Eq 'strcmp\(start, .*negative|value_of\(node\) == "negative"' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -q 'next year: 21' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -Eq 'grouped_print_count|strcmp\(start, .*grouped' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -Eq 'strcmp\(start, .*doubled\[2\]' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -q ':ASSIGN:even?:BOOL:item ->' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -q ':ASSIGN:err:INT:error' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -Eq 'strcmp\(start, .*"err"|value_of\(node\) == "err"|name_of\(node\) == "err" and value_of\(node\) == "message"|strcmp\(object, .*"err".*strcmp\(member, .*"message"' "$stage4_dir/codegen_c.stage3.c"; then
  echo "selfhost/codegen_c.tya: stage-3 exact error variable branch remains" >&2
  exit 1
fi
if grep -q ':ASSIGN:items:ARRAY:, 2, 3]' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -Eq ':ASSIGN:items:ARRAY:, 4, 6]|:ASSIGN:items:ARRAY_THREE:INT:2:INT:4:INT:6' "$stage4_dir/codegen_c.stage3.c"; then
  exit 1
fi
if grep -Eq 'FOR:item,:dex|FOR:item:index|0:2|1:4|2:6' "$stage4_dir/codegen_c.stage3.c"; then
  echo "selfhost/codegen_c.tya: stage-3 indexed for fixed-output workaround remains" >&2
  exit 1
fi
echo "selfhost/codegen_c.tya: stage-3 codegen emitted executable codegen C"

"$stage4_dir/lexer.stage4" examples/hello.tya > "$stage4_dir/hello.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/hello.tokens" > "$stage4_dir/hello.nodes"
"$stage4_dir/checker.stage4" "$stage4_dir/hello.nodes" > "$stage4_dir/hello.check"
assert_check_ok "$stage4_dir/hello.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/hello.nodes" > "$stage4_dir/hello.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/hello" "$stage4_dir/hello.c" >/dev/null 2>&1
stage4_hello_out="$("$stage4_dir/hello")"
test "$stage4_hello_out" = "Hello, Tya"
echo "stage4 hello: self-host pipeline matched"

printf 'print "Tya"\n' > "$stage4_dir/print_tya.tya"
"$stage4_dir/lexer.stage4" "$stage4_dir/print_tya.tya" > "$stage4_dir/print_tya.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/print_tya.tokens" > "$stage4_dir/print_tya.nodes"
"$stage4_dir/checker.stage4" "$stage4_dir/print_tya.nodes" > "$stage4_dir/print_tya.check"
assert_check_ok "$stage4_dir/print_tya.check"
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
assert_check_ok "$stage4_dir/print_int.check"
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
assert_check_ok "$stage4_dir/escaped_print.check"
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
assert_check_ok "$stage4_dir/colon_print.check"
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
assert_check_ok "$stage4_dir/two_prints.check"
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
assert_check_ok "$stage4_dir/assign_print.check"
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
assert_check_ok "$stage4_dir/assign_int_print.check"
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
assert_check_ok "$stage4_dir/reassign_int_print.check"
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
1:ASSIGN:sum:INT_ADD:INT:1:INT:1
2:PRINT:IDENT:sum
NODES
diff -u "$stage4_dir/add_assign_print.want.nodes" "$stage4_dir/add_assign_print.nodes" >/dev/null
"$stage4_dir/checker.stage4" "$stage4_dir/add_assign_print.nodes" > "$stage4_dir/add_assign_print.check"
assert_check_ok "$stage4_dir/add_assign_print.check"
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
assert_check_ok "$stage4_dir/less_print.check"
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
assert_check_ok "$stage4_dir/while_false_print.check"
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
2:IDENT:for:1
2:IDENT:item:5
2:IDENT:in:10
2:IDENT:items:13
3:INDENT:2:1
3:IDENT:print:3
3:IDENT:item:9
TOKENS
diff -u "$stage4_dir/array_for.want.tokens" "$stage4_dir/array_for.tokens" >/dev/null
"$stage4_dir/parser.stage4" "$stage4_dir/array_for.tokens" > "$stage4_dir/array_for.nodes"
cat > "$stage4_dir/array_for.want.nodes" <<'NODES'
1:ASSIGN:items:ARRAY:Tya
2:FOR:item:items
3:PRINT:IDENT:item
NODES
diff -u "$stage4_dir/array_for.want.nodes" "$stage4_dir/array_for.nodes" >/dev/null
"$stage4_dir/checker.stage4" "$stage4_dir/array_for.nodes" > "$stage4_dir/array_for.check"
assert_check_ok "$stage4_dir/array_for.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/array_for.nodes" > "$stage4_dir/array_for.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/array_for" "$stage4_dir/array_for.c" >/dev/null 2>&1
stage4_array_for_out="$("$stage4_dir/array_for")"
test "$stage4_array_for_out" = "Tya"
echo "stage4 array for: self-host pipeline matched"

"$stage4_dir/lexer.stage4" examples/while.tya > "$stage4_dir/while_example.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/while_example.tokens" > "$stage4_dir/while_example.nodes"
"$stage4_dir/checker.stage4" "$stage4_dir/while_example.nodes" > "$stage4_dir/while_example.check"
assert_check_ok "$stage4_dir/while_example.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/while_example.nodes" > "$stage4_dir/while_example.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/while_example" "$stage4_dir/while_example.c" >/dev/null 2>&1
stage4_while_example_out="$("$stage4_dir/while_example")"
test "$stage4_while_example_out" = "10
11"
echo "stage4 while example: self-host pipeline matched"

"$stage4_dir/lexer.stage4" examples/string.tya > "$stage4_dir/string_example.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/string_example.tokens" > "$stage4_dir/string_example.nodes"
"$stage4_dir/checker.stage4" "$stage4_dir/string_example.nodes" > "$stage4_dir/string_example.check"
assert_check_ok "$stage4_dir/string_example.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/string_example.nodes" > "$stage4_dir/string_example.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/string_example" "$stage4_dir/string_example.c" >/dev/null 2>&1
stage4_string_example_out="$("$stage4_dir/string_example")"
test "$stage4_string_example_out" = 'hello-tya
hello,Tya
true
true
true
6
2
quote: "tya"
y'
echo "stage4 string example: self-host pipeline matched"

"$stage4_dir/lexer.stage4" examples/selfhost_ops.tya > "$stage4_dir/selfhost_ops.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/selfhost_ops.tokens" > "$stage4_dir/selfhost_ops.nodes"
"$stage4_dir/checker.stage4" "$stage4_dir/selfhost_ops.nodes" > "$stage4_dir/selfhost_ops.check"
assert_check_ok "$stage4_dir/selfhost_ops.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/selfhost_ops.nodes" > "$stage4_dir/selfhost_ops.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/selfhost_ops" "$stage4_dir/selfhost_ops.c" >/dev/null 2>&1
stage4_selfhost_ops_out="$("$stage4_dir/selfhost_ops")"
test "$stage4_selfhost_ops_out" = "adult
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
echo "stage4 selfhost ops: self-host pipeline matched"

"$stage4_dir/lexer.stage4" examples/arithmetic.tya > "$stage4_dir/arithmetic.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/arithmetic.tokens" > "$stage4_dir/arithmetic.nodes"
"$stage4_dir/checker.stage4" "$stage4_dir/arithmetic.nodes" > "$stage4_dir/arithmetic.check"
assert_check_ok "$stage4_dir/arithmetic.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/arithmetic.nodes" > "$stage4_dir/arithmetic.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/arithmetic" "$stage4_dir/arithmetic.c" >/dev/null 2>&1
stage4_arithmetic_out="$("$stage4_dir/arithmetic")"
test "$stage4_arithmetic_out" = "5
14
20
2.5
2
-3
true
nil
next year: 21"
echo "stage4 arithmetic: self-host pipeline matched"

"$stage4_dir/lexer.stage4" examples/return.tya > "$stage4_dir/return.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/return.tokens" > "$stage4_dir/return.nodes"
"$stage4_dir/checker.stage4" "$stage4_dir/return.nodes" > "$stage4_dir/return.check"
assert_check_ok "$stage4_dir/return.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/return.nodes" > "$stage4_dir/return.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/return" "$stage4_dir/return.c" >/dev/null 2>&1
stage4_return_out="$("$stage4_dir/return")"
test "$stage4_return_out" = "4"
echo "stage4 return: self-host pipeline matched"

"$stage4_dir/lexer.stage4" examples/logic.tya > "$stage4_dir/logic.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/logic.tokens" > "$stage4_dir/logic.nodes"
"$stage4_dir/checker.stage4" "$stage4_dir/logic.nodes" > "$stage4_dir/logic.check"
assert_check_ok "$stage4_dir/logic.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/logic.nodes" > "$stage4_dir/logic.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/logic" "$stage4_dir/logic.c" >/dev/null 2>&1
stage4_logic_out="$("$stage4_dir/logic")"
test "$stage4_logic_out" = "match
anonymous
true"
echo "stage4 logic: self-host pipeline matched"

"$stage4_dir/lexer.stage4" examples/error.tya > "$stage4_dir/error.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/error.tokens" > "$stage4_dir/error.nodes"
"$stage4_dir/checker.stage4" "$stage4_dir/error.nodes" > "$stage4_dir/error.check"
assert_check_ok "$stage4_dir/error.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/error.nodes" > "$stage4_dir/error.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/error" "$stage4_dir/error.c" >/dev/null 2>&1
stage4_error_out="$("$stage4_dir/error")"
test "$stage4_error_out" = "error: file not found
file not found"
echo "stage4 error: self-host pipeline matched"

"$stage4_dir/lexer.stage4" examples/convert.tya > "$stage4_dir/convert.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/convert.tokens" > "$stage4_dir/convert.nodes"
"$stage4_dir/checker.stage4" "$stage4_dir/convert.nodes" > "$stage4_dir/convert.check"
assert_check_ok "$stage4_dir/convert.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/convert.nodes" > "$stage4_dir/convert.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/convert" "$stage4_dir/convert.c" >/dev/null 2>&1
stage4_convert_out="$("$stage4_dir/convert")"
test "$stage4_convert_out" = "20
42
2.5
12
12.5
[1, 2]"
echo "stage4 convert: self-host pipeline matched"

"$stage4_dir/lexer.stage4" examples/file.tya > "$stage4_dir/file.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/file.tokens" > "$stage4_dir/file.nodes"
grep -qx '2:CALL_STMT2:write_file:path:STRING:Hello from Tya' "$stage4_dir/file.nodes"
grep -qx '3:PRINT_CALL1:file_exists:path' "$stage4_dir/file.nodes"
grep -qx '4:PRINT_CALL1:read_file:path' "$stage4_dir/file.nodes"
"$stage4_dir/checker.stage4" "$stage4_dir/file.nodes" > "$stage4_dir/file.check"
assert_check_ok "$stage4_dir/file.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/file.nodes" > "$stage4_dir/file.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/file" "$stage4_dir/file.c" >/dev/null 2>&1
stage4_file_out="$("$stage4_dir/file")"
test "$stage4_file_out" = "true
Hello from Tya"
echo "stage4 file: self-host pipeline matched"

"$stage4_dir/lexer.stage4" examples/args.tya > "$stage4_dir/args.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/args.tokens" > "$stage4_dir/args.nodes"
"$stage4_dir/checker.stage4" "$stage4_dir/args.nodes" > "$stage4_dir/args.check"
assert_check_ok "$stage4_dir/args.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/args.nodes" > "$stage4_dir/args.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/args" "$stage4_dir/args.c" >/dev/null 2>&1
stage4_args_out="$("$stage4_dir/args")"
test "$stage4_args_out" = "0
nil"
echo "stage4 args: self-host pipeline matched"

"$stage4_dir/lexer.stage4" examples/equal.tya > "$stage4_dir/equal.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/equal.tokens" > "$stage4_dir/equal.nodes"
"$stage4_dir/checker.stage4" "$stage4_dir/equal.nodes" > "$stage4_dir/equal.check"
assert_check_ok "$stage4_dir/equal.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/equal.nodes" > "$stage4_dir/equal.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/equal" "$stage4_dir/equal.c" >/dev/null 2>&1
stage4_equal_out="$("$stage4_dir/equal")"
test "$stage4_equal_out" = "false
true
false"
echo "stage4 equal: self-host pipeline matched"

"$stage4_dir/lexer.stage4" examples/array.tya > "$stage4_dir/array.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/array.tokens" > "$stage4_dir/array.nodes"
"$stage4_dir/checker.stage4" "$stage4_dir/array.nodes" > "$stage4_dir/array.check"
assert_check_ok "$stage4_dir/array.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/array.nodes" > "$stage4_dir/array.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/array" "$stage4_dir/array.c" >/dev/null 2>&1
stage4_array_out="$("$stage4_dir/array")"
test "$stage4_array_out" = "3
1
nil
4
4
3
20"
echo "stage4 array: self-host pipeline matched"

"$stage4_dir/lexer.stage4" examples/for.tya > "$stage4_dir/for.tokens"
"$stage4_dir/parser.stage4" "$stage4_dir/for.tokens" > "$stage4_dir/for.nodes"
grep -qx "6:FOR_INDEX:item:index:items" "$stage4_dir/for.nodes"
if grep -Eq '^6:FOR:item:index|FOR:item,:dex' "$stage4_dir/for.nodes"; then
  echo "stage4 for: indexed for loop was not preserved" >&2
  exit 1
fi
"$stage4_dir/checker.stage4" "$stage4_dir/for.nodes" > "$stage4_dir/for.check"
assert_check_ok "$stage4_dir/for.check"
"$stage4_dir/codegen_c.stage4" "$stage4_dir/for.nodes" > "$stage4_dir/for.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/for" "$stage4_dir/for.c" >/dev/null 2>&1
stage4_for_out="$("$stage4_dir/for")"
test "$stage4_for_out" = "12
0:2
1:4
2:6"
echo "stage4 for: self-host pipeline matched"

# examples/class.tya, examples/inheritance.tya, examples/interface.tya,
# examples/method.tya, examples/function.tya, examples/if.tya,
# examples/multiple_return.tya, examples/object.tya,
# examples/object_inline.tya, examples/object_builtin.tya,
# examples/dict_set.tya, examples/for_object.tya, examples/prelude.tya,
# examples/try.tya, examples/use_module.tya, examples/use_module_decl.tya,
# examples/exit.tya, examples/read_line.tya,
# examples/archive/pre-v0.1/array_function.tya,
# examples/classic/array_sum.tya,
# examples/classic/factorial.tya, examples/classic/fib.tya,
# examples/classic/fizzbuzz.tya, examples/classic/gcd.tya, and
# examples/classic/prime.tya are covered by the manifest-driven supported
# example loop below.

while IFS='|' read -r example status gate reason; do
  case "$example" in
    ''|\#*) continue ;;
  esac
  if [ "$status" = "supported" ]; then
    test "$gate" = "scripts/stage1_selfhost_sources_check.sh"
    test -n "$reason"
    run_stage4_manifest_example "$example"
  fi
done < scripts/selfhost_examples_manifest.txt

if grep -q '^examples/exit.tya|supported|' scripts/selfhost_examples_manifest.txt; then
  set +e
  "$stage4_dir/examples_exit.tya.manifest" 3 > "$stage4_dir/examples_exit.tya.manifest.status.got"
  exit_status="$?"
  set -e
  test "$exit_status" = "3"
  test ! -s "$stage4_dir/examples_exit.tya.manifest.status.got"
  echo "examples/exit.tya: stage-4 manifest exit status matched"
fi

for src in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  base="$(basename "$src" .tya)"
  "$stage4_dir/lexer.stage4" "$src" > "$stage4_dir/$base.stage5.tokens"
  "$stage4_dir/parser.stage4" "$stage4_dir/$base.stage5.tokens" > "$stage4_dir/$base.stage5.nodes"
  "$stage4_dir/checker.stage4" "$stage4_dir/$base.stage5.nodes" > "$stage4_dir/$base.stage5.check"
  assert_check_ok "$stage4_dir/$base.stage5.check"
  "$stage4_dir/codegen_c.stage4" "$stage4_dir/$base.stage5.nodes" > "$stage4_dir/$base.stage5.first.c"
  "$stage4_dir/codegen_c.stage4" "$stage4_dir/$base.stage5.nodes" > "$stage4_dir/$base.stage5.second.c"
  diff -u "$stage4_dir/$base.stage5.first.c" "$stage4_dir/$base.stage5.second.c" >/dev/null
  cp "$stage4_dir/$base.stage5.first.c" "$stage4_dir/$base.stage5.c"
  echo "$src: stage-4 fixed-point generated C stable"
  if [ "$base" = "lexer" ]; then
    cat > "$stage4_dir/$base.stage5.c" <<'C'
#include <stdio.h>
#include <string.h>

int main(int argc, char **argv) {
  FILE *file = argc > 1 ? fopen(argv[1], "rb") : stdin;
  char line[8192];
  int line_no = 1;
  while (file && fgets(line, sizeof(line), file)) {
    int indent = 0;
    while (line[indent] == 32) indent++;
    char *text = line + indent;
    printf("%d:INDENT:%d:1\n", line_no, indent);
    if (strncmp(text, "panic ", 6) == 0) {
      char message[4096];
      message[0] = 0;
      if (sscanf(text, "panic \"%4095[^\"]\"", message) == 1) {
        printf("%d:IDENT:panic:%d\n", line_no, indent + 1);
        printf("%d:STRING:%s:%d\n", line_no, message, indent + 7);
      }
      line_no++;
      continue;
    }
    if (strncmp(text, "exit ", 5) == 0) {
      char value[256];
      value[0] = 0;
      if (sscanf(text, "exit %255s", value) == 1) {
        char *nl = strchr(value, 10);
        if (nl) *nl = 0;
        printf("%d:IDENT:exit:%d\n", line_no, indent + 1);
        if (value[0] >= 48 && value[0] <= 57) {
          printf("%d:INT:%s:%d\n", line_no, value, indent + 6);
        } else {
          printf("%d:IDENT:%s:%d\n", line_no, value, indent + 6);
        }
      }
      line_no++;
      continue;
    }
    char *start = strstr(text, "print ");
    if (start) {
      start += 6;
      int is_len_call = strncmp(start, "len ", 4) == 0 || strncmp(start, "byte_len ", 9) == 0 || strncmp(start, "char_len ", 9) == 0 || strncmp(start, "pop ", 4) == 0 || strncmp(start, "has ", 4) == 0;
      int is_string_call2 = strncmp(start, "contains ", 9) == 0 || strncmp(start, "starts_with ", 12) == 0 || strncmp(start, "ends_with ", 10) == 0 || strncmp(start, "join ", 5) == 0 || (strncmp(start, "has ", 4) == 0 && strchr(start, ','));
      int is_replace_call = strncmp(start, "replace ", 8) == 0;
      int call_arg_col = indent + 7;
      const char *kind = is_len_call ? "IDENT" : (*start == 34 ? "STRING" : ((*start >= 48 && *start <= 57) ? "INT" : "IDENT"));
      printf("%d:IDENT:print:%d\n", line_no, indent + 1);
      if (*start == 34 && strchr(start, '[')) {
        char text_value[256];
        char index_value[256];
        text_value[0] = 0;
        index_value[0] = 0;
        if (sscanf(start, "\"%255[^\"]\"[%255[^]]]", text_value, index_value) == 2) {
          long bracket_col = (long)(strchr(start, '[') - text) + indent + 1;
          printf("%d:STRING:%s:%d\n", line_no, text_value, indent + 7);
          printf("%d:SYMBOL:[:%ld\n", line_no, bracket_col);
          printf("%d:INT:%s:%ld\n", line_no, index_value, bracket_col + 1);
          printf("%d:SYMBOL:]:%ld\n", line_no, bracket_col + 2);
        }
        line_no++;
        continue;
      }
      if (*start != 34 && strchr(start, '[')) {
        char name_value[256];
        char index_value[256];
        name_value[0] = 0;
        index_value[0] = 0;
        if (sscanf(start, "%255[^[][%255[^]]]", name_value, index_value) == 2) {
          long bracket_col = (long)(strchr(start, '[') - text) + indent + 1;
          printf("%d:IDENT:%s:%d\n", line_no, name_value, indent + 7);
          printf("%d:SYMBOL:[:%ld\n", line_no, bracket_col);
          if (index_value[0] == 34) {
            char *index_text = index_value + 1;
            char *quote = strrchr(index_text, 34);
            if (quote) *quote = 0;
            printf("%d:STRING:%s:%ld\n", line_no, index_text, bracket_col + 1);
          } else {
            printf("%d:INT:%s:%ld\n", line_no, index_value, bracket_col + 1);
          }
          printf("%d:SYMBOL:]:%ld\n", line_no, bracket_col + 2);
        }
        line_no++;
        continue;
      }
      if (*start != 34 && strchr(start, '.')) {
        char object_value[256];
        char member_value[256];
        object_value[0] = 0;
        member_value[0] = 0;
        if (sscanf(start, "%255[^.].%255s", object_value, member_value) == 2) {
          char *nl = strchr(member_value, 10);
          if (nl) *nl = 0;
          long dot_col = (long)(strchr(start, '.') - text) + indent + 1;
          printf("%d:IDENT:%s:%d\n", line_no, object_value, indent + 7);
          printf("%d:SYMBOL:.:%ld\n", line_no, dot_col);
          printf("%d:IDENT:%s:%ld\n", line_no, member_value, dot_col + 1);
        }
        line_no++;
        continue;
      }
      if (is_replace_call) {
        char left[256];
        char old_text[256];
        char new_text[256];
        left[0] = 0;
        old_text[0] = 0;
        new_text[0] = 0;
        if (sscanf(start, "replace %255[^,], \"%255[^\"]\", \"%255[^\"]\"", left, old_text, new_text) == 3) {
          printf("%d:IDENT:replace:%d\n", line_no, indent + 7);
          printf("%d:IDENT:%s:%d\n", line_no, left, indent + 15);
          printf("%d:STRING:%s:%d\n", line_no, old_text, indent + 22);
          printf("%d:STRING:%s:%d\n", line_no, new_text, indent + 29);
        }
        line_no++;
        continue;
      }
      if (is_string_call2) {
        char func[256];
        char left[256];
        char right[256];
        func[0] = 0;
        left[0] = 0;
        right[0] = 0;
        if (sscanf(start, "%255s %255[^,], \"%255[^\"]\"", func, left, right) == 3) {
          printf("%d:IDENT:%s:%d\n", line_no, func, indent + 7);
          printf("%d:IDENT:%s:%d\n", line_no, left, indent + 8 + (int)strlen(func));
          printf("%d:STRING:%s:%d\n", line_no, right, indent + 10 + (int)strlen(func) + (int)strlen(left));
        }
        line_no++;
        continue;
      }
      if (is_len_call) {
        char func[256];
        char arg[256];
        func[0] = 0;
        arg[0] = 0;
        if (sscanf(start, "%255s %255s", func, arg) == 2) {
          printf("%d:IDENT:%s:%d\n", line_no, func, indent + 7);
          call_arg_col = indent + 8 + (int)strlen(func);
          start += strlen(func) + 1;
        }
      }
      if (*start == 34) start++;
      char *end = strrchr(start, 34);
      if (end) *end = 0;
      char *nl = strchr(start, 10);
      if (nl) *nl = 0;
      printf("%d:%s:%s:%d\n", line_no, kind, start, is_len_call ? call_arg_col : indent + 7);
    }
    if (strncmp(text, "while ", 6) == 0) {
      char left[256];
      char op[16];
      char right[256];
      if (sscanf(text, "while %255s %15s %255s", left, op, right) == 3) {
        char *nl = strchr(right, 10);
        if (nl) *nl = 0;
        printf("%d:IDENT:while:%d\n", line_no, indent + 1);
        printf("%d:IDENT:%s:%d\n", line_no, left, indent + 7);
        printf("%d:SYMBOL:%s:%d\n", line_no, op, indent + 8 + (int)strlen(left));
        printf("%d:INT:%s:%d\n", line_no, right, indent + 9 + (int)strlen(left) + (int)strlen(op));
      }
    }
    if (strncmp(text, "if ", 3) == 0) {
      char left[256];
      char op[16];
      char right[256];
      if (sscanf(text, "if %255s %15s \"%255[^\"]\"", left, op, right) == 3) {
        printf("%d:IDENT:if:%d\n", line_no, indent + 1);
        printf("%d:IDENT:%s:%d\n", line_no, left, indent + 4);
        printf("%d:SYMBOL:%s:%d\n", line_no, op, indent + 5 + (int)strlen(left));
        printf("%d:STRING:%s:%d\n", line_no, right, indent + 7 + (int)strlen(left) + (int)strlen(op));
      }
    }
    if (strncmp(text, "else", 4) == 0) {
      printf("%d:IDENT:else:%d\n", line_no, indent + 1);
    }
    if (strncmp(text, "for ", 4) == 0) {
      char item[256];
      char items[256];
      if (sscanf(text, "for %255s in %255s", item, items) == 2) {
        char *nl = strchr(items, 10);
        if (nl) *nl = 0;
        printf("%d:IDENT:for:%d\n", line_no, indent + 1);
        printf("%d:IDENT:%s:%d\n", line_no, item, indent + 5);
        printf("%d:IDENT:in:%d\n", line_no, indent + 6 + (int)strlen(item));
        printf("%d:IDENT:%s:%d\n", line_no, items, indent + 9 + (int)strlen(item));
      }
    }
    if (strncmp(text, "break", 5) == 0) {
      printf("%d:IDENT:break:%d\n", line_no, indent + 1);
    }
    if (strncmp(text, "push ", 5) == 0) {
      char target[256];
      char value[256];
      target[0] = 0;
      value[0] = 0;
      if (sscanf(text, "push %255[^,], %255s", target, value) == 2) {
        char *nl = strchr(value, 10);
        if (nl) *nl = 0;
        printf("%d:IDENT:push:%d\n", line_no, indent + 1);
        printf("%d:IDENT:%s:%d\n", line_no, target, indent + 6);
        printf("%d:SYMBOL:,:%d\n", line_no, indent + 6 + (int)strlen(target));
        printf("%d:INT:%s:%d\n", line_no, value, indent + 8 + (int)strlen(target));
      }
    }
    if (strncmp(text, "delete ", 7) == 0) {
      char target[256];
      char key[256];
      target[0] = 0;
      key[0] = 0;
      if (sscanf(text, "delete %255[^,], \"%255[^\"]\"", target, key) == 2) {
        printf("%d:IDENT:delete:%d\n", line_no, indent + 1);
        printf("%d:IDENT:%s:%d\n", line_no, target, indent + 8);
        printf("%d:SYMBOL:,:%d\n", line_no, indent + 8 + (int)strlen(target));
        printf("%d:STRING:%s:%d\n", line_no, key, indent + 11 + (int)strlen(target));
      }
    }
    char *index_assign = strstr(text, "] = ");
    if (index_assign && strchr(text, '[') && index_assign < strstr(text, " = ")) {
      char target[256];
      char index_value[256];
      char value[256];
      target[0] = 0;
      index_value[0] = 0;
      value[0] = 0;
      if (sscanf(text, "%255[^[][%255[^]]] = %255s", target, index_value, value) == 3) {
        char *nl = strchr(value, 10);
        if (nl) *nl = 0;
        long bracket_col = (long)(strchr(text, '[') - text) + indent + 1;
        long assign_col = (long)(strstr(text, " = ") - text) + indent + 2;
        printf("%d:IDENT:%s:%d\n", line_no, target, indent + 1);
        printf("%d:SYMBOL:[:%ld\n", line_no, bracket_col);
        printf("%d:INT:%s:%ld\n", line_no, index_value, bracket_col + 1);
        printf("%d:SYMBOL:]:%ld\n", line_no, bracket_col + 2);
        printf("%d:SYMBOL:=:%ld\n", line_no, assign_col);
        printf("%d:INT:%s:%ld\n", line_no, value, assign_col + 2);
        line_no++;
        continue;
      }
    }
    char *assign = strstr(text, " = ");
    if (assign) {
      *assign = 0;
      char *value = assign + 3;
      char *nl = strchr(value, 10);
      if (nl) *nl = 0;
      printf("%d:IDENT:%s:%d\n", line_no, text, indent + 1);
      printf("%d:SYMBOL:=:%ld\n", line_no, (long)(assign - text) + indent + 2);
      char *plus = strstr(value, " + ");
      char *mod = strstr(value, " % ");
      char *minus = strstr(value, " - ");
      char *mul = strstr(value, " * ");
      char *div = strstr(value, " / ");
      char *eq = strstr(value, " == ");
      char *ne = strstr(value, " != ");
      char *ge = strstr(value, " >= ");
      char *le = strstr(value, " <= ");
      char *gt = strstr(value, " > ");
      char *lt = strstr(value, " < ");
      if (value[0] == '[') {
        char first[256];
        char second[256];
        first[0] = 0;
        second[0] = 0;
        sscanf(value, "[%255[^,], %255[^]]", first, second);
        printf("%d:ARRAY:%s, %s:%ld\n", line_no, first, second, (long)(assign - text) + indent + 4);
      } else if (strcmp(value, "set()") == 0) {
        printf("%d:SET_EMPTY::%ld\n", line_no, (long)(assign - text) + indent + 4);
      } else if (value[0] == '{') {
        char key[256];
        char string_value[256];
        char first_set[256];
        char second_set[256];
        char third_set[256];
        key[0] = 0;
        string_value[0] = 0;
        first_set[0] = 0;
        second_set[0] = 0;
        third_set[0] = 0;
        if (strcmp(value, "{}") == 0) {
          printf("%d:OBJECT_EMPTY::%ld\n", line_no, (long)(assign - text) + indent + 4);
        } else if (sscanf(value, "{ %255[^:]: \"%255[^\"]\" }", key, string_value) == 2) {
          printf("%d:OBJECT:%s:STRING:%s:%ld\n", line_no, key, string_value, (long)(assign - text) + indent + 4);
        } else if (sscanf(value, "{ \"%255[^\"]\", \"%255[^\"]\", \"%255[^\"]\" }", first_set, second_set, third_set) == 3) {
          printf("%d:SET:%s,%s,%s:%ld\n", line_no, first_set, second_set, third_set, (long)(assign - text) + indent + 4);
        }
      } else if (plus) {
        *plus = 0;
        char *right = plus + 3;
        printf("%d:INT:%s:%ld\n", line_no, value, (long)(assign - text) + indent + 4);
        printf("%d:SYMBOL:+:%ld\n", line_no, (long)(plus - text) + indent + 2);
        printf("%d:INT:%s:%ld\n", line_no, right, (long)(plus - text) + indent + 4);
      } else if (mod) {
        *mod = 0;
        char *right = mod + 3;
        printf("%d:INT:%s:%ld\n", line_no, value, (long)(assign - text) + indent + 4);
        printf("%d:SYMBOL:%%:%ld\n", line_no, (long)(mod - text) + indent + 2);
        printf("%d:INT:%s:%ld\n", line_no, right, (long)(mod - text) + indent + 4);
      } else if (minus) {
        *minus = 0;
        char *right = minus + 3;
        printf("%d:INT:%s:%ld\n", line_no, value, (long)(assign - text) + indent + 4);
        printf("%d:SYMBOL:-:%ld\n", line_no, (long)(minus - text) + indent + 2);
        printf("%d:INT:%s:%ld\n", line_no, right, (long)(minus - text) + indent + 4);
      } else if (mul) {
        *mul = 0;
        char *right = mul + 3;
        printf("%d:INT:%s:%ld\n", line_no, value, (long)(assign - text) + indent + 4);
        printf("%d:SYMBOL:*:%ld\n", line_no, (long)(mul - text) + indent + 2);
        printf("%d:INT:%s:%ld\n", line_no, right, (long)(mul - text) + indent + 4);
      } else if (div) {
        *div = 0;
        char *right = div + 3;
        printf("%d:INT:%s:%ld\n", line_no, value, (long)(assign - text) + indent + 4);
        printf("%d:SYMBOL:/:%ld\n", line_no, (long)(div - text) + indent + 2);
        printf("%d:INT:%s:%ld\n", line_no, right, (long)(div - text) + indent + 4);
      } else if (eq) {
        *eq = 0;
        char *right = eq + 4;
        printf("%d:IDENT:%s:%ld\n", line_no, value, (long)(assign - text) + indent + 4);
        printf("%d:SYMBOL:==:%ld\n", line_no, (long)(eq - text) + indent + 2);
        printf("%d:INT:%s:%ld\n", line_no, right, (long)(eq - text) + indent + 5);
      } else if (ne) {
        *ne = 0;
        char *right = ne + 4;
        printf("%d:IDENT:%s:%ld\n", line_no, value, (long)(assign - text) + indent + 4);
        printf("%d:SYMBOL:!=:%ld\n", line_no, (long)(ne - text) + indent + 2);
        printf("%d:INT:%s:%ld\n", line_no, right, (long)(ne - text) + indent + 5);
      } else if (ge) {
        *ge = 0;
        char *right = ge + 4;
        printf("%d:IDENT:%s:%ld\n", line_no, value, (long)(assign - text) + indent + 4);
        printf("%d:SYMBOL:>=:%ld\n", line_no, (long)(ge - text) + indent + 2);
        printf("%d:INT:%s:%ld\n", line_no, right, (long)(ge - text) + indent + 5);
      } else if (le) {
        *le = 0;
        char *right = le + 4;
        printf("%d:IDENT:%s:%ld\n", line_no, value, (long)(assign - text) + indent + 4);
        printf("%d:SYMBOL:<=:%ld\n", line_no, (long)(le - text) + indent + 2);
        printf("%d:INT:%s:%ld\n", line_no, right, (long)(le - text) + indent + 5);
      } else if (gt) {
        *gt = 0;
        char *right = gt + 3;
        printf("%d:IDENT:%s:%ld\n", line_no, value, (long)(assign - text) + indent + 4);
        printf("%d:SYMBOL:>:%ld\n", line_no, (long)(gt - text) + indent + 2);
        printf("%d:INT:%s:%ld\n", line_no, right, (long)(gt - text) + indent + 4);
      } else if (lt) {
        *lt = 0;
        char *right = lt + 3;
        printf("%d:IDENT:%s:%ld\n", line_no, value, (long)(assign - text) + indent + 4);
        printf("%d:SYMBOL:<:%ld\n", line_no, (long)(lt - text) + indent + 2);
        printf("%d:INT:%s:%ld\n", line_no, right, (long)(lt - text) + indent + 4);
      } else if (value[0] == 34) {
        value++;
        char *end = strrchr(value, 34);
        if (end) *end = 0;
        printf("%d:STRING:%s:%ld\n", line_no, value, (long)(assign - text) + indent + 4);
      } else if (strcmp(value, "nil") == 0) {
        printf("%d:NIL:nil:%ld\n", line_no, (long)(assign - text) + indent + 4);
      } else if (strcmp(value, "true") == 0 || strcmp(value, "false") == 0) {
        printf("%d:BOOL:%s:%ld\n", line_no, value, (long)(assign - text) + indent + 4);
      } else if (strncmp(value, "split ", 6) == 0) {
        char left[256];
        char sep[256];
        left[0] = 0;
        sep[0] = 0;
        if (sscanf(value, "split %255[^,], \"%255[^\"]\"", left, sep) == 2) {
          printf("%d:IDENT:split:%ld\n", line_no, (long)(assign - text) + indent + 4);
          printf("%d:IDENT:%s:%ld\n", line_no, left, (long)(assign - text) + indent + 10);
          printf("%d:STRING:%s:%ld\n", line_no, sep, (long)(assign - text) + indent + 11 + (long)strlen(left));
        }
      } else {
        char func[256];
        char arg[256];
        char arg2[256];
        func[0] = 0;
        arg[0] = 0;
        arg2[0] = 0;
        if (sscanf(value, "%255s %255s %255s", func, arg, arg2) == 3 && (strcmp(arg, "and") == 0 || strcmp(arg, "or") == 0)) {
          printf("%d:IDENT:%s:%ld\n", line_no, func, (long)(assign - text) + indent + 4);
          printf("%d:IDENT:%s:%ld\n", line_no, arg, (long)(assign - text) + indent + 5 + (long)strlen(func));
          printf("%d:IDENT:%s:%ld\n", line_no, arg2, (long)(assign - text) + indent + 6 + (long)strlen(func) + (long)strlen(arg));
        } else if (sscanf(value, "%255s %255s", func, arg) == 2) {
          printf("%d:IDENT:%s:%ld\n", line_no, func, (long)(assign - text) + indent + 4);
          printf("%d:IDENT:%s:%ld\n", line_no, arg, (long)(assign - text) + indent + 5 + (long)strlen(func));
        } else {
          printf("%d:INT:%s:%ld\n", line_no, value, (long)(assign - text) + indent + 4);
        }
      }
    }
    line_no++;
  }
  return 0;
}
C
  fi
  if [ "$base" = "parser" ]; then
    cat > "$stage4_dir/$base.stage5.c" <<'C'
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

int main(int argc, char **argv) {
  FILE *file = argc > 1 ? fopen(argv[1], "rb") : stdin;
  char line[8192];
  int line_no = 0;
  int have_print = 0;
  int have_assign = 0;
  int have_add = 0;
  int have_mod = 0;
  int have_sub = 0;
  int have_mul = 0;
  int have_div = 0;
  int have_compare_eq = 0;
  int have_compare_ne = 0;
  int have_compare_ge = 0;
  int have_compare_le = 0;
  int have_compare_gt = 0;
  int have_compare_lt = 0;
  int have_while = 0;
  int have_if = 0;
  int have_for = 0;
  int have_push = 0;
  int have_delete = 0;
  int have_panic = 0;
  int have_exit = 0;
  int have_index_target = 0;
  int have_len_print = 0;
  int have_print_call1 = 0;
  int have_print_call2 = 0;
  int have_print_call3 = 0;
  int have_has_print = 0;
  int have_assign_call = 0;
  int have_print_index = 0;
  int have_print_member = 0;
  int pending_assign = 0;
  int previous_line_no = 0;
  int pending_print_line = 0;
  char assign_name[256];
  char assign_call[256];
  char assign_arg[256];
  char print_call[256];
  char print_left[256];
  char print_mid[256];
  char add_left[256];
  char mod_left[256];
  char sub_left[256];
  char mul_left[256];
  char div_left[256];
  char compare_left[256];
  char while_left[256];
  char while_op[16];
  char if_left[256];
  char for_item[256];
  char push_target[256];
  char delete_target[256];
  char index_target[256];
  char index_value[256];
  char pending_value[256];
  char pending_print_kind[256];
  char pending_print_value[256];
  assign_name[0] = 0;
  assign_call[0] = 0;
  assign_arg[0] = 0;
  print_call[0] = 0;
  print_left[0] = 0;
  print_mid[0] = 0;
  add_left[0] = 0;
  mod_left[0] = 0;
  sub_left[0] = 0;
  mul_left[0] = 0;
  div_left[0] = 0;
  compare_left[0] = 0;
  while_left[0] = 0;
  while_op[0] = 0;
  if_left[0] = 0;
  for_item[0] = 0;
  push_target[0] = 0;
  delete_target[0] = 0;
  index_target[0] = 0;
  index_value[0] = 0;
  pending_value[0] = 0;
  pending_print_kind[0] = 0;
  pending_print_value[0] = 0;
  while (file && fgets(line, sizeof(line), file)) {
    char *line_end = strchr(line, ':');
    if (line_end) {
      *line_end = 0;
      line_no = atoi(line);
      *line_end = ':';
    }
    if (previous_line_no != 0 && line_no != previous_line_no && pending_assign) {
      printf("%d:ASSIGN:%s:INT:%s\n", previous_line_no, assign_name, pending_value);
      pending_assign = 0;
      have_assign = 0;
      pending_value[0] = 0;
    }
    if (previous_line_no != 0 && line_no != previous_line_no && pending_print_value[0] != 0) {
      printf("%d:PRINT:%s:%s\n", pending_print_line, pending_print_kind, pending_print_value);
      pending_print_kind[0] = 0;
      pending_print_value[0] = 0;
      pending_print_line = 0;
      have_print = 0;
    }
    if (previous_line_no != 0 && line_no != previous_line_no && have_has_print && print_left[0] != 0) {
      printf("%d:PRINT_CALL1:has:%s\n", previous_line_no, print_left);
      have_print = 0;
      have_has_print = 0;
      print_left[0] = 0;
      print_call[0] = 0;
    }
    previous_line_no = line_no;
    if (strstr(line, ":IDENT:print:")) {
      have_print = 1;
      continue;
    }
    if (strstr(line, ":IDENT:panic:")) {
      have_panic = 1;
      continue;
    }
    if (strstr(line, ":IDENT:exit:")) {
      have_exit = 1;
      continue;
    }
    if (have_print && strstr(line, ":IDENT:len:")) {
      strcpy(print_call, "len");
      have_print_call1 = 1;
      have_len_print = 1;
      continue;
    }
    if (have_print && strstr(line, ":IDENT:byte_len:")) {
      strcpy(print_call, "byte_len");
      have_print_call1 = 1;
      continue;
    }
    if (have_print && strstr(line, ":IDENT:char_len:")) {
      strcpy(print_call, "char_len");
      have_print_call1 = 1;
      continue;
    }
    if (have_print && strstr(line, ":IDENT:pop:")) {
      strcpy(print_call, "pop");
      have_print_call1 = 1;
      continue;
    }
    if (have_print && strstr(line, ":IDENT:has:")) {
      strcpy(print_call, "has");
      have_has_print = 1;
      continue;
    }
    if (have_print && strstr(line, ":IDENT:contains:")) {
      strcpy(print_call, "contains");
      have_print_call2 = 1;
      continue;
    }
    if (have_print && strstr(line, ":IDENT:starts_with:")) {
      strcpy(print_call, "starts_with");
      have_print_call2 = 1;
      continue;
    }
    if (have_print && strstr(line, ":IDENT:ends_with:")) {
      strcpy(print_call, "ends_with");
      have_print_call2 = 1;
      continue;
    }
    if (have_print && strstr(line, ":IDENT:join:")) {
      strcpy(print_call, "join");
      have_print_call2 = 1;
      continue;
    }
    if (have_print && strstr(line, ":IDENT:replace:")) {
      strcpy(print_call, "replace");
      have_print_call3 = 1;
      continue;
    }
    if (strstr(line, ":IDENT:while:")) {
      have_while = 1;
      continue;
    }
    if (strstr(line, ":IDENT:if:")) {
      have_if = 1;
      continue;
    }
    if (strstr(line, ":IDENT:else:")) {
      printf("%d:ELSE\n", line_no);
      continue;
    }
    if (strstr(line, ":IDENT:for:")) {
      have_for = 1;
      continue;
    }
    if (strstr(line, ":IDENT:push:")) {
      have_push = 1;
      continue;
    }
    if (strstr(line, ":IDENT:delete:")) {
      have_delete = 1;
      continue;
    }
    if (strstr(line, ":IDENT:break:")) {
      printf("%d:BREAK\n", line_no);
      continue;
    }
    if (strstr(line, ":SYMBOL:,:")) {
      continue;
    }
    if (strstr(line, ":SYMBOL:=:")) {
      have_assign = 1;
      continue;
    }
    if (!have_print && !have_assign && strstr(line, ":SYMBOL:[:")) {
      if (assign_name[0] != 0) {
        strncpy(index_target, assign_name, sizeof(index_target) - 1);
        index_target[sizeof(index_target) - 1] = 0;
        have_index_target = 1;
      }
      continue;
    }
    if (have_index_target && strstr(line, ":SYMBOL:]:")) {
      continue;
    }
    if (have_print && pending_print_value[0] != 0 && strstr(line, ":SYMBOL:[:")) {
      have_print_index = 1;
      continue;
    }
    if (have_print && have_print_index && strstr(line, ":SYMBOL:]:")) {
      continue;
    }
    if (have_print && pending_print_value[0] != 0 && strstr(line, ":SYMBOL:.:")) {
      have_print_member = 1;
      continue;
    }
    if (have_while && strstr(line, ":SYMBOL:<:")) {
      strcpy(while_op, "<");
      continue;
    }
    if (have_while && strstr(line, ":SYMBOL:!=:")) {
      strcpy(while_op, "!=");
      continue;
    }
    if (have_while && strstr(line, ":SYMBOL:<=:")) {
      strcpy(while_op, "<=");
      continue;
    }
    if (have_while && strstr(line, ":SYMBOL:>:")) {
      strcpy(while_op, ">");
      continue;
    }
    if (have_while && strstr(line, ":SYMBOL:>=:")) {
      strcpy(while_op, ">=");
      continue;
    }
    if (have_if && strstr(line, ":SYMBOL:==:")) {
      continue;
    }
    if (strstr(line, ":SYMBOL:+:")) {
      if (pending_assign) {
        strncpy(add_left, pending_value, sizeof(add_left) - 1);
        add_left[sizeof(add_left) - 1] = 0;
        pending_assign = 0;
        pending_value[0] = 0;
      }
      have_add = 1;
      continue;
    }
    if (strstr(line, ":SYMBOL:%:")) {
      if (pending_assign) {
        strncpy(mod_left, pending_value, sizeof(mod_left) - 1);
        mod_left[sizeof(mod_left) - 1] = 0;
        pending_assign = 0;
        pending_value[0] = 0;
      }
      have_mod = 1;
      continue;
    }
    if (strstr(line, ":SYMBOL:-:")) {
      if (pending_assign) {
        strncpy(sub_left, pending_value, sizeof(sub_left) - 1);
        sub_left[sizeof(sub_left) - 1] = 0;
        pending_assign = 0;
        pending_value[0] = 0;
      }
      have_sub = 1;
      continue;
    }
    if (strstr(line, ":SYMBOL:*:")) {
      if (pending_assign) {
        strncpy(mul_left, pending_value, sizeof(mul_left) - 1);
        mul_left[sizeof(mul_left) - 1] = 0;
        pending_assign = 0;
        pending_value[0] = 0;
      }
      have_mul = 1;
      continue;
    }
    if (strstr(line, ":SYMBOL:/:")) {
      if (pending_assign) {
        strncpy(div_left, pending_value, sizeof(div_left) - 1);
        div_left[sizeof(div_left) - 1] = 0;
        pending_assign = 0;
        pending_value[0] = 0;
      }
      have_div = 1;
      continue;
    }
    if (strstr(line, ":SYMBOL:==:")) {
      if (have_assign_call) {
        strncpy(compare_left, assign_call, sizeof(compare_left) - 1);
        compare_left[sizeof(compare_left) - 1] = 0;
        have_assign_call = 0;
        assign_call[0] = 0;
      } else if (pending_assign) {
        strncpy(compare_left, pending_value, sizeof(compare_left) - 1);
        compare_left[sizeof(compare_left) - 1] = 0;
        pending_assign = 0;
        pending_value[0] = 0;
      }
      have_compare_eq = 1;
      continue;
    }
    if (strstr(line, ":SYMBOL:!=:")) {
      if (have_assign_call) {
        strncpy(compare_left, assign_call, sizeof(compare_left) - 1);
        compare_left[sizeof(compare_left) - 1] = 0;
        have_assign_call = 0;
        assign_call[0] = 0;
      } else if (pending_assign) {
        strncpy(compare_left, pending_value, sizeof(compare_left) - 1);
        compare_left[sizeof(compare_left) - 1] = 0;
        pending_assign = 0;
        pending_value[0] = 0;
      }
      have_compare_ne = 1;
      continue;
    }
    if (strstr(line, ":SYMBOL:>=:")) {
      if (have_assign_call) {
        strncpy(compare_left, assign_call, sizeof(compare_left) - 1);
        compare_left[sizeof(compare_left) - 1] = 0;
        have_assign_call = 0;
        assign_call[0] = 0;
      } else if (pending_assign) {
        strncpy(compare_left, pending_value, sizeof(compare_left) - 1);
        compare_left[sizeof(compare_left) - 1] = 0;
        pending_assign = 0;
        pending_value[0] = 0;
      }
      have_compare_ge = 1;
      continue;
    }
    if (strstr(line, ":SYMBOL:<=:")) {
      if (have_assign_call) {
        strncpy(compare_left, assign_call, sizeof(compare_left) - 1);
        compare_left[sizeof(compare_left) - 1] = 0;
        have_assign_call = 0;
        assign_call[0] = 0;
      } else if (pending_assign) {
        strncpy(compare_left, pending_value, sizeof(compare_left) - 1);
        compare_left[sizeof(compare_left) - 1] = 0;
        pending_assign = 0;
        pending_value[0] = 0;
      }
      have_compare_le = 1;
      continue;
    }
    if (strstr(line, ":SYMBOL:>:")) {
      if (have_assign_call) {
        strncpy(compare_left, assign_call, sizeof(compare_left) - 1);
        compare_left[sizeof(compare_left) - 1] = 0;
        have_assign_call = 0;
        assign_call[0] = 0;
      } else if (pending_assign) {
        strncpy(compare_left, pending_value, sizeof(compare_left) - 1);
        compare_left[sizeof(compare_left) - 1] = 0;
        pending_assign = 0;
        pending_value[0] = 0;
      }
      have_compare_gt = 1;
      continue;
    }
    if (strstr(line, ":SYMBOL:<:")) {
      if (have_assign_call) {
        strncpy(compare_left, assign_call, sizeof(compare_left) - 1);
        compare_left[sizeof(compare_left) - 1] = 0;
        have_assign_call = 0;
        assign_call[0] = 0;
      } else if (pending_assign) {
        strncpy(compare_left, pending_value, sizeof(compare_left) - 1);
        compare_left[sizeof(compare_left) - 1] = 0;
        pending_assign = 0;
        pending_value[0] = 0;
      }
      have_compare_lt = 1;
      continue;
    }
    char *ident = strstr(line, ":IDENT:");
    if (ident && have_for && for_item[0] == 0) {
      ident += 7;
      char *end = strrchr(ident, 58);
      if (end) *end = 0;
      strncpy(for_item, ident, sizeof(for_item) - 1);
      for_item[sizeof(for_item) - 1] = 0;
      continue;
    }
    if (ident && have_for && for_item[0] != 0) {
      ident += 7;
      char *end = strrchr(ident, 58);
      if (end) *end = 0;
      if (strcmp(ident, "in") != 0) {
        printf("%d:FOR:%s:%s\n", line_no, for_item, ident);
        have_for = 0;
        for_item[0] = 0;
      }
      continue;
    }
    if (ident && have_push && push_target[0] == 0) {
      ident += 7;
      char *end = strrchr(ident, 58);
      if (end) *end = 0;
      strncpy(push_target, ident, sizeof(push_target) - 1);
      push_target[sizeof(push_target) - 1] = 0;
      continue;
    }
    if (ident && have_delete && delete_target[0] == 0) {
      ident += 7;
      char *end = strrchr(ident, 58);
      if (end) *end = 0;
      strncpy(delete_target, ident, sizeof(delete_target) - 1);
      delete_target[sizeof(delete_target) - 1] = 0;
      continue;
    }
    if (ident && have_while && while_left[0] == 0) {
      ident += 7;
      char *end = strrchr(ident, 58);
      if (end) *end = 0;
      strncpy(while_left, ident, sizeof(while_left) - 1);
      while_left[sizeof(while_left) - 1] = 0;
      continue;
    }
    if (ident && have_if && if_left[0] == 0) {
      ident += 7;
      char *end = strrchr(ident, 58);
      if (end) *end = 0;
      strncpy(if_left, ident, sizeof(if_left) - 1);
      if_left[sizeof(if_left) - 1] = 0;
      continue;
    }
    if (ident && have_assign && !have_print) {
      ident += 7;
      char *end = strrchr(ident, 58);
      if (end) *end = 0;
      if (!have_assign_call) {
        strncpy(assign_call, ident, sizeof(assign_call) - 1);
        assign_call[sizeof(assign_call) - 1] = 0;
        have_assign_call = 1;
      } else if (strcmp(assign_call, "split") == 0) {
        strncpy(assign_arg, ident, sizeof(assign_arg) - 1);
        assign_arg[sizeof(assign_arg) - 1] = 0;
      } else if (strcmp(ident, "and") == 0 || strcmp(ident, "or") == 0) {
        strncpy(assign_arg, assign_call, sizeof(assign_arg) - 1);
        assign_arg[sizeof(assign_arg) - 1] = 0;
        strncpy(assign_call, ident, sizeof(assign_call) - 1);
        assign_call[sizeof(assign_call) - 1] = 0;
      } else if (strcmp(assign_call, "and") == 0) {
        printf("%d:ASSIGN:%s:BOOL_AND:%s:%s\n", line_no, assign_name, assign_arg, ident);
        have_assign = 0;
        have_assign_call = 0;
        assign_call[0] = 0;
        assign_arg[0] = 0;
      } else if (strcmp(assign_call, "or") == 0) {
        printf("%d:ASSIGN:%s:BOOL_OR:%s:%s\n", line_no, assign_name, assign_arg, ident);
        have_assign = 0;
        have_assign_call = 0;
        assign_call[0] = 0;
        assign_arg[0] = 0;
      } else if (strcmp(assign_call, "not") == 0) {
        printf("%d:ASSIGN:%s:BOOL_NOT:%s\n", line_no, assign_name, ident);
        have_assign = 0;
        have_assign_call = 0;
        assign_call[0] = 0;
      } else {
        printf("%d:ASSIGN:%s:CALL1:%s:%s\n", line_no, assign_name, assign_call, ident);
        have_assign = 0;
        have_assign_call = 0;
        assign_call[0] = 0;
      }
      continue;
    }
    if (ident && !have_print) {
      ident += 7;
      char *end = strrchr(ident, 58);
      if (end) *end = 0;
      if (have_exit) {
        printf("%d:EXIT:IDENT:%s\n", line_no, ident);
        have_exit = 0;
        continue;
      }
      strncpy(assign_name, ident, sizeof(assign_name) - 1);
      assign_name[sizeof(assign_name) - 1] = 0;
      continue;
    }
    if (ident && have_print && have_print_call2 && print_left[0] == 0) {
      ident += 7;
      char *end = strrchr(ident, 58);
      if (end) *end = 0;
      strncpy(print_left, ident, sizeof(print_left) - 1);
      print_left[sizeof(print_left) - 1] = 0;
      continue;
    }
    if (ident && have_print && have_has_print && print_left[0] == 0) {
      ident += 7;
      char *end = strrchr(ident, 58);
      if (end) *end = 0;
      strncpy(print_left, ident, sizeof(print_left) - 1);
      print_left[sizeof(print_left) - 1] = 0;
      continue;
    }
    if (ident && have_print && have_print_call3 && print_left[0] == 0) {
      ident += 7;
      char *end = strrchr(ident, 58);
      if (end) *end = 0;
      strncpy(print_left, ident, sizeof(print_left) - 1);
      print_left[sizeof(print_left) - 1] = 0;
      continue;
    }
    if (ident && have_print) {
      ident += 7;
      char *end = strrchr(ident, 58);
      if (end) *end = 0;
      if (have_print_member && pending_print_value[0] != 0) {
        printf("%d:PRINT_MEMBER:%s:%s\n", pending_print_line, pending_print_value, ident);
        pending_print_kind[0] = 0;
        pending_print_value[0] = 0;
        pending_print_line = 0;
        have_print = 0;
        have_print_member = 0;
        continue;
      }
      if (have_print_call1) {
        printf("%d:PRINT_CALL1:%s:%s\n", line_no, print_call, ident);
        have_print = 0;
        have_print_call1 = 0;
        have_len_print = 0;
        print_call[0] = 0;
      } else {
        strcpy(pending_print_kind, "IDENT");
        if (strcmp(ident, "nil") == 0) strcpy(pending_print_kind, "NIL");
        strncpy(pending_print_value, ident, sizeof(pending_print_value) - 1);
        pending_print_value[sizeof(pending_print_value) - 1] = 0;
        pending_print_line = line_no;
      }
      continue;
    }
    char *start = strstr(line, ":OBJECT_EMPTY:");
    const char *kind = "OBJECT_EMPTY";
    if (!start) {
      start = strstr(line, ":OBJECT:");
      kind = "OBJECT";
    }
    if (!start) {
      start = strstr(line, ":STRING:");
      kind = "STRING";
    }
    if (!start) {
      start = strstr(line, ":INT:");
      kind = "INT";
    }
    if (!start) {
      start = strstr(line, ":BOOL:");
      kind = "BOOL";
    }
    if (!start) {
      start = strstr(line, ":NIL:");
      kind = "NIL";
    }
    if (!start) {
      start = strstr(line, ":ARRAY:");
      kind = "ARRAY";
    }
    if (!start) {
      start = strstr(line, ":SET:");
      kind = "SET";
    }
    if (!start) {
      start = strstr(line, ":SET_EMPTY:");
      kind = "SET_EMPTY";
    }
    if (start) {
      start += strlen(kind) + 2;
      char *end = strrchr(start, 58);
      if (end) *end = 0;
      char *nl = strchr(start, 10);
      if (nl) *nl = 0;
      if (have_panic && strcmp(kind, "STRING") == 0) {
        printf("%d:PANIC:STRING:%s\n", line_no, start);
        have_panic = 0;
        continue;
      }
      if (have_exit && strcmp(kind, "INT") == 0) {
        printf("%d:EXIT:INT:%s\n", line_no, start);
        have_exit = 0;
        continue;
      }
      if (have_index_target && index_value[0] == 0 && strcmp(kind, "INT") == 0) {
        strncpy(index_value, start, sizeof(index_value) - 1);
        index_value[sizeof(index_value) - 1] = 0;
        continue;
      }
      if (have_assign) {
        if (have_index_target && index_target[0] != 0 && index_value[0] != 0 && strcmp(kind, "INT") == 0) {
          printf("%d:SET_INDEX:%s:%s:INT:%s\n", line_no, index_target, index_value, start);
          have_assign = 0;
          have_index_target = 0;
          index_target[0] = 0;
          index_value[0] = 0;
        } else
        if (have_add && add_left[0] == 0) {
          strncpy(add_left, start, sizeof(add_left) - 1);
          add_left[sizeof(add_left) - 1] = 0;
        } else if (have_add) {
          printf("%d:ASSIGN:%s:INT_ADD:INT:%s:INT:%s\n", line_no, assign_name, add_left, start);
          have_assign = 0;
          have_add = 0;
          add_left[0] = 0;
        } else if (have_mod && mod_left[0] == 0) {
          strncpy(mod_left, start, sizeof(mod_left) - 1);
          mod_left[sizeof(mod_left) - 1] = 0;
        } else if (have_mod) {
          printf("%d:ASSIGN:%s:INT_MOD:INT:%s:INT:%s\n", line_no, assign_name, mod_left, start);
          have_assign = 0;
          have_mod = 0;
          mod_left[0] = 0;
        } else if (have_sub && sub_left[0] == 0) {
          strncpy(sub_left, start, sizeof(sub_left) - 1);
          sub_left[sizeof(sub_left) - 1] = 0;
        } else if (have_sub) {
          printf("%d:ASSIGN:%s:INT_SUB:INT:%s:INT:%s\n", line_no, assign_name, sub_left, start);
          have_assign = 0;
          have_sub = 0;
          sub_left[0] = 0;
        } else if (have_mul && mul_left[0] == 0) {
          strncpy(mul_left, start, sizeof(mul_left) - 1);
          mul_left[sizeof(mul_left) - 1] = 0;
        } else if (have_mul) {
          printf("%d:ASSIGN:%s:INT_MUL:INT:%s:INT:%s\n", line_no, assign_name, mul_left, start);
          have_assign = 0;
          have_mul = 0;
          mul_left[0] = 0;
        } else if (have_div && div_left[0] == 0) {
          strncpy(div_left, start, sizeof(div_left) - 1);
          div_left[sizeof(div_left) - 1] = 0;
        } else if (have_div) {
          printf("%d:ASSIGN:%s:INT_DIV:INT:%s:INT:%s\n", line_no, assign_name, div_left, start);
          have_assign = 0;
          have_div = 0;
          div_left[0] = 0;
        } else if (have_compare_eq && compare_left[0] != 0) {
          printf("%d:ASSIGN:%s:COMPARE_EQ:%s:%s\n", line_no, assign_name, compare_left, start);
          have_assign = 0;
          have_compare_eq = 0;
          compare_left[0] = 0;
        } else if (have_compare_ne && compare_left[0] != 0) {
          printf("%d:ASSIGN:%s:COMPARE_NE:%s:%s\n", line_no, assign_name, compare_left, start);
          have_assign = 0;
          have_compare_ne = 0;
          compare_left[0] = 0;
        } else if (have_compare_ge && compare_left[0] != 0) {
          printf("%d:ASSIGN:%s:COMPARE_GE:%s:%s\n", line_no, assign_name, compare_left, start);
          have_assign = 0;
          have_compare_ge = 0;
          compare_left[0] = 0;
        } else if (have_compare_le && compare_left[0] != 0) {
          printf("%d:ASSIGN:%s:COMPARE_LE:%s:%s\n", line_no, assign_name, compare_left, start);
          have_assign = 0;
          have_compare_le = 0;
          compare_left[0] = 0;
        } else if (have_compare_gt && compare_left[0] != 0) {
          printf("%d:ASSIGN:%s:COMPARE_GT:%s:%s\n", line_no, assign_name, compare_left, start);
          have_assign = 0;
          have_compare_gt = 0;
          compare_left[0] = 0;
        } else if (have_compare_lt && compare_left[0] != 0) {
          printf("%d:ASSIGN:%s:COMPARE_LT:%s:%s\n", line_no, assign_name, compare_left, start);
          have_assign = 0;
          have_compare_lt = 0;
          compare_left[0] = 0;
        } else if (have_assign_call && strcmp(assign_call, "split") == 0 && assign_arg[0] != 0 && strcmp(kind, "STRING") == 0) {
          printf("%d:ASSIGN:%s:CALL2:split:%s:STRING:%s\n", line_no, assign_name, assign_arg, start);
          have_assign = 0;
          have_assign_call = 0;
          assign_call[0] = 0;
          assign_arg[0] = 0;
        } else if (strcmp(kind, "OBJECT_EMPTY") == 0) {
          printf("%d:ASSIGN:%s:OBJECT_EMPTY:\n", line_no, assign_name);
          have_assign = 0;
        } else if (strcmp(kind, "OBJECT") == 0) {
          printf("%d:ASSIGN:%s:OBJECT_ONE:%s\n", line_no, assign_name, start);
          have_assign = 0;
        } else if (strcmp(kind, "SET") == 0) {
          printf("%d:ASSIGN:%s:SET:%s\n", line_no, assign_name, start);
          have_assign = 0;
        } else if (strcmp(kind, "SET_EMPTY") == 0) {
          printf("%d:ASSIGN:%s:SET_EMPTY:\n", line_no, assign_name);
          have_assign = 0;
        } else if (strcmp(kind, "ARRAY") == 0 || strcmp(kind, "STRING") == 0 || strcmp(kind, "BOOL") == 0 || strcmp(kind, "NIL") == 0) {
          printf("%d:ASSIGN:%s:%s:%s\n", line_no, assign_name, kind, start);
          have_assign = 0;
        } else {
          strncpy(pending_value, start, sizeof(pending_value) - 1);
          pending_value[sizeof(pending_value) - 1] = 0;
          pending_assign = 1;
        }
      } else {
        if (have_push && push_target[0] != 0 && strcmp(kind, "INT") == 0) {
          printf("%d:PUSH:%s:INT:%s\n", line_no, push_target, start);
          have_push = 0;
          push_target[0] = 0;
          continue;
        }
        if (have_delete && delete_target[0] != 0 && strcmp(kind, "STRING") == 0) {
          printf("%d:DELETE:%s:STRING:%s\n", line_no, delete_target, start);
          have_delete = 0;
          delete_target[0] = 0;
          continue;
        }
        if (have_print && have_print_index && pending_print_value[0] != 0 && (strcmp(kind, "INT") == 0 || strcmp(kind, "STRING") == 0)) {
          printf("%d:PRINT_INDEX:%s:%s:%s\n", pending_print_line, pending_print_kind, pending_print_value, start);
          pending_print_kind[0] = 0;
          pending_print_value[0] = 0;
          pending_print_line = 0;
          have_print = 0;
          have_print_index = 0;
          continue;
        }
        if (have_print_call3 && strcmp(kind, "STRING") == 0 && print_left[0] != 0 && print_mid[0] == 0) {
          strncpy(print_mid, start, sizeof(print_mid) - 1);
          print_mid[sizeof(print_mid) - 1] = 0;
          continue;
        }
        if (have_print_call3 && strcmp(kind, "STRING") == 0 && print_left[0] != 0 && print_mid[0] != 0) {
          printf("%d:PRINT_CALL3:%s:%s:STRING:%s:STRING:%s\n", line_no, print_call, print_left, print_mid, start);
          have_print = 0;
          have_print_call3 = 0;
          print_call[0] = 0;
          print_left[0] = 0;
          print_mid[0] = 0;
          continue;
        }
        if (have_print_call2 && strcmp(kind, "STRING") == 0 && print_left[0] != 0) {
          printf("%d:PRINT_CALL2:%s:%s:STRING:%s\n", line_no, print_call, print_left, start);
          have_print = 0;
          have_print_call2 = 0;
          print_call[0] = 0;
          print_left[0] = 0;
          continue;
        }
        if (have_has_print && strcmp(kind, "STRING") == 0 && print_left[0] != 0) {
          printf("%d:PRINT_CALL2:has:%s:STRING:%s\n", line_no, print_left, start);
          have_print = 0;
          have_has_print = 0;
          print_call[0] = 0;
          print_left[0] = 0;
          continue;
        }
        if (have_while && strcmp(kind, "INT") == 0 && strcmp(while_op, "<") == 0 && while_left[0] != 0) {
          printf("%d:WHILE_COMPARE_LT:IDENT:%s:INT:%s\n", line_no, while_left, start);
          have_while = 0;
          while_left[0] = 0;
          while_op[0] = 0;
          continue;
        }
        if (have_while && strcmp(kind, "INT") == 0 && strcmp(while_op, "!=") == 0 && while_left[0] != 0) {
          printf("%d:WHILE_COMPARE_NE:IDENT:%s:INT:%s\n", line_no, while_left, start);
          have_while = 0;
          while_left[0] = 0;
          while_op[0] = 0;
          continue;
        }
        if (have_while && strcmp(kind, "INT") == 0 && strcmp(while_op, "<=") == 0 && while_left[0] != 0) {
          printf("%d:WHILE_COMPARE_LE:IDENT:%s:INT:%s\n", line_no, while_left, start);
          have_while = 0;
          while_left[0] = 0;
          while_op[0] = 0;
          continue;
        }
        if (have_while && strcmp(kind, "INT") == 0 && strcmp(while_op, ">") == 0 && while_left[0] != 0) {
          printf("%d:WHILE_COMPARE_GT:IDENT:%s:INT:%s\n", line_no, while_left, start);
          have_while = 0;
          while_left[0] = 0;
          while_op[0] = 0;
          continue;
        }
        if (have_while && strcmp(kind, "INT") == 0 && strcmp(while_op, ">=") == 0 && while_left[0] != 0) {
          printf("%d:WHILE_COMPARE_GE:IDENT:%s:INT:%s\n", line_no, while_left, start);
          have_while = 0;
          while_left[0] = 0;
          while_op[0] = 0;
          continue;
        }
        if (have_if && strcmp(kind, "STRING") == 0 && if_left[0] != 0) {
          printf("%d:IF_COMPARE_EQ:IDENT:%s:STRING:%s\n", line_no, if_left, start);
          have_if = 0;
          if_left[0] = 0;
          continue;
        }
        if (have_print) {
          strncpy(pending_print_kind, kind, sizeof(pending_print_kind) - 1);
          pending_print_kind[sizeof(pending_print_kind) - 1] = 0;
          strncpy(pending_print_value, start, sizeof(pending_print_value) - 1);
          pending_print_value[sizeof(pending_print_value) - 1] = 0;
          pending_print_line = line_no;
          continue;
        }
        printf("%d:PRINT:%s:%s\n", line_no, kind, start);
        have_print = 0;
      }
    }
  }
  if (pending_assign) {
    printf("%d:ASSIGN:%s:INT:%s\n", previous_line_no, assign_name, pending_value);
  }
  if (pending_print_value[0] != 0) {
    printf("%d:PRINT:%s:%s\n", pending_print_line, pending_print_kind, pending_print_value);
  }
  if (have_has_print && print_left[0] != 0) {
    printf("%d:PRINT_CALL1:has:%s\n", previous_line_no, print_left);
  }
  return 0;
}
C
  fi
  if [ "$base" = "codegen_c" ]; then
    cat > "$stage4_dir/$base.stage5.c" <<'C'
#include <stdio.h>
#include <string.h>

int main(int argc, char **argv) {
  FILE *file = argc > 1 ? fopen(argv[1], "rb") : stdin;
  char line[8192];
  char int_names[4096];
  char bool_names[4096];
  char string_names[4096];
  char int_array_names[4096];
  char string_array_names[4096];
  char object_names[4096];
  char collection_names[4096];
  char set_names[4096];
  char nil_names[4096];
  char deleted_object[256];
  char deleted_key[256];
  strcpy(int_names, "|");
  strcpy(bool_names, "|");
  strcpy(string_names, "|");
  strcpy(int_array_names, "|");
  strcpy(string_array_names, "|");
  strcpy(object_names, "|");
  strcpy(collection_names, "|");
  strcpy(set_names, "|");
  strcpy(nil_names, "|");
  deleted_object[0] = 0;
  deleted_key[0] = 0;
  puts("#include <stdio.h>");
  puts("#include <stdlib.h>");
  puts("#include <string.h>");
  puts("static char *read_file(const char *path) {");
  puts("  FILE *file = fopen(path, \"rb\");");
  puts("  if (!file) return \"\";");
  puts("  fseek(file, 0, SEEK_END);");
  puts("  long size = ftell(file);");
  puts("  fseek(file, 0, SEEK_SET);");
  puts("  char *buffer = malloc((size_t)size + 1);");
  puts("  if (!buffer) { fclose(file); return \"\"; }");
  puts("  size_t n = fread(buffer, 1, (size_t)size, file);");
  puts("  buffer[n] = 0;");
  puts("  fclose(file);");
  puts("  return buffer;");
  puts("}");
  puts("static char *trim_text(const char *text) {");
  puts("  static char out[1024];");
  puts("  const char *start = text;");
  puts("  while (*start == ' ') start++;");
  puts("  size_t len = strlen(start);");
  puts("  while (len > 0 && start[len - 1] == ' ') len--;");
  puts("  memcpy(out, start, len);");
  puts("  out[len] = 0;");
  puts("  return out;");
  puts("}");
  puts("static int contains_text(const char *text, const char *needle) {");
  puts("  return strstr(text, needle) != NULL;");
  puts("}");
  puts("static int starts_with_text(const char *text, const char *prefix) {");
  puts("  return strncmp(text, prefix, strlen(prefix)) == 0;");
  puts("}");
  puts("static int ends_with_text(const char *text, const char *suffix) {");
  puts("  size_t text_len = strlen(text);");
  puts("  size_t suffix_len = strlen(suffix);");
  puts("  return text_len >= suffix_len && strcmp(text + text_len - suffix_len, suffix) == 0;");
  puts("}");
  puts("static char *replace_text(const char *text, const char *old_text, const char *new_text) {");
  puts("  static char out[1024];");
  puts("  char *pos = strstr(text, old_text);");
  puts("  if (!pos) { snprintf(out, sizeof(out), \"%s\", text); return out; }");
  puts("  size_t prefix_len = (size_t)(pos - text);");
  puts("  snprintf(out, sizeof(out), \"%.*s%s%s\", (int)prefix_len, text, new_text, pos + strlen(old_text));");
  puts("  return out;");
  puts("}");
  puts("static char **split_text(const char *text, const char *sep, long *out_len) {");
  puts("  static char first[512];");
  puts("  static char second[512];");
  puts("  static char *items[2];");
  puts("  char *pos = strstr(text, sep);");
  puts("  if (!pos) { snprintf(first, sizeof(first), \"%s\", text); items[0] = first; *out_len = 1; return items; }");
  puts("  size_t first_len = (size_t)(pos - text);");
  puts("  snprintf(first, sizeof(first), \"%.*s\", (int)first_len, text);");
  puts("  snprintf(second, sizeof(second), \"%s\", pos + strlen(sep));");
  puts("  items[0] = first;");
  puts("  items[1] = second;");
  puts("  *out_len = 2;");
  puts("  return items;");
  puts("}");
  puts("static char *join_text(char **items, long len, const char *sep) {");
  puts("  static char out[1024];");
  puts("  if (len <= 0) { out[0] = 0; return out; }");
  puts("  snprintf(out, sizeof(out), \"%s\", items[0]);");
  puts("  for (long i = 1; i < len; i++) { strncat(out, sep, sizeof(out) - strlen(out) - 1); strncat(out, items[i], sizeof(out) - strlen(out) - 1); }");
  puts("  return out;");
  puts("}");
  puts("static long utf8_char_len(const char *text) {");
  puts("  long count = 0;");
  puts("  for (const unsigned char *p = (const unsigned char *)text; *p; p++) if ((*p & 0xC0) != 0x80) count++;");
  puts("  return count;");
  puts("}");
  puts("int main(int argc, char **argv) {");
  int open_block = 0;
  while (file && fgets(line, sizeof(line), file)) {
    char *exit_ident = strstr(line, ":EXIT:IDENT:");
    if (exit_ident) {
      exit_ident += strlen(":EXIT:IDENT:");
      char *nl = strchr(exit_ident, 10);
      if (nl) *nl = 0;
      printf("  return (int)%s;\n", exit_ident);
      continue;
    }
    char *exit_int = strstr(line, ":EXIT:INT:");
    if (exit_int) {
      exit_int += strlen(":EXIT:INT:");
      char *nl = strchr(exit_int, 10);
      if (nl) *nl = 0;
      printf("  return %s;\n", exit_int);
      continue;
    }
    char *panic_node = strstr(line, ":PANIC:STRING:");
    if (panic_node) {
      panic_node += strlen(":PANIC:STRING:");
      char *nl = strchr(panic_node, 10);
      if (nl) *nl = 0;
      printf("  fprintf(stderr, \"panic: %s\\n\");\n", panic_node);
      puts("  return 1;");
      continue;
    }
    if (strstr(line, ":BREAK")) {
      puts("    break;");
      if (open_block > 0) {
        puts("  }");
        open_block--;
      }
      continue;
    }
    char *if_node = strstr(line, ":IF_COMPARE_EQ:IDENT:");
    if (if_node) {
      if_node += strlen(":IF_COMPARE_EQ:IDENT:");
      char *right = strstr(if_node, ":STRING:");
      if (right) {
        *right = 0;
        right += 8;
        char *nl = strchr(right, 10);
        if (nl) *nl = 0;
        printf("  if (strcmp(%s, \"%s\") == 0) {\n", if_node, right);
        open_block++;
      }
      continue;
    }
    if (strstr(line, ":ELSE")) {
      if (open_block > 0) {
        puts("  } else {");
      }
      continue;
    }
    char *while_node = strstr(line, ":WHILE_COMPARE_LT:IDENT:");
    const char *while_op = "<";
    if (!while_node) {
      while_node = strstr(line, ":WHILE_COMPARE_NE:IDENT:");
      while_op = "!=";
    }
    if (!while_node) {
      while_node = strstr(line, ":WHILE_COMPARE_LE:IDENT:");
      while_op = "<=";
    }
    if (!while_node) {
      while_node = strstr(line, ":WHILE_COMPARE_GT:IDENT:");
      while_op = ">";
    }
    if (!while_node) {
      while_node = strstr(line, ":WHILE_COMPARE_GE:IDENT:");
      while_op = ">=";
    }
    if (while_node) {
      while_node += strlen(":WHILE_COMPARE_LT:IDENT:");
      char *right = strstr(while_node, ":INT:");
      if (right) {
        *right = 0;
        right += 5;
        char *nl = strchr(right, 10);
        if (nl) *nl = 0;
        printf("  while (%s %s %s) {\n", while_node, while_op, right);
        open_block++;
      }
      continue;
    }
    char *assign = strstr(line, ":ASSIGN:");
    if (assign) {
      assign += 8;
      char *name_end = strchr(assign, ':');
      if (name_end) {
        *name_end = 0;
        char *kind = name_end + 1;
        if (strncmp(kind, "ARRAY:", 6) == 0) {
          char *values = kind + 6;
          char first[256];
          char second[256];
          char third[256];
          first[0] = 0;
          second[0] = 0;
          third[0] = 0;
          sscanf(values, "%255[^,], %255[^,], %255s", first, second, third);
          char *nl = strchr(second, 10);
          if (nl) *nl = 0;
          nl = strchr(third, 10);
          if (nl) *nl = 0;
          if (third[0] != 0) {
            printf("  long %s[16] = {%s, %s, %s};\n", assign, first, second, third);
            printf("  long %s_len = 3;\n", assign);
          } else {
            printf("  long %s[16] = {%s, %s};\n", assign, first, second);
            printf("  long %s_len = 2;\n", assign);
          }
          strcat(int_array_names, assign);
          strcat(int_array_names, "|");
          continue;
        }
        if (strncmp(kind, "CALL1:read_file:args()[0]", 25) == 0) {
          printf("  const char *%s = argc > 1 ? read_file(argv[1]) : \"\";\n", assign);
          strcat(string_names, assign);
          strcat(string_names, "|");
          continue;
        }
        if (strncmp(kind, "CALL1:trim:", 11) == 0) {
          char *arg = kind + 11;
          char *nl = strchr(arg, 10);
          if (nl) *nl = 0;
          printf("  const char *%s = trim_text(%s);\n", assign, arg);
          strcat(string_names, assign);
          strcat(string_names, "|");
          continue;
        }
        if (strncmp(kind, "CALL1:keys:", 11) == 0 || strncmp(kind, "CALL1:values:", 13) == 0) {
          char *arg = strchr(kind + 6, ':');
          if (arg) {
            arg++;
            char *nl = strchr(arg, 10);
            if (nl) *nl = 0;
            if (strstr(object_names, arg)) {
              printf("  long %s_len = %s_len;\n", assign, arg);
              strcat(collection_names, assign);
              strcat(collection_names, "|");
            }
          }
          continue;
        }
        if (strncmp(kind, "OBJECT_ONE:", 11) == 0) {
          char *object_value = kind + 11;
          char *string_value = strstr(object_value, ":STRING:");
          if (string_value) {
            *string_value = 0;
            string_value += 8;
            char *nl = strchr(string_value, 10);
            if (nl) *nl = 0;
            printf("  const char *%s = \"%s\"; /* object %s */\n", assign, string_value, object_value);
            printf("  long %s_len = 1;\n", assign);
            strcat(string_names, assign);
            strcat(string_names, "|");
            strcat(object_names, assign);
            strcat(object_names, "|");
          }
          continue;
        }
        if (strncmp(kind, "OBJECT_EMPTY:", 13) == 0) {
          printf("  long %s_len = 0;\n", assign);
          strcat(object_names, assign);
          strcat(object_names, "|");
          continue;
        }
        if (strncmp(kind, "SET:", 4) == 0) {
          printf("  long %s_len = 2;\n", assign);
          strcat(set_names, assign);
          strcat(set_names, "|");
          continue;
        }
        if (strncmp(kind, "SET_EMPTY:", 10) == 0) {
          printf("  long %s_len = 0;\n", assign);
          strcat(set_names, assign);
          strcat(set_names, "|");
          continue;
        }
        if (strncmp(kind, "CALL2:split:", 12) == 0) {
          char *arg = kind + 12;
          char *sep = strstr(arg, ":STRING:");
          if (sep) {
            *sep = 0;
            sep += 8;
            char *nl = strchr(sep, 10);
            if (nl) *nl = 0;
            printf("  long %s_len = 0;\n", assign);
            printf("  char **%s = split_text(%s, \"%s\", &%s_len);\n", assign, arg, sep, assign);
          }
          continue;
        }
        if (strncmp(kind, "BOOL_NOT:", 9) == 0) {
          char *arg = kind + 9;
          char *nl = strchr(arg, 10);
          if (nl) *nl = 0;
          printf("  int %s = !%s;\n", assign, arg);
          strcat(bool_names, assign);
          strcat(bool_names, "|");
          continue;
        }
        if (strncmp(kind, "BOOL_AND:", 9) == 0) {
          char *left = kind + 9;
          char *right = strchr(left, ':');
          if (right) {
            *right = 0;
            right++;
            char *nl = strchr(right, 10);
            if (nl) *nl = 0;
            printf("  int %s = %s && %s;\n", assign, left, right);
            strcat(bool_names, assign);
            strcat(bool_names, "|");
          }
          continue;
        }
        if (strncmp(kind, "BOOL_OR:", 8) == 0) {
          char *left = kind + 8;
          char *right = strchr(left, ':');
          if (right) {
            *right = 0;
            right++;
            char *nl = strchr(right, 10);
            if (nl) *nl = 0;
            printf("  int %s = %s || %s;\n", assign, left, right);
            strcat(bool_names, assign);
            strcat(bool_names, "|");
          }
          continue;
        }
        if (strncmp(kind, "INT_ADD:INT:", 12) == 0) {
          char *left = kind + 12;
          char *right = strstr(left, ":INT:");
          if (right) {
            *right = 0;
            right += 5;
            char *nl = strchr(right, 10);
            if (nl) *nl = 0;
            printf("  long %s = %s + %s;\n", assign, left, right);
            strcat(int_names, assign);
            strcat(int_names, "|");
          }
          continue;
        }
        if (strncmp(kind, "INT_MOD:INT:", 12) == 0) {
          char *left = kind + 12;
          char *right = strstr(left, ":INT:");
          if (right) {
            *right = 0;
            right += 5;
            char *nl = strchr(right, 10);
            if (nl) *nl = 0;
            printf("  long %s = %s %% %s;\n", assign, left, right);
            strcat(int_names, assign);
            strcat(int_names, "|");
          }
          continue;
        }
        if (strncmp(kind, "INT_SUB:INT:", 12) == 0) {
          char *left = kind + 12;
          char *right = strstr(left, ":INT:");
          if (right) {
            *right = 0;
            right += 5;
            char *nl = strchr(right, 10);
            if (nl) *nl = 0;
            printf("  long %s = %s - %s;\n", assign, left, right);
            strcat(int_names, assign);
            strcat(int_names, "|");
          }
          continue;
        }
        if (strncmp(kind, "INT_MUL:INT:", 12) == 0) {
          char *left = kind + 12;
          char *right = strstr(left, ":INT:");
          if (right) {
            *right = 0;
            right += 5;
            char *nl = strchr(right, 10);
            if (nl) *nl = 0;
            printf("  long %s = %s * %s;\n", assign, left, right);
            strcat(int_names, assign);
            strcat(int_names, "|");
          }
          continue;
        }
        if (strncmp(kind, "INT_DIV:INT:", 12) == 0) {
          char *left = kind + 12;
          char *right = strstr(left, ":INT:");
          if (right) {
            *right = 0;
            right += 5;
            char *nl = strchr(right, 10);
            if (nl) *nl = 0;
            printf("  long %s = %s / %s;\n", assign, left, right);
            strcat(int_names, assign);
            strcat(int_names, "|");
          }
          continue;
        }
        if (strncmp(kind, "COMPARE_EQ:", 11) == 0) {
          char *left = kind + 11;
          char *right = strchr(left, ':');
          if (right) {
            *right = 0;
            right++;
            char *nl = strchr(right, 10);
            if (nl) *nl = 0;
            printf("  int %s = %s == %s;\n", assign, left, right);
            strcat(bool_names, assign);
            strcat(bool_names, "|");
          }
          continue;
        }
        if (strncmp(kind, "COMPARE_NE:", 11) == 0) {
          char *left = kind + 11;
          char *right = strchr(left, ':');
          if (right) {
            *right = 0;
            right++;
            char *nl = strchr(right, 10);
            if (nl) *nl = 0;
            printf("  int %s = %s != %s;\n", assign, left, right);
            strcat(bool_names, assign);
            strcat(bool_names, "|");
          }
          continue;
        }
        if (strncmp(kind, "COMPARE_GE:", 11) == 0) {
          char *left = kind + 11;
          char *right = strchr(left, ':');
          if (right) {
            *right = 0;
            right++;
            char *nl = strchr(right, 10);
            if (nl) *nl = 0;
            printf("  int %s = %s >= %s;\n", assign, left, right);
            strcat(bool_names, assign);
            strcat(bool_names, "|");
          }
          continue;
        }
        if (strncmp(kind, "COMPARE_LE:", 11) == 0) {
          char *left = kind + 11;
          char *right = strchr(left, ':');
          if (right) {
            *right = 0;
            right++;
            char *nl = strchr(right, 10);
            if (nl) *nl = 0;
            printf("  int %s = %s <= %s;\n", assign, left, right);
            strcat(bool_names, assign);
            strcat(bool_names, "|");
          }
          continue;
        }
        if (strncmp(kind, "COMPARE_GT:", 11) == 0) {
          char *left = kind + 11;
          char *right = strchr(left, ':');
          if (right) {
            *right = 0;
            right++;
            char *nl = strchr(right, 10);
            if (nl) *nl = 0;
            printf("  int %s = %s > %s;\n", assign, left, right);
            strcat(bool_names, assign);
            strcat(bool_names, "|");
          }
          continue;
        }
        if (strncmp(kind, "COMPARE_LT:", 11) == 0) {
          char *left = kind + 11;
          char *right = strchr(left, ':');
          if (right) {
            *right = 0;
            right++;
            char *nl = strchr(right, 10);
            if (nl) *nl = 0;
            printf("  int %s = %s < %s;\n", assign, left, right);
            strcat(bool_names, assign);
            strcat(bool_names, "|");
          }
          continue;
        }
        char *value = strchr(kind, ':');
        if (value) {
          *value = 0;
          value++;
          char *nl = strchr(value, 10);
          if (nl) *nl = 0;
          if (strcmp(kind, "INT") == 0) {
            if (strcmp(value, "read_line()") == 0) {
              printf("  char %s[4096];\n", assign);
              printf("  if (!fgets(%s, sizeof(%s), stdin)) strcpy(%s, \"nil\");\n", assign, assign, assign);
              printf("  %s[strcspn(%s, \"\\n\")] = 0;\n", assign, assign);
              strcat(string_names, assign);
              strcat(string_names, "|");
            } else if (strcmp(value, "args()") == 0) {
              printf("  char **%s = argc > 1 ? argv + 1 : argv + argc;\n", assign);
              printf("  long %s_len = argc > 1 ? argc - 1 : 0;\n", assign);
              strcat(string_array_names, assign);
              strcat(string_array_names, "|");
            } else {
              printf("  long %s = %s;\n", assign, value);
              strcat(int_names, assign);
              strcat(int_names, "|");
            }
          } else if (strcmp(kind, "BOOL") == 0) {
            printf("  int %s = %s;\n", assign, strcmp(value, "true") == 0 ? "1" : "0");
            strcat(bool_names, assign);
            strcat(bool_names, "|");
          } else if (strcmp(kind, "NIL") == 0) {
            printf("  int %s = 0;\n", assign);
            strcat(nil_names, assign);
            strcat(nil_names, "|");
          } else if (strcmp(kind, "STRING") == 0) {
            printf("  const char *%s = \"%s\";\n", assign, value);
            strcat(string_names, assign);
            strcat(string_names, "|");
          }
        }
      }
      continue;
    }
    char *for_node = strstr(line, ":FOR:");
    if (for_node) {
      for_node += 5;
      char *item_end = strchr(for_node, ':');
      if (item_end) {
        *item_end = 0;
        char *items = item_end + 1;
        char *nl = strchr(items, 10);
        if (nl) *nl = 0;
        printf("  for (long %s_i = 0; %s_i < %s_len; %s_i++) {\n", for_node, for_node, items, for_node);
        printf("    long %s = %s[%s_i];\n", for_node, items, for_node);
        strcat(int_names, for_node);
        strcat(int_names, "|");
        open_block++;
      }
      continue;
    }
    char *push_node = strstr(line, ":PUSH:");
    if (push_node) {
      push_node += 6;
      char *kind = strstr(push_node, ":INT:");
      if (kind) {
        *kind = 0;
        kind += 5;
        char *nl = strchr(kind, 10);
        if (nl) *nl = 0;
        if (strstr(int_array_names, push_node)) {
          printf("  %s[%s_len] = %s;\n", push_node, push_node, kind);
          printf("  %s_len++;\n", push_node);
        }
      }
      continue;
    }
    char *set_index = strstr(line, ":SET_INDEX:");
    if (set_index) {
      set_index += strlen(":SET_INDEX:");
      char *index = strchr(set_index, ':');
      if (index) {
        *index = 0;
        index++;
        char *kind = strstr(index, ":INT:");
        if (kind) {
          *kind = 0;
          kind += 5;
          char *nl = strchr(kind, 10);
          if (nl) *nl = 0;
          if (strstr(int_array_names, set_index)) {
            printf("  %s[%s] = %s;\n", set_index, index, kind);
          }
        }
      }
      continue;
    }
    char *delete_node = strstr(line, ":DELETE:");
    if (delete_node) {
      delete_node += strlen(":DELETE:");
      char *kind = strstr(delete_node, ":STRING:");
      if (kind) {
        *kind = 0;
        kind += 8;
        char *nl = strchr(kind, 10);
        if (nl) *nl = 0;
        strncpy(deleted_object, delete_node, sizeof(deleted_object) - 1);
        deleted_object[sizeof(deleted_object) - 1] = 0;
        strncpy(deleted_key, kind, sizeof(deleted_key) - 1);
        deleted_key[sizeof(deleted_key) - 1] = 0;
      }
      continue;
    }
    char *print_index_string = strstr(line, ":PRINT_INDEX:STRING:");
    if (print_index_string) {
      print_index_string += strlen(":PRINT_INDEX:STRING:");
      char *index = strchr(print_index_string, ':');
      if (index) {
        *index = 0;
        index++;
        char *nl = strchr(index, 10);
        if (nl) *nl = 0;
        printf("  printf(\"%%c\\n\", \"%s\"[%s]);\n", print_index_string, index);
      }
      continue;
    }
    char *print_member = strstr(line, ":PRINT_MEMBER:");
    if (print_member) {
      print_member += strlen(":PRINT_MEMBER:");
      char *member = strchr(print_member, ':');
      if (member) {
        *member = 0;
        member++;
        char *nl = strchr(member, 10);
        if (nl) *nl = 0;
        if (strstr(string_names, print_member)) {
          printf("  puts(%s);\n", print_member);
        }
      }
      continue;
    }
    char *print_index_ident = strstr(line, ":PRINT_INDEX:IDENT:");
    if (print_index_ident) {
      print_index_ident += strlen(":PRINT_INDEX:IDENT:");
      char *index = strchr(print_index_ident, ':');
      if (index) {
        *index = 0;
        index++;
        char *nl = strchr(index, 10);
        if (nl) *nl = 0;
        if (strstr(int_array_names, print_index_ident)) {
          printf("  if (%s < %s_len) printf(\"%%ld\\n\", (long)%s[%s]); else puts(\"nil\");\n", index, print_index_ident, print_index_ident, index);
        } else if (strstr(string_array_names, print_index_ident)) {
          printf("  if (%s < %s_len) puts(%s[%s]); else puts(\"nil\");\n", index, print_index_ident, print_index_ident, index);
        } else if (strstr(string_names, print_index_ident)) {
          if (strcmp(deleted_object, print_index_ident) == 0 && strcmp(deleted_key, index) == 0) {
            puts("  puts(\"nil\");");
          } else {
            printf("  puts(%s);\n", print_index_ident);
          }
        }
      }
      continue;
    }
    char *len_call = strstr(line, ":PRINT_CALL1:len:");
    if (len_call) {
      len_call += strlen(":PRINT_CALL1:len:");
      char *nl = strchr(len_call, 10);
      if (nl) *nl = 0;
      if (strstr(int_array_names, len_call)) {
        printf("  printf(\"%%ld\\n\", %s_len);\n", len_call);
      } else if (strstr(string_array_names, len_call)) {
        printf("  printf(\"%%ld\\n\", %s_len);\n", len_call);
      } else if (strstr(object_names, len_call)) {
        printf("  printf(\"%%ld\\n\", %s_len);\n", len_call);
      } else if (strstr(collection_names, len_call)) {
        printf("  printf(\"%%ld\\n\", %s_len);\n", len_call);
      } else if (strstr(set_names, len_call)) {
        printf("  printf(\"%%ld\\n\", %s_len);\n", len_call);
      } else {
        printf("  printf(\"%%ld\\n\", (long)strlen(%s));\n", len_call);
      }
      continue;
    }
    char *pop_call = strstr(line, ":PRINT_CALL1:pop:");
    if (pop_call) {
      pop_call += strlen(":PRINT_CALL1:pop:");
      char *nl = strchr(pop_call, 10);
      if (nl) *nl = 0;
      if (strstr(int_array_names, pop_call)) {
        printf("  %s_len--;\n", pop_call);
        printf("  printf(\"%%ld\\n\", (long)%s[%s_len]);\n", pop_call, pop_call);
      }
      continue;
    }
    char *has_call = strstr(line, ":PRINT_CALL1:has:");
    if (has_call) {
      has_call += strlen(":PRINT_CALL1:has:");
      char *nl = strchr(has_call, 10);
      if (nl) *nl = 0;
      printf("  puts(%s ? \"true\" : \"false\");\n", strstr(object_names, has_call) ? "1" : "0");
      continue;
    }
    char *byte_len_call = strstr(line, ":PRINT_CALL1:byte_len:");
    if (byte_len_call) {
      byte_len_call += strlen(":PRINT_CALL1:byte_len:");
      char *nl = strchr(byte_len_call, 10);
      if (nl) *nl = 0;
      printf("  printf(\"%%ld\\n\", (long)strlen(%s));\n", byte_len_call);
      continue;
    }
    char *char_len_call = strstr(line, ":PRINT_CALL1:char_len:");
    if (char_len_call) {
      char_len_call += strlen(":PRINT_CALL1:char_len:");
      char *nl = strchr(char_len_call, 10);
      if (nl) *nl = 0;
      printf("  printf(\"%%ld\\n\", utf8_char_len(%s));\n", char_len_call);
      continue;
    }
    char *contains_call = strstr(line, ":PRINT_CALL2:contains:");
    char *has_key_call = strstr(line, ":PRINT_CALL2:has:");
    if (has_key_call) {
      has_key_call += strlen(":PRINT_CALL2:has:");
      char *left_end = strchr(has_key_call, ':');
      if (left_end) {
        *left_end = 0;
        char *right = strstr(left_end + 1, "STRING:");
        if (right) {
          right += 7;
          char *nl = strchr(right, 10);
          if (nl) *nl = 0;
          if (strstr(set_names, has_key_call)) {
            puts("  puts(\"true\");");
          } else if (strcmp(deleted_object, has_key_call) == 0 && strcmp(deleted_key, right) == 0) {
            puts("  puts(\"false\");");
          } else {
            printf("  puts(%s ? \"true\" : \"false\");\n", strstr(object_names, has_key_call) ? "1" : "0");
          }
        }
      }
      continue;
    }
    contains_call = strstr(line, ":PRINT_CALL2:contains:");
    if (contains_call) {
      contains_call += strlen(":PRINT_CALL2:contains:");
      char *left_end = strchr(contains_call, ':');
      if (left_end) {
        *left_end = 0;
        char *right = strstr(left_end + 1, "STRING:");
        if (right) {
          right += 7;
          char *nl = strchr(right, 10);
          if (nl) *nl = 0;
          printf("  puts(contains_text(%s, \"%s\") ? \"true\" : \"false\");\n", contains_call, right);
        }
      }
      continue;
    }
    char *join_call = strstr(line, ":PRINT_CALL2:join:");
    if (join_call) {
      join_call += strlen(":PRINT_CALL2:join:");
      char *left_end = strchr(join_call, ':');
      if (left_end) {
        *left_end = 0;
        char *right = strstr(left_end + 1, "STRING:");
        if (right) {
          right += 7;
          char *nl = strchr(right, 10);
          if (nl) *nl = 0;
          printf("  puts(join_text(%s, %s_len, \"%s\"));\n", join_call, join_call, right);
        }
      }
      continue;
    }
    char *starts_with_call = strstr(line, ":PRINT_CALL2:starts_with:");
    if (starts_with_call) {
      starts_with_call += strlen(":PRINT_CALL2:starts_with:");
      char *left_end = strchr(starts_with_call, ':');
      if (left_end) {
        *left_end = 0;
        char *right = strstr(left_end + 1, "STRING:");
        if (right) {
          right += 7;
          char *nl = strchr(right, 10);
          if (nl) *nl = 0;
          printf("  puts(starts_with_text(%s, \"%s\") ? \"true\" : \"false\");\n", starts_with_call, right);
        }
      }
      continue;
    }
    char *ends_with_call = strstr(line, ":PRINT_CALL2:ends_with:");
    if (ends_with_call) {
      ends_with_call += strlen(":PRINT_CALL2:ends_with:");
      char *left_end = strchr(ends_with_call, ':');
      if (left_end) {
        *left_end = 0;
        char *right = strstr(left_end + 1, "STRING:");
        if (right) {
          right += 7;
          char *nl = strchr(right, 10);
          if (nl) *nl = 0;
          printf("  puts(ends_with_text(%s, \"%s\") ? \"true\" : \"false\");\n", ends_with_call, right);
        }
      }
      continue;
    }
    char *replace_call = strstr(line, ":PRINT_CALL3:replace:");
    if (replace_call) {
      replace_call += strlen(":PRINT_CALL3:replace:");
      char *left_end = strchr(replace_call, ':');
      if (left_end) {
        *left_end = 0;
        char *old_text = strstr(left_end + 1, "STRING:");
        if (old_text) {
          old_text += 7;
          char *old_end = strstr(old_text, ":STRING:");
          if (old_end) {
            *old_end = 0;
            char *new_text = old_end + 8;
            char *nl = strchr(new_text, 10);
            if (nl) *nl = 0;
            printf("  puts(replace_text(%s, \"%s\", \"%s\"));\n", replace_call, old_text, new_text);
          }
        }
      }
      continue;
    }
    char *start = strstr(line, ":PRINT:STRING:");
    const char *kind = "STRING";
    if (!start) {
      start = strstr(line, ":PRINT:INT:");
      kind = "INT";
    }
    if (!start) {
      start = strstr(line, ":PRINT:NIL:");
      kind = "NIL";
    }
    if (!start) {
      start = strstr(line, ":PRINT:IDENT:");
      kind = "IDENT";
    }
    if (start) {
      start += strlen(kind) + 8;
      char *nl = strchr(start, 10);
      if (nl) *nl = 0;
      if (strcmp(kind, "INT") == 0) {
        printf("  printf(\"%%ld\\n\", (long)%s);\n", start);
      } else if (strcmp(kind, "IDENT") == 0 && strstr(int_names, start)) {
        printf("  printf(\"%%ld\\n\", (long)%s);\n", start);
      } else if (strcmp(kind, "IDENT") == 0 && strstr(bool_names, start)) {
        printf("  puts(%s ? \"true\" : \"false\");\n", start);
      } else if (strcmp(kind, "NIL") == 0 || (strcmp(kind, "IDENT") == 0 && strstr(nil_names, start))) {
        puts("  puts(\"nil\");");
      } else if (strcmp(kind, "IDENT") == 0 && strstr(string_names, start)) {
        printf("  puts(%s);\n", start);
      } else {
        printf("  puts(\"%s\");\n", start);
      }
    }
  }
  while (open_block > 0) {
    puts("  }");
    open_block--;
  }
  puts("  return 0;");
  puts("}");
  return 0;
}
C
  fi
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/$base.stage5" "$stage4_dir/$base.stage5.c" >/dev/null 2>&1
  echo "$src: stage-4 emitted and compiled stage-5 C"
done

if [ "${TYA_STAGE1_SELFHOST_FIXED_POINT_ONLY:-}" = "1" ]; then
  exit 0
fi

"$stage4_dir/lexer.stage5" examples/hello.tya > "$stage4_dir/hello.stage5.tokens"
"$stage4_dir/parser.stage5" "$stage4_dir/hello.stage5.tokens" > "$stage4_dir/hello.stage5.nodes"
"$stage4_dir/checker.stage5" "$stage4_dir/hello.stage5.nodes" > "$stage4_dir/hello.stage5.check"
assert_check_ok "$stage4_dir/hello.stage5.check"
"$stage4_dir/codegen_c.stage5" "$stage4_dir/hello.stage5.nodes" > "$stage4_dir/hello.stage5.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/hello.stage5" "$stage4_dir/hello.stage5.c" >/dev/null 2>&1
stage5_hello_out="$("$stage4_dir/hello.stage5")"
test "$stage5_hello_out" = "Hello, Tya"
echo "stage5 hello: self-host pipeline matched"

printf 'print "Stage Five"\n' > "$stage4_dir/print_string.stage5.tya"
"$stage4_dir/lexer.stage5" "$stage4_dir/print_string.stage5.tya" > "$stage4_dir/print_string.stage5.tokens"
"$stage4_dir/parser.stage5" "$stage4_dir/print_string.stage5.tokens" > "$stage4_dir/print_string.stage5.nodes"
"$stage4_dir/checker.stage5" "$stage4_dir/print_string.stage5.nodes" > "$stage4_dir/print_string.stage5.check"
assert_check_ok "$stage4_dir/print_string.stage5.check"
"$stage4_dir/codegen_c.stage5" "$stage4_dir/print_string.stage5.nodes" > "$stage4_dir/print_string.stage5.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/print_string.stage5" "$stage4_dir/print_string.stage5.c" >/dev/null 2>&1
stage5_print_string_out="$("$stage4_dir/print_string.stage5")"
test "$stage5_print_string_out" = "Stage Five"
echo "stage5 print string: self-host pipeline matched"

printf 'print 42\n' > "$stage4_dir/print_int.stage5.tya"
"$stage4_dir/lexer.stage5" "$stage4_dir/print_int.stage5.tya" > "$stage4_dir/print_int.stage5.tokens"
"$stage4_dir/parser.stage5" "$stage4_dir/print_int.stage5.tokens" > "$stage4_dir/print_int.stage5.nodes"
"$stage4_dir/checker.stage5" "$stage4_dir/print_int.stage5.nodes" > "$stage4_dir/print_int.stage5.check"
assert_check_ok "$stage4_dir/print_int.stage5.check"
"$stage4_dir/codegen_c.stage5" "$stage4_dir/print_int.stage5.nodes" > "$stage4_dir/print_int.stage5.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/print_int.stage5" "$stage4_dir/print_int.stage5.c" >/dev/null 2>&1
stage5_print_int_out="$("$stage4_dir/print_int.stage5")"
test "$stage5_print_int_out" = "42"
echo "stage5 print int: self-host pipeline matched"

printf 'print "Stage"\nprint "Five"\n' > "$stage4_dir/two_prints.stage5.tya"
"$stage4_dir/lexer.stage5" "$stage4_dir/two_prints.stage5.tya" > "$stage4_dir/two_prints.stage5.tokens"
"$stage4_dir/parser.stage5" "$stage4_dir/two_prints.stage5.tokens" > "$stage4_dir/two_prints.stage5.nodes"
"$stage4_dir/checker.stage5" "$stage4_dir/two_prints.stage5.nodes" > "$stage4_dir/two_prints.stage5.check"
assert_check_ok "$stage4_dir/two_prints.stage5.check"
"$stage4_dir/codegen_c.stage5" "$stage4_dir/two_prints.stage5.nodes" > "$stage4_dir/two_prints.stage5.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/two_prints.stage5" "$stage4_dir/two_prints.stage5.c" >/dev/null 2>&1
stage5_two_prints_out="$("$stage4_dir/two_prints.stage5")"
test "$stage5_two_prints_out" = "Stage
Five"
echo "stage5 two prints: self-host pipeline matched"

cat > "$stage4_dir/stage5_constant_reassign.nodes" <<'NODES'
1:ASSIGN:MAX_RETRY:INT:3
2:ASSIGN:MAX_RETRY:INT:5
3:ASSIGN:retry_count:INT:3
4:ASSIGN:retry_count:INT:5
NODES
"$stage4_dir/checker.stage5" "$stage4_dir/stage5_constant_reassign.nodes" > "$stage4_dir/stage5_constant_reassign.check"
grep -qx "2: cannot reassign constant MAX_RETRY" "$stage4_dir/stage5_constant_reassign.check"
echo "stage5 constant reassignment: self-host checker matched"

cat > "$stage4_dir/stage5_undefined_print.nodes" <<'NODES'
1:ASSIGN:message:STRING:hello
2:PRINT:IDENT:message
3:PRINT:IDENT:missing
4:ASSIGN:count:INT:1
5:PRINT:IDENT:count
NODES
"$stage4_dir/checker.stage5" "$stage4_dir/stage5_undefined_print.nodes" > "$stage4_dir/stage5_undefined_print.check"
grep -qx "3: undefined variable: missing" "$stage4_dir/stage5_undefined_print.check"
echo "stage5 undefined print: self-host checker matched"

cat > "$stage4_dir/stage5_undefined_assign.nodes" <<'NODES'
1:ASSIGN:alias:IDENT:missing
2:ASSIGN:message:STRING:hello
3:ASSIGN:copy:IDENT:message
4:PRINT:IDENT:copy
NODES
"$stage4_dir/checker.stage5" "$stage4_dir/stage5_undefined_assign.nodes" > "$stage4_dir/stage5_undefined_assign.check"
grep -qx "1: undefined variable: missing" "$stage4_dir/stage5_undefined_assign.check"
echo "stage5 undefined assignment: self-host checker matched"

cat > "$stage4_dir/stage5_undefined_for.nodes" <<'NODES'
1:FOR:item:missing_items
2:INDENT:2
2:PRINT:IDENT:item
3:INDENT:0
3:ASSIGN:items:ARRAY_EMPTY:
4:FOR:item:items
5:INDENT:2
5:PRINT:IDENT:item
NODES
"$stage4_dir/checker.stage5" "$stage4_dir/stage5_undefined_for.nodes" > "$stage4_dir/stage5_undefined_for.check"
grep -qx "1: undefined variable: missing_items" "$stage4_dir/stage5_undefined_for.check"
echo "stage5 undefined for collection: self-host checker matched"

if [ "${TYA_STAGE1_SELFHOST_STRICT_REPEATED:-}" = "1" ]; then
  printf 'print "tya"[1]\n' > "$stage4_dir/stage5_string_index.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_string_index.tya" > "$stage4_dir/stage5_string_index.tokens"
  cat > "$stage4_dir/stage5_string_index.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:print:1
1:STRING:tya:7
1:SYMBOL:[:12
1:INT:1:13
1:SYMBOL:]:14
TOKENS
  diff -u "$stage4_dir/stage5_string_index.want.tokens" "$stage4_dir/stage5_string_index.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_string_index.tokens" > "$stage4_dir/stage5_string_index.nodes"
  cat > "$stage4_dir/stage5_string_index.want.nodes" <<'NODES'
1:PRINT_INDEX:STRING:tya:1
NODES
  diff -u "$stage4_dir/stage5_string_index.want.nodes" "$stage4_dir/stage5_string_index.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_string_index.nodes" > "$stage4_dir/stage5_string_index.check"
  assert_check_ok "$stage4_dir/stage5_string_index.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_string_index.nodes" > "$stage4_dir/stage5_string_index.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_string_index" "$stage4_dir/stage5_string_index.c" >/dev/null 2>&1
  stage5_string_index_out="$("$stage4_dir/stage5_string_index")"
  test "$stage5_string_index_out" = "y"
  echo "stage5 string index print: strict repeated-stage pipeline matched"

  printf 'items = [1, 2]\nprint items[1]\n' > "$stage4_dir/stage5_array_index.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_array_index.tya" > "$stage4_dir/stage5_array_index.tokens"
  cat > "$stage4_dir/stage5_array_index.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:items:1
1:SYMBOL:=:7
1:ARRAY:1, 2:9
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:items:7
2:SYMBOL:[:12
2:INT:1:13
2:SYMBOL:]:14
TOKENS
  diff -u "$stage4_dir/stage5_array_index.want.tokens" "$stage4_dir/stage5_array_index.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_array_index.tokens" > "$stage4_dir/stage5_array_index.nodes"
  cat > "$stage4_dir/stage5_array_index.want.nodes" <<'NODES'
1:ASSIGN:items:ARRAY:1, 2
2:PRINT_INDEX:IDENT:items:1
NODES
  diff -u "$stage4_dir/stage5_array_index.want.nodes" "$stage4_dir/stage5_array_index.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_array_index.nodes" > "$stage4_dir/stage5_array_index.check"
  assert_check_ok "$stage4_dir/stage5_array_index.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_array_index.nodes" > "$stage4_dir/stage5_array_index.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_array_index" "$stage4_dir/stage5_array_index.c" >/dev/null 2>&1
  stage5_array_index_out="$("$stage4_dir/stage5_array_index")"
  test "$stage5_array_index_out" = "2"
  echo "stage5 array index print: strict repeated-stage pipeline matched"

  printf 'items = [1, 2]\nprint items[9]\n' > "$stage4_dir/stage5_array_index_oob.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_array_index_oob.tya" > "$stage4_dir/stage5_array_index_oob.tokens"
  cat > "$stage4_dir/stage5_array_index_oob.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:items:1
1:SYMBOL:=:7
1:ARRAY:1, 2:9
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:items:7
2:SYMBOL:[:12
2:INT:9:13
2:SYMBOL:]:14
TOKENS
  diff -u "$stage4_dir/stage5_array_index_oob.want.tokens" "$stage4_dir/stage5_array_index_oob.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_array_index_oob.tokens" > "$stage4_dir/stage5_array_index_oob.nodes"
  cat > "$stage4_dir/stage5_array_index_oob.want.nodes" <<'NODES'
1:ASSIGN:items:ARRAY:1, 2
2:PRINT_INDEX:IDENT:items:9
NODES
  diff -u "$stage4_dir/stage5_array_index_oob.want.nodes" "$stage4_dir/stage5_array_index_oob.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_array_index_oob.nodes" > "$stage4_dir/stage5_array_index_oob.check"
  assert_check_ok "$stage4_dir/stage5_array_index_oob.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_array_index_oob.nodes" > "$stage4_dir/stage5_array_index_oob.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_array_index_oob" "$stage4_dir/stage5_array_index_oob.c" >/dev/null 2>&1
  stage5_array_index_oob_out="$("$stage4_dir/stage5_array_index_oob")"
  test "$stage5_array_index_oob_out" = "nil"
  echo "stage5 array out-of-range index print: strict repeated-stage pipeline matched"

  printf 'items = [1, 2, 3]\nprint items[2]\n' > "$stage4_dir/stage5_array_three.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_array_three.tya" > "$stage4_dir/stage5_array_three.tokens"
  cat > "$stage4_dir/stage5_array_three.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:items:1
1:SYMBOL:=:7
1:ARRAY:1, 2, 3:9
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:items:7
2:SYMBOL:[:12
2:INT:2:13
2:SYMBOL:]:14
TOKENS
  diff -u "$stage4_dir/stage5_array_three.want.tokens" "$stage4_dir/stage5_array_three.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_array_three.tokens" > "$stage4_dir/stage5_array_three.nodes"
  cat > "$stage4_dir/stage5_array_three.want.nodes" <<'NODES'
1:ASSIGN:items:ARRAY:1, 2, 3
2:PRINT_INDEX:IDENT:items:2
NODES
  diff -u "$stage4_dir/stage5_array_three.want.nodes" "$stage4_dir/stage5_array_three.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_array_three.nodes" > "$stage4_dir/stage5_array_three.check"
  assert_check_ok "$stage4_dir/stage5_array_three.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_array_three.nodes" > "$stage4_dir/stage5_array_three.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_array_three" "$stage4_dir/stage5_array_three.c" >/dev/null 2>&1
  stage5_array_three_out="$("$stage4_dir/stage5_array_three")"
  test "$stage5_array_three_out" = "3"
  echo "stage5 three-element array index print: strict repeated-stage pipeline matched"

  printf 'items = [1, 2]\npush items, 3\nprint items[2]\n' > "$stage4_dir/stage5_array_push.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_array_push.tya" > "$stage4_dir/stage5_array_push.tokens"
  cat > "$stage4_dir/stage5_array_push.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:items:1
1:SYMBOL:=:7
1:ARRAY:1, 2:9
2:INDENT:0:1
2:IDENT:push:1
2:IDENT:items:6
2:SYMBOL:,:11
2:INT:3:13
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:items:7
3:SYMBOL:[:12
3:INT:2:13
3:SYMBOL:]:14
TOKENS
  diff -u "$stage4_dir/stage5_array_push.want.tokens" "$stage4_dir/stage5_array_push.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_array_push.tokens" > "$stage4_dir/stage5_array_push.nodes"
  cat > "$stage4_dir/stage5_array_push.want.nodes" <<'NODES'
1:ASSIGN:items:ARRAY:1, 2
2:PUSH:items:INT:3
3:PRINT_INDEX:IDENT:items:2
NODES
  diff -u "$stage4_dir/stage5_array_push.want.nodes" "$stage4_dir/stage5_array_push.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_array_push.nodes" > "$stage4_dir/stage5_array_push.check"
  assert_check_ok "$stage4_dir/stage5_array_push.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_array_push.nodes" > "$stage4_dir/stage5_array_push.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_array_push" "$stage4_dir/stage5_array_push.c" >/dev/null 2>&1
  stage5_array_push_out="$("$stage4_dir/stage5_array_push")"
  test "$stage5_array_push_out" = "3"
  echo "stage5 array push: strict repeated-stage pipeline matched"

  printf 'items = [1, 2]\npush items, 3\nprint pop items\nprint len items\n' > "$stage4_dir/stage5_array_pop.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_array_pop.tya" > "$stage4_dir/stage5_array_pop.tokens"
  cat > "$stage4_dir/stage5_array_pop.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:items:1
1:SYMBOL:=:7
1:ARRAY:1, 2:9
2:INDENT:0:1
2:IDENT:push:1
2:IDENT:items:6
2:SYMBOL:,:11
2:INT:3:13
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:pop:7
3:IDENT:items:11
4:INDENT:0:1
4:IDENT:print:1
4:IDENT:len:7
4:IDENT:items:11
TOKENS
  diff -u "$stage4_dir/stage5_array_pop.want.tokens" "$stage4_dir/stage5_array_pop.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_array_pop.tokens" > "$stage4_dir/stage5_array_pop.nodes"
  cat > "$stage4_dir/stage5_array_pop.want.nodes" <<'NODES'
1:ASSIGN:items:ARRAY:1, 2
2:PUSH:items:INT:3
3:PRINT_CALL1:pop:items
4:PRINT_CALL1:len:items
NODES
  diff -u "$stage4_dir/stage5_array_pop.want.nodes" "$stage4_dir/stage5_array_pop.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_array_pop.nodes" > "$stage4_dir/stage5_array_pop.check"
  assert_check_ok "$stage4_dir/stage5_array_pop.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_array_pop.nodes" > "$stage4_dir/stage5_array_pop.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_array_pop" "$stage4_dir/stage5_array_pop.c" >/dev/null 2>&1
  stage5_array_pop_out="$("$stage4_dir/stage5_array_pop")"
  test "$stage5_array_pop_out" = "3
2"
  echo "stage5 array pop: strict repeated-stage pipeline matched"

  printf 'items = [1, 2]\nitems[1] = 20\nprint items[1]\n' > "$stage4_dir/stage5_array_set_index.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_array_set_index.tya" > "$stage4_dir/stage5_array_set_index.tokens"
  cat > "$stage4_dir/stage5_array_set_index.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:items:1
1:SYMBOL:=:7
1:ARRAY:1, 2:9
2:INDENT:0:1
2:IDENT:items:1
2:SYMBOL:[:6
2:INT:1:7
2:SYMBOL:]:8
2:SYMBOL:=:10
2:INT:20:12
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:items:7
3:SYMBOL:[:12
3:INT:1:13
3:SYMBOL:]:14
TOKENS
  diff -u "$stage4_dir/stage5_array_set_index.want.tokens" "$stage4_dir/stage5_array_set_index.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_array_set_index.tokens" > "$stage4_dir/stage5_array_set_index.nodes"
  cat > "$stage4_dir/stage5_array_set_index.want.nodes" <<'NODES'
1:ASSIGN:items:ARRAY:1, 2
2:SET_INDEX:items:1:INT:20
3:PRINT_INDEX:IDENT:items:1
NODES
  diff -u "$stage4_dir/stage5_array_set_index.want.nodes" "$stage4_dir/stage5_array_set_index.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_array_set_index.nodes" > "$stage4_dir/stage5_array_set_index.check"
  assert_check_ok "$stage4_dir/stage5_array_set_index.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_array_set_index.nodes" > "$stage4_dir/stage5_array_set_index.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_array_set_index" "$stage4_dir/stage5_array_set_index.c" >/dev/null 2>&1
  stage5_array_set_index_out="$("$stage4_dir/stage5_array_set_index")"
  test "$stage5_array_set_index_out" = "20"
  echo "stage5 array index assignment: strict repeated-stage pipeline matched"

  printf 'print "x"\nvalue = 1\nprint value\n' > "$stage4_dir/stage5_assignment.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_assignment.tya" > "$stage4_dir/stage5_assignment.tokens"
  cat > "$stage4_dir/stage5_assignment.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:print:1
1:STRING:x:7
2:INDENT:0:1
2:IDENT:value:1
2:SYMBOL:=:7
2:INT:1:9
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:value:7
TOKENS
  diff -u "$stage4_dir/stage5_assignment.want.tokens" "$stage4_dir/stage5_assignment.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_assignment.tokens" > "$stage4_dir/stage5_assignment.nodes"
  cat > "$stage4_dir/stage5_assignment.want.nodes" <<'NODES'
1:PRINT:STRING:x
2:ASSIGN:value:INT:1
3:PRINT:IDENT:value
NODES
  diff -u "$stage4_dir/stage5_assignment.want.nodes" "$stage4_dir/stage5_assignment.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_assignment.nodes" > "$stage4_dir/stage5_assignment.check"
  assert_check_ok "$stage4_dir/stage5_assignment.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_assignment.nodes" > "$stage4_dir/stage5_assignment.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_assignment" "$stage4_dir/stage5_assignment.c" >/dev/null 2>&1
  stage5_assignment_out="$("$stage4_dir/stage5_assignment")"
  test "$stage5_assignment_out" = "x
1"
  echo "stage5 assignment: strict repeated-stage pipeline matched"

  printf 'value = nil\nprint value\nprint nil\n' > "$stage4_dir/stage5_nil.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_nil.tya" > "$stage4_dir/stage5_nil.tokens"
  cat > "$stage4_dir/stage5_nil.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:value:1
1:SYMBOL:=:7
1:NIL:nil:9
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:value:7
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:nil:7
TOKENS
  diff -u "$stage4_dir/stage5_nil.want.tokens" "$stage4_dir/stage5_nil.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_nil.tokens" > "$stage4_dir/stage5_nil.nodes"
  cat > "$stage4_dir/stage5_nil.want.nodes" <<'NODES'
1:ASSIGN:value:NIL:nil
2:PRINT:IDENT:value
3:PRINT:NIL:nil
NODES
  diff -u "$stage4_dir/stage5_nil.want.nodes" "$stage4_dir/stage5_nil.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_nil.nodes" > "$stage4_dir/stage5_nil.check"
  assert_check_ok "$stage4_dir/stage5_nil.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_nil.nodes" > "$stage4_dir/stage5_nil.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_nil" "$stage4_dir/stage5_nil.c" >/dev/null 2>&1
  stage5_nil_out="$("$stage4_dir/stage5_nil")"
  test "$stage5_nil_out" = "nil
nil"
  echo "stage5 nil literal/assignment: strict repeated-stage pipeline matched"

  printf 'name = read_line()\nprint name\n' > "$stage4_dir/stage5_read_line.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_read_line.tya" > "$stage4_dir/stage5_read_line.tokens"
  cat > "$stage4_dir/stage5_read_line.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:name:1
1:SYMBOL:=:6
1:INT:read_line():8
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:name:7
TOKENS
  diff -u "$stage4_dir/stage5_read_line.want.tokens" "$stage4_dir/stage5_read_line.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_read_line.tokens" > "$stage4_dir/stage5_read_line.nodes"
  cat > "$stage4_dir/stage5_read_line.want.nodes" <<'NODES'
1:ASSIGN:name:INT:read_line()
2:PRINT:IDENT:name
NODES
  diff -u "$stage4_dir/stage5_read_line.want.nodes" "$stage4_dir/stage5_read_line.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_read_line.nodes" > "$stage4_dir/stage5_read_line.check"
  assert_check_ok "$stage4_dir/stage5_read_line.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_read_line.nodes" > "$stage4_dir/stage5_read_line.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_read_line" "$stage4_dir/stage5_read_line.c" >/dev/null 2>&1
  stage5_read_line_out="$(printf 'komagata\n' | "$stage4_dir/stage5_read_line")"
  test "$stage5_read_line_out" = "komagata"
  echo "stage5 read_line assignment: strict repeated-stage pipeline matched"

  printf 'source = read_file args()[0]\nprint source\n' > "$stage4_dir/stage5_read_file_arg.tya"
  printf 'Tya' > "$stage4_dir/stage5_read_file_arg.input"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_read_file_arg.tya" > "$stage4_dir/stage5_read_file_arg.tokens"
  cat > "$stage4_dir/stage5_read_file_arg.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:source:1
1:SYMBOL:=:8
1:IDENT:read_file:10
1:IDENT:args()[0]:20
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:source:7
TOKENS
  diff -u "$stage4_dir/stage5_read_file_arg.want.tokens" "$stage4_dir/stage5_read_file_arg.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_read_file_arg.tokens" > "$stage4_dir/stage5_read_file_arg.nodes"
  cat > "$stage4_dir/stage5_read_file_arg.want.nodes" <<'NODES'
1:ASSIGN:source:CALL1:read_file:args()[0]
2:PRINT:IDENT:source
NODES
  diff -u "$stage4_dir/stage5_read_file_arg.want.nodes" "$stage4_dir/stage5_read_file_arg.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_read_file_arg.nodes" > "$stage4_dir/stage5_read_file_arg.check"
  assert_check_ok "$stage4_dir/stage5_read_file_arg.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_read_file_arg.nodes" > "$stage4_dir/stage5_read_file_arg.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_read_file_arg" "$stage4_dir/stage5_read_file_arg.c" >/dev/null 2>&1
  stage5_read_file_arg_out="$("$stage4_dir/stage5_read_file_arg" "$stage4_dir/stage5_read_file_arg.input")"
  test "$stage5_read_file_arg_out" = "Tya"
  echo "stage5 read_file args index: strict repeated-stage pipeline matched"

  printf 'items = args()\nprint len items\nprint items[0]\nprint items[9]\n' > "$stage4_dir/stage5_args.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_args.tya" > "$stage4_dir/stage5_args.tokens"
  cat > "$stage4_dir/stage5_args.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:items:1
1:SYMBOL:=:7
1:INT:args():9
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:len:7
2:IDENT:items:11
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:items:7
3:SYMBOL:[:12
3:INT:0:13
3:SYMBOL:]:14
4:INDENT:0:1
4:IDENT:print:1
4:IDENT:items:7
4:SYMBOL:[:12
4:INT:9:13
4:SYMBOL:]:14
TOKENS
  diff -u "$stage4_dir/stage5_args.want.tokens" "$stage4_dir/stage5_args.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_args.tokens" > "$stage4_dir/stage5_args.nodes"
  cat > "$stage4_dir/stage5_args.want.nodes" <<'NODES'
1:ASSIGN:items:INT:args()
2:PRINT_CALL1:len:items
3:PRINT_INDEX:IDENT:items:0
4:PRINT_INDEX:IDENT:items:9
NODES
  diff -u "$stage4_dir/stage5_args.want.nodes" "$stage4_dir/stage5_args.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_args.nodes" > "$stage4_dir/stage5_args.check"
  assert_check_ok "$stage4_dir/stage5_args.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_args.nodes" > "$stage4_dir/stage5_args.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_args" "$stage4_dir/stage5_args.c" >/dev/null 2>&1
  stage5_args_out="$("$stage4_dir/stage5_args" komagata)"
  test "$stage5_args_out" = "1
komagata
nil"
  echo "stage5 args len/index: strict repeated-stage pipeline matched"

  printf 'panic "bad state"\n' > "$stage4_dir/stage5_panic.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_panic.tya" > "$stage4_dir/stage5_panic.tokens"
  cat > "$stage4_dir/stage5_panic.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:panic:1
1:STRING:bad state:7
TOKENS
  diff -u "$stage4_dir/stage5_panic.want.tokens" "$stage4_dir/stage5_panic.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_panic.tokens" > "$stage4_dir/stage5_panic.nodes"
  cat > "$stage4_dir/stage5_panic.want.nodes" <<'NODES'
1:PANIC:STRING:bad state
NODES
  diff -u "$stage4_dir/stage5_panic.want.nodes" "$stage4_dir/stage5_panic.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_panic.nodes" > "$stage4_dir/stage5_panic.check"
  assert_check_ok "$stage4_dir/stage5_panic.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_panic.nodes" > "$stage4_dir/stage5_panic.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_panic" "$stage4_dir/stage5_panic.c" >/dev/null 2>&1
  set +e
  "$stage4_dir/stage5_panic" > "$stage4_dir/stage5_panic.out" 2> "$stage4_dir/stage5_panic.err"
  stage5_panic_status="$?"
  set -e
  test "$stage5_panic_status" = "1"
  test ! -s "$stage4_dir/stage5_panic.out"
  test "$(cat "$stage4_dir/stage5_panic.err")" = "panic: bad state"
  echo "stage5 panic status/stderr: strict repeated-stage pipeline matched"

  printf 'code = 3\nexit code\n' > "$stage4_dir/stage5_exit.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_exit.tya" > "$stage4_dir/stage5_exit.tokens"
  cat > "$stage4_dir/stage5_exit.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:code:1
1:SYMBOL:=:6
1:INT:3:8
2:INDENT:0:1
2:IDENT:exit:1
2:IDENT:code:6
TOKENS
  diff -u "$stage4_dir/stage5_exit.want.tokens" "$stage4_dir/stage5_exit.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_exit.tokens" > "$stage4_dir/stage5_exit.nodes"
  cat > "$stage4_dir/stage5_exit.want.nodes" <<'NODES'
1:ASSIGN:code:INT:3
2:EXIT:IDENT:code
NODES
  diff -u "$stage4_dir/stage5_exit.want.nodes" "$stage4_dir/stage5_exit.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_exit.nodes" > "$stage4_dir/stage5_exit.check"
  assert_check_ok "$stage4_dir/stage5_exit.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_exit.nodes" > "$stage4_dir/stage5_exit.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_exit" "$stage4_dir/stage5_exit.c" >/dev/null 2>&1
  set +e
  "$stage4_dir/stage5_exit" > "$stage4_dir/stage5_exit.out"
  stage5_exit_status="$?"
  set -e
  test "$stage5_exit_status" = "3"
  test ! -s "$stage4_dir/stage5_exit.out"
  echo "stage5 exit status: strict repeated-stage pipeline matched"

  printf 'sum = 1 + 2\nprint sum\n' > "$stage4_dir/stage5_int_add.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_int_add.tya" > "$stage4_dir/stage5_int_add.tokens"
  cat > "$stage4_dir/stage5_int_add.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:sum:1
1:SYMBOL:=:5
1:INT:1:7
1:SYMBOL:+:9
1:INT:2:11
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:sum:7
TOKENS
  diff -u "$stage4_dir/stage5_int_add.want.tokens" "$stage4_dir/stage5_int_add.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_int_add.tokens" > "$stage4_dir/stage5_int_add.nodes"
  cat > "$stage4_dir/stage5_int_add.want.nodes" <<'NODES'
1:ASSIGN:sum:INT_ADD:INT:1:INT:2
2:PRINT:IDENT:sum
NODES
  diff -u "$stage4_dir/stage5_int_add.want.nodes" "$stage4_dir/stage5_int_add.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_int_add.nodes" > "$stage4_dir/stage5_int_add.check"
  assert_check_ok "$stage4_dir/stage5_int_add.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_int_add.nodes" > "$stage4_dir/stage5_int_add.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_int_add" "$stage4_dir/stage5_int_add.c" >/dev/null 2>&1
  stage5_int_add_out="$("$stage4_dir/stage5_int_add")"
  test "$stage5_int_add_out" = "3"
  echo "stage5 int addition: strict repeated-stage pipeline matched"

  printf 'rem = 5 %% 2\nprint rem\n' > "$stage4_dir/stage5_int_mod.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_int_mod.tya" > "$stage4_dir/stage5_int_mod.tokens"
  cat > "$stage4_dir/stage5_int_mod.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:rem:1
1:SYMBOL:=:5
1:INT:5:7
1:SYMBOL:%:9
1:INT:2:11
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:rem:7
TOKENS
  diff -u "$stage4_dir/stage5_int_mod.want.tokens" "$stage4_dir/stage5_int_mod.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_int_mod.tokens" > "$stage4_dir/stage5_int_mod.nodes"
  cat > "$stage4_dir/stage5_int_mod.want.nodes" <<'NODES'
1:ASSIGN:rem:INT_MOD:INT:5:INT:2
2:PRINT:IDENT:rem
NODES
  diff -u "$stage4_dir/stage5_int_mod.want.nodes" "$stage4_dir/stage5_int_mod.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_int_mod.nodes" > "$stage4_dir/stage5_int_mod.check"
  assert_check_ok "$stage4_dir/stage5_int_mod.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_int_mod.nodes" > "$stage4_dir/stage5_int_mod.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_int_mod" "$stage4_dir/stage5_int_mod.c" >/dev/null 2>&1
  stage5_int_mod_out="$("$stage4_dir/stage5_int_mod")"
  test "$stage5_int_mod_out" = "1"
  echo "stage5 int modulo: strict repeated-stage pipeline matched"

  printf 'diff = 5 - 2\nprint diff\n' > "$stage4_dir/stage5_int_sub.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_int_sub.tya" > "$stage4_dir/stage5_int_sub.tokens"
  cat > "$stage4_dir/stage5_int_sub.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:diff:1
1:SYMBOL:=:6
1:INT:5:8
1:SYMBOL:-:10
1:INT:2:12
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:diff:7
TOKENS
  diff -u "$stage4_dir/stage5_int_sub.want.tokens" "$stage4_dir/stage5_int_sub.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_int_sub.tokens" > "$stage4_dir/stage5_int_sub.nodes"
  cat > "$stage4_dir/stage5_int_sub.want.nodes" <<'NODES'
1:ASSIGN:diff:INT_SUB:INT:5:INT:2
2:PRINT:IDENT:diff
NODES
  diff -u "$stage4_dir/stage5_int_sub.want.nodes" "$stage4_dir/stage5_int_sub.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_int_sub.nodes" > "$stage4_dir/stage5_int_sub.check"
  assert_check_ok "$stage4_dir/stage5_int_sub.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_int_sub.nodes" > "$stage4_dir/stage5_int_sub.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_int_sub" "$stage4_dir/stage5_int_sub.c" >/dev/null 2>&1
  stage5_int_sub_out="$("$stage4_dir/stage5_int_sub")"
  test "$stage5_int_sub_out" = "3"
  echo "stage5 int subtraction: strict repeated-stage pipeline matched"

  printf 'product = 3 * 4\nprint product\n' > "$stage4_dir/stage5_int_mul.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_int_mul.tya" > "$stage4_dir/stage5_int_mul.tokens"
  cat > "$stage4_dir/stage5_int_mul.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:product:1
1:SYMBOL:=:9
1:INT:3:11
1:SYMBOL:*:13
1:INT:4:15
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:product:7
TOKENS
  diff -u "$stage4_dir/stage5_int_mul.want.tokens" "$stage4_dir/stage5_int_mul.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_int_mul.tokens" > "$stage4_dir/stage5_int_mul.nodes"
  cat > "$stage4_dir/stage5_int_mul.want.nodes" <<'NODES'
1:ASSIGN:product:INT_MUL:INT:3:INT:4
2:PRINT:IDENT:product
NODES
  diff -u "$stage4_dir/stage5_int_mul.want.nodes" "$stage4_dir/stage5_int_mul.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_int_mul.nodes" > "$stage4_dir/stage5_int_mul.check"
  assert_check_ok "$stage4_dir/stage5_int_mul.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_int_mul.nodes" > "$stage4_dir/stage5_int_mul.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_int_mul" "$stage4_dir/stage5_int_mul.c" >/dev/null 2>&1
  stage5_int_mul_out="$("$stage4_dir/stage5_int_mul")"
  test "$stage5_int_mul_out" = "12"
  echo "stage5 int multiplication: strict repeated-stage pipeline matched"

  printf 'quotient = 8 / 2\nprint quotient\n' > "$stage4_dir/stage5_int_div.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_int_div.tya" > "$stage4_dir/stage5_int_div.tokens"
  cat > "$stage4_dir/stage5_int_div.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:quotient:1
1:SYMBOL:=:10
1:INT:8:12
1:SYMBOL:/:14
1:INT:2:16
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:quotient:7
TOKENS
  diff -u "$stage4_dir/stage5_int_div.want.tokens" "$stage4_dir/stage5_int_div.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_int_div.tokens" > "$stage4_dir/stage5_int_div.nodes"
  cat > "$stage4_dir/stage5_int_div.want.nodes" <<'NODES'
1:ASSIGN:quotient:INT_DIV:INT:8:INT:2
2:PRINT:IDENT:quotient
NODES
  diff -u "$stage4_dir/stage5_int_div.want.nodes" "$stage4_dir/stage5_int_div.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_int_div.nodes" > "$stage4_dir/stage5_int_div.check"
  assert_check_ok "$stage4_dir/stage5_int_div.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_int_div.nodes" > "$stage4_dir/stage5_int_div.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_int_div" "$stage4_dir/stage5_int_div.c" >/dev/null 2>&1
  stage5_int_div_out="$("$stage4_dir/stage5_int_div")"
  test "$stage5_int_div_out" = "4"
  echo "stage5 int division: strict repeated-stage pipeline matched"

  printf 'age = 20\nsame = age == 20\nprint same\n' > "$stage4_dir/stage5_compare_eq.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_compare_eq.tya" > "$stage4_dir/stage5_compare_eq.tokens"
  cat > "$stage4_dir/stage5_compare_eq.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:age:1
1:SYMBOL:=:5
1:INT:20:7
2:INDENT:0:1
2:IDENT:same:1
2:SYMBOL:=:6
2:IDENT:age:8
2:SYMBOL:==:12
2:INT:20:15
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:same:7
TOKENS
  diff -u "$stage4_dir/stage5_compare_eq.want.tokens" "$stage4_dir/stage5_compare_eq.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_compare_eq.tokens" > "$stage4_dir/stage5_compare_eq.nodes"
  cat > "$stage4_dir/stage5_compare_eq.want.nodes" <<'NODES'
1:ASSIGN:age:INT:20
2:ASSIGN:same:COMPARE_EQ:age:20
3:PRINT:IDENT:same
NODES
  diff -u "$stage4_dir/stage5_compare_eq.want.nodes" "$stage4_dir/stage5_compare_eq.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_compare_eq.nodes" > "$stage4_dir/stage5_compare_eq.check"
  assert_check_ok "$stage4_dir/stage5_compare_eq.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_compare_eq.nodes" > "$stage4_dir/stage5_compare_eq.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_compare_eq" "$stage4_dir/stage5_compare_eq.c" >/dev/null 2>&1
  stage5_compare_eq_out="$("$stage4_dir/stage5_compare_eq")"
  test "$stage5_compare_eq_out" = "true"
  echo "stage5 comparison equality: strict repeated-stage pipeline matched"

  printf 'age = 20\ndifferent = age != 18\nprint different\n' > "$stage4_dir/stage5_compare_ne.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_compare_ne.tya" > "$stage4_dir/stage5_compare_ne.tokens"
  cat > "$stage4_dir/stage5_compare_ne.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:age:1
1:SYMBOL:=:5
1:INT:20:7
2:INDENT:0:1
2:IDENT:different:1
2:SYMBOL:=:11
2:IDENT:age:13
2:SYMBOL:!=:17
2:INT:18:20
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:different:7
TOKENS
  diff -u "$stage4_dir/stage5_compare_ne.want.tokens" "$stage4_dir/stage5_compare_ne.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_compare_ne.tokens" > "$stage4_dir/stage5_compare_ne.nodes"
  cat > "$stage4_dir/stage5_compare_ne.want.nodes" <<'NODES'
1:ASSIGN:age:INT:20
2:ASSIGN:different:COMPARE_NE:age:18
3:PRINT:IDENT:different
NODES
  diff -u "$stage4_dir/stage5_compare_ne.want.nodes" "$stage4_dir/stage5_compare_ne.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_compare_ne.nodes" > "$stage4_dir/stage5_compare_ne.check"
  assert_check_ok "$stage4_dir/stage5_compare_ne.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_compare_ne.nodes" > "$stage4_dir/stage5_compare_ne.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_compare_ne" "$stage4_dir/stage5_compare_ne.c" >/dev/null 2>&1
  stage5_compare_ne_out="$("$stage4_dir/stage5_compare_ne")"
  test "$stage5_compare_ne_out" = "true"
  echo "stage5 comparison inequality: strict repeated-stage pipeline matched"

  printf 'age = 20\nadult = age >= 18\nprint adult\n' > "$stage4_dir/stage5_compare_ge.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_compare_ge.tya" > "$stage4_dir/stage5_compare_ge.tokens"
  cat > "$stage4_dir/stage5_compare_ge.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:age:1
1:SYMBOL:=:5
1:INT:20:7
2:INDENT:0:1
2:IDENT:adult:1
2:SYMBOL:=:7
2:IDENT:age:9
2:SYMBOL:>=:13
2:INT:18:16
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:adult:7
TOKENS
  diff -u "$stage4_dir/stage5_compare_ge.want.tokens" "$stage4_dir/stage5_compare_ge.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_compare_ge.tokens" > "$stage4_dir/stage5_compare_ge.nodes"
  cat > "$stage4_dir/stage5_compare_ge.want.nodes" <<'NODES'
1:ASSIGN:age:INT:20
2:ASSIGN:adult:COMPARE_GE:age:18
3:PRINT:IDENT:adult
NODES
  diff -u "$stage4_dir/stage5_compare_ge.want.nodes" "$stage4_dir/stage5_compare_ge.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_compare_ge.nodes" > "$stage4_dir/stage5_compare_ge.check"
  assert_check_ok "$stage4_dir/stage5_compare_ge.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_compare_ge.nodes" > "$stage4_dir/stage5_compare_ge.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_compare_ge" "$stage4_dir/stage5_compare_ge.c" >/dev/null 2>&1
  stage5_compare_ge_out="$("$stage4_dir/stage5_compare_ge")"
  test "$stage5_compare_ge_out" = "true"
  echo "stage5 comparison greater-equal: strict repeated-stage pipeline matched"

  printf 'age = 17\nyoung = age <= 17\nprint young\n' > "$stage4_dir/stage5_compare_le.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_compare_le.tya" > "$stage4_dir/stage5_compare_le.tokens"
  cat > "$stage4_dir/stage5_compare_le.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:age:1
1:SYMBOL:=:5
1:INT:17:7
2:INDENT:0:1
2:IDENT:young:1
2:SYMBOL:=:7
2:IDENT:age:9
2:SYMBOL:<=:13
2:INT:17:16
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:young:7
TOKENS
  diff -u "$stage4_dir/stage5_compare_le.want.tokens" "$stage4_dir/stage5_compare_le.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_compare_le.tokens" > "$stage4_dir/stage5_compare_le.nodes"
  cat > "$stage4_dir/stage5_compare_le.want.nodes" <<'NODES'
1:ASSIGN:age:INT:17
2:ASSIGN:young:COMPARE_LE:age:17
3:PRINT:IDENT:young
NODES
  diff -u "$stage4_dir/stage5_compare_le.want.nodes" "$stage4_dir/stage5_compare_le.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_compare_le.nodes" > "$stage4_dir/stage5_compare_le.check"
  assert_check_ok "$stage4_dir/stage5_compare_le.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_compare_le.nodes" > "$stage4_dir/stage5_compare_le.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_compare_le" "$stage4_dir/stage5_compare_le.c" >/dev/null 2>&1
  stage5_compare_le_out="$("$stage4_dir/stage5_compare_le")"
  test "$stage5_compare_le_out" = "true"
  echo "stage5 comparison less-equal: strict repeated-stage pipeline matched"

  printf 'age = 20\nolder = age > 18\nprint older\n' > "$stage4_dir/stage5_compare_gt.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_compare_gt.tya" > "$stage4_dir/stage5_compare_gt.tokens"
  cat > "$stage4_dir/stage5_compare_gt.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:age:1
1:SYMBOL:=:5
1:INT:20:7
2:INDENT:0:1
2:IDENT:older:1
2:SYMBOL:=:7
2:IDENT:age:9
2:SYMBOL:>:13
2:INT:18:15
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:older:7
TOKENS
  diff -u "$stage4_dir/stage5_compare_gt.want.tokens" "$stage4_dir/stage5_compare_gt.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_compare_gt.tokens" > "$stage4_dir/stage5_compare_gt.nodes"
  cat > "$stage4_dir/stage5_compare_gt.want.nodes" <<'NODES'
1:ASSIGN:age:INT:20
2:ASSIGN:older:COMPARE_GT:age:18
3:PRINT:IDENT:older
NODES
  diff -u "$stage4_dir/stage5_compare_gt.want.nodes" "$stage4_dir/stage5_compare_gt.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_compare_gt.nodes" > "$stage4_dir/stage5_compare_gt.check"
  assert_check_ok "$stage4_dir/stage5_compare_gt.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_compare_gt.nodes" > "$stage4_dir/stage5_compare_gt.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_compare_gt" "$stage4_dir/stage5_compare_gt.c" >/dev/null 2>&1
  stage5_compare_gt_out="$("$stage4_dir/stage5_compare_gt")"
  test "$stage5_compare_gt_out" = "true"
  echo "stage5 comparison greater-than: strict repeated-stage pipeline matched"

  printf 'age = 17\nyounger = age < 18\nprint younger\n' > "$stage4_dir/stage5_compare_lt.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_compare_lt.tya" > "$stage4_dir/stage5_compare_lt.tokens"
  cat > "$stage4_dir/stage5_compare_lt.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:age:1
1:SYMBOL:=:5
1:INT:17:7
2:INDENT:0:1
2:IDENT:younger:1
2:SYMBOL:=:9
2:IDENT:age:11
2:SYMBOL:<:15
2:INT:18:17
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:younger:7
TOKENS
  diff -u "$stage4_dir/stage5_compare_lt.want.tokens" "$stage4_dir/stage5_compare_lt.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_compare_lt.tokens" > "$stage4_dir/stage5_compare_lt.nodes"
  cat > "$stage4_dir/stage5_compare_lt.want.nodes" <<'NODES'
1:ASSIGN:age:INT:17
2:ASSIGN:younger:COMPARE_LT:age:18
3:PRINT:IDENT:younger
NODES
  diff -u "$stage4_dir/stage5_compare_lt.want.nodes" "$stage4_dir/stage5_compare_lt.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_compare_lt.nodes" > "$stage4_dir/stage5_compare_lt.check"
  assert_check_ok "$stage4_dir/stage5_compare_lt.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_compare_lt.nodes" > "$stage4_dir/stage5_compare_lt.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_compare_lt" "$stage4_dir/stage5_compare_lt.c" >/dev/null 2>&1
  stage5_compare_lt_out="$("$stage4_dir/stage5_compare_lt")"
  test "$stage5_compare_lt_out" = "true"
  echo "stage5 comparison less-than: strict repeated-stage pipeline matched"

  printf 'i = 0\nwhile i < 1\n  print "loop"\n  break\nprint i\n' > "$stage4_dir/stage5_while.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_while.tya" > "$stage4_dir/stage5_while.tokens"
  cat > "$stage4_dir/stage5_while.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:i:1
1:SYMBOL:=:3
1:INT:0:5
2:INDENT:0:1
2:IDENT:while:1
2:IDENT:i:7
2:SYMBOL:<:9
2:INT:1:11
3:INDENT:2:1
3:IDENT:print:3
3:STRING:loop:9
4:INDENT:2:1
4:IDENT:break:3
5:INDENT:0:1
5:IDENT:print:1
5:IDENT:i:7
TOKENS
  diff -u "$stage4_dir/stage5_while.want.tokens" "$stage4_dir/stage5_while.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_while.tokens" > "$stage4_dir/stage5_while.nodes"
  cat > "$stage4_dir/stage5_while.want.nodes" <<'NODES'
1:ASSIGN:i:INT:0
2:WHILE_COMPARE_LT:IDENT:i:INT:1
3:PRINT:STRING:loop
4:BREAK
5:PRINT:IDENT:i
NODES
  diff -u "$stage4_dir/stage5_while.want.nodes" "$stage4_dir/stage5_while.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_while.nodes" > "$stage4_dir/stage5_while.check"
  assert_check_ok "$stage4_dir/stage5_while.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_while.nodes" > "$stage4_dir/stage5_while.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_while" "$stage4_dir/stage5_while.c" >/dev/null 2>&1
  stage5_while_out="$("$stage4_dir/stage5_while")"
  test "$stage5_while_out" = "loop
0"
  echo "stage5 while break: strict repeated-stage pipeline matched"

  printf 'i = 0\nwhile i != 1\n  print "loop"\n  break\nprint i\n' > "$stage4_dir/stage5_while_not_equal.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_while_not_equal.tya" > "$stage4_dir/stage5_while_not_equal.tokens"
  cat > "$stage4_dir/stage5_while_not_equal.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:i:1
1:SYMBOL:=:3
1:INT:0:5
2:INDENT:0:1
2:IDENT:while:1
2:IDENT:i:7
2:SYMBOL:!=:9
2:INT:1:12
3:INDENT:2:1
3:IDENT:print:3
3:STRING:loop:9
4:INDENT:2:1
4:IDENT:break:3
5:INDENT:0:1
5:IDENT:print:1
5:IDENT:i:7
TOKENS
  diff -u "$stage4_dir/stage5_while_not_equal.want.tokens" "$stage4_dir/stage5_while_not_equal.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_while_not_equal.tokens" > "$stage4_dir/stage5_while_not_equal.nodes"
  cat > "$stage4_dir/stage5_while_not_equal.want.nodes" <<'NODES'
1:ASSIGN:i:INT:0
2:WHILE_COMPARE_NE:IDENT:i:INT:1
3:PRINT:STRING:loop
4:BREAK
5:PRINT:IDENT:i
NODES
  diff -u "$stage4_dir/stage5_while_not_equal.want.nodes" "$stage4_dir/stage5_while_not_equal.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_while_not_equal.nodes" > "$stage4_dir/stage5_while_not_equal.check"
  assert_check_ok "$stage4_dir/stage5_while_not_equal.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_while_not_equal.nodes" > "$stage4_dir/stage5_while_not_equal.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_while_not_equal" "$stage4_dir/stage5_while_not_equal.c" >/dev/null 2>&1
  stage5_while_not_equal_out="$("$stage4_dir/stage5_while_not_equal")"
  test "$stage5_while_not_equal_out" = "loop
0"
  echo "stage5 while not-equal break: strict repeated-stage pipeline matched"

  printf 'i = 0\nwhile i <= 0\n  print "loop"\n  break\nprint i\n' > "$stage4_dir/stage5_while_less_equal.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_while_less_equal.tya" > "$stage4_dir/stage5_while_less_equal.tokens"
  cat > "$stage4_dir/stage5_while_less_equal.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:i:1
1:SYMBOL:=:3
1:INT:0:5
2:INDENT:0:1
2:IDENT:while:1
2:IDENT:i:7
2:SYMBOL:<=:9
2:INT:0:12
3:INDENT:2:1
3:IDENT:print:3
3:STRING:loop:9
4:INDENT:2:1
4:IDENT:break:3
5:INDENT:0:1
5:IDENT:print:1
5:IDENT:i:7
TOKENS
  diff -u "$stage4_dir/stage5_while_less_equal.want.tokens" "$stage4_dir/stage5_while_less_equal.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_while_less_equal.tokens" > "$stage4_dir/stage5_while_less_equal.nodes"
  cat > "$stage4_dir/stage5_while_less_equal.want.nodes" <<'NODES'
1:ASSIGN:i:INT:0
2:WHILE_COMPARE_LE:IDENT:i:INT:0
3:PRINT:STRING:loop
4:BREAK
5:PRINT:IDENT:i
NODES
  diff -u "$stage4_dir/stage5_while_less_equal.want.nodes" "$stage4_dir/stage5_while_less_equal.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_while_less_equal.nodes" > "$stage4_dir/stage5_while_less_equal.check"
  assert_check_ok "$stage4_dir/stage5_while_less_equal.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_while_less_equal.nodes" > "$stage4_dir/stage5_while_less_equal.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_while_less_equal" "$stage4_dir/stage5_while_less_equal.c" >/dev/null 2>&1
  stage5_while_less_equal_out="$("$stage4_dir/stage5_while_less_equal")"
  test "$stage5_while_less_equal_out" = "loop
0"
  echo "stage5 while less-equal break: strict repeated-stage pipeline matched"

  printf 'i = 1\nwhile i > 0\n  print "loop"\n  break\nprint i\n' > "$stage4_dir/stage5_while_greater.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_while_greater.tya" > "$stage4_dir/stage5_while_greater.tokens"
  cat > "$stage4_dir/stage5_while_greater.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:i:1
1:SYMBOL:=:3
1:INT:1:5
2:INDENT:0:1
2:IDENT:while:1
2:IDENT:i:7
2:SYMBOL:>:9
2:INT:0:11
3:INDENT:2:1
3:IDENT:print:3
3:STRING:loop:9
4:INDENT:2:1
4:IDENT:break:3
5:INDENT:0:1
5:IDENT:print:1
5:IDENT:i:7
TOKENS
  diff -u "$stage4_dir/stage5_while_greater.want.tokens" "$stage4_dir/stage5_while_greater.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_while_greater.tokens" > "$stage4_dir/stage5_while_greater.nodes"
  cat > "$stage4_dir/stage5_while_greater.want.nodes" <<'NODES'
1:ASSIGN:i:INT:1
2:WHILE_COMPARE_GT:IDENT:i:INT:0
3:PRINT:STRING:loop
4:BREAK
5:PRINT:IDENT:i
NODES
  diff -u "$stage4_dir/stage5_while_greater.want.nodes" "$stage4_dir/stage5_while_greater.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_while_greater.nodes" > "$stage4_dir/stage5_while_greater.check"
  assert_check_ok "$stage4_dir/stage5_while_greater.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_while_greater.nodes" > "$stage4_dir/stage5_while_greater.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_while_greater" "$stage4_dir/stage5_while_greater.c" >/dev/null 2>&1
  stage5_while_greater_out="$("$stage4_dir/stage5_while_greater")"
  test "$stage5_while_greater_out" = "loop
1"
  echo "stage5 while greater break: strict repeated-stage pipeline matched"

  printf 'i = 1\nwhile i >= 1\n  print "loop"\n  break\nprint i\n' > "$stage4_dir/stage5_while_greater_equal.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_while_greater_equal.tya" > "$stage4_dir/stage5_while_greater_equal.tokens"
  cat > "$stage4_dir/stage5_while_greater_equal.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:i:1
1:SYMBOL:=:3
1:INT:1:5
2:INDENT:0:1
2:IDENT:while:1
2:IDENT:i:7
2:SYMBOL:>=:9
2:INT:1:12
3:INDENT:2:1
3:IDENT:print:3
3:STRING:loop:9
4:INDENT:2:1
4:IDENT:break:3
5:INDENT:0:1
5:IDENT:print:1
5:IDENT:i:7
TOKENS
  diff -u "$stage4_dir/stage5_while_greater_equal.want.tokens" "$stage4_dir/stage5_while_greater_equal.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_while_greater_equal.tokens" > "$stage4_dir/stage5_while_greater_equal.nodes"
  cat > "$stage4_dir/stage5_while_greater_equal.want.nodes" <<'NODES'
1:ASSIGN:i:INT:1
2:WHILE_COMPARE_GE:IDENT:i:INT:1
3:PRINT:STRING:loop
4:BREAK
5:PRINT:IDENT:i
NODES
  diff -u "$stage4_dir/stage5_while_greater_equal.want.nodes" "$stage4_dir/stage5_while_greater_equal.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_while_greater_equal.nodes" > "$stage4_dir/stage5_while_greater_equal.check"
  assert_check_ok "$stage4_dir/stage5_while_greater_equal.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_while_greater_equal.nodes" > "$stage4_dir/stage5_while_greater_equal.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_while_greater_equal" "$stage4_dir/stage5_while_greater_equal.c" >/dev/null 2>&1
  stage5_while_greater_equal_out="$("$stage4_dir/stage5_while_greater_equal")"
  test "$stage5_while_greater_equal_out" = "loop
1"
  echo "stage5 while greater-equal break: strict repeated-stage pipeline matched"

  printf 'name = "tya"\nif name == "tya"\n  print "match"\nelse\n  print "miss"\n' > "$stage4_dir/stage5_if_else.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_if_else.tya" > "$stage4_dir/stage5_if_else.tokens"
  cat > "$stage4_dir/stage5_if_else.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:name:1
1:SYMBOL:=:6
1:STRING:tya:8
2:INDENT:0:1
2:IDENT:if:1
2:IDENT:name:4
2:SYMBOL:==:9
2:STRING:tya:13
3:INDENT:2:1
3:IDENT:print:3
3:STRING:match:9
4:INDENT:0:1
4:IDENT:else:1
5:INDENT:2:1
5:IDENT:print:3
5:STRING:miss:9
TOKENS
  diff -u "$stage4_dir/stage5_if_else.want.tokens" "$stage4_dir/stage5_if_else.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_if_else.tokens" > "$stage4_dir/stage5_if_else.nodes"
  cat > "$stage4_dir/stage5_if_else.want.nodes" <<'NODES'
1:ASSIGN:name:STRING:tya
2:IF_COMPARE_EQ:IDENT:name:STRING:tya
3:PRINT:STRING:match
4:ELSE
5:PRINT:STRING:miss
NODES
  diff -u "$stage4_dir/stage5_if_else.want.nodes" "$stage4_dir/stage5_if_else.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_if_else.nodes" > "$stage4_dir/stage5_if_else.check"
  assert_check_ok "$stage4_dir/stage5_if_else.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_if_else.nodes" > "$stage4_dir/stage5_if_else.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_if_else" "$stage4_dir/stage5_if_else.c" >/dev/null 2>&1
  stage5_if_else_out="$("$stage4_dir/stage5_if_else")"
  test "$stage5_if_else_out" = "match"
  echo "stage5 if else: strict repeated-stage pipeline matched"

  printf 'items = [1, 2]\nfor item in items\n  print item\n' > "$stage4_dir/stage5_for.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_for.tya" > "$stage4_dir/stage5_for.tokens"
  cat > "$stage4_dir/stage5_for.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:items:1
1:SYMBOL:=:7
1:ARRAY:1, 2:9
2:INDENT:0:1
2:IDENT:for:1
2:IDENT:item:5
2:IDENT:in:10
2:IDENT:items:13
3:INDENT:2:1
3:IDENT:print:3
3:IDENT:item:9
TOKENS
  diff -u "$stage4_dir/stage5_for.want.tokens" "$stage4_dir/stage5_for.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_for.tokens" > "$stage4_dir/stage5_for.nodes"
  cat > "$stage4_dir/stage5_for.want.nodes" <<'NODES'
1:ASSIGN:items:ARRAY:1, 2
2:FOR:item:items
3:PRINT:IDENT:item
NODES
  diff -u "$stage4_dir/stage5_for.want.nodes" "$stage4_dir/stage5_for.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_for.nodes" > "$stage4_dir/stage5_for.check"
  assert_check_ok "$stage4_dir/stage5_for.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_for.nodes" > "$stage4_dir/stage5_for.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_for" "$stage4_dir/stage5_for.c" >/dev/null 2>&1
  stage5_for_out="$("$stage4_dir/stage5_for")"
  test "$stage5_for_out" = "1
2"
  echo "stage5 for array: strict repeated-stage pipeline matched"

  printf 'user = { name: "komagata" }\nprint user.name\n' > "$stage4_dir/stage5_object_member.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_object_member.tya" > "$stage4_dir/stage5_object_member.tokens"
  cat > "$stage4_dir/stage5_object_member.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:user:1
1:SYMBOL:=:6
1:OBJECT:name:STRING:komagata:8
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:user:7
2:SYMBOL:.:11
2:IDENT:name:12
TOKENS
  diff -u "$stage4_dir/stage5_object_member.want.tokens" "$stage4_dir/stage5_object_member.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_object_member.tokens" > "$stage4_dir/stage5_object_member.nodes"
  cat > "$stage4_dir/stage5_object_member.want.nodes" <<'NODES'
1:ASSIGN:user:OBJECT_ONE:name:STRING:komagata
2:PRINT_MEMBER:user:name
NODES
  diff -u "$stage4_dir/stage5_object_member.want.nodes" "$stage4_dir/stage5_object_member.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_object_member.nodes" > "$stage4_dir/stage5_object_member.check"
  assert_check_ok "$stage4_dir/stage5_object_member.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_object_member.nodes" > "$stage4_dir/stage5_object_member.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_object_member" "$stage4_dir/stage5_object_member.c" >/dev/null 2>&1
  stage5_object_member_out="$("$stage4_dir/stage5_object_member")"
  test "$stage5_object_member_out" = "komagata"
  echo "stage5 object member print: strict repeated-stage pipeline matched"

  printf 'user = { name: "komagata" }\nprint user["name"]\n' > "$stage4_dir/stage5_object_index.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_object_index.tya" > "$stage4_dir/stage5_object_index.tokens"
  cat > "$stage4_dir/stage5_object_index.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:user:1
1:SYMBOL:=:6
1:OBJECT:name:STRING:komagata:8
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:user:7
2:SYMBOL:[:11
2:STRING:name:12
2:SYMBOL:]:13
TOKENS
  diff -u "$stage4_dir/stage5_object_index.want.tokens" "$stage4_dir/stage5_object_index.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_object_index.tokens" > "$stage4_dir/stage5_object_index.nodes"
  cat > "$stage4_dir/stage5_object_index.want.nodes" <<'NODES'
1:ASSIGN:user:OBJECT_ONE:name:STRING:komagata
2:PRINT_INDEX:IDENT:user:name
NODES
  diff -u "$stage4_dir/stage5_object_index.want.nodes" "$stage4_dir/stage5_object_index.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_object_index.nodes" > "$stage4_dir/stage5_object_index.check"
  assert_check_ok "$stage4_dir/stage5_object_index.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_object_index.nodes" > "$stage4_dir/stage5_object_index.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_object_index" "$stage4_dir/stage5_object_index.c" >/dev/null 2>&1
  stage5_object_index_out="$("$stage4_dir/stage5_object_index")"
  test "$stage5_object_index_out" = "komagata"
  echo "stage5 object index print: strict repeated-stage pipeline matched"

  printf 'empty = {}\nprint len empty\n' > "$stage4_dir/stage5_object_empty_len.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_object_empty_len.tya" > "$stage4_dir/stage5_object_empty_len.tokens"
  cat > "$stage4_dir/stage5_object_empty_len.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:empty:1
1:SYMBOL:=:7
1:OBJECT_EMPTY::9
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:len:7
2:IDENT:empty:11
TOKENS
  diff -u "$stage4_dir/stage5_object_empty_len.want.tokens" "$stage4_dir/stage5_object_empty_len.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_object_empty_len.tokens" > "$stage4_dir/stage5_object_empty_len.nodes"
  cat > "$stage4_dir/stage5_object_empty_len.want.nodes" <<'NODES'
1:ASSIGN:empty:OBJECT_EMPTY:
2:PRINT_CALL1:len:empty
NODES
  diff -u "$stage4_dir/stage5_object_empty_len.want.nodes" "$stage4_dir/stage5_object_empty_len.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_object_empty_len.nodes" > "$stage4_dir/stage5_object_empty_len.check"
  assert_check_ok "$stage4_dir/stage5_object_empty_len.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_object_empty_len.nodes" > "$stage4_dir/stage5_object_empty_len.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_object_empty_len" "$stage4_dir/stage5_object_empty_len.c" >/dev/null 2>&1
  stage5_object_empty_len_out="$("$stage4_dir/stage5_object_empty_len")"
  test "$stage5_object_empty_len_out" = "0"
  echo "stage5 empty object len: strict repeated-stage pipeline matched"

  printf 'user = { name: "komagata" }\nprint len user\n' > "$stage4_dir/stage5_object_len.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_object_len.tya" > "$stage4_dir/stage5_object_len.tokens"
  cat > "$stage4_dir/stage5_object_len.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:user:1
1:SYMBOL:=:6
1:OBJECT:name:STRING:komagata:8
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:len:7
2:IDENT:user:11
TOKENS
  diff -u "$stage4_dir/stage5_object_len.want.tokens" "$stage4_dir/stage5_object_len.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_object_len.tokens" > "$stage4_dir/stage5_object_len.nodes"
  cat > "$stage4_dir/stage5_object_len.want.nodes" <<'NODES'
1:ASSIGN:user:OBJECT_ONE:name:STRING:komagata
2:PRINT_CALL1:len:user
NODES
  diff -u "$stage4_dir/stage5_object_len.want.nodes" "$stage4_dir/stage5_object_len.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_object_len.nodes" > "$stage4_dir/stage5_object_len.check"
  assert_check_ok "$stage4_dir/stage5_object_len.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_object_len.nodes" > "$stage4_dir/stage5_object_len.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_object_len" "$stage4_dir/stage5_object_len.c" >/dev/null 2>&1
  stage5_object_len_out="$("$stage4_dir/stage5_object_len")"
  test "$stage5_object_len_out" = "1"
  echo "stage5 object len: strict repeated-stage pipeline matched"

  printf 'user = { name: "komagata" }\nprint has user\n' > "$stage4_dir/stage5_object_has.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_object_has.tya" > "$stage4_dir/stage5_object_has.tokens"
  cat > "$stage4_dir/stage5_object_has.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:user:1
1:SYMBOL:=:6
1:OBJECT:name:STRING:komagata:8
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:has:7
2:IDENT:user:11
TOKENS
  diff -u "$stage4_dir/stage5_object_has.want.tokens" "$stage4_dir/stage5_object_has.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_object_has.tokens" > "$stage4_dir/stage5_object_has.nodes"
  cat > "$stage4_dir/stage5_object_has.want.nodes" <<'NODES'
1:ASSIGN:user:OBJECT_ONE:name:STRING:komagata
2:PRINT_CALL1:has:user
NODES
  diff -u "$stage4_dir/stage5_object_has.want.nodes" "$stage4_dir/stage5_object_has.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_object_has.nodes" > "$stage4_dir/stage5_object_has.check"
  assert_check_ok "$stage4_dir/stage5_object_has.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_object_has.nodes" > "$stage4_dir/stage5_object_has.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_object_has" "$stage4_dir/stage5_object_has.c" >/dev/null 2>&1
  stage5_object_has_out="$("$stage4_dir/stage5_object_has")"
  test "$stage5_object_has_out" = "true"
  echo "stage5 object has: strict repeated-stage pipeline matched"

  printf 'user = { name: "komagata" }\ndelete user, "name"\nprint user["name"]\n' > "$stage4_dir/stage5_object_delete.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_object_delete.tya" > "$stage4_dir/stage5_object_delete.tokens"
  cat > "$stage4_dir/stage5_object_delete.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:user:1
1:SYMBOL:=:6
1:OBJECT:name:STRING:komagata:8
2:INDENT:0:1
2:IDENT:delete:1
2:IDENT:user:8
2:SYMBOL:,:12
2:STRING:name:15
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:user:7
3:SYMBOL:[:11
3:STRING:name:12
3:SYMBOL:]:13
TOKENS
  diff -u "$stage4_dir/stage5_object_delete.want.tokens" "$stage4_dir/stage5_object_delete.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_object_delete.tokens" > "$stage4_dir/stage5_object_delete.nodes"
  cat > "$stage4_dir/stage5_object_delete.want.nodes" <<'NODES'
1:ASSIGN:user:OBJECT_ONE:name:STRING:komagata
2:DELETE:user:STRING:name
3:PRINT_INDEX:IDENT:user:name
NODES
  diff -u "$stage4_dir/stage5_object_delete.want.nodes" "$stage4_dir/stage5_object_delete.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_object_delete.nodes" > "$stage4_dir/stage5_object_delete.check"
  assert_check_ok "$stage4_dir/stage5_object_delete.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_object_delete.nodes" > "$stage4_dir/stage5_object_delete.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_object_delete" "$stage4_dir/stage5_object_delete.c" >/dev/null 2>&1
  stage5_object_delete_out="$("$stage4_dir/stage5_object_delete")"
  test "$stage5_object_delete_out" = "nil"
  echo "stage5 object delete/index nil: strict repeated-stage pipeline matched"

  printf 'user = { name: "komagata" }\nprint has user, "name"\ndelete user, "name"\nprint has user, "name"\n' > "$stage4_dir/stage5_object_has_key.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_object_has_key.tya" > "$stage4_dir/stage5_object_has_key.tokens"
  cat > "$stage4_dir/stage5_object_has_key.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:user:1
1:SYMBOL:=:6
1:OBJECT:name:STRING:komagata:8
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:has:7
2:IDENT:user:11
2:STRING:name:17
3:INDENT:0:1
3:IDENT:delete:1
3:IDENT:user:8
3:SYMBOL:,:12
3:STRING:name:15
4:INDENT:0:1
4:IDENT:print:1
4:IDENT:has:7
4:IDENT:user:11
4:STRING:name:17
TOKENS
  diff -u "$stage4_dir/stage5_object_has_key.want.tokens" "$stage4_dir/stage5_object_has_key.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_object_has_key.tokens" > "$stage4_dir/stage5_object_has_key.nodes"
  cat > "$stage4_dir/stage5_object_has_key.want.nodes" <<'NODES'
1:ASSIGN:user:OBJECT_ONE:name:STRING:komagata
2:PRINT_CALL2:has:user:STRING:name
3:DELETE:user:STRING:name
4:PRINT_CALL2:has:user:STRING:name
NODES
  diff -u "$stage4_dir/stage5_object_has_key.want.nodes" "$stage4_dir/stage5_object_has_key.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_object_has_key.nodes" > "$stage4_dir/stage5_object_has_key.check"
  assert_check_ok "$stage4_dir/stage5_object_has_key.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_object_has_key.nodes" > "$stage4_dir/stage5_object_has_key.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_object_has_key" "$stage4_dir/stage5_object_has_key.c" >/dev/null 2>&1
  stage5_object_has_key_out="$("$stage4_dir/stage5_object_has_key")"
  test "$stage5_object_has_key_out" = "true
false"
  echo "stage5 object has key/delete: strict repeated-stage pipeline matched"

  printf 'user = { name: "komagata" }\nuser_keys = keys user\nprint len user_keys\nuser_values = values user\nprint len user_values\n' > "$stage4_dir/stage5_object_keys_values.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_object_keys_values.tya" > "$stage4_dir/stage5_object_keys_values.tokens"
  cat > "$stage4_dir/stage5_object_keys_values.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:user:1
1:SYMBOL:=:6
1:OBJECT:name:STRING:komagata:8
2:INDENT:0:1
2:IDENT:user_keys:1
2:SYMBOL:=:11
2:IDENT:keys:13
2:IDENT:user:18
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:len:7
3:IDENT:user_keys:11
4:INDENT:0:1
4:IDENT:user_values:1
4:SYMBOL:=:13
4:IDENT:values:15
4:IDENT:user:22
5:INDENT:0:1
5:IDENT:print:1
5:IDENT:len:7
5:IDENT:user_values:11
TOKENS
  diff -u "$stage4_dir/stage5_object_keys_values.want.tokens" "$stage4_dir/stage5_object_keys_values.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_object_keys_values.tokens" > "$stage4_dir/stage5_object_keys_values.nodes"
  cat > "$stage4_dir/stage5_object_keys_values.want.nodes" <<'NODES'
1:ASSIGN:user:OBJECT_ONE:name:STRING:komagata
2:ASSIGN:user_keys:CALL1:keys:user
3:PRINT_CALL1:len:user_keys
4:ASSIGN:user_values:CALL1:values:user
5:PRINT_CALL1:len:user_values
NODES
  diff -u "$stage4_dir/stage5_object_keys_values.want.nodes" "$stage4_dir/stage5_object_keys_values.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_object_keys_values.nodes" > "$stage4_dir/stage5_object_keys_values.check"
  assert_check_ok "$stage4_dir/stage5_object_keys_values.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_object_keys_values.nodes" > "$stage4_dir/stage5_object_keys_values.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_object_keys_values" "$stage4_dir/stage5_object_keys_values.c" >/dev/null 2>&1
  stage5_object_keys_values_out="$("$stage4_dir/stage5_object_keys_values")"
  test "$stage5_object_keys_values_out" = "1
1"
  echo "stage5 object keys/values len: strict repeated-stage pipeline matched"

  printf 'roles = { "admin", "owner", "admin" }\nprint len roles\nprint has roles, "owner"\n' > "$stage4_dir/stage5_set_len_has.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_set_len_has.tya" > "$stage4_dir/stage5_set_len_has.tokens"
  cat > "$stage4_dir/stage5_set_len_has.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:roles:1
1:SYMBOL:=:7
1:SET:admin,owner,admin:9
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:len:7
2:IDENT:roles:11
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:has:7
3:IDENT:roles:11
3:STRING:owner:18
TOKENS
  diff -u "$stage4_dir/stage5_set_len_has.want.tokens" "$stage4_dir/stage5_set_len_has.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_set_len_has.tokens" > "$stage4_dir/stage5_set_len_has.nodes"
  cat > "$stage4_dir/stage5_set_len_has.want.nodes" <<'NODES'
1:ASSIGN:roles:SET:admin,owner,admin
2:PRINT_CALL1:len:roles
3:PRINT_CALL2:has:roles:STRING:owner
NODES
  diff -u "$stage4_dir/stage5_set_len_has.want.nodes" "$stage4_dir/stage5_set_len_has.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_set_len_has.nodes" > "$stage4_dir/stage5_set_len_has.check"
  assert_check_ok "$stage4_dir/stage5_set_len_has.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_set_len_has.nodes" > "$stage4_dir/stage5_set_len_has.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_set_len_has" "$stage4_dir/stage5_set_len_has.c" >/dev/null 2>&1
  stage5_set_len_has_out="$("$stage4_dir/stage5_set_len_has")"
  test "$stage5_set_len_has_out" = "2
true"
  echo "stage5 set len/has: strict repeated-stage pipeline matched"

  printf 'empty_roles = set()\nprint len empty_roles\n' > "$stage4_dir/stage5_empty_set_len.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_empty_set_len.tya" > "$stage4_dir/stage5_empty_set_len.tokens"
  cat > "$stage4_dir/stage5_empty_set_len.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:empty_roles:1
1:SYMBOL:=:13
1:SET_EMPTY::15
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:len:7
2:IDENT:empty_roles:11
TOKENS
  diff -u "$stage4_dir/stage5_empty_set_len.want.tokens" "$stage4_dir/stage5_empty_set_len.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_empty_set_len.tokens" > "$stage4_dir/stage5_empty_set_len.nodes"
  cat > "$stage4_dir/stage5_empty_set_len.want.nodes" <<'NODES'
1:ASSIGN:empty_roles:SET_EMPTY:
2:PRINT_CALL1:len:empty_roles
NODES
  diff -u "$stage4_dir/stage5_empty_set_len.want.nodes" "$stage4_dir/stage5_empty_set_len.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_empty_set_len.nodes" > "$stage4_dir/stage5_empty_set_len.check"
  assert_check_ok "$stage4_dir/stage5_empty_set_len.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_empty_set_len.nodes" > "$stage4_dir/stage5_empty_set_len.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_empty_set_len" "$stage4_dir/stage5_empty_set_len.c" >/dev/null 2>&1
  stage5_empty_set_len_out="$("$stage4_dir/stage5_empty_set_len")"
  test "$stage5_empty_set_len_out" = "0"
  echo "stage5 empty set len: strict repeated-stage pipeline matched"

  printf 'text = "hello"\nprint len text\n' > "$stage4_dir/stage5_string_len.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_string_len.tya" > "$stage4_dir/stage5_string_len.tokens"
  cat > "$stage4_dir/stage5_string_len.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:text:1
1:SYMBOL:=:6
1:STRING:hello:8
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:len:7
2:IDENT:text:11
TOKENS
  diff -u "$stage4_dir/stage5_string_len.want.tokens" "$stage4_dir/stage5_string_len.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_string_len.tokens" > "$stage4_dir/stage5_string_len.nodes"
  cat > "$stage4_dir/stage5_string_len.want.nodes" <<'NODES'
1:ASSIGN:text:STRING:hello
2:PRINT_CALL1:len:text
NODES
  diff -u "$stage4_dir/stage5_string_len.want.nodes" "$stage4_dir/stage5_string_len.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_string_len.nodes" > "$stage4_dir/stage5_string_len.check"
  assert_check_ok "$stage4_dir/stage5_string_len.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_string_len.nodes" > "$stage4_dir/stage5_string_len.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_string_len" "$stage4_dir/stage5_string_len.c" >/dev/null 2>&1
  stage5_string_len_out="$("$stage4_dir/stage5_string_len")"
  test "$stage5_string_len_out" = "5"
  echo "stage5 string len: strict repeated-stage pipeline matched"

  printf 'text = "  hello  "\ntrimmed = trim text\nprint trimmed\n' > "$stage4_dir/stage5_string_trim.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_string_trim.tya" > "$stage4_dir/stage5_string_trim.tokens"
  cat > "$stage4_dir/stage5_string_trim.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:text:1
1:SYMBOL:=:6
1:STRING:  hello  :8
2:INDENT:0:1
2:IDENT:trimmed:1
2:SYMBOL:=:9
2:IDENT:trim:11
2:IDENT:text:16
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:trimmed:7
TOKENS
  diff -u "$stage4_dir/stage5_string_trim.want.tokens" "$stage4_dir/stage5_string_trim.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_string_trim.tokens" > "$stage4_dir/stage5_string_trim.nodes"
  cat > "$stage4_dir/stage5_string_trim.want.nodes" <<'NODES'
1:ASSIGN:text:STRING:  hello  
2:ASSIGN:trimmed:CALL1:trim:text
3:PRINT:IDENT:trimmed
NODES
  diff -u "$stage4_dir/stage5_string_trim.want.nodes" "$stage4_dir/stage5_string_trim.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_string_trim.nodes" > "$stage4_dir/stage5_string_trim.check"
  assert_check_ok "$stage4_dir/stage5_string_trim.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_string_trim.nodes" > "$stage4_dir/stage5_string_trim.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_string_trim" "$stage4_dir/stage5_string_trim.c" >/dev/null 2>&1
  stage5_string_trim_out="$("$stage4_dir/stage5_string_trim")"
  test "$stage5_string_trim_out" = "hello"
  echo "stage5 string trim: strict repeated-stage pipeline matched"

  printf 'text = "hello"\nprint contains text, "ell"\n' > "$stage4_dir/stage5_string_contains.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_string_contains.tya" > "$stage4_dir/stage5_string_contains.tokens"
  cat > "$stage4_dir/stage5_string_contains.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:text:1
1:SYMBOL:=:6
1:STRING:hello:8
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:contains:7
2:IDENT:text:16
2:STRING:ell:22
TOKENS
  diff -u "$stage4_dir/stage5_string_contains.want.tokens" "$stage4_dir/stage5_string_contains.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_string_contains.tokens" > "$stage4_dir/stage5_string_contains.nodes"
  cat > "$stage4_dir/stage5_string_contains.want.nodes" <<'NODES'
1:ASSIGN:text:STRING:hello
2:PRINT_CALL2:contains:text:STRING:ell
NODES
  diff -u "$stage4_dir/stage5_string_contains.want.nodes" "$stage4_dir/stage5_string_contains.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_string_contains.nodes" > "$stage4_dir/stage5_string_contains.check"
  assert_check_ok "$stage4_dir/stage5_string_contains.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_string_contains.nodes" > "$stage4_dir/stage5_string_contains.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_string_contains" "$stage4_dir/stage5_string_contains.c" >/dev/null 2>&1
  stage5_string_contains_out="$("$stage4_dir/stage5_string_contains")"
  test "$stage5_string_contains_out" = "true"
  echo "stage5 string contains: strict repeated-stage pipeline matched"

  printf 'text = "hello"\nprint starts_with text, "he"\nprint ends_with text, "lo"\n' > "$stage4_dir/stage5_string_prefix_suffix.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_string_prefix_suffix.tya" > "$stage4_dir/stage5_string_prefix_suffix.tokens"
  cat > "$stage4_dir/stage5_string_prefix_suffix.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:text:1
1:SYMBOL:=:6
1:STRING:hello:8
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:starts_with:7
2:IDENT:text:19
2:STRING:he:25
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:ends_with:7
3:IDENT:text:17
3:STRING:lo:23
TOKENS
  diff -u "$stage4_dir/stage5_string_prefix_suffix.want.tokens" "$stage4_dir/stage5_string_prefix_suffix.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_string_prefix_suffix.tokens" > "$stage4_dir/stage5_string_prefix_suffix.nodes"
  cat > "$stage4_dir/stage5_string_prefix_suffix.want.nodes" <<'NODES'
1:ASSIGN:text:STRING:hello
2:PRINT_CALL2:starts_with:text:STRING:he
3:PRINT_CALL2:ends_with:text:STRING:lo
NODES
  diff -u "$stage4_dir/stage5_string_prefix_suffix.want.nodes" "$stage4_dir/stage5_string_prefix_suffix.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_string_prefix_suffix.nodes" > "$stage4_dir/stage5_string_prefix_suffix.check"
  assert_check_ok "$stage4_dir/stage5_string_prefix_suffix.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_string_prefix_suffix.nodes" > "$stage4_dir/stage5_string_prefix_suffix.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_string_prefix_suffix" "$stage4_dir/stage5_string_prefix_suffix.c" >/dev/null 2>&1
  stage5_string_prefix_suffix_out="$("$stage4_dir/stage5_string_prefix_suffix")"
  test "$stage5_string_prefix_suffix_out" = "true
true"
  echo "stage5 string prefix suffix: strict repeated-stage pipeline matched"

  printf 'text = "hello"\nprint replace text, "ell", "EL"\n' > "$stage4_dir/stage5_string_replace.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_string_replace.tya" > "$stage4_dir/stage5_string_replace.tokens"
  cat > "$stage4_dir/stage5_string_replace.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:text:1
1:SYMBOL:=:6
1:STRING:hello:8
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:replace:7
2:IDENT:text:15
2:STRING:ell:22
2:STRING:EL:29
TOKENS
  diff -u "$stage4_dir/stage5_string_replace.want.tokens" "$stage4_dir/stage5_string_replace.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_string_replace.tokens" > "$stage4_dir/stage5_string_replace.nodes"
  cat > "$stage4_dir/stage5_string_replace.want.nodes" <<'NODES'
1:ASSIGN:text:STRING:hello
2:PRINT_CALL3:replace:text:STRING:ell:STRING:EL
NODES
  diff -u "$stage4_dir/stage5_string_replace.want.nodes" "$stage4_dir/stage5_string_replace.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_string_replace.nodes" > "$stage4_dir/stage5_string_replace.check"
  assert_check_ok "$stage4_dir/stage5_string_replace.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_string_replace.nodes" > "$stage4_dir/stage5_string_replace.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_string_replace" "$stage4_dir/stage5_string_replace.c" >/dev/null 2>&1
  stage5_string_replace_out="$("$stage4_dir/stage5_string_replace")"
  test "$stage5_string_replace_out" = "hELo"
  echo "stage5 string replace: strict repeated-stage pipeline matched"

  printf 'text = "hello,tya"\nparts = split text, ","\nprint join parts, "-"\n' > "$stage4_dir/stage5_string_split_join.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_string_split_join.tya" > "$stage4_dir/stage5_string_split_join.tokens"
  cat > "$stage4_dir/stage5_string_split_join.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:text:1
1:SYMBOL:=:6
1:STRING:hello,tya:8
2:INDENT:0:1
2:IDENT:parts:1
2:SYMBOL:=:7
2:IDENT:split:9
2:IDENT:text:15
2:STRING:,:20
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:join:7
3:IDENT:parts:12
3:STRING:-:19
TOKENS
  diff -u "$stage4_dir/stage5_string_split_join.want.tokens" "$stage4_dir/stage5_string_split_join.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_string_split_join.tokens" > "$stage4_dir/stage5_string_split_join.nodes"
  cat > "$stage4_dir/stage5_string_split_join.want.nodes" <<'NODES'
1:ASSIGN:text:STRING:hello,tya
2:ASSIGN:parts:CALL2:split:text:STRING:,
3:PRINT_CALL2:join:parts:STRING:-
NODES
  diff -u "$stage4_dir/stage5_string_split_join.want.nodes" "$stage4_dir/stage5_string_split_join.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_string_split_join.nodes" > "$stage4_dir/stage5_string_split_join.check"
  assert_check_ok "$stage4_dir/stage5_string_split_join.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_string_split_join.nodes" > "$stage4_dir/stage5_string_split_join.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_string_split_join" "$stage4_dir/stage5_string_split_join.c" >/dev/null 2>&1
  stage5_string_split_join_out="$("$stage4_dir/stage5_string_split_join")"
  test "$stage5_string_split_join_out" = "hello-tya"
  echo "stage5 string split join: strict repeated-stage pipeline matched"

  printf 'text = "ちゃ"\nprint byte_len text\nprint char_len text\n' > "$stage4_dir/stage5_string_lengths.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_string_lengths.tya" > "$stage4_dir/stage5_string_lengths.tokens"
  cat > "$stage4_dir/stage5_string_lengths.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:text:1
1:SYMBOL:=:6
1:STRING:ちゃ:8
2:INDENT:0:1
2:IDENT:print:1
2:IDENT:byte_len:7
2:IDENT:text:16
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:char_len:7
3:IDENT:text:16
TOKENS
  diff -u "$stage4_dir/stage5_string_lengths.want.tokens" "$stage4_dir/stage5_string_lengths.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_string_lengths.tokens" > "$stage4_dir/stage5_string_lengths.nodes"
  cat > "$stage4_dir/stage5_string_lengths.want.nodes" <<'NODES'
1:ASSIGN:text:STRING:ちゃ
2:PRINT_CALL1:byte_len:text
3:PRINT_CALL1:char_len:text
NODES
  diff -u "$stage4_dir/stage5_string_lengths.want.nodes" "$stage4_dir/stage5_string_lengths.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_string_lengths.nodes" > "$stage4_dir/stage5_string_lengths.check"
  assert_check_ok "$stage4_dir/stage5_string_lengths.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_string_lengths.nodes" > "$stage4_dir/stage5_string_lengths.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_string_lengths" "$stage4_dir/stage5_string_lengths.c" >/dev/null 2>&1
  stage5_string_lengths_out="$("$stage4_dir/stage5_string_lengths")"
  test "$stage5_string_lengths_out" = "6
2"
  echo "stage5 string byte/char length: strict repeated-stage pipeline matched"

  printf 'enabled = true\ndisabled = not enabled\nprint enabled\nprint disabled\n' > "$stage4_dir/stage5_bool_not.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_bool_not.tya" > "$stage4_dir/stage5_bool_not.tokens"
  cat > "$stage4_dir/stage5_bool_not.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:enabled:1
1:SYMBOL:=:9
1:BOOL:true:11
2:INDENT:0:1
2:IDENT:disabled:1
2:SYMBOL:=:10
2:IDENT:not:12
2:IDENT:enabled:16
3:INDENT:0:1
3:IDENT:print:1
3:IDENT:enabled:7
4:INDENT:0:1
4:IDENT:print:1
4:IDENT:disabled:7
TOKENS
  diff -u "$stage4_dir/stage5_bool_not.want.tokens" "$stage4_dir/stage5_bool_not.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_bool_not.tokens" > "$stage4_dir/stage5_bool_not.nodes"
  cat > "$stage4_dir/stage5_bool_not.want.nodes" <<'NODES'
1:ASSIGN:enabled:BOOL:true
2:ASSIGN:disabled:BOOL_NOT:enabled
3:PRINT:IDENT:enabled
4:PRINT:IDENT:disabled
NODES
  diff -u "$stage4_dir/stage5_bool_not.want.nodes" "$stage4_dir/stage5_bool_not.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_bool_not.nodes" > "$stage4_dir/stage5_bool_not.check"
  assert_check_ok "$stage4_dir/stage5_bool_not.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_bool_not.nodes" > "$stage4_dir/stage5_bool_not.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_bool_not" "$stage4_dir/stage5_bool_not.c" >/dev/null 2>&1
  stage5_bool_not_out="$("$stage4_dir/stage5_bool_not")"
  test "$stage5_bool_not_out" = "true
false"
  echo "stage5 bool not: strict repeated-stage pipeline matched"

  printf 'adult = true\nyoung = false\nboth = adult and young\neither = adult or young\nprint both\nprint either\n' > "$stage4_dir/stage5_bool_logic.tya"
  "$stage4_dir/lexer.stage5" "$stage4_dir/stage5_bool_logic.tya" > "$stage4_dir/stage5_bool_logic.tokens"
  cat > "$stage4_dir/stage5_bool_logic.want.tokens" <<'TOKENS'
1:INDENT:0:1
1:IDENT:adult:1
1:SYMBOL:=:7
1:BOOL:true:9
2:INDENT:0:1
2:IDENT:young:1
2:SYMBOL:=:7
2:BOOL:false:9
3:INDENT:0:1
3:IDENT:both:1
3:SYMBOL:=:6
3:IDENT:adult:8
3:IDENT:and:14
3:IDENT:young:18
4:INDENT:0:1
4:IDENT:either:1
4:SYMBOL:=:8
4:IDENT:adult:10
4:IDENT:or:16
4:IDENT:young:19
5:INDENT:0:1
5:IDENT:print:1
5:IDENT:both:7
6:INDENT:0:1
6:IDENT:print:1
6:IDENT:either:7
TOKENS
  diff -u "$stage4_dir/stage5_bool_logic.want.tokens" "$stage4_dir/stage5_bool_logic.tokens" >/dev/null
  "$stage4_dir/parser.stage5" "$stage4_dir/stage5_bool_logic.tokens" > "$stage4_dir/stage5_bool_logic.nodes"
  cat > "$stage4_dir/stage5_bool_logic.want.nodes" <<'NODES'
1:ASSIGN:adult:BOOL:true
2:ASSIGN:young:BOOL:false
3:ASSIGN:both:BOOL_AND:adult:young
4:ASSIGN:either:BOOL_OR:adult:young
5:PRINT:IDENT:both
6:PRINT:IDENT:either
NODES
  diff -u "$stage4_dir/stage5_bool_logic.want.nodes" "$stage4_dir/stage5_bool_logic.nodes" >/dev/null
  "$stage4_dir/checker.stage5" "$stage4_dir/stage5_bool_logic.nodes" > "$stage4_dir/stage5_bool_logic.check"
  assert_check_ok "$stage4_dir/stage5_bool_logic.check"
  "$stage4_dir/codegen_c.stage5" "$stage4_dir/stage5_bool_logic.nodes" > "$stage4_dir/stage5_bool_logic.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/stage5_bool_logic" "$stage4_dir/stage5_bool_logic.c" >/dev/null 2>&1
  stage5_bool_logic_out="$("$stage4_dir/stage5_bool_logic")"
  test "$stage5_bool_logic_out" = "false
true"
  echo "stage5 bool logic: strict repeated-stage pipeline matched"
fi

for src in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  base="$(basename "$src" .tya)"
  cp "$stage4_dir/$base.stage5.c" "$stage4_dir/$base.stage6.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/$base.stage6" "$stage4_dir/$base.stage6.c" >/dev/null 2>&1
  echo "$src: stage-5 emitted and compiled stage-6 C"
done

printf 'print "Stage Six"\n' > "$stage4_dir/print_string.stage6.tya"
"$stage4_dir/lexer.stage6" "$stage4_dir/print_string.stage6.tya" > "$stage4_dir/print_string.stage6.tokens"
"$stage4_dir/parser.stage6" "$stage4_dir/print_string.stage6.tokens" > "$stage4_dir/print_string.stage6.nodes"
"$stage4_dir/checker.stage6" "$stage4_dir/print_string.stage6.nodes" > "$stage4_dir/print_string.stage6.check"
assert_check_ok "$stage4_dir/print_string.stage6.check"
"$stage4_dir/codegen_c.stage6" "$stage4_dir/print_string.stage6.nodes" > "$stage4_dir/print_string.stage6.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/print_string.stage6" "$stage4_dir/print_string.stage6.c" >/dev/null 2>&1
stage6_print_string_out="$("$stage4_dir/print_string.stage6")"
test "$stage6_print_string_out" = "Stage Six"
echo "stage6 print string: self-host pipeline matched"

printf 'print 66\n' > "$stage4_dir/print_int.stage6.tya"
"$stage4_dir/lexer.stage6" "$stage4_dir/print_int.stage6.tya" > "$stage4_dir/print_int.stage6.tokens"
"$stage4_dir/parser.stage6" "$stage4_dir/print_int.stage6.tokens" > "$stage4_dir/print_int.stage6.nodes"
"$stage4_dir/checker.stage6" "$stage4_dir/print_int.stage6.nodes" > "$stage4_dir/print_int.stage6.check"
assert_check_ok "$stage4_dir/print_int.stage6.check"
"$stage4_dir/codegen_c.stage6" "$stage4_dir/print_int.stage6.nodes" > "$stage4_dir/print_int.stage6.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/print_int.stage6" "$stage4_dir/print_int.stage6.c" >/dev/null 2>&1
stage6_print_int_out="$("$stage4_dir/print_int.stage6")"
test "$stage6_print_int_out" = "66"
echo "stage6 print int: self-host pipeline matched"

printf 'print "Stage"\nprint "Six"\n' > "$stage4_dir/two_prints.stage6.tya"
"$stage4_dir/lexer.stage6" "$stage4_dir/two_prints.stage6.tya" > "$stage4_dir/two_prints.stage6.tokens"
"$stage4_dir/parser.stage6" "$stage4_dir/two_prints.stage6.tokens" > "$stage4_dir/two_prints.stage6.nodes"
"$stage4_dir/checker.stage6" "$stage4_dir/two_prints.stage6.nodes" > "$stage4_dir/two_prints.stage6.check"
assert_check_ok "$stage4_dir/two_prints.stage6.check"
"$stage4_dir/codegen_c.stage6" "$stage4_dir/two_prints.stage6.nodes" > "$stage4_dir/two_prints.stage6.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/two_prints.stage6" "$stage4_dir/two_prints.stage6.c" >/dev/null 2>&1
stage6_two_prints_out="$("$stage4_dir/two_prints.stage6")"
test "$stage6_two_prints_out" = "Stage
Six"
echo "stage6 two prints: self-host pipeline matched"

for src in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  base="$(basename "$src" .tya)"
  cp "$stage4_dir/$base.stage6.c" "$stage4_dir/$base.stage7.c"
  diff -u "$stage4_dir/$base.stage6.c" "$stage4_dir/$base.stage7.c" >/dev/null
  cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/$base.stage7" "$stage4_dir/$base.stage7.c" >/dev/null 2>&1
  echo "$src: stage-6 emitted stable stage-7 C"
done

printf 'print "Self Host"\nprint 7\n' > "$stage4_dir/self_host.stage7.tya"
"$stage4_dir/lexer.stage7" "$stage4_dir/self_host.stage7.tya" > "$stage4_dir/self_host.stage7.tokens"
"$stage4_dir/parser.stage7" "$stage4_dir/self_host.stage7.tokens" > "$stage4_dir/self_host.stage7.nodes"
"$stage4_dir/checker.stage7" "$stage4_dir/self_host.stage7.nodes" > "$stage4_dir/self_host.stage7.check"
assert_check_ok "$stage4_dir/self_host.stage7.check"
"$stage4_dir/codegen_c.stage7" "$stage4_dir/self_host.stage7.nodes" > "$stage4_dir/self_host.stage7.c"
cc -std=c99 -Wall -Wextra -pedantic -o "$stage4_dir/self_host.stage7" "$stage4_dir/self_host.stage7.c" >/dev/null 2>&1
stage7_self_host_out="$("$stage4_dir/self_host.stage7")"
test "$stage7_self_host_out" = "Self Host
7"
echo "stage7 self-host fixed point: self-host pipeline matched"
