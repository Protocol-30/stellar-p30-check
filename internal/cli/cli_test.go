package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Protocol-30/stellar-p30-check/internal/report"
	"github.com/Protocol-30/stellar-p30-check/internal/rules"
)

var update = flag.Bool("update", false, "rewrite golden files in testdata/golden")

const fixtures = "../../testdata/fixtures"

var fixedNow = func() time.Time { return time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC) }

type injected []report.Finding

func (injected) ID() string { return "P30-TEST-001" }
func (f injected) Check(context.Context, rules.Input) (rules.Output, error) {
	return rules.Output{Findings: f}, nil
}

func run(t *testing.T, rs []rules.Rule, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = Run(context.Background(), args, Env{Stdout: &out, Stderr: &errb, Now: fixedNow, Rules: rs})
	return code, out.String(), errb.String()
}

func finding(sev report.Severity, conf report.Confidence) report.Finding {
	return report.Finding{
		RuleID: "P30-TEST-001", Severity: sev, Confidence: conf,
		Path: "go.mod", Line: 1, Message: "injected for exit-code tests",
	}
}

func TestExitCodePolicy(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name     string
		findings []report.Finding
		failOn   string
		want     int
	}{
		{"no findings, gate off", nil, "none", ExitOK},
		{"no findings, gate on", nil, "high", ExitOK},
		{"high/high, gate off", []report.Finding{finding(report.SeverityHigh, report.ConfidenceHigh)}, "none", ExitOK},
		{"high/high, gate on", []report.Finding{finding(report.SeverityHigh, report.ConfidenceHigh)}, "high", ExitGated},
		{"high/medium, gate on", []report.Finding{finding(report.SeverityHigh, report.ConfidenceMedium)}, "high", ExitOK},
		{"medium/high, gate on", []report.Finding{finding(report.SeverityMedium, report.ConfidenceHigh)}, "high", ExitOK},
		{"info/low, gate on", []report.Finding{finding(report.SeverityInfo, report.ConfidenceLow)}, "high", ExitOK},
		{"mixed, gate on", []report.Finding{
			finding(report.SeverityMedium, report.ConfidenceHigh),
			finding(report.SeverityHigh, report.ConfidenceHigh),
		}, "high", ExitGated},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, _, stderr := run(t, []rules.Rule{injected(c.findings)}, "scan", root, "--fail-on", c.failOn)
			if code != c.want {
				t.Errorf("exit = %d, want %d (stderr %q)", code, c.want, stderr)
			}
		})
	}
}

func TestDefaultGateIsOff(t *testing.T) {
	code, _, _ := run(t, []rules.Rule{injected{finding(report.SeverityHigh, report.ConfidenceHigh)}}, "scan", t.TempDir())
	if code != ExitOK {
		t.Errorf("exit = %d, want 0 without --fail-on", code)
	}
}

func TestUsageErrors(t *testing.T) {
	dir := t.TempDir()
	cases := map[string][]string{
		"no args":           {},
		"unknown command":   {"audit", dir},
		"no path":           {"scan"},
		"two paths":         {"scan", dir, dir},
		"bad format":        {"scan", dir, "--format", "sarif"},
		"bad fail-on":       {"scan", dir, "--fail-on", "medium"},
		"unknown flag":      {"scan", dir, "--deep"},
		"missing flag arg":  {"scan", dir, "--format"},
		"zero max bytes":    {"scan", dir, "--max-file-bytes", "0"},
		"negative max":      {"scan", dir, "--max-files", "-1"},
		"non-numeric limit": {"scan", dir, "--max-files", "lots"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := run(t, nil, args...)
			if code != ExitError {
				t.Errorf("exit = %d, want %d", code, ExitError)
			}
			if stdout != "" {
				t.Errorf("usage error wrote to stdout: %q", stdout)
			}
			if stderr == "" {
				t.Error("usage error printed nothing to stderr")
			}
		})
	}
}

func TestScanFailuresExit2(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file.go")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{
		"missing path": filepath.Join(t.TempDir(), "nope"),
		"file path":    file,
		"nul byte":     "bad\x00path",
	} {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := run(t, nil, "scan", path)
			if code != ExitError || stdout != "" || !strings.Contains(stderr, "scan failed") {
				t.Errorf("exit=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
		})
	}

	dir := t.TempDir()
	for _, n := range []string{"a.go", "b.go"} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	code, stdout, stderr := run(t, nil, "scan", dir, "--max-files", "1")
	if code != ExitError || stdout != "" || !strings.Contains(stderr, "no partial report") {
		t.Errorf("limit: exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestHelpAndVersion(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}, {"scan", "--help"}} {
		code, stdout, _ := run(t, nil, args...)
		if code != ExitOK || !strings.Contains(stdout, "p30check scan <path>") {
			t.Errorf("%v: exit=%d stdout=%q", args, code, stdout)
		}
	}
	code, stdout, _ := run(t, nil, "version")
	if code != ExitOK || stdout != "p30check "+Version+" (report schema 1)\n" {
		t.Errorf("version: exit=%d stdout=%q", code, stdout)
	}
}

func TestFlagsBeforeOrAfterPath(t *testing.T) {
	dir := filepath.Join(fixtures, "go-module")
	_, after, _ := run(t, nil, "scan", dir, "--format", "json")
	_, before, _ := run(t, nil, "scan", "--format=json", dir)
	if after != before || !json.Valid([]byte(after)) {
		t.Errorf("flag position changed output:\n%s\n---\n%s", after, before)
	}
	code, _, stderr := run(t, nil, "scan", "--", "--format")
	if code != ExitError || !strings.Contains(stderr, "scan failed") {
		t.Errorf("'--' should make the next argument a path: exit=%d stderr=%q", code, stderr)
	}
}

// TestGoldenReports pins the full text and JSON output for the discovery
// fixtures. Run "go test ./internal/cli -update" after an intended change.
func TestGoldenReports(t *testing.T) {
	targets := map[string]string{ // golden name -> target relative to the repository root
		"go-module":              "testdata/fixtures/go-module",
		"js-app":                 "testdata/fixtures/js-app",
		"integration-legacy-app": "testdata/integration/legacy-app",
		"integration-modern-app": "testdata/integration/modern-app",
	}
	for fixture, target := range targets {
		for _, format := range []string{"text", "json"} {
			t.Run(fixture+"/"+format, func(t *testing.T) {
				// Run from the repository root so the echoed target is
				// relative and identical on every machine.
				t.Chdir("../..")
				code, stdout, stderr := run(t, nil, "scan", target, "--format", format)
				if code != ExitOK || stderr != "" {
					t.Fatalf("exit=%d stderr=%q", code, stderr)
				}
				_, again, _ := run(t, nil, "scan", target, "--format", format)
				if stdout != again {
					t.Fatal("two scans of the same fixture differ")
				}

				ext := map[string]string{"text": ".txt", "json": ".json"}[format]
				golden := filepath.Join("testdata", "golden", fixture+ext)
				if *update {
					if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(golden, []byte(stdout), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				want, err := os.ReadFile(golden)
				if err != nil {
					t.Fatalf("read golden (run with -update to create): %v", err)
				}
				if stdout != string(want) {
					t.Errorf("output differs from %s; rerun with -update if intended\n--- got\n%s", golden, stdout)
				}
			})
		}
	}
}

// TestIntegrationApps scans two small projects end to end with the default
// rule set: the legacy one must trip every rule and the CI gate, the modern
// one must produce no findings or warnings at all.
func TestIntegrationApps(t *testing.T) {
	t.Chdir("../..")
	type rep struct {
		Findings []report.Finding `json:"findings"`
		Warnings []report.Warning `json:"warnings"`
	}
	scan := func(target string) (int, rep) {
		code, stdout, stderr := run(t, nil, "scan", target, "--format", "json", "--fail-on", "high")
		var r rep
		if err := json.Unmarshal([]byte(stdout), &r); err != nil {
			t.Fatalf("%s: bad JSON (stderr %q): %v", target, stderr, err)
		}
		return code, r
	}

	code, legacy := scan("testdata/integration/legacy-app")
	if code != ExitGated {
		t.Errorf("legacy-app exit = %d, want %d", code, ExitGated)
	}
	byRule := map[string]int{}
	for _, f := range legacy.Findings {
		byRule[f.RuleID]++
	}
	for _, id := range rules.Planned {
		if byRule[id] == 0 {
			t.Errorf("legacy-app: no %s finding; got %v", id, byRule)
		}
	}

	code, modern := scan("testdata/integration/modern-app")
	if code != ExitOK || len(modern.Findings) != 0 || len(modern.Warnings) != 0 {
		t.Errorf("modern-app: exit=%d findings=%+v warnings=%+v", code, modern.Findings, modern.Warnings)
	}
}

// TestSelfScan scans this repository with the default rules. Everything
// reported must come from the deliberate fixtures under testdata/; the
// scanner's own code must be clean.
func TestSelfScan(t *testing.T) {
	t.Chdir("../..")
	code, stdout, stderr := run(t, nil, "scan", ".", "--format", "json")
	if code != ExitOK {
		t.Fatalf("exit=%d stderr=%q", code, stderr)
	}
	var r struct {
		Findings []report.Finding `json:"findings"`
		Warnings []report.Warning `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(stdout), &r); err != nil {
		t.Fatal(err)
	}
	for _, f := range r.Findings {
		if !strings.HasPrefix(f.Path, "testdata/") {
			t.Errorf("finding outside testdata: %s:%d %s", f.Path, f.Line, f.Message)
		}
	}
	for _, w := range r.Warnings {
		if w.Path != "" && !strings.HasPrefix(w.Path, "testdata/") {
			t.Errorf("warning outside testdata: %+v", w)
		}
	}
}
