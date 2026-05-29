# Feature: `to_string` And `inspect`

## Goal
Make user-defined instance display customizable with Ruby-style human-readable and debug-readable methods, using `to_string()` for normal display and `inspect()` for debugging. `print`, `println`, and string interpolation should use `to_string()` when available, while `inspect()` should provide a stable developer-facing representation. Struct and record values should also print well by default even though they cannot define methods.

## Context
Current docs describe a `Stringable` protocol based on `to_s()`, and stdlib code already uses `to_s()` in several places. Tya also has some internal/runtime use of `to_string`. This feature intentionally changes the public display protocol from `to_s()` to `to_string()` and does not keep `to_s()` as a compatibility alias.

Other language precedents:

- Ruby uses `to_s` for human-readable output and `inspect` for debugging.
- Python separates `__str__` and `__repr__`.
- Rust separates `Display` and `Debug`.
- Swift separates `description` and `debugDescription`.

Tya should follow the same two-surface model, with Tya-style method names.

## Behavior
- `to_string()` is the public human-readable display hook.
- `inspect()` is the public developer/debug display hook.
- `to_s()` is removed from the public protocol and is not treated specially.
- `print(value)`, `println(value)`, and string interpolation `{value}` format values through the normal display surface:
  - primitive values use their existing normal display,
  - user-defined objects call `value.to_string()` when that method exists,
  - if a user-defined object has no `to_string()`, normal display falls back to the default object display,
  - struct and record values use their built-in field display because struct and record bodies cannot define methods.
- `to_string()` must return a string. Returning any other value is a runtime error.
- `inspect()` must return a string. Returning any other value is a runtime error.
- `inspect(value)` should be available as a top-level built-in function for debug display.
- `value.inspect()` should be available on objects and primitives.
- User-defined `inspect()` overrides the default debug display for that object.
- When a user-defined object does not define `inspect()`, default inspect output is stable and includes:
  - the class name,
  - public instance fields,
  - field values formatted with inspect-style display.
- Struct and record values have built-in `to_string()` and `inspect()` behavior:
  - both surfaces include the declared type name and every field,
  - field values are formatted with inspect-style display so strings are quoted and nested values remain unambiguous,
  - output shape is `TypeName(field: value, other: value)`,
  - field order follows declaration order,
  - records and structs with the same fields still display with their own declared type name.
- Default inspect output must not include private fields.
- Default inspect output must be deterministic:
  - fields appear in class declaration order,
  - struct and record fields appear in declaration order,
  - inherited public fields appear before child-class public fields,
  - dictionary keys continue to use the existing stable ordering if dictionary inspect includes dictionaries.
- Cyclic structures and cyclic object graphs must terminate with a stable cycle marker instead of recursing forever.
- Primitive `to_string()` and `inspect()` behavior:
  - strings: `to_string()` returns the raw string; `inspect()` returns a quoted, escaped string.
  - numbers, booleans, and nil: both surfaces return their stable literal-like text.
  - arrays and dictionaries: `to_string()` keeps the current normal display behavior; `inspect()` uses inspect-style display for nested values.
  - bytes: keep the existing normal display for `to_string()`; `inspect()` must be stable and unambiguous.
- String interpolation uses `to_string()`, not `inspect()`.
- Error messages that include user values may keep using existing display unless this feature touches the relevant code path; tests should cover `print`, `println`, interpolation, and explicit `inspect`.

Examples:

```tya
class User
  name: nil

  initialize: user_name ->
    self.name = user_name

  to_string: -> name

  inspect: -> "User(name: {name.inspect()})"

user = User("komagata")
print(user)
print("{user}")
print(user.inspect())
print(inspect(user))
```

Expected output:

```text
komagata
komagata
User(name: "komagata")
User(name: "komagata")
```

Default inspect:

```tya
class Point
  x: 0
  y: 0

point = Point()
print(point.inspect())
```

Expected output shape:

```text
Point(x: 0, y: 0)
```

Struct and record display:

```tya
struct User
  name
  age: 0

record Point
  x
  y

print(User("komagata", 45))
print("{Point(1, 2)}")
print(Point(1, 2).inspect())
```

Expected output:

```text
User(name: "komagata", age: 45)
Point(x: 1, y: 2)
Point(x: 1, y: 2)
```

## Scope
- Update docs from `to_s()`/`Stringable` to `to_string()` and document `inspect()`.
- Update Japanese docs in the same places.
- Update stdlib and tests from `to_s()` to `to_string()` where they depend on the display protocol.
- Update runtime/codegen display functions so object display calls user-defined `to_string()`.
- Add runtime/codegen support for `inspect(value)` and `value.inspect()`.
- Add default inspect output for user-defined objects.
- Add built-in `to_string()` and `inspect()` behavior for struct and record values.
- Add cycle protection for inspect/default display paths touched by this feature.
- Update checker rules for built-ins and method calls as needed.
- Update formatter/parser tests only if method names, built-ins, or examples require it.
- Update LSP behavior if built-in symbol knowledge or hover/completion documentation includes display methods.

## Out of Scope
- No `to_s()` compatibility alias.
- No `repr`, `debug_string`, or alternate debug method names.
- No format specifiers such as `{value:debug}`.
- No per-type formatting options.
- No JSON or serialization behavior changes; `inspect()` is not a serialization protocol.
- No guarantee that default inspect output is parseable Tya source.
- No change to `Serializable.to_data()`.
- No ability to declare custom methods inside struct or record bodies.

## Acceptance Criteria
- A class-defined `to_string()` controls `print(object)`.
- A class-defined `to_string()` controls string interpolation `{object}`.
- A class-defined `inspect()` controls `object.inspect()` and `inspect(object)`.
- If `inspect()` is not defined on an object, default inspect includes class name and public fields in deterministic order.
- `print(struct_value)`, `println(struct_value)`, and `{struct_value}` produce `StructName(field: value, ...)` in declaration order.
- `print(record_value)`, `println(record_value)`, and `{record_value}` produce `RecordName(field: value, ...)` in declaration order.
- `struct_value.inspect()` and `record_value.inspect()` produce the same deterministic type-and-field shape, using inspect-style nested values.
- Private fields do not appear in default inspect.
- `to_string()` returning a non-string raises a runtime error.
- `inspect()` returning a non-string raises a runtime error.
- `to_s()` is no longer special: defining only `to_s()` does not affect `print`, interpolation, or `Stringable` conformance.
- Existing stdlib/tests/docs no longer rely on `to_s()` as the public display protocol.
- Primitive `inspect()` is stable and distinct from `to_string()` for strings.
- Arrays/dictionaries nested in inspect output use inspect-style nested formatting.
- Struct/record values nested in arrays, dictionaries, objects, or other struct/record values use the same type-and-field display.
- Cyclic arrays, dictionaries, objects, structs, and records do not recurse forever when inspected or displayed through struct/record fields.

## Verification
```sh
go test ./internal/checker ./internal/codegen ./internal/eval ./internal/lsp ./tests -count=1
go test ./... -count=1
```
