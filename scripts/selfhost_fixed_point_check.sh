#!/usr/bin/env sh
set -eu

out="$(TYA_STAGE1_SELFHOST_FIXED_POINT_ONLY=1 sh scripts/stage1_selfhost_sources_check.sh)"

for src in selfhost/lexer.tya selfhost/parser.tya selfhost/checker.tya selfhost/codegen_c.tya; do
  line="$src: stage-4 fixed-point generated C stable"
  printf '%s\n' "$out" | grep -qx "$line"
  printf '%s\n' "$line"
done
