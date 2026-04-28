#!/usr/bin/env sh
set -eu

out_dir="${TMPDIR:-/tmp}/tya-go-emit-examples"

mkdir -p "$out_dir"

for src in examples/arithmetic.tya examples/function.tya examples/while.tya examples/if.tya examples/array.tya examples/string.tya; do
  base="$(basename "$src" .tya)"
  go run ./cmd/tya "$src" > "$out_dir/$base.want"
  go run ./cmd/tya --emit-c "$src" > "$out_dir/$base.c"
  gcc "$out_dir/$base.c" runtime/tya_runtime.c -I runtime -o "$out_dir/$base"
  "$out_dir/$base" > "$out_dir/$base.got"
  diff -u "$out_dir/$base.want" "$out_dir/$base.got"
  echo "$src: matched"
done
