package lsp

import "encoding/json"

// Position is the LSP zero-origin (line, character) pair.
// Tya identifiers are ASCII-only, so Character is treated as a
// byte offset on the encoded UTF-8 source. UTF-16 strictness is
// queued for v0.53+ (see SPEC §Position).
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range is an inclusive-start, exclusive-end span of positions.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location is a (URI, Range) pair used by goto-definition.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextEdit is a single textual edit to apply to a document.
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// Diagnostic severity values.
const (
	DiagSeverityError       = 1
	DiagSeverityWarning     = 2
	DiagSeverityInformation = 3
	DiagSeverityHint        = 4
)

// Diagnostic is one finding reported by publishDiagnostics.
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity,omitempty"`
	Code     string `json:"code,omitempty"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

// MarkupKind values.
const (
	MarkupPlainText = "plaintext"
	MarkupMarkdown  = "markdown"
)

// MarkupContent is a typed body for hover responses.
type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// Hover is the response shape for textDocument/hover.
type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

// CompletionItem kinds (subset).
const (
	CompletionKindFunction = 3
	CompletionKindVariable = 6
	CompletionKindClass    = 7
	CompletionKindModule   = 9
	CompletionKindKeyword  = 14
)

// CompletionItem is one entry in a CompletionList.
type CompletionItem struct {
	Label         string         `json:"label"`
	Kind          int            `json:"kind,omitempty"`
	Detail        string         `json:"detail,omitempty"`
	Documentation *MarkupContent `json:"documentation,omitempty"`
}

// CompletionList wraps a (possibly partial) list of completion
// items.
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// TextDocumentItem is a freshly-opened document's payload.
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// TextDocumentIdentifier identifies a document by URI.
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// VersionedTextDocumentIdentifier adds a monotonically increasing
// version number to TextDocumentIdentifier.
type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

// TextDocumentContentChangeEvent is one delta in a didChange. We
// only implement the full-replace form (TextDocumentSyncKind.Full),
// so Range is always nil and Text always carries the whole document.
type TextDocumentContentChangeEvent struct {
	Range *Range `json:"range,omitempty"`
	Text  string `json:"text"`
}

// TextDocumentPositionParams is shared by hover, definition, and
// completion request bodies.
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// DidOpenTextDocumentParams matches LSP textDocument/didOpen.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidChangeTextDocumentParams matches LSP textDocument/didChange.
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// DidSaveTextDocumentParams matches LSP textDocument/didSave.
type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         *string                `json:"text,omitempty"`
}

// DidCloseTextDocumentParams matches LSP textDocument/didClose.
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DocumentFormattingParams matches LSP textDocument/formatting.
type DocumentFormattingParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Options      json.RawMessage        `json:"options"`
}

// PublishDiagnosticsParams matches LSP textDocument/publishDiagnostics.
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Version     *int         `json:"version,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// InitializeParams matches LSP initialize (minimal subset).
type InitializeParams struct {
	ProcessID    *int            `json:"processId"`
	RootURI      string          `json:"rootUri,omitempty"`
	Capabilities json.RawMessage `json:"capabilities"`
}

// InitializeResult is the response to initialize.
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   ServerInfo         `json:"serverInfo"`
}

// ServerInfo identifies the server to the client.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerCapabilities advertises which LSP features tya supports.
type ServerCapabilities struct {
	TextDocumentSync           TextDocumentSyncOptions `json:"textDocumentSync"`
	HoverProvider              bool                    `json:"hoverProvider"`
	DefinitionProvider         bool                    `json:"definitionProvider"`
	CompletionProvider         *CompletionOptions      `json:"completionProvider,omitempty"`
	DocumentFormattingProvider bool                    `json:"documentFormattingProvider"`
}

// TextDocumentSyncOptions tells the client which doc-sync mode to
// use. Change == 1 means Full (the whole text is resent on every
// change).
type TextDocumentSyncOptions struct {
	OpenClose bool         `json:"openClose"`
	Change    int          `json:"change"`
	Save      *SaveOptions `json:"save,omitempty"`
}

// SaveOptions controls didSave payload shape.
type SaveOptions struct {
	IncludeText bool `json:"includeText"`
}

// CompletionOptions tells the client how to invoke completion.
type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
	ResolveProvider   bool     `json:"resolveProvider"`
}
