# Self-Host Work Protocol

This file is the restart point for autonomous self-hosting work.

## Non-Stop Protocol

When asked to continue self-hosting work:

1. Read `SELFHOST_WORK.md`, then `ROADMAP.md`.
2. Pick the first unchecked task in `Current Queue`.
3. Implement the smallest useful slice that moves that task forward.
4. Add or update focused tests for that slice.
5. Run the focused test first.
6. Run `go test ./... -count=1`.
7. Run `sh scripts/selfhost_bootstrap_check.sh`.
8. Update `ROADMAP.md` and this file when a slice is complete.
9. Commit with `Masaki Komagata <komagata@gmail.com>`.
10. Return to step 2 without waiting for confirmation.

Only stop for a true blocker:

- tests cannot be made to pass after a bounded fix attempt
- a design choice would invalidate existing language semantics
- the work requires external input that cannot be inferred from the repository

Do not stop merely because a commit was made. A commit is a checkpoint, not an
endpoint.

## Current Queue

- [x] Make the Go-emitted stage-1 self-host compiler run `examples/selfhost_ops.tya`
  - [x] Reproduce and capture the current C type mismatch from stage-1 codegen
  - [x] Preserve `INT` and `BOOL` assignment type information through Go-emitted `selfhost/codegen_c.tya`
  - [x] Ensure stage-1 generated C prints bool/int values with the correct format
  - [x] Add `examples/selfhost_ops.tya` to `scripts/selfhost_bootstrap_check.sh`
- [ ] Expand parser subset toward full expression parsing
  - [x] Parse grouped comparison assignments
  - [x] Parse simple `and` / `or` assignments
  - [x] Parse two-or-more element array literals
  - [x] Parse simple inline object literals
- [ ] Expand checker subset
  - [x] Track block-local names for `if`, `while`, and `for`
  - [x] Reject duplicate simple function params in self-host checker
  - [x] Add parity coverage for duplicate and invalid names
- [ ] Expand self-host C codegen subset
  - [x] Emit simple function bodies instead of comments for one-expression returns
  - [x] Emit simple function calls to generated functions
  - [x] Emit object placeholders beyond comments
- [ ] Bootstrap stage 2
  - [x] Use stage-1 compiler binaries to compile selfhost sources to C
  - [ ] Compile stage-1 emitted selfhost C into stage-2 binaries
  - [ ] Compare stage-1 and stage-2 generated C for deterministic output

## Last Resolved Blocker

The Go C emitter used to drop user-defined function call statements such as
`rememberType names, types, name, "INT"`. That made the Go-emitted stage-1
self-host codegen lose type metadata and emit `puts(adult)` for bool/int
values. Generic user function call statements are now emitted and
`scripts/go_emit_selfhost_ops_check.sh` covers the stage-1 path.
