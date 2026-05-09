// Package diag provides the shared diagnostic model and renderers used
// across the Tya toolchain. v0.29 introduces the model and ships the
// checker's strict diagnostics through it; later releases will migrate
// the remaining stages.
package diag

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Severity int

const (
	Error Severity = iota
	Warning
)

func (s Severity) String() string {
	if s == Warning {
		return "warning"
	}
	return "error"
}

type Pos struct {
	Line int `json:"line"`
	Col  int `json:"col"`
}

type Region struct {
	File  string `json:"file"`
	Start Pos    `json:"start"`
	End   Pos    `json:"end"`
}

type Diagnostic struct {
	Severity Severity `json:"-"`
	Code     string   `json:"code"`
	Title    string   `json:"title"`
	Message  string   `json:"message"`
	Primary  Region   `json:"primary"`
	Hints    []string `json:"hints"`
	Notes    []string `json:"notes"`
	Source   string   `json:"source"`
}

// SourceMap caches file bytes by path so renderers can draw snippets
// without each stage re-reading files.
type SourceMap struct {
	files map[string][]string
}

func NewSourceMap() *SourceMap {
	return &SourceMap{files: map[string][]string{}}
}

// Add registers src under file. Newlines are normalized to \n.
func (sm *SourceMap) Add(file string, src []byte) {
	text := strings.ReplaceAll(string(src), "\r\n", "\n")
	sm.files[file] = strings.Split(text, "\n")
}

// AddFromDisk reads file off disk and registers it.
func (sm *SourceMap) AddFromDisk(file string) error {
	if _, ok := sm.files[file]; ok {
		return nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	sm.Add(file, data)
	return nil
}

// Line returns the 1-indexed line of file, or "", false when unavailable.
func (sm *SourceMap) Line(file string, n int) (string, bool) {
	if sm == nil {
		return "", false
	}
	lines, ok := sm.files[file]
	if !ok || n < 1 || n > len(lines) {
		return "", false
	}
	return lines[n-1], true
}

type ColorMode int

const (
	ColorAuto ColorMode = iota
	ColorAlways
	ColorNever
)

type Format int

const (
	FormatHuman Format = iota
	FormatJSON
)

type RenderOptions struct {
	Color     ColorMode
	IsTTY     bool
	NoColor   bool // typically derived from os.Getenv("NO_COLOR")
	TermWidth int
}

func (o RenderOptions) useColor() bool {
	switch o.Color {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	}
	if o.NoColor {
		return false
	}
	return o.IsTTY
}

// ParseColorMode resolves --color flag values.
func ParseColorMode(s string) (ColorMode, error) {
	switch s {
	case "", "auto":
		return ColorAuto, nil
	case "always":
		return ColorAlways, nil
	case "never":
		return ColorNever, nil
	}
	return ColorAuto, fmt.Errorf("invalid --color value %q (want auto|always|never)", s)
}

// ParseFormat resolves --format flag values.
func ParseFormat(s string) (Format, error) {
	switch s {
	case "", "human":
		return FormatHuman, nil
	case "json":
		return FormatJSON, nil
	}
	return FormatHuman, fmt.Errorf("invalid --format value %q (want human|json)", s)
}

// ANSI codes.
const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiRed    = "\x1b[31m"
	ansiYellow = "\x1b[33m"
	ansiBlue   = "\x1b[34m"
	ansiCyan   = "\x1b[36m"
)

// Render produces the human-readable string for diags.
func Render(diags []Diagnostic, sm *SourceMap, opts RenderOptions) string {
	var b strings.Builder
	color := opts.useColor()
	for i, d := range diags {
		if i > 0 {
			b.WriteString("\n")
		}
		renderOne(&b, d, sm, color)
	}
	if len(diags) > 0 {
		errs, warns := 0, 0
		for _, d := range diags {
			if d.Severity == Warning {
				warns++
			} else {
				errs++
			}
		}
		b.WriteString("\n")
		fmt.Fprintf(&b, "Found %d error(s), %d warning(s).\n", errs, warns)
	}
	return b.String()
}

func renderOne(b *strings.Builder, d Diagnostic, sm *SourceMap, color bool) {
	sevColor := ansiRed
	if d.Severity == Warning {
		sevColor = ansiYellow
	}

	// Banner. "-- TITLE <dashes> file:line:col" padded to col 70.
	title := strings.ToUpper(d.Title)
	loc := fmt.Sprintf("%s:%d:%d", d.Primary.File, d.Primary.Start.Line, d.Primary.Start.Col)
	const target = 70
	dashes := target - 3 - len(title) - 1 - 1 - len(loc)
	if dashes < 3 {
		dashes = 3
	}
	dashStr := strings.Repeat("-", dashes)
	if color {
		fmt.Fprintf(b, "%s%s-- %s %s%s %s%s%s\n",
			ansiBold, sevColor, title, dashStr, ansiReset,
			ansiCyan, loc, ansiReset)
	} else {
		fmt.Fprintf(b, "-- %s %s %s\n", title, dashStr, loc)
	}
	b.WriteString("\n")
	b.WriteString(d.Message)
	b.WriteString("\n\n")

	// Snippet.
	if line, ok := sm.Line(d.Primary.File, d.Primary.Start.Line); ok {
		gutter := fmt.Sprintf("%4d", d.Primary.Start.Line)
		fmt.Fprintf(b, "%s | %s\n", gutter, line)
		// Underline row.
		spaces := strings.Repeat(" ", len(gutter))
		startCol := d.Primary.Start.Col
		endCol := d.Primary.End.Col
		if endCol <= startCol {
			endCol = startCol + 1
		}
		// Underline width covers [startCol, endCol).
		under := strings.Repeat(" ", startCol-1) + strings.Repeat("^", endCol-startCol)
		if color {
			fmt.Fprintf(b, "%s   %s%s%s%s\n", spaces, ansiBold, sevColor, under, ansiReset)
		} else {
			fmt.Fprintf(b, "%s   %s\n", spaces, under)
		}
	} else {
		b.WriteString("(snippet unavailable)\n")
	}

	for _, h := range d.Hints {
		b.WriteString("\n")
		if color {
			fmt.Fprintf(b, "%s%sHint:%s %s\n", ansiBold, ansiBlue, ansiReset, h)
		} else {
			fmt.Fprintf(b, "Hint: %s\n", h)
		}
	}
	for _, note := range d.Notes {
		b.WriteString("\n")
		if color {
			fmt.Fprintf(b, "%s%sNote:%s %s\n", ansiBold, ansiDim, ansiReset, note)
		} else {
			fmt.Fprintf(b, "Note: %s\n", note)
		}
	}

	b.WriteString("\n")
	if color {
		fmt.Fprintf(b, "%s(%s)%s\n", ansiDim, d.Code, ansiReset)
	} else {
		fmt.Fprintf(b, "(%s)\n", d.Code)
	}
}

// RenderJSON produces NDJSON of diagnostics followed by a summary object.
func RenderJSON(diags []Diagnostic) string {
	var b strings.Builder
	type wire struct {
		Severity string   `json:"severity"`
		Code     string   `json:"code"`
		Title    string   `json:"title"`
		Message  string   `json:"message"`
		Primary  Region   `json:"primary"`
		Hints    []string `json:"hints"`
		Notes    []string `json:"notes"`
		Source   string   `json:"source"`
	}
	for _, d := range diags {
		w := wire{
			Severity: d.Severity.String(),
			Code:     d.Code,
			Title:    d.Title,
			Message:  d.Message,
			Primary:  d.Primary,
			Hints:    nilToEmpty(d.Hints),
			Notes:    nilToEmpty(d.Notes),
			Source:   d.Source,
		}
		buf, _ := json.Marshal(w)
		b.Write(buf)
		b.WriteString("\n")
	}
	type summary struct {
		Errors   int `json:"errors"`
		Warnings int `json:"warnings"`
	}
	type sumWire struct {
		Summary summary `json:"summary"`
	}
	errs, warns := 0, 0
	for _, d := range diags {
		if d.Severity == Warning {
			warns++
		} else {
			errs++
		}
	}
	buf, _ := json.Marshal(sumWire{Summary: summary{Errors: errs, Warnings: warns}})
	b.Write(buf)
	b.WriteString("\n")
	return b.String()
}

func nilToEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// SortByPos sorts diagnostics by file, then start line, then start col.
// Useful for deterministic multi-error output.
func SortByPos(diags []Diagnostic) {
	sort.SliceStable(diags, func(i, j int) bool {
		a, b := diags[i].Primary, diags[j].Primary
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Start.Line != b.Start.Line {
			return a.Start.Line < b.Start.Line
		}
		return a.Start.Col < b.Start.Col
	})
}

// Errors filters error-severity diagnostics.
func Errors(diags []Diagnostic) []Diagnostic {
	out := diags[:0:0]
	for _, d := range diags {
		if d.Severity == Error {
			out = append(out, d)
		}
	}
	return out
}
