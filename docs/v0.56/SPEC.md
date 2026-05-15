---
layout: doc
title: Spec
permalink: /v0.56/spec/
---

# Tya v0.56 Specification

> **Status:** shipped. The `tya version` constant is `0.56.0`.
> v0.56 unifies the public signatures of `parser.Parse` /
> `codegen.EmitC*` / `runner.RunFile` around the
> `(X, []diag.Diagnostic, error)` shape and adds
> expression-level recovery for the parser. The language surface
> is unchanged from v0.55.

## Theme

v0.54 migrated every compiler stage's errors to the structured
`diag.Diagnostic` type and added statement-level recovery in the
parser. v0.56 finishes that migration:

- Public Parser / Codegen / Runner entry points return the same
  three-value shape `(result, []diag.Diagnostic, error)` so
  callers can iterate `diags` directly instead of unwrapping
  `*XxxError` via `errors.As`.
- Expression-level recovery in the parser keeps `CallExpr`,
  `ArrayLit`, and `DictLit` element parsing going past per-arg
  errors — one bad expression no longer hides its siblings.

`*ParserError` / `*CodegenError` / `*RunnerError` are retained
as `error` wrappers for backwards compatibility; new code can
simply read the diags slice.

## New API signatures

### Parser

```go
func Parse(toks []token.Token) (*ast.Program, []diag.Diagnostic, error)
func ParseWithComments(toks []token.Token, comments []CommentInfo) (*ast.Program, []diag.Diagnostic, error)
```

- `diags` carries every recoverable diagnostic (statement-level
  + expression-level). Empty when the parse is clean.
- `err` is `*ParserError{Diags: diags}` for compatibility with
  `errors.As(err, &perr)`. It is `nil` when `diags` is empty.
- Invariant: `len(diags) == 0 ⟺ err == nil`.

### Codegen

```go
func EmitC(prog *ast.Program) (string, []diag.Diagnostic, error)
func EmitCWithPath(prog *ast.Program, sourcePath string) (string, []diag.Diagnostic, error)
func EmitCWithCoverage(prog *ast.Program, sourcePath string, opt *CoverageOptions) (string, *CoverageRegistry, []diag.Diagnostic, error)
```

Codegen remains fail-fast in v0.56 — `diags` is `nil` on success
or a single-entry slice (copied from
`err.(*CodegenError).Diags`) on failure. Multi-error codegen is
scoped out to v0.57+.

### Runner

```go
func RunFile(path string, in io.Reader, out io.Writer, args []string) ([]diag.Diagnostic, error)
```

- Returns the parser's recoverable diagnostic slice alongside
  the first fatal error (lex / parser / checker / eval).
- `RunnerError` widens from a single `Diag` field to a
  `Diags []diag.Diagnostic` slice. A `Diag()` method returns the
  first entry for pre-v0.56 call sites.
- `AsRunnerError` continues to unwrap from the `error` chain.

## Expression-level recovery

The parser now recovers at three expression-list boundaries:

1. **`CallExpr` argument list** (`f(a, b, c)`) — each bad
   argument records a diagnostic and the parser skips to the
   next `,` or `)`.
2. **`ArrayLit` elements** (`[a, b, c]`) — same shape, with `]`
   as the close marker.
3. **`DictLit` prop values** (`{k1: v1, k2: v2}`) — only the
   prop *value* triggers recovery; a missing or malformed key
   still aborts the literal.

Nested brackets are skipped over so an inner stray `,` cannot
fool the outer recovery loop into stopping early.

Binary chains (`a + broken + c`) and member chains
(`a.broken.c`) remain whole-expression failures in v0.56 and are
scope-out for v0.57+.

### New helper

```go
// internal/parser/recovery.go
func (p *Parser) skipToCommaOrClose(closes ...token.Type)
func (p *Parser) skipBalanced(open, close token.Type)
```

Called from CallExpr / ArrayLit / DictLit parse loops. NEWLINE,
DEDENT, EOF, and the supplied close brackets halt the scan; the
helper does NOT consume the boundary token.

## Migration guide

The new signatures break source compatibility for direct
callers. The mechanical migration is:

```go
// Pre-v0.56
prog, err := parser.Parse(toks)

// v0.56
prog, _, err := parser.Parse(toks)

// or, if you want multi-error visibility:
prog, diags, err := parser.Parse(toks)
for _, d := range diags { renderDiag(d) }
```

Same shape for `EmitC` / `EmitCWithPath`, with an extra `_` in
the destructure. `EmitCWithCoverage` gains a fourth `[]Diag`
return value before `error`. `RunFile` gains a leading
`[]Diag` return value before `error`.

`errors.As(err, &perr)` continues to work, so callers that
prefer the wrapper-based path do not need to migrate.

## Scope-out (v0.57+)

- Codegen multi-error (`unsupported AST shape` 全件 collect
  instead of bailing on the first).
- Runner multi-error (low priority — eval is inherently
  fail-fast).
- Binary-chain / MemberExpr-chain expression-level recovery.
- Gradual deprecation of `*ParserError` / `*CodegenError` /
  `*RunnerError` once all callers iterate diags directly.
