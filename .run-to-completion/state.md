# Run To Completion State

Updated: 2026-04-29 19:11:11 JST

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
- active: commit two-argument call parser slice.

## Next Action

Review final diff and commit the two-argument call parser slice.

## Remaining Work Estimate

Many iterations overall, low confidence. The immediate docs checkpoint is less
than 1 iteration.
