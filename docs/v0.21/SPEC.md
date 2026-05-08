# Tya v0.21 Specification

This document is the specification for Tya v0.21 after v0.20 standard attached
`math` and `path` modules.

## Theme

Tya v0.21 adds native-backed standard library modules for file and process
access.

v0.21 introduces the smallest native-backed stdlib surface: `file` and `os`.
These modules expose external system operations through explicit imports and
module functions.

## Goals

- Add the native-backed stdlib mechanism needed by `file` and `os`.
- Add a `file` standard module.
- Add an `os` standard module.
- Keep native-backed APIs import-only and explicit.
- Keep native-backed failures integrated with structured errors.
- Avoid adding new global built-ins.
- Avoid broad filesystem, time, network, and streaming APIs.

## Included in v0.21

v0.21 includes all v0.20 behavior and adds:

- native-backed standard library module support
- `file.read(path)`
- `file.write(path, text)`
- `file.exists?(path)`
- `os.args()`
- `os.env(name)`
- `os.exit(code)`
- structured-error `raise` behavior for native operation failures
- source-oriented diagnostics for invalid native-backed stdlib calls

## Not Included in v0.21

v0.21 does not include:

- new language syntax
- new global built-ins
- removal of existing global IO/process built-ins
- directory listing
- directory creation or removal
- file removal or rename
- stat metadata
- path expansion
- current-directory APIs
- time/date APIs
- HTTP APIs
- JSON or CSV parsers
- permissions APIs
- streaming IO
- binary IO
- async IO

## Native-backed Stdlib Rules

Native-backed stdlib APIs are exposed only as module functions.

```tya
import file
import os
```

They are not available without import.

Native-backed operation failures raise structured errors. They do not call
`panic`.

```tya
import file

try
  text = file.read("missing.txt")
  println text
catch err
  println err
```

`try` and `catch` can handle native-backed operation failures because they are
raised errors.

## `file`

The `file` module contains basic whole-file text helpers.

Functions:

- `file.read(path)`
- `file.write(path, text)`
- `file.exists?(path)`

Examples:

```tya
import file

if file.exists?("memo.txt")
  text = file.read("memo.txt")
  println text

file.write("out.txt", "hello")
```

`file.read(path)` reads the entire file and returns it as a string.

`file.write(path, text)` writes `text` to `path` and returns `nil` on success.

`file.exists?(path)` returns `true` when the path exists and `false` when it
does not exist. If existence cannot be determined, it raises an error.

All `file` functions require string path arguments. `file.write(path, text)`
also requires string text.

`file` is text-oriented in v0.21. Binary IO is not part of this version.

## `os`

The `os` module contains basic process-environment helpers.

Functions:

- `os.args()`
- `os.env(name)`
- `os.exit(code)`

Examples:

```tya
import os

args = os.args()
home = os.env("HOME")

if len(args) == 0
  os.exit(1)
```

`os.args()` returns the command-line arguments as an array of strings.

`os.env(name)` returns the environment variable value as a string. It returns
`nil` when the variable is not present.

`os.exit(code)` exits the process with `code`.

`os.env(name)` requires a string name.

`os.exit(code)` requires an integer code.

## Existing Global Built-ins

v0.21 adds `file.*` and `os.*` as the preferred APIs for file and process
operations.

Existing global IO/process built-ins remain available for compatibility in
v0.21. Their deprecation or removal is a later-version decision.

v0.21 does not add any new global built-ins.

## Diagnostics

v0.21 implementations should report source-oriented errors for:

- missing imports for `file` or `os`
- unknown `file` or `os` module functions
- wrong argument counts
- wrong argument kinds
- failed `file.read(path)`
- failed `file.write(path, text)`
- indeterminate `file.exists?(path)`
- invalid `os.exit(code)` values

Diagnostics should mention the module name, function name, expected argument
shape, and actual value kind when available.
