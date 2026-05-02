# Standard Library

This documents the standard functions currently available to Tya programs.

## Core

```tya
print "hello"
panic "bad state"
exit 1
```

```tya
err = error "file not found"
print err.message
```

## Conversion

```tya
print to_string 20
print to_int "42"
print to_float "2.5"
print to_number "12.5"
print div 5, 2
```

## Strings

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

## Objects

```tya
user = { name: "komagata", age: 20 }

print keys user
print values user
print has user, "name"
delete user, "age"
print equal user, { name: "komagata" }
```

## Files And Process

```tya
write_file "/tmp/memo.txt", "hello"
print read_file "/tmp/memo.txt"
print file_exists "/tmp/memo.txt"
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
