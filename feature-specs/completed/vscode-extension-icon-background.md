# Feature: VS Code Extension Icon Background

## Goal

Make the Tya VS Code extension icon easier to see in VS Code and marketplace surfaces by replacing its transparent background with a rounded background that matches the homepage top-page base color.

## Context

- The VS Code extension is packaged from `editors/vscode`.
- `editors/vscode/package.json` points marketplace icon metadata at `icon.png`.
- The editable source icon is `editors/vscode/icon.svg`; the generated package icon is `editors/vscode/icon.png`.
- The homepage top-page background uses `--paper: #fbfaf5` in `docs/assets/document.css`, with a subtle grid over that base. The icon should use the base `#fbfaf5` color as a solid fill rather than attempting to reproduce the page grid.
- The current icon can be hard to see when transparent areas blend into VS Code or marketplace backgrounds.

## Behavior

- Update both icon assets:
  - `editors/vscode/icon.svg`
  - `editors/vscode/icon.png`
- The icon background must be a rounded rectangle, not a square or transparent canvas.
- The rounded background fill must be `#fbfaf5`, matching the homepage top-page `--paper` base color.
- Keep the existing Tya mark composition recognizable:
  - same 128x128 canvas
  - same large `T` form
  - same accent wedge
  - same or visually equivalent foreground colors, adjusted only if needed for contrast against `#fbfaf5`
- `icon.png` must remain a 128x128 PNG suitable for VS Code Marketplace packaging.
- `icon.svg` and `icon.png` must visually match. The PNG should be generated from the SVG or otherwise updated from the same design source.
- `editors/vscode/package.json` should continue to reference `"icon": "icon.png"` unless VS Code packaging requirements change.

## Scope

- Update the VS Code extension icon source and packaged icon:
  - `editors/vscode/icon.svg`
  - `editors/vscode/icon.png`
- Verify the package metadata still references the PNG icon.
- If the repo has an existing image generation or conversion command for these assets, use it. Otherwise, use a deterministic local conversion command such as ImageMagick, Inkscape, or another already available tool and document the command in the implementation notes or final response.

## Out of Scope

- Changing the Tya website logo in `docs/assets/tya-logo.png`.
- Changing extension name, publisher, version, README text, release workflow, or marketplace publication settings.
- Redesigning the brand mark beyond adding the requested rounded homepage-color background.
- Releasing or publishing a new extension package.

## Acceptance Criteria

- `editors/vscode/icon.svg` has a rounded `#fbfaf5` background.
- `editors/vscode/icon.png` is 128x128 and visually matches the SVG.
- The PNG no longer has fully transparent background corners or transparent interior areas that make the icon hard to see.
- The Tya mark remains clearly legible at small sizes such as 32x32.
- `editors/vscode/package.json` still points to `icon.png`.
- No website logo assets are changed.

## Verification

```sh
file editors/vscode/icon.png
rg -n '"icon": "icon.png"' editors/vscode/package.json
git diff --check
```
