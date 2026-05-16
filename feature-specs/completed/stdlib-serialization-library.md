---
status: completed
goal_ready: false
---

# Feature: Serialization Stdlib Library

## Goal

Add a standard `serialization` library that converts Tya values and class
instances to stable data shapes and back, so save files, config snapshots,
asset metadata, caches, and small network payloads can be implemented without
each application hand-writing object mapping code.

## Context

Tya already has `json.Json` and `toml.Toml` for parsing and dumping data trees.
Those libraries operate on primitive values, arrays, and dictionaries. Planned
stdlib work increasingly returns class instances for domain values, which makes
plain JSON/TOML dumping less convenient for save data and application state.

The first serialization library should be explicit and predictable. It should
not try to serialize arbitrary closures, native resources, streams, sockets, or
cycles.

## Behavior

- Add a public `serialization` stdlib package.
- Import shape:

  ```tya
  import serialization as serialization

  data = serialization.Serializer.to_data(player)
  text = serialization.Serializer.to_json(player)
  loaded = serialization.Serializer.from_json(text, Player)
  ```

- Public class:
  - `serialization.Serializer`
- Supported input values:
  - `nil`
  - booleans
  - numbers
  - strings
  - bytes, when encoded through an explicit option
  - arrays
  - dictionaries with string keys
  - class instances with serializable public fields
- Serialization output from `to_data` is composed only of primitives, arrays,
  and dictionaries that `json.Json.dump` can emit.
- Deserialization into a class returns class instances, not dictionaries.
- Unsupported values raise clear `serialization` errors naming the unsupported
  type or field.

## API

- `Serializer.to_data(value)` converts a value to a JSON-compatible data tree.
- `Serializer.to_data(value, options)` supports options.
- `Serializer.from_data(data, klass)` creates an instance of `klass` from a data
  tree.
- `Serializer.from_data(data, klass, options)` supports options.
- `Serializer.to_json(value)` returns compact JSON text.
- `Serializer.to_json(value, options)` supports options.
- `Serializer.from_json(text, klass)` parses JSON and creates an instance of
  `klass`.
- `Serializer.from_json(text, klass, options)` supports options.
- `Serializer.to_toml(value)` returns TOML text for dictionary-like top-level
  values.
- `Serializer.from_toml(text, klass)` parses TOML and creates an instance of
  `klass`.

## Options

- Options are dictionaries.
- `bytes: "base64"` encodes bytes as base64 strings.
- `bytes: "array"` encodes bytes as arrays of integers.
- Missing `bytes` option rejects bytes values.
- `include_class: true` adds a class-name marker to serialized class instances.
- `class_key: "$class"` overrides the marker key when `include_class` is true.
- `fields: ["name", "score"]` limits serialized class fields.
- `defaults: dict` supplies default field values during deserialization.
- Unknown options raise clear `serialization` errors.

## Class Mapping

- Class instances serialize from their public fields.
- Private fields and methods are not serialized.
- Field order in emitted dictionaries should be deterministic where the runtime
  exposes deterministic key order; otherwise tests should compare parsed data.
- `from_data(data, klass)` creates a `klass` instance and assigns fields from
  `data`.
- If `klass.from_serialized(data)` exists, `from_data` calls it and returns its
  result.
- If `value.to_serialized()` exists, `to_data` calls it and serializes the
  returned value.
- Missing fields are left at constructor/default values when possible, or filled
  from `defaults` when provided.
- Unknown fields in data are assigned as public fields unless
  `strict_fields: true` is provided.
- `strict_fields: true` rejects fields not already present on the constructed
  instance or not listed in `fields`.

## Cycles and Identity

- Cyclic arrays, dictionaries, or object graphs are rejected with a clear
  `serialization: cycle detected` error.
- Shared references are serialized by value. Object identity is not preserved.
- Native resources, file handles, sockets, tasks, channels, mutexes, and other
  runtime resources are rejected unless a custom `to_serialized()` method maps
  them to data.

## Scope

- `stdlib/serialization/Serializer.tya`
- Runtime/checker/codegen support only if public-field discovery or instance
  construction cannot be implemented in Tya.
- `tests/stdlib_serialization_test.tya`
- `docs/STDLIB.md`
- Next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`
- Optional examples under `examples/serialization/`

## Out of Scope

- Binary serialization formats.
- Schema evolution framework beyond defaults and custom hooks.
- Preserving object identity or cyclic graphs.
- Serializing functions, closures, tasks, streams, sockets, native handles, or
  other runtime resources directly.
- Automatic encryption, compression, checksums, or signing.
- Replacing `json.Json` or `toml.Toml`.

## Acceptance Criteria

- `import serialization as serialization` exposes `serialization.Serializer`.
- Primitive values, arrays, and string-key dictionaries round-trip through
  `to_data` and JSON helpers.
- Class instances serialize from public fields.
- Deserialization into a class returns a class instance with the expected
  `.class`.
- Custom `to_serialized()` and `from_serialized(data)` hooks work.
- Bytes values require explicit encoding options and reject by default.
- Cycles raise clear serialization errors.
- Unsupported runtime resources raise clear serialization errors.
- `strict_fields`, `fields`, `defaults`, `include_class`, and `class_key`
  behave as documented.
- Existing `json` and `toml` stdlib tests remain green.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run 'Test.*Serialization|Test.*Json|Test.*Toml|TestSelfhostV01Scripts' -count=1
go test ./... -count=1
```

Manual smoke after implementation:

```sh
tya run examples/serialization/save_file.tya
```

## Dependencies

- Uses existing `json.Json`, `toml.Toml`, and base64 bytes support.
- Should align with the stdlib class-style PRD.
- May need a small reflection/introspection helper for public fields and class
  construction if the current runtime cannot express it in Tya.

## Open Questions

None.
