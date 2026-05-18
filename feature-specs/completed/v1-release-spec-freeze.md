# Feature: v1.0.0 Release Specification Freeze

## Goal

Freeze the remaining non-obvious release-level language specification decisions
needed to ship Tya v1.0.0 with a defensible language contract, self-host story,
diagnostic model, platform target, and compatibility policy.

## Context

`docs/SPEC.md` and `docs/STRICT_SEMANTICS.md` already define the current v1
language surface. The completed v1 feature specs freeze strict dynamic
semantics, syntax exclusions, runtime boundaries, and the unified structured
error model.

This spec records the final release-level decisions that affect how those
language specs become a v1.0.0 product promise. The accepted direction keeps
Tya's "no hesitation" principle: one documented behavior, explicit failures,
stable diagnostics, and a no-Go self-host bootstrap path without forcing a risky
same-release deletion of the Go implementation.

## Behavior

- v1.0.0 ships with the self-hosted compiler as the primary compiler direction,
  while the Go implementation remains in the repository as a documented
  reference and bootstrap recovery path.
  - A released `tya` binary must be able to rebuild the checked-in self-hosted
    compiler and prove a latest-spec fixed point without requiring Go on the
    user's machine.
  - Removing `cmd/tya` and `internal/*` Go implementation sources is not a
    v1.0.0 requirement.
  - Remaining Go-only behavior must be tracked as a self-host parity gap.
- The self-hosted compiler's v1.0.0 acceptance target is the full language
  surface described by `docs/SPEC.md`.
  - A subset-only self-host compiler is not sufficient for v1.0.0.
  - Any temporary self-host limitation found during implementation must block
    the release or be resolved before the v1.0.0 tag.
- v1.0.0 compatibility covers the standard-library and package APIs documented
  in `docs/SPEC.md`.
  - Language-required stdlib APIs and documented public stdlib surface are part
    of the v1 compatibility contract.
  - HTTP expansion, GUI wrappers, native external libraries, and ecosystem
    packages that are not part of the documented core v1 language surface are
    post-v1 compatibility work unless explicitly promoted into `docs/SPEC.md`.
- Pre-v1 behavior that contradicts `docs/SPEC.md` is corrected before v1.0.0
  without a deprecation window.
  - The fix must include a clear diagnostic where the old behavior becomes
    invalid.
  - The release notes or migration documentation must describe the user-visible
    change and the accepted replacement.
- All user-facing diagnostics across parser, checker, codegen, runtime, CLI,
  LSP, formatter, package manager, and release/bootstrap tooling use stable
  diagnostic codes for v1.0.0.
  - Runtime failures must expose stable codes in addition to stable messages.
  - Diagnostic JSON output must include the same stable code as the human
    diagnostic.
  - LSP diagnostics must match CLI diagnostic codes and messages for the same
    source failure.
- v1.0.0 release artifacts target Linux, macOS, and Windows on x86_64 and
  arm64.
  - Linux and macOS must run the no-Go bootstrap proof in release verification.
  - Windows must pass installer, build, run, and packaging smoke coverage for
    v1.0.0.
  - A Windows no-Go bootstrap proof is desirable but is not required to block
    the initial v1.0.0 release.
- The WebAssembly target remains documented for v1.0.0, but it is not a
  release-blocking target.
  - WASM behavior must not contradict `docs/SPEC.md`.
  - WASM-specific gaps must be documented separately from the core language
    release gates.
- `docs/STRICT_SEMANTICS.md` gains an explicit dynamic-allowances section.
  - The section documents behavior that remains valid precisely because Tya is
    dynamically typed, including runtime-kind checks, valid `nil` positions,
    identity equality for non-collection reference values, deep equality for
    arrays and dictionaries, and runtime errors for operations whose operand
    kinds are only known during execution.
- v1.x compatibility guarantees cover accepted syntax, documented public APIs,
  documented runtime behavior, stable diagnostic codes, CLI JSON diagnostic
  schema, and release artifact semantics.
  - Undocumented implementation details, unsupported experimental flags,
    internal package layout, generated C internals, external package internals,
    and behavior outside `docs/SPEC.md` are not compatibility guarantees.

## Scope

- `docs/SPEC.md`
- `docs/STRICT_SEMANTICS.md`
- `ROADMAP.md`
- `docs/VERSIONS.md` and v1.0.0 release notes or migration notes
- self-host coverage manifest and fixed-point verification scripts
- parser, checker, codegen, runtime, CLI, LSP, formatter, package manager, and
  release/bootstrap diagnostics
- release packaging and installer verification for Linux, macOS, and Windows
- testscript fixtures and docs-contract tests that enforce the release contract

## Out of Scope

- Deleting the Go implementation at v1.0.0.
- Shipping a subset-only self-host compiler as the v1.0.0 self-host story.
- Promoting HTTP expansion, GUI wrappers, or external native packages into the
  v1 compatibility contract unless they are explicitly documented in
  `docs/SPEC.md`.
- Making WASM a v1.0.0 release-blocking target.
- Adding new syntax or static types.
- Preserving a deprecation window for behavior that contradicts the accepted
  v1 specification.

## Acceptance Criteria

- `docs/SPEC.md` states the v1.0.0 compatibility boundary and does not promise
  compatibility for undocumented implementation details or post-v1 ecosystem
  packages.
- `docs/STRICT_SEMANTICS.md` includes a dynamic-allowances section that
  distinguishes valid dynamic behavior from invalid implicit conversion.
- `ROADMAP.md` states that v1.0.0 keeps the Go implementation as a
  reference/bootstrap recovery path while requiring a no-Go self-host bootstrap
  proof from release artifacts.
- A self-host coverage manifest maps every `docs/SPEC.md` language feature to
  self-host lexer, parser, AST, checker, C emitter, runtime, and fixture
  coverage.
- The v1.0.0 release gate fails if any documented `docs/SPEC.md` language
  feature is unsupported by the self-host compiler.
- All user-facing failures from parser, checker, codegen, runtime, CLI, LSP,
  formatter, package manager, and release/bootstrap tooling expose stable
  diagnostic codes.
- CLI diagnostic JSON and LSP diagnostics report the same stable code used by
  the human-readable diagnostic.
- Tests prove that pre-v1 behavior contradictory to `docs/SPEC.md` is rejected
  with a clear diagnostic and documented migration path.
- Release verification covers Linux, macOS, and Windows x86_64 and arm64
  artifacts, with no-Go bootstrap proof required on Linux and macOS.
- WASM target documentation clearly marks it as documented but not
  release-blocking for v1.0.0.
- Existing self-host v01 and v02 fixed-point gates remain valid until replaced
  by the latest-spec v1.0.0 fixed-point gate.

## Tests To Add

Documentation contract tests:

- `TestSpecDocumentsV1CompatibilityBoundary`
  - Expected: `docs/SPEC.md` documents the v1.x compatibility guarantees and
    explicitly excludes undocumented internals and post-v1 ecosystem packages.

- `TestStrictSemanticsDocumentsDynamicAllowances`
  - Expected: `docs/STRICT_SEMANTICS.md` contains a dynamic-allowances section
    covering runtime-kind checks, valid `nil` positions, equality behavior, and
    dynamic runtime errors.

- `TestRoadmapDocumentsV1GoRecoveryPath`
  - Expected: `ROADMAP.md` states that Go remains as a reference/bootstrap
    recovery path for v1.0.0 and that no-Go bootstrap proof remains required.

- `TestSpecDocumentsWasmAsNonBlockingTarget`
  - Expected: v1 docs describe WASM as documented but not release-blocking.

Self-host and release tests:

- `TestSelfhostCoverageManifestCoversSpec`
  - Reads the self-host coverage manifest.
  - Expected: every tracked `docs/SPEC.md` language feature has explicit
    lexer, parser, AST, checker, C emitter, runtime, and fixture status.
  - Expected: any unsupported documented v1 language feature fails the v1.0.0
    release gate.

- `TestReleaseNoGoBootstrapRequiredOnUnix`
  - Exercises the release verification script with Go hidden from `PATH` on
    Linux or macOS.
  - Expected: a released `tya` binary rebuilds the checked-in self-host
    compiler and proves the latest-spec fixed point.

- `TestReleaseWindowsSmokeCoverage`
  - Exercises Windows release artifact smoke fixtures.
  - Expected: installer, `tya build`, `tya run`, and package layout checks pass
    for supported Windows architectures without requiring the Windows no-Go
    bootstrap proof.

Diagnostic tests:

- `TestAllUserFacingDiagnosticsHaveStableCodes`
  - Audits parser, checker, codegen, runtime, CLI, formatter, package manager,
    and bootstrap diagnostic fixtures.
  - Expected: every user-facing failure includes a stable diagnostic code.

- `TestDiagnosticJsonAndLspCodesMatchCli`
  - Runs representative failures through CLI human output, CLI JSON output, and
    LSP diagnostics.
  - Expected: stable diagnostic codes and messages match for the same source
    failure.

- `TestRuntimeErrorsExposeStableCodes`
  - Runs representative runtime failures for wrong kind, wrong arity, invalid
    state, closed resource, closed channel send, invalid UTF-8, and non-error
    `raise`.
  - Expected: each error exposes a stable diagnostic code and stable message.

Migration tests:

- `TestPreV1ContradictionsRejectedWithMigrationDocs`
  - Covers representative pre-v1 behaviors that conflict with `docs/SPEC.md`.
  - Expected: each invalid program fails with a stable diagnostic code and is
    referenced by release notes or migration documentation.

Testscript coverage:

- `v1_release_spec_freeze.txtar`
  - Covers CLI-level diagnostics for representative parser, checker, codegen,
    runtime, formatter, package manager, and bootstrap failures.
  - Covers `--json` diagnostic code stability.
  - Covers accepted dynamic allowances such as truthiness, valid `nil`
    positions, deep equality, and runtime-kind errors.

- `v1_release_artifacts.txtar`
  - Covers artifact metadata, supported platform declarations, managed Zig
    availability, and release bootstrap command behavior.

## Verification

```sh
go test ./tests -run 'TestSpecDocumentsV1CompatibilityBoundary|TestStrictSemanticsDocumentsDynamicAllowances|TestRoadmapDocumentsV1GoRecoveryPath|TestSpecDocumentsWasmAsNonBlockingTarget' -count=1
go test ./tests -run 'TestSelfhostCoverageManifestCoversSpec|TestReleaseNoGoBootstrapRequiredOnUnix|TestReleaseWindowsSmokeCoverage' -count=1 -timeout=20m
go test ./tests -run 'TestAllUserFacingDiagnosticsHaveStableCodes|TestDiagnosticJsonAndLspCodesMatchCli|TestRuntimeErrorsExposeStableCodes|TestPreV1ContradictionsRejectedWithMigrationDocs' -count=1 -timeout=20m
go test ./tests -run 'TestV.*Scripts|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
