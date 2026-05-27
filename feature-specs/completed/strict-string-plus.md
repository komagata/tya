# Feature: Strict String Plus

## Goal

Make `+` stop performing implicit string formatting for mixed operands. String
conversion for embedded values should happen through string interpolation only,
while `+` remains valid for same-domain addition or concatenation.

## Context

Tya is dynamically typed but aims for strict semantics with no implicit
conversions. Current behavior still allows expressions such as `"count: " + 1`
because `+` formats through string conversion when either operand is a string.
This is documented in `docs/SPEC.md` and `docs/STRICT_SEMANTICS.md`, and is
implemented in both the Go interpreter and C runtime.

The current self-host compilers rely on mixed string `+` in diagnostic and C
emitter string construction. A direct runtime change breaks the maintained
self-host invariant, so implementation must first remove those dependencies
from `selfhost/v01/` and `selfhost/v02/`.

## Behavior

- `String + String` remains valid and returns `String`.
- `Number + Number` remains valid and returns a number according to existing
  numeric rules.
- `Bytes + Bytes` remains valid and returns concatenated bytes.
- `String + non-String` is invalid.
- `non-String + String` is invalid.
- String interpolation remains the only accepted way to format arbitrary values
  into strings.

```tya
print("a" + "b")       # valid
print(1 + 2)           # valid
print(b"a" + b"b")     # valid
print("count: {count}") # valid

print("count: " + count) # invalid
print(count + " items")  # invalid
```

## Scope

- Update `selfhost/v01/` and `selfhost/v02/` so they no longer depend on mixed
  string `+`; use string interpolation for value formatting in generated
  messages and emitted C fragments.
- Update Go interpreter `+` behavior in `internal/eval`.
- Update C runtime/codegen behavior for `+` so `tya run` and `tya build`
  agree.
- Update `docs/SPEC.md` and `docs/STRICT_SEMANTICS.md`.
- Add or update focused tests for interpreter behavior, build/runtime behavior,
  and strict semantics testscript coverage.
- Update existing examples/tests/lib/selfhost sources that currently rely on
  mixed string `+`.

## Out of Scope

- No changes to string interpolation behavior.
- No changes to `print` display behavior.
- No new conversion API.
- No static typing work.
- No changes to operators other than `+`.

## Acceptance Criteria

- `String + String` passes in interpreter and built executable paths.
- `Number + Number` passes in interpreter and built executable paths.
- `Bytes + Bytes` passes in interpreter and built executable paths.
- `String + non-String` fails in interpreter and built executable paths.
- `non-String + String` fails in interpreter and built executable paths.
- String interpolation can format representative non-string values.
- `selfhost/v01/compiler.tya` fixed-point tests pass.
- `selfhost/v02/` fixed-point tests pass.
- Existing examples, stdlib tests, and strict semantics tests are updated to the
  new rule.

## Verification

```sh
go test ./internal/eval ./internal/codegen ./internal/checker -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestSelfhostV02Scripts -count=1 -timeout=20m
go test ./tests -run TestV65Scripts -count=1
go test ./... -count=1 -timeout=20m
```
