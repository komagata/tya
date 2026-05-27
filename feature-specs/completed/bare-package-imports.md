---
status: completed
goal_ready: false
---

# Feature: Bare Package Imports

## Goal

Make object-oriented code read naturally by letting ordinary package imports
bring public class and interface names into the current file scope, while still
providing an explicit namespace escape hatch through aliased imports.

## Context

Directory packages currently synthesize a namespace dictionary. For a package
such as `lib/unittest/` containing `TestCase.tya` and `TestRunner.tya`,
external users write:

```tya
import unittest

class UserTest < unittest.TestCase
```

That is explicit but noisy in class-oriented positions such as `extends` and
`implements`. Tya should feel Java-adjacent in its class model while remaining
as natural to write as Ruby-style object-oriented code.

## Behavior

- `import package/path` without `as` imports the package's public class and
  interface names directly into the current file scope.
- The import does not create a package namespace binding.
- `import package/path as alias_name` imports the package as a namespace under
  `alias_name` and does not import its public names bare.
- `alias_name` is used with member access:

```tya
import web/testing as web_testing

class ControllerTest < web_testing.TestCase
```

- Bare imported names may be used anywhere a local class or interface name can
  be used, including:
  - superclass positions
  - interface `implements` lists
  - constructor calls
  - static member calls
  - type/introspection expressions that already accept class values
- If two unaliased imports export the same public class/interface name, the
  compiler reports an import name conflict.
- If an unaliased import exports a public class/interface name that conflicts
  with a top-level binding in the importing file, the compiler reports an import
  name conflict.
- Aliased imports can be used to resolve conflicts:

```tya
import unittest
import web/testing as web_testing

class UnitTest < TestCase
class WebTest < web_testing.TestCase
```

- Existing same-directory sibling class visibility remains unchanged.
- Existing within-package bare references remain unchanged.

## Scope

- `docs/SPEC.md`
- `docs/STDLIB.md` examples that currently use package namespace access
- `internal/runner` import/package synthesis
- `internal/checker` import binding and conflict diagnostics
- `internal/codegen` import/package name resolution
- LSP symbol/definition logic for imported classes/interfaces
- script tests under `tests/testdata/`
- existing stdlib and examples that rely on namespace package imports

## Out of Scope

- Importing top-level functions or variables as bare names.
- Star-import or selective-import syntax.
- Keeping namespace package access for unaliased imports.
- Applying the new behavior to aliased imports.
- Multi-version package loading semantics.

## Acceptance Criteria

- `import unittest` makes `TestCase`, `TestSuite`, and `TestRunner` usable as
  bare names when those classes exist in the package.
- `import net/http` makes `Request` and `Response` usable as bare names.
- `import net/http as http` keeps `Request` and `Response` out of bare scope and
  exposes them as `http.Request` and `http.Response`.
- Two unaliased package imports that both export `Request` fail with a clear
  import name conflict.
- A local top-level class/function/value that conflicts with an unaliased
  imported class/interface fails with a clear import name conflict.
- Same-package sibling class references still work.
- Existing package alias tests continue to pass after their expectations are
  updated to the new alias-as-namespace behavior.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run TestV44Script -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Open Questions

None.
