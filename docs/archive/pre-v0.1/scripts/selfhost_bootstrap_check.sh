#!/usr/bin/env sh
set -eu

sh scripts/selfhost_check.sh >/dev/null
sh scripts/selfhost_compile_check.sh >/dev/null
sh scripts/go_emit_selfhost_compile_check.sh >/dev/null
sh scripts/go_emit_selfhost_ops_check.sh >/dev/null
if [ "${TYA_SKIP_STAGE1_SELFHOST_SOURCES:-}" != "1" ]; then
  sh scripts/stage1_selfhost_sources_check.sh >/dev/null
fi

out="$(sh scripts/go_emit_selfhost_run_check.sh)"
if [ "$out" != "Hello, Tya" ]; then
  printf 'unexpected stage-1 output: %s\n' "$out" >&2
  exit 1
fi

printf 'selfhost bootstrap: ok\n'
