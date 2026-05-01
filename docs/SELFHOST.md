# Self-Hosting

Tya's self-hosting compiler is in the prototype stage. The Tya-written
compiler components can tokenize, parse, check, emit C for, compile, and run
the current bootstrap subset of the language through generated stage-4 tools.
They do not yet implement the full language.

## Supported Subset

The current self-hosted lexer supports identifiers, ints, floats, strings with
basic escapes, comments, source line markers, indentation counts, symbols,
`->`, `==`, `!=`, `<=`, and `>=`.

The current self-hosted parser supports a line-oriented node format for simple
assignments, bool assignments, integer addition assignments, comparison
assignments, empty arrays, indexing, `push`, `return`, `if`, `else`, `while`,
`for item in items`, `break`, `continue`, simple function headers, inline
returns, one-, two-, and three-argument calls, and simple print calls.

The current self-hosted checker supports simple undefined-name checks for
assignments, prints, conditions, calls, pushes, returns, indexes, and `for`
collections. It does not yet implement full lexical scope parity with the Go
checker.

The current self-hosted C code generator emits compileable C for the prototype
node format. It supports simple scalar assignments and prints, selected
comparison conditions, basic `if` / `else` / `while` / `for` blocks, `break`,
`continue`, simple array placeholders, and several placeholder paths used to
compile the self-host source files. It still skips real function bodies.

## Bootstrap Checks

Run the current full self-host gate:

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

The current bootstrap pipeline compiles the self-host compiler components with
the Go C emitter, uses those stage-1 binaries to produce stage-2 tools, uses
the generated tools again to produce stage-3 and stage-4 tools, and runs the
stage-4 pipeline across the supported executable examples.

## Completion Criteria

The current self-host gate is achieved when `go test ./... -count=1` and
`sh scripts/selfhost_bootstrap_check.sh` pass from a clean checkout. Full
language parity remains tracked in `ROADMAP.md`.
