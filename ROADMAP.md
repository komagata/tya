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
- [ ] Adopt Elm-grade diagnostics across the toolchain
  - [ ] Author the diagnostics philosophy document
    - [ ] Add `docs/DIAGNOSTICS.md` as the single source of truth for how every Tya error, warning, and lint is written.
    - [ ] State the philosophy: every diagnostic must say what was expected, what was found, why it is wrong, and where reasonable suggest a concrete next action — in a kind, non-blaming tone.
    - [ ] Define the required diagnostic shape: stable error code (e.g. `TYA0042`), one-line title, source span with caret, expected-vs-found block, and a "hint" or "did you mean" line when applicable.
    - [ ] Define a doc URL convention so each error code links to a longer explanation page under `docs/errors/`.
    - [ ] Forbid jargon-only messages and stack-trace-only messages in user-facing output; internal panics are a bug.
  - [ ] Build the diagnostic infrastructure
    - [ ] Introduce a shared diagnostic type used by lexer, parser, checker, codegen, runner, and the future `tya lsp`.
    - [ ] Add a registry of error codes with titles and doc URLs, asserted in tests so codes are stable and unique.
    - [ ] Add a `--format=json` option for machine-readable diagnostics so editors, CI, and AI agents can consume them.
  - [ ] Migrate existing diagnostics
    - [ ] Audit every existing error message in lexer, parser, checker, codegen, runner, and stdlib loaders against `docs/DIAGNOSTICS.md`.
    - [ ] Rewrite each to match the required shape and assign a stable error code.
    - [ ] Add positive and negative tests pinning the rewritten messages.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point through the migration.

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

- [ ] Rename `tya fmt` to `tya format` and lock it in as the one canonical formatter
  - [ ] Define the formatter policy
    - [ ] Decide on a target minor version and add `docs/vX.Y/SPEC.md` for the formatter policy.
    - [ ] Rename the subcommand from `tya fmt` to `tya format`. The full word is the only spelling; no short alias is provided. Short subcommand names like `fmt` are rejected on principle.
    - [ ] Add `docs/FORMAT.md` as the single source of truth for canonical Tya formatting, in the spirit of `gofmt`.
    - [ ] State the policy: there is exactly one canonical format. The formatter has no style flags, no config file, no per-project overrides, and no opt-out for individual rules.
    - [ ] Forbid introducing options like line width, quote style, or indent size; the canonical choice is baked into the formatter itself.
    - [ ] Define the only public CLI surface: `tya format [paths...]` (write in place by default), `tya format --check` (exit non-zero on diff), `tya format --stdin` (read from stdin, write to stdout). No other flags.
    - [ ] Specify the canonical rules (indentation, spacing, trailing commas, string quote normalization, import grouping, blank-line rules) as the source of truth in `docs/FORMAT.md`.
  - [ ] Resolve every "two valid spellings" point in current Tya syntax in `docs/FORMAT.md`
    - [ ] Decide the canonical form for dictionary literals: block form (`user =\n  name: ...`) vs inline form (`{ name: ..., age: ... }`), including the length / element-count threshold that switches between them.
    - [ ] Decide the canonical form for array literals: single-line vs multi-line, and the threshold that switches between them.
    - [ ] Decide trailing comma policy for multi-line arrays, dicts, and argument lists (required / forbidden — pick one).
    - [ ] Decide whether the formatter rewrites string concatenation (`"Hello, " + name`) into interpolation (`"Hello, {name}"`) or leaves it alone, and document the decision (rewriting risks changing semantics, but two spellings violate the one-canonical-form policy).
    - [ ] Decide blank-line rules: between top-level definitions, between functions, around `import` blocks, and inside indented blocks.
    - [ ] Decide operator spacing: binary operators, unary operators, `:` in dict key-value pairs, `,` in lists, `->` in function expressions.
    - [ ] Decide function body form: single-line lambda (`f = x -> expr`) vs multi-line block (`f = x ->\n  ...`), and when each is canonical.
    - [ ] Decide `return value` vs `return value, nil` policy in multiple-return functions until a static type system removes the ambiguity.
    - [ ] Confirm string quote normalization: `"..."` is canonical; document why `'...'` is not used outside embedded examples.
    - [ ] Confirm `elseif` is canonical and `else if` is rejected by the formatter (and ideally by the parser).
    - [ ] Decide import ordering and grouping: alphabetical, stdlib-vs-user separation, blank-line separators.
    - [ ] Decide canonical position for `case _` in `match` statements (e.g. last only).
    - [ ] Decide canonical empty-collection forms (`{}` and `[]`) and forbid alternative spellings.
    - [ ] Decide whether an `else` branch with an empty body is rewritten away or kept.
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

- [ ] Ship test coverage support
  - [ ] Define coverage scope
    - [ ] Decide on a target minor version and add `docs/vX.Y/SPEC.md` for the coverage feature.
    - [ ] Specify the granularity: line coverage as the baseline, with branch coverage as an explicit follow-up if needed.
    - [ ] Specify the CLI surface: `tya test --cover` runs tests with instrumentation and writes a coverage profile; `tya cover report` prints a human-readable summary; `tya cover html` writes an HTML report; `tya cover --format=json` emits machine-readable output for CI and AI agents.
    - [ ] Specify the profile file format and location (e.g. `.tya/coverage/profile`).
    - [ ] Keep mutation testing, MC/DC coverage, and cross-process / cross-binary aggregation out of the initial scope.
  - [ ] Implement the coverage pipeline
    - [ ] Instrument generated C with per-statement counters keyed by Tya source span; preserve the `selfhost/v01/compiler.tya` fixed point.
    - [ ] Aggregate counters at process exit into the profile file.
    - [ ] Map profile entries back to Tya source spans for reporting.
    - [ ] Exclude generated code, third-party dependencies under `.tya/packages/`, and the stdlib by default; allow `--include` / `--exclude` overrides.
    - [ ] Diagnose misuse (missing profile, mismatched source) with structured errors.
  - [ ] Keep coverage documentation and tests aligned
    - [ ] Document `tya test --cover`, `tya cover report`, `tya cover html`, and `--format=json` in `docs/`.
    - [ ] Add CLI and end-to-end tests covering profile generation, reporting, JSON output, and include/exclude behavior.
    - [ ] Add a recommended CI snippet (fail when coverage drops below a threshold).
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.

- [ ] Ship `markdown` stdlib module
  - [ ] Define markdown scope
    - [ ] Decide on a target minor version and add `docs/vX.Y/SPEC.md` for the `markdown` module.
    - [ ] Target the CommonMark specification as the baseline dialect.
    - [ ] Specify the GitHub Flavored Markdown extensions to support (tables, task lists, strikethrough, autolinks, fenced code with info string).
    - [ ] Specify the public API: `markdown.parse(text) -> ast`, `markdown.to_html(text) -> string`, `markdown.to_html_ast(ast) -> string`, `markdown.render(ast, visitor) -> string`.
    - [ ] Specify the AST shape (block / inline node kinds, attributes, source spans) and treat it as part of the module's compatibility surface.
    - [ ] Keep math extensions, footnote extensions beyond GFM, custom directive syntax, and a Markdown writer / formatter out of the initial scope.
  - [ ] Implement the module
    - [ ] Implement the parser in pure Tya, no native dependency, so it works under every `tya build` target including WASM.
    - [ ] Implement HTML rendering with safe escaping by default; expose an opt-in for raw HTML pass-through with a clear security note.
    - [ ] Provide a visitor / walk helper for custom rendering (e.g. extracting headings for a table of contents).
    - [ ] Diagnose malformed input gracefully — Markdown is forgiving by spec; surface only structural problems via structured errors.
  - [ ] Keep markdown documentation and tests aligned
    - [ ] Document the API, supported GFM extensions, AST schema, and the security posture for raw HTML in `docs/STDLIB.md`.
    - [ ] Run a representative subset of the CommonMark test suite as part of `go test ./...`.
    - [ ] Add unittest-form tests for each GFM extension and for AST round-trips through the visitor API.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.

## Verification Reference

Default verification:

```sh
go test ./... -count=1
```

Focused verification should prefer tests for the touched lexer, parser, checker,
C emitter, runtime, examples, stdlib, or docs. The self-host fixed-point gate is
part of the maintained project invariant and must stay green.
