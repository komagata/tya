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
