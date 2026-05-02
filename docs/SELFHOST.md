# Self-Hosting

Tya's self-hosting compiler has a complete automated bootstrap gate for the
current supported subset. The Tya-written compiler components can tokenize,
parse, check, emit C for, compile, and run the supported bootstrap subset of
the language through generated stage-4 and stage-5 tools. They also reach
stable stage-7 generated C for the self-host compiler sources. They do not yet
implement the full language.

## Supported Subset

The current self-hosted lexer supports identifiers, ints, floats, strings with
basic escapes, comments, source line markers, indentation counts, symbols,
`->`, `==`, `!=`, `<=`, and `>=`.

The current self-hosted parser supports a line-oriented node format for simple
assignments, bool assignments, simple arithmetic assignments, comparison
assignments, one- and two-element arrays, one-property inline objects,
indexing, `push`, `return`, `if`, `else`, `while`, array/object `for` subset
forms, `break`, `continue`, simple function headers, inline returns,
selected one-, two-, three-, and four-argument signatures, selected one-, two-,
and three-argument calls, two-target assignment, simple `try` calls, member
prints, and simple print calls.

The current self-hosted checker supports simple undefined-name checks for
assignments, prints, conditions, calls, pushes, returns, indexes, and `for`
collections. It also rejects invalid assignment binding names and reassignment
of all-caps constants in the supported node subset. It does not yet implement
full lexical scope parity with the Go checker.

The current self-hosted C code generator emits compileable C for the prototype
node format. It supports simple scalar assignments and prints, selected
comparison conditions, basic `if` / `else` / `while` / `for` blocks, `break`,
`continue`, simple arrays, string array index assignments, object placeholders,
selected string builtins, simple return-function bodies, and several source-specific paths used to
compile the self-host source files. It still does not provide general codegen
for the full language.

## Complete Bootstrap Gate

Run the complete self-host bootstrap gate from the repository root:

```sh
sh scripts/selfhost_bootstrap_check.sh
```

The script runs:

- `scripts/selfhost_check.sh`
- `scripts/selfhost_compile_check.sh`
- `scripts/go_emit_selfhost_compile_check.sh`
- `scripts/go_emit_selfhost_run_check.sh`

Expected output:

```text
selfhost bootstrap: ok
```

This command is the single documented verification command for self-hosting. It
runs the Tya source checks, self-host generated-C compile checks, Go-emitted
self-host compile/run checks, the stage-generated supported-example parity
gate, repeated bootstrap stages, and deterministic fixed-point checks.

The bootstrap pipeline compiles the self-host compiler components with the Go C
emitter, uses those stage-1 binaries to produce stage-2 tools, uses the
generated tools again to produce later stages, runs the stage-4 pipeline across
every example marked supported in the parity manifest, compares each generated
binary's output with the Go interpreter, uses the stage-4 tools to compile all
four self-host compiler sources into stage-5 C binaries, uses stage-5 tools to
compile stage-6 binaries, and verifies stable stage-7 generated C for the
self-host compiler sources.

The explicit fixed-point gate is:

```sh
sh scripts/selfhost_fixed_point_check.sh
```

That gate rebuilds the generated toolchain through stage 4, emits C twice for
`selfhost/lexer.tya`, `selfhost/parser.tya`, `selfhost/checker.tya`, and
`selfhost/codegen_c.tya` with the stage-4 generated code generator, and fails
if any byte-for-byte diff appears between the repeated generated C files. It
is also covered by `sh scripts/selfhost_bootstrap_check.sh`. It does not claim
full language parity; it proves deterministic regeneration for the complete
self-host compiler source set supported by the current generated toolchain.

Example parity status is tracked in
`scripts/selfhost_examples_manifest.txt`. Each example is classified as
supported, expected-failing with its next missing feature, or out-of-scope as a
fixture/module support file. `go test ./tests -run Selfhost -count=1` fails if
a new example lacks a self-host parity classification.

## Remaining Full-Parity Gaps

The complete self-host goal is broader than the current bootstrap gate:

- Replace the line-oriented parser/node-string subset with structured parsing
  for the full Go parser grammar.
- Bring checker scope, naming, constants, imports/modules, object/member,
  builtin, and diagnostic behavior to Go checker parity.
- Emit general C for functions, methods, objects, arrays, imports, errors,
  `try`, multi-return values, interpolation, unary operations, and the full
  documented standard library without source-specific fallback paths.
- Promote the expected-failing examples in
  `scripts/selfhost_examples_manifest.txt` into generated-tool parity targets.
- Keep deterministic generated-C comparisons through the final self-host fixed
  point.

## Completion Criteria

The current self-host gate is achieved when `go test ./... -count=1` and
`sh scripts/selfhost_bootstrap_check.sh` pass from a clean checkout. Full
language parity remains tracked in `SELFHOST_WORK.md` and `ROADMAP.md`.
