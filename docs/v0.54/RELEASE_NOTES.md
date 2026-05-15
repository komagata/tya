---
layout: doc
title: Release Notes
permalink: /v0.54/release-notes/
---

# Tya v0.54 Release Notes

> **Status:** shipped. `tya version` reports `0.54.0` and
> `ROADMAP.md` carries the matching `Released` entry.

## TL;DR

v0.54 finishes the diagnostics pipeline migration:

- Parser, Codegen, and Runner errors now flow through the same
  `diag.Diagnostic` channel as Lexer and Checker — every error
  carries a stable `[TYA-EXXXX]` code.
- The parser keeps going past one error, so a single
  `tya check` run surfaces every top-level statement violation.
- Undefined-name errors gain `did you mean "…"?` suggestions
  driven by a tiny new `internal/util` Levenshtein helper.

The language surface is unchanged from v0.53.

## What's new

### Structured diagnostics for Parser / Codegen / Runner

```
$ tya check bad.tya
-- RETURN MUST BE INSIDE A FUNCTION ---- bad.tya:1:1

return must be inside a function near "return"

   1 | return 42
       ^

(TYA-E0120)

-- BREAK MUST BE INSIDE A LOOP -------- bad.tya:2:1
…
```

New code bands:

- `TYA-E0100-0199` Parser (expected token, indented block,
  position constraint, reserved name, pattern syntax, expression).
- `TYA-E0601-0606` Codegen (unsupported AST shapes).
- `TYA-E0840-0858` Runner (filename, entry-module conflict,
  import name conflict, undefined variable).

### Multi-error parsing

`block()` and `program()` now run statement-level recovery:
after recording a diagnostic, the parser skips to NEWLINE /
DEDENT / EOF and resumes. Every diagnostic flows out as a
`*parser.ParserError{Diags: []diag.Diagnostic}`.

The LSP server (`internal/lsp/diagnostics.go`) and CLI
(`cmd/tya/main.go`) both `errors.As`-unwrap the wrapper and emit
the full list.

### did-you-mean hints

```
$ tya check typo.tya
typo.tya:2:7: undefined variable strng; did you mean "string"?
```

The new `internal/util/strdist.go::Suggest` returns up to three
candidates with Levenshtein distance ≤ 2, drawing from the
builtin function list and the current scope's defined names.

## Compatibility

- Language surface: unchanged from v0.53.
- `tya.toml` schema: unchanged.
- CLI: every existing subcommand keeps its v0.53 behaviour.
  Stderr formatting changes are additive (banner + code) and
  existing tests that grep substring like `"undefined variable"`
  continue to match.

## Migration

Nothing required. Optional:

1. Upgrade to v0.54 (`brew install komagata/tap/tya`).
2. Re-run `tya check` on existing source — you'll see every
   diagnostic in one pass instead of just the first.
3. Consume the new `[TYA-EXXXX]` codes in your editor / CI lint
   rules.

## Implementation notes

- New helpers:
  - `internal/util/strdist.go` — Levenshtein + `Suggest`
  - `internal/parser/{codes,classify,diagnostic,recovery}.go` —
    structured errors + multi-error scaffolding
  - `internal/codegen/errors.go` — `CodegenError` wrapper
  - `internal/runner/errors.go` — `RunnerError` + `undefinedNameHint`
  - `internal/checker/suggest.go` — checker-side
    undefined-name hint formatting
- The parser keeps the legacy `line:col: msg near "tok"` format
  on `*ParserError.Error()` so existing string-grep tests stay
  green; structured payload lifts out via `errors.As`.
- LSP and CLI both unwrap `*parser.ParserError`,
  `*runner.RunnerError`, and `*codegen.CodegenError` into the
  shared `emitDiagnostics` renderer.

## Tests

- `tests/testdata/v54_diag/multi_error.txtar` — three top-level
  position constraint violations surface in one pass.
- `tests/testdata/v54_diag/did_you_mean.txtar` — typo'd
  identifier gets the `did you mean …?` hint.
- All pre-existing tests stay green.

## Looking ahead (v0.55+ candidates)

From `ROADMAP.md` § Future Work:

- Parser signature change (`Parse() → (*Program, []Diag, error)`)
- Expression-level recovery
- Codegen / Runner signature change for `[]Diag` slices
- LSP code actions for did-you-mean quick fixes
- VS Code Marketplace publication (still queued from v0.53)
- Toolchain track other Epics (`tya lint` v3, `tya task` v2,
  `tya doc` v2)
- Language Epics (raw `"` in `{expr}` interpolation,
  primitive class sugar, interface stackable trait, WASM target)

Self-host work (`ROADMAP.md` § Scheduled M8 / M9 / M10) remains
deferred until the v1.0.0 prep window.
