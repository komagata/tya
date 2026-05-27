---
status: completed
goal_ready: false
---

# Feature: Binary Stdlib Library

## Goal

Add a standard `binary` library for endian-aware reading and writing of numeric
values from `bytes`, so asset loaders, image/audio/font tooling, network
protocol helpers, and native-library bindings can share one small binary-data
foundation.

## Context

Tya already has a `bytes` value type, byte literals, byte conversion builtins,
binary file I/O, and binary stream modes in `io` and `net/socket`. Those APIs
move raw bytes around, but callers still need a reusable way to interpret
structured bytes as integers and floats.

The public API should follow the stdlib class-style direction. Cursor-based
readers and writers should be class instances, not dictionaries.

## Behavior

- Add a public `binary` stdlib package.
- Import shape:

  ```tya
  import binary

  reader = binary.Reader.new(b"\x34\x12", { endian: "little" })
  value = reader.read_u16()

  writer = binary.Writer.new({ endian: "big" })
  writer.write_u16(0x1234)
  out = writer.bytes()
  ```

- Public classes:
  - `binary.Reader`
  - `binary.Writer`
- `Reader.new(data)` and `Reader.new(data, options)` return `Reader`
  instances.
- `Writer.new()` and `Writer.new(options)` return `Writer` instances.
- Options are dictionaries. Supported options:
  - `{ endian: "big" }`
  - `{ endian: "little" }`
- Default endian is `"big"`.
- Methods without an endian suffix use the instance default endian.
- Methods with `_le` or `_be` suffixes override the instance default endian.
- Reads past the end of the input raise a clear `binary` error.
- Invalid endian names, invalid seek positions, negative byte counts, and
  numeric values outside the target integer range raise clear `binary` errors.

## Reader

- `Reader.new(data)` creates a reader over a `bytes` value with big-endian
  default order.
- `Reader.new(data, options)` creates a reader with explicit options.
- `reader.position()` returns the current byte offset.
- `reader.seek(offset)` moves to an absolute byte offset and returns `reader`.
- `reader.skip(count)` advances by `count` bytes and returns `reader`.
- `reader.remaining()` returns unread byte count.
- `reader.eof?()` returns true when no bytes remain.
- `reader.read_bytes(count)` returns exactly `count` bytes and advances.
- `reader.read_u8()` reads an unsigned 8-bit integer.
- `reader.read_i8()` reads a signed 8-bit integer.
- `reader.read_u16()`, `reader.read_i16()`, `reader.read_u32()`, and
  `reader.read_i32()` use the reader default endian.
- `reader.read_u16_le()`, `reader.read_i16_le()`, `reader.read_u32_le()`, and
  `reader.read_i32_le()` read little-endian integers.
- `reader.read_u16_be()`, `reader.read_i16_be()`, `reader.read_u32_be()`, and
  `reader.read_i32_be()` read big-endian integers.
- `reader.read_f32()`, `reader.read_f64()`, `reader.read_f32_le()`,
  `reader.read_f64_le()`, `reader.read_f32_be()`, and `reader.read_f64_be()`
  read IEEE 754 floats.

## Writer

- `Writer.new()` creates an empty writer with big-endian default order.
- `Writer.new(options)` creates an empty writer with explicit options.
- `writer.position()` returns the current output byte count.
- `writer.bytes()` returns the accumulated `bytes` value.
- `writer.write_bytes(data)` appends a `bytes` value and returns `writer`.
- `writer.write_u8(value)` writes an unsigned 8-bit integer and returns
  `writer`.
- `writer.write_i8(value)` writes a signed 8-bit integer and returns `writer`.
- `writer.write_u16(value)`, `writer.write_i16(value)`,
  `writer.write_u32(value)`, and `writer.write_i32(value)` use the writer
  default endian and return `writer`.
- `writer.write_u16_le(value)`, `writer.write_i16_le(value)`,
  `writer.write_u32_le(value)`, and `writer.write_i32_le(value)` write
  little-endian integers and return `writer`.
- `writer.write_u16_be(value)`, `writer.write_i16_be(value)`,
  `writer.write_u32_be(value)`, and `writer.write_i32_be(value)` write
  big-endian integers and return `writer`.
- `writer.write_f32(value)`, `writer.write_f64(value)`,
  `writer.write_f32_le(value)`, `writer.write_f64_le(value)`,
  `writer.write_f32_be(value)`, and `writer.write_f64_be(value)` write IEEE 754
  floats and return `writer`.

## Numeric Semantics

- Integer methods require integer-valued numbers.
- Supported integer ranges:
  - `u8`: `0..255`
  - `i8`: `-128..127`
  - `u16`: `0..65535`
  - `i16`: `-32768..32767`
  - `u32`: `0..4294967295`
  - `i32`: `-2147483648..2147483647`
- Signed reads use two's-complement interpretation.
- Float methods use IEEE 754 binary32 and binary64 representation.
- `NaN`, positive infinity, and negative infinity round-trip when the runtime
  supports them.
- `u64` and `i64` are intentionally excluded until Tya has a precise integer or
  BigInt story. `Number` cannot safely represent every 64-bit integer.

## Scope

- `lib/binary/Reader.tya`
- `lib/binary/Writer.tya`
- `tests/stdlib_binary_test.tya`
- Runtime/checker/codegen builtins only where pure Tya cannot portably handle
  IEEE 754 float packing or unpacking.
- `docs/STDLIB.md`
- Next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`
- Optional example under `examples/binary/`

## Out of Scope

- `u64` and `i64` integer support.
- Bit-level readers or writers.
- Varint, LEB128, protobuf, MessagePack, CBOR, or struct-schema parsing.
- Checksums, compression, encryption, or encoding formats.
- Memory mapping.
- Direct file or socket streaming APIs. Callers should combine this package
  with existing `file`, `io`, or `net/socket` byte APIs.
- Mutable in-place editing of existing `bytes` values.

## Acceptance Criteria

- `import binary` exposes `binary.Reader` and `binary.Writer`.
- `Reader.new` and `Writer.new` return class instances with the expected
  `.class`.
- Default-endian methods use big endian unless options specify `"little"`.
- `_le` and `_be` methods ignore the instance default endian and use the
  requested byte order.
- Reader position, seek, skip, remaining, EOF, and byte-slice behavior are
  deterministic.
- Integer reads and writes cover the documented signed and unsigned ranges.
- Out-of-range integer writes raise clear `binary` errors.
- Reads past EOF raise clear `binary` errors without advancing past invalid
  input.
- `f32` and `f64` reads/writes round-trip representative finite values,
  infinities, and NaN.
- `writer.write_*` methods return `writer`, allowing chained writes.
- `writer.bytes()` returns a `bytes` value that can be passed to existing
  `bytes_array`, `file.File.write_bytes`, and `io` binary writers.
- Existing bytes, file, io, and socket tests remain green.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run 'Test.*Binary|Test.*Bytes|Test.*Io' -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

Manual smoke after implementation:

```sh
tya run examples/binary/read_header.tya
```

## Dependencies

- Builds on the existing `bytes` value type and byte conversion builtins.
- Should remain independent from planned `image`, `asset`, `raylib`, and
  native interop packages, but those packages may depend on `binary`.

## Open Questions

None.
