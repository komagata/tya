---
layout: doc
title: v1.0 Release Checklist
permalink: /v1.0/release-checklist/
---

# Tya v1.0.0 Release Checklist

Status: release gate checklist. This document records pass/fail gates for the
v1.0.0 release. The language and tool contract remains in `docs/SPEC.md`.

## Required Gates

- [ ] Strict semantics gate passes for the current v1.0.0 rule matrix in
  `docs/STRICT_SEMANTICS.md`.
- [ ] Latest self-host fixed point passes for the release-critical self-host
  compiler line.
- [ ] No-Go bootstrap proof passes: released artifacts rebuild the self-hosted
  compiler and prove the fixed point without requiring Go.
- [ ] Structured diagnostics coverage passes for parser, checker, codegen,
  runtime, CLI, LSP, stdlib, release, and bootstrap failures.
- [ ] Standard-library blocker APIs are implemented, documented, and covered:
  `regex/Regex`, filesystem utilities, `time/Time`, environment/process APIs,
  and `hmac/Hmac`.
- [ ] Frozen v1.0 docs are present and aligned with `docs/SPEC.md`,
  `docs/STRICT_SEMANTICS.md`, release notes, and migration notes.
- [ ] Release artifacts build and smoke-test for every supported platform and
  include the expected standard library, runtime, installer, and editor assets.
- [ ] Package-manager behavior matches the v1 contract: explicit dependencies,
  lockfile hashes, no implicit dependency fetching by ordinary commands, and no
  public central-registry publishing command.

## Release Commands

The release gate is repository-internal. Before tagging v1.0.0, run the full
release gate and repository test suite:

```sh
scripts/release_gate.sh
go test ./... -count=1
```
