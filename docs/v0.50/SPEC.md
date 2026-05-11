# Tya v0.50 Specification

> **Status:** shipped. The `tya version` constant is `0.50.0`.
> v0.50 extends the three toolchain subcommands introduced in v0.49
> (`tya new`, `tya task`, `tya lint`) so they are usable on a real
> project.

## Theme

v0.49 cut the first slice of the Toolchain track. v0.50 extends
all three subcommands without adding new ones, focusing on the
flags and behaviors that turn the v0.49 minimal shapes into a
practical workflow.

The language surface is unchanged from v0.49. Existing programs
keep compiling.

## `tya lint` — autofix + 3 new rules

### CLI

```
tya lint [--fix] [paths...]
```

`--fix` applies autofixable findings in place. v0.50 ships autofix
for `TYAL0001 unused local` only (line removal); other rules are
warning-only.

### Rules

| Code | Description | Autofix in v0.50 |
|------|-------------|-------------------|
| TYAL0001 | unused local binding (v0.49) | yes (line removal) |
| TYAL0003 | redundant `if true` / `if false` | no (queued for v0.51+) |
| TYAL0004 | deeply nested blocks (depth ≥ 5) | no |
| TYAL0005 | very long functions (body > 50 statements) | no |

Thresholds are hardcoded constants in
`internal/checker/lint_rules.go`:
`nestingThreshold = 5`, `longFunctionThreshold = 50`. Threshold
configuration via `tya.toml [lint]` is queued for v0.51+.

`TYAL0002` is reserved (originally planned for dead code after
`return` / `raise`; deferred so the autofix surface stays small in
v0.50).

### Output format

Unchanged from v0.49:

```
<path>:<line>:<col>: <code> <message>
```

Sorted by path/line/column. Findings stream to stdout. tya exits 1
when any non-autofixed finding remains; 0 when clean; 2 on
argument or I/O errors.

### Out of scope (v0.50)

- Per-line opt-out (`# tya-lint-ignore: TYAL0001`). Aligns with Go
  culture: prefer fixing flagged code over suppressing it. May
  ship later if a concrete need appears.
- `--format=json` output.
- AST-rewriting autofix for TYAL0003 (currently requires safe
  `formatter.Unparse` round-trip; deferred to v0.51+).

## `tya new` — flags + templates

### CLI

```
tya new [flags] <name>
tya new --here [flags]
```

Flags:

| Flag | Default | Effect |
|------|---------|--------|
| `--here` | off | initialise the current directory (no `<name>`) |
| `--template app\|lib` | `app` | scaffold variant |
| `--force` | off | overwrite an existing target directory |
| `--no-git` | off (git init runs) | skip the automatic `git init` |

### Scaffold contents

Both templates write `tya.toml` (with `[tasks]` sample entries
`run` and `test`), `.gitignore`, and a short `README.md`.

`app` template (default):

```
<name>/
  tya.toml
  src/main.tya       # print("Hello, Tya!")
  tests/main_test.tya
  .gitignore
  README.md
```

`lib` template:

```
<name>/
  tya.toml
  src/<PascalName>.tya   # class skeleton
  tests/<pascalname>_test.tya
  .gitignore
  README.md
```

`<PascalName>` is `<name>` rewritten to PascalCase, splitting on
`-`, `_`, and `.` and capitalising the first letter of each part
(`my-lib` → `MyLib`).

### git init

Runs `git init --quiet <target>` after writing the scaffold. If
`git` is missing or fails, prints a warning to stderr and proceeds
without aborting (exit 0). `--no-git` skips the call entirely.

### Errors and exit codes

| Code | When |
|------|------|
| TYA-E0910 | invalid project name (contains a path separator) |
| TYA-E0911 | target directory already exists (and `--force` not given) |
| TYA-E0912 | invalid `--template` value |
| TYA-E0913 | `--here` combined with a target name |

## `tya task` — parallel execution

### New TOML form

A task can be declared as a table with a `cmds` array and an
optional `parallel = true` flag. The string and array forms from
v0.49 are unchanged.

```toml
[tasks]
ci      = "tya format && tya test"           # string form (sequential single)
release = ["tya build", "git push --tags"]   # array form (sequential)

[tasks.watch]                                # table form
cmds     = ["tya format --watch", "tya lsp"]
parallel = true                              # if false or absent, behaves like array form
```

### Execution model

When `parallel = true`:

- Every `cmds` entry runs under its own `/bin/sh -c` concurrently.
- tya waits for **all** child processes (case B). It does not
  SIGTERM siblings when one fails.
- Each line of every child's stdout/stderr is prefixed with
  `[<index> <truncated cmd>] ` and streamed to the parent's
  stdout/stderr. `<index>` is 1-origin; the command is truncated
  to 16 characters with `…` as the truncation marker.
- After all children finish, if any exited non-zero, tya itself
  exits with the **first** non-zero exit code observed (defined
  as: smallest index among the failing commands).
- The final stderr line is `[TYA-E0903] task "<name>" parallel:
  <failed list>` where `<failed list>` enumerates every failure
  as `#<idx> ("<cmd>") exit <code>`.

When `parallel = false` (or absent): identical to the v0.49 array
form (sequential, stop on first failure, TYA-E0901 on failure).

### Argument passthrough

Unchanged from v0.49: extra args after `<name>` are POSIX-quoted
and appended to **every** command in the task (parallel form
included).

### Out of scope (v0.50)

- Task dependency graph (`depends-on = [...]`).
- `--watch` driver.
- Per-task environment variables.
- TTY-aware coloured prefixes (always plain text in v0.50).

## Diagnostic code registry update

v0.50 adds two codes; the rest of the `TYA-E090x` range continues
to belong to the toolchain subcommands.

| Code | Subcommand | Meaning |
|------|------------|---------|
| TYA-E0900 | `tya task` | unknown task name (v0.49) |
| TYA-E0901 | `tya task` | array-form task: one entry failed (v0.49) |
| TYA-E0902 | `tya task` | no `tya.toml` found (v0.49) |
| TYA-E0903 | `tya task` | parallel-form task: one or more cmds failed (v0.50) |
| TYA-E0910 | `tya new`  | invalid project name (v0.49) |
| TYA-E0911 | `tya new`  | target directory already exists (v0.49) |
| TYA-E0912 | `tya new`  | invalid `--template` value (v0.50) |
| TYA-E0913 | `tya new`  | `--here` and target name conflict (v0.50) |
| TYAL0001  | `tya lint` | unused local (v0.49; autofix in v0.50) |
| TYAL0003  | `tya lint` | redundant `if true` / `if false` (v0.50) |
| TYAL0004  | `tya lint` | deeply nested block (v0.50) |
| TYAL0005  | `tya lint` | very long function (v0.50) |

`TYAL0002` is reserved.

## Compatibility

- Language surface unchanged from v0.49 (and v0.48 before it).
- `tya.toml` schema: backwards compatible. v0.49 manifests parse
  unchanged in v0.50 (new table form is purely additive).
- CLI: no existing subcommand changed shape. v0.49 manifests/usage
  continue to work.
