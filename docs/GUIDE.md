# Tya Guide

Tya is a small indentation-based dynamic language inspired by CoffeeScript.
This guide is for reading from top to bottom.

## Run A Program

```sh
go run ./cmd/tya examples/hello.tya
go run ./cmd/tya run examples/hello.tya
```

`tya run` compiles the source to C, builds it with `cc`, and runs the
resulting binary.

## Values

```tya
name = "Tya"
age = 1
pi = 3.14
ready = true
missing = nil
```

Strings can contain interpolated expressions.

```tya
print "Hello, {name}"
```

## Names

Use `snake_case` for variables, functions, modules, and object properties.
Use `SCREAMING_SNAKE_CASE` for constants.

```tya
user_name = "komagata"
MAX_COUNT = 10
```

See `docs/NAMING.md` for the full naming rules.

## Conditions

```tya
if age >= 20
  print "adult"
else
  print "young"
```

Use `and`, `or`, and `not` for logic.

```tya
if ready and not missing
  print "ok"
```

## Loops

```tya
count = 0
while count < 3
  print count
  count = count + 1
```

```tya
items = ["a", "b"]

for item in items
  print item

for item, index in items
  print "{index}: {item}"
```

Use `of` to iterate dictionary keys and values.

```tya
user = { name: "komagata", age: 20 }

for key, value of user
  print "{key}: {value}"
```

## Functions

Functions use `->`. The last expression is returned implicitly.

```tya
greet = name -> "Hello, {name}"

print greet "Tya"
```

Use an indented body for multiple statements.

```tya
double = value ->
  result = value * 2
  result
```

Use `return` when returning early or returning multiple values.

```tya
parse_user = text ->
  if text == ""
    return nil, error "empty user"
  return { name: text }, nil
```

## Arrays And Dictionaries

```tya
items = [1, 2]
push items, 3
print items[0]
```

```tya
user =
  name: "komagata"
  age: 20

print user["name"]
```

Legacy method objects can use `@property` as the receiver.

```tya
counter =
  count: 0

  inc: ->
    @count = @count + 1
    @count

print counter.inc()
```

## Errors

Tya uses error values, not exceptions.

```tya
user, err = parse_user ""
if err
  print err.message
```

Inside a function, `try` propagates the error part of a `value, err` result.

```tya
load_user = text ->
  user = try parse_user(text)
  user["name"]
```

## Modules

Import another `.tya` file from the same directory.

```tya
import greeting

print greeting.hello("komagata")
```

Each module exposes exactly one public top-level binding. The binding must
match the file name.

```tya
# greeting.tya
greeting =
  hello: name -> "Hello, {name}"
```

## Standard Library

See `docs/API.md` for built-in functions such as `print`, `len`, `map`,
`read_file`, and `to_string`.
