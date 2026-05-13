---
status: approved
goal_ready: true
---

# Feature: Net IP Address Stdlib Library

## Goal

Add a `net/ip` standard library package for parsing, normalizing, comparing,
and classifying IPv4 and IPv6 addresses and CIDR networks.

## Context

Networking APIs need a shared representation for addresses. Keeping IP parsing
in its own stdlib package avoids duplicating address logic across socket,
HTTP, URL, and future DNS support.

## Behavior

- Add `stdlib/net/ip/Address.tya`.
- Add `stdlib/net/ip/Network.tya` if CIDR networks are clearer as a separate
  class.
- Public APIs:
  - `ip.Address.parse(text)`
  - `ip.Address.valid?(text)`
  - `ip.Address.loopback?(addr)`
  - `ip.Address.private?(addr)`
  - `ip.Address.unspecified?(addr)`
  - `ip.Address.to_s(addr)`
  - `ip.Address.version(addr)` returns `4` or `6`
  - `ip.Network.parse(cidr)`
  - `ip.Network.contains?(network, addr)`
- IPv4 dotted decimal is supported.
- IPv6 compressed and full forms are supported.
- IPv4-mapped IPv6 addresses are accepted and normalized deterministically.
- CIDR parsing supports IPv4 and IPv6 prefix lengths.
- Invalid addresses raise structured errors from `parse` and return `false`
  from `valid?`.

## Scope

- `stdlib/net/ip/`
- runtime/native support only if pure Tya parsing is not practical
- `docs/STDLIB.md`
- next release docs
- stdlib tests for IPv4, IPv6, CIDR, classification, and invalid input
- `ROADMAP.md`

## Out of Scope

- DNS resolution.
- MAC addresses.
- Public suffix / domain parsing.
- GeoIP.
- IP arithmetic beyond CIDR containment.

## Acceptance Criteria

- IPv4 parse/format round trips for common addresses.
- IPv6 parse/format round trips for full and compressed addresses.
- CIDR containment works for IPv4 and IPv6.
- `private?`, `loopback?`, and `unspecified?` match conventional ranges.
- Invalid addresses fail deterministically.
- Socket APIs can use the same address representation later.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Open Questions

None.
