# Feature: Format long binary expression statements

## Goal

Make `tya format` wrap long binary expressions used directly as expression
statements with the same leading-operator style already used for long
assignment right-hand sides.

## Context

GitHub issue #24 reports that the formatter wraps long binary expressions on an
assignment RHS, but leaves the same expression on one line when it is used as a
standalone expression statement. This can happen in methods whose final
expression is a long boolean chain, such as `Color.nearly_equal?`.

`docs/SPEC.md` defines an 80-column column limit and says long operator chains
use formatter-defined continuation forms. The formatter already has binary
wrapping support in `internal/formatter/unparse.go`; expression statements must
route through that support when their rendered form exceeds the column limit.

Relevant files include:

- `internal/formatter/unparse.go`
- `internal/formatter/unparse_test.go`
- `tests/cli_test.go`
- `docs/SPEC.md`
- affected formatted sources such as `lib/color.tya`

## Behavior

- When an expression statement is a binary expression and the rendered line
  exceeds the canonical column limit, format it using the existing
  leading-operator style.
- The first operand remains on the original statement indentation.
- Continuation lines are indented one level and start with the binary operator.
- The output style must match existing assignment RHS binary wrapping.
- Short binary expression statements that fit within the column limit stay on
  one line.
- Mixed-precedence or unsupported binary trees should keep the existing
  formatter fallback behavior rather than changing associativity.
- Formatting must remain idempotent.

Example target shape:

```tya
abs(self.r - other.r) <= tolerance
  and abs(self.g - other.g) <= tolerance
  and abs(self.b - other.b) <= tolerance
  and abs(self.a - other.a) <= tolerance
```

## Scope

- Update expression-statement formatting in `internal/formatter/unparse.go`.
- Add formatter tests for a long expression-statement binary chain, a short
  binary expression statement, and idempotency.
- Add or update CLI format coverage if existing format tests exercise the
  accepted-to-formatted syntax catalog.
- Reformat affected `.tya` files only when they currently contain the reported
  long expression-statement shape.

## Out of Scope

- Changing parser accepted syntax for binary expressions.
- Changing binary operator precedence or associativity.
- Introducing a configurable line length.
- Reworking binary wrapping for assignment RHS, return statements, conditions,
  arrays, dictionaries, or calls.
- Updating archived versioned specs under `docs/v*/`.

## Acceptance Criteria

- The reproduction from issue #24 formats to the leading-operator multi-line
  form shown in this spec.
- A short expression statement such as `a == b` stays on one line.
- A long binary expression statement and a long assignment RHS binary
  expression use the same continuation style.
- Running the formatter twice on the same source produces identical output.
- `tya format --check` reports drift before formatting the issue reproduction
  and passes after formatting it.
- Existing formatter and CLI tests continue to pass.
- The self-host fixed-point tests still pass.

## Verification

```sh
gofmt -w internal/formatter/unparse.go internal/formatter/unparse_test.go tests/cli_test.go
go test ./internal/formatter -run 'Binary|Wrap|Idempotent' -count=1
go test ./tests -run 'TestCLIFormat|TestSelfhostV01Scripts' -count=1
go test ./... -count=1
```
