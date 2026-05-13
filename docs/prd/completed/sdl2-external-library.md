---
status: completed
goal_ready: false
---

# Feature: SDL2 External Library

## Goal

Create an external Tya package that exposes a practical SDL2 binding through the
native package system, so Tya programs can build simple windowed apps, games,
input demos, and asset-embedding examples without adding SDL2 to the standard
library.

## Context

Tya has native package support for third-party C bindings and asset embedding
for shipping assets in one binary. The roadmap explicitly keeps SDL2 out of the
language stdlib and says SDL2 bindings should be published as external package
examples once native dependency and linking conventions are stable.

SDL2 is a lower-level and widely available foundation than raylib. The first
binding should expose enough surface for a simple 2D app while keeping the API
small, class-style, and explicit about resource lifetimes.

Assumed repository and package identity:

- repository: `https://github.com/komagata/tya-sdl2`
- package name: `sdl2`
- import path: `import sdl2 as sdl2`
- first release target: `v0.1.0`

## Behavior

- Provide an external native package with this layout:

  ```text
  tya-sdl2/
    tya.toml
    src/sdl2/
      Sdl.tya
      Window.tya
      Renderer.tya
      Texture.tya
      Surface.tya
      Event.tya
      Keyboard.tya
      Mouse.tya
      Timer.tya
      Rect.tya
      Point.tya
    native/sdl2_binding.c
    include/sdl2_binding.h
    tests/sdl2_test.tya
    examples/
    README.md
  ```

- Applications consume the package through a git dependency:

  ```toml
  [dependencies]
  sdl2 = { git = "https://github.com/komagata/tya-sdl2", tag = "v0.1.0" }
  ```

- The package manifest declares native dependencies through `pkg-config`:

  ```toml
  [native]
  sources = ["native/sdl2_binding.c"]
  headers = ["include/sdl2_binding.h"]
  include_dirs = ["include"]
  pkg_config = ["sdl2"]
  cflags = []
  ldflags = []
  ```

- Public API follows class-style Tya wrappers over SDL2:

  ```tya
  import color as color
  import sdl2 as sdl2

  sdl2.Sdl.init()
  window = sdl2.Window.create("Hello Tya", 800, 450)
  renderer = sdl2.Renderer.create(window)

  running = true
  while running
    event = sdl2.Event.poll()
    while event != nil
      if event.quit?()
        running = false
      event = sdl2.Event.poll()

    renderer.clear(color.Color.black())
    renderer.present()

  renderer.destroy()
  window.destroy()
  sdl2.Sdl.quit()
  ```

- The wrapper keeps method names idiomatic Tya snake_case while preserving SDL2
  concepts.
- Native resources are represented as Tya class instances or resource handles.
- Public Tya APIs must not expose raw pointers.
- Resources that must be freed by SDL2 expose explicit `destroy()` or `free()`
  methods.
- Double destroy/free is a no-op or a clear error. Prefer no-op for user
  ergonomics and test the chosen behavior.

## API Surface

- Lifecycle:
  - `Sdl.init()`
  - `Sdl.init(flags)`
  - `Sdl.quit()`
  - `Sdl.error()`
  - documented init flag constants for video, audio, timer, events, and
    everything.
- Window:
  - `Window.create(title, width, height)`
  - `Window.create(title, width, height, options)`
  - `window.destroy()`
  - `window.title()`
  - `window.set_title(title)`
  - `window.width()`
  - `window.height()`
  - `window.size()`
- Renderer:
  - `Renderer.create(window)`
  - `Renderer.create(window, options)`
  - `renderer.destroy()`
  - `renderer.clear(color)`
  - `renderer.set_draw_color(color)`
  - `renderer.present()`
  - `renderer.draw_point(x, y)`
  - `renderer.draw_line(x1, y1, x2, y2)`
  - `renderer.draw_rect(rect)`
  - `renderer.fill_rect(rect)`
  - `renderer.copy(texture, src_rect, dst_rect)`
- Rect and point values:
  - `Rect.new(x, y, width, height)`
  - `Point.new(x, y)`
  - Rect and point wrappers may also accept stdlib `geometry.Rect` and
    `geometry.Point` where practical.
- Texture and surface:
  - `Texture.load_bmp(renderer, path)`
  - `Texture.load_bmp_from_bytes(renderer, bytes)`
  - `texture.destroy()`
  - `texture.width`
  - `texture.height`
  - `Surface.load_bmp(path)`
  - `Surface.load_bmp_from_bytes(bytes)`
  - `surface.free()`
  - `surface.width`
  - `surface.height`
- Events and input:
  - `Event.poll()` returns an `Event` instance or `nil`.
  - `event.type()`
  - `event.quit?()`
  - `event.key_down?()`
  - `event.key_up?()`
  - `event.key_code()`
  - `event.mouse_button_down?()`
  - `event.mouse_button_up?()`
  - `event.mouse_x()`
  - `event.mouse_y()`
  - documented event, key, and mouse constants.
  - `Keyboard.key_down?(key)` for current keyboard state.
  - `Mouse.x()`, `Mouse.y()`, and `Mouse.button_down?(button)`.
- Timing:
  - `Timer.ticks()`
  - `Timer.delay(ms)`
  - `Timer.performance_counter()`
  - `Timer.performance_frequency()`

## Colors and Geometry

- Renderer drawing APIs accept stdlib `color.Color` instances.
- SDL-specific color constants may be convenience methods returning
  `color.Color`.
- Rect and point APIs should interoperate with planned stdlib `geometry`
  instances when available, but the SDL2 package may keep lightweight
  `sdl2.Rect` and `sdl2.Point` wrappers for direct SDL2 value conversion.

## Asset Embedding

- The package must include examples that load embedded BMP assets:

  ```tya
  embed "assets/player.bmp" as player_bmp

  texture = sdl2.Texture.load_bmp_from_bytes(renderer, player_bmp)
  ```

- Failed loads raise clear errors that name the operation or asset.
- PNG/JPEG loading through SDL_image is out of scope for the first SDL2 package.
  Use BMP for the no-extra-native-dependency v0.1.0 binding.

## Native Boundary

- Native C wrappers should be thin and explicit.
- The wrapper owns conversion between Tya values and SDL structs.
- The wrapper owns SDL error retrieval and should include SDL's error string in
  raised Tya errors when available.
- The package should avoid relying on finalizers for correctness.
- Headless environments should fail or skip display-dependent tests with a
  documented reason instead of hanging.

## Scope

- New external repository `komagata/tya-sdl2`.
- `tya.toml` native package manifest using `pkg_config = ["sdl2"]`.
- Tya wrapper classes under `src/sdl2/`.
- C native binding under `native/` and public package header under `include/`.
- Examples:
  - hello window,
  - shapes,
  - keyboard movement,
  - embedded BMP texture,
  - event loop.
- Tests:
  - compile/link smoke,
  - color, rect, point, event, and resource-handle value conversion,
  - resource lifecycle wrappers where they can be tested without an interactive
    display,
  - example build checks.
- README documenting installation, host dependencies, API coverage, asset
  loading, display/headless notes, and cleanup rules.

## Out of Scope

- Adding SDL2 to Tya stdlib.
- Covering the entire SDL2 API in v0.1.0.
- SDL_image, SDL_mixer, SDL_ttf, SDL_net, or other SDL extension libraries.
- OpenGL/Vulkan context management.
- Game framework abstractions such as scenes, entities, ECS, physics, tile maps,
  cameras, or animation systems.
- Bundling SDL2 source or binaries.
- WASM/browser SDL builds.
- Cross-platform packaging of host SDL2 itself.
- Replacing the planned stdlib `image` package or raylib external package.

## Acceptance Criteria

- A separate `komagata/tya-sdl2` repository contains a valid native Tya package
  manifest.
- A project can depend on the package by git URL, run `tya install`, import
  `sdl2`, and compile/link against host SDL2 through `pkg-config`.
- `examples/hello.tya` opens a window, clears it, processes quit events, and
  closes cleanly.
- Shape drawing methods render without crashing in a smoke example.
- Keyboard input can move a rectangle or sprite in an example.
- Embedded BMP bytes can be loaded into a texture and drawn.
- Window, renderer, texture, and surface resources expose explicit cleanup
  methods and handle double cleanup according to documented behavior.
- Missing SDL2 `pkg-config` support fails with a diagnostic from the native
  package build path.
- The package test suite passes through `tya test` where a display is
  available.
- Headless CI documents which tests are compile-only or skipped when no display
  is present.
- A fixture app proves path dependency consumption from outside the package
  repository.

## Verification

```sh
pkg-config --exists sdl2
tya install
tya doctor native
tya test
tya run examples/hello.tya
tya run examples/embedded_bmp.tya
```

For this repository's spec tracking only:

```sh
test -f docs/prd/sdl2-external-library.md
rg -n "SDL2 External Library" docs/prd/sdl2-external-library.md
```

## Dependencies

- Requires completed native package support.
- Requires host SDL2 development files discoverable through `pkg-config`.
- Uses existing Tya asset embedding for embedded BMP examples.
- Should accept stdlib `color.Color` once the planned color stdlib lands.
- May interoperate with planned `geometry` values, but must not depend on a game
  framework.

## Open Questions

None.
