package tests

import (
	"encoding/json"
	"strings"
	"testing"
)

func initLSP(t *testing.T) *lspProc {
	t.Helper()
	p := startLSP(t)
	p.request("initialize", map[string]any{"processId": 1, "capabilities": map[string]any{}})
	p.notify("initialized", map[string]any{})
	return p
}

func TestLSPInitialize(t *testing.T) {
	p := startLSP(t)
	defer p.close()
	res := p.request("initialize", map[string]any{"processId": 1, "capabilities": map[string]any{}})
	var got struct {
		Capabilities struct {
			HoverProvider              bool `json:"hoverProvider"`
			DefinitionProvider         bool `json:"definitionProvider"`
			DocumentFormattingProvider bool `json:"documentFormattingProvider"`
			CompletionProvider         struct {
				ResolveProvider bool `json:"resolveProvider"`
			} `json:"completionProvider"`
			TextDocumentSync struct {
				OpenClose bool `json:"openClose"`
				Change    int  `json:"change"`
			} `json:"textDocumentSync"`
			RenameProvider struct {
				PrepareProvider bool `json:"prepareProvider"`
			} `json:"renameProvider"`
			InlayHintProvider      bool `json:"inlayHintProvider"`
			CallHierarchyProvider  bool `json:"callHierarchyProvider"`
			SelectionRangeProvider bool `json:"selectionRangeProvider"`
			CodeLensProvider       struct {
				ResolveProvider bool `json:"resolveProvider"`
			} `json:"codeLensProvider"`
			FoldingRangeProvider bool `json:"foldingRangeProvider"`
			DocumentLinkProvider struct {
				ResolveProvider bool `json:"resolveProvider"`
			} `json:"documentLinkProvider"`
		} `json:"capabilities"`
		ServerInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}
	if err := json.Unmarshal(res, &got); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	if !got.Capabilities.HoverProvider {
		t.Error("hover provider not advertised")
	}
	if !got.Capabilities.DefinitionProvider {
		t.Error("definition provider not advertised")
	}
	if !got.Capabilities.DocumentFormattingProvider {
		t.Error("formatting provider not advertised")
	}
	if !got.Capabilities.TextDocumentSync.OpenClose || got.Capabilities.TextDocumentSync.Change != 2 {
		t.Errorf("bad textDocumentSync: %+v", got.Capabilities.TextDocumentSync)
	}
	if !got.Capabilities.RenameProvider.PrepareProvider {
		t.Error("prepareRename not advertised")
	}
	if !got.Capabilities.InlayHintProvider || !got.Capabilities.CallHierarchyProvider || !got.Capabilities.SelectionRangeProvider || !got.Capabilities.FoldingRangeProvider {
		t.Errorf("missing polish providers: %+v", got.Capabilities)
	}
	if got.ServerInfo.Name != "tya" {
		t.Errorf("server name = %q", got.ServerInfo.Name)
	}
}

func TestLSPDiagnosticsOnDidOpen(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_test_diag.tya")
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": "tya",
			"version":    1,
			"text":       "x =\n",
		},
	})
	raw := p.expectNotification("textDocument/publishDiagnostics")
	var got struct {
		URI         string `json:"uri"`
		Diagnostics []any  `json:"diagnostics"`
	}
	json.Unmarshal(raw, &got)
	if got.URI != uri {
		t.Errorf("uri mismatch: %s", got.URI)
	}
	if len(got.Diagnostics) == 0 {
		t.Error("expected at least one diagnostic for invalid source")
	}
}

func TestLSPFormatting(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_test_format.tya")
	// Original source has a redundant trailing newline / spacing.
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": "tya",
			"version":    1,
			"text":       "x   =   1\n",
		},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/formatting", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"options":      map[string]any{"tabSize": 2, "insertSpaces": true},
	})
	var edits []struct {
		NewText string `json:"newText"`
	}
	if err := json.Unmarshal(res, &edits); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	if len(edits) != 1 {
		t.Fatalf("expected 1 edit, got %d: %s", len(edits), res)
	}
	if !strings.Contains(edits[0].NewText, "x = 1") {
		t.Errorf("formatted text missing canonical assignment: %q", edits[0].NewText)
	}
}

func TestLSPHover(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_test_hover.tya")
	src := "# greet someone\ngreet = name -> name\nprint(greet(\"a\"))\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri": uri, "languageId": "tya", "version": 1, "text": src,
		},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 2, "character": 7}, // on `greet` of `print(greet("a"))`
	})
	if string(res) == "null" {
		t.Fatalf("expected hover content, got null")
	}
	var got struct {
		Contents struct{ Value string } `json:"contents"`
	}
	if err := json.Unmarshal(res, &got); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	if !strings.Contains(got.Contents.Value, "greet(name)") {
		t.Errorf("hover missing signature: %q", got.Contents.Value)
	}
}

func TestLSPDefinition(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_test_def.tya")
	src := "greet = name -> name\nprint(greet(\"a\"))\n"
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri": uri, "languageId": "tya", "version": 1, "text": src,
		},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 1, "character": 7},
	})
	var locs []struct {
		URI   string `json:"uri"`
		Range struct {
			Start struct{ Line, Character int }
		} `json:"range"`
	}
	if err := json.Unmarshal(res, &locs); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	if len(locs) != 1 {
		t.Fatalf("expected 1 location, got %d: %s", len(locs), res)
	}
	if locs[0].URI != uri {
		t.Errorf("location uri mismatch: %s", locs[0].URI)
	}
	if locs[0].Range.Start.Line != 0 {
		t.Errorf("expected line 0 for definition, got %d", locs[0].Range.Start.Line)
	}
}

func TestLSPCompletion(t *testing.T) {
	p := initLSP(t)
	defer p.close()
	uri := fileURI("/tmp/lsp_test_comp.tya")
	p.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri": uri, "languageId": "tya", "version": 1, "text": "",
		},
	})
	p.expectNotification("textDocument/publishDiagnostics")
	res := p.request("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 0, "character": 0},
	})
	var list struct {
		Items []struct {
			Label string `json:"label"`
		} `json:"items"`
	}
	if err := json.Unmarshal(res, &list); err != nil {
		t.Fatalf("decode: %v\n%s", err, res)
	}
	got := map[string]bool{}
	for _, it := range list.Items {
		got[it.Label] = true
	}
	for _, want := range []string{"print", "if"} {
		if !got[want] {
			t.Errorf("completion missing %q (items: %d)", want, len(list.Items))
		}
	}
}

func TestLSPShutdownExit(t *testing.T) {
	p := initLSP(t)
	res := p.request("shutdown", nil)
	if string(res) != "null" {
		t.Errorf("shutdown result = %s, want null", res)
	}
	p.notify("exit", nil)
	p.close()
}
