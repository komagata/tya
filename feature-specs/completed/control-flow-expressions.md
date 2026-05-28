# Feature: Control Flow Expressions

## Goal

Make `if`, `while`, `for`, and `match` usable as expressions while preserving
their existing statement behavior. This allows values such as `color = if ...
elseif ... else ...` without introducing temporary assignment variables in user
code.

## Context

Tya currently parses these forms as statements. `docs/SPEC.md` already describes
function bodies as implicitly returning the final evaluated statement or
expression, and the completed dynamic-edge semantics spec records intended
values for `if`, `for`, and `match`. This feature turns that value behavior into
the language surface by allowing control-flow forms anywhere an expression is
accepted.

The implementation must keep canonical syntax indentation-based and must keep
`elseif` as the canonical spelling. Existing statement-form programs must keep
parsing, checking, formatting, evaluating, and compiling the same way.

## Behavior

- `if` is an expression.
- `if ... else ...` is an expression.
- `if ... elseif ... else ...` is an expression.
- An `if` expression evaluates the selected branch body and returns that body's
  final value.
- If no `if` branch is selected and there is no `else`, the expression evaluates
  to `nil`.
- `while` is an expression.
- `while` evaluates to the last completed body value.
- A `while` body that never executes evaluates to `nil`.
- `break` exits the nearest loop and leaves the loop expression value as the
  last completed body value before the break, or `nil` if none exists.
- `continue` skips to the next iteration and discards the current iteration's
  partial value.
- `for ... in` is an expression.
- `for` evaluates to the last completed body value.
- A `for` body that never executes evaluates to `nil`.
- `break` and `continue` inside `for` follow the same value rules as `while`.
- `match` is an expression.
- `match` evaluates the selected case body and returns that body's final value.
- If no `match` case matches, the expression evaluates to `nil`.
- Control-flow expressions may appear in ordinary expression positions,
  including assignment RHS, return values, function-call arguments, array
  literals, dictionary values, and nested inside other control-flow expressions.
- Branch and loop bodies may still contain multiple statements. The final
  expression statement in a selected body is the body value.
- An explicit `return` inside a body keeps its current function-return behavior;
  it does not merely become the control-flow expression value.
- `raise` inside a body keeps its current error behavior.
- Bindings created inside control-flow bodies keep the current block scoping
  rules and do not leak outward.
- Statement position remains valid:

```tya
if ready
  print("ready")
```

- Expression position becomes valid:

```tya
label = if age >= 20
  "adult"
elseif age >= 13
  "teen"
else
  "young"
```

```tya
last = for item in items
  item.name
```

```tya
result = match status
case "ok"
  "success"
case _
  "fallback"
```

## Scope

- Update `docs/SPEC.md` to describe control-flow expressions and their values.
- Update parser and AST so `if`, `while`, `for`, and `match` can appear in
  expression position without removing statement-position support.
- Update checker traversal for the new expression nodes or unified
  statement/expression representation.
- Update eval interpreter behavior for expression-position control flow.
- Update C codegen so expression-position control flow produces a `TyaValue`
  result while preserving statement-position behavior.
- Update formatter/unparser to emit canonical multi-line control-flow
  expressions, including assignment RHS forms such as `name = if ...`.
- Add parser, checker, eval, codegen, and formatter tests covering each
  supported control-flow expression.
- Update self-host compiler support if its parser, AST, checker, or codegen
  has a separate representation for these constructs.

## Out of Scope

- `try/catch` expressions.
- `scope` expressions.
- `select` expressions.
- Pattern matching features beyond the current `match` case syntax.
- Changing the semantics of `return`, `raise`, `break`, or `continue` outside
  the value rules described above.
- Adding type inference or static branch type checking.
- Replacing indentation block syntax with inline ternary-style syntax.

## Acceptance Criteria

- `value = if cond ...` parses, checks, formats, evaluates, and compiles.
- `if ... elseif ... else ...` formats with canonical `elseif`, not nested
  `else if`.
- `if` with no selected branch and no `else` evaluates to `nil`.
- `while` and `for` expression values are the last completed body value.
- Empty `while` and empty `for` expression values are `nil`.
- `break` preserves the last completed loop body value and `continue` discards
  the current partial body value.
- `match` expression values come from the selected case body, and unmatched
  `match` evaluates to `nil`.
- Control-flow expressions work in at least assignment RHS, return value,
  function-call argument, array literal, dictionary value, and nested
  expression contexts.
- Existing statement-form control flow remains valid and behavior-compatible.
- Formatter output is idempotent for control-flow expressions.
- Self-host fixed point remains intact.

## Verification

```sh
go test ./internal/parser ./internal/checker ./internal/eval ./internal/codegen ./internal/formatter -count=1
go test ./tests -run 'TestSelfhostV01Scripts' -count=1
go test ./... -count=1
```
