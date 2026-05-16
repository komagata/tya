# Feature: Windows Socket Support via WinSock2

## Goal

Make `net/socket` and the HTTP client/server runtime work on Windows by replacing the current Windows "not supported" stubs with WinSock2-backed implementations.

## Context

- This is part of ROADMAP **Expand HTTP protocol coverage**.
- `runtime/tya_runtime.c` currently raises `net/socket: not supported on Windows` for socket operations under `_WIN32`.
- `net/http.Client` and `net/http.Server` depend on socket/runtime support.

## Behavior

- Implement Windows support for:
  - `socket_connect`
  - `socket_server_listen`
  - `socket_server_accept`
  - `socket_read`
  - `socket_read_line`
  - `socket_write`
  - `socket_close`
  - `socket_closed`
  - local/remote address helpers
  - server close/local address helpers
- Initialize and clean up WinSock2 safely.
- Preserve existing Tya-level `net/socket` API behavior.
- Keep POSIX behavior unchanged.
- Ensure generated C links required Windows socket libraries when building for Windows.

## Scope

- Update `runtime/tya_runtime.c` and `runtime/tya_runtime.h`.
- Update build/link flag logic in the CLI C build path if needed.
- Add Windows-specific tests or CI workflow coverage where available.
- Document Windows socket support in `docs/SPEC.md` and `docs/ja/spec.md` if the stdlib support matrix mentions platform support.

## Out of Scope

- TLS.
- IPv6 expansion beyond parity with current POSIX behavior.
- Asynchronous IOCP integration.
- HTTP keep-alive or chunked response behavior.

## Acceptance Criteria

- `net/socket` examples compile and run on Windows.
- `net/http.Client.get("http://...")` works on Windows for a local HTTP server.
- A compiled `net/http.Server` can accept a local request on Windows.
- POSIX socket tests still pass.
- Cross-platform build logic links WinSock2 only on Windows.

## Verification

```sh
go test ./... -count=1
# On Windows CI or a Windows host:
go test ./tests -run 'TestNetSocketScripts|TestV58Scripts' -count=1
```
