---
status: completed
goal_ready: false
---

# Feature: URL Parser Stdlib Extensions

## Goal

Strengthen the existing `url.Url` stdlib package into a reliable URL parser and
builder for HTTP clients, package tooling, and user programs.

## Context

Tya already has `stdlib/url/Url.tya` with `encode`, `decode`, `encode_query`,
`decode_query`, `parse`, and `build`. Existing tests cover a basic full URL and
query parsing. The next networking features need stricter behavior around
relative URLs, IPv6 hosts, normalization, query dictionaries, and error cases.

## Behavior

- Keep the class-style API under `url.Url`.
- Add or document:
  - `Url.parse(text)`
  - `Url.build(parts)`
  - `Url.resolve(base, ref)`
  - `Url.normalize(text)`
  - `Url.query_dict(query)`
  - `Url.encode_query(value)`
  - `Url.decode_query(text)`
- `parse` handles:
  - absolute URLs
  - relative references
  - host/port
  - username/password
  - IPv6 hosts in brackets
  - path, query, and fragment
- `build(parse(url))` round trips normalized supported URLs.
- `resolve(base, ref)` implements browser-style relative URL resolution for
  path/query/fragment references.
- Query parsing preserves duplicate keys by returning ordered pairs; helper
  `query_dict` can collapse into a dictionary where values with duplicates are
  arrays.
- Invalid percent escapes and malformed ports raise structured errors.
- Existing `Url.encode` / `Url.decode` behavior remains compatible.

## Scope

- `stdlib/url/Url.tya`
- `tests/stdlib_url_test.tya`
- `docs/STDLIB.md`
- next release docs
- `ROADMAP.md`

## Out of Scope

- Public suffix lists.
- IDNA/punycode in the first version.
- DNS lookup.
- HTTP client behavior.
- URLPattern-style route matching.

## Acceptance Criteria

- Existing URL stdlib tests remain green.
- IPv6 host URLs parse and build.
- Relative references resolve against a base URL.
- Query duplicate keys are preserved in ordered-pair form.
- Invalid percent escapes and invalid ports raise clear errors.
- HTTP client specs can rely on `url.Url.parse` for request URL handling.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Open Questions

None.
