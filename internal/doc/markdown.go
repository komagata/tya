package doc

import (
	"fmt"
	"regexp"
	"strings"
)

// Block is the intermediate parse representation of a Markdown blob.
// v0.51 supports headings, paragraphs, code fences, and ordered /
// unordered list items at a single indent level.
type Block struct {
	Kind  string // "heading" | "paragraph" | "code" | "ul" | "ol"
	Level int    // heading level (1..6)
	Lang  string // code fence info string
	Lines []string
}

var (
	headingRE = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	bulletRE  = regexp.MustCompile(`^(\s*)- (.+)$`)
	orderedRE = regexp.MustCompile(`^(\s*)\d+\. (.+)$`)
	fenceRE   = regexp.MustCompile("^```([A-Za-z0-9_+-]*)\\s*$")
	codeRE    = regexp.MustCompile("`([^`]+)`")
	linkRE    = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	boldRE    = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	italicRE  = regexp.MustCompile(`\*([^*]+)\*`)
)

// ParseMarkdown turns a Markdown source string into a flat list of
// Block values. The renderer in RenderHTML / RenderText consumes
// this representation.
func ParseMarkdown(src string) []Block {
	if src == "" {
		return nil
	}
	lines := strings.Split(src, "\n")
	var blocks []Block
	var paragraph []string
	inCode := false
	codeLang := ""
	var codeLines []string

	flushParagraph := func() {
		if len(paragraph) == 0 {
			return
		}
		blocks = append(blocks, Block{Kind: "paragraph", Lines: paragraph})
		paragraph = nil
	}

	for _, line := range lines {
		if m := fenceRE.FindStringSubmatch(line); m != nil {
			if inCode {
				blocks = append(blocks, Block{Kind: "code", Lang: codeLang, Lines: codeLines})
				inCode = false
				codeLang = ""
				codeLines = nil
			} else {
				flushParagraph()
				inCode = true
				codeLang = m[1]
			}
			continue
		}
		if inCode {
			codeLines = append(codeLines, line)
			continue
		}
		if strings.TrimSpace(line) == "" {
			flushParagraph()
			continue
		}
		if m := headingRE.FindStringSubmatch(line); m != nil {
			flushParagraph()
			blocks = append(blocks, Block{
				Kind:  "heading",
				Level: len(m[1]),
				Lines: []string{m[2]},
			})
			continue
		}
		if m := bulletRE.FindStringSubmatch(line); m != nil {
			flushParagraph()
			blocks = appendListItem(blocks, "ul", m[2])
			continue
		}
		if m := orderedRE.FindStringSubmatch(line); m != nil {
			flushParagraph()
			blocks = appendListItem(blocks, "ol", m[2])
			continue
		}
		paragraph = append(paragraph, line)
	}
	flushParagraph()
	if inCode {
		blocks = append(blocks, Block{Kind: "code", Lang: codeLang, Lines: codeLines})
	}
	return blocks
}

func appendListItem(blocks []Block, kind, item string) []Block {
	if len(blocks) > 0 && blocks[len(blocks)-1].Kind == kind {
		blocks[len(blocks)-1].Lines = append(blocks[len(blocks)-1].Lines, item)
		return blocks
	}
	return append(blocks, Block{Kind: kind, Lines: []string{item}})
}

// RenderHTML converts a parsed Markdown block stream into an HTML
// fragment string suitable for embedding inside a static page.
func RenderHTML(blocks []Block) string {
	var b strings.Builder
	for _, blk := range blocks {
		switch blk.Kind {
		case "heading":
			text := inlineToHTML(blk.Lines[0])
			fmt.Fprintf(&b, "<h%d>%s</h%d>\n", blk.Level, text, blk.Level)
		case "paragraph":
			joined := strings.Join(blk.Lines, " ")
			fmt.Fprintf(&b, "<p>%s</p>\n", inlineToHTML(joined))
		case "code":
			lang := ""
			if blk.Lang != "" {
				lang = fmt.Sprintf(` class="language-%s"`, escapeHTML(blk.Lang))
			}
			fmt.Fprintf(&b, "<pre><code%s>%s</code></pre>\n", lang, escapeHTML(strings.Join(blk.Lines, "\n")))
		case "ul", "ol":
			fmt.Fprintf(&b, "<%s>\n", blk.Kind)
			for _, item := range blk.Lines {
				fmt.Fprintf(&b, "<li>%s</li>\n", inlineToHTML(item))
			}
			fmt.Fprintf(&b, "</%s>\n", blk.Kind)
		}
	}
	return b.String()
}

// RenderText converts a parsed Markdown block stream into a plain
// text representation used by the default `tya doc` output. The
// styling matches the rough format used by godoc-like text dumps.
func RenderText(blocks []Block) string {
	var b strings.Builder
	for i, blk := range blocks {
		if i > 0 {
			b.WriteString("\n")
		}
		switch blk.Kind {
		case "heading":
			fmt.Fprintf(&b, "=== %s ===\n", inlineToText(blk.Lines[0]))
		case "paragraph":
			fmt.Fprintf(&b, "%s\n", inlineToText(strings.Join(blk.Lines, " ")))
		case "code":
			for _, line := range blk.Lines {
				fmt.Fprintf(&b, "    %s\n", line)
			}
		case "ul":
			for _, item := range blk.Lines {
				fmt.Fprintf(&b, "- %s\n", inlineToText(item))
			}
		case "ol":
			for idx, item := range blk.Lines {
				fmt.Fprintf(&b, "%d. %s\n", idx+1, inlineToText(item))
			}
		}
	}
	return b.String()
}

// inlineToHTML applies the v0.51 subset of inline Markdown:
// backticked code, links, **bold**, *italic*. It HTML-escapes the
// raw input first so user content can never inject markup.
func inlineToHTML(value string) string {
	escaped := escapeHTML(value)
	escaped = codeRE.ReplaceAllString(escaped, "<code>$1</code>")
	escaped = linkRE.ReplaceAllStringFunc(escaped, func(match string) string {
		m := linkRE.FindStringSubmatch(match)
		if m == nil {
			return match
		}
		return fmt.Sprintf(`<a href="%s">%s</a>`, escapeHTML(m[2]), m[1])
	})
	escaped = boldRE.ReplaceAllString(escaped, "<strong>$1</strong>")
	escaped = italicRE.ReplaceAllString(escaped, "<em>$1</em>")
	return escaped
}

// inlineToText strips inline Markdown markers for the plain-text
// output path.
func inlineToText(value string) string {
	v := codeRE.ReplaceAllString(value, "$1")
	v = linkRE.ReplaceAllString(v, "$1")
	v = boldRE.ReplaceAllString(v, "$1")
	v = italicRE.ReplaceAllString(v, "$1")
	return v
}

var htmlEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&#39;",
)

func escapeHTML(value string) string {
	return htmlEscaper.Replace(value)
}
