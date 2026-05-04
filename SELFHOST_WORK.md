# Self-Host Work

This is the canonical internal planning document for Tya self-hosting work.
User-facing language documentation belongs in `docs/REFERENCE.md`,
`docs/GUIDE.md`, and `docs/STDLIB.md`; roadmap-level summaries belong in
`ROADMAP.md`.

Delete this file after full self-hosting is complete and the remaining work
queue is empty.

## Current Status

The supported-subset bootstrap gate is complete:

```sh
go test ./... -count=1
sh scripts/selfhost_bootstrap_check.sh
```

The bootstrap pipeline currently:

- compiles the Tya-written lexer, parser, checker, and C generator with the Go
  C emitter
- uses generated tools through repeated stages
- runs every example marked `supported` in
  `scripts/selfhost_examples_manifest.txt`
- compares supported generated-program output with the Go interpreter
- reaches stable stage-7 generated C for the self-host compiler sources

This is not full language parity. The self-host compiler still uses a
line-oriented node-string parser, subset checker behavior, subset C lowering,
and some source-specific generated-tool paths.

## Commands

Run the complete gate from the repository root:

```sh
sh scripts/selfhost_bootstrap_check.sh
```

Expected output:

```text
selfhost bootstrap: ok
```

Run the development pipeline on the default fixture:

```sh
sh scripts/selfhost.sh
```

Focused checks:

```sh
sh scripts/selfhost_check.sh
sh scripts/selfhost_compile_check.sh
sh scripts/go_emit_selfhost_compile_check.sh
sh scripts/go_emit_selfhost_run_check.sh
sh scripts/selfhost_fixed_point_check.sh
```

## Bootstrap Pipeline

The bootstrap pipeline:

1. Compiles `selfhost/lexer.tya`, `selfhost/parser.tya`,
   `selfhost/checker.tya`, and `selfhost/codegen_c.tya` with the Go C emitter.
2. Uses the generated stage-1 tools to emit and compile later-stage tools.
3. Runs generated tools on every example marked `supported` in
   `scripts/selfhost_examples_manifest.txt`.
4. Compares generated binary output with the Go interpreter for supported
   examples.
5. Verifies byte-stable generated C for the self-host compiler sources.

The explicit fixed-point gate is:

```sh
sh scripts/selfhost_fixed_point_check.sh
```

It is also covered by `sh scripts/selfhost_bootstrap_check.sh`.

## Example Parity

Example parity status is tracked in:

```text
scripts/selfhost_examples_manifest.txt
```

Each example is classified as:

- `supported`: run by the stage-generated self-host tools in the bootstrap gate
- `expected-failing`: runnable by the Go implementation, but missing a listed
  self-host feature
- `out-of-scope`: fixture or support file, not a standalone parity target

`go test ./tests -run Selfhost -count=1` fails if a new example lacks a
self-host parity classification.

## Self-Host Sources

The Tya-written compiler pieces are:

- `selfhost/lexer.tya`
- `selfhost/parser.tya`
- `selfhost/checker.tya`
- `selfhost/codegen_c.tya`

## Work Protocol

When continuing self-host work:

1. Read this file.
2. Pick the first unchecked task in `Current Queue`.
3. Implement the smallest useful slice that moves that task forward.
4. Add or update focused tests for that slice.
5. Run the focused self-host script first when applicable.
6. Run `go test ./... -count=1`.
7. Run `sh scripts/selfhost_bootstrap_check.sh`.
8. Update this file and `ROADMAP.md` only when the status meaningfully changes.
9. Commit with `Masaki Komagata <komagata@gmail.com>`.
10. Continue unless there is a true blocker.

Only stop for a true blocker:

- tests cannot be made to pass after a bounded fix attempt
- a design choice would invalidate existing language semantics
- the work requires external input that cannot be inferred from the repository

## Current Queue

- [ ] Replace line-oriented parser shortcuts with structured AST parsing.
  - [ ] Preserve nested expression structure instead of flattening ad hoc node
    strings such as `ASSIGN:*:CALL*`, `IF_COMPARE_*`, and `PRINT_CALL*`.
  - [ ] Parse the full expression grammar from the Go parser: precedence for
    mixed arithmetic, comparison, equality, logical operators, grouped
    expressions, unary `not` and unary minus, method calls, member access,
    indexing, and calls with arbitrary expression arguments.
  - [ ] Parse function literals in expression positions without relying on
    source-specific fallback paths.
  - [ ] Parse full statement and definition forms: object blocks with methods
    and property assignment, array index assignment, imports, constants,
    implicit last-expression returns, multi-value assignment/return beyond the
    current two-value cases, and `try` propagation in general expression
    positions.

- [ ] Bring the self-host checker to Go checker parity.
  - [ ] Model lexical scopes, block/function boundaries, reassignment, and
    shadowing consistently with `internal/checker`.
  - [ ] Carry index/key loop bindings distinctly instead of collapsing supported
    `for` forms to one value binding.
  - [ ] Enforce constants, imports/module public-binding rules, object member
    names, method receiver rules, duplicate declarations, optional unused
    checks, break/continue/return placement, and naming diagnostics with source
    line parity.
  - [ ] Check all expression forms and builtin arities rather than only the
    current node-string subset.

- [ ] Replace prototype C lowering with general executable code generation.
  - [ ] Replace source-specific functional array paths with general
    closure/function-value lowering.
  - [ ] Emit real functions, closures/function values, methods with `@`,
    object and array mutation, indexing, imports/prelude loading, error values,
    `try`, multi-return values, interpolation, unary operations, and all
    standard-library calls documented in `docs/STDLIB.md`.
  - [ ] Remove generated-C fallback stubs and example-specific recognizers;
    generated tools should be produced from parsed self-host source, not
    source-name or line-pattern special cases.
  - [ ] Generate C against the runtime ABI used by the Go emitter, or document
    and converge any intentionally smaller ABI.

- [ ] Broaden bootstrap parity gates.
  - [ ] Promote every runnable `examples/*.tya` and selected
    `examples/classic/*.tya` program from `expected-failing` to `supported` in
    `scripts/selfhost_examples_manifest.txt`.
  - [ ] Add negative parser/checker fixtures for unsupported or invalid
    language features as they become supported.
  - [ ] Keep deterministic C comparisons for each generated stage and example
    category.

## Next Unsupported Example Dependency

`examples/class.tya` is the next broad unsupported dependency. It requires
class declarations, constructors, instance fields, instance member access, and
method calls in the self-host parser/checker/codegen pipeline. Recent language
work also added class statics and predicate methods, so class parity should
account for:

- unprefixed instance fields and methods
- `@field` instance access
- `@@field` / `@@method` class members
- predicate names ending in `?`

## Verification Order

Use the smallest relevant command first, then the full gates:

```sh
sh scripts/stage1_selfhost_sources_check.sh
go test ./... -count=1
sh scripts/selfhost_bootstrap_check.sh
```

## Source Map

- `selfhost/lexer.tya`: Tya-written lexer
- `selfhost/parser.tya`: Tya-written parser for the current node-string subset
- `selfhost/checker.tya`: Tya-written checker for the current node-string subset
- `selfhost/codegen_c.tya`: Tya-written C generator
- `scripts/selfhost_examples_manifest.txt`: example parity classification
- `scripts/stage1_selfhost_sources_check.sh`: stage-generated supported-example
  parity and repeated-stage checks
- `scripts/selfhost_bootstrap_check.sh`: top-level self-host gate

## Completion Criteria

Supported-subset self-hosting is complete when `go test ./... -count=1` and
`sh scripts/selfhost_bootstrap_check.sh` pass from a clean checkout.

Full self-hosting is complete only when:

- the Tya-written compiler reaches a stable fixed point without source-specific
  fallback behavior
- generated tools implement the same lexer/parser/checker/codegen behavior as
  the Go implementation for the full language
- every runnable non-fixture example is a generated-tool parity target
- the documented standard library surface is supported by generated code
