---
status: completed
goal_ready: false
---

# Feature: Slim-Style HTML Template External Package

## Goal

Create an external Tya package that provides a Ruby Slim-style,
indentation-based HTML template renderer for application views, while keeping
HTML-specific template syntax out of the Tya standard library.

## Context

Tya already ships `template.Template` as a generic text-template stdlib package.
It supports `{{ ... }}` interpolation, conditionals, loops, partials, file
rendering, strict missing-value errors, and optional HTML escaping.

This feature is different: it defines an HTML-first template language where
indentation describes the DOM tree, tags are concise, and HTML escaping is the
default. The package should live in an external repository so the core language
and stdlib stay small, and so syntax can evolve independently from Tya releases.

Assumed repository and package identity:

- repository: `https://github.com/komagata/tya-slim`
- package name: `slim`
- import path: `import slim as slim`
- primary API: `slim.Template`

## Behavior

- Provide a pure-Tya package with this layout:

  ```text
  tya-slim/
    tya.toml
    src/slim/Template.tya
    tests/template_test.tya
    examples/
    README.md
  ```

- Applications consume the package through a git dependency:

  ```toml
  [dependencies]
  slim = { git = "https://github.com/komagata/tya-slim", tag = "v0.1.0" }
  ```

- Public API:

  ```tya
  import slim as slim

  html = slim.Template.render(source, data)
  html = slim.Template.render(source, data, options)
  html = slim.Template.render_file("views/index.slim", data)
  html = slim.Template.render_file("views/index.slim", data, options)
  ```

- `Template.render` returns an HTML string.
- `Template.render_file` reads a template file and renders it with identical
  semantics.
- Missing values render as an empty string by default.
- `{ strict: true }` turns missing values, invalid paths, unknown partials, and
  malformed template syntax into render errors.
- HTML escaping is enabled by default for interpolated text and attribute values.
- Raw HTML insertion must be explicit.

## Template Syntax

- Indentation defines parent/child nesting. Two spaces are canonical.
- Tabs are rejected with a clear syntax error.
- Mixed indentation widths are rejected when they make the tree ambiguous.
- Blank lines are ignored.
- Lines starting with `/` are template comments and do not render.
- `doctype html` renders `<!DOCTYPE html>`.
- A bare tag name renders an element:

  ```slim
  html
    body
      h1 Hello
  ```

  renders:

  ```html
  <html><body><h1>Hello</h1></body></html>
  ```

- CSS shorthand is supported:

  ```slim
  div#main.content.primary
  ```

  renders:

  ```html
  <div id="main" class="content primary"></div>
  ```

- Attributes follow the tag name as `key=value` pairs:

  ```slim
  a href=user.url title=user.name Link
  input type="checkbox" checked=user.active
  ```

- Attribute values can be:
  - quoted string literals,
  - dotted/indexed data paths such as `user.name` or `items[0].name`,
  - booleans.
- Boolean attributes render only when truthy:

  ```slim
  input disabled=user.locked
  ```

- Text content after a tag is escaped by default.
- `= path.or.expression` inserts escaped data.
- `== path.or.expression` inserts trusted raw HTML.
- `| text` inserts escaped text exactly after the marker.
- Inline interpolation inside text uses `#{path}` and is escaped:

  ```slim
  p Hello, #{user.name}
  ```

- Conditionals:

  ```slim
  - if user.admin
    p Admin
  - else
    p User
  ```

- Loops:

  ```slim
  - for item in items
    li = item.name
  ```

- Partials:

  ```slim
  == render "row", item
  ```

  Partials are resolved from an explicit `partials` option map first. File-based
  partials are allowed only when an explicit `partials_dir` option is provided.

- Supported expression grammar is intentionally small:
  - literals: strings, numbers, booleans, nil,
  - dotted/indexed paths,
  - unary `not`,
  - equality and inequality,
  - truthiness checks,
  - no arbitrary Tya code execution.

## Scope

- New external repository `komagata/tya-slim`.
- Pure Tya implementation of parser, AST, renderer, escaping, and file loading.
- `tya.toml` package manifest and git-dependency installation path.
- Unit tests for parsing, rendering, escaping, strict mode, partials, and errors.
- Example templates for layout, list rendering, form controls, and partials.
- README documenting installation, syntax, API, escaping, and security model.
- Optional docs in this repository that point users from the generic
  `template.Template` stdlib to the external `slim` package for HTML-specific
  view templates.

## Out of Scope

- Adding the package to Tya stdlib.
- Changing Tya language syntax.
- Native code.
- Full Ruby Slim compatibility.
- Arbitrary embedded Tya evaluation inside templates.
- Template inheritance, layout blocks, streaming rendering, or incremental DOM
  rendering in the first version.
- Whitespace-control syntax beyond indentation and line text rules.
- A central package registry or `tya publish`.

## Acceptance Criteria

- A separate `komagata/tya-slim` repository contains a valid Tya package
  manifest and importable `src/slim/Template.tya`.
- A Tya application can depend on the package with a git dependency, run
  `tya install`, and import `slim`.
- `slim.Template.render("p Hello", {})` returns `<p>Hello</p>`.
- Nested indentation renders nested HTML in the documented order.
- `#id` and `.class` shorthand render correct `id` and `class` attributes.
- Attribute values and interpolated text are HTML-escaped by default.
- `==` raw insertion bypasses escaping only where explicitly used.
- `- if` / `- else` and `- for` render the expected branches and loop items.
- Missing values render as empty strings by default and raise in strict mode.
- Template comments do not render.
- Invalid indentation, tabs, unclosed control blocks, and malformed attributes
  produce line/column errors.
- File-based partial rendering is rooted by `partials_dir` and cannot escape
  that root with `..`.
- The package test suite passes through `tya test`.
- A fixture app in tests proves git/path dependency consumption works from
  outside the package repository.

## Verification

```sh
tya install
tya test
tya run examples/basic.tya
tya run examples/form.tya
```

For this repository's spec tracking only:

```sh
test -f feature-specs/slim-html-template-external-package.md
rg -n "Slim-Style HTML Template External Package" feature-specs/slim-html-template-external-package.md
```

## Dependencies

- Requires existing Tya package git dependency support.
- Reuses ordinary Tya file I/O for `render_file` and file-based partials.
- Can optionally reuse ideas from stdlib `template.Template`, but should not
  depend on private stdlib internals.

## Open Questions

None.
