---
status: approved
goal_ready: true
---

# Feature: Net Socket Stdlib Library

## Goal

Add a low-level `net/socket` standard library package for TCP client/server
sockets so Tya programs can build network protocols below `net/http`.

## Context

Tya already has `net/http` server work and planned HTTP client support, but no
general socket API. A small socket layer is useful for custom protocols, local
dev tools, test servers, and as a foundation for future stdlib networking.

## Behavior

- Add `stdlib/net/socket/Socket.tya`.
- Add `stdlib/net/socket/Server.tya` if a separate listener class keeps the API
  clearer.
- TCP client:
  - `socket.Socket.connect(host, port)`
  - `socket.Socket.connect(host, port, options)`
- TCP server:
  - `socket.Server.listen(host, port)`
  - `socket.Server.listen(host, port, options)`
  - `server.accept()`
  - `server.close()`
- Socket instance methods:
  - `read(size)`
  - `read_line()`
  - `write(value)`
  - `write_line(value)`
  - `close()`
  - `closed?()`
  - `local_address()`
  - `remote_address()`
- `value` may be `string` or `bytes`; binary reads return `bytes` when opened
  with `{ mode: "binary" }`.
- Timeouts are configurable with `{ timeout: seconds }`.
- Connection, DNS, timeout, and closed-socket failures raise structured errors.
- Server sockets bind only TCP in the first version.

## Scope

- `stdlib/net/socket/`
- runtime/native socket support
- `docs/STDLIB.md`
- next release docs
- script tests using localhost
- `ROADMAP.md`

## Dependencies

- Implement `docs/prd/stdlib-net-ip-library.md` first if socket address values
  should reuse the shared IP address representation.
- Implement `docs/prd/stdlib-io-stream-library.md` first if socket reads and
  writes should share stream abstractions.

## Out of Scope

- UDP.
- TLS.
- Unix domain sockets.
- Non-blocking/event-loop APIs.
- WebSocket.
- HTTP behavior; use `net/http` for HTTP.

## Acceptance Criteria

- A Tya program can start a TCP server on localhost, accept one connection, read
  a line, write a response, and close cleanly.
- A Tya program can connect to that server with `socket.Socket.connect`.
- Read/write works for text and bytes.
- Timeout and connection-refused failures produce clear structured errors.
- `net/http` tests continue to pass.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run TestV.*Script -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Open Questions

None.
