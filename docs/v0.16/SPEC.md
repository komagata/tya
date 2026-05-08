# Tya v0.16 Specification

This document is the specification for Tya v0.16 after v0.15 structured error
handling.

## Theme

Tya v0.16 is about pattern matching and string interpolation polish.

v0.14 adds destructuring assignment. v0.16 reuses that small pattern vocabulary
for `match` statements. Tya already has string interpolation; v0.16 makes the
interpolation rules explicit and adds brace escaping.

## Goals

- Add block `match value`.
- Add `case pattern` branches.
- Support literal, wildcard, binding, array, dictionary, and nested patterns.
- Keep `match` as a statement, not an expression.
- Run only the first matching case.
- Keep pattern matching small by excluding guards, OR patterns, rest patterns, and exhaustiveness checks.
- Formalize string interpolation behavior.
- Add `{{` and `}}` brace escaping in interpolated strings.

## Included in v0.16

v0.16 includes all v0.15 behavior and adds:

- `match value`
- `case pattern`
- literal patterns
- `_` wildcard pattern
- binding patterns
- array patterns
- dictionary patterns with explicit string keys
- nested patterns
- first-match-only execution
- no fallthrough
- formal string interpolation rules
- interpolation expression diagnostics
- `{{` and `}}` literal brace escapes

## Not Included in v0.16

v0.16 does not include:

- match expressions
- `else` branches in `match`
- pattern guards
- OR patterns
- rest patterns
- class object patterns
- typed patterns
- regex patterns
- exhaustiveness checks
- destructuring in function parameters
- destructuring in `for`
- destructuring catch bindings
- interpolation format specifiers
- interpolation statement blocks
- interpolation assignment targets
- type annotations
- generics
- package manager
- native-backed stdlib

## Match Statement

`match value` evaluates `value` once and compares it against each `case` pattern
in order.

```tya
match value
  case nil
    print "nil"
  case true
    print "true"
  case _
    print "other"
```

Only the first matching case runs. There is no fallthrough.

```tya
match "ok"
  case "ok"
    print "first"
  case "ok"
    print "second"
```

This prints `first`.

If no case matches, the `match` statement does nothing.

```tya
match "ok"
  case "error"
    print "error"
```

This prints nothing.

## Literal Patterns

Literal patterns match by value.

```tya
match status
  case "ok"
    print "ok"
  case "error"
    print "error"
```

v0.16 literal patterns include:

- `nil`
- `true`
- `false`
- numbers
- strings

Literal equality uses the same value equality as normal `==`.

## Wildcard Pattern

`_` matches any value and does not bind it.

```tya
match value
  case _
    print "anything"
```

`_` is the normal way to write a default case in v0.16.

## Binding Pattern

A bare name pattern matches any value and binds that value for the case block.

```tya
match value
  case name
    print name
```

The binding is local to the case block.

```tya
match value
  case name
    print name

print name
```

The final `print name` is invalid unless `name` was already defined outside the
match statement.

If an outer variable has the same name, the case binding shadows it inside the
case block.

## Array Patterns

Array patterns match arrays with exactly the same number of elements.

```tya
match value
  case [name, age]
    print name
  case _
    print "not a user tuple"
```

`[name, age]` matches arrays of length 2 and binds the first element to `name`
and the second element to `age`.

Array length mismatch means the case does not match. It is not a runtime error.

```tya
match ["komagata"]
  case [name, age]
    print name
  case _
    print "fallback"
```

This prints `fallback`.

## Dictionary Patterns

Dictionary patterns match dictionaries that contain the listed explicit string
keys.

```tya
match user
  case {"name": name, "email": email}
    print name + " <" + email + ">"
  case _
    print "unknown"
```

Extra dictionary keys are ignored.

```tya
match {"name": "komagata", "age": 48}
  case {"name": name}
    print name
```

This matches and prints `komagata`.

Missing keys mean the case does not match. They are not runtime errors.

Dictionary pattern keys must be string literals.

```tya
match user
  case {name: value}
    print value
```

This is invalid because v0.16 does not include dictionary key shorthand.

## Nested Patterns

Array and dictionary patterns may be nested.

```tya
match response
  case {"type": "ok", "value": [name, email]}
    print name + " <" + email + ">"
  case {"type": "error", "message": message}
    print message
  case _
    print "unknown"
```

Nested mismatch means the case does not match.

## Match Statement Semantics

`match` is a statement, not an expression.

```tya
result = match value
  case "ok"
    1
```

This is invalid in v0.16.

Use assignment inside case blocks instead.

```tya
result = nil

match value
  case "ok"
    result = 1
  case _
    result = 0
```

Case bindings are local to the matched case block. Bindings from a case that
does not match are not created.

## String Interpolation

String interpolation evaluates `{expression}` inside a string and inserts the
string representation of the expression value.

```tya
name = "komagata"
print "Hello, {name}"
```

This prints `Hello, komagata`.

Interpolation uses the same conversion behavior as `to_string`.

```tya
age = 48
print "age: {age}"
```

This prints `age: 48`.

## Interpolation Expressions

An interpolation must contain exactly one expression.

```tya
print "next age: {age + 1}"
print "ready: {enabled and ready}"
print "name: {user["name"]}"
```

Empty interpolation is invalid.

```tya
print "Hello, {}"
```

Unclosed interpolation is invalid.

```tya
print "Hello, {name"
```

Interpolation expressions may read values and call functions. They are not
assignment targets and must not contain statements.

## Brace Escaping

`{{` inserts a literal `{` and `}}` inserts a literal `}`.

```tya
print "literal {{ brace }}"
```

This prints `literal { brace }`.

A single unmatched `}` is invalid.

```tya
print "bad } brace"
```

## Diagnostics

v0.16 implementations should report source-oriented errors for:

- `match` without a value expression
- `case` outside `match`
- invalid pattern syntax
- non-string dictionary keys in patterns
- match statement used as an expression
- duplicate binding names inside one pattern
- empty interpolation
- unclosed interpolation
- unmatched `}` in strings
- invalid interpolation expression

Diagnostics should mention whether the error is in a match pattern or string
interpolation when that distinction is relevant.
