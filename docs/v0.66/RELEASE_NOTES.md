---
layout: doc
title: Release Notes
permalink: /v0.66/release-notes/
---

# Tya v0.66 Release Notes

v0.66 freezes the v1 release-gate policy and locks in the remaining v1
standard-library blocker contracts.

## Release Gate

- The latest self-host compiler line is the v1.0.0 release-critical
  self-host gate.
- `selfhost/v01` remains maintained legacy regression coverage, but no longer
  defines v1.0.0 language validity or release readiness once the latest
  self-host gate is green.
- `scripts/dev_selfhost_smoke.sh` provides a fast development self-host smoke
  gate.
- `scripts/release_gate.sh` runs the full release-tag gate: latest self-host
  fixed point, no-Go bootstrap proof, and full repository conformance checks.

## Standard Library

- `regex/Regex` is part of the v1 stdlib blocker set.
- Filesystem utilities in `file/File` and `dir/Dir` are part of the v1 stdlib
  blocker set.
- `time/Time` is part of the v1 stdlib blocker set.
- Environment and process APIs in `os/Os` and `process/Process` are part of
  the v1 stdlib blocker set.
- `hmac/Hmac` is part of the v1 stdlib blocker set.

## Specification

- v1 authority order is now documented: specification first, latest self-host
  implementation second when it implements the documented v1 surface, and the
  Go implementation third as reference/bootstrap recovery.
- v1.x syntax evolution is conservative: ordinary v1.x releases may add APIs,
  tooling, diagnostics, and compatible runtime behavior, but new syntax is
  reserved for v2 planning or an explicitly accepted experimental path.
- Package-manager behavior is documented as v1 public contract, including
  manifest validation, explicit git/path dependencies, lockfiles, native
  metadata, and no central registry.
- `docs/DIAGNOSTICS.md` lists every current public `TYA-E....` diagnostic code.

## Verification

The release gate passed:

```sh
go test ./... -count=1 -timeout=20m
```
