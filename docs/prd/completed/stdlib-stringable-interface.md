# Feature: Stdlib Stringable Interface

## Goal

Define a standard `Stringable` interface for values that provide a stable
string representation through `to_s()`.

## Context

Every runtime value already exposes `to_s()` as a primitive/common method, and
stdlib code frequently uses `value.to_s()` for formatting, logging, JSON/TOML
emission, and error messages. There is no stdlib interface that lets user
classes explicitly declare that contract.

Primitive classes in Tya are wrapper class singletons over tagged `TyaValue`
runtime kinds. Treating primitives as `Stringable` must not box primitive
values.

## Behavior

- Add a standard interface:

  ```tya
  interface Stringable
    to_s = ->
  ```

- `Stringable.to_s()` returns a string.
- The method must be side-effect free for ordinary formatting use. The checker
  does not enforce purity, but the docs must state this expectation.
- Primitive classes formally conform to `Stringable`:
  - `Number implements Stringable`
  - `String implements Stringable`
  - `Array implements Stringable`
  - `Dict implements Stringable`
  - `Boolean implements Stringable`
  - `Nil implements Stringable`
- Primitive conformance is a language/runtime conformance rule and must preserve
  the existing tagged `TyaValue` representation.
- User classes may implement `Stringable` by defining `to_s()`.
- `Stringable` is for human-readable representation, not structured
  serialization.

## Scope

- Add the stdlib interface file for `Stringable`.
- Teach the checker/runtime documentation that primitive classes conform to
  `Stringable` where interface conformance is checked or surfaced.
- Add tests for user classes implementing `Stringable`.
- Add tests for primitive values being accepted where `Stringable` conformance
  is required, if the checker exposes primitive interface conformance in this
  feature.
- Update `docs/API.md`, `docs/SPEC.md`, and `docs/STDLIB.md`.

## Out of Scope

- Changing the output format of existing `to_s()` methods.
- Adding `inspect`, `debug_s`, or pretty-printing protocols.
- Using `Stringable` for JSON/TOML/XML serialization.
- Boxing primitive values into ordinary objects.
- Enforcing purity of `to_s()` in the checker.

## Acceptance Criteria

- `Stringable` is available as a stdlib interface.
- A class with `to_s()` can implement `Stringable`.
- A class without `to_s()` cannot implement `Stringable`.
- Existing primitive `to_s()` behavior is unchanged.
- Primitive `value.class` behavior remains unchanged.
- Docs clearly state that `Stringable` is human-readable and `Serializable` is
  the structured data protocol.

## Verification

```sh
go test ./... -count=1
go test ./tests -run 'TestV11Scripts|TestV12Scripts|TestV59Scripts' -count=1
```
