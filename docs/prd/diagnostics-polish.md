---
status: approved
goal_ready: true
---

# Feature: Diagnostics Polish

## Goal

Finish the remaining diagnostics polish by making codegen collect every
unsupported AST shape it can find in one pass, while preserving the parser's
existing expression-level recovery behavior for chained and list expressions.

## Context

Tya v0.54 moved Parser, Codegen, and Runner errors into the shared
`diag.Diagnostic` pipeline and added statement-level parser recovery. Tya v0.56
unified the parser, codegen, and runner signatures around
`(value, []diag.Diagnostic, error)` and added expression-list recovery tests
under `tests/testdata/v56_diag/`.

The remaining gap is codegen. `internal/codegen/c.go` still documents codegen as
fail-fast and returns after the first unsupported AST shape. `CodegenError`
already supports multiple diagnostics, so the main work is to collect and return
all recoverable codegen diagnostics deterministically without emitting invalid C.

## Behavior

- Codegen collects every unsupported AST shape that can be identified without
  depending on C output from a previous failed statement.
- Multiple codegen diagnostics are returned as one `*codegen.CodegenError` whose
  `Diags` slice contains every collected diagnostic.
- `EmitC`, `EmitCWithPath`, and `EmitCWithCoverage` keep their public return
  shapes and return the same diagnostics slice that is stored on the
  `CodegenError`.
- CLI diagnostic rendering prints every collected codegen diagnostic through the
  existing shared renderer.
- LSP diagnostics include every collected codegen diagnostic when the LSP path
  reaches codegen diagnostics.
- Diagnostic ordering is stable and source ordered.
- Existing codegen diagnostic codes remain stable:
  - `TYA-E0601` unsupported assignment target
  - `TYA-E0602` unsupported statement or expression
  - `TYA-E0603` unsupported match pattern
  - `TYA-E0604` `try` outside function body
  - `TYA-E0605` non-tuple multi-assignment
  - `TYA-E0606` destructuring target is not an identifier
  - `TYA-E0610` embed file not found
  - `TYA-E0611` embed glob matched no files
- Parser expression-level recovery remains covered for:
  - binary chains such as `a + @ + c`
  - member chains such as `a.@.c`
  - call arguments, array literals, and dictionary literals
- When the feature is implemented and released, `ROADMAP.md` is cleaned up so
  already-shipped Diagnostics polish work is no longer left as open remaining
  work.

## Scope

- `internal/codegen/c.go`
- `internal/codegen/errors.go`
- CLI diagnostic propagation in `cmd/tya/main.go` only if needed
- LSP diagnostic propagation in `internal/lsp/diagnostics.go` only if needed
- focused tests under `tests/testdata/`
- parser tests only for missing binary-chain or member-chain recovery coverage
- `ROADMAP.md`
- `docs/roadmap.html` if roadmap HTML is regenerated during release prep

## Out of Scope

- Adding new diagnostic codes unless an existing unsupported shape has no
  suitable `TYA-E06xx` code.
- Changing the text format of existing diagnostics.
- Recovering from runtime C compiler failures.
- Emitting partial C output after codegen diagnostics have been collected.
- Refactoring codegen architecture beyond the minimum needed to accumulate
  diagnostics.
- Checker multi-error behavior.
- Auditing unrelated roadmap epics outside Diagnostics polish.

## Acceptance Criteria

- A fixture with two independent unsupported codegen shapes reports two
  `TYA-E06xx` diagnostics in one command run.
- A fixture with unsupported codegen shapes before and after an otherwise valid
  statement reports both unsupported shapes and does not stop at the first one.
- `EmitCWithCoverage` returns a nil C source and nil coverage registry when
  codegen diagnostics are present.
- `CodegenError.Error()` remains compatible with existing single-error substring
  assertions.
- `tya check`, `tya build`, and `tya run` render all collected codegen
  diagnostics through the existing human renderer.
- Existing v0.56 parser expression recovery tests still pass.
- New or updated parser coverage proves binary-chain and member-chain recovery
  if that coverage is not already present.
- `ROADMAP.md` no longer lists implemented Diagnostics polish items as open
  work after the implementation/release change lands.
- If release prep rebuilds docs HTML, `docs/roadmap.html` reflects the same
  roadmap cleanup.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run TestV56Script -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Open Questions

None.
