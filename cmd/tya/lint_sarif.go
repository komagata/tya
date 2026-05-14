package main

import (
	"encoding/json"
	"io"
	"sort"
)

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	ShortDescription sarifMessage      `json:"shortDescription"`
	HelpURI          string            `json:"helpUri"`
	Properties       sarifRuleProperty `json:"properties"`
}

type sarifRuleProperty struct {
	Tags []string `json:"tags"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
}

func writeLintSARIF(w io.Writer, findings []lintFinding) error {
	rulesByID := map[string]sarifRule{}
	results := make([]sarifResult, 0, len(findings))
	for _, f := range findings {
		info := lintRuleMetadata(f.Code)
		if _, ok := rulesByID[f.Code]; !ok {
			rulesByID[f.Code] = sarifRule{
				ID:               f.Code,
				Name:             info.Title,
				ShortDescription: sarifMessage{Text: info.Title},
				HelpURI:          info.DocURL,
				Properties:       sarifRuleProperty{Tags: []string{"tya", "lint"}},
			}
		}
		line := f.Line
		if line < 1 {
			line = 1
		}
		col := f.Col
		if col < 1 {
			col = 1
		}
		results = append(results, sarifResult{
			RuleID:  f.Code,
			Level:   "warning",
			Message: sarifMessage{Text: f.Message},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{URI: f.Path},
					Region:           sarifRegion{StartLine: line, StartColumn: col},
				},
			}},
		})
	}
	rules := make([]sarifRule, 0, len(rulesByID))
	for _, r := range rulesByID {
		rules = append(rules, r)
	}
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].ID < rules[j].ID
	})
	log := sarifLog{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           "tya lint",
				InformationURI: "https://tya-lang.org/lint.html",
				Rules:          rules,
			}},
			Results: results,
		}},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(log)
}
