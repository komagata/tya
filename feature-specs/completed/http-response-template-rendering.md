# Feature: HTTP Response Template Rendering

## Goal

`net/http.Server` handlers can return rendered HTML responses with `app.render(...)` and `app.render_html(...)`, using the existing `template.Template` renderer from either template files or embedded template sources. Small web apps should not need to hand-wire `Template.render_file`, response headers, and response dictionaries in every handler.

## Context

- `ROADMAP.md` listed HTTP response template integration as complete under `Finish net/http v2`, but `lib/net/http/Server.tya` does not expose template response helpers and the v58 HTTP tests do not cover them.
- `lib/template/Template.tya` already provides `Template.render(source, data, options)`, `Template.render_html(source, data)`, and `Template.render_file(path, data, options)`.
- `lib/net/http/Server.tya` already has response helper precedent with `redirect(path, status)`, route helpers, middleware, custom error handlers, `dispatch`, and static assets.

## Behavior

- Add instance helpers to `net/http.Server`:
  - `app.render(template, data, options)`
  - `app.render_html(template, data, options)`
- `options` is optional. When omitted or `nil`, defaults are used.
- Both helpers return a response dictionary suitable for a route handler:
  - `status`: defaults to `200`
  - `headers`: includes `Content-Type: text/html; charset=utf-8` by default
  - `body`: rendered template text
- Supported `options` keys:
  - `status`: response status code
  - `headers`: extra response headers merged into the response
  - `content_type`: overrides the default `Content-Type`
  - `template_options`: options passed to `Template.render`, `Template.render_html`, or `Template.render_file`
- Template input supports both file paths and embedded/source templates:
  - If `template` is a string that names an existing file, render with `Template.render_file(template, data, template_options)`.
  - Otherwise, treat `template` as template source and render with `Template.render(template, data, template_options)`.
  - Embedded byte assets are accepted as template source after UTF-8 text conversion.
- `app.render_html(...)` has the same input and response behavior as `app.render(...)`, but renders with HTML escaping enabled. If `template_options` is present, `render_html` still forces HTML escaping.
- Custom `headers` are merged after defaults, so callers may intentionally override `Content-Type` through either `content_type` or `headers["Content-Type"]`.
- Template render errors, missing file errors, and invalid embedded byte text errors raise normally and are handled by the existing server error handler path.
- Example file-template handler:

```tya
import net/http as http

app = http.Server()
app.get("/users/:id", req ->
  return app.render("views/user.html", { id: req["path_params"]["id"] })
)
```

- Example embedded-template handler:

```tya
import net/http as http

embed "views/user.html" as user_view

app = http.Server()
app.get("/users/:id", req ->
  return app.render_html(user_view, { id: req["path_params"]["id"] })
)
```

## Scope

- Update `lib/net/http/Server.tya` to expose `render` and `render_html`.
- Use the existing `lib/template/Template.tya`; do not add a new template syntax or renderer.
- Add focused HTTP stdlib tests that cover:
  - file template rendering
  - embedded/source template rendering
  - `render_html` escaping
  - default response status, headers, and body
  - custom `status`, `headers`, and `content_type`
  - error handling through the existing server error handler
- Add or update a script test under the current HTTP test area if needed to prove the behavior through `tya run` or the compiled runtime path.
- Document the public helper API in `docs/SPEC.md` and `docs/ja/spec.md`.
- Update `ROADMAP.md` after implementation so the HTTP template integration task is checked only when these tests and docs exist.

## Out of Scope

- Implicit response dictionaries such as `{ template: "...", data: ... }`.
- View directory conventions, layout inheritance, automatic partial discovery, template caching, and development reload.
- New template syntax or changes to `template.Template` semantics beyond what the HTTP helpers require.
- Broader HTTP protocol work: keep-alive, server-side chunked transfer encoding, TLS, cookies, multipart request bodies, and per-request arenas.

## Acceptance Criteria

- `net/http.Server` exposes `render` and `render_html` helpers with the behavior above.
- A route handler can return `app.render("views/page.html", data)` and receive a `200` HTML response with the rendered file body.
- A route handler can return `app.render_html(embedded_template, data)` and receive escaped HTML from an embedded/source template.
- Callers can customize status and headers without losing the default HTML content type unless they override it.
- Render failures are covered by a test that demonstrates the existing `app.error(...)` handler path still handles them.
- `docs/SPEC.md` and `docs/ja/spec.md` describe the helper API in matching English and Japanese sections.
- `ROADMAP.md` no longer marks HTTP template integration complete before the implementation exists, and is checked again only after the feature is implemented and verified.

## Verification

```sh
go test ./tests -run TestV58Scripts -count=1
go test ./tests -run TestStdlib -count=1
go test ./... -count=1
```
