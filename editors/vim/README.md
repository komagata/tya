# tya — Vim / Neovim syntax files

This directory contains plain Vim runtime files for syntax coloring, filetype
detection, and indentation. They work in Vim 8+ and Neovim.

## Install

Download the editor asset bundle from:

https://github.com/komagata/tya/releases/tag/editors-assets-v0.61.0

Then extract it and copy the Vim runtime directories, or copy them from a
local checkout.

Copy the runtime directories into your Vim config:

```sh
mkdir -p ~/.vim
cp -R editors/vim/ftdetect editors/vim/indent editors/vim/syntax ~/.vim/
```

For Neovim:

```sh
mkdir -p ~/.config/nvim
cp -R editors/vim/ftdetect editors/vim/indent editors/vim/syntax ~/.config/nvim/
```

Open a `.tya` file. Vim should set `filetype=tya`, load
`syntax/tya.vim`, and use the indentation rules from `indent/tya.vim`.
