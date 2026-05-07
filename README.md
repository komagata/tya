<p align="center">
  <img src="docs/assets/tya-logo.png" alt="Tya logo" width="180">
</p>

# Tya

Tya is a small indentation-based dynamic language inspired by CoffeeScript.

This repository currently contains a Go lexer, parser, AST, checker,
C emitter, C runtime, and historical implementation experiments.

Tya v0.1 is defined as a compile-to-C language. The current language authority
is `docs/SPEC.md` plus `docs/API.md`; older self-host and class/module
planning documents are archived under `docs/archive/pre-v0.1/`.

## Run

```sh
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

## Documentation

- `docs/GUIDE.md`: read this first to learn Tya.
- `docs/SPEC.md`: Tya v0.1 language spec.
- `docs/API.md`: Tya v0.1 built-in function reference.
- `docs/NAMING.md`: naming rules.
- `ROADMAP.md`: current v0.1 implementation roadmap.
- `docs/archive/pre-v0.1/`: historical pre-v0.1 plans and self-host notes.

## v0.1 Scope

The v0.1 implementation is in progress. This list describes the frozen language
surface that the Go compile-to-C path should satisfy.

- `.tya` file runner
- 2-space indentation with `INDENT` / `DEDENT`
- comments with `#`
- variables and reassignment
- constants with `SCREAMING_SNAKE_CASE` and reassignment checks
- `nil`, booleans, ints, floats, strings
- string interpolation with expressions: `"next year: {age + 1}"`
- string escapes: `\"`, `\\`, `\n`, `\t`
- arrays and dictionaries
- mutable array elements: `items[1] = 20`
- dictionary literals: `{ name: "komagata" }`
- dictionary access with `dictionary["name"]`
- functions and implicit last-expression return
- explicit `return`
- multiple assignment and returns: `value, err = read_thing()`
- `try` propagation for `value, err`
- arithmetic, comparison, equality, and logical operators
- unary `not` and `-`
- grouped expressions with parentheses
- `if` / `elseif` / `else`
- `while`, `break`, and `continue`
- error values via `error "message"`
- array `for item in items` and dictionary `for key, value of dictionary` loops
- `module name` declarations and same-directory `import module_name`
- `module.member` access
- builtins: `print`, `len`, `push`, `pop`, `keys`, `values`, `has`, `delete`,
  `split`, `join`, `trim`, `replace`, `contains`, `starts_with`, `ends_with`,
  `read_file`, `write_file`, `file_exists`, `args`, `env`, `error`, `exit`,
  `panic`, `to_string`, `to_int`, `to_float`, `to_number`

## Examples

```tya
greet = user -> "Hello, " + user["name"]

user = { name: "komagata", age: 20 }

print greet user
```

```tya
module greeting
  hello = name -> "Hello, {name}"
```

```tya
import greeting

print greeting.hello("komagata")
```

Classic milestone examples live under `examples/classic/`:

```sh
go run ./cmd/tya examples/classic/fib.tya
go run ./cmd/tya run examples/classic/fizzbuzz.tya
```
