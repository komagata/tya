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

## Current Direction

Tya is implemented as a small compile-to-C language. The latest released
specification is **v0.44**. Frozen release documents live under
`docs/vX.Y/` and `docs/vX.Y.Z/`; the latest editable specification, API,
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
`docs/vX.Y/`. For older releases (v0.24 – v0.42) see
[`docs/VERSIONS.md`](docs/VERSIONS.md).

- **v0.58** — `net/http` stdlib with Sinatra-style HTTP/1.1
  server (`docs/v0.58/SPEC.md`, `docs/v0.58/RELEASE_NOTES.md`).
  New multi-segment module `import net/http`; `stdlib/net/http/
  Server.tya` defines `class Server` with `.get` / `.post` /
  `.put` / `.delete` route registration and `.run(port)` accept
  loop. `:name` path segments capture into `req["params"]`;
  query strings parse into `req["query"]`; header names are
  lowercased on the way in. Handlers return a `{ status, body,
  headers }` dict (string or bytes body). `app.run(0)` lets the
  OS pick a free port and prints `listening on N` to stderr for
  test harness latch-on. Single-threaded sequential connection
  handling — slow handlers block other clients; concurrency is
  deferred to v0.59+. Implementation: new
  `runtime/tya_http_server.{h,c}` (POSIX sockets + handwritten
  HTTP/1.1 parser + dispatcher via existing `tya_call1`),
  `http_server_run` builtin wired through
  `internal/codegen/c.go::callBuiltin` and
  `internal/checker/checker.go::BuiltinNames`, first
  multi-segment stdlib path (resolver side already worked via
  v0.44 `resolvePackageDir`). Per-request string buffers
  intentionally leak — per-request arena queued for v0.59+.
  Windows is a build-time stub. Language surface unchanged from
  v0.57.
- **v0.57** — Asset embedding (`docs/v0.57/SPEC.md`,
  `docs/v0.57/RELEASE_NOTES.md`). New top-level
  `embed "pattern" as name` statement bakes file contents into
  the compiled binary at codegen time. Single-file patterns
  produce a `bytes` value; `*` and `**` glob patterns produce
  a `dict<string, bytes>` keyed by the path relative to the
  source file. Glob keys are normalized to `/`-separated form
  on all hosts and ordered alphabetically. New diagnostics
  `TYA-E0610 embed source not found` and
  `TYA-E0611 embed glob matched zero files` surface at codegen
  through `tya run` / `tya build` / `tya emit-c`. `embed` is now
  a reserved name. Implementation: new
  `internal/codegen/embed.go` (path resolve, glob walk, bytes
  reading), `ast.EmbedStmt`, parser top-level statement,
  checker top-level binding registration, formatter
  round-trip. Foundation for single-binary distribution — HTTP
  server stdlib and SDL / raylib bindings to follow in later
  Epics. Language surface unchanged from v0.56.
- **v0.56** — Diagnostics signature unification + expression-
  level recovery (`docs/v0.56/SPEC.md`,
  `docs/v0.56/RELEASE_NOTES.md`). Public entry points now share
  the `(X, []diag.Diagnostic, error)` shape:
  `parser.Parse` / `ParseWithComments` →
  `(*ast.Program, []Diag, error)`;
  `codegen.EmitC` / `EmitCWithPath` →
  `(string, []Diag, error)`;
  `codegen.EmitCWithCoverage` →
  `(string, *CoverageRegistry, []Diag, error)`;
  `runner.RunFile` → `([]Diag, error)`. `RunnerError` widens
  `Diag` → `Diags []diag.Diagnostic` (single-value `.Diag()`
  helper kept for source compatibility). 65 caller sites
  migrated. New `internal/parser/recovery.go::skipToCommaOrClose`
  helper drives expression-level recovery in `CallExpr` /
  `ArrayLit` / `DictLit` element lists — each bad element
  records its diagnostic and the parser advances past the next
  `,` (or close bracket) so siblings are still parsed.
  `*ParserError` / `*CodegenError` / `*RunnerError` wrappers
  continue to satisfy `errors.As` for callers that prefer
  unwrapping. Language surface unchanged from v0.55.
- **v0.55** — `tya lint` v3 extension (`docs/v0.55/SPEC.md`,
  `docs/v0.55/RELEASE_NOTES.md`). `tya lint` becomes CI-ready:
  per-line `# tya-lint-ignore[: CODE[, CODE...]]` opt-out
  comments (inline or full-line, wildcard or code-scoped);
  `--format=json` machine-readable output with `version` /
  `findings[{path,line,col,code,message,autofixable}]`; new
  `TYAL0002` rule reports dead code after `return` / `raise`
  in every block (function bodies, `if` arms, `while` / `for`
  bodies, `try` / `catch`, `match` cases); and `--fix` gains
  TYAL0003 `if true` / `if false` unwrap-if autofix that drops
  the header line and de-indents the body by two spaces (runs
  before the existing TYAL0001 line-delete so positions stay
  stable). The `lint` subcommand owns its own `--format` flag;
  other subcommands continue to read the global
  `--format=human|json` for diagnostic rendering. Language
  surface unchanged from v0.54.
- **v0.54** — Diagnostics pipeline migration (`docs/v0.54/SPEC.md`,
  `docs/v0.54/RELEASE_NOTES.md`). Parser, Codegen, and Runner
  errors now flow through the same `diag.Diagnostic` channel as
  Lexer and Checker, each carrying a stable `[TYA-EXXXX]` code.
  New code bands: `TYA-E0100-0199` (parser: expected token /
  block / position constraint / reserved name / pattern / expression),
  `TYA-E0601-0606` (codegen: unsupported AST shapes), `TYA-E0840`
  / `E0856` / `E0857` / `E0858` (runner: filename, entry-module
  conflict, import name conflict, undefined variable). Parser
  gains statement-level recovery (`block()` and `program()`) so
  one `tya check` run surfaces every top-level error at once
  via `*parser.ParserError{Diags}`. Undefined-name errors gain
  `did you mean "…"?` hints driven by a new
  `internal/util/strdist.go::Suggest` Levenshtein helper. LSP
  (`internal/lsp/diagnostics.go::DiagnosticsFor`) and CLI
  (`cmd/tya/main.go::printDiagnostic`) both `errors.As`-unwrap
  the new wrapper types and route every diagnostic into the
  shared structured renderer. Language surface unchanged from
  v0.53.
- **v0.53** — `tya lsp` v2 full IDE feature set
  (`docs/v0.53/SPEC.md`, `docs/v0.53/RELEASE_NOTES.md`). The
  Language Server gains: cross-file `textDocument/definition` via
  `import` and `mod.foo`; scope-aware `textDocument/references` and
  `textDocument/rename` (top-level / local / param);
  `textDocument/rangeFormatting` (heuristic A widening to the
  smallest enclosing top-level Stmt); `textDocument/codeAction`
  quick fixes for `TYAL0001` (line-delete) and `TYAL0003`
  (if-unwrap); `textDocument/semanticTokens/full` over a 9-type
  legend; `textDocument/documentSymbol` (hierarchical) and
  `workspace/symbol` (substring filter); incremental document sync
  (`TextDocumentSyncKind.Incremental`); `positionEncoding` advertised
  as `utf-8`. Setup recipes ship under `editors/neovim/`,
  `editors/zed/`, and `editors/emacs/`. New diagnostic codes
  `TYA-E0932` (workspace scan, recoverable) and `TYA-E0933` (rename
  conflict). Language surface unchanged from v0.52.
- **v0.52** — `tya lsp` Language Server MVP (`docs/v0.52/SPEC.md`,
  `docs/v0.52/RELEASE_NOTES.md`). New toolchain subcommand
  `tya lsp [--log <file>]` speaks LSP JSON-RPC 2.0 over stdio.
  Advertised capabilities: `textDocumentSync` (Full), hover,
  definition, completion, and document formatting. Diagnostics
  fire on `didOpen` / `didChange` / `didSave` and reuse
  `checker.CheckAll` + `parser.OrphanComments` + `CollectUnused` +
  `CollectLintFindings`. Hover renders the function signature plus
  the leading `#`-comment block. Definition is same-file only (cross-file
  resolution queued for v0.53+). Completion lists same-file
  top-level bindings + stdlib module names + builtins + keywords.
  VS Code extension scaffold under `editors/vscode/` (manual
  install via `npx vsce package`; Marketplace publication queued
  for v0.53+). New diagnostic codes `TYA-E0930` (startup / argument
  failure) and `TYA-E0931` (server I/O failure). `internal/checker`
  gets a small public accessor `BuiltinNames()`; `internal/doc`'s
  `funcSignature` is exported as `FuncSignature` for hover reuse.
  Language surface unchanged from v0.51.
- **v0.51** — `tya doc` source documentation generator
  (`docs/v0.51/SPEC.md`, `docs/v0.51/RELEASE_NOTES.md`). New
  toolchain subcommand `tya doc [paths...]` walks top-level
  declarations in `.tya` files, picks up the leading `#`-comment
  block as a Markdown body, and prints a plain-text summary to
  stdout. `tya doc --html <out>` writes a multi-page static site
  (`<out>/index.html` + `<out>/items/<kind>_<name>.html` +
  `<out>/style.css`). Extracted kinds: `class`, `module`,
  `interface`, and `function` (= top-level `name = … -> …` whose
  RHS is a function literal). Top-level bindings whose name starts
  with `_` are excluded. Self-contained Markdown subset: headings,
  paragraphs, fenced code, `- ` / `1. ` lists, inline `` `code` ``,
  `[link](url)`, `**bold**`, `*italic*`. New diagnostic codes
  `TYA-E0920` (missing `--html` argument) and `TYA-E0923` (`src/`
  not found with no explicit paths). Language surface unchanged
  from v0.50.
- **v0.50** — Toolchain extension pack (`docs/v0.50/SPEC.md`,
  `docs/v0.50/RELEASE_NOTES.md`). `tya lint` gains rules
  `TYAL0003` (redundant `if true` / `if false`), `TYAL0004` (deep
  nesting, threshold 5), and `TYAL0005` (long function body,
  threshold 50), plus `--fix` for `TYAL0001` unused locals.
  `tya new` gains `--here`, `--template app|lib`, `--force`,
  `--no-git`, and runs `git init` by default. `tya task` accepts
  a new TOML table form `[tasks.<name>] cmds = [...]; parallel =
  true` that runs every entry concurrently and exits with the
  first non-zero exit code after waiting for all children;
  output is line-prefixed `[<index> <truncated cmd>] `. New
  diagnostic codes `TYA-E0903` (parallel failure),
  `TYA-E0912` / `TYA-E0913` (`tya new` flag misuse), `TYAL0003` /
  `TYAL0004` / `TYAL0005`. Language surface unchanged from v0.49.
- **v0.49** — Toolchain track kickoff: three new CLI subcommands
  (`docs/v0.49/SPEC.md`, `docs/v0.49/RELEASE_NOTES.md`). `tya new
  <name>` scaffolds a minimal project (`tya.toml` + `src/main.tya`
  + `.gitignore`) with a sample `[tasks]` entry. `tya task [name]
  [args...]` lists and runs tasks defined under a new `[tasks]`
  table in `tya.toml`; commands run under `/bin/sh -c` from the
  project root, trailing arguments are POSIX-quoted and appended
  to the command (`$@` style), and array-form tasks run each
  entry in order stopping on the first failure. `tya lint
  [paths...]` ships rule `TYAL0001` — unused local — as the first
  member of a planned rule set; findings print
  `path:line:col: TYAL0001 unused local "name"`, exit 1 when any
  finding is reported. New diagnostic code range `TYA-E090x`
  reserved for these subcommands (`E0900` unknown task, `E0901`
  array-form failure, `E0902` no tya.toml, `E0910` invalid project
  name, `E0911` project dir exists). Manifest-discovery helper
  extracted to `pkg.FindManifest` and shared with the existing
  `projectRoot`. Language surface unchanged from v0.48. Build
  driver now links `-lm` on non-Windows hosts and the runtime
  defines `_XOPEN_SOURCE`/`_DEFAULT_SOURCE` so strict glibc
  defaults (Arch Linux) accept the compiled programs.
- **v0.48** — Canonical receiver rule + formatter v0.46 keyword
  surface completion (`docs/v0.48/SPEC.md`,
  `docs/v0.48/RELEASE_NOTES.md`). G1 (strict bare-name receivers)
  is codified — bare identifiers inside class method bodies
  resolve to locals / params / imports only, never to class
  members; this was already enforced from v0.46 as a side effect
  of G2/G4. G6 wires `[TYA-E0413]` as a strict-mode warning when
  `<DeclaringClass>.foo` is written inside the declaring class
  body, and `tya format` rewrites the same shape to the canonical
  `Self.foo`. The formatter also now consistently emits the v0.46
  keyword surface (`private`, `static`, `self.`, `Self.`,
  `initialize`) for every class shape, rewriting any remaining
  legacy sigils on output.
- **v0.47** — Class-member surface clean cut. The legacy v0.45
  syntax (`@`, `@@`, `_`-prefix, `init` / `_init`) is now rejected
  with structured diagnostics (`docs/v0.47/SPEC.md`,
  `docs/v0.47/RELEASE_NOTES.md`). Wired codes: `[TYA-E0407]`
  (`_`-prefix on class members), `[TYA-E0410]` (`@`/`@@` sigils),
  `[TYA-E0411]` (`self` inside `static` methods), `[TYA-E0414]`
  (`init`/`_init` as constructor name). `selfhost/v01/` keeps the
  v0.43 surface via a path-based permissive-legacy mode on the
  checker (`checker.SetPermissiveLegacy` + `runner.IsLegacyV01Path`).
  All of `examples/`, `stdlib/`, and `tests/testdata/v0[6-9]…v45/`
  migrated to the new surface; the `v09/private_members` fixture
  was retired (it pinned the v0.45-only `_`-prefix external-access
  heuristic, which has no v0.46+ equivalent). Deferred to v0.48:
  G1 full strict bare-name receivers (the diagnostics catch the
  common legacy paths but a complete resolver walk is future
  work) and G6 formatter rewrite of `<DeclaringClass>.foo` →
  `Self.foo`.
- **v0.46** — Sigil-free, keyword-based class-member surface
  (`docs/v0.46/SPEC.md`, `docs/v0.46/RELEASE_NOTES.md`). Transitional
  release: the new keywords (`private`, `static`, `self`, `Self`,
  `initialize`) are the canonical form for class members and replace
  the v0.45 sigil-based surface (`@`, `@@`, `_`-prefix, `init`). The
  legacy syntax still parses for backward compatibility; the
  clean-cut removal of legacy syntax plus G4 (strict bare-name
  receivers) is deferred to v0.47. stdlib and examples migrated to
  the new surface; the v01 self-host keeps its v0.43 surface
  unchanged.
- **v0.45** — User-facing completion of the class-oriented
  namespace and entry-file model (`docs/v0.45/SPEC.md`,
  `docs/v0.45/RELEASE_NOTES.md`). Three v0.44 follow-ups land:
  every `examples/*.tya` migrated to the new model (no `module`
  declarations remain under `examples/`); cross-file private class
  enforcement via `[TYA-E0406]` (origin-file metadata propagated
  through the runner and checked at member access, bare-Ident
  class reference, and `extends`); five concurrency-related stdlib
  packages (`runtime`, `time`, `channel`, `sync`, `task`) migrated
  to the v0.44 directory-package + PascalCase class form. The
  remaining v0.44 follow-ups — Tya-written self-host on the v0.44
  surface, the `string`/`array`/`dict` migration that depends on
  it, the `module` keyword removal, and the `docs/SPEC.md`
  promotion — ride together into the v0.4x pre-v1.0.0 Epic. v0.45
  also scaffolds `selfhost/v02/` (CI gate green; lexer + parser
  accept `@@` single token, `abstract`/`final` class modifiers, and
  the `override` method modifier) as the foundation for that Epic.
- **v0.44** — Class-oriented namespace and entry-file model
  (`docs/v0.44/SPEC.md`, `docs/v0.44/MIGRATION.md`,
  `docs/v0.44/RELEASE_NOTES.md`). Directory-as-package import,
  PascalCase class files, lowercase script files, within-package
  bare class references (calls / member / extends / implements /
  super), same-directory sibling auto-visibility, same-segment
  package collision detection, full import path validation,
  cycle detection through directory packages, TYA_PATH search,
  aliased imports, interface-in-package, all read-only CLI
  commands accept class files (check / format / --tokens /
  --emit-c / --check-unused), 19 of 27 stdlib packages migrated
  to the class form, `[TYA-EXXXX]` structured diagnostic codes
  on every new runtime error, strict no-paren-call mode
  (`print x` / `assert x` / `assert_equal a, b` are removed —
  use parentheses), `[TYA-E0307]` outer-assign rejection inside
  lambdas. Held back to v0.45: M5 cross-file private
  enforcement, M6 remaining 8 stdlib packages, M7 examples/*
  migration, M8 self-host v0.44 surface, M9 `module` keyword
  removal, M10 docs/SPEC.md promotion.
- **v0.43** — Concurrency known-gap close-out: cooperative
  cancellation (`task.cancel` / `task.cancelled?` / `task.current`),
  `scope` body raises run cleanup before unwinding,
  `channel.select` multiplex.

## Scheduled

Epics with assigned minor versions, on the path to v1.0.0. Each Epic is
implemented in numbered STEPs. Every STEP must pass `go test ./... -count=1`
and `go test ./tests -run TestSelfhostV01Scripts -count=1` before the next
STEP starts. The STEP also keeps `docs/vX.Y/SPEC.md` consistent with the
implementation up to that STEP.

### v0.4x — Self-host v0.46+ surface + `module` keyword retirement

The remainder of the class-oriented namespace model-completion work,
deferred to the v1.0.0 prep window because it requires landing a
Tya-written compiler on the **v0.46+ keyword surface**
(`selfhost/v02/`) before the `module` keyword can be retired (`v01`
still consumes it). Held-back stdlib packages and docs promotion
ride the same Epic for a coherent cut.

- [ ] **M8 self-host v02 on v0.46+ surface** *(critical path)*
  - [ ] Grow `selfhost/v02/compiler.tya` to resolve directory
    packages and parse class files on the v0.46/v0.47/v0.48
    surface (`private` / `static` / `self.` / `Self.` /
    `initialize` / `extends` / `implements` / `interface` /
    `abstract` / `final` / `override`).
  - [ ] Keep the `v01` fixed point on the v0.43 surface until v02
    reaches parity; both compilers live side by side. The Go
    reference impl already exempts `selfhost/v01/` from v0.47
    diagnostics via `checker.SetPermissiveLegacy`.
  - [ ] Prove stage-2 == stage-3 fixed point for v02 on a v0.48-
    surface program (`TestSelfhostV02Scripts`).
  - [ ] Replace `TestSelfhostV01Scripts` with the v02 gate only
    after parity is proven and at a release boundary.

  Landed scaffolding (M8.0 – M8.2d, in main): v02 directory
  exists as a byte-equivalent copy of v01; `TestSelfhostV02Scripts`
  gate is green; v02 lexer emits `@@` as a single SYMBOL with
  parser-side consistency; v02 parser accepts `abstract`/`final`
  class modifiers and the `override` method modifier (codegen
  treats them as no-ops). Smoke tests cover `extends`/`super`,
  `@@` class members, `abstract`/`final`, and `override`.

- [ ] **M6 remaining stdlib (3 packages)** *(blocked on M8)*
  - [ ] `string`, `array`, `dict` migrate to class form once v02
    resolves directory-package imports — `v01` currently consumes
    them as single-file modules (see `docs/v0.44/SPEC.md`
    §"Self-Host Invariant Constraint").
  - [ ] Update `docs/STDLIB.md` per package as it lands.

- [ ] **M9 `module` keyword removal** *(blocked on M8 + M6 remaining)*
  - [ ] Parser rejects `module` as a reserved-word error
    (`[TYA-E0200]`).
  - [ ] Checker, formatter, and C emitter drop every `module` code
    path.
  - [ ] Delete any remaining `module`-only files; retire `v01`.

- [ ] **M10 docs promotion** *(final)*
  - [ ] Promote `docs/v0.44/SPEC.md` content into `docs/SPEC.md`.
  - [ ] Rewrite `docs/NAMING.md` for the new file-kind rules and
    remove the legacy "Module Rule" section.
  - [ ] Update `docs/STDLIB.md`, `docs/CANONICAL_SYNTAX.md`,
    `docs/GUIDE.md`, `docs/API.md`, `docs/TERMINOLOGY.md`,
    `docs/LIBRARIES.md` to drop module-era language.
  - [x] Document public `toml` as class-style `toml.Toml.parse` /
    `toml.Toml.dump`, distinct from private toolchain TOML parsing.
    - Spec: `docs/prd/completed/toml-stdlib-class-style-docs.md`
  - [ ] Rebuild HTML via `node scripts/build_docs_pages.js`.
  - [ ] Add a Released entry to this file when the Epic ships.

Diagnostic-code wiring:

- [ ] `[TYA-E0200]` — emitted by M9 `module` keyword rejection.

## Future Work

Epics below are committed direction but not yet scheduled to a specific
minor version. Each will be scoped into a `docs/vX.Y/SPEC.md` when picked up.

### Toolchain

- [x] **Migrate remaining stages to the diagnostics pipeline** *(v0.54
  delivered the core. v0.56 finished the API/recovery polish.)*
  - [x] Parser → `TYA-E0100`–`E0299` (E0100-E0180 buckets in use).
  - [x] Codegen → `TYA-E0600`–`E0799` (E0601-E0606 in use).
  - [x] Runner → `TYA-E0800`–`E0899` (E0840 / E0856-E0858 added on top
    of v0.4x's E0850-E0855).
  - [x] Add did-you-mean suggestions for unknown-name diagnostics
    (Levenshtein-based `internal/util.Suggest`).
  - [x] Add multi-error parsing (statement-level recovery in
    `program()` and `block()`).
  - [x] Parser signature change to `Parse() → (*Program, []Diag, error)`
    for symmetry with `lexer.LexWithComments`. *(v0.56)*
  - [x] Expression-level recovery for `CallExpr` / `ArrayLit` /
    `DictLit` element lists. *(v0.56)*
  - [x] Codegen / Runner signature change to also return `[]Diag`
    slices. *(v0.56)*
  - [x] Binary-chain and member-chain expression-level recovery
    (`a + broken + c`, `a.broken.c`).
  - [x] Codegen multi-error mode (collect every `unsupported AST shape`
    instead of bailing on the first).
    - Spec: `docs/prd/completed/diagnostics-polish.md`

- [x] **Ship `tya lsp` Language Server** *(v0.52 MVP + v0.53 full IDE
  feature set delivered. Remaining items below are Marketplace
  publication and post-MVP polish.)*
  - [x] Define LSP scope; ship `tya lsp` as a subcommand of the same
    binary so compiler and language server cannot drift in version.
  - [x] Speak LSP over stdio (JSON-RPC) for VS Code, Zed, Neovim, Emacs.
  - [x] Diagnostics on save / on change (TYA-E* / TYAL*).
  - [x] Formatting (full + range, backed by `formatter.Unparse`).
  - [x] Hover (functions with leading `#` comments).
  - [x] Go-to-definition (same-file + cross-file via `import`).
  - [x] Completion (in-scope names + stdlib module names + builtins + keywords).
  - [x] References + rename (scope-aware: top-level / local / param).
  - [x] Code actions for common diagnostics (TYAL0001 / TYAL0003 quick fixes).
  - [x] Semantic tokens (`textDocument/semanticTokens/full`, 9 token types).
  - [x] Incremental document sync (`TextDocumentSyncKind.Incremental`).
  - [x] Document symbols + workspace symbols.
  - [x] Ship a minimal VS Code extension scaffold at `editors/vscode/`
    (manual install via `npx vsce package`).
  - [x] Setup recipes for Zed / Neovim / Emacs (`editors/<name>/`).
  - [ ] Marketplace publication of the VS Code extension (publisher
    ID, signed VSIX, icon, GH Actions release pipeline). *(v0.54+)*
  - [ ] `prepareRename` (rename preview). *(v0.54+)*
  - [ ] Semantic token modifiers (`readonly`, `deprecated`, …). *(v0.54+)*
  - [ ] Range formatting at AST slice precision (current: heuristic A). *(v0.54+)*
  - [ ] Inlay hints / call hierarchy / selection range / code lens /
    folding range / document link. *(v0.54+)*

- [x] **Ship `tya doc` source documentation generator** *(v0.51 shipped
  the minimal form: `tya doc` text + `tya doc --html <out>` static site
  over `src/`. Remaining work below.)*
  - [x] Define doc comment syntax: contiguous comment lines immediately
    preceding a top-level definition. Body is Markdown rendered by the
    self-contained `internal/doc` renderer.
  - [x] Discover every top-level binding under `src/`.
  - [x] CLI surface: `tya doc` (text), `tya doc --html <out>` (static site).
  - [ ] Stdlib re-exports (follow `import` statements). *(v0.52+)*
  - [ ] `tya doc --serve` (HTTP). *(v0.52+)*
  - [ ] `tya doc --json` (machine-readable). *(v0.52+)*
  - [ ] Reuse the public Tya self-introspection library. *(v0.52+)*
  - [ ] Diagnose orphan doc comments, duplicate definitions, unparseable
    Markdown bodies. *(reserved as TYA-E0921 / TYA-E0922, v0.52+)*

- [ ] **Extend `tya new` project scaffolder** *(v0.49 shipped the
  minimal form: `tya new <name>` → tya.toml + src/main.tya +
  .gitignore. Remaining work below.)*
  - [ ] `tya new --here` initialize current directory.
  - [ ] `--template app|lib` (built-in fixed templates).
  - [ ] `--force` to overwrite existing target.
  - [ ] Initialize git by default; `--no-git` to skip.
  - [ ] Default template includes `tests/` with one passing
    unittest and a minimal `README.md`.

- [ ] **Extend `tya task` project task runner** *(v0.49 shipped the
  minimal form: `[tasks]` table with string + array forms,
  `/bin/sh -c` execution, project-root CWD, POSIX-quoted argument
  passthrough, structured failure diagnostic. Remaining work
  below.)*
  - [ ] Parallel execution syntax (decide between `parallel = [...]`
    table form and a dedicated keyword).
  - [ ] File-watching mode (`tya task <name> --watch`).
  - [ ] Task dependency graphs (depend-on declaration).
  - [ ] Per-task environment variables.

- [x] **Package command runner**
  - [x] `tya tool <command> [args...]` runs package-declared Tya tools from
    locked dependencies or explicit one-shot git/path sources.
    - Spec: `docs/prd/completed/tya-tool-package-command.md`

- [ ] **Extend `tya lint` source linter** *(v0.49 shipped rule
  `TYAL0001` unused local + CLI surface. v0.50 added `--fix`
  line-delete + `TYAL0003 / 0004 / 0005` warnings. v0.55 added
  per-line opt-out, `--format=json`, `TYAL0002` dead code, and
  `TYAL0003` `--fix` unwrap-if autofix. Remaining work below.)*
  - [x] `--fix` autofix mode. *(v0.50: `TYAL0001` line-delete;
    v0.55: `TYAL0003` unwrap-if)*
  - [x] `--format=json` machine-readable output. *(v0.55)*
  - [x] Additional rules: dead code after `return` / `raise`
    *(`TYAL0002`, v0.55)*; redundant `if true` / `if false`
    *(`TYAL0003`, v0.50)*; deeply nested blocks
    *(`TYAL0004`, v0.50)*; very long functions
    *(`TYAL0005`, v0.50)*.
  - [ ] Additional rules: suspicious `for` index patterns,
    unused function parameters, shadowed bindings.
  - [ ] Extend `TYAL0001` `--fix` to also drop the body of
    multi-line bindings (e.g. `name = ->` followed by an
    indented block) so `--fix` does not leave orphan indented
    lines. *(carryover from v0.50; the current line-delete only
    removes the binding's own line)*
  - [x] Per-line opt-out via `# tya-lint-ignore: TYAL0001`. *(v0.55)*
  - [ ] File-scope opt-out (`# tya-lint-ignore-file: TYAL0001`).
  - [ ] `--format=sarif` (SARIF v2.1 standard) for ingest by
    third-party CI tools.
  - [ ] `TYAL0004` / `TYAL0005` AST autofix (deep-nesting
    flattening / long-function split). Currently warning-only
    because the right reshape is a human-judgement call.
  - [ ] Unify the LSP `unwrap-if` code action helper
    (`internal/lsp/code_actions.go`) with the CLI
    `applyUnwrapIf` (`cmd/tya/lint_autofix.go`) so both call
    the same rewriter.
  - [ ] Each rule has a stable code (`TYAL00xx`), title, doc URL.

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
  - [x] Round-trip tests over a representative corpus.

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

- [x] **Ship asset embedding for single-binary distribution** *(v0.57)*
  - [x] `embed "assets/logo.png" as logo` bakes file bytes into the compiled
    binary at build time.
  - [x] Directory / glob form (`embed "static/**" as static`) returns a dict
    keyed by relative path.
  - [x] Single-file form returns `bytes`; glob form returns
    `dict<string, bytes>`. v0.57 always loads as `bytes` — use
    `bytes_text` to recover text. `as bytes` / `as text`
    modifiers and extension-based auto-detection deferred to
    v0.58+.
  - [x] HTTP server stdlib (the natural consumer of glob embeds) *(v0.58)*
  - [ ] SDL / raylib bindings (the natural consumer of game-asset
    embeds) *(v0.59+)*
  - [ ] Build-time asset transforms (minify / gzip / hash). *(v0.59+)*
  - [ ] `tya embed --list` CLI introspection. *(v0.59+)*
  - [ ] `app.static("/assets", embedded_dict)` helper that
    serves a v0.57 glob-embed dict through the v0.58 HTTP
    server. *(v0.59+)*

- [ ] **Ship `net/http` v2: client, templates, concurrency**
  - [ ] Goal: complete the HTTP toolchain that v0.58 started so
    tya can both serve and consume HTTP. v0.58 shipped a
    single-threaded Sinatra-style server; v2 adds the missing
    pieces.
  - [ ] HTTP **client** — `import net/http`, `http.get(url)`,
    `http.post(url, body)`, `http.request(method, url, opts)`.
    `stdlib/net/http/Client.tya` is already reserved.
  - [x] **Template engine** — integrate the generic `template.Template`
    stdlib renderer for HTTP response templates.
  - [ ] **Concurrency** for the server — thread pool or
    `tya_task_new` wiring so a slow handler does not block
    other clients.
  - [ ] **keep-alive**, **chunked transfer encoding**,
    **multipart bodies**, **HTTPS/TLS**, **cookies**,
    **middleware**, **HEAD/PATCH/OPTIONS**.
  - [ ] **Windows** support via WinSock2 (v0.58 is a POSIX-only
    build-time stub).
  - [ ] Per-request arena to bound the v0.58 intentional
    string-buffer leak.

- [ ] **Primitive literals as class-instance sugar**
  - [ ] Follow-up to v0.44 class-oriented namespace. Extend
    "everything is a class" to literal values: `1` and `1.0` are
    both sugar for `Number(...)` (a single numeric class — no
    Integer/Float split, matching the current `TyaValue.number`
    `double` representation in `runtime/tya_runtime.h`),
    `"hello"` for `String("hello")`, `[1, 2]` for `Array(1, 2)`,
    `{a: 1}` for `Dict("a", 1)`, `true` / `false` for
    `Boolean(true)` / `Boolean(false)`, `nil` for the unique
    `Nil` instance.
  - [ ] Method-call syntax on literals is required: `42.to_s()`,
    `"hi".len()`, `true.to_s()`, `[1,2].len()` all dispatch through
    the wrapper class. Lexer must keep `42.0` (float literal) and
    `42.foo` (method call) unambiguous (Ruby rule: a digit
    immediately after `.` means the dot is part of a float literal;
    otherwise it is a method call).
  - [ ] Operators (`+`, `-`, `*`, `/`, `<`, `==`, `[]`, ...) are
    **not user-redefinable**. They desugar to fixed method names
    on the wrapper class (e.g. `a + b` → `a.__add__(b)`,
    `a == b` → `a.__eq__(b)`). User classes may **define** these
    methods to participate, but cannot **override** them on the
    built-in primitive classes.
  - [ ] Monkey-patching primitive classes (`Number`, `String`,
    `Boolean`, `Nil`, `Array`, `Dict`) is forbidden. The method
    table of each built-in wrapper is fixed at compile time so the
    optimizer can keep the fast path always live.
  - [ ] Cross-type equality is a method-level decision, not a
    language-level one. Standard built-in `__eq__` implementations
    are type-strict: `String#__eq__` returns `false` for
    non-`String` arguments, etc. Because numbers are a single
    `Number` class, `1 == 1.0` is naturally `true` (no cross-type
    case to resolve). Users who want lenient comparison write it
    in their own class's `__eq__`.
  - [ ] No automatic type coercion. There is no Integer/Float
    split today, so the question of fixnum→float or float→bignum
    promotion does not arise. If a future Epic introduces a
    distinct integer type, promotion semantics are decided then.
  - [ ] Runtime representation: keep the current `TyaValue`
    tagged-union (`runtime/tya_runtime.h`, `kind` enum + payload
    fields; `TYA_NUMBER` already stores both integer- and
    fractional-valued numbers as `double`). Wrapper classes
    (`Number`, `Boolean`, `Nil`, `String`, `Array`, `Dict`) are
    process-global singleton class objects created once at
    runtime startup; `x.class` returns the appropriate singleton
    based on the value's `kind` with no allocation. No boxing
    of primitives into heap objects.
  - [ ] Hidden-from-user fast path: the C emitter lowers operator
    desugaring on known-primitive operands directly to the
    existing C arithmetic / comparison helpers, bypassing method
    dispatch. Because monkey-patching and operator redefinition
    are forbidden, this fast path is unconditional — there is no
    "redefinition check" of the kind CRuby needs. Method dispatch
    is only used when the static type is unknown or the receiver
    is a user class.
  - [ ] Migrate stdlib and examples once the wrapper classes land.
  - [ ] Land a separate `docs/vX.Y/SPEC.md` when scheduled.

- [ ] **Grow `interface` toward stackable-trait capability**
  - [ ] **Phase 1: default methods on `interface`**
    - [ ] Allow `interface` declarations to provide method bodies as
      default implementations, in the spirit of Java 8 default methods.
    - [ ] An implementing class inherits the default body unless it
      provides its own override.
    - [ ] Multiple interfaces declaring the same default-method name
      and signature is a compile-time conflict; the implementing
      class must resolve it explicitly. No implicit "first wins"
      rule.
    - [ ] No state, no initialization, no `super` chaining at this
      phase. Defaults may call `self.<other_method>()` only.
    - [ ] Extend checker, C emitter, formatter, and diagnostics for
      the new shape. Reserve a diagnostic code range for default-method
      conflicts and missing required methods.
    - [ ] Keep `selfhost/v01/compiler.tya` (or its v0.44 successor)
      compiling itself to a stable stage-2/stage-3 fixed point.
  - [ ] **Phase 2: state and initialization on `interface`**
    - [ ] Allow `interface` to declare instance fields and an
      initialization block that runs as part of the implementing
      class's construction.
    - [ ] Define a single rule for diamond inheritance of state. The
      starting position is "explicit resolution required": if the
      same ancestor interface contributes the same field via two
      paths, the implementing class must resolve it. Do not silently
      duplicate or silently merge.
    - [ ] Define and document the order in which interface init
      blocks run relative to each other and to the class body. The
      order must be deterministic and stable across recompiles.
    - [ ] Define `self.<field>` scoping inside default methods and
      init blocks: which interface's field is being referenced and
      how name collisions are reported.
    - [ ] C emitter: lay out interface-contributed fields in the
      implementing struct without duplication for the diamond-resolved
      cases; document the layout rule.
    - [ ] Self-host implications: either keep the self-host compiler
      written without using interface state, or migrate it to the
      new shape and re-prove stage-2 == stage-3.
  - [ ] **Phase 3 (optional): linearization and stackable trait**
    - [ ] Decide whether `super` inside an `interface` default method
      means "the parent" (simple) or "the next interface in a
      linearization order" (Scala-style stackable). Until this Epic
      reaches Phase 3, `super` keeps its parent-class meaning only.
    - [ ] If adopted: implement C3 linearization in the checker;
      route `super.method(...)` through the linearized order;
      document the rule in `docs/SPEC.md`.
    - [ ] Re-evaluate vtable / dispatch design in the C emitter for
      stackable dispatch.
    - [ ] Re-prove the self-host fixed point on the new surface.
  - [ ] **Naming decision: keep `interface`, or rename to `trait`**
    - [ ] Defer the rename until at least Phase 2 has shipped and
      real Tya code uses interface state. The choice is editorial,
      not technical: if the strengthened `interface` clearly behaves
      like a Scala trait in stdlib usage, consider a rename in a
      single dedicated Epic with a destructive migration plan.
    - [ ] Do not introduce both `interface` and `trait` as separate
      kinds. One concept, one keyword, consistent with the
      "no hesitation" philosophy.

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

- [x] **Allow raw `"` inside `{expr}` interpolation body**
  - [x] Today the lexer reads a string literal as "until the next
    unescaped `"`", so an interpolation `{user["name"]}` is cut at
    the inner `"`. The user has to write `{user[\"name\"]}` or hoist
    the expression into a local. Make the lexer balance `{` / `}`
    while inside an interpolation expression so the body can
    contain quoted sub-expressions verbatim.
  - [ ] Round-trip through the existing interpolation pipeline
    unchanged; only lexer scanning state needs to track depth.
  - [x] Update the `"""..."""` and raw `r"..."` lexer paths the same
    way so the rule is uniform.
  - [x] Add positive lexer + script tests pinning
    `"Hello, {user["name"]}!"` and the matching multi-line and raw
    cases.
  - [x] Reference: most modern interpolation languages already
    allow this (JavaScript template literals, C# `$"..."`, Kotlin,
    Ruby `"#{...}"`, Scala, Dart, Elixir, Swift `\( )`). Python
    aligned in 3.12 via PEP 701; Tya should follow the same model.
    The minority that still forbids nested same-quote is PHP
    `"...{$x['k']}..."` and pre-3.12 Python f-strings — Tya's
    current behavior matches that older / minority position and
    should not stay there.
    - Spec: `docs/prd/completed/raw-quotes-in-interpolation.md`

### Stdlib extensions

- [x] **Generic template library**
  - [x] `template.Template` renders text templates with interpolation,
    conditionals, loops, partials, file rendering, and optional HTML escaping.
    - Spec: `docs/prd/completed/stdlib-template-library.md`

- [x] **CLI helper library**
  - [x] `cli.Cli` parses flags, positional args, defaults, required options,
    `--`, and deterministic usage text.
    - Spec: `docs/prd/completed/stdlib-cli-library.md`

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
