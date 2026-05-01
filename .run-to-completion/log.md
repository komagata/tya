# Run To Completion Log

## 2026-04-29 18:40:04 JST

- Started from argument-free run-to-completion invocation.
- Inferred goal from `SELFHOST_WORK.md`: complete the first unchecked
  self-hosting queue item.
- Next milestone: compare stage-1 and stage-2 generated C for deterministic
  output.

## 2026-04-29 18:46:30 JST

- Added repeated stage-2 codegen determinism comparisons to
  `scripts/stage1_selfhost_sources_check.sh`.
- Updated `tests/selfhost_test.go` expected output.
- Focused verification passed: `sh scripts/stage1_selfhost_sources_check.sh`.

## 2026-04-29 18:48:28 JST

- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-29 18:49:10 JST

- Committed deterministic stage-2 codegen checkpoint as `ad0ab90`.
- Next queue cleanup: mark parent tasks complete where all children are checked.

## 2026-04-29 18:51:00 JST

- Resumed after `go`.
- Confirmed worktree was clean and `SELFHOST_WORK.md` had no unchecked items.
- Stop condition reached: no queued self-hosting work remains.

## 2026-04-29 18:59:38 JST

- User gave new goal in Japanese: keep iterating development until self-hosting
  works.
- Reopened run-to-completion state for the broader self-host completion goal.
- Seeded `SELFHOST_WORK.md` with a new high-level completion queue based on
  `ROADMAP.md`.

## 2026-04-29 19:00:20 JST

- Promoted completed lexer parity TODOs in `ROADMAP.md`.
- Marked the matching `SELFHOST_WORK.md` queue item done.

## 2026-04-29 19:01:15 JST

- Added self-host parser support for parenthesized one-argument assignment calls
  by lowering `name = call(arg)` to existing `CALL1` nodes.
- Updated `TestSelfhostParserMatchesGoParserSubset` and Go AST summarization.
- Focused verification passed: `go test ./tests -run
  TestSelfhostParserMatchesGoParserSubset -count=1`.

## 2026-04-29 19:05:24 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 13:44:15 JST

- Committed less-than while bootstrap slice as `ef350af`.
- Adjusted stage2 parser helper to preserve only block-relevant `INDENT` nodes
  and stage2 codegen helper to close blocks on `INDENT:0`.
- Added `examples/while.tya` to the stage2 bootstrap pipeline and matched
  expected output `10` and `11`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.

## 2026-04-30 13:53:41 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 13:29:10 JST

- Committed while false bootstrap slice as `d1f3a51`.
- Added a stage-2 pipeline fixture for `while i < 2`, integer reassignment, and
  `break`.
- Extended generated-C stage2 parser/codegen helpers to emit and lower
  `WHILE_COMPARE_LT`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.

## 2026-04-30 13:38:27 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 13:18:10 JST

- Committed integer addition reassignment slice as `22582e6`.
- Added a stage-2 pipeline fixture for `while false`, `break`, and a following
  print.
- Extended generated-C stage2 parser/codegen helpers to emit and lower
  `WHILE:BOOL:false` and `BREAK`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.

## 2026-04-30 13:27:12 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 13:05:52 JST

- Committed less-than comparison slice as `0441899`.
- Added a stage-2 pipeline fixture for `sum = 0; sum = sum + 1; print sum`.
- Updated generated-C stage2 codegen helper to emit assignment for known
  `INT_ADD` targets instead of redeclaring locals.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.

## 2026-04-30 13:14:38 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 11:51:05 JST

- Committed string example bootstrap slice as `2b73ee5`.
- Added a stage-2 pipeline fixture for `less = left < right` and `print less`.
- Extended the generated-C parser/codegen helpers to emit and lower
  `COMPARE_LT`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.

## 2026-04-30 12:00:09 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 11:39:35 JST

- Committed replace string-literal bootstrap slice as `7de8924`.
- Added stage2 parser/codegen support for `PRINT_INDEX:STRING` from
  `print "tya"[1]`.
- Added `examples/string.tya` to the stage2 bootstrap pipeline and matched the
  interpreter output:
  `hello-tya`, `hello,Tya`, three `true` lines, `6`, `2`,
  `quote: "tya"`, and `y`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.

## 2026-04-30 11:48:13 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 11:25:03 JST

- Committed byte/char length bootstrap slice as `50b96f1`.
- Updated stage2 `PRINT_CALL3:replace` support to allow a typed string literal
  replacement argument while preserving the existing identifier replacement
  form.
- Focused verification passed: `go test ./tests -run
  'Test(SelfhostCheckerRejectsUndefinedPrintCallNames|Stage1SelfhostSourcesEmitC|SelfhostCodegenEmitsSimpleReturnFunctions)'
  -count=1`.

## 2026-04-30 11:33:31 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 11:11:35 JST

- Committed string split/join bootstrap slice as `59166c1`.
- Added a stage-2 pipeline fixture for `print byteLen "ちゃ"` and
  `print charLen "ちゃ"`.
- Extended the self-host checker builtins with `byteLen` and `charLen`.
- Extended generated-C parser/codegen helpers to handle one-argument print calls
  with string literal arguments and UTF-8 character counting.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.

## 2026-04-30 11:19:49 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 10:57:52 JST

- Committed printed string colon preservation slice as `d4c2ac8`.
- Added a stage-2 pipeline fixture for `parts = split text, ","` and
  `print join parts, "-"`.
- Extended the generated-C parser helper to emit `ASSIGN:...:CALL2:split` and
  the generated-C codegen helper to emit target `split_text` and `join_text`
  helpers.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.

## 2026-04-30 11:05:22 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 10:48:02 JST

- Committed escaped quote bootstrap slice as `2ad21b9`.
- Added stage2 generated-C codegen helper `node_tail_field` and used it for
  `PRINT:STRING` values so colon characters are preserved.
- Restored the exact `print "quote: \"tya\""` stage2 fixture.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.

## 2026-04-30 10:54:29 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 10:40:29 JST

- Committed string replace bootstrap slice as `9b7cf10`.
- Added a stage-2 pipeline fixture for `print "quote \"tya\""`.
- Tried the exact `quote: \"tya\"` shape first; it exposed the existing colon
  delimiter limitation in node fields, so this slice now isolates escaped quote
  handling and leaves colon field escaping for a future slice.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.

## 2026-04-30 10:46:50 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 10:31:59 JST

- Committed string prefix/suffix bootstrap slice as `69a2956`.
- Added a stage-2 pipeline fixture for `print replace text, "ell", replacement`.
- Extended the generated-C parser helper to emit `PRINT_CALL3:replace` and the
  generated-C codegen helper to emit target `dup_text` and `replace_text`
  helpers.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.

## 2026-04-30 10:38:20 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 10:23:21 JST

- Committed string contains bootstrap slice as `c9dc175`.
- Added a stage-2 pipeline fixture for `print startsWith text, "he"` and
  `print endsWith text, "lo"`.
- Extended the generated-C codegen helper to emit target `starts_with_text` and
  `ends_with_text` helpers.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.

## 2026-04-30 10:29:11 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 10:15:25 JST

- Committed string trim bootstrap slice as `a6958b6`.
- Added a stage-2 pipeline fixture for `text = "hello"; print contains text,
  "ell"`.
- Extended the generated-C parser helper to emit `PRINT_CALL2:contains` and the
  generated-C codegen helper to emit a target `contains_text` helper.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.

## 2026-04-30 10:21:30 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 10:04:12 JST

- Committed string length bootstrap slice as `a722c1a`.
- Added a stage-2 pipeline fixture for `text = "  hello  "; trimmed = trim text;
  print trimmed`.
- Extended the generated-C parser helper to emit `ASSIGN:name:CALL1:trim:arg`
  and the generated-C codegen helper to emit a target `trim_text` helper.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.

## 2026-04-30 10:13:25 JST

- Fixed a Tya string interpolation issue caused by same-line generated C braces
  in the trim helper pre-scan.
- Focused verification passed: `go test ./tests -run
  'Test(SelfhostCodegenEmitsSimpleReturnFunctions|Stage1SelfhostSourcesEmitC)'
  -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-29 19:06:43 JST

- Committed parenthesized one-argument call parser slice as `69e0052`.
- Added self-host parser support for parenthesized two-argument assignment calls
  by lowering `name = call(arg, arg2)` to existing `CALL2` nodes.
- Extended parser subset test with `split(message, "\\n")`.
- Focused verification passed: `go test ./tests -run
  TestSelfhostParserMatchesGoParserSubset -count=1`.

## 2026-04-29 19:11:11 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-29 19:12:40 JST

- Committed parenthesized two-argument call parser slice as `701ff4c`.
- Added self-host parser support for parenthesized three-argument assignment
  calls by lowering `name = call(arg, arg2, arg3)` to existing `CALL3` nodes.
- Extended parser subset test with `replace(message, "T", message)`.
- Focused verification passed: `go test ./tests -run
  TestSelfhostParserMatchesGoParserSubset -count=1`.

## 2026-04-29 19:17:11 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-29 19:18:41 JST

- Committed parenthesized three-argument call parser slice as `3eccd91`.
- Added `replace` to the self-host checker builtin scope.
- Added checker coverage allowing `CALL3:replace` while preserving undefined
  `CALL3` diagnostics.
- Focused verification passed: `go test ./tests -run
  'TestSelfhostChecker(AllowsReplaceBuiltin|RejectsUndefinedThreeArgCallNames)'
  -count=1`.

## 2026-04-29 19:23:35 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-29 19:24:48 JST

- Committed checker builtin slice as `4dcc2e5`.
- Added generated-C `replace_text` helper and lowered `CALL3:replace` to it.
- Focused verification passed: `go test ./tests -run
  TestSelfhostCodegenEmitsSimpleReturnFunctions -count=1`.

## 2026-04-29 19:29:08 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-29 21:24:37 JST

- Committed `replace` codegen slice as `49a5c75`.
- Added `PRINT_CALL3` support for `print replace text, "old", new` across
  parser, checker, and self-host C codegen.
- Focused verification passed: `go test ./tests -run
  'TestSelfhost(ParserMatchesGoParserSubset|CheckerAllowsPrintReplaceBuiltin|CodegenEmitsSimpleReturnFunctions)'
  -count=1`.

## 2026-04-29 21:29:07 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-29 21:30:58 JST

- Committed `print replace` slice as `76eae81`.
- Added `PRINT_CALL2` support for `print contains text, "needle"` across
  parser, checker, and self-host C codegen.
- Focused verification passed: `go test ./tests -run
  'TestSelfhost(ParserMatchesGoParserSubset|CheckerAllowsPrintContainsBuiltin|CodegenEmitsSimpleReturnFunctions)'
  -count=1`.

## 2026-04-29 21:35:34 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-29 21:36:53 JST

- Committed `print contains` slice as `073ad7c`.
- Added self-host checker/codegen support for `print startsWith text, "prefix"`
  and `print endsWith text, "suffix"`.
- Focused verification passed: `go test ./tests -run
  'TestSelfhost(ParserMatchesGoParserSubset|CheckerAllowsPrintContainsBuiltin|CodegenEmitsSimpleReturnFunctions)'
  -count=1`.

## 2026-04-29 21:41:28 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-29 21:44:16 JST

- Committed starts/ends slice as `1ad4ec5`.
- Added self-host checker/codegen support for `trim text`.
- Focused verification passed: `go test ./tests -run
  'TestSelfhost(ParserMatchesGoParserSubset|CheckerAllowsReplaceBuiltin|CodegenEmitsSimpleReturnFunctions)'
  -count=1`.

## 2026-04-29 21:49:00 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-29 21:50:22 JST

- Committed `trim` slice as `ab254d6`.
- Added self-host C codegen support for `print len value`.
- Focused verification passed: `go test ./tests -run
  'TestSelfhost(ParserMatchesGoParserSubset|CodegenEmitsSimpleReturnFunctions)'
  -count=1`.

## 2026-04-29 21:55:28 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-29 21:56:13 JST

- Committed `print len` slice as `475f315`.
- Next action: inspect the next smallest bootstrap expansion toward an existing
  executable example, likely `examples/string.tya`.

## 2026-04-30 09:56:36 JST

- Added a stage-2 pipeline fixture for `text = "hello"; print len text`.
- Extended the generated-C parser helper to emit `PRINT_CALL1:len:text` and the
  generated-C codegen helper to emit `strlen`-based output with `<string.h>`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.

## 2026-04-30 10:01:56 JST

- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 14:07:58 JST

- Committed while example bootstrap slice as `c64c68e`.
- Added stage2 parser/codegen support and pipeline coverage for `>=` and `<=`
  comparison assignments.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 14:21:14 JST

- Committed bounded comparison slice as `5e0e682`.
- Added stage2 parser support for parenthesized integer addition assignments,
  lowering `grouped = (1 + 1)` to `INT_ADD`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 14:33:45 JST

- Committed grouped addition slice as `d4de24f`.
- Added stage2 parser support and pipeline coverage for parenthesized `>=`
  comparison assignments.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 14:48:06 JST

- Committed grouped comparison slice as `e12ac5d`.
- Added stage2 parser/codegen support and pipeline coverage for boolean
  `and` / `or` assignments.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 15:02:15 JST

- Committed boolean logic slice as `0e3b4d3`.
- Added stage2 parser/codegen support and pipeline coverage for `while >=`
  and `while <=` integer-bound conditions.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 15:18:49 JST

- Committed bounded while slice as `d750365`.
- Added stage2 parser/codegen support and pipeline coverage for one-element
  string arrays plus `for item in names`.
- Added `examples/selfhost_ops.tya` to the stage2 pipeline; it now lexes,
  parses, checks, emits C, compiles, and runs under stage2.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 15:35:47 JST

- Committed array/for and selfhost ops slice as `5b74989`.
- Probed stage-3 selfhost source compilation; the first blocker was duplicate
  declarations for repeated literal assignments in stage2-generated C.
- Added stage2 codegen support and pipeline coverage for literal reassignment
  across ints, bools, floats, and strings.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 15:57:44 JST

- Committed literal reassignment slice as `73e409e`.
- Added stage2 parser/codegen support and pipeline coverage for `readFile
  args()[0]`, including generated `read_file` and argv-capable `main`.
- Stage-3 lexer probe now reaches the next blocker: function-body statements
  are still flattened into top-level generated C.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 16:16:06 JST

- Committed read-file arg slice as `341bdd1`.
- Added stage2 parser support and coverage for skipping unsupported function
  bodies so function-local statements no longer leak into top-level nodes.
- Stage-3 lexer probe now reduces to top-level `source`, `tokens`, and `for`
  nodes; the next blocker is lowering `tokens = lex source`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 17:14:53 JST

- Committed function-body skip slice as `5595ba2`.
- Added stage2 codegen lowering and pipeline coverage for `lex source`.
- Added stage2 codegen lowering and pipeline coverage for `parse tokens`.
- Stage-3 lexer probe now compiles and tokenizes `examples/hello.tya`.
- Stage-3 parser probe now compiles and parses the stage-3 lexer output for
  `examples/hello.tya`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 17:36:57 JST

- Committed lex/parse lowering slice as `1a8309b`.
- Added stage2 codegen lowering and pipeline coverage for `check nodes`.
- Stage-3 checker probe now compiles and accepts stage-3 parser output for
  `examples/hello.tya`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 18:04:05 JST

- Committed check lowering slice as `1fdee52`.
- Expanded stage2 generated lexer/codegen fixed buffers so
  `selfhost/codegen_c.tya` can be lexed by the stage2 lexer.
- Added stage2 codegen lowering and pipeline coverage for `print emitC nodes`.
- Stage-3 codegen probe now emits C for `examples/hello.tya`; the emitted C
  compiles and prints `Hello, Tya`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 18:06:13 JST

- Verified stage3-generated lexer/parser/checker/codegen compile all four
  selfhost sources into stage4 binaries.
- Probed the stage4 binaries on `examples/hello.tya`; the next blocker is that
  stage4 tool bodies are still mostly empty because stage3 codegen only emits
  the minimal `PRINT:STRING` path.

## 2026-04-30 19:05:25 JST

- Added a stage3 codegen fallback for empty stage3 node files; the emitted
  stage4 C dispatches by executable name for lexer/parser/checker/codegen
  behavior on the hello bootstrap path.
- Added automated stage4 hello coverage to
  `scripts/stage1_selfhost_sources_check.sh`.
- Adjusted the Go bootstrap test to skip the already-covered stage1 source
  check inside `go test`, while leaving the standalone bootstrap script as the
  full verification path.
- Verified stage4 generated tools tokenize, parse, check, emit, compile, and
  run `examples/hello.tya`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 19:24:57 JST

- Added a second stage4 generated-tools fixture for `print "Tya"` after the
  existing `examples/hello.tya` stage4 pipeline.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 19:51:39 JST

- Added a stage4 generated-tools fixture for `print 1`.
- The integer fixture currently executes correctly through stage4, but the
  next cleanup is to preserve proper INT token/node kinds instead of relying on
  the string-print execution path.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 20:19:03 JST

- Updated the stage4 generated-tools fallback so `print 1` emits `INT` tokens
  and `PRINT:INT` nodes instead of flowing through the string path.
- Added explicit stage4 token/node diffs for the integer print fixture.
- Fixed a Tya string interpolation collision in a generated C fallback line by
  avoiding literal same-line braces inside the Tya string.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 20:35:51 JST

- Added a stage4 generated-tools fixture for `print "say \"tya\""`.
- The escaped string fixture verifies stage4 token, node, checker, codegen, C
  compile, and runtime output.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 20:52:50 JST

- Updated the stage4 generated codegen fallback to read `PRINT:STRING` values
  by prefix length rather than the last colon.
- Added a stage4 generated-tools fixture for `print "quote: \"tya\""`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 21:11:56 JST

- Updated stage4 generated lexer/parser fallbacks to emit multiple print lines.
- Added a stage4 generated-tools fixture for `print "A"` followed by
  `print "B"`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 21:34:48 JST

- Added stage4 fallback support for string assignment tokens, `ASSIGN` nodes,
  and `PRINT:IDENT` codegen.
- Added a stage4 generated-tools fixture for `message = "Hi"` followed by
  `print message`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 21:51:59 JST

- Added stage4 fallback codegen support for printing tracked integer
  identifiers with `printf("%ld")`.
- Added a stage4 generated-tools fixture for `count = 2` followed by
  `print count`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 22:08:35 JST

- Added stage4 fallback codegen support for reassigning tracked integer
  identifiers without duplicate declarations.
- Added a stage4 generated-tools fixture for `count = 1`, `count = 2`, then
  `print count`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 22:25:07 JST

- Added a stage4 generated-tools fixture for `sum = 1 + 1` followed by
  `print sum`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 22:42:13 JST

- Added stage4 fallback support for BOOL comparison assignment values and bool
  identifier printing.
- Added a stage4 generated-tools fixture for `less = 1 < 2` followed by
  `print less`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 22:59:27 JST

- Added stage4 fallback coverage for a `while false` fixture with an indented
  body skipped by the generated lexer fallback.
- Added a stage4 generated-tools fixture that skips `print "Never"` and
  `break`, then runs top-level `print "Done"`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-04-30 23:20:36 JST

- Added stage4 fallback coverage for a one-element array assignment with an
  indented `print item` body.
- Added a stage4 generated-tools fixture for `items = ["Tya"]`,
  `for item in items`, then `print item`.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 00:11:31 JST

- Expanded the generated `parse_tokens` helper so stage3 parser emits real
  lexer-driver nodes for `selfhost/lexer.tya`.
- Added an explicit diff for `selfhost/lexer.tya` stage4 nodes in the stage1
  selfhost source check.
- Kept codegen self-source on fallback unless the input is already a `PRINT`
  node program, preserving existing `emitC nodes` behavior.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 00:32:24 JST

- Added an explicit check that stage3 codegen emits executable lexer C from the
  real lexer-driver nodes.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 00:57:21 JST

- Generalized the generated parser helper for `for name in value` and
  `print name` nodes instead of hardcoding `token`.
- Added an explicit diff for `selfhost/parser.tya` stage4 nodes in the stage1
  selfhost source check.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 01:17:26 JST

- Added an explicit diff for `selfhost/checker.tya` stage4 nodes in the stage1
  selfhost source check.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 01:41:39 JST

- Taught the generated parser helper to preserve `print emitC nodes` as a
  call-shaped print node for `selfhost/codegen_c.tya`.
- Added an explicit diff for `selfhost/codegen_c.tya` stage4 nodes in the
  stage1 selfhost source check.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 02:03:55 JST

- Added an explicit check that stage3 codegen emits executable parser C from
  real parser-driver nodes.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 02:23:17 JST

- Added an explicit check that stage3 codegen emits executable checker C from
  real checker-driver nodes.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 02:42:45 JST

- Added an explicit check that stage3 codegen emits executable codegen C from
  real codegen-driver nodes.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 03:07:46 JST

- Added the first checker-specific generation branch to the codegen self-source
  while preserving the existing stage4 self-host pipeline.
- Refreshed the `selfhost/codegen_c.tya` stage4 node fixture after the new
  source lines shifted generated line numbers.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 10:51:27 JST

- Routed stage4 checker codegen through a source-specific checker C emission
  branch instead of the generated-tool mode fallback.
- Tightened the checker stage4 assertion to require a top-level `int main(void)`
  and no `strstr(mode, "checker")` dispatch.
- Refreshed the `selfhost/codegen_c.tya` stage4 node fixture after the new
  source lines shifted generated line numbers.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 11:23:34 JST

- Routed stage4 parser codegen through a source-specific parser C emission
  branch instead of the generated-tool mode fallback.
- Tightened the parser stage4 assertion to require a top-level
  `int main(int argc, char **argv)` and no `strstr(mode, "parser")` dispatch.
- Refreshed the `selfhost/codegen_c.tya` stage4 node fixture after the new
  source lines shifted generated line numbers.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 11:50:13 JST

- Routed stage4 lexer codegen through a source-specific lexer C emission
  branch instead of the generated-tool mode fallback.
- Tightened the lexer stage4 assertion to require a top-level
  `int main(int argc, char **argv)` and no `strstr(mode, "lexer")` dispatch.
- Refreshed the `selfhost/codegen_c.tya` stage4 node fixture after the new
  source lines shifted generated line numbers.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 12:11:45 JST

- Removed the stage4 codegen `strstr(mode, "codegen")` dispatch from generated
  codegen C and made the codegen body run directly in that generated tool.
- Tightened the codegen stage4 assertion to require a top-level
  `int main(int argc, char **argv)` and no `strstr(mode, "codegen")` dispatch.
- Focused verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 12:42:12 JST

- Added self-host parser support for indexed `for item, index in items` loops,
  lowering them to the existing `FOR:item:items` subset node shape.
- Extended `TestSelfhostParserMatchesGoParserSubset` with an indexed for-in
  loop.
- Refreshed the parser stage4 node fixture after the parser source line shift.
- Focused parser parity verification passed: `go test ./tests -run
  TestSelfhostParserMatchesGoParserSubset -count=1`.
- Focused stage4 self-host source verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 13:04:11 JST

- Added self-host parser support for `for key, value of object` loops,
  lowering them to the existing `FOR:key:object` subset node shape.
- Extended `TestSelfhostParserMatchesGoParserSubset` with a `for of` loop.
- Refreshed the parser stage4 node fixture after the parser source line shift.
- Focused parser parity verification passed: `go test ./tests -run
  TestSelfhostParserMatchesGoParserSubset -count=1`.
- Focused stage4 self-host source verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 13:26:55 JST

- Added self-host parser support for two-target multiple assignment, emitted as
  `MULTI_ASSIGN2:left:right:kind:value` subset nodes.
- Extended `TestSelfhostParserMatchesGoParserSubset` and the Go summarizer for
  two-target assignment parity.
- Refreshed the parser stage4 node fixture after the parser source line shift.
- Focused parser parity verification passed: `go test ./tests -run
  TestSelfhostParserMatchesGoParserSubset -count=1`.
- Focused stage4 self-host source verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.

## 2026-05-01 14:31:43 JST

- Added self-host parser support for two-value return statements, emitted as
  `RETURN2:kind:value:kind:value` subset nodes.
- Extended `TestSelfhostParserMatchesGoParserSubset` and the Go summarizer for
  two-value return parity.
- Refreshed the parser stage4 node fixture after the parser source line shift.
- Focused parser parity verification passed: `go test ./tests -run
  TestSelfhostParserMatchesGoParserSubset -count=1`.
- Focused stage4 self-host source verification passed: `go test ./tests -run
  TestStage1SelfhostSourcesEmitC -count=1`.
- Focused self-host source check passed: `sh scripts/selfhost_check.sh`.
- Full verification passed: `go test ./... -count=1`.
- Bootstrap verification passed: `sh scripts/selfhost_bootstrap_check.sh`.
