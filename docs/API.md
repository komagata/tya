# Tya API

This document defines the current standard built-in functions. Primitive
string, array, dict, number, boolean, and nil operations live on their wrapper
classes; use method syntax such as `"hi".upper()`, `[1, 2].len()`, and
`value.to_s()`.

## Core

```tya
print value
assert value
assert_equal expected, actual
panic "bad state"
exit 1
```

`print` writes a value followed by a newline. `assert` stops execution with a
source-oriented diagnostic when the value is falsey. `assert_equal` compares
with deep equality and prints expected and actual values on failure. `panic`
stops execution with an error. `exit` exits with the specified status code.

```tya
err = error "file not found"
print err["message"]
```

`error` returns an error value with a `message`.

## Primitive Methods

```tya
value.to_s()
value.class
" tya ".trim()
"tya".upper()
"a,b".split(",")
["a", "b"].join(",")

items = [1, 2, 3]
print items.map(item -> item * 2)
print items.filter(item -> item > 1)
print items.find(item -> item == 2)
print items.any(item -> item == 3)
print items.all(item -> item > 0)
sum = total, item -> total + item
print items.reduce(0, sum)
```

See the v0.59 specification for the exhaustive primitive method tables.
Use `x.class` to inspect the runtime class wrapper for primitive and object
values.

## Equality

```tya
equal left, right
```

`equal` performs deep equality for arrays and dictionaries. The `==` operator
keeps the v0.1 runtime equality behavior.

## Dictionaries

```tya
delete dictionary, key
```

```tya
user = { name: "komagata", age: 20 }

print user.keys()
print user.values()
print user.has("name")
delete user, "age"
```

## Files

```tya
read_file path
write_file path, text
file_exists path
```

## Input

```tya
read_line()
```

`read_line` reads one line from standard input without the trailing newline. It
returns `nil` at EOF.

## Process

```tya
args()
env name
```

## Naming

Standard built-in functions use `snake_case`. CamelCase builtin spellings are
not part of the language spec.
