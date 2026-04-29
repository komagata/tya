# Run To Completion Progress

Updated: 2026-04-29 21:35:34 JST

Active phase: commit `print contains` slice.

Completed:

- Read the required skills and repo self-host protocol.
- Identified the first unchecked queue item in `SELFHOST_WORK.md`.
- Added repeated stage-2 generated-C determinism checks for supported subset fixtures.
- Focused `sh scripts/stage1_selfhost_sources_check.sh` passes.
- `go test ./... -count=1` passes.
- `sh scripts/selfhost_bootstrap_check.sh` passes.
- Committed deterministic stage-2 codegen checkpoint as `ad0ab90`.
- Committed parent queue cleanup as `d4ff7f3`.
- Confirmed `SELFHOST_WORK.md` has no unchecked queue items.
- Received new goal: continue development until full self-hosting works.
- Found remaining Self-Host Completion TODOs in `ROADMAP.md`.
- Seeded `SELFHOST_WORK.md` with a new high-level completion queue.
- Marked completed lexer parity TODOs in `ROADMAP.md`.
- Added self-host parser support for `name = call(arg)`.
- Focused parser parity test passes.
- `sh scripts/selfhost_check.sh` passes.
- `go test ./... -count=1` passes.
- `sh scripts/selfhost_bootstrap_check.sh` passes.
- Committed one-argument call parser slice as `69e0052`.
- Added self-host parser support for `name = call(arg, arg2)`.
- Focused parser parity test passes for `split(message, "\\n")`.
- `sh scripts/selfhost_check.sh` passes for the two-argument call slice.
- `go test ./... -count=1` passes for the two-argument call slice.
- `sh scripts/selfhost_bootstrap_check.sh` passes for the two-argument call slice.
- Committed two-argument call parser slice as `701ff4c`.
- Added self-host parser support for `name = call(arg, arg2, arg3)`.
- Focused parser parity test passes for `replace(message, "T", message)`.
- `sh scripts/selfhost_check.sh` passes for the three-argument call slice.
- `go test ./... -count=1` passes for the three-argument call slice.
- `sh scripts/selfhost_bootstrap_check.sh` passes for the three-argument call slice.
- Committed three-argument call parser slice as `3eccd91`.
- Added `replace` to the self-host checker builtin scope.
- Focused checker tests for `replace` and undefined `CALL3` pass.
- `sh scripts/selfhost_check.sh` passes for the checker builtin slice.
- `go test ./... -count=1` passes for the checker builtin slice.
- `sh scripts/selfhost_bootstrap_check.sh` passes for the checker builtin slice.
- Committed checker builtin slice as `4dcc2e5`.
- Added self-host C codegen lowering for `replace(text, old, new)`.
- Focused codegen test passes.
- `sh scripts/selfhost_check.sh` passes for the `replace` codegen slice.
- `go test ./... -count=1` passes for the `replace` codegen slice.
- `sh scripts/selfhost_bootstrap_check.sh` passes for the `replace` codegen slice.
- Committed `replace` codegen slice as `49a5c75`.
- Added parser/checker/codegen support for `print replace text, "old", new`.
- Focused parser/checker/codegen tests pass.
- `sh scripts/selfhost_check.sh` passes for the `print replace` slice.
- `go test ./... -count=1` passes for the `print replace` slice.
- `sh scripts/selfhost_bootstrap_check.sh` passes for the `print replace` slice.
- Committed `print replace` slice as `76eae81`.
- Added parser/checker/codegen support for `print contains text, "needle"`.
- Focused parser/checker/codegen tests pass.
- `sh scripts/selfhost_check.sh` passes for the `print contains` slice.
- `go test ./... -count=1` passes for the `print contains` slice.
- `sh scripts/selfhost_bootstrap_check.sh` passes for the `print contains` slice.

Remaining:

- Commit the `print contains` slice.

Estimate: many iterations overall; current parser slice less than 1 iteration.
