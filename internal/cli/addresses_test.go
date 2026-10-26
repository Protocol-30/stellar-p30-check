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

func TestAddressFixtureReportsAndGate(t *testing.T) {
	for _, tc := range []struct {
		fixture    string
		findings   int
		severity   report.Severity
		confidence report.Confidence
	}{
		{"m3-js-reject-w", 1, report.SeverityHigh, report.ConfidenceHigh},
		{"m3-js-allow-w", 0, "", ""},
		{"m3-js-account-only", 0, "", ""},
		{"m3-js-negative-contexts", 0, "", ""},
		{"m3-js-scope-uncertain", 1, report.SeverityMedium, report.ConfidenceMedium},
		{"m3-go-review", 1, report.SeverityMedium, report.ConfidenceMedium},
		{"m3-js-sac-policy", 1, report.SeverityMedium, report.ConfidenceLow},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			var prior []byte
			for _, gate := range []string{"none", "high"} {
				var stdout, stderr bytes.Buffer
				args := []string{"scan", filepath.Join("..", "..", "testdata", tc.fixture), "--format=json", "--fail-on=" + gate}
				code := Run(args, &stdout, &stderr, Options{Now: func() time.Time { return fixedNow }})
				want := 0
				if gate == "high" && tc.severity == report.SeverityHigh && tc.confidence == report.ConfidenceHigh {
					want = 1
				}
				if code != want || stderr.Len() != 0 {
					t.Fatalf("exit=%d want%d stdout=%s stderr=%s", code, want, &stdout, &stderr)
				}
				var r report.ScanReport
				if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
					t.Fatal(err)
				}
				if r.SchemaVersion != "2" || !r.Coverage.Complete || r.Coverage.RulesImplemented != 2 || len(r.Findings) != tc.findings {
					t.Fatalf("incorrect report: %+v", r)
				}
				for _, f := range r.Findings {
					if f.RuleID != "P30-ADDR-001" || f.Severity != tc.severity || f.Confidence != tc.confidence || f.Line <= 0 || len(f.References) == 0 || f.Evidence == "" {
						t.Fatalf("incorrect address finding: %+v", f)
					}
				}
				if gate == "none" {
					prior = bytes.Clone(stdout.Bytes())
				} else if !bytes.Equal(prior, stdout.Bytes()) {
					t.Fatal("gate policy changed report content with fixed clock")
				}
			}
		})
	}
}

func TestAddressHighFindingIsWithheldForIncompleteDiscovery(t *testing.T) {
	var stdout, stderr bytes.Buffer
	opts := Options{Scan: func(path string, limits walker.Limits) (walker.Result, error) {
		r, err := walker.Scan(path, limits)
		if err != nil {
			return r, err
		}
		r.Complete = false
		return r, errors.New("incomplete discovery after source and manifest reads")
	}}
	code := Run([]string{"scan", filepath.Join("..", "..", "testdata", "m3-js-reject-w"), "--format=json", "--fail-on=high"}, &stdout, &stderr, opts)
	if code != 2 {
		t.Fatalf("exit=%d", code)
	}
	var r report.ScanReport
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Coverage.Complete || len(r.Findings) != 0 || r.Error == "" {
		t.Fatalf("partial source proof promoted: %+v", r)
	}
}

func TestAddressSourceSyntaxLimitationIsExplicit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.ts"), []byte("export function { if return StrKey.decodeMuxedContract()"), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"scan", dir, "--format=json"}, &stdout, &stderr, Options{}); code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, &stderr)
	}
	var r report.ScanReport
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if !r.Coverage.Complete || len(r.Findings) != 0 || !strings.Contains(strings.Join(r.Limitations, " "), "broken.ts") || !strings.Contains(strings.Join(r.Limitations, " "), "syntax") {
		t.Fatalf("source parse limitation hidden: %+v", r)
	}
}

func TestAddressResourceExhaustionIsIncomplete(t *testing.T) {
	dir := t.TempDir()
	// 60000 valid blank declarations fit the read budget but exceed the
	// analyzer's actual 200000-node budget; no fake analyzer result is injected.
	source := "package resource\n" + strings.Repeat("var _ = 0\n", 60000)
	if err := os.WriteFile(filepath.Join(dir, "large.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"scan", dir, "--format=json", "--fail-on=high"}, &stdout, &stderr, Options{}); code != 2 {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, &stdout, &stderr)
	}
	var r report.ScanReport
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Coverage.Complete || r.Error == "" || len(r.Findings) != 0 || !strings.Contains(r.Error, "nodes") || !strings.Contains(strings.Join(r.Limitations, " "), "withheld") {
		t.Fatalf("resource failure hidden: %+v", r)
	}
}
