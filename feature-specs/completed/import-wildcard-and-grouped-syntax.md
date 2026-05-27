# Feature: Import Wildcard and Grouped Syntax

## Goal
Change Tya import syntax so package-wide imports use an explicit Java-like `/*` suffix, and multiple imports can be written and formatted as an indented `import` block.

## Context
Current Tya import syntax accepts one import per line, for example `import base64` or `import net/http as http`. Directory package imports currently do not require a wildcard marker, so a path can be ambiguous between a single module-style import and a package-wide import. Formatted Syntax also currently treats imports as atomic one-line statements.

This feature makes package-wide import intent explicit with `/*` and adds a canonical grouped import form for multiple imports. It is independent from `feature-specs/one-line-function-method-formatting.md` and `feature-specs/class-interface-member-blank-lines.md`, but should be implemented after those queued formatter specs because it also changes formatter output.

## Behavior
- Package-wide imports use `/*` as the final path suffix.
- `import base64/*` imports all public names from the `base64` directory package into the current scope, matching the current unaliased directory package import behavior.
- Old package-wide imports without `/*`, such as `import base64` for a directory package, are no longer accepted as package-wide imports.
- Imports without `/*`, such as `import base64/base64` or `import net/http/client`, remain valid single file/module imports.
- Wildcard imports may use aliases.
- `import net/http/* as http` imports the `net/http` directory package under the alias namespace, matching the current aliased directory package import behavior.
- `*` is only valid as the final segment in the exact form `path/*`.
- Invalid wildcard forms include `import *`, `import base64*`, `import base64/*/foo`, and `import base64/**`.
- Multiple imports can be written as an indented import block:

```tya
import
  base64/base64
  net/http/client
  net/http/server
```

- Import block entries accept the same path, wildcard, and alias syntax as single-line imports:

```tya
import
  base64/*
  net/http/client as client
  net/http/* as http
```

- An `import` block must contain at least one import entry.
- Import block entries are indented one level under `import`.
- Import block entries are not prefixed with the `import` keyword.
- Import statements and import blocks remain top-level only.
- `tya format` uses one-line import syntax when there is exactly one import.
- `tya format` uses grouped import syntax when there are two or more consecutive top-level imports.
- Formatter output sorts grouped import entries using the existing import sorting behavior.
- Formatter output removes trailing whitespace from import block entries.
- Formatting remains idempotent.

Examples:

```tya
import base64
```

is no longer a package-wide import. Package-wide import must be:

```tya
import base64
```

```tya
import base64/base64
import net/http/client
```

formats to:

```tya
import
  base64/base64
  net/http/client
```

```tya
import net/http/* as http
```

keeps a one-line form because it is a single import.

## Scope
- Update lexer/parser import handling to accept `path/*` and indented import blocks.
- Update AST representation as needed, or parse import blocks into the existing flat `ImportStmt` list when that keeps downstream behavior simpler.
- Update formatter/unparser to emit canonical grouped import syntax for two or more consecutive imports.
- Update runner import collection, module resolution, package directory resolution, top-level import stripping, and synthesized package source behavior for explicit wildcard package imports.
- Update import validation so `*` is only accepted as the final segment in `path/*`.
- Update error messages for invalid wildcard import syntax and deprecated directory package imports without `/*`.
- Update `docs/SPEC.md` and `docs/ja/spec.md`, including the Formatted Syntax section that currently says imports are atomic and not line-wrapped.
- Update examples and stdlib imports if current package-wide imports rely on the old no-wildcard form.
- Add parser, formatter, runner, and integration tests for wildcard imports, grouped imports, aliases, invalid wildcard forms, old directory package import rejection, sorting, and idempotence.

## Out of Scope
- No recursive import wildcard such as `**`.
- No mid-path wildcard.
- No relative import syntax.
- No per-name import list syntax.
- No formatter configuration for grouped import style.
- No change to public/private visibility rules for package imports.
- No change to single file/module import semantics other than disambiguating them from package-wide imports.

## Acceptance Criteria
- `import base64/*` imports public names from the `base64` package.
- `import net/http/* as http` imports the `net/http` package under the `http` alias namespace.
- `import base64/base64` remains a valid single file/module import.
- `import base64` no longer works as a package-wide import for a directory package.
- `import *`, `import base64*`, `import base64/*/foo`, and `import base64/**` are rejected with clear parse or validation errors.
- A grouped import block with normal imports, wildcard imports, and aliases parses successfully.
- A grouped import block with no entries is rejected.
- Two or more consecutive one-line imports format into one grouped import block.
- One import formats as one line.
- Grouped import entries are sorted consistently with existing import sorting.
- Running the formatter twice produces identical output.
- Documentation in English and Japanese describes the new wildcard and grouped import syntax.

## Verification
```sh
go test ./internal/parser ./internal/formatter ./internal/runner ./tests -count=1
go test ./... -count=1
```
