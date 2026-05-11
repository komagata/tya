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
	s := &Server{conn: NewConn(in, out, log), store: NewStore(), log: log}
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
				Change:    1,
				Save:      &SaveOptions{IncludeText: false},
			},
			HoverProvider:              true,
			DefinitionProvider:         true,
			CompletionProvider:         &CompletionOptions{ResolveProvider: false},
			DocumentFormattingProvider: true,
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
	last := p.ContentChanges[len(p.ContentChanges)-1]
	s.store.Change(p.TextDocument.URI, p.TextDocument.Version, last.Text)
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
	locs, err := Definition(d, p.Position.Line, p.Position.Character)
	if err != nil {
		return s.conn.WriteResponse(m.ID, []Location{})
	}
	if locs == nil {
		locs = []Location{}
	}
	return s.conn.WriteResponse(m.ID, locs)
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
