# Tya v0.53 Release Notes

> **Status:** shipped. `tya version` reports `0.53.0` and
> `ROADMAP.md` carries the matching `Released` entry.

## TL;DR

v0.53 promotes `tya lsp` from MVP to a full IDE-grade server:
cross-file definition, references, rename, range formatting, code
actions, semantic tokens, document and workspace symbols, plus
incremental document sync. Setup recipes ship for Neovim, Zed, and
Emacs. The language surface is unchanged from v0.52.

## What's new

### LSP feature additions

- **`textDocument/definition`** now follows `import` statements
  and `mod.foo` member access.
- **`textDocument/references`** returns every occurrence,
  workspace-wide for top-level bindings and function-scoped for
  local / param.
- **`textDocument/rename`** produces a `WorkspaceEdit` honouring
  the same scope rules. Conflicts are rejected with
  `[TYA-E0933]`.
- **`textDocument/rangeFormatting`** runs the canonical formatter
  on the smallest top-level Stmt window enclosing the request.
- **`textDocument/codeAction`** offers quick fixes for
  `TYAL0001 unused local` (line-delete) and
  `TYAL0003 redundant if true/false` (unwrap).
- **`textDocument/semanticTokens/full`** emits the LSP delta
  encoding for a 9-type legend.
- **`textDocument/documentSymbol`** returns the hierarchical
  outline (class → methods, module → members).
- **`workspace/symbol`** filters every top-level symbol in
  workspace `*.tya` files by case-insensitive substring.
- **Incremental document sync** (`TextDocumentSyncKind.Incremental`)
  is advertised. `positionEncoding` is `"utf-8"`.

### New editor recipes

| Editor | Path |
|--------|------|
| Neovim (nvim-lspconfig) | [`editors/neovim/`](../editors/neovim) |
| Zed | [`editors/zed/`](../editors/zed) |
| Emacs (eglot / lsp-mode) | [`editors/emacs/`](../editors/emacs) |

Each recipe is a short README plus a single drop-in config file
that points the editor at the system `tya` binary as the LSP server.

### VS Code extension

`editors/vscode/package.json` is bumped to `0.53.0`. The README and
manifest already pick up every new feature via
`vscode-languageclient` because all changes are server-side.
Manual install instructions remain identical:

```sh
cd editors/vscode
npm install
npm run compile
npx vsce package
code --install-extension tya-0.53.0.vsix
```

Marketplace publication (publisher registration, signed VSIX,
icon, GH Actions release pipeline) is queued for v0.54+.

## Compatibility

- Language surface: unchanged from v0.49 — v0.52.
- `tya.toml` schema: unchanged.
- CLI: every existing subcommand keeps its v0.52 behaviour. All
  new LSP methods are additive.

## Migration

Nothing required. Optional:

1. Upgrade to v0.53 (`brew install komagata/tap/tya` or build from
   source).
2. Re-build the VS Code extension once with `npm run compile &&
   npx vsce package` and install the new VSIX. (No `package.json`
   changes mean an old VSIX still works against the new server.)
3. New editor recipes:
   - Neovim: copy `editors/neovim/init.lua.example` into your config.
   - Zed: merge `editors/zed/extension.json.example` into
     `~/.config/zed/settings.json`.
   - Emacs: copy `editors/emacs/setup.el.example`.

## Implementation notes

- New package files under `internal/lsp/`: `workspace.go`,
  `references.go`, `rename.go`, `range_format.go`,
  `code_actions.go`, `semantic_tokens.go`,
  `document_symbols.go`, `workspace_symbols.go`, `incremental.go`,
  `finder_v2.go`, `protocol_v2.go`.
- `internal/checker/autofix.go` exposes
  `LintAutofixHints(prog)` for the code-action handler.
- `Workspace` is lazy — it only parses files on demand and caches
  the result so cross-file rename / references stay incremental
  after the first scan.
- All new functionality is reachable through the existing
  `cmd/tya/lsp.go` subcommand. No CLI changes.

## Tests

`tests/lsp_v2_test.go` adds 11 subprocess scenarios that share the
`tests/lsp_helper.go` harness from v0.52. The 7 original v0.52
cases continue to pass.

## Looking ahead (v0.54+ candidates)

From `ROADMAP.md` § Future Work § Toolchain:

- VS Code Marketplace publication
- `prepareRename` (rename preview)
- Range formatting at AST slice precision
- Semantic token modifiers
- Inlay hints, call hierarchy, selection range, code lens,
  folding range, document link
- Toolchain track other Epics (`tya lint` v3, `tya task` v2,
  `tya doc` v2, diagnostics pipeline migration)
- Language-track Epics (raw `"` inside `{expr}` interpolation,
  primitive class sugar, interface stackable trait surface,
  WebAssembly target)

Self-host work (`ROADMAP.md` § Scheduled M8 / M9 / M10) remains
deferred until the v1.0.0 prep window.
