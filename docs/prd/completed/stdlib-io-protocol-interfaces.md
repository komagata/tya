# Feature: Stdlib IO Protocol Interfaces

## Goal

Define standard I/O protocol interfaces so files, sockets, process streams,
buffers, and future stream-like values can advertise readable, writable,
flushable, and closeable behavior with explicit `implements` contracts.

## Context

The stdlib already has `io.Reader`, `io.Writer`, `net/socket.Socket`,
`net/socket.Server`, channel close methods, and low-level stream builtins.
These values expose methods such as `read`, `write`, `flush`, and `close`, but
there is no shared stdlib interface that documents or checks the protocol.

Existing interface syntax supports requirement-only methods and default
methods. Existing class names `io.Reader` and `io.Writer` must not be renamed or
replaced by this feature.

## Behavior

- Add standard interfaces:

  ```tya
  interface Readable
    read = size ->

  interface Writable
    write = data ->

  interface Closable
    close = ->

  interface Flushable
    flush = ->
  ```

- `Readable.read(size)` reads at most `size` units from the receiver.
- For text streams, `read(size)` returns a string.
- For binary streams, `read(size)` returns bytes.
- Chunk reads return an empty value at EOF, matching current `io.Reader`
  behavior.
- `Writable.write(data)` writes strings, bytes, or values accepted by the
  receiver. Text-oriented writers may stringify non-string values according to
  existing writer behavior.
- `write(data)` returns the number of bytes written when the receiver can
  report it. Existing `io.Writer.write` behavior remains authoritative for
  stdlib file/process streams.
- `Flushable.flush()` flushes buffered writes and returns `nil`.
- `Closable.close()` releases or detaches the resource and returns `nil`.
- Closing borrowed process streams such as stdin/stdout/stderr remains a no-op,
  matching current `io` behavior.
- Implementations may implement any subset of the four interfaces. For example,
  read-only streams implement `Readable` and `Closable`, while buffered writers
  implement `Writable`, `Flushable`, and `Closable`.
- Existing classes should declare `implements` where the methods already match:
  - `io.Reader implements Readable, Closable`
  - `io.Writer implements Writable, Flushable, Closable`
  - `net/socket.Socket implements Readable, Writable, Closable`
  - `net/socket.Server implements Closable`
- Interface names are intentionally capability adjectives (`Readable`,
  `Writable`, `Closable`, `Flushable`) to avoid conflicts with existing
  concrete classes named `Reader` and `Writer`.

## Scope

- Add stdlib interface files for the four protocols.
- Update existing stdlib I/O and socket classes to declare matching
  `implements` clauses.
- Update package docs in `docs/STDLIB.md` and API docs where protocol names are
  useful.
- Add checker/testscript coverage for successful and failed implementations.
- Add black-box tests showing `Io.copy`-style functions can accept values by
  protocol shape once the stdlib exposes protocol-based APIs.

## Out of Scope

- Renaming existing `io.Reader` or `io.Writer` classes.
- Adding generic type parameters for bytes/text stream kinds.
- Changing EOF behavior for `io.Reader.read`.
- Adding async I/O or nonblocking stream protocols.
- Changing channel close semantics.
- Adding `read_line`, `write_line`, or `each_line` requirements to the base
  interfaces. Those are convenience APIs on concrete classes, not the minimal
  protocol.

## Acceptance Criteria

- `Readable`, `Writable`, `Closable`, and `Flushable` are available as stdlib
  interfaces.
- A class missing `read(size)` cannot implement `Readable`.
- A class missing `write(data)` cannot implement `Writable`.
- A class missing `close()` cannot implement `Closable`.
- A class missing `flush()` cannot implement `Flushable`.
- Existing `io.Reader`, `io.Writer`, `net/socket.Socket`, and
  `net/socket.Server` continue to pass their current tests after declaring
  matching interfaces.
- Existing `io` tests for `read`, `write`, `flush`, `close`, and borrowed
  process streams still pass.
- Documentation distinguishes protocol interfaces from concrete reader/writer
  classes.

## Verification

```sh
go test ./... -count=1
go test ./tests -run 'TestIOScripts|TestNetSocketScripts|TestV11Scripts|TestV12Scripts' -count=1
```
