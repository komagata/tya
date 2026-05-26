package doc

import (
	"encoding/json"
	"io"
	"strings"
)

type jsonReport struct {
	Version     string           `json:"version"`
	Items       []jsonItem       `json:"items"`
	Diagnostics []jsonDiagnostic `json:"diagnostics"`
}

type jsonItem struct {
	Name           string      `json:"name"`
	Kind           string      `json:"kind"`
	Signature      string      `json:"signature"`
	RawDoc         string      `json:"raw_doc"`
	Doc            string      `json:"doc"`
	TypeHint       string      `json:"type,omitempty"`
	Params         []ParamDoc  `json:"params,omitempty"`
	Return         *ReturnDoc  `json:"return,omitempty"`
	Options        []OptionDoc `json:"options,omitempty"`
	Path           string      `json:"path"`
	Line           int         `json:"line"`
	ReexportedFrom string      `json:"reexported_from,omitempty"`
}

type jsonDiagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
}

// FormatJSON writes the stable machine-readable documentation shape
// used by `tya doc --json`.
func FormatJSON(report Report, w io.Writer) error {
	payload := jsonReport{
		Version:     report.Version,
		Items:       make([]jsonItem, 0, len(report.Items)),
		Diagnostics: make([]jsonDiagnostic, 0, len(report.Diagnostics)),
	}
	if payload.Version == "" {
		payload.Version = "1"
	}
	for _, item := range report.Items {
		payload.Items = append(payload.Items, jsonItem{
			Name:           item.Name,
			Kind:           item.Kind,
			Signature:      item.Signature,
			RawDoc:         item.RawDoc,
			Doc:            strings.TrimSpace(RenderText(ParseMarkdown(item.RawDoc))),
			TypeHint:       item.TypeHint,
			Params:         item.Params,
			Return:         item.Return,
			Options:        item.Options,
			Path:           item.FilePath,
			Line:           item.Line,
			ReexportedFrom: item.ReexportedFrom,
		})
	}
	for _, diag := range report.Diagnostics {
		payload.Diagnostics = append(payload.Diagnostics, jsonDiagnostic{
			Code:     diag.Code,
			Severity: diag.Severity,
			Message:  diag.Message,
			Path:     diag.FilePath,
			Line:     diag.Line,
			Col:      diag.Col,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
