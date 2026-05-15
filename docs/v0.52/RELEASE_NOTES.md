---
layout: doc
title: Release Notes
permalink: /v0.52/release-notes/
---

# Tya v0.52 Release Notes

> **Status:** shipped. `tya version` reports `0.52.0` and
> `ROADMAP.md` carries the matching `Released` entry.

## TL;DR

v0.52 adds the fifth toolchain subcommand: **`tya lsp`**, an LSP
(Language Server Protocol) server. The same `tya` binary now
implements the language server, so the compiler and the editor
never drift out of sync.

A VS Code extension scaffold ships in `editors/vscode/` for the
manual install path; Marketplace publication is queued for v0.53+.

The language surface is unchanged from v0.49 / v0.50 / v0.51.

## What's new

### `tya lsp`

```sh
$ tya lsp [--log /tmp/tya-lsp.log]
```

LSP JSON-RPC 2.0 over stdio. v0.52 advertises:

- `textDocumentSync` = Full
- `hoverProvider`
- `definitionProvider`
- `completionProvider`
- `documentFormattingProvider`

Diagnostics fire on `didOpen` / `didChange` / `didSave` and cover
the same surface as the `tya check` and `tya lint` CLIs.

### VS Code extension (`editors/vscode/`)

Minimal TypeScript wrapper around `vscode-languageclient`. Ships
with:

- Static TextMate grammar for syntax highlighting
- Settings `tya.executable` + `tya.trace.server`
- `package.json` manifest at `version: 0.52.0`

Install manually:

```sh
cd editors/vscode
npm install
npm run compile
npx vsce package
code --install-extension tya-0.52.0.vsix
```

## Compatibility

- Language: unchanged from v0.49 / v0.50 / v0.51.
- `tya.toml` schema: unchanged.
- CLI: every existing subcommand keeps its v0.51 behaviour. The
  `tya lsp` subcommand is purely additive.

## Migration

Nothing required. To opt in:

1. `brew install komagata/tap/tya` (or upgrade) to a v0.52 binary.
2. Build the VS Code extension once: `cd editors/vscode && npm install && npm run compile && npx vsce package && code --install-extension tya-0.52.0.vsix`.
3. Open any `.tya` file. Diagnostics, hover, goto-definition, and
   format-on-save should work out of the box.

## Implementation notes

- New package `internal/lsp/` with 14 source files plus a unit-test file.
- JSON-RPC framing (`rpc.go`) is hand-rolled on top of `encoding/json`. No new external dependencies. `go.mod` is unchanged.
- LSP protocol types (`protocol.go`) are hand-written, minimal, and live in one file so future fields are one-line additions.
- Diagnostics pipeline reuses `checker.CheckAll`, `parser.OrphanComments`, `checker.CollectUnused`, and `checker.CollectLintFindings` so the LSP server stays semantically identical to `tya check` + `tya lint`.
- Formatting routes directly through `formatter.Unparse`.
- Hover/definition/completion build a fresh `SymbolIndex` per request — programs are small enough that the rebuild cost beats incremental bookkeeping.
- `internal/checker.BuiltinNames()` is now a public accessor (used by completion).
- `internal/doc.FuncSignature` was renamed from `funcSignature` so hover can render canonical signatures.

## Tests

`tests/lsp_test.go` exercises seven subprocess scenarios (build a
binary once with `os.MkdirTemp`, then run all cases against it).

## Looking ahead (v0.53+ candidates)

From `ROADMAP.md` § Future Work § Toolchain:

- Cross-file definition resolution (follow `import`)
- Rename / references
- Range formatting (AST slice unparser)
- Code actions (TYAL quick fixes)
- Semantic tokens
- VS Code Marketplace publication (publisher ID, signed VSIX)
- Helix / Neovim / Zed setup recipes
- Incremental document sync

Self-host work (`ROADMAP.md` § Scheduled M8/M9/M10) remains
deferred until the v1.0.0 prep window.
