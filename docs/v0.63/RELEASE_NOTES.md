# Tya v0.63 Release Notes

v0.63 advances the Tya-written self-host compiler under `selfhost/v02/` to a
current-spec proof gate. The Go implementation remains the active reference,
but v02 now has focused lexer/parser, checker, C emitter, and fixed-point
coverage for representative current language families.

## Language

- Function literals in the active compile-to-C path now support lexical
  closures over enclosing function parameters and locals. Captures snapshot the
  current `TyaValue`, so heap-backed values remain shared values without deep
  copying.
- Closure bodies may read captured bindings, pass closures through higher-order
  APIs, and use captured closures with `spawn` / `await`. Direct reassignment
  to an outer binding and indexed or member mutation through a captured binding
  are rejected.

## Self-Host

- `selfhost/v02/compiler.tya` recognizes current syntax families including
  imports with paths and aliases, interfaces, class modifiers, `implements`,
  `scope`, `try`/`catch`, `raise`, `match`, `select`, `spawn`, `await`,
  `embed`, raw/bytes/triple/tagged strings, heredocs, predicate names, and
  bitwise/shift parsing.
- The v02 checker validates selected current semantic families, including import
  bindings, interface contracts, class context for `self`/`super`/`Self`,
  control-flow traversal, select arm bindings, and single-file embed existence.
- The v02 C emitter handles selected current runtime families, including
  single-file `embed`, interface/implements declarations that erase for runtime,
  class dispatch, interpolation, primitive string helpers, and deterministic
  unsupported-codegen diagnostics for parsed forms that v02 does not emit yet.
- `TestSelfhostV02Scripts` now documents the applicable v02 full-spec coverage
  and proves the v02 stage-2/stage-3 fixed point.
- The v02 fixed-point gate also checks that every `.tya` file under
  `selfhost/v02/` can be compiled by stage 1.

## Roadmap

- `ROADMAP.md` now distinguishes the completed `selfhost/v02/` current-spec
  proof from the later v1.0 work to remove the Go reference implementation and
  ship bootstrap binaries.

## Verification

The release gate is:

```sh
go test ./... -count=1
```

The published v0.63.0 tag passed the full suite, including the maintained v01
and v02 self-host fixed-point tests.
