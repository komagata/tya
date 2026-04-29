#!/usr/bin/env sh
set -eu

out_dir="$(mktemp -d "${TMPDIR:-/tmp}/tya-stage1-selfhost-sources.XXXXXX")"

mkdir -p "$out_dir"

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

"$out_dir/parser.stage2" "$out_dir/literals.stage2.tokens" > "$out_dir/literals.stage2.nodes"
cat > "$out_dir/literals.want.nodes" <<'NODES'
1:ASSIGN:ratio:FLOAT:12.5
2:ASSIGN:text:STRING:a"b
NODES
diff -u "$out_dir/literals.want.nodes" "$out_dir/literals.stage2.nodes" >/dev/null
echo "literal assignments: stage-2 parser matched"

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
