# Tya v0.41 Specification

This document is the in-progress specification for Tya v0.41. It is updated
as each STEP of the v0.41 GC Epic lands.

## Theme

v0.41 ships a precise mark-and-sweep garbage collector for the C runtime.
The collector is single-threaded and stop-the-world. It is the foundation
for v0.42 Tya Concurrency, which will extend the collector for multiple
worker threads.

The GC has no user-visible language semantics. Programs that ran on v0.40
run identically on v0.41 except for memory pressure and timing.

## Goals

- Bound the resident set of long-running Tya programs.
- Reclaim cyclic data without extra programmer effort.
- Keep the runtime small and dependency-free; no Boehm GC.
- Provide a small introspection API for tests, benchmarks, and
  documentation.

## Non-goals (deferred)

- Generational, incremental, or concurrent collection.
- Weak references and finalizers.
- User-tunable GC parameters.
- Multi-thread support; that arrives with v0.42 Concurrency.

## Implementation status

v0.41 is implemented in five STEPs. Each STEP keeps every existing test
green and preserves the self-host fixed point.

### STEP 1 — GC-aware allocator (landed)

Every heap allocation that holds Tya runtime values now carries a
`TyaGcHeader` as its first field. The four GC-tracked struct kinds are
`TyaArray`, `TyaDict` (also used for object-style dicts and function
member tables), `TyaFunction`, and `TyaBytes`.

A central allocator routes every tracked allocation through
`tya_gc_alloc(size, kind)`, which:

- calls `malloc` for the requested size,
- initializes the header (`mark = 0`, `kind`),
- prepends the new header to the global linked list `tya_gc_head`,
- increments `tya_gc_alloc_count` and `tya_gc_alloc_bytes`,
- returns the pointer.

Internal allocations owned by tracked structs (e.g.
`array->items`, `dict->entries`, `bytes->data`, char strings) remain
plain `malloc` calls. They are reclaimed when their owning tracked
struct is reclaimed. STEP 1 does not free anything yet, so these are
counted toward live memory until subsequent STEPs land.

**STEP 1 introduces no new language syntax.** It adds one builtin and
one stdlib module:

- Builtin `runtime_gc_stats() -> dict` returns a snapshot of the
  collector counters.
- Stdlib `runtime` module re-exports the builtin as
  `runtime.gc_stats()`.

The dict returned by `runtime.gc_stats()` has these keys:

| key           | meaning                                                  |
|---------------|----------------------------------------------------------|
| `alloc_count` | total tracked allocations made since program start       |
| `alloc_bytes` | total tracked allocation bytes since program start       |
| `freed_count` | total tracked allocations reclaimed by collections       |
| `freed_bytes` | total tracked allocation bytes reclaimed                 |
| `live_count`  | `alloc_count - freed_count`                              |
| `live_bytes`  | `alloc_bytes - freed_bytes`                              |

In STEP 1, no collection runs, so `freed_count` and `freed_bytes` are
always `0`, and `live_*` equals `alloc_*`.

### STEP 2 — Mark phase (planned)

Scan roots (value stack, active locals, module globals,
currently-active closure environments, in-flight error reraise slots)
and transitively mark every reachable tracked allocation.

### STEP 3 — Sweep phase (planned)

Walk the linked list, free unmarked allocations and any inner
allocations they own, reset mark bits.

### STEP 4 — Trigger policy and `runtime.gc()` API (planned)

Allocation-threshold trigger; `runtime.gc()` for explicit invocation in
tests and benchmarks.

### STEP 5 — Documentation and examples (planned)

Long-running examples demonstrating bounded resident set; cycle
reclamation tests; finalization of this spec.

## Observable language behavior

None at STEP 1 beyond the new builtin. The next STEPs will not change
language semantics either; they will only reclaim memory that is no
longer reachable.
