# Tya Handoff Notes

> A snapshot for another Claude Code (or human) picking up Tya work.
> The authoritative documents are still `CLAUDE.md`, `AGENTS.md`,
> and `ROADMAP.md` — this file is a session-bridging summary
> covering recent context and pending decisions that aren't
> obvious from the codebase alone.

## Current state (2026-05-12)

- **Latest release: v0.49.0** (tagged, GitHub Release published,
  Homebrew tap `komagata/tap` updated; `brew install
  komagata/tap/tya` returns 0.49.0 end-to-end).
- `cmd/tya/main.go` → `const version = "0.49.0"`.
- `git status` is clean on `main`; `main` matches `origin/main`.
- `go test ./... -count=1` is green, including both self-host
  gates (`TestSelfhostV01Scripts`, `TestSelfhostV02Scripts`).
- **Self-host work (M8/M9/M10) is deferred to the v1.0.0 prep
  window per user direction.** Tooling-track Epics from `ROADMAP.md`
  § Future Work are the active line of work between now and v1.0.0.

End-user install:

```sh
brew install komagata/tap/tya
```

## What just shipped (v0.46 → v0.49)

The class-member surface was rebuilt across three minors, then the
toolchain track was kicked off:

| Release | Theme                                                | Key artifact                          |
| ------- | ---------------------------------------------------- | ------------------------------------- |
| v0.46   | Sigil-free keyword surface (additive)                | `private`, `static`, `self`, `Self`, `initialize` reserved |
| v0.47   | Clean cut of legacy v0.45 surface                    | [TYA-E0407] [TYA-E0410] [TYA-E0411] [TYA-E0414] |
| v0.48   | Canonical `Self.foo` rule + formatter rewrite        | [TYA-E0413] strict warning + formatter rewrites |
| v0.49   | Toolchain kickoff: `tya new` + `tya task` + `tya lint` | `[tasks]` table + `TYA-E0900..0911` + `TYAL0001` |

Frozen SPECs: `docs/v0.46/SPEC.md`, `docs/v0.47/SPEC.md`,
`docs/v0.48/SPEC.md`, `docs/v0.49/SPEC.md`. Release notes
alongside each.

v0.49 also shipped a build-environment fix: the build driver now
links `-lm` on non-Windows hosts and `runtime/tya_runtime.c`
defines `_XOPEN_SOURCE`/`_DEFAULT_SOURCE` so glibc strict defaults
(Arch Linux) accept the compiled programs. The selfhost fixtures
under `tests/testdata/v0{1,2}_selfhost/` also carry `-lpthread
-lm` on every embedded `cc` invocation. Without these, builds on
Arch Linux fail with `undefined reference to strptime/sin/log2`
etc.

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

## What's next

**Toolchain track** is the active line of work (per user
direction). Candidate Epics in `ROADMAP.md` § Future Work §
Toolchain, ordered roughly by smallest-blast-radius first:

1. **diagnostics pipeline migration** — Parser → `TYA-E0100–
   0299`, Codegen → `TYA-E0600–0799`, Runner → `TYA-E0800–0899`.
   Add did-you-mean + multi-error parsing.
2. **`tya lint` extension** — additional rules on top of v0.49's
   `TYAL0001` (`if true`/dead code/`for` patterns/long functions),
   `--fix` autofix, `--format=json`, per-line opt-out
   (`# tya-lint-ignore: TYAL0001`).
3. **`tya new` extension** — `--here`, `--template app|lib`,
   `--force`, default git init, `tests/` + `README.md`
   boilerplate.
4. **`tya task` extension** — parallel execution syntax,
   `--watch`, dependency graphs, per-task env vars.
5. **`tya doc` source documentation generator**.
6. **`tya lsp` Language Server**.
7. **Public Tya self-introspection library** (`compiler.lexer`,
   `compiler.parser`, etc. as stdlib modules).

**Self-host work is deferred** — M8/M9/M10 from `ROADMAP.md` §
Scheduled remain critical-path but **only at the very end** of
the v1.0.0 prep window. The earlier HANDOFF advice ("M8 is next,
5–7 sessions of careful Tya-in-Tya programming") is correct in
its mechanics but **not the right time to start**. Do not pick up
M8 unless the user explicitly redirects to selfhost work.

Scaffolding that landed earlier is still in `main`:
- v02 mirrors v01 byte-for-byte (modulo +71 lines of M8.1–M8.2d
  patches).
- `TestSelfhostV02Scripts` is green.
- v02 parser accepts `@@` single token + `abstract` / `final` /
  `override` modifiers (codegen no-op).

The "Future Work" section of `ROADMAP.md` lists everything else
on the v1.0.0 horizon (WASM target, embed, primitive-as-class
sugar, interface defaults, raw `"` in interpolation, etc.).

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

Followed four times in this arc; use the `tya-release` skill —
the authoritative file is `~/.claude/skills/tya-release/SKILL.md`
(invoke via the `Skill` tool with `skill: "tya-release"`). The
skill picks up from "version bump + SPEC/RELEASE_NOTES committed
on `main`" and walks through:

1. Verify preconditions (clean main, version bumped, SPEC/RELEASE
   NOTES frozen, ROADMAP Released entry, docs HTML rebuilt,
   `go test ./... -count=1` green, `gh` authed).
2. Tag `vX.Y.Z` and push tag.
3. `scripts/build_release_packages.sh X.Y.Z` → 5 platform tarballs
   + `.sha256` files in `dist/`.
4. `gh release create vX.Y.Z` with the platform packages
   (GitHub auto-attaches source `.tar.gz` / `.zip`).
5. Compute source archive sha256
   (`curl -sL <archive-url> | shasum -a 256`).
6. Sync the `komagata/homebrew-tap` repo (separate from `tya`):
   clone if missing, copy the in-tree `Formula/tya.rb`, replace
   `REPLACE_AFTER_TAG_PUSH` with the source sha256, commit, push.
   **Do NOT "reset" the in-tree Formula afterwards** — the only
   file `brew` reads is the tap copy.
7. End-user verify on the release host:
   `brew install komagata/tap/tya && tya version && brew test
   komagata/tap/tya`. On Linux, install Linuxbrew first via the
   official `install.sh` script if absent.

Skill gotchas captured from past releases:

- **Pre-bump `Formula/tya.rb` test block to v0.44+ paren-mandatory
  syntax.** `print "x"` / `assert true` no longer parse. Do not
  call `tya test` on a script-shaped fixture either — v0.44+
  rejects top-level statements in `_test.tya`.
- **Add the new version entry to `scripts/build_docs_pages.js`
  AND `docs/VERSIONS.md`** before rebuilding HTML.
- **Linux build hosts need `-lm` on every `cc`/`gcc` invocation**
  (the v0.49 release wired this into the build driver +
  selfhost fixtures; check it stays wired going forward).

## Quick links

- Releases: https://github.com/komagata/tya/releases
- Homepage: https://tya-lang.org/
- Tap: https://github.com/komagata/homebrew-tap (local clone at
  `~/Projects/komagata/homebrew-tap/`)
- ROADMAP: [`ROADMAP.md`](../ROADMAP.md)
- Latest SPEC: [`docs/v0.49/SPEC.md`](v0.49/SPEC.md)
