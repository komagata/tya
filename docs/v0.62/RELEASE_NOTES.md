---
layout: doc
title: Release Notes
permalink: /v0.62/release-notes/
---

# Tya v0.62 Release Notes

v0.62 is the current released implementation. It finishes the `net/http` v2
surface, extends `tya lint`, and separates editable Markdown documentation from
the generated public website.

## Standard Library

- `net/http.Client` provides practical client helpers for `GET`, `POST`,
  `PUT`, `PATCH`, `DELETE`, and `HEAD` requests.
- `net/http.Response` wraps response status, headers, body, JSON parsing, and
  success checks.
- `net/http.Server` keeps the v0.61 routing extensions and now runs accepted
  connections through cooperative tasks so concurrent requests do not serialize
  behind one handler.

## Linter

- New `TYAL0006` rule reports suspicious array `for` bindings where an
  index-like name appears before the value binding.
- New `TYAL0007` rule reports unused function parameters. Use `_` for
  intentionally ignored parameters.
- New `TYAL0008` rule reports shadowed bindings in the same or an outer lexical
  scope.
- `tya lint --format=sarif` emits SARIF 2.1.0 for code-scanning tools.
- JSON and SARIF output include each rule's stable title and documentation URL.
- File-scope opt-outs are available with `# tya-lint-ignore-file: TYAL0001`.
- `TYAL0001 --fix` removes full multi-line binding blocks instead of leaving
  orphan indented lines.
- The CLI autofix path reuses the same unwrap-if rewrite hints as the LSP code
  action path.

## Documentation and Website

- The editable Markdown documentation remains under `docs/`.
- The generated GitHub Pages website now lives under `site/`.
- GitHub Pages deploys from the `site/` artifact built by the
  `GitHub Pages` workflow.
- The generated website includes the new Lint reference page at
  `https://tya-lang.org/lint.html`.

## Verification

The release gate is:

```sh
go test ./... -count=1
```

The published v0.62.0 tag passed the full suite, including the maintained
self-host fixed-point tests.
