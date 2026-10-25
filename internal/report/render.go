package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
)

// Render writes text or a complete JSON document. Text quotes all strings that
// may originate in a scan target; embedded controls cannot forge report lines.
func Render(w io.Writer, format string, r ScanReport) error {
	if format != "text" && format != "json" {
		return fmt.Errorf("unsupported report format %q", format)
	}
	r = clone(r)
	if err := Normalize(&r); err != nil {
		return fmt.Errorf("normalize report: %w", err)
	}
	if format == "json" {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(r); err != nil {
			return fmt.Errorf("write JSON report: %w", err)
		}
		return nil
	}
	var out bytes.Buffer
	fmt.Fprintf(&out, "p30check %s\nTarget: %s\nChecked at UTC: %s\n", safe(r.ToolVersion), safe(r.Target), safe(r.CheckedAtUTC))
	status := "incomplete"
	if r.Coverage.Complete {
		status = "complete within selected regular-file scope"
	}
	fmt.Fprintf(&out, "Discovery: %s; rules implemented: %d\n", status, r.Coverage.RulesImplemented)
	if r.Coverage.RulesImplemented == 0 {
		out.WriteString("No compatibility rules are implemented; this report does not assess Protocol 30 compatibility.\n")
	}
	fmt.Fprintf(&out, "Files scanned: %d; entries visited: %d; bytes read: %d\n", r.Coverage.FilesScanned, r.Coverage.EntriesVisited, r.Coverage.BytesRead)
	fmt.Fprintf(&out, "Findings: %d (high: %d, medium: %d, info: %d)\n", r.Summary.Total, r.Summary.High, r.Summary.Medium, r.Summary.Info)
	if r.Error != "" {
		fmt.Fprintf(&out, "Scan error: %s\n", safe(r.Error))
	}
	for _, f := range r.Findings {
		fmt.Fprintf(&out, "Finding: rule=%s severity=%s confidence=%s path=%s", safe(f.RuleID), safe(string(f.Severity)), safe(string(f.Confidence)), safe(f.Path))
		if f.Line != 0 {
			fmt.Fprintf(&out, " line=%d", f.Line)
		}
		fmt.Fprintf(&out, "\n  Message: %s\n  Evidence: %s\n  Remediation: %s\n", safe(f.Message), safe(f.Evidence), safe(f.Remediation))
		for _, ref := range f.References {
			fmt.Fprintf(&out, "  Reference: %s\n", safe(ref))
		}
	}
	fmt.Fprintf(&out, "File inventory: %d\n", len(r.Inventory))
	for _, item := range r.Inventory {
		fmt.Fprintf(&out, "  %s kind=%s bytes=%d\n", safe(item.Path), safe(item.Kind), item.Size)
	}
	fmt.Fprintf(&out, "Skipped entries: %d\n", len(r.Coverage.Skipped))
	for _, item := range r.Coverage.Skipped {
		fmt.Fprintf(&out, "  %s reason=%s\n", safe(item.Path), safe(item.Reason))
	}
	for _, limitation := range r.Limitations {
		fmt.Fprintf(&out, "Limitation: %s\n", safe(limitation))
	}
	if _, err := io.Copy(w, &out); err != nil {
		return fmt.Errorf("write text report: %w", err)
	}
	return nil
}

func safe(s string) string {
	return strconv.QuoteToASCII(s)
}

func clone(r ScanReport) ScanReport {
	r.Findings = slices.Clone(r.Findings)
	for i := range r.Findings {
		r.Findings[i].References = slices.Clone(r.Findings[i].References)
	}
	r.Limitations = slices.Clone(r.Limitations)
	r.Inventory = slices.Clone(r.Inventory)
	r.Coverage.Skipped = slices.Clone(r.Coverage.Skipped)
	return r
}
