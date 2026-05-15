---
layout: doc
title: Spec
permalink: /v0.39/spec/
---

# Tya v0.39 Specification — Canonical Syntax (Surface Cleanup)

Tya v0.39 finalizes the Canonical Syntax surface that landed in
v0.38. The formatter subcommand is now spelled with the full word
`tya format`, and its style opt-outs are removed: there is exactly
one canonical layout, with no flag to disable any of it.

This release is purely a CLI-surface tightening. The serializer,
parser, and AST changes were already shipped in v0.38; v0.39 just
removes the transitional spellings.

## Goals (v0.39)

- The formatter subcommand is `tya format`. The legacy short
  spelling `tya fmt` is rejected with a structured hint.
- The transitional `--text` and `--ast` opt-outs are removed from
  `tya format`. The AST-driven canonical serializer is the only
  user-visible behavior; the text-pass fallback for unsupported
  inputs is now an invisible safety net rather than a flag.
- `--text` typed by a user yields `unknown format option: --text`.
- `--ast` typed by a user yields `unknown format option: --ast`.

## Non-Goals (v0.39)

- Any change to the canonical layout produced by the formatter.
- Any change to the parser, AST, lexer, or runtime.
- Any change to comment-position diagnostics (still
  `TYA-E0150`, emitted by `tya check`).

## CLI Surface

```
tya format [-w] [path]
```

| Flag | Behavior                                              |
|------|--------------------------------------------------------|
| (none) | Print the canonical AST-driven layout to stdout.   |
| `-w`   | Write canonical output back to the file.           |

There are no other flags. `--text`, `--ast`, and any other style
flag are rejected.

`tya fmt` returns exit code 2 with a structured hint pointing at
`tya format`.

## Self-Host Invariant

No language change; the self-host fixed point holds.
`TestSelfhostV01Scripts` continues to pass.

## Acceptance Criteria

A v0.39 build is acceptable when:

1. `tya format path.tya` prints canonical output.
2. `tya format -w path.tya` writes canonical output in place.
3. `tya format --text path.tya` and `tya format --ast path.tya`
   both fail with `unknown format option: …`.
4. `tya fmt path.tya` fails with a hint pointing at `tya format`.
5. `go test ./... -count=1` passes, including the self-host
   invariant.

## References

- [`docs/CANONICAL_SYNTAX.md`](../CANONICAL_SYNTAX.md) — the
  authoritative canonical-syntax specification.
- v0.38 SPEC: Canonical Syntax landing.
