---
layout: doc
title: Spec
permalink: /v0.52/spec/
---

# Tya v0.52 Specification

> **Status:** shipped. The `tya version` constant is `0.52.0`.
> v0.52 adds the `tya lsp` Language Server. The language surface
> is unchanged from v0.49 / v0.50 / v0.51.

## Theme

v0.51 closed the documentation tooling slice with `tya doc`. v0.52
adds the fifth toolchain subcommand: **`tya lsp`**, a Language
Server Protocol (LSP) implementation that ships in the same
binary. Editors (VS Code, Helix, Neovim, Emacs, Zed) can now
surface diagnostics, formatting, hover, go-to-definition, and
completion against tya source without any extra build step.

A VS Code extension scaffold lives at `editors/vscode/` and ships
as part of the repository (manual install via `npx vsce package`
in v0.52; Marketplace publication queued for v0.53+).

## `tya lsp` — Language Server (stdio JSON-RPC)

### CLI

```
tya lsp [--log <file>]
```

- Speaks LSP JSON-RPC 2.0 over stdio.
- `--log <file>` writes timestamped debug lines to a file. Without
  it no logs are emitted; stderr stays clean so editors that capture
  stderr do not see spurious output.
- Exit codes: `0` clean shutdown, `1` runtime I/O failure
  (`TYA-E0931`), `2` argument failure (`TYA-E0930`).

### Capabilities advertised

```json
{
  "textDocumentSync": { "openClose": true, "change": 1, "save": { "includeText": false } },
  "hoverProvider": true,
  "definitionProvider": true,
  "completionProvider": { "resolveProvider": false },
  "documentFormattingProvider": true
}
```

`change = 1` means `TextDocumentSyncKind.Full`. The server stores
each `didChange` text in full; incremental sync is v0.53+ work.

### Lifecycle methods

| Method | Direction | Behaviour |
|--------|-----------|-----------|
| `initialize` | request | Returns `serverInfo.{name:"tya", version}` and the capabilities object. |
| `initialized` | notification | Acknowledged silently. |
| `shutdown` | request | Returns `null`. Sets the shutting-down flag. |
| `exit` | notification | Ends the server loop (exit code 0). |

### Document sync

- `textDocument/didOpen` — stores text + publishes diagnostics.
- `textDocument/didChange` — replaces text + republishes diagnostics.
- `textDocument/didSave` — refreshes diagnostics (text reused as-is unless `includeText` was true).
- `textDocument/didClose` — drops the document and publishes an empty diagnostic list to clear markers.

### Diagnostics (publishDiagnostics)

Every store mutation triggers a synchronous re-check. Pipeline:

1. `lexer.LexWithComments`
2. `parser.ParseWithComments`
3. `checker.CheckAll(prog, nil, path, true)` (no modules in MVP)
4. `parser.OrphanComments`
5. `checker.CollectUnused` (TYAL0001 surfaced as warnings)
6. `checker.CollectLintFindings` (TYAL0003 / TYAL0004 / TYAL0005)

Severity mapping: tya `diag.Error` → LSP `1`, `diag.Warning` → `2`.
Each finding's `code` matches the tya diagnostic registry
(`TYA-E…`, `TYAL…`). The `source` is always `"tya"`.

### Formatting (textDocument/formatting)

`formatter.Unparse(prog)` is run on the current text. The response
is one full-document TextEdit, or an empty edit list when the
buffer is already canonical. Range formatting is v0.53+.

### Hover (textDocument/hover)

The identifier under the cursor is resolved against a symbol index
built from the current file. For local top-level bindings, the
response contains the signature in a `tya` code fence plus the
leading `#`-comment block. Stdlib module names and builtin
function names are recognised and produce a short generic blurb.

### Definition (textDocument/definition)

Same-file resolution only in v0.52. The token position of the
defining declaration's name is returned as a single Location.
Cross-file lookup via `import` is v0.53+.

### Completion (textDocument/completion)

Flat list of candidates is returned every time. Sources:

1. Top-level bindings in the current file (kind: function / class / module / interface).
2. Standard attached library module names (kind: module).
3. Builtin function names (kind: function).
4. Language keywords (kind: keyword).

The client filters by prefix. `triggerCharacters` is empty in v0.52.

### Positions

LSP positions are 0-origin `(line, character)`. tya internally uses
1-origin tokens; the LSP boundary converts in both directions.

`character` is defined by LSP as UTF-16 code units. tya identifiers
are ASCII, so v0.52 treats `character` as a byte offset. UTF-16
strictness is queued for v0.53+ alongside rename/range-formatting.

### Scope (single-file MVP)

- No `import` traversal; diagnostics and symbol resolution operate
  on the open buffer only.
- No incremental document sync (`Change: 1` = Full).
- No workspace symbols / document outline.

These deferrals are intentional. See §Non-goals.

## VS Code extension (`editors/vscode/`)

A small TypeScript wrapper around `vscode-languageclient` ships in
the repository. It registers the `tya` language, contributes a
TextMate grammar for syntax highlighting, and launches `tya lsp`
as the LSP server.

### Manual install

```sh
cd editors/vscode
npm install
npm run compile
npx vsce package
code --install-extension tya-0.52.0.vsix
```

Marketplace publication (publisher ID, signed VSIX, icon) is queued
for v0.53+.

### Settings

| Setting | Default | Effect |
|---------|---------|--------|
| `tya.executable` | `"tya"` | Path to the tya binary used as the LSP server. |
| `tya.trace.server` | `"off"` | Built-in `vscode-languageclient` tracing. |

### Syntax grammar coverage

The shipped `syntaxes/tya.tmLanguage.json` covers comments,
strings, numbers, keywords (control / declaration), language
literals, function-call positions, and operators. Semantic
highlighting is v0.53+.

## Error codes added

| Code | Subcommand | Meaning |
|------|------------|---------|
| `TYA-E0930` | `tya lsp` | Startup / argument failure (bad `--log`, unknown flag) |
| `TYA-E0931` | `tya lsp` | I/O failure (broken pipe, framing error) |

## Non-goals for v0.52

- Rename / references / code actions / range formatting
- Cross-file definition resolution
- Incremental document sync
- Marketplace publication of the VS Code extension
- Setup recipes for other editors (Helix / Neovim / Zed) — covered
  in v0.53+.
- Semantic tokens / document outline.

## Compatibility

- Language surface: unchanged from v0.49 / v0.50 / v0.51.
- `tya.toml` schema: unchanged.
- CLI: every existing subcommand keeps its v0.51 behaviour. The
  `tya lsp` subcommand is purely additive.

## Tests

Seven Go subprocess tests live in `tests/lsp_test.go`:

1. `TestLSPInitialize` — capabilities advertised.
2. `TestLSPDiagnosticsOnDidOpen` — bad source ⇒ ≥1 diagnostic.
3. `TestLSPFormatting` — `formatter.Unparse` round-trip.
4. `TestLSPHover` — function signature visible.
5. `TestLSPDefinition` — same-file location.
6. `TestLSPCompletion` — builtins / stdlib / keywords surfaced.
7. `TestLSPShutdownExit` — clean shutdown path.

`tests/lsp_helper.go` builds the binary once via `os.MkdirTemp` so
the seven cases share the build cost.
