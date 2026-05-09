# Tya v0.41 Specification

## Theme

v0.41 ships a precise mark-and-sweep garbage collector for the C runtime.
The collector is single-threaded and stop-the-world. It is the foundation
for v0.42 Tya Concurrency, which will extend the collector for multiple
worker threads.

The GC has no user-visible language semantics. Programs that ran on v0.40
run identically on v0.41 except for memory pressure and timing.

## Goals

- Bound the resident set of long-running Tya programs at points where the
  collector is allowed to run.
- Reclaim cyclic data without extra programmer effort.
- Keep the runtime small and dependency-free; no Boehm GC.
- Provide a small introspection API for tests, benchmarks, and
  documentation.

## Non-goals (deferred to later minor versions)

- Generational, incremental, or concurrent collection.
- Weak references and finalizers.
- User-tunable GC parameters.
- Multi-thread support; that arrives with v0.42 Concurrency.
- Precise tracking of locals inside user functions; collections inside
  function bodies are not safe in v0.41 (see Safety contract below).

## Observable language behavior

None beyond two new APIs in the `runtime` stdlib module:

```tya
import runtime

stats = runtime.gc_stats()
runtime.gc()
```

`runtime.gc_stats()` returns a dict snapshot of the GC counters with
keys:

| key             | meaning                                                  |
|-----------------|----------------------------------------------------------|
| `alloc_count`   | total tracked allocations made since program start       |
| `alloc_bytes`   | total tracked allocation bytes since program start       |
| `freed_count`   | total tracked allocations reclaimed by collections       |
| `freed_bytes`   | total tracked allocation bytes reclaimed                 |
| `live_count`    | `alloc_count - freed_count`                              |
| `live_bytes`    | `alloc_bytes - freed_bytes`                              |
| `collect_count` | number of collections performed                          |
| `threshold`     | live_count threshold that triggers an auto-collection    |

`runtime.gc()` runs a full mark-and-sweep collection. See the safety
contract below.

## Implementation

### Tracked allocations

Every heap allocation that holds Tya runtime values now carries a
`TyaGcHeader` as its first field. The four GC-tracked struct kinds are:

- `TyaArray`
- `TyaDict` (also used for object-style dicts and function member tables)
- `TyaFunction`
- `TyaBytes`

A central allocator routes every tracked allocation through
`tya_gc_alloc(size, kind)`, which:

- calls `malloc` for the requested size,
- initializes the header (`mark = 0`, `kind`, `size`),
- prepends the new header to the global linked list `tya_gc_head`,
- increments `tya_gc_alloc_count` and `tya_gc_alloc_bytes`,
- returns the pointer.

Internal allocations owned by tracked structs (e.g. `array->items`,
`dict->entries`, `bytes->data`, char strings) remain plain `malloc`
calls. They are reclaimed when their owning tracked struct is reclaimed.

### Roots

Generated code calls `tya_gc_register_root(&g_<name>)` for every
module-level `TyaValue` global at `main()` startup, so the collector can
trace them as roots. The active raise-frame chain is also walked as
roots so that an in-flight `raise` value survives a collection.

Locals inside user functions are **not** roots in v0.41.

### Mark phase

Mark traversal recurses through:

- `TyaArray.items[i]` for each `i` in `0..len`.
- `TyaDict.entries[i].value` for each non-null entry key.
- `TyaFunction.receiver`, `TyaFunction.parent`, and
  `TyaFunction.members`.
- `TyaBytes` is a leaf.

Headers reach a fixed point: a header is marked at most once per
collection, and the traversal short-circuits on already-marked headers,
so cycles are handled without extra effort.

### Sweep phase

Sweep walks `tya_gc_head` once. For each header:

- If `mark == 0`, the header is unlinked from the list, and
  `tya_gc_free_one` runs the kind-specific free routine
  (which also frees the inner buffer). `tya_gc_freed_count` and
  `tya_gc_freed_bytes` advance by the freed object's size.
- If `mark == 1`, the mark bit is reset to `0` for the next collection.

### Trigger policy

Two trigger paths exist:

1. **Explicit**: a Tya program calls `runtime.gc()`, which calls
   `tya_gc_collect`. This is the primary way to run the collector in
   v0.41.

2. **Automatic safe-point**: the code generator emits
   `tya_gc_maybe_collect();` between top-level statements in `main()`.
   That helper checks the live-count threshold and, when the threshold
   is exceeded, runs a collection. The threshold is `2 * live_count`
   after the previous collection, with a minimum of `1024`. This
   trigger covers programs whose work happens at the top level — a
   sequence of top-level stmts that allocate and discard data — but
   does **not** fire inside a `while` or `for` body.

## Safety contract

`tya_gc_collect` reclaims any tracked allocation that is not reachable
from a registered root (module global) or the active raise chain.
Locals inside user functions, including the implicit `__iter` value of
a `for x in y` loop, are not roots. Collecting while such a local
holds the only reference to a heap value would free that value out
from under the local, leading to use-after-free.

In v0.41 the collector is therefore safe to run only at points where
every live local TyaValue is also reachable from a registered root.
The two safe places are:

1. The top level of the program, between top-level statements
   (`main()`-level boundaries). The generator's automatic trigger fires
   only here.
2. Any point in user code where the program has assigned every
   currently-needed TyaValue into a top-level binding before calling
   `runtime.gc()`. This is the safe contract for explicit
   `runtime.gc()` calls inside loops or functions: the program must
   have spilled live values to top-level bindings first.

Future minor versions will extend the runtime with precise local
tracking (a shadow stack written by the code generator) or
conservative stack scanning, at which point the collector will become
safe to call from any context. v0.41 leaves that work for v0.42+.

## Examples

```tya
# Reclamation of an unreachable subgraph.
import runtime

root = []
i = 0
while i < 50
  push(root, [i, i + 1, i + 2])
  i = i + 1

peak = runtime.gc_stats()["live_count"]
root = []
runtime.gc()
after = runtime.gc_stats()["live_count"]
print "peak: " + to_string(peak)
print "after: " + to_string(after)
```

```tya
# Reclamation of a cycle.
import runtime

root = []
i = 0
while i < 30
  c = {}
  c["self"] = c
  push(root, c)
  i = i + 1

root = []
runtime.gc()
print runtime.gc_stats()["live_count"]
```

See `examples/long_running_loop.tya` for a long-running loop that calls
`runtime.gc()` inside its body to keep the resident set bounded.
