# Feature: Homepage Standard Library API Docs

## Goal

Publish human-readable standard-library API documentation on the Tya homepage at `/stdlib/`, generated from stdlib source comments through `tya doc`, and commit the generated pages so GitHub Pages can serve them with the existing Jekyll site.

## Context

`tya doc` already extracts source-comment API documentation and supports text, JSON, and HTML output. Current docs mention:

```sh
tya doc --json stdlib
tya doc --html ./out
```

`docs/SPEC.md` states that generated stdlib API documentation is produced from comments with `tya doc`, and `internal/doc` has stdlib documentation coverage tests.

The public website is built from Jekyll sources under `docs/`:

```sh
bundle exec jekyll build --source docs --destination _site
```

The homepage currently links to Guide and Spec, but does not expose a browsable standard-library API reference. This feature should make the generated stdlib API docs part of the public website instead of only a local CLI artifact.

## Behavior

- The website exposes stdlib API documentation at `/stdlib/`.
- The stdlib API docs are human-readable HTML pages suitable for browsing from the homepage.
- The docs are generated from public comments in `stdlib/` through the `tya doc` pipeline, not handwritten as a separate API reference.
- Generated stdlib docs are committed under `docs/stdlib/` so GitHub Pages can publish them without requiring the Pages workflow to run `tya doc`.
- The stdlib docs include public standard-library classes, interfaces, methods, static methods, and constants that `tya doc` can extract.
- The stdlib docs should preserve stable ordering so repeated generation produces deterministic diffs.
- The docs should include enough package/class context for users to find import paths such as `json/Json`, `file/File`, `net/http/Server`, and `net/ip/Address`.
- The homepage and documentation navigation should link to `/stdlib/`.
- The generated pages should use the same broad visual language as the existing docs site, or be wrapped/adapted so they do not look like an unrelated site.
- Missing stdlib doc comments should remain a test failure through existing or expanded coverage.
- Documentation diagnostics from `tya doc` must be visible during generation and should fail the generation command when they are errors.

## Scope

- `tya doc` HTML output or a small stdlib-doc generation wrapper if needed to target Jekyll-friendly output under `docs/stdlib/`.
- Generated/committed files under `docs/stdlib/`.
- Homepage link updates in `docs/index.html`.
- Documentation navigation updates in `docs/_includes/nav.html` and `docs/_data/i18n.yml` if needed.
- Docs build or verification scripts only if needed to make regeneration repeatable.
- Tests for generated stdlib API docs, deterministic output, and homepage/navigation links.
- Existing stdlib source doc comments only where generation exposes missing or unusable public API documentation.

## Out of Scope

- Replacing `tya doc` with a new documentation engine.
- Publishing generated docs outside the existing GitHub Pages/Jekyll site.
- Adding search, client-side JavaScript, or versioned API browsing.
- Changing public stdlib APIs.
- Rewriting all stdlib comments for style beyond fixing missing or clearly broken generated documentation.
- Requiring GitHub Pages workflow to run `tya doc` before Jekyll build.
- Localizing the generated stdlib API reference into Japanese in this feature.

## Acceptance Criteria

- `docs/stdlib/index.html` or an equivalent Jekyll-served entry page exists and is committed.
- The public URL `/stdlib/` is the canonical stdlib API docs entry point.
- The homepage links to `/stdlib/`.
- The shared docs navigation links to `/stdlib/` for English pages.
- The stdlib docs include representative APIs including `Json.parse`, `File.read`, `Math.abs`, `Template.render`, `Server.get`, and `Address.parse`.
- Generated stdlib API pages include source-derived signatures and rendered doc comments.
- Running the documented regeneration command twice with no source changes produces no meaningful diff.
- Jekyll builds the website with the committed stdlib docs.
- Existing `tya doc` JSON/text/HTML behavior for user projects remains backward compatible.
- Full repository verification still passes, including the self-host invariant.

## Verification

```sh
gofmt -w <changed-go-files>
go test ./internal/doc -count=1
go test ./tests -run 'TestDocs|TestV51Scripts|TestSelfhostV01Scripts' -count=1
go test ./... -count=1
bundle exec jekyll build --source docs --destination _site
```

Regeneration smoke check:

```sh
tya doc --html docs/stdlib stdlib
git diff -- docs/stdlib
```
