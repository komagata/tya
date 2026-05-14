---
status: completed
goal_ready: false
---

# Feature: Selfhost V02 Lexer And Parser Current Spec

## Goal

Migrate the `selfhost/v02/` lexer, parser, and AST construction layer so it can
parse the current repository language surface accepted by the Go compiler,
without changing checker or C emitter semantics beyond what is needed to keep
the existing v02 fixed point runnable.

## Context

`selfhost/v02/` is the next-surface Tya-written compiler. It already has
stage-1/stage-2/stage-3 testscript coverage under
`tests/testdata/v02_selfhost/`, but that coverage is still far behind the latest
repository specification. The selected overall migration policy is maximum
coverage: by the end of the split PRD sequence, v02 should be exercised against
the same black-box specification fixture families that define Go compiler
behavior.

This first PRD is intentionally limited to the front end. It should make the v02
compiler able to lex and parse current-spec syntax into deterministic AST-like
dictionaries. Later PRDs will enforce checker semantics, emit C for the full
surface, and run the full fixture set through v02.

Current language authority remains the Go lexer, parser, AST, checker, C
emitter, runtime, `docs/SPEC.md`, `docs/API.md`, `docs/STDLIB.md`, and
`tests/testdata/**` scripts. `selfhost/v01/` remains the maintained legacy
fixed-point invariant and must stay green.

## Behavior

- `selfhost/v02/compiler.tya` recognizes every token and grammar form needed by
  the current repository specification, including syntax introduced after the
  existing v02 fixtures.
- Parser output stays deterministic and dictionary/array based, matching the
  existing v02 style rather than introducing a new object model.
- Unsupported checker or codegen behavior may still fail after parse, but parse
  failures should be limited to genuinely invalid source.
- The existing v02 fixed-point script keeps passing after each small parser
  migration checkpoint.
- New parser-focused fixtures compare v02 parser acceptance against the Go
  compiler for current syntax families before deeper checker/codegen work
  begins.
- The implementation remains hand-written. Do not add parser generators or
  grammar frameworks.

## Scope

- `selfhost/v02/compiler.tya`
- `selfhost/v02/ast.tya`
- parser/front-end-focused fixtures under `tests/testdata/v02_selfhost/`
- parser/front-end-focused harness updates in `tests/*v02*_test.go` if needed
- small docs updates only if needed to clarify that this PRD covers front-end
  migration, not checker/codegen completion

Implementation order must be small and reviewable:

1. Inventory current Go parser syntax families from `docs/SPEC.md` and
   `tests/testdata/v*/`.
2. Add a focused v02 parser acceptance fixture for one syntax family.
3. Implement only the lexer/parser/AST changes needed for that family.
4. Re-run the focused v02 fixture and `TestSelfhostV02Scripts`.
5. Repeat for the next syntax family.

Suggested syntax-family checkpoints:

- reserved words and context-sensitive words through the current spec
- imports, aliases, bare package imports, directory packages, and package file
  forms
- class and interface headers and bodies, including current modifiers,
  inheritance, implementation, defaults, fields, methods, and initializer shapes
- current function/lambda forms and canonical continuation forms
- match expressions/statements and patterns
- try/raise forms
- spawn/await/scope/select/channel syntax
- embed syntax
- raw, bytes, triple-quoted, interpolated, and multi-line string forms
- current primitive method call surfaces as ordinary member/call syntax

## Out of Scope

- Checker correctness for the newly parsed forms except for maintaining existing
  v02 self-host behavior.
- C emitter support for newly parsed forms except for preserving the existing
  fixed point.
- Running the full Go compiler black-box fixture suite through v02.
- Removing `cmd/tya` or any `internal/*` Go source.
- Making v02 the default compiler.
- Replacing or deleting `selfhost/v01/`.
- Reworking the AST representation into classes or a generated schema.

## Acceptance Criteria

- The v02 lexer recognizes all current token forms needed by `docs/SPEC.md` and
  the black-box fixture families under `tests/testdata/`.
- The v02 parser accepts valid current-spec syntax families listed in Scope and
  produces deterministic AST dictionaries for them.
- Invalid parser fixtures fail deterministically and do not silently produce
  malformed AST dictionaries.
- Existing `tests/testdata/v02_selfhost/fixed_point.txtar` still passes.
- Existing `TestSelfhostV01Scripts` still passes.
- No Go implementation files are removed.
- The changes are staged as small parser-family checkpoints rather than one
  broad rewrite.

## Verification

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestSelfhostV02Scripts -count=1
go test ./tests -run TestV02Scripts -count=1
go test ./... -count=1
```

## Dependencies

None. This is the first split PRD in the v02 current-spec migration sequence.

## Open Questions

None.
