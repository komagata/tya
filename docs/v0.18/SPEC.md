---
layout: doc
title: Spec
permalink: /v0.18/spec/
---

# Tya v0.18 Specification

This document is the specification for Tya v0.18 after v0.17 import aliases and
module loading rules.

## Theme

Tya v0.18 is about expanding module-style standard APIs for strings, arrays,
and dictionaries.

Tya keeps built-in values lightweight. String, array, and dictionary operations
are written as module functions such as `string.trim(value)`,
`array.join(values, separator)`, and `dict.has(value, key)`.

v0.18 expands those module APIs instead of adding built-in value methods or
built-in collection classes.

## Goals

- Expand the `string` module with practical text helpers.
- Expand the `array` module with practical collection helpers.
- Expand the `dict` module with practical map helpers.
- Keep module function style as the primary API style.
- Keep built-in value representation lightweight.
- Avoid built-in collection classes and monkey patching.

## Included in v0.18

v0.18 includes all v0.17 behavior and adds:

- additional `string.*` functions
- additional `array.*` functions
- additional `dict.*` functions
- Go-like minimal global built-ins: `print`, `println`, `len`, and `panic`
- clearer mutation rules for collection helpers
- source-oriented errors for unsupported argument shapes

## Not Included in v0.18

v0.18 does not include:

- built-in value method calls such as `value.join(":")`
- `String`, `Array`, or `Dictionary` class objects
- built-in class inheritance
- monkey patching built-in values
- user-defined extension methods
- method extraction from built-in values
- operator methods
- `[]` or `[]=` method syntax
- property-style access such as `values.length`
- tuple type or tuple literal
- package manager
- native-backed stdlib

## Built-ins

Tya v0.18 keeps global built-in functions small and Go-like.

The only functions available without import are:

- `print(value)`
- `println(value)`
- `len(value)`
- `panic(value)`

`print` writes a value. `println` writes a value followed by a newline.

`len(value)` returns the length of a string, array, or dictionary. It remains a
global built-in because length checks are common in control-flow code.

`panic(value)` stops the program with a fatal error. `panic` is not caught by
`try` or `catch`.

Other standard operations are not global built-ins. They live in explicit
standard modules such as `string`, `array`, and `dict`.

`try`, `catch`, and `raise` are language syntax, not built-in functions.
`try` and `catch` handle structured errors raised with `raise`; they do not
catch `panic`.

## String Module

The `string` module contains functions that operate on string values.

Existing string helpers remain available. v0.18 standardizes this practical
surface:

- `string.len(value)`
- `string.byte_len(value)`
- `string.char_len(value)`
- `string.trim(value)`
- `string.contains(value, needle)`
- `string.starts_with(value, prefix)`
- `string.ends_with(value, suffix)`
- `string.replace(value, old, new)`
- `string.split(value, separator)`
- `string.join(values, separator)`
- `string.lines(value)`
- `string.upcase(value)`
- `string.downcase(value)`

Examples:

```tya
import string

print string.trim("  tya  ")
print string.contains("hello", "ell")
print string.join(["a", "b", "c"], ":")
print string.lines("a\nb")
print string.upcase("tya")
```

`string.join(values, separator)` joins an array of values after converting each
element with the same conversion behavior as `to_string`.

`string.lines(value)` splits a string into lines without keeping line-ending
characters.

## Array Module

The `array` module contains functions that operate on array values.

Existing array helpers remain available. v0.18 standardizes this practical
surface:

- `array.len(values)`
- `array.first(values)`
- `array.last(values)`
- `array.push(values, value)`
- `array.pop(values)`
- `array.slice(values, start, end)`
- `array.reverse(values)`
- `array.join(values, separator)`
- `array.map(values, function)`
- `array.filter(values, function)`
- `array.find(values, function)`
- `array.any(values, function)`
- `array.all(values, function)`
- `array.each(values, function)`
- `array.reduce(values, initial, function)`

Examples:

```tya
import array

values = ["a", "b", "c"]

print array.first(values)
print array.last(values)
print array.slice(values, 0, 2)
print array.reverse(values)
print array.join(values, ":")
```

`array.push(values, value)` mutates `values` and returns the mutated array.

`array.pop(values)` mutates `values` and returns the removed value.

`array.slice(values, start, end)` returns a new array from `start` inclusive to
`end` exclusive. Negative indexes are not part of v0.18.

`array.reverse(values)` returns a new reversed array and does not mutate the
input array.

## Dict Module

The `dict` module contains functions that operate on dictionary values.

Existing dict helpers remain available. v0.18 standardizes this practical
surface:

- `dict.len(value)`
- `dict.has(value, key)`
- `dict.get(value, key)`
- `dict.get(value, key, default)`
- `dict.set(value, key, item)`
- `dict.delete(value, key)`
- `dict.keys(value)`
- `dict.values(value)`
- `dict.merge(left, right)`

Examples:

```tya
import dict

user = {"name": "komagata"}

print dict.has(user, "name")
print dict.get(user, "name")
print dict.get(user, "email", "none")
print dict.keys(user)
```

`dict.get(value, key)` returns `nil` when the key is missing.

`dict.get(value, key, default)` returns `default` when the key is missing.

`dict.set(value, key, item)` mutates `value` and returns the mutated
dictionary.

`dict.delete(value, key)` mutates `value` and returns the deleted value,
or `nil` when the key is missing.

`dict.merge(left, right)` returns a new dictionary. Keys from `right`
override keys from `left`.

The old `dictionary` module name is not kept as a compatibility alias. v0.18
standardizes on `dict` only.

## API Style

v0.18 keeps module function style as the standard style for built-in value
operations.

```tya
import string
import array
import dict

name = string.trim("  tya  ")
joined = array.join(["a", "b"], ":")
exists = dict.has({"name": name}, "name")
```

The following method-call style is not part of v0.18:

```tya
"  tya  ".trim()
["a", "b"].join(":")
{"name": "tya"}.has("name")
```

This keeps the object model smaller and keeps collection APIs explicit.

## Argument Errors

Module functions should report source-oriented errors for invalid argument
counts and invalid argument kinds.

```tya
array.first("not array")
string.split("a,b,c")
dict.keys(["not", "dictionary"])
```

Diagnostics should mention the module name, function name, expected argument
shape, and actual value kind when available.

## Diagnostics

v0.18 implementations should report source-oriented errors for:

- missing module imports when a module name is not in scope
- unknown `string`, `array`, or `dict` module functions
- wrong argument counts
- wrong argument kinds
- unsupported negative indexes in `array.slice`
- callback arity mismatches in higher-order array functions

Diagnostics should mention the module name and function name when available.
