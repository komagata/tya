---
layout: doc
title: Release Notes
permalink: /v1.0/release-notes/
---

# Tya v1.0.0 Release Notes

Status: draft release contract for the v1.0.0 specification freeze.

v1.0.0 freezes the documented language, standard-library, package, diagnostic,
and release artifact behavior in `docs/SPEC.md`. Compatibility covers accepted
syntax, documented public APIs, documented runtime behavior, stable diagnostic
codes, CLI JSON diagnostic schema, and release artifact semantics.

Undocumented implementation details, generated C internals, internal Go package
layout, unsupported experimental flags, external package internals, and post-v1
ecosystem packages are not compatibility guarantees.

The self-hosted compiler is the primary compiler direction. The Go
implementation remains in the repository as a reference implementation and
bootstrap recovery path for v1.0.0. Release verification requires the no-Go
self-host bootstrap proof on Linux and macOS release artifacts.

## Migration Notes

Pre-v1 behavior that contradicts `docs/SPEC.md` is rejected without a
deprecation window. Representative replacements:

- Use structured `raise error(...)` and `try/catch/finally`; public `value, err`
  failure conventions are not v1 API contracts.
- Use `catch err` only; branch on `err["kind"]`, `err["code"]`, or
  `err["data"]` inside the catch body.
- Use `try/finally` and explicit `close()` methods for cleanup; `defer` is not
  v1 syntax.
- Use documented wrapper-class APIs instead of removed top-level primitive
  helpers.
- Use explicit conversion methods; operations do not implicitly convert values.

WebAssembly remains documented for v1.0.0 but is not a release-blocking target.
# Tya v1.0 Release Notes

Tya v1.0 freezes the public language, runtime, stdlib, diagnostics, and release
contract documented in `SPEC.md`.

The v1 stdlib blocker set is implemented and documented: `regex/Regex`,
filesystem utilities in `file/File` and `dir/Dir`, `time/Time`,
environment/process APIs in `os/Os` and `process/Process`, and `hmac/Hmac`.

All user-facing diagnostics use the stable `TYA-E....` namespace. Runtime
structured errors keep domain `kind` and `code` fields for programmatic
handling while CLI/LSP/reporting surfaces stable Tya diagnostics.

The release gate is repository-internal and uses Go tests, testscript fixtures,
self-host fixed-point checks, and release packaging checks. Tya v1.0 does not
add a public `tya conformance` command.
