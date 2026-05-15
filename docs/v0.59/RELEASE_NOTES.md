---
layout: doc
title: Release Notes
permalink: /v0.59/release-notes/
---

# Tya v0.59 Release Notes

> **Status:** shipped. `tya version` reports `0.59.0`.

v0.59 makes primitive values behave like class instances. Numbers, strings,
arrays, dicts, booleans, and nil now expose their standard behavior through
method calls:

```tya
42.to_s()
"hi".upper()
[1, 2, 3].len()
{ name: "tya" }.keys()
true.to_s()
nil.class.name
```

The six primitive class identifiers are reserved at top level: `Number`,
`String`, `Array`, `Dict`, `Boolean`, and `Nil`. `x.class` returns the wrapper
class singleton for both primitive values and user-defined instances.

## Breaking Changes

- `kind(x)` is removed. Use `x.class` for identity checks or `x.class.name`
  for a string label.
- The `string`, `array`, and `dict` stdlib modules are removed. Use methods on
  the value: `s.trim()`, `items.map(fn)`, `data.keys()`.
- Top-level primitive helper builtins such as `len`, `trim`, `contains`,
  `push`, `pop`, `keys`, `values`, `map`, `filter`, `reduce`, `to_string`,
  `to_int`, `to_float`, and `to_number` are removed.
- The primitive class identifiers cannot be rebound, subclassed, or
  monkey-patched.

## Migration Examples

| Before | After |
|---|---|
| `kind(x)` | `x.class.name` |
| `kind(x) == "string"` | `x.class == String` |
| `len(items)` | `items.len()` |
| `string.blank(s)` | `s.blank?()` |
| `string.split(s, ",")` | `s.split(",")` |
| `array.map(items, fn)` | `items.map(fn)` |
| `dict.has(data, "name")` | `data.has("name")` |
| `to_string(value)` | `value.to_s()` |

## Verification

This release updates the Go compiler, C runtime, interpreter, self-hosted
compiler sources, examples, stdlib modules, and test fixtures to the new
method surface. The release gate is:

```sh
go test ./... -count=1
```
