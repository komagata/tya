# Feature: v1 Unified Error Handling

## Goal

Make Tya v1.0.0 error handling use one user-facing model:
`raise` / `try` / `catch` / `finally` with structured error values.

## Context

The current specification permits multiple error-handling styles:

- language-level `raise`, `try`, `catch`, and `finally`;
- `try` as an expression;
- public APIs that may return `value, err` pairs;
- raising any non-`nil` value.

That gives users multiple ways to represent the same failure path. Tya's design
goal is to avoid hesitation, so v1.0.0 should provide one canonical error
model. Multiple return values remain useful for ordinary multi-value results,
but they are no longer the public error convention.

## Behavior

- User-facing error handling is `raise` / `try` / `catch` / `finally`.
- `try` is a statement only.
  - `try` expressions are not part of v1.0.0.
  - A function that needs to recover and return a value should use an ordinary
    local binding, `try/catch` statement, and final expression or `return`.
- Public standard-library and package APIs do not use `value, err` pairs as the
  error convention.
  - External failures such as file, process, network, parse, compression,
    digest, time, random, native-wrapper, and serialization failures raise
    structured error values.
  - Programmer errors such as invalid argument kind, wrong arity,
    out-of-range writes, invalid state, and closed-resource misuse also raise
    structured error values.
- Multiple return values remain valid for ordinary successful values.
  - `min, max = bounds(items)` is valid.
  - `value, err = read_file(path)` is not a valid public API shape for
    v1.0.0.
- `raise` accepts only error values.
  - `raise error("failed")` is valid.
  - `raise "failed"`, `raise 1`, `raise { message: "failed" }`, and
    `raise nil` are invalid.
- `catch err` catches raised error values.
  - `catch err` is the only catch syntax.
  - typed catch, pattern catch, value-filtered catch, and multiple catch clauses
    are not part of v1.0.0.
  - Branching by error details is written inside the catch body with ordinary
    `if` or `match` statements.
- Error values are a distinct runtime kind.
  - Required public fields:
    - `message`: human-readable message string;
    - `kind`: broad machine-readable category string;
    - `code`: specific machine-readable code string;
    - `data`: dictionary with structured context;
    - `cause`: another error value or `nil`.
  - Error display uses `message`.
  - Error equality follows ordinary object/error identity unless an explicit
    method documents another comparison.
- `error(message, options = {})` is the public constructor.
  - `error("failed")` is valid and fills default `kind`, `code`, `data`, and
    `cause` values.
  - `error("not found", { kind: "io", code: "file_not_found", data: { path:
    path }, cause: err })` is valid.
  - Unknown option keys are invalid.
  - `message`, `kind`, and `code` must be strings.
  - `data` must be a dictionary.
  - `cause` must be an error or `nil`.
- `finally` remains part of the language.
  - Cleanup uses `try/finally`.
  - `defer` remains outside v1.0.0.
- Error-to-value conversion is a library concern, not syntax.
  - A standard-library helper such as `Result.capture(fn)` may convert a raised
    error into a value shape.
  - Tya does not add `Result` syntax, `?` propagation syntax, or a second
    language-level error channel.

## Scope

- `docs/SPEC.md`
- `docs/STRICT_SEMANTICS.md`
- parser and checker diagnostics for invalid `try` expressions, invalid
  `raise` operands, and unsupported catch forms
- evaluator, codegen, C runtime, and standard-library API behavior for
  structured error values
- standard-library docs and fixtures that currently describe `value, err`
  public APIs
- examples and tests that use `try` expressions or `value, err` as an error
  convention
- self-host sources only where required to preserve fixed-point gates

## Out of Scope

- Removing multiple return values.
- Adding static typed exceptions.
- Adding typed catch, pattern catch, multiple catch clauses, or catch filters.
- Adding `defer`.
- Adding `Result` syntax or `?` propagation syntax.
- Preserving a deprecation window for public `value, err` error APIs.

## Acceptance Criteria

- `docs/SPEC.md` presents `raise` / `try` / `catch` / `finally` as the only
  language-level user-facing error model.
- `docs/SPEC.md` no longer presents `try` expressions as valid.
- `docs/SPEC.md` no longer allows public APIs to choose between raising and
  returning `value, err` pairs implicitly.
- Public standard-library APIs that can fail document raised structured errors.
- `raise` accepts only error values and rejects all non-error values.
- `catch err` catches structured error values and binds the exact error value.
- `finally` behavior remains unchanged and is the canonical cleanup construct.
- Multiple return values remain valid for non-error multi-value results.
- Interpreter and generated C behavior agree for covered error construction,
  raising, catching, and invalid raise operands.
- Existing self-host fixed-point gates remain valid.

## Tests To Add

Parser/checker tests:

- `TestParseRejectsTryExpression`
  - A function body uses `value = try ... catch ...`.
  - Expected: parser diagnostic that `try` is a statement only.

- `TestCheckRejectsRaiseNonErrorValues`
  - `raise "failed"`
  - `raise 1`
  - `raise { message: "failed" }`
  - `raise nil`
  - Expected: checker or runtime diagnostics requiring an error value.

- `TestParseRejectsUnsupportedCatchForms`
  - typed catch;
  - pattern catch;
  - multiple catch clauses;
  - catch filters.
  - Expected: parser diagnostics.

- `TestCheckRejectsPublicValueErrStdlibShape`
  - Standard-library public API fixtures expose a `value, err` failure
    convention.
  - Expected: documentation or API audit failure.

Eval/runtime tests:

- `TestRunErrorConstructorDefaults`
  - `err = error("failed")`
  - Expected: `message == "failed"`, default `kind`, default `code`, empty
    `data`, and `cause == nil`.

- `TestRunErrorConstructorOptions`
  - Creates an error with `kind`, `code`, `data`, and `cause`.
  - Expected: fields match and invalid option types raise constructor errors.

- `TestRunRaiseCatchStructuredError`
  - Raises `error("failed", { kind: "io", code: "file_not_found" })`.
  - Expected: `catch err` receives the same error value and can inspect fields.

- `TestRunRaiseNonErrorFails`
  - Raises string, number, dictionary, object, and `nil`.
  - Expected: each fails with a clear diagnostic or runtime error.

- `TestRunFinallyRemainsCleanupConstruct`
  - `finally` runs for normal completion, `raise`, `return`, `break`, and
    `continue`.
  - Expected: existing finally behavior remains stable.

- `TestRunMultiReturnStillValidForOrdinaryValues`
  - `min, max = bounds(items)` style fixture.
  - Expected: multiple successful return values remain valid.

Testscript coverage:

- `v1_unified_error_handling.txtar`
  - Covers valid `raise error(...)`, `try/catch`, `try/finally`, error field
    inspection, and ordinary multi-return.
  - Covers invalid `try` expression, non-error raise operands, unsupported catch
    forms, and public `value, err` error API examples.

- `v1_stdlib_error_contract.txtar`
  - Covers representative file, JSON parse, network or process, compression,
    and serialization failures raising structured error values rather than
    returning `value, err` pairs.

Documentation tests:

- `TestSpecDocumentsSingleErrorModel`
  - Ensures `docs/SPEC.md` does not document `try` expressions or public
    `value, err` error APIs as valid v1.0.0 behavior.

- `TestStdlibDocsUseStructuredRaisedErrors`
  - Ensures public stdlib failure docs use raised structured errors and include
    `kind` and `code` where meaningful.

## Verification

```sh
go test ./internal/parser -count=1
go test ./internal/checker -count=1
go test ./internal/eval -count=1
go test ./internal/codegen -count=1
go test ./tests -run 'TestV.*Scripts|TestSelfhostV01Scripts|TestSelfhostV02Scripts|TestSpecDocumentsSingleErrorModel|TestStdlibDocsUseStructuredRaisedErrors' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
