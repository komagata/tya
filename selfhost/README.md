# Self-Hosting

This directory contains the Tya-written compiler pieces used by the automated
self-host bootstrap gate.

Run the complete self-host bootstrap gate from the repository root:

```sh
sh scripts/selfhost_bootstrap_check.sh
```

This single command runs the source checks, generated-C compile checks, the
stage-generated supported-example parity gate, repeated bootstrap stages, and
fixed-point checks for deterministic generated C.

Development pipeline:

```sh
sh scripts/selfhost.sh
```

Self-host source checks:

```sh
sh scripts/selfhost_check.sh
```

Self-host generated-C compile checks:

```sh
sh scripts/selfhost_compile_check.sh
```

Go-emitted self-host C compile checks:

```sh
sh scripts/go_emit_selfhost_compile_check.sh
```

Go-emitted self-host pipeline run check:

```sh
sh scripts/go_emit_selfhost_run_check.sh
```

The current implementation proves that Tya can run Tya-written compiler
components through repeated generated stages for the supported subset. The
compiler components still do not understand the full language.

Current supported subset:

- Lexer: identifiers, ints, strings, comments, symbols, common two-character
  operators, source lines, and indentation counts
- Parser: line-oriented nodes for simple assignments, simple arithmetic assignments, simple function headers
  and inline returns, `push`, `return`, `if`, `else`, one-, two-, and
  three-argument function calls, selected four-argument function signatures,
  `while`, `for`, `break`, `continue`, direct comparison, `!=`, `>=`, `<=`,
  and `or` conditions, one-argument call conditions, call comparison
  conditions, negated call conditions, comparison and call-based `while`
  conditions, call-with-call-index arguments, call indexing, return calls,
  indexing, indentation, member print nodes, multiple-return subset nodes, and
  one-argument print calls
- Checker: simple undefined assignment / print / condition names, invalid
  assignment binding names, and constant reassignment
- C codegen: string/int assignments, simple integer addition assignments,
  simple comparison / `!=` / `>=` / `<=` assignments, variable-copy assignments, bool assignments,
  unary `not`, one-element array paths for simple `push` / `for`, reassignment,
  carried function headers and inline returns, one-, two-, and three-argument calls, `push` and `return`
  commands, string/int/bool print nodes, carried indexing, call indexing,
  call-with-call-index arguments, one-argument call conditions,
  call comparison conditions, direct comparison / `!=` / `>=` / `<=` conditions, prototype `hasT`
  predicate conditions, prototype `len(parts) < 3`, simple `or`
  conditions, negated call conditions, comparison and call-based
  `while` conditions, return calls, simple value-copy calls, one-argument print calls, placeholder
  call/index assignment declarations, self-host source compile smoke checks,
  and simple literal or variable `if` / `else` / `while` / `for` blocks with `break` /
  `continue`

The bootstrap scripts now reach stable stage-7 generated C for the self-host
compiler sources. `scripts/selfhost_fixed_point_check.sh` also verifies that
the stage-4 generated toolchain emits byte-stable C for the lexer, parser,
checker, and C code generator self-host sources across repeated runs. The
remaining full-parity gap inventory is maintained in `../SELFHOST_WORK.md`.
