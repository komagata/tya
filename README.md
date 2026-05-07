<p align="center">
  <a href="https://tya-lang.org/">
    <img src="docs/assets/tya-logo.png" alt="Tya logo" width="220">
  </a>
</p>

# Tya

Tya is a small indentation-based dynamic language with CoffeeScript feel and
Golang practicality.

Tya v0.1 is a compile-to-C language. The reference implementation contains a Go
lexer, parser, AST, checker, C emitter, C runtime, CLI, examples, and v0.1
specification tests.

Website: <https://tya-lang.org/>

## Requirements

- Go
- A C compiler available as `cc`

## Install

```sh
go install github.com/komagata/tya/cmd/tya@v0.1.0
```

For local development from this repository:

```sh
go run ./cmd/tya version
```

## Run

```sh
tya run examples/hello.tya
```

`tya run` builds a temporary executable, runs it, and removes the temporary
file after execution.

To keep the executable:

```sh
tya build examples/hello.tya -o hello
./hello
```

To print the installed version:

```sh
tya version
```

## Example

```tya
user =
  name: "komagata"
  age: 20

greet = user -> "Hello, " + user["name"]

if user["age"] >= 20
  print greet(user)
```

Modules live in separate files.

```tya
# greeting.tya
module greeting
  hello = name -> "Hello, {name}"
```

```tya
# main.tya
import greeting

print greeting.hello("komagata")
```

## Documentation

- [Guide](https://tya-lang.org/guide.html): read this first to learn Tya.
- [Spec](https://tya-lang.org/spec.html): Tya v0.1 language specification.
- [API](https://tya-lang.org/api.html): v0.1 built-in function reference.
- [Naming](https://tya-lang.org/naming.html): naming rules.
- [Roadmap](https://tya-lang.org/roadmap.html): current remaining-work plan.

Markdown source files are kept in `docs/` for editing. The public website uses
the generated HTML pages under `docs/*.html`.

## Language Scope

Tya v0.1 includes:

- `.tya` files
- indentation-based blocks
- comments, assignments, multiple assignment, and constants
- `nil`, booleans, numbers, strings, arrays, dictionaries, functions, errors,
  and modules
- string interpolation
- dictionary index access with `dictionary["name"]`
- function literals and function calls
- `if` / `elseif` / `else`
- `while`, `for`, `break`, and `continue`
- implicit final-expression returns, explicit `return`, and multiple return
  values
- `try` error propagation
- same-directory `import module_name` and `module.member` access
- standard builtins listed in the API document
- compile-to-C execution through `tya run` and `tya build`

Tya v0.1 does not include objects, classes, interfaces, inheritance, async,
macros, package management, set literals, import aliases, or dictionary member
access.

## Test

```sh
go test ./... -count=1
```
