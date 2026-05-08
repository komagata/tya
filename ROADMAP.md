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
specification is v0.21. Frozen release documents live under `docs/vX.Y.Z/` and
`docs/vX.Y/`; the latest editable specification, API, stdlib, and naming
documents live directly under `docs/`.

Tya uses semantic versioning. Specification changes happen at the minor version
level, such as `v0.21` and `v0.22`. Patch releases such as `v0.21.1` must not
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

- [x] Ship filesystem standard library expansion (deferred from v0.22)
  - [x] Implement `dir` module: `dir.list`, `dir.mkdir`, `dir.rmdir` with structured errors.
  - [x] Expand `file` module: `file.remove`, `file.rename`, `file.stat` (kind/size/readable/writable/executable).
  - [x] Expand `path` and `os` modules: `path.expand_user`, `os.cwd`, `os.chdir`.
  - [x] Keep time/date, streaming IO, binary IO, async IO, recursive walking, `mkdir_all`, `remove_all`, copy, symlink, chmod/chown, file handles, `$VAR` path expansion, and platform-specific path separators out of scope.
  - [x] Add tests, regenerate docs, preserve the selfhost fixed point.
- [x] Ship v0.23 data-format and utility stdlib expansion
  - [x] Define v0.23 scope
    - [x] Add `docs/v0.23/SPEC.md`.
    - [x] Add `toml`, `json`, `csv`, `base64`, and `url` standard modules.
    - [x] Keep all five implementations pure Tya (no new builtins).
    - [x] Use `parse`/`dump` (or `encode`/`decode`) shape across modules.
    - [x] Keep schema validation, streaming, comment-preserving round-trip, formatting-preserving round-trip, TOML 1.1 extensions, JSON5, CSV dialects beyond RFC 4180, URL-safe base64, IDN handling, HTTP, YAML, regex, random, time, and crypto out of v0.23.
  - [x] Implement the `toml` module
    - [x] Lex TOML tokens (keys, strings, numbers, booleans, dates, brackets, equals, dots, newlines, comments).
    - [x] Parse top-level key/value pairs, `[table]` and `[[array.of.tables]]` headers, inline tables, arrays.
    - [x] Map TOML primitive types to Tya values per the SPEC.
    - [x] Implement `toml.dump` emitting tables, inline tables, and arrays of tables.
    - [x] Report syntax errors with line numbers.
    - [x] Reject `nil`, functions, classes, objects, modules, and non-string dict keys on dump.
  - [x] Implement the `json` module
    - [x] Parse RFC 8259 JSON into Tya values (object→dict, array, string, int/float, bool, null→nil).
    - [x] Implement `json.dump(value)` (compact) and `json.dump(value, indent)` (pretty).
    - [x] Reject `nan`/`inf`/`-inf` and non-string dict keys on dump.
    - [x] Report syntax errors with line numbers.
  - [x] Implement the `csv` module
    - [x] Parse RFC 4180 CSV with optional header row producing dicts.
    - [x] Accept a `separator` option (default `,`, `\t` for TSV).
    - [x] Emit `csv.dump` quoting fields containing the separator, quote, CR, or LF.
    - [x] Accept LF or CRLF on parse, emit LF on dump.
    - [x] Report unterminated quoted fields and malformed escapes as structured errors.
  - [x] Implement the `base64` module
    - [x] Implement `base64.encode(text)` using the standard alphabet with `=` padding.
    - [x] Implement `base64.decode(text)` accepting standard alphabet input with or without padding, ignoring whitespace.
    - [x] Treat decoded bytes as UTF-8 and raise on invalid UTF-8.
  - [x] Implement the `url` module
    - [x] Implement `url.encode` and `url.decode` per RFC 3986.
    - [x] Implement `url.encode_query` and `url.decode_query` (decode `+` and `%20` to space).
    - [x] Implement `url.parse` returning scheme/user/password/host/port/path/query/fragment.
    - [x] Implement `url.build` as the inverse.
    - [x] Report malformed percent-escapes and non-UTF-8 byte sequences as structured errors.
  - [x] Keep v0.23 documentation and tests aligned
    - [x] Update latest docs when v0.23 behavior is implemented.
    - [x] Keep `docs/v0.23/` aligned with the v0.23 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Add parser, writer, module, and negative tests for each new module.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [ ] Ship v0.24 package manifest and version resolution
  - [ ] Define v0.24 package manifest scope
    - [ ] Add `docs/v0.24/SPEC.md`.
    - [ ] Decide the package manifest filename (placeholder: `Tyafile`).
    - [ ] Specify the manifest format as TOML, parsed by the v0.23 `toml` standard module.
    - [ ] Specify the resolved-version lock filename and format (placeholder: `Tyafile.lock`).
    - [ ] Specify package source identity (name plus version constraints).
    - [ ] Specify version operators `~>`, `>=`, `<`, `=`.
    - [ ] Specify Bundler-style single-version-per-source resolution policy.
    - [ ] Specify `tya install` to resolve and write the lock file.
    - [ ] Specify `tya update [package]` to recompute resolution for one or all packages.
    - [ ] Specify import resolution to honor the lock file for declared dependencies.
    - [ ] Keep multi-version coexistence, package alias, `unique` declarations, semver-aware type identity, remote registry install, native dependency build, content-addressed lock checksums, and circular dependency healing out of v0.24.
  - [ ] Add manifest parsing
    - [ ] Parse the manifest via the `toml` standard module.
    - [ ] Read package metadata section.
    - [ ] Read dependencies section with version constraints.
    - [ ] Report manifest validation errors with source locations.
  - [ ] Add version constraint resolver
    - [ ] Implement backtracking dependency resolver.
    - [ ] Pick the highest version satisfying all constraints.
    - [ ] Detect and report unsolvable constraint sets (diamond conflicts).
    - [ ] Write resolved versions to the lock file.
  - [ ] Wire dependency loading into module resolution
    - [ ] Resolve manifest-declared dependencies before bundled stdlib lookup.
    - [ ] Honor the lock file for reproducible loads.
    - [ ] Preserve same-directory and `TYA_PATH` precedence.
  - [ ] Add `tya install` and `tya update` CLI commands
    - [ ] Add `tya install` to read the manifest, resolve, and write the lock file.
    - [ ] Add `tya update [package]` to recompute the lock for one or all packages.
    - [ ] Report missing or conflicting requirements with source-oriented diagnostics.
  - [ ] Keep v0.24 documentation and tests aligned
    - [ ] Update latest docs when v0.24 behavior is implemented.
    - [ ] Keep `docs/v0.24/` aligned with the v0.24 minor specification.
    - [ ] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [ ] Add CLI, resolver, lockfile, and negative tests for v0.24.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.

## Verification Reference

Default verification:

```sh
go test ./... -count=1
```

Focused verification should prefer tests for the touched lexer, parser, checker,
C emitter, runtime, examples, stdlib, or docs. The self-host fixed-point gate is
part of the maintained project invariant and must stay green.
