# Self-Hosting

Tya's self-hosting compiler is in the prototype stage. The Tya-written
compiler components can tokenize, parse, check, emit C for, compile, and run a
small subset of the language. They do not yet implement the full language.

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

The current stage-1 pipeline compiles the self-host compiler components with
the Go C emitter, then runs those binaries on `examples/hello.tya`. It should
print `Hello, Tya` internally before the wrapper reports success.

## Completion Criteria

Phase 5 is complete only when the Tya-written compiler can compile the existing
executable examples, compile itself through a stage-1 compiler, compile itself
again through a stage-2 compiler, and produce deterministic generated C across
stage 1 and stage 2 from a clean checkout.
