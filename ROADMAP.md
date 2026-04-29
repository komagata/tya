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
- CLI parses combinable mode flags such as `--check-unused --emit-c`
- Checker reports source locations for invalid `for` loop binding names
- Checker reports source locations for invalid and duplicate function parameter names
- Checker reports source locations for invalid and duplicate object property names

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
- Generated C includes source-line comments for Tya assignment statements
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
- Stage-2 self-host codegen emits deterministic C for supported subset fixtures
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
- Self-hosted parser/codegen carries simple `else` blocks
- Self-hosted parser/checker/codegen carries simple `!=`, `>=`, and `<=` conditions
- Self-hosted parser/checker/codegen carries simple `!=`, `>=`, and `<=` comparison assignments
- Self-hosted parser/checker/codegen carries simple `!=`, `>=`, and `<=` `while` conditions

Self-Host Completion TODO:

- [x] Autonomous work protocol
  - [x] Keep `SELFHOST_WORK.md` as the restart point and task queue for continuing without confirmation
- [x] Lexer parity
  - [x] Tokenize all literals supported by the Go lexer: nil, bool, int, float, string escapes
    - [x] Tokenize ints, floats, identifiers, bool/nil words, and basic string escapes in the self-host lexer
  - [x] Tokenize all operators and delimiters used by the language
    - [x] Tokenize current Go lexer operators and delimiters in the self-host lexer output format
  - [x] Preserve enough source span data for useful self-host diagnostics
  - [x] Add lexer golden tests comparing Go lexer output and Tya lexer output
- [ ] Parser parity
  - [ ] Replace line-oriented node stubs with a structured AST format
  - [ ] Parse grouped expressions and normal precedence for arithmetic, comparison, equality, and logical operators
    - [x] Parse grouped integer addition assignments in the self-host parser subset
    - [x] Parse grouped comparison assignments in the self-host parser subset
    - [x] Parse simple `and` / `or` assignments in the self-host parser subset
  - [ ] Parse arrays, inline objects, indented objects, member access, indexing, and assignment targets
    - [x] Parse one-element array literals in the self-host parser subset
    - [x] Parse two-element array literals in the self-host parser subset
    - [x] Parse one-property inline object literals in the self-host parser subset
  - [ ] Parse function literals, method literals, calls with and without parentheses, and function-value calls
    - [x] Parse one-argument parenthesized function calls in assignment expressions
    - [x] Parse two-argument parenthesized function calls in assignment expressions
  - [ ] Parse multiple assignment, multiple return, `try`, `for in`, `for of`, `break`, `continue`, and `else`
  - [ ] Add parser golden tests comparing Go parser AST shape and Tya parser AST shape
    - [x] Add subset parser golden tests for assignments, comparisons, blocks, `else`, `while`, `push`, and `for`
- [ ] Checker parity
  - [ ] Implement lexical scopes for top-level, functions, loops, and blocks
    - [x] Keep simple `if`, `while`, and `for` block-local names out of outer self-host checker scopes
  - [ ] Check undefined names, duplicate constants, duplicate params, duplicate object members, and invalid assignment targets
    - [x] Reject duplicate simple function params in the self-host checker subset
    - [x] Reject invalid simple function and loop binding names in the self-host checker subset
  - [ ] Check control-flow placement for `break`, `continue`, `return`, and `try`
  - [ ] Port optional unused binding checks or decide they stay Go-only
  - [ ] Add checker parity tests against Go checker diagnostics
    - [x] Add subset checker parity tests for undefined variable diagnostics
    - [x] Add subset checker coverage for duplicate params and invalid binding names
- [ ] Self-hosted C codegen parity
  - [ ] Emit real C functions instead of comments / skipped function bodies
    - [x] Emit simple one-argument identity return function bodies in the self-host C codegen subset
  - [ ] Emit full expression lowering through the existing C runtime ABI
    - [x] Emit simple calls to generated one-argument self-host C functions
  - [ ] Emit arrays, objects, member access, index access, methods with `@`, and object property assignment
    - [x] Emit one-property object placeholders in the self-host C codegen subset
    - [x] Emit empty arrays and `push` as dynamic string arrays in the self-host C codegen subset
  - [ ] Emit loops, conditionals, `try`, multiple return assignment, function values, and standard builtins
    - [x] Lower `readFile args()[0]` and argv-capable `main` in the self-host C codegen subset
    - [x] Add a plain-C lexer helper scaffold for stage-2 token emission
    - [x] Lower `split(source, "\\n")` to a generated-C line splitter for stage-2 parser input
  - [ ] Add generated-C parity tests comparing interpreter output and self-hosted codegen output
    - [x] Add subset generated-C parity tests for `examples/selfhost_ops.tya`
- [ ] Bootstrap pipeline
  - [ ] Make the Tya-written compiler compile the existing executable examples
    - [x] Make the Go-emitted stage-1 self-host compiler compile and run `examples/selfhost_ops.tya`
  - [ ] Compile the Tya-written compiler with the Go compiler and run it as the stage-1 compiler
    - [x] Compile self-host compiler components with the Go C emitter and run the stage-1 pipeline on `examples/hello.tya`
  - [ ] Use the stage-1 compiler to compile the Tya-written compiler again
    - [x] Use stage-1 compiler binaries to emit C for the self-host compiler sources
    - [x] Compile stage-1 emitted self-host C into stage-2 binaries
    - [x] Lower stage-2 input file reads through `readFile args()[0]`
    - [x] Add generated-C lexer helper scaffold for stage-2 token emission
    - [x] Run the stage-2 lexer on `examples/hello.tya`
    - [x] Tokenize integer literals in the stage-2 generated lexer
    - [x] Tokenize arrows and two-character comparison operators in the stage-2 generated lexer
    - [x] Tokenize float literals and basic string escapes in the stage-2 generated lexer
    - [x] Lower stage-2 newline splitting for parser input files
    - [x] Preserve stage-2 parser token lines with dynamic arrays and `push`
    - [x] Run the stage-2 parser on `examples/hello.tya`
    - [x] Parse integer literal assignments in the stage-2 generated parser
    - [x] Parse float and string literal assignments in the stage-2 generated parser
    - [x] Run the stage-2 checker on `examples/hello.tya`
    - [x] Run the stage-2 checker on literal assignments
    - [x] Run the stage-2 codegen output for `examples/hello.tya`
    - [x] Run the stage-2 codegen output for literal assignments
    - [x] Run a stage-2 pipeline for printing an assigned integer
    - [x] Run a stage-2 pipeline for printing assigned string and float values
    - [x] Run a stage-2 pipeline for integer addition
    - [x] Run a stage-2 pipeline for boolean assignment and print
    - [x] Run a stage-2 pipeline for equality comparison
    - [x] Run a stage-2 pipeline for inequality comparison
  - [ ] Compare stage-1 and stage-2 generated C for deterministic output
  - [x] Add a single `scripts/selfhost_bootstrap_check.sh` that runs the current bootstrap gate
- [ ] Documentation and release readiness
  - [x] Document supported self-host subset versus full language
  - [x] Document bootstrap commands and expected artifacts
  - [ ] Mark Phase 5 complete only after `go test ./...` and bootstrap checks pass from a clean checkout

## Non-Goals For Now

- LLVM
- ANTLR
- Tree-sitter
- Package manager
- Async
- Classes
- Macros
- Exceptions
