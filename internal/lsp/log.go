package lsp

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Logger is the minimal logging interface used by the server.
// Implementations must be safe for concurrent use.
type Logger interface {
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}

// NullLogger discards every log line. Used when no `--log` flag
// is given so stderr stays free of noise.
var NullLogger Logger = nullLogger{}

type nullLogger struct{}

func (nullLogger) Infof(string, ...any)  {}
func (nullLogger) Errorf(string, ...any) {}

// NewLogger writes timestamped lines to w. Each line is flushed
// atomically so concurrent goroutines never interleave output.
func NewLogger(w io.Writer) Logger {
	return &fileLogger{w: w}
}

type fileLogger struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *fileLogger) write(level, format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.w, "%s %s ", time.Now().UTC().Format(time.RFC3339Nano), level)
	fmt.Fprintf(l.w, format, args...)
	fmt.Fprintln(l.w)
}

func (l *fileLogger) Infof(format string, args ...any)  { l.write("INFO", format, args...) }
func (l *fileLogger) Errorf(format string, args ...any) { l.write("ERROR", format, args...) }
