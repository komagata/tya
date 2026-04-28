#!/usr/bin/env sh
set -eu

input="${1:-examples/selfhost_input.tya}"
out_dir="$(mktemp -d "${TMPDIR:-/tmp}/tya-selfhost.XXXXXX")"

mkdir -p "$out_dir"

go run ./cmd/tya selfhost/lexer.tya "$input" > "$out_dir/tokens.txt"
go run ./cmd/tya selfhost/parser.tya "$out_dir/tokens.txt" > "$out_dir/nodes.txt"
go run ./cmd/tya selfhost/checker.tya "$out_dir/nodes.txt"
go run ./cmd/tya selfhost/codegen_c.tya "$out_dir/nodes.txt" > "$out_dir/main.c"
gcc "$out_dir/main.c" -o "$out_dir/main"
"$out_dir/main"
