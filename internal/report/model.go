// Package report defines the versioned scan report model and its renderers.
package report

import (
	"fmt"
	"sort"
	"time"
)

// SchemaVersion identifies the JSON report contract. Additive, optional
// fields keep the version; renamed, removed or re-typed fields bump it.
const SchemaVersion = "1"

// Severity is how much a confirmed issue would matter.
type Severity string

const (
	SeverityInfo   Severity = "info"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

// Confidence is how strong the evidence behind a finding is.
type Confidence string

const (
	ConfidenceLow    Confidence = "low"
	ConfidenceMedium Confidence = "medium"
	ConfidenceHigh   Confidence = "high"
)

// Severities lists every valid severity, lowest first.
var Severities = []Severity{SeverityInfo, SeverityMedium, SeverityHigh}

func (s Severity) rank() int {
	for i, v := range Severities {
		if v == s {
			return i
		}
	}
	return -1
}

// Valid reports whether s is a known severity.
func (s Severity) Valid() bool { return s.rank() >= 0 }

// Valid reports whether c is a known confidence level.
func (c Confidence) Valid() bool {
	switch c {
	case ConfidenceLow, ConfidenceMedium, ConfidenceHigh:
		return true
	}
	return false
}

// Finding is one compatibility signal produced by a rule.
type Finding struct {
	RuleID      string     `json:"rule_id"`
	Severity    Severity   `json:"severity"`
	Confidence  Confidence `json:"confidence"`
	Path        string     `json:"path"`
	Line        int        `json:"line,omitempty"`
	Message     string     `json:"message"`
	Evidence    string     `json:"evidence"`
	Remediation string     `json:"remediation"`
	References  []string   `json:"references"`
}

// Validate rejects findings a rule should never emit.
func (f Finding) Validate() error {
	switch {
	case f.RuleID == "":
		return fmt.Errorf("finding has no rule_id")
	case !f.Severity.Valid():
		return fmt.Errorf("finding %s: invalid severity %q", f.RuleID, f.Severity)
	case !f.Confidence.Valid():
		return fmt.Errorf("finding %s: invalid confidence %q", f.RuleID, f.Confidence)
	case f.Path == "":
		return fmt.Errorf("finding %s: no path", f.RuleID)
	case f.Line < 0:
		return fmt.Errorf("finding %s: negative line %d", f.RuleID, f.Line)
	case f.Message == "":
		return fmt.Errorf("finding %s: no message", f.RuleID)
	}
	return nil
}

// Inventory categories.
const (
	CategoryManifest = "manifest"
	CategorySource   = "source"
)

// InventoryItem is a discovered file relevant to Protocol 30 checks.
type InventoryItem struct {
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	Category string `json:"category"`
}

// SkippedEntry is a filesystem entry the walker deliberately did not inspect.
type SkippedEntry struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// Warning is a scan-level notice that is not a finding, such as a check
// that does not exist yet for a discovered manifest.
type Warning struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

// Summary holds counts derived from the rest of the report.
type Summary struct {
	ManifestFiles      int              `json:"manifest_files"`
	SourceFiles        int              `json:"source_files"`
	SkippedEntries     int              `json:"skipped_entries"`
	RulesExecuted      int              `json:"rules_executed"`
	Findings           int              `json:"findings"`
	FindingsBySeverity map[Severity]int `json:"findings_by_severity"`
	Warnings           int              `json:"warnings"`
}

// ScanReport is the root of every rendered report.
type ScanReport struct {
	SchemaVersion string          `json:"schema_version"`
	ToolVersion   string          `json:"tool_version"`
	Target        string          `json:"target"`
	CheckedAtUTC  string          `json:"checked_at_utc"`
	Findings      []Finding       `json:"findings"`
	Summary       Summary         `json:"summary"`
	Inventory     []InventoryItem `json:"inventory"`
	Skipped       []SkippedEntry  `json:"skipped"`
	Warnings      []Warning       `json:"warnings"`
	Limitations   []string        `json:"limitations"`
}

// New builds a report with sorted contents and a computed summary. Nil
// slices become empty so JSON output always carries arrays.
func New(toolVersion, target string, checkedAt time.Time, rulesExecuted int,
	findings []Finding, inventory []InventoryItem, skipped []SkippedEntry,
	warnings []Warning, limitations []string) ScanReport {
	r := ScanReport{
		SchemaVersion: SchemaVersion,
		ToolVersion:   toolVersion,
		Target:        target,
		CheckedAtUTC:  checkedAt.UTC().Format(time.RFC3339),
		Findings:      append([]Finding{}, findings...),
		Inventory:     append([]InventoryItem{}, inventory...),
		Skipped:       append([]SkippedEntry{}, skipped...),
		Warnings:      append([]Warning{}, warnings...),
		Limitations:   append([]string{}, limitations...),
	}
	for i := range r.Findings {
		if r.Findings[i].References == nil {
			r.Findings[i].References = []string{}
		}
	}
	SortFindings(r.Findings)
	sort.Slice(r.Inventory, func(i, j int) bool { return r.Inventory[i].Path < r.Inventory[j].Path })
	sort.Slice(r.Skipped, func(i, j int) bool { return r.Skipped[i].Path < r.Skipped[j].Path })
	sort.SliceStable(r.Warnings, func(i, j int) bool {
		a, b := r.Warnings[i], r.Warnings[j]
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Path < b.Path
	})

	r.Summary = Summary{
		SkippedEntries:     len(r.Skipped),
		RulesExecuted:      rulesExecuted,
		Findings:           len(r.Findings),
		FindingsBySeverity: map[Severity]int{},
		Warnings:           len(r.Warnings),
	}
	for _, s := range Severities {
		r.Summary.FindingsBySeverity[s] = 0
	}
	for _, f := range r.Findings {
		r.Summary.FindingsBySeverity[f.Severity]++
	}
	for _, it := range r.Inventory {
		switch it.Category {
		case CategoryManifest:
			r.Summary.ManifestFiles++
		case CategorySource:
			r.Summary.SourceFiles++
		}
	}
	return r
}

// SortFindings orders findings by path, line, rule, then severity and
// confidence (strongest first), then message, so output is deterministic.
func SortFindings(fs []Finding) {
	sort.SliceStable(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.RuleID != b.RuleID {
			return a.RuleID < b.RuleID
		}
		if a.Severity != b.Severity {
			return a.Severity.rank() > b.Severity.rank()
		}
		if a.Confidence != b.Confidence {
			return confidenceRank(a.Confidence) > confidenceRank(b.Confidence)
		}
		return a.Message < b.Message
	})
}

func confidenceRank(c Confidence) int {
	switch c {
	case ConfidenceLow:
		return 0
	case ConfidenceMedium:
		return 1
	case ConfidenceHigh:
		return 2
	}
	return -1
}
