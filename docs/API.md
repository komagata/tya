# Tya API

This document lists the built-in functions available to Tya programs.

## Core

```tya
print value
panic "bad state"
exit 1
```

`print` writes a value and a newline. `panic` stops with an error. `exit`
terminates with a status code.

```tya
err = error "file not found"
print err.message
```

`error` returns an error value with a `message` property.

## Conversion

```tya
to_string value
to_int value
to_float value
to_number value
div left, right
```

```tya
print to_string 20
print to_int "42"
print to_float "2.5"
print to_number "12.5"
print div 5, 2
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
byte_len text
char_len text
```

```tya
text = trim "  hello,tya  "
parts = split text, ","

print join parts, "-"
print replace text, "tya", "Tya"
print contains text, "hello"
print starts_with text, "hello"
print ends_with text, "tya"
print byte_len "ちゃ"
print char_len "ちゃ"
```

## Arrays

```tya
len value
push array, value
pop array
map array, function
filter array, function
find array, function
any array, function
all array, function
each array, function
reduce array, initial, function
```

```tya
items = [1, 2, 3, 4]

double = item -> item * 2
is_even = item -> item % 2 == 0
add = total, item -> total + item

print map items, double
print filter items, is_even
print find items, is_even
print any items, is_even
print all items, is_even
print reduce items, 0, add
```

```tya
items = [1, 2]
push items, 3
print pop items
print len items
```

## Dictionaries

```tya
keys dictionary
values dictionary
has dictionary, key
delete dictionary, key
equal left, right
```

```tya
user = { name: "komagata", age: 20 }

print keys user
print values user
print has user, "name"
delete user, "age"
print equal user, { name: "komagata" }
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
read_line()
```

```tya
items = args()
print len items
print env "HOME"
```

```tya
name = read_line()
print "Hello, {name}"
```

## Naming

Standard library APIs use snake_case names. CamelCase builtin spellings are not
part of the language surface.
