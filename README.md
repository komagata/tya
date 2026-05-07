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
- [Versions](https://tya-lang.org/versions.html): release snapshots of the spec
  and API.
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

## Development

Clone the repository and verify the local toolchain first.

```sh
git clone https://github.com/komagata/tya.git
cd tya
go run ./cmd/tya version
go test ./... -count=1
```

The compiler is intentionally hand-written Go. The main implementation areas
are:

- `internal/lexer/`: source text to tokens.
- `internal/parser/`: tokens to AST.
- `internal/ast/`: AST node definitions.
- `internal/checker/`: language and module validation.
- `internal/codegen/`: C emitter.
- `internal/runner/`: source loading, module loading, and run helpers.
- `runtime/`: C runtime used by generated programs.
- `cmd/tya/`: user-facing CLI.
- `tests/`: CLI, example, and specification-level tests.

Useful local commands:

```sh
go run ./cmd/tya run examples/hello.tya
go run ./cmd/tya build examples/hello.tya -o hello
go run ./cmd/tya version
```

Developer inspection commands are intentionally not part of the public CLI
surface, but they are useful when working on the compiler:

```sh
go run ./cmd/tya --tokens examples/hello.tya
go run ./cmd/tya --emit-c examples/hello.tya
go run ./cmd/tya --check-unused examples/hello.tya
```

Run focused compile-to-C checks when changing examples, argument handling, C
emission, imports, runtime execution, or stdlib loading.

```sh
sh scripts/go_emit_examples_check.sh
sh scripts/go_emit_args_check.sh
```

The website is served from `docs/` by GitHub Pages. Markdown source files are
converted to static HTML pages with:

```sh
node scripts/build_docs_pages.js
```

When changing released language docs, keep versioned snapshots under
`docs/vX.Y.Z/` and regenerate the HTML pages. For v0.1.0, the frozen documents
live under `docs/v0.1.0/`.

Before committing Go changes, format touched Go files and run the default test
suite.

```sh
gofmt -w path/to/changed.go
go test ./... -count=1
```

Historical pre-v0.1 self-host notes and experiments live under
`docs/archive/pre-v0.1/`. They are reference material, not current v0.1
authority or default verification gates.
