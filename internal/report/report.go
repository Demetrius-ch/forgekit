package report

import (
	"fmt"
	"time"
)

// JSONSchemaVersion is the current JSON output schema version.
// Increment when making breaking changes to the JSON output format.
const JSONSchemaVersion = "1"

// Severity levels shared by doctor, analyze and check.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// LegacySeverity maps to PASS/WARNING/ERROR/INFO display labels.
func (s Severity) LegacySeverity() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError, SeverityCritical:
		return "ERROR"
	default:
		return "INFO"
	}
}

// PassLabel returns PASS when severity is info-only success marker.
func PassLabel() string { return "PASS" }

// Finding is a single diagnostic from any ForgeKit command.
type Finding struct {
	ID          string   `json:"id"`
	Name        string   `json:"name,omitempty"`
	Category    string   `json:"category"`
	Severity    Severity `json:"severity"`
	File        string   `json:"file,omitempty"`
	Line        int      `json:"line,omitempty"`
	Column      int      `json:"column,omitempty"`
	Message     string   `json:"message"`
	Explanation string   `json:"explanation,omitempty"`
	Suggestion  string   `json:"suggestion,omitempty"`
}

// Result is the stable JSON schema for machine-readable output.
// Used by analyze, doctor, check commands.
type Result struct {
	SchemaVersion string    `json:"schema_version"`
	Tool          string    `json:"tool"`
	Version       string    `json:"version"`
	Command       string    `json:"command"`
	Project       string    `json:"project,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
	Summary       Summary   `json:"summary"`
	Findings      []Finding `json:"findings"`
}

type Summary struct {
	Pass     int `json:"pass"`
	Info     int `json:"info"`
	Warning  int `json:"warning"`
	Error    int `json:"error"`
	Critical int `json:"critical"`
}

// BuildSummary counts findings by severity.
func BuildSummary(findings []Finding) Summary {
	var s Summary
	for _, f := range findings {
		switch f.Severity {
		case SeverityInfo:
			if f.Category == "pass" {
				s.Pass++
			} else {
				s.Info++
			}
		case SeverityWarning:
			s.Warning++
		case SeverityError:
			s.Error++
		case SeverityCritical:
			s.Critical++
		}
	}
	return s
}

// UniqueFindings removes duplicate findings while preserving order.
func UniqueFindings(findings []Finding) []Finding {
	seen := make(map[string]struct{})
	out := make([]Finding, 0, len(findings))
	for _, f := range findings {
		key := fmt.Sprintf("%s|%s|%s|%s", f.ID, f.Category, f.File, f.Message)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, f)
	}
	return out
}

// HasFailures returns true when errors or critical findings exist.
func HasFailures(s Summary) bool {
	return s.Error > 0 || s.Critical > 0
}
