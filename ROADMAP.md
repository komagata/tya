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

Tya v1.0.0 is the version at which all six language commitments hold and
are publicly defensible. The homepage's short feature list currently highlights
commitments 1, 3, and 4; the remaining commitments are release-quality gates
for the language and implementation.

1. **Canonical Syntax** — every program has exactly one source representation.
   The formatter is part of the language, not a separate opinion.
   *(Shipped in v0.38 / v0.39.)*
2. **Dynamically typed**, indentation-based, with strict semantics
   (no implicit conversions, no `nil` arithmetic).
3. **Compiles to C** for a small, portable runtime.
4. **All-in-one toolchain** (Gleam-style) — the `tya` binary holds the
   compiler, formatter, language server, test runner, doc generator, and
   package manager.
5. **Structured diagnostics** — every error has a stable code, an
   expected/found block, an actionable hint, and a linked explanation.
6. **Self-hosted** — the Tya compiler is written in Tya itself. A released
   pre-built `tya` binary can rebuild the Tya compiler and prove a stable
   stage-2/stage-3 fixed point without requiring Go on the user's machine.
   Removing the Go reference implementation is the final transition step after
   that bootstrap path is routine and release-quality.

Each commitment maps to one or more Epics below:

- Commitment 1 → landed and documented in current SPEC/homepage.
- Commitment 2 → strict-semantics audit still needs an explicit v1.0.0 gate.
- Commitment 3 → shipped and maintained; cross-compilation and one-binary
  distribution are now the public framing.
- Commitment 4 → maintain and polish the all-in-one toolchain: `tya check`,
  `tya test`, `tya format`, `tya lsp`, `tya doc`, `tya new`, `tya task`,
  `tya lint`, and package tooling.
- Commitment 5 → diagnostics polish and remaining structured-error coverage.
- Commitment 6 → *Migrate selfhost compiler to latest spec and remove
  the required Go dependency* (see Future Work / Self-host).

Other Epics (HTTP expansion, documentation generator extensions, task runner
extensions, editor publication, and ecosystem polish) are valuable but not
strictly required for v1.0.0 unless they become blockers for the commitments
above.

Current v1.0.0 blockers:

1. Make the strict-semantics audit explicit and close any gaps it finds.
2. Finish the self-hosted compiler through the latest spec and prove the
   latest-spec stage-2/stage-3 fixed point.
3. Make the no-Go bootstrap path explicit: a released `tya` binary rebuilds
   the self-hosted compiler, proves the latest-spec fixed point, and runs the
   compiler conformance suite without `go` installed.
4. Decide the exact v1.0.0 relationship between the self-hosted compiler and
   the Go implementation: full Go removal at v1.0.0, or a documented
   transition release where Go remains as a reference/bootstrap recovery path.
5. Complete the remaining structured-diagnostic coverage needed for the
   supported parser/checker/codegen/runtime/tool failures.

## Current Direction

Tya is implemented as a small, hand-written compile-to-C language. The latest
released specification is **v0.65**. Frozen release documents live under
`docs/vX.Y/`; release history is tracked in [`docs/VERSIONS.md`](docs/VERSIONS.md)
and the per-version release notes, not in this roadmap.

Latest editable documentation:

1. [`docs/SPEC.md`](docs/SPEC.md)
1. [`docs/GUIDE.md`](docs/GUIDE.md)

Active implementation authority remains:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
C runtime
specification tests
selfhost/v01 fixed point
selfhost/v02 latest-spec proof gates
```

[`docs/static-typing-discussion.md`](docs/static-typing-discussion.md) records a
static-typing discussion note. It is intentionally not current language
authority, not an accepted direction, not on the roadmap, and not scheduled for
implementation.

Go interpreter behavior, ASTMODE, archived node-string experiments, and
`docs/archive/pre-v0.1/` are not specification authority.

## Implementation Tooling Policy

The compiler implementation should stay hand-written. Do not add a parser
generator or large grammar framework. In particular, avoid introducing
Participle, goyacc, Pigeon, ANTLR, or Tree-sitter as compiler front-end
authority. They may be useful references or editor tooling later, but the active
compiler path should remain explicit Go code.

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

## Near Term

- [ ] **Finish `net/http` v2**
  - [x] HTTP client: `http.Client.get(url)`, `http.Client.post(url, body)`, and
    `http.Client.request(method, url, opts)`.
  - [x] Integrate the generic `template.Template` stdlib renderer for HTTP
    response templates.
  - [x] Server concurrency so slow yielding handlers do not block other ready
    clients.
  - [x] Chunked transfer decoding in the HTTP client.
  - [x] Server middleware and HEAD/PATCH/OPTIONS routing helpers.

- [ ] **Expand HTTP protocol coverage** *(post-v1.0.0 unless it blocks a
  supported release use case)*
  - [x] Cookies.
  - [x] Multipart bodies.
  - [x] Server-side chunked transfer encoding.
  - [x] Keep-alive.
  - [x] HTTPS/TLS.
  - [x] Windows support via WinSock2.
  - [x] Per-request arena to bound the v0.58 string-buffer leak.

## Toolchain

- [ ] **Documentation generator extensions** *(polish; not a v1.0.0 blocker
  unless docs publication requires it)*
  - [x] Stdlib re-exports by following imports.
  - [x] `tya doc --json`.
  - [ ] Reuse the public Tya self-introspection library when it exists.
  - [x] Diagnose orphan doc comments, duplicate definitions, and unparseable
    Markdown bodies.

- [x] **Task runner extensions** *(polish; not a v1.0.0 blocker)*
  - [x] Parallel execution syntax.
  - [x] File-watching mode: `tya task <name> --watch`.
  - [x] Task dependency graphs.
  - [x] Per-task environment variables.

- [x] **Linter extensions**
  - [x] Additional rules: suspicious `for` index patterns, unused function
    parameters, and shadowed bindings.
  - [x] Extend `TYAL0001 --fix` to remove full multi-line bindings without
    leaving orphan indented lines.
  - [x] File-scope opt-out: `# tya-lint-ignore-file: TYAL0001`.
  - [x] `--format=sarif`.
  - [x] Share the LSP unwrap-if code action and CLI autofix rewriter.
  - [x] Each rule has a stable code, title, and doc URL.

## Language

- [x] **Strict semantics audit for v1.0.0**
  - [x] Enumerate the no-implicit-conversion, no-`nil` arithmetic, truthiness,
    comparison, indexing, assignment, argument, and return-value rules that
    define the v1.0.0 strict-semantics contract.
  - [x] Map each rule to SPEC text and at least one active test or testscript
    fixture.
  - [x] Add or fix diagnostics where invalid programs currently fail late,
    fail unclearly, or execute with non-strict behavior.
  - [x] Freeze v1.0.0 syntax exclusions across SPEC, parser/checker
    diagnostics, formatter behavior, and CLI fixtures.
  - [x] Unify v1.0.0 user-facing error handling on structured
    `raise` / `try` / `catch` / `finally`.
  - [ ] Document any intentionally dynamic behavior that remains valid in
    v1.0.0.

- [ ] **Migrate selfhost compiler to the latest spec and remove Go dependency**
  - [x] Prove the `selfhost/v02/` current-spec compiler gate.
    - [x] Migrate the v02 lexer/parser surface to current syntax.
    - [x] Migrate the v02 checker surface for selected current semantic
      families.
    - [x] Migrate the v02 C emitter for selected current runtime families and
      deterministic unsupported-codegen failures.
    - [x] Verify the v02 stage-2 == stage-3 fixed point and document the
      applicable full-spec fixture coverage.
  - [ ] Bring `selfhost/` from the v02 current-spec proof to the v1.0.0 spec:
    lexer, parser, AST, checker, C emitter, and runner.
  - [ ] Define a no-Go bootstrap contract:
    - [ ] A release artifact supplies a trusted stage-1 `tya` binary.
    - [ ] That binary compiles the checked-in self-host compiler source to a
      stage-2 compiler.
    - [ ] The stage-2 compiler recompiles the same source to stage-3.
    - [ ] Stage-2 and stage-3 generated C output, diagnostics, and selected
      runtime behavior are byte-for-byte stable or otherwise explicitly
      normalized.
    - [ ] The verification command fails clearly when `go` is absent but the
      release bootstrap artifact is missing.
  - [x] Add a `scripts/bootstrap_no_go.sh` proof that runs from a release
    `tya` binary plus the checked-in source tree and does not invoke `go`.
  - [ ] Add CI coverage for the no-Go bootstrap proof in an environment where
    `go` is intentionally unavailable or hidden from `PATH`.
  - [ ] Maintain a selfhost coverage manifest mapping latest `docs/SPEC.md`
    features to selfhost lexer/parser/checker/emitter support and fixtures.
  - [ ] Implement all language features in the self-hosted compiler.
  - [ ] Verify stage-2 == stage-3 fixed point at the latest spec.
  - [ ] Distribute pre-built `tya` binaries via Homebrew, curl install scripts,
    and GitHub Releases as the bootstrap source.
  - [ ] Make release packaging prefer the self-hosted compiler artifact once
    the no-Go bootstrap proof is green across supported platforms.
  - [ ] Freeze the Go implementation as a reference/bootstrap recovery path and
    require any remaining Go-only behavior to be tracked as a selfhost parity
    gap.
  - [ ] Remove `cmd/tya` and `internal/*` Go sources only after the no-Go
    bootstrap path is release-quality, documented, and routinely used.
  - [ ] Migrate Go-based tests to Tya-based tests where practical; keep
    specification-driven black-box tests in any language that runs against the
    `tya` binary.

## Stdlib Extensions

## Editor and Ecosystem

- [ ] **Ship syntax coloring for major editors**
  - [x] Define required editor targets: VS Code, Emacs, Vim, and GitHub.
  - [x] Document canonical token taxonomy.
  - [x] House editor assets under `editors/`.
  - [ ] Publish VS Code support to Marketplace and Open VSX.
  - [ ] Publish Emacs mode to MELPA.
  - [x] Add Vim / Neovim syntax, filetype, and indent files.
  - [x] Add Tree-sitter grammar assets and highlight queries for GitHub Linguist.
  - [ ] Register a Tree-sitter grammar with GitHub Linguist.

## Verification Reference

Default verification:

```sh
go test ./... -count=1
```

Focused verification should prefer tests for the touched lexer, parser, checker,
C emitter, runtime, examples, stdlib, or docs. The self-host fixed-point gate is
part of the maintained project invariant and must stay green.
