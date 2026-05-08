# Tya v0.20 Specification

This document is the specification for Tya v0.20 after v0.19 predicate names.

## Theme

Tya v0.20 expands the standard attached library with small, dependency-free
modules.

v0.20 adds `math` and `path`. These modules improve everyday scripting without
adding new language syntax, parsers, network behavior, package management, or
native-backed libraries.

## Goals

- Add a `math` standard module.
- Add a `path` standard module.
- Keep both modules import-only and explicit.
- Keep the APIs small enough to implement in Tya or existing runtime helpers.
- Avoid JSON, CSV, regex, HTTP, date/time, and other heavier libraries.
- Avoid moving existing global built-ins in this version.

## Included in v0.20

v0.20 includes all v0.19 behavior and adds:

- `math` standard module
- `path` standard module
- source-oriented diagnostics for invalid `math` and `path` arguments
- documentation and tests for both modules

## Not Included in v0.20

v0.20 does not include:

- new language syntax
- changes to v0.18 global built-ins
- JSON parser
- CSV parser
- regex engine
- HTTP client or server
- date/time library
- native-backed standard modules
- package manager
- remote module install
- versioned dependencies

## Importing

`math` and `path` are standard attached-library modules. They are not available
without import.

```tya
import math
import path
```

The standard module search behavior from v0.17 applies.

## `math`

The `math` module contains small numeric helpers.

Functions:

- `math.abs(value)`
- `math.min(left, right)`
- `math.max(left, right)`
- `math.clamp(value, min, max)`

Examples:

```tya
import math

print math.abs(-3)
print math.min(2, 5)
print math.max(2, 5)
print math.clamp(12, 0, 10)
```

`math.abs(value)` returns the absolute value of an integer or float.

`math.min(left, right)` returns the smaller number.

`math.max(left, right)` returns the larger number.

`math.clamp(value, min, max)` returns `min` when `value < min`, `max` when
`value > max`, and `value` otherwise.

All `math` functions require numeric arguments. Passing a non-number is an
error.

`math.clamp(value, min, max)` is an error when `min > max`.

## `path`

The `path` module contains lexical path helpers. It does not access the file
system.

Functions:

- `path.join(parts)`
- `path.clean(value)`
- `path.basename(value)`
- `path.dirname(value)`
- `path.extname(value)`

Examples:

```tya
import path

print path.join(["tmp", "tya", "memo.txt"])
print path.clean("tmp/./tya/../memo.txt")
print path.basename("/tmp/tya/memo.txt")
print path.dirname("/tmp/tya/memo.txt")
print path.extname("/tmp/tya/memo.txt")
```

`path.join(parts)` joins an array of path segments with `/` and cleans the
result.

`path.clean(value)` normalizes `.` segments, `..` segments, and repeated `/`
separators lexically.

`path.basename(value)` returns the final path segment.

`path.dirname(value)` returns the path without the final segment. It returns `.`
when no directory segment exists.

`path.extname(value)` returns the final file extension including the leading
`.`. It returns `""` when the basename has no extension.

The `path` module uses `/` as the path separator. Platform-specific path
behavior is not part of v0.20.

All `path` functions require string arguments, except `path.join(parts)`, which
requires an array of strings.

## Diagnostics

v0.20 implementations should report source-oriented errors for:

- missing imports for `math` or `path`
- unknown `math` or `path` module functions
- wrong argument counts
- wrong argument kinds
- non-number arguments passed to `math` functions
- `math.clamp(value, min, max)` where `min > max`
- non-string values passed to `path` functions
- non-string array items passed to `path.join(parts)`

Diagnostics should mention the module name, function name, expected argument
shape, and actual value kind when available.
