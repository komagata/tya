package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditorSyntaxAssets(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	required := []string{
		"editors/TOKENS.md",
		"editors/PUBLISHING.md",
		"editors/SHIP_STATUS.md",
		"editors/syntax-sample.tya",
		".github/workflows/editor-assets.yml",
		".github/workflows/publish-vscode-extension.yml",
		"editors/vscode/package.json",
		"editors/vscode/LICENSE.md",
		"editors/vscode/syntaxes/tya.tmLanguage.json",
		"editors/vim/ftdetect/tya.vim",
		"editors/vim/indent/tya.vim",
		"editors/vim/syntax/tya.vim",
		"editors/emacs/tya-mode.el",
		"editors/emacs/melpa-recipe",
		"editors/github-linguist/README.md",
		"editors/github-linguist/tya.gitattributes",
		"editors/github-linguist/languages.yml.example",
		"editors/tree-sitter-tya/grammar.js",
		"editors/tree-sitter-tya/package.json",
		"editors/tree-sitter-tya/tree-sitter.json",
		"editors/tree-sitter-tya/queries/highlights.scm",
		"editors/tree-sitter-tya/src/parser.c",
		"editors/tree-sitter-tya/src/node-types.json",
	}
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("%s missing: %v", rel, err)
		}
	}

	sample := readText(t, filepath.Join(root, "editors/syntax-sample.tya"))
	for _, needle := range []string{"interface Named", "class User implements Named", "spawn ->", "select", "match self.count", "0x2a", "0b1010", "b\"tya\""} {
		if !strings.Contains(sample, needle) {
			t.Fatalf("syntax sample missing %q", needle)
		}
	}

	var pkg struct {
		Contributes struct {
			Languages []struct {
				ID         string   `json:"id"`
				Extensions []string `json:"extensions"`
			} `json:"languages"`
			Grammars []struct {
				Language  string `json:"language"`
				ScopeName string `json:"scopeName"`
				Path      string `json:"path"`
			} `json:"grammars"`
			ConfigurationDefaults map[string]map[string]any `json:"configurationDefaults"`
		} `json:"contributes"`
	}
	readJSON(t, filepath.Join(root, "editors/vscode/package.json"), &pkg)
	if len(pkg.Contributes.Languages) == 0 || pkg.Contributes.Languages[0].ID != "tya" {
		t.Fatalf("VS Code package does not register tya language")
	}
	if !contains(pkg.Contributes.Languages[0].Extensions, ".tya") {
		t.Fatalf("VS Code package does not register .tya extension")
	}
	if len(pkg.Contributes.Grammars) == 0 || pkg.Contributes.Grammars[0].ScopeName != "source.tya" {
		t.Fatalf("VS Code package does not register source.tya grammar")
	}
	tyaDefaults := pkg.Contributes.ConfigurationDefaults["[tya]"]
	if got := tyaDefaults["editor.defaultFormatter"]; got != "komagata.tya" {
		t.Fatalf("VS Code package default formatter = %v, want komagata.tya", got)
	}
	if got := tyaDefaults["editor.formatOnSave"]; got != true {
		t.Fatalf("VS Code package format on save = %v, want true", got)
	}

	var grammar struct {
		ScopeName  string `json:"scopeName"`
		Repository map[string]any
	}
	readJSON(t, filepath.Join(root, "editors/vscode/syntaxes/tya.tmLanguage.json"), &grammar)
	if grammar.ScopeName != "source.tya" {
		t.Fatalf("grammar scope = %q, want source.tya", grammar.ScopeName)
	}
	for _, key := range []string{"comments", "strings", "numbers", "keywords", "literals", "functions", "operators"} {
		if _, ok := grammar.Repository[key]; !ok {
			t.Fatalf("grammar repository missing %s", key)
		}
	}

	vimSyntax := readText(t, filepath.Join(root, "editors/vim/syntax/tya.vim"))
	for _, needle := range []string{"syntax keyword tyaKeyword", "syntax keyword tyaDeclaration", "let b:current_syntax = \"tya\""} {
		if !strings.Contains(vimSyntax, needle) {
			t.Fatalf("Vim syntax missing %q", needle)
		}
	}

	emacsMode := readText(t, filepath.Join(root, "editors/emacs/tya-mode.el"))
	for _, needle := range []string{"Version: 0.61.0", "Package-Requires:", "URL: https://github.com/komagata/tya", "define-derived-mode tya-mode", "tya-mode-font-lock-keywords", "(provide 'tya-mode)"} {
		if !strings.Contains(emacsMode, needle) {
			t.Fatalf("Emacs mode missing %q", needle)
		}
	}

	linguistAttrs := readText(t, filepath.Join(root, "editors/github-linguist/tya.gitattributes"))
	if !strings.Contains(linguistAttrs, "*.tya linguist-language=Tya") {
		t.Fatalf("GitHub Linguist attributes do not map .tya")
	}
	linguistYAML := readText(t, filepath.Join(root, "editors/github-linguist/languages.yml.example"))
	for _, needle := range []string{"Tya:", "extensions:", "- \".tya\"", "tm_scope: source.tya"} {
		if !strings.Contains(linguistYAML, needle) {
			t.Fatalf("GitHub Linguist example missing %q", needle)
		}
	}

	melpaRecipe := readText(t, filepath.Join(root, "editors/emacs/melpa-recipe"))
	for _, needle := range []string{"tya-mode", ":fetcher github", ":repo \"komagata/tya\"", "editors/emacs/tya-mode.el"} {
		if !strings.Contains(melpaRecipe, needle) {
			t.Fatalf("MELPA recipe missing %q", needle)
		}
	}

	var treeSitterPkg struct {
		Name       string `json:"name"`
		TreeSitter []struct {
			Scope      string   `json:"scope"`
			FileTypes  []string `json:"file-types"`
			Highlights string   `json:"highlights"`
		} `json:"tree-sitter"`
	}
	readJSON(t, filepath.Join(root, "editors/tree-sitter-tya/package.json"), &treeSitterPkg)
	if treeSitterPkg.Name != "tree-sitter-tya" || len(treeSitterPkg.TreeSitter) == 0 {
		t.Fatalf("Tree-sitter package metadata is incomplete")
	}
	if treeSitterPkg.TreeSitter[0].Scope != "source.tya" || !contains(treeSitterPkg.TreeSitter[0].FileTypes, "tya") {
		t.Fatalf("Tree-sitter package does not register source.tya / tya")
	}
	if treeSitterPkg.TreeSitter[0].Highlights != "queries/highlights.scm" {
		t.Fatalf("Tree-sitter package highlights = %q", treeSitterPkg.TreeSitter[0].Highlights)
	}
}

func readJSON(t *testing.T, path string, dst any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		t.Fatalf("%s is invalid JSON: %v", path, err)
	}
}

func readText(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
