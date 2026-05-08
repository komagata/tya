#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

const root = path.resolve(__dirname, "..");
const docsDir = path.join(root, "docs");

const pages = [
  { source: path.join(docsDir, "GUIDE.md"), output: path.join(docsDir, "guide.html"), title: "Guide" },
  { source: path.join(docsDir, "SPEC.md"), output: path.join(docsDir, "spec.html"), title: "Spec" },
  { source: path.join(docsDir, "API.md"), output: path.join(docsDir, "api.html"), title: "API" },
  { source: path.join(docsDir, "STDLIB.md"), output: path.join(docsDir, "stdlib.html"), title: "Stdlib" },
  { source: path.join(docsDir, "NAMING.md"), output: path.join(docsDir, "naming.html"), title: "Naming" },
  { source: path.join(docsDir, "VERSIONS.md"), output: path.join(docsDir, "versions.html"), title: "Versions" },
  { source: path.join(root, "ROADMAP.md"), output: path.join(docsDir, "roadmap.html"), title: "Roadmap" },
  { source: path.join(docsDir, "v0.22", "SPEC.md"), output: path.join(docsDir, "v0.22", "spec.html"), title: "Spec v0.22", versioned: true },
  { source: path.join(docsDir, "v0.21", "SPEC.md"), output: path.join(docsDir, "v0.21", "spec.html"), title: "Spec v0.21", versioned: true },
  { source: path.join(docsDir, "v0.20", "SPEC.md"), output: path.join(docsDir, "v0.20", "spec.html"), title: "Spec v0.20", versioned: true },
  { source: path.join(docsDir, "v0.19", "SPEC.md"), output: path.join(docsDir, "v0.19", "spec.html"), title: "Spec v0.19", versioned: true },
  { source: path.join(docsDir, "v0.18", "SPEC.md"), output: path.join(docsDir, "v0.18", "spec.html"), title: "Spec v0.18", versioned: true },
  { source: path.join(docsDir, "v0.17", "SPEC.md"), output: path.join(docsDir, "v0.17", "spec.html"), title: "Spec v0.17", versioned: true },
  { source: path.join(docsDir, "v0.16", "SPEC.md"), output: path.join(docsDir, "v0.16", "spec.html"), title: "Spec v0.16", versioned: true },
  { source: path.join(docsDir, "v0.15", "SPEC.md"), output: path.join(docsDir, "v0.15", "spec.html"), title: "Spec v0.15", versioned: true },
  { source: path.join(docsDir, "v0.14", "SPEC.md"), output: path.join(docsDir, "v0.14", "spec.html"), title: "Spec v0.14", versioned: true },
  { source: path.join(docsDir, "v0.13", "SPEC.md"), output: path.join(docsDir, "v0.13", "spec.html"), title: "Spec v0.13", versioned: true },
  { source: path.join(docsDir, "v0.12", "SPEC.md"), output: path.join(docsDir, "v0.12", "spec.html"), title: "Spec v0.12", versioned: true },
  { source: path.join(docsDir, "v0.11", "SPEC.md"), output: path.join(docsDir, "v0.11", "spec.html"), title: "Spec v0.11", versioned: true },
  { source: path.join(docsDir, "v0.10", "SPEC.md"), output: path.join(docsDir, "v0.10", "spec.html"), title: "Spec v0.10", versioned: true },
  { source: path.join(docsDir, "v0.9", "SPEC.md"), output: path.join(docsDir, "v0.9", "spec.html"), title: "Spec v0.9", versioned: true },
  { source: path.join(docsDir, "v0.8", "SPEC.md"), output: path.join(docsDir, "v0.8", "spec.html"), title: "Spec v0.8", versioned: true },
  { source: path.join(docsDir, "v0.7", "SPEC.md"), output: path.join(docsDir, "v0.7", "spec.html"), title: "Spec v0.7", versioned: true },
  { source: path.join(docsDir, "v0.6", "SPEC.md"), output: path.join(docsDir, "v0.6", "spec.html"), title: "Spec v0.6", versioned: true },
  { source: path.join(docsDir, "v0.5", "SPEC.md"), output: path.join(docsDir, "v0.5", "spec.html"), title: "Spec v0.5", versioned: true },
  { source: path.join(docsDir, "v0.4", "SPEC.md"), output: path.join(docsDir, "v0.4", "spec.html"), title: "Spec v0.4", versioned: true },
  { source: path.join(docsDir, "v0.3", "SPEC.md"), output: path.join(docsDir, "v0.3", "spec.html"), title: "Spec v0.3", versioned: true },
  { source: path.join(docsDir, "v0.3", "STDLIB.md"), output: path.join(docsDir, "v0.3", "stdlib.html"), title: "Stdlib v0.3", versioned: true },
  { source: path.join(docsDir, "v0.2.0", "SPEC.md"), output: path.join(docsDir, "v0.2.0", "spec.html"), title: "Spec v0.2.0", versioned: true },
  { source: path.join(docsDir, "v0.2.0", "API.md"), output: path.join(docsDir, "v0.2.0", "api.html"), title: "API v0.2.0", versioned: true },
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
  const versionedSpec = clean.match(/^(v\d+\.\d+(?:\.\d+)?)\/SPEC\.md$/);
  if (versionedSpec) {
    return `${versionedSpec[1]}/spec.html`;
  }
  const versionedApi = clean.match(/^(v\d+\.\d+(?:\.\d+)?)\/API\.md$/);
  if (versionedApi) {
    return `${versionedApi[1]}/api.html`;
  }
  const versionedStdlib = clean.match(/^(v\d+\.\d+(?:\.\d+)?)\/STDLIB\.md$/);
  if (versionedStdlib) {
    return `${versionedStdlib[1]}/stdlib.html`;
  }
  if (page && page.versioned) {
    const versionedMapping = {
      "SPEC.md": "spec.html",
      "API.md": "api.html",
      "STDLIB.md": "stdlib.html",
      "docs/SPEC.md": "spec.html",
      "docs/API.md": "api.html",
      "docs/STDLIB.md": "stdlib.html",
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
    "STDLIB.md": "stdlib.html",
    "NAMING.md": "naming.html",
    "VERSIONS.md": "versions.html",
    "v0.4.md": "v0.4/spec.html",
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

function closeLists(state, html, targetIndent = -1) {
  while (state.lists.length > 0 && state.lists[state.lists.length - 1].indent > targetIndent) {
    const list = state.lists.pop();
    html.push(`</${list.type}>`);
  }
}

function ensureList(state, html, type, indent) {
  closeLists(state, html, indent);
  const current = state.lists[state.lists.length - 1];
  if (current && current.indent === indent && current.type !== type) {
    state.lists.pop();
    html.push(`</${current.type}>`);
  }
  const next = state.lists[state.lists.length - 1];
  if (!next || next.indent < indent || next.type !== type) {
    html.push(`<${type}>`);
    state.lists.push({ type, indent });
  }
}

function markdownToHtml(markdown, page) {
  const html = [];
  const lines = markdown.split(/\r?\n/);
  const state = { lists: [], inCode: false, codeLang: "", code: [], paragraph: [] };

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
        closeLists(state, html);
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
      closeLists(state, html);
      continue;
    }

    const heading = line.match(/^(#{1,6})\s+(.+)$/);
    if (heading) {
      flushParagraph();
      closeLists(state, html);
      const level = heading[1].length;
      const text = heading[2];
      html.push(`<h${level} id="${escapeHtml(slugify(text))}">${inlineMarkdown(text, page)}</h${level}>`);
      continue;
    }

    const bullet = line.match(/^(\s*)- (.+)$/);
    if (bullet) {
      flushParagraph();
      ensureList(state, html, "ul", bullet[1].length);
      html.push(`<li>${inlineMarkdown(bullet[2], page)}</li>`);
      continue;
    }

    const ordered = line.match(/^(\s*)1\. (.+)$/);
    if (ordered) {
      flushParagraph();
      ensureList(state, html, "ol", ordered[1].length);
      html.push(`<li>${inlineMarkdown(ordered[2], page)}</li>`);
      continue;
    }

    closeLists(state, html);
    state.paragraph.push(line);
  }

  flushParagraph();
  if (state.inCode) {
    html.push(`<pre><code${state.codeLang ? ` class="language-${escapeHtml(state.codeLang)}"` : ""}>${escapeHtml(state.code.join("\n"))}</code></pre>`);
  }
  closeLists(state, html);
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
        <a href="${prefix}stdlib.html">Stdlib</a>
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
