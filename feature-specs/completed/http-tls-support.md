# Feature: HTTP TLS Support

## Goal

Add HTTPS/TLS support to Tya's HTTP client and server so programs can make secure outbound requests and optionally serve HTTPS for local or production-like deployments.

## Context

- This is part of ROADMAP **Expand HTTP protocol coverage**.
- `stdlib/net/http/Client.tya` currently supports only `http://` URLs.
- `runtime/tya_http_server.c` currently serves plain HTTP over TCP.
- This is the hardest item in the group because it requires TLS library selection, linking, certificate handling, and cross-platform behavior.

## Behavior

- `http.Client.request`, `get`, and `post` accept `https://` URLs.
- TLS verification is enabled by default for client requests.
- Client options support:
  - `timeout`
  - `insecure_skip_verify`
  - optional `ca_file`
- `http.Server.run_tls(port, cert_file, key_file, options)` starts an HTTPS server.
- Server TLS options support a minimal stable set:
  - `host`
  - `timeout`
- Plain `run(port)` remains unchanged.
- TLS errors raise clear `http.tls:` or `http.request:` messages.
- Document the selected TLS backend and platform requirements.

## Scope

- Choose and integrate a TLS backend compatible with the C runtime and project goals.
- Update runtime networking/TLS code for client and server paths.
- Update `stdlib/net/http/Client.tya` URL scheme handling.
- Update `stdlib/net/http/Server.tya` with `run_tls`.
- Update CLI build/link logic for TLS dependencies.
- Add integration tests with a local self-signed HTTPS server/client setup.
- Update `docs/SPEC.md`, `docs/ja/spec.md`, and installation/doctor docs if dependencies are required.

## Out of Scope

- HTTP/2.
- Mutual TLS.
- Certificate generation commands.
- ACME/Let's Encrypt automation.
- WebSockets over TLS.
- TLS session tuning beyond secure defaults.

## Acceptance Criteria

- `http.Client.get("https://...")` works against a trusted HTTPS endpoint or a local test CA.
- `insecure_skip_verify: true` works only when explicitly set.
- `http.Server.run_tls(...)` serves a local HTTPS request.
- Plain HTTP behavior and tests remain unchanged.
- Docs explain TLS dependency requirements and security defaults.

## Verification

```sh
go test ./tests -run TestV58Scripts -count=1
go test ./... -count=1
```
