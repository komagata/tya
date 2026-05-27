---
status: completed
goal_ready: false
---

# Feature: XML Stdlib Library

## Goal

Add a standard `xml` library that can parse, inspect, and emit practical XML
documents, with first-version support strong enough to parse common JUnit XML
test-result output for tools such as a flaky-test history collector.

## Context

Tya already has JSON, TOML, CSV, Markdown, and template support, but no XML
library. A useful flaky-test tool for GitHub Actions can be language-agnostic if
it ingests JUnit XML, because many test runners can emit JUnit or xUnit-style
XML.

The first XML library should be a small DOM-style API. It should cover common
machine-generated XML and JUnit output, while explicitly avoiding high-risk and
large-scope XML features such as DTDs, external entities, XPath, XSLT, and
streaming SAX parsing.

The public node values should be class instances, not dictionaries, matching
the class-style direction for stdlib-owned domain values.

## Behavior

- Add a public `xml` stdlib package.
- Import shape:

  ```tya
  import xml

  doc = xml.Xml.parse(text)
  root = doc.root()

  for test in root.find_all_recursive("testcase")
    name = test.attr("classname", "") + "." + test.attr("name", "")
    status = "passed"
    if test.find("failure") != nil
      status = "failed"
    println name + " " + status
  ```

- Public classes:
  - `xml.Xml`
  - `xml.Document`
  - `xml.Element`
  - `xml.Text`
  - `xml.Comment`
  - `xml.CData`
- `Xml.parse(text)` returns a `Document` instance.
- `Xml.dump(document)` returns XML text.
- Parsed node values are class instances.
- Invalid XML raises clear `xml.parse` errors with line and column when
  practical.
- External entities and DTDs are rejected.

## Document

- `doc.root()` returns the root `Element`.
- `doc.children()` returns top-level nodes.
- `doc.version()` returns the XML declaration version when present, otherwise
  `"1.0"`.
- `doc.encoding()` returns the XML declaration encoding when present, otherwise
  `nil`.
- `doc.to_s()` returns `Xml.dump(doc)`.

## Element

- `element.name` exposes the element name as written, including namespace prefix
  when present, such as `"testsuite"` or `"x:node"`.
- `element.attrs` exposes a dictionary of attribute names to decoded string
  values.
- `element.children` exposes child nodes in document order.
- `element.attr(name)` returns an attribute value or `nil`.
- `element.attr(name, default)` returns an attribute value or `default`.
- `element.has_attr?(name)` returns whether an attribute exists.
- `element.text()` returns concatenated descendant text and CDATA content.
- `element.child_elements()` returns immediate child elements.
- `element.find(name)` returns the first immediate child element with `name`, or
  `nil`.
- `element.find_all(name)` returns immediate child elements with `name`.
- `element.find_recursive(name)` returns the first descendant element with
  `name`, or `nil`.
- `element.find_all_recursive(name)` returns descendant elements with `name` in
  document order.
- `element.to_s()` returns XML text for this element subtree.

## Text, Comment, and CDATA

- `Text.new(text)` creates a text node.
- `text_node.text` exposes decoded text content.
- `Comment.new(text)` creates a comment node.
- `comment.text` exposes comment content.
- `CData.new(text)` creates a CDATA node.
- `cdata.text` exposes CDATA content.
- Parser preserves comments and CDATA as nodes.
- `Element.text()` includes `Text` and `CData` content but excludes comments.

## Parsing Requirements

- Support XML declaration:
  - `<?xml version="1.0"?>`
  - `<?xml version="1.0" encoding="UTF-8"?>`
- Support start/end tags.
- Support self-closing tags.
- Support attributes with single or double quotes.
- Decode predefined XML entities:
  - `&amp;`
  - `&lt;`
  - `&gt;`
  - `&quot;`
  - `&apos;`
- Decode numeric character references:
  - decimal, such as `&#10;`
  - hex, such as `&#x0a;`
- Support comments: `<!-- comment -->`.
- Support CDATA: `<![CDATA[...]]>`.
- Preserve element and attribute names with namespace prefixes as raw names.
- Ignore processing instructions other than XML declaration, or preserve them
  only if the implementation chooses to add an internal representation. They are
  not part of the first public API.
- Reject DTD declarations and external entities.
- Reject malformed nesting, duplicate attributes on the same element,
  unterminated comments, unterminated CDATA, unterminated tags, and invalid
  entity references.

## JUnit XML Support

The first version must parse common JUnit/xUnit XML shapes:

```xml
<testsuite name="pkg" tests="2" failures="1" errors="0" skipped="0">
  <testcase classname="pkg.Test" name="passes" time="0.01"/>
  <testcase classname="pkg.Test" name="fails" time="0.02">
    <failure message="expected true"><![CDATA[stack trace]]></failure>
    <system-out>stdout text</system-out>
    <system-err>stderr text</system-err>
  </testcase>
</testsuite>
```

Also support a top-level `<testsuites>` wrapper containing one or more
`<testsuite>` children.

Required JUnit-facing behavior:

- `find_all_recursive("testcase")` finds test cases under either `<testsuite>`
  or `<testsuites>`.
- Attribute parsing supports common JUnit attributes:
  - `name`
  - `classname`
  - `file`
  - `line`
  - `time`
  - `tests`
  - `failures`
  - `errors`
  - `skipped`
- Child elements such as `failure`, `error`, `skipped`, `system-out`, and
  `system-err` are parsed as normal elements.
- Failure/error body text may be ordinary text or CDATA.
- XML emitted by common Go, Ruby, Java, JavaScript, Python, and pytest JUnit
  exporters should parse when it does not require DTD, external entities, or
  namespace resolution.

## Dumping

- `Xml.dump(document)` emits valid XML for supported node types.
- `Xml.dump(element)` emits valid XML for an element subtree.
- Text and attribute values are escaped.
- Empty elements may be emitted as self-closing tags.
- Dumping preserves element order and attribute values. Attribute ordering may
  be deterministic if the runtime exposes deterministic dictionary order; tests
  should not depend on arbitrary dictionary ordering unless implemented.
- `Xml.escape_text(text)` returns XML text-escaped content.
- `Xml.escape_attr(text)` returns XML attribute-escaped content.

## Scope

- `lib/xml/Xml.tya`
- `lib/xml/Document.tya`
- `lib/xml/Element.tya`
- `lib/xml/Text.tya`
- `lib/xml/Comment.tya`
- `lib/xml/CData.tya`
- `tests/stdlib_xml_test.tya`
- JUnit fixture tests under `tests/testdata/xml/` or inline test fixtures.
- `docs/STDLIB.md`
- Next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`
- Optional example under `examples/xml/`

## Out of Scope

- DTD support.
- External entities.
- Entity expansion beyond predefined and numeric references.
- XML schema validation.
- XPath.
- XSLT.
- Streaming SAX or pull parser APIs.
- Namespace URI resolution.
- XML canonicalization.
- Full HTML parsing.
- A dedicated JUnit parser API. This PRD only requires XML support sufficient
  for JUnit consumers to build on top.

## Acceptance Criteria

- `import xml` exposes `Xml`, `Document`, `Element`, `Text`, `Comment`,
  and `CData`.
- `Xml.parse(text)` returns a `Document` instance with a class-instance root
  element.
- Element names, attributes, children, text, comments, and CDATA parse
  correctly.
- Attribute access, immediate find, recursive find, and text concatenation work.
- XML declarations parse and expose version/encoding.
- Predefined entities and numeric character references decode correctly.
- Dumping escapes text and attributes correctly.
- DTD and external entity inputs are rejected.
- Malformed XML raises clear `xml.parse` errors.
- A fixture with top-level `<testsuite>` JUnit XML parses and exposes all
  `<testcase>` elements, attributes, failure text, stdout, and stderr.
- A fixture with top-level `<testsuites>` JUnit XML parses and exposes nested
  `<testcase>` elements.
- JUnit failure/error bodies using CDATA parse correctly.
- Existing JSON/TOML/Markdown stdlib tests remain green.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run 'Test.*Xml|Test.*Json|Test.*Toml|Test.*Markdown|TestSelfhostV01Scripts' -count=1
go test ./... -count=1
```

Manual smoke after implementation:

```sh
tya run examples/xml/junit_summary.tya
```

## Dependencies

- Uses existing string, array, and dictionary primitives.
- Should align with the stdlib class-style PRD.
- The future flaky-test tool can depend on this library for JUnit ingestion.

## Open Questions

None.
