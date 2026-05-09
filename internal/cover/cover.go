// Package cover implements the v0.30 coverage profile format and
// reporting. The profile is a small text format describing
// per-statement counter hits keyed by Tya source position.
package cover

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

const formatVersion = 1
const Header = "# tya-cover 1"

// File describes one source file in the profile.
type File struct {
	ID   int
	Path string
}

// Stmt describes one statement counter.
type Stmt struct {
	ID     int
	FileID int
	Line   int
	Col    int
}

// Profile is an in-memory representation of a coverage profile.
type Profile struct {
	Files []File
	Stmts []Stmt
	Hits  map[int]int // stmt id -> count
}

// New returns an empty profile with an initialized hits map.
func New() *Profile {
	return &Profile{Hits: map[int]int{}}
}

// Parse reads a profile from r.
func Parse(r io.Reader) (*Profile, error) {
	p := New()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	first := true
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimRight(sc.Text(), "\r")
		if first {
			first = false
			if line != Header {
				return nil, fmt.Errorf("invalid header on line %d: %q (want %q)", lineNo, line, Header)
			}
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "F":
			if len(fields) < 3 {
				return nil, fmt.Errorf("malformed F record on line %d", lineNo)
			}
			id, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, fmt.Errorf("F record id on line %d: %w", lineNo, err)
			}
			path := decodePath(strings.Join(fields[2:], " "))
			p.Files = append(p.Files, File{ID: id, Path: path})
		case "S":
			if len(fields) != 5 {
				return nil, fmt.Errorf("malformed S record on line %d", lineNo)
			}
			id, e1 := strconv.Atoi(fields[1])
			fid, e2 := strconv.Atoi(fields[2])
			ln, e3 := strconv.Atoi(fields[3])
			col, e4 := strconv.Atoi(fields[4])
			if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
				return nil, fmt.Errorf("S record fields on line %d", lineNo)
			}
			p.Stmts = append(p.Stmts, Stmt{ID: id, FileID: fid, Line: ln, Col: col})
		case "H":
			if len(fields) != 3 {
				return nil, fmt.Errorf("malformed H record on line %d", lineNo)
			}
			id, e1 := strconv.Atoi(fields[1])
			n, e2 := strconv.Atoi(fields[2])
			if e1 != nil || e2 != nil {
				return nil, fmt.Errorf("H record fields on line %d", lineNo)
			}
			p.Hits[id] += n
		default:
			return nil, fmt.Errorf("unknown record %q on line %d", fields[0], lineNo)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if first {
		return nil, fmt.Errorf("empty profile (missing header)")
	}
	return p, nil
}

// Write serializes p to w.
func Write(w io.Writer, p *Profile) error {
	bw := bufio.NewWriter(w)
	fmt.Fprintln(bw, Header)
	files := append([]File(nil), p.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].ID < files[j].ID })
	for _, f := range files {
		fmt.Fprintf(bw, "F %d %s\n", f.ID, encodePath(f.Path))
	}
	stmts := append([]Stmt(nil), p.Stmts...)
	sort.Slice(stmts, func(i, j int) bool { return stmts[i].ID < stmts[j].ID })
	for _, s := range stmts {
		fmt.Fprintf(bw, "S %d %d %d %d\n", s.ID, s.FileID, s.Line, s.Col)
	}
	ids := make([]int, 0, len(p.Hits))
	for id := range p.Hits {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		if p.Hits[id] == 0 {
			continue
		}
		fmt.Fprintf(bw, "H %d %d\n", id, p.Hits[id])
	}
	return bw.Flush()
}

// Merge sums hits and unions records from b into a.
func Merge(a, b *Profile) {
	known := map[int]bool{}
	for _, f := range a.Files {
		known[f.ID] = true
	}
	for _, f := range b.Files {
		if !known[f.ID] {
			a.Files = append(a.Files, f)
			known[f.ID] = true
		}
	}
	knownStmt := map[int]bool{}
	for _, s := range a.Stmts {
		knownStmt[s.ID] = true
	}
	for _, s := range b.Stmts {
		if !knownStmt[s.ID] {
			a.Stmts = append(a.Stmts, s)
			knownStmt[s.ID] = true
		}
	}
	for id, n := range b.Hits {
		a.Hits[id] += n
	}
}

func encodePath(p string) string {
	var b strings.Builder
	for i := 0; i < len(p); i++ {
		c := p[i]
		switch c {
		case ' ':
			b.WriteString("%20")
		case '%':
			b.WriteString("%25")
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func decodePath(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			h1 := s[i+1]
			h2 := s[i+2]
			if (h1 == '2' && h2 == '0') || (h1 == '2' && h2 == '5') {
				if h2 == '0' {
					b.WriteByte(' ')
				} else {
					b.WriteByte('%')
				}
				i += 2
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// FileSummary is a per-file roll-up used for reports.
type FileSummary struct {
	Path       string
	Statements int
	Hits       int
	Missed     int
	Lines      []LineHit
}

// LineHit is a per-line roll-up.
type LineHit struct {
	Line      int
	Hits      int
	Coverable bool
}

// Coverage returns Hits/Statements as a percentage. Returns 0 when no
// statements.
func (s FileSummary) Coverage() float64 {
	if s.Statements == 0 {
		return 0
	}
	return float64(s.Hits) / float64(s.Statements) * 100
}

// Summarize returns per-file summaries sorted by path.
func Summarize(p *Profile) []FileSummary {
	byFile := map[int]*FileSummary{}
	pathByID := map[int]string{}
	for _, f := range p.Files {
		pathByID[f.ID] = f.Path
		byFile[f.ID] = &FileSummary{Path: f.Path}
	}
	lineSeen := map[int]map[int]int{} // file_id -> line -> hits
	for _, s := range p.Stmts {
		fs := byFile[s.FileID]
		if fs == nil {
			continue
		}
		fs.Statements++
		hits := p.Hits[s.ID]
		if hits > 0 {
			fs.Hits++
		} else {
			fs.Missed++
		}
		if _, ok := lineSeen[s.FileID]; !ok {
			lineSeen[s.FileID] = map[int]int{}
		}
		lineSeen[s.FileID][s.Line] += hits
	}
	out := make([]FileSummary, 0, len(byFile))
	for id, fs := range byFile {
		lines := []LineHit{}
		linesByNum := lineSeen[id]
		nums := make([]int, 0, len(linesByNum))
		for n := range linesByNum {
			nums = append(nums, n)
		}
		sort.Ints(nums)
		for _, n := range nums {
			lines = append(lines, LineHit{Line: n, Hits: linesByNum[n], Coverable: true})
		}
		fs.Lines = lines
		out = append(out, *fs)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// RenderText writes the table format to w.
func RenderText(w io.Writer, summaries []FileSummary) error {
	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, "%-30s %5s %5s %7s %9s\n", "File", "Stmts", "Hit", "Missed", "Coverage")
	totalStmts, totalHits, totalMissed := 0, 0, 0
	for _, s := range summaries {
		fmt.Fprintf(bw, "%-30s %5d %5d %7d %8.1f%%\n",
			s.Path, s.Statements, s.Hits, s.Missed, s.Coverage())
		totalStmts += s.Statements
		totalHits += s.Hits
		totalMissed += s.Missed
	}
	fmt.Fprintln(bw, strings.Repeat("-", 60))
	pct := 0.0
	if totalStmts > 0 {
		pct = float64(totalHits) / float64(totalStmts) * 100
	}
	fmt.Fprintf(bw, "%-30s %5d %5d %7d %8.1f%%\n", "Total", totalStmts, totalHits, totalMissed, pct)
	return bw.Flush()
}

// RenderJSON writes the v0.30 JSON shape to w.
func RenderJSON(w io.Writer, p *Profile, profilePath, toolVersion string) error {
	type lineWire struct {
		Line      int  `json:"line"`
		Hits      int  `json:"hits"`
		Coverable bool `json:"coverable"`
	}
	type fileWire struct {
		Path       string     `json:"path"`
		Statements int        `json:"statements"`
		Hits       int        `json:"hits"`
		Lines      []lineWire `json:"lines"`
	}
	type totalsWire struct {
		Statements int `json:"statements"`
		Hits       int `json:"hits"`
		Files      int `json:"files"`
	}
	type doc struct {
		Tool    string     `json:"tool"`
		Version string     `json:"version"`
		Format  int        `json:"format"`
		Profile string     `json:"profile"`
		Files   []fileWire `json:"files"`
		Totals  totalsWire `json:"totals"`
	}
	summaries := Summarize(p)
	files := make([]fileWire, 0, len(summaries))
	tStmts, tHits := 0, 0
	for _, s := range summaries {
		lines := make([]lineWire, 0, len(s.Lines))
		for _, l := range s.Lines {
			lines = append(lines, lineWire{Line: l.Line, Hits: l.Hits, Coverable: l.Coverable})
		}
		files = append(files, fileWire{Path: s.Path, Statements: s.Statements, Hits: s.Hits, Lines: lines})
		tStmts += s.Statements
		tHits += s.Hits
	}
	out := doc{
		Tool:    "tya",
		Version: toolVersion,
		Format:  formatVersion,
		Profile: profilePath,
		Files:   files,
		Totals:  totalsWire{Statements: tStmts, Hits: tHits, Files: len(files)},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// ReadProfile is a convenience that opens path and parses it.
func ReadProfile(path string) (*Profile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// WriteProfile writes p to path, creating directories as needed.
func WriteProfile(path string, p *Profile) error {
	if dir := dirOf(path); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return Write(f, p)
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == os.PathSeparator {
			return path[:i]
		}
	}
	return ""
}
