# tya — GitHub Linguist notes

GitHub syntax highlighting is provided through
[GitHub Linguist](https://github.com/github-linguist/linguist). Linguist can
consume either the TextMate grammar in
[`../vscode/syntaxes/tya.tmLanguage.json`](../vscode/syntaxes/tya.tmLanguage.json)
or the Tree-sitter grammar scaffold in
[`../tree-sitter-tya/`](../tree-sitter-tya/) for `source.tya`.

To register Tya upstream, open a pull request against `github-linguist/linguist`
with a `languages.yml` entry like:

[`languages.yml.example`](./languages.yml.example)

The `language_id` must be assigned by Linguist maintainers. Until that upstream
PR lands, repositories can opt in locally with:

```gitattributes
*.tya linguist-language=Tya
```

The canonical token taxonomy is documented in
[`../TOKENS.md`](../TOKENS.md).
