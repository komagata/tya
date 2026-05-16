---
status: completed
goal_ready: false
---

# Feature: Selfhost V02 Checker Current Spec

## Goal

Migrate the `selfhost/v02/` checker so programs accepted or rejected by the
current Go checker receive equivalent semantic treatment in the Tya-written
compiler, while preserving the v01 and v02 self-host fixed-point gates.

## Context

This PRD depends on `selfhost-v02-lexer-parser-current-spec.md`. After v02 can
parse current syntax into deterministic AST dictionaries, the checker must grow
from the older surface to current repository semantics. The overall migration
target is maximum coverage: the completed v02 sequence should be able to run the
same black-box specification fixture families that define the Go compiler.

The Go checker remains the current authority during this PRD. `selfhost/v02/`
should mirror its user-visible accept/reject behavior where practical, but this
PRD does not require byte-for-byte diagnostic text or stable Go diagnostic
codes.

## Behavior

- `selfhost/v02/compiler.tya` performs current semantic validation for parsed
  language forms before C emission.
- Valid programs from the current black-box fixture families pass v02 checking.
- Invalid semantic fixtures fail deterministically before codegen.
- Checker behavior is implemented as small semantic-family checkpoints.
- v02 diagnostics are deterministic and actionable enough to maintain the
  self-host compiler, even when they are less polished than Go diagnostics.
- Existing v01 and v02 fixed-point scripts remain green throughout the final
  merged result.

## Scope

- `selfhost/v02/compiler.tya`
- `selfhost/v02/ast.tya` only when checker traversal needs AST helper updates
- checker-focused fixtures under `tests/testdata/v02_selfhost/`
- v02 test harness updates if needed to exercise checker-only acceptance and
  rejection cases

Implementation checkpoints should follow semantic families rather than broad
rewrites:

1. top-level binding, import, alias, package, and visibility rules
2. function and lambda scope rules
3. class rules, inheritance, modifiers, constructors, `self`, `super`, fields,
   methods, class methods, and privacy
4. interface rules, inheritance, defaults, fields, implementations, and
   initializer restrictions
5. control-flow rules for `return`, `break`, `continue`, `try`, `raise`,
   `match`, `scope`, `spawn`, `await`, and `select`
6. current primitive class/method surface and removed legacy APIs
7. embed and native-package semantic checks that happen before codegen
8. diagnostics for invalid current-spec fixtures

## Out of Scope

- C emitter implementation for newly accepted valid forms, except for preserving
  the existing fixed point.
- Running the final full black-box fixture suite through v02; that belongs to
  `selfhost-v02-full-spec-fixed-point.md`.
- Removing Go sources or making v02 the default compiler.
- Matching every Go diagnostic message byte-for-byte.
- Replacing `selfhost/v01/`.

## Acceptance Criteria

- v02 checker accepts current-spec valid semantic fixtures selected from the Go
  black-box fixture families.
- v02 checker rejects current-spec invalid semantic fixtures deterministically.
- v02 checker no longer relies on legacy-only semantic exemptions for the v02
  compiler itself.
- Existing v02 fixed-point scripts still pass.
- Existing v01 fixed-point scripts still pass.
- No Go implementation files are removed.
- Changes are staged as small semantic-family checkpoints.

## Verification

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestSelfhostV02Scripts -count=1
go test ./tests -run TestV02Scripts -count=1
go test ./... -count=1
```

## Dependencies

- `feature-specs/selfhost-v02-lexer-parser-current-spec.md`

## Open Questions

None.
