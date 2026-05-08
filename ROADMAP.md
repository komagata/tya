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
and run representative programs through the maintained v0.4 surface.

## Current Direction

Tya v0.4 is implemented as a small compile-to-C language. The frozen release
documents are:

1. [`docs/v0.1.0/SPEC.md`](docs/v0.1.0/SPEC.md)
1. [`docs/v0.1.0/API.md`](docs/v0.1.0/API.md)
1. [`docs/v0.2.0/SPEC.md`](docs/v0.2.0/SPEC.md)
1. [`docs/v0.2.0/API.md`](docs/v0.2.0/API.md)

Tya uses semantic versioning. Specification changes happen at the minor version
level, such as `v0.3` and `v0.4`. Patch releases such as `v0.3.1` must not
change language or standard-library semantics. In other words, the `x` in
`0.0.x` is never a specification-change unit. Therefore, specification
documents use minor-version labels such as `v0.3`.

Latest editable documentation is:

1. [`docs/SPEC.md`](docs/SPEC.md)
1. [`docs/API.md`](docs/API.md)
1. [`docs/STDLIB.md`](docs/STDLIB.md)
1. [`docs/NAMING.md`](docs/NAMING.md)

Current planned minor-version documents are:

1. [`docs/v0.3/SPEC.md`](docs/v0.3/SPEC.md)
1. [`docs/v0.3/STDLIB.md`](docs/v0.3/STDLIB.md)
1. [`docs/v0.4/SPEC.md`](docs/v0.4/SPEC.md)
1. [`docs/v0.5/SPEC.md`](docs/v0.5/SPEC.md)

The v0.4 reference implementation remains:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
C runtime
v0.4 specification tests
```

Go interpreter behavior, ASTMODE, and legacy archived node-string experiments
are not v0.4 authority. The maintained `selfhost/v01/compiler.tya` fixed point
is still required not to regress.

## Implementation Tooling Policy

The v0.4 compiler implementation should stay hand-written:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
```

Do not add a parser generator or large grammar framework for v0.4. In
particular, avoid introducing Participle, goyacc, Pigeon, ANTLR, or Tree-sitter
as compiler-front-end authority. They may be useful references or future editor
tooling, but the active compiler path should remain explicit Go code.

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

Use small test-support dependencies where they make the v0.4 specification
easier to verify:

```text
github.com/google/go-cmp/cmp
github.com/rogpeppe/go-internal/testscript
```

Use `go-cmp` for readable token, AST, diagnostic, and generated-output diffs.
Use `testscript` for CLI-level specification tests, especially `tya run`,
`tya build`, expected stdout/stderr, and negative examples.

## Current Roadmap

- [x] Ship v0.3 standard attached libraries
  - [x] Define v0.3 attached library scope
    - [x] Decide that JSON and CSV parsers are deferred from v0.3.
    - [x] Keep JSON and CSV out of builtins and out of initial stdlib scope.
    - [x] Specify that v0.3 adds attached libraries, not a package manager.
    - [x] Document v0.3 scope in `docs/SPEC.md` and `docs/STDLIB.md`.
  - [x] Add stdlib import search
    - [x] Add a `stdlib/` directory for shipped `.tya` modules.
    - [x] Search stdlib after the importing file's directory and `TYA_PATH`.
    - [x] Keep user modules and `TYA_PATH` entries higher priority than stdlib.
    - [x] Keep module file name and `module` declaration matching rules.
    - [x] Add tests for same-directory, `TYA_PATH`, and stdlib precedence.
  - [x] Package stdlib with installed Tya
    - [x] Make installed `tya` find `share/tya/stdlib` outside the source checkout.
    - [x] Install `stdlib/*` from the Homebrew Formula.
    - [x] Add an installed-layout test for runtime plus stdlib lookup.
  - [x] Add initial lightweight stdlib modules
    - [x] Add `stdlib/string.tya`.
    - [x] Add `string.blank(text)`.
    - [x] Add `string.present(text)`.
    - [x] Add `stdlib/array.tya`.
    - [x] Add `array.empty(items)`.
    - [x] Add `array.first(items)`.
    - [x] Add tests and examples for every initial stdlib function.
  - [x] Keep v0.3 documentation and release snapshots aligned
    - [x] Update latest `docs/SPEC.md` and `docs/STDLIB.md` when v0.3 behavior is implemented.
    - [x] Keep `docs/v0.3/` aligned with the v0.3 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Create a patch-tag snapshot only when an exact release archive needs one.
    - [x] Update README install, run, development, and documentation sections for v0.3.
- [x] Ship v0.4 testing and script confidence
  - [x] Decide that v0.4 focuses on tests instead of expanding stdlib.
  - [x] Keep native-backed stdlib, JSON, and CSV out of v0.4.
  - [x] Document v0.4 direction in `docs/v0.4/SPEC.md`.
  - [x] Add `tya test`.
    - [x] With no argument, discover `*_test.tya` under the current directory.
    - [x] With a directory argument, discover `*_test.tya` under that directory.
    - [x] With a file argument, run that file only.
    - [x] Exit non-zero when any test file fails.
  - [x] Add assertions.
    - [x] Add `assert value`.
    - [x] Add `assert_equal expected, actual`.
    - [x] Use deep equality for `assert_equal`.
    - [x] Emit source-oriented assertion diagnostics.
  - [x] Add stdlib tests as first-class examples.
    - [x] Add `tests/stdlib_string_test.tya`.
    - [x] Add `tests/stdlib_array_test.tya`.
    - [x] Ensure stdlib tests run through `tya test`.
  - [x] Keep v0.4 documentation and release snapshots aligned.
    - [x] Update latest docs when v0.4 behavior is implemented.
    - [x] Keep `docs/v0.4/` aligned with the v0.4 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Create a patch-tag snapshot only when an exact release archive needs one.
    - [x] Update README install, run, development, and documentation sections for v0.4.
- [ ] Ship v0.5 minimal classes and objects
  - [x] Define v0.5 class scope
    - [x] Add `docs/v0.5/SPEC.md`.
    - [x] Specify `class Name`, constructor calls, `init`, `@field` instance fields, methods, and module class access.
    - [x] Reserve `@@field` for future class variables and keep it invalid in v0.5.
    - [x] Exclude inheritance, `super`, interfaces, class methods, class fields, and visibility from v0.5.
    - [x] Keep dictionary member access with `dict.key` out of v0.5.
  - [ ] Add class syntax to the compiler front end
    - [ ] Add lexer/parser support for class declarations.
    - [ ] Add AST nodes for class declarations, methods, object field access, and object field assignment.
    - [ ] Add checker diagnostics for class naming, duplicate methods, invalid `@field`, invalid `@@field`, and invalid dot access.
  - [ ] Add class runtime and C emission
    - [ ] Emit object construction through `ClassName(args...)`.
    - [ ] Call `init` during construction and ignore its explicit return value.
    - [ ] Support public instance field read/write through dot access.
    - [ ] Support instance method calls with `object.method(args...)`.
  - [ ] Keep modules and self-host compatible
    - [ ] Expose classes declared inside modules as PascalCase module members.
    - [ ] Preserve existing module member access behavior.
    - [ ] Preserve the `selfhost/v01/compiler.tya` fixed point.
  - [ ] Keep v0.5 documentation and tests aligned
    - [ ] Update latest docs when v0.5 behavior is implemented.
    - [ ] Keep `docs/v0.5/` aligned with the v0.5 minor specification.
    - [ ] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [ ] Add compiler, runtime, module, and negative tests for v0.5 classes.
    - [ ] Update README examples when v0.5 is implemented.

## Verification Reference

Default verification:

```sh
go test ./... -count=1
```

Focused verification should prefer tests for the touched lexer, parser, checker,
C emitter, runtime, examples, stdlib, or docs. The self-host fixed-point gate is
part of the maintained project invariant and must stay green.
