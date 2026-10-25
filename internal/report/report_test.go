package report

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

var fixedTime = time.Date(2026, 10, 25, 12, 0, 0, 0, time.FixedZone("WAT", 3600))

func finding(path string, line int, rule string, sev Severity, conf Confidence) Finding {
	return Finding{
		RuleID: rule, Severity: sev, Confidence: conf, Path: path, Line: line,
		Message: "m", Evidence: "e", Remediation: "r",
	}
}

func TestSortFindingsIsStableAndTotal(t *testing.T) {
	in := []Finding{
		finding("b.go", 1, "P30-A", SeverityInfo, ConfidenceLow),
		finding("a.go", 9, "P30-A", SeverityInfo, ConfidenceLow),
		finding("a.go", 2, "P30-B", SeverityInfo, ConfidenceLow),
		finding("a.go", 2, "P30-A", SeverityMedium, ConfidenceLow),
		finding("a.go", 2, "P30-A", SeverityHigh, ConfidenceLow),
		finding("a.go", 2, "P30-A", SeverityHigh, ConfidenceHigh),
	}
	want := []Finding{in[5], in[4], in[3], in[2], in[1], in[0]}

	// Every input permutation must sort to the same order.
	for shift := 0; shift < len(in); shift++ {
		perm := append(append([]Finding{}, in[shift:]...), in[:shift]...)
		SortFindings(perm)
		if !reflect.DeepEqual(perm, want) {
			t.Fatalf("shift %d: got %+v\nwant %+v", shift, perm, want)
		}
	}
}

func TestFindingValidate(t *testing.T) {
	ok := finding("a.go", 1, "P30-X", SeverityHigh, ConfidenceHigh)
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid finding rejected: %v", err)
	}
	bad := map[string]func(*Finding){
		"no rule":        func(f *Finding) { f.RuleID = "" },
		"bad severity":   func(f *Finding) { f.Severity = "critical" },
		"bad confidence": func(f *Finding) { f.Confidence = "certain" },
		"no path":        func(f *Finding) { f.Path = "" },
		"negative line":  func(f *Finding) { f.Line = -1 },
		"no message":     func(f *Finding) { f.Message = "" },
	}
	for name, mutate := range bad {
		f := ok
		mutate(&f)
		if err := f.Validate(); err == nil {
			t.Errorf("%s: Validate accepted %+v", name, f)
		}
	}
}

func sampleReport() ScanReport {
	return New("0.1.0-test", "./repo", fixedTime, 1,
		[]Finding{
			finding("z.ts", 3, "P30-ADDR-001", SeverityMedium, ConfidenceMedium),
			finding("a.go", 0, "P30-DEP-001", SeverityHigh, ConfidenceHigh),
		},
		[]InventoryItem{
			{Path: "z.ts", Kind: "source-ts", Category: CategorySource},
			{Path: "go.mod", Kind: "go-mod", Category: CategoryManifest},
		},
		[]SkippedEntry{{Path: "node_modules", Reason: "ignored directory"}},
		[]Warning{{Code: "w2", Message: "second"}, {Code: "w1", Path: "go.mod", Message: "first"}},
		[]string{"limited"},
	)
}

func TestNewComputesSummaryAndSorts(t *testing.T) {
	r := sampleReport()
	if r.CheckedAtUTC != "2026-10-25T11:00:00Z" {
		t.Errorf("checked_at_utc = %q, want UTC RFC3339", r.CheckedAtUTC)
	}
	want := Summary{
		ManifestFiles: 1, SourceFiles: 1, SkippedEntries: 1, RulesExecuted: 1, Findings: 2, Warnings: 2,
		FindingsBySeverity: map[Severity]int{SeverityInfo: 0, SeverityMedium: 1, SeverityHigh: 1},
	}
	if !reflect.DeepEqual(r.Summary, want) {
		t.Errorf("summary = %+v\nwant %+v", r.Summary, want)
	}
	if r.Findings[0].Path != "a.go" || r.Inventory[0].Path != "go.mod" || r.Warnings[0].Code != "w1" {
		t.Errorf("report contents not sorted: %+v", r)
	}
	if r.Findings[0].References == nil {
		t.Error("nil references not normalized to empty slice")
	}
}

func TestJSONHasStableSchemaFields(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderJSON(&buf, New("v", "t", fixedTime, 0, nil, nil, nil, nil, nil)); err != nil {
		t.Fatal(err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &root); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"schema_version", "tool_version", "target", "checked_at_utc",
		"findings", "summary", "inventory", "skipped", "warnings", "limitations",
	} {
		if _, ok := root[key]; !ok {
			t.Errorf("JSON root missing %q", key)
		}
	}
	if len(root) != 10 {
		t.Errorf("JSON root has %d keys, want 10; update SchemaVersion and docs when changing the contract", len(root))
	}
	// Empty collections must be arrays, never null.
	for _, key := range []string{"findings", "inventory", "skipped", "warnings", "limitations"} {
		if string(root[key]) != "[]" {
			t.Errorf("%s = %s, want []", key, root[key])
		}
	}
	if string(root["schema_version"]) != `"1"` {
		t.Errorf("schema_version = %s", root["schema_version"])
	}
}

func TestJSONFindingFields(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderJSON(&buf, sampleReport()); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Findings []map[string]any `json:"findings"`
	}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	f := decoded.Findings[1] // z.ts:3
	for _, key := range []string{"rule_id", "severity", "confidence", "path", "line", "message", "evidence", "remediation", "references"} {
		if _, ok := f[key]; !ok {
			t.Errorf("finding missing %q: %v", key, f)
		}
	}
	if _, ok := decoded.Findings[0]["line"]; ok {
		t.Error("unknown line (0) should be omitted")
	}
}

func TestTextAndJSONCountsAgree(t *testing.T) {
	r := sampleReport()
	var text bytes.Buffer
	if err := RenderText(&text, r); err != nil {
		t.Fatal(err)
	}
	out := text.String()
	for _, want := range []string{
		"Findings (2): 1 high, 1 medium, 0 info",
		"[high severity, high confidence] P30-DEP-001 a.go\n",
		"P30-ADDR-001 z.ts:3",
		"Inventory:  1 manifest file(s), 1 source file(s), 1 skipped entry",
		"[w1] go.mod: first",
		"node_modules (ignored directory)",
		"  - limited",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("text report missing %q:\n%s", want, out)
		}
	}
	if got := strings.Count(out, " severity, "); got != r.Summary.Findings {
		t.Errorf("text lists %d findings, summary says %d", got, r.Summary.Findings)
	}
}

func TestTextEscapesControlCharacters(t *testing.T) {
	r := New("v", "t", fixedTime, 0, nil,
		[]InventoryItem{{Path: "evil\x1b[2Jname.go", Kind: "source-go", Category: CategorySource}},
		[]SkippedEntry{{Path: "line\nbreak", Reason: "ignored directory"}}, nil, nil)
	var buf bytes.Buffer
	if err := RenderText(&buf, r); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.ContainsRune(out, '\x1b') || strings.Contains(out, "line\nbreak") {
		t.Errorf("control characters reached text output: %q", out)
	}
	if !strings.Contains(out, `"evil\x1b[2Jname.go"`) {
		t.Errorf("escaped path not shown: %q", out)
	}
}

func TestParseFormat(t *testing.T) {
	for _, ok := range []string{"text", "json"} {
		if _, err := ParseFormat(ok); err != nil {
			t.Errorf("ParseFormat(%q): %v", ok, err)
		}
	}
	for _, bad := range []string{"", "JSON", "sarif", "yaml"} {
		if _, err := ParseFormat(bad); err == nil {
			t.Errorf("ParseFormat(%q) accepted", bad)
		}
	}
}
