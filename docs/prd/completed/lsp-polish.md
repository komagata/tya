---
status: completed
goal_ready: false
---

# Feature: LSP Polish

## Goal

Finish the LSP polish items deferred after v0.53 so `tya lsp` feels complete in
daily editor use: safer rename previews, richer semantic tokens, precise range
formatting, common navigation helpers, and publishable VS Code packaging.

## Context

Tya v0.52 introduced `tya lsp` over stdio JSON-RPC. v0.53 expanded it with
cross-file definition, references, rename, range formatting, quick-fix code
actions, semantic tokens, document symbols, workspace symbols, and incremental
document sync.

The remaining v0.53 out-of-scope items are polish features rather than language
semantics. They should be additive LSP methods or editor packaging work, and
must preserve the existing v0.52/v0.53 server behavior for clients that do not
request the new capabilities.

## Behavior

- Publish the VS Code extension from `editors/vscode`:
  - create or document the publisher ID,
  - include an icon and marketplace metadata,
  - generate a signed VSIX release artifact,
  - add release automation that packages the extension from CI.
- Add `textDocument/prepareRename`.
  - Valid rename targets return the range that will be renamed and a placeholder
    matching the current identifier.
  - Invalid rename targets return an LSP error with the existing rename conflict
    diagnostic code family, without mutating workspace state.
  - The validity rules match `textDocument/rename`.
- Add semantic token modifiers.
  - Advertise a stable modifier legend containing at least `readonly`,
    `deprecated`, `definition`, and `defaultLibrary` when supported by local
    analysis.
  - Existing token type numeric order remains stable.
  - Modifiers are encoded through the LSP bitset field in semantic token data.
- Improve `textDocument/rangeFormatting` from top-level-statement widening to
  AST-slice precision.
  - Formatting a selected expression, block, method body, class body, or
    contiguous statement list changes only the minimal syntactic region that can
    be safely re-emitted.
  - If the requested range cannot be mapped to a safe AST slice, fall back to
    the existing v0.53 top-level-statement widening behavior.
  - Parse or formatting failures return no edits, matching current LSP behavior.
- Add inlay hints.
  - Advertise `inlayHintProvider`.
  - Provide parameter-name hints at call sites when the callee signature is known
    and the argument is not already named.
  - Provide inferred binding/class hints only when the checker can resolve the
    value without guesswork.
  - Do not invent hints from dynamic runtime-only information.
- Add call hierarchy.
  - Advertise `callHierarchyProvider`.
  - Support prepare, incoming calls, and outgoing calls for top-level functions,
    class methods, and module functions where static symbol resolution is
    available.
  - Unknown dynamic calls return an empty result rather than an approximate one.
- Add selection range.
  - Advertise `selectionRangeProvider`.
  - Return nested ranges from identifier/expression to statement, block, class
    or module, and full document.
- Add code lens.
  - Advertise `codeLensProvider`.
  - Provide low-noise lenses for runnable test classes or test methods when the
    project shape supports executing them with `tya test`.
  - Do not add decorative or count-only lenses.
- Add folding ranges.
  - Advertise `foldingRangeProvider`.
  - Fold class, module, interface, function/method bodies, block statements,
    multi-line literals, and leading doc-comment blocks.
- Add document links.
  - Advertise `documentLinkProvider`.
  - Link import paths to resolved source files.
  - Link doc-comment Markdown links when they are syntactically valid.
  - Unresolvable imports or malformed links are omitted, not reported as LSP
    errors.
- Keep `positionEncoding: "utf-8"` unless the rest of the LSP server is migrated
  deliberately to another encoding in a separate spec.

## Scope

- LSP protocol wire types in `internal/lsp/protocol*.go`.
- Server capability advertisement and request routing in `internal/lsp/server.go`.
- Feature implementations under `internal/lsp/`.
- Shared symbol, scope, AST range, formatter, checker, and workspace helpers
  needed by the new LSP methods.
- LSP subprocess tests under `tests/`, plus focused unit tests under
  `internal/lsp` where the behavior is easier to pin without JSON-RPC framing.
- VS Code extension metadata, icon, package scripts, and CI release packaging.
- Documentation updates for editor setup and the newly supported LSP features.

## Out of Scope

- Replacing the stdio JSON-RPC server with TCP, WebSocket, or a daemon.
- Implementing a persistent project-wide index beyond the current lazy workspace
  cache unless a specific feature needs a small local cache extension.
- Full dynamic type inference for hints, call hierarchy, or semantic modifiers.
- Non-VS-Code marketplace publication.
- Changing language syntax or checker semantics.
- Reworking existing v0.52/v0.53 LSP features except where needed to share
  range, symbol, or rename validation helpers.

## Acceptance Criteria

- `initialize` advertises every newly implemented provider using LSP-compatible
  capability fields.
- `textDocument/prepareRename` accepts and rejects the same identifiers as
  `textDocument/rename`.
- Semantic tokens keep the existing token type order and add modifier bitsets
  without breaking v0.53 clients.
- Range formatting changes the smallest safe AST slice for covered cases and
  falls back to v0.53 widening when precision is unsafe.
- Inlay hints are deterministic and only appear where the server has static
  evidence.
- Call hierarchy returns correct incoming and outgoing calls for static
  top-level functions, module functions, and class methods.
- Selection ranges nest from the cursor's smallest syntax node up to the full
  document.
- Code lenses are limited to actionable test-running commands.
- Folding ranges cover all documented block-like structures and doc-comment
  blocks.
- Document links resolve imports and valid doc-comment links without surfacing
  extra diagnostics.
- The VS Code extension can be packaged into a signed VSIX from CI and has the
  metadata required for marketplace publication.
- Existing LSP tests from v0.52/v0.53 remain green.
- `go test ./... -count=1` passes, including the self-host fixed-point tests.

## Verification

```sh
go test ./internal/lsp -count=1
go test ./tests -run 'TestLSP|TestLSPV2|Test.*LSP' -count=1
go test ./... -count=1
cd editors/vscode && npm install && npm run compile && npx vsce package
```

## Dependencies

- Builds on the v0.52/v0.53 LSP server.
- Reuses the existing formatter, parser, checker, workspace scan, and symbol
  helpers where possible.
- Some features will be easier to implement after the public self-introspection
  library, but this PRD does not require waiting for that library.

## Open Questions

None.
