package cover

import (
	"bytes"
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	p := New()
	p.Files = []File{{ID: 0, Path: "src/foo bar.tya"}, {ID: 1, Path: "src/baz%qux.tya"}}
	p.Stmts = []Stmt{{ID: 0, FileID: 0, Line: 3, Col: 1}, {ID: 1, FileID: 0, Line: 4, Col: 1}}
	p.Hits[0] = 7
	p.Hits[1] = 0
	var buf bytes.Buffer
	if err := Write(&buf, p); err != nil {
		t.Fatal(err)
	}
	q, err := Parse(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Files) != 2 || q.Files[0].Path != "src/foo bar.tya" {
		t.Errorf("file roundtrip lost: %+v", q.Files)
	}
	if q.Hits[0] != 7 {
		t.Errorf("hit count lost: %d", q.Hits[0])
	}
	if q.Hits[1] != 0 {
		t.Errorf("zero hit should remain zero, got %d", q.Hits[1])
	}
}

func TestMerge(t *testing.T) {
	a := New()
	a.Files = []File{{ID: 0, Path: "a.tya"}}
	a.Stmts = []Stmt{{ID: 0, FileID: 0, Line: 1, Col: 1}}
	a.Hits[0] = 3
	b := New()
	b.Files = []File{{ID: 0, Path: "a.tya"}, {ID: 1, Path: "b.tya"}}
	b.Stmts = []Stmt{{ID: 0, FileID: 0, Line: 1, Col: 1}, {ID: 1, FileID: 1, Line: 2, Col: 1}}
	b.Hits[0] = 2
	b.Hits[1] = 5
	Merge(a, b)
	if len(a.Files) != 2 {
		t.Fatalf("merge files: %v", a.Files)
	}
	if a.Hits[0] != 5 {
		t.Errorf("merge hits[0]: %d", a.Hits[0])
	}
	if a.Hits[1] != 5 {
		t.Errorf("merge hits[1]: %d", a.Hits[1])
	}
}

func TestRenderText(t *testing.T) {
	p := New()
	p.Files = []File{{ID: 0, Path: "a.tya"}}
	p.Stmts = []Stmt{{ID: 0, FileID: 0, Line: 1}, {ID: 1, FileID: 0, Line: 2}}
	p.Hits[0] = 1
	var buf bytes.Buffer
	if err := RenderText(&buf, Summarize(p)); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{"File", "a.tya", "Total", "50.0%"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestRenderJSON(t *testing.T) {
	p := New()
	p.Files = []File{{ID: 0, Path: "a.tya"}}
	p.Stmts = []Stmt{{ID: 0, FileID: 0, Line: 1}}
	p.Hits[0] = 1
	var buf bytes.Buffer
	if err := RenderJSON(&buf, p, ".tya/coverage/profile", "0.30.0"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{`"tool": "tya"`, `"version": "0.30.0"`, `"path": "a.tya"`, `"hits": 1`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestParseEmpty(t *testing.T) {
	if _, err := Parse(strings.NewReader("")); err == nil {
		t.Error("empty input should error")
	}
	if _, err := Parse(strings.NewReader("not a header\n")); err == nil {
		t.Error("invalid header should error")
	}
}
