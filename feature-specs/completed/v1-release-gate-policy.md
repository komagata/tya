# Feature: v1 Release Gate Policy

## Goal

Freeze the v1.0.0 release-gate policy for self-host verification, implementation
authority, legacy compatibility, release-test speed, existing v1 stdlib
blockers, package-manager scope, diagnostic documentation, and post-v1 syntax
evolution.

## Context

The v1.0.0 language surface is already defined by `docs/SPEC.md`,
`docs/STRICT_SEMANTICS.md`, and completed v1 feature specs. This spec records
the final release-gate policy decisions needed to decide when v1.0.0 is ready
to ship.

The accepted direction is to keep v1.0.0 strict and canonical while avoiding
unnecessary release friction from legacy self-host variants. The latest
self-hosted compiler is the release-critical self-host proof. Legacy self-host
coverage remains useful, but it must not define the v1 public contract.

The v1 stdlib blocker set named in the public contract is already implemented
in the repository: `regex/Regex`, filesystem utilities, `time/Time`,
environment/process APIs, and `hmac/Hmac`. Their v1 requirement is continued
release-gate verification, not future implementation planning.

## Behavior

- v1.0.0 release-critical self-host verification uses the latest self-host
  compiler line.
  - The latest self-host compiler must prove the v1.0.0 fixed point and the
    no-Go bootstrap path required by the release contract.
  - `selfhost/v01` is downgraded to a legacy regression gate.
  - `selfhost/v01` failures may block changes that intentionally preserve v01,
    but v01 does not define v1.0.0 language validity or release readiness once
    the latest self-host gate is green.
- Specification authority order for v1.0.0 is:
  - `docs/SPEC.md` and frozen `docs/v1.0/SPEC.md`;
  - the latest self-host compiler behavior when it implements the documented
    v1 surface;
  - the Go implementation as reference and bootstrap recovery path.
  If the Go implementation and latest self-host compiler disagree, the behavior
  matching the v1 specification is authoritative.
- Legacy syntax and legacy APIs are rejected outside explicitly documented
  legacy paths.
  - `selfhost/v01` may keep legacy compatibility needed for its own fixed
    point.
  - Public v1 programs must receive diagnostics for legacy syntax or APIs that
    are not part of the v1 public surface.
  - Warning-only compatibility is not part of v1.0.0.
- Release verification has two tiers.
  - Normal development and pull-request verification may use a fast self-host
    smoke gate.
  - The v1.0.0 release tag gate must run the full latest self-host fixed point,
    no-Go bootstrap proof, and full repository conformance checks.
- The implemented v1 stdlib blocker set remains release-critical.
  - `regex/Regex`, filesystem utilities, `time/Time`,
    environment/process APIs, and `hmac/Hmac` must stay implemented,
    documented, and tested.
  - Regressions in these APIs block v1.0.0.
  - These APIs are not treated as post-v1 work.
- Package-manager behavior is part of the v1 public contract.
  - Manifest validation, explicit dependencies, lockfiles, local path
    dependencies, git URL dependencies, native package metadata, and dependency
    integrity checks are v1 behavior.
  - A central package registry is outside v1.0.0.
- Diagnostic documentation must make stable codes discoverable.
  - Every public `TYA-E....` diagnostic code must have a short explanation page
    or a generated reference entry.
  - Long tutorial-style documentation for every code is not required for
    v1.0.0.
- v1.x syntax evolution is intentionally conservative.
  - New syntax is not added in ordinary v1.x releases.
  - v1.x may add standard-library APIs, package APIs, tooling improvements,
    diagnostics, and compatible runtime behavior.
  - New syntax requires v2 planning or an explicitly accepted experimental
    feature path outside the stable v1 public contract.

## Scope

- `docs/SPEC.md`
- `docs/STRICT_SEMANTICS.md`
- frozen `docs/v1.0/` documents
- `ROADMAP.md`
- latest self-host fixed-point and no-Go bootstrap scripts
- `selfhost/v01` legacy gate documentation
- release verification scripts and CI configuration
- diagnostic documentation or generated diagnostic reference
- package-manager docs and tests
- stdlib tests for regex, filesystem utilities, time, environment/process, and
  HMAC

## Out of Scope

- Removing `selfhost/v01` immediately.
- Making `selfhost/v01` define v1.0.0 language validity.
- Adding a public conformance command.
- Adding a central package registry for v1.0.0.
- Adding new v1.x syntax as ordinary compatible evolution.
- Reclassifying implemented v1 stdlib blockers as post-v1 work.

## Acceptance Criteria

- `ROADMAP.md` identifies the latest self-host compiler as the v1.0.0
  release-critical self-host gate and `selfhost/v01` as legacy regression
  coverage.
- `docs/SPEC.md` or release documentation states the v1 authority order:
  specification, latest self-host implementation, then Go reference/recovery.
- Public v1 checks reject legacy syntax and APIs outside documented legacy
  paths.
- CI or release scripts distinguish fast development self-host smoke checks
  from full release-tag fixed-point and no-Go bootstrap checks.
- Tests prove `regex/Regex`, filesystem utilities, `time/Time`,
  environment/process APIs, and `hmac/Hmac` remain implemented and documented.
- Package-manager docs and tests cover v1 manifest, lockfile, dependency, and
  native metadata behavior while excluding a central registry.
- Every public `TYA-E....` code appears in a generated or written diagnostic
  reference.
- v1.x language-evolution docs state that new syntax is reserved for v2 or an
  explicit experimental process.

## Tests To Add

Documentation tests:

- `TestRoadmapDocumentsLatestSelfhostReleaseGate`
  - Expected: `ROADMAP.md` marks the latest self-host compiler as the v1.0.0
    release-critical gate and `selfhost/v01` as legacy regression coverage.

- `TestSpecDocumentsV1AuthorityOrder`
  - Expected: docs define SPEC first, latest self-host second, Go
    reference/recovery third.

- `TestSpecDocumentsV1SyntaxEvolutionPolicy`
  - Expected: v1.x docs reject ordinary new syntax additions and point new
    syntax to v2 or an accepted experimental path.

- `TestDiagnosticReferenceCoversPublicCodes`
  - Expected: every public `TYA-E....` code emitted by fixtures appears in the
    diagnostic reference.

Release tests:

- `TestReleaseGateUsesLatestSelfhost`
  - Expected: release verification runs the latest self-host fixed point and
    no-Go bootstrap proof.

- `TestDevelopmentGateCanUseFastSelfhostSmoke`
  - Expected: development verification exposes a fast self-host smoke command
    distinct from the full release gate.

Compatibility tests:

- `TestPublicV1RejectsLegacySyntaxOutsideLegacyPaths`
  - Representative legacy syntax/API snippets are checked outside
    `selfhost/v01`.
  - Expected: stable diagnostics.

- `TestV1StdlibBlockersRemainImplemented`
  - Imports and exercises `regex/Regex`, filesystem utilities, `time/Time`,
    environment/process APIs, and `hmac/Hmac`.
  - Expected: interpreter and generated C behavior pass.

Package tests:

- `TestPackageManagerV1Contract`
  - Covers manifest validation, explicit git/local dependencies, lockfile
    integrity, native metadata, and the absence of central registry publishing.

Testscript coverage:

- `v1_release_gate_policy.txtar`
  - Covers representative release-gate docs, stdlib blocker checks, legacy
    rejection, and package-manager contract behavior.

## Verification

```sh
go test ./tests -run 'TestRoadmapDocumentsLatestSelfhostReleaseGate|TestSpecDocumentsV1AuthorityOrder|TestSpecDocumentsV1SyntaxEvolutionPolicy|TestDiagnosticReferenceCoversPublicCodes' -count=1
go test ./tests -run 'TestReleaseGateUsesLatestSelfhost|TestDevelopmentGateCanUseFastSelfhostSmoke|TestPublicV1RejectsLegacySyntaxOutsideLegacyPaths|TestV1StdlibBlockersRemainImplemented|TestPackageManagerV1Contract' -count=1 -timeout=20m
go test ./tests -run 'TestV.*Scripts|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
