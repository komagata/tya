# Tya Development Guidelines

Tya is a language without hesitation — an indentation-based, dynamically typed, compile-to-C language. The reference implementation is hand-written in Go. Tya commits to a Canonical Syntax (every program has exactly one source representation) and an all-in-one toolchain. Before editing, inspect the existing lexer, parser, AST, checker, C emitter, runner, tests, and examples that are closest to the requested behavior.

## Project Rules

- Prefer the current hand-written compiler style over adding parser generators or large dependencies.
- Keep language syntax and naming consistent with `docs/SPEC.md`.
- Keep stdlib behavior consistent with `docs/SPEC.md`.
- Treat `docs/archive/pre-v0.1/` as historical reference only. Archived self-host, class/module, and stdlib notes are not current v0.1 authority.
- Preserve the existing Tya-written self-host compiler fixed point. Once self-host works, do not regress it while implementing later versions. Changes to lexer, parser, AST, checker, C emitter, runtime, CLI, examples, stdlib, or tests must keep `selfhost/v01/compiler.tya` compiling itself to a stable stage-2/stage-3 fixed point.
- Use `ROADMAP.md` as the single source of truth for TODO, TASK, and roadmap planning. Express roadmap work as Markdown nested task lists: top-level items are Epics, second-level items are Milestones, and third-level items are Tasks. A Milestone is complete when every Task below it is complete, and an Epic is complete when every Milestone below it is complete.
- Do not rewrite unrelated user changes.

## Verification

Run `gofmt -w` on changed Go files.

Default verification:

```sh
go test ./... -count=1
```

The self-host invariant is part of default verification. Do not skip or weaken
`TestSelfhostV01Scripts`; it proves the Tya-written compiler can compile itself
to a stable fixed point.

Use focused scripts under `scripts/` when changes affect examples, C emission, imports, runtime execution, or stdlib loading.
