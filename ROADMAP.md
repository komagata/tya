# Tya Roadmap

`ROADMAP.md` is the single source of truth for current TODO, TASK, and roadmap
planning.

Pre-v0.1 planning documents and self-host migration notes are archived under
[`docs/archive/pre-v0.1/`](docs/archive/pre-v0.1/). They are historical
references, not current language or implementation authority.

## Self-Host Invariant

The Tya-written compiler fixed point is a maintained invariant. Later language,
runtime, CLI, stdlib, and documentation work must not regress
`selfhost/v01/compiler.tya`.

Required evidence:

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
```

This gate proves that the Tya-written compiler can compile itself to stable
stage-2/stage-3 C output, and that the self-hosted stage-2 compiler can compile
and run representative programs through the maintained surface.

## v1.0.0 Goal

Tya v1.0.0 is the version at which all five language commitments hold and
are publicly defensible. These commitments are also Tya's external "at a
glance" feature list:

1. **Canonical Syntax** — every program has exactly one source representation.
   The formatter is part of the language, not a separate opinion.
   *(Shipped in v0.38 / v0.39.)*
2. **Dynamically typed**, indentation-based, with strict semantics
   (no implicit conversions, no `nil` arithmetic).
3. **Compiles to C** for a small, portable runtime.
4. **All-in-one toolchain** (Gleam-style) — the `tya` binary holds the
   compiler, formatter, language server, test runner, doc generator, and
   package manager.
5. **Kind diagnostics** (Elm-grade) — every error has a stable code, an
   expected/found block, an actionable hint, and a linked explanation.
6. **Self-hosted** — the Tya compiler is written in Tya itself. The Go
   reference implementation is removed at v1.0.0 release. Bootstrap from
   a pre-built `tya` binary distributed via the project's release
   channels.

Each commitment maps to one or more Epics below:

- Commitment 1 → *(landed)*.
- Commitment 2 → strict-semantics audit (currently implicit; to be made
  an explicit Epic).
- Commitment 3 → already shipped; maintained.
- Commitment 4 → *Ship `tya lsp` Language Server*, *Ship `tya doc` source
  documentation generator*, *Ship `tya new` project scaffolder*, *Ship
  `tya task` project task runner*, plus existing `tya check` /
  `tya test` / `tya format` / package manager.
- Commitment 5 → *Migrate remaining stages to the diagnostics pipeline*.
- Commitment 6 → *Migrate selfhost compiler to latest spec and remove
  the Go reference implementation* (see Future Work / Self-host).

Other Epics (WASM target, syntax coloring, embedding,
self-introspection library, coverage extensions, markdown extensions,
lint, multi-line string extensions) are valuable but not strictly
required for v1.0.0. They may ship before or after v1.0.0.

Scheduled minor versions on the path to v1.0.0:

- **v0.41** — Ship a precise mark-and-sweep garbage collector for the C
  runtime. Single-threaded, stop-the-world. Foundation for v0.42
  Concurrency.
- **v0.42** — Ship Tya Concurrency: `spawn` / `await` / `scope`
  keywords, `channel` / `sync` stdlib modules, `task` value type. 1:1
  OS-thread implementation; multi-thread GC extension. M:N scheduler
  is deferred to a later minor.

## Current Direction

Tya is implemented as a small compile-to-C language. The latest released
specification is **v0.39**. Frozen release documents live under
`docs/vX.Y.Z/` and `docs/vX.Y/`; the latest editable specification, API,
stdlib, and naming documents live directly under `docs/`.

Tya uses semantic versioning. Specification changes happen at the minor version
level. Patch releases must not change language or standard-library semantics.

Latest editable documentation:

1. [`docs/SPEC.md`](docs/SPEC.md)
1. [`docs/API.md`](docs/API.md)
1. [`docs/STDLIB.md`](docs/STDLIB.md)
1. [`docs/NAMING.md`](docs/NAMING.md)
1. [`docs/CANONICAL_SYNTAX.md`](docs/CANONICAL_SYNTAX.md)

The reference implementation is:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
C runtime
specification tests
```

Go interpreter behavior, ASTMODE, and legacy archived node-string experiments
are not specification authority. The maintained `selfhost/v01/compiler.tya`
fixed point must not regress.

## Implementation Tooling Policy

The compiler implementation should stay hand-written:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
```

Do not add a parser generator or large grammar framework. In particular, avoid
introducing Participle, goyacc, Pigeon, ANTLR, or Tree-sitter as compiler
front-end authority. They may be useful references or future editor tooling,
but the active compiler path should remain explicit Go code.

After the Go implementation reaches a complete lexer, parser, AST, checker, and
C emitter for the current specification, continue self-host work in the same
component order:

```text
Tya lexer
Tya parser
Tya AST
Tya checker
Tya C emitter
```

Each Tya component must preserve the self-host fixed point before moving to the
next component.

Use small test-support dependencies where they make the specification easier to
verify:

```text
github.com/google/go-cmp/cmp
github.com/rogpeppe/go-internal/testscript
```

Use `go-cmp` for readable token, AST, diagnostic, and generated-output diffs.
Use `testscript` for CLI-level specification tests, especially `tya run`,
`tya build`, expected stdout/stderr, and negative examples.

## Released

Recent shipped minor versions, newest first. Frozen specs live under
`docs/vX.Y/`.

- **v0.43** — Concurrency known-gap close-out: cooperative
  cancellation (`task.cancel` / `task.cancelled?` / `task.current`),
  `scope` body raises run cleanup before unwinding,
  `channel.select` multiplex.
- **v0.42** — Tya Concurrency: `spawn` / `await` / `scope` keywords;
  `task` / `channel` / `mutex` / `atomic_integer` / `wait_group` value
  kinds; `channel` and `sync` stdlib modules; multi-thread GC.
- **v0.41** — precise mark-and-sweep garbage collector for the C runtime.
  `runtime.gc()` and `runtime.gc_stats()`; auto-trigger between top-level
  statements. Foundation for v0.42 Concurrency.
- **v0.40** — raw and bytes string extensions: `r"..."`, `r"""..."""`,
  `b"""..."""`.
- **v0.39** — Canonical Syntax surface cleanup. `tya format` only spelling;
  `--text` / `--ast` opt-outs removed.
- **v0.38** — Canonical Syntax landing. AST-driven serializer is the default;
  examples / stdlib / selfhost normalized; selfhost compiler extended for
  canonical wrap forms.
- **v0.37** — `formatter.Unparse` foundation for the common-case subset.
- **v0.36** — comment attachment recurses into nested Stmt-list bodies.
- **v0.35** — per-stmt `Comments` map at top level.
- **v0.34** — lexer comment capture; `Program.HeaderComments`.
- **v0.33** — parser accepts `(a, b) -> body` parenthesized multi-parameter
  lambda.
- **v0.32** — lexer diagnostics migration (`TYA-E0001`–`TYA-E0017`); pure-Tya
  `markdown.to_html` foundation.
- **v0.31** — multi-line `"""..."""` string literals.
- **v0.30** — test coverage foundation: `tya test --cover`, `tya cover`,
  `# tya-cover 1` profile format.
- **v0.29** — diagnostics foundation: `internal/diag` model, human + JSON
  renderers, color modes, `TYA-Xnnnn` code namespace.
- **v0.28** — strict compile-time checks (shadowing, unused
  imports/args/private definitions).
- **v0.27** — hexadecimal and binary integer literals.
- **v0.26** — external packages, `tya.toml`, version resolution.
- **v0.25** — bitwise operators, byte sequences, binary file I/O.
- **v0.24** — `time`, `random`, `process`, `hex`, `digest`, `secure_random`,
  `matrix` modules; `math` expansion.

## Scheduled

Epics with assigned minor versions, on the path to v1.0.0. Each Epic is
implemented in numbered STEPs. Every STEP must pass `go test ./... -count=1`
and `go test ./tests -run TestSelfhostV01Scripts -count=1` before the next
STEP starts. The STEP also keeps `docs/vX.Y/SPEC.md` consistent with the
implementation up to that STEP.

### v0.44 — Class-oriented namespace and entry-file model

Replace the snake_case `module name` namespace with directory-as-package,
require every importable file to be a PascalCase class file, and define
the entry file as a lowercase script file desugared to an unnamed class.
This Epic is destructive: it removes the `module` keyword, renames every
stdlib file, and rewrites the self-host compiler.

Spec: [`docs/v0.44/SPEC.md`](docs/v0.44/SPEC.md).

The Epic ships in additive STEPs first so existing tests stay green, then
in migration STEPs that move each consumer (stdlib, examples, selfhost)
onto the new model, then in removal STEPs that delete the legacy
`module` surface. The self-host fixed point gate must stay green at
every STEP boundary.

- [ ] **Land class-oriented namespace and entry-file model**
  - [x] **M1: Spec and design groundwork**
    - [x] Land `docs/v0.44/SPEC.md` with the new model.
    - [x] Add `docs/v0.44/spec.html` rendering and link it from
      `docs/VERSIONS.md`.
    - [x] Reserve diagnostic code ranges for the new errors (parser,
      checker, runner) and document them in `docs/v0.44/SPEC.md`.
    - [x] Document the migration policy: `module` files keep working
      until M8; class files coexist additively from M2 onward.
  - [x] **M2: Parser and checker accept class files additively**
    - [x] Parser allows a `.tya` file whose top level is one or more
      `class` / `interface` declarations with no `module` wrapper.
    - [x] Checker enforces "PascalCase filename → public class name
      matches filename" only when the new shape is used.
    - [x] Checker enforces "exactly one public class per class file"
      and "additional classes are private to the file".
    - [x] Add positive and negative parser/checker tests for class
      files. Existing `module` files keep parsing.
    - [x] No change to runner, resolver, stdlib, or selfhost yet.
  - [x] **M3: Resolver gains directory-as-package import**
    - [x] `import path/to/pkg` resolves to a directory.
    - [x] Resolution roots: entry directory, `TYA_PATH`, `stdlib/`.
    - [x] Reject `..`, absolute paths, and paths whose terminal segment
      is not a valid `snake_case` package name.
    - [x] Reject importing a directory that contains a script file at
      its leaf (no script-file imports).
    - [x] Same-directory class-file siblings are auto-visible without
      prefix.
    - [x] Cross-package access is `<terminal-segment>.<ClassName>`.
    - [x] Existing single-file `module` imports keep working.
    - [x] Add testdata under `tests/testdata/v44/` covering nested
      packages, name-collision diagnostics, and rejected paths.
    - [x] Within-package class-to-class bare references resolve
      through the `<currentModule>.<Name>` fallback in checker and
      codegen (commit f6d276b). Sibling classes inside the same
      synthesized package call each other without prefix.
    - [x] Same-segment package collision detection (commit c3bfa48e).
      Two directory packages whose paths share the terminal segment
      are caught at synthesis time with a "package name conflict"
      diagnostic, both for unaliased and aliased imports. Pinned by
      `tests/testdata/v44/import_collision.txtar`.
  - [ ] **M4: Compact entry-file desugaring**
    - [x] Runner accepts only lowercase script files for `tya run`.
    - [x] PascalCase class files are rejected as runner targets with a
      structured diagnostic.
    - [ ] Lowercase script files desugar to an implicit unnamed class
      with `main` containing the file's top-level statements.
    - [ ] Top-level bindings become locals of `main`.
    - [ ] Define and reserve the `_Anonymous` (or equivalent) implicit
      class name and ensure it is unspeakable from user code.
    - [ ] Update CLI tests for `tya run`.
  - [ ] **M5: Private-class enforcement**
    - [ ] Class files: every class beyond the public one is private.
    - [ ] Script files: every class is private.
    - [ ] Private classes are not visible to other files.
    - [ ] Diagnostics for cross-file private references.
    - [ ] Note: full enforcement requires AST-level `OriginFile`
      tracking on every `ClassDecl` propagated through synthesis →
      parse → check; the source-concat pipeline does not carry that
      information today. Implementation is paired with M8 self-host
      migration when AST-merge replaces source-concat.
  - [ ] **M6: Stdlib migration**
    - [x] Add `stdlib/<pkg>/<Class>.tya` form for each existing stdlib
      module, in dependency order.
    - [x] Per package: introduce class file, port functionality,
      switch internal users, delete the old `module` file.
    - [x] Add `os.Os.args()` (or finalized API) as part of the `os`
      migration. The migrated `stdlib/os/Os.tya` exposes
      `@@args` which delegates to the existing `args()` builtin;
      callers spell it `os.Os.args()`. Existing call sites that
      remain in the legacy `module` form continue to use `args()`
      directly until M7 examples migration.
    - [ ] Migrate, in order: `runtime`, `os`, `path`, `string`, `array`,
      `dict`, `math`, `time`, `random`, `process`, `hex`, `digest`,
      `secure_random`, `matrix`, `file`, `dir`, `csv`, `json`,
      `markdown`, `base64`, `channel`, `sync`. Each item is its own
      STEP that keeps tests green.
      - [x] `path`, `random`, `hex`, `base64`, `digest`,
        `secure_random` (commits dcf7cd9, 5763729)
      - [x] `process`, `dir`, `file`, `os`, `math`, `csv`, `matrix`,
        `json`, `toml`, `url` (commit 0cbf27e + revert 6138f80)
      - [x] `unittest`, `value`, `markdown` (commits d8a8cd0, cb51b56,
        0aa3a4c)
      - [ ] `string`, `array`, `dict` — held back for v0.1 self-host
        compatibility (the v0.1 compiler resolves `import X` only as
        single-file modules; see `docs/v0.44/SPEC.md` §"Self-Host
        Invariant Constraint"). Migrate alongside M8.
      - [ ] `runtime`, `time`, `channel`, `sync`, `task` — held back
        because their callers in `examples/` and `tests/testdata/`
        carry pending working-tree edits unrelated to v0.44; migrate
        when the working tree is clean.
    - [ ] Update `docs/STDLIB.md` per package as it lands.
  - [ ] **M7: Examples migration**
    - [ ] Convert each `examples/*.tya` to the new model: lowercase
      script files for entries, PascalCase class files for libraries.
    - [ ] Remove obsolete `module` declarations from examples.
    - [ ] Update example-driven tests and `scripts/go_emit_examples_check.sh`.
  - [ ] **M8: Self-host compiler migration**
    - [ ] Land a class-shaped `selfhost/v02/compiler.tya` (or
      equivalent path; final naming TBD) targeting the v0.44 surface.
    - [ ] Preserve the v01 fixed point on the v0.43 surface; both
      live side by side until parity is proven.
    - [ ] Stage-2 == stage-3 fixed point on the v0.44 surface.
    - [ ] Replace `TestSelfhostV01Scripts` target only after parity
      and a release boundary.
  - [ ] **M9: Remove the `module` keyword**
    - [ ] Parser rejects `module` as a reserved-word error.
    - [ ] Checker, formatter, and emitter drop `module` paths.
    - [ ] Delete remaining `module`-only files.
    - [ ] Remove `module`-related diagnostics that no longer apply.
  - [ ] **M10: Documentation cleanup**
    - [ ] Promote `docs/v0.44/SPEC.md` content into `docs/SPEC.md`.
    - [ ] Rewrite `docs/NAMING.md` for the new file-kind rules and
      remove the Module Rule section.
    - [ ] Update `docs/STDLIB.md` cross-references.
    - [ ] Update `docs/CANONICAL_SYNTAX.md` for the file-kind rules.
    - [ ] Update `docs/GUIDE.md`, `docs/API.md`, `docs/TERMINOLOGY.md`,
      `docs/LIBRARIES.md` as needed.
    - [ ] Rebuild HTML via `node scripts/build_docs_pages.js`.
    - [ ] Add a release note entry under "Released" in this file.

_v0.42 shipped — see Released above and `docs/v0.42/SPEC.md`._
_v0.43 shipped — see Released above and `docs/v0.43/SPEC.md`._

## Future Work

Epics below are committed direction but not yet scheduled to a specific
minor version. Each will be scoped into a `docs/vX.Y/SPEC.md` when picked up.

### Toolchain

- [ ] **Migrate remaining stages to the diagnostics pipeline**
  - [ ] Parser → `TYA-E0100`–`E0299`.
  - [ ] Codegen → `TYA-E0600`–`E0799`.
  - [ ] Runner → `TYA-E0800`–`E0899`.
  - [ ] Add did-you-mean suggestions for unknown-name diagnostics.
  - [ ] Add multi-error parsing.

- [ ] **Ship `tya lsp` Language Server**
  - [ ] Define LSP scope; ship `tya lsp` as a subcommand of the same
    binary so compiler and language server cannot drift in version.
  - [ ] Speak LSP over stdio (JSON-RPC) for VS Code, Zed, Helix, Neovim,
    Emacs.
  - [ ] Implement diagnostics on save / on change, hover, go-to-definition,
    find-references, completion (in-scope names + module members + stdlib),
    formatting (full + range, backed by `tya format`), rename, code
    actions for common diagnostics.
  - [ ] Publish a minimal VS Code extension and document Zed / Helix /
    Neovim / Emacs setup.

- [ ] **Ship `tya doc` source documentation generator**
  - [ ] Define doc comment syntax: contiguous comment lines immediately
    preceding a top-level definition. Body is Markdown rendered with the
    `markdown` stdlib module.
  - [ ] Discover every top-level binding under `src/` plus stdlib re-exports.
  - [ ] CLI surface: `tya doc` (text), `tya doc --html <out>` (static site),
    `tya doc --serve` (HTTP), `tya doc --json` (machine-readable).
  - [ ] Reuse the public Tya self-introspection library.
  - [ ] Diagnose orphan doc comments, duplicate definitions, unparseable
    Markdown bodies.

- [ ] **Ship `tya new` project scaffolder**
  - [ ] CLI: `tya new <name>`, `tya new --here`, fixed templates
    (`--template app|lib`).
  - [ ] Default template: `tya.toml`, `src/main.tya` hello-world, `tests/`
    with one passing unittest, `.gitignore`, minimal `README.md`.
  - [ ] Refuse to overwrite existing files unless `--force`. Initialize git
    by default; `--no-git` to skip.

- [ ] **Ship `tya task` project task runner**
  - [ ] `[tasks]` table in `tya.toml` is the single source of truth.
  - [ ] String form (`ci = "tya format && tya test"`) and array form.
  - [ ] CLI: `tya task <name>` and `tya task` (list).
  - [ ] Resolve `tya.toml` from working directory upward; execute via the
    user's shell with the project root as CWD.
  - [ ] Keep parallelism, file-watching, task graphs out of the initial
    scope.

- [ ] **Ship `tya lint` source linter**
  - [ ] Boundary: `tya format` is layout, `tya check` is correctness, `tya
    lint` is stylistic / semantic best practices.
  - [ ] CLI: `tya lint [paths]`, `tya lint --fix`, `tya lint --format=json`.
  - [ ] Initial rule set: unused locals, dead code after `return` / `raise`,
    redundant `if true` / `if false`, suspicious `for` index patterns,
    deeply nested blocks, very long functions.
  - [ ] Each rule has a stable code (e.g. `TYAL0001`), title, doc URL.
  - [ ] Per-line opt-out via `# tya-lint-ignore: TYAL0001`.

- [ ] **Ship a public Tya self-introspection library**
  - [ ] Goal: Tya programs and external tools can lex, parse, walk, and
    re-emit Tya source through a stable, documented API. Unlocks
    codemods, linters, doc extractors, refactoring, AI-agent edits, and
    `tya lsp`.
  - [ ] Surface as Tya stdlib modules built on the self-host components:
    `compiler.lexer`, `compiler.parser`, `compiler.ast`, `compiler.checker`,
    `compiler.format`.
  - [ ] AST is the single canonical representation; node kinds, fields,
    spans documented.
  - [ ] Round-trip tests over a representative corpus.

### Language

- [ ] **Ship WebAssembly compilation target**
  - [ ] Use Zig (`zig cc --target=wasm32-wasi` and
    `zig cc --target=wasm32-freestanding`) as the WASM toolchain.
  - [ ] CLI: `tya build --target wasm32-wasi`,
    `tya build --target wasm32-browser`.
  - [ ] Decide system interface: WASI for CLI-style programs, browser-friendly
    subset omits filesystem and process APIs.
  - [ ] Gate unavailable stdlib modules per target with structured errors at
    import time.

- [ ] **Ship asset embedding for single-binary distribution**
  - [ ] `embed "assets/logo.png" as logo` bakes file bytes into the compiled
    binary at build time.
  - [ ] Directory / glob form (`embed "static/**" as static`) returns a dict
    keyed by relative path.
  - [ ] Text loads as `string`, binary as `bytes`; explicit `as bytes` /
    `as text` modifier supported.

- [ ] **Primitive literals as class-instance sugar**
  - [ ] Follow-up to v0.44 class-oriented namespace. Extend
    "everything is a class" to literal values: `1` and `1.0` are
    both sugar for `Number(...)` (a single numeric class — no
    Integer/Float split, matching the current `TyaValue.number`
    `double` representation in `runtime/tya_runtime.h`),
    `"hello"` for `String("hello")`, `[1, 2]` for `Array(1, 2)`,
    `{a: 1}` for `Dict("a", 1)`, `true` / `false` for
    `Boolean(true)` / `Boolean(false)`, `nil` for the unique
    `Nil` instance.
  - [ ] Method-call syntax on literals is required: `42.to_s()`,
    `"hi".len()`, `true.to_s()`, `[1,2].len()` all dispatch through
    the wrapper class. Lexer must keep `42.0` (float literal) and
    `42.foo` (method call) unambiguous (Ruby rule: a digit
    immediately after `.` means the dot is part of a float literal;
    otherwise it is a method call).
  - [ ] Operators (`+`, `-`, `*`, `/`, `<`, `==`, `[]`, ...) are
    **not user-redefinable**. They desugar to fixed method names
    on the wrapper class (e.g. `a + b` → `a.__add__(b)`,
    `a == b` → `a.__eq__(b)`). User classes may **define** these
    methods to participate, but cannot **override** them on the
    built-in primitive classes.
  - [ ] Monkey-patching primitive classes (`Number`, `String`,
    `Boolean`, `Nil`, `Array`, `Dict`) is forbidden. The method
    table of each built-in wrapper is fixed at compile time so the
    optimizer can keep the fast path always live.
  - [ ] Cross-type equality is a method-level decision, not a
    language-level one. Standard built-in `__eq__` implementations
    are type-strict: `String#__eq__` returns `false` for
    non-`String` arguments, etc. Because numbers are a single
    `Number` class, `1 == 1.0` is naturally `true` (no cross-type
    case to resolve). Users who want lenient comparison write it
    in their own class's `__eq__`.
  - [ ] No automatic type coercion. There is no Integer/Float
    split today, so the question of fixnum→float or float→bignum
    promotion does not arise. If a future Epic introduces a
    distinct integer type, promotion semantics are decided then.
  - [ ] Runtime representation: keep the current `TyaValue`
    tagged-union (`runtime/tya_runtime.h`, `kind` enum + payload
    fields; `TYA_NUMBER` already stores both integer- and
    fractional-valued numbers as `double`). Wrapper classes
    (`Number`, `Boolean`, `Nil`, `String`, `Array`, `Dict`) are
    process-global singleton class objects created once at
    runtime startup; `x.class` returns the appropriate singleton
    based on the value's `kind` with no allocation. No boxing
    of primitives into heap objects.
  - [ ] Hidden-from-user fast path: the C emitter lowers operator
    desugaring on known-primitive operands directly to the
    existing C arithmetic / comparison helpers, bypassing method
    dispatch. Because monkey-patching and operator redefinition
    are forbidden, this fast path is unconditional — there is no
    "redefinition check" of the kind CRuby needs. Method dispatch
    is only used when the static type is unknown or the receiver
    is a user class.
  - [ ] Migrate stdlib and examples once the wrapper classes land.
  - [ ] Land a separate `docs/vX.Y/SPEC.md` when scheduled.

- [ ] **Grow `interface` toward stackable-trait capability**
  - [ ] **Phase 1: default methods on `interface`**
    - [ ] Allow `interface` declarations to provide method bodies as
      default implementations, in the spirit of Java 8 default methods.
    - [ ] An implementing class inherits the default body unless it
      provides its own override.
    - [ ] Multiple interfaces declaring the same default-method name
      and signature is a compile-time conflict; the implementing
      class must resolve it explicitly. No implicit "first wins"
      rule.
    - [ ] No state, no initialization, no `super` chaining at this
      phase. Defaults may call `self.<other_method>()` only.
    - [ ] Extend checker, C emitter, formatter, and diagnostics for
      the new shape. Reserve a diagnostic code range for default-method
      conflicts and missing required methods.
    - [ ] Keep `selfhost/v01/compiler.tya` (or its v0.44 successor)
      compiling itself to a stable stage-2/stage-3 fixed point.
  - [ ] **Phase 2: state and initialization on `interface`**
    - [ ] Allow `interface` to declare instance fields and an
      initialization block that runs as part of the implementing
      class's construction.
    - [ ] Define a single rule for diamond inheritance of state. The
      starting position is "explicit resolution required": if the
      same ancestor interface contributes the same field via two
      paths, the implementing class must resolve it. Do not silently
      duplicate or silently merge.
    - [ ] Define and document the order in which interface init
      blocks run relative to each other and to the class body. The
      order must be deterministic and stable across recompiles.
    - [ ] Define `self.<field>` scoping inside default methods and
      init blocks: which interface's field is being referenced and
      how name collisions are reported.
    - [ ] C emitter: lay out interface-contributed fields in the
      implementing struct without duplication for the diamond-resolved
      cases; document the layout rule.
    - [ ] Self-host implications: either keep the self-host compiler
      written without using interface state, or migrate it to the
      new shape and re-prove stage-2 == stage-3.
  - [ ] **Phase 3 (optional): linearization and stackable trait**
    - [ ] Decide whether `super` inside an `interface` default method
      means "the parent" (simple) or "the next interface in a
      linearization order" (Scala-style stackable). Until this Epic
      reaches Phase 3, `super` keeps its parent-class meaning only.
    - [ ] If adopted: implement C3 linearization in the checker;
      route `super.method(...)` through the linearized order;
      document the rule in `docs/SPEC.md`.
    - [ ] Re-evaluate vtable / dispatch design in the C emitter for
      stackable dispatch.
    - [ ] Re-prove the self-host fixed point on the new surface.
  - [ ] **Naming decision: keep `interface`, or rename to `trait`**
    - [ ] Defer the rename until at least Phase 2 has shipped and
      real Tya code uses interface state. The choice is editorial,
      not technical: if the strengthened `interface` clearly behaves
      like a Scala trait in stdlib usage, consider a rename in a
      single dedicated Epic with a destructive migration plan.
    - [ ] Do not introduce both `interface` and `trait` as separate
      kinds. One concept, one keyword, consistent with the
      "no hesitation" philosophy.

- [ ] **Forbid closure write-back (strict shadowing extension)**
  - [ ] Background: Tya function bodies can read outer-scope bindings
    but plain `=` assignment to such a name silently creates a new
    local instead of writing back to the enclosing scope. This is a
    common source of bugs (counters that "don't update", caches that
    "don't accumulate") and conflicts with the kind-diagnostics
    commitment.
  - [ ] Decision: forbid `=` reassignment of any name that resolves
    above the current function body. Emit `TYA-E0307`
    "Assignment to outer binding" with hints — pass a dict/array
    argument and mutate its contents (`state["k"] = ...`), use
    `sync.atomic_integer` for a single counter, or rename the inner
    binding to shadow intentionally.
  - [ ] Reading outer bindings stays allowed. Mutating an outer
    container via `push` / index assignment / dict member assignment
    stays allowed (the container itself is not reassigned).
  - [ ] Checker: extend `internal/checker/strict.go` with a per-scope
    `funcBody` flag and a `resolvesAboveFunction` walk that skips
    `strictPredeclared` builtins; fire on `AssignStmt` targets whose
    name resolves above the enclosing function body.
  - [ ] Migrate `selfhost/v01/compiler.tya`, stdlib, and examples to
    comply: rename top-level globals that collide with inner
    function-local bindings (e.g. top-level `tokens` / `errors`),
    and replace `xs = []` resets inside function bodies with
    `while len(xs) > 0; pop(xs)` patterns.
  - [ ] Add positive and negative checker tests; a doc page
    describing the rule and the recommended workarounds.
  - [ ] Schedule into a minor version after v0.44; do not bundle
    with the class-oriented namespace work.

- [ ] **Migrate selfhost compiler to latest spec and remove Go reference**
  - [ ] Bring `selfhost/` up from v0.1 surface to the v1.0.0 spec
    (lexer, parser, AST, checker, C emitter, runner).
  - [ ] Implement all language features (including v0.41 GC interface and
    v0.42 Concurrency) in the self-hosted compiler.
  - [ ] Verify stage-2 == stage-3 fixed point at the latest spec.
  - [ ] Distribute pre-built `tya` binaries via Homebrew, curl install
    scripts, and GitHub Releases as the bootstrap source.
  - [ ] Remove `cmd/tya` and `internal/*` Go sources.
  - [ ] Migrate Go-based tests to Tya-based tests where practical;
    keep specification-driven black-box tests in any language that runs
    against the `tya` binary.
  - [ ] This Epic ships at v1.0.0 and is tied to Commitment 6.

- [ ] **Allow raw `"` inside `{expr}` interpolation body**
  - [ ] Today the lexer reads a string literal as "until the next
    unescaped `"`", so an interpolation `{user["name"]}` is cut at
    the inner `"`. The user has to write `{user[\"name\"]}` or hoist
    the expression into a local. Make the lexer balance `{` / `}`
    while inside an interpolation expression so the body can
    contain quoted sub-expressions verbatim.
  - [ ] Round-trip through the existing interpolation pipeline
    unchanged; only lexer scanning state needs to track depth.
  - [ ] Update the `"""..."""` and raw `r"..."` lexer paths the same
    way so the rule is uniform.
  - [ ] Add positive lexer + script tests pinning
    `"Hello, {user["name"]}!"` and the matching multi-line and raw
    cases.
  - [ ] Reference: most modern interpolation languages already
    allow this (JavaScript template literals, C# `$"..."`, Kotlin,
    Ruby `"#{...}"`, Scala, Dart, Elixir, Swift `\( )`). Python
    aligned in 3.12 via PEP 701; Tya should follow the same model.
    The minority that still forbids nested same-quote is PHP
    `"...{$x['k']}..."` and pre-3.12 Python f-strings — Tya's
    current behavior matches that older / minority position and
    should not stay there.

### Stdlib extensions

- [ ] **Extend coverage tooling (post-v0.30.0)**
  - [ ] `tya cover html` self-contained report.
  - [ ] `--include` / `--exclude` glob filters on `tya cover`.
  - [ ] Read `coverage.include` / `coverage.exclude` from `Tyafile`.
  - [ ] Give `BreakStmt` / `ContinueStmt` source positions and instrument
    them.
  - [ ] Map coverage to per-import-source lines instead of the synthesized
    entry path.
  - [ ] Recommended CI snippet (fail when coverage drops below a threshold).

- [ ] **Extend `markdown` stdlib module (post-v0.32 foundation)**
  - [ ] Public AST: `markdown.parse(text) -> ast`, `markdown.to_html_ast(ast)
    -> string`, `markdown.render(ast, visitor) -> string`, AST node spec.
  - [ ] GFM extensions: tables, task lists, strikethrough, fenced-code
    info-string class.
  - [ ] Reference link definitions, images, nested lists, setext headings,
    HTML blocks.
  - [ ] Raw HTML pass-through with security note.
  - [ ] CommonMark conformance subset run as part of `go test ./...`.

- [ ] **Extend multi-line strings further** (heredoc-style markers,
  language-tagged interpolation specifiers like `sql"""..."""`) — only if
  demand emerges.

### Editor and ecosystem

- [ ] **Ship syntax coloring for the major editors**
  - [ ] Define **major editors**: VS Code, Emacs, Vim, GitHub. Others (Zed,
    Helix, Neovim, JetBrains) welcome but not required for "major editor"
    coverage.
  - [ ] Canonical token taxonomy (keyword, builtin, type, string,
    interpolated expression, number, operator, comment, function name,
    parameter, module name) documented in `docs/`.
  - [ ] House all editor assets under `editors/`: `editors/vscode/`,
    `editors/emacs/`, `editors/vim/`, `editors/tree-sitter-tya/`.
  - [ ] VS Code: TextMate grammar + minimal extension; publish to
    Marketplace and Open VSX.
  - [ ] Emacs: `tya-mode.el` derived mode; publish to MELPA.
  - [ ] Vim: `syntax/`, `ftdetect/`, `indent/` files (also covers Neovim).
  - [ ] GitHub: Tree-sitter grammar registered with `github-linguist/linguist`.

## Verification Reference

Default verification:

```sh
go test ./... -count=1
```

Focused verification should prefer tests for the touched lexer, parser, checker,
C emitter, runtime, examples, stdlib, or docs. The self-host fixed-point gate is
part of the maintained project invariant and must stay green.
