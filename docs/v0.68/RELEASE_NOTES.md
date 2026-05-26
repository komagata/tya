---
layout: doc
title: Release Notes
permalink: /v0.68/release-notes/
---

# Tya v0.68 Release Notes

v0.68 changes class and interface member declarations from assignment-like
`=` syntax to canonical member-declaration `:` syntax.

## Class and Interface Syntax

Class and interface bodies now use `:` for member declarations:

```tya
interface Named
  name: ->

class User implements Named
  NAME: "guest"
  static count: 0

  initialize: name = "guest" ->
    self.name = name

  name: ->
    self.name
```

The old class/interface member declaration spelling with `=` is rejected with
a targeted diagnostic. Ordinary top-level bindings, local assignments, and
assignments inside method bodies still use `=`.

Method declarations may be parsed with or without parentheses around the
parameter list, but `tya format` emits the canonical form without parentheses:

```tya
class User
  label: name = "guest" ->
    name
```

## Tooling

- `tya format` emits `:` for every class and interface member declaration.
- Editor syntax samples and grammars recognize the new member syntax.
- The self-host compiler fixtures are updated so the fixed-point invariant
  remains covered under the new syntax.

## Verification

The release gate is expected to pass:

```sh
go test ./... -count=1 -timeout=20m
```
