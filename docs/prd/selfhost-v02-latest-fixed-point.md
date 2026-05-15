# Feature: Selfhost V02 Latest Fixed Point

## Goal

Prove the updated `selfhost/v02/` compiler against latest-spec representative
fixtures and maintain the stage-2/stage-3 fixed point after the lexer/parser,
checker, and C emitter follow-up work is complete.

## Context

This is the final PRD in the latest-spec follow-up sequence. Earlier completed
PRDs already proved a v02 current-spec gate; this follow-up exists because the
repository has since added lexical closures, iterable sequence protocols, and
standard protocol interfaces. This spec turns the component work into a
reviewable proof without removing the Go reference implementation.

## Behavior

- `TestSelfhostV02Scripts` includes latest-spec fixtures for lexical closures,
  iterable protocols, and standard protocol interfaces where they are
  meaningful for the self-host compiler proof.
- Stage 1, stage 2, and stage 3 continue to compile
  `selfhost/v02/compiler.tya`.
- Stage-2 C and stage-3 C remain byte-for-byte stable under the repository
  fixed-point comparison.
- v02 can compile every `.tya` source that is part of `selfhost/v02/`.
- Existing Go compiler black-box tests and v01 self-host tests still pass.
- The v02 full-spec manifest documents covered latest-spec fixtures and any
  excluded fixture families.

## Scope

- `selfhost/v02/compiler.tya`
- `selfhost/v02/ast.tya`
- `tests/testdata/v02_selfhost/`
- `tests/testdata/v02_selfhost/full_spec_manifest.md`
- v02 test harness code for latest-spec proof coverage
- `ROADMAP.md` only if its self-host wording needs clarification after this
  latest-spec follow-up

## Out of Scope

- Removing `cmd/tya` or any `internal/*` Go source.
- Making v02 the default `tya` command.
- Deleting `selfhost/v01/`.
- Final v1.0 bootstrap policy.
- Requiring v02 to implement non-compiler tooling such as `tya doc`, `tya lsp`,
  `tya lint`, `tya task`, release packaging, or editor tooling.
- Requiring external network services or platform-specific native libraries for
  the self-host proof.

## Acceptance Criteria

- `TestSelfhostV02Scripts` proves the latest v02 stage-2/stage-3 fixed point.
- The v02 proof covers latest lexical closure, iterable, and protocol-interface
  fixture families.
- Any skipped fixture family has a documented Out-of-Scope reason, not missing
  compiler support for the covered language surface.
- `selfhost/v02/` compiles all of its own `.tya` sources.
- `TestSelfhostV01Scripts` still passes.
- `go test ./... -count=1 -timeout=20m` passes.
- No Go implementation files are removed.

## Verification

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestSelfhostV02Scripts -count=1
go test ./... -count=1 -timeout=20m
```

## Dependencies

- `docs/prd/selfhost-v02-latest-lexer-parser.md`
- `docs/prd/selfhost-v02-latest-checker.md`
- `docs/prd/selfhost-v02-latest-c-emitter.md`
