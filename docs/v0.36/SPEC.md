# Tya v0.36 Specification — Canonical Syntax, Step 4

Tya v0.36 is the fourth step in the multi-version landing of
**Canonical Syntax** described in
[`docs/CANONICAL_SYNTAX.md`](../CANONICAL_SYNTAX.md). It extends the
v0.35 per-statement comment attachment to **nested statement bodies**
so the formatter (v0.37+) has comment metadata for every statement
position the formatter will need to render.

v0.36 is intentionally additive and non-breaking. The formatter
itself remains the v0.2 conservative text pass; the AST-driven
formatter is now scheduled for v0.37.

## Goals (v0.36)

- `parser.ParseWithComments` populates `Program.Comments` for
  statements inside the following nested bodies:
  - `IfStmt.Then`, `IfStmt.Else`
  - `WhileStmt.Body`
  - `ForInStmt.Body`
  - `TryCatchStmt.Try`, `TryCatchStmt.Catch`
  - `MatchStmt.Cases[i].Body`
  - `FuncLit.Body` (when reached via an `AssignStmt` rhs or an
    `ExprStmt` expression)
- Each nested block uses `outer indent + 2` as the §3.1 indent rule's
  required indent for leading comments.
- The default `Parse(toks)` path is unchanged.

## Non-Goals (v0.36)

- Comment attachment for `ModuleDecl.Members` and `ClassDecl.Methods`
  / `Fields` / `Vars`. Those are not Stmt slices; they require
  separate AST surgery and are deferred to v0.37 alongside the
  AST-driven formatter that needs to emit them.
- Forbidden comment positions (§3.4) parser-side rejection.
- Formatter changes. The v0.2 text formatter continues to ship.

## Surface

```go
// Same shape as v0.35; v0.36 just populates more of it.
type Program struct {
    Stmts          []Stmt
    HeaderComments []string
    Comments       map[Stmt]StmtComments
}
```

### Attachment rules (v0.36 scope)

For every Stmt in any visited block (top-level or nested):

- **Leading**: contiguous full-line `#` lines immediately before the
  Stmt, at the same indent as the Stmt. Indent = depth × 2.
- **LineEnd**: a single non-full-line `#` comment whose source line
  equals the Stmt's start line.
- A comment that becomes a leading or line-end comment is removed
  from the available pool.

Header comments (§3.3) are computed first by v0.34's logic.

The walker recurses into every Stmt-list body of structural Stmt
types and into the body of any `FuncLit` reached via an `AssignStmt`
right-hand side or an `ExprStmt` expression. It does not yet enter
`ModuleDecl.Members` or `ClassDecl.Methods` (those carry expressions,
not Stmts directly).

## Acceptance Criteria

A v0.36 build is acceptable when:

1. `parser.ParseWithComments` populates `Program.Comments` for
   leading and line-end comments inside nested bodies of
   if/while/for/match/try and `FuncLit` bodies.
2. The default `parser.Parse` returns programs with `Comments == nil`
   and CLI behavior is unchanged.
3. `go test ./... -count=1` passes, including the self-host
   invariant.

## Multi-Version Plan (updated)

| Version | Step                                                      |
|---------|-----------------------------------------------------------|
| v0.33   | Parenthesized multi-parameter lambda. (Done.)             |
| v0.34   | Lexer comment capture; `Program.HeaderComments`. (Done.)  |
| v0.35   | Per-stmt `Comments` map at top level. (Done.)             |
| v0.36   | Comment attachment for nested stmt bodies. (This release.)|
| v0.37   | Module / class member comment attachment + AST-driven formatter v1 (operator spacing, blank-line rules, single-line forms, empty `else` removal, import sort, empty-collection normalization, comment emission). |
| v0.38   | Formatter v2 (multi-line wrap, 80-col, `"""..."""` rewrite, idempotency). |
| v0.39   | Normalize codebase with formatter; reject non-canonical forms at parse time. |

## Self-Host Invariant

`Parse` is unchanged. The self-host pipeline does not call
`ParseWithComments`. `TestSelfhostV01Scripts` continues to pass.
