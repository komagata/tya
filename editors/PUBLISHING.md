# Publishing Editor Syntax Support

This checklist records the external publishing work for Tya editor syntax
coloring. Marketplace and Open VSX publication are complete; MELPA and GitHub
Linguist are tracked as deferred follow-up work.

## Manual Editor Asset Bundle

- https://github.com/komagata/tya/releases/tag/editors-assets-v0.61.0

The bundle contains Emacs, Vim/Neovim, Tree-sitter, Linguist helper files, the
shared token taxonomy, and the shared syntax sample. This is a manual-install
distribution path; Marketplace, Open VSX, MELPA, and Linguist publication are
tracked separately below.

## VS Code Marketplace

Artifact:

```sh
scripts/verify_editor_assets.sh
```

Manual-install release:

- https://github.com/komagata/tya/releases/tag/editors-vscode-v0.65.2

Publish:

```sh
cd editors/vscode
npx vsce publish
```

Or run the GitHub Actions workflow `Publish VS Code extension` with
`target=marketplace` after configuring the `VSCE_PAT` repository secret.

Published package:

- `komagata.tya` version `0.65.2`

Requirements:

- Visual Studio Marketplace publisher: `komagata`
- `VSCE_PAT` or an interactive `vsce login komagata`
- Generated package: `editors/vscode/tya-0.65.2.vsix`

## Open VSX

Artifact:

```sh
scripts/verify_editor_assets.sh
```

Publish:

```sh
cd editors/vscode
npx ovsx publish
```

Or run the GitHub Actions workflow `Publish VS Code extension` with
`target=open-vsx` after configuring the `OVSX_PAT` repository secret.

Published package:

- `komagata.tya` version `0.65.1`

Requirements:

- Open VSX namespace: `komagata`
- `OVSX_PAT` or an interactive login

## MELPA

Deferred follow-up: MELPA publication is outside the current syntax coloring
ship scope.

Asset:

- `editors/emacs/tya-mode.el`
- `editors/emacs/melpa-recipe`

Submit a pull request to `melpa/melpa` adding the recipe. MELPA's build should
install `tya-mode.el` and expose `tya-mode` for `.tya` files.

## GitHub Linguist

Deferred follow-up: GitHub Linguist registration is outside the current syntax
coloring ship scope.

Assets:

- `editors/vscode/syntaxes/tya.tmLanguage.json`
- `editors/tree-sitter-tya/`
- `editors/syntax-sample.tya`
- `editors/github-linguist/languages.yml.example`
- https://github.com/komagata/tree-sitter-tya

Submit a pull request to `github-linguist/linguist` adding the language entry
and grammar wiring required by the current Linguist contribution process.

Current blocker: Linguist requires sufficient in-the-wild usage for new
languages. GitHub code search for `extension:tya -is:fork` reported 124 results
on 2026-05-13, below Linguist's threshold for a new language PR. The
Tree-sitter grammar itself is MIT licensed and available as a standalone public
repository.
