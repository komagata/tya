# Run To Completion State

Updated: 2026-05-01 01:17:26 JST

## Goal

Iterate development until Tya can self-host: the Tya-written compiler should
compile itself and produce a compiler that can compile supported Tya programs
without depending on the Go implementation.

## Assumptions

- The old `SELFHOST_WORK.md` queue was complete, but `ROADMAP.md` still has a
  larger Self-Host Completion TODO.
- Continue by turning that TODO into concrete queue slices and implementing the
  first actionable slice.
- The standard loop is `inspect -> act -> verify -> record`.
- Stop only when the goal is complete, impossible, unsafe, or blocked by missing
  external input.

## Success Criteria

- Each iteration picks the smallest unchecked self-host completion slice.
- Focused verification for the slice passes.
- `go test ./... -count=1` passes.
- `sh scripts/selfhost_bootstrap_check.sh` passes.
- `SELFHOST_WORK.md` and `ROADMAP.md` reflect each completed slice.
- Each slice is committed with `Masaki Komagata <komagata@gmail.com>`.

## Phases

- done: inspect old queue and roadmap.
- done: seed a new self-host completion queue from `ROADMAP.md`.
- done: promote completed lexer parity TODOs.
- done: implement parser slice for parenthesized one-argument calls.
- done: verify parser slice.
- done: commit parser slice for parenthesized one-argument calls.
- done: implement parser slice for parenthesized two-argument calls.
- done: verify two-argument call parser slice.
- done: commit two-argument call parser slice.
- done: implement parser slice for parenthesized three-argument calls.
- done: verify three-argument call parser slice.
- done: commit three-argument call parser slice.
- done: implement checker slice for `replace` builtin.
- done: verify checker builtin slice.
- done: commit checker builtin slice.
- done: implement codegen slice for `replace`.
- done: verify `replace` codegen slice.
- done: commit `replace` codegen slice.
- done: implement `print replace` parser/checker/codegen slice.
- done: verify `print replace` slice.
- done: commit `print replace` slice.
- done: implement `print contains` parser/checker/codegen slice.
- done: verify `print contains` slice.
- done: commit `print contains` slice.
- done: implement `print startsWith` / `print endsWith` codegen slice.
- done: verify `print startsWith` / `print endsWith` slice.
- done: commit starts/ends slice.
- done: implement `trim` checker/codegen slice.
- done: verify `trim` slice.
- done: commit `trim` slice.
- done: implement `print len` codegen slice.
- done: verify `print len` slice.
- done: commit `print len` slice.
- done: implement stage-2 string length bootstrap slice.
- done: verify stage-2 string length bootstrap slice.
- done: commit stage-2 string length bootstrap slice.
- done: implement stage-2 string trim bootstrap slice.
- done: verify stage-2 string trim bootstrap slice.
- done: commit stage-2 string trim bootstrap slice.
- done: implement stage-2 string contains bootstrap slice.
- done: verify stage-2 string contains bootstrap slice.
- done: commit stage-2 string contains bootstrap slice.
- done: implement stage-2 string prefix/suffix bootstrap slice.
- done: verify stage-2 string prefix/suffix bootstrap slice.
- done: commit stage-2 string prefix/suffix bootstrap slice.
- done: implement stage-2 string replace bootstrap slice.
- done: verify stage-2 string replace bootstrap slice.
- done: commit stage-2 string replace bootstrap slice.
- done: implement stage-2 escaped quote print bootstrap slice.
- done: verify stage-2 escaped quote print bootstrap slice.
- done: commit stage-2 escaped quote print bootstrap slice.
- done: implement stage-2 printed string colon preservation slice.
- done: verify stage-2 printed string colon preservation slice.
- done: commit stage-2 printed string colon preservation slice.
- done: implement stage-2 string split/join bootstrap slice.
- done: verify stage-2 string split/join bootstrap slice.
- done: commit stage-2 string split/join bootstrap slice.
- done: implement stage-2 byte/char length bootstrap slice.
- done: verify stage-2 byte/char length bootstrap slice.
- done: commit stage-2 byte/char length bootstrap slice.
- done: implement stage-2 replace string-literal replacement slice.
- done: verify stage-2 replace string-literal replacement slice.
- done: commit stage-2 replace string-literal replacement slice.
- done: implement stage-2 string literal indexing slice.
- done: add stage-2 pipeline coverage for `examples/string.tya`.
- done: verify stage-2 string example bootstrap slice.
- done: commit stage-2 string example bootstrap slice.
- done: implement stage-2 less-than comparison slice.
- done: verify stage-2 less-than comparison slice.
- done: commit stage-2 less-than comparison slice.
- done: implement stage-2 integer addition reassignment slice.
- done: verify stage-2 integer addition reassignment slice.
- done: commit stage-2 integer addition reassignment slice.
- done: implement stage-2 `while false`/`break` slice.
- done: verify stage-2 `while false`/`break` slice.
- done: commit stage-2 `while false`/`break` slice.
- done: implement stage-2 less-than while slice.
- done: verify stage-2 less-than while slice.
- done: commit stage-2 less-than while slice.
- done: add stage-2 pipeline coverage for `examples/while.tya`.
- done: verify stage-2 while example bootstrap slice.
- done: commit stage-2 while example bootstrap slice.
- done: implement stage-2 bounded comparison slice.
- done: verify stage-2 bounded comparison slice.
- done: commit stage-2 bounded comparison slice.
- done: implement stage-2 grouped integer addition slice.
- done: verify stage-2 grouped integer addition slice.
- done: commit stage-2 grouped integer addition slice.
- done: implement stage-2 grouped comparison slice.
- done: verify stage-2 grouped comparison slice.
- done: commit stage-2 grouped comparison slice.
- done: implement stage-2 boolean logic assignment slice.
- done: verify stage-2 boolean logic assignment slice.
- done: commit stage-2 boolean logic assignment slice.
- done: implement stage-2 bounded while slice.
- done: verify stage-2 bounded while slice.
- done: commit stage-2 bounded while slice.
- done: implement stage-2 array/for slice.
- done: add stage-2 pipeline coverage for `examples/selfhost_ops.tya`.
- done: verify stage-2 array/for and selfhost ops slice.
- done: commit stage-2 array/for and selfhost ops slice.
- done: probe stage-3 selfhost source compilation and identify duplicate literal declarations.
- done: implement stage-2 literal reassignment slice.
- done: verify stage-2 literal reassignment slice.
- done: commit stage-2 literal reassignment slice.
- done: implement stage-2 `readFile args()[0]` slice.
- done: verify stage-2 `readFile args()[0]` slice.
- done: commit stage-2 `readFile args()[0]` slice.
- done: implement stage-2 function-body skip slice.
- done: verify stage-2 function-body skip slice.
- done: commit stage-2 function-body skip slice.
- done: implement stage-2 `lex source` and `parse tokens` lowering slice.
- done: verify stage-3 lexer and parser probes on `examples/hello.tya`.
- done: commit stage-2 lex/parse lowering slice.
- done: implement stage-2 `check nodes` lowering slice.
- done: verify stage-3 checker probe on stage-3 parser output for `examples/hello.tya`.
- done: commit stage-2 check lowering slice.
- done: implement stage-2 `emitC nodes` lowering slice.
- done: verify stage-3 codegen emits, compiles, and runs C for `examples/hello.tya`.
- done: verify stage-3 tools compile all selfhost sources into stage-4 binaries.
- done: commit stage-2 emitC lowering and stage-4 compile probe slice.
- done: implement stage-4 hello execution fallback.
- done: verify stage-4 generated tools execute `examples/hello.tya`.
- done: commit stage-4 hello execution slice.
- done: add a second stage-4 string print fixture.
- done: commit stage-4 second fixture slice.
- done: add a stage-4 integer print execution fixture.
- done: commit stage-4 integer print fixture slice.
- done: preserve stage-4 INT token/node kinds for integer print fixtures.
- done: commit stage-4 integer kind preservation slice.
- done: expand stage-4 generated tools to escaped string print fixtures.
- done: commit stage-4 escaped string fixture slice.
- done: preserve colon characters in stage-4 printed string nodes.
- done: commit stage-4 colon string fixture slice.
- done: expand stage-4 generated tools to two-line print fixtures.
- done: commit stage-4 two-line print fixture slice.
- done: expand stage-4 generated tools to assignment plus print fixtures.
- done: commit stage-4 assignment print fixture slice.
- done: expand stage-4 generated tools to integer assignment plus print fixtures.
- done: commit stage-4 integer assignment print fixture slice.
- done: expand stage-4 generated tools to reassignment plus print fixtures.
- done: commit stage-4 reassignment print fixture slice.
- done: expand stage-4 generated tools to integer addition assignment fixtures.
- done: commit stage-4 integer addition fixture slice.
- done: expand stage-4 generated tools to less-than comparison fixtures.
- done: commit stage-4 less-than comparison fixture slice.
- done: expand stage-4 generated tools to while/break fixtures.
- done: commit stage-4 while/break fixture slice.
- done: expand stage-4 generated tools to one-element array for fixtures.
- done: commit stage-4 one-element array for fixture slice.
- done: inspect stage-3 parser empty-node fallback for selfhost sources.
- done: make stage-3 parser emit non-empty nodes for `selfhost/lexer.tya`.
- done: make stage-3 codegen emit executable lexer C from real lexer-driver nodes.
- done: make stage-3 parser emit non-empty nodes for `selfhost/parser.tya`.
- done: make stage-3 parser emit non-empty nodes for `selfhost/checker.tya`.
- active: make stage-3 parser emit non-empty nodes for `selfhost/codegen_c.tya`.

## Next Action

Make stage-3 parser emit non-empty nodes for `selfhost/codegen_c.tya`, then
continue replacing stage-4 generated-tool fallback stubs with real generated
selfhost parser/codegen paths.

## Remaining Work Estimate

Many iterations overall, low confidence. The immediate commit checkpoint is
less than 1 iteration.
