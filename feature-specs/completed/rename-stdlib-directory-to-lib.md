# Feature: Rename stdlib Directory to lib

## Goal
Rename the repository and installed standard-library source directory from `stdlib/` to `lib/`, while keeping "standard library" as the conceptual name of the bundled library.

## Context
Tya currently stores bundled standard-library sources under the repository root `stdlib/`. The release packages install those files under `share/tya/stdlib`, tests and scripts commonly set `TYA_STDLIB_DIR`, and generated API docs are built from `stdlib` into `docs/stdlib`.

The desired source layout is shorter and closer to Ruby-style naming: the standard library remains the "standard library", but its source directory is named `lib/`.

This feature should be implemented after the currently queued formatter/import specs, especially `feature-specs/import-wildcard-and-grouped-syntax.md`, because both features touch import resolution, standard-library lookup, formatter fixtures, and docs generation paths.

## Behavior
- Rename the repository root directory `stdlib/` to `lib/`.
- Standard-library source files live under `lib/`.
- Installed release packages place bundled standard-library sources under `share/tya/lib`.
- `TYA_LIB_DIR` becomes the primary environment variable for overriding the bundled standard-library source directory.
- `TYA_STDLIB_DIR` remains accepted as a deprecated compatibility alias during this migration.
- If both `TYA_LIB_DIR` and `TYA_STDLIB_DIR` are set, `TYA_LIB_DIR` takes precedence.
- CLI, runner, checker, formatter, docs, examples, tests, release packaging, and generated-doc commands use `lib/` as the current source path.
- The user-facing concept and prose may continue to say "standard library".
- Current docs and generated API docs move from `docs/stdlib` to `docs/lib`.
- Public docs links and navigation use `/lib/` for the generated standard-library API reference.
- `tya doc --json lib` and `tya doc --html docs/lib lib` become the canonical generated-doc commands.
- Existing current docs, README content, scripts, tests, workflows, and feature specs are updated from `stdlib/` path references to `lib/` where they describe the active source layout.
- Historical version specs and completed feature specs are also mechanically updated when they contain concrete active repository paths such as `stdlib/foo.tya`, so repository-wide path search does not keep pointing users at the old source directory.
- Generated API docs no longer emit item IDs, source paths, or links prefixed by `stdlib/`; they use `lib/`.

## Scope
- Rename the directory tree `stdlib/` to `lib/`.
- Update Go code that discovers or embeds the standard-library directory.
- Update import resolution and package loading defaults from `stdlib` to `lib`.
- Add `TYA_LIB_DIR` support and keep `TYA_STDLIB_DIR` as a deprecated fallback.
- Update release packaging scripts from `share/tya/stdlib` to `share/tya/lib`.
- Update tests and txtar fixtures that set `TYA_STDLIB_DIR` or refer to `stdlib/`.
- Update docs, README, handoff docs, examples, scripts, GitHub workflows, and generated-doc commands.
- Regenerate generated API docs under `docs/lib`.
- Remove or replace generated API docs under `docs/stdlib`.
- Update site navigation and links to point to `/lib/`.
- Update feature specs and historical docs with concrete `lib/` repository paths where keeping the old path would be misleading.
- Add compatibility tests for `TYA_LIB_DIR`, `TYA_STDLIB_DIR`, and precedence when both are set.

## Out of Scope
- No rename of the concept "standard library" to "lib" in prose.
- No change to import path syntax or public package names solely because the source directory changes.
- No change to standard-library APIs.
- No recursive redesign of package/module semantics beyond path lookup updates.
- No permanent support for installing both `share/tya/stdlib` and `share/tya/lib` as equal canonical locations.

## Acceptance Criteria
- The repository contains `lib/` and no longer contains the active source directory `stdlib/`.
- Standard-library imports still work from user programs after the rename.
- `TYA_LIB_DIR` overrides the default bundled standard-library directory.
- `TYA_STDLIB_DIR` still works as a deprecated fallback when `TYA_LIB_DIR` is unset.
- `TYA_LIB_DIR` wins when both environment variables are set.
- Release package scripts install bundled standard-library sources under `share/tya/lib`.
- Current docs and README use `lib/` for source paths and generated-doc commands.
- Generated API docs are under `docs/lib` and use `lib/` in source paths and item IDs.
- Site navigation points to `/lib/` instead of `/stdlib/`.
- Tests and fixtures no longer assume `stdlib/` as the active source directory.
- Full repo search for active-source path references does not leave misleading `stdlib/` paths outside intentionally historical prose.

## Verification
```sh
go test ./internal/runner ./internal/doc ./tests -count=1
go run ./cmd/tya doc --json lib
go run ./cmd/tya doc --html docs/lib lib
mise exec ruby@3.4 -- bundle exec jekyll build --source docs --destination _site
go test ./... -count=1
```
