#!/usr/bin/env sh
set -eu

out_dir="$(mktemp -d "${TMPDIR:-/tmp}/tya-selfhost-check.XXXXXX")"
mkdir -p "$out_dir"
go build -o "$out_dir/tya" ./cmd/tya

for source in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  name="$(basename "$source" .tya)"
  "$out_dir/tya" selfhost/lexer.tya "$source" > "$out_dir/$name.tokens"
  "$out_dir/tya" selfhost/parser.tya "$out_dir/$name.tokens" > "$out_dir/$name.nodes"
  result="$("$out_dir/tya" selfhost/checker.tya "$out_dir/$name.nodes")"
  if [ "$result" != "ok" ]; then
    printf '%s\n%s\n' "$source" "$result"
    exit 1
  fi
  printf '%s: ok\n' "$source"
done
