# Feature: v1 Public Contract Finalization

## Goal

Freeze the final v1.0.0 public-contract decisions for generated-C runtime
strictness, stable diagnostic codes, stdlib blocker scope, compiler
introspection compatibility, frozen release docs, platform-dependent APIs,
conformance gates, and legacy compatibility aliases.

## Context

`docs/SPEC.md`, `docs/STRICT_SEMANTICS.md`, and the completed v1 specs already
define the main language surface: strict dynamic semantics, excluded syntax,
structured errors, runtime boundaries, release/self-host policy, and v1 stdlib
gaps. This spec records the remaining non-obvious decisions that affect what
Tya v1.0.0 publicly promises.

The accepted direction is to keep the v1 contract crisp: interpreter and
generated-C behavior match, every user-facing diagnostic has one stable code
namespace, required stdlib gaps block the release, platform-dependent APIs are
importable everywhere, and legacy compatibility does not become part of the v1
public API.

## Behavior

- Generated-C runtime strictness matches the interpreter for the public
  v1.0.0 execution path.
  - Compatibility fallbacks currently documented as C-runtime exceptions in
    `docs/STRICT_SEMANTICS.md` must be removed before v1.0.0.
  - Self-host sources must be migrated before removing a fallback when the
    existing self-host fixed point depends on it.
  - Public v1 behavior must not rely on generated C returning `nil`, reading
    numeric fallback fields, or otherwise accepting a program the interpreter
    rejects.
- Stable diagnostic codes use one namespace.
  - Parser, checker, codegen, runtime, CLI, LSP, formatter, package manager,
    stdlib, release, and bootstrap failures use `TYA-E....` stable codes.
  - Runtime structured error values may carry domain-specific `kind` and
    machine-readable `code` fields, but the user-facing diagnostic code remains
    `TYA-E....`.
  - CLI human output, CLI JSON output, LSP diagnostics, and runtime failure
    reporting expose the same stable diagnostic code for the same failure.
- The v1.0.0 stdlib blocker set includes:
  - `regex/Regex`;
  - filesystem utilities in `file/File` and `dir/Dir`;
  - the `time/Time` contract;
  - environment and process APIs in `os/Os` and `process/Process`;
  - `hmac/Hmac`.
  These specs must be implemented, documented, and tested before v1.0.0.
- Compiler introspection stdlib compatibility is intentionally narrow.
  - Stable v1 APIs are limited to the minimal documented entry points such as
    `Lexer.lex`, `Parser.parse`, `Checker.check`, `Format.format`, and
    explicitly documented AST helper methods.
  - Full AST dictionary shapes, every internal checker detail, and all
    implementation helper fields are not v1 compatibility guarantees unless
    documented in `docs/SPEC.md`.
  - Tooling may expose more data, but undocumented fields may change across
    v1.x releases.
- v1.0.0 release documentation is frozen under `docs/v1.0/`.
  - The release creates `docs/v1.0/SPEC.md`,
    `docs/v1.0/RELEASE_NOTES.md`, and `docs/v1.0/MIGRATION.md`.
  - `docs/SPEC.md` remains the editable current-development specification
    after the v1.0.0 release.
  - Frozen docs must match the v1 release artifact behavior.
- Platform-dependent v1 stdlib APIs are importable on every supported release
  platform.
  - Unsupported operations fail only when called.
  - Unsupported operation failures are structured errors with stable
    `TYA-E....` diagnostic codes.
  - Programs can import the same stdlib packages on Linux, macOS, and Windows
    without import-time platform branching.
- v1 conformance remains a repository release gate, not a public v1 command.
  - The v1.0.0 release process may use Go tests, testscript fixtures, shell
    scripts, and release packaging checks from the repository.
  - Tya does not add `tya conformance` or promise a public conformance runner
    for v1.0.0.
  - Release notes may describe the internal verification command set, but
    external users are not promised a stable conformance CLI.
- Legacy aliases and compatibility facades are excluded from the v1 public API.
  - Public docs must prefer canonical class-style stdlib APIs and canonical
    language spellings.
  - Remaining aliases required only for `selfhost/v01` or bootstrap recovery
    must be documented as `legacy compatibility only`.
  - Legacy compatibility aliases are not v1.x compatibility guarantees.

## Scope

- `docs/SPEC.md`
- `docs/STRICT_SEMANTICS.md`
- `docs/v1.0/SPEC.md`
- `docs/v1.0/RELEASE_NOTES.md`
- `docs/v1.0/MIGRATION.md`
- `ROADMAP.md`
- parser, checker, codegen, runtime, CLI, LSP, formatter, package manager,
  stdlib, release, and bootstrap diagnostics
- generated-C runtime strictness checks
- self-host source migration required to remove C-runtime compatibility
  fallbacks
- stdlib blocker specs for regex, filesystem, time, environment/process, and
  HMAC
- compiler introspection stdlib docs and tests
- release verification tests and scripts

## Out of Scope

- Adding a public `tya conformance` command for v1.0.0.
- Introducing separate diagnostic namespaces such as `TYA-R....` or
  `TYA-C....`.
- Preserving generated-C runtime behavior that contradicts the interpreter for
  public v1 programs.
- Making all compiler-internal AST and checker details v1 stable.
- Promising platform support by failing imports on unsupported operations.
- Treating legacy aliases as v1 public API.

## Acceptance Criteria

- `docs/STRICT_SEMANTICS.md` no longer lists generated-C compatibility
  exemptions for public v1.0.0 behavior.
- Interpreter and generated-C behavior agree for strict dynamic semantics,
  callable checks, member access, indexing, nil handling, arithmetic, error
  handling, channels, and stdlib blocker APIs.
- All user-facing diagnostics use `TYA-E....` codes across parser, checker,
  codegen, runtime, CLI, LSP, formatter, package manager, stdlib, release, and
  bootstrap tooling.
- Runtime structured errors keep domain `kind` and `code` fields while also
  surfacing a stable `TYA-E....` diagnostic code.
- The v1.0.0 release gate includes implemented and passing coverage for
  `regex/Regex`, filesystem utilities, `time/Time`, environment/process APIs,
  and `hmac/Hmac`.
- `docs/SPEC.md` documents the stable compiler introspection entry points and
  explicitly excludes undocumented AST/checker internals from v1 compatibility.
- Frozen docs exist under `docs/v1.0/` and match the release behavior.
- Platform-dependent stdlib packages import successfully on supported release
  platforms; unsupported operations fail at call time with structured
  `TYA-E....` errors.
- The release gate remains repo-internal and does not require a stable public
  conformance command.
- Public docs either remove legacy aliases or mark unavoidable ones as
  `legacy compatibility only`.
- Existing self-host fixed-point gates remain valid until replaced by the
  latest-spec v1.0.0 fixed-point gate.

## Tests To Add

Parser/checker/runtime/codegen tests:

- `TestGeneratedCStrictnessMatchesInterpreter`
  - Runs representative invalid and valid programs through interpreter and
    generated C.
  - Expected: generated C rejects the same invalid public v1 programs and
    produces the same valid results.

- `TestRuntimeDiagnosticsUseStableTyaECodes`
  - Exercises wrong kind, wrong arity, non-callable calls, invalid member
    access, invalid indexing, closed channel send, invalid UTF-8, and stdlib
    structured errors.
  - Expected: every user-facing failure exposes a `TYA-E....` code.

- `TestRuntimeErrorValuesCarryDiagnosticCode`
  - Creates and catches structured errors from runtime and stdlib failures.
  - Expected: error values preserve domain `kind` and `code` while diagnostic
    reporting uses the stable `TYA-E....` code.

Documentation tests:

- `TestStrictSemanticsHasNoPublicCRuntimeExemptions`
  - Expected: `docs/STRICT_SEMANTICS.md` does not document generated-C
    behavior that accepts public v1 programs rejected by the interpreter.

- `TestSpecDocumentsUnifiedDiagnosticNamespace`
  - Expected: `docs/SPEC.md` documents one `TYA-E....` diagnostic-code
    namespace for all user-facing failures.

- `TestSpecDocumentsV1StdlibBlockerSet`
  - Expected: `docs/SPEC.md` lists regex, filesystem utilities, time,
    environment/process, and HMAC as part of the v1 stdlib surface.

- `TestSpecDocumentsCompilerIntrospectionCompatibilityBoundary`
  - Expected: `docs/SPEC.md` documents stable compiler introspection entry
    points and excludes undocumented AST/checker internals from v1
    compatibility.

- `TestFrozenV10DocsExist`
  - Expected: `docs/v1.0/SPEC.md`, `docs/v1.0/RELEASE_NOTES.md`, and
    `docs/v1.0/MIGRATION.md` exist and describe the released behavior.

- `TestPublicDocsMarkLegacyAliases`
  - Expected: public docs contain no undocumented compatibility aliases; any
    unavoidable alias is marked `legacy compatibility only`.

Stdlib and platform tests:

- `TestV1StdlibBlockersImplemented`
  - Imports and exercises `regex/Regex`, filesystem utilities, `time/Time`,
    environment/process APIs, and `hmac/Hmac`.
  - Expected: all pass through `tya run` and generated C.

- `TestPlatformDependentStdlibImportsEverywhere`
  - Imports platform-dependent stdlib packages on supported release platforms.
  - Expected: imports succeed; unsupported operations fail only at call time
    with structured `TYA-E....` errors.

Release tests:

- `TestReleaseGateIsRepoInternal`
  - Expected: release verification uses repository tests/scripts and does not
    require or document a stable `tya conformance` command.

Testscript coverage:

- `v1_public_contract_finalization.txtar`
  - Covers generated-C/interpreter strictness parity, stable diagnostic codes,
    stdlib blocker imports, platform-dependent unsupported-operation failures,
    and legacy alias documentation checks where practical.

## Verification

```sh
go test ./internal/eval -run 'Strict|Diagnostic|Runtime|Stdlib' -count=1
go test ./internal/codegen -run 'Strict|Diagnostic|Runtime|Stdlib' -count=1
go test ./tests -run 'TestStrictSemanticsHasNoPublicCRuntimeExemptions|TestSpecDocumentsUnifiedDiagnosticNamespace|TestSpecDocumentsV1StdlibBlockerSet|TestSpecDocumentsCompilerIntrospectionCompatibilityBoundary|TestFrozenV10DocsExist|TestPublicDocsMarkLegacyAliases' -count=1
go test ./tests -run 'TestV1StdlibBlockersImplemented|TestPlatformDependentStdlibImportsEverywhere|TestReleaseGateIsRepoInternal|TestV.*Scripts|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
