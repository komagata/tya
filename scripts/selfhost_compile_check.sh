#!/usr/bin/env sh
set -eu

out_dir="${TMPDIR:-/tmp}/tya-selfhost-compile"

mkdir -p "$out_dir"

for src in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  base="$(basename "$src" .tya)"
  go run ./cmd/tya selfhost/lexer.tya "$src" > "$out_dir/$base.tokens"
  go run ./cmd/tya selfhost/parser.tya "$out_dir/$base.tokens" > "$out_dir/$base.nodes"
  go run ./cmd/tya selfhost/codegen_c.tya "$out_dir/$base.nodes" > "$out_dir/$base.c"
  cc -std=c99 -Wall -Wextra -pedantic -o "$out_dir/$base" "$out_dir/$base.c" >/dev/null 2>&1
  echo "$src: compiled"
done
