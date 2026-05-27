# Feature: Spec Page Organization

## Goal

Reorganize the homepage specification page so it is concise, normative,
complete for the current v1.0.0 surface, and ordered for readers who need the
language contract rather than a tutorial.

## Context

`docs/SPEC.md` is the public specification page for Tya. It currently contains
the full language, tooling, package, and standard-library surface, but its
ordering and level of explanation mix specification and guide-style material.
`docs/GUIDE.md` remains the right place for learning-oriented explanation.

The accepted direction is to keep `docs/SPEC.md` as the single concise public
contract for the language, imports/packages, CLI/tooling, and standard-library
boundaries. Fine-grained standard-library method references should live in
generated API docs where possible. Completed feature specs remain design
history, not required reading for v1.0.0 users.

## Behavior

- `docs/SPEC.md` remains one public specification page covering:
  - the language syntax and semantics;
  - import and package behavior;
  - CLI/tool behavior;
  - public standard-library boundaries;
  - release, distribution, and system-level guarantees.
- The page is reorganized in this order:
  - Overview;
  - v1.0.0 Compatibility Boundary;
  - Source and Lexical Structure;
  - Values and Kinds;
  - Declarations and Scope;
  - Expressions;
  - Statements;
  - Imports and Packages;
  - Runtime and Concurrency;
  - Errors and Diagnostics;
  - Built-In Tools;
  - Standard Library;
  - Distribution and System Considerations.
- The specification is written as normative rules.
  - Keep short examples only where they disambiguate syntax or behavior.
  - Move tutorial-style explanations, walkthroughs, and longer examples to
    `docs/GUIDE.md` if they are still useful.
- Standard-library coverage in `docs/SPEC.md` is concise.
  - SPEC lists public stdlib packages and their stability boundaries.
  - Detailed method inventories and signatures should be verified through
    generated API docs such as `tya doc --json lib`.
  - v1.0.0 stdlib blocker APIs remain explicitly called out:
    `regex/Regex`, filesystem utilities, `time/Time`,
    environment/process APIs, and `hmac/Hmac`.
- `ROADMAP.md` must not contradict the accepted v1.0.0 decisions.
  - Old "decide ..." blocker wording is replaced with the accepted decision.
  - ROADMAP continues to track implementation work, not alternate
    specification authority.
- Completed feature specs remain design history.
  - v1.0.0 public authority is `docs/SPEC.md`,
    `docs/STRICT_SEMANTICS.md`, and frozen `docs/v1.0/*` documents.
  - Readers should not need to inspect `feature-specs/completed/` to know the
    current v1.0.0 language contract.
- The v1.0.0 release checklist lives outside SPEC.
  - Add `docs/v1.0/RELEASE_CHECKLIST.md`.
  - The checklist records release pass/fail gates; `docs/SPEC.md` records the
    language and tool contract.

## Scope

- `docs/SPEC.md`
- `docs/GUIDE.md` only if guide-style material is moved there
- `docs/STRICT_SEMANTICS.md` only for cross-reference alignment
- `docs/v1.0/RELEASE_CHECKLIST.md`
- `ROADMAP.md`
- documentation contract tests under `tests/`

## Out of Scope

- Changing language behavior.
- Changing stdlib APIs.
- Rewriting completed feature specs.
- Making generated stdlib API docs part of this feature beyond linking or
  referencing them.
- Implementing release automation beyond documenting the release checklist.

## Acceptance Criteria

- `docs/SPEC.md` follows the accepted section order.
- `docs/SPEC.md` is concise and normative, with guide-style prose reduced or
  moved to `docs/GUIDE.md`.
- `docs/SPEC.md` still covers the current language, import/package, tool,
  stdlib, distribution, and system surface.
- `docs/SPEC.md` explicitly states that generated stdlib API docs provide
  detailed package/member references.
- `ROADMAP.md` no longer presents already accepted v1.0.0 decisions as open
  decisions.
- Completed feature specs are clearly treated as design history, while
  `docs/SPEC.md`, `docs/STRICT_SEMANTICS.md`, and `docs/v1.0/*` are the public
  authority.
- `docs/v1.0/RELEASE_CHECKLIST.md` exists and records release gates separately
  from the specification.
- Existing documentation links and homepage navigation continue to resolve.

## Tests To Add

Documentation tests:

- `TestSpecUsesAcceptedSectionOrder`
  - Reads `docs/SPEC.md`.
  - Expected: top-level headings appear in the accepted order.

- `TestSpecDocumentsPublicAuthority`
  - Expected: `docs/SPEC.md` states that completed feature specs are design
    history and that public v1 authority is SPEC, STRICT_SEMANTICS, and frozen
    v1.0 docs.

- `TestSpecReferencesGeneratedStdlibDocs`
  - Expected: `docs/SPEC.md` points detailed stdlib API references to generated
    stdlib docs such as `tya doc --json lib`.

- `TestRoadmapHasNoOpenV1DecisionContradictions`
  - Expected: `ROADMAP.md` does not contain open-decision wording for v1.0.0
    decisions already accepted in completed specs.

- `TestV10ReleaseChecklistExists`
  - Expected: `docs/v1.0/RELEASE_CHECKLIST.md` exists and includes release
    gates for strict semantics, latest self-host fixed point, no-Go bootstrap,
    structured diagnostics, stdlib blockers, frozen docs, release artifacts,
    and package-manager behavior.

Link tests:

- `TestSpecInternalLinksResolve`
  - Expected: important local links from `docs/SPEC.md` resolve, including
    `docs/STRICT_SEMANTICS.md`, generated-doc references where applicable, and
    frozen v1.0 docs.

## Verification

```sh
go test ./tests -run 'TestSpecUsesAcceptedSectionOrder|TestSpecDocumentsPublicAuthority|TestSpecReferencesGeneratedStdlibDocs|TestRoadmapHasNoOpenV1DecisionContradictions|TestV10ReleaseChecklistExists|TestSpecInternalLinksResolve' -count=1
go test ./tests -run 'TestSpecDocuments|TestV.*Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
