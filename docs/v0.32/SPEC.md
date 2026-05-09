# Tya v0.32 Specification — Lexer Diagnostics + Markdown Foundation

Tya v0.32 advances two epics by their first foundation step:

1. **Diagnostics migration**: the lexer's user-visible errors are
   migrated onto the `internal/diag` pipeline introduced in v0.29.
   Lexer errors now render with banners, source snippets, codes, and
   hints. The remaining stages (parser, codegen, runner, fmt) and
   did-you-mean / multi-error parsing are deferred to v0.33+.
2. **`markdown` stdlib module**: a minimal pure-Tya Markdown-to-HTML
   converter exposing only `markdown.to_html(text) -> string`. A
   subset of CommonMark is supported (block-level: ATX headings,
   paragraphs, thematic breaks, fenced code blocks, blockquotes,
   single-level unordered/ordered lists; inline: emphasis, strong,
   inline code, links, autolinks). Public AST, visitor, GFM
   extensions, raw HTML pass-through, nested lists, images, and
   reference links are deferred.

The language itself does not change. v0.32 is a foundation release
for both epics.

## Goals

- Every lexer error renders through `internal/diag` with a stable
  `TYA-Exxxx` code, banner, snippet, and hint.
- `stdlib/markdown.tya` ships and exposes `markdown.to_html(text)`.
- Both `tya run` and `tya build` paths surface lexer diagnostics in
  the new format when emitted.
- The `selfhost/v01/compiler.tya` fixed point continues to pass.

## Non-Goals (v0.32)

- Migrating parser, codegen, runner, or fmt errors to `internal/diag`.
- Multi-error parsing.
- Did-you-mean suggestions.
- Public Markdown AST (`markdown.parse`, `markdown.to_html_ast`,
  `markdown.render`).
- GFM extensions (tables, task lists, strikethrough, autolinks
  beyond the simplest `<URL>` form, fenced-code info-string
  rendering).
- Raw HTML pass-through.
- Nested lists.
- Reference links and images.
- Loose-vs-tight list distinction for paragraph wrapping.
- Hard line breaks via two trailing spaces.
- Setext headings.
- HTML blocks.
- Link reference definitions.
- Running the full CommonMark conformance suite.

## Lexer Diagnostics Migration

### Codes

The following codes are assigned in `docs/v0.32/CODES.md` and the
v0.29 `docs/v0.29/CODES.md` index is extended:

| Code        | Title                              | Source |
|-------------|------------------------------------|--------|
| `TYA-E0001` | Tabs are forbidden                 | lexer  |
| `TYA-E0002` | Trailing whitespace                | lexer  |
| `TYA-E0003` | Bad indentation step               | lexer  |
| `TYA-E0004` | Inconsistent indentation           | lexer  |
| `TYA-E0005` | Unterminated string                | lexer  |
| `TYA-E0006` | Unterminated escape                | lexer  |
| `TYA-E0007` | Unknown escape                     | lexer  |
| `TYA-E0008` | Unterminated triple-quoted string  | lexer  |
| `TYA-E0009` | Mixed indentation in triple-string | lexer  |

Codes assigned but not yet shipped (reserved for future lexer
errors): `TYA-E0010`–`TYA-E0099`.

### Pipeline

`lexer.Lex` continues to return `[]token.Token, []error` for
backward compatibility, but the error values are now
`*lexerDiagnostic`, an internal type implementing `error` and
exposing the underlying `diag.Diagnostic`. The CLI's existing
`printDiagnostic` is extended to detect lexer diagnostics the same
way it already detects strict-checker diagnostics, and renders them
through `diag.Render` / `diag.RenderJSON`.

For commands that surface only the first lexer error today (`tya
run`, `tya build`, `tya emit-c`), behavior is unchanged: the first
diagnostic is rendered in the new format. `tya check` already
collects multi-error strict diagnostics; lexer errors stay
fail-fast in v0.32 (multi-error parsing is deferred).

### Test fixtures

Existing fixtures that pin lexer error wording with `stderr '...'`
are updated to match the new banner format or, where the wording
itself is incidental, switched to assert on the error code
(`stderr 'TYA-E000\d'`). Wording-pinning fixtures stay byte-exact
against the new banner.

## `markdown` stdlib module

### API

```tya
import markdown

html = markdown.to_html(text)
```

`markdown.to_html(text: string) -> string` parses `text` as Markdown
and returns an HTML string. Output is always HTML-escaped at the
text level: `&` becomes `&amp;`, `<` becomes `&lt;`, `>` becomes
`&gt;`, `"` becomes `&quot;`, `'` becomes `&#39;`.

### Supported syntax

Block-level:

| Syntax                     | Output                              |
|----------------------------|-------------------------------------|
| `# H1` to `###### H6`      | `<h1>…</h1>` to `<h6>…</h6>`        |
| Paragraph (one or more lines, separated by blank lines) | `<p>…</p>`     |
| `---`, `***`, `___` on a line by itself | `<hr/>`                |
| Triple backticks fenced block (```) | `<pre><code>…</code></pre>`, content escaped, info string ignored |
| `> quote text`             | `<blockquote><p>…</p></blockquote>` (single-level only) |
| `- item` / `* item` / `+ item` (one level) | `<ul><li>…</li>…</ul>` |
| `1. item` / `2. item` (one level)         | `<ol><li>…</li>…</ol>` |

Inline:

| Syntax              | Output                              |
|---------------------|-------------------------------------|
| `*text*` / `_text_` | `<em>text</em>`                     |
| `**text**` / `__text__` | `<strong>text</strong>`         |
| `` `code` ``        | `<code>code</code>` (escaped)       |
| `[text](url)`       | `<a href="url">text</a>` (URL escaped, text rendered with inline rules) |
| `<https://example.com>` | `<a href="https://example.com">https://example.com</a>` autolink |

### Deferred syntax (out of v0.32 scope)

- Setext headings (`===`, `---`)
- Hard line breaks (two trailing spaces)
- Nested lists / loose vs. tight distinction
- Reference link definitions (`[id]: url`)
- Images (`![alt](url)`)
- HTML blocks
- GFM tables, task lists, strikethrough
- Inline raw HTML
- Code-block info-string class (`<code class="language-…">`)

Markdown that uses unsupported syntax does not error: the parser
falls back to treating the unsupported construct as plain text where
reasonable, so user input never crashes the converter.

### Implementation

`stdlib/markdown.tya` is pure Tya. No native dependency. It runs
under every `tya` build target.

### Public API ergonomics

The module exposes only `to_html` in v0.32. Future v0.33+ releases
add `markdown.parse`, `markdown.to_html_ast`, `markdown.render`, the
public AST, visitor helpers, and GFM extensions.

### Tests

A new `tests/stdlib_markdown_test.tya` exercises each supported
syntax and a few edge cases (escaping, mixed inline, empty input).
The full CommonMark conformance suite is not part of v0.32.

## Self-Host Invariant

Neither change touches the self-host pipeline. The lexer migration
keeps the `lexer.Lex` signature, and the markdown module is loaded
only when imported. `TestSelfhostV01Scripts` continues to pass.

## Acceptance Criteria

A v0.32.0 build is acceptable when:

1. Each lexer error path emits a `diag.Diagnostic` with a
   `TYA-E000n` code, banner, snippet, and hint.
2. `--format=json` and `--color` flags affect lexer diagnostics the
   same way they already affect strict-checker diagnostics.
3. `import markdown` resolves and `markdown.to_html(text)` produces
   HTML for the supported syntax.
4. `go test ./... -count=1` passes, including the self-host
   invariant.

## Deferred to v0.33+

- Migrate parser / codegen / runner / fmt errors to the diagnostics
  pipeline.
- Add did-you-mean suggestions for unknown-name diagnostics.
- Add multi-error parsing.
- Public `markdown.parse` / AST / visitor.
- GFM extensions in `markdown`.
- Reference links, images, nested lists, setext headings, HTML
  blocks in `markdown`.
- Run a representative CommonMark conformance subset as part of
  `go test ./...`.
