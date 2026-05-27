---
layout: doc
title: Release Notes
permalink: /v0.45/release-notes/
---

# Tya v0.45 Release Notes

> **Status:** shipped. The `tya version` constant is `0.45.0` and
> `ROADMAP.md` carries the matching `Released` entry.

## TL;DR

v0.45 is a **user-facing polish release** on top of v0.44's
class-oriented namespace and entry-file model. Three items land:

1. Every `examples/*.tya` is on the new model (no `module`
   declarations remain).
2. **Cross-file private class enforcement** — `[TYA-E0406]`.
3. **Five stdlib packages migrated to class form**: `runtime`,
   `time`, `channel`, `sync`, `task`.

The remaining v0.44 follow-ups (self-host v02 on the v0.44 surface,
`string` / `array` / `dict` migration, `module` keyword removal,
`docs/SPEC.md` promotion) move to a dedicated pre-v1.0.0 Epic. See
`ROADMAP.md` § v0.4x.

## What's new

### Cross-file private class enforcement

In v0.44 a class file's non-public sibling classes already had
"private to the declaring file" intent, but the source-concat
package synthesis lost the file boundary by the time the checker
ran. v0.45 propagates the origin file onto every `ClassDecl` and
enforces the rule.

```tya
# pkg/Util.tya
class Util
  @@run = ->
    Internal.do_work()              # OK if Internal lives in Util.tya
                                    # [TYA-E0406] if Internal lives in
                                    # Helper.tya — private to that file
```

The check covers four reference shapes:

- `Internal()` constructor calls inside another class body.
- `Internal.member` static member access.
- `class Sub extends Internal` inheritance.
- `pkg.Internal()` reach-into from an entry script.

Diagnostic format:

```
3:5: [TYA-E0406] private class Internal is not visible from Util.tya
                 (declared in Helper.tya)
```

Single-file scripts and the legacy `module` shape leave the origin
field empty and bypass the check.

### Examples migration

```text
examples/
  greeting/Greeting.tya     # was: examples/greeting.tya (module)
  util/Util.tya             # was: examples/util.tya     (module)
  use_module.tya            # calls greeting.Greeting.hello(...)
  use_module_decl.tya       # calls util.Util.foo / util.Util.bar()
```

No `module` declarations remain anywhere under `examples/`.

### stdlib migration (concurrency tier)

Five packages move from `lib/<pkg>.tya` (single-file module) to
`lib/<pkg>/<Pkg>.tya` (directory package, PascalCase public
class). Their public surface is now reached via
`<pkg>.<Pkg>.<member>`:

```tya
import runtime
stats = runtime.Runtime.gc_stats()
runtime.Runtime.gc()

import time
now = time.Time.now()
time.Time.sleep(0.1)

import channel
c = channel.Channel.new(10)
channel.Channel.send(c, "hi")
print(channel.Channel.receive(c))

import sync
m = sync.Sync.mutex()
sync.Sync.lock(m)

import task
me = task.Task.current()
while not task.Task.cancelled?(me)
  do_one_step()
```

Callers in `examples/`, `tests/`, `tests/testdata/`, and
`docs/STDLIB.md` are updated. The legacy single-file
`lib/{runtime,time,channel,sync,task}.tya` are deleted.

`string`, `array`, `dict` stay on the single-file shape; v01's
self-host still consumes them as single-file modules. The migration
ships with the v0.4x self-host upgrade.

## Diagnostic codes

| Code           | Stage   | Wired in | Condition                                    |
| -------------- | ------- | -------- | -------------------------------------------- |
| `[TYA-E0406]`  | checker | v0.45    | Cross-file reference to a private class.     |

All v0.44 codes (E0400, E0402–E0405, E0850–E0855) remain
unchanged.

## Migration

User code that imported `runtime` / `time` / `channel` / `sync` /
`task` and called the public surface needs a one-token rewrite:

```text
runtime.gc_stats()      → runtime.Runtime.gc_stats()
time.now()              → time.Time.now()
channel.new(10)         → channel.Channel.new(10)
sync.mutex()            → sync.Sync.mutex()
task.cancelled?(t)      → task.Task.cancelled?(t)
```

Importing the package itself is unchanged (`import runtime`); only
the member access path gains the class segment.

User code that relied on cross-file references into a non-public
sibling class now needs to either (a) move the reference into the
declaring file, (b) make the target class public by renaming it to
match the file name, or (c) move the target class into its own
file. The diagnostic points at the declaring file directly.

## Self-host

`TestSelfhostV01Scripts` remains the maintained self-host gate.
`TestSelfhostV02Scripts` is added in v0.45 as scaffolding for the
upcoming v02 self-host on the v0.44 surface; it currently mirrors
v01 on the v0.1-surface fixed-point program and grows toward v0.44
parity in the v0.4x Epic.

## Verification

```sh
go test ./... -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestSelfhostV02Scripts -count=1
sh scripts/go_emit_examples_check.sh
```

All four are green at the v0.45.0 tag.

## Cross-References

- [`docs/v0.45/SPEC.md`](SPEC.md) — frozen v0.45 specification.
- [`docs/v0.44/SPEC.md`](../v0.44/SPEC.md) — the model itself.
- [`docs/v0.44/MIGRATION.md`](../v0.44/MIGRATION.md) — v0.44 user
  migration matrix; the v0.45 changes are additive on top of it.
- [`ROADMAP.md`](../../ROADMAP.md) — v0.45 under `Released`; v0.4x
  under `Scheduled`.
