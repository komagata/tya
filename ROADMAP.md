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
- Initial array runtime with length, indexing, and push support
- Initial string runtime support for length, indexing, and `contains`
- Runtime equality helper for nil, bool, number, string, and array identity
- Runtime addition helper for numeric addition and string concatenation
- Initial generated-C runtime support for `args`, `readFile`, `split`, `toString`, and `toInt`
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
- Scripted self-host source checks for lexer, parser, checker, and C codegen
- Self-hosted lexer recognizes common two-character operators
- Self-hosted lexer emits source line and indentation-count tokens
- Self-hosted parser/codegen carries simple `if true` blocks into generated C
- Self-hosted checker detects simple undefined assignment names
- Self-hosted checker detects simple undefined print names
- Self-hosted checker detects simple undefined condition names
- Self-hosted C codegen emits simple string/int assignments and prints
- Self-hosted C codegen emits simple variable-copy assignments
- Self-hosted C codegen emits simple reassignments
- Self-hosted parser/checker/codegen carries simple function headers
- Self-hosted parser/checker/codegen carries simple inline function returns
- Self-hosted parser/checker/codegen carries simple return calls
- Self-hosted parser/checker/codegen carries simple one-argument calls
- Self-hosted parser/checker/codegen carries simple two-argument calls
- Self-hosted parser/checker/codegen carries simple three-argument calls
- Self-hosted parser/checker/codegen carries simple indexing
- Self-hosted self-host example covers variable index reads
- Self-hosted parser/checker/codegen carries simple direct comparison conditions
- Self-hosted parser/checker/codegen carries simple `or` comparison conditions
- Self-hosted parser/checker/codegen carries simple one-argument call conditions
- Self-hosted parser/checker/codegen carries simple call-and-call conditions
- Self-hosted parser/checker/codegen carries simple call comparison conditions
- Self-hosted parser/checker/codegen carries simple negated call conditions
- Self-hosted parser/checker/codegen carries simple call-based `while` conditions
- Self-hosted parser/checker/codegen carries simple zero-argument call indexing
- Self-hosted parser/checker/codegen carries simple call-with-call-index arguments
- Self-hosted parser/codegen handles simple integer addition assignments
- Self-hosted parser/codegen handles simple comparison assignments
- Self-hosted parser/codegen handles simple unary `not` assignments
- Self-hosted parser/codegen handles empty array placeholders
- Self-hosted parser/codegen carries simple `while false` blocks into generated C
- Self-hosted codegen emits simple variable conditions for `if` / `while`
- Self-hosted parser/checker/codegen carries simple `for x in xs` blocks
- Self-hosted parser/codegen carries simple `break` / `continue` statements
- Self-hosted parser/checker/codegen carries simple `push` commands
- Self-hosted parser/checker/codegen carries simple `return` commands
- Self-hosted parser/codegen emits simple bool assignments and prints
- Self-hosted parser/checker/codegen carries simple one-argument print calls
- Self-hosted C codegen emits placeholder declarations for call/index assignments
- Self-hosted C codegen skips function bodies until real function emission lands
- Self-hosted source files now parse, check, generate C, and compile as C smoke tests
- Go C emitter can emit compile-smoke C for self-host lexer/parser/checker/codegen sources
- Go C emitter can run simple generated-C array programs with `len`, indexing, and `push`
- Go C emitter can run simple generated-C string programs with `len`, indexing, and `contains`
- Go C emitter uses runtime equality and truthiness for generated logical comparisons
- Go C emitter uses runtime addition for generated numeric addition and string concatenation
- Go C emitter can run simple generated-C file/argument/split/conversion programs
- Go C emitter can run simple generated-C array `for` loops

## Non-Goals For Now

- LLVM
- ANTLR
- Tree-sitter
- Package manager
- Async
- Classes
- Macros
- Exceptions
