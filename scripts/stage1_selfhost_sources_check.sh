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
