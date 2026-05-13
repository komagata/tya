---
status: approved
goal_ready: true
---

# Feature: Compress Stdlib Library

## Goal

Add a `compress` standard library package for common gzip/zlib compression and
decompression of strings, bytes, and files.

## Context

Tya already has bytes, base64, hex, digest, file I/O, and asset embedding work
on the roadmap. Compression is a common companion for package artifacts, static
assets, HTTP responses, and data files. The first version should stay narrow
and use broadly available formats.

## Behavior

- Add `stdlib/compress/Compress.tya`.
- Public APIs:
  - `compress.Compress.gzip(value)`
  - `compress.Compress.gunzip(bytes)`
  - `compress.Compress.zlib(value)`
  - `compress.Compress.unzlib(bytes)`
  - `compress.Compress.gzip_file(src, dst)`
  - `compress.Compress.gunzip_file(src, dst)`
- `value` may be a `string` or `bytes`.
- Compression returns `bytes`.
- Decompression returns `bytes`; callers can convert to text explicitly.
- Invalid compressed data raises a structured error.
- File helpers stream when possible and do not require loading large files into
  memory.
- Asset-embedding transforms may reuse this package later.

## Scope

- `stdlib/compress/Compress.tya`
- runtime/native support if pure Tya implementation is not practical
- `docs/STDLIB.md`
- next release docs
- tests for gzip/zlib round trips, invalid input, and file helpers
- `ROADMAP.md`

## Dependencies

- Implement `docs/prd/stdlib-io-stream-library.md` first if file helpers need
  streaming rather than whole-file reads.

## Out of Scope

- zip/tar archives.
- brotli, zstd, lz4, bzip2, xz.
- Streaming compression API in the first version.
- Automatic HTTP compression negotiation.
- Checksums beyond those required by gzip/zlib formats.

## Acceptance Criteria

- `import compress` exposes `compress.Compress`.
- String and bytes inputs gzip/gunzip round trip.
- zlib/unzlib round trips.
- Invalid input raises a clear error.
- File gzip/gunzip helpers round trip a file.
- Existing `base64`, `hex`, `digest`, and `file` tests remain green.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Open Questions

None.
