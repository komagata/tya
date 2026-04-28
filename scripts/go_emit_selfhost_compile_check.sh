#!/usr/bin/env sh
set -eu

out_dir="$(mktemp -d "${TMPDIR:-/tmp}/tya-go-emit-selfhost.XXXXXX")"

mkdir -p "$out_dir"

for src in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  base="$(basename "$src" .tya)"
  go run ./cmd/tya --emit-c "$src" > "$out_dir/$base.c"
  gcc "$out_dir/$base.c" runtime/tya_runtime.c -I runtime -o "$out_dir/$base"
  echo "$src: go-emit compiled"
done
