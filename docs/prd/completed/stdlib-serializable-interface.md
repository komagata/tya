# Feature: Stdlib Serializable Interface

## Goal

Define a standard `Serializable` interface for values that can convert
themselves into data-only trees suitable for JSON, TOML, XML adapters, storage,
and message passing.

## Context

The existing `serialization.Serializer` supports primitive values, arrays,
dictionaries with string keys, bytes with explicit options, class public fields,
and hooks named `to_serialized()` / `Class.from_serialized(data)`. The hook is
useful but not represented as a stdlib interface. Interfaces currently cannot
require static members, so deserialization construction is intentionally outside
this interface.

## Behavior

- Add a standard interface:

  ```tya
  interface Serializable
    to_data = ->
  ```

- `to_data()` returns a data-only value made from:
  - `nil`
  - booleans
  - numbers
  - strings
  - bytes only when a downstream serializer explicitly supports them
  - arrays of data-only values
  - dictionaries with string keys and data-only values
- `to_data()` must not return arbitrary class instances, functions, tasks,
  channels, resources, or errors.
- `serialization.Serializer.to_data(value, options)` should prefer
  `value.to_data()` for values that implement `Serializable`.
- The existing `to_serialized()` hook remains supported for compatibility.
  If both `to_data()` and `to_serialized()` are present, `to_data()` is the
  canonical protocol and should be preferred by new docs.
- `Serializable` is for structured machine-readable data. Human-readable text
  remains `Stringable.to_s()`.
- This interface does not include `from_data` because current interfaces do not
  support static requirements.
- Deserialization remains handled by existing serializer APIs such as
  `Serializer.from_data(data, Class)` and `Class.from_serialized(data)`.

## Scope

- Add the stdlib interface file for `Serializable`.
- Update `serialization.Serializer.to_data` to recognize `to_data()` as the
  canonical hook while preserving `to_serialized()` compatibility.
- Update serialization docs to prefer `to_data()`.
- Add tests for classes implementing `Serializable`.
- Add tests for conflict/precedence when both hooks exist.
- Add tests rejecting unsupported `to_data()` results.

## Out of Scope

- Static `from_data` interface requirements.
- Replacing `Class.from_serialized(data)`.
- Removing `to_serialized()` compatibility.
- Schema validation.
- Cyclic graph serialization beyond existing serializer behavior.
- A dedicated `JsonSerializable` interface.
- Automatically serializing private fields.

## Acceptance Criteria

- `Serializable` is available as a stdlib interface.
- A class with `to_data()` can implement `Serializable`.
- A class without `to_data()` cannot implement `Serializable`.
- `Serializer.to_data(value, options)` uses `to_data()` for `Serializable`
  values.
- Existing `to_serialized()` tests continue to pass.
- If both `to_data()` and `to_serialized()` exist, `to_data()` is used and this
  precedence is documented.
- Unsupported values returned from `to_data()` raise serialization errors.
- Docs distinguish `Serializable.to_data()` from `Stringable.to_s()`.

## Verification

```sh
go test ./... -count=1
go test ./tests -run 'TestV11Scripts|TestV12Scripts' -count=1
```
