# v02 Full-Spec Manifest

`TestSelfhostV02Scripts` is the v02 full-spec proof harness. It runs the
current compiler/language fixture families that are meaningful for the
Tya-written compiler, while leaving non-compiler tooling to the Go reference
implementation until the later Go-removal phase.

## Covered Through v02

- Fixed point: `fixed_point.txtar` proves stage-2 C equals stage-3 C for
  `selfhost/v02/compiler.tya`, and stage1 compiles every `.tya` source in
  `selfhost/v02/`.
- Historical compiler surface: `current_v04.txtar`, `current_v05.txtar`,
  `current_v06.txtar`, `current_v07.txtar`, `abstract_final.txtar`, and
  `override.txtar` run representative class, inheritance, override, and
  primitive runtime behavior through v02-generated C.
- Current front-end/checker/emitter surface: `parser_current_surface.txtar`
  exercises current imports, reserved words, interfaces, class modifiers,
  control syntax, embed syntax, string forms, lambdas, predicate names,
  bitwise parsing, checker accepts/rejects, selected current runtime emission,
  and deterministic unsupported-codegen failures.

## Excluded Until Later Phases

- CLI/tooling families such as `tya doc`, `tya lsp`, `tya lint`, `tya task`,
  `tya new`, package installation, and release packaging remain Go-reference
  tests. They do not prove the self-host compiler's lexer/parser/checker/C
  emitter fixed point.
- Native, network, HTTP server, editor, and platform integration fixtures remain
  Go-reference tests because they depend on environment, package, or runtime
  services outside this v02 compiler proof.
- Full Go-removal and bootstrapped distribution are future work. This manifest
  proves the v02 current-spec self-host gate, not the v1.0 removal of `cmd/tya`
  or `internal/*`.
