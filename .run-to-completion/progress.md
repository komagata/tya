# Run To Completion Progress

Updated: 2026-04-29 19:17:11 JST

Active phase: commit three-argument call parser slice.

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

Remaining:

- Commit the three-argument call parser slice.

Estimate: many iterations overall; current parser slice less than 1 iteration.
