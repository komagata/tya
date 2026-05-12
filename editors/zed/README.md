# tya — Zed setup

Tya's Language Server (`tya lsp`) speaks LSP JSON-RPC 2.0 over
stdio. [Zed](https://zed.dev/) wires it up through its `languages`
configuration.

## Requirements

- Zed 0.140 or later
- `tya` v0.52 or later on `PATH`

## Setup

Add the following block to `~/.config/zed/settings.json` (or paste
[`extension.json.example`](./extension.json.example)) under the
top-level object:

```json
{
  "languages": {
    "tya": {
      "language_server": {
        "command": "tya",
        "args": ["lsp"]
      },
      "file_types": ["tya"]
    }
  }
}
```

Restart Zed. Open any `.tya` file under a directory containing a
`tya.toml` — diagnostics, formatting, hover, goto-definition,
references, rename, code actions, and semantic tokens will all
work via the same binary.

A dedicated Zed extension (Marketplace publication) is queued for
v0.54+.

## Features (v0.53)

- Diagnostics on save / on change
- Formatting (full + range)
- Hover, goto-definition (cross-file), references, rename
- Code actions (TYAL0001 / TYAL0003 quick fixes)
- Document outline, workspace symbols
- Semantic tokens
