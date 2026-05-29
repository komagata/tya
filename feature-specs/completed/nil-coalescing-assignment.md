# Feature: Nil-Coalescing Assignment

## Goal
Add a `??=` assignment operator that assigns the right-hand expression only when the assignment target currently evaluates to `nil`. This gives Tya a concise, explicit way to initialize optional state without treating `false`, `0`, or `""` as missing.

## Context
Tya currently supports ordinary assignment with `=` for bindings, fields, and indexed collection entries. Tya truthiness makes only `nil` and `false` falsey, but this feature is specifically about `nil`, not falsey values.

Other languages use similar syntax for nullish assignment:

- JavaScript has `??=` and only evaluates/assigns the right operand when the left side is nullish (`null` or `undefined`).
- C# has `??=` for assignment when the left side is `null`.
- Ruby's `||=` is not the right model for Tya because Ruby assigns for both `nil` and `false`, while this feature must preserve explicit `false`.

The recommended Tya spelling is therefore `??=`, not `||=`.

## Behavior
- `target ??= value` is valid wherever ordinary single-target assignment to `target = value` is valid.
- `target` may be:
  - a local or top-level binding name,
  - an instance or static field/member assignment target such as `self.name`,
  - a member assignment target such as `object.name` when ordinary assignment allows it,
  - an indexed assignment target such as `items[index]` or `dict[key]` when ordinary assignment allows it.
- `target ??= value` reads the current target value once.
- If the current target value is `nil`, the right-hand expression is evaluated and assigned to `target`.
- If the current target value is not `nil`, no assignment happens and the right-hand expression is not evaluated.
- `false`, `0`, `0.0`, `""`, `[]`, and `{}` are not `nil`; they must not trigger assignment.
- The expression value of `target ??= value` is the final target value:
  - the newly assigned value when the target was `nil`,
  - the existing value when the target was not `nil`.
- As a statement, the expression value may be ignored.
- `??=` must follow the same binding, kind-preservation, field declaration, constant, capture, and mutability rules as ordinary assignment.
- `??=` is single-target only. Multiple assignment such as `a, b ??= pair()` is invalid.
- `??=` is not a general nil-coalescing expression operator. This feature does not add `??`.
- The formatter emits exactly one space around `??=`.

Examples:

```tya
name = nil
name ??= "guest"
print(name) # guest

enabled = false
enabled ??= true
print(enabled) # false

count = 0
count ??= 1
print(count) # 0
```

Short-circuiting:

```tya
calls = 0
fallback = ->
  calls = calls + 1
  "fallback"

name = "set"
name ??= fallback()
print(name)  # set
print(calls) # 0
```

Fields and indexed targets:

```tya
class Config
  name: nil

  initialize: ->
    self.name ??= "default"

items = [nil]
items[0] ??= "first"

user = {}
user["name"] ??= "guest"
```

## Scope
- Add a lexer token for `??=`.
- Update parser and AST representation so nil-coalescing assignment is distinguishable from ordinary assignment.
- Update formatter/unparser to preserve and canonicalize `??=`.
- Update checker rules so `??=` uses the same target validation and type/kind rules as ordinary single-target assignment.
- Update code generation and runtime/eval paths so `??=` short-circuits and evaluates the assignment target only once where observable.
- Update LSP parsing, diagnostics, semantic behavior, and any code actions affected by assignment syntax.
- Update current docs in `docs/SPEC.md` and Japanese docs if they describe assignment syntax or token vocabulary.
- Add focused tests for lexer/parser, formatter, checker/codegen/eval behavior, and CLI script-level behavior.

## Out of Scope
- No `??` expression operator.
- No `||=` operator.
- No `&&=` or other compound assignment operators.
- No arithmetic compound assignment such as `+=`.
- No multi-target nil-coalescing assignment.
- No compatibility rewrite from `x = x or y` or any other idiom.
- No change to Tya truthiness.

## Acceptance Criteria
- `value = nil; value ??= "fallback"` assigns `"fallback"`.
- `value = "set"; value ??= "fallback"` keeps `"set"`.
- `false`, `0`, `""`, empty arrays, and empty dictionaries do not trigger assignment.
- The right-hand expression is not evaluated when the target is not `nil`.
- `??=` works for local bindings, `self.field`, normal member assignment targets supported by `=`, array indexes, and dictionary indexes.
- `??=` rejects invalid assignment targets with the same style of diagnostic as ordinary assignment.
- `??=` rejects constants and other immutable targets consistently with ordinary assignment.
- `??=` preserves the existing strict reassignment kind rules.
- `??=` is invalid for multiple assignment.
- Formatter output canonicalizes spacing to `target ??= value`.
- Docs describe `??=` as nil-only assignment and explicitly distinguish it from Ruby-style falsey `||=`.

## Verification
```sh
go test ./internal/lexer ./internal/parser ./internal/formatter ./internal/checker ./internal/codegen ./internal/eval ./internal/lsp ./tests -count=1
go test ./... -count=1
```
