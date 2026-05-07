# Tya Roadmap

`ROADMAP.md` is the single source of truth for current TODO, TASK, and roadmap
planning.

Pre-v0.1 planning documents and self-host migration notes are archived under
[`docs/archive/pre-v0.1/`](docs/archive/pre-v0.1/). They are historical
references, not current language or implementation authority.

## Current Direction

Tya v0.1 is frozen as a small compile-to-C language. The authoritative
specification is:

1. [`docs/SPEC.md`](docs/SPEC.md)
1. [`docs/API.md`](docs/API.md)
1. [`docs/NAMING.md`](docs/NAMING.md)

The v0.1 reference implementation is:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
C runtime
v0.1 specification tests
```

Go interpreter behavior, current `selfhost/*`, ASTMODE, legacy node strings,
and self-host bootstrap gates are not v0.1 authority.

## Implementation Tooling Policy

The v0.1 compiler implementation should stay hand-written:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
```

Do not add a parser generator or large grammar framework for v0.1. In
particular, avoid introducing Participle, goyacc, Pigeon, ANTLR, or Tree-sitter
as compiler-front-end authority. They may be useful references or future editor
tooling, but the v0.1 compiler path should remain explicit Go code.

Use small test-support dependencies where they make the v0.1 specification
easier to verify:

```text
github.com/google/go-cmp/cmp
github.com/rogpeppe/go-internal/testscript
```

Use `go-cmp` for readable token, AST, diagnostic, and generated-output diffs.
Use `testscript` for CLI-level specification tests, especially `tya run`,
`tya build`, expected stdout/stderr, and negative examples.

## Current Roadmap

There are no active roadmap items right now.

Add new work here as unchecked Markdown nested task lists. Remove completed
tasks instead of keeping `[x]` history in this file.

## Verification Reference

Default verification:

```sh
go test ./... -count=1
```

Focused verification should prefer tests for the touched lexer, parser, checker,
C emitter, runtime, examples, or docs. Self-host bootstrap checks are historical
pre-v0.1 gates and are not default v0.1 verification.
