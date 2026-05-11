# Tya Handoff Notes

> A snapshot for another Claude Code (or human) picking up Tya work.
> The authoritative documents are still `CLAUDE.md`, `AGENTS.md`,
> and `ROADMAP.md` — this file is a session-bridging summary
> covering recent context and pending decisions that aren't
> obvious from the codebase alone.

## Current state (2026-05-11)

- **Latest release: v0.48.0** (tagged, GitHub Release published,
  Homebrew tap `komagata/tap` updated).
- `cmd/tya/main.go` → `const version = "0.48.0"`.
- `git status` is clean on `main`; `main` matches `origin/main`.
- `go test ./... -count=1` is green, including both self-host
  gates (`TestSelfhostV01Scripts`, `TestSelfhostV02Scripts`).

End-user install:

```sh
brew install komagata/tap/tya
```

## What just shipped (v0.46 → v0.48)

The class-member surface was rebuilt across three minors:

| Release | Theme                                                | Key artifact                          |
| ------- | ---------------------------------------------------- | ------------------------------------- |
| v0.46   | Sigil-free keyword surface (additive)                | `private`, `static`, `self`, `Self`, `initialize` reserved |
| v0.47   | Clean cut of legacy v0.45 surface                    | [TYA-E0407] [TYA-E0410] [TYA-E0411] [TYA-E0414] |
| v0.48   | Canonical `Self.foo` rule + formatter rewrite        | [TYA-E0413] strict warning + formatter rewrites |

Frozen SPECs: `docs/v0.46/SPEC.md`, `docs/v0.47/SPEC.md`,
`docs/v0.48/SPEC.md`. Release notes alongside each.

### Selfhost / legacy code interop

`selfhost/v01/compiler.tya` is **frozen at the v0.43 surface**
(uses `@`, `@@`, `_`-prefix, `init`). The Go reference checker
exempts it via:

```go
defer checker.SetPermissiveLegacy(runner.IsLegacyV01Path(path))()
```

This is wired in `RunFile`, `compileToCWithCover`, `checkFile`,
and the developer-flag main path. **Don't remove the exemption
until M8/v02 supersedes v01.**

## What's next (`ROADMAP.md` § Scheduled)

The v0.4x Epic is the only remaining structural work before
v1.0.0:

1. **M8** — grow `selfhost/v02/compiler.tya` to the v0.46+ surface
   and prove its stage-2/stage-3 fixed point. Scaffolding (M8.0–
   M8.2d) is landed: v02 mirrors v01 byte-for-byte, the test gate
   `TestSelfhostV02Scripts` is green, parser accepts `@@` single
   token + `abstract`/`final`/`override` modifiers. The remaining
   M8 work is genuinely 5–7 sessions of careful Tya-in-Tya
   programming; do not try to land it in one sitting.
2. **M6 remaining** — migrate `string`/`array`/`dict` stdlib
   packages to class form (blocked on M8 because v01 still imports
   them as single-file modules).
3. **M9** — remove the `module` keyword (`[TYA-E0200]`), retire
   v01, drop module code paths from parser/checker/formatter/
   codegen.
4. **M10** — promote `docs/v0.44/SPEC.md` content to
   `docs/SPEC.md`, rewrite `docs/NAMING.md`, etc.

The "Future Work" section of `ROADMAP.md` lists everything else
on the v1.0.0 horizon (LSP, doc generator, new project
scaffolder, lint, WASM target, etc.).

## Process notes from recent sessions

These captured corrections from past sessions — read before
inventing your own deferral plan.

### Don't unilaterally defer SPEC items

When a SPEC says "clean cut, no deprecation window," it means
clean cut. If a STEP's scope feels too big to finish in one
session, **don't relabel the SPEC to call the release
"transitional" and push the rest to a follow-up minor.** Ask
the user explicitly; otherwise treat the SPEC as a contract.

This came up between v0.46 and v0.47: v0.46.0 shipped without
clean cut because I unilaterally deferred. v0.47.0 then
implemented the actual SPEC promise. The user explicitly flagged
the deferral as unacceptable. The fix landed as v0.47/v0.48 but
the better path would have been to honor v0.46's SPEC the first
time.

### Tests must verify SPEC, not weaken assertions

When a test fails after a legitimate semantic change (e.g.
v0.46's `self` reversal), update the assertion to match the new
SPEC, not the old. If a fixture pins legacy-only behavior that
has no new equivalent (e.g. v09/private_members's `_`-prefix
external-access heuristic), retire the fixture rather than
preserving it as a test-only legacy island.

### Path-exempt selfhost/v01 only

The permissive-legacy mechanism in v0.47 is **only** for
`selfhost/v01/`. Don't generalize it to other paths or to a CLI
flag without explicit SPEC support — that creates two language
surfaces and defeats the canonical-syntax invariant.

## Layout reminders

```
.
├── CLAUDE.md, AGENTS.md          # project guidance (auto-loaded)
├── ROADMAP.md                    # single source of truth for plans
├── cmd/tya/                      # CLI entry point
├── internal/
│   ├── ast/                      # AST types
│   ├── lexer/                    # tokens
│   ├── parser/                   # parser
│   ├── checker/                  # type-/scope-/strict checks
│   ├── codegen/                  # C emitter
│   ├── formatter/                # `tya format` / unparser
│   ├── runner/                   # source loading, package synth, RunFile
│   └── eval/                     # tree-walking interpreter (test/tooling)
├── runtime/                      # C runtime linked into emitted programs
├── stdlib/                       # standard library Tya sources
├── selfhost/
│   ├── v01/                      # frozen v0.43-surface Tya compiler
│   └── v02/                      # in-progress v0.46+ surface (M8)
├── tests/                        # CLI/example/spec integration tests
│   └── testdata/
│       ├── v01_selfhost/         # v01 fixed-point gate fixtures
│       ├── v02_selfhost/         # v02 (M8) gate fixtures
│       └── v04…v48/              # per-version regression fixtures
├── docs/                         # frozen spec snapshots + editable docs
└── scripts/                      # build_release_packages.sh, doc builder, etc.
```

## Verification commands

```sh
# Full suite (~8 minutes)
go test ./... -count=1

# Self-host gates only (~2 minutes)
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestSelfhostV02Scripts -count=1

# Focused: code generation through real cc(1) (slow but catches regressions)
sh scripts/go_emit_examples_check.sh
```

When a verification check fails after a refactor, **investigate
the root cause before adjusting the test**. The test corpus is
the SPEC's executable form.

## Release flow

Followed three times in this arc; use the `tya-release` skill
(`/<plugin-prefix>tya-release` or `Skill` tool). The skill walks
through:

1. Bump `cmd/tya/main.go` `const version` + README + pinned
   testdata version strings.
2. Write `docs/v0.<minor>/SPEC.md` (frozen) and
   `docs/v0.<minor>/RELEASE_NOTES.md`.
3. Move v0.<minor> entry from `## Scheduled` to `## Released` in
   `ROADMAP.md`.
4. Rebuild docs HTML via `node scripts/build_docs_pages.js`.
5. `go test ./... -count=1` green, then commit, push, tag.
6. `scripts/build_release_packages.sh <X.Y.Z>` → 5 platform
   tarballs in `dist/`.
7. `gh release create v<X.Y.Z>` with the platform packages.
8. Update `komagata/homebrew-tap` (sibling repo at
   `~/Projects/komagata/homebrew-tap/`): bump url, sha256, and
   the `assert_equal` version in `Formula/tya.rb`.
9. End-user verify: `brew install komagata/tap/tya && tya version`.

## Quick links

- Releases: https://github.com/komagata/tya/releases
- Homepage: https://tya-lang.org/
- Tap: https://github.com/komagata/homebrew-tap
- ROADMAP: [`ROADMAP.md`](../ROADMAP.md)
- Latest SPEC: [`docs/v0.48/SPEC.md`](v0.48/SPEC.md)
