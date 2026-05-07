#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

const root = path.resolve(__dirname, "..");
const docsDir = path.join(root, "docs");

const pages = [
  { source: path.join(docsDir, "GUIDE.md"), output: path.join(docsDir, "guide.html"), title: "Guide" },
  { source: path.join(docsDir, "SPEC.md"), output: path.join(docsDir, "spec.html"), title: "Spec" },
  { source: path.join(docsDir, "API.md"), output: path.join(docsDir, "api.html"), title: "API" },
  { source: path.join(docsDir, "NAMING.md"), output: path.join(docsDir, "naming.html"), title: "Naming" },
  { source: path.join(root, "ROADMAP.md"), output: path.join(docsDir, "roadmap.html"), title: "Roadmap" },
  { source: path.join(docsDir, "ROADMAP_STRUCTURE.md"), output: path.join(docsDir, "roadmap-structure.html"), title: "Roadmap Structure" },
];

function escapeHtml(value) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function slugify(value) {
  return value
    .toLowerCase()
    .replace(/`([^`]+)`/g, "$1")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function rewriteHref(href) {
  const clean = href.replace(/^docs\//, "");
  const mapping = {
    "GUIDE.md": "guide.html",
    "SPEC.md": "spec.html",
    "API.md": "api.html",
    "NAMING.md": "naming.html",
    "ROADMAP.md": "roadmap.html",
    "../ROADMAP.md": "roadmap.html",
    "docs/ROADMAP_STRUCTURE.md": "roadmap-structure.html",
    "ROADMAP_STRUCTURE.md": "roadmap-structure.html",
  };
  return mapping[href] || mapping[clean] || href;
}

function inlineMarkdown(value) {
  const escaped = escapeHtml(value);
  return escaped
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_, label, href) => {
      return `<a href="${escapeHtml(rewriteHref(href))}">${label}</a>`;
    });
}

function closeList(state, html) {
  if (state.listType) {
    html.push(`</${state.listType}>`);
    state.listType = null;
  }
}

function markdownToHtml(markdown) {
  const html = [];
  const lines = markdown.split(/\r?\n/);
  const state = { listType: null, inCode: false, codeLang: "", code: [], paragraph: [] };

  function flushParagraph() {
    if (state.paragraph.length === 0) {
      return;
    }
    html.push(`<p>${inlineMarkdown(state.paragraph.join(" "))}</p>`);
    state.paragraph = [];
  }

  for (const line of lines) {
    const fence = line.match(/^```(\w+)?\s*$/);
    if (fence) {
      if (state.inCode) {
        html.push(`<pre><code${state.codeLang ? ` class="language-${escapeHtml(state.codeLang)}"` : ""}>${escapeHtml(state.code.join("\n"))}</code></pre>`);
        state.inCode = false;
        state.codeLang = "";
        state.code = [];
      } else {
        flushParagraph();
        closeList(state, html);
        state.inCode = true;
        state.codeLang = fence[1] || "";
      }
      continue;
    }

    if (state.inCode) {
      state.code.push(line);
      continue;
    }

    if (line.trim() === "") {
      flushParagraph();
      closeList(state, html);
      continue;
    }

    const heading = line.match(/^(#{1,6})\s+(.+)$/);
    if (heading) {
      flushParagraph();
      closeList(state, html);
      const level = heading[1].length;
      const text = heading[2];
      html.push(`<h${level} id="${escapeHtml(slugify(text))}">${inlineMarkdown(text)}</h${level}>`);
      continue;
    }

    const bullet = line.match(/^- (.+)$/);
    if (bullet) {
      flushParagraph();
      if (state.listType !== "ul") {
        closeList(state, html);
        html.push("<ul>");
        state.listType = "ul";
      }
      html.push(`<li>${inlineMarkdown(bullet[1])}</li>`);
      continue;
    }

    const ordered = line.match(/^1\. (.+)$/);
    if (ordered) {
      flushParagraph();
      if (state.listType !== "ol") {
        closeList(state, html);
        html.push("<ol>");
        state.listType = "ol";
      }
      html.push(`<li>${inlineMarkdown(ordered[1])}</li>`);
      continue;
    }

    closeList(state, html);
    state.paragraph.push(line);
  }

  flushParagraph();
  if (state.inCode) {
    html.push(`<pre><code${state.codeLang ? ` class="language-${escapeHtml(state.codeLang)}"` : ""}>${escapeHtml(state.code.join("\n"))}</code></pre>`);
  }
  closeList(state, html);
  return html.join("\n");
}

function renderPage(page, body) {
  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>${escapeHtml(page.title)} - Tya</title>
  <link rel="stylesheet" href="document.css">
</head>
<body>
  <main class="doc-shell">
    <nav class="doc-nav" aria-label="Documentation">
      <a class="brand" href="index.html">
        <img src="assets/tya-logo.png" alt="Tya">
        <span>Tya</span>
      </a>
      <div class="links">
        <a href="guide.html">Guide</a>
        <a href="spec.html">Spec</a>
        <a href="api.html">API</a>
        <a href="naming.html">Naming</a>
        <a href="roadmap.html">Roadmap</a>
        <a href="roadmap-structure.html">Structure</a>
      </div>
    </nav>
    <article class="doc-content">
${body}
    </article>
  </main>
</body>
</html>
`;
}

for (const page of pages) {
  const markdown = fs.readFileSync(page.source, "utf8");
  const body = markdownToHtml(markdown);
  fs.writeFileSync(page.output, renderPage(page, body));
}
