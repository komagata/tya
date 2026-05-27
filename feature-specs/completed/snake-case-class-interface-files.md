# Feature: Snake Case Class and Interface Files

## Goal
Change the class/interface file naming convention so source file names use `snake_case` while the public class or interface declared inside uses the corresponding `PascalCase` name.

## Context
Tya currently treats `PascalCase.tya` files as class files. The filename must match the public class or interface name exactly, and directory packages expose public classes/interfaces from those PascalCase files.

This should change to a Ruby-like convention:

- file name: `snake_case.tya`
- public class/interface name: corresponding `PascalCase`

Examples:

- `base64.tya` declares `class Base64`
- `secure_random.tya` declares `class SecureRandom`
- `wait_group.tya` declares `class WaitGroup`
- acronym-style names use normal snake_case conversion, e.g. `HTTPServer` maps to `http_server.tya`

The user-visible import/API surface does not change. For example, `import base64 as base64` still exposes `base64.Base64`.

## Behavior
- A class/interface file is a `snake_case.tya` file that declares exactly one public class or public interface whose name maps to that filename.
- Public class/interface matching uses `PascalCase` to `snake_case` conversion instead of exact filename equality.
- PascalCase class/interface filenames are no longer supported as class files.
- Directory packages contain snake_case class/interface files and expose their public classes/interfaces directly.
- A script entry may see sibling public classes/interfaces without import when their files use the new snake_case convention.
- Import paths remain slash-separated `snake_case` segments.
- Public API names remain PascalCase class/interface names.
- Additional classes or interfaces in a class/interface file remain private to that file.
- Files that do not satisfy the new filename-to-class/interface mapping should produce clear diagnostics.

## Scope
- Update resolver/package loading for class/interface file discovery.
- Update class/interface public-name matching from exact filename comparison to snake_case filename mapping.
- Rename all stdlib PascalCase class/interface files to snake_case.
- Update stdlib docs generation paths and generated API docs.
- Update tests, fixtures, examples, and docs that reference PascalCase class/interface file paths.
- Update `docs/SPEC.md`, `docs/ja/spec.md`, and current version docs that describe class files, directory packages, importable package classes/interfaces, or PascalCase filenames.
- Update LSP/document symbol or diagnostics behavior if it assumes PascalCase class/interface file paths.
- Update formatter/checker/parser tests as needed when fixtures include class/interface filenames.
- Include a migration note and treat this as a minor-version language/package convention change.

## Out of Scope
- No selective import or per-class import syntax.
- No change to class/interface names themselves; they remain PascalCase.
- No change to import path syntax; import path segments remain snake_case.
- No compatibility fallback for old PascalCase class/interface filenames.
- No runtime API rename such as changing `base64.Base64` to `base64.base64`.

## Acceptance Criteria
- `lib/base64/base64.tya` declares `class Base64` and is accepted as the public class file.
- `lib/secure_random/secure_random.tya` declares `class SecureRandom` and is accepted.
- `lib/sync/wait_group.tya` declares `class WaitGroup` and is accepted.
- A PascalCase class/interface file such as `Base64.tya` is rejected or ignored according to the new class-file rules, and tests cover the behavior.
- Directory package imports continue to expose PascalCase public API names from snake_case files.
- Sibling implicit class/interface visibility works with snake_case filenames.
- The full stdlib test suite passes after all stdlib class/interface files are renamed.
- Generated stdlib API docs no longer reference PascalCase source filenames for renamed files.
- Specs and migration docs describe the breaking filename convention change and minor-version release requirement.

## Verification
```sh
gofmt -w internal/**/*.go cmd/**/*.go tests/**/*.go
go test ./... -count=1
go run ./cmd/tya doc --html docs/lib lib
mise exec ruby@3.4 -- bundle exec jekyll build --source docs --destination _site
```
