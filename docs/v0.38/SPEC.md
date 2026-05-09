# Tya v0.38 Specification — Canonical Syntax (Landing)

Tya v0.38 lands the **Canonical Syntax** described in
[`docs/CANONICAL_SYNTAX.md`](../CANONICAL_SYNTAX.md): every Tya
program now has a single canonical source representation, and
`tya fmt` (alias `tya format`) is the canonical serializer. The
v0.33–v0.37 minor releases progressively introduced parenthesized
multi-parameter lambdas, the comment-capture lexer pipeline, the
per-statement comment AST, the `Unparse` foundation, and the §5
wrap rules. v0.38 connects them into a default-on canonical
formatter.

## Goals (v0.38)

- `tya fmt` (and `tya format`) default to the AST-driven canonical
  serializer (`internal/formatter.Unparse`). The v0.2 conservative
  text pass is retained as a graceful fallback when Unparse cannot
  handle a node and as `--text` opt-out.
- Lexer accepts multi-line `(...)` and `[...]` plus binary-operator
  leading-operator continuation lines.
- Formatter emits all CANONICAL §5 wrap forms (call, array, dict
  block, binary chain, if/while parens, lambda body) and §6.3
  triple-quoted string rewrite, honoring the §5.1 80-column limit.
- Parser-side AST records `HeaderComments` (§3.3) and per-statement
  `LeadingComments` / `LineEndComment` (§3.1, §3.2) via
  `parser.ParseWithComments`; `Unparse` emits them.
- The §11 BinaryExpr precedence rule is honored: operands at lower
  precedence (or right-side-of-same-precedence) get parenthesized.
- `examples/`, `stdlib/`, and `selfhost/v01/compiler.tya` are
  normalized with the canonical formatter; the self-host fixed
  point holds.
- The Tya-written self-host compiler in `selfhost/v01/compiler.tya`
  is extended so it parses the canonical wrap forms (dict block
  form, leading-operator binary chain, multi-line bracket
  continuations) and can compile its own canonical form.

## Non-Goals (v0.38)

- Parser-side rejection of §3.4 forbidden comment positions
  (block-trailing, file-trailing, brackets-internal, all-comment
  block bodies). Comments at those positions are silently dropped
  by the formatter today; structured diagnostics for them are
  scheduled for v0.38.x.
- Renaming `tya fmt` to `tya format` and removing the short
  spelling. Tracked as a separate Epic.
- `docs/FORMAT.md` as a standalone canonical-formatter document.
  CANONICAL_SYNTAX.md remains the single source of truth in v0.38.

## CLI Surface

```
tya fmt [-w] [--text] [path]
tya format [-w] [--text] [path]
```

| Flag     | Behavior                                                     |
|----------|--------------------------------------------------------------|
| (none)   | AST-driven canonical serializer (default in v0.38).         |
| `-w`     | Write canonical output back to the file.                    |
| `--text` | Use the v0.2 text pass (transitional opt-out). Removed in a future release. |
| `--ast`  | Explicitly request the AST serializer (already the default).|

When the AST serializer encounters an unsupported AST shape, the
CLI gracefully falls back to the text pass for that file. This
keeps editor save hooks safe even on programs that exercise
features not yet covered by `Unparse`.

## Self-Host Realignment

`selfhost/v01/compiler.tya` is itself a Tya program; previously it
parsed only the v0.1 surface and could not consume the canonical
wrap forms. v0.38 extends the Tya-written compiler so it can:

- suppress NEWLINE / INDENT / DEDENT inside `(` `[` brackets and
  around binary-operator continuation lines (mirrors the Go-side
  `suppressBracketNewlines` post-process)
- parse the §5.3.3 dict block form `IDENT =\n  key: value\n  ...`

With these in place, `tya fmt selfhost/v01/compiler.tya` produces
canonical output that the same compiler can then compile, so the
self-host fixed point continues to hold against the canonical
formatter.

## Tests

- `internal/formatter/unparse_test.go` covers each wrap rule with
  an idempotency assertion.
- `internal/formatter/corpus_test.go` walks every `.tya` source in
  `stdlib/`, `examples/`, and `selfhost/` and asserts that
  `Unparse` either produces parseable output or returns an
  unsupported error (the CLI fallback path).
- `tests/testdata/v01_selfhost/...` continues to gate the
  self-host fixed point.
- `go test ./... -count=1` is green on this commit.

## Acceptance Criteria

A v0.38 build is acceptable when:

1. `tya fmt path.tya` and `tya format path.tya` produce the
   canonical AST-driven layout by default.
2. Running `tya fmt -w` over `examples/`, `stdlib/`, and
   `selfhost/v01/` is idempotent.
3. `TestSelfhostV01Scripts` passes.
4. `go test ./... -count=1` passes.
5. Each CANONICAL §5 wrap rule and §6.3 triple-quote rewrite is
   exercised by at least one unit test in
   `internal/formatter/unparse_test.go`.
6. The corpus round-trip test in
   `internal/formatter/corpus_test.go` reports zero unsupported
   files for the maintained codebase (archived pre-v0.1 sources
   may remain unsupported).

## Deferred to v0.38.x / v0.39

- Parser-side rejection of CANONICAL §3.4 forbidden comment
  positions, with structured diagnostics.
- Removal of the `--text` opt-out once the AST serializer covers
  every AST shape v0.1 supports.
- Source-position fidelity inside expressions (sub-expression
  spans for diagnostics).
- Promotion of `tya fmt` to `tya format` as the only spelling.

## References

- [`docs/CANONICAL_SYNTAX.md`](../CANONICAL_SYNTAX.md) — the
  authoritative canonical-syntax specification.
- v0.33 SPEC: parenthesized multi-parameter lambda.
- v0.34 SPEC: lexer comment capture, `Program.HeaderComments`.
- v0.35 SPEC: per-stmt `Comments` map at top level.
- v0.36 SPEC: nested-body comment attachment.
- v0.37 SPEC: `Unparse` foundation for the common-case subset.
