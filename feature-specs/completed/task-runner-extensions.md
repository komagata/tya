# Feature: Task Runner Extensions

## Goal

Polish `tya task` so project tasks can model real development workflows with dependencies, per-task environment variables, and file watching while preserving the existing string, array, and parallel task forms.

## Context

- `ROADMAP.md` tracks **Task runner extensions** as polish, not a v1.0.0 blocker.
- Existing `tya task` behavior:
  - discovers `tya.toml` by walking up from the current directory
  - lists `[tasks]` entries with no task name
  - runs string tasks under `/bin/sh -c`
  - runs array tasks sequentially and stops on the first failure
  - supports table-form parallel tasks with `cmds = [...]` and `parallel = true`
  - appends extra CLI args to task commands using POSIX shell quoting
- ROADMAP still lists parallel execution syntax, but the current implementation and tests already cover `parallel = true`. This feature should keep that behavior stable and complete the remaining task-runner polish items.

## Behavior

- Preserve existing task forms:
  - `task = "command"`
  - `task = ["command 1", "command 2"]`
  - `[tasks.name] cmds = [...]; parallel = true`
- Add dependency graphs with `depends_on`:
  - table-form tasks may declare `depends_on = ["build", "lint"]`
  - dependencies run before the requested task
  - dependencies run once per invocation even when shared by multiple downstream tasks
  - dependency order is deterministic and follows the order written in `depends_on`
  - dependency cycles fail before running any task command
  - unknown dependencies fail before running any task command
- Add per-task environment variables with `env`:
  - table-form tasks may declare `env = { KEY = "value" }`
  - task env is merged over the inherited process environment
  - extra CLI args do not affect env values
  - env applies to every command in the selected task, including sequential and parallel commands
  - dependencies use their own `env`, not the env of the downstream task
- Add file watching mode:
  - `tya task <name> --watch` runs the task once, then reruns it when project files change
  - the default watch set includes `.tya`, `tya.toml`, and files under `src/`, `tests/`, `stdlib/`, and `examples/` when those paths exist under the project root
  - changes under `.git/`, `node_modules/`, `_site/`, build output directories, and hidden cache directories are ignored
  - rapid consecutive changes are debounced
  - while a task run is active, new changes schedule exactly one rerun after the active run finishes
  - `Ctrl-C` exits cleanly and terminates any running child process group
- Add table-form watch options:
  - `watch = ["src/**/*.tya", "tests/**/*.tya"]` to override the default watch set
  - `ignore = ["tmp/**", "dist/**"]` to add ignore globs
- CLI argument parsing:
  - `--watch` is consumed by `tya task`, not appended to the task command
  - arguments after `--` are passed through to the task command even if they look like flags
  - existing `tya task lint --fix` behavior remains backward compatible, so unknown flags after the task name continue to be treated as task args unless they are recognized task-runner flags before `--`
- Diagnostics:
  - keep existing `TYA-E0900` through `TYA-E0903`
  - add stable codes for dependency cycle, unknown dependency, invalid env table, invalid watch pattern, and watch runtime failure
  - errors should include the task name and the offending dependency/key/pattern

## Scope

- Update task manifest parsing in `internal/pkg/manifest.go`.
- Update task execution in `cmd/tya/task.go`.
- Add any small internal watcher helper needed by `tya task --watch`.
- Add focused testscript fixtures under `tests/testdata/v49_task/`, `tests/testdata/v50_task/`, or a newer task directory for:
  - dependency order
  - shared dependency runs once
  - dependency cycles
  - unknown dependencies
  - per-task env
  - env isolation between dependencies and downstream tasks
  - `--watch` flag parsing
  - watch reruns on a changed `.tya` file
  - ignored watch paths do not rerun
  - existing string/array/parallel/args behavior remains compatible
- Update `docs/SPEC.md` and `docs/ja/spec.md`.
- Update README task examples if useful.
- Update `ROADMAP.md` after implementation to mark completed subitems accurately, noting that parallel syntax was already implemented before this spec.

## Out of Scope

- A long-running daemon or persistent task server.
- A full Make/Ninja-style incremental build engine.
- Remote task execution.
- Cross-platform shell abstraction beyond the existing task shell behavior.
- Parallel dependency graph scheduling beyond existing `parallel = true` command groups.
- Rich terminal UI, progress bars, or interactive dashboards.

## Acceptance Criteria

- Existing `tya task` tests for string, array, args passthrough, cwd discovery, listing, missing task, and parallel execution still pass.
- `depends_on` runs dependencies before the requested task in deterministic order.
- Shared dependencies run once.
- Dependency cycles and unknown dependencies fail before running commands and use stable diagnostics.
- `env` variables are visible to the task commands and do not leak across unrelated tasks.
- `tya task <name> --watch` consumes `--watch` as a runner flag and reruns after a watched file change.
- `tya task <name> -- --watch` passes `--watch` to the task command.
- Watch mode ignores documented ignored directories and debounces rapid changes.
- English and Japanese specs document task dependencies, env, watch mode, and the existing parallel form.

## Verification

```sh
go test ./internal/pkg -count=1
go test ./tests -run 'TestV49Scripts|TestV50Scripts' -count=1
go test ./... -count=1
```
