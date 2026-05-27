# Feature: Format Import Blank Lines

## Goal

`tya format` should remove unnecessary blank lines inside a consecutive import block so manually separated imports are rewritten to the canonical import layout.

## Context

Issue #11 reports that `tya format -w` currently preserves blank lines between consecutive import statements:

```tya
import os

import cli

class Cli
  initialize = ->
    self.value = 1
```

The desired formatted output is:

```tya
import os
import cli

class Cli
  initialize = ->
    self.value = 1
```

Current formatter logic in `internal/formatter/unparse.go` treats stdlib and user import groups as separated by a blank line. This feature changes that canonical layout: consecutive imports belong to one import block even when they mix stdlib and user imports.

## Behavior

- `tya format` removes blank lines between consecutive top-level import statements.
- Consecutive imports remain a single import block even when stdlib imports and user imports are adjacent.
- A single blank line remains between the final import in an import block and the following non-import top-level declaration.
- `tya format --check` reports formatted syntax drift when a file contains blank lines between consecutive imports.
- Import sorting behavior remains otherwise unchanged unless the implementation discovers it must be adjusted to satisfy the single import block layout.
- Blank lines between non-import top-level sections are not changed by this feature.
- Non-consecutive import statements separated by an intervening non-import statement are not merged across that non-import statement.

## Scope

- Formatter top-level layout logic in `internal/formatter/unparse.go`.
- Formatter or CLI tests that cover import block blank-line normalization.
- Existing formatter tests may need golden output updates where they currently expect a blank line between stdlib and user imports.

## Out of Scope

- Changing parser, checker, import resolution, or runtime import behavior.
- Adding or removing import sorting categories beyond what is necessary to remove blank lines inside consecutive import blocks.
- Collapsing blank lines between non-import top-level definitions.
- Moving imports across non-import statements.
- Changing comment attachment semantics except where comments naturally stay attached to the import statement they already describe.

## Acceptance Criteria

- Given issue #11's example, `tya format -w` rewrites it to consecutive imports followed by one blank line before `class Cli`.
- `tya format --check` fails before formatting and passes after formatting for a file with blank lines between consecutive imports.
- A formatter unit test or CLI test verifies that mixed lib/user consecutive imports do not get a blank line between them.
- Existing intentionally separate non-import top-level sections are not collapsed unexpectedly.
- Formatter output remains idempotent for the new import layout.

## Verification

```sh
go test ./internal/formatter ./cmd/tya -count=1
go test ./tests -run TestV44Scripts/tya_format -count=1
```
