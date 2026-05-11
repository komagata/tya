package tests

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// lspBinary is the path to a pre-built `tya` binary that the LSP
// tests share. Set in TestMain (see lsp_test.go) so each test
// avoids the ~3s compile cost.
var (
	lspBinaryOnce sync.Once
	lspBinary     string
)

func ensureLSPBinary(t *testing.T) string {
	t.Helper()
	lspBinaryOnce.Do(func() {
		repo, err := filepath.Abs("..")
		if err != nil {
			t.Fatal(err)
		}
		dir, err := os.MkdirTemp("", "tya-lsp-bin-")
		if err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(dir, "tya-lsp")
		cmd := exec.Command("go", "build", "-o", out, "./cmd/tya")
		cmd.Dir = repo
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build tya: %v\n%s", err, b)
		}
		lspBinary = out
	})
	return lspBinary
}

// lspProc is a running `tya lsp` subprocess driven by JSON-RPC.
type lspProc struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	nextID int
	t      *testing.T
}

func startLSP(t *testing.T) *lspProc {
	t.Helper()
	bin := ensureLSPBinary(t)
	cmd := exec.Command(bin, "lsp")
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return &lspProc{cmd: cmd, stdin: in, stdout: bufio.NewReader(out), t: t}
}

// request sends a request and reads exactly one matching response.
func (p *lspProc) request(method string, params any) json.RawMessage {
	p.t.Helper()
	p.nextID++
	id := p.nextID
	p.writeMessage(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	})
	for {
		m := p.readMessage()
		if m["id"] == nil {
			continue
		}
		gotID, _ := strconv.Atoi(fmt.Sprintf("%v", m["id"]))
		if gotID != id {
			continue
		}
		if errVal, ok := m["error"]; ok && errVal != nil {
			p.t.Fatalf("request %s error: %v", method, errVal)
		}
		b, _ := json.Marshal(m["result"])
		return b
	}
}

// notify sends a server-bound notification (no response expected).
func (p *lspProc) notify(method string, params any) {
	p.t.Helper()
	p.writeMessage(map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	})
}

// expectNotification consumes messages until it finds the named
// notification and returns its params. Times out after 2 seconds.
func (p *lspProc) expectNotification(method string) json.RawMessage {
	p.t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		m := p.readMessage()
		if m["method"] == method {
			b, _ := json.Marshal(m["params"])
			return b
		}
	}
	p.t.Fatalf("did not see notification %q within 2s", method)
	return nil
}

func (p *lspProc) close() {
	_ = p.stdin.Close()
	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = p.cmd.Process.Kill()
	}
}

func (p *lspProc) writeMessage(payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		p.t.Fatal(err)
	}
	if _, err := fmt.Fprintf(p.stdin, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		p.t.Fatal(err)
	}
	if _, err := p.stdin.Write(body); err != nil {
		p.t.Fatal(err)
	}
}

func (p *lspProc) readMessage() map[string]any {
	p.t.Helper()
	length := -1
	for {
		line, err := p.stdout.ReadString('\n')
		if err != nil {
			p.t.Fatalf("read header: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if i := strings.IndexByte(line, ':'); i >= 0 {
			name := strings.TrimSpace(line[:i])
			value := strings.TrimSpace(line[i+1:])
			if strings.EqualFold(name, "Content-Length") {
				length, _ = strconv.Atoi(value)
			}
		}
	}
	if length < 0 {
		p.t.Fatal("no Content-Length header before body")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(p.stdout, body); err != nil {
		p.t.Fatalf("read body: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		p.t.Fatalf("decode body: %v\n%s", err, body)
	}
	return m
}

// fileURI is a small wrapper used by tests to manufacture file:// URIs.
func fileURI(path string) string {
	abs, _ := filepath.Abs(path)
	return "file://" + filepath.ToSlash(abs)
}
