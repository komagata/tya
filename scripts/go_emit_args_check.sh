#!/usr/bin/env sh
set -eu

out_dir="${TMPDIR:-/tmp}/tya-go-emit-args"

mkdir -p "$out_dir"

TYA_EXAMPLE=hello go run ./cmd/tya examples/args.tya foo > "$out_dir/args.want"
go run ./cmd/tya --emit-c examples/args.tya > "$out_dir/args.c"
gcc "$out_dir/args.c" runtime/tya_runtime.c -I runtime -o "$out_dir/args"
TYA_EXAMPLE=hello "$out_dir/args" foo > "$out_dir/args.got"
diff -u "$out_dir/args.want" "$out_dir/args.got"
echo "examples/args.tya: matched"
