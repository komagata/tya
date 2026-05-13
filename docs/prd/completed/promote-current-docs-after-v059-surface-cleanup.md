---
status: completed
goal_ready: false
---

# Feature: Promote Current Docs After The v0.59 Surface Cleanup

## Goal

Bring the editable documentation set up to the current post-v0.59 language and
stdlib surface, then rebuild the generated HTML docs so readers no longer see
module-era or removed primitive-helper-module guidance in current docs.

## Context

The current editable docs are the active documentation surface:

- `docs/SPEC.md`
- `docs/API.md`
- `docs/STDLIB.md`
- `docs/NAMING.md`
- `docs/CANONICAL_SYNTAX.md`
- `docs/GUIDE.md`
- `docs/TERMINOLOGY.md`
- `docs/LIBRARIES.md`

Older frozen version docs under `docs/vX.Y/` are historical and should not be
rewritten by this cleanup. The cleanup follows the v0.59 surface direction:
primitive helper APIs live as primitive methods, `x.class` is the class
introspection surface, and the removed `string`, `array`, and `dict` stdlib
modules must not be presented as current public APIs.

## Behavior

- Audit the editable docs for stale module-era language.
- Update current docs to describe module-free source and class-oriented package
  behavior consistently.
- Preserve the term "module" only where it is still intentionally used by the
  current language model, such as importable source units or standard library
  modules.
- Remove or rewrite examples that still show the retired top-level `module`
  keyword as current syntax.
- Keep `docs/archive/pre-v0.1/` and frozen `docs/vX.Y/` docs unchanged.
- Keep primitive method APIs consistent across docs.
- Document `x.class` consistently where primitive class introspection is
  discussed.
- Ensure removed `string`, `array`, and `dict` modules are not documented as
  current importable stdlib modules.
- Keep `toml` documented as class-style `toml.Toml.parse` and
  `toml.Toml.dump`.
- Keep `docs/VERSIONS.md` as the release-history index, not `ROADMAP.md`.
- Rebuild generated HTML docs with:

  ```sh
  node scripts/build_docs_pages.js
  ```

- Generated HTML should reflect the edited Markdown.

## Scope

- Editable Markdown docs:
  - `docs/SPEC.md`
  - `docs/API.md`
  - `docs/STDLIB.md`
  - `docs/NAMING.md`
  - `docs/CANONICAL_SYNTAX.md`
  - `docs/GUIDE.md`
  - `docs/TERMINOLOGY.md`
  - `docs/LIBRARIES.md`
- Generated HTML docs produced by `scripts/build_docs_pages.js`.
- Minimal examples inside docs needed to keep current syntax accurate.
- `ROADMAP.md`.

## Out of Scope

- Changing frozen release docs under `docs/vX.Y/`.
- Changing archived planning docs under `docs/archive/pre-v0.1/`.
- Implementing language, compiler, stdlib, or runtime behavior.
- Renaming current import/module terminology if it still reflects active import
  resolution.
- Broad editorial rewrites unrelated to stale surface cleanup.
- Updating release history beyond references needed for consistency.

## Acceptance Criteria

- Current editable docs no longer present the retired `module` keyword as valid
  current syntax.
- Current editable docs consistently describe primitive methods instead of
  removed `string`, `array`, and `dict` modules.
- `x.class` is documented consistently where class introspection is covered.
- `toml.Toml.parse` and `toml.Toml.dump` remain the documented public TOML API.
- `docs/NAMING.md` no longer says a module file defines a top-level `module`.
- `docs/CANONICAL_SYNTAX.md` examples and rules no longer rely on the retired
  `module` declaration.
- `docs/GUIDE.md`, `docs/TERMINOLOGY.md`, and `docs/LIBRARIES.md` use current
  import/package terminology without implying the retired keyword is required.
- `node scripts/build_docs_pages.js` completes successfully.
- Generated HTML files corresponding to edited Markdown are updated.
- The self-host fixed-point invariant is not affected.

## Verification

Documentation build:

```sh
node scripts/build_docs_pages.js
```

Focused stale-surface scan:

```sh
rg -n '(^|[^a-z_])module [a-z_]|import string|import array|import dict|string module|array module|dict module' docs/SPEC.md docs/API.md docs/STDLIB.md docs/NAMING.md docs/CANONICAL_SYNTAX.md docs/GUIDE.md docs/TERMINOLOGY.md docs/LIBRARIES.md
```

Self-host invariant if implementation files are touched unexpectedly:

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
```

## Dependencies

- Use `ROADMAP.md` only as active planning context.
- Treat frozen version docs and archived docs as historical references.
- Keep changes surgical and tied to current-doc consistency.

## Open Questions

None.
