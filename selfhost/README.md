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
- Parser: simple assignment, `if`, `while`, indentation, and print nodes
- Checker: duplicate assignment node detection and simple undefined print names
- C codegen: string/int assignments, simple integer addition assignments,
  bool assignments, string/int/bool print nodes, and simple `if true` /
  `while false` blocks
