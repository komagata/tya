---
status: completed
goal_ready: false
---

# Feature: Geometry Stdlib Library

## Goal

Add a pure Tya `geometry` standard library package with small 2D/3D vector and
shape helpers that can be reused by games, image tooling, UI/layout code,
charts, and general spatial calculations.

## Context

Tya already has `math.Math` for scalar math and `matrix.Matrix` for general
matrix operations. Planned graphics and image work will also need common
geometry values, but those values should not be owned by a game-specific
package or by `raylib`.

The first geometry library should be simple, deterministic, and class-style. It
should not introduce operator overloading or native code. Geometry constructors
return class instances, not dictionaries, so values carry their type and can grow
instance methods later without changing the public model.

## Behavior

- Add a public `geometry` stdlib package.
- Import shape:

  ```tya
  import geometry as geo

  p = geo.Vector2.new(10, 20)
  v = geo.Vector2.new(3, 4)
  unit = geo.Vector2.normalize(v)

  r = geo.Rect.new(0, 0, 100, 50)
  if geo.Rect.contains_point?(r, p)
    println "inside"
  ```

- Public classes:
  - `geometry.Vector2`
  - `geometry.Vector3`
  - `geometry.Point`
  - `geometry.Size`
  - `geometry.Rect`
  - `geometry.Circle`
- Geometry values are instances of their public classes.
- Instances expose numeric public fields:
  - `Vector2`: `x`, `y`
  - `Vector3`: `x`, `y`, `z`
  - `Point`: `x`, `y`
  - `Size`: `width`, `height`
  - `Rect`: `x`, `y`, `width`, `height`
  - `Circle`: `x`, `y`, `radius`
- Geometry values are immutable by API convention. Methods do not mutate inputs;
  operations return new class instances.
- Constructors validate numeric fields and raise geometry errors for invalid
  input.
- Static helpers accept only instances of the expected geometry class unless a
  method explicitly documents conversion from arrays or related geometry types.

## Vector2

- `Vector2.new(x, y)`
- `Vector2.zero()`
- `Vector2.one()`
- `Vector2.add(a, b)`
- `Vector2.sub(a, b)`
- `Vector2.scale(v, k)`
- `Vector2.div(v, k)` raises when `k == 0`
- `Vector2.dot(a, b)`
- `Vector2.length(v)`
- `Vector2.length_squared(v)`
- `Vector2.distance(a, b)`
- `Vector2.distance_squared(a, b)`
- `Vector2.normalize(v)` returns zero for the zero vector.
- `Vector2.lerp(a, b, t)`
- `Vector2.clamp(v, min_v, max_v)`
- `Vector2.equal?(a, b)`
- `Vector2.nearly_equal?(a, b, epsilon)`
- `Vector2.to_array(v)` returns `[x, y]`.
- `Vector2.from_array(values)` accepts a two-number array.

## Vector3

- `Vector3.new(x, y, z)`
- `Vector3.zero()`
- `Vector3.one()`
- `Vector3.add(a, b)`
- `Vector3.sub(a, b)`
- `Vector3.scale(v, k)`
- `Vector3.div(v, k)` raises when `k == 0`
- `Vector3.dot(a, b)`
- `Vector3.cross(a, b)`
- `Vector3.length(v)`
- `Vector3.length_squared(v)`
- `Vector3.distance(a, b)`
- `Vector3.distance_squared(a, b)`
- `Vector3.normalize(v)` returns zero for the zero vector.
- `Vector3.lerp(a, b, t)`
- `Vector3.equal?(a, b)`
- `Vector3.nearly_equal?(a, b, epsilon)`
- `Vector3.to_array(v)` returns `[x, y, z]`.
- `Vector3.from_array(values)` accepts a three-number array.

## Point and Size

- `Point.new(x, y)`
- `Point.from_vector(v)`
- `Point.to_vector(p)`
- `Point.translate(p, v)`
- `Point.distance(a, b)`
- `Point.equal?(a, b)`
- `Size.new(width, height)`
- `Size.zero()`
- `Size.area(size)`
- `Size.aspect_ratio(size)` raises when `height == 0`.
- `Size.equal?(a, b)`
- Width, height, x, and y may be negative only where the operation explicitly
  allows it. `Size.new` rejects negative dimensions.

## Rect

- `Rect.new(x, y, width, height)`
- `Rect.from_points(a, b)` creates the smallest rect spanning two points.
- `Rect.from_center(center, size)`
- `Rect.left(rect)`
- `Rect.right(rect)`
- `Rect.top(rect)`
- `Rect.bottom(rect)`
- `Rect.center(rect)`
- `Rect.size(rect)`
- `Rect.area(rect)`
- `Rect.empty?(rect)`
- `Rect.contains_point?(rect, point)`
- `Rect.contains_rect?(outer, inner)`
- `Rect.intersects?(a, b)`
- `Rect.intersection(a, b)` returns an empty rect when there is no overlap.
- `Rect.union(a, b)`
- `Rect.expand(rect, amount)`
- `Rect.translate(rect, vector)`
- `Rect.inflate(rect, dx, dy)`
- `Rect.clamp_point(rect, point)`
- `Rect.equal?(a, b)`
- `Rect.new` rejects negative width or height. Zero width or height is allowed
  and makes `Rect.empty?` true.
- Containment uses inclusive left/top edges and exclusive right/bottom edges:
  `x <= point.x < x + width`, `y <= point.y < y + height`.

## Circle

- `Circle.new(x, y, radius)`
- `Circle.from_center(center, radius)`
- `Circle.center(circle)`
- `Circle.area(circle)`
- `Circle.circumference(circle)`
- `Circle.contains_point?(circle, point)`
- `Circle.intersects_circle?(a, b)`
- `Circle.intersects_rect?(circle, rect)`
- `Circle.bounding_rect(circle)`
- `Circle.translate(circle, vector)`
- `Circle.equal?(a, b)`
- `Circle.new` rejects negative radius. Zero radius is allowed.

## Scope

- `stdlib/geometry/Vector2.tya`
- `stdlib/geometry/Vector3.tya`
- `stdlib/geometry/Point.tya`
- `stdlib/geometry/Size.tya`
- `stdlib/geometry/Rect.tya`
- `stdlib/geometry/Circle.tya`
- `tests/stdlib_geometry_test.tya`
- `docs/STDLIB.md`
- Next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`
- Optional examples under `examples/geometry/`

## Out of Scope

- Operator overloading.
- Native code.
- Mutable vector objects.
- Matrix transforms or affine transform composition.
- Quaternions.
- Polygon clipping, triangulation, convex hulls, or pathfinding.
- Physics simulation.
- Coupling to `raylib`, `image`, or any game framework.
- Color values.

## Acceptance Criteria

- `import geometry as geo` exposes `Vector2`, `Vector3`, `Point`, `Size`,
  `Rect`, and `Circle`.
- Constructors return instances of the documented geometry classes.
- Constructed values expose the documented public fields and have the expected
  `.class`.
- Constructors reject non-number fields.
- `Vector2` arithmetic, dot product, length, distance, normalization, lerp,
  clamp, array conversion, and equality functions work.
- `Vector3` arithmetic, dot product, cross product, length, distance,
  normalization, lerp, array conversion, and equality functions work.
- `Point` and `Size` helpers work, including negative-size rejection and
  aspect-ratio error on zero height.
- `Rect` containment, intersection, union, expansion, translation, inflation,
  and empty-rect behavior are deterministic and documented.
- `Circle` containment, circle intersection, rect intersection, bounding rect,
  and translation work.
- Zero-vector normalization returns zero instead of raising or producing NaN.
- Divide-by-zero helpers raise clear geometry errors.
- Existing `math` and `matrix` stdlib tests remain green.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run 'Test.*Geometry|Test.*Math|Test.*Matrix' -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

Manual smoke after implementation:

```sh
tya run examples/geometry/collision.tya
```

## Dependencies

- Uses existing `math.Math` scalar helpers where useful.
- Should remain independent from planned `color`, `transform2d`, `image`, and
  `raylib` packages.

## Open Questions

None.
