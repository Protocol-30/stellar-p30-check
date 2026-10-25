// Package report defines the versioned output and exit policy for p30check.
package report

import (
	"cmp"
	"fmt"
	"slices"
	"time"
)

// SchemaVersion changes when the report's consumer-visible contract changes.
const SchemaVersion = "1"

type Severity string

const (
	SeverityInfo   Severity = "info"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

type Confidence string

const (
	ConfidenceLow    Confidence = "low"
	ConfidenceMedium Confidence = "medium"
	ConfidenceHigh   Confidence = "high"
)

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

type Summary struct {
	Total  int `json:"total"`
	Info   int `json:"info"`
	Medium int `json:"medium"`
	High   int `json:"high"`
}

// InventoryItem describes a discovered manifest, lockfile, or selected source
// file; its contents are deliberately not included in the report.
type InventoryItem struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
	Size int64  `json:"size"`
}

type Skipped struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// Coverage distinguishes deliberate exclusions from unexpectedly incomplete
// scans. Complete describes discovery scope, not protocol compatibility.
type Coverage struct {
	Complete         bool      `json:"complete"`
	RulesImplemented int       `json:"rules_implemented"`
	FilesScanned     int       `json:"files_scanned"`
	EntriesVisited   int       `json:"entries_visited"`
	BytesRead        int64     `json:"bytes_read"`
	Skipped          []Skipped `json:"skipped"`
}

type ScanReport struct {
	SchemaVersion string          `json:"schema_version"`
	ToolVersion   string          `json:"tool_version"`
	Target        string          `json:"target"`
	CheckedAtUTC  string          `json:"checked_at_utc"`
	Findings      []Finding       `json:"findings"`
	Summary       Summary         `json:"summary"`
	Limitations   []string        `json:"limitations"`
	Inventory     []InventoryItem `json:"inventory"`
	Coverage      Coverage        `json:"coverage"`
	Error         string          `json:"error,omitempty"`
}

// New accepts a clock value rather than reading the clock internally so tests
// and reproducible callers can choose exactly the same timestamp.
func New(target, version string, now time.Time) ScanReport {
	return ScanReport{
		SchemaVersion: SchemaVersion,
		ToolVersion:   version,
		Target:        target,
		CheckedAtUTC:  now.UTC().Format(time.RFC3339Nano),
		Findings:      []Finding{},
		Limitations:   []string{},
		Inventory:     []InventoryItem{},
		Coverage:      Coverage{Skipped: []Skipped{}},
	}
}

func validateFinding(f Finding) error {
	switch f.Severity {
	case SeverityInfo, SeverityMedium, SeverityHigh:
	default:
		return fmt.Errorf("finding %q: invalid severity %q", f.RuleID, f.Severity)
	}
	switch f.Confidence {
	case ConfidenceLow, ConfidenceMedium, ConfidenceHigh:
	default:
		return fmt.Errorf("finding %q: invalid confidence %q", f.RuleID, f.Confidence)
	}
	if f.Line < 0 {
		return fmt.Errorf("finding %q: line must not be negative", f.RuleID)
	}
	return nil
}

// Normalize validates finding enums, recomputes counts, and orders collections.
// It mutates r. Render clones its input before calling Normalize.
func Normalize(r *ScanReport) error {
	if r == nil {
		return fmt.Errorf("cannot normalize a nil report")
	}
	for _, f := range r.Findings {
		if err := validateFinding(f); err != nil {
			return err
		}
	}
	if r.Findings == nil {
		r.Findings = []Finding{}
	}
	if r.Limitations == nil {
		r.Limitations = []string{}
	}
	if r.Inventory == nil {
		r.Inventory = []InventoryItem{}
	}
	if r.Coverage.Skipped == nil {
		r.Coverage.Skipped = []Skipped{}
	}
	r.Summary = Summary{Total: len(r.Findings)}
	for i := range r.Findings {
		f := &r.Findings[i]
		if f.References == nil {
			f.References = []string{}
		}
		slices.Sort(f.References)
		switch f.Severity {
		case SeverityInfo:
			r.Summary.Info++
		case SeverityMedium:
			r.Summary.Medium++
		case SeverityHigh:
			r.Summary.High++
		}
	}
	slices.SortFunc(r.Findings, compareFindings)
	slices.SortFunc(r.Inventory, func(a, b InventoryItem) int {
		if c := cmp.Compare(a.Path, b.Path); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Kind, b.Kind); c != 0 {
			return c
		}
		return cmp.Compare(a.Size, b.Size)
	})
	slices.SortFunc(r.Coverage.Skipped, func(a, b Skipped) int {
		if c := cmp.Compare(a.Path, b.Path); c != 0 {
			return c
		}
		return cmp.Compare(a.Reason, b.Reason)
	})
	return nil
}

func compareFindings(a, b Finding) int {
	for _, c := range []int{
		cmp.Compare(a.Path, b.Path),
		cmp.Compare(a.Line, b.Line),
		cmp.Compare(a.RuleID, b.RuleID),
		cmp.Compare(a.Severity, b.Severity),
		cmp.Compare(a.Confidence, b.Confidence),
		cmp.Compare(a.Message, b.Message),
		cmp.Compare(a.Evidence, b.Evidence),
		cmp.Compare(a.Remediation, b.Remediation),
	} {
		if c != 0 {
			return c
		}
	}
	return slices.Compare(a.References, b.References)
}

// ExitCode applies an opt-in gate only to high-severity, high-confidence
// findings. An incomplete scan or invalid policy always remains an error.
func ExitCode(r ScanReport, failOn string) (int, error) {
	if failOn != "none" && failOn != "high" {
		return 2, fmt.Errorf("unsupported fail-on policy %q", failOn)
	}
	for _, f := range r.Findings {
		if err := validateFinding(f); err != nil {
			return 2, err
		}
	}
	if r.Error != "" || !r.Coverage.Complete {
		return 2, nil
	}
	if failOn == "high" {
		for _, f := range r.Findings {
			if f.Severity == SeverityHigh && f.Confidence == ConfidenceHigh {
				return 1, nil
			}
		}
	}
	return 0, nil
}
