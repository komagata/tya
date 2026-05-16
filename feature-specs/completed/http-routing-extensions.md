---
status: completed
goal_ready: false
---

# Feature: HTTP Routing Extensions

## Goal

Extend `net/http.Server` routing so a small Tya web app can organize routes,
middleware, errors, and redirects cleanly while still compiling into a single
binary with the existing `net/http` server model.

## Context

Tya v0.58 introduced a minimal Sinatra-style HTTP server:

```tya
import net/http

app = http.Server()
app.get("/", _req -> { status: 200, body: "Hello" })
app.get("/users/:id", req -> { status: 200, body: req["params"]["id"] })
app.run(8080)
```

The current router supports GET, POST, PUT, DELETE, exact path segments, and
`:name` path parameters. It does not yet support middleware, route groups,
custom 404/500 handlers, route names, HEAD/PATCH/OPTIONS, wildcard segments,
or redirect helpers.

This feature focuses on routing ergonomics only. It should not take ownership
of unrelated HTTP v2 work such as keep-alive, multipart parsing, TLS, cookies,
or request concurrency.

## Behavior

- Keep existing route registration behavior compatible:

  ```tya
  app.get("/users/:id", handler)
  app.post("/users", handler)
  app.put("/users/:id", handler)
  app.delete("/users/:id", handler)
  ```

- Add method helpers:

  ```tya
  app.patch(path, handler)
  app.options(path, handler)
  app.head(path, handler)
  app.any(path, handler)
  app.route(method, path, handler)
  app.route(method, path, handler, options)
  ```

- `app.route` accepts uppercase or lowercase method names and stores methods
  uppercase.
- `app.any(path, handler)` matches every supported method.
- HEAD defaults to GET behavior with an empty response body when no explicit
  HEAD route matches.
- OPTIONS returns an `Allow` header for matching paths when no explicit OPTIONS
  route matches.
- Route registration remains chainable and returns `self`.

## Route Patterns

- Existing `:name` path parameters continue to work.
- Add wildcard tail segments:

  ```tya
  app.get("/assets/*path", handler)
  ```

  A wildcard must be the final segment and captures the remaining path under
  `req["params"]["path"]`.

- Add optional trailing slash matching through route options:

  ```tya
  app.get("/users", handler, { trailing_slash: "ignore" })
  ```

- Default trailing slash behavior remains strict to preserve v0.58 semantics.
- Duplicate parameter names in one pattern are rejected at registration time.
- Invalid route patterns raise a clear error before `run`.
- Matching order is deterministic:
  1. earlier registered routes win over later registered routes,
  2. exact/static segments and params follow registration order,
  3. wildcard routes only match when their registered position is reached.

## Route Groups

- Add `app.group(prefix, fn)`:

  ```tya
  app.group("/admin", group ->
    group.get("/", admin_index)
    group.get("/users/:id", admin_user)
  )
  ```

- A group prefixes every route registered inside the callback.
- Nested groups are supported.
- Group route registration remains chainable.
- Group prefixes are normalized so `"/admin"` + `"/users"` becomes
  `"/admin/users"`.
- Group-specific middleware applies only to routes inside the group.

## Middleware

- Add global and group middleware:

  ```tya
  app.use(logger)
  app.group("/admin", group ->
    group.use(require_admin)
    group.get("/", dashboard)
  )
  ```

- Middleware signature:

  ```tya
  middleware = req, next -> response
  ```

- `next.call(req)` invokes the next middleware or the route handler.
- Middleware may:
  - return a response directly,
  - modify `req` before calling `next`,
  - modify the response returned by `next`.
- Middleware runs in registration order from outermost/global to innermost/group.
- If middleware raises, the 500 handler path is used.
- Middleware for one group does not affect routes registered before entering the
  group or outside that group.

## Error and Fallback Handlers

- Add custom not-found and error handlers:

  ```tya
  app.not_found(req -> { status: 404, body: "missing" })
  app.error(err, req -> { status: 500, body: "error" })
  ```

- `not_found` handles unmatched routes.
- `error` handles handler or middleware raises.
- If a custom error handler itself raises or returns a non-dict, the server
  falls back to the existing simple 500 response.
- Default 404 and 500 behavior remains compatible with v0.58 when no custom
  handlers are registered.

## Named Routes and Redirects

- Route options may include a name:

  ```tya
  app.get("/users/:id", show_user, { name: "user" })
  ```

- `app.path(name, params)` builds a path:

  ```tya
  app.path("user", { id: "42" }) # "/users/42"
  ```

- Missing params raise a clear error.
- Extra params are ignored unless strict route-building is enabled.
- `app.redirect(path)` returns a 302 response.
- `app.redirect(path, status)` supports 301, 302, 303, 307, and 308.

## Request Shape

- Preserve existing request dictionary fields:
  - `method`,
  - `path`,
  - `params`,
  - `query`,
  - `headers`,
  - `body`.
- Add:
  - `route`: route metadata dictionary when matched,
  - `path_params`: alias of `params` for readability.
- Existing handlers that read `req["params"]` remain compatible.

## Scope

- `stdlib/net/http/Server.tya`
- HTTP runtime route dispatch in `runtime/tya_http_server.c` if route metadata
  needs C-side awareness.
- Checker/codegen builtin plumbing only if the route table shape passed to
  `http_server_run` changes.
- Tests for route registration, matching, groups, middleware, fallback handlers,
  named route generation, HEAD, OPTIONS, PATCH, wildcard routes, and invalid
  patterns.
- Example web app under `examples/http_demo/` showing grouped routes and
  middleware.
- `docs/STDLIB.md`
- Next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`

## Out of Scope

- Keep-alive.
- Chunked transfer encoding.
- Multipart/form-data parsing.
- Cookie and session helpers.
- HTTPS/TLS.
- Request concurrency or C10K runtime work.
- Static asset serving beyond keeping compatibility with the planned
  `Server.static` feature.
- HTTP client APIs.
- A full Rails-like framework, controllers, ORM, migrations, or view rendering.

## Acceptance Criteria

- Existing v0.58 `.get`, `.post`, `.put`, `.delete`, `:name` params, 404, and
  500 behaviors remain compatible.
- `patch`, `options`, `head`, `any`, and generic `route` registration work.
- HEAD falls back to matching GET routes and suppresses the response body.
- OPTIONS returns a correct `Allow` header when no explicit route handles it.
- Wildcard tail routes capture the rest of the path.
- Invalid route patterns fail before serving requests.
- Route groups prefix paths and compose nested prefixes correctly.
- Global and group middleware run in deterministic order.
- Middleware can short-circuit, mutate request dictionaries, and post-process
  responses.
- Custom 404 and 500 handlers override default responses.
- Named routes build paths from params and raise on missing params.
- `app.redirect` returns the documented status and `Location` header.
- Request dictionaries preserve `params` and include `path_params` for matched
  routes.
- `go test ./... -count=1` passes, including the self-host fixed-point tests.

## Verification

```sh
go test ./tests -run 'Test.*HTTP|TestV58|Test.*Route' -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
tya run examples/http_demo/routing.tya
```

## Dependencies

- Builds on the v0.58 `net/http.Server`.
- Should remain compatible with the asset-embedding static route PRD.
- Can ship before or after request concurrency work; routing behavior should not
  depend on concurrent request handling.

## Open Questions

None.
