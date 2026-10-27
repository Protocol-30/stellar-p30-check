package scan

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Protocol-30/stellar-p30-check/internal/report"
	"github.com/Protocol-30/stellar-p30-check/internal/rules"
	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

var fixed = func() time.Time { return time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC) }

type fakeRule struct {
	id       string
	findings []report.Finding
	warnings []report.Warning
	err      error
	saw      *rules.Input
}

func (r fakeRule) ID() string { return r.id }
func (r fakeRule) Check(_ context.Context, in rules.Input) (rules.Output, error) {
	if r.saw != nil {
		*r.saw = in
	}
	return rules.Output{Findings: r.findings, Warnings: r.warnings}, r.err
}

func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for p, body := range files {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func codes(ws []report.Warning) []string {
	var out []string
	for _, w := range ws {
		out = append(out, w.Code+":"+w.Path)
	}
	return out
}

func TestRunWithoutRulesIsAnHonestInventory(t *testing.T) {
	root := tree(t, map[string]string{"go.mod": "module x\n", "web/package.json": "{}", "main.go": ""})
	rep, err := Run(context.Background(), Config{Target: root, ToolVersion: "t", Now: fixed})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Findings) != 0 {
		t.Errorf("findings = %v, want none", rep.Findings)
	}
	got := strings.Join(codes(rep.Warnings), " ")
	want := "check-not-implemented:go.mod check-not-implemented:web/package.json no-rules:"
	if got != want {
		t.Errorf("warnings = %q\nwant %q", got, want)
	}
	if rep.Summary.ManifestFiles != 2 || rep.Summary.SourceFiles != 1 || rep.Summary.RulesExecuted != 0 {
		t.Errorf("summary = %+v", rep.Summary)
	}
	if rep.CheckedAtUTC != "2026-10-25T00:00:00Z" {
		t.Errorf("clock not injected: %q", rep.CheckedAtUTC)
	}
}

func TestRunEmptyTreeWarns(t *testing.T) {
	rep, err := Run(context.Background(), Config{Target: t.TempDir(), Now: fixed})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(codes(rep.Warnings), " "); got != "no-relevant-files: no-rules:" {
		t.Errorf("warnings = %q", got)
	}
}

func TestRunCollectsAndValidatesRuleFindings(t *testing.T) {
	root := tree(t, map[string]string{"go.mod": "module x\n"})
	good := report.Finding{
		RuleID: "P30-DEP-001", Severity: report.SeverityHigh, Confidence: report.ConfidenceHigh,
		Path: "go.mod", Line: 1, Message: "m",
	}
	rep, err := Run(context.Background(), Config{
		Target: root, Now: fixed,
		Rules: []rules.Rule{fakeRule{id: "P30-DEP-001", findings: []report.Finding{good}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Findings) != 1 || rep.Summary.RulesExecuted != 1 {
		t.Fatalf("report = %+v", rep)
	}
	got := strings.Join(codes(rep.Warnings), " ")
	if want := "rules-not-implemented:"; got != want {
		t.Errorf("warnings = %q, want %q", got, want)
	}
	if !strings.Contains(rep.Warnings[0].Message, "P30-ADDR-001, P30-XDR-001, P30-TS-001") {
		t.Errorf("missing rules not listed: %q", rep.Warnings[0].Message)
	}

	bad := good
	bad.Severity = "critical"
	_, err = Run(context.Background(), Config{
		Target: root, Now: fixed,
		Rules: []rules.Rule{fakeRule{id: "P30-DEP-001", findings: []report.Finding{bad}}},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid finding") {
		t.Errorf("invalid finding accepted, err = %v", err)
	}
}

func TestRunMergesRuleWarnings(t *testing.T) {
	root := tree(t, map[string]string{"go.mod": ""})
	w := report.Warning{Code: "manifest-unparsed", Path: "go.mod", Message: "bad"}
	rep, err := Run(context.Background(), Config{Target: root, Now: fixed,
		Rules: []rules.Rule{fakeRule{id: "P30-DEP-001", warnings: []report.Warning{w}}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(codes(rep.Warnings), " "); got != "manifest-unparsed:go.mod rules-not-implemented:" {
		t.Errorf("warnings = %q", got)
	}
}

func TestRunMergesDuplicateWarnings(t *testing.T) {
	root := tree(t, map[string]string{"a.go": "package"})
	w := report.Warning{Code: "source-unparsed", Path: "a.go", Message: "bad"}
	rep, err := Run(context.Background(), Config{Target: root, Now: fixed, Rules: []rules.Rule{
		fakeRule{id: "P30-DEP-001", warnings: []report.Warning{w}},
		fakeRule{id: "P30-ADDR-001", warnings: []report.Warning{w}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(codes(rep.Warnings), " "); got != "rules-not-implemented: source-unparsed:a.go" {
		t.Errorf("warnings = %q", got)
	}
}

func TestRunPropagatesRuleErrorsAndCancellation(t *testing.T) {
	root := tree(t, map[string]string{"go.mod": ""})
	boom := errors.New("boom")
	if _, err := Run(context.Background(), Config{Target: root, Rules: []rules.Rule{fakeRule{id: "R", err: boom}}}); !errors.Is(err, boom) {
		t.Errorf("err = %v, want boom", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Run(ctx, Config{Target: root, Rules: []rules.Rule{fakeRule{id: "R"}}}); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestRunWalkErrorsAreScanErrors(t *testing.T) {
	if _, err := Run(context.Background(), Config{Target: filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Error("missing target accepted")
	}
	root := tree(t, map[string]string{"a.go": "", "b.go": ""})
	if _, err := Run(context.Background(), Config{Target: root, Walk: walker.Options{MaxFiles: 1}}); !errors.Is(err, walker.ErrLimit) {
		t.Errorf("err = %v, want ErrLimit", err)
	}
}

func TestReadFileIsBoundedAndRootConfined(t *testing.T) {
	outside := tree(t, map[string]string{"secret.txt": "secret"})
	root := tree(t, map[string]string{"go.mod": "module x\n", "big.go": strings.Repeat("x", 64), "sub/a.go": "package a"})
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "link.go")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	var in rules.Input
	_, err := Run(context.Background(), Config{
		Target: root, Now: fixed, Walk: walker.Options{MaxFileBytes: 32},
		Rules: []rules.Rule{fakeRule{id: "R", saw: &in}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if b, err := in.ReadFile("go.mod"); err != nil || string(b) != "module x\n" {
		t.Errorf("ReadFile(go.mod) = %q, %v", b, err)
	}
	if b, err := in.ReadFile("sub/a.go"); err != nil || string(b) != "package a" {
		t.Errorf("ReadFile(sub/a.go) = %q, %v", b, err)
	}
	if _, err := in.ReadFile("big.go"); !errors.Is(err, ErrTooLarge) {
		t.Errorf("ReadFile(big.go) err = %v, want ErrTooLarge", err)
	}
	for _, bad := range []string{"", ".", "..", "../secret.txt", "/etc/passwd", "sub/../go.mod", "./go.mod", "link.go"} {
		if b, err := in.ReadFile(bad); err == nil {
			t.Errorf("ReadFile(%q) = %q, want error", bad, b)
		}
	}
}
