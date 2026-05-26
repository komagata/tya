package doc

import (
	"fmt"
	"io"
	"strings"
)

// WriteDiagnostics writes stable diagnostics to w.
func WriteDiagnostics(diags []Diagnostic, w io.Writer) error {
	for _, diag := range diags {
		if _, err := fmt.Fprintf(w, "%s:%d:%d: %s %s: %s\n",
			diag.FilePath, diag.Line, diag.Col, diag.Severity, diag.Code, diag.Message); err != nil {
			return err
		}
	}
	return nil
}

func HasErrorDiagnostics(diags []Diagnostic) bool {
	for _, diag := range diags {
		if diag.Severity == "error" {
			return true
		}
	}
	return false
}

// FormatText writes a plain-text rendering of items to w. The
// layout uses simple `## kind name` headers and indented body
// blocks so the output is readable from a terminal.
func FormatText(items []DocItem, w io.Writer) error {
	if len(items) == 0 {
		_, err := fmt.Fprintln(w, "(no documented bindings)")
		return err
	}
	for i, item := range items {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "## %s %s\n", item.Kind, item.Name); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "    %s\n", item.Signature); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "    %s:%d\n", item.FilePath, item.Line); err != nil {
			return err
		}
		if err := writeTextMetadata(item, w); err != nil {
			return err
		}
		body := strings.TrimSpace(RenderText(ParseMarkdown(item.RawDoc)))
		if body == "" {
			continue
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		for _, line := range strings.Split(body, "\n") {
			if _, err := fmt.Fprintf(w, "    %s\n", line); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeTextMetadata(item DocItem, w io.Writer) error {
	if item.TypeHint != "" {
		if _, err := fmt.Fprintf(w, "    type: %s\n", item.TypeHint); err != nil {
			return err
		}
	}
	for _, param := range item.Params {
		line := fmt.Sprintf("    param %s: %s", param.Name, param.TypeHint)
		if param.Description != "" {
			line += " - " + param.Description
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	for _, option := range item.Options {
		line := fmt.Sprintf("    option %s.%s: %s", option.Param, option.Key, option.TypeHint)
		if option.Description != "" {
			line += " - " + option.Description
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	if item.Return != nil {
		line := fmt.Sprintf("    return: %s", item.Return.TypeHint)
		if item.Return.Description != "" {
			line += " - " + item.Return.Description
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}
