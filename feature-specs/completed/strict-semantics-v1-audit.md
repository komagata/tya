# Feature: Strict Semantics v1 Audit

## Goal

Make Tya's v1.0.0 strict-semantics contract explicit, tested, and consistently diagnosed across parser, checker, interpreter, C emitter, runtime, CLI, LSP, and self-host gates.

## Context

- `ROADMAP.md` tracks **Strict semantics audit for v1.0.0** as a v1.0.0 blocker.
- Current `internal/checker/strict.go` mostly covers strict name hygiene:
  - shadowed bindings
  - unused imports
  - unused arguments
  - canonical `Self` member references
  - captured mutation checks
- The roadmap item is broader than the current strict pass. It includes no-implicit-conversion, no-`nil` arithmetic, truthiness, comparison, indexing, assignment, argument, and return-value rules.
- Tya is dynamically typed, so this feature is not a static type system. The goal is to define which dynamic behaviors are valid, which invalid programs must fail early or clearly, and which dynamic behaviors intentionally remain valid.

## Behavior

- Create a strict-semantics contract table covering at least:
  - implicit conversions
  - arithmetic operands
  - string concatenation and interpolation
  - `nil` in arithmetic, comparison, indexing, calls, member access, conditions, loops, match, and select
  - truthiness rules
  - equality and ordering comparison rules
  - array/string/dict indexing rules
  - assignment and reassignment rules
  - constants and class/interface names
  - function and method argument counts
  - multi-return assignment and return-value arity
  - return outside function
  - break/continue outside loops
  - callable vs non-callable values
  - member access on primitives, classes, instances, modules, and dictionaries
  - pattern matching validity
- For each rule, record:
  - valid behavior
  - invalid behavior
  - whether the failure should be parser, checker, runtime, or C-runtime
  - stable diagnostic/error code when available
  - SPEC section anchor
  - active test or testscript fixture
- Add or update current docs:
  - `docs/SPEC.md`
  - `docs/ja/spec.md`
  - a compact audit table under `docs/` if the table is too large for SPEC
- Invalid programs should fail as early as feasible:
  - parser errors for syntactic invalidity
  - checker diagnostics for statically visible invalidity
  - runtime errors for dynamic invalidity that cannot be known statically
- Diagnostics should be structured where the existing diagnostic pipeline can express them:
  - stable `TYA-E....` code
  - title
  - expected/found or equivalent concrete message
  - actionable hint when useful
  - source region where available
- Preserve intentionally dynamic behavior:
  - equality may compare values of different runtime classes when SPEC says so
  - `nil` may be returned and compared to `nil`
  - dictionary lookup may return `nil` for missing keys only where documented
  - dynamic dispatch remains valid where member/call validity cannot be proven statically
- Ensure interpreter and compiled C behavior agree for all covered valid and invalid programs.
- Ensure `tya check`, `tya run`, `tya build`, LSP diagnostics, and self-host fixed-point gates do not disagree on accepted source syntax.

## Scope

- Audit and update:
  - `docs/SPEC.md`
  - `docs/ja/spec.md`
  - `internal/parser`
  - `internal/checker`
  - `internal/eval`
  - `internal/codegen`
  - `runtime/tya_runtime.c`
  - `runtime/tya_runtime.h`
  - `cmd/tya`
  - `internal/lsp`
  - `selfhost/v01` and `selfhost/v02` only where needed to keep documented gates passing
- Add focused fixtures for each contract rule:
  - parser/checker unit tests where the failure is static
  - testscript fixtures where CLI behavior or compiled/runtime parity matters
  - LSP diagnostic tests for representative checker diagnostics
  - self-host gate tests if a strict rule touches the self-hosted compiler surface
- Add an audit artifact, for example `docs/STRICT_SEMANTICS.md`, when the rule matrix is too detailed for SPEC.

## Out of Scope

- Adding a static type system.
- Requiring full compile-time inference for dynamic values.
- Removing intentionally dynamic language behavior that SPEC explicitly keeps.
- Redesigning syntax.
- Changing canonical formatting rules except where a strict-semantics rule exposes an existing contradiction.
- Implementing new v1.0.0 features unrelated to semantic validity.

## Acceptance Criteria

- Every roadmap-listed rule family has an explicit contract entry:
  - no-implicit-conversion
  - no-`nil` arithmetic
  - truthiness
  - comparison
  - indexing
  - assignment
  - argument rules
  - return-value rules
- Every contract entry maps to current SPEC text or adds missing SPEC text in English and Japanese.
- Every contract entry maps to at least one active test or testscript fixture.
- Static invalid programs produce checker/parser diagnostics instead of silently compiling or failing unclearly later.
- Dynamic invalid programs produce consistent interpreter and C-runtime errors where static checking is not feasible.
- `tya check`, `tya run`, and `tya build` agree on the validity of covered programs.
- LSP diagnostics expose representative strict checker failures with matching codes and warning/error severity.
- Self-host v01 and v02 gates still pass, or any required legacy exception is explicitly documented.
- The final audit table has no blank rule, SPEC, test, or diagnostic fields.

## Verification

```sh
go test ./internal/parser -count=1
go test ./internal/checker -count=1
go test ./internal/eval -count=1
go test ./internal/codegen -count=1
go test ./tests -run 'TestV.*Scripts|TestSelfhostV01Scripts|TestSelfhostV02Scripts|TestLSP' -count=1
go test ./... -count=1
```
