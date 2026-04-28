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

The current implementation is intentionally tiny. It proves that Tya can run
Tya-written compiler components before those components understand the full
language.

Current supported subset:

- Lexer: identifiers, ints, strings, comments, symbols, common two-character
  operators, source lines, and indentation counts
- Parser: simple assignment, simple function headers and inline returns,
  `push`, `return`, `if`, one-, two-, and three-argument function calls, `while`,
  `for`, `break`, `continue`, direct comparison and `or` conditions,
  one-argument call conditions, call comparison conditions, negated call conditions, call-based `while`
  conditions, call-with-call-index arguments, call indexing, return calls,
  indexing, indentation, and print nodes
- Checker: simple undefined assignment / print / condition names
- C codegen: string/int assignments, simple integer addition assignments,
  simple comparison assignments, variable-copy assignments, bool assignments,
  unary `not`, empty array placeholders, reassignment, carried function headers
  and inline returns, one-, two-, and three-argument calls, `push` and `return`
  commands, string/int/bool print nodes, carried indexing, call indexing,
  call-with-call-index arguments, one-argument call conditions,
  call comparison conditions, direct comparison conditions, simple `or`
  conditions, negated call conditions, call-based
  `while` conditions, return calls, placeholder call/index assignment
  declarations, self-host source compile smoke checks, and simple literal or
  variable `if` / `while` / `for` blocks with `break` / `continue`
