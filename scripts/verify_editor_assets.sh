#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$repo_root"
go test ./tests -run TestEditorSyntaxAssets -count=1
node --check editors/tree-sitter-tya/grammar.js
node -e 'JSON.parse(require("fs").readFileSync("editors/vscode/syntaxes/tya.tmLanguage.json", "utf8")); JSON.parse(require("fs").readFileSync("editors/tree-sitter-tya/package.json", "utf8")); JSON.parse(require("fs").readFileSync("editors/tree-sitter-tya/tree-sitter.json", "utf8"));'

if command -v tree-sitter >/dev/null 2>&1; then
  (cd editors/tree-sitter-tya && tree-sitter generate)
  if (cd editors/tree-sitter-tya && tree-sitter parse ../syntax-sample.tya | grep -q "ERROR"); then
    echo "tree-sitter parse produced ERROR nodes for editors/syntax-sample.tya" >&2
    exit 1
  fi
else
  echo "tree-sitter not found; skipping tree-sitter generate" >&2
fi

if command -v nvim >/dev/null 2>&1; then
  nvim --headless -u NONE \
    +"set rtp^=$repo_root/editors/vim" \
    +"filetype on" \
    +"syntax on" \
    +"edit /tmp/tya-editor-asset-check.tya" \
    +"if &filetype !=# 'tya' | cquit | endif" \
    +"syntax list tyaKeyword" \
    +"qa!"
else
  echo "nvim not found; skipping Vim runtime load check" >&2
fi

(cd editors/vscode && npm run compile && npm run package)
