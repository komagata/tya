package doc

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Site bundles the input items and the output root for HTML
// generation. Callers populate Items and Title, then call Generate.
type Site struct {
	Title string
	Items []DocItem
}

// Generate writes the static HTML site rooted at outDir:
//
//	<outDir>/
//	  index.html             -- listing of all bindings
//	  items/<kind>_<name>.html
//	  style.css              -- copy of defaultCSS
//
// Existing files are overwritten. Existing directories are reused.
// Duplicate (kind, name) pairs across files cause a warning written
// to warnOut and a "last-write-wins" outcome for the items page.
func (s *Site) Generate(outDir string, warnOut io.Writer) error {
	if err := os.MkdirAll(filepath.Join(outDir, "items"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "style.css"), []byte(defaultCSS), 0o644); err != nil {
		return err
	}
	seen := map[string]string{}
	for _, item := range s.Items {
		fname := itemFileName(item)
		if prev, ok := seen[fname]; ok && warnOut != nil {
			fmt.Fprintf(warnOut, "warning: duplicate doc item %s/%s (was %s, now %s)\n",
				item.Kind, item.Name, prev, item.FilePath)
		}
		seen[fname] = item.FilePath
		body := s.renderItemPage(item)
		dst := filepath.Join(outDir, "items", fname)
		if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
			return err
		}
	}
	indexBody := s.renderIndex()
	return os.WriteFile(filepath.Join(outDir, "index.html"), []byte(indexBody), 0o644)
}

func (s *Site) renderIndex() string {
	title := s.Title
	if title == "" {
		title = "API"
	}
	var b strings.Builder
	fmt.Fprintf(&b, pageHead, escapeHTML(title), "style.css")
	fmt.Fprintf(&b, "<h1>%s</h1>\n", escapeHTML(title))
	if len(s.Items) == 0 {
		b.WriteString("<p>No documented bindings.</p>\n")
	} else {
		byKind := map[string][]DocItem{}
		for _, item := range s.Items {
			byKind[item.Kind] = append(byKind[item.Kind], item)
		}
		kinds := make([]string, 0, len(byKind))
		for k := range byKind {
			kinds = append(kinds, k)
		}
		sort.Slice(kinds, func(i, j int) bool {
			return kindOrder(kinds[i]) < kindOrder(kinds[j])
		})
		for _, kind := range kinds {
			fmt.Fprintf(&b, "<h2>%s</h2>\n<ul>\n", escapeHTML(kindTitle(kind)))
			for _, item := range byKind[kind] {
				href := "items/" + itemFileName(item)
				fmt.Fprintf(&b, "<li><a href=\"%s\"><code>%s</code></a> &mdash; <code>%s</code></li>\n",
					escapeHTML(href), escapeHTML(item.Signature), escapeHTML(item.FilePath))
			}
			b.WriteString("</ul>\n")
		}
	}
	b.WriteString(pageFoot)
	return b.String()
}

func (s *Site) renderItemPage(item DocItem) string {
	var b strings.Builder
	title := fmt.Sprintf("%s %s", item.Kind, item.Name)
	fmt.Fprintf(&b, pageHead, escapeHTML(title), "../style.css")
	b.WriteString(`<p><a href="../index.html">&larr; back to index</a></p>` + "\n")
	fmt.Fprintf(&b, "<h1>%s <code>%s</code></h1>\n", escapeHTML(item.Kind), escapeHTML(item.Name))
	fmt.Fprintf(&b, "<pre><code>%s</code></pre>\n", escapeHTML(item.Signature))
	fmt.Fprintf(&b, "<p><small>%s:%d</small></p>\n", escapeHTML(item.FilePath), item.Line)
	if strings.TrimSpace(item.RawDoc) != "" {
		b.WriteString(RenderHTML(ParseMarkdown(item.RawDoc)))
	} else {
		b.WriteString(`<p><em>(no doc comment)</em></p>` + "\n")
	}
	b.WriteString(pageFoot)
	return b.String()
}

func kindTitle(kind string) string {
	switch kind {
	case "module":
		return "Modules"
	case "class":
		return "Classes"
	case "interface":
		return "Interfaces"
	case "function":
		return "Functions"
	}
	return kind
}

// itemFileName returns "<kind>_<sanitized-name>.html". The kind
// prefix prevents collisions between e.g. a `Foo` class and a `foo`
// function in the same project.
func itemFileName(item DocItem) string {
	return item.Kind + "_" + sanitizeFileName(item.Name) + ".html"
}

func sanitizeFileName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}

const pageHead = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>%s</title>
<link rel="stylesheet" href="%s">
</head>
<body>
<div class="doc-shell">
<div class="doc-content">
`

const pageFoot = `</div>
</div>
</body>
</html>
`

// defaultCSS is a copy of docs/document.css. Keep these two in
// sync manually; the SPEC documents the requirement.
const defaultCSS = `:root {
  color-scheme: light;
  --ink: #17201b;
  --muted: #5f6b64;
  --line: #dfe8e1;
  --panel: #f7faf7;
  --code: #15201b;
  --accent: #2f7d5b;
  --paper: #fffdf7;
}

* {
  box-sizing: border-box;
}

body {
  margin: 0;
  background: var(--paper);
  color: var(--ink);
  font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  line-height: 1.65;
}

a {
  color: var(--accent);
}

.doc-shell {
  width: min(920px, calc(100% - 32px));
  margin: 0 auto;
  padding: 18px 0 56px;
}

.doc-content {
  padding-top: 24px;
}

.doc-content h1 {
  margin: 0 0 16px;
  font-size: clamp(2.2rem, 7vw, 4rem);
  line-height: 0.98;
}

.doc-content h2 {
  margin: 42px 0 12px;
  padding-top: 10px;
  border-top: 1px solid var(--line);
  font-size: 1.55rem;
}

.doc-content h3 {
  margin: 28px 0 8px;
  font-size: 1.2rem;
}

.doc-content p,
.doc-content li {
  font-size: 1rem;
}

.doc-content p {
  margin: 12px 0;
}

.doc-content ul,
.doc-content ol {
  margin: 12px 0 18px;
  padding-left: 24px;
}

.doc-content code {
  border-radius: 5px;
  background: #eef4ee;
  padding: 0.12em 0.34em;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.92em;
}

.doc-content pre {
  overflow-x: auto;
  margin: 18px 0;
  border: 1px solid #26392f;
  border-radius: 8px;
  background: var(--code);
  padding: 16px;
}

.doc-content pre code {
  display: block;
  min-width: max-content;
  background: transparent;
  color: #eaf2ed;
  padding: 0;
  font-size: 0.92rem;
  line-height: 1.55;
}
`
