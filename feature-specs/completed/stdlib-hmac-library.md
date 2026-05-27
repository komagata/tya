# Feature: stdlib HMAC Library

## Goal

Add HMAC helpers to the v1.0.0 standard library so Tya programs can sign and
verify messages using the existing digest surface without introducing a full
cryptography suite.

## Context

Tya already has `digest/Digest` for hash functions and
`secure_random/SecureRandom` for secure random bytes and UUIDs. Many practical
CLI, webhook, HTTP, and package-integrity workflows need keyed message
authentication. Full encryption, public-key cryptography, and certificate APIs
are broader than the v1.0.0 stdlib baseline, but HMAC is small and commonly
expected.

The accepted direction is a class-style `hmac/Hmac` package. It uses explicit
algorithm names, accepts string or bytes input, returns bytes by default, and
offers hex/base64 convenience helpers.

## Behavior

- `import hmac` exposes `hmac.Hmac`.
- `Hmac.digest(algorithm, key, message)` returns raw bytes.
  - `algorithm` must be one of `sha256`, `sha384`, or `sha512`.
  - `key` must be string or bytes.
  - `message` must be string or bytes.
  - String inputs are encoded as UTF-8 bytes.
- `Hmac.hexdigest(algorithm, key, message)` returns lowercase hexadecimal text.
- `Hmac.base64digest(algorithm, key, message)` returns base64 text.
- `Hmac.verify(algorithm, key, message, expected)` returns a bool.
  - `expected` may be raw bytes, lowercase/uppercase hex text, or base64 text
    when `options["encoding"]` is provided.
- `Hmac.verify(algorithm, key, message, expected, options = {})` supports:
  - `encoding`: `"raw"`, `"hex"`, or `"base64"`, default `"raw"` for bytes
    expected values and `"hex"` for string expected values.
- Verification uses constant-time comparison for equal-length byte sequences.
- Unsupported algorithms, unknown option keys, invalid encodings, wrong kinds,
  and malformed expected values raise structured errors with `kind: "crypto"`
  and stable `code` values.

## Scope

- `lib/hmac/Hmac.tya`
- runtime-backed HMAC intrinsics for interpreter and generated C
- integration with existing `digest`, `hex`, and `base64` behavior as needed
- `docs/SPEC.md`
- `docs/STRICT_SEMANTICS.md`
- runtime/codegen tests

## Out of Scope

- General encryption APIs such as AES.
- Public-key cryptography.
- Password hashing.
- Certificate validation APIs.
- Streaming HMAC contexts.
- Implicit algorithm selection.

## Acceptance Criteria

- `docs/SPEC.md` documents `hmac/Hmac` as part of the v1.0.0 stdlib surface.
- HMAC SHA-256, SHA-384, and SHA-512 produce known test-vector outputs.
- Raw bytes, hex, and base64 helpers work in interpreter and generated C.
- Verification uses constant-time comparison for byte sequences of equal
  length.
- Unsupported algorithms and malformed inputs raise structured crypto errors
  with stable codes.
- The spec clearly states that broader cryptography is outside v1.0.0.
- Existing self-host fixed-point gates remain valid.

## Tests To Add

Eval/runtime tests:

- `TestRunHmacKnownVectors`
  - Uses RFC-style SHA-256, SHA-384, and SHA-512 test vectors.
  - Expected: raw, hex, and base64 outputs match known values.

- `TestRunHmacStringAndBytesInputs`
  - Signs equivalent string and bytes inputs.
  - Expected: outputs match when text bytes are identical.

- `TestRunHmacVerifyEncodings`
  - Verifies raw bytes, hex text, and base64 text.
  - Expected: correct signatures return true; incorrect signatures return
    false.

- `TestRunHmacStructuredErrors`
  - Unsupported algorithm, malformed hex/base64 expected values, unknown
    option, and wrong kinds.
  - Expected: structured crypto errors with stable codes.

Codegen tests:

- `TestEmitCHmacProgram`
  - Builds and runs known vector and verification fixtures.

Testscript coverage:

- `v1_stdlib_hmac.txtar`
  - Covers CLI-level valid and invalid HMAC behavior.

Documentation tests:

- `TestSpecDocumentsHmacStdlib`
  - Expected: `docs/SPEC.md` lists `hmac/Hmac`, supported algorithms, return
    encodings, and out-of-scope broader crypto.

## Verification

```sh
go test ./internal/eval -run Hmac -count=1
go test ./internal/codegen -run Hmac -count=1
go test ./tests -run 'TestV.*Scripts|TestSpecDocumentsHmacStdlib|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
