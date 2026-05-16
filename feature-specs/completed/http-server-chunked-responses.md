# Feature: HTTP Server Chunked Responses

## Goal

Allow `net/http.Server` handlers to send HTTP/1.1 chunked responses when the body length is not known up front, enabling simple streaming responses without buffering the entire body first.

## Context

- This is part of ROADMAP **Expand HTTP protocol coverage**.
- The HTTP client already decodes chunked responses.
- `runtime/tya_http_server.c` currently writes `Content-Length` and `Connection: close` for every response.

## Behavior

- Response dictionaries may request chunked transfer with `chunked: true`.
- When `chunked: true`, the runtime writes `Transfer-Encoding: chunked` and does not write `Content-Length`.
- Supported body sources:
  - array of strings/bytes, written as one chunk per item
  - channel yielding strings/bytes and closing when complete
- Empty chunks are skipped except for the final terminating chunk.
- Handler errors before response writing still use the existing 500 behavior.
- If a chunk source yields a non-string/non-bytes value, the server terminates the response stream and closes the connection.
- Non-chunked responses keep current `Content-Length` behavior.

## Scope

- Update response serialization in `runtime/tya_http_server.c`.
- Add any minimal runtime helpers needed to consume array/channel chunk sources.
- Add script tests with `curl --raw` or a socket client proving:
  - `Transfer-Encoding: chunked`
  - no `Content-Length`
  - correct chunk body order
  - non-chunked responses are unchanged
- Update `docs/SPEC.md` and `docs/ja/spec.md`.

## Out of Scope

- Request-side chunked decoding for the server.
- HTTP trailers.
- Compression.
- Server-Sent Events convenience APIs.
- WebSockets.
- Keep-alive support beyond what is required to send one chunked response.

## Acceptance Criteria

- A route returning `{status: 200, chunked: true, body: ["a", "b"]}` produces a valid chunked response body `ab`.
- Chunked responses omit `Content-Length`.
- Existing normal responses still include `Content-Length`.
- Tests cover string and bytes chunks.
- Specs document the response dictionary extension.

## Verification

```sh
go test ./tests -run TestV58Scripts -count=1
go test ./... -count=1
```
