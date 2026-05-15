---
layout: doc
title: Spec
permalink: /v0.54/spec/
---

# Tya v0.54 Specification

> **Status:** shipped. The `tya version` constant is `0.54.0`.
> v0.54 routes Parser / Codegen / Runner errors through the same
> structured `diag.Diagnostic` channel used by Lexer and Checker,
> adds multi-error parsing, and surfaces `did-you-mean`
> suggestions for undefined names. The language surface is
> unchanged from v0.53.

## Theme

Tya's compiler stages have historically emitted diagnostics in
two flavours: `diag.Diagnostic` for lexer + checker, and free-form
`fmt.Errorf` strings for parser, codegen, and runner. v0.54
finishes the migration so every stage speaks the same structured
shape. The visible payoff:

- Every error has a stable `[TYA-EXXXX]` code.
- The CLI renders rich blocks (banner, source pointer, code).
- The LSP receives every parse error in one pass (no more
  one-error-and-stop on `didOpen` / `didChange`).
- Undefined names get a `did you mean "…"?` hint when a close
  candidate exists.

## New diagnostic codes

Parser (`TYA-E0100-0299`):

| Code | Bucket |
|------|--------|
| `TYA-E0100` | Expected token / syntax (open-of-block, bracket, keyword) |
| `TYA-E0101` | Expected indented block |
| `TYA-E0102` | Unexpected token |
| `TYA-E0120` | Position constraint (return / break / continue / module / outside scope) |
| `TYA-E0140` | Reserved name / keyword used as identifier |
| `TYA-E0141` | Deprecated or unsupported syntax |
| `TYA-E0160` | Pattern / multi-assignment syntax |
| `TYA-E0180` | Expected expression |

Codegen (`TYA-E0600-0699`):

| Code | Meaning |
|------|---------|
| `TYA-E0601` | Unsupported assignment target |
| `TYA-E0602` | Unsupported statement or expression |
| `TYA-E0603` | Unsupported match pattern |
| `TYA-E0604` | `try` outside function body |
| `TYA-E0605` | Non-tuple multi-assignment |
| `TYA-E0606` | Destructuring target is not an Ident |

Runner (`TYA-E0800-0899`):

| Code | Meaning |
|------|---------|
| `TYA-E0840` | Invalid Tya filename |
| `TYA-E0850-0855` | (existing) module / package validation |
| `TYA-E0856` | Entry file redefines its module |
| `TYA-E0857` | Import name conflict |
| `TYA-E0858` | Undefined variable (with optional did-you-mean) |

## Multi-error parsing

The parser now keeps parsing after a statement-level failure. When
`block()` or `program()` encounters an error, the diagnostic is
recorded on the parser's accumulator, `skipToNextStmt()` advances
to the next plausible statement boundary (NEWLINE / DEDENT / EOF),
and parsing resumes. When `Parse()` returns it flushes the
accumulator into `*parser.ParserError{Diags: []diag.Diagnostic}`.

```
$ tya check three_top_level_returns.tya
-- RETURN MUST BE INSIDE A FUNCTION ---- three_top_level_returns.tya:1:1
…
(TYA-E0120)

-- BREAK MUST BE INSIDE A LOOP -------- three_top_level_returns.tya:2:1
…
(TYA-E0120)

-- CONTINUE MUST BE INSIDE A LOOP ----- three_top_level_returns.tya:3:1
…
(TYA-E0120)

Found 3 error(s), 0 warning(s).
```

Recovery is statement-level only — expression-level recovery is
queued for v0.55+ (see "Out of scope").

## Did-you-mean suggestions

Undefined-name errors (`TYA-E0858` in runner, the same surface in
checker) pass the offending name plus the candidate pool (builtin
function names + the current scope's defined names) through
`util.Suggest`. Suggestions whose Levenshtein distance to the
queried name is ≤ 2 are appended to the diagnostic as a `hint`:

```
typo.tya:2:7: undefined variable strng; did you mean "string"?
```

At most three candidates are listed. When no candidate qualifies,
no hint is emitted (the original error stands unchanged).

`util.Suggest` lives at `internal/util/strdist.go` and is reused by
runner and checker; future stages can call the same helper.

## LSP and CLI integration

- `internal/lsp/diagnostics.go::DiagnosticsFor` now `errors.As`-
  unwraps `*parser.ParserError` and emits every contained
  diagnostic via `publishDiagnostics`. Single-error fallback path
  is preserved for non-parser errors.
- `cmd/tya/main.go::printDiagnostic` adds parser / runner / codegen
  unwrap branches so the same rich-render path is used for every
  stage.

## Out of scope (v0.55+)

- Parser signature change (`Parse() → (*Program, []Diag, error)`)
  for symmetry with `lexer.LexWithComments`.
- Expression-level recovery (currently statement-level only).
- LSP code actions for did-you-mean fixes (apply the suggestion
  with a single click).
- Codegen / Runner signature change to also return `[]Diag` slices.

## Compatibility

- Language surface: unchanged from v0.53.
- `tya.toml` schema: unchanged.
- CLI: every existing subcommand keeps its v0.53 behaviour.
  Diagnostic output gains the `[TYA-EXXXX]` prefix and the
  banner-style render for stages that previously emitted only
  `line:col: msg`.
- Existing tests that grep substrings like `"undefined variable"`
  continue to match.

## Tests

`tests/testdata/v54_diag/` adds two testscript fixtures —
`multi_error.txtar` (three top-level position violations in one
pass) and `did_you_mean.txtar` (typo with hint). All
pre-existing tests stay green; the parser unit tests keep their
legacy substring expectations because `*ParserError.Error()`
preserves the `line:col: msg near "tok"` format on
`Error()`.
