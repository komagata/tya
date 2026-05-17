# Feature: Package, Class, Concurrency, And Tooling Semantics

## Goal

Make package resolution, class field rules, interface arity, task/channel
failure behavior, test discovery, diagnostics, formatting, documentation, C
output stability, runtime error classification, and experimental-feature policy
deterministic and testable.

## Context

This spec records accepted behavior for the current dynamically typed Tya
surface. It is not a static typing plan. It is intended for a future
implementation pass and therefore includes the tests that should be added with
that implementation.

The behavior here complements:

- `feature-specs/unambiguous-dynamic-semantics.md`
- `feature-specs/dynamic-edge-semantics.md`
- `feature-specs/numeric-call-and-canonical-semantics.md`
- `feature-specs/control-display-and-platform-semantics.md`

Self-host compatibility is required. Any implementation must preserve the v01
and v02 self-host fixed point.

## Behavior

- Import/package resolution order is fixed:
  1. current project;
  2. locked dependencies;
  3. `TYA_PATH`;
  4. bundled standard library.
- The first matching package or file wins.
- Local packages may shadow standard-library packages without an error.
- Local package shadowing is intentional and supports replacement,
  dependency-injection, and project-local overrides.
- When `tya.lock` exists, locked dependency versions are authoritative.
- `tya.toml` dependency declarations do not silently override `tya.lock`.
- If `tya.toml` and `tya.lock` disagree, the user must run the documented lock
  update/install command before the changed dependency graph is used.
- Import cycles are invalid.
- `tya check` reports an import-cycle diagnostic instead of recursing forever.
- Private helper classes declared in a class file are visible only inside that
  same file.
- Private helper classes are not visible from other files in the same directory
  package.
- Reading a missing instance field is a runtime error.
- Missing fields do not silently evaluate to `nil`.
- A field intended to contain no value must be initialized explicitly with
  `nil`.
- Instance fields may be declared in the class body with initializers.
- Instance fields may also be created by assignment to `self.<name>` inside
  instance methods and constructors.
- Top-level `self.field = value` is invalid because `self` is only valid inside
  instance methods and constructors.
- External object field assignment follows the current runtime object mutation
  rules and is not made a static declaration check by this feature.
- Private-member access is rejected by `tya check` when statically knowable.
- Private-member access that can only be detected at runtime is a runtime
  error.
- Interface method requirements use exact arity.
- A class method that implements an interface requirement must have the same
  arity as the requirement.
- Default parameters do not satisfy an interface arity mismatch.
- A task that raises stores the raised value.
- `await task` re-raises a value raised inside the task.
- Receiving from a closed channel returns `nil`.
- Sending to a closed channel is a runtime error.
- Task cancellation is cooperative.
- Tya does not force-kill cancelled tasks.
- Awaiting a cancelled task returns or raises a documented cancellation error.
- `tya test` discovers only files named `*_test.tya`.
- Test declarations inside ordinary non-test `.tya` files are not discovered
  automatically.
- `tya test` runs test files in ascending file path order.
- Tests inside one file run in definition order.
- `tya test` does not run tests in parallel unless a future explicit option
  requests parallelism.
- `tya check` reports multiple errors when recovery is possible.
- If parser recovery is not possible, `tya check` may stop at the unrecoverable
  parser error.
- Any `tya check` validation failure exits with code `1`.
- `tya format` never rewrites a file that cannot be parsed.
- Formatting invalid source reports an error and exits without modifying the
  file.
- LSP diagnostics use the same diagnostic codes and messages as `tya check`.
- LSP-only alternate wording for the same source validity issue is invalid.
- A doc comment attaches only to the immediately following class, interface,
  method, function, or constant.
- A blank line between a doc comment and a declaration breaks attachment.
- Orphan doc comments are documentation diagnostics.
- Generated C is byte-for-byte stable for the same source and same toolchain
  version.
- Generated C does not embed absolute paths, timestamps, random identifiers, or
  other nondeterministic data unless explicitly requested by a debug option.
- User-code failures are language runtime errors.
- Compiler or runtime implementation bugs are internal errors.
- Internal errors use a distinct exit code from ordinary user-code failures.
- v1.0 has no experimental feature gates.
- Only behavior accepted into the SPEC is implemented as language behavior.
- If experimental features are introduced later, they must require an explicit
  flag such as `--experimental-name`.

## Scope

- Update `docs/SPEC.md` with these accepted semantics.
- Update `docs/STRICT_SEMANTICS.md` once active tests exist.
- Update package resolver and lockfile handling where behavior differs.
- Update checker/import-loader cycle diagnostics.
- Update class checker/runtime behavior for private helpers, field creation
  through `self`, missing field reads, and explicit `nil` initialization.
- Update interface checker behavior for exact arity with default parameters.
- Update task/channel runtime behavior.
- Update `tya test` discovery and ordering behavior where needed.
- Update CLI diagnostics and exit-code behavior where needed.
- Update formatter behavior for invalid source.
- Update LSP diagnostic plumbing to share checker/parser diagnostic wording.
- Update doc comment extraction/diagnostics.
- Update C emitter to avoid nondeterministic output.
- Add focused unit tests and testscript coverage listed below.

## Out of Scope

- Static typing syntax, generics, overloads, or typed interfaces.
- Stable dictionary iteration.
- Parallel test execution without an explicit future option.
- Warning-only deprecation lifecycle.
- Experimental feature flags for v1.0.
- Making local package shadowing an error.

## Acceptance Criteria

- Import resolution follows current project, locked dependencies, `TYA_PATH`,
  then bundled stdlib.
- A local package with the same path as stdlib wins without an error.
- `tya.lock` controls dependency versions when present.
- Import cycles fail with a clear diagnostic.
- Private helper classes are usable in their declaring file and invisible from
  sibling package files.
- Reading a missing field is a runtime error.
- `self.field = nil` makes a field explicitly initialized to `nil`.
- `self.field = value` inside constructors and instance methods creates or
  updates instance fields.
- Top-level `self.field = value` is rejected.
- Static private access errors are caught by `tya check`; dynamic private
  access errors fail at runtime.
- Interface implementation arity must match exactly.
- Default parameters do not make an arity mismatch acceptable.
- `await` re-raises task failures.
- `receive` on a closed channel returns `nil`.
- `send` on a closed channel is a runtime error.
- Cancellation is cooperative and await reports a documented cancellation
  error.
- `tya test` discovers only `*_test.tya`.
- `tya test` order is deterministic: path order, then definition order.
- `tya check` reports multiple recoverable errors and exits with code `1`.
- `tya format` does not rewrite invalid source.
- LSP diagnostics match `tya check` diagnostic code and message.
- Doc comments attach only across no blank line.
- Generated C is stable for repeated runs.
- User runtime errors and internal errors use distinct classifications and exit
  codes.
- No experimental feature is available without SPEC acceptance.

## Tests To Add

Package/import tests:

- `TestResolveImportPriority`
  - Fixture with local, dependency, `TYA_PATH`, and stdlib candidates proves
    priority order.
- `TestResolveLocalPackageShadowsStdlib`
  - Local package with stdlib path wins without diagnostic.
- `TestLockfileVersionWins`
  - `tya.lock` dependency version is used when it differs from `tya.toml`.
- `TestLockfileMismatchRequiresUpdate`
  - Changed `tya.toml` without lock update reports the documented command or
    diagnostic.
- `TestCheckRejectsImportCycle`
  - Import cycle reports a clear cycle path and terminates.

Class/interface tests:

- `TestPrivateHelperClassFileVisibility`
  - Helper class is usable in the same class file and rejected from sibling
    files.
- `TestRunMissingFieldReadErrors`
  - Reading a field never assigned or declared with value errors.
- `TestRunExplicitNilFieldRead`
  - Field explicitly initialized to `nil` reads as `nil`.
- `TestCheckAllowsSelfFieldCreationInInstanceMethods`
  - `self.unknown = value` inside `initialize` or an instance method is valid.
- `TestCheckRejectsTopLevelSelfFieldAssignment`
  - `self.unknown = value` outside a method fails because `self` is undefined.
- `TestCheckPrivateMemberAccess`
  - Statically known private access fails at check time.
- `TestRunPrivateMemberAccess`
  - Dynamic private access fails at runtime.
- `TestCheckInterfaceArityExactMatch`
  - Matching arity passes and mismatching arity fails.
- `TestCheckInterfaceArityIgnoresDefaultParameters`
  - Default parameters do not satisfy a required arity mismatch.

Task/channel tests:

- `TestRunAwaitReraisesTaskFailure`
  - Task raise is observed by `await`.
- `TestRunReceiveClosedChannelReturnsNil`
  - Receive after close returns `nil`.
- `TestRunSendClosedChannelErrors`
  - Send after close raises/runtime-errors as documented.
- `TestRunTaskCancellationIsCooperative`
  - Cancellation does not force-kill immediately and await reports the
    documented cancellation result.

Tooling tests:

- `TestTyaTestDiscoversOnlyTestFiles`
  - `*_test.tya` is discovered and ordinary `.tya` files are not.
- `TestTyaTestOrderIsDeterministic`
  - Path order and definition order are observable in output.
- `TestTyaCheckReportsMultipleRecoverableErrors`
  - Multiple independent errors are reported and exit code is `1`.
- `TestTyaCheckStopsOnUnrecoverableParserError`
  - Unrecoverable parse failure stops safely with exit code `1`.
- `TestTyaFormatDoesNotRewriteInvalidSource`
  - Invalid file contents remain byte-for-byte unchanged.
- `TestLspDiagnosticsMatchTyaCheck`
  - Diagnostic code and message match CLI check output.
- `TestDocCommentAttachment`
  - No blank line attaches; blank line creates an orphan doc diagnostic.
- `TestGeneratedCStable`
  - Two builds from the same source/toolchain produce identical C output.
- `TestGeneratedCNoNondeterministicMetadata`
  - Generated C has no absolute path, timestamp, or random id in ordinary mode.
- `TestRuntimeErrorVsInternalErrorExitCodes`
  - User runtime error and forced internal error fixtures use distinct exit
    codes.
- `TestNoExperimentalFeatureWithoutSpec`
  - Experimental flags or syntax are rejected unless explicitly added by a
    later SPEC.

Testscript coverage:

- Add or extend package resolver fixtures for local/stdlib shadowing, lockfile
  priority, and import cycles.
- Add class fixtures for private helper visibility, declared fields, `self`
  field creation, missing field reads, and explicit `nil` field initialization.
- Add concurrency fixtures for task failure propagation and closed-channel
  behavior.
- Add CLI fixtures for `tya test`, `tya check`, `tya format`, LSP diagnostics,
  doc generation, generated C stability, and exit-code classification.
- Include compiled C coverage for class fields, interfaces, tasks/channels, and
  generated C stability where applicable.

## Verification

```sh
gofmt -w internal/parser internal/checker internal/eval internal/codegen internal/format internal/lsp internal/doc internal/resolver cmd runtime
go test ./internal/parser ./internal/checker ./internal/eval ./internal/codegen ./internal/format ./internal/lsp ./internal/doc -count=1
go test ./tests -run 'TestV44Scripts|TestV65Scripts|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./tests -run 'TestExamplesGolden|TestV02Scripts|TestStdlibBinaryScript' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
