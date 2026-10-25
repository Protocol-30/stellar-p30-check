package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"unicode"
)

// Format names an output renderer.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// ParseFormat validates a --format value.
func ParseFormat(s string) (Format, error) {
	switch Format(s) {
	case FormatText, FormatJSON:
		return Format(s), nil
	}
	return "", fmt.Errorf("unsupported format %q (want text or json)", s)
}

// Render writes r to w in the given format.
func Render(w io.Writer, r ScanReport, f Format) error {
	switch f {
	case FormatJSON:
		return RenderJSON(w, r)
	case FormatText:
		return RenderText(w, r)
	}
	return fmt.Errorf("unsupported format %q", f)
}

// RenderJSON writes r as indented JSON followed by a newline.
func RenderJSON(w io.Writer, r ScanReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return fmt.Errorf("encode json report: %w", err)
	}
	return nil
}

// RenderText writes a human-readable report. Paths and other values taken
// from the scanned tree are escaped so they cannot inject control sequences
// into a terminal.
func RenderText(w io.Writer, r ScanReport) error {
	tw := &errWriter{w: w}
	s := r.Summary

	tw.printf("p30check %s - Stellar Protocol 30 compatibility scan (report schema %s)\n", r.ToolVersion, r.SchemaVersion)
	tw.printf("Target:     %s\n", clean(r.Target))
	tw.printf("Checked at: %s\n", r.CheckedAtUTC)
	tw.printf("Inventory:  %d manifest file(s), %d source file(s), %d skipped entr%s\n",
		s.ManifestFiles, s.SourceFiles, s.SkippedEntries, plural(s.SkippedEntries, "y", "ies"))
	tw.printf("Rules run:  %d\n", s.RulesExecuted)

	tw.printf("\nFindings (%d):", s.Findings)
	if s.Findings == 0 {
		tw.printf(" none reported\n")
	} else {
		tw.printf(" %d high, %d medium, %d info\n",
			s.FindingsBySeverity[SeverityHigh], s.FindingsBySeverity[SeverityMedium], s.FindingsBySeverity[SeverityInfo])
		for _, f := range r.Findings {
			loc := clean(f.Path)
			if f.Line > 0 {
				loc += ":" + strconv.Itoa(f.Line)
			}
			tw.printf("  [%s severity, %s confidence] %s %s\n", f.Severity, f.Confidence, clean(f.RuleID), loc)
			tw.printf("      %s\n", clean(f.Message))
			if f.Evidence != "" {
				tw.printf("      evidence: %s\n", clean(f.Evidence))
			}
			if f.Remediation != "" {
				tw.printf("      fix:      %s\n", clean(f.Remediation))
			}
			for _, ref := range f.References {
				tw.printf("      ref:      %s\n", clean(ref))
			}
		}
	}

	if len(r.Warnings) > 0 {
		tw.printf("\nWarnings (%d):\n", len(r.Warnings))
		for _, wn := range r.Warnings {
			if wn.Path != "" {
				tw.printf("  [%s] %s: %s\n", clean(wn.Code), clean(wn.Path), clean(wn.Message))
			} else {
				tw.printf("  [%s] %s\n", clean(wn.Code), clean(wn.Message))
			}
		}
	}

	if len(r.Inventory) > 0 {
		tw.printf("\nDiscovered files:\n")
		for _, it := range r.Inventory {
			tw.printf("  %-8s %-16s %s\n", it.Category, it.Kind, clean(it.Path))
		}
	}

	if len(r.Skipped) > 0 {
		tw.printf("\nSkipped:\n")
		for _, sk := range r.Skipped {
			tw.printf("  %s (%s)\n", clean(sk.Path), clean(sk.Reason))
		}
	}

	if len(r.Limitations) > 0 {
		tw.printf("\nLimitations:\n")
		for _, l := range r.Limitations {
			tw.printf("  - %s\n", clean(l))
		}
	}
	return tw.err
}

// clean quotes a string if it holds anything other than printable
// characters, so file names with escape codes or newlines stay inert.
func clean(s string) string {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return strconv.Quote(s)
		}
	}
	return s
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

type errWriter struct {
	w   io.Writer
	err error
}

func (e *errWriter) printf(format string, args ...any) {
	if e.err != nil {
		return
	}
	if _, err := fmt.Fprintf(e.w, format, args...); err != nil {
		e.err = fmt.Errorf("write text report: %w", err)
	}
}
