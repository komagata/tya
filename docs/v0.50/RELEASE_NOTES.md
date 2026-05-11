# Tya v0.50 Release Notes

> **Status:** shipped. `tya version` reports `0.50.0` and
> `ROADMAP.md` carries the matching `Released` entry.

## TL;DR

v0.49 introduced `tya new` / `tya task` / `tya lint` in minimal
form. v0.50 extends each of them so they cover a real project
workflow.

- **`tya lint`** — three new rules (`TYAL0003` `if true`/`false`,
  `TYAL0004` deep nesting, `TYAL0005` long functions) plus a
  `--fix` mode that removes unused-local lines in place.
- **`tya new`** — `--here`, `--template app|lib`, `--force`,
  `--no-git`, and **automatic `git init` on by default**.
  `app` keeps the v0.49 script template; `lib` scaffolds a class
  file plus a corresponding test.
- **`tya task`** — TOML table form `[tasks.foo] cmds = [...];
  parallel = true` runs every command concurrently, waits for all
  to finish (case B), prefixes each output line with
  `[<index> <truncated cmd>] `, and exits with the first non-zero
  child exit code.

The language surface is unchanged from v0.49.

## What's new

### `tya lint`

```sh
$ tya lint src
src/foo.tya:5:0: TYAL0003 redundant `if true`
src/foo.tya:12:0: TYAL0004 deeply nested block (depth >= 5)
src/foo.tya:42:3: TYAL0005 function body has 78 statements (> 50)
src/foo.tya:51:1: TYAL0001 unused local "tmp"

$ tya lint --fix src
# `tmp` line removed in place; TYAL0003/0004/0005 still printed
src/foo.tya:5:0: TYAL0003 redundant `if true`
src/foo.tya:12:0: TYAL0004 deeply nested block (depth >= 5)
src/foo.tya:42:3: TYAL0005 function body has 78 statements (> 50)
```

`--fix` only rewrites unused-local lines in v0.50. AST-rewriting
autofix for `if true`/`if false` is queued for v0.51+.

### `tya new`

```sh
$ tya new myapp                        # app template, git init
$ tya new --template lib --here mylib  # lib template, cwd
$ tya new --no-git --force scratch     # skip git, overwrite if exists
```

The `app` template gains `tests/main_test.tya` (a 1-test
boilerplate that uses `unittest`) and `README.md`. The `lib`
template scaffolds a class file (`src/<PascalName>.tya`) plus a
matching test file. Both templates include sample `[tasks]`
entries: `run` and `test`.

### `tya task`

```toml
[tasks.watch]
cmds     = ["tya format --watch", "tya lsp"]
parallel = true
```

```sh
$ tya task watch
[1 tya format --…] Watching src/...
[2 tya lsp]       LSP listening on stdin
```

Failure mode (case B — all wait, first non-zero exit code wins):

```
$ tya task ci-parallel
[1 tya format] ok
[3 tya test] FAIL: ...
[2 tya check] ok
[TYA-E0903] task "ci-parallel" parallel: #3 ("tya test") exit 1
$ echo $?
1
```

## Compatibility

- Language: unchanged from v0.49.
- Manifest schema: backwards compatible. The new `[tasks.<name>]`
  table form sits next to the existing string and array forms.
- CLI: every existing subcommand retains its v0.49 behavior
  unchanged. New flags are additive.

## Migration

Nothing required. Optional:

1. Add a `[tasks.<name>] parallel = true` task to your
   `tya.toml` for concurrent workflows (e.g. dev servers, watch
   processes).
2. Run `tya lint --fix` to autoclean unused locals across `src/`.
3. New projects: `tya new --template lib <name>` gives a typed
   class entrypoint instead of a bare script.

## Implementation notes

- The Task TOML schema is now a tagged union:
  `Kind ∈ {"string", "array", "parallel"}`. The Go reference
  implementation models this in
  `internal/pkg/manifest.go::Task`.
- `internal/checker.CollectLintFindings` is a new public API on
  top of the existing AST walker. It returns a flat
  `[]LintFinding` covering TYAL0003/0004/0005.
- `cmd/tya/lint.go::fixUnusedLines` does line-based source
  rewriting, not AST rewriting. This keeps comments / formatting
  intact for v0.50; richer AST-rewriting autofix lands later.
- `cmd/tya/task.go::runParallel` uses one goroutine per child
  plus two more goroutines per child for stdout/stderr line
  pumping with `bufio.Scanner`. Pipes close on `cmd.Wait()` after
  scanners drain.

## Looking ahead (v0.51+ candidates)

From `ROADMAP.md` § Future Work § Toolchain:

- `tya lint`: TYAL0002 dead code after `return`/`raise`,
  per-line opt-out, `--format=json`, threshold configuration via
  `tya.toml [lint]`, AST-rewriting autofix for TYAL0003.
- `tya new`: extra templates (web, cli), README variants.
- `tya task`: dependency graph (`depends-on`), `--watch` driver,
  per-task environment variables.
- `tya doc` and `tya lsp` MVPs (separate Epics).

Self-host work (`ROADMAP.md` § Scheduled M8/M9/M10) remains
deferred until the v1.0.0 prep window.
