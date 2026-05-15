---
layout: doc
title: Spec
permalink: /v0.55/spec/
---

# Tya v0.55 Specification

> **Status:** shipped. The `tya version` constant is `0.55.0`.
> v0.55 extends `tya lint` with per-line opt-out comments, a
> machine-readable JSON output, a new `TYAL0002` dead-code rule,
> and an AST-level autofix for `TYAL0003`. The language surface
> is unchanged from v0.54.

## Theme

v0.49 shipped `tya lint` as a real subcommand. v0.50 added line-
deleting `--fix` for `TYAL0001` and warned on `TYAL0003 /
TYAL0004 / TYAL0005`. v0.55 turns the lint into a CI-ready tool:
finer-grained suppression, structured output, more analysis, and
more autofix coverage.

## CLI

```
tya lint [--fix] [--format=text|json] [paths...]
```

- `--fix` rewrites the source in place. v0.55 applies two passes:
  1. **`TYAL0003` unwrap-if** — `if true` / `if false` blocks are
     unwrapped: the header line is dropped and the body is
     de-indented by two spaces.
  2. **`TYAL0001` line-delete** (existing since v0.50) — lines
     introducing unused locals are removed.
- `--format=text` (default) prints `path:line:col: CODE message`
  one per line, matching v0.50.
- `--format=json` emits a single JSON object with a `findings`
  array (schema below).
- Exit codes (unchanged): `0` clean / `1` findings remain / `2`
  argument or I/O error.

The `lint` subcommand owns its own `--format` flag. Other
subcommands continue to recognise the global
`--format=human|json` for diagnostic rendering.

## Per-line opt-out

Comments of the form `# tya-lint-ignore[: CODE[, CODE...]]`
suppress matching findings.

- `# tya-lint-ignore` (no code list) suppresses every code on
  the target line.
- `# tya-lint-ignore: TYAL0001` suppresses just the listed codes.
- Inline comments (`x = 1  # tya-lint-ignore: TYAL0001`) target
  the line they sit on.
- Full-line comments target the **following** statement (the
  next source line).
- Codes are case-sensitive. Surrounding whitespace is ignored.

When `--fix` deletes a line, any opt-out comment on that line
is removed with it (line-delete drops the whole line verbatim).

## JSON output schema

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

- `version` mirrors the `tya version` constant. Consumers that
  parse the report should treat unknown fields as forward-
  compatible additions.
- `findings` is always an array (possibly empty). It is sorted
  by `(path, line, col, code)`.
- `autofixable` is `true` for `TYAL0001` and `TYAL0003`; `false`
  for `TYAL0002`, `TYAL0004`, and `TYAL0005`.

## Lint rules

| Code      | Trigger                              | Autofix       |
|-----------|--------------------------------------|---------------|
| `TYAL0001`| unused local                          | line-delete   |
| `TYAL0002`| **dead code after `return` / `raise`**| —             |
| `TYAL0003`| redundant `if true` / `if false`     | **unwrap-if** |
| `TYAL0004`| deeply nested block (depth ≥ 5)      | —             |
| `TYAL0005`| function body > 50 statements        | —             |

### TYAL0002 dead code after `return` / `raise`

Inside any block (function body, `if` arm, `while` body, `for`
body, `try` / `catch`, `match` case), once a `return` or `raise`
statement is reached every subsequent statement at the same
nesting level emits `TYAL0002 dead code after <return|raise>`
pointing at the unreachable statement.

The first unreachable statement is reported; the rule then
continues so consecutive dead statements each yield one finding.

### TYAL0003 AST autofix

`tya lint --fix` rewrites the source so that:

- `if true` with a non-empty `Then` body: the `if true` header
  line is removed, and the body is de-indented by two spaces.
- `if false` with a non-empty `Else` body: the `if false`
  header is removed and the `else:` body is de-indented.

Lines outside the construct are left untouched. Edge cases
(empty `Then`, empty `Else`, single-line bodies on the header
line) fall back to no edit, matching the LSP code-action
behaviour from v0.51.

## Scope-out (v0.56+)

- `TYAL0004` / `TYAL0005` AST autofixes (deep-nesting flattening
  and long-function splits are human-judgement calls).
- `--format=sarif` (SARIF v2.1 standard).
- File-scope opt-out (`# tya-lint-ignore-file: TYAL0001`).
- Unifying the LSP code-action `unwrap-if` helper with the CLI
  `applyUnwrapIf` so both call the same rewriter.
- Additional rules (`suspicious for loop`, `unused param`, etc.).
