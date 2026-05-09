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
   (Unprecedented among practical text-based languages.)
2. **Dynamically typed**, indentation-based, with strict semantics
   (no implicit conversions, no `nil` arithmetic).
3. **Compiles to C** for a small, portable runtime.
4. **All-in-one toolchain** (Gleam-style) — the `tya` binary holds the
   compiler, formatter, language server, test runner, doc generator, and
   package manager.
5. **Kind diagnostics** (Elm-grade) — every error has a stable code, an
   expected/found block, an actionable hint, and a linked explanation.

Each commitment maps to one or more Future Work Epics below:

- Commitment 1 → *Rename `tya fmt` to `tya format` and lock it in as the
  one canonical formatter*, *Ship multi-line string literal syntax*.
- Commitment 2 → strict-semantics audit (currently implicit; to be made
  an explicit Epic).
- Commitment 3 → already shipped; maintained.
- Commitment 4 → *Ship `tya lsp` Language Server*, *Ship `tya doc` source
  documentation generator*, *Ship `tya new` project scaffolder*, *Ship
  `tya task` project task runner*, *Lock in `tya fmt` …*, plus existing
  `tya check` / `tya test` / package manager.
- Commitment 5 → *Adopt Elm-grade diagnostics across the toolchain*.

Other Future Work Epics (GC, WASM target, syntax coloring, embedding,
self-introspection library, coverage, markdown stdlib, lint, Omakase
Declaration adoption) are valuable but not strictly required for v1.0.0.
They may ship before v1.0.0 if convenient, or in v1.x after v1.0.0.

## Current Direction

Tya is implemented as a small compile-to-C language. The latest released
specification is v0.23. Frozen release documents live under `docs/vX.Y.Z/` and
`docs/vX.Y/`; the latest editable specification, API, stdlib, and naming
documents live directly under `docs/`.

Tya uses semantic versioning. Specification changes happen at the minor version
level, such as `v0.23` and `v0.24`. Patch releases such as `v0.23.1` must not
change language or standard-library semantics.

Latest editable documentation:

1. [`docs/SPEC.md`](docs/SPEC.md)
1. [`docs/API.md`](docs/API.md)
1. [`docs/STDLIB.md`](docs/STDLIB.md)
1. [`docs/NAMING.md`](docs/NAMING.md)

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

## Current Roadmap

- [x] Ship v0.24 scripting toolkit and lightweight numerics
  - [x] Define v0.24 scope
    - [x] Add `docs/v0.24/SPEC.md`.
    - [x] Add `time`, `random`, `process`, `hex`, `digest`, `secure_random`, and `matrix` standard modules.
    - [x] Expand `math` with `sqrt`, `pow`, `floor`, `ceil`, `round`, `trunc`, `log`, `log2`, `log10`, `exp`, `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `atan2`, `pi`, `e`.
    - [x] Keep all native-backed APIs import-only and explicit.
    - [x] Use structured `raise` for native operation failures.
    - [x] Keep byte-array type, streaming digest, HTTP/TCP/UDP/TLS, regex, yaml, xml, markdown, async/threads, subprocess pipes, matrix inverse/eigenvalues, and shell-string parsing out of v0.24.
  - [x] Implement the `time` module
    - [x] Add `time.now`, `time.sleep`, `time.format`, `time.parse`, `time.since`.
    - [x] Use UNIX timestamp seconds (float, sub-second precision) as the time value.
    - [x] Support `"iso"`, `"date"`, `"time"`, `"unix"` format layouts.
    - [x] Raise on invalid `time.parse` input or negative `time.sleep` argument.
  - [x] Implement the `random` module (PRNG, seedable)
    - [x] Add `random.seed`, `random.int`, `random.float`, `random.choice`, `random.shuffle`.
    - [x] Use a Mersenne Twister or equivalent PRNG; seedable by int or string.
    - [x] Raise on empty `random.choice` input or invalid `random.int` range.
  - [x] Expand the `math` module
    - [x] Wire libm functions (`sqrt`, `pow`, `floor`, `ceil`, `round`, `trunc`, `log`, `log2`, `log10`, `exp`, trig and inverse trig, `atan2`).
    - [x] Expose `math.pi` and `math.e` as numeric constants (not functions).
    - [x] Raise on `sqrt` of negative numbers and on non-positive `log` arguments.
  - [x] Implement the `process` module
    - [x] Add `process.run(command, options)` returning `{exit_code, stdout, stderr}`.
    - [x] Accept array form only (no shell-string).
    - [x] Support `cwd`, `env`, and `input` options.
    - [x] Buffer stdout/stderr fully into memory.
    - [x] Raise only on launch failures; non-zero exit codes are returned in the result.
  - [x] Implement the `hex` module
    - [x] Add `hex.encode` (lowercase) and `hex.decode` (case-insensitive).
    - [x] Raise on odd-length or non-hex input to `hex.decode`.
  - [x] Implement the `digest` module
    - [x] Add `md5`, `sha1`, `sha256`, `sha384`, `sha512` returning lowercase hex strings.
    - [x] Implement digests in C without external deps for portability (target macOS and Linux).
    - [x] Hash UTF-8 bytes of the input string; do not introduce a byte-array type.
  - [x] Implement the `secure_random` module
    - [x] Add `bytes`, `hex`, `base64`, `uuid` (RFC 4122 v4), and `int`.
    - [x] Source entropy from `getentropy` (macOS/BSD), `getrandom`, or `/dev/urandom` as fallback.
    - [x] Use rejection sampling in `secure_random.int` to avoid modulo bias.
  - [x] Implement the `matrix` module (pure Tya)
    - [x] Represent a matrix as `{rows, cols, data}`.
    - [x] Add `new`, `zero`, `identity`, `at`, `set`, `add`, `sub`, `scale`, `mul`, `transpose`, `det`, `equal?`.
    - [x] Implement `det` via cofactor expansion for sizes up to 4x4; raise for larger sizes.
    - [x] Validate dimensions on construction and per-operation.
  - [x] Keep v0.24 documentation and tests aligned
    - [x] Update latest docs when v0.24 behavior is implemented.
    - [x] Keep `docs/v0.24/` aligned with the v0.24 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Add unittest-form tests for each new module.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [x] Ship v0.25 bit-level operations and byte sequences
  - [x] Define v0.25 scope
    - [x] Add `docs/v0.25/SPEC.md`.
    - [x] Specify bitwise operators `&`, `|`, `^`, `~`, `<<`, `>>` on integers.
    - [x] Specify the `bytes` value type with `b"..."` literal, `\xHH` escapes, indexing returning int, slicing, concat, len.
    - [x] Specify `file.read_bytes` and `file.write_bytes`.
    - [x] Specify bytes-aware updates to `digest`, `secure_random`, `hex`, and `base64` (keep string input compatibility).
    - [x] Document the `hex.decode` / `base64.decode` return-type breaking change (now bytes).
    - [x] Keep arbitrary-precision integers, fixed-width integer types, mutable byte buffers, character-set conversion, and streaming IO out of v0.25.
  - [x] Add bitwise operators
    - [x] Lex `&`, `|`, `^`, `~`, `<<`, `>>` tokens (avoid conflict with existing operators).
    - [x] Add precedence levels to the parser.
    - [x] Reject non-integer operands with structured errors.
    - [x] Emit C bitwise operators in codegen on `(long)x.number`.
    - [x] Add eval support for the new operators.
  - [x] Add the `bytes` value type
    - [x] Add `TYA_BYTES` value kind with separate length to the C runtime.
    - [x] Add `bytes`, `bytes_of`, `bytes_text`, `bytes_array`, `bytes_concat`, `bytes_slice` builtins.
    - [x] Lex and parse `b"..."` literals with `\xHH` escapes.
    - [x] Wire indexing, length, equality, concat through eval and codegen.
    - [x] Update `kind` to return `"bytes"`.
  - [x] Add binary file I/O
    - [x] Add `file.read_bytes(path)` and `file.write_bytes(path, b)` builtins.
    - [x] Wire stdlib `file` module wrappers.
  - [x] Update existing stdlib for bytes
    - [x] Make `digest.*` accept either string or bytes.
    - [x] Change `secure_random.bytes(n)` to return a bytes value.
    - [x] Make `hex.encode` accept either string or bytes; `hex.decode` returns bytes.
    - [x] Make `base64.encode` accept either string or bytes; `base64.decode` returns bytes.
  - [x] Keep v0.25 documentation and tests aligned
    - [x] Update latest docs when v0.25 behavior is implemented.
    - [x] Keep `docs/v0.25/` aligned with the v0.25 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Add unittest-form tests for bitwise operators, the bytes type, binary IO, and the migrated digest/secure_random/hex/base64 modules.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [x] Ship v0.26 external packages and version resolution
  - [x] Define v0.26 scope
    - [x] Add `docs/v0.26/SPEC.md`.
    - [x] Specify the `tya.toml` manifest (name, version, dependencies, dev-dependencies).
    - [x] Specify the `tya.lock` lockfile (deterministic resolved versions, source identity, checksums).
    - [x] Specify version constraint syntax (`^x.y.z`, `~x.y.z`, `>=x.y.z, <a.b.c`, exact).
    - [x] Specify Bundler-style single-version-per-package resolution with backtracking.
    - [x] Specify git and path sources; defer central registry to a later version.
    - [x] Specify import resolution order: same dir → `tya.toml` deps → `TYA_PATH` → bundled stdlib.
    - [x] Specify the package directory layout (`src/` for public modules).
  - [x] Implement manifest and lockfile parsing
    - [x] Parse `tya.toml` via the `toml` standard module.
    - [x] Validate manifest fields and version strings.
    - [x] Read and write `tya.lock` deterministically.
  - [x] Implement version constraint resolver
    - [x] Implement backtracking dependency resolver picking the highest valid version.
    - [x] Detect and report unsolvable constraint sets (diamond conflicts) with source-oriented diagnostics.
  - [x] Implement source fetchers
    - [x] Add a git fetcher (clone + checkout tag/rev) with caching under `.tya/cache`.
    - [x] Add a path fetcher (symlink or direct read).
    - [x] Verify and record checksums in the lockfile.
  - [x] Wire dependency loading into module resolution
    - [x] Resolve manifest-declared dependencies before `TYA_PATH` and bundled stdlib.
    - [x] Honor the lockfile for reproducible loads.
    - [x] Preserve same-directory precedence.
  - [x] Add CLI commands
    - [x] Add `tya install` (resolve and write lockfile, download packages to `.tya/packages/`).
    - [x] Add `tya update [pkg]` (recompute the lockfile for one or all packages).
    - [x] Add `tya add <pkg> [constraint]` and `tya remove <pkg>` (edit `tya.toml` + re-resolve).
    - [x] Add `tya outdated` (report newer versions available).
    - [x] Report missing or conflicting requirements with source-oriented diagnostics.
  - [x] Keep v0.26 documentation and tests aligned
    - [x] Update latest docs when v0.26 behavior is implemented.
    - [x] Keep `docs/v0.26/` aligned with the v0.26 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Add CLI, resolver, fetcher, and lockfile tests.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [x] Ship v0.27 hexadecimal and binary integer literals
  - [x] Define v0.27 scope
    - [x] Add `docs/v0.27/SPEC.md`.
    - [x] Specify `0xFF` / `0xff` hexadecimal integer literals.
    - [x] Specify `0b1010` binary integer literals.
    - [x] Specify underscore digit-group separators in decimal, hex, and binary literals.
    - [x] Keep octal literals, hex floats, numeric type suffixes, and big-int out of v0.27.
  - [x] Implement hex and binary literals
    - [x] Lex `0x`/`0X` followed by hex digits (and underscores) into an `INT` token.
    - [x] Lex `0b`/`0B` followed by binary digits (and underscores) into an `INT` token.
    - [x] Reject `0x` / `0b` with no digits, and digits outside the base.
    - [x] Allow underscore separators in plain decimal integer and float literals.
    - [x] Make value handling identical to existing decimal integer literals (no AST change required).
  - [x] Keep v0.27 documentation and tests aligned
    - [x] Update latest docs when v0.27 behavior is implemented.
    - [x] Keep `docs/v0.27/` aligned with the v0.27 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Add lexer and end-to-end tests for hex, binary, and underscore literals.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [x] Ship v0.28 strict compile-time checks
  - [x] Define v0.28 scope
    - [x] Add `docs/v0.28/SPEC.md`.
    - [x] Specify shadowing forbidden across nested scopes.
    - [x] Specify unused imports as compile errors.
    - [x] Specify unused function arguments as compile errors with `_` opt-out.
    - [x] Specify unused private top-level definitions (leading `_`) as compile errors.
    - [x] Keep unused-local-variable check opt-in via `--check-unused`.
    - [x] Keep tab/trailing-whitespace lint, main.tya restriction, and naming-convention expansions out of v0.28.
  - [x] Implement shadowing check
    - [x] Track enclosing scope chain in the checker.
    - [x] Reject any new binding that matches a name visible in a strictly enclosing scope.
    - [x] Allow same-scope reassignment.
    - [x] Treat `_` as a non-binding discard; underscore-prefixed names still bind.
  - [x] Implement unused-import check
    - [x] Track module/alias bindings introduced by `import`.
    - [x] Mark them used on any read.
    - [x] Diagnose any unused binding at the importing file's top level.
  - [x] Implement unused-argument check
    - [x] Track parameter names per function.
    - [x] Mark used on any read in the body.
    - [x] Skip names equal to `_` and names starting with `_`.
    - [x] Skip parameters of abstract method declarations.
  - [x] Implement unused-private-top-level check
    - [x] Detect top-level bindings whose names start with `_`.
    - [x] Diagnose if no expression in the file references the name.
  - [x] Migrate the project to satisfy the new checks
    - [x] Audit stdlib, examples, and tests for shadowing / unused imports / unused args / unused private defs and fix or rename to `_`.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
  - [x] Keep v0.28 documentation and tests aligned
    - [x] Update latest docs when v0.28 behavior is implemented.
    - [x] Keep `docs/v0.28/` aligned with the v0.28 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [ ] Add positive and negative tests for each new check.

## Future Work (Unscheduled)

These epics are committed direction but not yet scheduled to a specific minor
version. They will be scoped into a `docs/vX.Y/SPEC.md` when picked up.

- [ ] Ship `tya lsp` Language Server
  - [ ] Define LSP scope
    - [ ] Decide on a target minor version and add `docs/vX.Y/SPEC.md` for the LSP surface.
    - [ ] Ship `tya lsp` as a subcommand of the same `tya` binary so the compiler and language server cannot drift in version.
    - [ ] Speak Language Server Protocol over stdio (JSON-RPC) so VS Code, Zed, Helix, Neovim, and Emacs can all drive it.
  - [ ] Implement core editor features
    - [ ] Diagnostics on save and on change, sourced from the same checker the CLI uses.
    - [ ] Hover with type / inferred shape / doc comment.
    - [ ] Go to definition and find references for top-level bindings, imports, and locals.
    - [ ] Document and workspace symbol search.
    - [ ] Completion for in-scope names, imported module members, and stdlib.
    - [ ] Formatting (full document and range) backed by `tya format`.
    - [ ] Rename for safe symbols.
    - [ ] Code actions for the most common diagnostics (e.g. add missing import, remove unused import, wrap in `try`).
  - [ ] Editor integration
    - [ ] Publish a minimal VS Code extension that just spawns `tya lsp`.
    - [ ] Document Zed / Helix / Neovim / Emacs setup in `docs/`.
- [x] Ship v0.29 diagnostics foundation
  - [x] Define v0.29 scope
    - [x] Add `docs/v0.29/SPEC.md`.
    - [x] Specify the shared `internal/diag` model, human + JSON renderers, color modes, and the `TYA-Xnnnn` code namespace.
    - [x] Migrate the v0.28 checker strict diagnostics to the new pipeline.
    - [x] Make `tya check` collect multiple strict diagnostics in one run.
    - [x] Keep lexer / parser / codegen / runner / fmt errors out of v0.29.
  - [x] Implement diagnostics infrastructure
    - [x] Build `internal/diag` with `Diagnostic`, `Region`, `SourceMap`, human renderer, JSON renderer, color resolution.
    - [x] Wire `--format=human|json` and `--color=auto|always|never` through every CLI subcommand.
    - [x] Honor `NO_COLOR`.
  - [x] Migrate checker strict diagnostics
    - [x] Assign codes `TYA-E0301`–`TYA-E0306`.
    - [x] Emit banner, snippet, hint, and code for each.
    - [x] Document every code in `docs/v0.29/CODES.md`.
  - [x] Verification
    - [x] Add `internal/diag` unit tests.
    - [x] Add `tests/testdata/v29/diagnostics.txtar` golden test.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [x] Ship v0.32 lexer diagnostics + markdown foundation
  - [x] Define v0.32 scope (`docs/v0.32/SPEC.md`).
  - [x] Migrate lexer errors to `diag.Diagnostic` (`TYA-E0001`–`TYA-E0017`).
  - [x] Wire CLI to render lexer diagnostics in human and JSON formats.
  - [x] Ship `stdlib/markdown.tya` with `markdown.to_html(text)` covering ATX headings, paragraphs, thematic breaks, fenced code blocks, blockquotes, single-level lists, emphasis, strong, inline code, links, autolinks.
  - [x] Add `tests/testdata/v32/{lexer_diag,markdown}.txtar` golden tests.
  - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [ ] Migrate remaining stages to the diagnostics pipeline
  - [ ] Parser → `TYA-E0100`–`E0299`.
  - [ ] Codegen → `TYA-E0600`–`E0799`.
  - [ ] Runner → `TYA-E0800`–`E0899`.
  - [ ] Fmt → `TYA-E0900`–`E0999`.
  - [ ] Add did-you-mean suggestions for unknown-name diagnostics.
  - [ ] Add multi-error parsing.

- [ ] Ship WebAssembly compilation target
  - [ ] Define WASM scope
    - [ ] Decide on a target minor version and add `docs/vX.Y/SPEC.md` for the WASM target.
    - [ ] Use Zig (`zig cc --target=wasm32-wasi` and `zig cc --target=wasm32-freestanding`) as the WASM toolchain. Tya emits C as today and Zig compiles it to `.wasm`; do not depend on Emscripten or a separate WASI SDK.
    - [ ] Decide the system interface: WASI for CLI-style programs, plus a browser-friendly subset that omits filesystem and process APIs.
    - [ ] Define which stdlib modules are available per target (e.g. `file` / `process` unavailable in browser, available under WASI).
  - [ ] Implement the WASM build path
    - [ ] Add `tya build --target wasm32-wasi` and `tya build --target wasm32-browser` to the CLI.
    - [ ] Wire `zig cc` into the runner for WASM builds, with clear diagnostics when the `zig` binary is missing or too old.
    - [ ] Gate unavailable stdlib modules per target with structured errors at import time.
    - [ ] Provide a minimal JS glue example for running a Tya-compiled `.wasm` in a browser and in Node.
  - [ ] Keep WASM documentation and tests aligned
    - [ ] Document the WASM target, supported stdlib, and known limitations in `docs/`.
    - [ ] Add CLI and end-to-end tests that build representative examples to `.wasm` and execute them under a WASI runtime.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.

- [ ] Ship `tya task` project task runner
  - [ ] Define task runner scope
    - [ ] Decide on a target minor version and add `docs/vX.Y/SPEC.md` for the task runner.
    - [ ] Define a `[tasks]` table in `tya.toml` as the single source of truth (no separate `Taskfile` / `justfile` / `Makefile`).
    - [ ] Support both string form (`ci = "tya fmt && tya test"`) and array form (sequence of commands) for tasks.
    - [ ] Keep the runner intentionally small: no file-level dependency tracking, no incremental rebuild, no implicit rules. Build-graph concerns stay inside `tya build`.
    - [ ] Keep parallelism, file-watching, and task graphs (task-depends-on-task) out of the initial scope.
  - [ ] Implement the runner
    - [ ] Add `tya task <name>` and `tya task` (list available tasks) subcommands.
    - [ ] Resolve `tya.toml` from the working directory upward, like the existing manifest loader.
    - [ ] Execute commands through the user's shell with the project root as the working directory and the manifest's environment.
    - [ ] Stream stdout / stderr through unchanged and propagate the exit code of the failed step.
    - [ ] Diagnose unknown task names, malformed `[tasks]` entries, and empty command lists with structured errors.
  - [ ] Keep task runner documentation and tests aligned
    - [ ] Document `[tasks]` syntax and `tya task` usage in `docs/`.
    - [ ] Add CLI tests for task discovery, success, failure propagation, and diagnostics.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.

- [ ] Ship asset embedding for single-binary distribution
  - [ ] Define embedding scope
    - [ ] Decide on a target minor version and add `docs/vX.Y/SPEC.md` for the embedding feature.
    - [ ] Specify an `embed` directive (e.g. `embed "assets/logo.png" as logo`) that bakes a file's bytes into the compiled binary at build time.
    - [ ] Specify directory / glob form (e.g. `embed "static/**" as static`) returning a dictionary keyed by relative path.
    - [ ] Specify the value type: text files load as string, binary files as `bytes` (v0.25 type); allow an explicit `as bytes` / `as text` modifier.
    - [ ] Specify path resolution relative to the source file declaring the embed, with structured errors for missing files.
    - [ ] Keep runtime filesystem mounting, compression, and encryption out of scope.
  - [ ] Implement the embedding pipeline
    - [ ] Read embedded files at compile time and emit their bytes as C constants in codegen.
    - [ ] Expose them to the program as ordinary `string` / `bytes` / dict values with no runtime IO.
    - [ ] Record embedded paths in build output for reproducible builds.
    - [ ] Diagnose missing files, unreadable files, and ambiguous globs with structured errors.
  - [ ] Keep embedding documentation and tests aligned
    - [ ] Document the `embed` directive, supported forms, and reproducible-build behavior in `docs/`.
    - [ ] Add end-to-end tests that build a binary embedding text and binary assets and verify the running binary serves them without filesystem access.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.

- [-] Adopt Canonical Syntax (multi-version landing per `docs/CANONICAL_SYNTAX.md`)
  - [x] **v0.33** Step 1: parser accepts `(a, b) -> body` parenthesized multi-parameter lambda (additive).
  - [x] **v0.34** Step 2: lexer captures comments via `LexWithComments`; `ast.Program` gains `HeaderComments`; `parser.ParseWithComments` populates it per §3.3. Existing `Lex` / `Parse` APIs unchanged. (`else if` and single-line trailing commas are already rejected by the existing grammar.)
  - [x] **v0.35** Step 3: per-stmt `Comments map[Stmt]StmtComments` populated by `ParseWithComments` for top-level statements (leading + line-end). Inner-stmt attachment deferred.
  - [x] **v0.36** Step 4: comment attachment recurses into if/while/for/match/try bodies and `FuncLit.Body`.
  - [x] **v0.37** Step 5: `formatter.Unparse(prog)` foundation covering imports, simple assignments, expression statements, returns/raises/break/continue, `if`/`elseif`/`else`, `while`, `for`, single-line + block-bodied lambdas, and the common literal/operator/call/member/index/array/dict expressions. Returns an error for module/class/match/try (deferred).
  - In-progress (committed to main, not yet released):
    - Unparse covers modules / classes / interfaces / match / try; emits header + per-stmt comments; applies §8.4 import sort + §3.5 blank-line rules; §5 wrap rules for call / array / dict block / binary chain leading-operator / if/while parens / lambda body; §6.3 long-string `"""..."""` rewrite; §11 BinaryExpr precedence parens; lexer/parser accept multi-line `(...)` / `[...]` and leading-operator continuation lines; corpus round-trip test pre-flights normalization; `tya fmt --ast` opt-in.
    - `examples/` and `stdlib/` already normalized via `tya fmt --ast -w`.
  - [ ] **Outstanding for v0.38 release**:
    - [ ] Realign `selfhost/v01/compiler.tya` with the canonical formatter. Currently `tya fmt --ast -w` over `selfhost/` breaks the self-host fixed point because the Tya-written v0.1 compiler does not parse the dict block form (§5.3.3), the leading-operator binary chain (§5.3.5), or multi-line `(...)` / `[...]` continuations. Two options: (a) extend `selfhost/v01/compiler.tya`'s Tya-side lexer/parser to accept these forms, then run `tya fmt --ast -w selfhost/`, or (b) make the formatter avoid emitting any §5 wrap forms when called on a v0.1-only input (rejected as context-dependent). Option (a) is the maintained path.
    - [ ] Reject non-canonical forms at parse time: §3.4 forbidden comment positions (block-trailing, file-trailing, brackets-internal, all-comment block bodies); enforce structured diagnostics via `internal/diag`.
    - [ ] Make `tya fmt` / `tya format` use the AST serializer by default (currently behind `--ast`). The text pass remains as the documented fallback for unsupported inputs but graceful-fallback is invisible to the user.
    - [ ] Normalize `tests/testdata/` and `selfhost/` once selfhost realignment lands.
    - [ ] Add `docs/v0.38/SPEC.md` summarizing the landed Canonical Syntax and bump version + cut release.

- [ ] Rename `tya fmt` to `tya format` and lock it in as the one canonical formatter
  - [ ] Define the formatter policy
    - [ ] Decide on a target minor version and add `docs/vX.Y/SPEC.md` for the formatter policy.
    - [ ] Rename the subcommand from `tya fmt` to `tya format`. The full word is the only spelling; no short alias is provided. Short subcommand names like `fmt` are rejected on principle.
    - [ ] Add `docs/FORMAT.md` as the single source of truth for canonical Tya formatting, in the spirit of `gofmt`.
    - [ ] State the policy: there is exactly one canonical format. The formatter has no style flags, no config file, no per-project overrides, and no opt-out for individual rules.
    - [ ] Forbid introducing options like line width, quote style, or indent size; the canonical choice is baked into the formatter itself.
    - [ ] Define the only public CLI surface: `tya format [paths...]` (write in place by default), `tya format --check` (exit non-zero on diff), `tya format --stdin` (read from stdin, write to stdout). No other flags.
    - [ ] Specify the canonical rules (indentation, spacing, trailing commas, string quote normalization, import grouping, blank-line rules) as the source of truth in `docs/FORMAT.md`.
  - [ ] Adopt the **Canonical Syntax** principle in `docs/FORMAT.md`
    - [ ] State the principle: Tya has a Canonical Syntax. Every Tya program has exactly **one** source representation. `unparse(parse(source)) == source` byte-for-byte. The formatter is the canonical serializer, not a beautifier. The single representation is a property of the language specification, not of an optional tool.
    - [ ] Document the language-level invariant: `source ↔ AST` is a bijection (subject to the atomic-token exception below). This is a maintained property, second only to the self-host fixed point.
    - [ ] Document the atomic-token exception: a single atomic token (identifier, numeric literal, string literal after multi-line normalization, import path) that exceeds the column limit is emitted as-is. The formatter never chooses to exceed; the exception only reflects an unbreakable token in user code.
  - [ ] Lock in the canonical comment rules
    - [ ] Allow exactly three comment kinds: **leading comment** (`#` lines immediately before a node, same indent, attached to that node), **line-end comment** (single `#` on the same line as a statement), **file header comment** (`#` lines at file start, separated from the body by exactly one blank line, attached to the file AST node).
    - [ ] Reject every other comment position with a parse error: block-end comments (no following node), file-end comments, comments inside expressions / argument lists / brackets, blocks whose body is comments only.
    - [ ] Define deterministic blank-line rules: exactly one blank line between top-level definitions; exactly one blank line before any in-block statement that has a leading comment block, except when that statement is the first in its block; otherwise no blank lines.
    - [ ] Add positive and negative tests for each comment position rule.
  - [ ] Lock in the canonical long-line wrapping rules
    - [ ] Set the column limit to **80**.
    - [ ] Define the wrap algorithm: render single-line; if the rendered length plus current indent exceeds 80, emit the construct's multi-line form; recurse into nested constructs minimally (only what overflows).
    - [ ] Function calls multi-line form: each argument on its own line, **trailing comma required**, closing paren on its own line at the call's indent.
    - [ ] Array literals multi-line form: each element on its own line, **trailing comma required**, closing bracket on its own line at the literal's indent.
    - [ ] Dict literals: single-line is the inline form `{ k: v, ... }`; multi-line is the block form (key per line, no braces, no commas). The column limit drives the choice; users do not pick.
    - [ ] Function expressions with multiple parameters: introduce `(a, b) -> body` syntax. Single-line stays as today; multi-line wraps each parameter on its own line with required trailing comma.
    - [ ] Binary operator chains: when wrapped, use **leading-operator** style — operators start each continuation line, indented `+2` from the expression's start.
    - [ ] `if` / `while` long conditions: wrap the condition in parentheses (formatter-inserted), each sub-expression on its own line at `+2` indent, closing paren on its own line at the keyword's indent, body indented as usual.
    - [ ] `for x in iterable` and `match value`: wrap the iterable / value with the normal construct rule (e.g. function call wrap); no extra outer parentheses.
    - [ ] Function body `f = x -> long_expr`: when the single-line form exceeds 80, switch to block form `f = x ->\n  ...` and recursively wrap the body.
    - [ ] Continuation indent is exactly `+2` from the parent.
    - [ ] Single-line forms forbid trailing commas; multi-line forms require them.
    - [ ] Imports are atomic: do not wrap; allow the rare 80+ line if a path is unusually long.
    - [ ] String literals are atomic: do not split. If a long single-line `"..."` exceeds 80 and contains a newline-friendly content, the formatter rewrites it to the multi-line `"""..."""` form. Otherwise it is emitted as-is.
    - [ ] Add positive and negative golden tests for each construct's wrap behavior at the boundary (just under, just over 80).
  - [ ] Resolve remaining canonical-form decisions in `docs/FORMAT.md`
    - [ ] Decide whether the formatter rewrites string concatenation (`"Hello, " + name`) into interpolation (`"Hello, {name}"`) or leaves it alone. Default: leave alone (rewriting risks changing semantics).
    - [ ] Decide operator spacing: one space around binary operators, one space after `,` and `:`, one space around `->`. No space inside `(`/`[`/`{`.
    - [ ] Decide `return value` vs `return value, nil` policy in multiple-return functions until a static type system removes the ambiguity.
    - [ ] Confirm string quote normalization: `"..."` is canonical.
    - [ ] Confirm `elseif` is canonical and `else if` is rejected by the parser.
    - [ ] Decide import ordering and grouping: alphabetical, stdlib-vs-user separation, blank-line separators between groups.
    - [ ] Decide canonical position for `case _` in `match` statements (e.g. last only).
    - [ ] Decide canonical empty-collection forms (`{}` and `[]`) and forbid alternative spellings.
    - [ ] Decide whether an `else` branch with an empty body is rewritten away or kept.
    - [ ] Document the project-policy boundary: per-project rules like maximum identifier length belong in `tya lint`, not in the language or the formatter.
  - [ ] Migrate the subcommand name
    - [ ] Replace the `fmt` subcommand registration with `format` in `cmd/tya/`.
    - [ ] Remove `tya fmt` entirely (no compatibility alias). Diagnose `tya fmt` invocations with a structured error pointing to `tya format`.
    - [ ] Update all docs, examples, scripts, README, AGENTS.md, and CLAUDE.md references from `tya fmt` to `tya format`.
  - [ ] Bring the implementation in line with the policy
    - [ ] Audit the existing `internal/formatter/` for any flags, env vars, or hidden configuration and remove them.
    - [ ] Make the formatter idempotent: running it twice must produce identical output.
    - [ ] Make output stable across platforms (line endings, locale-independent).
    - [ ] Wire `tya format --check` into recommended CI usage and document it in `docs/`.
  - [ ] Keep formatter documentation and tests aligned
    - [ ] Add golden-file tests covering each rule in `docs/FORMAT.md`, including idempotency tests.
    - [ ] Add a CLI test that any attempt to pass an unknown flag fails with a structured diagnostic.
    - [ ] Add a CLI test that `tya fmt` is rejected with a diagnostic naming `tya format`.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.

- [ ] Ship `tya new` project scaffolder
  - [ ] Define scaffolder scope
    - [ ] Decide on a target minor version and add `docs/vX.Y/SPEC.md` for `tya new`.
    - [ ] Specify the CLI surface: `tya new <name>` creates a new project directory; `tya new --here` initializes the current directory.
    - [ ] Specify the default template: `tya.toml` manifest, `src/main.tya` with a runnable hello-world, `tests/` with one passing unittest, `.gitignore`, and a minimal `README.md`.
    - [ ] Specify a small, fixed set of templates (e.g. `--template app`, `--template lib`); keep custom / remote templates out of scope.
    - [ ] Specify behavior on conflict: refuse to overwrite existing files unless `--force` is given, with a structured diagnostic listing the conflicts.
    - [ ] Initialize a git repository by default; allow `--no-git` to skip.
  - [ ] Implement the scaffolder
    - [ ] Embed template files into the `tya` binary (reuse the future asset-embedding feature once shipped; until then, embed via Go).
    - [ ] Validate the project name against the same rules `tya.toml` requires for `name`, with a structured diagnostic on rejection.
    - [ ] Render templates with the project name and the current Tya version substituted in.
    - [ ] Print next-step guidance after success (`cd <name> && tya run`).
  - [ ] Keep scaffolder documentation and tests aligned
    - [ ] Document `tya new`, available templates, and conflict / `--force` behavior in `docs/`.
    - [ ] Add CLI tests for each template, for `--here`, for name validation, for conflict diagnostics, and for `--no-git`.
    - [ ] Verify generated projects build and test green via `tya run` and `tya test` in CI.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.

- [ ] Ship a public Tya self-introspection library
  - [ ] Define library scope
    - [ ] Decide on a target minor version and add `docs/vX.Y/SPEC.md` for the self-introspection library.
    - [ ] State the goal: Tya programs (and external tools) can lex, parse, walk, and re-emit Tya source through a stable, documented API. This unlocks codemods, linters, doc extractors, refactoring tools, AI-agent code edits, and the `tya lsp` server.
    - [ ] Specify the surface as Tya stdlib modules, written in Tya on top of the existing self-host components: `compiler.lexer`, `compiler.parser`, `compiler.ast`, `compiler.checker`, `compiler.format`.
    - [ ] Specify the AST as the single canonical representation. Define each node kind, its fields, and its source span.
    - [ ] Keep code generation (C emission), runtime evaluation, and macro / quasi-quote constructs out of the initial scope.
  - [ ] Implement the API
    - [ ] `compiler.lexer.tokenize(source) -> tokens` returning tokens with kind, lexeme, and source span.
    - [ ] `compiler.parser.parse(source) -> ast` and `compiler.parser.parse_tokens(tokens) -> ast`.
    - [ ] AST walking helpers: visitor / walk callback, child enumeration, parent lookup.
    - [ ] AST construction helpers for codemods, with span-preserving and span-synthesizing variants.
    - [ ] `compiler.format.print(ast) -> source` that round-trips through the canonical formatter (shares the implementation with `tya format`).
    - [ ] `compiler.checker.check(ast) -> diagnostics` returning the same structured diagnostics the CLI emits.
    - [ ] Stable diagnostic shape reused across the CLI, LSP, and this library.
  - [ ] Stability and reuse
    - [ ] Treat the AST schema and module surface as part of the language's compatibility promise; breaking changes ride minor versions and are documented.
    - [ ] Reuse the same Tya-implemented lexer / parser / checker that the self-host compiler uses, so the library and the compiler cannot diverge.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point; the library is a re-export of the self-host components, not a parallel reimplementation.
  - [ ] Keep library documentation and tests aligned
    - [ ] Document each module, every node kind, and the AST schema in `docs/`.
    - [ ] Add round-trip tests: `format(parse(source)) == format(parse(format(parse(source))))` over a representative corpus.
    - [ ] Add a small example codemod (e.g. rename an identifier across a file) demonstrated in `docs/`.
    - [ ] Add tests that the library's diagnostics match the CLI's diagnostics byte-for-byte for a fixed corpus.

- [x] Ship v0.30 test coverage foundation
  - [x] Define v0.30 scope
    - [x] Add `docs/v0.30/SPEC.md`.
    - [x] Specify statement coverage projected to line coverage.
    - [x] Specify CLI: `tya test --cover` and `tya cover [--format=json] [--profile FILE]`.
    - [x] Specify the profile file format (`# tya-cover 1`, `F` / `S` / `H` records) and default location (`.tya/coverage/profile`).
    - [x] Defer HTML report, `--include` / `--exclude`, `Tyafile` `coverage:` section, branch / MC/DC, cross-binary aggregation, and `Break`/`Continue` instrumentation.
  - [x] Implement the coverage pipeline
    - [x] Instrument generated C with per-statement counters via `EmitCWithCoverage`.
    - [x] Add `runtime/tya_cover.c` with `tya_cov_init` / `tya_cov_inc` and an atexit fragment writer keyed on `TYA_COVERAGE_FRAGMENT`.
    - [x] Wire `tya test --cover` to build with instrumentation, run, merge fragments with the codegen registry, and write the profile.
    - [x] Implement `internal/cover` profile parser, serializer, merger, summary, text and JSON renderers.
    - [x] Exclude stdlib, `.tya/packages/`, and synthesized test-suite paths by default.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point (non-coverage builds emit identical C).
  - [x] Verification
    - [x] Add `internal/cover` round-trip and merge tests.
    - [x] Add `tests/testdata/v30/coverage.txtar` golden test.
    - [x] `go test ./...` remains green.
- [ ] Extend coverage tooling (post-v0.30.0)
  - [ ] Implement `tya cover html` self-contained report.
  - [ ] Add `--include` / `--exclude` glob filters to `tya cover`.
  - [ ] Read `coverage.include` / `coverage.exclude` from `Tyafile`.
  - [ ] Give `BreakStmt` / `ContinueStmt` source positions and instrument them.
  - [ ] Map coverage to per-import-source lines instead of the synthesized entry path.
  - [ ] Add a recommended CI snippet (fail when coverage drops below a threshold).

- [ ] Extend `markdown` stdlib module (post-v0.32 foundation)
  - [ ] Public AST: `markdown.parse(text) -> ast`, `markdown.to_html_ast(ast) -> string`, `markdown.render(ast, visitor) -> string`, AST node spec.
  - [ ] GFM extensions: tables, task lists, strikethrough, fenced-code info-string class.
  - [ ] Reference link definitions, images, nested lists, setext headings, HTML blocks.
  - [ ] Raw HTML pass-through with security note.
  - [ ] CommonMark conformance subset run as part of `go test ./...`.
  - [ ] `docs/STDLIB.md` markdown section.

- [ ] Ship a garbage collector for the C runtime
  - [ ] Define GC scope
    - [ ] Decide on a target minor version and add `docs/vX.Y/SPEC.md` for the GC.
    - [ ] State the goal: long-running Tya programs should not leak heap memory; short scripts should run with bounded resident set.
    - [ ] Decide the GC strategy: start with a conservative mark-and-sweep collector over the existing C runtime values (string, dict, array, bytes, error, function closure). Defer generational, incremental, and concurrent collectors.
    - [ ] Decide trigger policy: allocation threshold based, with an explicit `runtime.gc()` for tests and benchmarks.
    - [ ] Specify which roots are scanned: the value stack, currently active locals, module-level globals, and finalized-but-not-yet-collected closures.
    - [ ] Keep weak references, finalizers, and tunable GC parameters out of the initial scope.
    - [ ] Preserve current language semantics: GC must be invisible to Tya programs except for memory pressure and timing.
  - [ ] Implement the collector
    - [ ] Replace the current allocation path with a GC-aware allocator that records every heap-allocated value.
    - [ ] Implement marking from the rooted set; sweep frees unreachable values and returns memory to the allocator.
    - [ ] Ensure correctness across all current value kinds (string, dict, array, bytes, error, function, closure environments) and mixed nesting.
    - [ ] Make the collector deterministic enough that the self-host fixed point and existing CLI tests stay green.
    - [ ] Avoid regressing startup time and small-script performance materially; document any trade-offs.
  - [ ] Keep GC documentation and tests aligned
    - [ ] Document the GC strategy, trigger policy, and `runtime.gc()` in `docs/`.
    - [ ] Add stress tests that allocate aggressively in loops and assert bounded resident set.
    - [ ] Add cycle tests (mutually referencing dicts / arrays / closures) and assert they are reclaimed.
    - [ ] Add a focused leak test under `valgrind` or `leaks` for representative programs.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.

- [ ] Ship `tya doc` source documentation generator
  - [ ] Define doc generator scope
    - [ ] Decide on a target minor version and add `docs/vX.Y/SPEC.md` for `tya doc`.
    - [ ] Specify the doc comment syntax: a contiguous run of comment lines immediately preceding a top-level definition (function, constant, module-level binding) is its doc comment. No separate `/** */` form.
    - [ ] Specify the doc comment body as Markdown, rendered with the `markdown` stdlib module once that ships.
    - [ ] Specify the discovered surface: every top-level binding in every `.tya` file under `src/` plus the package's stdlib re-exports.
    - [ ] Specify the CLI surface: `tya doc` prints to stdout in plain text; `tya doc --html <out>` writes a static HTML site; `tya doc --serve` runs a local HTTP server; `tya doc --json` emits a machine-readable model for editors and AI agents.
    - [ ] Keep cross-package linking, search, and versioned doc archives out of the initial scope; design data structures so they can be added later.
  - [ ] Implement the generator
    - [ ] Reuse the public Tya self-introspection library to parse sources and pair doc comments with their definitions.
    - [ ] Build an in-memory doc model: package, modules, definitions, signatures, source spans, doc comment text.
    - [ ] Render plain text, JSON, and a small static HTML site (one page per module, with a package index).
    - [ ] Use `tya format` to render code examples in signatures consistently.
    - [ ] Diagnose orphan doc comments, duplicate definitions, and unparseable Markdown bodies with structured errors.
  - [ ] Keep doc generator documentation and tests aligned
    - [ ] Document doc comment conventions, supported Markdown subset, and the CLI in `docs/`.
    - [ ] Add CLI tests for plain text, JSON, and HTML outputs over a representative package.
    - [ ] Add a golden-file test for the static HTML site so the rendering stays stable.
    - [ ] Generate and publish documentation for the bundled stdlib as a worked example.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.

- [ ] Ship syntax coloring for the major editors
  - [ ] Define editor coverage and shared assets
    - [ ] Decide on a target minor version and add `docs/vX.Y/SPEC.md` for editor syntax coloring.
    - [ ] Define the **major editors** that Tya commits to supporting: **VS Code, Emacs, Vim, GitHub**. Other editors (Zed, Helix, Neovim, JetBrains) are welcome but not required for "major editor" coverage.
    - [ ] Document the canonical token taxonomy (keyword, builtin, type, string, interpolated expression, number, operator, comment, function name, parameter, module name) once, in `docs/`, so each editor grammar maps to the same names.
    - [ ] Keep these grammars as editor-tooling assets only. They are not authority over the language; the hand-written compiler in `internal/parser/` (and its self-hosted successor) remains the sole authority for what Tya accepts. See *Implementation Tooling Policy*.
    - [ ] House all editor assets under a top-level `editors/` directory: `editors/vscode/`, `editors/emacs/`, `editors/vim/`, `editors/tree-sitter-tya/`.
  - [ ] VS Code support
    - [ ] Author a TextMate grammar (`editors/vscode/syntaxes/tya.tmLanguage.json`) covering the canonical token taxonomy.
    - [ ] Build a minimal extension (`editors/vscode/`) that registers the grammar, the `.tya` file association, and a language configuration (comment toggling, bracket pairs, indent rules).
    - [ ] Publish the extension to the VS Code Marketplace and Open VSX.
  - [ ] Emacs support
    - [ ] Author `editors/emacs/tya-mode.el` as a `define-derived-mode` major mode using `font-lock` keywords for the canonical token taxonomy.
    - [ ] Auto-associate `.tya` files via `auto-mode-alist`.
    - [ ] Provide indentation rules consistent with `tya format`.
    - [ ] Publish to MELPA.
  - [ ] Vim support
    - [ ] Author `editors/vim/syntax/tya.vim`, `editors/vim/ftdetect/tya.vim`, and `editors/vim/indent/tya.vim` covering the canonical token taxonomy and `.tya` auto-detection.
    - [ ] Verify the same files work under Neovim as a free side-effect (no separate maintenance burden).
    - [ ] Document installation via plug.vim, packer, lazy.nvim, and Vim 8 native packages.
  - [ ] GitHub support
    - [ ] Author a Tree-sitter grammar (`editors/tree-sitter-tya/`) covering the canonical token taxonomy.
    - [ ] Register Tya with [github-linguist/linguist](https://github.com/github-linguist/linguist): file extension `.tya`, color, sample file, and the Tree-sitter grammar pointer so GitHub can highlight Tya source on the web and in pull requests.
    - [ ] Confirm rendering on a public Tya repository.
  - [ ] Keep editor documentation and tests aligned
    - [ ] Document install instructions for each major editor in `docs/`.
    - [ ] Add a small corpus of representative `.tya` snippets and snapshot-test each grammar against expected token classifications.
    - [ ] Run grammar tests in CI so regressions are caught before publish.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.

- [ ] Ship `tya lint` source linter
  - [ ] Define lint scope
    - [ ] Decide on a target minor version and add `docs/vX.Y/SPEC.md` for `tya lint`.
    - [ ] State the boundary against existing tools: `tya format` enforces canonical formatting (no opinions about logic), `tya check` enforces compile-time correctness, `tya lint` enforces stylistic and semantic best practices that are not formatting and not type errors.
    - [ ] Specify the CLI surface: `tya lint [paths...]`, `tya lint --fix` (apply safe auto-fixes), `tya lint --format=json` (machine-readable output for CI and AI agents).
    - [ ] Define an initial rule set: unused locals, dead code after `return` / `raise`, redundant `if true`/`if false`, `==` / `!=` against `nil` instead of truthiness when intent is ambiguous, shadowed catch bindings, suspicious `for` index patterns, deeply nested blocks, very long functions.
    - [ ] Each rule has a stable code (e.g. `TYAL0001`), a one-line title, and a doc URL under `docs/lints/`, mirroring the diagnostics philosophy.
    - [ ] Keep custom user-defined rules, plugin systems, and lint-config files out of the initial scope. The rule set ships with the compiler and is enabled by default.
    - [ ] Allow per-line opt-out via a single comment form (e.g. `# tya-lint-ignore: TYAL0001`); no project-wide config file.
  - [ ] Implement the linter
    - [ ] Reuse the public Tya self-introspection library to walk the AST and the checker's resolved bindings.
    - [ ] Emit findings through the shared structured diagnostic type used by `tya check` and `tya lsp`.
    - [ ] Implement `--fix` only for rules whose fix is unambiguous and idempotent; everything else is report-only.
    - [ ] Surface lints in `tya lsp` via diagnostics and code actions for fixable rules.
  - [ ] Keep linter documentation and tests aligned
    - [ ] Document each rule in `docs/lints/<code>.md` with a short rationale, a bad example, and a good example.
    - [ ] Add positive and negative tests pinning each rule's diagnostic and (where applicable) its `--fix` output.
    - [ ] Add CLI tests for plain output, `--format=json`, and exit-code behavior.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.

- [x] Ship v0.31 multi-line string literals
  - [x] Define v0.31 scope
    - [x] Add `docs/v0.31/SPEC.md`.
    - [x] Specify triple-quote `"""..."""` (single-line and multi-line forms).
    - [x] Specify newlines inside the body are part of the value.
    - [x] Specify `{expr}` interpolation reuses the existing pipeline.
    - [x] Specify indentation normalization keyed on the closing-`"""` indent.
    - [x] Specify escape rules (`\n`, `\t`, `\\`, `\"`, `\{`, `{{`, `}}`).
    - [x] Defer raw-string prefix, heredoc markers, language-tagged interpolation, bytes form, and the formatter rewrite rule.
  - [x] Implement multi-line strings
    - [x] Detect `"""` in `lexLine` and consume across line boundaries when no closing `"""` is on the opening line.
    - [x] Apply indentation normalization based on the closing `"""` indent.
    - [x] Reuse the existing interpolation pipeline (`{expr}`, `{{`, `}}`) without parser/codegen changes.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
  - [x] Verification
    - [x] Add lexer unit tests (single-line, multi-line normalized, interpolation, unterminated, mixed-indent).
    - [x] Add `tests/testdata/v31/multiline.txtar` end-to-end golden test.
    - [x] `go test ./...` remains green.
- [ ] Extend multi-line strings (post-v0.31.0)
  - [ ] Implement the formatter rewrite rule for long single-line `"..."` literals.
  - [ ] Add a raw-string prefix `r"""..."""` if needed.
  - [ ] Add a bytes equivalent `b"""..."""`.

## Verification Reference

Default verification:

```sh
go test ./... -count=1
```

Focused verification should prefer tests for the touched lexer, parser, checker,
C emitter, runtime, examples, stdlib, or docs. The self-host fixed-point gate is
part of the maintained project invariant and must stay green.
