# Tya v0.45 Specification

This document is the frozen specification for Tya v0.45, a
**user-facing polish release** for the class-oriented namespace and
entry-file model that shipped in v0.44.

## Theme

v0.44 shipped the **model**: directory-as-package, PascalCase class
files, lowercase script files, within-package bare class references,
same-segment collision detection, and structured `[TYA-EXXXX]`
diagnostic codes.

v0.45 lands the three v0.44 follow-ups that do not require a
Tya-written compiler on the v0.44 surface:

1. Every `examples/*.tya` is on the new model.
2. Cross-file private class visibility is now enforced
   (`[TYA-E0406]`).
3. Five of the eight held-back stdlib packages are migrated to
   class form.

The remainder of the v0.44 follow-up work — the v02 self-host on
the v0.44 surface, the `string`/`array`/`dict` stdlib migration
that depends on it, the `module` keyword removal, and the
`docs/SPEC.md` promotion — is scheduled into a later minor in the
v1.0.0 prep window (see `ROADMAP.md` § v0.4x).

## Goals

- **G1.** Migrate every `examples/*.tya` to the v0.44 model. Remove
  every `module` declaration from `examples/`.
- **G2.** Enforce cross-file private class visibility. A class
  beyond the file's public class is private to its declaring file;
  references from any other file — including sibling files inside
  the same directory package — raise `[TYA-E0406]`.
- **G3.** Migrate five concurrency-related stdlib packages
  (`runtime`, `time`, `channel`, `sync`, `task`) to the v0.44 class
  form. Update every caller and `docs/STDLIB.md`.

## Non-Goals (deferred to v0.4x)

- A Tya-written compiler on the v0.44 surface (`selfhost/v02/`).
- Migration of `string`, `array`, `dict` to class form — these
  packages are still consumed by `selfhost/v01/compiler.tya` as
  single-file modules.
- Removal of the `module` keyword.
- Promotion of `docs/v0.44/SPEC.md` content into `docs/SPEC.md`.

## G1 — Examples migration

### Outcome

`examples/util.tya` (legacy `module util`) → `examples/util/Util.tya`
(class form). `examples/greeting.tya` (legacy `module greeting`) →
`examples/greeting/Greeting.tya`. Consumers updated to the
`<pkg>.<Class>.<member>` surface:

```tya
# examples/use_module_decl.tya
import util

print(util.Util.foo)
print(util.Util.bar())
```

### Acceptance

- `find examples -name '*.tya' -exec grep -l '^module ' {} \;`
  returns no matches.
- `sh scripts/go_emit_examples_check.sh` is green.
- `go test ./... -count=1` is green.
- The self-host gate is green.

### Notes

The Go tree-walking eval interpreter (`internal/eval/`) does not
implement v0.44 class syntax. The two migrated examples
(`use_module.tya`, `use_module_decl.tya`) are therefore removed
from `TestExamplesGolden`; their codegen path remains covered by
`scripts/go_emit_examples_check.sh`.

## G2 — Cross-file private class enforcement

### Diagnostic

`[TYA-E0406]` — checker. Reserved in v0.44, wired in v0.45.

### Outcome

A class file's non-public sibling classes cannot be referenced
from outside the file that declared them — including sibling files
inside the same directory package and external entry scripts.

Positive: a private class is reachable from its own file (call,
member, extends).

Negative: every other cross-file reference raises `[TYA-E0406]`
with the site / declaring-file pair.

### Implementation

- `ast.ClassDecl` gains an `OriginFile string` field.
- The runner's `synthesizePackageSource` returns a class-name →
  origin-file map alongside the synthesized source. The runner
  stashes the map on `loadState.classOrigins` keyed by synthesized
  package module name.
- `runner.LoadUserSourceWithOrigins` /
  `runner.LoadSourceWithOrigins` expose the map. After the merged
  source is re-parsed, `runner.StampOriginFiles` walks the AST and
  writes `OriginFile` onto every `ClassDecl` in a synthesized
  `ModuleDecl`.
- `checker.classInfo` gains `originFile`;
  `predeclareModuleClass` copies it from the AST.
- `checker.checkCrossFilePrivate` enforces the rule: a class is
  public iff its bare name + `".tya"` equals its `OriginFile`;
  references to a private class from any other file raise
  `[TYA-E0406]`. Wired into `checkClassCall`, the within-package
  bare-Ident class lookup, the `kindModule` `MemberExpr` path, and
  the `extends`-parent resolution in `checkClass`.

Classes from single-file scripts or legacy `module` declarations
leave `OriginFile == ""` and bypass the check.

### Fixtures

Under `tests/testdata/v45/`:

- `cross_file_private_positive.txtar` — private class reachable
  from its own file (positive).
- `cross_file_private_member.txtar` — cross-file member access
  (negative).
- `cross_file_private_extends.txtar` — cross-file `extends`
  (negative).
- `cross_file_private_external.txtar` — entry script reaches into
  a package's private class (negative).

## G3 — stdlib migration (partial)

### Migrated

| Package    | Old shape                | New shape                            |
| ---------- | ------------------------ | ------------------------------------ |
| `runtime`  | `stdlib/runtime.tya`     | `stdlib/runtime/Runtime.tya`         |
| `time`     | `stdlib/time.tya`        | `stdlib/time/Time.tya`               |
| `channel`  | `stdlib/channel.tya`     | `stdlib/channel/Channel.tya`         |
| `sync`     | `stdlib/sync.tya`        | `stdlib/sync/Sync.tya`               |
| `task`     | `stdlib/task.tya`        | `stdlib/task/Task.tya`               |

Member access changes from `pkg.member(args)` to
`pkg.<Pkg>.member(args)`:

```tya
import runtime
stats = runtime.Runtime.gc_stats()

import time
t = time.Time.now()

import channel
c = channel.Channel.new(10)
channel.Channel.send(c, "hi")

import sync
m = sync.Sync.mutex()

import task
me = task.Task.current()
```

Callers updated in `examples/`, `tests/`, `tests/testdata/`, and
`docs/STDLIB.md`. The legacy single-file `stdlib/<pkg>.tya` files
are deleted.

### Held back (to v0.4x)

`string`, `array`, `dict` remain at `stdlib/<pkg>.tya` (single
file). `selfhost/v01/compiler.tya` still consumes them via
single-file `import` and must do so until the v02 self-host
upgrade lands.

## Verification

```sh
go test ./... -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestSelfhostV02Scripts -count=1
sh scripts/go_emit_examples_check.sh
```

All four are green at the v0.45 release tag.

## Release artifacts

- `cmd/tya/main.go` — `const version = "0.45.0"`.
- `docs/v0.45/SPEC.md` — this file, frozen.
- `docs/v0.45/RELEASE_NOTES.md` — release-time summary.
- `ROADMAP.md` — v0.45 entry under `Released`; v0.4x Epic carries
  M8 + M6 remaining + M9 + M10.

## Cross-References

- [`docs/v0.44/SPEC.md`](../v0.44/SPEC.md) — the model itself.
- [`ROADMAP.md`](../../ROADMAP.md) — `Released` § v0.45 mirrors
  this document at task-list granularity; `Scheduled` § v0.4x owns
  the carry-over.
