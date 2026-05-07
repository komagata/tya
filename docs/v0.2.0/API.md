# Tya v0.2 API

This document defines the standard built-in functions for Tya v0.2.

## Core

```tya
print value
panic "bad state"
exit 1
```

`print` writes a value followed by a newline. `panic` stops execution with an
error. `exit` exits with the specified status code.

```tya
err = error "file not found"
print err["message"]
```

`error` returns an error value with a `message`.

## Conversion

```tya
to_string value
to_int value
to_float value
to_number value
```

## Strings

```tya
split text, separator
join items, separator
trim text
replace text, old, new
contains text, search
starts_with text, prefix
ends_with text, suffix
```

## Arrays And Collections

```tya
len value
push array, value
pop array
map array, function
filter array, function
find array, function
any array, function
all array, function
reduce array, initial, function
```

```tya
items = [1, 2, 3]
print map(items, item -> item * 2)
print filter(items, item -> item > 1)
print find(items, item -> item == 2)
print any(items, item -> item == 3)
print all(items, item -> item > 0)
sum = total, item -> total + item
print reduce(items, 0, sum)
```

`len` works with strings, arrays, and dictionaries.

## Equality

```tya
equal left, right
```

`equal` performs deep equality for arrays and dictionaries. The `==` operator
keeps the v0.1 runtime equality behavior.

## Dictionaries

```tya
keys dictionary
values dictionary
has dictionary, key
delete dictionary, key
```

```tya
user = { name: "komagata", age: 20 }

print keys user
print values user
print has user, "name"
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

## Not In v0.2

The following functions are not standard builtins in v0.2.

```text
each
byte_len
char_len
div
set
```

## Naming

Standard built-in functions use `snake_case`. CamelCase builtin spellings are
not part of the language spec.
