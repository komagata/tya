#!/usr/bin/env sh
set -eu

out_dir="${TMPDIR:-/tmp}/tya-go-emit-selfhost-run"

mkdir -p "$out_dir"

go run ./cmd/tya --emit-c selfhost/lexer.tya > "$out_dir/lexer.c"
gcc "$out_dir/lexer.c" runtime/tya_runtime.c -I runtime -o "$out_dir/lexer"
"$out_dir/lexer" examples/hello.tya
