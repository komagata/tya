package lsp

import "sync"

// Document is one open editor buffer.
type Document struct {
	URI     string
	Path    string
	Version int
	Text    string
}

// Store is the URI-keyed map of open documents.
type Store struct {
	mu   sync.RWMutex
	docs map[string]*Document
}

// NewStore returns an empty document store.
func NewStore() *Store {
	return &Store{docs: map[string]*Document{}}
}

// Open inserts a document. Existing entries are overwritten.
func (s *Store) Open(uri string, version int, text string) {
	path, _ := URIToPath(uri)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[uri] = &Document{URI: uri, Path: path, Version: version, Text: text}
}

// Change replaces the document's text. Older versions are
// silently dropped (LSP guarantees monotonic Version per URI but
// some clients reorder; pick the latest by version).
func (s *Store) Change(uri string, version int, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.docs[uri]
	if !ok {
		path, _ := URIToPath(uri)
		s.docs[uri] = &Document{URI: uri, Path: path, Version: version, Text: text}
		return
	}
	if version < d.Version {
		return
	}
	d.Version = version
	d.Text = text
}

// Save records the new text (if the client opted into IncludeText)
// and otherwise updates nothing. Save also serves as a synthetic
// trigger for re-diagnostics in callers; this layer just stores.
func (s *Store) Save(uri string, text *string) {
	if text == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.docs[uri]; ok {
		d.Text = *text
	}
}

// Close drops the document.
func (s *Store) Close(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.docs, uri)
}

// Get returns the document and true if uri is open.
func (s *Store) Get(uri string) (*Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.docs[uri]
	return d, ok
}
