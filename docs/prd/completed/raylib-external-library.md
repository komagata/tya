---
status: completed
goal_ready: false
---

# Feature: Raylib External Library

## Goal

Create an external Tya package that exposes a practical subset of raylib through
the native package system, so Tya programs can build small games, visual demos,
and asset-embedding examples without adding graphics bindings to the standard
library.

## Context

Tya has native package support for third-party C bindings and asset embedding
for shipping game assets in one binary. Earlier roadmap notes explicitly keep
SDL2 and raylib out of stdlib. raylib is a good first graphics binding because
its C API is compact, beginner-friendly, and available through `pkg-config` on
many systems.

Assumed repository and package identity:

- repository: `https://github.com/komagata/tya-raylib`
- package name: `raylib`
- import path: `import raylib as raylib`
- first release target: `v0.1.0`

## Behavior

- Provide an external native package with this layout:

  ```text
  tya-raylib/
    tya.toml
    src/raylib/
      Window.tya
      Drawing.tya
      Vector2.tya
      Rectangle.tya
      Texture.tya
      Image.tya
      Input.tya
      Time.tya
      Audio.tya
    native/raylib_binding.c
    include/raylib_binding.h
    tests/raylib_test.tya
    examples/
    README.md
  ```

- Applications consume the package through a git dependency:

  ```toml
  [dependencies]
  raylib = { git = "https://github.com/komagata/tya-raylib", tag = "v0.1.0" }
  ```

- The package manifest declares the native dependency through `pkg-config`:

  ```toml
  [native]
  sources = ["native/raylib_binding.c"]
  headers = ["include/raylib_binding.h"]
  include_dirs = ["include"]
  pkg_config = ["raylib"]
  cflags = []
  ldflags = []
  ```

- Public API follows class-style Tya wrappers over raylib:

  ```tya
  import color as color
  import raylib as raylib

  raylib.Window.open(800, 450, "Hello Tya")
  raylib.Time.set_target_fps(60)

  while not raylib.Window.close_requested?()
    raylib.Drawing.begin()
    raylib.Drawing.clear(color.Color.white())
    raylib.Drawing.text("Hello, raylib", 190, 200, 20, color.Color.black())
    raylib.Drawing.end()

  raylib.Window.close()
  ```

- The wrapper should keep method names idiomatic Tya snake_case while preserving
  raylib concepts.
- Native resources are represented as Tya values or resource handles that users
  close/unload explicitly.

## API Surface

- Window and lifecycle:
  - `Window.open(width, height, title)`
  - `Window.close()`
  - `Window.close_requested?()`
  - `Window.ready?()`
  - `Window.set_title(title)`
  - `Window.set_size(width, height)`
  - `Window.width()`
  - `Window.height()`
- Drawing:
  - `Drawing.begin()`
  - `Drawing.end()`
  - `Drawing.clear(color)`
  - `Drawing.fps(x, y)`
  - `Drawing.text(text, x, y, size, color)`
  - `Drawing.line(x1, y1, x2, y2, color)`
  - `Drawing.circle(x, y, radius, color)`
  - `Drawing.rectangle(x, y, width, height, color)`
  - `Drawing.texture(texture, x, y, color)`
- Colors and value helpers:
  - drawing APIs accept stdlib `color.Color` instances,
  - the package may expose raylib-specific color constants as convenience
    methods returning `color.Color` values, such as `ray_white` and `blank`,
  - `Vector2.new(x, y)`
  - `Rectangle.new(x, y, width, height)`
- Input:
  - `Input.key_down?(key)`
  - `Input.key_pressed?(key)`
  - `Input.mouse_x()`
  - `Input.mouse_y()`
  - `Input.mouse_down?(button)`
  - documented key and mouse constants.
- Time:
  - `Time.set_target_fps(fps)`
  - `Time.frame_time()`
  - `Time.fps()`
- Textures and images:
  - `Texture.load(path)`
  - `Texture.load_from_bytes(name, bytes)`
  - `texture.unload()`
  - `texture.width`
  - `texture.height`
  - `Image.load(path)`
  - `Image.load_from_bytes(name, bytes)`
  - `image.unload()`
- Audio, if available with the host raylib build:
  - `Audio.init()`
  - `Audio.close()`
  - `Audio.sound(path)`
  - `Audio.sound_from_bytes(name, bytes)`
  - `sound.play()`
  - `sound.unload()`

## Asset Embedding

- The package must include examples that load embedded assets:

  ```tya
  embed "assets/player.png" as player_png

  texture = raylib.Texture.load_from_bytes("player.png", player_png)
  ```

- `load_from_bytes` requires a filename or extension hint so raylib can detect
  the asset type.
- Failed loads raise clear errors that name the asset.

## Native Boundary

- Native C wrappers should be thin and explicit.
- The wrapper owns conversion between Tya values and raylib structs.
- Color conversion uses stdlib `color.Color`.
- The public Tya API should not expose raw pointers.
- Resources that must be unloaded by raylib, such as textures, images, sounds,
  and music, must have explicit `unload()` methods.
- Double unload is a no-op or a clear error; choose one behavior and test it.
  Prefer no-op for user ergonomics.
- The package should avoid relying on finalizers for correctness.

## Scope

- New external repository `komagata/tya-raylib`.
- `tya.toml` native package manifest using `pkg_config = ["raylib"]`.
- Tya wrapper classes under `src/raylib/`.
- C native binding under `native/` and public package header under `include/`.
- Examples:
  - hello window,
  - shapes,
  - keyboard movement,
  - embedded texture,
  - optional audio.
- Tests:
  - compile/link smoke,
  - value conversion for stdlib `color.Color`, `Vector2`, and `Rectangle`,
  - resource lifecycle wrappers where they can be tested without an interactive
    display,
  - example build checks.
- README documenting installation, host dependencies, API coverage, asset
  loading, platform notes, and cleanup rules.

## Out of Scope

- Adding raylib to Tya stdlib.
- Covering the entire raylib API in v0.1.0.
- 3D camera/model APIs in the first version.
- Shader APIs in the first version.
- Game framework abstractions such as scenes, entities, ECS, physics, or tile
  maps.
- Bundling raylib source or binaries.
- WASM/browser raylib builds.
- Cross-platform packaging of host raylib itself.
- Replacing the planned stdlib `image` package.

## Acceptance Criteria

- A separate `komagata/tya-raylib` repository contains a valid native Tya
  package manifest.
- A project can depend on the package by git URL, run `tya install`, import
  `raylib`, and compile/link against host raylib through `pkg-config`.
- `examples/hello.tya` opens a window, draws text, and closes cleanly.
- Shape drawing methods render without crashing in a smoke example.
- Keyboard input can move a rectangle or sprite in an example.
- Embedded PNG bytes can be loaded into a texture and drawn.
- Textures/images/sounds expose explicit unload methods and handle double unload
  according to the documented behavior.
- Missing raylib `pkg-config` support fails with a diagnostic from the native
  package build path.
- The package test suite passes through `tya test` where a display is available.
- Headless CI documents which tests are compile-only or skipped when no display
  is present.
- A fixture app proves path dependency consumption from outside the package
  repository.

## Verification

```sh
pkg-config --exists raylib
tya install
tya doctor native
tya test
tya run examples/hello.tya
tya run examples/embedded_texture.tya
```

For this repository's spec tracking only:

```sh
test -f docs/prd/raylib-external-library.md
rg -n "Raylib External Library" docs/prd/raylib-external-library.md
```

## Dependencies

- Requires completed native package support.
- Requires host raylib development files discoverable through `pkg-config`.
- Uses existing Tya asset embedding for embedded texture examples.

## Open Questions

None.
