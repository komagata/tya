---
status: completed
goal_ready: false
---

# Feature: Selfhost V02 Full Spec Fixed Point

## Goal

Prove the completed `selfhost/v02/` compiler against the full current
black-box specification fixture set and maintain a stable stage-2/stage-3
fixed point for the latest self-hosted compiler.

## Context

This is the final PRD in the v02 current-spec migration sequence. Earlier PRDs
migrate the v02 lexer/parser, checker, and C emitter. This PRD turns those
component migrations into a full proof: v02 should be able to run the same
specification fixture families that define current Go compiler behavior, while
preserving the existing v01 invariant.

This PRD still does not remove the Go implementation. Go remains the reference
implementation until a later, separate v1.0 PRD decides when and how to remove
it and how bootstrap binaries are distributed.

## Behavior

- A v02-specific full-spec test path runs current black-box fixture families
  through `selfhost/v02/compiler.tya` instead of the Go compiler wherever the
  fixture represents language/compiler behavior.
- Stage 1, stage 2, and stage 3 continue to compile `selfhost/v02/compiler.tya`.
- Stage-2 C and stage-3 C for v02 are byte-for-byte stable under the repository
  fixed-point comparison.
- v02 can compile every `.tya` file that is part of `selfhost/v02/`.
- Existing Go compiler black-box tests still pass.
- Existing v01 self-host tests still pass.

## Scope

- `selfhost/v02/compiler.tya`
- `selfhost/v02/ast.tya`
- `tests/testdata/v02_selfhost/`
- new or updated test harness code for running current black-box fixture
  families through v02
- fixture metadata or skip lists only when a fixture is explicitly about Go CLI
  plumbing, release packaging, docs generation, lint-only behavior, native
  environment availability, network behavior, or another surface that is not
  meaningful for the self-host compiler proof
- `ROADMAP.md` wording that distinguishes "v02 current-spec self-host complete"
  from the later "remove Go reference implementation" phase

## Out of Scope

- Removing `cmd/tya` or any `internal/*` Go source.
- Making v02 the default `tya` command.
- Deleting `selfhost/v01/`.
- Final v1.0 bootstrap policy.
- Requiring v02 to implement non-compiler tooling such as `tya doc`, `tya lsp`,
  `tya lint`, `tya task`, package scaffolding, release packaging, or editor
  tooling unless a fixture is needed to prove compiler-language behavior.
- Requiring external network services or platform-specific native libraries for
  the self-host proof.

## Acceptance Criteria

- `TestSelfhostV02Scripts` proves the latest v02 stage-2/stage-3 fixed point.
- The v02 full-spec harness runs the applicable current black-box compiler
  fixture families through v02 and passes.
- Any skipped fixture family has a documented reason tied to Out of Scope, not
  missing compiler support.
- `selfhost/v02/` compiles all of its own `.tya` sources.
- `TestSelfhostV01Scripts` still passes.
- `go test ./... -count=1` passes.
- `ROADMAP.md` no longer implies that current-spec v02 self-hosting is still
  undone, but still keeps Go reference removal as future work.
- No Go implementation files are removed.

## Verification

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestSelfhostV02Scripts -count=1
go test ./... -count=1
```

## Dependencies

- `docs/prd/selfhost-v02-lexer-parser-current-spec.md`
- `docs/prd/selfhost-v02-checker-current-spec.md`
- `docs/prd/selfhost-v02-c-emitter-current-spec.md`

## Open Questions

None.
