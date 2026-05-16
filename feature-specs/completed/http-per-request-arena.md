# Feature: HTTP Per-Request Arena

## Goal

Bound the memory used by `net/http.Server` request parsing and response preparation by introducing per-request cleanup for temporary C allocations in the HTTP runtime.

## Context

- This is part of ROADMAP **Expand HTTP protocol coverage**.
- `runtime/tya_http_server.c` contains comments documenting intentional per-request leaks from duplicated header/query strings.
- This work is implementation quality, not a public API change.
- It should become more important after keep-alive, because one process may handle many requests over long-lived connections.

## Behavior

- Introduce a request-scoped allocation mechanism for temporary HTTP parser strings and buffers.
- Free request-scoped allocations after each request has been dispatched and the response has been written.
- Do not free memory owned by Tya GC values or response bodies still needed during response writing.
- Keep request dictionary contents valid for the duration of handler execution.
- Preserve existing request/response public shapes.
- Add stress coverage that sends many requests and verifies memory does not grow without bound beyond a small tolerance.

## Scope

- Refactor `runtime/tya_http_server.c` temporary allocation paths:
  - request line parsing
  - query parsing
  - header parsing
  - cookie/multipart parsing if those have landed
- Add C-level helper functions inside the HTTP runtime file or a small companion file.
- Add tests or a focused script for repeated requests.
- Update comments documenting removed intentional leaks.

## Out of Scope

- Replacing the project-wide runtime allocator or GC.
- Changing Tya value lifetime semantics.
- Optimizing general string concatenation outside HTTP.
- Changing public `net/http` APIs.

## Acceptance Criteria

- Existing HTTP behavior is unchanged.
- Intentional per-request leak comments are removed or rewritten to describe cleanup.
- Repeated request tests pass without obvious unbounded memory growth.
- The implementation is compatible with one-request-per-connection and keep-alive modes.

## Verification

```sh
go test ./tests -run TestV58Scripts -count=1
go test ./... -count=1
```
