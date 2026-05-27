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
//	  index.html             -- listing of documented pages
//	  items/<kind>_<name>.html
//	  style.css              -- copy of defaultCSS
//
// Existing files are overwritten. Existing directories are reused.
// Stale item pages from previous runs are removed before writing the
// new item pages.
func (s *Site) Generate(outDir string, warnOut io.Writer) error {
	itemsDir := filepath.Join(outDir, "items")
	if err := os.RemoveAll(itemsDir); err != nil {
		return err
	}
	if err := os.MkdirAll(itemsDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "style.css"), []byte(defaultCSS), 0o644); err != nil {
		return err
	}
	seen := map[string]string{}
	pages := s.pages()
	for _, page := range pages {
		fname := pageFileName(page.Item)
		if prev, ok := seen[fname]; ok && warnOut != nil {
			fmt.Fprintf(warnOut, "warning: duplicate doc page %s/%s (was %s, now %s)\n",
				page.Item.Kind, page.Item.Name, prev, page.Item.FilePath)
		}
		seen[fname] = page.Item.FilePath
		body := s.renderItemPage(page, pages)
		dst := filepath.Join(itemsDir, fname)
		if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
			return err
		}
	}
	indexBody := s.renderIndex(pages)
	return os.WriteFile(filepath.Join(outDir, "index.html"), []byte(indexBody), 0o644)
}

type docPage struct {
	Item    DocItem
	Members []DocItem
}

func (s *Site) pages() []docPage {
	containers := map[string]DocItem{}
	for _, item := range s.Items {
		if isContainerKind(item.Kind) {
			containers[pageKey(item)] = item
		}
	}
	members := map[string][]DocItem{}
	var pages []docPage
	for _, item := range s.Items {
		if isContainerKind(item.Kind) {
			continue
		}
		if owner, ok := memberOwner(item); ok {
			key := item.FilePath + "\x00" + owner
			if _, exists := containers[key]; exists {
				members[key] = append(members[key], item)
				continue
			}
		}
		pages = append(pages, docPage{Item: item})
	}
	for _, item := range s.Items {
		if !isContainerKind(item.Kind) {
			continue
		}
		key := pageKey(item)
		pages = append(pages, docPage{
			Item:    item,
			Members: members[key],
		})
	}
	sort.SliceStable(pages, func(i, j int) bool {
		if pages[i].Item.Kind != pages[j].Item.Kind {
			return kindOrder(pages[i].Item.Kind) < kindOrder(pages[j].Item.Kind)
		}
		if pages[i].Item.Name != pages[j].Item.Name {
			return pages[i].Item.Name < pages[j].Item.Name
		}
		return pages[i].Item.FilePath < pages[j].Item.FilePath
	})
	for i := range pages {
		sort.SliceStable(pages[i].Members, func(a, b int) bool {
			return docMemberLess(pages[i].Members[a], pages[i].Members[b])
		})
	}
	return pages
}

func docMemberLess(left, right DocItem) bool {
	if left.Kind != right.Kind {
		return kindOrder(left.Kind) < kindOrder(right.Kind)
	}
	return memberShortName(left.Name) < memberShortName(right.Name)
}

func pageKey(item DocItem) string {
	return item.FilePath + "\x00" + item.Name
}

func isContainerKind(kind string) bool {
	return kind == "class" || kind == "interface" || kind == "module"
}

func memberOwner(item DocItem) (string, bool) {
	if item.Kind != "variable" &&
		item.Kind != "constant" &&
		item.Kind != "class constant" &&
		item.Kind != "class variable" &&
		item.Kind != "instance variable" &&
		item.Kind != "method" &&
		item.Kind != "static method" {
		return "", false
	}
	before, _, ok := strings.Cut(item.Name, ".")
	return before, ok
}

func memberShortName(name string) string {
	_, after, ok := strings.Cut(name, ".")
	if !ok {
		return name
	}
	return after
}

func (s *Site) renderIndex(pages []docPage) string {
	title := s.Title
	if title == "" {
		title = "API"
	}
	var b strings.Builder
	fmt.Fprintf(&b, pageHead, escapeHTML(title), "style.css")
	b.WriteString(renderSidebar(title, pages, ""))
	b.WriteString(`<main class="doc-content">` + "\n")
	fmt.Fprintf(&b, "<h1>%s</h1>\n", escapeHTML(title))
	if len(pages) == 0 {
		b.WriteString("<p>No documented bindings.</p>\n")
	} else {
		byKind := map[string][]docPage{}
		for _, page := range pages {
			byKind[page.Item.Kind] = append(byKind[page.Item.Kind], page)
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
			for _, page := range byKind[kind] {
				href := "items/" + pageFileName(page.Item)
				fmt.Fprintf(&b, "<li><a href=\"%s\"><code>%s</code></a> &mdash; <code>%s</code>",
					escapeHTML(href), escapeHTML(page.Item.Signature), escapeHTML(page.Item.FilePath))
				if len(page.Members) > 0 {
					fmt.Fprintf(&b, " <span class=\"member-count\">%d members</span>", len(page.Members))
				}
				b.WriteString("</li>\n")
			}
			b.WriteString("</ul>\n")
		}
	}
	b.WriteString(pageFoot)
	return b.String()
}

func (s *Site) renderItemPage(page docPage, pages []docPage) string {
	var b strings.Builder
	item := page.Item
	title := fmt.Sprintf("%s %s", item.Kind, item.Name)
	fmt.Fprintf(&b, pageHead, escapeHTML(title), "../style.css")
	b.WriteString(renderSidebar(s.Title, pages, pageFileName(item)))
	b.WriteString(`<main class="doc-content">` + "\n")
	fmt.Fprintf(&b, "<h1>%s <code>%s</code></h1>\n", escapeHTML(item.Kind), escapeHTML(item.Name))
	b.WriteString(renderDocSection(item, ""))
	if len(page.Members) > 0 {
		byKind := map[string][]DocItem{}
		for _, member := range page.Members {
			byKind[member.Kind] = append(byKind[member.Kind], member)
		}
		kinds := make([]string, 0, len(byKind))
		for kind := range byKind {
			kinds = append(kinds, kind)
		}
		sort.Slice(kinds, func(i, j int) bool {
			return kindOrder(kinds[i]) < kindOrder(kinds[j])
		})
		for _, kind := range kinds {
			fmt.Fprintf(&b, "<h2>%s</h2>\n", escapeHTML(kindTitle(kind)))
			for _, member := range byKind[kind] {
				b.WriteString(renderDocSection(member, "member"))
			}
		}
	}
	b.WriteString(pageFoot)
	return b.String()
}

func renderSidebar(title string, pages []docPage, current string) string {
	if title == "" {
		title = "API"
	}
	var b strings.Builder
	b.WriteString("<aside class=\"doc-sidebar\">\n")
	fmt.Fprintf(&b, "<div class=\"sidebar-title\"><a href=\"%sindex.html\">%s</a></div>\n", sidebarRoot(current), escapeHTML(title))
	byKind := map[string][]docPage{}
	for _, page := range pages {
		byKind[page.Item.Kind] = append(byKind[page.Item.Kind], page)
	}
	kinds := make([]string, 0, len(byKind))
	for kind := range byKind {
		kinds = append(kinds, kind)
	}
	sort.Slice(kinds, func(i, j int) bool {
		return kindOrder(kinds[i]) < kindOrder(kinds[j])
	})
	for _, kind := range kinds {
		fmt.Fprintf(&b, "<h2>%s</h2>\n<ul>\n", escapeHTML(kindTitle(kind)))
		for _, page := range byKind[kind] {
			file := pageFileName(page.Item)
			href := "items/" + file
			if current != "" {
				href = file
			}
			class := ""
			if file == current {
				class = ` class="current"`
			}
			fmt.Fprintf(&b, "<li%s><a href=\"%s\"><code>%s</code></a></li>\n", class, escapeHTML(href), escapeHTML(page.Item.Name))
		}
		b.WriteString("</ul>\n")
	}
	b.WriteString("</aside>\n")
	return b.String()
}

func sidebarRoot(current string) string {
	if current == "" {
		return ""
	}
	return "../"
}

func renderDocSection(item DocItem, extraClass string) string {
	var b strings.Builder
	id := sanitizeFileName(item.Kind + "-" + item.Name)
	classes := "doc-item"
	if extraClass != "" {
		classes += " " + extraClass
	}
	fmt.Fprintf(&b, "<section id=\"%s\" class=\"%s\">\n", escapeHTML(id), escapeHTML(classes))
	if extraClass != "" {
		fmt.Fprintf(&b, "<h3><code>%s</code></h3>\n", escapeHTML(memberShortName(item.Name)))
	}
	fmt.Fprintf(&b, "<pre><code>%s</code></pre>\n", escapeHTML(item.Signature))
	fmt.Fprintf(&b, "<p><small>%s:%d</small></p>\n", escapeHTML(item.FilePath), item.Line)
	b.WriteString(renderHTMLMetadata(item))
	if strings.TrimSpace(item.RawDoc) != "" {
		b.WriteString(RenderHTML(ParseMarkdown(item.RawDoc)))
	} else {
		b.WriteString(`<p><em>(no doc comment)</em></p>` + "\n")
	}
	if source := sourceBlock(item); source != "" {
		b.WriteString("<details class=\"source\"><summary>Source</summary>\n")
		fmt.Fprintf(&b, "<pre><code>%s</code></pre>\n", escapeHTML(source))
		b.WriteString("</details>\n")
	}
	b.WriteString("</section>\n")
	return b.String()
}

func sourceBlock(item DocItem) string {
	raw, err := os.ReadFile(item.FilePath)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	if item.Line <= 0 || item.Line > len(lines) {
		return ""
	}
	start := item.Line - 1
	declIndent := leadingSpaces(lines[start])
	for start > 0 {
		prev := lines[start-1]
		if strings.TrimSpace(prev) == "" {
			break
		}
		if leadingSpaces(prev) != declIndent || !strings.HasPrefix(strings.TrimSpace(prev), "#") {
			break
		}
		start--
	}
	end := item.Line
	for end < len(lines) {
		line := lines[end]
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && leadingSpaces(line) <= declIndent {
			break
		}
		end++
	}
	return strings.TrimRight(strings.Join(lines[start:end], "\n"), "\n")
}

func leadingSpaces(line string) int {
	n := 0
	for n < len(line) && line[n] == ' ' {
		n++
	}
	return n
}

func renderHTMLMetadata(item DocItem) string {
	if item.TypeHint == "" && len(item.Params) == 0 && len(item.Options) == 0 && item.Return == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("<dl class=\"metadata\">\n")
	if item.TypeHint != "" {
		fmt.Fprintf(&b, "<dt>Type</dt><dd><code>%s</code></dd>\n", escapeHTML(item.TypeHint))
	}
	for _, param := range item.Params {
		fmt.Fprintf(&b, "<dt>Param <code>%s</code></dt><dd><code>%s</code>", escapeHTML(param.Name), escapeHTML(param.TypeHint))
		if param.Description != "" {
			fmt.Fprintf(&b, " %s", escapeHTML(param.Description))
		}
		b.WriteString("</dd>\n")
	}
	for _, option := range item.Options {
		fmt.Fprintf(&b, "<dt>Option <code>%s.%s</code></dt><dd><code>%s</code>", escapeHTML(option.Param), escapeHTML(option.Key), escapeHTML(option.TypeHint))
		if option.Description != "" {
			fmt.Fprintf(&b, " %s", escapeHTML(option.Description))
		}
		b.WriteString("</dd>\n")
	}
	if item.Return != nil {
		fmt.Fprintf(&b, "<dt>Return</dt><dd><code>%s</code>", escapeHTML(item.Return.TypeHint))
		if item.Return.Description != "" {
			fmt.Fprintf(&b, " %s", escapeHTML(item.Return.Description))
		}
		b.WriteString("</dd>\n")
	}
	b.WriteString("</dl>\n")
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
	case "class constant":
		return "Class Constants"
	case "class variable":
		return "Class Variables"
	case "instance variable":
		return "Instance Variables"
	case "constant":
		return "Constants"
	case "variable":
		return "Variables"
	case "static method":
		return "Static Methods"
	case "method":
		return "Methods"
	}
	return kind
}

// pageFileName returns a path-qualified stable file name. The kind and
// source path prefixes prevent collisions between same-named public
// items from different packages, such as net/http.Server and
// net/socket.Server in the standard library.
func pageFileName(item DocItem) string {
	path := strings.TrimSuffix(filepath.ToSlash(item.FilePath), ".tya")
	base := item.Kind + "_" + item.Name
	if path == "stdlib" || strings.HasPrefix(path, "stdlib/") || strings.Contains(path, "/stdlib/") {
		base = item.Kind + "_" + path + "_" + item.Name
	}
	return sanitizeFileName(base) + ".html"
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
`

const pageFoot = `</main>
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
  --sidebar: #f3f7f3;
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
  width: min(1180px, calc(100% - 32px));
  margin: 0 auto;
  padding: 18px 0 56px;
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  gap: 32px;
  align-items: start;
}

.doc-sidebar {
  width: 260px;
  position: sticky;
  top: 18px;
  max-height: calc(100vh - 36px);
  overflow: auto;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--sidebar);
  padding: 16px;
}

.sidebar-title {
  margin-bottom: 16px;
  font-weight: 700;
}

.doc-sidebar h2 {
  margin: 18px 0 6px;
  color: var(--muted);
  font-size: 0.78rem;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.doc-sidebar ul {
  list-style: none;
  margin: 0;
  padding: 0;
}

.doc-sidebar li {
  margin: 2px 0;
}

.doc-sidebar a {
  display: block;
  border-radius: 5px;
  padding: 4px 6px;
  text-decoration: none;
}

.doc-sidebar li.current a,
.doc-sidebar a:hover {
  background: #e3ece5;
}

.doc-content {
  min-width: 0;
  padding-top: 8px;
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

.doc-item {
  margin: 18px 0 28px;
}

.doc-item.member {
  border-top: 1px solid var(--line);
  padding-top: 18px;
}

.source {
  margin-top: 18px;
}

.source summary {
  cursor: pointer;
  color: var(--accent);
  font-weight: 700;
}

.metadata {
  display: grid;
  grid-template-columns: max-content minmax(0, 1fr);
  gap: 6px 12px;
  margin: 16px 0;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--panel);
  padding: 12px;
}

.metadata dt {
  color: var(--muted);
  font-weight: 700;
}

.metadata dd {
  margin: 0;
}

.member-count {
  color: var(--muted);
  font-size: 0.9em;
}

@media (max-width: 760px) {
  .doc-shell {
    display: block;
  }

  .doc-sidebar {
    position: static;
    max-height: none;
    margin-bottom: 18px;
  }
}
`
