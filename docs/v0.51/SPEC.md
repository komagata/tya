---
layout: doc
title: Spec
permalink: /v0.51/spec/
---

# Tya v0.51 Specification

> **Status:** shipped. The `tya version` constant is `0.51.0`.
> v0.51 adds the `tya doc` source documentation generator. The
> language surface is unchanged from v0.49 / v0.50.

## Theme

v0.49 cut the first slice of the Toolchain track (`tya new`,
`tya task`, `tya lint`). v0.50 extended those three. v0.51 adds the
fourth toolchain subcommand: `tya doc`, a source-documentation
generator that produces either plain text or a multi-page static
HTML site.

The goal of v0.51 is to round out the toolchain so a tya project
can be **documented** out of the box, completing the
"scaffold → run → test → lint → document" loop.

## `tya doc` — source documentation generator

### CLI

```
tya doc [paths...]              # text output to stdout
tya doc --html <out> [paths...] # multi-page HTML static site
```

- `paths` defaults to `src/` when omitted. If `src/` does not
  exist, tya exits 2 with `[TYA-E0923]`.
- Walking is recursive; every `*.tya` file under the path is
  considered.
- Project sources only. The v0.51 generator does **not** follow
  imports into the standard library or third-party packages
  (queued for v0.52+).

### Doc comment format

- The block of contiguous line-leading `#` comments immediately
  before a top-level declaration becomes its **doc comment**. This
  reuses the comment-attachment pipeline from v0.34
  (`parser.ParseWithComments`'s `StmtComments.Leading`).
- No dedicated syntax (`##`, `#:`, etc.). The body is interpreted
  as Markdown.
- A blank line between the comment block and the declaration
  detaches the comment.

### Extracted top-level declarations

| AST node | Kind | Signature string |
|----------|------|------------------|
| `ClassDecl` | `class` | `class <Name>` |
| `ModuleDecl` | `module` | `module <Name>` |
| `InterfaceDecl` | `interface` | `interface <Name>` |
| `AssignStmt` whose RHS is `*ast.FuncLit` | `function` | `<name>(<param1>, <param2>...)` |

- Top-level bindings whose name starts with `_` are excluded
  (convention: leading underscore = private).
- Class members (`private`, `static`, fields, methods) are **not**
  expanded into the index in v0.51. Class member documentation is
  queued for v0.52+.

### Markdown subset

The Markdown renderer is self-contained (no external dependency).
Supported elements:

- ATX headings (`# … ######`)
- Blank-line separated paragraphs
- Fenced code blocks (` ```lang `)
- Unordered list (`- `) and ordered list (`1. `) at a single
  indentation level
- Inline: `` `code` ``, `[label](url)`, `**bold**`, `*italic*`

Plain-text output renders headings as `=== … ===`, lists as
`- …` / `1. …`, code fences as indented blocks. Inline markers
are stripped.

HTML output renders the corresponding tags. All raw text is HTML-
escaped before inline rewriting, so user content cannot inject
markup.

### Error codes

| Code | Meaning | Exit |
|------|---------|------|
| `TYA-E0920` | `--html` requires a non-empty directory argument | 2 |
| `TYA-E0923` | `src/` not found (no explicit paths) | 2 |

`TYA-E0921` and `TYA-E0922` are reserved (orphan-doc-comment and
Markdown parse error). They are not implemented in v0.51.

### HTML site layout

```
<out>/
  index.html                      # grouped binding listing
  items/
    function_<sanitized_name>.html
    class_<sanitized_name>.html
    module_<sanitized_name>.html
    interface_<sanitized_name>.html
  style.css                       # snapshot of docs/document.css
```

- File names use the form `<kind>_<sanitized_name>.html` so a
  `Foo` class and a `foo` function never collide on
  case-insensitive filesystems.
- Duplicate `(kind, name)` pairs across different files emit a
  warning on stderr and the last-write wins.
- The `style.css` source is the constant `defaultCSS` in
  `internal/doc/html.go`; keep it manually in sync with
  `docs/document.css`.

### Out of scope (v0.51)

- `--serve` HTTP server.
- `--json` machine-readable output.
- Stdlib / third-party documentation (import follow).
- Class member details (methods, fields, private filter).
- Cross-reference linking between bindings.

## Diagnostic code registry update

v0.51 adds two codes inside the `TYA-E092x` toolchain band:

| Code | Subcommand | Meaning |
|------|------------|---------|
| TYA-E0900 | `tya task` | unknown task name (v0.49) |
| TYA-E0901 | `tya task` | array-form task failure (v0.49) |
| TYA-E0902 | `tya task` | no `tya.toml` found (v0.49) |
| TYA-E0903 | `tya task` | parallel-form task failure (v0.50) |
| TYA-E0910 | `tya new`  | invalid project name (v0.49) |
| TYA-E0911 | `tya new`  | target already exists (v0.49) |
| TYA-E0912 | `tya new`  | invalid `--template` (v0.50) |
| TYA-E0913 | `tya new`  | `--here` + name conflict (v0.50) |
| TYA-E0920 | `tya doc`  | `--html` needs argument (v0.51) |
| TYA-E0923 | `tya doc`  | `src/` missing on default path (v0.51) |
| TYAL0001  | `tya lint` | unused local (v0.49; autofix v0.50) |
| TYAL0003  | `tya lint` | redundant `if true`/`false` (v0.50) |
| TYAL0004  | `tya lint` | deeply nested block (v0.50) |
| TYAL0005  | `tya lint` | very long function (v0.50) |

`TYA-E0921`, `TYA-E0922`, and `TYAL0002` remain reserved.

## Compatibility

- Language surface: unchanged from v0.49 (and v0.50).
- `tya.toml` schema: unchanged.
- CLI: no existing subcommand changed shape. `tya doc` is purely
  additive.
