# Feature: Namespaced Class Imports

## Goal
Make imported class and interface APIs resolve through their import-path namespace by default, so standard-library and package classes can use natural names such as `Application`, `Result`, `Server`, and `Client` without leaking those names into every importing file.

## Context
Current wildcard directory imports expose public class and interface names directly:

```tya
import net/http/*

server = Server()
```

That makes class-oriented code short, but it also makes library authors avoid useful names such as `Application`, `Result`, `Client`, and `Server` because those names collide easily across packages. Existing alias imports avoid the collision by creating an alias namespace:

```tya
import net/http/* as http

server = http.Server()
```

The new model intentionally makes namespaced access the default and reserves bare imports for explicit `as` imports.

This is a breaking change that replaces the completed `bare-package-imports` behavior. Existing stdlib, examples, tests, and documentation that rely on bare public names from unaliased imports must migrate.

## Behavior
- A normal single-file class/interface import exposes the public class or interface through the import path's parent namespace.

```tya
import net/http/server

server = net.http.Server()
```

- The imported file's terminal segment is not part of the access path. `import net/http/server` exposes `net.http.Server`, not `net.http.server.Server`.
- The public name is not imported bare. `Server()` is invalid after `import net/http/server`.
- A shallow import follows the same rule:

```tya
import cli/application
import cli/result

app = cli.Application()
result = cli.Result()
```

- A one-segment class/interface import exposes the public class or interface through that segment:

```tya
import base64

encoded = base64.Base64("hello").encode()
```

- A wildcard directory import exposes all public classes and interfaces through the imported directory namespace:

```tya
import net/http/*

server = net.http.Server()
request = net.http.Request()
```

- Unaliased wildcard imports no longer import public class or interface names bare.
- `as` imports explicitly import public names bare and do not create an alias namespace.

```tya
import net/http/* as http

server = Server()
```

- For single-file imports, `as` also imports the public class or interface bare:

```tya
import net/http/server as http

server = Server()
```

- In an `as` import, the alias name is only an import-group label for conflict reporting and duplicate import validation. It is not a runtime or compile-time namespace binding. In the examples above, `http.Server()` is invalid.
- Two unaliased imports may coexist when their namespace-qualified public names differ:

```tya
import net/http/server
import net/tcp/server

http_server = net.http.Server()
tcp_server = net.tcp.Server()
```

- Two unaliased imports that would expose the same namespace-qualified public name are invalid.
- Two `as` imports that expose the same bare public name are invalid.
- An `as` import that exposes a bare public name conflicting with a local top-level binding is invalid.
- An unaliased namespaced import does not reserve the bare public name. A local `Server` binding may coexist with `net.http.Server`.
- An unaliased namespaced import does reserve every namespace segment needed for qualified access. A local top-level binding that conflicts with the first namespace segment is invalid:

```tya
import net/http/server

net = "local" # invalid: net is an import namespace
```

- Same-directory sibling class and interface visibility remains unchanged: files in the same directory package may still refer to sibling public classes by bare PascalCase name.
- Additional classes or interfaces in a class/interface file remain private to that file and are not exported through the import namespace.
- Script-file imports keep their existing module-style behavior unless they are explicitly class/interface files under the existing class-file rules.

## Scope
- Update `docs/SPEC.md` and `docs/ja/spec.md` import, directory package, class file, and standard-library examples.
- Update completed-design references only when they are linked from current docs or tests; do not rewrite historical feature specs except where tests require a new replacement fixture.
- Update parser/checker/resolver/runner/package synthesis as needed so unaliased class/interface imports create import-path namespace bindings instead of bare public-name bindings.
- Update code generation and runtime lookup for qualified imported class/interface names.
- Update import conflict diagnostics for namespace-qualified collisions, bare `as` collisions, duplicate imports, and local namespace-segment conflicts.
- Update LSP go-to-definition, completion, hover, and diagnostics for namespaced imported classes/interfaces.
- Migrate stdlib, examples, tests, and self-host sources from bare imported names to namespaced access.
- Add or update script tests around `import path/to/file`, `import path/to/*`, `as` bare imports, nested namespaces, conflict cases, and same-package sibling visibility.

## Out of Scope
- No new `module`, `namespace`, nested class, or `::` syntax.
- No selective per-name import list.
- No relative import syntax.
- No compatibility mode for the old bare unaliased wildcard behavior.
- No change to file naming rules: class/interface files remain `snake_case.tya` and public class/interface names remain PascalCase.
- No change to private companion class visibility.
- No change to same-directory sibling bare visibility inside a package.

## Acceptance Criteria
- `import cli/application` makes `cli.Application()` valid and `Application()` invalid.
- `import cli/result` makes `cli.Result()` valid and `Result()` invalid.
- `import net/http/server` makes `net.http.Server()` valid and `Server()` invalid.
- `import net/http/*` makes `net.http.Server()` and `net.http.Request()` valid, and makes `Server()` and `Request()` invalid.
- `import net/http/* as http` makes `Server()` and `Request()` valid, and makes `http.Server()` invalid.
- `import net/http/server as http` makes `Server()` valid, and makes `http.Server()` invalid.
- `import net/http/server` and `import net/tcp/server` can be used together as `net.http.Server()` and `net.tcp.Server()`.
- Two imports that would expose the same qualified public name fail with a clear import conflict.
- Two `as` imports that would expose the same bare public name fail with a clear import conflict.
- A local top-level binding can use `Server` when only `net.http.Server` is imported.
- A local top-level binding cannot use `net` when `net.http.Server` is imported.
- Existing same-package sibling references still work by bare PascalCase name.
- All stdlib, examples, tests, and self-host sources are migrated away from old unaliased bare import access.
- Documentation clearly states that this is a breaking replacement for previous bare package imports.

## Verification
```sh
gofmt -w internal/**/*.go cmd/**/*.go tests/**/*.go
go test ./tests -run TestV44Script -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```
