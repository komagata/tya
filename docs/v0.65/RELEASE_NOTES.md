---
layout: doc
title: Release Notes
permalink: /v0.65/release-notes/
---

# Tya v0.65 Release Notes

v0.65 removes legacy primitive helper API surfaces and keeps primitive
operations on their canonical receiver-method spelling.

## Language

- Receiver-method syntax remains the public primitive API, such as
  `" tya ".trim()`, `[1, 2].len()`, `{ name: "Tya" }.has("name")`,
  `42.to_s()`, `value.class`, and `value.class.name`.
- Receiverless primitive helpers such as `len(items)`, `trim(text)`,
  `keys(dict)`, `push(items, value)`, and `to_number(value)` are rejected.
- Lowercase pseudo-module primitive helpers such as `string.trim(text)`,
  `array.len(items)`, `dict.has(obj, key)`, and `value.nil?(value)` are
  rejected.
- Diagnostics for removed helper APIs include the canonical replacement.

## Implementation

- The evaluator and C emitter no longer expose the removed public helper
  entry points.
- Primitive receiver methods retain direct evaluator/codegen/runtime fast
  paths where those paths implement the canonical receiver-method behavior.
- The self-hosted v01 and v02 compilers no longer predeclare the removed
  primitive helper names.

## Documentation

- Current spec documents list only the retained low-level builtins and remove
  the obsolete primitive helper entries.
- Examples, self-host fixtures, and current test data use canonical
  receiver-method syntax.

## Verification

The release gate passed:

```sh
go test ./... -count=1 -timeout=20m
```
