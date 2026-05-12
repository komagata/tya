package main

import (
	"encoding/json"
	"io"
)

// lintFinding is the JSON wire shape emitted by `tya lint --format=json`.
type lintFinding struct {
	Path        string `json:"path"`
	Line        int    `json:"line"`
	Col         int    `json:"col"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	Autofixable bool   `json:"autofixable"`
}

// lintReport is the top-level JSON payload.
type lintReport struct {
	Version  string        `json:"version"`
	Findings []lintFinding `json:"findings"`
}

// writeLintJSON serialises report to w with two-space indenting so
// the output is friendly for human inspection while still being
// machine-readable.
func writeLintJSON(w io.Writer, report lintReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(report)
}
