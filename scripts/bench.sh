#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

runs="${TYA_BENCH_RUNS:-1}"
case "$runs" in
  ''|*[!0-9]*)
    echo "TYA_BENCH_RUNS must be a positive integer" >&2
    exit 1
    ;;
esac
if [ "$runs" -lt 1 ]; then
  echo "TYA_BENCH_RUNS must be a positive integer" >&2
  exit 1
fi

tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/tya-bench.XXXXXX")"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

tya_bin="$tmp_dir/tya"
go build -o "$tya_bin" ./cmd/tya

timestamp() {
  perl -MTime::HiRes=time -e 'printf "%.9f\n", time'
}

elapsed() {
  start="$1"
  end="$2"
  perl -e 'printf "%.3f", $ARGV[1] - $ARGV[0]' "$start" "$end"
}

avg_time() {
  count="$1"
  shift
  total="0"
  i=0
  while [ "$i" -lt "$count" ]; do
    start="$(timestamp)"
    "$@" >/dev/null
    end="$(timestamp)"
    delta="$(elapsed "$start" "$end")"
    total="$(perl -e 'printf "%.9f", $ARGV[0] + $ARGV[1]' "$total" "$delta")"
    i=$((i + 1))
  done
  perl -e 'printf "%.3f", $ARGV[0] / $ARGV[1]' "$total" "$count"
}

avg_time_env() {
  count="$1"
  env_name="$2"
  env_value="$3"
  shift 3
  total="0"
  i=0
  while [ "$i" -lt "$count" ]; do
    start="$(timestamp)"
    env "$env_name=$env_value" "$@" >/dev/null
    end="$(timestamp)"
    delta="$(elapsed "$start" "$end")"
    total="$(perl -e 'printf "%.9f", $ARGV[0] + $ARGV[1]' "$total" "$delta")"
    i=$((i + 1))
  done
  perl -e 'printf "%.3f", $ARGV[0] / $ARGV[1]' "$total" "$count"
}

printf "%-28s %10s %10s %s\n" "benchmark" "build_s" "run_s" "notes"
printf "%-28s %10s %10s %s\n" "----------------------------" "--------" "--------" "-----"

run_benchmark() {
  name="$1"
  src="$2"
  bin="$tmp_dir/$name"
  build_s="$(avg_time "$runs" "$tya_bin" build "$src" -o "$bin")"
  run_s="$(avg_time "$runs" "$bin")"
  printf "%-28s %10s %10s %s\n" "$name" "$build_s" "$run_s" "$src"
}

run_benchmark "dict" "benchmarks/dict_workload.tya"
run_benchmark "string_concat" "benchmarks/string_concat.tya"
run_benchmark "utf8_index" "benchmarks/utf8_index.tya"
run_benchmark "array" "benchmarks/array_workload.tya"

selfhost_c="$tmp_dir/selfhost_v01_stage2.c"
selfhost_s="$(avg_time_env "$runs" TYA_LEGACY_MODULES 1 "$tya_bin" run selfhost/v01/compiler.tya selfhost/v01/compiler.tya)"
env TYA_LEGACY_MODULES=1 "$tya_bin" run selfhost/v01/compiler.tya selfhost/v01/compiler.tya >"$selfhost_c"
printf "%-28s %10s %10s %s\n" "selfhost_v01_stage2" "$selfhost_s" "n/a" "stage-2 C generation"
