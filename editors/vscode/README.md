# tya — VS Code extension

Language support for [tya](https://github.com/komagata/tya). Provides
syntax highlighting plus a LSP-based language client that talks to
`tya lsp` (shipped with the tya compiler since v0.52).

## Requirements

- VS Code 1.75 or later
- `tya` v0.52 or later on `PATH` (or configure `tya.executable`)

## Features (v0.52)

- Diagnostics on save / on change
- `textDocument/formatting` (the canonical formatter)
- `textDocument/hover` — function signatures + leading doc comment
- `textDocument/definition` — same-file goto-definition
- `textDocument/completion` — top-level bindings, stdlib modules,
  builtins, and keywords

Cross-file resolution, rename, references, and code actions are
queued for v0.53+.

## Manual install (until Marketplace publication)

```sh
cd editors/vscode
npm install
npm run compile
npx vsce package
code --install-extension tya-0.54.0.vsix
```

## Settings

- `tya.executable` (default `tya`) — path to the tya LSP server.
- `tya.trace.server` (`off` / `messages` / `verbose`) — LSP trace verbosity.
