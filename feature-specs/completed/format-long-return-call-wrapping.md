# Feature: Format long return call wrapping

## Goal

Make `tya format` wrap a long single-value `return` whose value is a call
expression using the same canonical multi-line call style used for assignment
right-hand sides and expression statements.

## Context

GitHub issue #26 reports that the formatter leaves a long line such as:

```tya
return rgba(color_parts[0].trim().to_i(), color_parts[1].trim().to_i(), color_parts[2].trim().to_i(), color_parts[3].trim().to_i())
```

on one line even though it exceeds the 80-column Formatted Syntax target.

`docs/SPEC.md` defines an 80-column column limit except for one unbreakable
atomic token. The formatter already has call wrapping helpers in
`internal/formatter/unparse.go` for long assignment RHS calls and long
expression-statement calls. `ReturnStmt` needs to use the same wrapping path
for a single returned call expression.

Relevant files include:

- `internal/formatter/unparse.go`
- `internal/formatter/unparse_test.go`
- `tests/cli_test.go`
- `docs/SPEC.md`
- affected formatted examples or stdlib sources such as `lib/color.tya`

## Behavior

- When a `return` statement has exactly one value and the rendered
  `return <call>` line exceeds the canonical column limit, format it as:

  ```tya
  return rgba(
    color_parts[0].trim().to_i(),
    color_parts[1].trim().to_i(),
    color_parts[2].trim().to_i(),
    color_parts[3].trim().to_i()
  )
  ```

- The wrapping style must match existing multi-line call formatting:
  - opening parenthesis stays on the `return` line;
  - each argument is on its own indented line;
  - the closing parenthesis aligns with the `return` indentation.
- Long return call chains should use the same call-chain wrapping behavior as
  expression statements and assignment RHS calls where that behavior exists.
- Short `return call(...)` lines that fit within the column limit stay on one
  line.
- `return` with no value remains `return`.
- `return` with multiple values remains governed by the existing formatter
  behavior and must not be newly wrapped by this feature unless that behavior
  already exists.
- Formatting must remain idempotent.

## Scope

- Update `ReturnStmt` formatting in `internal/formatter/unparse.go`.
- Add formatter unit tests for long single-value return calls, short return
  calls, and idempotency.
- Add or update CLI format coverage if existing format tests exercise the
  accepted-to-formatted syntax catalog.
- Reformat affected `.tya` files only when they currently contain the reported
  long return-call shape.

## Out of Scope

- Changing parser accepted syntax for `return`.
- Changing runtime return semantics.
- Introducing a configurable line length.
- Wrapping multi-value `return a, b, c` statements.
- Reworking unrelated formatter wrap rules for arrays, dictionaries, binary
  expressions, or expression statements.
- Updating archived versioned specs under `docs/v*/`.

## Acceptance Criteria

- The reproduction from issue #26 formats to the multi-line call form shown in
  this spec.
- A short `return rgb(0, 0, 0)` stays on one line.
- A return call chain that exceeds the column limit follows the existing
  formatter's call-chain wrap style.
- Running the formatter twice on the same source produces identical output.
- `tya format --check` reports drift before formatting the issue reproduction
  and passes after formatting it.
- Existing formatter and CLI tests continue to pass.
- The self-host fixed-point tests still pass.

## Verification

```sh
gofmt -w internal/formatter/unparse.go internal/formatter/unparse_test.go tests/cli_test.go
go test ./internal/formatter -run 'Return|Wrap|Idempotent' -count=1
go test ./tests -run 'TestCLIFormat|TestSelfhostV01Scripts' -count=1
go test ./... -count=1
```
