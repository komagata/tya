---
layout: doc
title: Release Notes
permalink: /v0.64/release-notes/
---

# Tya v0.64 Release Notes

v0.64 adds first-class lexical closures, iterable sequence protocols, and a
set of standard-library protocol interfaces for comparison, equality, I/O,
serialization, and human-readable string conversion.

## Language

- Lexical closures now capture free variables across compile-to-C boundaries,
  including nested function literals and captured values that must survive GC.
- The checker rejects unsupported captured-binding indexed/member mutation with
  a stable diagnostic while preserving legacy self-host compatibility where it
  is still required.

## Iteration

- `Iterator`, `Iterable`, and `Sequence` define the standard iteration
  protocol used by `for ... in`.
- Arrays, dictionaries, and strings can be consumed through the protocol
  without boxing their primitive values.
- Sequence helpers provide lazy `map`, `filter`, `take`, and `to_a`
  composition over iterable values.

## Standard Protocols

- `Comparable` defines `compare(other)` with derived `lt?`, `lte?`, `gt?`,
  `gte?`, and `between?` helpers. Primitive numbers and strings expose the
  same methods.
- `Equatable` defines `equal?(other)`. Primitive scalar values follow normal
  runtime equality, while arrays and dictionaries use deep equality.
- `io` now exposes `Readable`, `Writable`, `Closable`, and `Flushable`
  protocol interfaces. Existing `io.Reader`, `io.Writer`, `net/socket.Socket`,
  and `net/socket.Server` declare the matching contracts.
- `serialization.Serializable` defines `to_data()` as the canonical structured
  serialization hook. `Serializer.to_data` prefers it over the older
  `to_serialized()` hook.
- `Stringable` defines the human-readable `to_s()` protocol and documents the
  primitive conformance rule for Number, String, Array, Dict, Boolean, and Nil.

## Documentation

- `docs/SPEC.md`, `docs/API.md`, and `docs/STDLIB.md` document the new
  protocols and distinguish `Stringable.to_s()` from
  `Serializable.to_data()`.
- Completed PRDs for the release were moved to `docs/prd/completed/`.

## Verification

The release gate is:

```sh
go test ./... -count=1
```

On the release machine, the full suite exceeded Go's default 10-minute package
timeout after all tests were already in the long `tya/tests` package. The same
release gate with an explicit 20-minute timeout passed:

```sh
go test ./... -count=1 -timeout=20m
```
