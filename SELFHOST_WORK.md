# Self-Host Work Protocol

This file is the restart point for autonomous self-hosting work.

## Non-Stop Protocol

When asked to continue self-hosting work:

1. Read `SELFHOST_WORK.md`, then `ROADMAP.md`.
2. Pick the first unchecked task in `Current Queue`.
3. Implement the smallest useful slice that moves that task forward.
4. Add or update focused tests for that slice.
5. Run the focused test first.
6. Run `go test ./... -count=1`.
7. Run `sh scripts/selfhost_bootstrap_check.sh`.
8. Update `ROADMAP.md` and this file when a slice is complete.
9. Commit with `Masaki Komagata <komagata@gmail.com>`.
10. Return to step 2 without waiting for confirmation.

Only stop for a true blocker:

- tests cannot be made to pass after a bounded fix attempt
- a design choice would invalidate existing language semantics
- the work requires external input that cannot be inferred from the repository

Do not stop merely because a commit was made. A commit is a checkpoint, not an
endpoint.

## Current Queue

- [x] Make the Go-emitted stage-1 self-host compiler run `examples/selfhost_ops.tya`
  - [x] Reproduce and capture the current C type mismatch from stage-1 codegen
  - [x] Preserve `INT` and `BOOL` assignment type information through Go-emitted `selfhost/codegen_c.tya`
  - [x] Ensure stage-1 generated C prints bool/int values with the correct format
  - [x] Add `examples/selfhost_ops.tya` to `scripts/selfhost_bootstrap_check.sh`
- [x] Expand parser subset toward full expression parsing
  - [x] Parse grouped comparison assignments
  - [x] Parse simple `and` / `or` assignments
  - [x] Parse two-or-more element array literals
  - [x] Parse simple inline object literals
- [x] Expand checker subset
  - [x] Track block-local names for `if`, `while`, and `for`
  - [x] Reject duplicate simple function params in self-host checker
  - [x] Add parity coverage for duplicate and invalid names
- [x] Expand self-host C codegen subset
  - [x] Emit simple function bodies instead of comments for one-expression returns
  - [x] Emit simple function calls to generated functions
  - [x] Emit object placeholders beyond comments
- [x] Bootstrap stage 2
  - [x] Use stage-1 compiler binaries to compile selfhost sources to C
  - [x] Compile stage-1 emitted selfhost C into stage-2 binaries
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
  - [x] Compare repeated stage-2 generated C for deterministic output
- [ ] Continue toward full self-host completion
  - [x] Promote completed lexer parity TODOs in `ROADMAP.md`
  - [ ] Expand self-host parser beyond line-oriented subset stubs
    - [x] Parse one-argument parenthesized function calls in assignment expressions
    - [x] Parse two-argument parenthesized function calls in assignment expressions
    - [x] Parse three-argument parenthesized function calls in assignment expressions
    - [x] Parse `print` calls with three-argument builtin calls
    - [x] Parse `print` calls with two-argument builtin calls
    - [x] Parse indexed `for item, index in items` loops in the self-host parser subset
    - [x] Parse `for key, value of object` loops in the self-host parser subset
    - [x] Parse two-target multiple assignment in the self-host parser subset
  - [ ] Expand self-host checker toward Go checker parity
    - [x] Recognize `replace` as a self-host checker builtin for three-argument calls
  - [ ] Expand self-host C codegen toward executable example parity
    - [x] Emit `replace(text, old, new)` calls in the self-host C codegen subset
    - [x] Emit `print replace(text, old, new)` calls in the self-host C codegen subset
    - [x] Emit `print contains(text, needle)` calls in the self-host C codegen subset
    - [x] Emit `print startsWith(text, prefix)` and `print endsWith(text, suffix)` calls in the self-host C codegen subset
    - [x] Emit `trim(text)` calls in the self-host C codegen subset
    - [x] Emit `print len(value)` calls in the self-host C codegen subset
  - [ ] Advance bootstrap from subset programs toward compiling existing examples
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
    - [x] Run a stage-2 pipeline for greater-or-equal and less-or-equal comparisons
    - [x] Run a stage-2 pipeline for grouped integer addition
    - [x] Run a stage-2 pipeline for grouped greater-or-equal comparison
    - [x] Run a stage-2 pipeline for boolean `and` / `or` assignments
    - [x] Run a stage-2 pipeline for greater-or-equal `while` with `break`
    - [x] Run a stage-2 pipeline for one-element string arrays and `for`
    - [x] Run a stage-2 pipeline for `examples/selfhost_ops.tya`
    - [x] Run a stage-2 pipeline for literal reassignment
    - [x] Run a stage-2 pipeline for `readFile args()[0]`
    - [x] Skip function bodies in the stage-2 parser subset
    - [ ] Advance stage-3 self-host compiler probe
      - [x] Generate, compile, and run a stage-3 lexer on `examples/hello.tya`
      - [x] Generate, compile, and run a stage-3 parser on stage-3 lexer output
      - [x] Generate and run the stage-3 checker on stage-3 parser output
      - [x] Generate and run the stage-3 codegen on stage-3 parser output
      - [x] Compile all stage-3 selfhost sources from stage-3 tools
      - [x] Make stage-4 generated tools execute `examples/hello.tya`
      - [x] Make stage-4 generated tools execute another bootstrap fixture beyond hello
      - [x] Expand stage-4 generated tools beyond single-line string print fixtures
      - [x] Preserve proper stage-4 token/node kinds for integer print fixtures
      - [x] Expand stage-4 generated tools to escaped string print fixtures
      - [x] Preserve colon characters in stage-4 printed string nodes
      - [x] Expand stage-4 generated tools to two-line print fixtures
      - [x] Expand stage-4 generated tools to assignment plus print fixtures
      - [x] Expand stage-4 generated tools to integer assignment plus print fixtures
      - [x] Expand stage-4 generated tools to reassignment plus print fixtures
      - [x] Expand stage-4 generated tools to integer addition assignment fixtures
      - [x] Expand stage-4 generated tools to less-than comparison fixtures
      - [x] Expand stage-4 generated tools to while/break fixtures
      - [x] Expand stage-4 generated tools to one-element array for fixtures
      - [ ] Replace stage-4 generated-tool fallback stubs with real generated selfhost parser/codegen paths
        - [x] Make stage-3 parser emit non-empty nodes for `selfhost/lexer.tya`
        - [x] Make stage-3 codegen emit executable lexer C from real lexer-driver nodes
        - [x] Make stage-3 parser emit non-empty nodes for `selfhost/parser.tya`
        - [x] Make stage-3 parser emit non-empty nodes for `selfhost/checker.tya`
        - [x] Make stage-3 parser emit non-empty nodes for `selfhost/codegen_c.tya`
        - [x] Make stage-3 codegen emit executable parser C from real parser-driver nodes
        - [x] Make stage-3 codegen emit executable checker C from real checker-driver nodes
        - [x] Make stage-3 codegen emit executable codegen C from real codegen-driver nodes
        - [x] Replace stage-4 generated-tool mode fallback with source-specific generated tools
          - [x] Replace stage-4 checker mode fallback with source-specific checker C
          - [x] Replace stage-4 parser mode fallback with source-specific parser C
          - [x] Replace stage-4 lexer mode fallback with source-specific lexer C
          - [x] Replace stage-4 codegen mode fallback with source-specific codegen C

## Last Resolved Blocker

The Go C emitter used to drop user-defined function call statements such as
`rememberType names, types, name, "INT"`. That made the Go-emitted stage-1
self-host codegen lose type metadata and emit `puts(adult)` for bool/int
values. Generic user function call statements are now emitted and
`scripts/go_emit_selfhost_ops_check.sh` covers the stage-1 path.
