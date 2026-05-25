<p align="center">
  <a href="https://tya-lang.org/">
    <img src="docs/assets/tya-logo.png" alt="Tya logo" width="220">
  </a>
</p>

# Tya

**A language without hesitation.**

Tya removes the small decisions that interrupt your work — how to format,
which tool to install, what an error means.

## At a glance

- **Accepted Syntax + Formatted Syntax** — Tya accepts editing-friendly source,
  while `tya format` emits one deterministic representation.
- **Dynamically typed**, indentation-based, with strict semantics
  (no implicit conversions, no `nil` arithmetic).
- **Compiles to C** for a small, portable runtime.
- **All-in-one toolchain** (Gleam-style) — the `tya` binary holds the
  compiler, formatter, language server, test runner, doc generator, and
  package manager.
- **Kind diagnostics** (Elm-grade) — every error has a stable code, an
  expected/found block, an actionable hint, and a linked explanation.

The reference implementation is a hand-written Go compiler (lexer, parser,
AST, checker, C emitter) with a small C runtime and a self-hosted Tya
compiler that compiles itself.

Website: <https://tya-lang.org/>

## Requirements

- A C compiler available as `cc`

## Install

On macOS, install Tya with Homebrew:

```sh
brew tap komagata/tap
brew install tya
```

For local formula development from this repository:

```sh
brew install --HEAD ./Formula/tya.rb
```

For v0.67.7, download the release source and build the `tya` command locally.
This currently requires Go because the reference implementation is written
in Go.

```sh
curl -L https://github.com/komagata/tya/archive/refs/tags/v0.67.7.tar.gz | tar xz
cd tya-0.67.7
go build -o tya ./cmd/tya
./tya version
```

## Run

Create `hello.tya`.

```tya
print("Hello, Tya")
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
tya format hello.tya
tya format -w hello.tya
tya format --check hello.tya
tya emit-c hello.tya
tya test
tya test tests
```

### Project workflow

```sh
tya new app           # scaffold tya.toml + src/main.tya + .gitignore
cd app
tya task              # list tasks defined under [tasks] in tya.toml
tya task run          # run the named task
tya lint src                  # report unused locals (TYAL0001) under a path
tya lint --fix src            # rewrite TYAL0001 line-deletes and TYAL0003 `if true` unwraps in place
tya lint --format=json src    # machine-readable findings for CI consumers
tya doc               # print source-comment API documentation
tya doc --json stdlib # emit generated stdlib API documentation
tya doc --html ./out  # write a multi-page HTML site
```

`tya task <name> [args...]` POSIX-quotes the trailing args and
appends them to the task command (mirrors `$@`). Array-form tasks
run each entry under `/bin/sh -c` in order and stop on the first
failure. Table-form tasks support `cmds = [...]`, `parallel = true`,
`depends_on = [...]`, per-task `env = { KEY = "value" }`, and
`tya task <name> --watch` for reruns on project file changes.

### Editor integration

`tya lsp` runs the Language Server (LSP JSON-RPC 2.0 over stdio)
from the same binary as the compiler. It supports the full IDE feature set:

- Diagnostics on save / on change
- Formatting (full + range, backed by `tya format`)
- Hover signatures + doc comments
- Goto-definition (cross-file via `import`)
- References, rename (top-level + local + param scope-aware)
- Code actions (TYAL0001 / TYAL0003 quick fixes)
- Document outline + workspace symbols
- Semantic tokens, incremental document sync

Setup recipes ship in [`editors/`](./editors):

- [`editors/vscode/`](./editors/vscode) — TextMate grammar + TypeScript extension
- [`editors/vim/`](./editors/vim) — Vim / Neovim syntax, filetype, and indent files
- [`editors/neovim/`](./editors/neovim) — nvim-lspconfig
- [`editors/zed/`](./editors/zed) — Zed `settings.json`
- [`editors/emacs/`](./editors/emacs) — syntax coloring plus eglot / lsp-mode
- [`editors/github-linguist/`](./editors/github-linguist) — GitHub Linguist registration notes
- [`editors/TOKENS.md`](./editors/TOKENS.md) — shared syntax-color token taxonomy

VS Code extension:

Install Tya from the Visual Studio Marketplace or Open VSX.
For a local package build:

```sh
cd editors/vscode
npm install
npm run compile
npx vsce package
code --install-extension tya-0.65.2.vsix
```

## Example

```tya
user = { name: "komagata", age: 20 }

greet = user -> "Hello, {user["name"]}!"

if user["age"] >= 20
  print(greet(user))
```

Imports expose public top-level bindings from separate files.

```tya
# greeting.tya
hello = name -> "Hello, {name}"
```

```tya
# main.tya
import greeting

print(greeting.hello("komagata"))
```

Primitive values expose their standard operations as methods.

```tya
print("  ".blank?())
print(["tya"].first())
```

## Documentation

- [Guide](https://tya-lang.org/guide/): read this first to learn Tya.
- [Spec](https://tya-lang.org/spec/): latest Tya language specification.
- [Versions](https://tya-lang.org/versions/): frozen historical release docs.

The public website is built from the Jekyll source files under `docs/`.

## Language Scope

The current implementation on `main` includes:

- `.tya` files
- indentation-based blocks
- comments, assignments, multiple assignment, and constants
- `nil`, booleans, numbers, strings, arrays, dictionaries, functions, and
  errors
- string interpolation
- dictionary index access with `dictionary["name"]`
- function literals and function calls
- `if` / `elseif` / `else`
- `while`, `for`, `break`, and `continue`
- implicit final-expression returns, explicit `return`, and multiple return
  values
- `try` error propagation
- same-directory, package dependency, `TYA_PATH`, and standard library import
  lookup
- git and path package dependencies through `tya.toml`, `tya.lock`, and
  `tya install`
- native package metadata through `[native]`, `tya doctor native`, and
  `tya new --template lib --native`
- package-provided tools through `[tools]` and `tya tool`
- standard library packages and APIs loaded from `stdlib/`
- standard builtins listed in the specification
- compile-to-C execution through `tya run`, `tya build`, and `tya emit-c`
- source checking through `tya check`
- test discovery and assertions through `tya test`
- conservative source formatting through `tya format`
- WebAssembly build targets with unsupported native packages rejected for
  WebAssembly builds
- minimal classes, constructor calls, `init`, public instance fields,
  instance methods, instance field defaults, class variables, class methods,
  single inheritance, class-level inheritance, class introspection, private
  members, private constructors, abstract classes, abstract methods, final
  classes, explicit interfaces, interface inheritance, stackable interface
  defaults, interface fields, interface initializer hooks, method overrides,
  class-method `self`, and `super(args...)`

External packages and tools such as SQLite, SDL2, GTK4, raylib, Slim,
Flakewatch, and Magvideo live in separate `komagata/*` repositories and are
consumed by git URL plus tag.

Current Tya does not include multiple inheritance, protected members, async,
macros, a central package registry, `tya publish`, mocking, benchmark, watch
mode, parallel test execution, or set literals.

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
- `internal/checker/`: language and import validation.
- `internal/codegen/`: C emitter.
- `internal/runner/`: source loading, import loading, and run helpers.
- `runtime/`: C runtime used by generated programs.
- `cmd/tya/`: user-facing CLI.
- `tests/`: CLI, example, and specification-level tests.

Useful local commands:

```sh
go run ./cmd/tya run examples/hello.tya
go run ./cmd/tya build examples/hello.tya -o hello
go run ./cmd/tya check examples/hello.tya
go run ./cmd/tya format examples/hello.tya
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

The website is served by GitHub Pages from the Jekyll source files under
`docs/`.

```sh
bundle install
bundle exec jekyll build --source docs --destination _site
```

Before committing Go changes, format touched Go files and run the default test
suite. The default suite includes the maintained self-host fixed-point check.

```sh
gofmt -w path/to/changed.go
go test ./... -count=1
```

The current `selfhost/v01/compiler.tya` fixed point is maintained and must not
regress.
