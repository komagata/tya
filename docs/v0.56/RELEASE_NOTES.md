---
layout: doc
title: Release Notes
permalink: /v0.56/release-notes/
---

# Tya v0.56 Release Notes

> **Status:** shipped. `tya version` reports `0.56.0` and
> `ROADMAP.md` carries the matching `Released` entry.

## TL;DR

v0.56 finishes the diagnostics pipeline work started in v0.54:

- Parser, Codegen, and Runner now share the
  `(X, []diag.Diagnostic, error)` shape — callers can iterate
  diagnostics directly instead of unwrapping via `errors.As`.
- Expression-level recovery in the parser: `CallExpr` /
  `ArrayLit` / `DictLit` element lists no longer collapse on
  the first bad element. One pass surfaces every per-element
  diagnostic.

The language surface is unchanged from v0.55.

## What's new

### Signature unification

```go
// Parser
prog, diags, err := parser.Parse(toks)
prog, diags, err := parser.ParseWithComments(toks, comments)

// Codegen
csrc, diags, err := codegen.EmitC(prog)
csrc, diags, err := codegen.EmitCWithPath(prog, sourcePath)
csrc, reg, diags, err := codegen.EmitCWithCoverage(prog, sourcePath, opt)

// Runner
diags, err := runner.RunFile(path, stdin, stdout, args)
```

The `diags` slice is `nil` (and `err == nil`) on a clean run,
and carries one or more `diag.Diagnostic` entries otherwise.
The `*ParserError` / `*CodegenError` / `*RunnerError` wrappers
continue to satisfy `errors.As` for callers that don't want to
migrate yet.

`RunnerError` widens from `Diag diag.Diagnostic` to
`Diags []diag.Diagnostic`. A `Diag()` method returns
`Diags[0]` for pre-v0.56 call sites that read `.Diag` as a
single value.

### Expression-level recovery

```tya
print(@, @, @)
```

```
-- EXPECTED INSTANCE FIELD NAME -------- file.tya:1:8
expected instance field name near ","
…
-- EXPECTED INSTANCE FIELD NAME -------- file.tya:1:11
…
-- EXPECTED INSTANCE FIELD NAME -------- file.tya:1:14
…

Found 3 error(s), 0 warning(s).
```

Three errors, one pass. The same recovery rules apply inside
`ArrayLit` (`[…]`) and `DictLit` (`{…}`) literals. Nested
brackets are skipped over so an inner stray `,` does not stop
outer-list recovery.

Binary chains (`a + broken + c`) and member chains
(`a.broken.c`) remain whole-expression failures in v0.56 and
are queued for v0.57+.

## Migration

For every direct caller of the renamed APIs, add a `_` to the
destructure or read the new `diags` slice:

```go
// Pre-v0.56
prog, err := parser.Parse(toks)

// v0.56 (no diags)
prog, _, err := parser.Parse(toks)

// v0.56 (consume diags)
prog, diags, err := parser.Parse(toks)
for _, d := range diags { … }
```

`errors.As(err, &perr)` continues to work, so callers that
unwrap the structured payload via the wrapper do not need to
move yet. The wrappers will be revisited once every caller has
migrated to direct slice access.

## Tooling

- 4 new fixtures under `tests/testdata/v56_diag/` covering
  CallExpr / ArrayLit / DictLit expression-level recovery and
  end-to-end signature propagation.
- `Formula/tya.rb` → `0.56.0`,
  `editors/vscode/package.json` → `0.56.0`.
- 65 caller sites (33 non-test + 32 test) migrated to the new
  signatures.

## Next

- Codegen multi-error (collect every `unsupported AST shape`
  instead of bailing on the first).
- Binary-chain and member-chain expression-level recovery.
- `*ParserError` / `*CodegenError` / `*RunnerError` deprecation
  once every caller iterates diags directly.
- Other Toolchain Epic items (`tya task` v2, `tya new` v2,
  `tya lint` v4).
