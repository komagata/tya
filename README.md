<p align="center">
  <a href="https://tya-lang.org/">
    <img src="docs/assets/tya-logo.png" alt="Tya logo" width="220">
  </a>
</p>

# Tya

Tya is a small indentation-based dynamic language with CoffeeScript feel and
Golang practicality.

Tya v0.13 is a compile-to-C language. The reference implementation contains a Go
lexer, parser, AST, checker, C emitter, C runtime, CLI, examples, and v0.13
specification tests.

Website: <https://tya-lang.org/>

## Requirements

- A C compiler available as `cc`

## Install

On macOS, install Tya with Homebrew:

```sh
brew install komagata/tap/tya
```

For local formula development from this repository:

```sh
brew install --HEAD ./Formula/tya.rb
```

For v0.25.0, download the release source and build the `tya` command locally.
This currently requires Go because the v0.13 reference implementation is written
in Go.

```sh
curl -L https://github.com/komagata/tya/archive/refs/tags/v0.25.0.tar.gz | tar xz
cd tya-0.25.0
go build -o tya ./cmd/tya
./tya version
```

## Run

Create `hello.tya`.

```tya
print "Hello, Tya"
```

```sh
tya run hello.tya
```

`tya run` builds a temporary executable, runs it, and removes the temporary
file after execution.

To keep the executable:

```sh
tya build hello.tya -o hello
./hello
```

To print the installed version:

```sh
tya version
```

To check, format, or inspect generated C:

```sh
tya check hello.tya
tya fmt hello.tya
tya fmt -w hello.tya
tya emit-c hello.tya
tya test
tya test tests
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

Standard attached libraries ship with Tya and use the same import syntax.

```tya
import string
import array

print string.blank("  ")
print array.first(["tya"])
```

## Documentation

- [Guide](https://tya-lang.org/guide.html): read this first to learn Tya.
- [Spec](https://tya-lang.org/spec.html): latest Tya language specification.
- [API](https://tya-lang.org/api.html): latest built-in function reference.
- [Stdlib](https://tya-lang.org/stdlib.html): standard attached library reference.
- [Naming](https://tya-lang.org/naming.html): naming rules.
- [Versions](https://tya-lang.org/versions.html): minor-version specs and release
  snapshots.
- [v0.26 Spec](https://tya-lang.org/v0.26/spec.html): external packages, `tya.toml`, and version resolution.
- [v0.25 Spec](https://tya-lang.org/v0.25/spec.html): bitwise operators, byte sequences, and binary file I/O.
- [v0.24 Spec](https://tya-lang.org/v0.24/spec.html): time, random, math expansion, process, hex, digest, secure_random, and matrix standard modules.
- [v0.23 Spec](https://tya-lang.org/v0.23/spec.html): TOML, JSON, CSV, base64, and URL standard modules.
- [v0.22 Spec](https://tya-lang.org/v0.22/spec.html): unittest standard module, module reflection, and `tya test` runner.
- [v0.21 Spec](https://tya-lang.org/v0.21/spec.html): native-backed `file` and `os` standard modules.
- [v0.20 Spec](https://tya-lang.org/v0.20/spec.html): standard attached `math` and `path` modules.
- [v0.19 Spec](https://tya-lang.org/v0.19/spec.html): predicate function and method names ending with `?`.
- [v0.18 Spec](https://tya-lang.org/v0.18/spec.html): expanded module-style string, array, and dict APIs.
- [v0.17 Spec](https://tya-lang.org/v0.17/spec.html): import aliases and module loading rules.
- [v0.16 Spec](https://tya-lang.org/v0.16/spec.html): pattern matching and string interpolation polish.
- [v0.15 Spec](https://tya-lang.org/v0.15/spec.html): structured error handling.
- [v0.14 Spec](https://tya-lang.org/v0.14/spec.html): destructuring assignment.
- [v0.13 Spec](https://tya-lang.org/v0.13/spec.html): explicit override and constructor chaining checks.
- [v0.12 Spec](https://tya-lang.org/v0.12/spec.html): interface inheritance and conflict diagnostics.
- [v0.11 Spec](https://tya-lang.org/v0.11/spec.html): explicit interfaces and implements.
- [v0.10 Spec](https://tya-lang.org/v0.10/spec.html): abstract methods and final classes.
- [v0.9 Spec](https://tya-lang.org/v0.9/spec.html): class visibility, private members, and abstract classes.
- [v0.8 Spec](https://tya-lang.org/v0.8/spec.html): class-level inheritance and introspection.
- [v0.7 Spec](https://tya-lang.org/v0.7/spec.html): released single inheritance.
- [v0.6 Spec](https://tya-lang.org/v0.6/spec.html): released class variables and class methods.
- [v0.5 Spec](https://tya-lang.org/v0.5/spec.html): released class syntax.
- [v0.4 Spec](https://tya-lang.org/v0.4/spec.html): released testing direction.
- [Roadmap](https://tya-lang.org/roadmap.html): current remaining-work plan.

Markdown source files are kept in `docs/` for editing. The public website uses
the generated HTML pages under `docs/*.html`.

## Language Scope

The current released implementation, Tya v0.13, includes:

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
- same-directory and `TYA_PATH` `import module_name` lookup with
  `module.member` access
- attached standard library modules loaded from `stdlib/`
- standard builtins listed in the API document
- compile-to-C execution through `tya run`, `tya build`, and `tya emit-c`
- source checking through `tya check`
- test discovery and assertions through `tya test`
- conservative source formatting through `tya fmt`
- minimal classes, constructor calls, `init`, public instance fields,
  instance methods, instance field defaults, class variables, class methods,
  single inheritance, class-level inheritance, class introspection, private
  members, private constructors, abstract classes, abstract methods, final
  classes, explicit interfaces, interface inheritance, method overrides,
  class-method `self`, and `super(args...)`

Tya v0.13 does not include implicit interfaces, multiple inheritance,
visibility keywords, protected members, async, macros, package management, remote
module install, JSON or CSV parsers, native standard modules, mocking,
coverage, benchmark, watch mode, parallel test execution, set literals, import
aliases, or dictionary member access.

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
go run ./cmd/tya check examples/hello.tya
go run ./cmd/tya fmt examples/hello.tya
go run ./cmd/tya emit-c examples/hello.tya
go run ./cmd/tya version
```

Developer inspection commands are intentionally not part of the public CLI
surface, but they are useful when working on the compiler:

```sh
go run ./cmd/tya --tokens examples/hello.tya
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

Tya uses semantic versioning. Specification changes happen at the minor version
level, such as `v0.3` and `v0.4`. Patch releases such as `v0.3.1` must not
change language or standard-library semantics. Specification documents use
minor-version labels such as `v0.3`.

When changing planned language docs, keep minor-version documents under
`docs/vX.Y/` and regenerate the HTML pages. Release snapshots for exact patch
tags may live under `docs/vX.Y.Z/`; the v0.2.0 frozen documents live under
`docs/v0.2.0/`.

Before committing Go changes, format touched Go files and run the default test
suite. The default suite includes the maintained self-host fixed-point check.

```sh
gofmt -w path/to/changed.go
go test ./... -count=1
```

Historical pre-v0.1 self-host notes and experiments live under
`docs/archive/pre-v0.1/`. They are reference material, but the current
`selfhost/v01/compiler.tya` fixed point is maintained and must not regress.
