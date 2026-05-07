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

