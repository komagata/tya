package lsp

import (
	"encoding/json"
	"errors"
	"io"
	"sync/atomic"
)

// Version is reported back to the LSP client in ServerInfo. The
// CLI overrides it at startup so it stays in sync with the rest
// of the tya binary.
var Version = "0.52.0"

// Server holds the running LSP server's state.
type Server struct {
	conn         *Conn
	store        *Store
	workspace    *Workspace
	log          Logger
	initialized  atomic.Bool
	shuttingDown atomic.Bool
}

// Run starts the LSP server loop. It returns when the client
// sends `exit` or stdin closes.
func Run(in io.Reader, out io.Writer, log Logger) error {
	if log == nil {
		log = NullLogger
	}
	s := &Server{conn: NewConn(in, out, log), store: NewStore(), workspace: NewWorkspace(), log: log}
	for {
		msg, err := s.conn.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			s.log.Errorf("[TYA-E0931] read: %v", err)
			return err
		}
		if err := s.handle(msg); err != nil {
			s.log.Errorf("handle %s: %v", msg.Method, err)
		}
		if s.shuttingDown.Load() && msg.Method == "exit" {
			return nil
		}
	}
}

func (s *Server) handle(m *Message) error {
	isReq := m.ID != nil
	switch m.Method {
	case "initialize":
		return s.onInitialize(m)
	case "initialized":
		return nil
	case "shutdown":
		s.shuttingDown.Store(true)
		return s.conn.WriteResponse(m.ID, nil)
	case "exit":
		return nil
	case "textDocument/didOpen":
		return s.onDidOpen(m)
	case "textDocument/didChange":
		return s.onDidChange(m)
	case "textDocument/didSave":
		return s.onDidSave(m)
	case "textDocument/didClose":
		return s.onDidClose(m)
	case "textDocument/formatting":
		return s.onFormatting(m)
	case "textDocument/hover":
		return s.onHover(m)
	case "textDocument/definition":
		return s.onDefinition(m)
	case "textDocument/completion":
		return s.onCompletion(m)
	case "textDocument/references":
		return s.onReferences(m)
	case "textDocument/rename":
		return s.onRename(m)
	case "textDocument/prepareRename":
		return s.onPrepareRename(m)
	case "textDocument/rangeFormatting":
		return s.onRangeFormatting(m)
	case "textDocument/codeAction":
		return s.onCodeAction(m)
	case "textDocument/semanticTokens/full":
		return s.onSemanticTokens(m)
	case "textDocument/documentSymbol":
		return s.onDocumentSymbol(m)
	case "textDocument/inlayHint":
		return s.onInlayHint(m)
	case "textDocument/selectionRange":
		return s.onSelectionRange(m)
	case "textDocument/codeLens":
		return s.onCodeLens(m)
	case "textDocument/foldingRange":
		return s.onFoldingRange(m)
	case "textDocument/documentLink":
		return s.onDocumentLink(m)
	case "textDocument/prepareCallHierarchy":
		return s.onPrepareCallHierarchy(m)
	case "callHierarchy/incomingCalls":
		return s.conn.WriteResponse(m.ID, EmptyIncomingCalls())
	case "callHierarchy/outgoingCalls":
		return s.conn.WriteResponse(m.ID, EmptyOutgoingCalls())
	case "workspace/symbol":
		return s.onWorkspaceSymbol(m)
	default:
		if isReq {
			return s.conn.WriteError(m.ID, codeMethodNotFound, "method "+m.Method+" not supported")
		}
		return nil
	}
}

func (s *Server) onInitialize(m *Message) error {
	res := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: TextDocumentSyncOptions{
				OpenClose: true,
				Change:    2, // Incremental (v0.53)
				Save:      &SaveOptions{IncludeText: false},
			},
			HoverProvider:                   true,
			DefinitionProvider:              true,
			ReferencesProvider:              true,
			RenameProvider:                  RenameOptions{PrepareProvider: true},
			CompletionProvider:              &CompletionOptions{ResolveProvider: false},
			DocumentFormattingProvider:      true,
			DocumentRangeFormattingProvider: true,
			CodeActionProvider:              &CodeActionOptions{CodeActionKinds: []string{CodeActionKindQuickFix}},
			SemanticTokensProvider: &SemanticTokensOptions{
				Legend: SemanticTokensLegendValue(),
				Range:  false,
				Full:   true,
			},
			DocumentSymbolProvider:  true,
			WorkspaceSymbolProvider: true,
			InlayHintProvider:       true,
			CallHierarchyProvider:   true,
			SelectionRangeProvider:  true,
			CodeLensProvider:        &CodeLensOptions{ResolveProvider: false},
			FoldingRangeProvider:    true,
			DocumentLinkProvider:    &DocumentLinkOptions{ResolveProvider: false},
			PositionEncoding:        "utf-8",
		},
		ServerInfo: ServerInfo{Name: "tya", Version: Version},
	}
	s.initialized.Store(true)
	return s.conn.WriteResponse(m.ID, res)
}

func (s *Server) onDidOpen(m *Message) error {
	var p DidOpenTextDocumentParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return err
	}
	s.store.Open(p.TextDocument.URI, p.TextDocument.Version, p.TextDocument.Text)
	if s.workspace != nil {
		s.workspace.EnsureRoot(p.TextDocument.URI)
		if path, err := URIToPath(p.TextDocument.URI); err == nil {
			s.workspace.PutText(path, p.TextDocument.Text)
		}
	}
	return s.publish(p.TextDocument.URI)
}

func (s *Server) onDidChange(m *Message) error {
	var p DidChangeTextDocumentParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return err
	}
	if len(p.ContentChanges) == 0 {
		return nil
	}
	text, _ := s.store.ApplyChanges(p.TextDocument.URI, p.TextDocument.Version, p.ContentChanges)
	if s.workspace != nil {
		if path, err := URIToPath(p.TextDocument.URI); err == nil {
			s.workspace.PutText(path, text)
		}
	}
	return s.publish(p.TextDocument.URI)
}

func (s *Server) onDidSave(m *Message) error {
	var p DidSaveTextDocumentParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return err
	}
	s.store.Save(p.TextDocument.URI, p.Text)
	return s.publish(p.TextDocument.URI)
}

func (s *Server) onDidClose(m *Message) error {
	var p DidCloseTextDocumentParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return err
	}
	if s.workspace != nil {
		if path, err := URIToPath(p.TextDocument.URI); err == nil {
			s.workspace.Drop(path)
		}
	}
	s.store.Close(p.TextDocument.URI)
	return s.conn.WriteNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:         p.TextDocument.URI,
		Diagnostics: []Diagnostic{},
	})
}

func (s *Server) publish(uri string) error {
	d, ok := s.store.Get(uri)
	if !ok {
		return nil
	}
	diags := DiagnosticsFor(d.Path, d.Text)
	v := d.Version
	return s.conn.WriteNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:         uri,
		Version:     &v,
		Diagnostics: diags,
	})
}

func (s *Server) onFormatting(m *Message) error {
	var p DocumentFormattingParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		return s.conn.WriteResponse(m.ID, []TextEdit{})
	}
	edits, err := Format(d.Text)
	if err != nil {
		s.log.Errorf("format %s: %v", p.TextDocument.URI, err)
		return s.conn.WriteResponse(m.ID, []TextEdit{})
	}
	return s.conn.WriteResponse(m.ID, edits)
}

func (s *Server) onHover(m *Message) error {
	var p TextDocumentPositionParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		return s.conn.WriteResponse(m.ID, nil)
	}
	h, err := HoverAt(d, p.Position.Line, p.Position.Character)
	if err != nil {
		return s.conn.WriteResponse(m.ID, nil)
	}
	return s.conn.WriteResponse(m.ID, h)
}

func (s *Server) onDefinition(m *Message) error {
	var p TextDocumentPositionParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		return s.conn.WriteResponse(m.ID, []Location{})
	}
	locs, err := DefinitionAt(DefinitionContext{Doc: d, Workspace: s.workspace}, p.Position.Line, p.Position.Character)
	if err != nil {
		return s.conn.WriteResponse(m.ID, []Location{})
	}
	if locs == nil {
		locs = []Location{}
	}
	return s.conn.WriteResponse(m.ID, locs)
}

func (s *Server) onReferences(m *Message) error {
	var p ReferenceParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		return s.conn.WriteResponse(m.ID, []Location{})
	}
	locs, err := References(DefinitionContext{Doc: d, Workspace: s.workspace}, p.Position.Line, p.Position.Character, p.Context.IncludeDeclaration)
	if err != nil {
		return s.conn.WriteResponse(m.ID, []Location{})
	}
	if locs == nil {
		locs = []Location{}
	}
	return s.conn.WriteResponse(m.ID, locs)
}

func (s *Server) onRename(m *Message) error {
	var p RenameParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		return s.conn.WriteError(m.ID, codeInvalidRequest, "[TYA-E0933] no open document")
	}
	edit, err := Rename(DefinitionContext{Doc: d, Workspace: s.workspace}, p.Position.Line, p.Position.Character, p.NewName)
	if err != nil {
		return s.conn.WriteError(m.ID, codeInvalidRequest, err.Error())
	}
	return s.conn.WriteResponse(m.ID, edit)
}

func (s *Server) onPrepareRename(m *Message) error {
	var p PrepareRenameParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		return s.conn.WriteError(m.ID, codeInvalidRequest, "[TYA-E0933] no open document")
	}
	res, err := PrepareRename(d, p.Position.Line, p.Position.Character)
	if err != nil {
		return s.conn.WriteError(m.ID, codeInvalidRequest, err.Error())
	}
	return s.conn.WriteResponse(m.ID, res)
}

func (s *Server) onRangeFormatting(m *Message) error {
	var p DocumentRangeFormattingParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		return s.conn.WriteResponse(m.ID, []TextEdit{})
	}
	edits, err := FormatRange(d.Text, p.Range)
	if err != nil {
		s.log.Errorf("rangeFormatting %s: %v", p.TextDocument.URI, err)
		return s.conn.WriteResponse(m.ID, []TextEdit{})
	}
	return s.conn.WriteResponse(m.ID, edits)
}

func (s *Server) onCodeAction(m *Message) error {
	var p CodeActionParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		return s.conn.WriteResponse(m.ID, []CodeAction{})
	}
	actions := CodeActions(d, p)
	if actions == nil {
		actions = []CodeAction{}
	}
	return s.conn.WriteResponse(m.ID, actions)
}

func (s *Server) onSemanticTokens(m *Message) error {
	var p SemanticTokensParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		return s.conn.WriteResponse(m.ID, SemanticTokens{Data: []uint32{}})
	}
	return s.conn.WriteResponse(m.ID, SemanticTokensFor(d.Text))
}

func (s *Server) onDocumentSymbol(m *Message) error {
	var p DocumentSymbolParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		return s.conn.WriteResponse(m.ID, []DocumentSymbol{})
	}
	syms := DocumentSymbolsFor(d)
	if syms == nil {
		syms = []DocumentSymbol{}
	}
	return s.conn.WriteResponse(m.ID, syms)
}

func (s *Server) onWorkspaceSymbol(m *Message) error {
	var p WorkspaceSymbolParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	syms := WorkspaceSymbolsFor(s.workspace, p.Query)
	if syms == nil {
		syms = []SymbolInformation{}
	}
	return s.conn.WriteResponse(m.ID, syms)
}

func (s *Server) onInlayHint(m *Message) error {
	var p InlayHintParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		return s.conn.WriteResponse(m.ID, []InlayHint{})
	}
	return s.conn.WriteResponse(m.ID, InlayHintsFor(d, p.Range))
}

func (s *Server) onSelectionRange(m *Message) error {
	var p SelectionRangeParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		return s.conn.WriteResponse(m.ID, []SelectionRange{})
	}
	return s.conn.WriteResponse(m.ID, SelectionRangesFor(d, p.Positions))
}

func (s *Server) onCodeLens(m *Message) error {
	var p CodeLensParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		return s.conn.WriteResponse(m.ID, []CodeLens{})
	}
	return s.conn.WriteResponse(m.ID, CodeLensesFor(d))
}

func (s *Server) onFoldingRange(m *Message) error {
	var p FoldingRangeParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		return s.conn.WriteResponse(m.ID, []FoldingRange{})
	}
	return s.conn.WriteResponse(m.ID, FoldingRangesFor(d.Text))
}

func (s *Server) onDocumentLink(m *Message) error {
	var p DocumentLinkParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		return s.conn.WriteResponse(m.ID, []DocumentLink{})
	}
	return s.conn.WriteResponse(m.ID, DocumentLinksFor(d))
}

func (s *Server) onPrepareCallHierarchy(m *Message) error {
	var p CallHierarchyPrepareParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		return s.conn.WriteResponse(m.ID, []CallHierarchyItem{})
	}
	return s.conn.WriteResponse(m.ID, PrepareCallHierarchy(d, p.Position))
}

func (s *Server) onCompletion(m *Message) error {
	var p TextDocumentPositionParams
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return s.conn.WriteError(m.ID, codeInvalidParams, err.Error())
	}
	d, _ := s.store.Get(p.TextDocument.URI)
	list, err := Completion(d)
	if err != nil {
		return s.conn.WriteResponse(m.ID, CompletionList{IsIncomplete: false, Items: []CompletionItem{}})
	}
	return s.conn.WriteResponse(m.ID, list)
}
