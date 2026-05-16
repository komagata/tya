# Feature: Remove Underscore Private Binding Semantics

## Goal

Make leading `_` ordinary spelling for value names instead of a visibility
marker. Top-level bindings that begin with `_` should behave like other
top-level bindings, including import namespace export, documentation, LSP
symbols, and checker behavior. Class member privacy remains expressed by the
`private` keyword.

## Context

Current implementation still has older `_`-based privacy behavior in several
places, while the current specification is being updated to remove that rule:

- `internal/runner/runner.go` excludes `_` top-level names from single-file
  import namespace synthesis.
- `internal/checker/checker.go` uses `isPrivateName` and rejects or special
  cases `_` names in binding, import, class member, and interface paths.
- `internal/checker/strict.go` still reports unused private top-level
  definitions for `_helper`-style names.
- `internal/doc/extract.go` and `internal/lsp/symbols.go` skip `_` names when
  extracting documentation and symbols.
- Parser and codegen still contain compatibility paths for old underscore
  private class members.
- Docs now describe leading `_` as having no visibility meaning for ordinary
  bindings, and class privacy as keyword-based via `private`.

The implementation should follow the new docs: names express naming category,
not accessibility. `private` remains the visibility construct for class fields,
methods, class variables, class methods, and constructors.

## Behavior

- A top-level assignment, function value, class declaration, or embed whose
  name begins with `_` is exported through a single-file import binding the
  same way as other top-level names.
- `tya check`, `tya run`, `tya build`, `tya emit-c`, `tya doc`, and `tya lsp`
  must not treat `_` top-level names as private.
- Unused `_helper`-style top-level definitions must not produce the old unused
  private top-level diagnostic.
- Leading `_` in function parameters may remain a convention for ignored
  parameters if that behavior is already separate from visibility, but it must
  not imply export or access control.
- Import paths and import aliases remain `snake_case` public path names; this
  feature does not make `_module` import paths valid unless existing import
  path rules already allow them.
- Class member privacy is controlled by `private`. The implementation should
  remove underscore-as-private compatibility for active, non-legacy class
  member paths where doing so does not break archived historical fixtures.
- `private` class members must keep their existing access restrictions and
  code generation behavior.
- Directory package public class/interface export rules remain filename- and
  declaration-based. This feature only removes leading `_` as a visibility
  marker.

## Scope

- `internal/runner/runner.go` and runner tests for single-file import namespace
  synthesis.
- `internal/checker/checker.go` and checker tests for name validation,
  top-level binding handling, class member privacy, and interface visibility.
- `internal/checker/strict.go` and diagnostics fixtures for removing unused
  private top-level behavior.
- `internal/doc/extract.go` and doc tests for extracting `_` top-level names.
- `internal/lsp/symbols.go` and LSP tests for exposing `_` top-level names.
- `internal/parser/parser.go` and `internal/codegen/c.go` only where active
  underscore-private class member compatibility conflicts with `private`.
- Current docs touched by this behavior, especially `docs/SPEC.md` and
  `docs/SPEC_ja.md`.
- Tests under `tests/` and `tests/testdata/` that assert old `_` private
  behavior.

## Out of Scope

- Adding a new top-level privacy feature.
- Changing `private` keyword syntax or class member semantics.
- Changing import path naming rules.
- Rewriting archived version docs under `docs/v*/`.
- Removing historical migration notes for old releases.
- Renaming existing stdlib or test helper names unless a test expectation must
  change.

## Acceptance Criteria

- A single-file module containing `_helper = -> ...` exposes
  `module._helper()` after `import module`.
- A top-level `_helper` no longer triggers an unused private top-level
  diagnostic.
- `tya doc` includes documented `_` top-level bindings.
- LSP document symbols include `_` top-level bindings.
- Class members remain private only when declared with `private`; `_name` alone
  must not create private access.
- Existing tests that intentionally cover historical versions may stay scoped
  to archived version behavior, but active current-spec tests must reflect the
  new rule.
- Current docs do not describe `_` as a visibility marker.
- Full repository tests pass after implementation.

## Verification

```sh
go test ./internal/runner -count=1
go test ./internal/checker -count=1
go test ./internal/doc -count=1
go test ./internal/lsp -count=1
go test ./tests -count=1
go test ./... -count=1 -timeout=20m
```
