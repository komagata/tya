---
status: completed
goal_ready: false
---

# Feature: Random Stdlib Extensions

## Goal

Extend the standard `random` library with independent seedable RNG instances,
non-mutating helpers, sampling, weighted choice, and small probability helpers
for tests, procedural generation, simulations, and simple games.

## Context

The current `random.Random` class provides a process-global seed, integer,
float, choice, and in-place shuffle API. That is enough for simple scripts but
awkward when a program needs independent deterministic streams, repeatable
procedural generation, or helpers such as sampling without replacement and
weighted choice.

These additions should keep `random` deterministic and non-cryptographic.
Security-sensitive code should continue to use `secure_random`.

## Behavior

- Keep the existing `random.Random` class and public methods.
- Add a public `random.Rng` class for independent pseudo-random generators.
- Import shape:

  ```tya
  import random

  rng = random.Rng.new(42)
  item = rng.weighted_choice(["common", "rare"], [90, 10])
  hand = rng.sample(deck, 5)
  ```

- `Random` static helpers continue to use the process-global RNG.
- `Rng` instances maintain independent state and return reproducible sequences
  for the same seed.
- Random APIs are not cryptographically secure.
- Invalid ranges, empty inputs, invalid sample sizes, and invalid weights raise
  clear `random` errors.

## Static Random API

- Preserve existing methods:
  - `Random.seed(value)`
  - `Random.int(min, max)`
  - `Random.float()`
  - `Random.choice(items)`
  - `Random.shuffle(items)`
- Add static methods:
  - `Random.bool()`
  - `Random.bool(probability)`
  - `Random.shuffle_copy(items)`
  - `Random.sample(items, count)`
  - `Random.weighted_choice(items, weights)`
  - `Random.weighted_index(weights)`
- `Random.shuffle(items)` remains in-place for compatibility and returns `nil`.
- `Random.shuffle_copy(items)` returns a shuffled copy and does not mutate
  `items`.

## Rng Instance API

- `Rng.new(seed)` creates an independent RNG.
- `rng.seed(value)` resets the instance seed and returns `rng`.
- `rng.int(min, max)` returns an integer in the inclusive range.
- `rng.float()` returns a number in `0.0 <= n < 1.0`.
- `rng.bool()` returns true or false with 50% probability.
- `rng.bool(probability)` returns true with probability `0.0..1.0`.
- `rng.choice(items)` returns one item from a non-empty array.
- `rng.shuffle(items)` shuffles an array in place and returns `nil`.
- `rng.shuffle_copy(items)` returns a shuffled copy.
- `rng.sample(items, count)` returns `count` unique items sampled without
  replacement.
- `rng.weighted_choice(items, weights)` returns one item using numeric weights.
- `rng.weighted_index(weights)` returns the selected index using numeric
  weights.

## Semantics

- `int(min, max)` is inclusive at both ends.
- `sample(items, 0)` returns an empty array.
- `sample(items, items.len())` returns a shuffled copy of all items.
- `sample(items, count)` rejects negative counts and counts larger than
  `items.len()`.
- Weighted helpers require:
  - `items.len() == weights.len()` for `weighted_choice`,
  - every weight is a number,
  - every weight is finite and non-negative,
  - at least one weight is greater than zero.
- Zero-weight items are never selected unless all positive-weight behavior would
  be impossible, which is rejected.
- `bool(probability)` requires `0.0 <= probability <= 1.0`.

## Scope

- `lib/random/Random.tya`
- `lib/random/Rng.tya`
- Runtime/checker/codegen builtins only if independent RNG state cannot be
  implemented cleanly in Tya.
- `tests/stdlib_random_test.tya`
- `docs/STDLIB.md`
- Next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`
- Optional examples under `examples/random/`

## Out of Scope

- Cryptographic randomness.
- Replacing `secure_random`.
- Guaranteeing a specific PRNG algorithm forever as a language compatibility
  promise.
- Probability distributions beyond uniform and weighted discrete choice.
- Thread-safe RNG instances.
- Global random state isolation per task.

## Acceptance Criteria

- Existing `Random` tests still pass.
- `random.Rng.new(seed)` returns an `Rng` instance.
- Two `Rng` instances with the same seed produce the same sequence.
- Different `Rng` instances do not interfere with each other's state.
- `Random.shuffle_copy` and `rng.shuffle_copy` do not mutate input arrays.
- `sample` handles zero, full-size, and invalid counts correctly.
- `weighted_choice` and `weighted_index` respect zero weights and reject invalid
  weights.
- `bool(probability)` handles `0.0`, `1.0`, and invalid probabilities.
- Existing `secure_random` tests remain green.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run 'Test.*Random|Test.*SecureRandom|TestSelfhostV01Scripts' -count=1
go test ./... -count=1
```

Manual smoke after implementation:

```sh
tya run examples/random/weighted_loot.tya
```

## Dependencies

- Builds on existing non-cryptographic random runtime helpers.
- Should not change `secure_random` behavior.

## Open Questions

None.
