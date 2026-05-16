# Feature: HTTP Multipart Request Bodies

## Goal

Add multipart/form-data parsing to `net/http.Server` request dictionaries so web handlers can receive form fields and uploaded files without manually parsing request bytes.

## Context

- This is part of ROADMAP **Expand HTTP protocol coverage**.
- `runtime/tya_http_server.c` already reads request bodies using `Content-Length` and exposes raw bytes as `req["body"]`.
- Request headers are lowercased in `req["headers"]`, so multipart parsing can detect `content-type`.
- Cookie support is easier and may land before this, but multipart parsing should not depend on cookies.

## Behavior

- For requests with `Content-Type: multipart/form-data; boundary=...`, the server populates:
  - `req["form"]`: dictionary of field name to string value
  - `req["files"]`: dictionary of field name to uploaded file metadata
- For non-multipart requests, `req["form"]` and `req["files"]` are empty dictionaries.
- File metadata dictionaries contain:
  - `filename`
  - `content_type`
  - `body`
  - `size`
- Multiple fields with the same name keep the last value.
- Multiple files with the same field name keep the last file for v1 of this feature.
- Multipart parsing preserves `req["body"]` as the original request body bytes.
- Parsing failures produce a `400 Bad Request` response before the handler runs.
- Enforce existing body-size limits. This feature does not introduce streaming uploads.

## Scope

- Update `runtime/tya_http_server.c` multipart parsing for compiled HTTP servers.
- Add tests under the active HTTP script test area for:
  - normal fields
  - one uploaded file
  - multiple fields
  - malformed boundary
  - non-multipart requests returning empty form/files dictionaries
- Update `docs/SPEC.md` and `docs/ja/spec.md`.

## Out of Scope

- Streaming multipart parsing.
- Multiple values per field or multiple files per field.
- Temporary file storage.
- Client-side multipart upload helpers.
- MIME sniffing.
- Increasing request body limits.

## Acceptance Criteria

- A compiled HTTP route can read `req["form"]["name"]`.
- A compiled HTTP route can read `req["files"]["avatar"]["filename"]`, `content_type`, `body`, and `size`.
- Raw `req["body"]` remains available.
- Malformed multipart requests return 400 and do not call the handler.
- English and Japanese specs document the behavior.

## Verification

```sh
go test ./tests -run TestV58Scripts -count=1
go test ./... -count=1
```
