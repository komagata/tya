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
- [ ] Ship v0.25 bit-level operations and byte sequences
  - [x] Define v0.25 scope
    - [x] Add `docs/v0.25/SPEC.md`.
    - [x] Specify bitwise operators `&`, `|`, `^`, `~`, `<<`, `>>` on integers.
    - [x] Specify the `bytes` value type with `b"..."` literal, `\xHH` escapes, indexing returning int, slicing, concat, len.
    - [x] Specify `file.read_bytes` and `file.write_bytes`.
    - [x] Specify bytes-aware updates to `digest`, `secure_random`, `hex`, and `base64` (keep string input compatibility).
    - [x] Document the `hex.decode` / `base64.decode` return-type breaking change (now bytes).
    - [x] Keep arbitrary-precision integers, fixed-width integer types, mutable byte buffers, character-set conversion, and streaming IO out of v0.25.
  - [ ] Add bitwise operators
    - [ ] Lex `&`, `|`, `^`, `~`, `<<`, `>>` tokens (avoid conflict with existing operators).
    - [ ] Add precedence levels to the parser.
    - [ ] Reject non-integer operands with structured errors.
    - [ ] Emit C bitwise operators in codegen on `(long)x.number`.
    - [ ] Add eval support for the new operators.
  - [ ] Add the `bytes` value type
    - [ ] Add `TYA_BYTES` value kind with separate length to the C runtime.
    - [ ] Add `bytes`, `bytes_of`, `bytes_text`, `bytes_array`, `bytes_concat`, `bytes_slice` builtins.
    - [ ] Lex and parse `b"..."` literals with `\xHH` escapes.
    - [ ] Wire indexing, length, equality, concat through eval and codegen.
    - [ ] Update `kind` to return `"bytes"`.
  - [ ] Add binary file I/O
    - [ ] Add `file.read_bytes(path)` and `file.write_bytes(path, b)` builtins.
    - [ ] Wire stdlib `file` module wrappers.
  - [ ] Update existing stdlib for bytes
    - [ ] Make `digest.*` accept either string or bytes.
    - [ ] Change `secure_random.bytes(n)` to return a bytes value.
    - [ ] Make `hex.encode` accept either string or bytes; `hex.decode` returns bytes.
    - [ ] Make `base64.encode` accept either string or bytes; `base64.decode` returns bytes.
  - [ ] Keep v0.25 documentation and tests aligned
    - [ ] Update latest docs when v0.25 behavior is implemented.
    - [ ] Keep `docs/v0.25/` aligned with the v0.25 minor specification.
    - [ ] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [ ] Add unittest-form tests for bitwise operators, the bytes type, binary IO, and the migrated digest/secure_random/hex/base64 modules.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [ ] Ship v0.26 external packages and version resolution
  - [ ] Define v0.26 scope
    - [ ] Add `docs/v0.26/SPEC.md`.
    - [ ] Specify the `tya.toml` manifest (name, version, dependencies, dev-dependencies).
    - [ ] Specify the `tya.lock` lockfile (deterministic resolved versions, source identity, checksums).
    - [ ] Specify version constraint syntax (`^x.y.z`, `~x.y.z`, `>=x.y.z, <a.b.c`, exact).
    - [ ] Specify Bundler-style single-version-per-package resolution with backtracking.
    - [ ] Specify git and path sources; defer central registry to a later version.
    - [ ] Specify import resolution order: same dir → `tya.toml` deps → `TYA_PATH` → bundled stdlib.
    - [ ] Specify the package directory layout (`src/` for public modules).
  - [ ] Implement manifest and lockfile parsing
    - [ ] Parse `tya.toml` via the `toml` standard module.
    - [ ] Validate manifest fields and version strings.
    - [ ] Read and write `tya.lock` deterministically.
  - [ ] Implement version constraint resolver
    - [ ] Implement backtracking dependency resolver picking the highest valid version.
    - [ ] Detect and report unsolvable constraint sets (diamond conflicts) with source-oriented diagnostics.
  - [ ] Implement source fetchers
    - [ ] Add a git fetcher (clone + checkout tag/rev) with caching under `.tya/cache`.
    - [ ] Add a path fetcher (symlink or direct read).
    - [ ] Verify and record checksums in the lockfile.
  - [ ] Wire dependency loading into module resolution
    - [ ] Resolve manifest-declared dependencies before `TYA_PATH` and bundled stdlib.
    - [ ] Honor the lockfile for reproducible loads.
    - [ ] Preserve same-directory precedence.
  - [ ] Add CLI commands
    - [ ] Add `tya install` (resolve and write lockfile, download packages to `.tya/packages/`).
    - [ ] Add `tya update [pkg]` (recompute the lockfile for one or all packages).
    - [ ] Add `tya add <pkg> [constraint]` and `tya remove <pkg>` (edit `tya.toml` + re-resolve).
    - [ ] Add `tya outdated` (report newer versions available).
    - [ ] Report missing or conflicting requirements with source-oriented diagnostics.
  - [ ] Keep v0.26 documentation and tests aligned
    - [ ] Update latest docs when v0.26 behavior is implemented.
    - [ ] Keep `docs/v0.26/` aligned with the v0.26 minor specification.
    - [ ] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [ ] Add CLI, resolver, fetcher, and lockfile tests.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.

## Verification Reference

Default verification:

```sh
go test ./... -count=1
```

Focused verification should prefer tests for the touched lexer, parser, checker,
C emitter, runtime, examples, stdlib, or docs. The self-host fixed-point gate is
part of the maintained project invariant and must stay green.
