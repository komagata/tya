# Feature: Selfhost V02 Latest Lexer And Parser

## Goal

Update the `selfhost/v02/` lexer, parser, and AST construction layer so it can
parse the latest repository language surface after v0.64, including lexical
closures, iterable protocol syntax use, and the new standard protocol interface
files.

## Context

`selfhost/v02/` already has a completed current-spec proof gate under
`tests/testdata/v02_selfhost/`. That proof predates the latest protocol and
closure work. The draft umbrella at
`docs/prd/drafts/migrate-selfhost-compiler-to-latest-spec.md` is not
implementation-ready by itself; this spec is the first executable follow-up
slice.

The Go implementation remains the language authority. Preserve the existing
hand-written v02 compiler style and dictionary/array AST representation.
`selfhost/v01/` remains the maintained fixed-point invariant.

## Behavior

- `selfhost/v02/compiler.tya` lexes and parses all source forms needed by the
  latest `docs/SPEC.md`, `docs/API.md`, and `docs/STDLIB.md` updates.
- Parser acceptance includes user-defined function literals used as lexical
  closures, `for ... in` iterable forms, interface declarations with predicate
  method names, package-local protocol interfaces, and standard interface files
  such as `stdlib/stringable.tya`.
- Parser failures remain deterministic for invalid syntax.
- Existing v02 parser and fixed-point fixtures continue to pass.

## Scope

- `selfhost/v02/compiler.tya`
- `selfhost/v02/ast.tya`
- parser/front-end fixtures under `tests/testdata/v02_selfhost/`
- focused harness updates in `tests/*v02*_test.go` only if needed

Implement as small parser-family checkpoints:

1. lexical-closure syntax and nested function literal forms
2. iterable `for ... in` parser coverage and sequence-style method chains
3. interface predicate method names such as `equal?`, `lt?`, and `between?`
4. package-local and root stdlib protocol interface source forms

## Out of Scope

- Checker correctness for the newly parsed forms.
- C emission for the newly parsed forms.
- Removing Go sources or making v02 the default compiler.
- Replacing or deleting `selfhost/v01/`.
- Reworking v02 AST representation into a generated schema.

## Acceptance Criteria

- v02 parser accepts valid latest-spec fixtures covering lexical closures,
  iterable syntax, and protocol interface declarations.
- Invalid parser fixtures fail deterministically.
- `TestSelfhostV02Scripts` still proves the existing v02 fixed point.
- `TestSelfhostV01Scripts` still passes.
- No Go implementation files are removed.
- Changes are staged as small front-end checkpoints.

## Verification

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestSelfhostV02Scripts -count=1
go test ./tests -run TestV02Scripts -count=1
go test ./... -count=1 -timeout=20m
```

## Dependencies

None. This is the first implementation spec in the latest-spec follow-up
sequence.
