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
print toString 20
print toInt "42"
print toFloat "2.5"
print toNumber "12.5"
print div 5, 2
```

## Strings

```tya
text = trim "  hello,tya  "
parts = split text, ","

print join parts, "-"
print replace text, "tya", "Tya"
print contains text, "hello"
print startsWith text, "hello"
print endsWith text, "tya"
print byteLen "ちゃ"
print charLen "ちゃ"
```

## Arrays

```tya
items = [1, 2, 3, 4]

double = item -> item * 2
isEven = item -> item % 2 == 0
add = total, item -> total + item

print map items, double
print filter items, isEven
print find items, isEven
print any items, isEven
print all items, isEven
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
writeFile "/tmp/memo.txt", "hello"
print readFile "/tmp/memo.txt"
print fileExists "/tmp/memo.txt"
```

```tya
items = args()
print len items
print env "HOME"
```

```tya
name = readLine()
print "Hello, {name}"
```
