# Tya Roadmap

Tya is a small indentation-based dynamic language implemented in Go. The
project is currently past the first interpreter/compiler pass and is working
toward full self-hosting and class/module completion.

Detailed self-host work tracking lives in `SELFHOST_WORK.md`. This file keeps
only project-level phase status.

## Phase 1: Go Interpreter

Status: first pass complete.

Implemented:

- Hand-written lexer, parser, AST, checker, and interpreter
- `.tya` runner CLI
- 2-space indentation and tab/trailing-whitespace errors
- Variables, constants, assignment, object property assignment, array index
  assignment
- `nil`, bool, int, float, string, array, object, function, error values
- Functions, methods with `@`, implicit last-expression returns, explicit
  `return`
- Multiple assignment and multiple return values
- Objects, inline object literals, arrays, indexing
- String interpolation and basic string escapes
- Arithmetic, comparison, equality, logical operators, grouped expressions,
  unary minus
- `if` / `else`, `while`, `break`, `continue`
- Array and object `for` loops
- Initial compile checks: constants, naming, duplicate members, undefined
  variables
- Standard library subset: printing, strings, collections, files, args/env,
  conversion, errors

Remaining hardening:

- Add parser and checker source spans to more AST nodes
- Wire optional unused variable / unused argument checks into a stricter CLI
  mode if desired
- Add variable shadowing checks beyond the current scope model
- Add top-level executable-code checks for non-`main.tya` files once module
  loading lands
- Improve checker scopes for reassignment versus duplicate definition

## Phase 2: Go Compiler That Emits C

Status: first pass complete.

Implemented:

- Initial `--emit-c` CLI path
- C emission for scalar expressions, assignments, `print`, `if`, `while`,
  `break`, and `continue`
- C emission for arrays, objects, named functions, method calls, function
  values, multiple return assignment, `try`, errors, and key standard builtins
- Generated C includes source-line comments for Tya assignment statements
- Generated-C parity checks cover executable examples and `args` / `env`
- `--emit-c` loads `stdlib/prelude.tya` like normal execution

Remaining hardening:

- Emit C for the remaining edge cases in the complete standard library surface
- Add source maps or generated-line diagnostics
- Finish class/module generated-C lowering as those language features stabilize

## Phase 3: C Runtime

Status: first pass complete.

Implemented:

- Initial `TyaValue` runtime for nil, bool, number, and string
- Array runtime with length, indexing, push, pop, and index assignment support
- String runtime support for length, indexing, `contains`, `trim`, `replace`,
  `starts_with`, `ends_with`, `byte_len`, and `char_len`
- Runtime equality, truthiness, logical helpers, and addition helper
- Generated-C runtime support for `args`, `env`, `read_file`, `write_file`,
  `split`, `to_string`, `to_int`, `to_float`, `to_number`, and `file_exists`
- Object runtime with literal construction, member reads, `len`, `has`, `keys`,
  `values`, `delete`, and object iteration
- Generated C can compile and run against `runtime/tya_runtime.c`
- GCC-backed Go tests for values, collections, objects, function calls,
  strings, files, conversions, and process exits

Remaining hardening:

- Add mark-and-sweep GC
- Broaden runtime tests for more nested object and error paths

## Phase 4: Standard Library

Status: first pass complete.

Implemented:

- `docs/STDLIB.md` with examples for available standard functions
- `stdlib/prelude.tya` with candidate Tya-level helpers
- Runner loads `stdlib/prelude.tya` for normal file execution

Remaining hardening:

- Move more builtins into Tya once imports/modules are complete
- Stabilize error conventions after `try` sees more use
- Version the standard library API

## Phase 5: Self-Hosting

Status: complete automated bootstrap gate for the supported subset; full
language parity remains.

Implemented:

- Tya-written lexer, parser, checker, and C generator under `selfhost/`
- Stage-generated toolchain that reaches stable stage-7 generated C for the
  self-host compiler sources
- Supported-example parity gate driven by
  `scripts/selfhost_examples_manifest.txt`
- Single bootstrap gate:

```sh
sh scripts/selfhost_bootstrap_check.sh
```

Remaining hardening:

- Replace line-oriented parser shortcuts with structured expression parsing for
  the full Go parser grammar
- Bring self-host checker diagnostics and scope rules to Go checker parity
- Replace source-specific generated-C paths with general codegen for the full
  language and documented standard library
- Promote examples classified as `expected-failing` in
  `scripts/selfhost_examples_manifest.txt` to generated-tool parity targets
- Keep repeated-stage and byte-stable generated-C checks passing as the
  supported subset expands

See `SELFHOST_WORK.md` for the current task queue and completion criteria.

## Class, Module, Dictionary, And Set Work

Status: class and module support is in progress; dictionary and set separation
is still pending.

`docs/CLASS_MODULE_DESIGN.md` describes the intended semantics. Current state:

- Curly and indented literals are still parsed as object literals, not separate
  dictionaries.
- Empty `{}` is an empty object. There is no set literal or `set()` builtin.
- The `.` operator reads and writes members on the current object value type.
- `@property` works in object methods and class instance methods.
- Class declarations support constructors, instance fields, property reads,
  bound method calls, class fields, and class methods.
- `self`, `super`, inheritance, and interface checking remain unimplemented.
- Initial `module` declarations define namespace values with `.` member access.
- Imports are handled by source loading before lexing/parsing the combined
  program.
- Import aliases are not implemented.
- Entry files execute top-level statements directly.
- The C emitter supports current objects, methods, dictionaries, sets,
  member/index access, and imports through source loading; dedicated class
  lowering remains incomplete.

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
5. Finish class semantics: `self`, `super`, inheritance, override arity checks,
   interfaces, and generated-C parity.
6. Finish module/import semantics: aliases, default imports, conflict checks,
   entry-file rules, and generated-C parity.
7. Update self-host example classifications as class/module/dictionary/set
   features become supported.

## Non-Goals For Now

- LLVM
- ANTLR
- Tree-sitter
- Package manager
- Async
- Macros
- Exceptions
