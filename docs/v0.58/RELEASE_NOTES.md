---
layout: doc
title: Release Notes
permalink: /v0.58/release-notes/
---

# Tya v0.58 Release Notes

> **Status:** shipped. `tya version` reports `0.58.0` and
> `ROADMAP.md` carries the matching `Released` entry.

## TL;DR

v0.58 introduces the **`net/http` standard module** with a
Sinatra-style HTTP/1.1 server:

```tya
import net/http

app = http.Server()
app.get("/", _req -> { status: 200, body: "Hello, World" })
app.get("/users/:id", req -> { status: 200, body: req["params"]["id"] })
app.run(8080)
```

v0.57 made it possible to bake assets into a tya binary. v0.58
adds a way to serve them. The HTTP client and richer server
features (keep-alive, middleware, multipart, cookies, threading)
are queued for follow-up Epics along with a tya template engine.

The language surface is unchanged from v0.57.

## What's new

### `import net/http`

The first multi-segment standard module. `stdlib/net/http/Server.tya`
defines `class Server`; the unaliased binding follows tya's
last-segment rule, so `http` becomes the namespace.
`stdlib/net/http/Client.tya` is reserved for a future Epic.

### Sinatra-style routes

```tya
app = http.Server()
app.get("/", show_home)
app.get("/users/:id", show_user)
app.post("/users", create_user)
app.put("/users/:id", update_user)
app.delete("/users/:id", destroy_user)
```

`.get` / `.post` / `.put` / `.delete` register a route and return
`self`, so calls can chain. `:name` segments capture a single
path segment and surface as `req["params"][name]`.

### Request / response shape

Every handler receives a single dict:

```tya
{
  method: "GET",                # always uppercase
  path: "/users/42",            # path without query string
  params: { id: "42" },         # path-pattern captures
  query: { limit: "10" },       # parsed query string
  headers: { ... },             # header names are lowercased
  body: <bytes>                 # request body bytes (empty for GET)
}
```

and returns a dict:

```tya
{
  status: 200,
  headers: { "content-type": "application/json" },
  body: "..."   # string or bytes
}
```

`headers` is optional; when omitted the server adds
`Content-Type: text/plain; charset=utf-8` and `Content-Length`.

### `app.run(0)` for OS-picked ports

`app.run(0)` lets the kernel assign a free port and prints
`listening on N` to stderr so test harnesses can latch on.

### Diagnostics

- `bind()` failure (port already in use, permission denied, etc.)
  raises a tya panic with the OS error message.
- Returning anything other than a dict from a handler surfaces as
  `500 Handler returned non-dict`.

## HTTP/1.1 support matrix

| Feature | Status |
|---------|--------|
| GET / POST / PUT / DELETE | ✅ |
| Content-Length request body | ✅ |
| `Connection: close` (per-request) | ✅ default |
| Case-insensitive header names | ✅ |
| HEAD / PATCH / OPTIONS | ❌ scope-out |
| keep-alive | ❌ scope-out |
| Chunked transfer encoding | ❌ scope-out |
| Multipart bodies | ❌ scope-out |
| HTTPS / TLS | ❌ scope-out |
| Cookies | ❌ scope-out |
| Middleware | ❌ scope-out |
| Static file helper | ❌ scope-out |
| Windows (WinSock2) | ❌ scope-out |

## Concurrency

v0.58 is **single-threaded**: one connection is read, dispatched,
and responded to before the next is accepted. A slow handler
blocks every other client. This is deliberate — a personal site
or low-traffic API runs fine, and the limitation is documented so
users do not deploy it at scale. v0.59+ will introduce a thread
pool (or wire the existing `tya_task_new` machinery in) so
requests can run concurrently.

Limits enforced inside the C runtime:

| Limit | Value |
|-------|-------|
| Max header bytes per request | 16 KiB |
| Max body bytes per request | 10 MiB |

Exceeding either limit yields `400 Bad Request`.

## Examples

`examples/http_demo/` has working samples:

- `hello.tya` — single `GET /` route.
- `api.tya` — path params + POST body echo.

## Migration

`import net/http` is the first multi-segment stdlib module. The
existing directory-as-package loader (v0.44) handles the resolver
side with no new wiring. Programs that already use single-segment
imports are unaffected.

## Tooling

- 4 new fixtures under `tests/testdata/v58_http/` cover GET, path
  params, POST + Content-Length, and 404.
- `runtime/tya_http_server.c` (+ `.h`) ships as a runtime module,
  copied into `share/tya/runtime/` by `scripts/build_release_packages.sh`.
- `Formula/tya.rb` → `0.58.0`,
  `editors/vscode/package.json` → `0.58.0`.

## Known limitations

- The C runtime intentionally leaks per-request string buffers
  because `tya_string` stores the pointer by reference and the
  dict GC does not own the underlying bytes. A per-request arena
  or a GC-tracked string allocator is queued for v0.59+.
- The builtin's argument types are not statically checked —
  passing a non-array or non-number to `http_server_run` will
  panic at runtime.
- Windows is a build-time stub (`tya_panic` on entry).

## Next

- HTTP **client** (`http.Client` — `http.get(url)`, `http.post`).
- **Template engine** with `{{var}}` interpolation, partials.
- **Concurrency** via thread pool or `tya_task_new`.
- **keep-alive**, **chunked transfer encoding**,
  **multipart bodies**, **HTTPS/TLS**, **cookies**, **middleware**,
  **HEAD/PATCH/OPTIONS**.
- **Static file helper** — `app.static("/assets", embedded_dict)`
  using the v0.57 `embed` mechanism.
- **Windows** support via WinSock2.
- A per-request arena to bound the v0.58 memory leak.
