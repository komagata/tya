# tree-sitter-tya

Tree-sitter grammar scaffold for Tya syntax highlighting. This is intended for
editor integrations that consume Tree-sitter queries and for a future GitHub
Linguist registration.

The grammar focuses on highlighting-safe parsing rather than full compiler
validation. The Tya compiler remains the source of truth for syntax errors.

## Files

- [`grammar.js`](./grammar.js) — Tree-sitter grammar definition
- [`queries/highlights.scm`](./queries/highlights.scm) — highlight captures
- [`package.json`](./package.json) — npm metadata for Tree-sitter tooling

## Generate

```sh
cd editors/tree-sitter-tya
npm install
npx tree-sitter generate
```

GitHub Linguist registration should reference this grammar once it is generated
and published in the form Linguist requires.
