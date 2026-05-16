---
status: completed
goal_ready: false
---

# Feature: CLI Stdlib Library

## Goal

Add a `cli` standard library package for parsing command-line flags and
rendering usage text, so Tya programs and package tools can build predictable
CLIs without hand-written argument parsing.

## Context

Tya already exposes raw process arguments through builtins and `os.args()`.
That is enough for tiny scripts, but package tools and scaffolded CLI apps need
common behavior: boolean flags, string/int options, repeated values, positional
arguments, defaults, required options, `--` handling, and generated help text.

This belongs in stdlib rather than the language. It also supports the planned
`tya tool` package command runner.

## Behavior

- Add `stdlib/cli/Cli.tya`.
- `cli.Cli.parse(args, spec)` returns a dictionary with:
  - `options`
  - `positionals`
  - `rest`
  - `errors`
- `spec` is a dictionary with `options` and optional `positionals`,
  `allow_unknown`, and `stop_at_double_dash`.
- Supported option types:
  - `bool`
  - `string`
  - `int`
  - `float`
  - `array`
- Long flags support `--name value` and `--name=value`.
- Boolean flags support `--flag` and `--no-flag`.
- Short aliases support `-v`, `-o value`, and grouped bool aliases such as
  `-abc`.
- `--` stops option parsing and places remaining args in `rest`.
- Defaults are applied when an option is absent.
- Required options produce structured parse errors.
- Unknown options produce parse errors unless `allow_unknown` is true.
- `cli.Cli.usage(command, spec)` returns deterministic usage text.
- `cli.Cli.parse_or_exit(args, spec)` prints usage/errors and exits non-zero.

## Scope

- `stdlib/cli/Cli.tya`
- `docs/STDLIB.md`
- next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`
- stdlib tests and/or script tests
- `ROADMAP.md`

## Out of Scope

- Interactive prompts.
- Terminal styling/colors.
- Shell completion generation.
- Subcommand dispatch beyond parsing the first positional.
- Environment-variable binding.

## Acceptance Criteria

- `import cli` exposes `cli.Cli`.
- Long, short, grouped boolean, and `--name=value` forms parse correctly.
- Defaults and required errors work.
- `--` handling preserves rest arguments.
- `usage` output is deterministic and includes aliases, defaults, and required
  markers.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Open Questions

None.
