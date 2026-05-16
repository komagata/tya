# Feature: Selfhost V02 Latest C Emitter

## Goal

Update the `selfhost/v02/` C emitter so latest-spec programs accepted by the
v02 front end and checker emit deterministic C with runtime behavior matching
the Go compiler for the covered closure, iterable, and protocol-interface
fixtures.

## Context

This spec depends on the latest lexer/parser and checker follow-up specs. The
completed v02 proof already emits C for selected current runtime families, but
the latest repository surface includes lexical closure runtime behavior,
runtime-backed iterable sequences, and protocol interface declarations that
erase or lower consistently during C emission.

Generated C must continue to link with the existing runtime files. The Go C
emitter remains the reference for behavior and runtime helper usage.

## Behavior

- v02 emits compilable C for lexical closures, including nested captures needed
  by representative latest-spec fixtures.
- v02 emits or erases iterable/protocol interface declarations consistently with
  the Go compiler.
- v02 emits C for selected `for ... in` iterable usage over arrays, strings, and
  dictionaries where covered by v02 fixtures.
- v02 keeps deterministic unsupported-codegen failures for valid forms not yet
  covered during intermediate checkpoints.
- Stage-2 and stage-3 C output for `selfhost/v02/compiler.tya` remains stable.

## Scope

- `selfhost/v02/compiler.tya`
- `selfhost/v02/ast.tya` only for emitter helper needs
- codegen/runtime fixtures under `tests/testdata/v02_selfhost/`
- v02 test harness updates for compiling and running emitted C

Implement as emitter-family checkpoints:

1. lexical closure environment/capture emission
2. interface/protocol declaration erasure and method dispatch preservation
3. iterable `for ... in` lowering for covered primitive values
4. selected sequence/protocol runtime helper calls if needed by fixtures
5. deterministic unsupported-codegen failures for remaining valid forms

## Out of Scope

- Rewriting the runtime architecture.
- Removing Go sources or making v02 the default compiler.
- Matching Go diagnostic text byte-for-byte.
- Replacing `selfhost/v01/`.
- Final full-suite v02 orchestration.

## Acceptance Criteria

- v02 emits compilable C for latest-spec valid fixtures selected from Go
  black-box coverage.
- Compiled v02 output matches expected stdout/stderr and exit behavior for
  those fixtures.
- Unsupported or invalid emission paths fail deterministically.
- v02 fixed point remains stage-2 == stage-3.
- v01 fixed point remains green.
- No Go implementation files are removed.
- Changes are staged as small emitter-family checkpoints.

## Verification

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestSelfhostV02Scripts -count=1
go test ./tests -run TestV02Scripts -count=1
go test ./... -count=1 -timeout=20m
```

## Dependencies

None. The lexer/parser and checker follow-up specs are already completed.
