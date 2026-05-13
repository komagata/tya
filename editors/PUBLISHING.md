# Publishing Editor Syntax Support

This checklist records the external publishing work for Tya editor syntax
coloring. The repository contains the distributable assets; the external steps
require account credentials or upstream maintainer review.

## VS Code Marketplace

Artifact:

```sh
scripts/verify_editor_assets.sh
```

Manual-install release:

- https://github.com/komagata/tya/releases/tag/editors-vscode-v0.61.0

Publish:

```sh
cd editors/vscode
npx vsce publish
```

Or run the GitHub Actions workflow `Publish VS Code extension` with
`target=marketplace` after configuring the `VSCE_PAT` repository secret.

Requirements:

- Visual Studio Marketplace publisher: `komagata`
- `VSCE_PAT` or an interactive `vsce login komagata`
- Generated package: `editors/vscode/tya-0.61.0.vsix`

## Open VSX

Artifact:

```sh
scripts/verify_editor_assets.sh
```

Publish:

```sh
cd editors/vscode
npx ovsx publish tya-0.61.0.vsix
```

Or run the GitHub Actions workflow `Publish VS Code extension` with
`target=open-vsx` after configuring the `OVSX_PAT` repository secret.

Requirements:

- Open VSX namespace: `komagata`
- `OVSX_PAT` or an interactive login

## MELPA

Asset:

- `editors/emacs/tya-mode.el`
- `editors/emacs/melpa-recipe`

Submit a pull request to `melpa/melpa` adding the recipe. MELPA's build should
install `tya-mode.el` and expose `tya-mode` for `.tya` files.

## GitHub Linguist

Assets:

- `editors/vscode/syntaxes/tya.tmLanguage.json`
- `editors/tree-sitter-tya/`
- `editors/syntax-sample.tya`
- `editors/github-linguist/languages.yml.example`

Submit a pull request to `github-linguist/linguist` adding the language entry
and grammar wiring required by the current Linguist contribution process.

Blocking decision: Linguist only accepts grammars with an allowed license
(`apache-2.0`, `bsd-2-clause`, `bsd-3-clause`, `cc0-1.0`, `isc`, `mit`,
`mpl-2.0`, `ncsa`, `permissive`, `unlicense`, `wtfpl`, or `zlib`). The Tya
repository currently has no project license and the editor grammar is marked
`UNLICENSED`, so the Linguist PR should wait until the grammar license is
explicitly decided.
