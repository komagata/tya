# Tya Roadmap

`ROADMAP.md` is the single source of truth for current TODO, TASK, and roadmap
planning.

Pre-v0.1 planning documents and self-host migration notes are archived under
[`docs/archive/pre-v0.1/`](docs/archive/pre-v0.1/). They are historical
references, not current language or implementation authority.

## Self-Host Invariant

The Tya-written compiler fixed point is a maintained invariant. Later language,
runtime, CLI, stdlib, and documentation work must not regress
`selfhost/v01/compiler.tya`.

Required evidence:

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
```

This gate proves that the Tya-written compiler can compile itself to stable
stage-2/stage-3 C output, and that the self-hosted stage-2 compiler can compile
and run representative programs through the maintained surface.

## v1.0.0 Goal

Tya v1.0.0 is the version at which all five language commitments hold and
are publicly defensible. These commitments are also Tya's external "at a
glance" feature list:

1. **Canonical Syntax** — every program has exactly one source representation.
   The formatter is part of the language, not a separate opinion.
   *(Shipped in v0.38 / v0.39.)*
2. **Dynamically typed**, indentation-based, with strict semantics
   (no implicit conversions, no `nil` arithmetic).
3. **Compiles to C** for a small, portable runtime.
4. **All-in-one toolchain** (Gleam-style) — the `tya` binary holds the
   compiler, formatter, language server, test runner, doc generator, and
   package manager.
5. **Kind diagnostics** (Elm-grade) — every error has a stable code, an
   expected/found block, an actionable hint, and a linked explanation.

Each commitment maps to one or more Epics below:

- Commitment 1 → *(landed)*.
- Commitment 2 → strict-semantics audit (currently implicit; to be made
  an explicit Epic).
- Commitment 3 → already shipped; maintained.
- Commitment 4 → *Ship `tya lsp` Language Server*, *Ship `tya doc` source
  documentation generator*, *Ship `tya new` project scaffolder*, *Ship
  `tya task` project task runner*, plus existing `tya check` /
  `tya test` / `tya format` / package manager.
- Commitment 5 → *Migrate remaining stages to the diagnostics pipeline*.

Other Epics (GC, WASM target, syntax coloring, embedding,
self-introspection library, coverage extensions, markdown extensions,
lint, multi-line string extensions) are valuable but not strictly
required for v1.0.0. They may ship before or after v1.0.0.

## Current Direction

Tya is implemented as a small compile-to-C language. The latest released
specification is **v0.39**. Frozen release documents live under
`docs/vX.Y.Z/` and `docs/vX.Y/`; the latest editable specification, API,
stdlib, and naming documents live directly under `docs/`.

Tya uses semantic versioning. Specification changes happen at the minor version
level. Patch releases must not change language or standard-library semantics.

Latest editable documentation:

1. [`docs/SPEC.md`](docs/SPEC.md)
1. [`docs/API.md`](docs/API.md)
1. [`docs/STDLIB.md`](docs/STDLIB.md)
1. [`docs/NAMING.md`](docs/NAMING.md)
1. [`docs/CANONICAL_SYNTAX.md`](docs/CANONICAL_SYNTAX.md)

The reference implementation is:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
C runtime
specification tests
```

Go interpreter behavior, ASTMODE, and legacy archived node-string experiments
are not specification authority. The maintained `selfhost/v01/compiler.tya`
fixed point must not regress.

## Implementation Tooling Policy

The compiler implementation should stay hand-written:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
```

Do not add a parser generator or large grammar framework. In particular, avoid
introducing Participle, goyacc, Pigeon, ANTLR, or Tree-sitter as compiler
front-end authority. They may be useful references or future editor tooling,
but the active compiler path should remain explicit Go code.

After the Go implementation reaches a complete lexer, parser, AST, checker, and
C emitter for the current specification, continue self-host work in the same
component order:

```text
Tya lexer
Tya parser
Tya AST
Tya checker
Tya C emitter
```

Each Tya component must preserve the self-host fixed point before moving to the
next component.

Use small test-support dependencies where they make the specification easier to
verify:

```text
github.com/google/go-cmp/cmp
github.com/rogpeppe/go-internal/testscript
```

Use `go-cmp` for readable token, AST, diagnostic, and generated-output diffs.
Use `testscript` for CLI-level specification tests, especially `tya run`,
`tya build`, expected stdout/stderr, and negative examples.

## Released

Recent shipped minor versions, newest first. Frozen specs live under
`docs/vX.Y/`.

- **v0.39** — Canonical Syntax surface cleanup. `tya format` only spelling;
  `--text` / `--ast` opt-outs removed.
- **v0.38** — Canonical Syntax landing. AST-driven serializer is the default;
  examples / stdlib / selfhost normalized; selfhost compiler extended for
  canonical wrap forms.
- **v0.37** — `formatter.Unparse` foundation for the common-case subset.
- **v0.36** — comment attachment recurses into nested Stmt-list bodies.
- **v0.35** — per-stmt `Comments` map at top level.
- **v0.34** — lexer comment capture; `Program.HeaderComments`.
- **v0.33** — parser accepts `(a, b) -> body` parenthesized multi-parameter
  lambda.
- **v0.32** — lexer diagnostics migration (`TYA-E0001`–`TYA-E0017`); pure-Tya
  `markdown.to_html` foundation.
- **v0.31** — multi-line `"""..."""` string literals.
- **v0.30** — test coverage foundation: `tya test --cover`, `tya cover`,
  `# tya-cover 1` profile format.
- **v0.29** — diagnostics foundation: `internal/diag` model, human + JSON
  renderers, color modes, `TYA-Xnnnn` code namespace.
- **v0.28** — strict compile-time checks (shadowing, unused
  imports/args/private definitions).
- **v0.27** — hexadecimal and binary integer literals.
- **v0.26** — external packages, `tya.toml`, version resolution.
- **v0.25** — bitwise operators, byte sequences, binary file I/O.
- **v0.24** — `time`, `random`, `process`, `hex`, `digest`, `secure_random`,
  `matrix` modules; `math` expansion.

## Future Work

Epics below are committed direction but not yet scheduled to a specific
minor version. Each will be scoped into a `docs/vX.Y/SPEC.md` when picked up.

### Toolchain

- [ ] **Migrate remaining stages to the diagnostics pipeline**
  - [ ] Parser → `TYA-E0100`–`E0299`.
  - [ ] Codegen → `TYA-E0600`–`E0799`.
  - [ ] Runner → `TYA-E0800`–`E0899`.
  - [ ] Add did-you-mean suggestions for unknown-name diagnostics.
  - [ ] Add multi-error parsing.

- [ ] **Ship `tya lsp` Language Server**
  - [ ] Define LSP scope; ship `tya lsp` as a subcommand of the same
    binary so compiler and language server cannot drift in version.
  - [ ] Speak LSP over stdio (JSON-RPC) for VS Code, Zed, Helix, Neovim,
    Emacs.
  - [ ] Implement diagnostics on save / on change, hover, go-to-definition,
    find-references, completion (in-scope names + module members + stdlib),
    formatting (full + range, backed by `tya format`), rename, code
    actions for common diagnostics.
  - [ ] Publish a minimal VS Code extension and document Zed / Helix /
    Neovim / Emacs setup.

- [ ] **Ship `tya doc` source documentation generator**
  - [ ] Define doc comment syntax: contiguous comment lines immediately
    preceding a top-level definition. Body is Markdown rendered with the
    `markdown` stdlib module.
  - [ ] Discover every top-level binding under `src/` plus stdlib re-exports.
  - [ ] CLI surface: `tya doc` (text), `tya doc --html <out>` (static site),
    `tya doc --serve` (HTTP), `tya doc --json` (machine-readable).
  - [ ] Reuse the public Tya self-introspection library.
  - [ ] Diagnose orphan doc comments, duplicate definitions, unparseable
    Markdown bodies.

- [ ] **Ship `tya new` project scaffolder**
  - [ ] CLI: `tya new <name>`, `tya new --here`, fixed templates
    (`--template app|lib`).
  - [ ] Default template: `tya.toml`, `src/main.tya` hello-world, `tests/`
    with one passing unittest, `.gitignore`, minimal `README.md`.
  - [ ] Refuse to overwrite existing files unless `--force`. Initialize git
    by default; `--no-git` to skip.

- [ ] **Ship `tya task` project task runner**
  - [ ] `[tasks]` table in `tya.toml` is the single source of truth.
  - [ ] String form (`ci = "tya format && tya test"`) and array form.
  - [ ] CLI: `tya task <name>` and `tya task` (list).
  - [ ] Resolve `tya.toml` from working directory upward; execute via the
    user's shell with the project root as CWD.
  - [ ] Keep parallelism, file-watching, task graphs out of the initial
    scope.

- [ ] **Ship `tya lint` source linter**
  - [ ] Boundary: `tya format` is layout, `tya check` is correctness, `tya
    lint` is stylistic / semantic best practices.
  - [ ] CLI: `tya lint [paths]`, `tya lint --fix`, `tya lint --format=json`.
  - [ ] Initial rule set: unused locals, dead code after `return` / `raise`,
    redundant `if true` / `if false`, suspicious `for` index patterns,
    deeply nested blocks, very long functions.
  - [ ] Each rule has a stable code (e.g. `TYAL0001`), title, doc URL.
  - [ ] Per-line opt-out via `# tya-lint-ignore: TYAL0001`.

- [ ] **Ship a public Tya self-introspection library**
  - [ ] Goal: Tya programs and external tools can lex, parse, walk, and
    re-emit Tya source through a stable, documented API. Unlocks
    codemods, linters, doc extractors, refactoring, AI-agent edits, and
    `tya lsp`.
  - [ ] Surface as Tya stdlib modules built on the self-host components:
    `compiler.lexer`, `compiler.parser`, `compiler.ast`, `compiler.checker`,
    `compiler.format`.
  - [ ] AST is the single canonical representation; node kinds, fields,
    spans documented.
  - [ ] Round-trip tests over a representative corpus.

### Language

- [ ] **Ship WebAssembly compilation target**
  - [ ] Use Zig (`zig cc --target=wasm32-wasi` and
    `zig cc --target=wasm32-freestanding`) as the WASM toolchain.
  - [ ] CLI: `tya build --target wasm32-wasi`,
    `tya build --target wasm32-browser`.
  - [ ] Decide system interface: WASI for CLI-style programs, browser-friendly
    subset omits filesystem and process APIs.
  - [ ] Gate unavailable stdlib modules per target with structured errors at
    import time.

- [ ] **Ship asset embedding for single-binary distribution**
  - [ ] `embed "assets/logo.png" as logo` bakes file bytes into the compiled
    binary at build time.
  - [ ] Directory / glob form (`embed "static/**" as static`) returns a dict
    keyed by relative path.
  - [ ] Text loads as `string`, binary as `bytes`; explicit `as bytes` /
    `as text` modifier supported.

- [ ] **Ship a garbage collector for the C runtime**
  - [ ] Conservative mark-and-sweep over current runtime values (string,
    dict, array, bytes, error, function closure).
  - [ ] Allocation-threshold trigger; explicit `runtime.gc()` for
    tests/benchmarks.
  - [ ] Roots: value stack, active locals, module globals,
    finalized-but-not-yet-collected closures.
  - [ ] Defer generational, incremental, concurrent variants and weak refs /
    finalizers.

### Stdlib extensions

- [ ] **Extend coverage tooling (post-v0.30.0)**
  - [ ] `tya cover html` self-contained report.
  - [ ] `--include` / `--exclude` glob filters on `tya cover`.
  - [ ] Read `coverage.include` / `coverage.exclude` from `Tyafile`.
  - [ ] Give `BreakStmt` / `ContinueStmt` source positions and instrument
    them.
  - [ ] Map coverage to per-import-source lines instead of the synthesized
    entry path.
  - [ ] Recommended CI snippet (fail when coverage drops below a threshold).

- [ ] **Extend `markdown` stdlib module (post-v0.32 foundation)**
  - [ ] Public AST: `markdown.parse(text) -> ast`, `markdown.to_html_ast(ast)
    -> string`, `markdown.render(ast, visitor) -> string`, AST node spec.
  - [ ] GFM extensions: tables, task lists, strikethrough, fenced-code
    info-string class.
  - [ ] Reference link definitions, images, nested lists, setext headings,
    HTML blocks.
  - [ ] Raw HTML pass-through with security note.
  - [ ] CommonMark conformance subset run as part of `go test ./...`.

- [ ] **Extend multi-line strings (post-v0.31.0)**
  - [ ] Raw-string prefix `r"""..."""`.
  - [ ] Bytes equivalent `b"""..."""`.
  - [ ] Heredoc-style markers, language-tagged interpolation specifiers (if
    needed).

### Editor and ecosystem

- [ ] **Ship syntax coloring for the major editors**
  - [ ] Define **major editors**: VS Code, Emacs, Vim, GitHub. Others (Zed,
    Helix, Neovim, JetBrains) welcome but not required for "major editor"
    coverage.
  - [ ] Canonical token taxonomy (keyword, builtin, type, string,
    interpolated expression, number, operator, comment, function name,
    parameter, module name) documented in `docs/`.
  - [ ] House all editor assets under `editors/`: `editors/vscode/`,
    `editors/emacs/`, `editors/vim/`, `editors/tree-sitter-tya/`.
  - [ ] VS Code: TextMate grammar + minimal extension; publish to
    Marketplace and Open VSX.
  - [ ] Emacs: `tya-mode.el` derived mode; publish to MELPA.
  - [ ] Vim: `syntax/`, `ftdetect/`, `indent/` files (also covers Neovim).
  - [ ] GitHub: Tree-sitter grammar registered with `github-linguist/linguist`.

## Verification Reference

Default verification:

```sh
go test ./... -count=1
```

Focused verification should prefer tests for the touched lexer, parser, checker,
C emitter, runtime, examples, stdlib, or docs. The self-host fixed-point gate is
part of the maintained project invariant and must stay green.
