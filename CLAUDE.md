# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

@AGENTS.md

When picking up Tya work after another session (or on a different
machine), also read [`docs/HANDOFF.md`](docs/HANDOFF.md) — it
summarizes the most recent release context, the selfhost/v01
permissive-legacy exemption mechanism, and process notes captured
from past corrections.

## Architecture

Tya is a compile-to-C language. A `.tya` source file flows through:

`internal/lexer/` (uses `internal/token/`) → `internal/parser/` → `internal/ast/` → `internal/checker/` → `internal/codegen/` (C emitter) → `runtime/` (C runtime linked into generated programs) → `internal/runner/` (orchestrates source loading, module resolution, `cc` invocation, and execution).

Two side packages sit alongside the main pipeline: `internal/formatter/` backs `tya fmt`, and `internal/eval/` is a Go-side tree-walking interpreter (used by tests and tooling — it is independent of the C emitter pipeline).

`cmd/tya/` is the user-facing CLI (`run`, `build`, `check`, `fmt`, `emit-c`, `test`, `version`). Module resolution searches the source's directory and `TYA_PATH`; attached standard modules live in `stdlib/`.

The Tya-written self-host compiler at `selfhost/v01/compiler.tya` is a maintained fixed point: it must compile itself to a stable stage-2/stage-3 output. `TestSelfhostV01Scripts` enforces this and must not be skipped or weakened.

`tests/` holds CLI, example, and spec-level integration tests. `docs/archive/pre-v0.1/` is historical reference only — not current authority.

## Common commands

Run the CLI from source:

```sh
go run ./cmd/tya run examples/hello.tya
go run ./cmd/tya build examples/hello.tya -o hello
go run ./cmd/tya check examples/hello.tya
go run ./cmd/tya emit-c examples/hello.tya
go run ./cmd/tya fmt -w path.tya
go run ./cmd/tya test
```

Developer-only flags (not part of the public CLI surface, but useful when hacking on the compiler):

```sh
go run ./cmd/tya --tokens examples/hello.tya
go run ./cmd/tya --check-unused examples/hello.tya
```

Default test suite (includes the self-host fixed-point check):

```sh
go test ./... -count=1
```

Single package / single test:

```sh
go test ./internal/parser -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
```

Focused scripts (use when changes touch examples, argument handling, C emission, imports, runtime execution, or stdlib loading):

```sh
sh scripts/go_emit_examples_check.sh
sh scripts/go_emit_args_check.sh
```

Build the Jekyll doc site:

```sh
bundle exec jekyll build --source docs --destination _site
```

## Versioning convention

Semantic versioning. Spec changes ride minor versions (`v0.3`, `v0.4`); patch releases must not change language or stdlib semantics. Frozen release docs live under `docs/vX.Y.Z/` (e.g. `docs/v0.1.0/`, `docs/v0.2.0/`); the latest editable spec and guide docs live directly under `docs/` (`SPEC.md`, `GUIDE.md`).

`ROADMAP.md` is the single source of truth for planned work. Its format is **Goal Tree**, defined in [`docs/GOAL_TREE.md`](docs/GOAL_TREE.md): a single Goal at the root, with Epic → Milestone → Task underneath, each node carrying a complete/incomplete status. `ROADMAP.md` uses the Markdown rendering (GFM nested task lists). Treat `ROADMAP.md` as a stable remaining-work plan, not a chronological log — follow the Stability Rules in `GOAL_TREE.md` before editing it.
