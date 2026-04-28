# Tya

Tya is a small indentation-based dynamic language inspired by CoffeeScript.

This repository currently contains a Go lexer, parser, AST, checker, and
interpreter.

## Run

```sh
go run ./cmd/tya examples/hello.tya
```

## Test

```sh
go test ./...
```

## Implemented

- `.tya` file runner
- 2-space indentation with `INDENT` / `DEDENT`
- comments with `#`
- variables and reassignment
- constants with `SCREAMING_SNAKE_CASE` and reassignment checks
- `nil`, booleans, ints, floats, strings
- string interpolation with expressions: `"next year: {age + 1}"`
- arrays and objects
- inline object literals: `{ name: "komagata" }`
- functions and implicit last-expression return
- explicit `return`
- method calls with `@property`
- arithmetic, comparison, equality, and logical operators
- grouped expressions with parentheses
- `if` / `else`
- `while`, `break`, and `continue`
- array `for item in items` and object `for key, value of object` loops
- builtins: `print`, `len`, `push`, `pop`, `keys`, `values`, `has`, `delete`,
  `equal`, `split`, `join`, `trim`,
  `replace`, `contains`, `startsWith`, `endsWith`, `byteLen`, `charLen`,
  `readFile`, `writeFile`, `fileExists`, `args`, `env`, `exit`, `panic`,
  `div`, `toString`, `toInt`, `toFloat`, `toNumber`

## Examples

```tya
greet = user -> "Hello, {user.name}"

user =
  name: "komagata"
  age: 20

print greet user
```

```tya
counter =
  count: 0

  inc: ->
    @count = @count + 1
    @count

print counter.inc()
print counter.inc()
```
