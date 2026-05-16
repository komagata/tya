---
status: completed
goal_ready: false
---

# Feature: Selfhost V02 C Emitter Current Spec

## Goal

Migrate the `selfhost/v02/` C emitter so programs that pass the current v02
front end and checker emit valid C with runtime behavior equivalent to the Go
compiler for the current language surface.

## Context

This PRD depends on the v02 lexer/parser and checker migration PRDs. Once v02
can parse and semantically validate current-spec programs, the emitter must
produce C that links with the existing Tya runtime and behaves like the Go
compiler output for the same source.

The migration should preserve the existing dictionary/array AST style and
hand-written compiler implementation. Go code remains the active reference
during this PRD.

## Behavior

- `selfhost/v02/compiler.tya` emits deterministic, compilable C for
  current-spec programs that pass v02 checking.
- Generated C links with the existing `runtime/tya_runtime.c` and required
  runtime flags used by the test scripts.
- Runtime behavior matches Go compiler behavior for the covered black-box
  fixture families.
- Unsupported emission paths fail deterministically and do not emit partial C
  that is treated as successful output.
- Stage-2 and stage-3 C output for `selfhost/v02/compiler.tya` remains stable.

## Scope

- `selfhost/v02/compiler.tya`
- `selfhost/v02/ast.tya` only for emitter helper needs
- codegen/runtime-behavior fixtures under `tests/testdata/v02_selfhost/`
- v02 test harness updates for compiling and running emitted C

Implementation checkpoints should be narrow:

1. expression and statement emission parity for current primitive method surface
2. imports, packages, and cross-file/class-file emission
3. class, inheritance, interface, field, method, and dispatch emission
4. control-flow emission for `match`, `try`/`raise`, loops, `scope`, `spawn`,
   `await`, `select`, and channels
5. literal and embed emission, including raw/bytes/triple-quoted/interpolated
   strings
6. native-package and external build metadata emission where required by
   current fixtures
7. deterministic unsupported-codegen failures for valid-but-not-yet-emitted
   forms during intermediate checkpoints

## Out of Scope

- Rewriting the runtime architecture.
- Removing Go sources or making v02 the default compiler.
- Matching Go diagnostic text byte-for-byte.
- Replacing `selfhost/v01/`.
- Final full-suite v02 orchestration; that belongs to
  `selfhost-v02-full-spec-fixed-point.md`.

## Acceptance Criteria

- v02 emits compilable C for current-spec valid fixtures selected from the Go
  black-box fixture families.
- Compiled v02 output matches expected stdout/stderr and exit behavior for those
  fixtures.
- v02 codegen failures are deterministic for unsupported or invalid emission
  paths.
- v02 fixed point remains stage-2 == stage-3.
- v01 fixed point remains green.
- No Go implementation files are removed.
- Changes are staged as small emitter-family checkpoints.

## Verification

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestSelfhostV02Scripts -count=1
go test ./tests -run TestV02Scripts -count=1
go test ./... -count=1
```

## Dependencies

- `feature-specs/selfhost-v02-lexer-parser-current-spec.md`
- `feature-specs/selfhost-v02-checker-current-spec.md`

## Open Questions

None.
