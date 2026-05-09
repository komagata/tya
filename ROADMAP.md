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
6. **Self-hosted** — the Tya compiler is written in Tya itself. The Go
   reference implementation is removed at v1.0.0 release. Bootstrap from
   a pre-built `tya` binary distributed via the project's release
   channels.

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
- Commitment 6 → *Migrate selfhost compiler to latest spec and remove
  the Go reference implementation* (see Future Work / Self-host).

Other Epics (WASM target, syntax coloring, embedding,
self-introspection library, coverage extensions, markdown extensions,
lint, multi-line string extensions) are valuable but not strictly
required for v1.0.0. They may ship before or after v1.0.0.

Scheduled minor versions on the path to v1.0.0:

- **v0.41** — Ship a precise mark-and-sweep garbage collector for the C
  runtime. Single-threaded, stop-the-world. Foundation for v0.42
  Concurrency.
- **v0.42** — Ship Tya Concurrency: `spawn` / `await` / `scope`
  keywords, `channel` / `sync` stdlib modules, `task` value type. 1:1
  OS-thread implementation; multi-thread GC extension. M:N scheduler
  is deferred to a later minor.

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

- **v0.40** — raw and bytes string extensions: `r"..."`, `r"""..."""`,
  `b"""..."""`.
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

## Scheduled

Epics with assigned minor versions, on the path to v1.0.0. Each Epic is
implemented in numbered STEPs. Every STEP must pass `go test ./... -count=1`
and `go test ./tests -run TestSelfhostV01Scripts -count=1` before the next
STEP starts. The STEP also keeps `docs/vX.Y/SPEC.md` consistent with the
implementation up to that STEP.

### v0.41 — Garbage collector

Single-threaded precise mark-and-sweep over the C runtime. Foundation for
v0.42 Concurrency (multi-thread GC extension lives in v0.42).

- [ ] **Ship v0.41 GC**
  - [ ] STEP 1: GC-aware allocator
    - [ ] Add a GC header (mark bit, kind field reuse) to every heap
      allocation in `runtime/`.
    - [ ] Route every `malloc` for runtime values (string, dict, array,
      bytes, error, function/closure environment, future task) through a
      central GC allocator.
    - [ ] Maintain a doubly-linked list (or vector) of live heap objects
      for scan/sweep.
    - [ ] Self-host fixed point preserved.
  - [ ] STEP 2: Mark phase
    - [ ] Implement root scanning: value stack, active locals, module
      globals, currently-active closure environments, error reraise
      slots.
    - [ ] Implement transitive mark over reachable values; handle cycles.
    - [ ] Add tests: simple reachability, cycle reachability, large
      object graphs.
    - [ ] Self-host fixed point preserved.
  - [ ] STEP 3: Sweep phase
    - [ ] Free unmarked objects; return memory to the allocator.
    - [ ] Reset mark bits after sweep.
    - [ ] Add cycle-reclamation tests; verify with `valgrind` or
      `leaks` on representative programs.
    - [ ] Self-host fixed point preserved.
  - [ ] STEP 4: Trigger policy + `runtime.gc()` API
    - [ ] Allocation-threshold trigger: GC runs when bytes-allocated
      since last GC exceeds a fraction of live-after-last-GC.
    - [ ] Add `runtime.gc()` builtin for tests / benchmarks (forces a GC).
    - [ ] Add `runtime.gc_stats()` returning collected/live/cycles.
    - [ ] Stress tests for bounded resident set on aggressive
      allocation loops.
    - [ ] Self-host fixed point preserved.
  - [ ] STEP 5: Documentation, examples, v0.41 SPEC finalization
    - [ ] Add `docs/v0.41/SPEC.md` describing observable GC behavior
      (no user-visible semantics changes; only memory pressure and
      timing).
    - [ ] Update `docs/SPEC.md` to point to v0.41.
    - [ ] Add `examples/long_running_loop.tya` demonstrating bounded
      resident set.
    - [ ] Run focused regressions: `sh scripts/go_emit_examples_check.sh`,
      `sh scripts/go_emit_args_check.sh`.
    - [ ] Self-host fixed point preserved.

### v0.42 — Tya Concurrency

`spawn` / `await` / `scope` keywords plus `channel` / `sync` stdlib
modules. 1:1 OS-thread implementation backed by `pthread`. M:N scheduler
deferred to a later minor.

- [ ] **Ship v0.42 Tya Concurrency**
  - [ ] STEP 1: Lexer / parser / AST for `spawn` / `await` / `scope`
    - [ ] Reserve keywords `spawn`, `await`, `scope`.
    - [ ] Add AST nodes: `SpawnExpr`, `AwaitExpr`, `ScopeBlock`.
    - [ ] Parser accepts: `spawn fn(args)`, `await task_expr`, `scope`
      indented block.
    - [ ] Checker: `spawn` requires callable, `await` requires task,
      `scope` body is a statement list.
    - [ ] Codegen returns "not yet implemented" structured diagnostic.
    - [ ] Self-host fixed point preserved.
  - [ ] STEP 2: Multi-thread GC extension + thread-safe allocator
    - [ ] Mutex-protect the allocator from STEP 1 of v0.41.
    - [ ] Stop-the-world for GC: suspend all worker threads at safe
      points, scan each thread's value stack and active locals, then
      resume.
    - [ ] Add a `task` value kind (`TYA_TASK`) and runtime task struct
      (pthread_t, return slot, completion condvar, cancel flag).
    - [ ] Self-host fixed point preserved.
  - [ ] STEP 3: `spawn` / `await` codegen and runtime
    - [ ] Codegen `spawn fn(args)` → spawn helper that creates a
      pthread running the function with copied/shared args.
    - [ ] Codegen `await task` → join helper returning the task's
      return value.
    - [ ] Propagate `raise` from a spawned task to its `await` caller.
    - [ ] Tests: single spawn/await, many spawn/await, raise
      propagation.
    - [ ] Self-host fixed point preserved.
  - [ ] STEP 4: `scope` block — structured concurrency
    - [ ] Codegen `scope`: track tasks created inside the block, await
      all on normal exit.
    - [ ] On `raise` inside the scope, signal cancel to outstanding
      tasks and re-raise after they settle.
    - [ ] Add `task.is_cancelled()` poll API for cooperative cancel.
    - [ ] Tests: bounded lifetime, raise-cancels-siblings, nested
      scopes.
    - [ ] Self-host fixed point preserved.
  - [ ] STEP 5: `channel` stdlib module — basic send / receive
    - [ ] Runtime: channel struct (mutex + condvar + bounded queue).
    - [ ] Stdlib API: `channel.new()`, `channel.new(buffer_size)`,
      `channel.send`, `channel.receive`, `channel.close`,
      `channel.closed?`.
    - [ ] Closed channel semantics: send raises, receive drains then
      returns nil.
    - [ ] Tests: producer-consumer, long-lived worker, close.
    - [ ] Self-host fixed point preserved.
  - [ ] STEP 6: `channel.receive_timeout` and `channel.select`
    - [ ] Runtime: `channel_receive_timeout` (cond timed wait).
    - [ ] Runtime: `channel_select` selecting one ready operation among
      a list.
    - [ ] Stdlib API: `channel.receive_timeout(c, seconds)`,
      `channel.select([...])`.
    - [ ] Tests: timeout, fairness, send/receive multiplexing.
    - [ ] Self-host fixed point preserved.
  - [ ] STEP 7: `sync` stdlib module
    - [ ] Runtime: mutex (pthread_mutex_t), atomic_integer (C11
      stdatomic), wait_group (counter + condvar).
    - [ ] Stdlib API: `sync.mutex`, `sync.lock`, `sync.unlock`,
      `sync.with_lock`, `sync.atomic_integer`,
      `sync.atomic_integer.add` / `.load` / `.store` /
      `.compare_and_swap`, `sync.wait_group`,
      `sync.wait_group.add` / `.done` / `.wait`.
    - [ ] Tests: shared cache + mutex, atomic counter, worker pool with
      wait_group.
    - [ ] Self-host fixed point preserved.
  - [ ] STEP 8: Documentation, examples, v0.42 SPEC finalization
    - [ ] Write `docs/CONCURRENCY.md` (full spec, alongside
      `docs/CANONICAL_SYNTAX.md`).
    - [ ] Add `docs/v0.42/SPEC.md` (release spec).
    - [ ] Add `examples/concurrent/` with concurrent fetch, Counter
      actor pattern, shared cache, worker pool, producer-consumer,
      timeout patterns.
    - [ ] Run integration tests covering combinations.
    - [ ] Self-host fixed point preserved.

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

- [ ] **Migrate selfhost compiler to latest spec and remove Go reference**
  - [ ] Bring `selfhost/` up from v0.1 surface to the v1.0.0 spec
    (lexer, parser, AST, checker, C emitter, runner).
  - [ ] Implement all language features (including v0.41 GC interface and
    v0.42 Concurrency) in the self-hosted compiler.
  - [ ] Verify stage-2 == stage-3 fixed point at the latest spec.
  - [ ] Distribute pre-built `tya` binaries via Homebrew, curl install
    scripts, and GitHub Releases as the bootstrap source.
  - [ ] Remove `cmd/tya` and `internal/*` Go sources.
  - [ ] Migrate Go-based tests to Tya-based tests where practical;
    keep specification-driven black-box tests in any language that runs
    against the `tya` binary.
  - [ ] This Epic ships at v1.0.0 and is tied to Commitment 6.

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

- [ ] **Extend multi-line strings further** (heredoc-style markers,
  language-tagged interpolation specifiers like `sql"""..."""`) — only if
  demand emerges.

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
