# Tya v0.23 Specification

This document is the specification for Tya v0.23 after v0.22 unit testing.

## Theme

Tya v0.23 adds a TOML standard library module.

v0.23 introduces `toml`, a pure-Tya `.toml` reader and writer. TOML is a
well-known configuration format with explicit types, table headers, and
nested tables, suited for editing by humans and parsing by tools.

## Goals

- Add a `toml` standard module.
- Parse TOML 1.0 documents into Tya values (dictionaries, arrays, primitives).
- Emit TOML text from Tya values.
- Keep the API surface small and explicit.
- Avoid schema validation, type coercion beyond standard TOML, and partial
  parsing.

## Included in v0.23

v0.23 includes all v0.22 behavior and adds:

- `toml` standard module
- `toml.parse(text)`
- `toml.dump(value)`

## Not Included in v0.23

v0.23 does not include:

- schema validation
- streaming parser
- preserving comments through round-trip
- preserving original formatting through round-trip
- TOML 1.1+ extensions
- binary content
- any other configuration format (YAML, JSON, NestedText, etc.)

## Importing

`toml` is a standard attached-library module. It is not available without
import.

```tya
import toml
```

The standard module search behavior from v0.17 applies.

## Data Model

`toml.parse(text)` returns Tya values according to the following mapping:

| TOML kind | Tya kind |
|---|---|
| string | string |
| integer | int |
| float | float |
| boolean | bool |
| array | array |
| inline table | dict |
| table | dict |
| array of tables | array of dicts |
| offset date-time | string (RFC 3339, as written) |
| local date-time | string (as written) |
| local date | string (as written) |
| local time | string (as written) |

Date and time values are returned as strings in v0.23. v0.23 does not add a
date/time value type to Tya.

The top-level result of `toml.parse(text)` is always a dict.

## `toml.parse(value)`

```tya
import toml

text = "
title = \"Example\"
[server]
host = \"127.0.0.1\"
port = 8080
"

config = toml.parse(text)
println config["title"]
println config["server"]["host"]
println config["server"]["port"]
```

`toml.parse(text)` parses a complete TOML document.

`toml.parse(text)` raises a structured error on syntax errors. Errors include
a line number and a short message.

`toml.parse(text)` requires a string argument.

## `toml.dump(value)`

```tya
import toml

config = {
  title: "Example",
  server: {
    host: "127.0.0.1",
    port: 8080,
  },
}

println toml.dump(config)
```

`toml.dump(value)` emits a TOML representation of `value`.

`value` must be a dict at the top level. Nested values follow the inverse of
the parse mapping:

| Tya kind | TOML kind |
|---|---|
| string | string |
| int | integer |
| float | float |
| bool | boolean |
| array of dicts | array of tables (or inline arrays of inline tables) |
| array | array |
| dict | table (or inline table) |
| nil | error (TOML has no null) |

`toml.dump(value)` decides between table and inline table form using a
simple rule: dictionaries appearing as values directly under a parent table
are emitted as `[parent.child]` table sections; dictionaries appearing inside
arrays are emitted as inline tables.

`toml.dump(value)` does not preserve key order across round-trip. Output key
order follows the dictionary iteration order of the input.

`toml.dump(value)` raises a structured error when:

- `value` is not a dict
- `value` contains a `nil`
- `value` contains a function, class, object, or module
- `value` contains a dict with a non-string key

## TOML Subset

v0.23 supports the following TOML 1.0 features:

- comments (`# ...`)
- bare keys, quoted keys (basic and literal)
- dotted keys
- basic strings, multi-line basic strings
- literal strings, multi-line literal strings
- integers (decimal, hexadecimal, octal, binary)
- floats (including special values `inf`, `-inf`, `nan`)
- booleans
- arrays (homogeneous and heterogeneous)
- inline tables
- tables and dotted-key headers
- arrays of tables
- offset / local date-time, local date, local time (returned as strings)

v0.23 does not support TOML 1.1 features that are not in 1.0.

## Diagnostics

v0.23 implementations should report source-oriented errors for:

- missing imports for `toml`
- unknown `toml` module functions
- wrong argument counts
- wrong argument kinds
- syntax errors during `toml.parse(text)` (with line number)
- unsupported value kinds during `toml.dump(value)`

Diagnostics should mention the module name, function name, expected argument
shape, and actual value kind when available.
