---
status: completed
goal_ready: false
---

# Feature: Generic Template Stdlib Library

## Goal

Add a generic `template` standard library package so Tya programs can render
text templates for HTML, configuration files, code generation, emails, docs, and
HTTP responses without tying template support to `net/http`.

## Context

`ROADMAP.md` currently queues a "Template engine with variable interpolation,
conditionals, loops, partials, and HTML escaping by default" under
`net/http v2`. That is useful for web servers, but the same capability is also
needed outside HTTP.

Tya already has string interpolation for simple inline expressions, but string
interpolation is not a full template system: it has no loops, conditionals,
partials, include boundaries, or configurable escaping. A stdlib package keeps
this as library behavior rather than adding new language syntax.

The implementation should follow current class-style stdlib conventions:

```tya
import template

html = template.Template.render("Hello, {{ name }}", { name: "Tya" })
```

## Behavior

- Add a public `template` stdlib package with a `Template` class.
- `Template.render(source, data)` renders a template string with a data
  dictionary.
- `Template.render(source, data, options)` accepts options when needed.
- `Template.render_file(path, data)` reads and renders a template file.
- `Template.render_file(path, data, options)` accepts the same options.
- Variable interpolation uses `{{ name }}` and dotted/indexed paths such as
  `{{ user.name }}` and `{{ items[0].name }}`.
- Missing values render as an empty string by default.
- Strict mode reports missing values as template errors:

```tya
Template.render("{{ user.name }}", {}, { strict: true })
```

- Conditionals:

```text
{{ if user.admin }}
Admin
{{ else }}
User
{{ end }}
```

- Loops:

```text
{{ for item in items }}
{{ item.name }}
{{ end }}
```

- Partials can be passed explicitly through options:

```tya
Template.render("{{ partial \"row\" user }}", data, {
  partials: { row: "<li>{{ name }}</li>" }
})
```

- File-based partial lookup is optional and must be rooted by an explicit
  `partials_dir` option.
- HTML escaping is available through `escape: "html"`.
- Plain text rendering is the default: `escape` defaults to `"none"` so generic
  templates do not unexpectedly rewrite config files or code output.
- `Template.render_html(source, data)` is a convenience wrapper equivalent to
  `Template.render(source, data, { escape: "html" })`.
- Raw insertion for trusted HTML uses an explicit marker:

```text
{{{ trusted_html }}}
```

- Template syntax errors include line and column when feasible.
- `net/http` template support should use this package rather than defining a
  separate HTTP-only template language.

## Scope

- `lib/template/Template.tya`
- `docs/STDLIB.md`
- next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`
- script tests under `tests/testdata/`
- stdlib tests for `Template.render`, `Template.render_file`,
  `Template.render_html`, conditionals, loops, partials, escaping, and errors
- `ROADMAP.md`
- `net/http` docs/spec wording only enough to point to `template.Template`

## Out of Scope

- New language syntax.
- Compile-time template macros.
- Streaming template rendering.
- Template inheritance / block overriding.
- Whitespace trimming syntax.
- User-defined functions/filters in the first version.
- Automatic filesystem partial lookup without an explicit root.
- Replacing Tya string interpolation.

## Acceptance Criteria

- `import template` exposes `template.Template`.
- `Template.render("Hello, {{ name }}", { name: "Tya" })` returns
  `"Hello, Tya"`.
- Dotted/indexed variable paths work for dictionaries and arrays.
- Missing values render as empty strings by default.
- Missing values raise a template error in strict mode.
- `if` / `else` / `end` conditionals render the expected branch.
- `for item in items` loops render each item in order.
- Explicit partials render and receive the selected data context.
- `Template.render_html` escapes `&`, `<`, `>`, `"`, and `'`.
- Triple-brace raw insertion bypasses HTML escaping only when explicitly used.
- `Template.render_file` reads a template file and renders it with the same
  semantics as `Template.render`.
- `net/http` template roadmap/docs refer to the generic `template` package
  instead of owning a separate template engine.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run TestV.*Script -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Open Questions

None.
