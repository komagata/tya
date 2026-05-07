# Tya Roadmap

`ROADMAP.md` is the single source of truth for TODO, TASK, and roadmap
planning. Supporting documents may explain design, usage, or verification
details, but remaining work must be summarized here.

## Roadmap Structure

Roadmap item definitions and maintenance rules live in
[`docs/ROADMAP_STRUCTURE.md`](docs/ROADMAP_STRUCTURE.md).

## Update Policy

To keep this file stable, do not edit `ROADMAP.md` for every passing test,
fixture adjustment, helper extraction, or narrow implementation slice. Report
that progress in chat using the Roadmap Structure format.

Edit this file only when one of these reasons applies:

1. A remaining Task is fully completed and should be removed.
1. A Milestone or Epic changes scope.
1. The medium- or long-term strategy changes.
1. The verification policy changes.

When an edit reason does not apply, leave this file unchanged even if useful
work was completed.

## Current Status

The supported-subset self-host bootstrap gate is complete:

```sh
go test ./... -count=1
sh scripts/selfhost_bootstrap_check.sh
```

Current self-hosting facts:

- The Tya-written lexer, parser, checker, and C generator compile through the
  self-host bootstrap pipeline.
- The bootstrap pipeline uses generated tools through repeated stages.
- The stage gate runs every example marked `supported` in
  `scripts/selfhost_examples_manifest.txt`.
- Supported generated-program output is compared with the Go interpreter.
- The repeated-stage gate reaches stage-7 stable C for all four self-host
  compiler sources.
- ASTMODE now emits direct statement nodes for the self-host compiler source
  statement kinds instead of leaving `AST_STMT` placeholders.
- ASTMODE checker accepts all four self-host compiler sources.
- ASTMODE codegen emits C that compiles for all four self-host compiler
  sources, but the generated programs are not yet verified as behaviorally
  equivalent to the legacy/generated-tool pipeline.
- ASTMODE codegen can lower simple one-argument string helper functions,
  including multiple `if arg == "value"` conditional string returns followed by
  a default `return arg`.
- ASTMODE parser now emits `AST_EXPR` for full-line expression statements in
  ASTMODE, which covers implicit last-expression returns such as the
  `tokens` tail expression in `lex(source)`.
- ASTMODE parser can keep simple four-argument parenthesized calls, bare
  multi-argument calls, and one-level nested call arguments as one `call(...)`
  expression while preserving the legacy parser's three-argument adapter limit
  outside ASTMODE.
- ASTMODE codegen can lower simple one-argument array-return helper functions
  when the function builds an array from string literals and the function
  argument. More complex array-return functions are intentionally not registered
  until their bodies can be lowered generically.
- ASTMODE codegen can execute a minimal array-return function body with a
  `while` loop, local integer counter, dynamic array `push`, and implicit array
  return. It also handles a narrow `if ... continue` pattern inside that loop.
  The loop counter name, initial integer value, and integer step are now derived
  from AST condition/local assignment nodes instead of being fixed to `i = 0`
  and `i = i + 1`. This is still a narrow generic slice, not full function-body
  lowering.
- ASTMODE codegen can use a local identifier alias assignment inside an
  array-return helper body as the value of a later `push`, including inside a
  loop. The same alias can also feed a local string-accumulation assignment,
  which is a small step toward statement-by-statement function-body lowering.
- Local aliases in array-return helper bodies can now come from both identifiers
  and string literals, so `item = "done"; push items, item` lowers through the
  same alias path.
- ASTMODE codegen now has a shared `ast_string_expr_c` helper for lowering
  string-like AST expressions used by array literals, object literal values,
  stdlib string-call arguments, function-call arguments, dynamic `push`, and
  `panic`. This is a small expression-lowering adapter step before broadening
  call argument support.
- Array-return helper registration now uses small shared predicates for
  string/argument push validation and function-local name lookup. This reduces
  duplicated collector validation before moving toward statement-by-statement
  function-body lowering.
- Array-return helper output now computes loop metadata once per generated
  helper and reuses the derived loop variable and step for loop entry,
  `continue`, and final increment emission.
- Array-return helper dynamic array `push` output now goes through a small
  `emit_dynarray_push` helper, keeping the repeated `realloc` / `dup_text` /
  length increment sequence in one emission path.
- Main-body AST `push` and legacy node `PUSH` dynamic-array output now also use
  `emit_dynarray_push`, so function-body and main-body dynamic push emission
  share the same helper.
- Main-body AST `push` and legacy node `PUSH` fixed-array output now use
  `emit_single_array_push`, so fixed and dynamic push emission both have shared
  statement-level helpers.
- Empty dynamic array initialization for AST `array0` now uses
  `emit_empty_dynarray`, starting the same statement-level helper split for
  array initialization.
- AST `array1` initialization now uses `emit_array1_init`. AST `array2`
  initialization has started moving in the same direction with an arity-safe
  `emit_array2_init` helper for the length and first element.
- AST `array3` initialization now also uses an arity-safe `emit_array3_init`
  helper for the length and first element. Remaining second and third element
  emission stays inline until a broader statement-emission environment removes
  the current arity pressure.
- AST `array2` and `array3` remaining element declarations now use
  `emit_array_item_init`, so all current fixed-array literal element
  declarations go through statement-level helpers.
- Array-return helper function bodies now use `emit_empty_dynarray` for their
  local result-array initialization, sharing the same empty-array emission path
  as main-body AST `array0`.
- Array-return helper string locals and alias locals now use
  `emit_string_local`, moving simple local variable declarations into the same
  statement-level helper style as array initialization and push output.
- Array-return helper string accumulation assignment now uses
  `emit_string_concat_assign`, so the narrow `text = text + part` lowering has
  a statement-level emission helper instead of inline C string assembly.
- Array-return helper loop `if ... continue` guard output now uses
  `emit_continue_guard`, moving the condition, loop increment, and `continue`
  block into a single statement-level helper.
- Array-return helper loop counter updates now use `emit_loop_increment` from
  both normal loop tail emission and `continue` guard emission.
- Array-return helper loop initialization and `while` opening now use
  `emit_loop_start`, so loop start, loop increment, and continue guard emission
  are all helper-backed.
- Array-return helper loop body emission now records supported body statements
  in AST node order before emitting them, so `push` and string accumulation
  statements keep source order instead of being grouped by statement kind.
- Array-return helper function tail now uses `emit_array_return`, moving
  `*out_len` assignment and array return into a shared helper.
- Array-return helper function header now uses `emit_array_func_header`, so the
  generated helper function wrapper is also partially helper-backed.
- Simple string helper function output now uses `emit_string_func_header` and
  `emit_string_return`, sharing wrapper/tail emission style with array-return
  helpers.
- Simple string helper function conditional returns now use
  `emit_string_case_return`, so string-case helper output follows the same
  statement-level helper style instead of adding another inline generated-C
  pattern.
- ASTMODE parser now renders two-value returns as `AST_RETURN2` instead of
  falling back to `AST_STMT:return`, and the self-host checker validates
  identifier references in both returned expressions.
- ASTMODE parser now renders two-target assignments as `AST_MULTI_ASSIGN2`,
  and the self-host checker validates both target names plus identifier
  references in the assigned expression.
- ASTMODE parser now renders two-argument statement calls such as
  `write_file path, trim name` as `AST_CALL_STMT2`; checker validates the
  argument expressions, and both direct and generated codegen can execute the
  `write_file` AST statement path.
- Generated AST codegen now executes a small `AST_DELETE` object-member
  deletion node stream, including `AST_ASSIGN:object1(...)` setup and
  `AST_PRINT:member(...)` output after deletion.
- Generated AST codegen now executes `AST_EXIT:int(...)` and
  `AST_PANIC:string(...)` node streams with the expected process status and
  panic stderr output.
- Generated AST codegen now executes `AST_BREAK` and `AST_CONTINUE` node
  streams inside simple `AST_WHILE` loops.
- Generated AST codegen now executes `AST_FOR_INDEX` node streams over
  fixed string-array AST assignments while preserving both item and index
  bindings.
- Generated AST codegen now executes integer-comparison `AST_IF` node streams
  with generic comparison operators such as `>=` and `!=`.
- Generated AST codegen now executes integer-step `AST_ASSIGN` updates for
  both `+` and `-` with arbitrary integer step values.
- Generated AST codegen now executes `AST_PRINT` node streams for integer,
  boolean, nil, and simple binary expressions.
- Generated AST codegen now executes shallow integer binary `AST_ASSIGN`
  expressions such as `int + int` and `ident * int`.
- Generated AST codegen now executes integer `ident op ident` binary
  `AST_ASSIGN` expressions when both operands are known integers.
- Generated AST codegen now executes string `ident + ident` binary
  `AST_ASSIGN` expressions when both operands are known strings, including
  self-updating concatenation.
- Generated AST codegen now executes string literal mixed concatenation
  `AST_ASSIGN` expressions such as `ident + string(...)` and
  `string(...) + ident`.
- Generated AST codegen now executes nested integer binary `AST_ASSIGN`
  expressions for parser precedence shapes such as
  `int + (int * int)`.
- Generated AST codegen now executes boolean, nil, and identifier-copy
  `AST_ASSIGN` nodes while preserving enough type information for later
  `AST_PRINT:ident(...)`.
- Generated AST codegen now executes nested integer binary `AST_PRINT`
  expressions for the same parser precedence shape covered by nested
  assignment expressions.
- Generated AST codegen now executes string `ident + ident` binary
  `AST_PRINT` expressions when both operands are known strings.
- Generated AST codegen now executes string literal mixed concatenation
  `AST_PRINT` expressions such as `ident + string(...)` and
  `string(...) + ident`.
- Generated AST codegen now uses one generated-C helper for AST binary
  comparison-operator classification in both shallow and nested `AST_PRINT`
  expression lowering.
- Generated AST codegen now uses one generated-C helper to decide whether an
  AST expression requires the `concat_text` helper, covering assign and print
  string concatenation forms.
- Generated AST codegen now uses one generated-C helper for known-name type
  checks in generated AST binary expression lowering, reducing duplicated
  `known_names` / `known_types` loops across assign and print paths.
- Generated AST codegen now executes `ident + call(trim ident(...))`
  string concatenation in both `AST_ASSIGN` and `AST_PRINT` paths.
- Generated AST codegen now executes `ident + call(to_string ident(...))`
  string concatenation in both `AST_ASSIGN` and `AST_PRINT` paths.
- Generated AST codegen now emits supported string-concat `AST_ASSIGN`
  results through one generated-C string assignment helper, reducing repeated
  declaration-versus-update branches across `ident + ident`, `trim`, and
  `to_string` concatenation forms.
- Generated AST codegen now emits supported string-concat `AST_PRINT` results
  through one generated-C string print helper, keeping assign/print concat
  compatibility paths aligned.
- ASTMODE codegen can now register simple `AST_RETURN2` multi-return helper
  bodies and execute `AST_MULTI_ASSIGN2` calls such as
  `user, err = parse_user "komagata"` without falling back to legacy
  `MULTI_ASSIGN2_CALL1` nodes.
- The full `examples/multiple_return.tya` surface now runs through the ASTMODE
  parser/checker/codegen pipeline, including string-key index prints such as
  `user["name"]` and string truthiness for `if err`.
- ASTMODE codegen now lowers boolean `and` / `or` binary expressions to C
  `&&` / `||`, including nested comparison operands such as
  `age >= 20 and name == "komagata"`.
- ASTMODE codegen now lowers integer array literals to `INTARRAY` storage and
  can execute `for item in items` over those arrays, covering
  `examples/classic/array_sum.tya`.
- Self-host checker builtin registration now goes through
  `remember_builtin_names`, and the registered standard builtin set covers
  conversion, file, collection, equality, object, and input helpers used by the
  examples.
- ASTMODE codegen can execute a narrow local string-accumulation pattern inside
  an array-return helper loop, including a post-loop `push` of the accumulated
  value.
- ASTMODE-generated lexer, parser, checker, and C generator binaries now run on
  a minimal `print "Hello"` pipeline: the generated lexer emits tokens, the
  generated parser emits a node stream, the generated checker accepts it, and
  the generated C generator emits C that compiles and runs. This is a minimal
  behavioral gate, not full self-host parity.
- The minimal ASTMODE-generated self-host pipeline also runs end-to-end with all
  four generated binaries chained together, instead of only testing each
  generated binary against Go-produced intermediate files.
- The ASTMODE-generated self-host pipeline now also runs a small string-array
  `for` program end-to-end, covering generated parser recognition of
  `ARRAY_TWO` string literals and generated codegen lowering of array iteration.
- The ASTMODE-generated self-host pipeline now runs an integer `while i < n`
  loop and a simple one-argument function call end-to-end, covering generated
  parser recognition of top-level `FUNC` nodes while skipping helper bodies,
  plus generated codegen emission of only the called simple identity helper.
- The generated self-host parser now has a real ASTMODE output path for a
  minimal `print "..."` token stream: `parse_ast` currently reuses the generated
  legacy parser as an intermediate, and `ast_nodes` converts legacy
  `PRINT:STRING` nodes to `AST_PRINT:string(...)`. This is the first generated
  ASTMODE parser slice, not full structural parser lowering.
- The generated self-host pipeline now accepts a small ASTMODE `if` / `else`
  program where generated parser output mixes legacy control nodes with
  `AST_PRINT:string(...)` statements, and generated codegen executes the AST
  print statements inside those legacy control blocks.
- The generated self-host parser ASTMODE path now also converts simple string
  assignments and identifier prints to `AST_ASSIGN:name:string(...)` and
  `AST_PRINT:ident(...)`, and the generated codegen compatibility layer can
  execute those AST string assignments while preserving later legacy control
  flow checks.
- The generated self-host parser ASTMODE path now converts simple string
  equality control flow to `AST_IF:binary(== ident(...) string(...))` and
  `AST_ELSE`, and the generated codegen compatibility layer can execute that
  AST control shape in the small generated pipeline.
- The generated self-host parser ASTMODE path now converts a small integer
  `while i < n` loop to `AST_WHILE:binary(< ident(i) int(n))`, along with
  integer initialization, identifier print, and `i = i + 1` as AST assignment
  nodes. The generated pipeline executes this AST while slice end-to-end.
- The generated self-host parser ASTMODE path now converts a small string-array
  `for item in items` program to `AST_ASSIGN:array2(...)`, `AST_FOR`, and
  `AST_PRINT:ident(...)`, and the generated pipeline executes that AST for-loop
  slice end-to-end.
- The generated self-host parser ASTMODE path now converts a simple
  one-argument function call program to `AST_FUNC`, `AST_ASSIGN:string(...)`,
  and `AST_PRINT:call(...)`, and the generated pipeline executes that AST call
  slice end-to-end. Function bodies are still skipped in this generated-parser
  slice, so this is not generic function-body lowering.
- The generated self-host parser ASTMODE path now preserves a simple function
  body `return value` as `AST_RETURN:ident(value)` while keeping legacy
  generated parser output on the previous function-body-skip path.
- The generated self-host parser ASTMODE path now also preserves a small
  array-return helper body shape: `items = []`, local identifier aliases,
  `push` of identifiers or string literals, and `return items` become
  `AST_ASSIGN:array0()`, `AST_ASSIGN:ident(...)`, `AST_PUSH`, and
  `AST_RETURN`.
- The ASTMODE-generated self-host pipeline now executes that small array-return
  helper shape end-to-end: generated parser emits the helper body AST,
  generated codegen emits a C helper returning a dynamic string array, and
  generated code can iterate the returned array.
- The ASTMODE-generated self-host pipeline now also executes a looping
  array-return helper shape with a local integer counter, `while i < n`,
  loop-body `push`, and counter increment.
- The ASTMODE-generated self-host pipeline now also executes a looping
  array-return helper shape with a narrow `if i == n` / counter increment /
  `continue` guard before the loop-body `push`.
- ASTMODE-generated array-return helper C output now routes dynamic string-array
  appends through a generated `push_text` helper instead of repeating the
  `realloc` / `strdup` sequence in every loop and non-loop push path.
- ASTMODE-generated parser/codegen now preserves supported array-return helper
  loop body statement order for `push` and string accumulation statements, so
  generated self-host tools match the Go-executed ASTMODE path for this
  statement-order slice.
- The ASTMODE-generated self-host pipeline now also executes alias-backed
  string accumulation inside an array-return helper loop, so `part = value`
  followed by `text = text + part` is covered in generated parser/checker/codegen
  end-to-end evidence.
- The ASTMODE-generated parser now preserves `ident + int` assignments as
  `AST_ASSIGN:binary(+ ident(...) int(...))`, allowing generated codegen to
  derive non-`i`, non-zero, non-`+1` array-return helper loop metadata from
  AST rather than from fixed counter assumptions.
- The ASTMODE-generated self-host pipeline now also executes a named-counter,
  non-unit-step array-return helper loop with an `if ... continue` guard,
  confirming that continue-path increments use derived loop metadata.
- Array-return helper statement records now keep a simple `AST_IF` condition
  with the statement body, so generated parser/checker/codegen can execute a
  conditional `push` without treating every helper-local `if` as a
  `continue` guard.
- The ASTMODE-generated parser now also preserves `ident + string` assignments
  as `AST_ASSIGN:binary(+ ident(...) string(...))`, and generated codegen can
  lower conditional plus unconditional string accumulation statements in an
  array-return helper loop.
- The ASTMODE-generated parser/codegen now executes helper-local conditional
  statements for `!=` and `>` integer comparisons in addition to `==`, reducing
  the generated array-return helper path's dependency on equality-only
  condition handling.
- The same ASTMODE-generated helper-local conditional statement path now also
  covers `>=` and `<=` integer comparisons, leaving fewer comparison operators
  outside the generated function-body lowering slice.
- The ASTMODE-generated helper-local conditional statement path now covers
  multiple statements in the same `if` body, including a string accumulation
  followed by a dynamic-array `push`.
- The ASTMODE-generated helper-local conditional statement path now also covers
  `else` bodies by carrying the previous `if` condition as a negated statement
  condition.
- The same helper-local `if` / `else` path now covers multiple statements per
  branch, including string accumulation followed by dynamic-array `push`.
- ASTMODE-generated parser now preserves every `INDENT` node in ASTMODE, and
  generated array-return helper codegen uses those indent nodes to emit
  post-loop `push` / string accumulation statements after the loop instead of
  moving them into the loop body.
- ASTMODE-generated array-return helper codegen now keeps loop-body emission and
  post-loop `if` / `else` emission in separate scan states, so post-loop
  conditional branches are no longer emitted inside the preceding `while`.
- ASTMODE-generated parser/codegen now carries generated array-return helper
  `while` conditions as `binary(op ident(...) int(...))` for supported integer
  comparisons instead of hard-coding `<`, and the generated helper loop emission
  executes the preserved comparison expression.
- ASTMODE-generated array-return helper `if ... continue` guards now preserve
  the `AST_IF` integer comparison operator instead of lowering every guard back
  to equality, so the generated helper loop can execute non-`==` continue
  conditions through the same guard path.
- ASTMODE-generated array-return helper loop metadata now records decrementing
  integer counter updates such as `count = count - 1` as a signed loop step,
  allowing generated helper loops to execute `while count > 0` without a
  separate loop emitter.
- ASTMODE-generated parser/codegen now preserves and executes a call expression
  as an array-return helper `push` value, starting with `push items, trim value`
  through the shared string-expression lowering path instead of treating `trim`
  as a plain identifier.
- ASTMODE-generated parser/codegen now also preserves and executes a call
  expression as the right-hand side of array-return helper string accumulation,
  starting with `text = text + trim value` through the same string-expression
  lowering and generated `trim_text` helper dependency path.
- The same generated ASTMODE call-expression string accumulation path now runs
  under helper-local `if` conditions, proving that conditional statement records
  reuse the expression lowering instead of requiring a separate `trim` branch.
- Generated ASTMODE post-loop `if` / `else` emission now also executes
  call-expression `push` values such as `push items, trim value`, keeping the
  post-loop statement pass aligned with the loop-body statement pass.
- ASTMODE-generated array-return helper codegen now carries `to_string` call
  expressions through the shared string-expression lowering path for both
  dynamic-array `push` values and string accumulation assignments. This covers
  helper-local integer values instead of only function arguments and `trim`
  string calls.
- ASTMODE-generated parser/codegen now preserves and executes string index
  assignment values inside array-return helpers, starting with
  `char = value[i]; push items, char`. This is a targeted step toward
  `lex(source)` because the self-host lexer repeatedly reads `source[i]`.
- ASTMODE-generated parser/codegen now also preserves and executes direct
  string index `push` values inside array-return helpers, starting with
  `push items, value[i]`. This moves index expressions through the same
  generated helper-body expression path instead of requiring a temporary alias.
- ASTMODE-generated parser/codegen now preserves and executes string index
  comparisons inside helper-local `if` conditions, including false conditions
  such as `if value[i] == "y"` when `value[i]` is not `y`. This prevents
  index-conditioned helper statements from being emitted unconditionally.
- ASTMODE-generated parser/codegen now preserves and executes compound
  helper-local `while` conditions combining an integer loop bound with a string
  index comparison, such as `while i < 3 and value[i] != "a"`. The generated
  loop stops on the index condition instead of only on the integer bound.
- ASTMODE parser/codegen now preserves and executes the same compound
  helper-local `while` shape when the loop bound is `len(value)`, such as
  `while i < len(value) and value[i] != "a"`. This removes the earlier
  ASTMODE `scalar_call_compare(nil)` fallback for this lexer-like pattern.
- ASTMODE-generated parser/codegen now preserves and executes string index
  values as the right-hand side of helper-local string accumulation, such as
  `text = text + value[i]` inside the `len(value)` compound loop shape.
- ASTMODE-generated parser C now uses a shared `ast_atom_from_token` helper for
  simple `IDENT` / `INT` / `STRING` atom rendering in binary assignment AST
  output. This reduces fixed-shape branches before broadening generated
  expression parsing.
- ASTMODE-generated parser C now also uses a shared `ast_index_from_tokens`
  helper for index operand rendering in binary assignment AST output, reducing
  the dedicated `ident + value[i]` string-shape branch.
- ASTMODE-generated parser C now also uses a shared `ast_call_from_tokens`
  helper for one-argument bare call operand rendering in binary assignment AST
  output, reducing the dedicated `ident + trim value` / `ident + to_string n`
  branch.
- ASTMODE-generated parser C now uses `ast_expr_from_tokens` as a shared
  expression-rendering entry point for simple binary assignment operands,
  delegating to atom, index, and call renderers.
- ASTMODE-generated parser C now also uses `ast_expr_from_tokens` for direct
  index `push` operands, so `push items, value[i]` shares the same expression
  renderer as binary assignment operands.
- ASTMODE-generated parser C now also uses `ast_expr_from_tokens` for string
  index comparison operands in helper-local `if` and compound `while`
  conditions.
- ASTMODE-generated parser C now uses a shared
  `ast_compound_while_condition` helper for the two supported compound
  `while` condition shapes, covering both integer bounds and `len(value)`
  bounds.
- ASTMODE-generated parser C now uses a shared `ast_binary_condition` helper
  for helper-local `if` binary condition output.
- ASTMODE-generated parser C now also uses `ast_binary_condition` plus
  `ast_expr_from_tokens` for simple integer-bound `while` comparisons such as
  `while i < 3`, `while count > 0`, and `while count <= 5`.
- ASTMODE-generated parser C now has exact token kind/text helpers and uses
  them for generated `if` / `else` and `push` recognizers, preventing string
  literals containing token-like text such as `:IDENT:if:` from being parsed as
  real syntax during repeated self-host stages.
- The same exact-token recognizer path now covers the generated parser's basic
  `print` shapes, preventing string literals containing token-like text such as
  `:IDENT:print:` from being parsed as print statements.
- The exact-token recognizer path now also covers generated parser `return`
  shapes, including string, identifier, nil/error, and object/nil returns.
- The exact-token recognizer path now also covers generated parser `while`
  shapes, including simple integer bounds, non-`<` integer bounds, and compound
  string-index conditions.
- ASTMODE-generated parser fixtures now assert preserved `INDENT` nodes even
  for simple top-level programs, so structural indentation is covered before
  richer statement-body lowering depends on it.
- The first generated ASTMODE parser fixtures now compare node shapes after
  stripping source line numbers, reducing churn from unrelated source movement
  while keeping statement order and indentation nodes covered.
- ASTMODE-generated parser/codegen now passes the stage-1 self-host source
  emission gate after hardening generated parser-source token recognizers
  against `:IDENT:` text embedded inside `STRING` tokens.
- Generated C `split_text` now treats the escaped delimiter `"\\n"` as a real
  newline delimiter, preserving the generated lexer escape behavior while still
  allowing stage-generated parser/checker programs to split node streams into
  real lines.
- Self-host AST generated-tool tests now compile the generated lexer, parser,
  checker, and C generator once per Go test process and reuse those binaries
  across generated-pipeline cases. This shortens the feedback loop without
  changing the self-host gates.
- Generated parser-source recognizers now use exact token checks for AST helper
  predicates, ASTMODE detection, multi-assignment calls, and `write_file`
  call-statement shapes, keeping stage-generated parser behavior from treating
  token-like string literal text as real syntax.
- Legacy `ASSIGN:INDEX` generated-C lowering now updates an existing string
  target instead of redeclaring it, which keeps repeated-stage self-generation
  compiling when helperized self-host code reassigns loop metadata from array
  lookups.
- The older generated `parse_tokens` output path now has its own exact token
  kind/text helpers, and its first parser-source recognizers for `print`,
  `source = read_file args[0]`, `tokens = lex source`, helper calls, and index
  assignment no longer rely on broad token-substring checks.
- The same older generated `parse_tokens` path now uses exact token checks for
  array `for`, indexed `for`, and identifier `print` recognizer conditions,
  including call-style identifier prints.
- ASTMODE-generated parser now preserves a function-body implicit comparison
  expression such as `char == " "` as `AST_EXPR:binary(...)`, and
  ASTMODE-generated codegen can emit and call the corresponding one-argument
  boolean helper function without widening helper signatures beyond the current
  self-host parser limit.
- ASTMODE-generated parser/codegen now also preserves and executes a
  one-argument implicit boolean helper whose body is a `contains` call such as
  `contains "012", char`, moving closer to generated execution of lexer helper
  functions like `is_digit(char)`.
- ASTMODE-generated parser/codegen can now preserve and execute a composed
  one-argument boolean helper whose body calls other boolean helpers with
  `or` / `and`, covering lexer-helper shapes such as
  `is_lower(char) or is_upper(char)`.
- The composed boolean helper path now also covers the current lexer helper
  family through `is_alpha(char)` and `is_alpha_num(char)`, including a mixed
  `or` chain that combines helper calls with `char == "_"`.
- ASTMODE-generated parser/codegen now also executes the lexer-style
  `is_space(char)` comparison chain with three string comparisons joined by
  `or`, including escaped newline and tab string literals.
- Generated AST parser output now uses a shared comparison-expression helper
  for both single implicit comparison expressions and the lexer-style
  comparison chain, reducing repeated shape-specific AST string assembly.
- Generated AST parser output now also uses the shared call-expression helper
  for parenthesized one-argument helper calls inside boolean helper chains,
  reducing duplicated `call(... ident(...))` AST string assembly.
- Generated AST parser output now also routes bare two-argument `contains`
  helper bodies such as `contains "0123456789", char` through the shared call
  expression helper, reducing another lexer-helper-specific AST string branch.
- Generated AST parser output now uses a shared binary-chain helper for
  three-term boolean chains, so the lexer-style `is_space` comparison chain
  and `is_alpha` helper-call chain share the same AST nesting construction.
- Generated AST parser output now uses a shared boolean-chain token helper for
  both three-comparison chains and mixed call/call/comparison chains, moving the
  boolean helper recognizers closer to a small expression parser.
- Generated AST parser output now also routes single comparisons, bare
  `contains` calls, two-call boolean chains, and three-term boolean chains
  through one `ast_bool_expr_from_tokens` entry point in function-body ASTMODE
  expression recognizers.
- Generated AST parser output now also uses one
  `ast_bool_expr_candidate_at` predicate for those function-body boolean
  expression recognizers, replacing the separate shape-specific parse-loop
  branches with one shared AST expression emission block.
- Generated AST parser output now also routes integer-bound and `len(value)`
  compound `while` condition construction through
  `ast_compound_while_from_tokens`, reducing duplicated parse-loop AST assembly
  while preserving the existing legacy fallback nodes.
- Generated AST parser output now also routes simple integer-bound `while`
  condition construction for `<`, `>=`, `>`, and `<=` through
  `ast_simple_while_from_tokens`, reducing another parse-loop condition
  assembly path while keeping legacy `WHILE_COMPARE_*` fallback nodes.
- Generated AST parser output now also routes simple/index `if` comparison
  condition construction for `==`, `!=`, `>=`, `>`, and `<=` through
  `ast_simple_if_from_tokens`, including `value[i] == "x"` style conditions
  used by array-return helpers.
- Generated AST parser output now also routes single-expression `return`
  statements through `ast_return_from_tokens`, so both `return "x"` and
  `return value` emit `AST_RETURN` in ASTMODE instead of keeping identifier
  returns on the legacy `RETURN:IDENT` path.
- Generated AST parser output now also routes `push` statement value parsing
  through `ast_push_from_tokens`, covering identifier, string, index, and
  one-argument call push values while preserving same-line call detection.
- Generated AST parser output now also routes two-argument `write_file`
  statement call parsing through `ast_call_stmt2_from_tokens` instead of
  formatting the `AST_CALL_STMT2` node directly in the parse loop.
- Generated AST parser output now also routes `for item in items` and
  `for item, index in items` parsing through `ast_for_from_tokens`, emitting
  `AST_FOR` / `AST_FOR_INDEX` directly in ASTMODE.
- Generated AST parser output now also routes `else`, `break`, and `continue`
  through `ast_control_from_tokens`, emitting `AST_ELSE`, `AST_BREAK`, and
  `AST_CONTINUE` directly in ASTMODE.
- Generated AST parser output now also routes `exit` and `panic` through
  `ast_effect_from_tokens`, emitting `AST_EXIT` and `AST_PANIC` directly in
  ASTMODE for identifier exits and string panics.
- Generated AST parser output now also routes simple assignment parsing through
  `ast_assign_from_tokens`, covering string, int, bool, nil, empty-array,
  identifier-alias, and identifier-index assignment shapes in ASTMODE while
  preserving legacy fallback nodes.
- `ast_assign_from_tokens` now also covers one- and two-item string array
  literal assignments, so `["A"]` and `["A", "B"]` lower directly to
  `array1(...)` / `array2(...)` AST assignment expressions in ASTMODE.
- `ast_assign_from_tokens` now also covers parenthesized one-argument call
  assignments such as `clean = trim(message)`, emitting a direct
  `AST_ASSIGN:...:call(...)` node in ASTMODE.
- There are currently no `expected-failing` standalone examples in the manifest;
  remaining `out-of-scope` entries are support fixtures.

This is still not full self-hosting. The self-host compiler still has a
line-oriented node-string parser, subset checker behavior, subset generated-C
lowering, and generated-tool fallback paths. Do not treat the stage-7 fixed
point alone as full self-host completion.

Important current gap:

- The repeated-stage compiler binaries can pass the supported subset, but later
  generated stages still do not behave like full equivalents of the self-host
  sources.
- The next durable AST codegen step is to run the ASTMODE-generated self-host
  compiler binaries against broader real inputs and close behavioral gaps
  without adding source-specific fallbacks.
- The current behavioral blocker is generic lowering for non-trivial function
  bodies such as `lex(source)`: the generated parser can preserve top-level
  function definitions without ingesting their bodies as main-program nodes, but
  nested loops/branches, broader local mutable state, dynamic arrays built from
  local values, and full function-body control flow are not yet emitted as real
  C functions.
- The temporary `call1_arg` expression shape has been normalized back into the
  normal `call` AST path. The next parser/codegen blocker before real
  `lex(source)` behavior is lowering non-trivial function bodies generically.
- Do not continue self-host progress by adding source-specific generated-C
  fallback branches. Use `docs/SELFHOST_AST.md` as the migration target and add
  legacy node-string adapters only as a compatibility layer.
- The strict repeated-stage audit is the focused gate for this gap:

```sh
TYA_STAGE1_SELFHOST_STRICT_REPEATED=1 sh scripts/stage1_selfhost_sources_check.sh
```

## Current Roadmap

- [ ] Finish self-hosting `current`
  - [ ] Introduce structured self-host AST representation `current`
    - [ ] Define a stable representation for program, block, statement, and
      expression nodes in self-host Tya. See `docs/SELFHOST_AST.md`.
    - [ ] Preserve source line and indentation information without encoding
      node meaning in colon-delimited strings.
    - [ ] Add adapters so existing node-string checker/codegen paths can keep
      running while AST-backed nodes are introduced.
    - [ ] Add focused fixtures that compare old node-string behavior with the
      new AST representation for currently supported examples.
  - [ ] Replace expression parsing with a real precedence parser
    - [ ] Stop adding new fixed-shape call node variants such as
      `CALL1_INDEX`, `CALL1_EXPR`, or `CALL1_MEMBER`; migrate toward a generic
      call expression adapter shared by parser, checker, and codegen.
    - [ ] Add generic expression adapter helpers before widening call argument
      support: one helper to render expression ASTs to legacy compatibility
      nodes, one helper to check expression ASTs, and one helper to lower
      expression ASTs.
      - [x] Start the codegen-side expression-lowering adapter with
        `ast_string_expr_c` for string-like arguments and values.
      - [x] Add small reusable predicates for array-return push validation so
        the collector path is less duplicated before deeper lowering work.
      - [x] Reuse derived loop metadata across array-return helper output
        instead of re-scanning the collector arrays for final increments.
      - [x] Add `emit_dynarray_push` for array-return helper dynamic push output
        while staying within the current self-host function arity limit.
      - [x] Route main-body `AST_PUSH` and legacy `PUSH` dynamic-array output
        through the same helper.
      - [x] Route fixed-array `AST_PUSH` and legacy `PUSH` output through
        `emit_single_array_push`.
      - [x] Route AST `array0` initialization through `emit_empty_dynarray`.
      - [x] Route AST `array1` initialization through `emit_array1_init`.
      - [x] Start AST `array2` initialization helper extraction without
        exceeding the current self-host function arity limit.
      - [x] Start AST `array3` initialization helper extraction with the same
        arity-safe pattern.
      - [x] Route remaining AST `array2` and `array3` element declarations
        through `emit_array_item_init`.
      - [x] Route array-return helper local result-array initialization through
        `emit_empty_dynarray`.
      - [x] Route array-return helper string local and alias declarations
        through `emit_string_local`.
      - [x] Route array-return helper string accumulation assignment through
        `emit_string_concat_assign`.
      - [x] Route array-return helper loop continue guards through
        `emit_continue_guard`.
      - [x] Route array-return helper loop counter updates through
        `emit_loop_increment`.
      - [x] Route array-return helper loop start through `emit_loop_start`.
      - [x] Preserve array-return helper loop statement order through an
        AST-ordered statement record.
      - [x] Route array-return helper function tail through
        `emit_array_return`.
      - [x] Route array-return helper function header through
        `emit_array_func_header`.
      - [x] Route simple string helper function header and return output
        through `emit_string_func_header` and `emit_string_return`.
      - [x] Route simple string helper conditional returns through
        `emit_string_case_return`.
    - [ ] Replace stage-source line-number fixtures with shape-focused checks
      where possible so AST parser work is not dominated by unrelated line
      movement in `selfhost/parser.tya` or `selfhost/checker.tya`.
    - [ ] Harden self-host parsing/codegen of generated C string-emission
      lines before routing more generated parser branches through shared
      helpers; otherwise apparently small helperization can break the
      `selfhost/codegen_c.tya` repeated-stage compile path.
      - [x] Add exact generated-parser token kind/text helpers and cover
        string literals containing token-like text such as `:IDENT:if:`.
      - [x] Extend exact generated-parser token checks to basic `print`
        recognizers and cover string literals containing `:IDENT:print:`.
      - [x] Extend exact generated-parser token checks to supported `return`
        recognizers and cover string literals containing `:IDENT:return:`.
      - [x] Extend exact generated-parser token checks to supported `while`
        recognizers and cover string literals containing `:IDENT:while:`.
      - [x] Extend exact generated-parser token checks to comparison
        assignment recognizers and cover string literals containing
        assignment-shaped `:SYMBOL:=:` / comparison token text.
      - [x] Extend exact generated-parser token checks to parenthesized
        comparison assignment recognizers, including exact `>=` versus `>`
        operator classification.
      - [x] Extend exact generated-parser token checks to parenthesized
        arithmetic assignment recognizers for `(left + right)` and
        `(left + right) * factor` shapes.
      - [x] Extend exact generated-parser token checks to non-parenthesized
        arithmetic assignment recognizers for `left + right` and
        `left - right` shapes.
      - [x] Extend exact generated-parser token checks to index/call-adjacent
        assignment recognizers for `ident + collection[index]`,
        `object[index]`, and `func(arg)` shapes.
      - [x] Extend exact generated-parser token checks to boolean, primitive
        literal, empty-array, and plain identifier assignment recognizers.
      - [x] Extend exact generated-parser token checks to string/int array
        literal and multi-argument call assignment recognizers.
      - [x] Extend exact generated-parser token checks to function definition
        and `CALL1_CALL0_INDEX` assignment recognizers.
      - [x] Extend exact generated-parser token checks to generated `print`
        call recognizers from one-argument through three-argument forms.
      - [x] Extend exact generated-parser token checks to generated
        control-flow recognizers for `for`, boolean `while`, `break`,
        `continue`, `exit`, and `panic`.
      - [x] Extend exact generated-parser token checks to AST helper
        predicates, ASTMODE detection, multi-assignment calls, and `write_file`
        call-statement recognizers.
      - [x] Fix legacy `ASSIGN:INDEX` generated-C lowering so existing string
        targets are assigned instead of redeclared during repeated self-host
        stages.
      - [x] Add exact token helpers to the older generated `parse_tokens`
        output path and use them for its first parser-source recognizers.
      - [x] Extend the older generated `parse_tokens` exact checks to array
        `for`, indexed `for`, and identifier `print` recognizer conditions.
    - [ ] After every few self-host slices, run a strategy review that checks
      whether work is looping, adding special cases, or only updating brittle
      fixtures; record any required pivot in this roadmap or
      `docs/SELFHOST_AST.md` before continuing.
    - [ ] Implement precedence-based parsing for literals, identifiers,
      grouping, unary operations, binary operations, calls, member access, and
      indexing.
    - [ ] Use expression ASTs in `if`, `while`, `print`, `return`, assignment,
      and multi-assignment positions.
    - [ ] Support calls with arbitrary expression arguments instead of
      fixed-arity node strings such as `PRINT_CALL1`, `PRINT_CALL2`, and
      `IF_CALL_EQ_AND_CALL_NE`.
    - [ ] Support function literals in expression positions without
      source-specific fallback paths.
    - [ ] Remove expression-specific parser shortcuts only after equivalent AST
      paths are covered by tests and the bootstrap gate.
  - [ ] Migrate statement and definition parsing to AST nodes
    - [ ] Convert assignment, call statement, `print`, `return`, `panic`, and
      `exit` parsing from line-oriented node strings to statement AST nodes.
    - [ ] Convert `if` / `else`, `while`, `break`, and `continue` parsing to
      block-aware AST nodes.
    - [ ] Convert array and object/dictionary `for` forms while preserving
      value and index/key bindings distinctly.
    - [ ] Convert function, method, object, class, module, import, constant,
      implicit last-expression return, and `try` forms.
      - [x] Render two-value `return` statements as `AST_RETURN2` and check
        both returned expression shapes.
      - [x] Render two-target assignments as `AST_MULTI_ASSIGN2` and check the
        targets plus assigned expression shape.
      - [x] Render two-argument statement calls as `AST_CALL_STMT2` and execute
        the `write_file` AST codegen path in the generated pipeline.
      - [x] Execute `AST_DELETE` object-member deletion through the generated
        AST codegen path.
      - [x] Execute `AST_EXIT` and `AST_PANIC` through the generated AST
        codegen path.
      - [x] Execute `AST_BREAK` and `AST_CONTINUE` through the generated AST
        codegen path.
      - [x] Execute `AST_FOR_INDEX` through the generated AST codegen path.
      - [x] Execute integer-comparison `AST_IF` conditions through the
        generated AST codegen path.
      - [x] Execute integer-step `AST_ASSIGN` updates through the generated
        AST codegen path.
      - [x] Execute primitive and simple binary `AST_PRINT` expressions through
        the generated AST codegen path.
      - [x] Execute shallow integer binary `AST_ASSIGN` expressions through
        the generated AST codegen path.
      - [x] Execute integer `ident op ident` binary `AST_ASSIGN` expressions
        through the generated AST codegen path.
      - [x] Execute string `ident + ident` binary `AST_ASSIGN` expressions
        through the generated AST codegen path.
      - [x] Execute string literal mixed concatenation `AST_ASSIGN`
        expressions through the generated AST codegen path.
      - [x] Execute nested integer binary `AST_ASSIGN` expressions through
        the generated AST codegen path.
      - [x] Execute boolean, nil, and identifier-copy `AST_ASSIGN` nodes
        through the generated AST codegen path.
      - [x] Execute nested integer binary `AST_PRINT` expressions through the
        generated AST codegen path.
      - [x] Execute string `ident + ident` binary `AST_PRINT` expressions
        through the generated AST codegen path.
      - [x] Execute string literal mixed concatenation `AST_PRINT` expressions
        through the generated AST codegen path.
      - [x] Share AST binary comparison-operator classification in the
        generated AST codegen path.
      - [x] Share AST string concatenation helper-dependency detection in the
        generated AST codegen path.
      - [x] Share generated AST binary expression known-type checks across
        assign and print paths.
      - [x] Execute `ident + call(trim ident(...))` string concatenation
        through generated AST assign and print paths.
      - [x] Execute `ident + call(to_string ident(...))` string concatenation
        through generated AST assign and print paths.
      - [x] Share generated AST string-concat assignment emission across
        declaration and update paths.
      - [x] Share generated AST string-concat print emission.
      - [x] Execute simple AST multi-return calls through `AST_RETURN2` helper
        registration and `AST_MULTI_ASSIGN2` call lowering.
      - [x] Run `examples/multiple_return.tya` through the AST parser/checker
        and generated C path.
      - [x] Lower AST boolean `and` / `or` expressions to valid C, including
        nested string comparisons.
      - [x] Lower AST integer array literals and `for` iteration over
        `INTARRAY` values.
      - [x] Centralize self-host checker builtin registration with
        `remember_builtin_names` and add missing standard builtin names.
      - [x] Add the first generated parser ASTMODE output path for
        `print "..."`, with generated codegen executing the resulting
        `AST_PRINT:string(...)` statement inside the existing control-flow
        compatibility layer.
      - [x] Extend generated parser ASTMODE output to simple string assignment
        and identifier print nodes, keeping the generated pipeline green while
        reducing the legacy node surface for straight-line statements.
      - [x] Extend generated parser ASTMODE output to simple `if x == "y"` and
        `else` control nodes, with generated codegen executing that AST control
        shape in the existing compatibility layer.
      - [x] Extend generated parser ASTMODE output to a small integer
        `while i < n` loop and execute it through generated codegen.
      - [x] Extend generated parser ASTMODE output to a small string-array
        `for item in items` loop and execute it through generated codegen.
      - [x] Extend generated parser ASTMODE output to a simple one-argument
        function call and execute it through generated codegen.
      - [x] Preserve a simple function-body return in generated parser ASTMODE
        output without changing legacy generated parser body skipping.
      - [x] Preserve a simple function-body implicit comparison expression as
        `AST_EXPR:binary(...)` in generated parser ASTMODE output.
      - [x] Preserve small array-return helper body statements in generated
        parser ASTMODE output: empty array assignment, local identifier alias,
        identifier/string `push`, and final array return.
      - [x] Execute a small ASTMODE-generated array-return helper end-to-end
        through generated parser, checker, codegen, C compilation, and runtime.
      - [x] Execute a looping ASTMODE-generated array-return helper with a
        local counter, `while`, loop-body `push`, and increment.
      - [x] Execute a looping ASTMODE-generated array-return helper with a
        narrow `if i == n` continue guard.
      - [x] Route ASTMODE-generated array-return helper dynamic pushes through
        a shared generated C `push_text` helper.
      - [x] Preserve ASTMODE-generated array-return helper loop statement
        order for `push` and string accumulation statements.
      - [x] Execute alias-backed string accumulation inside an
        ASTMODE-generated array-return helper loop.
      - [x] Preserve `ident + int` assignments in generated parser ASTMODE and
        execute a named-counter, non-unit-step array-return helper loop.
      - [x] Execute the same named-counter, non-unit-step loop metadata through
        an ASTMODE-generated `if ... continue` guard.
      - [x] Execute conditional `push` statements inside an ASTMODE-generated
        array-return helper loop without misclassifying the `if` as a
        continue guard.
      - [x] Execute conditional string accumulation with both identifier and
        string-literal right operands in an ASTMODE-generated array-return
        helper loop.
      - [x] Execute helper-local conditional `push` statements for `!=` and
        `>` integer comparisons in the ASTMODE-generated pipeline.
      - [x] Execute helper-local conditional `push` statements for `>=` and
        `<=` integer comparisons in the ASTMODE-generated pipeline.
      - [x] Execute multiple statements under one helper-local `if` body in
        the ASTMODE-generated pipeline.
      - [x] Execute helper-local `if` / `else` dynamic-array `push` branches in
        the ASTMODE-generated pipeline.
      - [x] Execute multiple statements in both helper-local `if` and `else`
        branches in the ASTMODE-generated pipeline.
      - [x] Preserve ASTMODE-generated post-loop array-return helper statements
        using ASTMODE `INDENT` nodes.
      - [x] Execute post-loop `if` / `else` array-return helper branches after
        a preceding generated ASTMODE `while`.
      - [x] Preserve and execute non-`<` generated ASTMODE `while` conditions
        in array-return helpers, starting with a `<=` loop parity test.
      - [x] Preserve and execute non-`==` generated ASTMODE `if ... continue`
        guard conditions in array-return helpers, starting with a `!=` guard.
      - [x] Preserve and execute decrementing generated ASTMODE loop-counter
        updates in array-return helpers, starting with `count = count - 1`.
      - [x] Preserve and execute generated ASTMODE call-expression `push` values
        in array-return helpers, starting with `push items, trim value`.
      - [x] Preserve and execute generated ASTMODE call-expression string
        accumulation values in array-return helpers, starting with
        `text = text + trim value`.
      - [x] Execute generated ASTMODE call-expression string accumulation values
        under helper-local `if` conditions.
      - [x] Execute generated ASTMODE call-expression `push` values under
        post-loop `if` / `else` branches.
      - [x] Execute generated ASTMODE `to_string` call-expression `push` and
        string accumulation values in array-return helpers.
      - [x] Execute generated ASTMODE string index assignment values in
        array-return helpers.
      - [x] Execute generated ASTMODE direct string index `push` values in
        array-return helpers.
      - [x] Execute generated ASTMODE string index comparisons in helper-local
        `if` conditions, including a false-condition guard.
      - [x] Execute generated ASTMODE compound `while` conditions that combine
        integer loop bounds with string index comparisons.
      - [x] Execute generated ASTMODE compound `while` conditions that combine
        `len(value)` loop bounds with string index comparisons.
      - [x] Execute generated ASTMODE string index values in helper-local
        string accumulation assignments.
      - [x] Execute a one-argument implicit boolean helper function in the
        ASTMODE-generated parser/checker/codegen pipeline.
      - [x] Execute a one-argument implicit `contains` boolean helper function
        in the ASTMODE-generated parser/checker/codegen pipeline.
      - [x] Execute a composed one-argument implicit boolean helper function
        whose body combines other boolean helper calls with `or` / `and`.
      - [x] Execute the lexer-style `is_alpha` / `is_alpha_num` boolean helper
        chain in the ASTMODE-generated parser/checker/codegen pipeline.
      - [x] Execute the lexer-style `is_space` comparison chain in the
        ASTMODE-generated parser/checker/codegen pipeline.
      - [x] Share generated parser comparison-expression AST construction
        between single comparisons and comparison chains.
      - [x] Share generated parser call-expression AST construction for
        parenthesized helper calls in boolean chains.
      - [x] Share generated parser call-expression AST construction for
        bare `contains string, ident` helper bodies.
      - [x] Share generated parser binary-chain AST construction for
        three-term boolean chains.
      - [x] Share generated parser boolean-chain token parsing for
        three-term comparison and mixed call/comparison chains.
      - [x] Route generated parser function-body boolean expression recognizers
        through one `ast_bool_expr_from_tokens` entry point.
      - [x] Route generated parser function-body boolean expression candidate
        checks through one `ast_bool_expr_candidate_at` predicate.
      - [x] Add a shared generated parser AST atom helper for simple binary
        assignment operands.
      - [x] Add a shared generated parser AST index helper for binary
        assignment operands.
      - [x] Add a shared generated parser AST call helper for binary
        assignment operands.
      - [x] Add a shared generated parser expression helper that dispatches to
        atom, index, and call operand renderers.
      - [x] Reuse the generated parser expression helper for direct index
        `push` operands.
      - [x] Reuse the generated parser expression helper for string index
        comparison operands in `if` and compound `while`.
      - [x] Add a shared generated parser compound-while condition helper for
        integer and `len(value)` loop bounds.
      - [x] Route generated parser compound-while token parsing for integer
        and `len(value)` loop bounds through one helper.
      - [x] Add a shared generated parser binary condition helper for
        helper-local `if` conditions.
      - [x] Reuse the generated parser binary condition helper for simple
        integer-bound `while` comparisons.
      - [x] Route generated parser simple integer-bound `while` token parsing
        through one helper.
      - [x] Route generated parser simple/index `if` comparison token parsing
        through one helper.
      - [x] Route generated parser single-expression `return` token parsing
        through one helper.
      - [x] Route generated parser `push` token parsing through one helper.
      - [x] Route generated parser `write_file` call statement token parsing
        through one helper.
      - [x] Route generated parser `for` token parsing through one helper.
      - [x] Route generated parser `else` / `break` / `continue` token parsing
        through one helper.
      - [x] Route generated parser `exit` / `panic` token parsing through one
        helper.
      - [x] Route generated parser simple assignment token parsing through one
        helper.
      - [x] Extend generated parser assignment helper to string array literal
        assignments.
      - [x] Extend generated parser assignment helper to parenthesized
        one-argument call assignments.
      - [x] Update ASTMODE-generated parser fixtures to require preserved
        top-level `INDENT` nodes.
      - [x] Start replacing generated ASTMODE parser line-number fixtures with
        shape-focused checks.
      - [x] Harden generated parser-source recognizers so they do not treat
        `:IDENT:` substrings embedded in string literals as syntax tokens.
      - [x] Keep generated newline escape preservation compatible with
        line-oriented self-host stages by normalizing `"\\n"` delimiters in
        generated `split_text`.
      - [x] Cache generated self-host AST test tools within one Go test
        process to keep the expanding generated-pipeline suite usable.
    - [ ] Remove node-string parsing for a statement family only after checker,
      codegen, and repeated-stage gates use the AST path.
  - [ ] Bring the self-host checker to Go checker parity
    - [ ] Model lexical scopes, block/function boundaries, reassignment, and
      shadowing consistently with `internal/checker`.
    - [ ] Carry index/key loop bindings distinctly instead of collapsing
      supported `for` forms to one value binding.
    - [ ] Enforce constants, imports/module public-binding rules, object member
      names, method receiver rules, duplicate declarations, optional unused
      checks, break/continue/return placement, and naming diagnostics with
      source line parity.
    - [ ] Check all expression forms and builtin arities rather than only the
      current node-string subset.
  - [ ] Replace prototype C lowering with AST-backed executable code generation
    - [ ] Lower real functions, closures/function values, methods with `@`,
      object and array mutation, indexing, imports/prelude loading, error
      values, `try`, multi-return values, interpolation, unary operations, and
      all standard-library calls documented in `docs/STDLIB.md`.
    - [ ] Generate C against the runtime ABI used by the Go emitter, or document
      and converge any intentionally smaller ABI.
    - [ ] Lower self-host lexer, parser, checker, and codegen node streams as
      normal programs rather than through generated-tool operation shortcuts.
    - [ ] Remove generated-C fallback stubs and example-specific recognizers
      only after an AST-backed generic path handles the same behavior.
  - [ ] Expand repeated-stage audit toward full language evidence
    - [ ] Keep `sh scripts/selfhost_bootstrap_check.sh` passing.
    - [ ] Keep
      `TYA_STAGE1_SELFHOST_STRICT_REPEATED=1 sh scripts/stage1_selfhost_sources_check.sh`
      passing.
    - [ ] Expand strict repeated-stage coverage from the current
      print/assignment/arithmetic/control-flow/array/object/string-builtin/bool
      subset toward every supported manifest language feature.
    - [ ] Require every new supported example to have a self-host parity
      classification in `scripts/selfhost_examples_manifest.txt`.
    - [ ] Add negative parser/checker fixtures for unsupported or invalid
      language features as they become supported.
  - [ ] Remove remaining generated-tool fallback behavior
    - [ ] Replace the generated lexer fallback keyed by `CALL1:lex` with
      generic lowering of the self-host lexer program.
    - [ ] Replace the generated parser fallback keyed by `CALL1:parse` or the
      coarse `FOR:node:nodes` parser-source stream with generic parser lowering.
    - [ ] Replace generated checker operation shortcuts keyed by `CALL1:check`
      or the coarse checker source stream with generic checker lowering.
    - [ ] Replace generated codegen operation shortcuts keyed by
      `PRINT_CALL1:emit_c:nodes` with generic codegen lowering.
    - [ ] Remove remaining narrow example lowering paths after the AST-backed
      parser/checker/codegen can handle the same source generally.
  - [ ] Meet full self-hosting completion criteria
    - [ ] Reach a stable fixed point without source-specific fallback behavior.
    - [ ] Match Go lexer/parser/checker/codegen behavior for the full language.
    - [ ] Make every runnable non-fixture example a generated-tool parity
      target.
    - [ ] Support the documented standard library surface in generated code.

- [ ] Finish class, module, dictionary, and set semantics
  - [ ] Separate dictionary and set semantics
    - [ ] Rename the current object-literal concept to dictionary in the AST,
      checker, interpreter, C emitter, docs, examples, and diagnostics while
      preserving existing behavior during the transition.
    - [ ] Implement inline and indented dictionary literals, bracket access for
      dictionaries, empty `{}` as an empty dictionary, and diagnostics for mixed
      dictionary/set entries.
    - [ ] Add set literals and the empty-set constructor, including
      interpreter, builtins, C runtime/codegen support, and collection docs.
    - [ ] Separate dictionary access from object/member access so dictionaries,
      sets, and arrays reject `.` and dictionaries use `[]`.
  - [ ] Finish class semantics
    - [ ] Add `self`.
    - [ ] Add `super`.
    - [ ] Add inheritance.
    - [ ] Add override arity checks.
    - [ ] Add interface checking.
    - [ ] Add generated-C parity.
  - [ ] Finish module/import semantics
    - [ ] Add import aliases.
    - [ ] Add default imports.
    - [ ] Add conflict checks.
    - [ ] Add entry-file rules.
    - [ ] Add generated-C parity.
  - [ ] Update self-host example classifications
    - [ ] Promote newly supported class/module/dictionary/set examples in
      `scripts/selfhost_examples_manifest.txt`.

- [ ] Harden the Go interpreter
  - [ ] Improve parser and checker diagnostics
    - [ ] Add parser and checker source spans to more AST nodes.
  - [ ] Add stricter checker modes
    - [ ] Wire optional unused variable and unused argument checks into a
      stricter CLI mode if desired.
    - [ ] Add variable shadowing checks beyond the current scope model.
    - [ ] Add top-level executable-code checks for non-`main.tya` files after
      module loading semantics are finalized.
    - [ ] Improve checker scopes for reassignment versus duplicate definition.

- [ ] Harden generated C output
  - [ ] Close remaining generated-C parity gaps
    - [ ] Emit C for remaining edge cases in the complete standard library
      surface.
    - [ ] Finish class/module generated-C lowering as those language features
      stabilize.
  - [ ] Improve generated-C diagnostics
    - [ ] Add source maps or generated-line diagnostics.

- [ ] Harden the C runtime
  - [ ] Add memory management
    - [ ] Add mark-and-sweep GC.
  - [ ] Broaden runtime test coverage
    - [ ] Add tests for more nested object paths.
    - [ ] Add tests for more error paths.

- [ ] Stabilize the standard library
  - [ ] Move implementation into Tya where practical
    - [ ] Move more builtins into Tya once imports/modules are complete.
  - [ ] Stabilize API behavior
    - [ ] Stabilize error conventions after `try` sees more use.
    - [ ] Version the standard library API.

## Current Limits

1. The self-host parser still recognizes many source forms by line-oriented
   token patterns and emits colon-delimited node strings.
1. The self-host checker still checks the current supported node subset rather
   than the full Go checker semantics.
1. The self-host/generated C path still contains generated-tool fallback
   behavior and narrow example-shape lowering.
1. Repeated-stage fixed point is useful evidence, but it is not full
   self-hosting evidence while later generated compiler binaries are still
   minimal or fallback-driven.
1. Curly and indented literals are still parsed as object literals, not separate
   dictionaries.
1. Empty `{}` is an empty object. There is no set literal or `set()` builtin.
1. The `.` operator reads and writes members on the current object value type.
1. `self`, `super`, inheritance, and interface checking remain unimplemented.
1. Import aliases are not implemented.
1. Entry files execute top-level statements directly.
1. Dedicated class lowering in the C emitter remains incomplete.

## Verification Reference

1. Lightweight self-host slice gate. Run this after each small implementation
   slice. Use this gate instead of the medium/heavy gates when the change is
   contained to one lowering/emission helper and does not alter stage
   fixed-point behavior.
   1. Focused `go test` for the touched behavior.
   1. `sh scripts/selfhost_check.sh`
1. Medium self-host slice gate. Run this after a coherent group of related
   slices, or when parser/checker/codegen boundaries change.
   1. `go test ./... -count=1`
   1. `sh scripts/stage1_selfhost_sources_check.sh`
1. Heavy self-host fixed-point gate. Run this after every few slices, before
   declaring a milestone complete, or when generated-stage/fixed-point behavior
   may have changed.
   1. `sh scripts/selfhost_bootstrap_check.sh`
   1. `TYA_STAGE1_SELFHOST_STRICT_REPEATED=1 sh scripts/stage1_selfhost_sources_check.sh`
1. Generated-C verification.
   1. `go test ./... -count=1`
   1. `sh scripts/go_emit_examples_check.sh`
   1. `sh scripts/go_emit_selfhost_compile_check.sh`
   1. `sh scripts/go_emit_selfhost_run_check.sh`
1. Parallel verification rule. When checks do not write the same generated
   files or depend on each other's output, run them concurrently and report the
   slowest failing gate first.

## References

1. Self-hosting references.
   1. `SELFHOST_WORK.md`
   1. `docs/SELFHOST.md`
   1. `selfhost/README.md`
   1. `selfhost/lexer.tya`
   1. `selfhost/parser.tya`
   1. `selfhost/checker.tya`
   1. `selfhost/codegen_c.tya`
   1. `scripts/selfhost_examples_manifest.txt`
   1. `scripts/stage1_selfhost_sources_check.sh`
   1. `scripts/selfhost_bootstrap_check.sh`
1. Language and runtime references.
   1. `docs/CLASS_MODULE_DESIGN.md`
   1. `docs/NAMING.md`
   1. `docs/REFERENCE.md`
   1. `docs/GUIDE.md`
   1. `docs/STDLIB.md`
   1. `stdlib/prelude.tya`
   1. `runtime/tya_runtime.c`
   1. `runtime/tya_runtime.h`

## Non-Goals For Now

1. LLVM
1. ANTLR
1. Tree-sitter
1. Package manager
1. Async
1. Macros
1. Exceptions
