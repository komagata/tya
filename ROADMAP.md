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
- Wire optional unused variable / unused argument checks into a stricter CLI mode if desired
- Add variable shadowing checks beyond the current scope model
- Add top-level executable-code checks for non-`main.tya` files once module loading lands
- Improve checker scopes for reassignment versus duplicate definition

Implemented hardening:

- Checker exposes optional `CheckUnused` analysis for unused bindings and function arguments
- CLI exposes `--check-unused` for strict unused binding checks
- Checker reports source locations for invalid `for` loop binding names

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
- C emission for arrays, objects, named functions, method calls, function values,
  multiple return assignment, `try`, errors, and key standard builtins
- Generated C parity checks cover executable examples and `args` / `env`
- `--emit-c` loads `stdlib/prelude.tya` like normal execution

Remaining hardening:

- Emit C for the remaining edge cases in the complete standard library surface
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
- Generated-C array runtime supports `pop` and index assignment
- Initial string runtime support for length, indexing, and `contains`
- Generated-C string runtime supports `trim`, `replace`, `startsWith`, `endsWith`, `byteLen`, and `charLen`
- Runtime equality helper for nil, bool, number, string, and array identity
- Runtime logical helpers preserve Tya `and` / `or` value semantics in generated C
- Runtime addition helper for numeric addition and string concatenation
- Initial generated-C runtime support for `args`, `readFile`, `split`, `toString`, and `toInt`
- Generated-C runtime supports `env`
- Initial object runtime with literal construction, member reads, and `len`
- Generated-C object runtime supports `has`, `keys`, `values`, and `delete`
- Generated-C object runtime supports `for key, value of object`
- Generated-C runtime supports `fileExists`, `toFloat`, and `toNumber`
- Generated-C runtime supports `writeFile`
- Generated-C runtime supports deep `equal`
- Runtime printing and truthiness helpers
- Runtime `toString` renders arrays and objects structurally
- Generated C can compile and run against `runtime/tya_runtime.c`
- C runtime has direct GCC-backed Go tests for values, collections, objects, function calls, strings, files, conversions, and process exits

Remaining hardening:

- Add mark-and-sweep GC
- Broaden runtime tests for more nested object and error paths

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
- Self-hosted C codegen emits simple `hasT(...)` string predicate conditions used by the prototype input
- Self-hosted parser/codegen preserves and emits the prototype `len(parts) < 3` condition
- Self-hosted parser/checker/codegen carries simple negated call conditions
- Self-hosted parser/checker/codegen carries simple call-based `while` conditions
- Self-hosted parser/checker/codegen emits simple direct comparison `while` conditions
- Self-hosted parser/checker/codegen carries simple zero-argument call indexing
- Self-hosted parser/checker/codegen carries simple call-with-call-index arguments
- Self-hosted parser/codegen handles simple integer addition assignments
- Self-hosted parser/codegen handles simple comparison assignments
- Self-hosted parser/codegen handles simple unary `not` assignments
- Self-hosted parser/codegen handles empty array placeholders
- Self-hosted C codegen emits a one-element array path for simple `push` and `for x in xs`
- Self-hosted parser/codegen carries simple `while false` blocks into generated C
- Self-hosted codegen emits simple variable conditions for `if` / `while`
- Self-hosted parser/checker/codegen carries simple `for x in xs` blocks
- Self-hosted parser/codegen carries simple `break` / `continue` statements
- Self-hosted parser/checker/codegen carries simple `push` commands
- Self-hosted parser/checker/codegen carries simple `return` commands
- Self-hosted C codegen emits simple `identity x` / `echo x` value-copy calls
- Self-hosted parser/codegen emits simple bool assignments and prints
- Self-hosted parser/checker/codegen carries simple one-argument print calls
- Self-hosted C codegen emits simple `print identity x` / `print echo x` calls
- Self-hosted C codegen emits placeholder declarations for call/index assignments
- Self-hosted C codegen skips function bodies until real function emission lands
- Self-hosted source files now parse, check, generate C, and compile as C smoke tests
- Go C emitter can emit compile-smoke C for self-host lexer/parser/checker/codegen sources
- Go C emitter can run simple generated-C array programs with `len`, indexing, and `push`
- Go C emitter can run `examples/array.tya`
- Go C emitter can run simple generated-C string programs with `len`, indexing, and `contains`
- Go C emitter uses runtime equality and truthiness for generated logical comparisons
- Go C emitter uses runtime addition for generated numeric addition and string concatenation
- Go C emitter can run simple generated-C file/argument/split/conversion programs
- Go C emitter can run simple generated-C array `for` loops
- Go C emitter can emit and run simple named Tya functions in generated C
- Go C emitter predeclares generated C locals and mangles reserved identifiers used by self-host sources
- Go C emitter expands simple string interpolation in generated C
- Go C emitter expands member paths in generated-C string interpolation
- Go C emitter expands simple addition expressions in generated-C string interpolation
- Go-emitted self-host lexer/parser/checker/codegen can compile and run a simple `.tya` source file through generated C
- Go C emitter can run simple generated-C object literal and member access programs
- Go C emitter can run `examples/function.tya`
- Go C emitter can run `examples/arithmetic.tya`
- Go C emitter can run `examples/string.tya`
- Go C emitter can run object, conversion, and file examples
- Go C emitter can run `examples/equal.tya`
- Go C emitter can run `examples/for.tya`
- Go C emitter can run `examples/for_object.tya`
- Go C emitter can run `examples/logic.tya`
- Go C emitter can run `examples/args.tya` with arguments and environment variables
- Go C emitter can run `examples/hello.tya`, `examples/return.tya`, and `examples/exit.tya`
- Go C emitter supports `readLine()` and can run `examples/read_line.tya`
- Go C emitter supports `exit(code)` statement calls with process exit status
- Go C emitter supports `panic(message)` with stderr output and exit status 1
- Go C emitter avoids C keyword name collisions and supports integer `%`
- Go C emitter supports function values for array builtins and can run `examples/array_function.tya`
- Go C emitter supports calling function values through aliases and indexed expressions
- Go C emitter supports function literals as runtime function values
- Parser supports no-paren calls with function literal arguments such as `map items, item -> item`
- Go C emitter supports `error(message)` values and can run `examples/error.tya`
- Go C emitter supports object-literal methods with `@field` access and can run `examples/method.tya`
- Go C emitter supports tuple-style multiple return assignment and can run `examples/multiple_return.tya`
- Go C emitter supports assignment-form `try` and can run `examples/try.tya`
- `--emit-c` loads `stdlib/prelude.tya` like normal execution and can run `examples/prelude.tya`
- Scripted generated-C parity checks for selected examples against interpreter output
- Self-hosted pipeline can compile and run `examples/while.tya`

## Non-Goals For Now

- LLVM
- ANTLR
- Tree-sitter
- Package manager
- Async
- Classes
- Macros
- Exceptions
