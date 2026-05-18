# Feature: v1 Runtime Boundaries

## Goal

Freeze v1.0.0 runtime-boundary decisions for numbers, text/binary data, error
values, cleanup, channels, tasks, resources, and assertions.

## Context

`feature-specs/v1-language-semantics-freeze.md` records high-impact semantic
decisions, and `feature-specs/v1-language-syntax-boundaries.md` records syntax
that v1.0.0 intentionally excludes. This spec records runtime-boundary
decisions where Tya must stay explicit and deterministic while remaining
dynamically typed.

The accepted direction is to avoid context-sensitive coercion, avoid hidden
resource lifetime behavior, and keep concurrency semantics small enough to be
portable across the C runtime and future self-hosted compiler.

## Behavior

- `number` remains a single runtime kind.
  - Tya does not split runtime numeric values into separate `Int` and `Float`
    kinds for v1.0.0.
  - `1 == 1.0` remains true.
  - Operations that require integer-compatible numbers, such as `%`, bitwise
    operators, shifts, and indexes, validate integer compatibility at runtime
    or at the earliest feasible static layer.
- `/` always performs number division.
  - `5 / 2` evaluates to `2.5`.
  - Integer division uses an explicit API such as `div()`.
- `NaN`, `Infinity`, `nan`, and `infinity` are not numeric literal spellings.
  - They remain ordinary identifiers where identifiers are valid.
  - Non-finite values, if supported, are exposed through explicit standard
    library APIs such as `Math.nan()` or `Math.infinity()`.
- String indexing is by Unicode rune.
  - Byte-level indexing uses bytes values.
  - Grapheme-cluster indexing is not part of v1.0.0.
- String and bytes values never convert implicitly.
  - String-producing APIs reject invalid UTF-8.
  - Bytes-producing APIs preserve raw data without UTF-8 validation.
  - Explicit conversion APIs document whether invalid UTF-8 raises an error or
    is handled by a named replacement strategy.
- Error values are a distinct runtime kind.
  - `error(message)` creates an error value.
  - Errors are not strings.
  - Error display uses the message.
  - Error values may carry optional structured data when documented by an API.
- `catch err` catches structured error values raised with `raise`.
  - `raise` accepts only error values.
  - Typed catches, pattern catches, and multiple catch clauses are not part of
    v1.0.0.
  - Branching by error kind, message, code, or data is written inside the catch
    body with `if` or `match`.
- `defer` is not part of v1.0.0.
  - Cleanup uses `try/finally`, explicit `close()`, and structured `scope`
    behavior.
- Closed-channel behavior is fixed.
  - Receiving from a closed channel returns `nil`.
  - Sending to a closed channel raises a runtime error.
- Task cancellation is not standardized for v1.0.0.
  - There is no language-level cancellation token or cancel API.
  - `scope` defines structured lifetime and waits for child tasks before
    leaving.
  - Cancellation may be introduced later by an explicit accepted spec.
- Resource finalization is explicit.
  - Programs must not rely on GC finalizers for correctness.
  - Resource APIs expose `close()` or an equivalent documented explicit cleanup
    method.
  - `try/finally` is the language-level cleanup mechanism.
- `assert` is not language syntax in v1.0.0.
  - Assertions live in unittest or standard-library APIs.
  - There is no build-mode-dependent assertion syntax.

## Scope

- `docs/SPEC.md`
- `docs/STRICT_SEMANTICS.md`
- numeric, text, bytes, error, channel, task, and resource runtime behavior
- parser/checker diagnostics for excluded syntax such as `defer` and `assert`
- interpreter, codegen, C runtime, and CLI fixtures for runtime parity
- self-host sources only where needed to preserve fixed-point gates

## Out of Scope

- Adding static numeric types.
- Adding non-finite numeric literals.
- Adding grapheme-cluster indexing.
- Adding typed catch, pattern catch, or multiple catch clauses.
- Adding `defer`.
- Adding standardized task cancellation.
- Adding finalizer-dependent resource semantics.
- Adding language-level `assert` syntax.

## Acceptance Criteria

- `docs/SPEC.md` documents the accepted runtime boundaries without
  contradictory wording.
- `docs/STRICT_SEMANTICS.md` maps each runtime-boundary rule to an active test,
  diagnostic, or runtime error.
- Interpreter and generated C behavior agree for:
  - single-kind numeric equality and division;
  - integer-only numeric operations;
  - non-finite identifier handling;
  - rune-based string indexing;
  - invalid UTF-8 rejection for string APIs;
  - bytes raw-data preservation;
  - error values as non-string runtime values;
  - catch-all structured error handling;
  - closed-channel receive and send behavior;
  - explicit resource cleanup.
- Parser/checker diagnostics reject excluded `defer`, typed/pattern catch,
  multiple catch clauses, standardized cancellation syntax, and language-level
  `assert`.
- Existing self-host fixed-point gates remain valid.

## Tests To Add

Parser/checker tests:

- `TestParseRejectsNonFiniteNumericLiterals`
  - `value = NaN`
  - `value = Infinity`
  - Expected: parsed as identifiers or rejected as undefined names, never as
    numeric literals.

- `TestParseRejectsTypedPatternAndMultipleCatch`
  - Representative snippets for typed catch, pattern catch, and multiple catch
    clauses.
  - Expected: parser diagnostics.

- `TestParseRejectsDefer`
  - `defer cleanup()`
  - Expected: parser diagnostic that `defer` is not part of Tya.

- `TestParseRejectsLanguageAssert`
  - `assert ready`
  - Expected: parser diagnostic or reserved-name invalidity.

- `TestParseRejectsCancellationSyntax`
  - Representative snippets for language-level cancel tokens or cancel blocks.
  - Expected: parser diagnostics.

Eval/runtime tests:

- `TestRunNumberKindAndDivisionBoundaries`
  - `1 == 1.0` is true.
  - `5 / 2` is `2.5`.
  - `%`, bitwise operators, shifts, and indexes reject non-integer-compatible
    numbers.

- `TestRunStringIndexingUsesRunes`
  - Indexes a multi-byte UTF-8 string by rune.
  - Expected: each index returns the documented character, not raw bytes.

- `TestRunStringBytesConversionBoundaries`
  - Invalid UTF-8 text conversion raises an error.
  - Bytes APIs preserve invalid UTF-8 bytes.

- `TestRunErrorValuesAreNotStrings`
  - `err = error("failed")`
  - Expected: `err` has error kind, displays as the message, and is not equal
    to `"failed"`.

- `TestRunCatchCatchesStructuredErrorsOnly`
  - Raises an error value.
  - Attempts to raise a string, number, object value, and `nil`.
  - Expected: the error value is caught by `catch err`; non-error operands are
    rejected before catch handling.

- `TestRunClosedChannelBoundaries`
  - Receiving from a closed channel returns `nil`.
  - Sending to a closed channel raises a runtime error.

- `TestRunScopeWaitsWithoutCancellation`
  - A scope with child tasks waits for completion.
  - Expected: no implicit cancellation token or automatic task cancellation
    behavior is observed.

- `TestRunResourceCleanupUsesFinally`
  - A resource-like test double records `close()`.
  - Expected: cleanup happens through explicit `finally`, not finalizer timing.

Testscript coverage:

- `v1_runtime_boundaries.txtar`
  - Covers CLI-level agreement for numeric boundaries, text/bytes boundaries,
    error values, catch behavior, closed channels, and explicit cleanup.

- `v1_excluded_runtime_syntax.txtar`
  - Covers rejection for `defer`, typed/pattern/multiple catch, cancellation
    syntax, and language-level `assert`.

## Verification

```sh
go test ./internal/parser -count=1
go test ./internal/checker -count=1
go test ./internal/eval -count=1
go test ./internal/codegen -count=1
go test ./tests -run 'TestV.*Scripts|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
