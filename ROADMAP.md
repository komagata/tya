# Tya Roadmap

Tya is currently in the Go interpreter phase. The priority is to keep the
language small, pleasant, and executable before starting code generation.

## Phase 1: Go Interpreter

Status: first pass complete.

Implemented:

- Hand-written lexer, parser, AST, checker, and interpreter
- `.tya` runner CLI
- 2-space indentation and tab/trailing-whitespace errors
- Variables, constants, assignment, object property assignment, array index assignment
- `nil`, bool, int, float, string, array, object, function, error values
- Functions, methods with `@`, implicit last-expression returns, explicit `return`
- Multiple assignment and multiple return values
- Objects, inline object literals, arrays, indexing
- String interpolation and basic string escapes
- Arithmetic, comparison, equality, logical operators, grouped expressions, unary minus
- `if` / `else`, `while`, `break`, `continue`
- Array and object `for` loops
- Initial compile checks: constants, naming, duplicate members, undefined variables
- Standard library subset: printing, strings, collections, files, args/env, conversion, errors

Remaining hardening:

- Add parser and checker source spans to more AST nodes
- Add unused variable / unused argument checks
- Add variable shadowing checks beyond the current scope model
- Add top-level executable-code checks for non-`main.tya` files once module loading lands
- Improve checker scopes for reassignment versus duplicate definition

## Phase 2: Go Compiler That Emits C

Status: first pass complete.

Goals:

- Define a small C ABI for Tya values
- Emit C for expressions, statements, functions, objects, arrays, and control flow
- Reuse checker output before codegen
- Compile simple programs equivalent to current examples
- Keep the generated C readable for debugging

Implemented:

- Initial `--emit-c` CLI path
- C emission for simple numeric/string/bool expressions, assignments, `print`,
  `if`, `while`, `break`, and `continue`
- GCC smoke test for generated C

Remaining hardening:

- Emit C for functions, arrays, objects, and errors
- Emit C for the complete standard library surface
- Add source maps or generated-line diagnostics

## Phase 3: C Runtime

Status: first pass complete.

Goals:

- Implement dynamic `Value`
- Implement strings, arrays, objects, functions, and error values
- Implement standard library functions currently available in the interpreter
- Add mark-and-sweep GC
- Add runtime tests independent of the Go interpreter

Implemented:

- Initial `TyaValue` runtime for nil, bool, number, and string
- Runtime printing and truthiness helpers
- Generated C can compile and run against `runtime/tya_runtime.c`

Remaining hardening:

- Add arrays, objects, functions, and errors to the C runtime
- Add mark-and-sweep GC
- Add runtime tests independent of the Go interpreter

## Phase 4: Standard Library

Status: first pass complete.

Goals:

- Move more behavior from builtins into Tya code where practical
- Stabilize names and error conventions
- Add documentation examples for each standard function

Implemented:

- `docs/STDLIB.md` with examples for available standard functions
- `stdlib/prelude.tya` with candidate Tya-level helpers for future automatic loading
- Runner loads `stdlib/prelude.tya` for normal file execution

Remaining hardening:

- Move more builtins into Tya once imports/modules are available
- Stabilize error conventions after `try` sees more use
- Version the standard library API

## Phase 5: Self-Hosting

Status: started.

Sequence:

1. Write lexer in Tya
2. Write parser in Tya
3. Write checker in Tya
4. Write C code generator in Tya
5. Compile the Tya compiler with the Go compiler
6. Compile the Tya compiler with itself

Implemented:

- Initial `selfhost/lexer.tya` that tokenizes a useful subset of Tya source
- Initial `selfhost/parser.tya` that recognizes simple assignment and print nodes
- Initial `selfhost/checker.tya` that detects duplicate assignment nodes
- Initial `selfhost/codegen_c.tya` that emits C stubs from simple node lines
- Go test coverage for the lexer -> parser -> checker -> C codegen prototype pipeline
- Self-hosted lexer recognizes common two-character operators
- Self-hosted lexer emits source line and indentation-count tokens
- Self-hosted parser/codegen carries simple `if true` blocks into generated C

## Non-Goals For Now

- LLVM
- ANTLR
- Tree-sitter
- Package manager
- Async
- Classes
- Macros
- Exceptions
