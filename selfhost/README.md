# Self-Hosting Prototype

This directory contains the first Tya-written compiler pieces.

Current pipeline:

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

The current implementation is intentionally tiny. It proves that Tya can run
Tya-written compiler components before those components understand the full
language.

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
checker, and C code generator self-host sources across repeated runs. This is
still a prototype subset. The remaining full-parity gap inventory is maintained
in `../SELFHOST_WORK.md`.
