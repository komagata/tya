#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

const root = path.resolve(__dirname, "..");
const docsDir = path.join(root, "docs");
const siteDir = path.join(root, "site");

const pages = [
  { source: path.join(docsDir, "GUIDE.md"), output: path.join(siteDir, "guide.html"), title: "Guide" },
  { source: path.join(docsDir, "SPEC.md"), output: path.join(siteDir, "spec.html"), title: "Spec" },
  { source: path.join(docsDir, "API.md"), output: path.join(siteDir, "api.html"), title: "API" },
  { source: path.join(docsDir, "STDLIB.md"), output: path.join(siteDir, "stdlib.html"), title: "Stdlib" },
  { source: path.join(docsDir, "LIBRARIES.md"), output: path.join(siteDir, "libraries.html"), title: "Libraries" },
  { source: path.join(docsDir, "NAMING.md"), output: path.join(siteDir, "naming.html"), title: "Naming" },
  { source: path.join(docsDir, "VERSIONS.md"), output: path.join(siteDir, "versions.html"), title: "Versions" },
  { source: path.join(docsDir, "CANONICAL_SYNTAX.md"), output: path.join(siteDir, "CANONICAL_SYNTAX.html"), title: "Canonical Syntax" },
  { source: path.join(root, "ROADMAP.md"), output: path.join(siteDir, "roadmap.html"), title: "Roadmap" },
  { source: path.join(docsDir, "LINT.md"), output: path.join(siteDir, "lint.html"), title: "Lint" },
  { source: path.join(docsDir, "v0.62", "SPEC.md"), output: path.join(siteDir, "v0.62", "spec.html"), title: "Spec v0.62", versioned: true },
  { source: path.join(docsDir, "v0.62", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.62", "release_notes.html"), title: "Release Notes v0.62", versioned: true },
  { source: path.join(docsDir, "v0.61", "SPEC.md"), output: path.join(siteDir, "v0.61", "spec.html"), title: "Spec v0.61", versioned: true },
  { source: path.join(docsDir, "v0.61", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.61", "release_notes.html"), title: "Release Notes v0.61", versioned: true },
  { source: path.join(docsDir, "v0.60", "SPEC.md"), output: path.join(siteDir, "v0.60", "spec.html"), title: "Spec v0.60", versioned: true },
  { source: path.join(docsDir, "v0.60", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.60", "release_notes.html"), title: "Release Notes v0.60", versioned: true },
  { source: path.join(docsDir, "v0.59", "SPEC.md"), output: path.join(siteDir, "v0.59", "spec.html"), title: "Spec v0.59", versioned: true },
  { source: path.join(docsDir, "v0.59", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.59", "release_notes.html"), title: "Release Notes v0.59", versioned: true },
  { source: path.join(docsDir, "v0.58", "SPEC.md"), output: path.join(siteDir, "v0.58", "spec.html"), title: "Spec v0.58", versioned: true },
  { source: path.join(docsDir, "v0.58", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.58", "release_notes.html"), title: "Release Notes v0.58", versioned: true },
  { source: path.join(docsDir, "v0.57", "SPEC.md"), output: path.join(siteDir, "v0.57", "spec.html"), title: "Spec v0.57", versioned: true },
  { source: path.join(docsDir, "v0.57", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.57", "release_notes.html"), title: "Release Notes v0.57", versioned: true },
  { source: path.join(docsDir, "v0.56", "SPEC.md"), output: path.join(siteDir, "v0.56", "spec.html"), title: "Spec v0.56", versioned: true },
  { source: path.join(docsDir, "v0.56", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.56", "release_notes.html"), title: "Release Notes v0.56", versioned: true },
  { source: path.join(docsDir, "v0.55", "SPEC.md"), output: path.join(siteDir, "v0.55", "spec.html"), title: "Spec v0.55", versioned: true },
  { source: path.join(docsDir, "v0.55", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.55", "release_notes.html"), title: "Release Notes v0.55", versioned: true },
  { source: path.join(docsDir, "v0.54", "SPEC.md"), output: path.join(siteDir, "v0.54", "spec.html"), title: "Spec v0.54", versioned: true },
  { source: path.join(docsDir, "v0.54", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.54", "release_notes.html"), title: "Release Notes v0.54", versioned: true },
  { source: path.join(docsDir, "v0.53", "SPEC.md"), output: path.join(siteDir, "v0.53", "spec.html"), title: "Spec v0.53", versioned: true },
  { source: path.join(docsDir, "v0.53", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.53", "release_notes.html"), title: "Release Notes v0.53", versioned: true },
  { source: path.join(docsDir, "v0.52", "SPEC.md"), output: path.join(siteDir, "v0.52", "spec.html"), title: "Spec v0.52", versioned: true },
  { source: path.join(docsDir, "v0.52", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.52", "release_notes.html"), title: "Release Notes v0.52", versioned: true },
  { source: path.join(docsDir, "v0.51", "SPEC.md"), output: path.join(siteDir, "v0.51", "spec.html"), title: "Spec v0.51", versioned: true },
  { source: path.join(docsDir, "v0.51", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.51", "release_notes.html"), title: "Release Notes v0.51", versioned: true },
  { source: path.join(docsDir, "v0.50", "SPEC.md"), output: path.join(siteDir, "v0.50", "spec.html"), title: "Spec v0.50", versioned: true },
  { source: path.join(docsDir, "v0.50", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.50", "release_notes.html"), title: "Release Notes v0.50", versioned: true },
  { source: path.join(docsDir, "v0.49", "SPEC.md"), output: path.join(siteDir, "v0.49", "spec.html"), title: "Spec v0.49", versioned: true },
  { source: path.join(docsDir, "v0.49", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.49", "release_notes.html"), title: "Release Notes v0.49", versioned: true },
  { source: path.join(docsDir, "v0.48", "SPEC.md"), output: path.join(siteDir, "v0.48", "spec.html"), title: "Spec v0.48", versioned: true },
  { source: path.join(docsDir, "v0.48", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.48", "release_notes.html"), title: "Release Notes v0.48", versioned: true },
  { source: path.join(docsDir, "v0.47", "SPEC.md"), output: path.join(siteDir, "v0.47", "spec.html"), title: "Spec v0.47", versioned: true },
  { source: path.join(docsDir, "v0.47", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.47", "release_notes.html"), title: "Release Notes v0.47", versioned: true },
  { source: path.join(docsDir, "v0.46", "SPEC.md"), output: path.join(siteDir, "v0.46", "spec.html"), title: "Spec v0.46", versioned: true },
  { source: path.join(docsDir, "v0.46", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.46", "release_notes.html"), title: "Release Notes v0.46", versioned: true },
  { source: path.join(docsDir, "v0.45", "SPEC.md"), output: path.join(siteDir, "v0.45", "spec.html"), title: "Spec v0.45", versioned: true },
  { source: path.join(docsDir, "v0.45", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.45", "release_notes.html"), title: "Release Notes v0.45", versioned: true },
  { source: path.join(docsDir, "v0.44", "SPEC.md"), output: path.join(siteDir, "v0.44", "spec.html"), title: "Spec v0.44", versioned: true },
  { source: path.join(docsDir, "v0.44", "MIGRATION.md"), output: path.join(siteDir, "v0.44", "migration.html"), title: "Migration v0.44", versioned: true },
  { source: path.join(docsDir, "v0.44", "RELEASE_NOTES.md"), output: path.join(siteDir, "v0.44", "release_notes.html"), title: "Release Notes v0.44", versioned: true },
  { source: path.join(docsDir, "v0.43", "SPEC.md"), output: path.join(siteDir, "v0.43", "spec.html"), title: "Spec v0.43", versioned: true },
  { source: path.join(docsDir, "v0.42", "SPEC.md"), output: path.join(siteDir, "v0.42", "spec.html"), title: "Spec v0.42", versioned: true },
  { source: path.join(docsDir, "v0.41", "SPEC.md"), output: path.join(siteDir, "v0.41", "spec.html"), title: "Spec v0.41", versioned: true },
  { source: path.join(docsDir, "v0.40", "SPEC.md"), output: path.join(siteDir, "v0.40", "spec.html"), title: "Spec v0.40", versioned: true },
  { source: path.join(docsDir, "v0.39", "SPEC.md"), output: path.join(siteDir, "v0.39", "spec.html"), title: "Spec v0.39", versioned: true },
  { source: path.join(docsDir, "v0.38", "SPEC.md"), output: path.join(siteDir, "v0.38", "spec.html"), title: "Spec v0.38", versioned: true },
  { source: path.join(docsDir, "v0.37", "SPEC.md"), output: path.join(siteDir, "v0.37", "spec.html"), title: "Spec v0.37", versioned: true },
  { source: path.join(docsDir, "v0.36", "SPEC.md"), output: path.join(siteDir, "v0.36", "spec.html"), title: "Spec v0.36", versioned: true },
  { source: path.join(docsDir, "v0.35", "SPEC.md"), output: path.join(siteDir, "v0.35", "spec.html"), title: "Spec v0.35", versioned: true },
  { source: path.join(docsDir, "v0.34", "SPEC.md"), output: path.join(siteDir, "v0.34", "spec.html"), title: "Spec v0.34", versioned: true },
  { source: path.join(docsDir, "v0.33", "SPEC.md"), output: path.join(siteDir, "v0.33", "spec.html"), title: "Spec v0.33", versioned: true },
  { source: path.join(docsDir, "v0.32", "SPEC.md"), output: path.join(siteDir, "v0.32", "spec.html"), title: "Spec v0.32", versioned: true },
  { source: path.join(docsDir, "v0.31", "SPEC.md"), output: path.join(siteDir, "v0.31", "spec.html"), title: "Spec v0.31", versioned: true },
  { source: path.join(docsDir, "v0.30", "SPEC.md"), output: path.join(siteDir, "v0.30", "spec.html"), title: "Spec v0.30", versioned: true },
  { source: path.join(docsDir, "v0.29", "SPEC.md"), output: path.join(siteDir, "v0.29", "spec.html"), title: "Spec v0.29", versioned: true },
  { source: path.join(docsDir, "v0.29", "CODES.md"), output: path.join(siteDir, "v0.29", "codes.html"), title: "Codes v0.29", versioned: true },
  { source: path.join(docsDir, "v0.28", "SPEC.md"), output: path.join(siteDir, "v0.28", "spec.html"), title: "Spec v0.28", versioned: true },
  { source: path.join(docsDir, "v0.27", "SPEC.md"), output: path.join(siteDir, "v0.27", "spec.html"), title: "Spec v0.27", versioned: true },
  { source: path.join(docsDir, "v0.26", "SPEC.md"), output: path.join(siteDir, "v0.26", "spec.html"), title: "Spec v0.26", versioned: true },
  { source: path.join(docsDir, "v0.25", "SPEC.md"), output: path.join(siteDir, "v0.25", "spec.html"), title: "Spec v0.25", versioned: true },
  { source: path.join(docsDir, "v0.24", "SPEC.md"), output: path.join(siteDir, "v0.24", "spec.html"), title: "Spec v0.24", versioned: true },
  { source: path.join(docsDir, "v0.23", "SPEC.md"), output: path.join(siteDir, "v0.23", "spec.html"), title: "Spec v0.23", versioned: true },
  { source: path.join(docsDir, "v0.22", "SPEC.md"), output: path.join(siteDir, "v0.22", "spec.html"), title: "Spec v0.22", versioned: true },
  { source: path.join(docsDir, "v0.21", "SPEC.md"), output: path.join(siteDir, "v0.21", "spec.html"), title: "Spec v0.21", versioned: true },
  { source: path.join(docsDir, "v0.20", "SPEC.md"), output: path.join(siteDir, "v0.20", "spec.html"), title: "Spec v0.20", versioned: true },
  { source: path.join(docsDir, "v0.19", "SPEC.md"), output: path.join(siteDir, "v0.19", "spec.html"), title: "Spec v0.19", versioned: true },
  { source: path.join(docsDir, "v0.18", "SPEC.md"), output: path.join(siteDir, "v0.18", "spec.html"), title: "Spec v0.18", versioned: true },
  { source: path.join(docsDir, "v0.17", "SPEC.md"), output: path.join(siteDir, "v0.17", "spec.html"), title: "Spec v0.17", versioned: true },
  { source: path.join(docsDir, "v0.16", "SPEC.md"), output: path.join(siteDir, "v0.16", "spec.html"), title: "Spec v0.16", versioned: true },
  { source: path.join(docsDir, "v0.15", "SPEC.md"), output: path.join(siteDir, "v0.15", "spec.html"), title: "Spec v0.15", versioned: true },
  { source: path.join(docsDir, "v0.14", "SPEC.md"), output: path.join(siteDir, "v0.14", "spec.html"), title: "Spec v0.14", versioned: true },
  { source: path.join(docsDir, "v0.13", "SPEC.md"), output: path.join(siteDir, "v0.13", "spec.html"), title: "Spec v0.13", versioned: true },
  { source: path.join(docsDir, "v0.12", "SPEC.md"), output: path.join(siteDir, "v0.12", "spec.html"), title: "Spec v0.12", versioned: true },
  { source: path.join(docsDir, "v0.11", "SPEC.md"), output: path.join(siteDir, "v0.11", "spec.html"), title: "Spec v0.11", versioned: true },
  { source: path.join(docsDir, "v0.10", "SPEC.md"), output: path.join(siteDir, "v0.10", "spec.html"), title: "Spec v0.10", versioned: true },
  { source: path.join(docsDir, "v0.9", "SPEC.md"), output: path.join(siteDir, "v0.9", "spec.html"), title: "Spec v0.9", versioned: true },
  { source: path.join(docsDir, "v0.8", "SPEC.md"), output: path.join(siteDir, "v0.8", "spec.html"), title: "Spec v0.8", versioned: true },
  { source: path.join(docsDir, "v0.7", "SPEC.md"), output: path.join(siteDir, "v0.7", "spec.html"), title: "Spec v0.7", versioned: true },
  { source: path.join(docsDir, "v0.6", "SPEC.md"), output: path.join(siteDir, "v0.6", "spec.html"), title: "Spec v0.6", versioned: true },
  { source: path.join(docsDir, "v0.5", "SPEC.md"), output: path.join(siteDir, "v0.5", "spec.html"), title: "Spec v0.5", versioned: true },
  { source: path.join(docsDir, "v0.4", "SPEC.md"), output: path.join(siteDir, "v0.4", "spec.html"), title: "Spec v0.4", versioned: true },
  { source: path.join(docsDir, "v0.3", "SPEC.md"), output: path.join(siteDir, "v0.3", "spec.html"), title: "Spec v0.3", versioned: true },
  { source: path.join(docsDir, "v0.3", "STDLIB.md"), output: path.join(siteDir, "v0.3", "stdlib.html"), title: "Stdlib v0.3", versioned: true },
  { source: path.join(docsDir, "v0.2.0", "SPEC.md"), output: path.join(siteDir, "v0.2.0", "spec.html"), title: "Spec v0.2.0", versioned: true },
  { source: path.join(docsDir, "v0.2.0", "API.md"), output: path.join(siteDir, "v0.2.0", "api.html"), title: "API v0.2.0", versioned: true },
  { source: path.join(docsDir, "v0.1.0", "SPEC.md"), output: path.join(siteDir, "v0.1.0", "spec.html"), title: "Spec v0.1.0", versioned: true },
  { source: path.join(docsDir, "v0.1.0", "API.md"), output: path.join(siteDir, "v0.1.0", "api.html"), title: "API v0.1.0", versioned: true },
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
      "LIBRARIES.md": "libraries.html",
      "LINT.md": "lint.html",
      "docs/SPEC.md": "spec.html",
      "docs/API.md": "api.html",
      "docs/STDLIB.md": "stdlib.html",
      "docs/LIBRARIES.md": "libraries.html",
      "docs/LINT.md": "lint.html",
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
    "LIBRARIES.md": "libraries.html",
    "LINT.md": "lint.html",
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
        <a href="${prefix}libraries.html">Libraries</a>
        <a href="${prefix}lint.html">Lint</a>
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
