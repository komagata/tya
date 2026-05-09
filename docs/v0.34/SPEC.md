# Tya v0.34 Specification — Canonical Syntax, Step 2

Tya v0.34 is the second step in the multi-version landing of
**Canonical Syntax** described in
[`docs/CANONICAL_SYNTAX.md`](../CANONICAL_SYNTAX.md). It introduces
the lexer / AST plumbing the formatter will need in v0.35–v0.36 to
preserve and emit comments deterministically.

v0.34 is intentionally additive and non-breaking. The full set of
parser-side rejections described in `docs/CANONICAL_SYNTAX.md` §11.1
is **not** new work in this release: prior versions already reject
`else if` (the parser requires an indented block after `else`) and
single-line trailing commas in `[1, 2, 3,]` / `(a, b, c,)` /
`{a: 1,}` (the parser fails when it expects an expression after the
comma). Multi-line trailing commas required by §5.4 are deferred to
v0.36 along with the multi-line wrap forms themselves.

## Goals (v0.34)

- The lexer captures `#` comments as a side-channel instead of
  silently stripping them.
- A new `lexer.LexWithComments(src) -> (tokens, comments, errs)` API
  exposes the captured comment slice.
- `ast.Program` gains a `HeaderComments []string` field for the
  file-header comment block (`docs/CANONICAL_SYNTAX.md` §3.3).
- A new `parser.ParseWithComments(toks, comments)` API populates
  `Program.HeaderComments` per the §3.3 rule (contiguous `#` lines
  starting at line 1 at indent 0, separated from the body by exactly
  one blank line).
- The default `Parse(toks)` API and the existing CLI behavior are
  unchanged. v0.34 is a pure addition.

## Non-Goals (v0.34)

- Per-statement `LeadingComments` and `LineEndComment` AST fields —
  deferred to v0.35 alongside formatter v1.
- Forbidden comment positions (§3.4) parser-side rejection — deferred
  to v0.35.
- Formatter changes — v0.35 starts using the new APIs.
- `else if` and single-line trailing-comma rejection — already enforced
  by the existing grammar; no new code required.

## Surface

### Lexer

```go
// LexWithComments runs the lexer and returns the captured comments
// alongside the token stream. Existing Lex() is unchanged.
func LexWithComments(src string) ([]token.Token, []Comment, []error)

type Comment struct {
    Line       int    // 1-based source line of the `#`
    Col        int    // 1-based column of the `#`
    Indent     int    // leading-space count of the source line
    Text       string // comment text, without the leading `#` and trimmed of trailing whitespace
    IsFullLine bool   // true when the source line is whitespace + `#…` (no statement before)
}
```

`Lex(src)` is unchanged: it returns `([]token.Token, []error)` and
discards the comments (they are still captured internally but not
exposed). All existing callers continue to work.

### AST

```go
type Program struct {
    Stmts          []Stmt
    HeaderComments []string // v0.34, see docs/CANONICAL_SYNTAX.md §3.3
}
```

`HeaderComments` holds the comment texts in source order, without the
leading `#`. When no header comment is present, the slice is nil.

### Parser

```go
// ParseWithComments populates Program.HeaderComments per §3.3.
// Existing Parse(toks) is unchanged.
func ParseWithComments(toks []token.Token, comments []CommentInfo) (*ast.Program, error)

type CommentInfo struct {
    Line       int
    Col        int
    Indent     int
    Text       string
    IsFullLine bool
}
```

The §3.3 attachment rule:

1. Skip non-full-line comments (line-end comments).
2. Skip indented comments (those are leading comments on a statement).
3. Find the first contiguous block of full-line top-level comments
   starting at line 1.
4. The block becomes `HeaderComments` only if there is **at least
   one blank line** between the last header comment and the first
   non-comment token. If the first non-comment token is on the line
   immediately following the last header comment (no blank line),
   the comments belong to that statement's leading-comment block
   (which v0.34 still discards — v0.35 will attach it).

## Acceptance Criteria

A v0.34 build is acceptable when:

1. `lexer.LexWithComments` returns a populated `[]Comment` for every
   `#` in the source.
2. `lexer.Lex` continues to return the same token stream as v0.33
   for every input. Existing tests pass unchanged.
3. `parser.ParseWithComments` populates `Program.HeaderComments`
   correctly for header-with-blank-line, leading-without-blank-line,
   line-end-only, and no-comment inputs.
4. The default `parser.Parse` continues to return programs without
   `HeaderComments` populated. Existing CLI behavior is unchanged.
5. `go test ./... -count=1` passes, including the self-host invariant.

## Multi-Version Plan

| Version | Step                                                             |
|---------|------------------------------------------------------------------|
| v0.33   | Parenthesized multi-parameter lambda. (Done.)                    |
| v0.34   | Lexer comment capture + `Program.HeaderComments`. (This release.)|
| v0.35   | Per-stmt `LeadingComments` / `LineEndComment`, formatter v1 (operator spacing, blank-line rules, single-line forms, empty `else` removal, import sort/grouping, empty-collection normalization). |
| v0.36   | Formatter v2 (per-construct multi-line wrap, 80-column limit, atomic-token exception, `"""..."""` rewrite, idempotency).                              |
| v0.37   | Normalize the entire codebase with the formatter; reject non-canonical forms at parse time.                                                           |

`docs/CANONICAL_SYNTAX.md` remains the single source of truth.

## Self-Host Invariant

The new APIs are additive. `parser.Parse` and `lexer.Lex` are
unchanged. The self-host pipeline does not call `LexWithComments` or
`ParseWithComments`. `TestSelfhostV01Scripts` continues to pass.
