# Tya Editor Token Taxonomy

Editor integrations should use this shared token taxonomy so `.tya` files look
consistent across VS Code, Vim / Neovim, Emacs, and GitHub.

## Filetype

- Language id: `tya`
- File extension: `.tya`
- TextMate scope: `source.tya`

## Tokens

| Tya syntax | Token group | TextMate scope |
|---|---|---|
| `# comment` | line comment | `comment.line.number-sign.tya` |
| `"text"` / `"""text"""` | string | `string.quoted.double.tya` |
| `b"raw"` | bytes string | `string.quoted.double.bytes.tya` |
| `{name}` inside strings | interpolation | `meta.interpolation.tya` |
| `123`, `0x2a`, `0b1010` | number | `constant.numeric.tya` |
| `true`, `false`, `nil` | literal | `constant.language.tya` |
| `if`, `elseif`, `else`, `while`, `for`, `in`, `break`, `continue`, `return`, `raise`, `try`, `catch`, `match`, `case`, `when`, `select`, `receive`, `send`, `timeout`, `default` | control keyword | `keyword.control.tya` |
| `class`, `module`, `interface`, `implements`, `extends`, `abstract`, `final`, `private`, `static`, `initialize`, `import`, `as` | declaration keyword | `storage.type.tya` |
| `self`, `Self`, `super` | language variable | `variable.language.tya` |
| `spawn`, `await`, `scope` | concurrency keyword | `keyword.other.tya` |
| `and`, `or`, `not` | logical operator | `keyword.operator.logical.tya` |
| `->`, `==`, `!=`, `<=`, `>=`, `<<`, `>>`, `+`, `-`, `*`, `/`, `%`, `=`, `.`, `,`, `:`, `&`, `|`, `^`, `~` | operator / punctuation | `keyword.operator.tya` |
| `Name` in `class Name`, `interface Name`, `module Name` | type / namespace declaration | `entity.name.type.tya` / `entity.name.namespace.tya` |
| `name` in `name = ->` or `name: ->` | function / method declaration | `entity.name.function.tya` |

Editor-specific token names may differ, but they should map back to these
semantic groups.
