---
layout: doc
title: Spec
permalink: /v0.14/spec/
---

# Tya v0.14 Specification

This document is the specification for Tya v0.14 after v0.13 explicit
`override` and constructor chaining checks.

## Theme

Tya v0.14 is about destructuring assignment.

Tya already supports multiple assignment, arrays, and dictionaries. v0.14 adds
small destructuring patterns on the left side of assignment so scripts can pull
values out of array and dictionary data without repetitive indexing.

The feature is intentionally limited to assignment. It is not pattern matching,
function parameter destructuring, or loop destructuring.

## Goals

- Add array destructuring assignment.
- Add dictionary destructuring assignment with explicit string keys.
- Allow nested destructuring patterns.
- Add `_` as a discard target inside destructuring patterns.
- Report runtime errors for shape mismatches and missing dictionary keys.
- Keep destructuring limited to assignment targets.

## Included in v0.14

v0.14 includes all v0.13 behavior and adds:

- `[name, age] = value`
- `{"name": name, "email": email} = value`
- nested array and dictionary destructuring
- `_` discard targets inside destructuring patterns
- runtime shape mismatch errors
- runtime missing-key errors

## Not Included in v0.14

v0.14 does not include:

- rest destructuring
- default values in destructuring patterns
- dictionary key shorthand
- function parameter destructuring
- `for` loop destructuring
- pattern matching
- class object destructuring
- destructuring in `catch` or future error handlers
- destructuring in import declarations
- type annotations
- generics
- dictionary member access with `dict.key`
- package manager
- native-backed stdlib

## Array Destructuring Assignment

Array destructuring assigns elements from an array value to local targets.

```tya
[name, age] = ["komagata", 48]

print name
print age
```

This assigns `"komagata"` to `name` and `48` to `age`.

The right-hand value must be an array with exactly the same number of elements
as the array pattern.

```tya
[name, age] = ["komagata"]
```

This is a runtime error because the array has 1 element but the pattern expects
2.

Extra values are also errors.

```tya
[name] = ["komagata", 48]
```

This is a runtime error because the array has 2 elements but the pattern expects
1.

## Dictionary Destructuring Assignment

Dictionary destructuring assigns values from explicit string keys.

```tya
{"name": name, "email": email} = user

print name
print email
```

The right-hand value must be a dictionary. Each key in the pattern must exist in
the dictionary.

```tya
{"name": name} = {}
```

This is a runtime error because the `name` key is missing.

Dictionary destructuring does not require the dictionary to contain only the
listed keys. Extra dictionary keys are ignored.

```tya
{"name": name} = {"name": "komagata", "email": "k@example.com"}
```

This is valid and assigns `"komagata"` to `name`.

Dictionary keys in destructuring patterns must be string literals.

```tya
{name: value} = user
```

This is invalid because v0.14 does not include dictionary key shorthand.

## Nested Destructuring

Destructuring patterns may be nested.

```tya
[name, [city, zip]] = ["komagata", ["Tokyo", "100-0001"]]
```

This assigns `"komagata"` to `name`, `"Tokyo"` to `city`, and `"100-0001"` to
`zip`.

Array and dictionary patterns may be combined.

```tya
{"user": [name, email]} = {"user": ["komagata", "k@example.com"]}
```

This assigns `"komagata"` to `name` and `"k@example.com"` to `email`.

Nested mismatches are runtime errors.

```tya
[name, [city, zip]] = ["komagata", "Tokyo"]
```

This is a runtime error because the nested value `"Tokyo"` is not an array.

## Discard Targets

`_` discards a destructured value.

```tya
[name, _] = ["komagata", 48]
```

This assigns `"komagata"` to `name` and ignores `48`.

`_` may appear more than once in the same destructuring pattern.

```tya
[_, name, _] = [1, "komagata", 3]
```

Discard targets do not create or update a variable named `_`.

## Assignment Semantics

Destructuring assignment is an assignment form. It may appear where a normal
assignment statement may appear.

```tya
user = ["komagata", 48]
[name, age] = user
```

Destructuring assignment may be used with existing variables.

```tya
name = ""
age = 0

[name, age] = ["komagata", 48]
```

This updates `name` and `age`.

Destructuring assignment is not an expression.

```tya
print([name, age] = user)
```

This is invalid.

## Evaluation Order

The right-hand expression is evaluated once before destructuring begins.

```tya
[name, age] = load_user()
```

`load_user()` is called once.

If destructuring fails, variables assigned before the failure may have been
updated. v0.14 does not guarantee atomic rollback for partial destructuring
failure.

Implementations should still report the error at the failing pattern location
when available.

## Multiple Assignment

Destructuring assignment is separate from existing multiple assignment.

```tya
name, age = "komagata", 48
```

This existing form remains valid.

A destructuring pattern may be one target in multiple assignment only if the
implementation can keep the evaluation rule simple and source-oriented. v0.14
does not require this form:

```tya
[name, age], city = user, "Tokyo"
```

Implementations may reject mixed destructuring and multiple assignment in
v0.14. Plain destructuring assignment remains the required feature.

## Modules

Destructuring assignment works inside modules and functions.

```tya
module users
  name_of = user ->
    {"name": name} = user
    name
```

## Diagnostics

v0.14 implementations should report source-oriented errors for:

- invalid destructuring pattern syntax
- non-string dictionary keys in destructuring patterns
- destructuring assignment used as an expression
- mixed destructuring and multiple assignment when unsupported
- runtime non-array value for array destructuring
- runtime array length mismatch
- runtime non-dictionary value for dictionary destructuring
- runtime missing dictionary key
- runtime nested destructuring mismatch

Diagnostics should mention the pattern kind, expected shape, and actual value
kind when available.
