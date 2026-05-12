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
| Vim / Neovim syntax coloring exists | `editors/vim/syntax/tya.vim` |
| Vim / Neovim filetype and indent exist | `editors/vim/ftdetect/tya.vim`, `editors/vim/indent/tya.vim` |
| Emacs syntax coloring exists | `editors/emacs/tya-mode.el` |
| MELPA submission asset exists | `editors/emacs/melpa-recipe` |
| GitHub Linguist submission assets exist | `editors/github-linguist/languages.yml.example`, `editors/tree-sitter-tya/` |
| Tree-sitter grammar generates | `tree-sitter generate` in `editors/tree-sitter-tya` |
| Repository validation covers editor assets | `tests/editor_assets_test.go` |
| CI validates editor assets | `.github/workflows/editor-assets.yml` |

## External Work Still Required

These steps require account credentials or upstream review and are not complete
from repository changes alone.

| Requirement | Blocking dependency |
|---|---|
| Publish VS Code support to Marketplace | Marketplace publisher access and `VSCE_PAT` / `vsce login` |
| Publish VS Code support to Open VSX | Open VSX namespace access and `OVSX_PAT` / login |
| Publish Emacs mode to MELPA | Pull request to `melpa/melpa` and maintainer review |
| Register Tree-sitter grammar with GitHub Linguist | Pull request to `github-linguist/linguist` and maintainer review |

## Last Local Verification

```sh
scripts/verify_editor_assets.sh
```
