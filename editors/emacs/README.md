# tya — Emacs setup

[`tya-mode.el`](./tya-mode.el) provides syntax coloring and `.tya`
file association. Tya's Language Server (`tya lsp`) speaks LSP
JSON-RPC 2.0 over stdio. Emacs integrates it via either
[`lsp-mode`](https://emacs-lsp.github.io/lsp-mode/) or
[`eglot`](https://github.com/joaotavora/eglot) (built into Emacs 29+).

## Requirements

- Emacs 28 or later (eglot bundled since 29)
- `tya` v0.52 or later on `PATH`

## Setup

Download the editor asset bundle from:

https://github.com/komagata/tya/releases/tag/editors-assets-v0.61.0

Then extract it and load `editors/emacs/tya-mode.el`.

Load [`tya-mode.el`](./tya-mode.el), then copy
[`setup.el.example`](./setup.el.example) wholesale or pick the snippet
for your LSP client.

```elisp
(add-to-list 'load-path "/path/to/tya/editors/emacs")
(require 'tya-mode)
```

### eglot

```elisp
(define-derived-mode tya-mode prog-mode "tya")
(add-to-list 'auto-mode-alist '("\\.tya\\'" . tya-mode))
(with-eval-after-load 'eglot
  (add-to-list 'eglot-server-programs '(tya-mode . ("tya" "lsp"))))
(add-hook 'tya-mode-hook #'eglot-ensure)
```

### lsp-mode

```elisp
(define-derived-mode tya-mode prog-mode "tya")
(add-to-list 'auto-mode-alist '("\\.tya\\'" . tya-mode))
(with-eval-after-load 'lsp-mode
  (lsp-register-client
    (make-lsp-client :new-connection (lsp-stdio-connection '("tya" "lsp"))
                     :major-modes '(tya-mode)
                     :server-id 'tya)))
(add-hook 'tya-mode-hook #'lsp)
```

## Features (v0.61)

- Diagnostics on save / on change
- Syntax coloring for Tya keywords, declarations, literals, numbers, comments,
  strings, and function declarations
- Formatting (full + range)
- Hover, goto-definition (cross-file), references, rename
- Code actions (TYAL0001 / TYAL0003 quick fixes)
- Document outline (`imenu`), workspace symbols
- Semantic tokens (lsp-mode 9+ / eglot 1.16+)
