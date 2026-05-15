---
layout: doc
title: Spec
permalink: /v0.19/spec/
---

# Tya v0.19 Specification

This document is the specification for Tya v0.19 after v0.18 module-style
standard APIs and minimal global built-ins.

## Theme

Tya v0.19 adds predicate names.

A function or method name may end with `?` when the callable answers a yes/no
question. Such a callable must return a boolean value.

## Goals

- Allow `?` at the end of function names.
- Allow `?` at the end of instance method names.
- Allow `?` at the end of class method names.
- Require predicate functions and methods to return boolean values.
- Keep predicate names visually distinct without adding a new type system.
- Keep `?` out of variable, module, class, field, and constant names.

## Included in v0.19

v0.19 includes all v0.18 behavior and adds:

- predicate function names such as `empty?`
- predicate instance method names such as `active?`
- predicate class method names such as `User.enabled?()`
- predicate module functions such as `array.empty?(values)`
- boolean return enforcement for predicate calls
- predicate naming convention that prefers `nil?` over `is_nil?`
- source-oriented diagnostics for invalid predicate names and return values

## Not Included in v0.19

v0.19 does not include:

- general `?` suffixes on variables
- `?` suffixes on module names
- `?` suffixes on class names
- `?` suffixes on field names
- `?` suffixes on constants
- optional chaining
- nil-coalescing operators
- ternary operators
- type annotations
- static boolean return inference
- predicate overloading
- method aliases
- `is_` or `has_` rewrites performed by the compiler

## Predicate Function Names

A function name may end with `?`.

```tya
empty? = values ->
  len(values) == 0

print empty?([])
```

The base name before `?` follows the normal function naming rule.

Allowed:

```tya
empty? = values -> len(values) == 0
has_name? = user -> dict.has(user, "name")
_internal? = value -> value == nil
```

Forbidden:

```tya
Empty? = value -> true
empty_? = value -> true
empty?? = value -> true
```

Predicate names should not repeat the predicate meaning with an `is_` prefix
when the shorter base name is clear.

Preferred:

```tya
nil? = value -> value == nil
empty? = values -> len(values) == 0
active? = user -> user["active"] == true
```

Avoid:

```tya
is_nil? = value -> value == nil
is_empty? = values -> len(values) == 0
is_active? = user -> user["active"] == true
```

`has_` is still allowed when it names possession rather than predicate syntax.

```tya
has_name? = user -> dict.has(user, "name")
```

## Predicate Methods

Instance methods may end with `?`.

```tya
class User
  init = name ->
    @name = name

  named? = ->
    @name != ""

user = User("komagata")
print user.named?()
```

Class methods may also end with `?`.

```tya
class User
  @@enabled = true

  @@enabled? = ->
    @@enabled

print User.enabled?()
```

Private predicate methods use the existing private naming rule with a leading
underscore.

```tya
class User
  _valid_name? = name ->
    name != ""
```

## Predicate Module Functions

Module function names may end with `?`.

```tya
module path
  absolute? = value ->
    string.starts_with(value, "/")
```

Use the predicate through normal module member access.

```tya
import path

print path.absolute?("/tmp/memo.txt")
```

Standard modules may use predicate names when the result is boolean.

```tya
import array
import dict
import value

print array.empty?([])
print dict.has?(user, "name")
print value.nil?(result)
```

## Boolean Return Requirement

Every call to a predicate function or method must return a boolean value.

```tya
ready? = ->
  true

print ready?()
```

Returning any non-boolean value is an error.

```tya
name? = ->
  "komagata"

print name?()
```

The call to `name?()` is invalid because the function returns a string.

This check is performed when the predicate call returns. Tya remains a dynamic
language and v0.19 does not add static return type inference.

## Syntax Boundary

`?` is part of the identifier only when it appears at the end of a callable
name.

Allowed callable names:

```text
empty?
has_name?
_internal?
```

Names that are not callable declarations cannot end with `?`.

```tya
active? = true       # invalid
class User?          # invalid
module user?         # invalid
@active? = true      # invalid
@@enabled? = true    # invalid as a class variable
```

The following are still ordinary calls and member calls:

```tya
empty?()
user.active?()
User.enabled?()
array.empty?(values)
```

## Diagnostics

v0.19 implementations should report source-oriented errors for:

- invalid `?` placement in a name
- `?` used on a non-callable binding
- predicate functions returning non-boolean values
- predicate instance methods returning non-boolean values
- predicate class methods returning non-boolean values
- predicate module functions returning non-boolean values

Diagnostics should mention the predicate name and the actual returned value kind
when available.
