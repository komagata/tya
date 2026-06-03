# Feature: Character-Indexed Standard Library Parsers

## Goal
Make Tya standard library parsers and scanners use `String` character/rune indexes consistently so Unicode text is parsed correctly without mixing byte offsets and string indexing.

## Context
`String` is Unicode text and `String#[]` indexes by Unicode rune. The follow-up to the String/Bytes semantics audit is to remove parser code that compares `byte_len()` offsets with `String#[]` lookups. This currently affects `xml/Xml` and may affect TOML, JSON, URL, template, color, base64, and other scanner-like modules. The concrete bug is `komagata/tya#33`, where `xml.Xml.parse` fails on a non-ASCII XML attribute value such as `name="test_日報"`.

Relevant files include:

- `lib/xml/xml.tya`
- `lib/toml.tya`
- `lib/json.tya`
- `lib/url.tya`
- `lib/template.tya`
- `lib/color.tya`
- `lib/base64.tya`
- `tests/stdlib_xml_test.tya`
- `tests/stdlib_toml_test.tya`
- `tests/stdlib_json_test.tya`
- `tests/stdlib_url_test.tya`
- `tests/stdlib_template_test.tya`
- `docs/SPEC.md`
- `docs/STRICT_SEMANTICS.md`

## Behavior
- Parser state positions for `String` inputs are character/rune indexes unless the parser explicitly converts to `Bytes`.
- Parser state lengths for `String` inputs use `len()` or `char_len()`, not `byte_len()`.
- Character scanners may use `text[i]`, `text.slice(start, end)`, and `text.index_of(...)` only with character/rune indexes.
- Character scanners must not use `byte_len()` to bound loops that read with `String#[]`.
- Error messages may keep the wording `at byte ...` only when the parser actually tracks byte offsets. Character-indexed parsers should report character positions or use neutral wording such as `at position ...`.
- XML parsing accepts UTF-8 text in attribute values, text nodes, CDATA, comments, processing instructions, and element/attribute names where the existing parser allows names.
- XML entity decoding preserves Unicode text and supports named entities plus decimal and hexadecimal numeric character references.
- XML parsing of ASCII-only inputs remains unchanged.
- TOML, JSON, URL, template, color, and base64 behaviors remain unchanged for existing ASCII-focused tests while gaining focused Unicode scanner coverage where applicable.
- If a module truly needs byte-level scanning, it must convert to `Bytes` and use byte APIs explicitly. That exception must be local and documented in the implementation.

## Scope
- Rewrite `lib/xml/xml.tya` to use character-indexed scanner state consistently.
- Add XML regression tests for non-ASCII attribute values and text content.
- Audit scanner-like stdlib modules for loops of the form `while i < text.byte_len()` followed by `text[i]`.
- Convert audited String scanners to character-indexed loops where they parse text.
- Add focused Unicode tests for each parser module that changes.
- Update docs if any parser error wording changes from byte offsets to character positions.
- Preserve public parser APIs and return shapes.

## Out of Scope
- Changing the global `String`/`Bytes` semantics. That belongs to `feature-specs/string-bytes-semantics-audit.md`.
- Rewriting parser modules around a new shared scanner abstraction unless duplication becomes clearly harmful during implementation.
- Adding streaming XML/TOML/JSON parsing.
- Adding XML DTD, external entity, namespace, or full XML specification support.
- Adding grapheme-cluster indexing.
- Adding lossy UTF-8 decoding.
- Changing Flakewatch. Flakewatch receives a separate spec after Tya's XML parser accepts Unicode JUnit XML.

## Acceptance Criteria
- `xml.Xml.parse("<testsuite><testcase name=\"test_日報\"/></testsuite>")` returns an XML document whose testcase `name` attribute is `"test_日報"`.
- XML text nodes preserve non-ASCII text such as `"日報"` without raising string/bytes concatenation errors.
- XML entity decoding preserves surrounding non-ASCII text, including examples such as `"日報&amp;質問"` and `"test_&#26085;&#x5831;"`.
- XML parse errors no longer report misleading byte offsets when the parser tracks character/rune positions.
- Repository search finds no remaining text parser loop that combines `text.byte_len()` bounds with `text[i]` character indexing, except for code that explicitly converts the input to `Bytes`.
- Existing ASCII parser tests still pass.
- New Unicode regression tests fail on the current bug and pass after the implementation.
- `komagata/tya#33` is resolved by the Tya implementation, without requiring a Flakewatch workaround.

## Verification
```sh
go test ./... -count=1
tya test tests/stdlib_xml_test.tya
tya test tests/stdlib_toml_test.tya
tya test tests/stdlib_json_test.tya
tya test tests/stdlib_url_test.tya
tya test tests/stdlib_template_test.tya
tya test tests
go test ./tests -run TestSelfhostV01Scripts -count=1
```
