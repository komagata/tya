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

- **Canonical Syntax** — every program has exactly one source representation.
  The formatter is part of the language, not a separate opinion.
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
brew install komagata/tap/tya
```

For local formula development from this repository:

```sh
brew install --HEAD ./Formula/tya.rb
```

For v0.61.0, download the release source and build the `tya` command locally.
This currently requires Go because the reference implementation is written
in Go.

```sh
curl -L https://github.com/komagata/tya/archive/refs/tags/v0.61.0.tar.gz | tar xz
cd tya-0.61.0
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
tya emit-c hello.tya
tya test
tya test tests
```

### Project workflow (v0.49+)

```sh
tya new app           # scaffold tya.toml + src/main.tya + .gitignore
cd app
tya task              # list tasks defined under [tasks] in tya.toml
tya task run          # run the named task
tya lint src                  # report unused locals (TYAL0001) under a path
tya lint --fix src            # rewrite TYAL0001 line-deletes and TYAL0003 `if true` unwraps in place
tya lint --format=json src    # machine-readable findings for CI consumers
tya doc               # print top-level binding documentation
tya doc --html ./out  # write a multi-page HTML site
```

`tya task <name> [args...]` POSIX-quotes the trailing args and
appends them to the task command (mirrors `$@`). Array-form tasks
run each entry under `/bin/sh -c` in order and stop on the first
failure.

### Editor integration (v0.53+)

`tya lsp` runs the Language Server (LSP JSON-RPC 2.0 over stdio)
from the same binary as the compiler. v0.53 supports the full IDE
feature set:

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

VS Code manual install:

```sh
cd editors/vscode
npm install
npm run compile
npx vsce package
code --install-extension tya-0.61.0.vsix
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

- [Guide](https://tya-lang.org/guide.html): read this first to learn Tya.
- [Spec](https://tya-lang.org/spec.html): latest Tya language specification.
- [API](https://tya-lang.org/api.html): latest built-in function reference.
- [Stdlib](https://tya-lang.org/stdlib.html): standard library reference.
- [Naming](https://tya-lang.org/naming.html): naming rules.
- [Versions](https://tya-lang.org/versions.html): minor-version specs and release
  snapshots.
- [v0.28 Spec](https://tya-lang.org/v0.28/spec.html): strict compile-time checks (shadowing, unused imports/args/private definitions).
- [v0.27 Spec](https://tya-lang.org/v0.27/spec.html): hexadecimal and binary integer literals.
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
  lookup with namespace member access
- git and path package dependencies through `tya.toml`, `tya.lock`, and
  `tya install`
- native package metadata through `[native]`, `tya doctor native`, and
  `tya new --template lib --native`
- package-provided tools through `[tools]` and `tya tool`
- standard library modules loaded from `stdlib/`
- standard builtins listed in the API document
- compile-to-C execution through `tya run`, `tya build`, and `tya emit-c`
- source checking through `tya check`
- test discovery and assertions through `tya test`
- conservative source formatting through `tya format`
- WebAssembly build targets with unsupported native packages rejected for WASM
  builds
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

Tya v0.61 does not include multiple inheritance, protected members, async,
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
