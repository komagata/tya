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
  { source: path.join(docsDir, "VERSIONS.md"), output: path.join(docsDir, "versions.html"), title: "Versions" },
  { source: path.join(root, "ROADMAP.md"), output: path.join(docsDir, "roadmap.html"), title: "Roadmap" },
  { source: path.join(docsDir, "v0.2", "SPEC.md"), output: path.join(docsDir, "v0.2", "spec.html"), title: "Spec v0.2 Draft", versioned: true },
  { source: path.join(docsDir, "v0.1.0", "SPEC.md"), output: path.join(docsDir, "v0.1.0", "spec.html"), title: "Spec v0.1.0", versioned: true },
  { source: path.join(docsDir, "v0.1.0", "API.md"), output: path.join(docsDir, "v0.1.0", "api.html"), title: "API v0.1.0", versioned: true },
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

function rewriteHref(href, page) {
  const clean = href.replace(/^docs\//, "");
  if (page && page.versioned) {
    const versionedMapping = {
      "SPEC.md": "spec.html",
      "API.md": "api.html",
      "docs/SPEC.md": "spec.html",
      "docs/API.md": "api.html",
      "NAMING.md": "../naming.html",
      "docs/NAMING.md": "../naming.html",
    };
    if (versionedMapping[href] || versionedMapping[clean]) {
      return versionedMapping[href] || versionedMapping[clean];
    }
  }
  const mapping = {
    "GUIDE.md": "guide.html",
    "SPEC.md": "spec.html",
    "API.md": "api.html",
    "NAMING.md": "naming.html",
    "VERSIONS.md": "versions.html",
    "ROADMAP.md": "roadmap.html",
    "../ROADMAP.md": "roadmap.html",
  };
  return mapping[href] || mapping[clean] || href;
}

function inlineMarkdown(value, page) {
  const escaped = escapeHtml(value);
  return escaped
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_, label, href) => {
      return `<a href="${escapeHtml(rewriteHref(href, page))}">${label}</a>`;
    });
}

function closeList(state, html) {
  if (state.listType) {
    html.push(`</${state.listType}>`);
    state.listType = null;
  }
}

function markdownToHtml(markdown, page) {
  const html = [];
  const lines = markdown.split(/\r?\n/);
  const state = { listType: null, inCode: false, codeLang: "", code: [], paragraph: [] };

  function flushParagraph() {
    if (state.paragraph.length === 0) {
      return;
    }
    html.push(`<p>${inlineMarkdown(state.paragraph.join(" "), page)}</p>`);
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
      html.push(`<h${level} id="${escapeHtml(slugify(text))}">${inlineMarkdown(text, page)}</h${level}>`);
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
      html.push(`<li>${inlineMarkdown(bullet[1], page)}</li>`);
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
      html.push(`<li>${inlineMarkdown(ordered[1], page)}</li>`);
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
  const prefix = page.versioned ? "../" : "";
  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>${escapeHtml(page.title)} - Tya</title>
  <link rel="stylesheet" href="${prefix}document.css">
</head>
<body>
  <main class="doc-shell">
    <nav class="doc-nav" aria-label="Documentation">
      <a class="brand" href="${prefix}index.html">
        <img src="${prefix}assets/tya-logo.png" alt="Tya">
        <span>Tya</span>
      </a>
      <div class="links">
        <a href="${prefix}guide.html">Guide</a>
        <a href="${prefix}spec.html">Spec</a>
        <a href="${prefix}api.html">API</a>
        <a href="${prefix}naming.html">Naming</a>
        <a href="${prefix}versions.html">Versions</a>
        <a href="${prefix}roadmap.html">Roadmap</a>
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
  const body = markdownToHtml(markdown, page);
  fs.mkdirSync(path.dirname(page.output), { recursive: true });
  fs.writeFileSync(page.output, renderPage(page, body));
}
