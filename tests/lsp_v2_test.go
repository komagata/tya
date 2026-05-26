package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper: write multi-file workspace fixture into a temp dir,
// return its absolute path.
func writeWorkspace(t *testing.T, files map[string]string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "tya-lsp-ws-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestLSPCrossFileDefinition(t *testing.T) {
	dir := writeWorkspace(t, map[string]string{
		"tya.toml":        "name = \"demo\"\nversion = \"0.1.0\"\nlicense = \"MIT\"\n",
		"src/helpers.tya": "# add two numbers\nadd = a, b -> a + b\n",
		"src/main.tya":    "import helpers\n\nhelpers.add(1, 2)\n",
	})
	p := initLSP(t)
	defer p.close()
	mainPath := filepath.Join(dir, "src/main.tya")
	helpersPath := filepath.Join(dir, "src/helpers.tya")
	mainURI := fileURI(mainPath)
	src, _ := os.ReadFile(mainPath)
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri": mainURI, "languageId": "tya", "version": 1, "text": string(src),
		},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     map[string]any{"line": 2, "character": 8}, // on `add` of helpers.add
	})
	var locs []struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(res, &locs); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	if len(locs) == 0 {
		t.Fatalf("expected at least one location, got %s", res)
	}
	wantURI := fileURI(helpersPath)
	if locs[0].URI != wantURI {
		t.Errorf("URI = %q, want %q", locs[0].URI, wantURI)
	}
}

func TestLSPRenameSameFileTopLevel(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_v2_rename_top.tya")
	src := "greet = name -> name\nprint(greet(\"hi\"))\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": src},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/rename", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 0, "character": 0},
		"newName":      "salute",
	})
	var edit struct {
		Changes map[string][]struct {
			NewText string `json:"newText"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(res, &edit); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	if len(edit.Changes[uri]) != 2 {
		t.Fatalf("expected 2 edits, got %d: %s", len(edit.Changes[uri]), res)
	}
	for _, e := range edit.Changes[uri] {
		if e.NewText != "salute" {
			t.Errorf("edit newText = %q", e.NewText)
		}
	}
}

func TestLSPRenameTopLevelClassDeclaration(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_v2_rename_class.tya")
	src := "class Widget\n  static name: () -> \"widget\"\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": src},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/rename", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 0, "character": 6},
		"newName":      "RenamedWidget",
	})
	var edit struct {
		Changes map[string][]struct {
			NewText string `json:"newText"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(res, &edit); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	if len(edit.Changes[uri]) != 1 {
		t.Fatalf("expected 1 edit, got %d: %s", len(edit.Changes[uri]), res)
	}
	if edit.Changes[uri][0].NewText != "RenamedWidget" {
		t.Errorf("edit newText = %q", edit.Changes[uri][0].NewText)
	}
}

func TestLSPRenameClassDeclarationRenamesMatchingFile(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	dir := t.TempDir()
	path := filepath.Join(dir, "Widget.tya")
	uri := fileURI(path)
	src := "class Widget\n  static name: () -> \"widget\"\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": src},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/rename", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 0, "character": 6},
		"newName":      "RenamedWidget",
	})
	var edit struct {
		Changes         map[string][]any `json:"changes"`
		DocumentChanges []struct {
			Kind   string `json:"kind"`
			OldURI string `json:"oldUri"`
			NewURI string `json:"newUri"`
		} `json:"documentChanges"`
	}
	if err := json.Unmarshal(res, &edit); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	if len(edit.Changes[uri]) != 1 {
		t.Fatalf("expected class text edit, got %s", res)
	}
	wantNewURI := fileURI(filepath.Join(dir, "RenamedWidget.tya"))
	foundRename := false
	for _, change := range edit.DocumentChanges {
		if change.Kind == "rename" && change.OldURI == uri && change.NewURI == wantNewURI {
			foundRename = true
		}
	}
	if !foundRename {
		t.Fatalf("expected file rename %s -> %s in %s", uri, wantNewURI, res)
	}
}

func TestLSPRenameClassDeclarationRenamesMismatchedFile(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	dir := t.TempDir()
	path := filepath.Join(dir, "OldWidget.tya")
	uri := fileURI(path)
	src := "class Widget\n  static name: () -> \"widget\"\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": src},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/rename", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 0, "character": 6},
		"newName":      "RenamedWidget",
	})
	var edit struct {
		DocumentChanges []struct {
			Kind   string `json:"kind"`
			OldURI string `json:"oldUri"`
			NewURI string `json:"newUri"`
		} `json:"documentChanges"`
	}
	if err := json.Unmarshal(res, &edit); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	wantNewURI := fileURI(filepath.Join(dir, "RenamedWidget.tya"))
	for _, change := range edit.DocumentChanges {
		if change.Kind == "rename" && change.OldURI == uri && change.NewURI == wantNewURI {
			return
		}
	}
	t.Fatalf("expected mismatched file rename %s -> %s in %s", uri, wantNewURI, res)
}

func TestLSPRenameClassDoesNotRenameImportedPackageClass(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	dir := writeWorkspace(t, map[string]string{
		"tya.toml":    "name = \"demo\"\nversion = \"0.1.0\"\nlicense = \"MIT\"\n",
		"src/Cli.tya": "import cli as cli\n\nclass Cli\n  static parse: args ->\n    cli.Cli(args).parse_or_exit(Self.option_spec())\n  static option_spec: () -> {}\n",
	})
	uri := fileURI(filepath.Join(dir, "src", "Cli.tya"))
	src, _ := os.ReadFile(filepath.Join(dir, "src", "Cli.tya"))
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": string(src)},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/rename", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 2, "character": 6},
		"newName":      "CommandLine",
	})
	var edit struct {
		Changes map[string][]struct {
			Range struct {
				Start struct {
					Line int `json:"line"`
				} `json:"start"`
			} `json:"range"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(res, &edit); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	for _, textEdit := range edit.Changes[uri] {
		if textEdit.Range.Start.Line == 4 {
			t.Fatalf("imported package class cli.Cli was renamed: %s", res)
		}
	}
}

func TestLSPRenameClassMethodDeclaration(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_v2_rename_class_method.tya")
	src := "class Widget\n  static call: () -> Self.helper()\n  static helper: () -> 1\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": src},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/rename", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 2, "character": 9},
		"newName":      "renamed_helper",
	})
	var edit struct {
		Changes map[string][]struct {
			NewText string `json:"newText"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(res, &edit); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	if len(edit.Changes[uri]) != 2 {
		t.Fatalf("expected 2 edits, got %d: %s", len(edit.Changes[uri]), res)
	}
}

func TestLSPRenameClassMethodReference(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_v2_rename_class_method_ref.tya")
	src := "class Widget\n  static call: () -> Self.helper()\n  static helper: () -> 1\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": src},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/rename", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 1, "character": 27},
		"newName":      "renamed_helper",
	})
	var edit struct {
		Changes map[string][]struct {
			NewText string `json:"newText"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(res, &edit); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	if len(edit.Changes[uri]) != 2 {
		t.Fatalf("expected 2 edits, got %d: %s", len(edit.Changes[uri]), res)
	}
}

func TestLSPRenameLocalInsideClassMethod(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_v2_rename_class_method_local.tya")
	src := "class Widget\n  static call: () ->\n    value = 1\n    value + 1\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": src},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/rename", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 2, "character": 4},
		"newName":      "amount",
	})
	var edit struct {
		Changes map[string][]struct {
			NewText string `json:"newText"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(res, &edit); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	if len(edit.Changes[uri]) != 2 {
		t.Fatalf("expected 2 edits, got %d: %s", len(edit.Changes[uri]), res)
	}
}

func TestLSPPrepareRename(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_prepare_rename.tya")
	src := "greet = name ->\n  println(name)\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": src},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/prepareRename", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 0, "character": 1},
	})
	var got struct {
		Placeholder string `json:"placeholder"`
		Range       struct {
			Start struct {
				Line int `json:"line"`
			} `json:"start"`
		} `json:"range"`
	}
	if err := json.Unmarshal(res, &got); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	if got.Placeholder != "greet" || got.Range.Start.Line != 0 {
		t.Fatalf("bad prepareRename: %+v", got)
	}
}

func TestLSPRenameLocal(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_v2_rename_local.tya")
	src := "outer = x\nf = y ->\n  z = y + 1\n  z * 2\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": src},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/rename", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 2, "character": 2}, // on `z`
		"newName":      "w",
	})
	var edit struct {
		Changes map[string][]any `json:"changes"`
	}
	if err := json.Unmarshal(res, &edit); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	if len(edit.Changes[uri]) == 0 {
		t.Fatalf("expected local rename edits, got %s", res)
	}
}

func TestLSPRenameConflict(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_v2_rename_conflict.tya")
	// Both 'a' and 'b' are top-level — renaming 'a' to 'b' should
	// be rejected.
	src := "a = 1\nb = 2\nprint(a + b)\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": src},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	// We expect an error response, not a success. Send raw.
	p.nextID++
	id := p.nextID
	p.writeMessage(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "textDocument/rename",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 0, "character": 0},
			"newName":      "b",
		},
	})
	// Drain until matching id.
	for {
		m := p.readMessage()
		if mid, ok := m["id"]; ok && mid != nil {
			if errVal, ok := m["error"]; ok && errVal != nil {
				body, _ := json.Marshal(errVal)
				if !strings.Contains(string(body), "TYA-E0933") {
					t.Errorf("error did not mention TYA-E0933: %s", body)
				}
				return
			}
			t.Fatalf("expected rename conflict error, got result %s", m["result"])
		}
	}
}

func TestLSPReferences(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_v2_refs.tya")
	src := "greet = n -> n\nprint(greet(\"a\"))\nprint(greet(\"b\"))\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": src},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/references", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 0, "character": 0},
		"context":      map[string]any{"includeDeclaration": true},
	})
	var locs []any
	if err := json.Unmarshal(res, &locs); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	if len(locs) < 3 {
		t.Fatalf("expected >=3 references (decl + 2 uses), got %d: %s", len(locs), res)
	}
}

func TestLSPRangeFormatting(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_v2_rangefmt.tya")
	src := "x = 1\n\n\ny = 2\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": src},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/rangeFormatting", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"range": map[string]any{
			"start": map[string]any{"line": 0, "character": 0},
			"end":   map[string]any{"line": 0, "character": 5},
		},
	})
	var edits []any
	if err := json.Unmarshal(res, &edits); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	// Heuristic A may return 0 or 1 edits depending on whether
	// formatter.Unparse reshapes the buffer. Either way, no error.
	_ = edits
}

func TestLSPCodeActionsUnused(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_v2_codeactions.tya")
	src := "f = x ->\n  y = 1\n  x\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": src},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/codeAction", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"range": map[string]any{
			"start": map[string]any{"line": 1, "character": 2},
			"end":   map[string]any{"line": 1, "character": 6},
		},
		"context": map[string]any{
			"diagnostics": []any{map[string]any{
				"code":     "TYAL0001",
				"message":  "unused local",
				"range":    map[string]any{"start": map[string]any{"line": 1, "character": 2}, "end": map[string]any{"line": 1, "character": 3}},
				"severity": 2,
			}},
		},
	})
	var actions []struct {
		Title string `json:"title"`
		Kind  string `json:"kind"`
	}
	if err := json.Unmarshal(res, &actions); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	found := false
	for _, a := range actions {
		if a.Kind == "quickfix" && strings.Contains(a.Title, "Remove unused") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unused-local quickfix, got %s", res)
	}
}

func TestLSPSemanticTokens(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_v2_sem.tya")
	src := "x = 1\nclass Foo\n  static bar: -> 42\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": src},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/semanticTokens/full", map[string]any{
		"textDocument": map[string]any{"uri": uri},
	})
	var st struct {
		Data []uint32 `json:"data"`
	}
	if err := json.Unmarshal(res, &st); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	if len(st.Data) == 0 || len(st.Data)%5 != 0 {
		t.Errorf("bad semantic tokens data length: %d", len(st.Data))
	}
}

func TestLSPPolishProviders(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_polish.tya")
	src := "# [docs](https://example.com)\nimport unittest\n\nclass SampleTest extends TestCase\n\n  test_one: ->\n    println(\"ok\")\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": src},
	})
	p.expectNotification("textDocument/publishDiagnostics")

	folds := p.request("textDocument/foldingRange", map[string]any{"textDocument": map[string]any{"uri": uri}})
	var folding []any
	if err := json.Unmarshal(folds, &folding); err != nil || len(folding) == 0 {
		t.Fatalf("foldingRange = %s err=%v", folds, err)
	}

	lenses := p.request("textDocument/codeLens", map[string]any{"textDocument": map[string]any{"uri": uri}})
	var codeLens []any
	if err := json.Unmarshal(lenses, &codeLens); err != nil || len(codeLens) == 0 {
		t.Fatalf("codeLens = %s err=%v", lenses, err)
	}

	links := p.request("textDocument/documentLink", map[string]any{"textDocument": map[string]any{"uri": uri}})
	var docLinks []any
	if err := json.Unmarshal(links, &docLinks); err != nil || len(docLinks) == 0 {
		t.Fatalf("documentLink = %s err=%v", links, err)
	}

	selections := p.request("textDocument/selectionRange", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"positions":    []any{map[string]any{"line": 1, "character": 2}},
	})
	var ranges []struct {
		Parent any `json:"parent"`
	}
	if err := json.Unmarshal(selections, &ranges); err != nil || len(ranges) != 1 || ranges[0].Parent == nil {
		t.Fatalf("selectionRange = %s err=%v", selections, err)
	}

	hints := p.request("textDocument/inlayHint", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"range":        map[string]any{"start": map[string]any{"line": 0, "character": 0}, "end": map[string]any{"line": 3, "character": 0}},
	})
	var inlay []any
	if err := json.Unmarshal(hints, &inlay); err != nil {
		t.Fatalf("inlayHint = %s err=%v", hints, err)
	}
}

func TestLSPIncrementalSync(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_v2_incsync.tya")
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": "x = 1\n"},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	// Insert a `y =` so a parse error appears.
	p.notify("textDocument/didChange", map[string]any{
		"textDocument": map[string]any{"uri": uri, "version": 2},
		"contentChanges": []any{map[string]any{
			"range": map[string]any{
				"start": map[string]any{"line": 1, "character": 0},
				"end":   map[string]any{"line": 1, "character": 0},
			},
			"text": "y =\n",
		}},
	})
	raw := p.expectNotification("textDocument/publishDiagnostics")
	var got struct {
		Diagnostics []any `json:"diagnostics"`
	}
	json.Unmarshal(raw, &got)
	if len(got.Diagnostics) == 0 {
		t.Error("expected at least one diagnostic after incremental edit")
	}
}

func TestLSPDocumentSymbols(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_v2_docsyms.tya")
	src := "class Foo\n  private LIMIT: 10\n  private static count: 0\n  private field: 1\n  initialize: ->\n    self.field = 1\n  static bar: -> 42\n  private helper: value -> value\n\ngreet = -> 1\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "tya", "version": 1, "text": src},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/documentSymbol", map[string]any{
		"textDocument": map[string]any{"uri": uri},
	})
	var syms []struct {
		Name   string `json:"name"`
		Detail string `json:"detail"`
		Kind   int    `json:"kind"`
		Range  struct {
			End struct {
				Line int `json:"line"`
			} `json:"end"`
		} `json:"range"`
		Children []struct {
			Name   string `json:"name"`
			Detail string `json:"detail"`
			Kind   int    `json:"kind"`
			Range  struct {
				Start struct {
					Line int `json:"line"`
				} `json:"start"`
			} `json:"range"`
		} `json:"children"`
	}
	if err := json.Unmarshal(res, &syms); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	got := map[string]bool{}
	for _, s := range syms {
		got[s.Name] = true
		for _, c := range s.Children {
			got[s.Name+"."+c.Name] = true
		}
	}
	if !got["Foo"] || !got["Foo.bar"] || !got["greet"] {
		t.Errorf("missing expected symbols, got %v", got)
	}
	wantDetails := map[string]string{
		"greet":          "()",
		"Foo.bar":        "() static",
		"Foo.helper":     "(value) private",
		"Foo.initialize": "()",
		"Foo.field":      "private",
		"Foo.LIMIT":      "private",
		"Foo.count":      "private static",
	}
	for _, s := range syms {
		if want, ok := wantDetails[s.Name]; ok && s.Detail != want {
			t.Fatalf("symbol %s detail = %q, want %q", s.Name, s.Detail, want)
		}
		for _, c := range s.Children {
			key := s.Name + "." + c.Name
			if want, ok := wantDetails[key]; ok && c.Detail != want {
				t.Fatalf("symbol %s detail = %q, want %q", key, c.Detail, want)
			}
		}
	}
	wantKinds := map[string]int{
		"Foo.initialize": 9,
		"Foo.field":      8,
	}
	for _, s := range syms {
		for _, c := range s.Children {
			key := s.Name + "." + c.Name
			if want, ok := wantKinds[key]; ok && c.Kind != want {
				t.Fatalf("symbol %s kind = %d, want %d", key, c.Kind, want)
			}
		}
	}
	for _, s := range syms {
		if s.Name != "Foo" || len(s.Children) == 0 {
			continue
		}
		if s.Range.End.Line < s.Children[0].Range.Start.Line {
			t.Fatalf("class range does not enclose method child: %s", res)
		}
	}
}

func TestLSPWorkspaceSymbols(t *testing.T) {
	dir := writeWorkspace(t, map[string]string{
		"tya.toml":  "name = \"demo\"\nversion = \"0.1.0\"\nlicense = \"MIT\"\n",
		"src/a.tya": "greet = -> 1\n",
		"src/b.tya": "class Greeter\n  static hi: -> 1\n",
	})
	p := initLSP(t)
	defer p.close()
	a := filepath.Join(dir, "src/a.tya")
	src, _ := os.ReadFile(a)
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri": fileURI(a), "languageId": "tya", "version": 1, "text": string(src),
		},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("workspace/symbol", map[string]any{"query": "gr"})
	var syms []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(res, &syms); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	got := map[string]bool{}
	for _, s := range syms {
		got[s.Name] = true
	}
	if !got["greet"] || !got["Greeter"] {
		t.Errorf("missing workspace symbols, got %v (res=%s)", got, res)
	}
}
