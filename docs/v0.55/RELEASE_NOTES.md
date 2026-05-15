---
layout: doc
title: Release Notes
permalink: /v0.55/release-notes/
---

# Tya v0.55 Release Notes

> **Status:** shipped. `tya version` reports `0.55.0` and
> `ROADMAP.md` carries the matching `Released` entry.

## TL;DR

v0.55 makes `tya lint` ready for CI use:

- Per-line `# tya-lint-ignore` opt-out comments.
- `--format=json` machine-readable output.
- New `TYAL0002` rule: dead code after `return` / `raise`.
- New autofix: `--fix` now unwraps redundant `if true` /
  `if false` blocks in place.

The language surface is unchanged from v0.54.

## What's new

### Per-line opt-out comments

```tya
# tya-lint-ignore: TYAL0001
tmp = 42                       # silences the unused-local finding

guard = false  # tya-lint-ignore   # silences every finding on this line
```

`# tya-lint-ignore` with no code list is a wildcard; a comma-
separated `: CODE` list scopes the suppression. Inline comments
target the same line, full-line comments target the next
statement.

### `--format=json`

```sh
$ tya lint --format=json src/
```

```json
{
  "version": "0.55.0",
  "findings": [
    {
      "path": "src/foo.tya",
      "line": 12,
      "col": 3,
      "code": "TYAL0001",
      "message": "unused local \"tmp\"",
      "autofixable": true
    }
  ]
}
```

Findings are sorted by `(path, line, col, code)`. Empty runs
emit `"findings": []`. Exit codes are unchanged
(`0` clean / `1` findings / `2` error).

### TYAL0002 dead code after `return` / `raise`

```tya
greet = ->
  return 1
  print("unreachable")   # TYAL0002 dead code after return
```

The rule fires inside every block (function bodies, `if` arms,
`while` / `for` bodies, `try` / `catch`, `match` cases). It is
warning-only (no autofix) — removing dead code is a human
judgement call.

### `--fix` autofix for `TYAL0003`

```tya
# before
if true
  print("hi")

# after `tya lint --fix`
print("hi")
```

`--fix` runs in two passes: first `TYAL0003` unwrap-if (which
shifts source positions), then `TYAL0001` line-delete on the
re-lexed source. Each pass writes the file in place.

## Compatibility

- The text output format (`path:line:col: CODE message`) is
  unchanged. Existing CI pipelines that grep `tya lint` output
  continue to work.
- The global `--format=human|json` for diagnostics is unchanged.
  Only the `lint` subcommand reads its own `--format` flag.
- LSP code action `unwrap-if` (v0.51) continues to apply its
  own edit; the CLI now performs an equivalent rewrite when
  `--fix` is supplied.

## Tooling

- `Formula/tya.rb` → `0.55.0`.
- `editors/vscode/package.json` → `0.55.0`.
- New fixtures in `tests/testdata/v55_lint/` covering opt-out,
  dead-code, `--fix` unwrap, and JSON output.

## Next

- `TYAL0004` / `TYAL0005` autofix (deep nesting and long-function
  splits remain warning-only; the right reshape is human-judged).
- `--format=sarif` for tools that ingest SARIF v2.1 directly.
- File-scope opt-out (`# tya-lint-ignore-file`).
- Other Toolchain Epic items (`tya task` v2, `tya doc` v2).
