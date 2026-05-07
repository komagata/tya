# Standard Library

The standard library documentation now lives in `docs/API.md`.

Keep this file as a compatibility pointer for existing project notes that still
refer to `docs/STDLIB.md`.

## Planned Shape

After self-hosting work settles, the standard library should move toward a
module-first shape. Standard library entry points should be modules, simple
operations should be module functions, and larger stateful APIs may expose
classes under their module.

```tya
import string
import file

parts = string.split("a,b,c", ",")
text = file.read("memo.txt")
```

For larger APIs:

```tya
import http

client = http.Client.new({ timeout: 30 })
response = client.get(url)
```

Keep global builtins small. Prefer adding new standard behavior under a module
unless the function is fundamental enough to feel like part of the language
surface.

## Initial Categories And Modules

```text
Core
  prelude
  convert
  error

Text
  string

Collections
  array
  dict
  set

IO
  io
  file
  path

System
  process
  env

Testing
  test
  assert

Math
  math
```

## Deferred Candidates

The following modules are useful, but should wait until the smaller standard
library surface is stable:

```text
Data
  json
  csv

Time
  time
  duration

Random
  random

Text
  regex
  unicode

Network
  http
```
