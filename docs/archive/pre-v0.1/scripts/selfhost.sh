#!/usr/bin/env sh
set -eu

input="${1:-examples/archive/pre-v0.1/selfhost_input.tya}"
out_dir="$(mktemp -d "${TMPDIR:-/tmp}/tya-selfhost.XXXXXX")"
cc_warning_flags=""
if printf '' | gcc -Wno-format-truncation -x c -fsyntax-only - >/dev/null 2>&1; then
  cc_warning_flags="-Wno-format-truncation"
fi

mkdir -p "$out_dir"

go run ./cmd/tya selfhost/lexer.tya "$input" > "$out_dir/tokens.txt"
go run ./cmd/tya selfhost/parser.tya "$out_dir/tokens.txt" > "$out_dir/nodes.txt"
go run ./cmd/tya selfhost/checker.tya "$out_dir/nodes.txt"
go run ./cmd/tya selfhost/codegen_c.tya "$out_dir/nodes.txt" > "$out_dir/main.c"
gcc $cc_warning_flags "$out_dir/main.c" -o "$out_dir/main"
"$out_dir/main"
