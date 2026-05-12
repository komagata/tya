# tya — Neovim setup

Tya's Language Server (`tya lsp`) speaks LSP JSON-RPC 2.0 over
stdio from the same binary as the compiler. Neovim integrates it
via [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig).
Syntax coloring, filetype detection, and indentation are provided by
the shared Vim runtime files in [`../vim/`](../vim).

## Requirements

- Neovim 0.10 or later
- `tya` v0.52 or later on `PATH`
- `nvim-lspconfig`

## Setup

Install the syntax files:

```sh
mkdir -p ~/.config/nvim
cp -R editors/vim/ftdetect editors/vim/indent editors/vim/syntax ~/.config/nvim/
```

Then copy the contents of [`init.lua.example`](./init.lua.example) into
your Neovim config (`~/.config/nvim/lua/tya.lua`, then `require
"tya"` from `init.lua`):

```lua
-- ~/.config/nvim/lua/tya.lua
local lspconfig = require("lspconfig")
local configs = require("lspconfig.configs")

if not configs.tya then
  configs.tya = {
    default_config = {
      cmd = { "tya", "lsp" },
      filetypes = { "tya" },
      root_dir = lspconfig.util.root_pattern("tya.toml", ".git"),
      settings = {},
    },
  }
end

lspconfig.tya.setup({})

vim.filetype.add({ extension = { tya = "tya" } })
```

## Features (v0.53)

- Diagnostics on save / on change
- Syntax coloring and indentation for `.tya` files
- `textDocument/formatting` and `textDocument/rangeFormatting`
- `textDocument/hover` — function signature + leading `#` comment
- `textDocument/definition` (cross-file via `import`)
- `textDocument/references`
- `textDocument/rename`
- `textDocument/codeAction` (TYAL0001 / TYAL0003 quick fixes)
- `textDocument/semanticTokens/full`
- `textDocument/documentSymbol` (outline)
- `workspace/symbol`

## Common keymaps

```lua
local on_attach = function(_, bufnr)
  local opts = { buffer = bufnr }
  vim.keymap.set("n", "gd", vim.lsp.buf.definition, opts)
  vim.keymap.set("n", "gr", vim.lsp.buf.references, opts)
  vim.keymap.set("n", "K",  vim.lsp.buf.hover, opts)
  vim.keymap.set("n", "<leader>rn", vim.lsp.buf.rename, opts)
  vim.keymap.set("n", "<leader>ca", vim.lsp.buf.code_action, opts)
end

lspconfig.tya.setup({ on_attach = on_attach })
```
