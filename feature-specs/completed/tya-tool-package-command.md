---
status: completed
goal_ready: false
---

# Feature: `tya tool` Package Command Runner

## Goal

Add an `npx`-style `tya tool` subcommand that runs a command provided by a Tya
package without requiring users to manually install or wire that package into
their project tasks.

## Context

Tya's v1.0 direction includes an all-in-one toolchain with package tooling.
The current package manager already has:

- `tya.toml`
- `tya.lock`
- `tya install`, `tya update`, `tya add`, `tya remove`, `tya outdated`
- git and path dependency sources
- per-project package materialization under `.tya/packages/`

The runner already has `tya run <file.tya> [args...]`, which executes a local
script file. The task runner already has `tya task [name] [args...]`, which
executes project-defined shell commands. Neither name clearly fits the use case
where a user wants to discover and run a package-provided tool once, or pin a
tool dependency and invoke it consistently across machines.

Tya does not yet define a central package registry, so the first version should
not depend on registry lookup. It should build on git/path dependencies and the
existing manifest/lockfile model.

The command is named `tya tool` instead of `tya exec` so the CLI reads as:

- `tya run`: run a local Tya script file.
- `tya task`: run a project task from `tya.toml`.
- `tya tool`: run a package-provided tool.

## Behavior

- Add `tya tool <command> [args...]`.
- `tya tool` first looks for `<command>` in tool declarations from the current
  project's locked dependencies.
- Package tools are declared in `tya.toml` by the package that provides them:

```toml
[tools]
format_docs = "tools/format_docs.tya"
```

- Tool entry paths are relative to the package root and must point to lowercase
  script files, not PascalCase class files.
- A consuming project can pin a tool package through normal dependencies or
  dev-dependencies, then run:

```sh
tya tool format_docs --check
```

- `tya tool` runs the selected Tya script with the same program execution path
  as `tya run`, forwarding stdin,
  stdout, stderr, process arguments, and exit status.
- The command runs with the current working directory set to the invoking
  project root.
- `tya tool` requires a `tya.toml` project by default.
- If `tya.lock` is missing or stale, `tya tool` fails with a diagnostic telling
  the user to run `tya install`.
- If multiple locked packages expose the same command name, `tya tool` fails
  with a command-name conflict diagnostic and lists the packages.
- A fully qualified form disambiguates conflicts:

```sh
tya tool package_name:format_docs --check
```

- Add a one-shot source form for packages not yet added to the manifest:

```sh
tya tool --git https://github.com/example/tya-tools --tag v1.2.0 format_docs
tya tool --path ../tya-tools format_docs
```

- One-shot source form resolves into an isolated cache under `.tya/cache/exec/`
  and does not edit `tya.toml` or `tya.lock`.
- One-shot git form requires `--tag` or `--rev`; branch execution is rejected in
  the first version to avoid unpinned remote code execution.
- `tya tool --offline <command>` only uses already materialized packages/cache
  and never fetches.
- `tya tool --list` prints available commands from locked dependencies.

## Scope

- `cmd/tya/main.go`
- new command implementation under `cmd/tya/`
- `internal/pkg/manifest.go`
- `internal/pkg/manager.go` or adjacent package source helpers for one-shot
  git/path resolution
- `docs/TERMINOLOGY.md`
- `docs/VERIFICATION.md` if exit-code conventions need an entry
- next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`
- script tests under `tests/testdata/`
- `ROADMAP.md`

## Dependencies

- Implement `feature-specs/stdlib-cli-library.md` first if `tya tool` will use the
  public `cli.Cli` parser for option handling.

## Out of Scope

- Central package registry lookup.
- Global installs.
- Executing tools from unpinned git branches.
- Executing non-Tya binaries from packages.
- Sandboxing package code beyond normal process isolation.
- Windows shell integration beyond what `tya run` already supports.
- Changing `tya run`.
- Changing `tya task`; tasks may call `tya tool`, but they remain
  project-local shell commands.

## Acceptance Criteria

- A package declaring `[tools] hello = "tools/hello.tya"` can be added as a
  path dependency and run with `tya tool hello`.
- A package declaring the same tool through a git tag dependency can be run
  after `tya install`.
- `tya tool hello --name komagata` forwards arguments to the tool script.
- The tool sees the invoking project root as its current working directory.
- The tool's stdout, stderr, stdin, and exit code are preserved.
- `tya tool --list` shows tools from locked dependencies in deterministic
  order.
- Two dependencies exporting the same tool name fail with a clear conflict
  diagnostic.
- `tya tool package_name:tool_name` resolves a conflict deterministically.
- One-shot `--path` execution works without editing `tya.toml` or `tya.lock`.
- One-shot `--git --tag` execution works through `.tya/cache/exec/`.
- One-shot `--git --branch` is rejected with an actionable diagnostic.
- Missing or stale lockfiles tell the user to run `tya install`.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run TestV.*Script -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Open Questions

None.
