# Tya v0.45 Specification

This document is the specification for Tya v0.45, the completion
release for the class-oriented namespace and entry-file model that
shipped in v0.44.

> **Audience.** A future agent / contributor who is about to land
> any of the v0.45 M-tasks. Read this together with
> [`docs/v0.44/SPEC.md`](../v0.44/SPEC.md) (the model itself) and
> [`ROADMAP.md`](../../ROADMAP.md) (the dependency chain
> M7 → M5 → M8 → M6 → M9 → M10).

## Theme

v0.44 shipped the **model**: directory-as-package, PascalCase class
files, lowercase script files, within-package bare class
references, same-segment collision detection, and structured
`[TYA-EXXXX]` diagnostic codes. Six follow-up tasks were held back
because they require either an AST-merge refactor or working-tree
coordination that did not fit the v0.44 window.

v0.45 lands those six tasks. After v0.45:

- the `module` keyword is gone;
- every stdlib package is in class-file form;
- every example is in class-file form;
- the Tya-written self-host compiler resolves directory packages;
- `docs/SPEC.md` carries the new model as the current spec.

## Goals

- **G1.** Migrate every `examples/*.tya` to the v0.44 model. Remove
  every `module` declaration from `examples/`.
- **G2.** Enforce cross-file private class visibility with a real
  AST-level pipeline. Source-concat is replaced by AST-merge with
  `OriginFile` propagation. Diagnostic `[TYA-E0406]`.
- **G3.** Bring the Tya-written self-host compiler up to the v0.44
  surface (directory packages, class files, script entries) and
  prove its stage-2 == stage-3 fixed point. Both `selfhost/v01/` and
  the new self-host live side by side until parity is proven.
- **G4.** Migrate the eight stdlib packages held back in v0.44 —
  `string`, `array`, `dict`, `runtime`, `time`, `channel`, `sync`,
  `task` — to the class form.
- **G5.** Remove the `module` keyword from the language. Parser
  rejects it with `[TYA-E0200]`. Checker, formatter, and C emitter
  drop every `module` code path.
- **G6.** Promote `docs/v0.44/SPEC.md` content into `docs/SPEC.md`.
  Rewrite `docs/NAMING.md` for the new file-kind rules. Update
  `docs/STDLIB.md`, `docs/CANONICAL_SYNTAX.md`, `docs/GUIDE.md`,
  `docs/API.md`, `docs/TERMINOLOGY.md`, `docs/LIBRARIES.md` to drop
  module-era language. Freeze `docs/v0.45/`.

## Non-Goals

v0.45 explicitly does **not** include:

- Removing the Go reference implementation. That ships at v1.0.0
  (Commitment 6).
- Primitive-as-class sugar (`1` desugared to `Integer(1)`).
- Module mixins / `include` / `static class` / Ruby-style `module`.
- Any change to lambda or class member syntax.
- Any change to the C runtime or GC.
- Automatic migration of third-party Tya code. The migration tool,
  if any, ships in a later minor.
- A new minor-version-level surface feature. v0.45 is a completion
  release; new features land at v0.46+.

## Dependency Chain

The six M-tasks have a fixed implementation order. Each STEP must
keep `go test ./... -count=1` and the self-host gate green before
the next STEP begins.

```
M7 (examples migration)        — independent, mechanical
M5 (cross-file private)        — independent, refactor
       ↓
M8 (selfhost v0.44 surface)    — critical path
       ↓
M6 (remaining 8 stdlib pkgs)   — blocked on M8
       ↓
M9 (module keyword removal)    — blocked on M8 + M6
       ↓
M10 (docs promotion)           — final
```

M7 and M5 can land in either order, in parallel with each other, and
before M8. Everything south of M8 must wait for M8.

## M7 — Examples Migration

### Goal

Every file under `examples/` is either:

- a lowercase **script file** (an entry); or
- a PascalCase **class file** (a library piece used by sibling
  script files).

No `module name` declaration remains in `examples/`.

### Scope

- `examples/*.tya` (top-level demos).
- `examples/classic/*.tya` (small programs).
- `examples/concurrent/*.tya` (concurrency demos).
- Any nested `examples/*/` directory.

### Procedure

1. For each entry file: confirm filename is lowercase. Convert any
   top-level `print "..."` / `assert ...` / `assert_equal ...` to
   the paren form (already enforced by v0.44 strict no-paren mode,
   but legacy files may have escaped — verify with `tya check`).
2. For each `module name + functions` library file referenced by an
   example: convert to a class file at `examples/<dir>/<Class>.tya`
   in PascalCase. Update the importing script to use the new
   `<dir>.<Class>` surface.
3. Delete the old `module` file once nothing imports it.
4. Update `scripts/go_emit_examples_check.sh` if its file-list
   assumptions change.
5. Update any test fixture that names an example by path.

### Acceptance

- `find examples -name '*.tya' -exec grep -l '^module ' {} \;`
  returns no matches.
- `sh scripts/go_emit_examples_check.sh` is green.
- `go test ./... -count=1` is green.
- The self-host gate is green.

### Diagnostics

None new. M7 is a content migration.

## M5 — Cross-File Private Class Enforcement

### Goal

When a class file or script file declares a class beyond its public
class, that class is **private to the declaring file**. References
from any other file — including sibling files in the same directory
package — are rejected.

### Why It Was Held Back

The v0.44 implementation synthesizes a directory package by
**source-concatenation**: it reads every `<dir>/*.tya`, concatenates
their source bytes, and parses the result as one logical module.
Every class in the synthesized AST therefore has the same single
"origin", and the checker cannot tell whether a reference crosses a
file boundary.

To enforce cross-file privacy, every `ClassDecl` must remember
**which file it was parsed from**. v0.45 replaces the source-concat
pipeline with an AST-merge pipeline that propagates this metadata.

### Implementation Sketch

1. Parse each file in the package directory independently into its
   own `ast.Program`. Annotate every top-level declaration with
   `OriginFile string` (the file path relative to the package root).
2. Merge the per-file programs into a single synthesized module
   AST. Preserve `OriginFile` on every node — do not lose it during
   merge.
3. In the checker, when resolving a reference to a class:
   - If the target class's `IsPublic` (filename-matching public
     class) is true, the reference is always allowed.
   - Otherwise, the reference is allowed only if the reference site's
     `OriginFile` equals the target class's `OriginFile`.
   - Otherwise, emit `[TYA-E0406] private class X is not visible
     from Y.tya (declared in Z.tya)` with a hint pointing at the
     declaring file.
4. Codegen must continue to lower private classes correctly. Since
   privacy is checked before codegen, the C emitter does not need
   `OriginFile` itself.

### Touch Points

- `internal/runner/runner.go` — replace `synthesizePackageSource`
  with the new AST-merge pipeline.
- `internal/ast/ast.go` — add `OriginFile string` to `ClassDecl` (and
  any other top-level decl that participates in visibility).
- `internal/parser/parser.go` — accept and propagate a file path
  parameter, attach it to every top-level decl.
- `internal/checker/checker.go` — enforce the cross-file check in
  the class-resolution path.
- `tests/testdata/v45/cross_file_private/` — positive and negative
  fixtures.

### Acceptance

- Positive: a private class is reachable from its own file (call,
  member, extends, implements, super, return).
- Negative: every other cross-file reference variant raises
  `[TYA-E0406]` with the correct site / declaring-file pair.
- `go test ./... -count=1` green; self-host gate green.

### Diagnostic

- `[TYA-E0406]` — checker, reserved in v0.44. **Wire in M5.**

## M8 — Self-Host Compiler v0.44 Surface

### Goal

The Tya-written self-host compiler resolves directory packages,
parses class files, runs script-file entries, and reaches its own
stage-2 == stage-3 fixed point on the v0.44 surface.

### Why It Is the Critical Path

The current `selfhost/v01/compiler.tya` is a v0.1 surface compiler.
It resolves `import X` by reading `X.tya` as a single-file module.
It does not understand directory packages, class files, or the
v0.44 import grammar. Three v0.45 deliverables block on its
upgrade:

- M6 cannot migrate `stdlib/string`, `stdlib/array`, `stdlib/dict`
  to class form while v01 still needs them as single-file modules
  (the v0.44 SPEC §"Self-Host Invariant Constraint" documents this).
- M9 cannot remove the `module` keyword while v01 still emits and
  consumes it.
- M10 cannot promote `docs/SPEC.md` to the new model while a
  released artifact contradicts the spec.

### Implementation Sketch

The recommended path is to land a **new** `selfhost/v02/compiler.tya`
on the v0.44 surface, side by side with `v01`. The v01 compiler
keeps compiling itself on the v0.43 surface — i.e. v01 is frozen at
v0.43 semantics — until v02 reaches stage-2 == stage-3 parity.

1. Copy `selfhost/v01/compiler.tya` to `selfhost/v02/compiler.tya`.
2. Extend the v02 lexer and parser for v0.44 syntax: `class`,
   `interface`, `@@member`, `extends`, `implements`, `super`,
   `abstract`, `final`, `override`, and the strict no-paren-call
   rule.
3. Extend the v02 runner to resolve directory packages — replicate
   the Go runner's `resolvePackageDir` / `synthesizePackageSource`
   semantics in Tya. (For v02 itself, source-concat is acceptable as
   a starting point; AST-merge can land later if v02 needs cross-file
   privacy.)
4. Add `tests/testdata/v02_selfhost/*.txtar` mirroring the v01 suite
   plus class-file and directory-package coverage.
5. Add `TestSelfhostV02Scripts` to `tests/`. Both gates run in CI.
6. Prove stage-2 == stage-3 byte-equivalence for v02 on the v0.44
   surface.

### Transition

- v0.45 ships with **both** `v01` (v0.43 fixed point) and `v02`
  (v0.44 fixed point) gates green.
- The default `TestSelfhostV01Scripts` gate continues to pass until
  M9 deletes the legacy `module` keyword. At that point, `v01`
  cannot compile itself any more and must be retired — see M9.
- At a release boundary (v0.46 or later), the project may delete
  `selfhost/v01/` entirely and elevate `v02` to be the sole gate.
  That decision is out of v0.45 scope.

### Acceptance

- `selfhost/v02/compiler.tya` exists and compiles itself.
- `TestSelfhostV02Scripts` is green.
- `TestSelfhostV01Scripts` is still green (v01 untouched).
- `go test ./... -count=1` green.

### Diagnostics

None new. M8 is a self-host upgrade.

## M6 — Remaining stdlib Migration

### Goal

Every stdlib package is in class-file form. The legacy `stdlib/<x>.tya`
single-file modules are deleted.

### Held-Back Packages

| Package    | Reason held back in v0.44                             | Unblocked by |
| ---------- | ----------------------------------------------------- | ------------ |
| `string`   | Used by v01 self-host (single-file import)            | M8           |
| `array`    | Used by v01 self-host (single-file import)            | M8           |
| `dict`     | Grouped with `string`/`array` for consistency         | M8           |
| `runtime`  | Cross-cutting; callers in `examples/` were unmigrated | M7           |
| `time`     | Same as `runtime`                                     | M7           |
| `channel`  | Same as `runtime`                                     | M7           |
| `sync`     | Same as `runtime`                                     | M7           |
| `task`     | Same as `runtime`                                     | M7           |

### Procedure

Per package:

1. Create `stdlib/<pkg>/<Class>.tya` in class-file form. Class name
   matches the existing module's "namespace" — e.g.
   `stdlib/string.tya` (legacy) → `stdlib/string/String.tya` with
   `class String` and the existing functions as `@@member`s.
2. Update every caller in `stdlib/`, `selfhost/v02/`, `examples/`,
   `tests/testdata/`, and the Go runner (if it hard-codes the
   module name) to use the new `string.String.<member>` surface.
3. Delete the legacy `stdlib/<pkg>.tya` file once nothing imports it.
4. Update `docs/STDLIB.md` per package as it lands.

### Acceptance

- `find stdlib -maxdepth 1 -name '*.tya'` returns only the eight
  packages that legitimately stay flat (none, after M6). All other
  stdlib content lives under `stdlib/<pkg>/<Class>.tya`.
- `go test ./... -count=1` green.
- Both self-host gates (v01 and v02) green.

### Diagnostics

None new.

## M9 — `module` Keyword Removal

### Goal

The `module` keyword is no longer accepted anywhere in the Tya
language.

### Implementation Sketch

1. Parser: when a top-level statement begins with `module`, emit
   `[TYA-E0200]` (parser) with the hint "the module keyword was
   removed in v0.45; use a directory package and PascalCase class
   files".
2. Checker: delete every `ModuleDecl` code path.
3. Formatter: delete every `ModuleDecl` unparse path.
4. C emitter: delete every `ModuleDecl` lowering.
5. Delete `selfhost/v01/` — it cannot compile itself any more on the
   v0.45 surface. The maintained self-host gate becomes
   `TestSelfhostV02Scripts` exclusively.
6. Delete any `module name` reference in `docs/`, `tests/testdata/`,
   and `examples/` that survived M6/M7.

### Pre-conditions

- M7 done (no `module` in `examples/`).
- M6 done (no `module` in `stdlib/`).
- M8 done (`selfhost/v02/` carries the maintained self-host gate).

### Acceptance

- `grep -rn '^module ' .` returns no matches in tracked code.
- Parser rejects `module x` with `[TYA-E0200]`.
- `go test ./... -count=1` green.

### Diagnostic

- `[TYA-E0200]` — parser, reserved in v0.44. **Wire in M9.**

## M10 — Docs Promotion

### Goal

`docs/SPEC.md` is the v0.44/v0.45 surface. The frozen v0.44 and v0.45
snapshots live under `docs/v0.44/` and `docs/v0.45/`.

### Procedure

1. Copy `docs/v0.44/SPEC.md` content into `docs/SPEC.md`, adapting
   "v0.44" → "current spec" where appropriate. Remove the "Self-Host
   Invariant Constraint" section (the constraint is resolved by M8).
2. Rewrite `docs/NAMING.md`:
   - Remove the "Module Rule" section.
   - Add the "File Kind Rule" (lowercase = script, PascalCase = class).
   - Update example fragments.
3. Update `docs/STDLIB.md` cross-references for the new
   `<pkg>.<Class>.<member>` surface.
4. Update `docs/CANONICAL_SYNTAX.md` for the file-kind rules.
5. Update `docs/GUIDE.md`, `docs/API.md`, `docs/TERMINOLOGY.md`,
   `docs/LIBRARIES.md` to drop module-era language and refer to
   classes and packages instead.
6. Rebuild HTML via `node scripts/build_docs_pages.js`.
7. Freeze `docs/v0.45/SPEC.md` (this file) as the v0.45 spec
   snapshot. Add a `docs/v0.45/RELEASE_NOTES.md` summarizing M5–M10.
   Add the v0.45 entry to `ROADMAP.md` under "Released".
8. Bump `const version` in `cmd/tya/main.go` to `0.45.0`.

### Acceptance

- `grep -rn 'module' docs/SPEC.md docs/NAMING.md docs/STDLIB.md` —
  only legitimate non-keyword uses remain (e.g. prose about
  modules-as-packages).
- HTML rendering builds without errors.
- `go test ./... -count=1` green.

## Errors Table (v0.45 deltas)

| Code        | Wired in   | Stage   | Condition                                                  |
| ----------- | ---------- | ------- | ---------------------------------------------------------- |
| `[TYA-E0406]` | M5       | checker | Cross-file reference to a private class.                   |
| `[TYA-E0200]` | M9       | parser  | `module` keyword used (removed in v0.45).                  |

All other v0.44 codes (E0400, E0402–E0405, E0850–E0855) remain
in place unchanged.

## Verification Reference

Every STEP must keep these green:

```sh
go test ./... -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1    # until M9 deletes v01
go test ./tests -run TestSelfhostV02Scripts -count=1    # starting in M8
sh scripts/go_emit_examples_check.sh                    # after M7
```

After M9 removes `selfhost/v01/`, drop the v01 line and keep the
v02 line as the sole self-host gate.

## Release Checklist (M10 finalization)

- [ ] `cmd/tya/main.go` `const version = "0.45.0"`.
- [ ] `docs/v0.45/SPEC.md` frozen.
- [ ] `docs/v0.45/RELEASE_NOTES.md` written.
- [ ] `docs/SPEC.md`, `docs/NAMING.md`, `docs/STDLIB.md`,
      `docs/CANONICAL_SYNTAX.md`, `docs/GUIDE.md`, `docs/API.md`,
      `docs/TERMINOLOGY.md`, `docs/LIBRARIES.md` updated.
- [ ] `ROADMAP.md` `Released` updated; `Scheduled` v0.45 entry removed.
- [ ] `node scripts/build_docs_pages.js` rebuilt.
- [ ] `go test ./... -count=1` green.
- [ ] `TestSelfhostV02Scripts` green; `v01` retired.
- [ ] Tag `v0.45.0`, build platform tarballs via
      `scripts/build_release_packages.sh 0.45.0`, create GitHub
      Release, update `komagata/homebrew-tap/Formula/tya.rb`.

## Cross-References

- [`docs/v0.44/SPEC.md`](../v0.44/SPEC.md) — the model itself.
- [`docs/v0.44/MIGRATION.md`](../v0.44/MIGRATION.md) — user-facing
  migration matrix.
- [`docs/v0.44/RELEASE_NOTES.md`](../v0.44/RELEASE_NOTES.md) — what
  shipped in v0.44.
- [`ROADMAP.md`](../../ROADMAP.md) — Scheduled § v0.45 entry mirrors
  this document at task-list granularity.
