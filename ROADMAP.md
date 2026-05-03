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

Status: complete automated bootstrap gate for the supported subset; full
language parity remains.

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
- Self-hosted checker rejects invalid assignment binding names and constant
  reassignment in the supported node subset
- Self-hosted C codegen emits simple string/int assignments and prints
- Self-hosted C codegen emits simple variable-copy assignments
- Self-hosted C codegen emits simple reassignments
- Self-hosted parser/checker/codegen carries simple function headers
- Self-hosted parser/checker/codegen carries simple inline function returns
- Self-hosted parser/checker/codegen carries simple return calls
- Self-hosted parser/checker carries `return { name: value }, nil` nodes
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
- Self-hosted parser/codegen handles simple boolean `and` / `or` assignments
- Self-hosted parser/checker/codegen carries simple `-`, `*`, `/`, and `%`
  arithmetic assignments
- Self-hosted C codegen emits simple `hasT(...)` string predicate conditions used by the prototype input
- Self-hosted parser/codegen preserves and emits the prototype `len(parts) < 3` condition
- Self-hosted C codegen emits string array index assignments for static and dynamic arrays
- Stage-2 generated codegen can lower `readFile args()[0]`, `lex source`, and `parse tokens`
- Stage-3 generated lexer tokenizes `examples/hello.tya`
- Stage-3 generated parser parses stage-3 lexer output for `examples/hello.tya`
- Stage-3 generated checker accepts stage-3 parser output for `examples/hello.tya`
- Stage-3 generated codegen emits, compiles, and runs C for `examples/hello.tya`
- Stage-3 generated tools compile all four selfhost sources into stage-4 binaries
- Stage-4 generated tools tokenize, parse, check, emit, compile, and run `examples/hello.tya`
- Stage-4 generated tools also execute a second single-line string print fixture
- Stage-4 generated tools execute a single-line integer print fixture
- Stage-4 generated tools preserve INT token/node kinds for integer print fixtures
- Stage-4 generated tools execute an escaped-quote string print fixture
- Stage-4 generated tools preserve colon characters in printed string nodes
- Stage-4 generated tools execute a two-line print fixture
- Stage-4 generated tools execute a string assignment plus print fixture
- Stage-4 generated tools execute an integer assignment plus print fixture
- Stage-4 generated tools execute an integer reassignment plus print fixture
- Stage-4 generated tools execute an integer addition assignment fixture
- Stage-4 generated tools execute a less-than comparison fixture
- Stage-4 generated tools execute a while false fixture with skipped block body
- Stage-4 generated tools execute a one-element array for fixture
- Stage-4 generated tools execute `examples/multiple_return.tya`
- Stage-4 generated tools execute `examples/while.tya`
- Stage-4 generated tools execute `examples/string.tya`
- Stage-4 generated tools execute `examples/selfhost_ops.tya`
- Stage-4 generated tools execute `examples/arithmetic.tya`
- Stage-4 generated tools execute `examples/function.tya`
- Stage-4 generated tools execute `examples/return.tya`
- Stage-4 generated tools execute `examples/object.tya`
- Stage-4 generated tools execute `examples/object_inline.tya`
- Stage-4 generated tools execute `examples/if.tya`
- Stage-4 generated tools execute `examples/logic.tya`
- Stage-4 generated tools execute `examples/error.tya`
- Stage-4 generated tools execute `examples/convert.tya`
- Stage-4 generated tools execute `examples/file.tya`
- Stage-4 generated tools execute `examples/args.tya`
- Stage-4 generated tools execute `examples/equal.tya`
- Stage-4 generated tools execute `examples/array.tya`
- Stage-4 generated tools execute `examples/for.tya`
- Stage-4 generated tools iterate every example marked supported in
  `scripts/selfhost_examples_manifest.txt` and compare generated binary output
  with the Go interpreter
- Stage-3 parser emits non-empty lexer-driver nodes for `selfhost/lexer.tya`
- Stage-3 codegen emits executable lexer C from real lexer-driver nodes
- Stage-3 parser emits non-empty parser-driver nodes for `selfhost/parser.tya`
- Stage-3 parser emits non-empty checker-driver nodes for `selfhost/checker.tya`
- Self-hosted parser/checker/codegen carries simple negated call conditions
- Self-hosted parser/checker/codegen carries simple call-based `while` conditions
- Self-hosted parser/checker/codegen emits simple direct comparison `while` conditions
- Self-hosted parser/checker/codegen carries simple zero-argument call indexing
- Self-hosted parser/checker/codegen carries simple call-with-call-index arguments
- Self-hosted parser/codegen handles simple integer addition assignments
- Self-hosted parser/codegen handles parenthesized integer addition assignments
- Self-hosted parser/codegen handles simple equality, inequality, and bounds comparison assignments
- Stage-4 generated tools compile all four self-host compiler sources into stage-5 C binaries
- Stage-5 generated tools run `examples/hello.tya` plus print-string, print-int, and two-print fixtures
- Stage-5 tool source compiles into stage-6 binaries that run print-string, print-int, and two-print fixtures
- Stage-6 tool source emits stable stage-7 C for all four self-host compiler sources
- `scripts/selfhost_fixed_point_check.sh` proves byte-stable stage-4
  generated C for `selfhost/lexer.tya`, `selfhost/parser.tya`,
  `selfhost/checker.tya`, and `selfhost/codegen_c.tya`
- `scripts/selfhost_bootstrap_check.sh` is the single documented self-host
  bootstrap gate, covering source checks, generated-C compile checks,
  supported example parity, repeated bootstrap stages, and fixed-point
  generated-C stability
- Self-hosted parser/codegen handles parenthesized bounds comparison assignments
- Self-hosted parser/codegen handles simple unary `not` assignments
- Self-hosted parser/codegen handles empty array placeholders
- Self-hosted C codegen emits a one-element array path for simple `push` and `for x in xs`
- Stage-2 self-host pipeline can run `examples/selfhost_ops.tya`
- Stage-2 self-host codegen emits literal reassignment without duplicate C declarations
- Stage-2 self-host codegen lowers `readFile args()[0]` with argv-capable generated C
- Stage-2 generated parser/checker covers `examples/multiple_return.tya` nodes
- Stage-2 self-host pipeline can run `examples/multiple_return.tya`
- Self-hosted parser/codegen carries simple `while false` blocks into generated C
- Self-hosted codegen emits simple variable conditions for `if` / `while`
- Self-hosted parser/checker/codegen carries simple `for x in xs` blocks
- Self-hosted parser/codegen carries simple `break` / `continue` statements
- Self-hosted parser/checker/codegen carries simple `push` commands
- Self-hosted parser/checker/codegen carries simple `return` commands
- Self-hosted C codegen emits simple `identity x` / `echo x` value-copy calls
- Self-hosted parser/codegen emits simple bool assignments and prints
- Self-hosted parser/codegen emits greater-or-equal and less-or-equal `while` conditions
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

## Class, Module, Dictionary, And Set Baseline

Status: class/module/dictionary/set implementation is in progress.

`docs/CLASS_MODULE_DESIGN.md` describes the intended semantics. The current
implementation has moved past the pre-class/module baseline, with remaining
import and C-emission gaps:

- Curly and indented literals are parsed as `ObjectLit`, not dictionaries.
  Empty `{}` is an empty object. There is no set literal or `set()` builtin.
- Inline object literal keys must be bare identifiers followed by `:`. Literal
  keys such as `"name": value`, set entries such as `{ "admin" }`, and mixed
  entries are parser errors today.
- The `.` operator reads and writes members on the current object value type.
  Dictionaries are not separate yet, so current examples such as `user.name`
  and imported `greeting.hello()` both use object member access.
- `@property` works in object methods and class instance methods. Initial
  `class` declarations support constructors, instance fields, property reads,
  and bound method calls, but `self`, `super`, inheritance, and interface
  checking remain unimplemented.
- Initial `module` declarations define namespace values with `.` member access.
  `extends`, `implements`, `interface`, and `super` still lex as identifiers
  and are not implemented.
- Imports are handled by source loading, before lexing/parsing the combined
  program. `import name` loads `name.tya` from the importing file's directory,
  recursively prepends imported source, rejects cycles, and requires imported
  files to define exactly one matching top-level `class` or `module`.
- Imported files may not contain top-level helper functions, variables, private
  assignments, or additional class/module declarations.
- Import aliases are not implemented. `import util as u` is rejected as an
  invalid module name today.
- Entry files execute top-level statements directly. They are not wrapped in an
  implicit `main` function.
- The C emitter works from the already-loaded combined AST. It supports current
  objects, methods, dictionaries, sets, member/index access, and imports through
  source loading. Module declarations lower to namespace objects, while class
  declarations remain explicitly rejected until dedicated lowering lands.

Ordered implementation checklist:

1. Rename the current object-literal concept to dictionary in the AST, checker,
   interpreter, C emitter, docs, examples, and diagnostics while preserving
   existing behavior during the transition.
2. Implement inline and indented dictionary literals, bracket access for
   dictionaries, empty `{}` as an empty dictionary, and diagnostics for mixed
   dictionary/set entries.
3. Add set literals and the empty-set constructor, including interpreter,
   builtins, C runtime/codegen support, and collection docs.
4. Separate dictionary access from object/member access so dictionaries, sets,
   and arrays reject `.` and dictionaries use `[]`.
5. Add `class` declarations, PascalCase name checks, constructors, instances,
   instance fields/methods, and `@property` behavior on class instances.
6. Add `module` declarations and module member access as namespace values
   rather than ordinary object literals.
7. Enforce one imported file per public `class` or `module`, including filename
   matching, no extra public/private top-level helpers, and clear diagnostics.
8. Implement Ruby-like default imports, `as` aliases, and import name conflict
   checks.
9. Add entry-file semantics: executable top-level entry code is wrapped in an
   implicit `main`, while entry files cannot directly define classes/modules.
10. Implement single inheritance, override arity checks, and explicit `super`
    calls in `init` and methods.
11. Implement interfaces and `implements` checks, including each class's
    implicit public API interface.
12. Finish generated-C/runtime parity, examples, self-host classification
    updates where applicable, and final user-facing docs.
- Scripted generated-C parity checks for selected examples against interpreter output
- Self-hosted pipeline can compile and run `examples/while.tya`
- Self-hosted parser/codegen carries simple `else` blocks
- Self-hosted parser/checker/codegen carries simple `!=`, `>=`, and `<=` conditions
- Self-hosted parser/checker/codegen carries simple `!=`, `>=`, and `<=` comparison assignments
- Self-hosted parser/checker/codegen carries simple `!=`, `>=`, and `<=` `while` conditions

Remaining hardening:

- Replace line-oriented parser shortcuts with structured expression parsing for
  the full Go parser grammar
- Bring self-host checker diagnostics and scope rules to Go checker parity
- Replace source-specific generated-C fallbacks with general codegen for the
  full language and documented standard library
- Promote the examples classified in `scripts/selfhost_examples_manifest.txt`
  from expected-failing to explicit generated-tool parity targets
- Keep the final bootstrap gate proving repeated self-compilation and
  byte-stable generated C as the self-host subset expands

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
    - [x] Parse three-argument parenthesized function calls in assignment expressions
    - [x] Keep unsupported function bodies out of stage-2 top-level nodes
    - [x] Parse `print` calls with three-argument builtin calls in the self-host parser subset
    - [x] Parse `print` calls with two-argument builtin calls in the self-host parser subset
  - [ ] Parse multiple assignment, multiple return, `try`, `for in`, `for of`, `break`, `continue`, and `else`
    - [x] Parse indexed `for item, index in items` loops in the self-host parser subset
    - [x] Parse `for key, value of object` loops in the self-host parser subset
    - [x] Parse two-target multiple assignment in the self-host parser subset
    - [x] Parse two-value return statements in the self-host parser subset
    - [x] Parse `target = try call(arg)` in the self-host parser subset
    - [x] Parse `left, right = call(arg)` in the self-host parser subset
    - [x] Parse `left, right = call "literal"` in the self-host parser subset
    - [x] Parse `return nil, error "message"` in the self-host parser subset
    - [x] Parse `print object.member` in the self-host parser subset
  - [ ] Add parser golden tests comparing Go parser AST shape and Tya parser AST shape
    - [x] Add subset parser golden tests for assignments, comparisons, blocks, `else`, `while`, `push`, and `for`
- [ ] Checker parity
  - [ ] Implement lexical scopes for top-level, functions, loops, and blocks
    - [x] Keep simple `if`, `while`, and `for` block-local names out of outer self-host checker scopes
  - [ ] Check undefined names, duplicate constants, duplicate params, duplicate object members, and invalid assignment targets
    - [x] Reject duplicate simple function params in the self-host checker subset
    - [x] Reject invalid simple function and loop binding names in the self-host checker subset
    - [x] Recognize `replace` as a self-host checker builtin for three-argument calls
    - [x] Check undefined names in two-value return nodes
    - [x] Check two-target multiple assignment nodes
    - [x] Check two-target assignment from one-argument calls
    - [x] Check literal arguments in two-target one-argument calls
    - [x] Check `return nil, error "message"` nodes
    - [x] Check `print object.member` base names
  - [ ] Check control-flow placement for `break`, `continue`, `return`, and `try`
    - [x] Reject `break` and `continue` outside loops in the self-host checker subset
    - [x] Reject return nodes outside functions in the self-host checker subset
    - [x] Reject top-level `try` in the self-host checker subset
  - [ ] Port optional unused binding checks or decide they stay Go-only
  - [ ] Add checker parity tests against Go checker diagnostics
    - [x] Add subset checker parity tests for undefined variable diagnostics
    - [x] Add subset checker coverage for duplicate params and invalid binding names
- [ ] Self-hosted C codegen parity
  - [ ] Emit real C functions instead of comments / skipped function bodies
    - [x] Emit simple one-argument identity return function bodies in the self-host C codegen subset
  - [ ] Emit full expression lowering through the existing C runtime ABI
    - [x] Emit simple calls to generated one-argument self-host C functions
    - [x] Emit `replace(text, old, new)` calls in the self-host C codegen subset
    - [x] Emit `print replace(text, old, new)` calls in the self-host C codegen subset
    - [x] Emit `print contains(text, needle)` calls in the self-host C codegen subset
    - [x] Emit `print startsWith(text, prefix)` and `print endsWith(text, suffix)` calls in the self-host C codegen subset
    - [x] Emit `trim(text)` calls in the self-host C codegen subset
    - [x] Emit `print len(value)` calls in the self-host C codegen subset
    - [x] Emit `print object.member` for one-property object placeholders
    - [x] Emit the current multiple-return example subset with string out-params
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
    - [x] Classify every example in `scripts/selfhost_examples_manifest.txt`
      as supported, expected-failing, or out-of-scope
    - [x] Make the Go-emitted stage-1 self-host compiler compile and run `examples/selfhost_ops.tya`
    - [x] Make stage-4 generated tools compile and run `examples/object.tya`
    - [x] Make stage-4 generated tools compile and run `examples/object_inline.tya`
    - [x] Make stage-4 generated tools compile and run `examples/if.tya`
    - [x] Make stage-4 generated tools compile and run `examples/logic.tya`
    - [x] Make stage-4 generated tools compile and run `examples/error.tya`
    - [x] Make stage-4 generated tools compile and run `examples/convert.tya`
    - [x] Make stage-4 generated tools compile and run `examples/file.tya`
    - [x] Make stage-4 generated tools compile and run `examples/args.tya`
    - [x] Make stage-4 generated tools compile and run `examples/equal.tya`
    - [x] Make stage-4 generated tools compile and run `examples/array.tya`
    - [x] Make stage-4 generated tools compile and run `examples/for.tya`
  - [ ] Compile the Tya-written compiler with the Go compiler and run it as the stage-1 compiler
    - [x] Compile self-host compiler components with the Go C emitter and run the stage-1 pipeline on `examples/hello.tya`
  - [ ] Use the stage-1 compiler to compile the Tya-written compiler again
    - [x] Use stage-1 compiler binaries to emit C for the self-host compiler sources
    - [x] Compile stage-1 emitted self-host C into stage-2 binaries
    - [x] Lower stage-2 input file reads through `readFile args()[0]`
    - [x] Run stage-5 generated tools on `examples/hello.tya`
    - [x] Run stage-5 generated tools on a print-string fixture
    - [x] Run stage-5 generated tools on a print-int fixture
    - [x] Run stage-5 generated tools on a two-print fixture
    - [x] Compile stage-6 tools from the stage-5 tool source and run a print-string fixture
    - [x] Run stage-6 generated tools on print-int and two-print fixtures
    - [x] Verify stable stage-7 C from the stage-6 tool source
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
    - [x] Run a stage-2 pipeline for printing string length
    - [x] Run a stage-2 pipeline for trimming and printing a string
    - [x] Run a stage-2 pipeline for printing string containment
    - [x] Run a stage-2 pipeline for printing string prefix and suffix checks
    - [x] Run a stage-2 pipeline for printing string replacement
    - [x] Run a stage-2 pipeline for printing escaped quote strings
    - [x] Preserve colon characters in stage-2 printed string nodes
    - [x] Run a stage-2 pipeline for splitting and joining strings
    - [x] Run a stage-2 pipeline for byte and character string lengths
    - [x] Run a stage-2 pipeline for replacement with a string literal replacement
    - [x] Run a stage-2 pipeline for string literal indexing
    - [x] Run a stage-2 pipeline for `examples/string.tya`
    - [x] Run a stage-2 pipeline for less-than comparison
    - [x] Run a stage-2 pipeline for integer addition reassignment
    - [x] Run a stage-2 pipeline for `while false` with `break`
    - [x] Run a stage-2 pipeline for less-than `while` with `break`
    - [x] Run a stage-2 pipeline for `examples/while.tya`
    - [x] Run a stage-2 pipeline for integer addition
    - [x] Run a stage-2 pipeline for boolean assignment and print
    - [x] Run a stage-2 pipeline for equality comparison
    - [x] Run a stage-2 pipeline for inequality comparison
  - [ ] Compare stage-1 and stage-2 generated C for deterministic output
  - [x] Add a single `scripts/selfhost_bootstrap_check.sh` that runs the current bootstrap gate
- [ ] Documentation and release readiness
  - [x] Document supported self-host subset versus full language
  - [x] Document bootstrap commands and expected artifacts
  - [x] Mark the supported-subset bootstrap gate complete after `go test ./...`
    and `sh scripts/selfhost_bootstrap_check.sh` pass from a clean checkout
  - [ ] Mark full Phase 5 language parity complete only after the remaining
    parser, checker, codegen, and example parity gaps are closed

## Non-Goals For Now

- LLVM
- ANTLR
- Tree-sitter
- Package manager
- Async
- Classes
- Macros
- Exceptions
