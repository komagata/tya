# Feature: Namespaced Directory Wildcard Imports

## Goal
Make directory package wildcard imports and grouped individual class/interface imports expose public names through a namespace by default, while keeping an explicit `as *` escape hatch for bare class/interface imports. `as name` imports should consistently mean namespace aliasing across script, class/interface, and wildcard imports.

## Context
Tya currently has conflicting import design history:

- Current `docs/v1.0/SPEC.md` still describes unaliased directory imports as importing public class/interface names bare.
- `feature-specs/completed/import-wildcard-and-grouped-syntax.md` introduced `import foo/*` and grouped import blocks.
- `feature-specs/completed/namespaced-class-imports.md` moved toward namespaced class imports, but its `as` behavior makes aliases expose bare names, which conflicts with older script/module alias behavior.
- Existing script/module imports already use `as` as a namespace alias, for example `import greeting as g` followed by `g.hello(...)`.

This feature resolves those conflicts by making namespaced access the default behavior for unaliased class/interface imports, making `as name` consistently define the namespace prefix, and reserving `as *` for intentionally importing public names bare.

## Behavior
- A wildcard directory package import exposes every public class/interface/struct/record in that directory through the import path namespace.

```tya
import foo/*

foo.Bar()
foo.Buz()
```

- A grouped import block containing individual class/interface files exposes those public names through the same namespace as the equivalent wildcard import.

```tya
import
  foo/bar
  foo/buz

foo.Bar()
foo.Buz()
```

- The two examples above are semantically equivalent when `foo/*` contains public `Bar` and `Buz` definitions and no additional public definitions that matter to the program.
- The terminal file segment is not part of the access path. `import foo/bar` exposes `foo.Bar()`, not `foo.bar.Bar()`.
- Nested paths use all directory segments before the file name or wildcard as the namespace.

```tya
import net/http/server
import net/http/request

net.http.Server()
net.http.Request()
```

- Unaliased imports do not create bare public bindings.

```tya
import foo/*

Bar() # invalid
foo.Bar() # valid
```

- `as name` consistently creates a namespace alias for class/interface file imports and wildcard directory imports.

```tya
import foo/* as f

f.Bar()
f.Buz()
```

```tya
import foo/bar as f

f.Bar()
```

- With `as`, the original import-path namespace is not created for that import.

```tya
import foo/bar as f

f.Bar() # valid
foo.Bar() # invalid unless another import creates foo
Bar() # invalid
```

- `as *` explicitly imports public names bare instead of creating a namespace.

```tya
import foo/* as *

Bar()
Buz()
foo.Bar() # invalid unless another import creates foo
```

```tya
import foo/bar as *

Bar()
foo.Bar() # invalid unless another import creates foo
```

- Grouped import block entries may use `as *` individually.

```tya
import
  foo/bar as *
  foo/buz as *

Bar()
Buz()
```

- Script/module imports keep the same namespace alias behavior they already have.

```tya
import greeting as g

g.hello("Tya")
```

- Same-directory sibling visibility remains unchanged: files in the same directory package may still refer to sibling public classes/interfaces/structs/records by bare PascalCase name without importing.
- Additional non-public declarations inside a class/interface file remain private to that file and are not exported through the namespace.
- Importing the same path twice is invalid, even if aliases differ.
- Two imports that expose the same namespace-qualified public name are invalid.
- Two `as name` imports that use the same alias and expose conflicting public names are invalid.
- `as *` imports reserve their imported bare public names. Two `as *` imports that expose the same bare public name are invalid, and an `as *` import that conflicts with a local top-level binding is invalid.
- A local top-level binding may use the same bare class name as an unaliased namespaced import because the bare name is not imported.

```tya
import foo/bar

class Bar
  static local?: -> true

Bar.local?()
foo.Bar()
```

- A local top-level binding may not conflict with the first namespace segment or alias required by an import.

```tya
import foo/bar

foo = "local" # invalid
```

```tya
import foo/bar as f

f = "local" # invalid
```

```tya
import foo/bar as *

class Bar # invalid: Bar is imported bare
```

## Scope
- Update current import/package docs in `docs/SPEC.md`, `docs/ja/spec.md`, `docs/GUIDE.md`, and current version docs where they describe directory packages, wildcard imports, grouped imports, import aliases, or bare imported names.
- Update parser/AST behavior so `as *` is valid import syntax and distinguishable from `as name`. Reuse the existing `ImportStmt` fields for `Name`, `Alias`, and `Wildcard` when sufficient; otherwise add the smallest import-mode representation needed.
- Update resolver/package loading/import synthesis so unaliased wildcard and individual class/interface imports create namespace-qualified public bindings.
- Update checker name resolution and conflict diagnostics for namespace-qualified imports, alias namespace imports, explicit `as *` bare imports, duplicate paths, local namespace conflicts, bare import conflicts, and no-longer-implicitly-imported bare names.
- Update code generation and runtime lookup for namespace-qualified imported class/interface/struct/record names.
- Update formatter/unparser import sorting/grouping only where canonical output or import grouping depends on the changed semantics.
- Update LSP diagnostics, completion, hover, go-to-definition, references, and semantic tokens for namespace imports, aliases, and explicit bare imports.
- Migrate stdlib, examples, tests, selfhost sources, and docs away from implicit bare public names imported from wildcard directory packages. Keep or introduce bare imported names only where the source explicitly uses `as *`.
- Add or update script tests for wildcard imports, grouped individual imports, nested namespaces, alias namespaces, explicit `as *` bare imports, bare-name failures without `as *`, local bare-name coexistence, bare import conflicts, namespace conflicts, duplicate imports, and same-package sibling visibility.

## Out of Scope
- No recursive wildcard imports such as `foo/**`.
- No mid-path wildcard imports.
- No selective import list syntax such as `import foo/{Bar, Buz}`.
- No compatibility mode for old bare wildcard imports.
- No warning-only transition for old bare wildcard imports.
- No per-name selective import list syntax; use `as *` for explicit bare import and namespace imports otherwise.
- No change to same-directory sibling bare visibility.
- No change to script/module import alias semantics except documenting that `as name` is consistent with class/interface imports.

## Acceptance Criteria
- `import foo/*` makes `foo.Bar()` and `foo.Buz()` valid when `foo/bar.tya` and `foo/buz.tya` export `Bar` and `Buz`.
- `import foo/*` does not make `Bar()` or `Buz()` valid.
- `import` blocks with `foo/bar` and `foo/buz` make `foo.Bar()` and `foo.Buz()` valid.
- `import foo/bar` exposes `foo.Bar()`, not `foo.bar.Bar()`.
- `import net/http/server` exposes `net.http.Server()`.
- `import foo/* as f` exposes `f.Bar()` and `f.Buz()`, and does not expose `foo.Bar()` or bare `Bar()`.
- `import foo/bar as f` exposes `f.Bar()`, and does not expose `foo.Bar()` or bare `Bar()`.
- `import foo/* as *` exposes bare `Bar()` and `Buz()`, and does not expose `foo.Bar()` unless another import creates `foo`.
- `import foo/bar as *` exposes bare `Bar()`, and does not expose `foo.Bar()` unless another import creates `foo`.
- Grouped imports can use `as *` on individual entries and expose those entries bare.
- Existing script/module alias behavior such as `import greeting as g` followed by `g.hello(...)` still works.
- Same-directory sibling class/interface/struct/record references still work bare.
- A local top-level `class Bar` may coexist with `import foo/bar` and can be used as `Bar`, while the imported class remains `foo.Bar`.
- A local top-level binding named `foo` conflicts with `import foo/bar`.
- A local top-level binding named `f` conflicts with `import foo/bar as f`.
- A local top-level binding named `Bar` conflicts with `import foo/bar as *`.
- Duplicate import paths fail clearly even if aliases differ.
- Imports that expose the same namespace-qualified public name fail clearly.
- Two `as *` imports that expose the same bare public name fail clearly.
- Documentation no longer describes wildcard or directory package imports as implicitly importing public names bare.

## Verification
```sh
go test ./internal/parser ./internal/formatter ./internal/checker ./internal/codegen ./internal/lsp ./tests -count=1
go test ./... -count=1
```
