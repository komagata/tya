# Tya Roadmap

`ROADMAP.md` is the single source of truth for current TODO, TASK, and roadmap
planning.

Pre-v0.1 planning documents and self-host migration notes are archived under
[`docs/archive/pre-v0.1/`](docs/archive/pre-v0.1/). They are historical
references, not current language or implementation authority.

## Current Direction

Tya v0.1 is released as a small compile-to-C language. The frozen v0.1 release
documents are:

1. [`docs/v0.1.0/SPEC.md`](docs/v0.1.0/SPEC.md)
1. [`docs/v0.1.0/API.md`](docs/v0.1.0/API.md)

The current active draft is v0.2:

1. [`docs/v0.2/SPEC.md`](docs/v0.2/SPEC.md)

Latest editable documentation is:

1. [`docs/SPEC.md`](docs/SPEC.md)
1. [`docs/API.md`](docs/API.md)
1. [`docs/NAMING.md`](docs/NAMING.md)

The v0.2 reference implementation remains:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
C runtime
v0.2 specification tests
```

Go interpreter behavior, current `selfhost/*`, ASTMODE, legacy node strings,
and self-host bootstrap gates are not v0.1 authority.

## Implementation Tooling Policy

The v0.2 compiler implementation should stay hand-written:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
```

Do not add a parser generator or large grammar framework for v0.2. In
particular, avoid introducing Participle, goyacc, Pigeon, ANTLR, or Tree-sitter
as compiler-front-end authority. They may be useful references or future editor
tooling, but the active compiler path should remain explicit Go code.

Use small test-support dependencies where they make the v0.2 specification
easier to verify:

```text
github.com/google/go-cmp/cmp
github.com/rogpeppe/go-internal/testscript
```

Use `go-cmp` for readable token, AST, diagnostic, and generated-output diffs.
Use `testscript` for CLI-level specification tests, especially `tya run`,
`tya build`, expected stdout/stderr, and negative examples.

## Current Roadmap

- [ ] Ship v0.2 friendly scripting
  - [ ] Add v0.2 standard builtins
    - [ ] Implement collection builtins: `map`, `filter`, `find`, `any`, `all`, and `reduce`.
    - [ ] Implement deep equality builtin: `equal(left, right)`.
    - [ ] Implement input builtin: `read_line()`.
    - [ ] Add API documentation and examples for every v0.2 builtin.
    - [ ] Add parser/checker/C-emitter/runtime tests for v0.2 builtins.
  - [ ] Add v0.2 user-facing CLI commands
    - [ ] Add `tya check file.tya`.
    - [ ] Add `tya fmt file.tya`.
    - [ ] Add `tya fmt -w file.tya`.
    - [ ] Add `tya emit-c file.tya`.
    - [ ] Replace or retire hidden `--emit-c` documentation in favor of `tya emit-c`.
    - [ ] Add CLI tests for success, diagnostics, and exit statuses.
  - [ ] Improve diagnostics
    - [ ] Add source-oriented diagnostic formatting with file, line, column, message, source line, and caret marker when available.
    - [ ] Keep user-facing diagnostics free of Go implementation terms.
    - [ ] Add golden tests for parser, checker, and CLI diagnostic output.
  - [ ] Add formatter
    - [ ] Define conservative formatting behavior for indentation, trailing whitespace, one statement per line, dictionaries, and inline literals.
    - [ ] Implement formatter output for `tya fmt`.
    - [ ] Implement in-place write behavior for `tya fmt -w`.
    - [ ] Add formatter idempotence tests.
  - [ ] Improve module ergonomics
    - [ ] Add `TYA_PATH` search after the importing file's directory.
    - [ ] Keep module file name and `module` declaration matching rules.
    - [ ] Preserve the v0.2 exclusion of import aliases and package manager behavior.
    - [ ] Add module loading tests for same-directory imports, `TYA_PATH`, and missing modules.
  - [ ] Keep v0.2 documentation and release snapshots aligned
    - [ ] Update latest `docs/SPEC.md` and `docs/API.md` when v0.2 behavior is implemented.
    - [ ] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [ ] Create `docs/v0.2.0/` spec and API snapshots before release.
    - [ ] Update README install, run, development, and documentation sections for v0.2.0.

## Verification Reference

Default verification:

```sh
go test ./... -count=1
```

Focused verification should prefer tests for the touched lexer, parser, checker,
C emitter, runtime, examples, or docs. Self-host bootstrap checks are historical
pre-v0.1 gates and are not default v0.1 verification.
