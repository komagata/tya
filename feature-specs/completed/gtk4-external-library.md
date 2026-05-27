---
status: completed
goal_ready: false
---

# Feature: GTK4 External Library

## Goal

Create an external Tya package that exposes a practical GTK4 binding through the
native package system, so Tya programs can build small Linux desktop GUI tools,
settings panels, and Omarchy/Hyprland-friendly utilities without adding GTK to
the standard library.

## Context

Tya can produce single-binary tools and already has native package support for
external C libraries. Omarchy and other Hyprland-based desktops are a good fit
for lightweight Linux-native GUI tools that do not need traditional menu bars or
desktop-environment window chrome. GTK4 is widely packaged on Linux and provides
common widgets, CSS styling, accessibility integration, clipboard support, and
Wayland/X11 backends.

This should be an external native package, not stdlib. The binding does not need
to cover every GTK4 API in the first release, but it should be broad enough that
ordinary Linux GUI applications do not immediately hit missing fundamentals.
Settings panels, file tools, database viewers, small editors, dashboards, and
Omarchy utilities should be practical without dropping into C for common UI
work.

Assumed repository and package identity:

- repository: `https://github.com/komagata/tya-gtk4`
- package name: `gtk4`
- import path: `import gtk4`
- first release target: `v0.1.0`

## Behavior

- Provide an external native package with this layout:

  ```text
  tya-gtk4/
    tya.toml
    src/gtk4/
      Application.tya
      Window.tya
      Box.tya
      Button.tya
      Label.tya
      Entry.tya
      TextView.tya
      Switch.tya
      CheckButton.tya
      ListBox.tya
      Image.tya
      Picture.tya
      ScrolledWindow.tya
      Stack.tya
      Notebook.tya
      HeaderBar.tya
      MenuButton.tya
      Popover.tya
      DropDown.tya
      ListStore.tya
      ColumnView.tya
      TreeExpander.tya
      Action.tya
      Shortcut.tya
      Css.tya
      Clipboard.tya
      Dialog.tya
      FileDialog.tya
      AlertDialog.tya
      ProgressBar.tya
      Spinner.tya
    native/gtk4_binding.c
    include/gtk4_binding.h
    tests/gtk4_test.tya
    examples/
    README.md
  ```

- Applications consume the package through a git dependency:

  ```toml
  [dependencies]
  gtk4 = { git = "https://github.com/komagata/tya-gtk4", tag = "v0.1.0" }
  ```

- The package manifest declares the native dependency through `pkg-config`:

  ```toml
  [native]
  sources = ["native/gtk4_binding.c"]
  headers = ["include/gtk4_binding.h"]
  include_dirs = ["include"]
  pkg_config = ["gtk4"]
  cflags = []
  ldflags = []
  ```

- Public API follows class-style Tya wrappers over GTK4:

  ```tya
  import gtk4

  app = gtk4.Application.new("org.example.Settings")
  app.on_activate(-> 
    window = gtk4.Window.new(app)
    window.set_title("Settings")
    window.set_default_size(480, 320)
    window.set_decorated(false)

    box = gtk4.Box.vertical(12)
    label = gtk4.Label.new("Omarchy Tool")
    button = gtk4.Button.new("Apply")
    button.on_clicked(-> println("clicked"))

    box.append(label)
    box.append(button)
    window.set_child(box)
    window.present()
  )

  app.run(args)
  ```

- Native GTK objects are represented as Tya class instances or resource handles.
- Public Tya APIs must not expose raw pointers.
- Widgets that wrap GObjects must keep references alive while Tya values are
  alive and release references when explicitly destroyed or when the package's
  lifecycle rules allow it.
- Callbacks from GTK signals into Tya functions must be supported for the
  documented signals.
- Missing GTK4 host dependencies fail through the native package build path with
  clear diagnostics.

## API Surface

- Application lifecycle:
  - `Application.new(app_id)`
  - `Application.new(app_id, options)`
  - `app.on_activate(fn)`
  - `app.on_open(fn)`
  - `app.add_action(action)`
  - `app.set_accels_for_action(action_name, shortcuts)`
  - `app.run(args)`
  - `app.quit()`
- Window:
  - `Window.new(app)`
  - `Window.application(app)`
  - `window.present()`
  - `window.close()`
  - `window.set_title(title)`
  - `window.set_default_size(width, height)`
  - `window.set_resizable(value)`
  - `window.set_decorated(value)`
  - `window.set_child(widget)`
  - `window.child()`
  - `window.add_action(action)`
  - `window.set_modal(value)`
  - `window.fullscreen()`
  - `window.unfullscreen()`
  - `window.on_close_request(fn)`
- Layout:
  - `Box.vertical(spacing)`
  - `Box.horizontal(spacing)`
  - `box.append(widget)`
  - `box.prepend(widget)`
  - `box.remove(widget)`
  - `box.set_margin(top, right, bottom, left)`
  - `box.add_css_class(name)`
  - `box.remove_css_class(name)`
  - `ScrolledWindow.new()`
  - `scrolled.set_child(widget)`
  - `Stack.new()`
  - `stack.add_named(widget, name)`
  - `stack.set_visible_child_name(name)`
  - `Notebook.new()`
  - `notebook.append_page(widget, label)`
  - `notebook.current_page()`
  - `notebook.set_current_page(index)`
  - `HeaderBar.new()`
  - `header.pack_start(widget)`
  - `header.pack_end(widget)`
- Basic widgets:
  - `Label.new(text)`
  - `label.set_text(text)`
  - `label.text()`
  - `Button.new(label)`
  - `button.set_label(label)`
  - `button.on_clicked(fn)`
  - `Entry.new()`
  - `entry.text()`
  - `entry.set_text(text)`
  - `entry.on_changed(fn)`
  - `Switch.new()`
  - `switch.active?()`
  - `switch.set_active(value)`
  - `switch.on_changed(fn)`
  - `CheckButton.new(label)`
  - `check.active?()`
  - `check.set_active(value)`
  - `check.on_toggled(fn)`
  - `ProgressBar.new()`
  - `progress.set_fraction(value)`
  - `progress.set_text(text)`
  - `Spinner.new()`
  - `spinner.start()`
  - `spinner.stop()`
- Lists:
  - `ListBox.new()`
  - `list.append(widget)`
  - `list.remove(widget)`
  - `list.selected_row()`
  - `list.on_row_selected(fn)`
  - `ListStore.new()`
  - `store.append(value)`
  - `store.remove(index)`
  - `store.clear()`
  - `store.len()`
  - `ColumnView.new(store)`
  - `column_view.append_text_column(title, getter)`
  - `column_view.on_activate(fn)`
  - `DropDown.from_strings(values)`
  - `dropdown.selected()`
  - `dropdown.set_selected(index)`
  - `dropdown.on_changed(fn)`
- Text:
  - `TextView.new()`
  - `text_view.text()`
  - `text_view.set_text(text)`
  - `text_view.set_monospace(value)`
  - `text_view.set_editable(value)`
- Images:
  - `Image.from_file(path)`
  - `Image.from_bytes(name, bytes)`
  - `image.set_pixel_size(size)`
  - `Picture.from_file(path)`
  - `Picture.from_bytes(name, bytes)`
  - `picture.set_can_shrink(value)`
  - `picture.set_content_fit(mode)`
- Menus and popovers:
  - `MenuButton.new()`
  - `menu_button.set_label(text)`
  - `menu_button.set_popover(popover)`
  - `Popover.new()`
  - `popover.set_child(widget)`
  - `popover.popup()`
  - `popover.popdown()`
- Actions and shortcuts:
  - `Action.new(name, fn)`
  - `Action.toggle(name, initial, fn)`
  - `Shortcut.new(trigger, action_name)`
  - `widget.add_action(action)`
  - `widget.add_shortcut(shortcut)`
- Dialogs:
  - `Dialog.message(parent, title, body)`
  - `Dialog.error(parent, title, body)`
  - `dialog.present()`
  - `dialog.close()`
  - `AlertDialog.new(title, body)`
  - `alert.add_response(id, label)`
  - `alert.choose(parent, fn)`
  - `FileDialog.open(parent, fn)`
  - `FileDialog.save(parent, fn)`
  - `FileDialog.select_folder(parent, fn)`
- Clipboard:
  - `Clipboard.default()`
  - `clipboard.set_text(text)`
  - `clipboard.read_text(fn)`
- CSS:
  - `Css.load(text)`
  - `Css.load_file(path)`
  - `Css.add_provider(css)`

## Common Application Coverage

The first release should support these ordinary app patterns without custom C:

- single-window settings panels,
- multi-page preference windows through `Stack` or `Notebook`,
- forms with validation,
- file open/save flows,
- list/table views for small to medium data sets,
- text editing areas for logs, notes, or generated output,
- image preview from file or embedded bytes,
- progress and loading states,
- confirmation/error dialogs,
- keyboard shortcuts and actions,
- clipboard copy/paste,
- CSS theming.

Where GTK4's full model/view APIs are too broad for v0.1, provide a pragmatic
wrapper such as `ListStore` + `ColumnView.append_text_column` that covers common
tables. The binding may expose richer model/factory APIs later.

## Styling and Omarchy Expectations

- The package should make it easy to build undecorated, content-first windows:
  - `window.set_decorated(false)`
  - no required header bar,
  - no required menu model,
  - CSS classes on widgets.
- The package should not assume GNOME/libadwaita application structure.
- Examples should include an Omarchy/Hyprland-style undecorated settings panel.
- GTK CSS should be documented as the supported styling mechanism.
- Dark/light theme integration should follow GTK settings when available, but
  custom CSS should be enough for first-version Omarchy tools.

## Native Boundary

- Native C wrappers should be thin and explicit.
- The wrapper owns conversion between Tya values and GTK/GObject pointers.
- Signal callbacks must not call freed Tya closures.
- Async GTK operations such as file dialogs and clipboard reads must call Tya
  callbacks with either a result value or a clear error.
- GTK main-loop ownership must be clear:
  - `app.run(args)` starts the loop,
  - callbacks execute on the GTK main thread,
  - the first version does not need cross-thread UI updates.
- Resource cleanup must be deterministic where practical:
  - `window.close()` closes windows,
  - object references are released by wrapper lifecycle code,
  - the package should avoid relying on finalizers for correctness.
- GTK errors and failed resource loads should raise clear `gtk4` errors.

## Scope

- New external repository `komagata/tya-gtk4`.
- `tya.toml` native package manifest using `pkg_config = ["gtk4"]`.
- Tya wrapper classes under `src/gtk4/`.
- C native binding under `native/` and public package header under `include/`.
- Examples:
  - hello window,
  - undecorated settings panel,
  - form with entries/switches/buttons,
  - multi-page preferences,
  - list/table view,
  - file open/save,
  - image from embedded bytes,
  - clipboard and shortcuts,
  - CSS styling.
- Tests:
  - compile/link smoke,
  - widget construction wrappers,
  - signal callback dispatch where a display is available,
  - action and shortcut registration,
  - file dialog API compile checks,
  - list/table model wrapper behavior,
  - CSS loading,
  - image-from-bytes conversion,
  - example build checks.
- README documenting installation, host dependencies, API coverage, Wayland/X11
  behavior, Omarchy/Hyprland notes, display/headless test notes, and cleanup
  rules.

## Out of Scope

- Adding GTK4 to Tya stdlib.
- Covering the entire GTK4 API in v0.1.0.
- libadwaita bindings.
- gtk4-layer-shell bindings.
- Full custom drawing/canvas APIs.
- Complete low-level GObject introspection.
- Full GTK model/factory coverage beyond the documented list/table helpers.
- Drag and drop in v0.1.0.
- Accessibility customization beyond what GTK widgets provide by default.
- Thread-safe UI updates from background tasks.
- Cross-platform Windows/macOS packaging.
- Bundling GTK itself.

## Acceptance Criteria

- A separate `komagata/tya-gtk4` repository contains a valid native Tya package
  manifest.
- A project can depend on the package by git URL, run `tya install`, import
  `gtk4`, and compile/link against host GTK4 through `pkg-config`.
- `examples/hello.tya` opens a GTK4 window and closes cleanly.
- `examples/settings_panel.tya` creates an undecorated Omarchy/Hyprland-style
  window using labels, buttons, switches, entries, layout, and CSS.
- A multi-page preferences example works through `Stack` or `Notebook`.
- A file-oriented example can open a file dialog, display selected text or image
  content, and save output.
- A list/table example can show rows from Tya data and handle row activation.
- Button, entry, switch, check button, list selection, and close-request
  callbacks can call Tya functions.
- Actions and keyboard shortcuts dispatch to Tya callbacks.
- Clipboard text read/write works.
- Alert/error dialogs and file dialogs call Tya callbacks with success or error
  results.
- Embedded image bytes can be displayed through `Image.from_bytes`.
- Missing GTK4 `pkg-config` support fails with a diagnostic from the native
  package build path.
- Headless CI documents which tests are compile-only or skipped when no display
  is present.
- A fixture app proves path dependency consumption from outside the package
  repository.

## Verification

In the external repository:

```sh
pkg-config --exists gtk4
tya install
tya doctor native
tya test
tya run examples/hello.tya
tya run examples/settings_panel.tya
```

For this repository's spec tracking only:

```sh
test -f feature-specs/gtk4-external-library.md
rg -n "GTK4 External Library" feature-specs/gtk4-external-library.md
```

## Dependencies

- Requires completed native package support.
- Requires host GTK4 development files discoverable through `pkg-config`.
- Uses existing Tya bytes and asset embedding for embedded image examples.
- May interoperate with planned `color`, `image`, and `serialization` stdlibs
  in examples, but the core binding should not depend on them.

## Open Questions

None.
