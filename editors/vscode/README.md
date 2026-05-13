# tya — VS Code extension

Language support for [tya](https://github.com/komagata/tya). Provides
TextMate syntax highlighting plus a LSP-based language client that talks to
`tya lsp` (shipped with the tya compiler since v0.52).

## Requirements

- VS Code 1.75 or later
- `tya` v0.52 or later on `PATH` (or configure `tya.executable`)

## Features (v0.61)

- Diagnostics on save / on change
- TextMate syntax highlighting for `.tya`
- `textDocument/formatting` and `textDocument/rangeFormatting`
- `textDocument/hover` — function signatures + leading doc comments
- `textDocument/definition` — cross-file via `import`
- `textDocument/references`
- `textDocument/rename`
- `textDocument/codeAction` — TYAL0001 / TYAL0003 quick fixes
- `textDocument/semanticTokens/full`
- `textDocument/documentSymbol`
- `workspace/symbol`

## Manual install (until Marketplace publication)

Download `tya-0.61.0.vsix` from:

https://github.com/komagata/tya/releases/tag/editors-vscode-v0.61.0

Then install it:

```sh
code --install-extension tya-0.61.0.vsix
```

Or build it locally:

```sh
cd editors/vscode
npm install
npm run compile
npx vsce package
code --install-extension tya-0.61.0.vsix
```

## Settings

- `tya.executable` (default `tya`) — path to the tya LSP server.
- `tya.trace.server` (`off` / `messages` / `verbose`) — LSP trace verbosity.
