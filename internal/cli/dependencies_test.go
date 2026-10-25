package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Protocol-30/stellar-p30-check/internal/report"
	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

func TestDependencyCapabilityReports(t *testing.T) {
	for _, tc := range []struct {
		fixture   string
		findings  int
		effective bool
		version   string
		cap88     string
		wHelpers  string
		warn      bool
	}{
		{"m2-js-maintenance", 1, true, "16.4.0", "supported", "unsupported", false},
		{"m2-js-modern", 0, true, "17.3.0", "supported", "supported", false},
		{"m2-js-legacy", 1, true, "16.3.1", "unsupported", "unsupported", false},
		{"m2-go-observed", 0, false, "v0.8.0", "supported", "supported", true},
		{"m2-js-unresolved", 0, false, "", "unknown", "unknown", true},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			var prior []byte
			for i := 0; i < 2; i++ {
				var stdout, stderr bytes.Buffer
				args := []string{"scan", filepath.Join("..", "..", "testdata", tc.fixture), "--format=json", "--fail-on=high"}
				code := Run(args, &stdout, &stderr, Options{Now: func() time.Time { return fixedNow }})
				if code != 0 || stderr.Len() != 0 {
					t.Fatalf("exit=%d stderr=%s stdout=%s", code, &stderr, &stdout)
				}
				var r report.ScanReport
				if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
					t.Fatal(err)
				}
				if !r.Coverage.Complete || r.Coverage.RulesImplemented != 1 || r.SchemaVersion != report.SchemaVersion || len(r.Dependencies) != 1 || len(r.Findings) != tc.findings {
					t.Fatalf("incorrect report: %+v", r)
				}
				d := r.Dependencies[0]
				if d.Effective != tc.effective || d.Version != tc.version {
					t.Fatalf("selection mismatch: %+v", d)
				}
				if d.Capabilities == nil || d.Capabilities.CAP88 != tc.cap88 || d.Capabilities.WAddressHelpers != tc.wHelpers {
					t.Fatalf("capabilities mismatch: %+v", d.Capabilities)
				}
				if tc.warn && len(r.DependencyWarnings) == 0 {
					t.Fatal("missing uncertainty warning")
				}
				for _, finding := range r.Findings {
					if finding.RuleID != "P30-DEP-001" || finding.Severity != report.SeverityMedium || finding.Confidence != report.ConfidenceHigh || len(finding.References) == 0 || finding.Evidence == "" {
						t.Fatalf("inaccurate finding: %+v", finding)
					}
				}
				if i == 0 {
					prior = bytes.Clone(stdout.Bytes())
				} else if !bytes.Equal(prior, stdout.Bytes()) {
					t.Fatal("dependency report changed with fixed clock")
				}
			}
		})
	}
}

func TestMalformedManifestIsIncomplete(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies": {`), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"scan", dir, "--format=json"}, &stdout, &stderr, Options{}); code != 2 {
		t.Fatalf("exit=%d stdout=%s", code, &stdout)
	}
	var r report.ScanReport
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Coverage.Complete || r.Error == "" || len(r.Findings) != 0 || len(r.Inventory) != 1 {
		t.Fatalf("invalid manifest hidden: %+v", r)
	}
}

func TestNULManifestDoesNotDisappearAsBinarySource(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{\"dependencies\": {\x00}}"), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"scan", dir, "--format=json"}, &stdout, &stderr, Options{}); code != 2 {
		t.Fatalf("exit=%d stdout=%s", code, &stdout)
	}
	var r report.ScanReport
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Coverage.Complete || !strings.Contains(r.Error, "package.json") || len(r.Findings) != 0 {
		t.Fatalf("NUL metadata exclusion hidden: %+v", r)
	}
}

func TestCapabilityFindingsWithheldForPartialTree(t *testing.T) {
	var stdout, stderr bytes.Buffer
	opts := Options{Scan: func(path string, limits walker.Limits) (walker.Result, error) {
		r, err := walker.Scan(path, limits)
		if err != nil {
			return r, err
		}
		r.Complete = false
		return r, errors.New("forced discovery failure after manifest reads")
	}}
	code := Run([]string{"scan", filepath.Join("..", "..", "testdata", "m2-js-maintenance"), "--format=json"}, &stdout, &stderr, opts)
	if code != 2 {
		t.Fatalf("exit=%d", code)
	}
	var r report.ScanReport
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Coverage.Complete || len(r.Findings) != 0 || len(r.Dependencies) != 1 || r.Dependencies[0].Effective || !strings.Contains(strings.Join(r.Limitations, " "), "withheld") {
		t.Fatalf("partial evidence promoted: %+v", r)
	}
}

func TestScanDoesNotExecutePackageScripts(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"name":"untrusted","scripts":{"preinstall":"touch should-not-exist","postinstall":"exit 99"},"dependencies":{"@stellar/stellar-sdk":"^17.3.0"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"scan", dir, "--format=json"}, &stdout, &stderr, Options{}); code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, &stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, "should-not-exist")); !os.IsNotExist(err) {
		t.Fatalf("unexpected script artifact: %v", err)
	}
	var r report.ScanReport
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if len(r.DependencyWarnings) == 0 || len(r.Findings) != 0 {
		t.Fatalf("range without lock not uncertain: %+v", r)
	}
}
