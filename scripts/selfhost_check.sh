#!/usr/bin/env sh
set -eu

out_dir="${TMPDIR:-/tmp}/tya-selfhost-check"
mkdir -p "$out_dir"

for source in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  name="$(basename "$source" .tya)"
  go run ./cmd/tya selfhost/lexer.tya "$source" > "$out_dir/$name.tokens"
  go run ./cmd/tya selfhost/parser.tya "$out_dir/$name.tokens" > "$out_dir/$name.nodes"
  result="$(go run ./cmd/tya selfhost/checker.tya "$out_dir/$name.nodes")"
  if [ "$result" != "ok" ]; then
    printf '%s\n%s\n' "$source" "$result"
    exit 1
  fi
  printf '%s: ok\n' "$source"
done
