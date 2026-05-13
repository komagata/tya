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
| VS Code publish workflow exists | `.github/workflows/publish-vscode-extension.yml` |
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
| Publish VS Code support to Marketplace | Marketplace publisher access and `VSCE_PAT` / `vsce login`; tracked by `komagata/tya#1` |
| Publish VS Code support to Open VSX | Open VSX namespace access and `OVSX_PAT` / login; tracked by `komagata/tya#1` |
| Publish Emacs mode to MELPA | Pull request `melpa/melpa#10013` is open; maintainer review / merge pending; tracked by `komagata/tya#2` |
| Register Tree-sitter grammar with GitHub Linguist | Blocked before PR: Linguist requires a syntax grammar with an allowed license (`apache-2.0`, `bsd-2-clause`, `bsd-3-clause`, `cc0-1.0`, `isc`, `mit`, `mpl-2.0`, `ncsa`, `permissive`, `unlicense`, `wtfpl`, or `zlib`). The Tya repository currently has no project license, and the editor grammar metadata is `UNLICENSED`; tracked by `komagata/tya#3`. |

## Last Local Verification

```sh
scripts/verify_editor_assets.sh
```

## Published Repository Evidence

- Main commit with editor assets: `014d87f`
- Main commit with Node 24 CI opt-in: `3102793`
- Main commit with Tree-sitter sample parse verification: `f2ffb11`
- GitHub Actions `Editor assets` run for `f2ffb11`: `25769086945`, status:
  success.
- MELPA pull request: https://github.com/melpa/melpa/pull/10013
- Follow-up issues: `komagata/tya#1`, `komagata/tya#2`, `komagata/tya#3`
