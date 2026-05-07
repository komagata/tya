# Tya Roadmap

`ROADMAP.md` is the single source of truth for current TODO, TASK, and roadmap
planning.

Pre-v0.1 planning documents and self-host migration notes are archived under
[`docs/archive/pre-v0.1/`](docs/archive/pre-v0.1/). They are historical
references, not current language or implementation authority.

## Current Direction

Tya v0.2.0 is implemented as a small compile-to-C language. The frozen release
documents are:

1. [`docs/v0.1.0/SPEC.md`](docs/v0.1.0/SPEC.md)
1. [`docs/v0.1.0/API.md`](docs/v0.1.0/API.md)
1. [`docs/v0.2.0/SPEC.md`](docs/v0.2.0/SPEC.md)
1. [`docs/v0.2.0/API.md`](docs/v0.2.0/API.md)

Latest editable documentation is:

1. [`docs/SPEC.md`](docs/SPEC.md)
1. [`docs/API.md`](docs/API.md)
1. [`docs/NAMING.md`](docs/NAMING.md)

The v0.3 reference implementation remains:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
C runtime
v0.3 specification tests
```

Go interpreter behavior, current `selfhost/*`, ASTMODE, legacy node strings,
and self-host bootstrap gates are not v0.3 authority.

## Implementation Tooling Policy

The v0.3 compiler implementation should stay hand-written:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
```

Do not add a parser generator or large grammar framework for v0.3. In
particular, avoid introducing Participle, goyacc, Pigeon, ANTLR, or Tree-sitter
as compiler-front-end authority. They may be useful references or future editor
tooling, but the active compiler path should remain explicit Go code.

Use small test-support dependencies where they make the v0.3 specification
easier to verify:

```text
github.com/google/go-cmp/cmp
github.com/rogpeppe/go-internal/testscript
```

Use `go-cmp` for readable token, AST, diagnostic, and generated-output diffs.
Use `testscript` for CLI-level specification tests, especially `tya run`,
`tya build`, expected stdout/stderr, and negative examples.

## Current Roadmap

- [ ] Ship v0.3 standard attached libraries
  - [ ] Define v0.3 attached library scope
    - [x] Decide that JSON and CSV parsers are deferred from v0.3.
    - [x] Keep JSON and CSV out of builtins and out of initial stdlib scope.
    - [x] Specify that v0.3 adds attached libraries, not a package manager.
    - [x] Document v0.3 scope in `docs/SPEC.md` and `docs/STDLIB.md`.
  - [ ] Add stdlib import search
    - [ ] Add a `stdlib/` directory for shipped `.tya` modules.
    - [ ] Search stdlib after the importing file's directory and `TYA_PATH`.
    - [ ] Keep user modules and `TYA_PATH` entries higher priority than stdlib.
    - [ ] Keep module file name and `module` declaration matching rules.
    - [ ] Add tests for same-directory, `TYA_PATH`, and stdlib precedence.
  - [ ] Package stdlib with installed Tya
    - [ ] Make installed `tya` find `share/tya/stdlib` outside the source checkout.
    - [ ] Install `stdlib/*` from the Homebrew Formula.
    - [ ] Add an installed-layout test for runtime plus stdlib lookup.
  - [ ] Add initial lightweight stdlib modules
    - [ ] Add `stdlib/string.tya`.
    - [ ] Add `string.blank(text)`.
    - [ ] Add `string.present(text)`.
    - [ ] Add `stdlib/array.tya`.
    - [ ] Add `array.empty(items)`.
    - [ ] Add `array.first(items)`.
    - [ ] Add tests and examples for every initial stdlib function.
  - [ ] Keep v0.3 documentation and release snapshots aligned
    - [ ] Update latest `docs/SPEC.md` and `docs/STDLIB.md` when v0.3 behavior is implemented.
    - [ ] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [ ] Create `docs/v0.3.0/` spec, API, and stdlib snapshots before release.
    - [ ] Update README install, run, development, and documentation sections for v0.3.0.
- [ ] Ship v0.4 testing and script confidence
  - [x] Decide that v0.4 focuses on tests instead of expanding stdlib.
  - [x] Keep native-backed stdlib, JSON, and CSV out of v0.4.
  - [x] Document v0.4 direction in `docs/v0.4.md`.
  - [ ] Add `tya test`.
    - [ ] With no argument, discover `*_test.tya` under the current directory.
    - [ ] With a directory argument, discover `*_test.tya` under that directory.
    - [ ] With a file argument, run that file only.
    - [ ] Exit non-zero when any test file fails.
  - [ ] Add assertions.
    - [ ] Add `assert value`.
    - [ ] Add `assert_equal expected, actual`.
    - [ ] Use deep equality for `assert_equal`.
    - [ ] Emit source-oriented assertion diagnostics.
  - [ ] Add stdlib tests as first-class examples.
    - [ ] Add `tests/stdlib_string_test.tya`.
    - [ ] Add `tests/stdlib_array_test.tya`.
    - [ ] Ensure stdlib tests run through `tya test`.
  - [ ] Keep v0.4 documentation and release snapshots aligned.
    - [ ] Update latest docs when v0.4 behavior is implemented.
    - [ ] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [ ] Create `docs/v0.4.0/` snapshots before release.
    - [ ] Update README install, run, development, and documentation sections for v0.4.0.

## Verification Reference

Default verification:

```sh
go test ./... -count=1
```

Focused verification should prefer tests for the touched lexer, parser, checker,
C emitter, runtime, examples, stdlib, or docs. Self-host bootstrap checks are
historical pre-v0.1 gates and are not default v0.3 verification.
