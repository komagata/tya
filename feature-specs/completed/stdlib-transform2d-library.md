---
status: completed
goal_ready: false
---

# Feature: Transform2D Stdlib Library

## Goal

Add a pure Tya `transform2d` standard library package for 2D affine transforms,
so games, image tooling, layout code, charts, and graphics bindings can share a
small class-style API for translation, rotation, scaling, composition, and
coordinate conversion.

## Context

The planned `geometry` stdlib covers points, vectors, sizes, rectangles, and
circles, but intentionally leaves matrix transforms and affine transform
composition out of scope. Tya already has a general `matrix.Matrix` library,
but a dedicated 2D transform type is more ergonomic for common graphics and
layout work and avoids making callers build 3x3 matrices by hand.

The public value should be a class instance, not a dictionary, matching the
class-style direction for stdlib-owned domain values.

## Behavior

- Add a public `transform2d` stdlib package.
- Import shape:

  ```tya
  import geometry as geo
  import transform2d as transform2d

  t = transform2d.Transform2D.translation(10, 20)
  r = transform2d.Transform2D.rotation(Math.pi() / 2)
  world = transform2d.Transform2D.compose(t, r)

  p = geo.Point.new(1, 0)
  moved = transform2d.Transform2D.apply_point(world, p)
  ```

- Public class:
  - `transform2d.Transform2D`
- `Transform2D` instances expose numeric public fields:
  - `a`
  - `b`
  - `c`
  - `d`
  - `tx`
  - `ty`
- Fields represent the affine matrix:

  ```text
  [ a  c  tx ]
  [ b  d  ty ]
  [ 0  0   1 ]
  ```

- Applying a transform to a point uses:
  - `x' = a * x + c * y + tx`
  - `y' = b * x + d * y + ty`
- Transform values are immutable by API convention. Operations return new
  `Transform2D` instances and do not mutate inputs.
- Constructors validate numeric fields and raise clear `transform2d` errors for
  invalid input.
- Angles are radians.
- Static helpers accept only `Transform2D` and planned `geometry` class
  instances unless a method explicitly documents array conversion.

## Constructors

- `Transform2D.new(a, b, c, d, tx, ty)` creates a transform from explicit
  components.
- `Transform2D.identity()` returns the identity transform.
- `Transform2D.translation(x, y)` returns a translation.
- `Transform2D.scale(sx, sy)` returns a non-uniform scale.
- `Transform2D.uniform_scale(s)` returns a uniform scale.
- `Transform2D.rotation(radians)` returns a rotation around the origin.
- `Transform2D.rotation_around(radians, point)` returns a rotation around a
  `geometry.Point`.
- `Transform2D.skew(x_radians, y_radians)` returns an x/y skew transform.
- `Transform2D.from_array(values)` accepts `[a, b, c, d, tx, ty]`.
- `Transform2D.to_array(transform)` returns `[a, b, c, d, tx, ty]`.

## Operations

- `Transform2D.compose(a, b)` returns the transform that applies `b` first, then
  `a`. This matches matrix multiplication order `a * b`.
- `Transform2D.translate(transform, x, y)` composes a translation after an
  existing transform.
- `Transform2D.scale_by(transform, sx, sy)` composes a scale after an existing
  transform.
- `Transform2D.rotate(transform, radians)` composes a rotation after an existing
  transform.
- `Transform2D.determinant(transform)` returns `a * d - b * c`.
- `Transform2D.invertible?(transform)` returns false when the determinant is
  effectively zero.
- `Transform2D.inverse(transform)` returns the inverse transform or raises a
  clear error for non-invertible transforms.
- `Transform2D.equal?(a, b)` compares all fields exactly.
- `Transform2D.nearly_equal?(a, b, epsilon)` compares fields within `epsilon`.

## Geometry Integration

- `Transform2D.apply_point(transform, point)` accepts a `geometry.Point` and
  returns a `geometry.Point`.
- `Transform2D.apply_vector2(transform, vector)` accepts a `geometry.Vector2`
  and returns a `geometry.Vector2`.
- Applying a transform to a vector ignores translation and uses only the linear
  part:
  - `x' = a * x + c * y`
  - `y' = b * x + d * y`
- `Transform2D.apply_rect(transform, rect)` accepts a `geometry.Rect` and
  returns the axis-aligned bounding rectangle of the transformed four corners.
- `Transform2D.apply_size(transform, size)` accepts a `geometry.Size` and
  returns the axis-aligned size of the transformed rectangle from `(0, 0)` to
  `(width, height)`.
- Geometry integration may be implemented after the planned `geometry` stdlib
  lands. If `transform2d` lands first, the geometry-dependent methods must land
  in the same goal run once `geometry` is available.

## Matrix Interop

- `Transform2D.to_matrix(transform)` returns a 3x3 `matrix.Matrix` value when
  the class-style matrix API is available.
- `Transform2D.from_matrix(matrix)` accepts a 3x3 affine matrix with bottom row
  `[0, 0, 1]` and returns a `Transform2D`.
- Matrix interop must not force callers to use `matrix.Matrix` for normal 2D
  transform workflows.

## Scope

- `lib/transform2d/Transform2D.tya`
- `tests/stdlib_transform2d_test.tya`
- `docs/STDLIB.md`
- Next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`
- Optional example under `examples/transform2d/`

## Out of Scope

- 3D transforms.
- Quaternions.
- Perspective projection.
- Scene graphs, transform hierarchies, cameras, sprites, or animation systems.
- Operator overloading.
- Native code.
- Mutable transform objects.
- Coupling to `raylib`, `image`, or any game framework.
- Replacing the general-purpose `matrix` stdlib.

## Acceptance Criteria

- `import transform2d as transform2d` exposes `transform2d.Transform2D`.
- Constructors return `Transform2D` instances.
- Constructed transforms expose `a`, `b`, `c`, `d`, `tx`, and `ty` fields and
  have `.class == transform2d.Transform2D`.
- `identity`, `translation`, `scale`, `uniform_scale`, `rotation`,
  `rotation_around`, and `skew` produce deterministic components.
- `compose(a, b)` applies `b` first, then `a`, and this order is covered by
  tests.
- `apply_point` and `apply_vector2` produce the documented results and differ
  correctly on translation.
- `apply_rect` returns an axis-aligned bounding rectangle for transformed
  corners.
- `determinant`, `invertible?`, and `inverse` work for representative
  transforms.
- Non-invertible transforms raise clear `transform2d` errors from `inverse`.
- `to_array`, `from_array`, exact equality, and nearly-equal comparison work.
- Matrix interop works once class-style `matrix.Matrix` is available.
- Existing `math`, `matrix`, and `geometry` stdlib tests remain green.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run 'Test.*Transform2D|Test.*Geometry|Test.*Matrix|Test.*Math' -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

Manual smoke after implementation:

```sh
tya run examples/transform2d/compose.tya
```

## Dependencies

- Uses existing numeric and `math.Math` helpers.
- Depends on the planned `geometry` stdlib for `Point`, `Vector2`, `Rect`, and
  `Size` integration.
- Matrix interop should wait for or align with the class-style `matrix.Matrix`
  cleanup from the stdlib class-style PRD.
- Should remain independent from planned `image`, `raylib`, and game-specific
  packages, though those packages may depend on `transform2d`.

## Open Questions

None.
