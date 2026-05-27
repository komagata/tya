# Feature: Stdlib API Doc Comments

## Goal

Make the bundled standard library self-documenting by adding source comments to public stdlib APIs and generating user-facing API documentation from those comments.

## Context

- Tya's public stdlib lives under `lib/` and is imported with the same syntax as user packages.
- Current stdlib source files mostly expose public classes, interfaces, and static methods without leading documentation comments.
- `tya doc` already extracts leading `#` comment blocks for top-level documented bindings.
- A separate queued spec, `feature-specs/documentation-generator-extensions.md`, covers `tya doc --json`, re-export following, and documentation diagnostics.
- This feature focuses on stdlib source content and generated API docs, not on redesigning the documentation generator.

## Behavior

- Add leading `#` doc comments to public stdlib APIs.
- Public API coverage includes:
  - public class files under `lib/**/PascalCase.tya`
  - public interfaces under `lib/`
  - public static methods intended for users
  - public constructors such as `new`, `parse`, `read`, `write`, `connect`, and similar entry points
- Comments should be concise and useful:
  - first sentence describes what the API does
  - parameters are explained when the names are not self-evident
  - return shape is described for dictionaries, arrays, bytes, domain objects, and error/result conventions
  - side effects are stated for filesystem, network, process, task, logging, synchronization, random, and runtime APIs
  - errors/raises are documented when a method validates inputs or can fail due to I/O/runtime conditions
  - examples are added for complex packages where a short usage snippet prevents ambiguity
- Do not document private helper methods as public API.
  - Private helper methods inside public classes may remain undocumented.
  - If a helper is currently public only because Tya lacks private static methods, add a comment that marks it as internal only when needed by the doc generator.
- Generated stdlib API docs must be produced from source comments.
  - The generated output should include class/interface names, method signatures, source paths, and rendered comments.
  - The docs should be suitable for publication under the project website.
- Establish a coverage gate.
  - A test or script should fail when public stdlib classes/interfaces or public user-facing methods lack doc comments.
  - The gate may allow an explicit internal allowlist for helper methods that are intentionally omitted from public docs.
- Keep English as the source-code comment language for stdlib API comments.
  - Japanese explanatory docs may link to the generated API reference, but source comments do not need to be duplicated in Japanese.

## Scope

- Add or update comments in stdlib source files under `lib/`.
- Add a stdlib API documentation generation command or script using `tya doc` or the enhanced documentation generator.
- Add generated API docs source/output under `docs/` only if the repository already treats generated docs as committed publication content.
- Add coverage tests for missing stdlib API comments.
- Update docs:
  - `docs/SPEC.md`
  - `docs/ja/spec.md`
  - `docs/GUIDE.md` or README if they should point users to the generated stdlib API reference
- Coordinate with `feature-specs/documentation-generator-extensions.md`:
  - if doc-generator extensions land first, use their JSON/re-export support
  - if this lands first, add comments and a minimal generation path without depending on unfinished features

## Out of Scope

- Changing stdlib API behavior.
- Renaming stdlib classes, methods, packages, or import paths.
- Rewriting implementation code for style while adding comments.
- Writing long tutorials for every package.
- Translating every stdlib API comment into Japanese.
- Publishing a website deploy as part of implementation.

## Acceptance Criteria

- Every public stdlib class and interface has a leading doc comment.
- Every public user-facing stdlib static method has either a leading doc comment or is covered by an explicit internal/helper allowlist.
- Generated stdlib API documentation includes at least:
  - package/path
  - class or interface name
  - method signatures
  - rendered comments
  - source path and line
- Generated docs include representative entries for core packages:
  - `math`
  - `file`
  - `json`
  - `toml`
  - `net/http`
  - `net/socket`
  - `template`
  - `unittest`
- A verification test fails if a new public stdlib API is added without documentation.
- Existing stdlib behavior tests remain unchanged.
- English and Japanese specs mention that stdlib API documentation is generated from source comments.

## Verification

```sh
go test ./internal/doc -count=1
go test ./tests -run 'TestV51Scripts|TestStdlib' -count=1
go test ./... -count=1
```
