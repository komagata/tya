# Tya Development Guidelines

Tya is a small indentation-based dynamic language implemented in Go. Before editing, inspect the existing lexer, parser, AST, checker, interpreter, C emitter, runner, tests, and examples that are closest to the requested behavior.

## Project Rules

- Prefer the current hand-written compiler/interpreter style over adding parser generators or large dependencies.
- Keep language syntax and naming consistent with `docs/NAMING.md`.
- Keep stdlib behavior consistent with `docs/STDLIB.md`.
- Treat self-hosting changes as cross-cutting; check `SELFHOST_WORK.md`, `ROADMAP.md`, `docs/SELFHOST.md`, and `selfhost/README.md`.
- Do not rewrite unrelated user changes.

## Verification

Run `gofmt -w` on changed Go files.

Default verification:

```sh
go test ./... -count=1
```

For self-hosting work, also run:

```sh
sh scripts/selfhost_bootstrap_check.sh
```

Use focused scripts under `scripts/` when changes affect examples, C emission, imports, runtime execution, stdlib loading, or self-hosting.

## Ralph Tasks

When working from `.agent/tasks.json`, complete exactly one task, update task status and `.agent/logs/LOG.md`, commit, and stop. Do not push from a Ralph iteration.
