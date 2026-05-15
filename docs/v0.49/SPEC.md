---
layout: doc
title: Spec
permalink: /v0.49/spec/
---

# Tya v0.49 Specification

> **Status:** shipped. The `tya version` constant is `0.49.0`.
> v0.49 ships three independent toolchain additions:
>
> - **`tya new`** — project scaffolder
> - **`tya task`** — project task runner
> - **`tya lint`** — source linter (initial rule: `TYAL0001` unused locals)

## Theme

v0.48 closed the class-member surface arc. v0.49 starts the
"Toolchain" track from `ROADMAP.md` § Future Work, picking up the
three smallest end-user tools that compose well into a single
release. All three are CLI subcommands of the existing `tya`
binary; no new dependencies are added.

The language surface is unchanged from v0.48. Existing programs
keep compiling without modification.

## `tya new` — project scaffolder

CLI:

```
tya new <name>
```

Creates a new directory `<name>` in the current working directory
and writes a minimal project scaffold:

```
<name>/
  tya.toml      — name, version, sample [tasks] table
  src/main.tya  — hello-world entry point
  .gitignore    — .tya/ and dist/
```

`tya.toml` is generated with:

```toml
name = "<name>"
version = "0.1.0"

[tasks]
run = "tya run src/main.tya"
```

`src/main.tya`:

```tya
print("Hello, Tya!")
```

`.gitignore`:

```
.tya/
dist/
```

### Errors and exit codes

| Exit | When |
|------|------|
| 0 | Scaffold created |
| 1 | Validation or I/O error (see below) |

Diagnostic codes:

- `[TYA-E0910]` — invalid project name. Names containing the path
  separator are rejected.
- `[TYA-E0911]` — target directory already exists. Refuse to
  overwrite.

### Scope-out (deferred)

The following flags from `ROADMAP.md` § Future Work § `tya new`
are intentionally not in v0.49 and may land in a follow-up minor
when the need is concrete:

- `--here` (initialize the current directory)
- `--template app|lib`
- `--force` (overwrite an existing directory)
- Automatic `git init` and `--no-git`
- `tests/` boilerplate with a passing unittest
- `README.md` boilerplate

## `tya task` — project task runner

CLI:

```
tya task                       # list every entry under [tasks]
tya task <name>                # run the named task
tya task <name> <args...>      # run the named task with extra args appended
```

### Configuration

A new `[tasks]` table in `tya.toml`. Each key is a task name; each
value is either a single string (run that command verbatim) or an
array of strings (run each command in order, stopping on the first
failure).

```toml
[tasks]
ci      = "tya format && tya test"
release = ["tya build", "git tag v1.0.0", "git push --tags"]
```

The `[tasks]` table is additive on top of the existing manifest
shape; v0.48 manifests parse unchanged in v0.49.

### Execution model

- **Shell**: every command is passed to `/bin/sh -c "<command>"`.
  POSIX `sh` (not bash) is the contract; bash-specific syntax
  must be wrapped explicitly (`bash -c "..."`). This keeps task
  behavior stable across local and CI environments.
- **CWD**: the project root — the directory containing the
  resolved `tya.toml`. tya walks up from the current working
  directory to locate it.
- **stdin / stdout / stderr**: inherited from the parent.
- **Environment**: inherited from the parent. v0.49 has no
  per-task env mechanism.

### Argument passthrough

Extra arguments after `<name>` are POSIX-shell-quoted and appended
to the command, mirroring `$@` in shell scripts. Each argument is
wrapped in single quotes, with any internal `'` replaced by the
four-character sequence `'\''`.

```
$ tya task greet world peace      # runs:  echo hi 'world' 'peace'
$ tya task lint --fix             # runs:  tya lint '--fix'
```

### Array-form failure

For an array-form task, tya runs each entry under its own
`/bin/sh -c`. If a command exits non-zero, tya stops, reports the
failing entry, and propagates the child's exit code.

```
$ tya task fail
first
[TYA-E0901] task "fail" command #2 ("exit 7") failed with exit code 7
$ echo $?
7
```

### Errors and exit codes

| Exit | When |
|------|------|
| 0 | Task succeeded |
| Non-zero | Task's child process exit code (string form), or the failing array entry's exit code |

Diagnostic codes:

- `[TYA-E0900]` — `tya task <name>` for a name not in the manifest.
- `[TYA-E0901]` — array-form task: one entry failed. Includes the
  1-origin index, the original command string, and the exit code.
- `[TYA-E0902]` — no `tya.toml` found walking up from the current
  directory.

### Listing

`tya task` with no arguments prints each registered task in source
order on its own line, formatted as `<name>\t<summary>` where the
summary is the verbatim command for a string-form task or the
elements joined with ` && ` for an array-form task. Long summaries
are truncated to 80 characters with `…` as the truncation marker.

### Scope-out (deferred)

The following items from `ROADMAP.md` § Future Work § `tya task`
are intentionally not in v0.49:

- Parallel execution (the `[tasks]` Array form is fixed to
  "sequential, stop on first failure"; parallel execution is left
  to a future dedicated syntax)
- File watching
- Task dependency graphs
- Per-task environment variables (`[tasks.env]` or similar)

## `tya lint` — source linter

CLI:

```
tya lint [paths...]
```

Each path is either a `.tya` file or a directory. Directories are
walked recursively, picking up every file ending in `.tya`. With
no paths, lint defaults to the current directory.

### TYAL0001 — unused local

The single v0.49 rule. A binding is "unused" if it is introduced
as a local variable or function parameter and never read in its
scope. Sub-scopes (`if`/`while`/`for`/`try`/lambda bodies) are
checked recursively. The `_` placeholder is exempt by convention.

Finding format (one per line, sorted by path/line/column):

```
<path>:<line>:<col>: TYAL0001 unused local "<name>"
```

### Errors and exit codes

| Exit | When |
|------|------|
| 0 | No findings |
| 1 | At least one finding was reported |
| 2 | Argument or I/O error (missing path, parse error, etc.) |

### Scope-out (deferred)

The following items from `ROADMAP.md` § Future Work § `tya lint`
are intentionally not in v0.49 and will land as additional rules
or features in follow-up minors:

- `--fix` autofix mode
- `--format=json` output
- Additional rules (`if true`/`if false`, dead code after
  `return`/`raise`, suspicious `for` index patterns, deeply
  nested blocks, very long functions)
- Per-line opt-out via `# tya-lint-ignore: TYAL0001`
- Rule documentation URLs

## Diagnostic code registry

v0.49 reserves the `TYA-E090x` range for CLI/task-runner errors,
distinct from the runner range `TYA-E0800–E0899` that other
Future Work items will use:

| Code | Subcommand | Meaning |
|------|------------|---------|
| TYA-E0900 | `tya task` | Unknown task name |
| TYA-E0901 | `tya task` | Array-form task: one entry failed |
| TYA-E0902 | `tya task` | No `tya.toml` found |
| TYA-E0910 | `tya new`  | Invalid project name |
| TYA-E0911 | `tya new`  | Target directory already exists |
| TYAL0001  | `tya lint` | Unused local binding |
