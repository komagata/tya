---
layout: doc
title: Spec
permalink: /v0.53/spec/
---

# Tya v0.53 Specification

> **Status:** shipped. The `tya version` constant is `0.53.0`.
> v0.53 promotes the `tya lsp` MVP to a full LSP server. The
> language surface is unchanged from v0.49 — v0.52.

## Theme

v0.52 cut the LSP MVP. v0.53 unlocks the remaining IDE features
that turn `tya lsp` into a viable daily-driver server: cross-file
definition, references, rename, range formatting, code actions,
semantic tokens, document and workspace symbols, and incremental
document sync.

Setup recipes for Neovim, Zed, and Emacs ship under
[`editors/`](../editors). Marketplace publication of the VS Code
extension is queued for v0.54+.

## `tya lsp` v2 — added methods

| Method | Direction | Behaviour |
|--------|-----------|-----------|
| `textDocument/references` | request | Scope-aware: top-level → workspace scan; local / param → enclosing FuncLit |
| `textDocument/rename` | request | Returns `WorkspaceEdit`. Conflicts produce `TYA-E0933` |
| `textDocument/rangeFormatting` | request | Heuristic A: widen to smallest enclosing top-level Stmt(s), unparse, replace |
| `textDocument/codeAction` | request | Quick fixes for `TYAL0001` (line-delete) and `TYAL0003` (if-unwrap) |
| `textDocument/semanticTokens/full` | request | 9-type legend (keyword, variable, string, number, comment, operator, function, class, namespace) |
| `textDocument/documentSymbol` | request | Hierarchical (class → methods, module → members) |
| `workspace/symbol` | request | Case-insensitive substring match across `*.tya` under the workspace root |

`textDocument/definition` gains cross-file resolution when the
identifier is an `ImportStmt` binding or a `mod.foo` member.

## Capabilities (v0.53)

```json
{
  "textDocumentSync": { "openClose": true, "change": 2, "save": { "includeText": false } },
  "hoverProvider": true,
  "definitionProvider": true,
  "referencesProvider": true,
  "renameProvider": true,
  "completionProvider": { "resolveProvider": false },
  "documentFormattingProvider": true,
  "documentRangeFormattingProvider": true,
  "codeActionProvider": { "codeActionKinds": ["quickfix"] },
  "semanticTokensProvider": {
    "legend": { "tokenTypes": [...], "tokenModifiers": [] },
    "range": false,
    "full": true
  },
  "documentSymbolProvider": true,
  "workspaceSymbolProvider": true,
  "positionEncoding": "utf-8"
}
```

`change: 2` is `TextDocumentSyncKind.Incremental`. The server
advertises `positionEncoding: "utf-8"` so byte offsets match LSP
position offsets directly.

## Scope rules

### Rename / references

The identifier under the cursor is classified by `ScopeKindAt`:

- **top-level** — the binding appears in the file's top-level
  symbol index. Rename / references span every `*.tya` file in the
  workspace. Conflicts (e.g. the new name is already top-level in
  another file) produce `TYA-E0933`.
- **param** — the identifier matches one of the enclosing
  `FuncLit.Params`. Rename / references stay inside the function
  body and its parameter list.
- **local** — the identifier appears as a target of an
  `AssignStmt` inside the enclosing `FuncLit` body. Same
  containment rule as **param**.
- **unknown** — no match. Rename / references return no edits /
  no locations.

### Cross-file definition

`textDocument/definition` resolves cross-file in exactly two cases:

1. The cursor is on an `ImportStmt` binding — the definition is
   the imported module's source file (the first matching path
   among `src/<name>.tya`, `<name>.tya`, `<name>/<name>.tya`,
   `src/<name>/<name>.tya`).
2. The cursor is on the member of a `mod.foo` expression where
   `mod` is itself an `ImportStmt` binding — the definition is
   the same-name top-level binding inside the imported file.

All other identifiers resolve same-file only.

### Range formatting

`textDocument/rangeFormatting` widens the requested range to the
smallest set of contiguous top-level statements that fully overlap
it, runs `formatter.Unparse` on the entire program, and replaces
just the corresponding lines in the buffer. Ranges with no
top-level statement match are no-ops.

## Code actions

- `TYAL0001 unused local` → kind: `quickfix`, edit: delete the
  binding's line.
- `TYAL0003 redundant if true/false` → kind: `quickfix`, edit:
  drop the `if` header line and de-indent the chosen body by the
  header's indent (two spaces in canonical syntax).

The server only emits actions whose `code` is present in
`CodeActionContext.Diagnostics` (when the client supplies one).

## Semantic tokens

Legend (`tokenTypes`):

```
0: keyword       1: variable      2: string
3: number        4: comment       5: operator
6: function      7: class         8: namespace
```

`tokenModifiers` is the empty list. Identifiers in call position
are typed as `function`; identifiers that resolve to a top-level
class are typed as `class`, module / interface as `namespace`.
Unrecognised tokens are dropped.

## Position encoding

`positionEncoding` is advertised as `"utf-8"`. tya identifiers are
ASCII and source files are stored as UTF-8 bytes, so LSP
`Position.Character` is interpreted as a byte column inside the
line. Clients that force UTF-16 see correct positions for ASCII
code; non-ASCII characters in string literals may still skew by
LSP fractions of a code unit, which is acceptable for v0.53 (no
identifier resolution depends on string contents).

## Incremental document sync

`TextDocumentSyncKind.Incremental = 2` is advertised. The server
applies `TextDocumentContentChangeEvent.Range`-bounded edits
in-order using a byte-offset position translator. Each change
re-publishes diagnostics for that URI.

## New error codes

| Code | Subcommand | Meaning |
|------|------------|---------|
| `TYA-E0932` | `tya lsp` | Workspace scan failure (recoverable; logged) |
| `TYA-E0933` | `tya lsp` | Rename conflict (shadow, duplicate, or invalid identifier) |

## Out of scope (v0.54+)

- `prepareRename` (rename preview)
- Range formatting at AST slice precision (currently Heuristic A)
- Semantic token modifiers (`readonly`, `deprecated`, …)
- Inlay hints, call hierarchy, selection range, code lens,
  folding range, document link
- VS Code Marketplace publication (publisher registration,
  signed VSIX, icon, GH Actions release pipeline)

## Compatibility

- Language surface: unchanged from v0.49 — v0.52.
- `tya.toml` schema: unchanged.
- CLI: every existing subcommand keeps its v0.52 behaviour. The
  new LSP methods are additive.

## Tests

`tests/lsp_v2_test.go` adds 11 cases (in addition to the 7 v0.52
tests in `tests/lsp_test.go`), covering cross-file definition,
rename (top-level, local, conflict), references, range formatting,
quick fixes, semantic tokens, incremental sync, document outline,
and workspace symbol filtering.
