# Tya Benchmarks

These scripts provide a small, repeatable baseline for runtime and compiler
performance work. They are intentionally simple workloads that exercise the hot
paths targeted by the performance issues.

Run all benchmarks from the repository root:

```sh
sh scripts/bench.sh
```

The runner builds a temporary local `tya` binary, compiles each benchmark with
`tya build`, then runs the generated executable. The output table reports build
time and execution time separately. It also includes a self-host stage-2
generation row for `selfhost/v01/compiler.tya`.

Set `TYA_BENCH_RUNS` to average multiple runs:

```sh
TYA_BENCH_RUNS=3 sh scripts/bench.sh
```
