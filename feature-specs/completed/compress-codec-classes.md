# Feature: Compress Codec Classes

## Goal

Replace the monolithic `compress.Compress` API with format-specific codec
classes so gzip and zlib behavior lives behind object-oriented classes with a
shared minimal interface.

## Context

`stdlib/compress/Compress.tya` currently stores an optional value and exposes
all gzip and zlib operations from one class:

```tya
compress.Compress("hello").gzip_compress()
compress.Compress(data).gzip_decompress()
compress.Compress("hello").zlib_compress()
compress.Compress(data).zlib_decompress()
compress.Compress(src).gzip_compress_file(dst)
```

The gzip and zlib methods share the same structure but differ only by format.
This makes `Compress` a mixed facade rather than a format-specific object.

The new API should expose separate codec classes:

```tya
compress.Gzip().compress("hello")
compress.Gzip().decompress(data)
compress.Gzip().compress_file(src, dst)
compress.Gzip().decompress_file(src, dst)

compress.Zlib().compress("hello")
compress.Zlib().decompress(data)
compress.Zlib().compress_file(src, dst)
compress.Zlib().decompress_file(src, dst)
```

Compatibility with `compress.Compress` is not required.

## Behavior

- Remove `compress.Compress` as a public API.
- Add a public `compress.Codec` interface with the minimal byte/string codec
  contract:

```tya
interface Codec
  compress = value ->
  decompress = value ->
```

- Add a public `compress.Gzip` class that implements `Codec`.
- Add a public `compress.Zlib` class that implements `Codec`.
- `Gzip().compress(value)` compresses text or bytes using gzip and returns
  bytes.
- `Gzip().decompress(value)` decompresses gzip bytes and returns bytes.
- `Zlib().compress(value)` compresses text or bytes using zlib and returns
  bytes.
- `Zlib().decompress(value)` decompresses zlib bytes and returns bytes.
- Invalid compressed input keeps the existing structured error behavior from
  the current compression builtins.
- File APIs are concrete class methods, not part of `Codec`:

```tya
compress.Gzip().compress_file(src, dst)
compress.Gzip().decompress_file(src, dst)
compress.Zlib().compress_file(src, dst)
compress.Zlib().decompress_file(src, dst)
```

- `compress_file(src, dst)` reads the full source file, compresses it with the
  receiver format, writes the compressed bytes to `dst`, and returns `nil`.
- `decompress_file(src, dst)` reads the full source file, decompresses it with
  the receiver format, writes the decompressed bytes to `dst`, and returns
  `nil`.
- File helpers close files they open, including on normal completion. If the
  current stdlib lacks `try/finally` coverage needed for error-path closing,
  preserve current error behavior and keep robust cleanup as a separate task.
- Gzip and zlib file helpers should share private helper code where practical
  instead of duplicating read-all/write-all logic.
- Existing aliases such as `gzip`, `gunzip`, `zlib`, `unzlib`,
  `gzip_compress`, `gzip_decompress`, `zlib_compress`, `zlib_decompress`,
  `gzip_file`, and `gunzip_file` are not preserved on a compatibility facade.
- Tests, docs, and examples should use the new `Gzip` and `Zlib` class APIs.

## Scope

- Standard library:
  - remove or stop exporting `stdlib/compress/Compress.tya`;
  - add `stdlib/compress/Gzip.tya`;
  - add `stdlib/compress/Zlib.tya`;
  - add shared private helper code for file read/write if it can fit existing
    package visibility rules cleanly.
- Tests:
  - migrate `tests/stdlib_compress_test.tya` to `Gzip`, `Zlib`, and `Codec`;
  - add zlib file round-trip coverage;
  - keep invalid gzip and invalid zlib input coverage.
- Documentation:
  - update `docs/SPEC.md` examples that mention `Compress().gzip(value)`;
  - regenerate stdlib API docs if the documented public API changes;
  - update any stdlib docs or generated indexes that list `Compress`.
- Package behavior:
  - ensure `import compress as compress` exposes `compress.Gzip`,
    `compress.Zlib`, and `compress.Codec`;
  - ensure old `compress.Compress` is not documented as public API.

## Out of Scope

- Preserving `compress.Compress` compatibility.
- Adding compatibility aliases for old method names.
- Adding stream compression or decompression APIs.
- Adding reader/writer-based compression APIs.
- Adding file APIs to the `Codec` interface.
- Changing low-level compression builtins beyond what is necessary to support
  the new stdlib class layout.
- Changing compression formats, compression levels, gzip metadata, zlib
  options, or file naming conventions.
- Guaranteeing cleanup on error paths beyond the current stdlib capabilities.

## Acceptance Criteria

- `compress.Gzip().compress("hello")` returns gzip-compressed bytes.
- `compress.Gzip().decompress(data)` round-trips data produced by
  `Gzip().compress`.
- `compress.Zlib().compress("hello")` returns zlib-compressed bytes.
- `compress.Zlib().decompress(data)` round-trips data produced by
  `Zlib().compress`.
- `compress.Gzip().compress_file(src, dst)` writes a gzip-compressed file.
- `compress.Gzip().decompress_file(src, dst)` restores the original file
  contents.
- `compress.Zlib().compress_file(src, dst)` writes a zlib-compressed file.
- `compress.Zlib().decompress_file(src, dst)` restores the original file
  contents.
- `Gzip` and `Zlib` implement `Codec`.
- Invalid gzip input through `Gzip().decompress` raises.
- Invalid zlib input through `Zlib().decompress` raises.
- `tests/stdlib_compress_test.tya` uses only `compress.Gzip`,
  `compress.Zlib`, and `compress.Codec` where relevant.
- Generated stdlib docs no longer list `compress.Compress` as the primary
  compression API.
- Repository examples and spec docs no longer recommend
  `Compress().gzip(value)`.
- Stream compression remains absent from the public API.

## Verification

```sh
go test ./internal/doc -count=1
go test ./internal/eval -run TestRunStdlib -count=1
go test ./tests -run 'TestStdlibCompress|TestV65Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
