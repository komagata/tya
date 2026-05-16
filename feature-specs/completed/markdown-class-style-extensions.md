---
status: completed
goal_ready: false
---

# Feature: Markdown Class-Style Extensions

## Goal

Modernize the queued Markdown extensions so the public API follows Tya's
class-style stdlib convention: `import markdown` exposes the `Markdown` class,
and users call `Markdown.parse`, `Markdown.to_html_ast`, `Markdown.render`, and
`Markdown.to_html` instead of old module-style functions such as
`markdown.parse`.

## Context

`ROADMAP.md` still names this work "Markdown module extensions" and lists
module-style APIs:

```text
markdown.parse
markdown.to_html_ast
markdown.render
```

That wording came from the older module surface documented in
`docs/v0.32/SPEC.md`. The current stdlib implementation lives at
`stdlib/markdown/Markdown.tya` and already exposes `Markdown.to_html(text)` as a
static class method. Existing tests use:

```tya
import markdown

html = markdown.Markdown.to_html(src)
```

The future extension work should continue from the current class-file package
style, not revive module-level functions.

## Behavior

- Rename the roadmap item from "Markdown module extensions" to "Markdown
  class-style extensions".
- Keep `Markdown.to_html(text)` as the simple one-step API that parses and
  renders Markdown to an HTML string.
- Add `Markdown.parse(text)` as the public parser entry point.
- Add `Markdown.to_html_ast(ast)` to transform the Markdown AST into a public
  HTML-oriented AST representation.
- Add `Markdown.render(ast_or_html_ast)` to render a supported Markdown AST or
  HTML-oriented AST to an HTML string.
- Public AST nodes use dictionaries with stable `kind` fields and documented
  fields for each supported block/inline kind.
- Unsupported Markdown syntax continues to degrade to plain text where
  reasonable instead of raising user-visible parse errors.
- Existing supported Markdown remains HTML-escaped by default.
- GFM extensions are added behind the class-style API:
  - tables
  - task lists
  - strikethrough
  - fenced-code info strings
- CommonMark extensions are added behind the same API:
  - reference link definitions
  - images
  - nested lists
  - setext headings
  - HTML blocks
- Raw HTML pass-through is documented with a security note and is opt-in if an
  option object is needed to avoid changing the current escaping default.

## Scope

- `ROADMAP.md`
- `stdlib/markdown/Markdown.tya`
- `docs/STDLIB.md`
- next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`
- `tests/testdata/` script fixtures for class-style Markdown API behavior
- existing Markdown tests that mention module-style names

## Dependencies

- Implement `feature-specs/stdlib-template-library.md` first only if Markdown
  rendering is expected to reuse generic template rendering. Otherwise this spec
  can proceed independently.

## Out of Scope

- Reintroducing top-level module functions such as `markdown.parse`.
- Replacing the pure-Tya Markdown implementation with a native dependency.
- Full CommonMark conformance beyond the subset selected for this feature.
- Markdown editor preview UI.
- Sanitizing raw HTML beyond the documented opt-in pass-through behavior.

## Acceptance Criteria

- `ROADMAP.md` no longer calls this work "Markdown module extensions".
- No editable current documentation advertises `markdown.parse`,
  `markdown.to_html_ast`, or `markdown.render` as future APIs.
- A user can call `Markdown.parse(text)` after `import markdown`.
- A user can call `Markdown.to_html_ast(ast)` on the parse result.
- A user can call `Markdown.render(ast_or_html_ast)` and get deterministic HTML.
- `Markdown.to_html(text)` remains available and produces the same output as
  `Markdown.render(Markdown.to_html_ast(Markdown.parse(text)))` for supported
  syntax.
- Tables, task lists, strikethrough, and fenced-code info strings are covered by
  script tests.
- Reference links, images, nested lists, setext headings, and HTML blocks are
  covered by script tests for the selected CommonMark subset.
- Raw HTML pass-through behavior is documented and tested, including the default
  escaping behavior.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run TestV.*Script -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Open Questions

None.
