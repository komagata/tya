#!/usr/bin/env sh
set -eu

out_dir="$(mktemp -d "${TMPDIR:-/tmp}/tya-selfhost-compile.XXXXXX")"

mkdir -p "$out_dir"
go build -o "$out_dir/tya" ./cmd/tya

for src in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  base="$(basename "$src" .tya)"
  "$out_dir/tya" selfhost/lexer.tya "$src" > "$out_dir/$base.tokens"
  "$out_dir/tya" selfhost/parser.tya "$out_dir/$base.tokens" > "$out_dir/$base.nodes"
  "$out_dir/tya" selfhost/codegen_c.tya "$out_dir/$base.nodes" > "$out_dir/$base.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/$base" "$out_dir/$base.c" >/dev/null 2>&1
  echo "$src: compiled"
done
