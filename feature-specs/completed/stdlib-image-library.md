---
status: completed
goal_ready: false
---

# Feature: Stdlib Image Library

## Goal

Add a standard `image` library so Tya programs can inspect, decode, transform,
and write common raster image files without depending on external packages for
basic image workflows.

## Context

Tya already has bytes, file I/O, asset embedding, HTTP static serving, and
native package support. Those features let programs move image bytes around, but
there is no standard way to read dimensions, decode pixels, resize images, crop
sprites, or write simple image outputs.

The first image stdlib should stay narrow. It should cover common raster formats
and basic pixel operations, not become a full graphics engine. SDL2, raylib,
canvas APIs, GPU drawing, and game-rendering bindings remain external packages.

## Behavior

- Add a public `image` stdlib package.
- Import shape:

  ```tya
  import image as image

  img = image.Image.read("logo.png")
  thumb = img.resize(128, 128, { fit: "contain" })
  thumb.write("logo-thumb.png")
  ```

- Public classes:
  - `image.Image`
  - `image.Codec`
- `Image.read(path)` reads an image file and returns an `Image`.
- `Image.decode(bytes)` decodes image bytes and returns an `Image`.
- `Image.decode(bytes, options)` accepts format and frame options.
- `img.write(path)` writes using the file extension when the format is not
  specified.
- `img.write(path, options)` accepts explicit format and encoder options.
- `img.encode(format)` returns encoded bytes.
- `img.encode(format, options)` accepts encoder options.
- `Codec.identify(bytes)` returns metadata without decoding the full pixel
  buffer when the format permits it.
- `Codec.identify_file(path)` reads enough bytes to identify the file.
- Metadata dictionary fields:
  - `format`: `"png"`, `"jpeg"`, `"gif"`, `"bmp"`, or `"ppm"`,
  - `width`,
  - `height`,
  - `frames`,
  - `animated`,
  - `has_alpha`,
  - `color_space`.

## Formats

- Decode support:
  - PNG,
  - JPEG,
  - GIF,
  - BMP,
  - PPM/PGM/PBM Netpbm family.
- Encode support:
  - PNG,
  - JPEG,
  - BMP,
  - PPM.
- GIF encoding is out of scope for the first version.
- Animated GIF decoding is supported as frame access:

  ```tya
  img = image.Image.decode(bytes, { frame: 0 })
  frames = image.Image.decode_frames(bytes)
  ```

- Multi-frame images default to frame 0 when decoded through `Image.decode`.
- Unsupported or malformed formats raise clear image errors that name the format
  when known.

## Image Model

- `Image` stores pixels as 8-bit RGBA in row-major order.
- `img.width` and `img.height` expose dimensions.
- `img.format` stores the source format when known, or `nil` for newly created
  images.
- `img.has_alpha?()` returns true when at least one pixel can carry alpha.
- `img.bytes()` returns a copy of raw RGBA bytes.
- `Image.new(width, height)` creates a transparent RGBA image.
- `Image.new(width, height, color)` fills with the given color.
- Image operations accept `color.Color` instances from the stdlib `color`
  package.
- Channels are integers in `0..255`; out-of-range values raise an error.
- `img.pixel(x, y)` returns a color.
- `img.set_pixel(x, y, color)` mutates the pixel and returns `nil`.
- Pixel coordinates are zero-based.
- Out-of-bounds pixel access raises an image error.

## Operations

- `img.crop(x, y, width, height)` returns a new image.
- `img.resize(width, height)` returns a new image using deterministic nearest
  neighbor scaling.
- `img.resize(width, height, options)` supports:
  - `{ filter: "nearest" }`,
  - `{ filter: "bilinear" }`,
  - `{ fit: "stretch" }`,
  - `{ fit: "contain" }`,
  - `{ fit: "cover" }`,
  - `{ background: Color.rgb(...) }` for contain padding.
- `img.flip_horizontal()` returns a new image.
- `img.flip_vertical()` returns a new image.
- `img.rotate90()` returns a new image rotated clockwise.
- `img.grayscale()` returns a new image.
- `img.composite(over, x, y)` alpha-composites another image and returns a new
  image.
- Operations do not mutate the receiver unless the method name is explicitly
  mutating, such as `set_pixel`.

## Encoder Options

- PNG:
  - `{ compression: "default" | "fast" | "best" }`
- JPEG:
  - `{ quality: 1..100 }`, default 90,
  - alpha is composited over white unless `{ background: Color... }` is passed.
- BMP and PPM:
  - no compression options in the first version.
- Unknown options raise an image error in strict mode and are ignored otherwise
  only if the existing stdlib option style already does that. Prefer raising for
  unknown encoder options to catch mistakes.

## Implementation Notes

- Keep the public API in Tya under `lib/image/`.
- Use runtime/native-backed builtins only for codec-heavy decode/encode work.
- Prefer dependency-free, vendored C codec implementation files when needed so
  `image` does not require system libraries such as ImageMagick, libpng, libjpeg,
  or GraphicsMagick at runtime.
- Keep the codec boundary small:
  - Tya code owns API shape, validation, operation composition, and docs.
  - C/runtime code owns byte-level decoding and encoding.
- All decoded pixels crossing into Tya use the same RGBA byte layout.
- Large images should fail with a clear error before integer overflow or
  impossible allocation. The implementation must check `width * height * 4`.

## Scope

- `lib/image/Image.tya`
- `lib/image/Codec.tya`
- Runtime and codegen/checker builtin registration needed for image codecs.
- C runtime codec helpers or vendored codec sources.
- Tests for metadata, decode, encode, round-trip, pixel access, and operations.
- Small binary fixtures under `tests/testdata/` or generated deterministic
  fixtures checked into the repo.
- `docs/STDLIB.md`
- Next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`
- Any build-system changes needed to compile the image codec runtime on Linux
  and macOS.

## Out of Scope

- SVG, WebP, AVIF, HEIC, TIFF, ICO, and PDF.
- Color management beyond preserving the decoded sRGB-style RGBA values.
- EXIF orientation, EXIF metadata editing, IPTC, or XMP.
- Animated image encoding.
- Streaming decode or tiled processing.
- GPU acceleration, drawing primitives, fonts, text rendering, canvas APIs,
  windows, SDL2, raylib, or game loops.
- ImageMagick-compatible command surface.
- Lossless JPEG transforms.

## Acceptance Criteria

- `import image as image` exposes `Image` and `Codec`.
- `Image` operations that accept or return colors use stdlib `color.Color`
  instances.
- `Image.read("fixture.png")` decodes PNG dimensions and pixels correctly.
- `Image.read("fixture.jpg")` decodes JPEG dimensions and representative pixels
  within a documented lossy tolerance.
- `Image.read("fixture.gif")` decodes frame 0 and reports GIF metadata.
- `Image.read("fixture.bmp")` decodes BMP dimensions and pixels correctly.
- `Image.read("fixture.ppm")` decodes Netpbm dimensions and pixels correctly.
- `Codec.identify_file(path)` reports format, dimensions, alpha, frame count,
  and animation status without requiring full image rendering in the public API.
- `Image.new`, `pixel`, and `set_pixel` work with zero-based coordinates.
- `crop`, `resize`, flips, `rotate90`, `grayscale`, and `composite` return
  deterministic images.
- `encode("png")` and `write("out.png")` produce PNG bytes that `Image.decode`
  can read back.
- `encode("jpeg", { quality: 80 })` produces readable JPEG output.
- `encode("bmp")` and `encode("ppm")` produce readable output.
- Invalid headers, unsupported formats, out-of-bounds pixels, invalid channel
  values, impossible dimensions, and unknown formats raise clear errors.
- Non-image stdlib behavior remains unchanged.
- The self-host fixed point remains green.

## Verification

```sh
go test ./internal/checker ./internal/codegen -count=1
go test ./tests -run 'Test.*Image|TestV.*Script' -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

Manual smoke examples after implementation:

```sh
tya run examples/image/identify.tya tests/testdata/image/logo.png
tya run examples/image/thumbnail.tya tests/testdata/image/logo.png /tmp/logo-thumb.png
```

## Dependencies

- Requires the existing bytes and file stdlib behavior.
- May reuse asset embedding for examples and fixtures.
- Native package support is not required because this is a bundled stdlib, but
  the implementation should follow the same runtime-boundary discipline.

## Open Questions

None.
