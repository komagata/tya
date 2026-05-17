# Feature: Release, Dependency, Native, And Security Semantics

## Goal

Make project configuration, dependency fetching, lockfile integrity, offline
mode, native package boundaries, environment access, build artifacts, cleanup,
version reporting, archived docs, security boundaries, license metadata, and
package publishing policy deterministic and testable.

## Context

This spec records accepted operational behavior for the current Tya toolchain.
It is not a static typing plan. It is intended for a future implementation pass
and therefore includes the tests that should be added with that implementation.

The behavior here complements:

- `feature-specs/unambiguous-dynamic-semantics.md`
- `feature-specs/dynamic-edge-semantics.md`
- `feature-specs/numeric-call-and-canonical-semantics.md`
- `feature-specs/control-display-and-platform-semantics.md`
- `feature-specs/package-class-concurrency-tooling-semantics.md`

Self-host compatibility is required. Any implementation must preserve the v01
and v02 self-host fixed point.

## Behavior

- Unknown keys in `tya.toml` are errors.
- `tya.toml` typos must not be silently ignored.
- `tya format` formats only `.tya` source files.
- `tya format` does not rewrite `tya.toml`.
- Only `tya install` performs dependency network access by default.
- `tya check`, `tya run`, `tya build`, and `tya test` do not fetch missing
  dependencies automatically.
- If a required locked dependency is unavailable locally, non-install commands
  fail with an explicit error.
- Package dependencies use explicit git URLs or explicit local paths.
- Tya does not support arbitrary script download, registry magic, or implicit
  package source discovery.
- `tya.lock` records the resolved revision and content hash for each
  dependency.
- If fetched or cached dependency content does not match the lockfile hash, the
  command fails.
- Project-local `.tya/packages/` is the semantic dependency cache.
- A global cache may be used as an optimization, but it must not change build
  meaning.
- `--offline` is available for `install`, `check`, `build`, and `test`.
- Offline mode forbids network access.
- Offline mode uses only lockfile data and local caches.
- Offline mode fails if a required dependency is missing locally.
- Native packages use declarative metadata only.
- Native package metadata may declare `pkg_config`, `sources`, `headers`, and
  `libs`.
- Native packages may not run arbitrary shell build scripts.
- Tya does not provide an OS sandbox for native packages.
- Native packages build and link arbitrary C/native code and must be trusted.
- The native runtime ABI has a version.
- Native packages declare the ABI version range they support.
- A native package with an incompatible ABI range fails at build time.
- Environment variables are read only through explicit calls such as
  `env(name)` or documented standard-library APIs.
- Compiler/checker language semantics depend on as few environment variables as
  possible.
- Any environment variable that changes compiler/checker/tool semantics must be
  listed in `docs/SPEC.md`.
- `.env` files are not loaded automatically.
- Users or task-runner commands may load `.env` explicitly before invoking Tya.
- `tya build -o PATH` writes the executable to `PATH`.
- Without `-o`, `tya build` writes an executable in the current directory using
  the source basename.
- Intermediate build artifacts live under `.tya/build/`.
- `tya clean` removes `.tya/build/`.
- `tya clean` does not remove `.tya/packages/`.
- `tya clean --packages` removes `.tya/packages/`.
- Tya releases use SemVer.
- After v1.0, breaking changes require a major version bump.
- Before v1.0, breaking changes are allowed only when release notes explicitly
  mark them as breaking.
- The compiler reports the SPEC version it supports.
- `tya version --json` reports compiler, runtime, SPEC, and self-host versions.
- `docs/vX.Y/` and `docs/archive/` are historical documents.
- Active SPEC changes do not rewrite frozen version docs or archived planning
  docs.
- `tya run`, `tya build`, and `tya test` execute or build user code and are
  outside the safe-analysis trust boundary.
- Safe analysis uses `tya check` and `tya doc`.
- Package manifests require `license`.
- Dependency licenses are enumerable through `tya install` metadata and
  `tya doc --json`.
- There is no central package registry.
- There is no `tya publish` command.
- Packages are consumed by git URL plus lockfile data.

## Scope

- Update `docs/SPEC.md` with these accepted operational semantics.
- Update package manifest parsing and validation for unknown keys and required
  license metadata.
- Update dependency installer and resolver behavior for explicit dependencies,
  lockfile revision/hash integrity, project-local cache semantics, and offline
  mode.
- Update CLI behavior for network boundaries, build output defaults,
  `.tya/build/`, `tya clean`, `tya clean --packages`, `tya version --json`, and
  absent `tya publish`.
- Update native package handling for declarative metadata and ABI compatibility.
- Update docs for security boundaries, trusted native packages, environment
  variables, archived docs, SemVer, and package distribution policy.
- Add focused unit tests and testscript coverage listed below.

## Out of Scope

- Central package registry.
- `tya publish`.
- Arbitrary native build scripts.
- Automatic `.env` loading.
- OS sandboxing for user code or native packages.
- Changing archived docs under `docs/vX.Y/` or `docs/archive/`.
- Making non-install commands fetch dependencies automatically.

## Acceptance Criteria

- Unknown `tya.toml` keys fail validation.
- `tya format` leaves `tya.toml` byte-for-byte unchanged.
- `tya install` may fetch dependencies; `check`, `run`, `build`, and `test`
  never fetch automatically.
- Dependencies must be explicit git URLs or explicit local paths.
- `tya.lock` includes resolved revision and content hash.
- Hash mismatch fails.
- `.tya/packages/` controls dependency meaning; global cache is only an
  optimization.
- `--offline` forbids network access and fails on missing local dependencies.
- Native package shell build scripts are rejected.
- Native package declarative metadata is accepted.
- Native package ABI mismatch fails at build time.
- `.env` is not loaded automatically.
- Environment variables that affect tool semantics are documented.
- `tya build -o app` writes `app`.
- `tya build main.tya` writes a basename executable in the current directory.
- Intermediate artifacts are under `.tya/build/`.
- `tya clean` removes build artifacts only.
- `tya clean --packages` removes dependency cache.
- Release and SPEC versions are visible through `tya version --json`.
- Frozen docs and archives are not modified by active SPEC updates.
- Security docs distinguish safe analysis commands from code execution/build
  commands.
- `license` is required in `tya.toml`.
- Dependency license metadata can be listed.
- `tya publish` is unavailable and central registry behavior is absent.

## Tests To Add

Manifest tests:

- `TestManifestRejectsUnknownKeys`
  - `tya.toml` with an unknown top-level key fails.
- `TestManifestRequiresLicense`
  - Package manifest without `license` fails validation.
- `TestManifestAcceptsExplicitDependenciesOnly`
  - Git URL and local path dependencies pass; registry-style or implicit
    dependency sources fail.

Formatter tests:

- `TestFormatDoesNotRewriteTyaToml`
  - Running `tya format` over a project leaves `tya.toml` unchanged.

Dependency tests:

- `TestInstallFetchesDependencies`
  - `tya install` is the command that populates `.tya/packages/`.
- `TestNonInstallCommandsDoNotFetchDependencies`
  - `tya check`, `run`, `build`, and `test` fail when required dependencies
    are missing locally.
- `TestLockfileRecordsRevisionAndHash`
  - `tya.lock` contains resolved revision and content hash.
- `TestDependencyHashMismatchFails`
  - Tampered cached dependency fails with a hash mismatch diagnostic.
- `TestProjectLocalPackageCacheDefinesMeaning`
  - `.tya/packages/` controls the dependency used by build/check.
- `TestOfflineModeUsesOnlyLocalData`
  - `--offline` succeeds with cache present and fails when cache is missing.
- `TestGlobalCacheDoesNotChangeBuildMeaning`
  - Global cache presence does not override project-local lock/cache behavior.

Native package tests:

- `TestNativePackageRejectsBuildScripts`
  - Manifest build script fields fail.
- `TestNativePackageAcceptsDeclarativeMetadata`
  - `pkg_config`, `sources`, `headers`, and `libs` metadata pass.
- `TestNativePackageAbiRange`
  - Compatible ABI builds and incompatible ABI fails before compile/link.
- `TestNativePackageTrustBoundaryDocs`
  - Documentation or generated metadata states native packages are trusted code.

Environment tests:

- `TestEnvReadRequiresExplicitCall`
  - Program reads env only through `env(name)` or documented API.
- `TestDotEnvNotAutoLoaded`
  - A `.env` file in the project does not affect `env(name)` unless the parent
    process loads it.
- `TestDocumentedToolEnvironmentVariables`
  - Environment variables that change tool semantics are listed in `docs/SPEC.md`.

CLI artifact tests:

- `TestBuildOutputExplicitO`
  - `tya build -o out/app main.tya` writes `out/app`.
- `TestBuildOutputDefaultBasename`
  - `tya build src/main.tya` writes `main` or platform executable equivalent
    in the current directory.
- `TestBuildIntermediateArtifactsUnderTyaBuild`
  - Intermediate C/object files go under `.tya/build/`.
- `TestCleanRemovesBuildOnly`
  - `tya clean` removes `.tya/build/` and keeps `.tya/packages/`.
- `TestCleanPackagesRemovesDependencyCache`
  - `tya clean --packages` removes `.tya/packages/`.
- `TestVersionJsonIncludesVersions`
  - `tya version --json` includes compiler, runtime, SPEC, and self-host
    versions.
- `TestPublishCommandUnavailable`
  - `tya publish` reports unsupported/unknown command.

Release/docs/security tests:

- `TestSemverReleaseMetadata`
  - Release metadata follows SemVer shape.
- `TestArchivedDocsRemainHistorical`
  - Active-doc update commands do not rewrite `docs/vX.Y/` or `docs/archive/`.
- `TestSecurityBoundaryDocs`
  - Docs distinguish `check`/`doc` safe analysis from `run`/`build`/`test`
    code execution or native build boundaries.
- `TestDependencyLicenseListing`
  - Dependency licenses are available from install metadata and `tya doc --json`.

Testscript coverage:

- Add CLI fixtures for manifest validation, install/check/build/test network
  boundaries, offline mode, hash mismatch, build output paths, clean behavior,
  version JSON, and missing `publish`.
- Add native-package fixtures for declarative metadata, rejected build scripts,
  and ABI mismatch.
- Add documentation fixtures for security boundary text, environment variable
  listing, archived-doc immutability, and license listing.
- Include platform-specific executable suffix handling where needed.

## Verification

```sh
gofmt -w internal/manifest internal/resolver internal/package internal/native internal/doc cmd tests
go test ./internal/manifest ./internal/resolver ./internal/package ./internal/native ./internal/doc -count=1
go test ./tests -run 'TestPackage|TestNative|TestTask|TestTool|TestDocs|TestVersion' -count=1 -timeout=20m
go test ./tests -run 'TestExamplesGolden|TestV02Scripts|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
