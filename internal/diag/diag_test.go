package diag

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderHumanBasic(t *testing.T) {
	sm := NewSourceMap()
	sm.Add("main.tya", []byte("a = 1\nfor a in [1]\n  print a\n"))
	d := Diagnostic{
		Severity: Error,
		Code:     "TYA-E0301",
		Title:    "Shadowed binding",
		Message:  "This binding shadows the outer name `a`.",
		Primary:  Region{File: "main.tya", Start: Pos{Line: 2, Col: 5}, End: Pos{Line: 2, Col: 6}},
		Hints:    []string{"Rename the inner binding."},
		Source:   "checker",
	}
	out := Render([]Diagnostic{d}, sm, RenderOptions{Color: ColorNever})
	for _, want := range []string{
		"-- SHADOWED BINDING ",
		"main.tya:2:5",
		"This binding shadows",
		"   2 | for a in [1]",
		"^",
		"Hint: Rename the inner binding.",
		"(TYA-E0301)",
		"Found 1 error(s), 0 warning(s).",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
}

func TestRenderJSONShape(t *testing.T) {
	d := Diagnostic{
		Severity: Error,
		Code:     "TYA-E0302",
		Title:    "Unused import",
		Message:  "msg",
		Primary:  Region{File: "f.tya", Start: Pos{Line: 1, Col: 8}, End: Pos{Line: 1, Col: 14}},
		Source:   "checker",
	}
	out := RenderJSON([]Diagnostic{d})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines (diag + summary), got %d:\n%s", len(lines), out)
	}
	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("first line is not JSON: %v", err)
	}
	if first["severity"] != "error" || first["code"] != "TYA-E0302" {
		t.Errorf("unexpected first line: %v", first)
	}
	var summary map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &summary); err != nil {
		t.Fatalf("summary is not JSON: %v", err)
	}
	s := summary["summary"].(map[string]any)
	if s["errors"].(float64) != 1 || s["warnings"].(float64) != 0 {
		t.Errorf("unexpected summary: %v", s)
	}
}

func TestColorMode(t *testing.T) {
	o := RenderOptions{Color: ColorAuto, IsTTY: true, NoColor: false}
	if !o.useColor() {
		t.Error("auto + tty should use color")
	}
	o.NoColor = true
	if o.useColor() {
		t.Error("NO_COLOR should disable color in auto mode")
	}
	o = RenderOptions{Color: ColorAlways, NoColor: true}
	if !o.useColor() {
		t.Error("explicit always should override NO_COLOR")
	}
	o = RenderOptions{Color: ColorNever, IsTTY: true}
	if o.useColor() {
		t.Error("never should disable color")
	}
}

func TestParseFlags(t *testing.T) {
	if c, err := ParseColorMode("always"); err != nil || c != ColorAlways {
		t.Errorf("color always: %v %v", c, err)
	}
	if _, err := ParseColorMode("rainbow"); err == nil {
		t.Error("invalid color should error")
	}
	if f, err := ParseFormat("json"); err != nil || f != FormatJSON {
		t.Errorf("format json: %v %v", f, err)
	}
}

func TestSnippetUnavailable(t *testing.T) {
	d := Diagnostic{
		Severity: Error, Code: "TYA-E0301", Title: "X", Message: "m",
		Primary: Region{File: "missing.tya", Start: Pos{Line: 1, Col: 1}, End: Pos{Line: 1, Col: 2}},
	}
	out := Render([]Diagnostic{d}, NewSourceMap(), RenderOptions{Color: ColorNever})
	if !strings.Contains(out, "(snippet unavailable)") {
		t.Errorf("missing placeholder:\n%s", out)
	}
}
