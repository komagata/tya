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

- [ ] Ship v0.24 scripting toolkit and lightweight numerics
  - [x] Define v0.24 scope
    - [x] Add `docs/v0.24/SPEC.md`.
    - [x] Add `time`, `random`, `process`, `hex`, `digest`, `secure_random`, and `matrix` standard modules.
    - [x] Expand `math` with `sqrt`, `pow`, `floor`, `ceil`, `round`, `trunc`, `log`, `log2`, `log10`, `exp`, `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `atan2`, `pi`, `e`.
    - [x] Keep all native-backed APIs import-only and explicit.
    - [x] Use structured `raise` for native operation failures.
    - [x] Keep byte-array type, streaming digest, HTTP/TCP/UDP/TLS, regex, yaml, xml, markdown, async/threads, subprocess pipes, matrix inverse/eigenvalues, and shell-string parsing out of v0.24.
  - [ ] Implement the `time` module
    - [ ] Add `time.now`, `time.sleep`, `time.format`, `time.parse`, `time.since`.
    - [ ] Use UNIX timestamp seconds (float, sub-second precision) as the time value.
    - [ ] Support `"iso"`, `"date"`, `"time"`, `"unix"` format layouts.
    - [ ] Raise on invalid `time.parse` input or negative `time.sleep` argument.
  - [ ] Implement the `random` module (PRNG, seedable)
    - [ ] Add `random.seed`, `random.int`, `random.float`, `random.choice`, `random.shuffle`.
    - [ ] Use a Mersenne Twister or equivalent PRNG; seedable by int or string.
    - [ ] Raise on empty `random.choice` input or invalid `random.int` range.
  - [ ] Expand the `math` module
    - [ ] Wire libm functions (`sqrt`, `pow`, `floor`, `ceil`, `round`, `trunc`, `log`, `log2`, `log10`, `exp`, trig and inverse trig, `atan2`).
    - [ ] Expose `math.pi` and `math.e` as numeric constants (not functions).
    - [ ] Raise on `sqrt` of negative numbers and on non-positive `log` arguments.
  - [ ] Implement the `process` module
    - [ ] Add `process.run(command, options)` returning `{exit_code, stdout, stderr}`.
    - [ ] Accept array form only (no shell-string).
    - [ ] Support `cwd`, `env`, and `input` options.
    - [ ] Buffer stdout/stderr fully into memory.
    - [ ] Raise only on launch failures; non-zero exit codes are returned in the result.
  - [ ] Implement the `hex` module
    - [ ] Add `hex.encode` (lowercase) and `hex.decode` (case-insensitive).
    - [ ] Raise on odd-length or non-hex input to `hex.decode`.
  - [ ] Implement the `digest` module
    - [ ] Add `md5`, `sha1`, `sha256`, `sha384`, `sha512` returning lowercase hex strings.
    - [ ] Implement digests in C without external deps for portability (target macOS and Linux).
    - [ ] Hash UTF-8 bytes of the input string; do not introduce a byte-array type.
  - [ ] Implement the `secure_random` module
    - [ ] Add `bytes`, `hex`, `base64`, `uuid` (RFC 4122 v4), and `int`.
    - [ ] Source entropy from `getentropy` (macOS/BSD), `getrandom`, or `/dev/urandom` as fallback.
    - [ ] Use rejection sampling in `secure_random.int` to avoid modulo bias.
  - [ ] Implement the `matrix` module (pure Tya)
    - [ ] Represent a matrix as `{rows, cols, data}`.
    - [ ] Add `new`, `zero`, `identity`, `at`, `set`, `add`, `sub`, `scale`, `mul`, `transpose`, `det`, `equal?`.
    - [ ] Implement `det` via cofactor expansion for sizes up to 4x4; raise for larger sizes.
    - [ ] Validate dimensions on construction and per-operation.
  - [ ] Keep v0.24 documentation and tests aligned
    - [ ] Update latest docs when v0.24 behavior is implemented.
    - [ ] Keep `docs/v0.24/` aligned with the v0.24 minor specification.
    - [ ] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [ ] Add unittest-form tests for each new module.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [ ] Ship v0.25 bit-level operations and byte sequences
  - [ ] Add bitwise operators: `&`, `|`, `^`, `~`, `<<`, `>>` for integers.
  - [ ] Add a byte-sequence value type (literal form, indexing returns int, slicing, concat, len).
  - [ ] Add `file.read_bytes(path)` and `file.write_bytes(path, bytes)`.
  - [ ] Extend `digest`, `secure_random`, `hex`, and `base64` to accept and return byte sequences (keep string compatibility).
  - [ ] Document that these features unblock NES-emulator-class workloads.
  - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [ ] Future: package manifest and version resolution (deferred, schedule TBD)
  - [ ] Decide manifest filename (placeholder `Tyafile`) and lockfile format.
  - [ ] Specify version operators and Bundler-style single-version-per-source resolution.
  - [ ] Add `tya install` / `tya update` CLI commands.
  - [ ] Wire manifest dependencies into module resolution before bundled stdlib lookup.
  - [ ] Use the `toml` standard module to parse the manifest.

## Verification Reference

Default verification:

```sh
go test ./... -count=1
```

Focused verification should prefer tests for the touched lexer, parser, checker,
C emitter, runtime, examples, stdlib, or docs. The self-host fixed-point gate is
part of the maintained project invariant and must stay green.
