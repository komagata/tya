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

- [ ] Make the Go-emitted stage-1 self-host compiler run `examples/selfhost_ops.tya`
  - [ ] Reproduce and capture the current C type mismatch from stage-1 codegen
  - [ ] Preserve `INT` and `BOOL` assignment type information through Go-emitted `selfhost/codegen_c.tya`
  - [ ] Ensure stage-1 generated C prints bool/int values with the correct format
  - [ ] Add `examples/selfhost_ops.tya` to `scripts/selfhost_bootstrap_check.sh`
- [ ] Expand parser subset toward full expression parsing
  - [ ] Parse grouped comparison assignments
  - [ ] Parse simple `and` / `or` assignments
  - [ ] Parse two-or-more element array literals
  - [ ] Parse simple inline object literals
- [ ] Expand checker subset
  - [ ] Track block-local names for `if`, `while`, and `for`
  - [ ] Reject duplicate simple function params in self-host checker
  - [ ] Add parity coverage for duplicate and invalid names
- [ ] Expand self-host C codegen subset
  - [ ] Emit simple function bodies instead of comments for one-expression returns
  - [ ] Emit simple function calls to generated functions
  - [ ] Emit object placeholders beyond comments
- [ ] Bootstrap stage 2
  - [ ] Use stage-1 compiler binaries to compile selfhost sources to C
  - [ ] Compile stage-1 emitted selfhost C into stage-2 binaries
  - [ ] Compare stage-1 and stage-2 generated C for deterministic output

## Last Known Blocker

The normal interpreter-driven self-host pipeline can run `examples/selfhost_ops.tya`.
The Go-emitted stage-1 self-host pipeline currently fails when compiling the C
generated for `examples/selfhost_ops.tya`: bool/int variables such as `adult`,
`young`, `nonzero`, and `grouped` are emitted as values passed to `puts(...)`
instead of bool/int print paths.

That means the next work should focus on type preservation in the generated C
for `selfhost/codegen_c.tya` when that codegen itself is emitted by the Go C
emitter.
