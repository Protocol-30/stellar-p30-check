package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Protocol-30/stellar-p30-check/internal/deps"
)

func fixtureReport() ScanReport {
	r := New("sample", "0.1.0-dev", time.Date(2026, 10, 25, 18, 12, 13, 123000000, time.FixedZone("local", 3600)))
	r.Coverage.Complete = true
	return r
}

func fixtureFinding() Finding {
	return Finding{
		RuleID: "P30-TEST-001", Severity: SeverityHigh, Confidence: ConfidenceHigh,
		Path: "main.go", Line: 7, Message: "injected test finding",
		Evidence: "test only", Remediation: "review", References: []string{"https://example.com/spec"},
	}
}

func TestJSONContractAndClock(t *testing.T) {
	r := fixtureReport()
	var out bytes.Buffer
	if err := Render(&out, "json", r); err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"schema_version", "tool_version", "target", "checked_at_utc", "findings", "summary", "limitations", "inventory", "dependencies", "dependency_warnings", "coverage"} {
		if _, ok := doc[field]; !ok {
			t.Errorf("missing required root field %q", field)
		}
	}
	for _, field := range []string{"findings", "limitations", "inventory", "dependencies", "dependency_warnings"} {
		if string(doc[field]) != "[]" {
			t.Errorf("%s should be [] rather than null: %s", field, doc[field])
		}
	}
	if string(doc["checked_at_utc"]) != `"2026-10-25T17:12:13.123Z"` {
		t.Errorf("UTC timestamp = %s", doc["checked_at_utc"])
	}
	if string(doc["schema_version"]) != `"`+SchemaVersion+`"` {
		t.Errorf("schema version = %s", doc["schema_version"])
	}
	var coverage map[string]json.RawMessage
	if err := json.Unmarshal(doc["coverage"], &coverage); err != nil {
		t.Fatal(err)
	}
	if string(coverage["skipped"]) != "[]" || string(coverage["rules_implemented"]) != "0" || string(coverage["complete"]) != "true" {
		t.Errorf("coverage = %s", doc["coverage"])
	}
	if _, ok := doc["error"]; ok {
		t.Error("empty error should be omitted")
	}
	var second bytes.Buffer
	if err := Render(&second, "json", r); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), second.Bytes()) {
		t.Error("same clock/input yielded different reports")
	}
}

func TestNormalizeSummaryAndStableSort(t *testing.T) {
	r := fixtureReport()
	a := fixtureFinding()
	a.Path = "a.go"
	a.References = []string{"https://example.com/z", "https://example.com/a"}
	b := a
	b.Line = 3
	b.Severity = SeverityMedium
	c := a
	c.Path = "z.go"
	c.Severity = SeverityInfo
	d := a
	d.Message = "a message before injected test finding"
	e := a
	e.References = []string{"https://example.com/b"}
	r.Findings = []Finding{c, a, e, b, d}
	r.Inventory = []InventoryItem{{Path: "z/go.mod", Kind: "go_module"}, {Path: "a/package.json", Kind: "js_manifest"}}
	r.Coverage.Skipped = []Skipped{{Path: "z", Reason: "symlink"}, {Path: "a", Reason: "ignored"}}
	r.Limitations = []string{"keep this first", "keep this second"}
	if err := Normalize(&r); err != nil {
		t.Fatal(err)
	}
	if r.Summary != (Summary{Total: 5, High: 3, Medium: 1, Info: 1}) {
		t.Errorf("summary = %+v", r.Summary)
	}
	if r.Findings[0].Line != 3 || r.Findings[1].Message != d.Message || r.Findings[4].Path != "z.go" {
		t.Errorf("unexpected finding sort: %+v", r.Findings)
	}
	if !slices.Equal(r.Findings[2].References, []string{"https://example.com/a", "https://example.com/z"}) {
		t.Errorf("references = %v", r.Findings[2].References)
	}
	if r.Inventory[0].Path != "a/package.json" || r.Coverage.Skipped[0].Path != "a" {
		t.Error("inventory/skips not sorted")
	}
	if !slices.Equal(r.Limitations, []string{"keep this first", "keep this second"}) {
		t.Error("limitations order changed")
	}
	first, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(r.Findings)
	slices.Reverse(r.Inventory)
	slices.Reverse(r.Coverage.Skipped)
	if err := Normalize(&r); err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Error("normalization depends on input finding order")
	}
}

func TestNormalizeRejectsInvalidFindings(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Finding)
	}{
		{"severity", func(f *Finding) { f.Severity = "critical" }},
		{"confidence", func(f *Finding) { f.Confidence = "informational" }},
		{"empty severity", func(f *Finding) { f.Severity = "" }},
		{"empty confidence", func(f *Finding) { f.Confidence = "" }},
		{"negative line", func(f *Finding) { f.Line = -1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := fixtureReport()
			f := fixtureFinding()
			tt.edit(&f)
			r.Findings = []Finding{f}
			if err := Normalize(&r); err == nil {
				t.Fatal("invalid finding accepted")
			}
			var out bytes.Buffer
			if err := Render(&out, "json", r); err == nil || out.Len() != 0 {
				t.Fatal("render must reject invalid finding before output")
			}
		})
	}
	if err := Normalize(nil); err == nil {
		t.Error("nil report accepted")
	}
}

func TestNormalizeEmptyArraysAndUnknownLine(t *testing.T) {
	r := ScanReport{Findings: []Finding{fixtureFinding()}}
	r.Findings[0].Line = 0
	r.Findings[0].References = nil
	if err := Normalize(&r); err != nil {
		t.Fatal(err)
	}
	if r.Findings[0].References == nil || r.Limitations == nil || r.Inventory == nil || r.Dependencies == nil || r.DependencyWarnings == nil || r.Coverage.Skipped == nil {
		t.Error("nil collections were not normalized")
	}
	b, err := json.Marshal(r.Findings[0])
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["line"]; ok {
		t.Error("unknown line should be omitted")
	}
}

func TestExitCodePolicy(t *testing.T) {
	for _, severity := range []Severity{SeverityInfo, SeverityMedium, SeverityHigh} {
		for _, confidence := range []Confidence{ConfidenceLow, ConfidenceMedium, ConfidenceHigh} {
			for _, gate := range []string{"none", "high"} {
				t.Run(string(severity)+"/"+string(confidence)+"/"+gate, func(t *testing.T) {
					r := fixtureReport()
					f := fixtureFinding()
					f.Severity, f.Confidence = severity, confidence
					r.Findings = []Finding{f}
					want := 0
					if gate == "high" && severity == SeverityHigh && confidence == ConfidenceHigh {
						want = 1
					}
					if got, err := ExitCode(r, gate); got != want || err != nil {
						t.Errorf("exit = %d, %v; want %d, nil", got, err, want)
					}
				})
			}
		}
	}
	t.Run("empty completed scan", func(t *testing.T) {
		if got, err := ExitCode(fixtureReport(), "high"); got != 0 || err != nil {
			t.Errorf("exit = %d, %v", got, err)
		}
	})
	t.Run("incomplete scan", func(t *testing.T) {
		r := fixtureReport()
		r.Coverage.Complete = false
		r.Findings = []Finding{fixtureFinding()}
		if got, err := ExitCode(r, "high"); got != 2 || err != nil {
			t.Errorf("exit = %d, %v", got, err)
		}
	})
	t.Run("reported scan error", func(t *testing.T) {
		r := fixtureReport()
		r.Error = "read failed"
		if got, err := ExitCode(r, "none"); got != 2 || err != nil {
			t.Errorf("exit = %d, %v", got, err)
		}
	})
	t.Run("invalid policy", func(t *testing.T) {
		if got, err := ExitCode(fixtureReport(), "medium"); got != 2 || err == nil {
			t.Errorf("exit = %d, %v", got, err)
		}
	})
	t.Run("invalid finding", func(t *testing.T) {
		r := fixtureReport()
		f := fixtureFinding()
		f.Confidence = "informational"
		r.Findings = []Finding{f}
		if got, err := ExitCode(r, "none"); got != 2 || err == nil {
			t.Errorf("exit = %d, %v", got, err)
		}
	})
}

func TestTextEscapesAllTargetControlledStrings(t *testing.T) {
	r := fixtureReport()
	payload := "untrusted\x1b[31m\nforged finding\r\t\u202e"
	r.Target, r.ToolVersion, r.CheckedAtUTC = payload, payload, payload
	r.Error = payload
	f := fixtureFinding()
	f.RuleID, f.Path, f.Message, f.Evidence, f.Remediation = payload, payload, payload, payload, payload
	f.References = []string{payload}
	r.Findings = []Finding{f}
	r.Inventory = []InventoryItem{{Path: payload, Kind: payload, Size: 10}}
	r.Coverage.Skipped = []Skipped{{Path: payload, Reason: payload}}
	r.Limitations = []string{payload}
	var out bytes.Buffer
	if err := Render(&out, "text", r); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"\x1b", "\r", "\t", "\u202e", "\nforged finding"} {
		if strings.Contains(out.String(), forbidden) {
			t.Errorf("raw control %q found in text", forbidden)
		}
	}
	if strings.Count(out.String(), `untrusted\x1b[31m\nforged finding\r\t\u202e`) != 15 {
		t.Errorf("not all fields safely rendered:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "rules implemented: 0") || !strings.Contains(out.String(), "does not assess Protocol 30 compatibility") {
		t.Error("zero-rule limitation is missing")
	}
	var encoded bytes.Buffer
	if err := Render(&encoded, "json", r); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(encoded.Bytes()) || bytes.ContainsRune(encoded.Bytes(), '\x1b') {
		t.Error("JSON is invalid or has an unescaped ESC")
	}
	var decoded ScanReport
	if err := json.Unmarshal(encoded.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Findings[0].Evidence != payload {
		t.Error("JSON did not preserve evidence after decoding")
	}
}

func TestTextShowsIncompleteAndError(t *testing.T) {
	r := fixtureReport()
	r.Coverage.Complete = false
	r.Error = "file size ceiling exceeded"
	var out bytes.Buffer
	if err := Render(&out, "text", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Discovery: incomplete") || !strings.Contains(out.String(), "Scan error:") {
		t.Errorf("missing failure status:\n%s", out.String())
	}
}

func TestRenderDoesNotMutateInput(t *testing.T) {
	r := fixtureReport()
	f := fixtureFinding()
	f.References = []string{"z", "a"}
	g := fixtureFinding()
	g.Path = "a.go"
	r.Findings = []Finding{f, g}
	r.Inventory = []InventoryItem{{Path: "z"}, {Path: "a"}}
	r.Coverage.Skipped = []Skipped{{Path: "z"}, {Path: "a"}}
	before := clone(r)
	for _, format := range []string{"text", "json"} {
		if err := Render(io.Discard, format, r); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(r, before) {
		t.Error("Render changed its caller's slices or summary")
	}
}

type failedWriter struct{ err error }

func (w failedWriter) Write([]byte) (int, error) { return 0, w.err }

func TestRenderPropagatesWriterError(t *testing.T) {
	sentinel := errors.New("output closed")
	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			err := Render(failedWriter{sentinel}, format, fixtureReport())
			if !errors.Is(err, sentinel) {
				t.Errorf("error = %v", err)
			}
		})
	}
}

func TestRenderRejectsUnsupportedFormatWithoutOutput(t *testing.T) {
	var out bytes.Buffer
	if err := Render(&out, "sarif", fixtureReport()); err == nil || out.Len() != 0 {
		t.Error("unsupported format was not rejected before output")
	}
}

func fixtureDependency() deps.Dependency {
	return deps.Dependency{
		Ecosystem: "npm", Name: "@stellar/stellar-sdk", Group: "dependencies",
		Scope: ".", Path: "package.json", Declared: "^16.4.0", Version: "16.4.0",
		Resolution: "lockfile", Source: "package-lock.json", Effective: true,
		Capabilities: &deps.Capabilities{
			MatrixVersion: "1", CheckedAt: "2026-10-25", Version: "16.4.0",
			CAP88: "supported", MuxedContractXDR: "supported", WAddressHelpers: "unsupported",
			References: []string{"https://example.com/z", "https://example.com/a"},
		},
	}
}

func TestDependencyNormalizationAndInputImmutability(t *testing.T) {
	r := fixtureReport()
	a := fixtureDependency()
	a.Uncertainty = []string{"z uncertainty", "a uncertainty", "a uncertainty"}
	b := fixtureDependency()
	b.Path = "apps/other/package.json"
	b.Effective = false
	b.Capabilities = nil
	r.Dependencies = []deps.Dependency{a, b}
	r.DependencyWarnings = []deps.Warning{
		{Path: "z/package.json", Code: "unresolved", Message: "not selected"},
		{Path: "a/package.json", Code: "missing_lock", Message: "lock unavailable"},
	}
	before, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var first bytes.Buffer
	if err := Render(&first, "json", r); err != nil {
		t.Fatal(err)
	}
	if err := Render(io.Discard, "text", r); err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Error("Render changed caller dependency metadata or nested capabilities")
	}
	var decoded ScanReport
	if err := json.Unmarshal(first.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Dependencies[0].Path != b.Path || decoded.DependencyWarnings[0].Path != "a/package.json" {
		t.Errorf("dependency/warning sort is unstable: %+v %+v", decoded.Dependencies, decoded.DependencyWarnings)
	}
	normalized := decoded.Dependencies[1]
	if !slices.Equal(normalized.Uncertainty, []string{"a uncertainty", "z uncertainty"}) || !slices.Equal(normalized.Capabilities.References, []string{"https://example.com/a", "https://example.com/z"}) {
		t.Errorf("nested dependency metadata not normalized: %+v", normalized)
	}
	if decoded.Dependencies[0].Uncertainty == nil {
		t.Error("unknown/empty uncertainty must remain an array")
	}
	slices.Reverse(r.Dependencies)
	slices.Reverse(r.DependencyWarnings)
	var second bytes.Buffer
	if err := Render(&second, "json", r); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Error("dependency report depends on inventory input ordering")
	}
}

func TestDependencyTextEscapesAllTargetMetadata(t *testing.T) {
	r := fixtureReport()
	payload := "untrusted\x1b[31m\nforged dependency\r\t\u202e"
	d := fixtureDependency()
	d.Ecosystem, d.Name, d.Group, d.Alias = payload, payload, payload, payload
	d.Scope, d.Path, d.Declared, d.Version = payload, payload, payload, payload
	d.Resolution, d.Source, d.Uncertainty = payload, payload, []string{payload}
	d.Capabilities.MatrixVersion, d.Capabilities.CheckedAt, d.Capabilities.Version = payload, payload, payload
	d.Capabilities.References = []string{payload}
	r.Dependencies = []deps.Dependency{d}
	r.DependencyWarnings = []deps.Warning{{Path: payload, Code: payload, Message: payload}}
	var out bytes.Buffer
	if err := Render(&out, "text", r); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"\x1b", "\r", "\t", "\u202e", "\nforged dependency"} {
		if strings.Contains(out.String(), forbidden) {
			t.Errorf("raw dependency control %q found in text", forbidden)
		}
	}
	if strings.Count(out.String(), `untrusted\x1b[31m\nforged dependency\r\t\u202e`) != 18 {
		t.Errorf("not all dependency fields rendered safely:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "observed version:") || !strings.Contains(out.String(), "effective: true") {
		t.Error("text hides observed-version/effective-selection distinction")
	}
	var encoded bytes.Buffer
	if err := Render(&encoded, "json", r); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(encoded.Bytes()) || bytes.ContainsRune(encoded.Bytes(), '\x1b') {
		t.Error("dependency JSON has invalid or raw control text")
	}
}

func TestRejectInvalidDependencyMetadataBeforeOutput(t *testing.T) {
	tests := []struct {
		name string
		edit func(*ScanReport)
	}{
		{"dependency line", func(r *ScanReport) { r.Dependencies[0].Line = -1 }},
		{"warning line", func(r *ScanReport) { r.DependencyWarnings = []deps.Warning{{Line: -1}} }},
		{"CAP88 state", func(r *ScanReport) { r.Dependencies[0].Capabilities.CAP88 = "yes" }},
		{"muxed state", func(r *ScanReport) { r.Dependencies[0].Capabilities.MuxedContractXDR = "" }},
		{"helpers state", func(r *ScanReport) { r.Dependencies[0].Capabilities.WAddressHelpers = "maybe" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := fixtureReport()
			r.Dependencies = []deps.Dependency{fixtureDependency()}
			test.edit(&r)
			var out bytes.Buffer
			if err := Render(&out, "json", r); err == nil || out.Len() != 0 {
				t.Fatal("invalid dependency metadata was not rejected before writing")
			}
		})
	}
}
