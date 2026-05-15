---
layout: doc
title: Spec
permalink: /v0.37/spec/
---

# Tya v0.37 Specification — Canonical Syntax, Step 5

Tya v0.37 introduces the **AST-driven canonical serializer**
(`unparse`) that Canonical Syntax §11.3 requires. v0.37 ships the
foundation: `formatter.Unparse(prog *ast.Program) (string, error)`
that handles a useful subset of the AST. v0.38 extends coverage and
adds the wrap rules; v0.39 wires the serializer into the user-facing
`tya format` command and normalizes the codebase.

v0.37 does not change `tya fmt` behavior. The existing v0.2 text
formatter continues to ship. The serializer is exported for tests
and opt-in tooling.

## Goals (v0.37)

- New `internal/formatter.Unparse(prog) (string, error)` that
  renders `prog` as canonical Tya source for the supported subset.
- Coverage of the *common case* of Tya programs: imports, simple
  assignments (with all literal / operator / call / member / index
  / array / dict expressions), expression statements, returns,
  raises, break / continue, single-line and block-bodied lambdas,
  `if` / `elseif` / `else`, `while`, `for`.
- Operator spacing, single-line forms, and indentation per
  CANONICAL §4 / §7.
- `if … elseif …` is emitted using the canonical `elseif` keyword
  even when the source spelled it as nested `else / if`.

## Non-Goals (v0.37)

- Module / class / interface declarations — `Unparse` returns an
  unsupported error for these. v0.38 covers them.
- `match` / `try` / `raise` (raise is supported; match and
  try/catch are deferred).
- Multi-line wrap forms (CANONICAL §5). All output is single-line
  per construct in v0.37.
- 80-column limit and atomic-token exception.
- `"""..."""` rewrite rule (CANONICAL §6.3).
- Comment emission. `Unparse` does not yet read
  `Program.HeaderComments` or `Program.Comments`. v0.38 wires those
  in.
- Empty-`else` removal, import sort, blank-line rules. Deferred.
- Replacing `tya fmt` with the AST-driven serializer.
- Idempotency or full round-trip guarantees outside the supported
  subset.

## API

```go
package formatter

// Unparse renders prog as canonical Tya source. Returns an error
// when the AST contains a node v0.37 does not yet handle. The
// existing FormatSource(string) text pass is unchanged.
func Unparse(prog *ast.Program) (string, error)
```

The error is a sentinel-style `fmt.Errorf("formatter.Unparse: …")`.
Callers (tests, tooling) decide whether to fall back to the text
formatter or surface the error.

## Acceptance Criteria

A v0.37 build is acceptable when:

1. `formatter.Unparse` round-trips programs in the supported subset
   to canonical source: assignments, imports, expression
   statements, returns / raises / break / continue, `if` /
   `elseif` / `else`, `while`, `for`, single-line and block-bodied
   `name -> body` and `(a, b) -> body` lambdas, all literal /
   operator / call / member / index / array / dict expressions.
2. `Unparse` returns an unsupported error when the AST contains a
   `module` / `class` / `interface` / `match` / `try` node.
3. The default `tya fmt` and `tya format` continue to use the
   v0.2 text pass; no behavior change for existing CLI users.
4. `go test ./... -count=1` passes, including the self-host
   invariant.

## Multi-Version Plan (updated)

| Version | Step                                                   |
|---------|--------------------------------------------------------|
| v0.33   | Parenthesized multi-parameter lambda. (Done.)          |
| v0.34   | Lexer comment capture; `Program.HeaderComments`. (Done.)|
| v0.35   | Per-stmt `Comments` map at top level. (Done.)          |
| v0.36   | Nested-body comment attachment. (Done.)                |
| v0.37   | `Unparse` foundation for common-case subset. (This release.) |
| v0.38   | Extend `Unparse` to modules / classes / match / try; emit comments; add empty-`else` removal, import sort, blank-line rules. |
| v0.39   | Multi-line wrap (§5), 80-column limit, `"""..."""` rewrite (§6.3), idempotency. |
| v0.40   | Replace text formatter with AST-driven serializer; normalize codebase; reject non-canonical forms. |

## Self-Host Invariant

`Unparse` is not on the self-host path. The v0.2 text formatter
remains the only user-facing serializer. `TestSelfhostV01Scripts`
continues to pass.
