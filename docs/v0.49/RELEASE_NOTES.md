---
layout: doc
title: Release Notes
permalink: /v0.49/release-notes/
---

# Tya v0.49 Release Notes

> **Status:** shipped. The `tya version` constant is `0.49.0` and
> `ROADMAP.md` carries the matching `Released` entry.

## TL;DR

Three new CLI subcommands ship together as the first slice of the
"Toolchain" track in `ROADMAP.md` § Future Work:

- **`tya new <name>`** — scaffold a new project (tya.toml +
  src/main.tya + .gitignore).
- **`tya task [name] [args...]`** — list and run tasks defined
  under a `[tasks]` table in tya.toml. POSIX `sh -c` is the
  execution shell. Trailing arguments are POSIX-quoted and
  appended to the task command.
- **`tya lint [paths...]`** — initial rule `TYAL0001 unused
  local` reports every unused local binding (variables and
  parameters).

The language surface is unchanged from v0.48. Programs that
compiled under v0.48 keep compiling under v0.49.

## What's new

### `tya new`

Creates a minimal project tree. The generated `tya.toml`
already contains a sample `[tasks]` entry so you can `tya task
run` straight after scaffolding:

```sh
$ tya new myapp
Created myapp
  cd myapp && tya task run
$ cd myapp && tya task run
Hello, Tya!
```

The v0.49 scaffold is intentionally minimal: no `tests/`, no
`README.md`, no automatic `git init`. Larger templates are a
follow-up minor.

### `tya task`

Define recurring commands once in `tya.toml`:

```toml
[tasks]
ci      = "tya format && tya test"
release = ["tya build", "git tag v$(date +%Y%m%d)", "git push --tags"]
```

Run them by name:

```sh
$ tya task ci             # runs the string verbatim under /bin/sh -c
$ tya task release        # runs each array entry in order, stops on first failure
$ tya task ci -- --watch  # extra args are POSIX-quoted and appended
$ tya task                # lists every task in source order
```

Failure of an array-form task points at the failing entry:

```
[TYA-E0901] task "release" command #2 ("git tag v20260512") failed with exit code 128
```

### `tya lint`

Run the linter over a file or directory. Findings go to stdout
in `path:line:col: TYAL0001 unused local "name"` form, sorted
by path/line/column. Exit 1 when any finding is reported, 0 when
clean.

```sh
$ tya lint src
src/foo.tya:12:3: TYAL0001 unused local "tmp"
$ echo $?
1
```

The v0.49 rule set is exactly one rule. Additional rules
(`if true` / dead code after `return` / suspicious `for` index
patterns / `--fix` autofix) are a follow-up minor — the v0.49
scope is deliberately narrow so the CLI/output/exit-code contract
can settle before the rule pipeline grows.

## Implementation notes

- New diagnostic code range `TYA-E090x` is reserved for the
  three new subcommands (see SPEC § Diagnostic code registry).
- The `[tasks]` table is additive on top of the existing
  `tya.toml` manifest schema. v0.48 manifests parse unchanged.
- The manifest-discovery helper (walk up from `pwd` looking for
  `tya.toml`) was extracted into `pkg.FindManifest` and shared
  between the CLI's existing `projectRoot` and the new `tya
  task` subcommand.
- `internal/checker.CollectUnused(prog)` is a new public API on
  top of the existing scope walker; it returns every unused
  binding instead of the first one (which `CheckUnused` still
  returns for `--check-unused` strict mode).

## Compatibility

- **Language**: unchanged from v0.48.
- **Manifest schema**: backwards compatible — `[tasks]` is a new
  optional table.
- **CLI**: three new subcommands. No existing subcommands changed
  semantics. `tya version` reports `0.49.0`.

## Build environment fix

The compiled-program build path now passes `-lm` to `cc` on Linux
and other non-Windows hosts so that math intrinsics (`log2`,
`exp`, `sin`, `cos`, `atan2`, …) link successfully against glibc
out of the box. Likewise `runtime/tya_runtime.c` now defines
`_XOPEN_SOURCE` and `_DEFAULT_SOURCE` before any system header so
`strptime` and friends are visible on strict glibc defaults
(observed on recent Arch Linux). Both knobs are no-ops on macOS
where the host libc already exposes these symbols.

## Migration

Nothing to do. Optionally:

1. Add a `[tasks]` table to your project's `tya.toml`.
2. Run `tya lint` on existing sources and clean up any unused
   locals it reports (or ignore — exit 1 is informational, not
   blocking, unless wired into CI).

## Looking ahead

v0.50 (or later) candidates from `ROADMAP.md` § Future Work §
Toolchain:

- Additional lint rules + `--fix` autofix
- `tya doc` source documentation generator
- `tya lsp` Language Server
- diagnostics pipeline migration (Parser → `TYA-E01xx`,
  Codegen → `TYA-E06xx`, Runner → `TYA-E08xx`)
- public Tya self-introspection library

Self-host v02 work (`ROADMAP.md` § Scheduled M8/M9/M10) remains
deferred to the v1.0.0 prep window.
