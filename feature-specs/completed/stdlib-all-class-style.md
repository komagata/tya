---
status: completed
goal_ready: false
---

# Feature: Convert All Stdlib Public APIs to Class Style

## Goal

Make the entire Tya standard library use one public shape:
directory packages containing PascalCase class files, with users calling
stdlib behavior through class names instead of module-level functions, and with
stdlib-owned domain values represented as class instances unless there is a
clear performance, interoperability, or data-exchange reason to keep dictionaries.

## Context

The `stdlib/` tree is already physically class-file based: current standard
library files live under paths such as `stdlib/math/Math.tya`,
`stdlib/file/File.tya`, `stdlib/json/Json.tya`, and
`stdlib/net/http/Server.tya`. Earlier specs moved the implementation away from
`module name + free functions`, and primitive `string` / `array` / `dict`
module facades were removed in v0.59.

However, editable docs and some public examples still describe many stdlib
packages as module-style APIs:

```tya
import math
print math.abs(-3)

import file
text = file.read("memo.txt")
```

That keeps two apparent styles alive. This feature makes class style the only
documented and supported stdlib public API:

```tya
import math as math
print math.Math.abs(-3)

import file as file
text = file.File.read("memo.txt")
```

Some existing stdlib APIs also return dictionaries as domain values, such as
`matrix.Matrix.new`, `net/ip.Address.parse`, `net/ip.Network.parse`, and test
runner result summaries. Those should be reviewed separately from ordinary data
exchange dictionaries. When a value is a named stdlib concept, a class instance
is usually the better public shape.

## Behavior

- Every bundled stdlib package exposes public behavior through one or more
  PascalCase classes.
- Public calls use:

  ```text
  <import-binding>.<Class>.<static_method>(...)
  <Class>.<static_method>(...)        # only when imported without alias and in scope
  instance.method(...)
  ```

- Module-level function calls such as `math.abs(...)`, `file.read(...)`,
  `json.parse(...)`, `url.encode(...)`, `random.int(...)`, and
  `secure_random.hex(...)` are no longer documented as valid public APIs.
- Stdlib-owned domain values should return class instances, not loose
  dictionaries, when all of these are true:
  - the value has a named concept in the stdlib API,
  - callers are expected to pass the value back to stdlib methods,
  - fields and invariants matter,
  - the value is not primarily a JSON/TOML/Markdown-style data tree.
- Existing dictionary-returning public values should be migrated to class
  instances where practical:
  - `matrix.Matrix.new/zero/identity/add/sub/scale/mul/transpose`
  - `net/ip.Address.parse`
  - `net/ip.Network.parse`
  - `unittest.TestRunner.default().run`
  - future `geometry`, `image`, `sqlite`, `raylib`, and similar stdlib or
    package-owned values
- Keep dictionary results when dictionaries are the right data model:
  - parsed JSON, TOML, URL query dictionaries, and CSV rows,
  - Markdown public AST nodes, unless a separate AST object-model PRD replaces
    them,
  - HTTP request/response dictionaries, until a dedicated HTTP request/response
    object PRD exists,
  - option dictionaries passed into stdlib calls,
  - transient private parser/render state inside stdlib implementations.
- If converting a public dictionary value to an instance would cause a large
  runtime-speed regression, excessive allocation pressure, or a broad
  compatibility break, the implementation may keep the dictionary form. The
  exception must be documented in the PRD/release notes with the reason and a
  possible future migration path.
- Migrated values should expose public fields for simple data and class/static
  helpers for operations. Instance methods may be added when they improve
  readability without duplicating a large static API.
- Current stdlib class files remain the source of truth:
  - `base64.Base64`
  - `channel.Channel`
  - `cli.Cli`
  - `compress.Compress`
  - `csv.Csv`
  - `digest.Digest`
  - `dir.Dir`
  - `file.File`
  - `hex.Hex`
  - `io.Io`, `io.Reader`, `io.Writer`
  - `json.Json`
  - `log.Logger`
  - `markdown.Markdown`
  - `math.Math`
  - `matrix.Matrix`
  - `net/http.Server`
  - `net/ip.Address`, `net/ip.Network`
  - `net/socket.Socket`, `net/socket.Server`
  - `os.Os`
  - `path.Path`
  - `process.Process`
  - `random.Random`
  - `runtime.Runtime`
  - `secure_random.SecureRandom`
  - `sync.AtomicInteger`, `sync.Mutex`, `sync.WaitGroup`
  - `task.Task`
  - `template.Template`
  - `time.Time`
  - `toml.Toml`
  - `unittest.TestCase`, `unittest.TestRunner`, `unittest.TestSuite`
  - `url.Url`
  - `value.Value`
- Documentation should prefer import aliases when it clarifies the namespace:

  ```tya
  import json as json
  data = json.Json.parse(text)
  ```

- Documentation may use bare class names for packages that are conventionally
  imported without an alias:

  ```tya
  import unittest

  class MyTest extends TestCase
  ```

- Existing private runtime builtins that support class wrappers remain private
  implementation details. They are not public stdlib API even when their names
  are visible to wrapper code.
- If any legacy single-file stdlib module or compatibility facade remains, remove
  it unless it is required only for `selfhost/v01`; self-host exceptions must be
  path-gated and documented.
- If module-style member lookup currently works accidentally for a class-file
  package, add tests that pin the intended public behavior and update docs to
  stop relying on the accidental shape. The implementation may reject
  module-style calls immediately or preserve them internally as undocumented
  compatibility only when removing them would break current self-host or release
  invariants.
- The current release docs under `docs/vX.Y/` remain historical; editable docs
  and new release docs must use class style.

## Scope

- Audit every file under `stdlib/` for:
  - PascalCase filename,
  - matching public class name,
  - no lowercase importable script modules,
  - no `module` declarations,
  - no public top-level function bindings.
- Audit public stdlib return values that are dictionaries and classify them as:
  - domain value to migrate to class instance,
  - data exchange shape to keep as dictionary,
  - private implementation state that does not affect public API.
- Migrate practical domain values to class instances and update their tests.
- Update current editable docs:
  - `docs/STDLIB.md`,
  - `docs/SPEC.md`,
  - `docs/API.md`,
  - `docs/NAMING.md`,
  - `docs/TERMINOLOGY.md`,
  - `docs/LIBRARIES.md` where examples mention stdlib imports.
- Update examples and tests that still use module-style stdlib calls.
- Update LSP completion/hover docs or test fixtures if they advertise
  module-level stdlib functions.
- Add negative or documentation-driven tests that prevent reintroducing
  lowercase `stdlib/*.tya` modules and module-style stdlib examples in editable
  docs.
- Next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`.
- Generated docs HTML only when release prep rebuilds it.

## Out of Scope

- Changing primitive value methods such as `"x".len()` or `{ a: 1 }.keys()`.
- Moving `print`, `println`, `assert`, `assert_equal`, `panic`, `error`,
  `chr`, `ord`, or other language/runtime builtins into stdlib classes.
- Renaming existing stdlib classes.
- Reorganizing package names such as `secure_random` or `net/http`.
- Replacing private runtime helper builtins with pure Tya implementations.
- Forcing every dictionary in stdlib internals into a class. Private state,
  options, parsed data trees, and wire-format data may remain dictionaries.
- Introducing a hard rule that blocks dictionary returns even when performance or
  compatibility costs are too high.
- Migrating archived historical specs under `docs/vX.Y/` or
  `docs/archive/pre-v0.1/`.
- Removing self-host compatibility gates required by `selfhost/v01`.

## Acceptance Criteria

- `find stdlib -name '*.tya'` reports only files whose basename starts with an
  uppercase letter.
- Every stdlib `.tya` file's public class name matches its filename.
- No stdlib source contains a `module` declaration.
- Editable docs no longer advertise `math.abs`, `file.read`, `json.parse`,
  `url.encode`, `random.int`, `secure_random.hex`, or similar module-style
  calls as current public API.
- Editable docs show class-style calls such as `math.Math.abs`,
  `file.File.read`, `json.Json.parse`, `url.Url.encode`,
  `random.Random.int`, and `secure_random.SecureRandom.hex`.
- Repository examples and tests use class-style stdlib calls except for
  historical fixtures that intentionally test old behavior.
- Public stdlib domain values are either class instances or have an explicit
  documented reason for remaining dictionaries.
- `matrix.Matrix.new(...)` returns a `Matrix` instance unless implementation
  evidence shows a significant performance or compatibility problem.
- `net/ip.Address.parse(...)` and `net/ip.Network.parse(...)` return `Address`
  and `Network` instances unless implementation evidence shows a significant
  performance or compatibility problem.
- Public docs distinguish domain instances from data-exchange dictionaries.
- Any remaining compatibility for module-style stdlib calls is documented as
  internal/transitional and is not presented as public API.
- Self-host v01 still compiles to a stable fixed point.
- `go test ./... -count=1` passes.

## Verification

```sh
test -z "$(find stdlib -name '*.tya' -printf '%f\n' | rg '^[a-z]')"
test -z "$(rg -n '^module ' stdlib)"
rg -n 'math\.Math\.abs|file\.File\.read|json\.Json\.parse|url\.Url\.encode' docs/STDLIB.md
test -z "$(rg -n '\b(math|file|json|url|random|secure_random)\.[a-z_][a-z0-9_!?]*\(' docs/STDLIB.md docs/SPEC.md docs/API.md docs/LIBRARIES.md)"
rg -n 'Matrix instance|Address instance|Network instance|data-exchange dictionar' docs/STDLIB.md feature-specs/stdlib-all-class-style.md
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Dependencies

- Builds on the v0.44 class-oriented namespace model.
- Builds on the v0.59 primitive stdlib consolidation that removed
  `string` / `array` / `dict` module facades.
- Must account for any self-host v01 compatibility gates before deleting
  behavior that the fixed-point compiler still uses.

## Open Questions

None.
