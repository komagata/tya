# Tya v0.3 Standard Attached Library

This document defines the standard attached library for Tya v0.3.

The standard attached library is a set of `.tya` modules shipped with Tya. It is
not a package manager and it does not download third-party code.

## Importing

Standard modules use the same import syntax as user modules.

```tya
import string
import array
```

The module search order is:

1. The importing file's directory.
1. Directories listed in `TYA_PATH`, searched left to right.
1. The `stdlib/` directory shipped with Tya.

## Initial Scope

v0.3 starts with lightweight modules that prove the attached-library
mechanism.

Included in the initial scope:

- `string`
- `array`

Deferred from v0.3:

- JSON parser
- CSV parser
- native-backed standard modules
- package manager
- remote module install
- versioned dependencies

## `string`

```tya
import string

print string.blank("  ")
print string.present("tya")
```

Functions:

```tya
blank text
present text
```

`blank(text)` returns `true` when `trim(text) == ""`.

`present(text)` returns `not blank(text)`.

## `array`

```tya
import array

print array.empty([])
print array.first(["tya"])
```

Functions:

```tya
empty items
first items
```

`empty(items)` returns `len(items) == 0`.

`first(items)` returns `items[0]`.

## `math`

```tya
import math

print math.abs(-3)
print math.min(2, 5)
print math.max(2, 5)
print math.clamp(12, 0, 10)
```

Functions:

```tya
abs value
min left, right
max left, right
clamp value, min, max
```

`abs(value)` returns the absolute value of an integer or float.

`min(left, right)` returns the smaller number.

`max(left, right)` returns the larger number.

`clamp(value, min, max)` returns `min` when `value < min`, `max` when
`value > max`, and `value` otherwise. It raises an error when `min > max`.

## `path`

```tya
import path

print path.join(["tmp", "tya", "memo.txt"])
print path.clean("tmp/./tya/../memo.txt")
print path.basename("/tmp/tya/memo.txt")
print path.dirname("/tmp/tya/memo.txt")
print path.extname("/tmp/tya/memo.txt")
```

Functions:

```tya
join parts
clean value
basename value
dirname value
extname value
```

`join(parts)` joins an array of path segments with `/` and cleans the result.

`clean(value)` normalizes `.` segments, `..` segments, and repeated `/`
separators lexically.

`basename(value)` returns the final path segment.

`dirname(value)` returns the path without the final segment, or `.` when no
directory segment exists.

`extname(value)` returns the final file extension including the leading `.`,
or `""` when the basename has no extension.

The `path` module uses `/` as the path separator and does not access the file
system.
