# Syntax Coloring Ship Status

Objective: ship syntax coloring for major editors.

## Completed In Repository

| Requirement | Evidence |
|---|---|
| Required targets are defined | `ROADMAP.md` lists VS Code, Emacs, Vim, and GitHub. |
| Shared token taxonomy exists | `editors/TOKENS.md` |
| Shared syntax fixture exists | `editors/syntax-sample.tya` |
| VS Code syntax coloring exists | `editors/vscode/syntaxes/tya.tmLanguage.json`, registered from `editors/vscode/package.json` |
| VS Code package builds | `npm run compile` and `npm run package` in `editors/vscode` |
| VS Code manual-install package is published | GitHub Release `editors-vscode-v0.65.0` with `tya-0.65.0.vsix` |
| VS Code Marketplace package is published | `komagata.tya` version `0.65.0` on Visual Studio Marketplace |
| Open VSX package is published | `komagata.tya` version `0.65.0` on Open VSX |
| VS Code publish workflow exists | `.github/workflows/publish-vscode-extension.yml` |
| Vim / Neovim syntax coloring exists | `editors/vim/syntax/tya.vim` |
| Vim / Neovim filetype and indent exist | `editors/vim/ftdetect/tya.vim`, `editors/vim/indent/tya.vim` |
| Emacs syntax coloring exists | `editors/emacs/tya-mode.el` |
| Manual editor asset bundle is published | GitHub Release `editors-assets-v0.61.0` with `tya-editor-assets-v0.61.0.tar.gz` |
| Repository validation covers editor assets | `tests/editor_assets_test.go` |
| CI validates editor assets | `.github/workflows/editor-assets.yml` |

## Deferred Follow-up

These publishing integrations are intentionally outside the current syntax
coloring ship scope.

| Requirement | Status |
|---|---|
| Publish Emacs mode to MELPA | Pull request `melpa/melpa#10013` is open; maintainer review / merge pending; tracked by `komagata/tya#2` |
| Register Tree-sitter grammar with GitHub Linguist | GitHub code search for `extension:tya -is:fork` reports 124 results, below Linguist's new-language usage threshold; tracked by `komagata/tya#3`. |

## Last Local Verification

```sh
scripts/verify_editor_assets.sh
```

Last observed GitHub Actions verification:

```text
Publish VS Code extension / main / 25953906444 / success / 2026-05-16T05:29:57Z
```

## Published Repository Evidence

- Main commit with editor assets: `014d87f`
- Main commit with Node 24 CI opt-in: `3102793`
- Main commit with Tree-sitter sample parse verification: `f2ffb11`
- Main commit with manual VS Code publish workflow: `94ecbdb`
- Main commit with publishing follow-up issues: `d8af793`
- Main commit with VS Code manual-install release docs: `fe93c82`
- Main commit with manual editor asset bundle docs: `3aad8f2`
- Main commit licensing the Tree-sitter grammar as MIT: `018f8a6`
- Latest observed GitHub Actions `Publish VS Code extension` Marketplace run:
  `25953906444`, status: success.
- Latest observed GitHub Actions `Publish VS Code extension` Open VSX run:
  `25953906444`, status: success.
- VS Code manual-install release:
  https://github.com/komagata/tya/releases/tag/editors-vscode-v0.65.0
  (`tya-0.65.0.vsix`, sha256
  `667e9205153615b893484ea091cd8bc04ab41ee5df55533264621b0f9ad09216`)
- Manual editor asset bundle:
  https://github.com/komagata/tya/releases/tag/editors-assets-v0.61.0
  (`tya-editor-assets-v0.61.0.tar.gz`, sha256
  `87e2c78cf2d5a1fc224780d1f0db1dc2870ae008fcd85e584bd0159af49c8f8f`)
- Visual Studio Marketplace extension: `komagata.tya` version `0.65.0`
- Open VSX extension: `komagata.tya` version `0.65.0`
- MELPA pull request: https://github.com/melpa/melpa/pull/10013
- Standalone Tree-sitter grammar repository: https://github.com/komagata/tree-sitter-tya
- Follow-up issues: `komagata/tya#1`, `komagata/tya#2`, `komagata/tya#3`
