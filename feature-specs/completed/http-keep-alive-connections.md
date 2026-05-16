# Feature: HTTP Keep-Alive Connections

## Goal

Support multiple HTTP/1.1 requests on one TCP connection in `net/http.Server` so clients can reuse connections instead of paying a new connection setup cost for every request.

## Context

- This is part of ROADMAP **Expand HTTP protocol coverage**.
- `runtime/tya_http_server.c` currently handles one request per connection and always writes `Connection: close`.
- Server-side chunked responses can be implemented before this, but keep-alive must preserve existing one-request behavior for clients that ask to close.

## Behavior

- HTTP/1.1 requests default to keep-alive unless the request has `Connection: close`.
- HTTP/1.0 requests default to close unless the request has `Connection: keep-alive`.
- The server reads, dispatches, and writes multiple requests on the same accepted socket until:
  - the client closes
  - either side requests close
  - a parse/read error occurs
  - a configured request limit is reached
- Add a conservative per-connection request limit to prevent unbounded loops.
- Responses include `Connection: keep-alive` or `Connection: close` according to the decision.
- Request dictionaries expose `req["keep_alive"]` as a boolean.
- Existing route behavior, middleware, static serving, HEAD, and OPTIONS semantics remain unchanged per request.

## Scope

- Update `runtime/tya_http_server.c` connection loop and response header decision.
- Add script tests using a raw socket client to send two requests over one connection.
- Test `Connection: close` ends reuse.
- Update `docs/SPEC.md` and `docs/ja/spec.md`.

## Out of Scope

- HTTP pipelining where a client sends multiple requests before reading responses.
- HTTP/2.
- TLS.
- Long-lived streaming protocols.
- Configurable timeout tuning beyond a conservative default.

## Acceptance Criteria

- A single TCP connection can receive two sequential HTTP/1.1 requests and produce two valid responses.
- `Connection: close` causes the server to close after the response.
- `req["keep_alive"]` matches the server decision.
- Existing single-request tests continue to pass.

## Verification

```sh
go test ./tests -run TestV58Scripts -count=1
go test ./... -count=1
```
