# Tya

Tya is a small indentation-based dynamic language inspired by CoffeeScript.

This repository currently contains a Go lexer, parser, AST, checker,
interpreter, C emitter, and C runtime.

## Run

```sh
go run ./cmd/tya examples/hello.tya
go run ./cmd/tya run examples/hello.tya
go run ./cmd/tya --tokens examples/hello.tya
go run ./cmd/tya --emit-c examples/arithmetic.tya
go run ./cmd/tya --check-unused examples/hello.tya
```

`tya run` compiles the source to C in a temporary directory, builds it with
`cc`, and then runs the resulting binary.

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
- string escapes: `\"`, `\\`, `\n`, `\t`
- string indexing: `"tya"[1]`
- arrays and objects
- mutable array elements: `items[1] = 20`
- inline object literals: `{ name: "komagata" }`
- functions and implicit last-expression return
- explicit `return`
- multiple assignment and returns: `value, err = read_thing()`
- `try` propagation for `value, err`
- method calls with `@property`
- arithmetic, comparison, equality, and logical operators
- unary `not` and `-`
- grouped expressions with parentheses
- `if` / `else`
- `while`, `break`, and `continue`
- error values via `error "message"`
- array `for item in items` and object `for key, value of object` loops
- builtins: `print`, `len`, `push`, `pop`, `map`, `filter`, `find`, `any`,
  `all`, `each`, `reduce`, `keys`, `values`, `has`, `delete`,
  `equal`, `split`, `join`, `trim`,
  `replace`, `contains`, `starts_with`, `ends_with`, `byte_len`, `char_len`,
  `read_line`, `read_file`, `write_file`, `file_exists`, `args`, `env`, `error`,
  `exit`, `panic`,
  `div`, `to_string`, `to_int`, `to_float`, `to_number`
- `stdlib/prelude.tya` is loaded by the runner and `--emit-c`

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

Classic milestone examples live under `examples/classic/`:

```sh
go run ./cmd/tya examples/classic/fib.tya
go run ./cmd/tya run examples/classic/fizzbuzz.tya
```
