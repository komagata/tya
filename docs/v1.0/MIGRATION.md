# Tya v1.0 Migration

Tya v1.0 freezes the canonical class-style stdlib surface and the strict
runtime semantics documented in `SPEC.md`.

Use canonical package APIs for new code:

- `regex/Regex` for pattern matching.
- `file/File` and `dir/Dir` for filesystem utilities.
- `time/Time` for wall-clock times, monotonic timestamps, and durations.
- `os/Os` and `process/Process` for environment and process work.
- `hmac/Hmac` for keyed message authentication.

Legacy helper aliases that remain for `selfhost/v01` or bootstrap recovery are
legacy compatibility only. They are not v1.x compatibility guarantees and
public code should prefer the documented class-style APIs.

Generated-C runtime behavior is part of the public v1 execution contract.
Programs should not rely on pre-v1 compatibility fallbacks such as nil returns
from invalid calls or invalid indexing. Invalid public v1 programs are
diagnosed with stable `TYA-E....` codes.
