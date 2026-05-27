# Feature: HTTP Cookie Support

## Goal

Add first-class cookie parsing and response helpers to `net/http` so Tya web handlers can read request cookies and set response cookies without manually parsing `Cookie` headers or formatting `Set-Cookie` strings.

## Context

- `ROADMAP.md` tracks **Expand HTTP protocol coverage** as post-v1.0.0 unless a supported release use case needs it.
- The easiest implementation order for that roadmap group is:
  1. cookies
  2. multipart request bodies
  3. server-side chunked transfer encoding
  4. keep-alive
  5. per-request arena for HTTP server buffers
  6. Windows socket support via WinSock2
  7. HTTPS/TLS
- `runtime/tya_http_server.c` already parses request headers into `req["headers"]` with lowercase header names.
- `lib/net/http/Server.tya` already exposes response helpers such as `redirect(path, status)`.
- Response dictionaries already support a `headers` dictionary, but repeated `Set-Cookie` headers need explicit support because a plain dictionary cannot safely represent multiple headers with the same name.

## Behavior

- Request dictionaries gain `cookies`:
  - `req["cookies"]` is a dictionary of cookie name to cookie value.
  - If the request has no `Cookie` header, `req["cookies"]` is an empty dictionary.
  - Cookie names and values are trimmed around optional whitespace.
  - Empty cookie names are ignored.
  - Repeated cookie names keep the last value in header order.
  - Cookie values are not URL-decoded automatically.
  - Malformed cookie pairs without `=` are ignored.
- Add `http.Server.cookie(name, value, options)`:
  - returns a correctly formatted `Set-Cookie` header value string.
  - `options` is optional.
  - supported options:
    - `path`
    - `domain`
    - `max_age`
    - `expires`
    - `secure`
    - `http_only`
    - `same_site`
  - `same_site` accepts `Lax`, `Strict`, and `None`.
  - `SameSite=None` requires `secure: true`; otherwise the helper raises `http.cookie: SameSite=None requires Secure`.
  - Cookie names must be non-empty and must not contain control characters, spaces, `=`, `;`, or `,`.
  - Cookie values must not contain control characters, `;`, or `,`.
- Add `http.Server.with_cookie(response, name, value, options)`:
  - returns the same response dictionary after appending one `Set-Cookie` value.
  - preserves existing `status`, `body`, and headers.
  - supports adding multiple cookies to one response.
- Response dictionaries may include repeated headers through a reserved `header_values` dictionary:
  - `header_values["Set-Cookie"]` is an array of `Set-Cookie` header value strings.
  - The runtime writes each array entry as a separate `Set-Cookie` header line.
  - Existing single-value `headers` behavior remains unchanged.
- Example:

```tya
import net/http as http

app = http.Server()

app.get("/login", req ->
  resp = { status: 200, body: "ok" }
  return app.with_cookie(resp, "session", "abc123", {
    path: "/",
    http_only: true,
    secure: true,
    same_site: "Lax"
  })
)

app.get("/me", req ->
  session = req["cookies"].get("session", "")
  return { status: 200, body: session }
)
```

## Scope

- Update `runtime/tya_http_server.c` and `runtime/tya_http_server.h` so compiled HTTP servers:
  - populate `req["cookies"]`
  - write repeated response headers from `header_values`
- Update `lib/net/http/Server.tya` with cookie helper methods.
- Add focused script tests under `tests/testdata/v58_http/` or a newer HTTP test directory covering:
  - parsing a `Cookie` request header into `req["cookies"]`
  - no-cookie requests producing an empty cookie dictionary
  - multiple cookies in one request
  - `with_cookie` writing one `Set-Cookie` header
  - multiple `with_cookie` calls writing multiple `Set-Cookie` headers
  - validation failures for invalid names, values, and `SameSite=None` without `Secure`
- Add unit tests if lower-level parsing/formatting is factored into Go or C helpers.
- Update `docs/SPEC.md` and `docs/ja/spec.md` in matching English/Japanese sections for the new request and response behavior.
- Update `ROADMAP.md` only after implementation by checking the cookies sub-scope or linking it to the completed spec.

## Out of Scope

- Cookie signing, encryption, sessions, CSRF helpers, or storage abstractions.
- Automatic URL decoding or encoding of cookie values.
- Client-side cookie jars for `http.Client`.
- Multipart parsing.
- Keep-alive.
- Server-side chunked response streaming.
- HTTPS/TLS.
- Windows socket support.
- Per-request arena work for existing HTTP server allocation leaks.

## Acceptance Criteria

- A compiled `net/http.Server` route can read `req["cookies"]["name"]` from an incoming `Cookie` header.
- Requests without cookies receive `req["cookies"] == {}`.
- `app.cookie(...)` formats `Set-Cookie` values with the documented attributes and validation.
- `app.with_cookie(...)` can add one or more cookies to a normal response dictionary.
- Runtime response writing emits repeated `Set-Cookie` header lines when `header_values["Set-Cookie"]` contains multiple entries.
- Existing `headers` dictionary behavior stays backward compatible.
- Invalid cookie names, invalid values, unsupported `same_site`, and `SameSite=None` without `Secure` have focused tests.
- English and Japanese specs document the request `cookies` field and response cookie helper.

## Verification

```sh
go test ./tests -run TestV58Scripts -count=1
go test ./... -count=1
```
