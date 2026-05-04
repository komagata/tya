<p align="center">
  <img src="docs/assets/tya-logo.png" alt="Tya logo" width="180">
</p>

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

## Self-Host Gate

Run the complete self-host bootstrap gate from a clean checkout with:

```sh
sh scripts/selfhost_bootstrap_check.sh
```

This verifies the Tya-written compiler components, generated-C compile checks,
the stage-generated supported-example parity gate, repeated bootstrap stages,
and deterministic generated-C fixed-point checks. Full language parity work is
tracked in `SELFHOST_WORK.md`.

## Documentation

- `docs/GUIDE.md`: read this first to learn Tya.
- `docs/REFERENCE.md`: compact language reference.
- `docs/API.md`: built-in function reference.
- `docs/NAMING.md`: naming rules.
- `docs/CLASS_MODULE_DESIGN.md`: planned class/module/dict/set design.

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
- arrays and dictionaries
- mutable array elements: `items[1] = 20`
- inline dictionary literals: `{ name: "komagata" }`
- functions and implicit last-expression return
- explicit `return`
- multiple assignment and returns: `value, err = read_thing()`
- `try` propagation for `value, err`
- legacy method calls with `@property`
- arithmetic, comparison, equality, and logical operators
- unary `not` and `-`
- grouped expressions with parentheses
- `if` / `else`
- `while`, `break`, and `continue`
- error values via `error "message"`
- array `for item in items` and dictionary `for key, value of dictionary` loops
- builtins: `print`, `len`, `push`, `pop`, `map`, `filter`, `find`, `any`,
  `all`, `each`, `reduce`, `keys`, `values`, `has`, `delete`,
  `equal`, `split`, `join`, `trim`,
  `replace`, `contains`, `starts_with`, `ends_with`, `byte_len`, `char_len`,
  `read_line`, `read_file`, `write_file`, `file_exists`, `args`, `env`, `error`,
  `exit`, `panic`,
  `div`, `to_string`, `to_int`, `to_float`, `to_number`
- `stdlib/prelude.tya` is loaded by the runner and `--emit-c`
- same-directory modules with `import file_name`; each module file exposes
  exactly one public top-level binding matching the file name

## Examples

```tya
greet = user -> "Hello, " + user["name"]

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
