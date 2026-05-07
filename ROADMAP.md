# Tya Roadmap

`ROADMAP.md` is the single source of truth for current TODO, TASK, and roadmap
planning.

Pre-v0.1 planning documents and self-host migration notes are archived under
[`docs/archive/pre-v0.1/`](docs/archive/pre-v0.1/). They are historical
references, not current language or implementation authority.

## Roadmap Structure

Roadmap item definitions and maintenance rules live in
[`docs/ROADMAP_STRUCTURE.md`](docs/ROADMAP_STRUCTURE.md).

## Current Direction

Tya v0.1 is frozen as a small compile-to-C language. The authoritative
specification is:

1. [`docs/SPEC.md`](docs/SPEC.md)
1. [`docs/API.md`](docs/API.md)
1. [`docs/NAMING.md`](docs/NAMING.md)

The v0.1 reference implementation target is:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
C runtime
v0.1 specification tests
```

Go interpreter behavior, current `selfhost/*`, ASTMODE, legacy node strings,
and self-host bootstrap gates are not v0.1 authority. A Tya-written compiler
should be restarted after v0.1 works through the Go compile-to-C path, and it
must be AST-based from the start.

## Implementation Tooling Policy

The v0.1 compiler implementation should stay hand-written:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
```

Do not add a parser generator or large grammar framework for v0.1. In
particular, avoid introducing Participle, goyacc, Pigeon, ANTLR, or Tree-sitter
as compiler-front-end authority. They may be useful references or future editor
tooling, but the v0.1 compiler path should remain explicit Go code.

Use small test-support dependencies where they make the v0.1 specification
easier to verify:

```text
github.com/google/go-cmp/cmp
github.com/rogpeppe/go-internal/testscript
```

Use `go-cmp` for readable token, AST, diagnostic, and generated-output diffs.
Use `testscript` for CLI-level specification tests, especially `tya run`,
`tya build`, expected stdout/stderr, and negative examples.

## Current Roadmap

- [x] Finish the v0.1 Go compile-to-C reference implementation
  - [x] Align syntax and AST with the v0.1 spec
    - [x] Add `elseif` parsing and AST/checker/C-emitter support.
    - [x] Rename current object-literal semantics to dictionary semantics.
    - [x] Reject dictionary member access and keep `.` for module members.
    - [x] Add explicit `module name` declarations for imported module files.
    - [x] Reject parser-level v0.1 exclusions for class, interface, object,
      set, self, super, import alias, and no-paren calls.
  - [x] Align the checker with the v0.1 spec
    - [x] Enforce v0.1 naming and reserved-name rules.
    - [x] Enforce module-file rules: exactly one module matching the file name.
    - [x] Enforce the v0.1 exclusion list for class, interface, object, set,
      `@property`, `self`, `super`, and import alias syntax.
  - [x] Align generated C and runtime behavior with the v0.1 spec
    - [x] Ensure every v0.1 language example runs through `tya run`.
    - [x] Ensure dictionary iteration, indexing, mutation, and builtins emit C.
    - [x] Ensure module import and `module.member` calls emit C.
  - [x] Trim builtins to the v0.1 API
    - [x] Keep only the builtins listed in `docs/API.md`.
    - [x] Remove or quarantine tests that require non-v0.1 convenience
      builtins.
  - [x] Add v0.1 specification tests
    - [x] Add `go-cmp` for readable token, AST, diagnostic, and generated
      output diffs.
    - [x] Add `testscript` for CLI-level compile-to-C specification tests.
    - [x] Add focused parser/checker/C-emitter tests for every v0.1 feature.
    - [x] Add negative parser/lexer tests for v0.1 excluded syntax.
    - [x] Add an example parser gate based on v0.1 examples only.
    - [x] Add parser AST golden tests for core v0.1 AST shapes.
  - [x] Align the user-facing CLI with the v0.1 spec
    - [x] Define the v0.1 command set as `tya run`, `tya build`, and
      `tya version`.
    - [x] Make `tya run <file.tya> [args...]` build a temporary executable,
      run it, and clean up the temporary file.
    - [x] Add `tya build <file.tya> -o <output>` for writing an executable to
      disk.
    - [x] Add `tya version` while keeping `tya --version` as compatibility if
      useful.
    - [x] Move developer-only inspection commands behind a non-user-facing
      surface instead of documenting them on the public path.

- [x] Keep pre-v0.1 implementation paths out of the critical path
  - [x] Downgrade the Go interpreter path
    - [x] Treat it as optional or test-only instead of v0.1 authority.
    - [x] Avoid using interpreter-only behavior as a passing condition.
  - [x] Downgrade current self-host work
    - [x] Keep current `selfhost/*` and bootstrap scripts as archived
      implementation references.
    - [x] Remove self-host bootstrap scripts from default v0.1 verification.
    - [x] Quarantine legacy self-host tests behind explicit
      `selfhost_legacy && pre_v01_legacy_ast` build tags.

- [x] Start the Tya-written compiler after v0.1 is stable
  - [x] Build a new AST-based Tya compiler
    - [x] Define and keep the Tya-side AST representation before parser
      expansion. Statement and expression nodes are dictionaries with `kind`
      fields, stale `try_assign` node support has been removed, and the
      `--ast` verification output includes core expression/function/block
      shapes.
    - [x] Implement lexer, parser, checker, and C emitter against that AST.
      - [x] Complete the v0.1 lexer slice in the Tya-written compiler.
      - [x] Complete the v0.1 AST-based parser slice in the Tya-written
        compiler.
      - [x] Complete the v0.1 checker slice in the Tya-written compiler.
      - [x] Complete the v0.1 C emitter slice in the Tya-written compiler.
      Initial lexer tokenization and diagnostics are in place for v0.1 tokens,
      comments, indentation, tab rejection, string escapes, unterminated
      strings, unknown escapes, and unexpected characters. Initial `print` /
      assignment / multiple assignment / binary expression
      (`+`, `-`, `*`, `/`, `%`, `and`, `or`, comparison, equality) with
      initial precedence layering / `if` / multiple `elseif` /
      `else` / `while` / array / dictionary / index / index assignment /
      unary expression (`not`, `-`) / grouped expression / `for value in` /
      `for value, index in` / `for key, value of` slice is in place.
      Parser-owned diagnostics reject same-line unconsumed statement tokens
      such as no-paren calls and malformed block headers such as
      `elseif ready true`. Parser diagnostics reject v0.1 excluded words such
      as `class`, `interface`, `object`, `set`, `self`, and `super`, and reject
      set-literal-shaped `{}` forms.
      Parser state rejects `return` outside functions and `break` / `continue`
      outside loops, including loop-control statements inside nested functions.
      Parser-owned binding diagnostics reject reserved names in assignment,
      import/module, function parameter, and loop binding positions. Parser
      diagnostics now also reject member assignment and non-identifier multiple
      assignment targets before those shapes can reach later phases.
      Initial `break` / `continue` loop control is in place.
      Initial recursive block parsing and C emission for nested `if` /
      `while` / `for` statements inside blocks is in place.
      Expression-body function literal with zero to four arguments and matching
      function call slice is in place. Initial explicit-`return` block function
      and implicit last-expression return block function slice with zero to four
      arguments is in place. Call arguments support one-parameter function
      literals such as `map(items, item -> item * 2)`, and expression functions
      support `() -> value`. Initial parser AST dump verification is in place.
      Initial two-value `return` / tuple-style multiple assignment slice is in
      place.
      Initial `try` expression parsing is in place. `name = try call(...)`
      emits error propagation from block functions, and expression-body /
      final-expression `try` follows the current Go oracle behavior.
      Parser state rejects `try` outside function bodies.
      Initial checker scope hardening is in place for block function bodies and
      function parameters. Initial v0.1 checker diagnostics are in place for
      invalid variable/module/member names, duplicate function parameters,
      duplicate module members, duplicate/invalid dictionary keys, constant
      reassignment, and `.` access on dictionaries, arrays, and non-module
      values. Imported module files are checked for exactly one module
      declaration matching the imported file name and reject top-level
      non-import/non-module statements. Checker traversal now recurses through nested
      `if` / `while` / `for` blocks, while keeping block-local names from
      leaking outward.
      Initial `import module_name` / `module name` / `module.member` slice is
      in place for same-directory module loading and generated C module member
      calls.
      Initial identifier string interpolation slice is in place.
      Initial pure builtin calls (`len`, `has`, `keys`, `values`,
      `to_string`) are in place. Initial collection mutation and string builtin
      calls (`push`, `pop`, `delete`, `split`, `join`, `trim`, `replace`,
      `contains`, `starts_with`, `ends_with`) are in place. Initial conversion,
      file, process, error, panic, and exit builtin calls are in place. Initial
      one-level nested call/index arguments and binary call arguments are in
      place for builtin-oriented expressions such as `len(keys(user))`,
      `join(split(text, ","), ":")`, and `sum(2 + 3, 4)`.
      Initial top-level function value references from generated C functions
      are supported through globally visible C bindings.
      Initial C emitter assigned-name predeclaration is in place so reassigned
      variables inside generated C blocks remain visible at the correct outer
      C scope. C emitter expression lowering now uses AST expression nodes for
      nested builtin arguments and side-effect builtin statements such as
      `push`, `delete`, `write_file`, `panic`, and `exit`, so expression
      arguments are not lowered through atom-only fallbacks.
      The Tya-written compiler now self-compiles through a fixed-point gate:
      Go-generated stage1 builds stage2, stage2 builds stage3, and stage2 C
      output matches stage3 C output.
    - [x] Do not reintroduce legacy node strings or source-specific fallbacks.
  - [x] Use Go compile-to-C as the oracle
    - [x] Compare Tya compiler output against the v0.1 Go C-emitter behavior
      for the initial slice.
    - [x] Add a fixed-point test for the Tya-written compiler output.
    - [x] Add a stage2 compiler practical-input compile/run gate.
    - [x] Grow feature parity from small v0.1 slices, not from the old
      self-host bootstrap pipeline.

## Verification Reference

Default verification:

```sh
go test ./... -count=1
```

Focused verification should prefer tests for the touched lexer, parser, checker,
C emitter, runtime, examples, or docs. Self-host bootstrap checks are historical
pre-v0.1 gates and are not default v0.1 verification.
