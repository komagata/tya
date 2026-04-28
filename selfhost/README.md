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
- Parser: simple assignment, `push`, `return`, `if`, `while`, `break`,
  `continue`, indentation, and print nodes
- Checker: simple undefined assignment / print / condition names
- C codegen: string/int assignments, simple integer addition assignments,
  simple comparison assignments, variable-copy assignments, bool assignments,
  unary `not`, empty array placeholders, reassignment, carried `push` and
  `return` commands, string/int/bool print nodes, and simple literal or
  variable `if` / `while` blocks with `break` / `continue`
