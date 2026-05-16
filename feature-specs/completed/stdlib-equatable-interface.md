# Feature: Stdlib Equatable Interface

## Goal

Define a standard `Equatable` interface for values that provide domain equality
through `equal?(other)`.

## Context

Tya has `==` for normal runtime equality and `equal` for deep equality over
arrays and dictionaries. Several stdlib domain classes already expose static
`equal?` helpers, but there is no instance-level equality protocol that user
classes can implement.

## Behavior

- Add a standard interface:

  ```tya
  interface Equatable
    equal? = other ->
  ```

- `equal?(other)` returns `true` when `self` and `other` are equal under the
  receiver's domain semantics, otherwise `false`.
- Because the method name ends in `?`, non-boolean return values remain runtime
  errors under existing predicate-name rules.
- Primitive classes formally conform to `Equatable`:
  - `Number implements Equatable`
  - `String implements Equatable`
  - `Array implements Equatable`
  - `Dict implements Equatable`
  - `Boolean implements Equatable`
  - `Nil implements Equatable`
- Primitive `equal?` semantics must match existing equality behavior for the
  corresponding value kind. This feature must not change `==` or top-level
  `equal`.
- User classes may implement `Equatable` by defining `equal?(other)`.
- This feature does not automatically make `==` dispatch to `equal?`.
  Operator integration can be considered separately.
- Collection helpers that need equality may use `equal?` only where doing so is
  explicitly specified and tested. Existing primitive collection behavior must
  remain unchanged unless this spec names it.

## Scope

- Add the stdlib interface file for `Equatable`.
- Add primitive conformance metadata/documentation for core primitive classes.
- Add tests for user-defined classes implementing `Equatable`.
- Add tests that predicate enforcement rejects non-boolean `equal?` results.
- Update `docs/API.md`, `docs/SPEC.md`, and `docs/STDLIB.md`.

## Out of Scope

- Changing `==` semantics.
- Changing top-level `equal` semantics.
- Hash/equality contracts for dictionary keys or sets.
- Adding `Hashable`.
- Rewriting existing static `equal?` helpers in geometry/color/matrix packages.
- Deep-copy or structural comparison customization.

## Acceptance Criteria

- `Equatable` is available as a stdlib interface.
- A class with `equal?(other)` can implement `Equatable`.
- A class without `equal?(other)` cannot implement `Equatable`.
- `equal?` must return a boolean.
- Primitive equality behavior is unchanged.
- Docs clearly distinguish `==`, top-level `equal`, and `Equatable.equal?`.

## Verification

```sh
go test ./... -count=1
go test ./tests -run 'TestV11Scripts|TestV12Scripts|TestV19Scripts|TestV59Scripts' -count=1
```
