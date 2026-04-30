# Run To Completion State

Updated: 2026-04-30 10:21:30 JST

## Goal

Iterate development until Tya can self-host: the Tya-written compiler should
compile itself and produce a compiler that can compile supported Tya programs
without depending on the Go implementation.

## Assumptions

- The old `SELFHOST_WORK.md` queue was complete, but `ROADMAP.md` still has a
  larger Self-Host Completion TODO.
- Continue by turning that TODO into concrete queue slices and implementing the
  first actionable slice.
- The standard loop is `inspect -> act -> verify -> record`.
- Stop only when the goal is complete, impossible, unsafe, or blocked by missing
  external input.

## Success Criteria

- Each iteration picks the smallest unchecked self-host completion slice.
- Focused verification for the slice passes.
- `go test ./... -count=1` passes.
- `sh scripts/selfhost_bootstrap_check.sh` passes.
- `SELFHOST_WORK.md` and `ROADMAP.md` reflect each completed slice.
- Each slice is committed with `Masaki Komagata <komagata@gmail.com>`.

## Phases

- done: inspect old queue and roadmap.
- done: seed a new self-host completion queue from `ROADMAP.md`.
- done: promote completed lexer parity TODOs.
- done: implement parser slice for parenthesized one-argument calls.
- done: verify parser slice.
- done: commit parser slice for parenthesized one-argument calls.
- done: implement parser slice for parenthesized two-argument calls.
- done: verify two-argument call parser slice.
- done: commit two-argument call parser slice.
- done: implement parser slice for parenthesized three-argument calls.
- done: verify three-argument call parser slice.
- done: commit three-argument call parser slice.
- done: implement checker slice for `replace` builtin.
- done: verify checker builtin slice.
- done: commit checker builtin slice.
- done: implement codegen slice for `replace`.
- done: verify `replace` codegen slice.
- done: commit `replace` codegen slice.
- done: implement `print replace` parser/checker/codegen slice.
- done: verify `print replace` slice.
- done: commit `print replace` slice.
- done: implement `print contains` parser/checker/codegen slice.
- done: verify `print contains` slice.
- done: commit `print contains` slice.
- done: implement `print startsWith` / `print endsWith` codegen slice.
- done: verify `print startsWith` / `print endsWith` slice.
- done: commit starts/ends slice.
- done: implement `trim` checker/codegen slice.
- done: verify `trim` slice.
- done: commit `trim` slice.
- done: implement `print len` codegen slice.
- done: verify `print len` slice.
- done: commit `print len` slice.
- done: implement stage-2 string length bootstrap slice.
- done: verify stage-2 string length bootstrap slice.
- done: commit stage-2 string length bootstrap slice.
- done: implement stage-2 string trim bootstrap slice.
- done: verify stage-2 string trim bootstrap slice.
- done: commit stage-2 string trim bootstrap slice.
- done: implement stage-2 string contains bootstrap slice.
- done: verify stage-2 string contains bootstrap slice.
- active: commit stage-2 string contains bootstrap slice.

## Next Action

Commit the verified stage-2 string contains bootstrap slice, then pick the next
smallest bootstrap expansion toward `examples/string.tya`.

## Remaining Work Estimate

Many iterations overall, low confidence. The immediate bootstrap slice is less
than 1 iteration.
