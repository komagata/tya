# Self-Hosting Prototype

This directory contains the first Tya-written compiler pieces.

Current pipeline:

```sh
go run ./cmd/tya selfhost/lexer.tya examples/selfhost_input.tya > /tmp/selfhost.tokens
go run ./cmd/tya selfhost/parser.tya /tmp/selfhost.tokens > /tmp/selfhost.nodes
go run ./cmd/tya selfhost/checker.tya /tmp/selfhost.nodes
go run ./cmd/tya selfhost/codegen_c.tya /tmp/selfhost.nodes > /tmp/selfhost.c
gcc /tmp/selfhost.c -o /tmp/selfhost
/tmp/selfhost
```

The current implementation is intentionally tiny. It proves that Tya can run
Tya-written compiler components before those components understand the full
language.

Current supported subset:

- Lexer: identifiers, ints, strings, comments, symbols, common two-character
  operators, source lines, and indentation counts
- Parser: simple assignment nodes and print nodes
- Checker: duplicate assignment node detection
- C codegen: C stubs and string print nodes
