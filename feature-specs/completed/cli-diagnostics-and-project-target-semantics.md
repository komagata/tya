# Feature: CLI Diagnostics And Project Target Semantics

## Goal

Make CLI input modes, stdout/stderr boundaries, diagnostic rendering,
project-root discovery, target path resolution, signal handling, program args,
test filtering, failure output, and stack traces deterministic and testable.

## Context

This spec records accepted operational behavior for the current Tya toolchain.
It is not a static typing plan. It is intended for a future implementation pass
and therefore includes the tests that should be added with that implementation.

The behavior here complements:

- `feature-specs/unambiguous-dynamic-semantics.md`
- `feature-specs/dynamic-edge-semantics.md`
- `feature-specs/numeric-call-and-canonical-semantics.md`
- `feature-specs/control-display-and-platform-semantics.md`
- `feature-specs/package-class-concurrency-tooling-semantics.md`
- `feature-specs/release-dependency-native-security-semantics.md`

Self-host compatibility is required. Any implementation must preserve the v01
and v02 self-host fixed point.

## Behavior

- v1.0 does not include a REPL.
- v1.0 does not include `tya eval`.
- The canonical execution unit is a source file.
- `tya run -` reads source from stdin.
- Diagnostics for stdin source use `<stdin>` as the path.
- Relative imports for stdin source resolve from the current working directory.
- `tya check -` reads source from stdin.
- Successful machine-readable command output goes to stdout.
- Diagnostics, progress, and errors go to stderr.
- `--json` output goes to stdout.
- If JSON output cannot be produced because of an internal error, the internal
  error goes to stderr.
- Human-readable CLI output uses color only when stdout/stderr is a TTY.
- `NO_COLOR` disables color.
- `--no-color` disables color.
- The existing `--color=always|auto|never` diagnostic color option remains
  accepted for compatibility.
- Diagnostic messages are English.
- Diagnostic codes are stable and can be translated externally or documented in
  localized docs.
- Diagnostics are sorted by path, line, column, then code.
- Runtime errors are reported in occurrence order.
- JSON diagnostic output uses the existing stable NDJSON schema: diagnostic
  objects followed by a summary object.
- Each JSON diagnostic contains stable fields including `code`, `severity`,
  `message`, `primary`, `hints`, and `source`.
- `--quiet` produces no output on success.
- `--quiet` still prints minimal diagnostics on failure.
- `--quiet` does not change exit codes.
- `--verbose` writes operational details to stderr.
- Verbose details may include resolved imports, selected lib/runtime/native
  metadata, and C compiler commands.
- `--verbose` never pollutes stdout program output.
- Project root is the nearest ancestor directory containing `tya.toml`.
- If no `tya.toml` is found, the project root is the current working directory.
- User-provided relative target paths are resolved relative to the current
  working directory.
- User-provided target paths are not resolved relative to the project root.
- Project root is used for config discovery, dependency resolution, and task
  runner working directory behavior.
- CLI commands do not perform their own shell-style glob expansion for target
  paths.
- Shells may expand globs before invoking Tya.
- Task runner `watch` glob behavior remains separate and may use Tya-defined
  glob semantics.
- Ctrl-C interrupts running programs and builds.
- Ctrl-C exits with code `130`.
- On interruption, Tya removes temporary files when it can do so safely.
- Tya has no common CLI `--timeout` option.
- Timeout behavior may be added later to specific commands such as tests or
  task runner if specified separately.
- Program arguments for `tya run` may use `--` as the separator.
- `tya run file.tya -- arg1 arg2` makes `args()` return `["arg1", "arg2"]`.
- Passing program arguments without `--` remains supported for compatibility.
- `tya build` does not accept program arguments.
- Program arguments are passed to the generated executable at execution time.
- `tya test --filter PATTERN` filters tests by substring match on test name.
- `tya test --filter` does not use regular expressions.
- `tya test` failure output includes failed test name, file:line,
  expected/actual where available, and a minimal stack.
- Successful tests are not printed individually by default.
- Runtime errors include a Tya stack trace.
- Tya stack frames are rendered as `file:line:function`.
- C runtime/internal frames are hidden by default.
- C runtime/internal frames may be shown with `--verbose`.

## Scope

- Update `docs/SPEC.md` with these accepted CLI and diagnostic semantics.
- Update CLI command parsing for stdin targets, `--quiet`, `--verbose`,
  `--no-color`, `--filter`, `--`, and rejected argument forms where behavior
  differs.
- Update diagnostic sorting and JSON diagnostic schema.
- Update project-root discovery and target path handling where behavior differs.
- Update signal handling and temporary-file cleanup.
- Update test runner filtering and failure output.
- Update runtime stack trace rendering.
- Add focused unit tests and testscript coverage listed below.

## Out of Scope

- REPL.
- `tya eval`.
- Removing the existing `--color=always|auto|never` compatibility option.
- Regex test filters.
- Global CLI `--timeout`.
- Project-root-relative interpretation of user-provided target paths.
- Printing successful tests individually by default.

## Acceptance Criteria

- `tya run -` and `tya check -` read stdin and use `<stdin>` in diagnostics.
- Stdin relative imports resolve from current working directory.
- Human diagnostics/errors go to stderr.
- Program output and JSON output go to stdout.
- `NO_COLOR` and `--no-color` disable color.
- Diagnostic messages are English and sorted by path/line/column/code.
- `--json` diagnostic output has the documented stable NDJSON schema.
- `--quiet` suppresses success output but not failure diagnostics.
- `--verbose` writes operational details to stderr only.
- Nearest `tya.toml` determines project root.
- Without `tya.toml`, current working directory is the project root.
- Relative CLI targets are resolved against current working directory.
- CLI target globs are not expanded by Tya itself.
- Ctrl-C exits with code `130` and cleans temporary files when safe.
- No common `--timeout` option exists.
- `tya run file.tya -- arg1 arg2` passes args to `args()`.
- `tya run file.tya arg1` remains valid for compatibility.
- `tya build` rejects program args.
- `tya test --filter foo` uses substring matching.
- Test failure output includes name, location, expected/actual when available,
  and a minimal stack.
- Runtime stack traces show Tya frames by default and internal frames only in
  verbose mode.

## Tests To Add

CLI stdin tests:

- `TestRunReadsStdin`
  - `tya run -` executes source from stdin.
- `TestCheckReadsStdin`
  - `tya check -` validates source from stdin.
- `TestStdinDiagnosticsUseStdinPath`
  - Invalid stdin source reports `<stdin>`.
- `TestStdinRelativeImportsUseCwd`
  - Import from stdin resolves relative to current working directory.

Output and color tests:

- `TestCliStdoutStderrBoundaries`
  - Program output and JSON output use stdout; diagnostics use stderr.
- `TestNoColorEnvironment`
  - `NO_COLOR=1` disables ANSI color.
- `TestNoColorFlag`
  - `--no-color` disables ANSI color.
- `TestColorModeOption`
  - `--color=always`, `--color=auto`, and `--color=never` remain accepted.

Diagnostic tests:

- `TestDiagnosticLanguageEnglish`
  - Representative diagnostics use English message text.
- `TestDiagnosticSortOrder`
  - Multiple diagnostics are sorted by path, line, column, then code.
- `TestRuntimeErrorsOccurrenceOrder`
  - Runtime errors are reported in occurrence order.
- `TestJsonDiagnosticSchema`
  - JSON diagnostics include required diagnostic fields and a summary object.
- `TestQuietMode`
  - Success output is suppressed and failure diagnostics remain minimal.
- `TestVerboseModeWritesStderr`
  - Verbose import/build details go to stderr and stdout remains clean.

Project and target tests:

- `TestProjectRootNearestTyaToml`
  - Nearest ancestor `tya.toml` wins.
- `TestProjectRootDefaultsToCwd`
  - Without `tya.toml`, current working directory is project root.
- `TestRelativeTargetPathUsesCwd`
  - A relative CLI target is resolved from current working directory, not
    project root.
- `TestCliDoesNotExpandGlobs`
  - A literal glob target is not expanded by Tya.
- `TestTaskWatchGlobIsSeparate`
  - Task watch glob behavior remains covered separately.

Signal and timeout tests:

- `TestCtrlCExitCode`
  - Interrupted run/build exits with code `130`.
- `TestCtrlCCleansTemporaryFiles`
  - Temporary files are removed when interruption happens at a safe point.
- `TestNoCommonTimeoutFlag`
  - Common `--timeout` is rejected on commands without a command-specific
    timeout spec.

Argument tests:

- `TestRunArgsAfterSeparator`
  - `tya run file.tya -- arg1 arg2` makes `args()` return `["arg1", "arg2"]`.
- `TestRunArgsWithoutSeparatorStillSupported`
  - `tya run file.tya arg1` remains valid for compatibility.
- `TestBuildRejectsProgramArgs`
  - `tya build file.tya -- arg1` is invalid.

Test runner tests:

- `TestTestFilterSubstring`
  - `tya test --filter name` runs matching test names by substring.
- `TestTestFilterIsNotRegex`
  - Regex metacharacters are treated as literal substring text.
- `TestTestFailureOutput`
  - Failure output includes name, file:line, expected/actual when available, and
    minimal stack.
- `TestSuccessfulTestsNotPrintedByDefault`
  - Passing tests are summarized but not listed individually by default.

Stack trace tests:

- `TestRuntimeStackTraceTyaFrames`
  - Runtime errors show `file:line:function` Tya frames.
- `TestRuntimeStackTraceHidesInternalFrames`
  - Internal C/runtime frames are hidden by default.
- `TestVerboseStackTraceShowsInternalFrames`
  - Verbose mode may include internal frames.

Testscript coverage:

- Add CLI fixtures for stdin source, stdout/stderr separation, color disabling,
  quiet/verbose, project root discovery, relative target path resolution,
  program args, test filtering, failure output, and stack traces.
- Include interruption behavior where reliable in testscript; otherwise cover
  signal handling with a focused Go integration test.
- Include Windows path/executable suffix accommodations where relevant.

## Verification

```sh
gofmt -w cmd internal/cli internal/diagnostic internal/lsp internal/test internal/runtime tests
go test ./cmd/tya ./internal/diagnostic ./internal/lsp ./internal/test -count=1
go test ./tests -run 'TestCli|TestDiagnostics|TestRun|TestCheck|TestTest|TestStack' -count=1 -timeout=20m
go test ./tests -run 'TestExamplesGolden|TestV02Scripts|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
