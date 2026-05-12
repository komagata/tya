# tya — Emacs setup

Tya's Language Server (`tya lsp`) speaks LSP JSON-RPC 2.0 over
stdio. Emacs integrates it via either
[`lsp-mode`](https://emacs-lsp.github.io/lsp-mode/) or
[`eglot`](https://github.com/joaotavora/eglot) (built into Emacs 29+).

## Requirements

- Emacs 28 or later (eglot bundled since 29)
- `tya` v0.52 or later on `PATH`

## Setup

Either copy [`setup.el.example`](./setup.el.example) wholesale or
pick the snippet for your LSP client.

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

## Features (v0.53)

- Diagnostics on save / on change
- Formatting (full + range)
- Hover, goto-definition (cross-file), references, rename
- Code actions (TYAL0001 / TYAL0003 quick fixes)
- Document outline (`imenu`), workspace symbols
- Semantic tokens (lsp-mode 9+ / eglot 1.16+)
