# Tya Class And Module Implementation Summary

Tya is a small indentation-based dynamic language implemented in Go. The next implementation track is the language design in `docs/CLASS_MODULE_DESIGN.md`.

The goal is to distinguish dictionaries, sets, class instances, classes, and modules; add class and module declarations; enforce one-file-one-public-definition import rules; add import aliases and entry-file semantics; then extend the design with single inheritance, `super`, interfaces, and complete docs/tests/examples.

Primary verification:

- `go test ./... -count=1`

Use focused package tests while implementing:

- `go test ./internal/lexer ./internal/parser ./internal/checker -count=1`
- `go test ./internal/eval ./internal/runner -count=1`
- `go test ./internal/codegen ./tests -count=1`

Run self-host/bootstrap checks only when a change affects self-host manifests, scripts, generated-C gates, or self-host source expectations.

Important files:

- `docs/CLASS_MODULE_DESIGN.md`: source design
- `internal/lexer/`: tokens and keywords
- `internal/parser/`: syntax parsing
- `internal/ast/`: AST nodes
- `internal/checker/`: semantic checks
- `internal/eval/`: interpreter
- `internal/runner/`: file/import execution
- `internal/codegen/`: C emitter
- `runtime/`: generated-C runtime support
- `examples/` and `tests/`: regression coverage
