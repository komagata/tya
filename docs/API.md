# Tya v0.1 API

This document defines the standard built-in functions for Tya v0.1.

v0.1 fixes only the minimal API needed by the self-host compiler and basic
programs as standard builtins. Convenience functions can be added in later
versions.

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

`error` returns an error value with a `message`. In v0.1, `.` is reserved for
module member access, so read the message with `err["message"]`.

## Conversion

```tya
to_string value
to_int value
to_float value
to_number value
```

```tya
print to_string 20
print to_int "42"
print to_float "2.5"
print to_number "12.5"
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

```tya
text = trim "  hello,tya  "
parts = split text, ","

print join parts, "-"
print replace text, "tya", "Tya"
print contains text, "hello"
print starts_with text, "hello"
print ends_with text, "tya"
```

## Arrays

```tya
len value
push array, value
pop array
```

```tya
items = [1, 2]
push items, 3
print pop items
print len items
```

`len` works with strings, arrays, and dictionaries.

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

```tya
write_file "/tmp/memo.txt", "hello"
print read_file "/tmp/memo.txt"
print file_exists "/tmp/memo.txt"
```

## Process

```tya
args()
env name
```

```tya
items = args()
print len items
print env "HOME"
```

## Not In v0.1

The following functions are not standard builtins in v0.1.

```text
map
filter
find
any
all
each
reduce
byte_len
char_len
equal
div
read_line
set
```

## Naming

Standard built-in functions use `snake_case`. CamelCase builtin spellings are
not part of the language spec.
