# Feature: No-Go Selfhost Bootstrap Proof

## Goal

Add a local proof script that uses a released `tya` binary to rebuild the
self-hosted compiler and prove the stage-2/stage-3 fixed point without invoking
Go.

## Context

Tya's v1.0.0 direction is to make Go unnecessary for normal users building the
compiler from official release artifacts. The repository already maintains
`selfhost/v01/` and `selfhost/v02/` fixed-point gates, but those gates still run
inside Go tests and use the Go implementation as the orchestration layer.

This feature is the first narrow step toward a no-Go bootstrap path. It does not
remove the Go implementation, change the default compiler, or add release/CI
automation. It creates an explicit local contract that can later be wired into
CI, release packaging, and broader conformance checks.

## Behavior

- Add `scripts/bootstrap_no_go.sh`.
- The script targets `selfhost/v02/compiler.tya`.
- The script requires `TYA_BOOTSTRAP_TYA=/path/to/tya`.
- The script rejects a missing, non-executable, or failing bootstrap binary with
  a clear error.
- The script prepends a shim directory to `PATH` containing a failing `go`
  executable so any accidental Go invocation fails immediately.
- The script leaves normal native tools such as `cc`, `diff`, `mktemp`, and
  `timeout` available from the caller's environment.
- The script runs the following stages:
  - Use the bootstrap `tya` binary to compile `selfhost/v02/compiler.tya` into
    a stage-2 compiler.
  - Use the stage-2 compiler to compile `selfhost/v02/compiler.tya` into a
    stage-3 compiler.
  - Compare the generated stage-2 and stage-3 C output byte-for-byte.
- The script prints the bootstrap binary path, temporary work directory, stage
  names, and failing command context to stderr.
- Generated files live under a `mktemp -d` directory.
- On success, generated files are removed.
- On failure, generated files are removed unless `TYA_KEEP_BOOTSTRAP_TMP=1` is
  set.
- When `TYA_KEEP_BOOTSTRAP_TMP=1` is set, the script prints the retained
  directory path.

## Scope

- `scripts/bootstrap_no_go.sh`
- Tests that verify script behavior without requiring a network download.
- Documentation comments or focused test fixtures needed to explain the local
  bootstrap contract.

## Out of Scope

- Removing `cmd/tya` or `internal/*` Go sources.
- Making the self-hosted compiler the default `tya` compiler.
- Downloading bootstrap binaries from GitHub Releases.
- Adding GitHub Actions jobs.
- Hiding Go globally from the developer machine.
- Proving the full compiler conformance suite without Go.
- Comparing diagnostics or runtime behavior beyond the stage-2/stage-3 generated
  C fixed point.
- Supporting `selfhost/v01/` as the target of this proof.

## Acceptance Criteria

- Running `TYA_BOOTSTRAP_TYA=/path/to/tya scripts/bootstrap_no_go.sh` attempts a
  v02 stage-2/stage-3 fixed-point proof without invoking Go.
- If any command tries to execute `go`, the script fails through the shim and
  identifies the no-Go violation.
- If `TYA_BOOTSTRAP_TYA` is unset, missing, or non-executable, the script fails
  before doing stage work and explains the required variable.
- If stage-2 and stage-3 generated C differ, the script fails and reports the
  comparison step.
- If the proof succeeds, the script exits 0 and removes its temporary directory.
- If the proof fails with `TYA_KEEP_BOOTSTRAP_TMP=1`, the script leaves the
  temporary directory in place and prints its path.
- The implementation does not modify selfhost source semantics.
- Existing selfhost fixed-point tests remain valid.

## Tests To Add

Script tests:

- `TestBootstrapNoGoRequiresBootstrapBinary`
  - Runs `scripts/bootstrap_no_go.sh` without `TYA_BOOTSTRAP_TYA`.
  - Expects a non-zero exit and an error mentioning `TYA_BOOTSTRAP_TYA`.

- `TestBootstrapNoGoRejectsNonExecutableBootstrapBinary`
  - Points `TYA_BOOTSTRAP_TYA` at a non-executable file.
  - Expects a non-zero exit and an error identifying the bootstrap binary as
    unusable.

- `TestBootstrapNoGoInstallsGoShim`
  - Uses a small fake bootstrap command that tries to run `go`.
  - Expects a non-zero exit and an error showing the no-Go shim caught the
    invocation.

- `TestBootstrapNoGoKeepsTempOnFailureWhenRequested`
  - Runs the script with a fake bootstrap that fails after work directory
    creation and `TYA_KEEP_BOOTSTRAP_TMP=1`.
  - Expects stderr to include the retained directory path and that the directory
    still exists after failure.

- `TestBootstrapNoGoRemovesTempOnFailureByDefault`
  - Runs the same failure path without `TYA_KEEP_BOOTSTRAP_TMP`.
  - Expects the printed work directory path no longer exists.

End-to-end proof:

- `TestBootstrapNoGoSelfhostV02FixedPoint`
  - Builds or locates a local `tya` binary for test setup.
  - Runs `TYA_BOOTSTRAP_TYA=<binary> scripts/bootstrap_no_go.sh`.
  - Expects exit 0 and stderr showing stage-2, stage-3, and fixed-point compare
    steps.
  - The test setup may use Go to produce the bootstrap binary; the script under
    test must still fail if it invokes Go after installing its shim.

## Verification

```sh
go test ./tests -run 'TestBootstrapNoGo|TestSelfhostV02Scripts' -count=1 -timeout=20m
TYA_BOOTSTRAP_TYA="$(command -v tya)" scripts/bootstrap_no_go.sh
```
