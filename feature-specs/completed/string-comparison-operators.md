# Feature: String Comparison Operators

## Goal
Allow Tya programs to compare strings directly with `<`, `<=`, `>`, and `>=` using deterministic Unicode code point ordering.

## Context
`docs/SPEC.md` currently says ordering operators require numbers and that string ordering is not defined by these operators. The lexer and parser already accept the comparison operators, and the checker/evaluator/codegen already handle numeric ordering. String equality already works through `==` and `!=`. `String.compare` is already available as an explicit primitive method and should remain compatible.

## Behavior
- `<`, `<=`, `>`, and `>=` accept two string operands.
- String ordering is lexicographic by Unicode code point sequence.
- Prefix behavior follows lexicographic ordering:
  - `"a" < "aa"` is `true`.
  - `"aa" < "b"` is `true`.
  - equal strings satisfy `<=` and `>=`, but not `<` or `>`.
- Non-ASCII strings compare by code point, not locale collation.
- Numeric ordering behavior remains unchanged.
- Equality behavior remains unchanged.
- Mixed runtime kinds remain invalid for ordering:
  - `"1" < 2` is invalid.
  - `"a" < b"a"` is invalid.
- Checker diagnostics should accept obviously string-vs-string ordering and keep rejecting obviously invalid mixed primitive ordering.
- Runtime errors should still catch invalid dynamic comparisons when operand kinds are only known at runtime.
- C emission and interpreter behavior must match.

## Scope
- Update the language spec in `docs/SPEC.md` and Japanese/current v1 spec copies if they mirror the ordering rule.
- Update checker logic for `<`, `<=`, `>`, and `>=` to allow string-vs-string comparisons.
- Update interpreter evaluation for string ordering.
- Update C code generation/runtime support for string ordering.
- Add focused checker, interpreter, codegen, and testscript coverage.
- Include examples covering ASCII, prefix ordering, equality boundaries, non-ASCII code point ordering, and mixed-type errors.

## Out of Scope
- Locale-aware collation.
- Case-insensitive ordering.
- Natural sort ordering such as `"file2" < "file10"`.
- Bytes ordering.
- User-defined operator overloading or dispatching ordering operators to `compare`.
- Changes to `String.compare`, `Comparable`, `==`, or `!=`.

## Acceptance Criteria
- `"a" < "b"`, `"a" <= "a"`, `"b" > "a"`, and `"b" >= "b"` evaluate to `true`.
- `"a" > "b"` and `"a" < "a"` evaluate to `false`.
- `"a" < "aa"` and `"aa" < "b"` evaluate to `true`.
- Non-ASCII strings compare deterministically by Unicode code point.
- `tya check` accepts string-vs-string ordering.
- `tya check` rejects obvious mixed primitive ordering such as `"a" < 1`.
- Runtime execution raises an ordering type error for dynamic mixed-kind comparisons.
- `tya run` and compiled executables produce the same results.
- Existing numeric comparison behavior and tests continue to pass.

## Verification
```sh
go test ./internal/checker ./internal/eval ./internal/codegen -count=1
go test ./tests -run 'TestV65Scripts|string_comparison' -count=1
go test ./... -count=1
```
