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

- [ ] Ship v0.22 filesystem standard library expansion
  - [x] Define v0.22 filesystem stdlib scope
    - [x] Add `docs/v0.22/SPEC.md`.
    - [x] Add `dir.list(path)`, `dir.mkdir(path)`, and `dir.rmdir(path)`.
    - [x] Add `file.remove(path)`, `file.rename(old_path, new_path)`, and `file.stat(path)`.
    - [x] Add `path.expand_user(value)`.
    - [x] Add `os.cwd()` and `os.chdir(path)`.
    - [x] Define permissions API as `file.stat` booleans.
    - [x] Keep time/date, streaming IO, binary IO, async IO, recursive walking, `mkdir_all`, `remove_all`, copy, symlink, chmod/chown, file handles, `$VAR` path expansion, and platform-specific path separators out of v0.22.
  - [ ] Implement `dir` module
    - [ ] Add `dir.list(path)` with stable sorted names.
    - [ ] Add `dir.mkdir(path)` for one-level directory creation.
    - [ ] Add `dir.rmdir(path)` for empty directory removal.
    - [ ] Raise structured errors for native directory failures.
  - [ ] Expand `file` module
    - [ ] Add `file.remove(path)` for files only.
    - [ ] Add `file.rename(old_path, new_path)`.
    - [ ] Add `file.stat(path)` metadata dictionary.
    - [ ] Include `kind`, `size`, `readable`, `writable`, and `executable` in `file.stat`.
    - [ ] Keep time and platform-specific metadata out of `file.stat`.
  - [ ] Expand `path` and `os` modules
    - [ ] Add `path.expand_user(value)`.
    - [ ] Add `os.cwd()`.
    - [ ] Add `os.chdir(path)`.
    - [ ] Raise structured errors for native failures.
  - [ ] Keep v0.22 documentation and tests aligned
    - [ ] Update latest docs when v0.22 behavior is implemented.
    - [ ] Keep `docs/v0.22/` aligned with the v0.22 minor specification.
    - [ ] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [ ] Add compiler, runtime, module, C emission, and negative tests for v0.22 filesystem stdlib APIs.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [ ] Ship v0.23 NestedText standard module
  - [ ] Define v0.23 NestedText scope
    - [ ] Add `docs/v0.23/SPEC.md`.
    - [ ] Add `nestedtext` standard module reading and writing `.nt` files.
    - [ ] Specify `nestedtext.parse(text)` returning nested dicts, arrays, and strings.
    - [ ] Specify `nestedtext.dump(value)` emitting NestedText.
    - [ ] Treat every leaf value as a string with no implicit type coercion.
    - [ ] Support indented block dictionaries, lists, and multi-line strings (`> ` prefix).
    - [ ] Support `#` line comments.
    - [ ] Reject non-string dictionary keys.
    - [ ] Keep schema validation, type inference, anchors, references, and binary content out of v0.23.
  - [ ] Implement the NestedText parser
    - [ ] Add a lexer for indented blocks, list items, key/value lines, and multi-line strings.
    - [ ] Treat every scalar value as a string.
    - [ ] Report syntax errors with source locations.
  - [ ] Implement the NestedText writer
    - [ ] Emit indented block syntax.
    - [ ] Use `> ` multi-line strings when values contain newlines or leading whitespace.
    - [ ] Reject non-string keys and non-supported value kinds.
  - [ ] Keep v0.23 documentation and tests aligned
    - [ ] Update latest docs when v0.23 behavior is implemented.
    - [ ] Keep `docs/v0.23/` aligned with the v0.23 minor specification.
    - [ ] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [ ] Add parser, writer, module, and negative tests for v0.23 NestedText.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [ ] Ship v0.24 package manifest and version resolution
  - [ ] Define v0.24 package manifest scope
    - [ ] Add `docs/v0.24/SPEC.md`.
    - [ ] Decide the package manifest filename (placeholder: `Tyafile`).
    - [ ] Specify the manifest format as NestedText, parsed by the v0.23 `nestedtext` standard module.
    - [ ] Specify the resolved-version lock filename and format (placeholder: `Tyafile.lock`).
    - [ ] Specify package source identity (name plus version constraints).
    - [ ] Specify version operators `~>`, `>=`, `<`, `=`.
    - [ ] Specify Bundler-style single-version-per-source resolution policy.
    - [ ] Specify `tya install` to resolve and write the lock file.
    - [ ] Specify `tya update [package]` to recompute resolution for one or all packages.
    - [ ] Specify import resolution to honor the lock file for declared dependencies.
    - [ ] Keep multi-version coexistence, package alias, `unique` declarations, semver-aware type identity, remote registry install, native dependency build, content-addressed lock checksums, and circular dependency healing out of v0.24.
  - [ ] Add manifest parsing
    - [ ] Parse the manifest via the `nestedtext` standard module.
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
