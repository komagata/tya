// Package lsp implements the v0.52 `tya lsp` Language Server. It
// speaks JSON-RPC 2.0 over stdio per the Language Server Protocol
// (LSP). Only the request/response/notification framing lives in
// this file; protocol payload shapes are in protocol.go and the
// server loop is in server.go.
package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// JSON-RPC standard error codes.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603

	// LSP-specific error codes.
	codeServerNotInitialized = -32002
	codeRequestCancelled     = -32800
)

// Message is the on-the-wire shape of any JSON-RPC envelope.
// The presence of ID distinguishes a Request (has ID) from a
// Notification (no ID). The presence of Method distinguishes a
// Request/Notification from a Response.
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is the standard JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Conn is a half-duplex JSON-RPC connection sitting on top of a
// pair of stdio streams. Writes are serialised through a mutex so
// notifications generated from worker goroutines cannot interleave
// half-messages with responses.
type Conn struct {
	r   *bufio.Reader
	w   io.Writer
	mu  sync.Mutex
	log Logger
}

// NewConn wraps in/out as a JSON-RPC connection.
func NewConn(in io.Reader, out io.Writer, log Logger) *Conn {
	if log == nil {
		log = NullLogger
	}
	return &Conn{r: bufio.NewReader(in), w: out, log: log}
}

// Read blocks until a full JSON-RPC message is read or in is
// closed. The returned Message has Params/Result/Error populated
// as appropriate.
func (c *Conn) Read() (*Message, error) {
	length, err := readContentLength(c.r)
	if err != nil {
		return nil, err
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(c.r, body); err != nil {
		return nil, err
	}
	var m Message
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("decode JSON-RPC body: %w", err)
	}
	return &m, nil
}

// WriteResponse sends a successful Response with the given id and
// result payload.
func (c *Conn) WriteResponse(id *json.RawMessage, result any) error {
	b, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return c.writeEnvelope(Message{JSONRPC: "2.0", ID: id, Result: b})
}

// WriteError sends a Response whose Error field is populated.
func (c *Conn) WriteError(id *json.RawMessage, code int, msg string) error {
	return c.writeEnvelope(Message{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: msg},
	})
}

// WriteNotification sends a server-initiated notification.
func (c *Conn) WriteNotification(method string, params any) error {
	b, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return c.writeEnvelope(Message{JSONRPC: "2.0", Method: method, Params: b})
}

func (c *Conn) writeEnvelope(m Message) error {
	body, err := json.Marshal(m)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := fmt.Fprintf(c.w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = c.w.Write(body)
	return err
}

// readContentLength consumes header lines from r until the blank
// line that terminates the header block, returning the announced
// Content-Length.
func readContentLength(r *bufio.Reader) (int, error) {
	length := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return 0, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if length < 0 {
				return 0, fmt.Errorf("missing Content-Length header")
			}
			return length, nil
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		name := strings.TrimSpace(line[:colon])
		value := strings.TrimSpace(line[colon+1:])
		if strings.EqualFold(name, "Content-Length") {
			n, err := strconv.Atoi(value)
			if err != nil {
				return 0, fmt.Errorf("bad Content-Length %q: %w", value, err)
			}
			length = n
		}
	}
}
