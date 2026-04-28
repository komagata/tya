# Self-Hosting Prototype

This directory contains the first Tya-written compiler pieces.

Current pipeline:

```sh
sh scripts/selfhost.sh
```

The current implementation is intentionally tiny. It proves that Tya can run
Tya-written compiler components before those components understand the full
language.

Current supported subset:

- Lexer: identifiers, ints, strings, comments, symbols, common two-character
  operators, source lines, and indentation counts
- Parser: simple assignment, simple function headers, `push`, `return`, `if`,
  one- and two-argument function calls, `while`, `for`, `break`, `continue`,
  indexing, indentation, and print nodes
- Checker: simple undefined assignment / print / condition names
- C codegen: string/int assignments, simple integer addition assignments,
  simple comparison assignments, variable-copy assignments, bool assignments,
  unary `not`, empty array placeholders, reassignment, carried function headers,
  one- and two-argument calls, `push` and `return` commands, string/int/bool
  print nodes, carried indexing, and simple literal or variable `if` / `while`
  / `for` blocks with `break` / `continue`
