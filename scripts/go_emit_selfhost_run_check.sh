#!/usr/bin/env sh
set -eu

out_dir="$(mktemp -d "${TMPDIR:-/tmp}/tya-go-emit-selfhost-run.XXXXXX")"
cc_warning_flags=""
if printf '' | gcc -Wno-format-truncation -x c -fsyntax-only - >/dev/null 2>&1; then
  cc_warning_flags="-Wno-format-truncation"
fi

mkdir -p "$out_dir"

for src in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  base="$(basename "$src" .tya)"
  go run ./cmd/tya --emit-c "$src" > "$out_dir/$base.c"
  gcc $cc_warning_flags "$out_dir/$base.c" runtime/tya_runtime.c -I runtime -o "$out_dir/$base"
done

"$out_dir/lexer" examples/hello.tya > "$out_dir/tokens.txt"
"$out_dir/parser" "$out_dir/tokens.txt" > "$out_dir/nodes.txt"
"$out_dir/checker" "$out_dir/nodes.txt" > "$out_dir/check.txt"
grep -qx "ok" "$out_dir/check.txt"
"$out_dir/codegen_c" "$out_dir/nodes.txt" > "$out_dir/main.c"
gcc $cc_warning_flags "$out_dir/main.c" -o "$out_dir/main"
"$out_dir/main"
