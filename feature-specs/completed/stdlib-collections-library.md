---
status: completed
goal_ready: false
---

# Feature: Collections Stdlib Library

## Goal

Add a standard `collections` library with reusable class-style container types
for common data-structure needs that are awkward or error-prone to rebuild from
plain arrays and dictionaries in every application.

## Context

Tya already has built-in arrays and dictionaries plus primitive methods such as
`map`, `filter`, `reduce`, `keys`, and `has`. Those are good general-purpose
data containers, but they do not directly provide named container behavior such
as FIFO queues, double-ended queues, sets, or priority queues.

The stdlib class-style direction says stdlib-owned domain values should be class
instances unless there is a strong reason to keep dictionaries. Collection
objects are named stdlib concepts with invariants and behavior, so they should
be instances.

## Behavior

- Add a public `collections` stdlib package.
- Import shape:

  ```tya
  import collections as collections

  queue = collections.Queue.new()
  queue.push("job")
  next = queue.pop()

  seen = collections.Set.new()
  seen.add("asset.png")
  if seen.has?("asset.png")
    println "loaded"
  ```

- Public classes:
  - `collections.Stack`
  - `collections.Queue`
  - `collections.Deque`
  - `collections.Set`
  - `collections.PriorityQueue`
- All collection constructors return class instances.
- Collection instances are mutable containers by design.
- Empty removal/peek operations raise clear `collections` errors instead of
  returning `nil`, because `nil` is a valid stored value.
- `len()` returns the number of stored values for each collection.
- `empty?()` returns true when `len() == 0`.
- `clear()` removes all values and returns the collection instance.
- `to_array()` returns a snapshot array in the documented iteration order.

## Stack

- `Stack.new()` creates an empty LIFO stack.
- `Stack.from_array(values)` creates a stack whose next `pop()` returns the
  last item from `values`.
- `stack.push(value)` appends a value and returns `stack`.
- `stack.pop()` removes and returns the most recently pushed value.
- `stack.peek()` returns the most recently pushed value without removing it.
- `stack.len()`, `stack.empty?()`, `stack.clear()`, and `stack.to_array()` work.

## Queue

- `Queue.new()` creates an empty FIFO queue.
- `Queue.from_array(values)` creates a queue whose next `pop()` returns the
  first item from `values`.
- `queue.push(value)` appends a value to the back and returns `queue`.
- `queue.pop()` removes and returns the front value.
- `queue.peek()` returns the front value without removing it.
- `queue.len()`, `queue.empty?()`, `queue.clear()`, and `queue.to_array()` work.
- Queue implementation should avoid repeated whole-array shifting for normal
  push/pop usage.

## Deque

- `Deque.new()` creates an empty double-ended queue.
- `Deque.from_array(values)` creates a deque in front-to-back order.
- `deque.push_front(value)` inserts at the front and returns `deque`.
- `deque.push_back(value)` inserts at the back and returns `deque`.
- `deque.pop_front()` removes and returns the front value.
- `deque.pop_back()` removes and returns the back value.
- `deque.peek_front()` returns the front value without removing it.
- `deque.peek_back()` returns the back value without removing it.
- `deque.len()`, `deque.empty?()`, `deque.clear()`, and `deque.to_array()` work.

## Set

- `Set.new()` creates an empty insertion-ordered set.
- `Set.from_array(values)` creates a set by adding values in array order.
- `set.add(value)` adds a value if absent and returns `set`.
- `set.delete(value)` removes a value if present and returns true when a value
  was removed.
- `set.has?(value)` returns true when an equal value is present.
- `set.len()`, `set.empty?()`, `set.clear()`, and `set.to_array()` work.
- Set equality uses Tya value equality, so numbers, strings, booleans, `nil`,
  arrays, dictionaries, and class instances can be stored.
- `to_array()` returns values in first-insertion order.
- `Set.union(a, b)`, `Set.intersection(a, b)`, `Set.difference(a, b)`, and
  `Set.subset?(a, b)` return or compare `Set` instances.
- The first implementation may use linear equality checks to support all Tya
  values. A faster hash-based set can be added later if Tya gains a stable
  public hash primitive.

## PriorityQueue

- `PriorityQueue.new()` creates an empty stable min-priority queue.
- `PriorityQueue.from_array(items)` accepts an array of `[priority, value]`
  pairs.
- `pq.push(value, priority)` inserts a value with numeric priority and returns
  `pq`.
- `pq.pop()` removes and returns the value with the smallest priority.
- `pq.peek()` returns the next value without removing it.
- `pq.peek_priority()` returns the next priority without removing it.
- Ties with equal priority are popped in insertion order.
- Priorities must be numbers. Non-number priorities raise clear
  `collections.priority_queue` errors.
- `pq.len()`, `pq.empty?()`, `pq.clear()`, and `pq.to_array()` work.
- `to_array()` returns `[priority, value]` pairs in pop order without mutating
  the queue.

## Scope

- `lib/collections/Stack.tya`
- `lib/collections/Queue.tya`
- `lib/collections/Deque.tya`
- `lib/collections/Set.tya`
- `lib/collections/PriorityQueue.tya`
- `tests/stdlib_collections_test.tya`
- `docs/STDLIB.md`
- Next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`
- Optional examples under `examples/collections/`

## Out of Scope

- Replacing built-in Array or Dict behavior.
- Reintroducing removed `array` or `dict` module facades.
- Immutable or persistent collections.
- Sorted maps, ordered dictionaries, multimaps, bidirectional maps, tries, or
  bloom filters.
- A hash primitive or hash-table-backed public `HashSet`.
- Thread-safe collection variants.
- Generic type parameters or compile-time element typing.
- Native code.

## Acceptance Criteria

- `import collections as collections` exposes `Stack`, `Queue`, `Deque`, `Set`,
  and `PriorityQueue`.
- All constructors return class instances with the expected `.class`.
- `Stack` behaves as LIFO and preserves expected `to_array()` order.
- `Queue` behaves as FIFO and does not rely on repeated whole-array shifting for
  normal push/pop usage.
- `Deque` supports correct front/back push, pop, and peek behavior.
- Empty pop and peek operations raise clear collection-specific errors.
- `Set` stores unique values using Tya equality and preserves insertion order.
- `Set.union`, `Set.intersection`, `Set.difference`, and `Set.subset?` work.
- `PriorityQueue` pops the smallest numeric priority first and preserves
  insertion order for equal priorities.
- `PriorityQueue.to_array()` returns pop order without mutating the queue.
- `len`, `empty?`, `clear`, and `to_array` are covered for each collection.
- Existing Array and Dict primitive behavior remains unchanged.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run 'Test.*Collections|Test.*Array|TestSelfhostV01Scripts' -count=1
go test ./... -count=1
```

Manual smoke after implementation:

```sh
tya run examples/collections/path_queue.tya
```

## Dependencies

- Uses existing Array, Dict, and value equality behavior.
- Should align with the stdlib class-style PRD.
- Does not depend on planned `geometry`, `transform2d`, `image`, `raylib`, or
  web libraries, but those packages may use these collections.

## Open Questions

None.
