# Tya v0.58 Specification

> **Status:** shipped. The `tya version` constant is `0.58.0`.
> v0.58 introduces the `net/http` standard module with a
> Sinatra-style HTTP/1.1 server. The rest of the language
> surface is unchanged from v0.57.

## Theme

v0.57 made it possible to bake assets into a tya binary; v0.58
adds a way to serve them. The first piece of `net/http` is a
single-threaded server in the spirit of Sinatra: routes are
registered with `.get` / `.post` / `.put` / `.delete`, path
parameters use `:name`, and each handler is just a tya function
that maps a request dict to a response dict.

HTTP client and richer server features (keep-alive, middleware,
multipart, cookies, threading) are queued for follow-up Epics
along with a tya template engine.

## Module

```tya
import net/http
```

`stdlib/net/http/Server.tya` defines `class Server`. The
unaliased binding follows tya's last-segment rule, so `http`
becomes the namespace. `stdlib/net/http/Client.tya` is reserved
for a future Epic.

## API

```tya
import net/http

app = http.Server()
app.get("/", _req -> { status: 200, body: "Hello, World" })
app.get("/users/:id", show_user)
app.post("/users", create_user)
app.run(8080)
```

| Method | Behaviour |
|--------|-----------|
| `http.Server()` | Instantiate. The class default constructor seeds `routes = []`. |
| `app.get(path, handler)` | Register a GET route. Returns `self` so calls can chain. |
| `app.post(path, handler)` | Register a POST route. |
| `app.put(path, handler)` | Register a PUT route. |
| `app.delete(path, handler)` | Register a DELETE route. |
| `app.run(port)` | Bind the port (`0` lets the OS pick a free port) and enter the accept loop. Blocking. |

When `app.run(0)` is used, the chosen port is printed to stderr
as `listening on N` so harnesses can latch on.

## Request shape

The handler receives a single dict:

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

`bytes_text(req["body"])` recovers UTF-8 text.

## Response shape

The handler returns a dict:

```tya
{
  status: 200,
  headers: { "content-type": "application/json" },
  body: "..."   # string or bytes
}
```

`headers` is optional; when omitted the server adds
`Content-Type: text/plain; charset=utf-8` and `Content-Length`.
A string body is sent as-is; a bytes body is sent verbatim.

Returning anything other than a dict surfaces as a 500 response
(`Handler returned non-dict`).

## Path matching

- Plain segments compare byte-for-byte.
- `:name` segments capture the corresponding request segment as
  a string and store it under `req["params"][name]`. Multiple
  captures (`/users/:uid/posts/:pid`) are supported.
- Wildcards and regex are out of scope for v0.58.
- No match → 404 `Not Found`.

## HTTP/1.1 support

| Feature | Status |
|---------|--------|
| GET / POST / PUT / DELETE | ✅ |
| Content-Length request body | ✅ |
| `Connection: close` (per-request) | ✅ default |
| Case-insensitive header names | ✅ (lowercased on the way in) |
| HEAD / PATCH / OPTIONS | ❌ scope-out |
| keep-alive | ❌ scope-out |
| Chunked transfer encoding | ❌ scope-out |
| Multipart bodies | ❌ scope-out |
| HTTPS / TLS | ❌ scope-out |
| Cookies | ❌ scope-out |
| Middleware | ❌ scope-out |
| Static file helper | ❌ scope-out |

## Concurrency

v0.58 is **single-threaded**: one connection is read, dispatched,
and responded to before the next is accepted. A slow handler
blocks every other client. This is deliberate — a personal site
or low-traffic API runs fine, and the limitation is documented
so users do not deploy it at scale. v0.59+ will introduce a
thread pool (or wire the existing `tya_task_new` machinery in)
so requests can run concurrently.

Limits enforced inside the C runtime:

| Limit | Value |
|-------|-------|
| Max header bytes per request | 16 KiB |
| Max body bytes per request | 10 MiB |

Exceeding either limit yields `400 Bad Request`.

## Implementation notes

- New `runtime/tya_http_server.c` (+ `tya_http_server.h`) builds
  a POSIX socket server, parses HTTP/1.1 by hand, and dispatches
  to tya handlers via the existing `tya_call1`. Windows is a
  build-time stub (`tya_panic` on entry).
- `tya_http_server_run(routes, port)` is exposed as a builtin
  via `internal/codegen/c.go::callBuiltin` and listed in
  `internal/checker/checker.go::BuiltinNames` as
  `http_server_run`.
- `stdlib/net/http/Server.tya` is the public class. It owns an
  array `self.routes` and forwards to the builtin in `run`.
- The C side intentionally leaks per-request string buffers (path,
  params, header names/values) because the runtime's `tya_string`
  stores the pointer by reference and the dict GC does not own
  the underlying bytes. A per-request arena or a GC-tracked string
  allocator is queued for v0.59+.

## Diagnostics

- `bind()` failure (port already in use, permission denied, etc.)
  raises a tya panic with the OS error message.
- The builtin's argument types are not statically checked — passing
  a non-array or non-number to `http_server_run` will panic at
  runtime.

## Scope-out (v0.59+)

- HTTP **client** (`http.Client` — `http.get(url)`, `http.post`).
- **Template engine** (`net/http/Template` or
  `net/http/template`) with `{{var}}` interpolation, partials.
- **Concurrency** via thread pool or `tya_task_new`.
- **keep-alive**, **chunked transfer encoding**,
  **multipart bodies**, **HTTPS/TLS**, **cookies**, **middleware**,
  **HEAD/PATCH/OPTIONS**.
- **Static file helper** — `app.static("/assets", embedded_dict)`
  using the v0.57 `embed` mechanism.
- **Windows** support via WinSock2.
- A per-request arena to bound the v0.58 memory leak.
