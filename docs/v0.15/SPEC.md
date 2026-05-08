# Tya v0.15 Specification

This document is the specification for Tya v0.15 after v0.14 destructuring
assignment.

## Theme

Tya v0.15 is about structured error handling.

Tya already has error values and `try expression` for propagating `value, err`
returns from functions. v0.15 keeps that behavior and adds a separate
structured mechanism for raised errors: `raise` and block `try/catch`.

The feature is intentionally small. It adds throwing, catching, and re-raising,
but does not add `finally`, typed catches, error class hierarchies, or
try/catch expressions.

## Goals

- Add `raise expression`.
- Add block `try ... catch name ...`.
- Allow `_` as a catch discard variable.
- Let raised values propagate to the nearest enclosing `catch`.
- Allow catch blocks to re-raise.
- Keep existing `try expression` behavior unchanged.
- Keep block `try/catch` as a statement, not an expression.

## Included in v0.15

v0.15 includes all v0.14 behavior and adds:

- `raise expression`
- `try` block with required `catch name`
- arbitrary raised values
- catch-local error binding
- `_` catch discard binding
- raised error propagation
- re-raise from inside `catch`
- uncaught raised error reporting

## Not Included in v0.15

v0.15 does not include:

- `finally`
- `ensure`
- typed catch
- multiple catch clauses
- catch filters
- `catch` without a binding name
- try/catch as an expression
- destructuring catch bindings
- error class hierarchy
- stack trace API
- async error handling
- changing existing `try expression` semantics
- type annotations
- generics
- package manager
- native-backed stdlib

## `raise`

`raise expression` evaluates the expression and raises the resulting value.

```tya
read_config = ->
  raise error("missing config")
```

Raised values propagate to the nearest enclosing `catch`. If no `catch` handles
the value, program execution fails with an uncaught raised error.

Any value may be raised in v0.15.

```tya
raise "bad state"
raise error("bad state")
raise {"message": "bad state", "code": "bad_state"}
```

Code should prefer `error("message")` or a dictionary with a `message` key when
the raised value is intended to be displayed to users.

`raise` requires an expression.

```tya
raise
```

This is invalid.

## Block `try/catch`

A block `try/catch` catches raised values from the `try` block.

```tya
main = ->
  try
    read_config()
  catch err
    print err["message"]
```

If `read_config()` raises, the raised value is bound to `err` and the `catch`
block runs.

If the `try` block finishes without raising, the `catch` block is skipped.

```tya
try
  print "ok"
catch err
  print "failed"
```

This prints `ok` and does not run the catch block.

## Required Catch Binding

`catch` requires a binding name in v0.15.

```tya
try
  risky()
catch err
  print err
```

This is valid.

```tya
try
  risky()
catch
  print "failed"
```

This is invalid because `catch` has no binding name.

Use `_` to discard the raised value.

```tya
try
  risky()
catch _
  print "failed"
```

The `_` catch binding does not create or update a variable named `_`.

## Catch Binding Scope

The catch binding is local to the catch block.

```tya
try
  risky()
catch err
  print err

print err
```

The final `print err` is invalid because `err` is only defined inside the
`catch` block.

If an outer variable has the same name, the catch binding shadows it inside the
catch block.

```tya
err = "outer"

try
  risky()
catch err
  print err

print err
```

The final `print err` reads the outer `err`.

## Re-Raise

A catch block may raise the caught value again.

```tya
try
  risky()
catch err
  print err["message"]
  raise err
```

The re-raised value propagates to the next enclosing `catch`.

```tya
try
  try
    risky()
  catch err
    raise err
catch outer
  print outer["message"]
```

## Statement Semantics

Block `try/catch` is a statement, not an expression.

```tya
value = try
  risky()
catch err
  "fallback"
```

This is invalid in v0.15.

Use assignment inside the blocks instead.

```tya
value = nil

try
  value = risky()
catch err
  value = "fallback"
```

## Existing `try expression`

v0.15 keeps the existing `try expression` behavior unchanged.

```tya
load_user = text ->
  user = try parse_user(text)
  user["name"]
```

`try expression` still expects the target expression to return `value, err`.
If `err` is truthy, the current function returns `nil, err`. If `err` is
falsey, `value` becomes the expression value.

`try expression` is not a catch mechanism. If the expression raises, the raised
value propagates to an enclosing block `try/catch`.

```tya
try
  user = try parse_user(text)
catch err
  print err["message"]
```

In this example, returned `value, err` errors are handled by `try expression`.
Raised errors are handled by the outer block `try/catch`.

## Return, Break, and Continue

`return` from inside a `try` block returns from the current function and does not
run the `catch` block.

```tya
load = ->
  try
    return "ok"
  catch err
    return "failed"
```

This returns `"ok"`.

`return` from inside a `catch` block returns from the current function.

```tya
load = ->
  try
    raise error("failed")
  catch err
    return "fallback"
```

This returns `"fallback"`.

`break` and `continue` keep their normal loop behavior. They are not caught as
raised values.

## Modules

`raise` and block `try/catch` work inside modules and functions.

```tya
module config
  load = path ->
    raise error("missing config")

main = ->
  try
    config.load("config.json")
  catch err
    print err["message"]
```

## Diagnostics

v0.15 implementations should report source-oriented errors for:

- `raise` without an expression
- `catch` without a binding name
- `catch` outside block `try`
- block `try` without `catch`
- invalid catch binding names
- block `try/catch` used as an expression
- ambiguous `try` syntax between `try expression` and block `try/catch`
- uncaught raised values

Diagnostics should mention whether the code is using `try expression` or block
`try/catch` when that distinction is relevant.
