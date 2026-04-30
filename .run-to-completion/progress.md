# Run To Completion Progress

Updated: 2026-04-30 10:29:11 JST

Active phase: commit stage-2 string prefix/suffix bootstrap slice.

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
- Committed `print contains` slice as `073ad7c`.
- Added checker/codegen support for `print startsWith text, "prefix"` and `print endsWith text, "suffix"`.
- Focused parser/checker/codegen tests pass.
- `sh scripts/selfhost_check.sh` passes for starts/ends slice.
- `go test ./... -count=1` passes for starts/ends slice.
- `sh scripts/selfhost_bootstrap_check.sh` passes for starts/ends slice.
- Committed starts/ends slice as `1ad4ec5`.
- Added checker/codegen support for `trim text`.
- Focused parser/checker/codegen tests pass.
- `sh scripts/selfhost_check.sh` passes for the `trim` slice.
- `go test ./... -count=1` passes for the `trim` slice.
- `sh scripts/selfhost_bootstrap_check.sh` passes for the `trim` slice.
- Committed `trim` slice as `ab254d6`.
- Added self-host C codegen support for `print len value`.
- Focused parser/codegen tests pass.
- `sh scripts/selfhost_check.sh` passes for the `print len` slice.
- `go test ./... -count=1` passes for the `print len` slice.
- `sh scripts/selfhost_bootstrap_check.sh` passes for the `print len` slice.
- Committed `print len` slice as `475f315`.
- Added a stage-2 pipeline fixture for `text = "hello"; print len text`.
- Focused `go test ./tests -run TestStage1SelfhostSourcesEmitC -count=1` passes.
- `sh scripts/selfhost_check.sh` passes for the string length bootstrap slice.
- `go test ./... -count=1` passes for the string length bootstrap slice.
- `sh scripts/selfhost_bootstrap_check.sh` passes for the string length bootstrap slice.
- Committed string length bootstrap slice as `a722c1a`.
- Added a stage-2 pipeline fixture for `text = "  hello  "; trimmed = trim text; print trimmed`.
- Focused `go test ./tests -run TestStage1SelfhostSourcesEmitC -count=1` passes for the trim slice.
- Fixed a Tya string interpolation issue from same-line generated C braces in the trim helper scan.
- `sh scripts/selfhost_check.sh` passes for the trim bootstrap slice.
- `go test ./... -count=1` passes for the trim bootstrap slice.
- `sh scripts/selfhost_bootstrap_check.sh` passes for the trim bootstrap slice.
- Committed string trim bootstrap slice as `a6958b6`.
- Added a stage-2 pipeline fixture for `text = "hello"; print contains text, "ell"`.
- Focused `go test ./tests -run TestStage1SelfhostSourcesEmitC -count=1` passes for the contains slice.
- `sh scripts/selfhost_check.sh` passes for the contains bootstrap slice.
- `go test ./... -count=1` passes for the contains bootstrap slice.
- `sh scripts/selfhost_bootstrap_check.sh` passes for the contains bootstrap slice.
- Committed string contains bootstrap slice as `c9dc175`.
- Added a stage-2 pipeline fixture for `startsWith` and `endsWith`.
- Focused `go test ./tests -run TestStage1SelfhostSourcesEmitC -count=1` passes for the prefix/suffix slice.
- `sh scripts/selfhost_check.sh` passes for the prefix/suffix bootstrap slice.
- `go test ./... -count=1` passes for the prefix/suffix bootstrap slice.
- `sh scripts/selfhost_bootstrap_check.sh` passes for the prefix/suffix bootstrap slice.

Remaining:

- Commit the verified stage-2 string prefix/suffix bootstrap slice.

Estimate: many iterations overall; current bootstrap slice less than 1 iteration.
