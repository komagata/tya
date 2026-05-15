# Feature: Stdlib Comparable Interface

## Goal

Define a standard `Comparable` interface for values that can provide total
ordering through a single `compare(other)` method and derived predicate
defaults.

## Context

Arrays already expose `sort()` and `sort_by(fn)`, and primitive numbers and
strings support ordering operators. There is no stdlib interface for user
classes to advertise ordering behavior. Ruby's `Comparable`, Java's
`Comparable`, and Scala ordering patterns all point to a single comparison
primitive with derived convenience operations.

## Behavior

- Add a standard interface:

  ```tya
  interface Comparable
    compare = other ->

    lt? = other ->
      self.compare(other) < 0

    lte? = other ->
      self.compare(other) <= 0

    gt? = other ->
      self.compare(other) > 0

    gte? = other ->
      self.compare(other) >= 0

    between? = min, max ->
      self.gte?(min) and self.lte?(max)
  ```

- `compare(other)` returns a negative number when `self` sorts before `other`,
  `0` when they are equal for ordering, and a positive number when `self` sorts
  after `other`.
- `compare(other)` should raise a runtime error when `other` cannot be compared
  to `self`.
- The derived predicate defaults must require boolean results because their
  names end in `?`.
- Primitive `Number` and `String` formally conform to `Comparable`.
- Primitive conformance must preserve existing primitive representation and
  ordering performance.
- This feature does not automatically make `<`, `<=`, `>`, or `>=` dispatch to
  user-defined `compare`. Operator integration can be a later feature.
- This feature may update stdlib collection helpers to prefer `compare` where a
  `Comparable` value is explicitly expected, but existing primitive `sort()`
  behavior must remain unchanged.

## Scope

- Add the stdlib interface file for `Comparable`.
- Add checker/runtime documentation for primitive `Number` and `String`
  conformance.
- Add tests for default methods on user-defined comparable classes.
- Add tests for invalid missing `compare` implementations.
- Update docs describing the distinction between `compare` and ordering
  operators.

## Out of Scope

- Generic `Comparable<T>` typing.
- Operator overloading.
- Changing cross-type primitive ordering rules.
- Replacing `sort_by(fn)`.
- Locale-aware string collation.
- Partial ordering or `NaN`-specific ordering policy beyond existing number
  behavior.

## Acceptance Criteria

- `Comparable` is available as a stdlib interface.
- A class implementing `compare(other)` receives working default `lt?`,
  `lte?`, `gt?`, `gte?`, and `between?` methods.
- A class missing `compare(other)` cannot implement `Comparable`.
- `lt?`, `lte?`, `gt?`, `gte?`, and `between?` return booleans.
- Primitive number and string ordering behavior remains unchanged.
- Docs state that operators do not yet dispatch to user-defined `compare`.

## Verification

```sh
go test ./... -count=1
go test ./tests -run 'TestV11Scripts|TestV12Scripts|TestV61Scripts|TestV59Scripts' -count=1
```
