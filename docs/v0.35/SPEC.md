---
layout: doc
title: Spec
permalink: /v0.35/spec/
---

# Tya v0.35 Specification — Canonical Syntax, Step 3

Tya v0.35 is the third step in the multi-version landing of
**Canonical Syntax** described in
[`docs/CANONICAL_SYNTAX.md`](../CANONICAL_SYNTAX.md). It extends the
v0.34 comment-capture infrastructure with **per-statement comment
attachment** so the formatter (v0.36) can preserve and emit comments
deterministically.

v0.35 is intentionally additive and non-breaking. The formatter
itself remains the v0.2 conservative text pass; AST-driven `unparse`
work lands in v0.36.

## Goals (v0.35)

- `ast.Program` gains a `Comments map[Stmt]StmtComments` populated
  by `parser.ParseWithComments`.
- `StmtComments{Leading []string, LineEnd string}` carries the
  attached comments per `docs/CANONICAL_SYNTAX.md` §3.1 (leading
  comment block at the same indent) and §3.2 (line-end comment on
  the same source line as the statement's start).
- The default `Parse(toks)` path is unchanged and leaves
  `Program.Comments` nil. Existing CLI behavior is identical to
  v0.34.

## Non-Goals (v0.35)

- Comment attachment for nested statements (inside `if` / `while` /
  `for` / `match` / module / class bodies). v0.35 only attaches
  comments to top-level statements (`Program.Stmts`). Inner
  statements arrive in v0.36 alongside the formatter's wrap rules.
- Forbidden comment positions (§3.4) parser-side rejection.
- Formatter changes. The text-based formatter from v0.2 continues to
  ship.
- Any breaking change to `Lex`, `Parse`, or AST node types.

## Surface

```go
// internal/ast
type Program struct {
    Stmts          []Stmt
    HeaderComments []string                   // v0.34
    Comments       map[Stmt]StmtComments      // v0.35
}

type StmtComments struct {
    Leading []string  // contiguous # lines immediately before stmt, same indent
    LineEnd string    // single # comment on the same source line as stmt's start
}
```

`ParseWithComments` populates `Comments` for each top-level statement
that has a leading and/or line-end comment per CANONICAL §3.1 / §3.2.
Statements without comments do not appear in the map.

### Attachment rules (v0.35 scope)

Given a top-level statement at start line `S` and indent 0:

- **Leading**: walk the comments slice backward starting from line
  `S - 1`. While each comment is full-line, has indent 0, and is on
  a strictly contiguous line (no blank line gap), prepend its text
  to `Leading`. Stop at the first comment that breaks the chain.
- **LineEnd**: the first non-full-line comment whose `Line == S`.
- A comment that becomes a leading or line-end comment is removed
  from the available pool, so it does not also become the trailing
  comment of an earlier statement.

The header-comment block (§3.3) is computed first, by v0.34's logic.
Its comments are excluded from the leading-comment pool.

## Acceptance Criteria

A v0.35 build is acceptable when:

1. `parser.ParseWithComments` populates `Program.Comments` for every
   top-level statement with leading or line-end comments per the
   rules above.
2. Inputs without comments leave `Program.Comments` as nil (or an
   empty map) and existing tests pass unchanged.
3. The default `parser.Parse` continues to return programs with
   `Comments == nil`. CLI behavior is unchanged.
4. `go test ./... -count=1` passes, including the self-host
   invariant.

## Multi-Version Plan (updated)

| Version | Step                                                                    |
|---------|-------------------------------------------------------------------------|
| v0.33   | Parenthesized multi-parameter lambda. (Done.)                           |
| v0.34   | Lexer comment capture; `Program.HeaderComments`. (Done.)                |
| v0.35   | Per-stmt `Comments` map at top level. (This release.)                   |
| v0.36   | Comment attachment for nested stmts; AST-driven formatter v1 (operator spacing, blank-line rules, single-line forms, empty `else` removal, import sort/grouping, empty-collection normalization, comment emission). |
| v0.37   | Formatter v2 (per-construct multi-line wrap, 80-column limit, atomic-token exception, `"""..."""` rewrite, idempotency).                                  |
| v0.38   | Normalize the entire codebase with the formatter; reject non-canonical forms at parse time.                                                                |

(Originally v0.36 was planned to ship both formatter v1 and v2. The
schedule shifts each step one release later to keep individual
releases reviewable.)

`docs/CANONICAL_SYNTAX.md` remains the single source of truth.

## Self-Host Invariant

`Parse` is unchanged. The self-host pipeline does not call
`ParseWithComments`. `TestSelfhostV01Scripts` continues to pass.
