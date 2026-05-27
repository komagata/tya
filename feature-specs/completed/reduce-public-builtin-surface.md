# Feature: Reduce Public Builtin Surface

## Goal

Reduce Tya's user-facing built-in function surface to a small core. File,
directory, path, process, I/O, stream, bytes, random, compression, digest,
socket, compiler, and collection helper operations should be exposed through
class-style standard-library APIs. The low-level runtime functions may remain
as internal intrinsics used by the standard library, selfhost, or runtime
support code.

## Context

`docs/SPEC.md` currently lists many standalone built-ins. Several already have
class-style wrappers in `lib/`, for example:

- `file/File` wraps file read/write/stat/remove/rename/bytes operations.
- `dir/Dir` wraps directory listing and creation/removal.
- `path/Path` wraps path helpers including user expansion.
- `io/Io`, `io/Reader`, and `io/Writer` wrap stream handles.
- `time/Time`, `random/Random`, `secure_random/SecureRandom`,
  `compress/Compress`, `digest/Digest`, `net/socket/Socket`, and
  `compiler/*` wrap low-level runtime support functions.

The desired language surface is that ordinary users import classes and call
class or instance methods. Standalone built-ins should be rare and reserved for
core language/runtime necessities.

## Behavior

- Keep the public built-in list intentionally small. The intended remaining
  public built-ins are:
  - `print(value)`
  - `println(value)`
  - `error(message)`
  - `exit(status)`
  - `args()`
  - `env(name)`
- Remove file, directory, and path helpers from the public built-in surface.
  Public replacements:
  - `File.read(path)`
  - `File.write(path, text)`
  - `File.append(path, text)`
  - `File.exists?(path)`
  - `File.stat(path)`
  - `File.remove(path)`
  - `File.rename(old_path, new_path)`
  - `File.read_bytes(path)`
  - `File.write_bytes(path, bytes)`
  - `Dir.list(path)`
  - `Dir.mkdir(path)`
  - `Dir.rmdir(path)`
  - `Path.expand_user(path)`
- Remove process working-directory helpers from the public built-in surface.
  Public replacements:
  - `Process.cwd()`
  - `Process.chdir(path)`
- Remove input and stream helpers from the public built-in surface. Public
  replacements:
  - `Io.stdin()`
  - `Io.stdout()`
  - `Io.stderr()`
  - `Io.open(path, mode)`
  - `Reader#read(size)`
  - `Reader#read_line()`
  - `Reader#eof?()`
  - `Reader#close()`
  - `Writer#write(value)`
  - `Writer#write_line(value)`
  - `Writer#flush()`
  - `Writer#close()`
- Remove text, bytes, and conversion helpers from the public built-in surface
  where an instance or class API should exist. Public replacements should be
  class-style or ordinary receiver APIs:
  - `chr(number)` and `ord(string)` move to a class API such as
    `String.chr(number)` and `String.ord(string)`, or another repository
    approved class-style spelling.
  - `type(value)` and `kind(value)` move to `value.class`,
    `value.class.name`, or a class-style `Value` API.
  - `to_string(value)` moves to `value.to_s()`.
  - `to_int(value)` moves to `value.to_i()`.
  - `to_float(value)` moves to `value.to_f()`.
  - `to_number(value)` is removed; use an approved explicit conversion API if
    one remains necessary.
  - `bytes(value)`, `bytes_of(string)`, `bytes_text(bytes)`,
    `bytes_array(bytes)`, `bytes_concat(left, right)`, and
    `bytes_slice(bytes, start, end)` move to a `Bytes` or `Binary` class API,
    while valid bytes receiver operations may keep primitive fast paths.
- Remove collection helper built-ins from the public surface. Public
  replacements:
  - `equal(left, right)` should use `==` or `left.equal?(right)` where
    appropriate.
  - `delete(dict, key)` moves to `dict.delete(key)`.
  - `has(dict, key)` moves to `dict.has(key)` or `dict.has?(key)`.
  - `keys(dict)` moves to `dict.keys()`.
  - `values(dict)` moves to `dict.values()`.
  - `pop(array)` moves to `array.pop()`.
- Remove low-level standard-library support helpers from the public built-in
  surface. Public replacements:
  - `Time.now()`, `Time.sleep(seconds)`, `Time.format(...)`,
    `Time.parse(...)`, `Time.since(...)`
  - `Random.seed(seed)`, `Random.int(min, max)`, `Random.float()`
  - `Compress.gzip(value)`, `Compress.gunzip(value)`,
    `Compress.zlib(value)`, `Compress.unzlib(value)`
  - `Digest.md5(value)`, `Digest.sha1(value)`, `Digest.sha256(value)`,
    `Digest.sha384(value)`, `Digest.sha512(value)`
  - `SecureRandom.bytes(size)`, `SecureRandom.int(min, max)`
  - `Socket.connect(host, port, options)`, `socket.read(size)`,
    `socket.read_line()`, `socket.write(value)`, `socket.close()`
  - `Server.listen(host, port, options)`, `server.accept()`,
    `server.close()`, `server.local_address()`
  - `socket.closed?()`, `socket.local_address()`,
    `socket.remote_address()`
  - `Lexer.lex(source)`, `Lexer.lex_with_comments(source)`,
    `Parser.parse(source)`, `Parser.parse_tokens(tokens)`,
    `Ast.children(node)`, `Ast.kind(node)`, `Ast.span(node)`,
    `Checker.check(source)`, `Checker.check_ast(program)`,
    `Format.format(source)`, `Format.unparse(program)`
- Any checker/codegen/evaluator built-in name that is only a backing helper
  for these classes and follows the same low-level naming pattern is treated
  as internal unless it is explicitly listed in the retained public built-ins.
- The low-level functions may remain in the evaluator, C codegen, and runtime
  as internal intrinsics if they are needed to implement standard-library
  classes, selfhost, or runtime-backed APIs.
- Checker and lint diagnostics should guide direct public use of removed
  built-ins to the class-style replacement.
- `docs/SPEC.md`, `docs/ja/spec.md`, and `docs/GUIDE.md` should present the
  class-style APIs, not the internal intrinsic names.

## Scope

- `docs/SPEC.md`, `docs/ja/spec.md`, and `docs/GUIDE.md`.
- Standard-library wrappers under `lib/`, including adding missing methods
  such as `File.append`, `Process.cwd`, and `Process.chdir`.
- Checker diagnostics for removed public built-ins.
- Lint/autofix rules where straightforward replacements are mechanical.
- Evaluator/codegen/runtime changes only where needed to hide, reject, or
  reclassify public use while keeping stdlib internals working.
- Examples and active tests that currently call the removed built-ins directly.
- Selfhost sources only where they should no longer use public-facing removed
  helpers directly.

## Out of Scope

- Removing runtime intrinsics that are still needed by stdlib wrappers.
- Removing ordinary instance methods on primitive values.
- Redesigning the package/import system.
- Removing archived historical fixtures or docs.
- Changing the semantics of the retained operations.
- Forcing class-style wrappers through slow generic dispatch when runtime
  intrinsics are simpler or faster.

## Acceptance Criteria

- SPEC no longer lists file, directory, path, process cwd/chdir, stream, bytes,
  collection helper, random, compression, digest, socket, or compiler helper
  functions as public standalone built-ins.
- The class-style replacement APIs exist for every removed public operation
  that remains supported.
- `File.append(path, text)`, `Process.cwd()`, and `Process.chdir(path)` exist.
- Direct public use of removed built-ins is rejected or diagnosed with a clear
  replacement.
- Standard-library code can still use internal intrinsics or an equivalent
  private implementation path.
- Examples, active fixtures, and selfhost sources no longer rely on removed
  public built-in spellings except where explicitly treated as internal.
- Native and WASM documentation clearly distinguishes supported public APIs
  from target-specific runtime limitations.
- Existing selfhost fixed-point invariants remain green.

## Verification

```sh
go test ./internal/checker -count=1
go test ./internal/eval -count=1
go test ./internal/codegen -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestSelfhostV02Scripts -count=1
go test ./tests -run 'TestV01Scripts|TestV18Scripts|TestV40Scripts|TestV57Scripts|TestV58Scripts|TestV63Scripts' -count=1
bundle exec jekyll build --source docs --destination _site
go test ./... -count=1 -timeout=20m
```
