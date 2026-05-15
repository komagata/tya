---
layout: doc
title: Release Notes
permalink: /v0.51/release-notes/
---

# Tya v0.51 Release Notes

> **Status:** shipped. `tya version` reports `0.51.0` and
> `ROADMAP.md` carries the matching `Released` entry.

## TL;DR

v0.51 adds the fourth toolchain subcommand: **`tya doc`**, a
source documentation generator. It walks `src/` (or any paths you
pass), extracts top-level declarations with their leading
`#`-comment block as Markdown, and produces either a plain-text
dump or a multi-page static HTML site.

The language surface is unchanged from v0.49 / v0.50.

## What's new

### `tya doc`

Text output (default):

```sh
$ tya doc
## function greet
    greet(name)
    src/greet.tya:2

    Returns a greeting for the given name.
```

HTML output:

```sh
$ tya doc --html ./docs/api
$ ls docs/api
index.html  items/  style.css
```

`docs/api/index.html` groups bindings by kind (modules, classes,
interfaces, functions). Each binding has its own page under
`items/<kind>_<name>.html` showing the signature, source location,
and rendered Markdown body.

### What gets documented

- **Top-level** `class`, `module`, `interface`, and `function`
  declarations (function = `name = … -> …` assignment whose value
  is a function literal).
- A `#`-comment block immediately above the declaration, at the
  same indentation level, becomes the doc body.
- Bindings whose name starts with `_` are skipped (convention:
  leading underscore = private).

### Markdown support

The renderer is self-contained (no new dependency). It covers:

- Headings (`#` through `######`)
- Paragraphs (blank-line separated)
- Fenced code blocks
- `- ` and `1. ` lists at a single indent
- Inline `` `code` ``, `[link](url)`, `**bold**`, `*italic*`
- All raw text is HTML-escaped before inline rewriting

## Compatibility

- Language: unchanged from v0.49 / v0.50.
- `tya.toml` schema: unchanged.
- CLI: every existing subcommand keeps its v0.50 behavior. The
  `tya doc` subcommand is purely additive.

## Migration

Nothing required. Optional:

1. Add doc comments above your top-level declarations so
   `tya doc` produces something meaningful.
2. Run `tya doc --html ./docs/api` to publish HTML pages alongside
   your project.

## Implementation notes

- New package `internal/doc/`:
  - `extract.go` — walks `parser.ParseWithComments` output, pulls
    `StmtComments.Leading` for each top-level declaration.
  - `markdown.go` — self-contained Block parser and HTML/text
    renderer.
  - `text.go` — `tya doc` (no flags) formatter.
  - `html.go` — `Site.Generate()` writes `index.html`,
    `items/<kind>_<name>.html`, and `style.css` (snapshot of
    `docs/document.css` kept as a Go constant).
- CLI wiring lives in `cmd/tya/doc.go`. `cmd/tya/main.go::docCommand`
  is the entry point and the `usage()` function lists the new
  invocation form.
- `DocItem.Signature` reflects parameter names only; type
  information is not present in the AST and therefore not shown.

## Looking ahead (v0.52+ candidates)

From `ROADMAP.md` § Future Work § Toolchain:

- `tya doc --serve` (local HTTP server with autoreload)
- `tya doc --json` (machine-readable output)
- Stdlib re-export (follow `import` statements into stdlib)
- Class member detail pages (methods, fields, `private` filter)
- Cross-binding links

Self-host work (`ROADMAP.md` § Scheduled M8/M9/M10) remains
deferred until the v1.0.0 prep window.
