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
| VS Code manual-install package is published | GitHub Release `editors-vscode-v0.61.0` with `tya-0.61.0.vsix` |
| VS Code publish workflow exists | `.github/workflows/publish-vscode-extension.yml` |
| Vim / Neovim syntax coloring exists | `editors/vim/syntax/tya.vim` |
| Vim / Neovim filetype and indent exist | `editors/vim/ftdetect/tya.vim`, `editors/vim/indent/tya.vim` |
| Emacs syntax coloring exists | `editors/emacs/tya-mode.el` |
| MELPA submission asset exists | `editors/emacs/melpa-recipe` |
| GitHub Linguist submission assets exist | `editors/github-linguist/languages.yml.example`, `editors/tree-sitter-tya/` |
| Standalone Tree-sitter grammar repository exists | https://github.com/komagata/tree-sitter-tya |
| Tree-sitter grammar has an allowed Linguist license | `editors/tree-sitter-tya/LICENSE`, `editors/tree-sitter-tya/package.json`, `editors/tree-sitter-tya/tree-sitter.json` use MIT |
| Tree-sitter grammar generates | `tree-sitter generate` in `editors/tree-sitter-tya` |
| Manual editor asset bundle is published | GitHub Release `editors-assets-v0.61.0` with `tya-editor-assets-v0.61.0.tar.gz` |
| Repository validation covers editor assets | `tests/editor_assets_test.go` |
| CI validates editor assets | `.github/workflows/editor-assets.yml` |

## External Work Still Required

These steps require account credentials or upstream review and are not complete
from repository changes alone.

| Requirement | Blocking dependency |
|---|---|
| Publish VS Code support to Marketplace | Marketplace publisher access and `VSCE_PAT` / `vsce login`; tracked by `komagata/tya#1` |
| Publish VS Code support to Open VSX | Open VSX namespace access and `OVSX_PAT` / login; tracked by `komagata/tya#1` |
| Publish Emacs mode to MELPA | Pull request `melpa/melpa#10013` is open; maintainer review / merge pending; tracked by `komagata/tya#2` |
| Register Tree-sitter grammar with GitHub Linguist | Blocked before PR: GitHub code search for `extension:tya -is:fork` reports 124 results, below Linguist's new-language usage threshold; tracked by `komagata/tya#3`. |

## Last Local Verification

```sh
scripts/verify_editor_assets.sh
```

Last observed GitHub Actions verification:

```text
Editor assets / main / 25769943202 / success / 2026-05-13T00:14:24Z
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
- Latest observed GitHub Actions `Editor assets` run: `25769943202`, status:
  success.
- VS Code manual-install release:
  https://github.com/komagata/tya/releases/tag/editors-vscode-v0.61.0
  (`tya-0.61.0.vsix`, sha256
  `305322bcae342e81db145297329a9941e89eb6ed52c23afbe5812e59b4d3b67d`)
- Manual editor asset bundle:
  https://github.com/komagata/tya/releases/tag/editors-assets-v0.61.0
  (`tya-editor-assets-v0.61.0.tar.gz`, sha256
  `87e2c78cf2d5a1fc224780d1f0db1dc2870ae008fcd85e584bd0159af49c8f8f`)
- MELPA pull request: https://github.com/melpa/melpa/pull/10013
- Standalone Tree-sitter grammar repository: https://github.com/komagata/tree-sitter-tya
- Follow-up issues: `komagata/tya#1`, `komagata/tya#2`, `komagata/tya#3`
