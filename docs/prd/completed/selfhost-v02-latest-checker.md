# Feature: Selfhost V02 Latest Checker

## Goal

Update the `selfhost/v02/` checker so latest-spec programs parsed by v02 receive
semantic treatment consistent with the Go checker for lexical closures,
iterable protocol use, and standard protocol interfaces.

## Context

This spec depends on `selfhost-v02-latest-lexer-parser.md`. The completed v02
current-spec proof covers selected current semantic families, but the latest
repository work added closure checks, iterable protocol behavior, and standard
interfaces such as `Comparable`, `Equatable`, `Readable`, `Serializable`, and
`Stringable`.

The Go checker remains the authority. v02 diagnostics may be simpler than Go
diagnostics, but they must be deterministic and actionable.

## Behavior

- v02 checker accepts valid lexical-closure programs that capture enclosing
  bindings across nested function literals.
- v02 checker rejects unsupported captured-binding indexed/member mutation in
  the same behavioral cases covered by the Go checker.
- v02 checker accepts valid `for ... in` iterable usage and selected protocol
  interface declarations.
- v02 checker validates interface requirements with predicate method names such
  as `equal?`, `lt?`, `lte?`, `gt?`, `gte?`, and `between?`.
- v02 checker accepts stdlib protocol source files that are part of the latest
  repository surface.
- Existing v01 and v02 fixed-point gates remain green.

## Scope

- `selfhost/v02/compiler.tya`
- `selfhost/v02/ast.tya` only when checker traversal helpers are required
- checker-focused fixtures under `tests/testdata/v02_selfhost/`
- v02 harness updates if needed for checker-only valid/invalid cases

Implement as semantic-family checkpoints:

1. lexical closure capture analysis and invalid captured mutation
2. iterable `for ... in` checker behavior
3. interface predicate method validation
4. stdlib protocol interface acceptance
5. deterministic invalid-fixture diagnostics

## Out of Scope

- C emitter implementation for newly accepted valid forms.
- Full v02 black-box fixture orchestration.
- Removing Go sources or making v02 the default compiler.
- Matching every Go diagnostic byte-for-byte.
- Replacing `selfhost/v01/`.

## Acceptance Criteria

- v02 checker accepts latest-spec valid semantic fixtures selected from the Go
  black-box fixture families.
- v02 checker rejects latest-spec invalid semantic fixtures deterministically.
- v02 checker handles lexical closures and protocol interfaces without
  legacy-only exemptions for `selfhost/v02/` itself.
- Existing v02 fixed-point scripts still pass.
- Existing v01 fixed-point scripts still pass.
- No Go implementation files are removed.
- Changes are staged as small semantic-family checkpoints.

## Verification

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestSelfhostV02Scripts -count=1
go test ./tests -run TestV02Scripts -count=1
go test ./... -count=1 -timeout=20m
```

## Dependencies

- `docs/prd/selfhost-v02-latest-lexer-parser.md`
