# Tya v0.33 Specification — Canonical Syntax, Step 1

Tya v0.33 begins the multi-version push toward **Canonical Syntax**, the
property defined in [`docs/CANONICAL_SYNTAX.md`](../CANONICAL_SYNTAX.md):
every well-formed Tya program has exactly one source representation,
and the formatter (`tya format`) is the canonical serializer.

The full Canonical Syntax landing requires changes to the parser, the
AST, the formatter, and the entire normalized codebase. v0.33 is the
**first step** of that landing. It is intentionally additive and
non-breaking so the rest of the work can land in subsequent minor
releases without disturbing existing programs.

## Goals (v0.33)

- Parser accepts the parenthesized multi-parameter lambda form
  `(a, b) -> body` (`docs/CANONICAL_SYNTAX.md` §5.3.4).
- The existing comma-separated form `a, b -> body` and the
  single-parameter form `name -> body` remain valid; this release is
  purely additive.
- The full Canonical Syntax direction is referenced from this SPEC and
  from `ROADMAP.md` so the path through v0.34–v0.37 is visible.

## Non-Goals (v0.33)

- Comment AST attributes (`leading_comments`, `line_end_comment`,
  `file_header_comments`) — deferred to v0.34.
- Parser rejection of `else if`, single-line trailing commas, and
  forbidden comment positions — deferred to v0.34.
- Formatter wrap rules per construct — deferred to v0.35–v0.36.
- Triple-quoted string rewrite rule — deferred to v0.36.
- Codebase-wide normalization and turning non-canonical forms into
  parse errors — deferred to v0.37.

## Surface Syntax

```
ParenFunc = '(' Ident { ',' Ident } ')' '->' FuncBody
```

Examples:

```tya
add = (a, b) -> a + b

square = (x) -> x * x

triple = (a, b, c) ->
  return a + b + c
```

The body grammar is identical to today's `name -> body` form: an
inline expression after `->`, or a block body when `->` is followed
by a newline + INDENT.

### Disambiguation

`(expr)` (a parenthesized expression) and `(a, b) -> body` (a
parenthesized parameter list) are disambiguated by lookahead. The
parser commits to the parameter-list form only when it sees:

```
LPAREN IDENT (',' IDENT)* RPAREN ARROW
```

In every other case `(` opens a grouping expression. The empty form
`() -> body` continues to work as before.

### Acceptance Criteria

A v0.33 build is acceptable when:

1. `add = (a, b) -> a + b` parses, type-checks, and runs to produce
   `add(2, 3) == 5`.
2. `(x) -> x * x` parses and runs.
3. `(a, b, c) ->` followed by an indented block body parses and runs.
4. The pre-existing forms `a, b -> body`, `a -> body`, `() -> body`,
   and `-> body` continue to work unchanged.
5. `go test ./... -count=1` passes, including the self-host invariant.

## Multi-Version Plan

| Version | Canonical Syntax progress |
|---------|----------------------------|
| v0.33   | Parenthesized multi-parameter lambda (this release).        |
| v0.34   | Add `leading_comments` / `line_end_comment` / `file_header_comments` to AST nodes. Parser attaches comments per `docs/CANONICAL_SYNTAX.md` §3. Reject `else if`. Reject single-line trailing commas in `[]`/`()`/`{}`. Reject forbidden comment positions (§3.4). |
| v0.35   | Formatter v1: operator spacing (§7), blank-line rules (§3.5), single-line forms only, empty `else` removal (§8.7), import sorting and grouping (§8.4), empty-collection normalization (§8.6). |
| v0.36   | Formatter v2: per-construct multi-line wrap (§5), 80-column limit, atomic-token exception, `"""..."""` rewrite (§6.3), idempotency. |
| v0.37   | Normalize `stdlib/`, `examples/`, `selfhost/`, `tests/testdata/` with the formatter. Verify self-host fixed point. Reject non-canonical forms at parse time once the codebase is fully normalized. |

`docs/CANONICAL_SYNTAX.md` remains the single source of truth for the
target shape. Each minor release's SPEC document references it.

## Self-Host Invariant

The self-host pipeline does not use the `(a, b) -> body` form. Its
source remains valid v0.33 source. `TestSelfhostV01Scripts` continues
to pass.

## Testing Strategy

1. Parser unit tests for `(a, b) -> body`, `(x) -> body`,
   `(a, b, c) ->\n  block`, mixed with the existing forms.
2. CLI script test (`tests/testdata/v33/paren_lambda.txtar`) for an
   end-to-end `tya run`.
3. Self-host fixed point (`TestSelfhostV01Scripts`) continues to
   pass.
4. Default test suite (`go test ./...`) remains green.
