package rules

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Protocol-30/stellar-p30-check/internal/deps"
	"github.com/Protocol-30/stellar-p30-check/internal/report"
)

func loadFixtures(t *testing.T) []struct {
	Name         string          `json:"name"`
	Input        deps.Dependency `json:"input"`
	Capabilities []string        `json:"capabilities"`
	Findings     int             `json:"findings"`
	Warnings     []string        `json:"warnings"`
} {
	t.Helper()
	data, err := os.ReadFile("testdata/capabilities.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Name         string          `json:"name"`
		Input        deps.Dependency `json:"input"`
		Capabilities []string        `json:"capabilities"`
		Findings     int             `json:"findings"`
		Warnings     []string        `json:"warnings"`
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	return fixtures
}

func TestVerifiedMatrixFixtures(t *testing.T) {
	for _, f := range loadFixtures(t) {
		t.Run(f.Name, func(t *testing.T) {
			observed, findings, warnings := Evaluate([]deps.Dependency{f.Input})
			if len(observed) != 1 || observed[0].Capabilities == nil {
				t.Fatal("missing observed capability annotation")
			}
			caps := observed[0].Capabilities
			if got := []string{caps.CAP88, caps.MuxedContractXDR, caps.WAddressHelpers}; !slices.Equal(got, f.Capabilities) {
				t.Fatalf("capabilities=%v want%v", got, f.Capabilities)
			}
			if caps.Version != f.Input.Version || caps.MatrixVersion != MatrixVersion || caps.CheckedAt != CheckedAt {
				t.Fatalf("missing provenance: %+v", caps)
			}
			if len(findings) != f.Findings {
				t.Fatalf("findings=%+v wantcount%d", findings, f.Findings)
			}
			var codes []string
			for _, w := range warnings {
				codes = append(codes, w.Code)
				if w.Path != f.Input.Path || w.Line != f.Input.Line {
					t.Fatalf("warning source lost: %+v", w)
				}
			}
			if !slices.Equal(codes, f.Warnings) {
				t.Fatalf("warnings=%+v want%v", warnings, f.Warnings)
			}
			for _, finding := range findings {
				if finding.RuleID != DependencyRuleID || finding.Severity != report.SeverityMedium || finding.Confidence != report.ConfidenceHigh {
					t.Fatalf("unexpected classification: %+v", finding)
				}
				if finding.Path != f.Input.Path || finding.Line != f.Input.Line || !strings.Contains(finding.Evidence, f.Input.Source) || !strings.Contains(finding.Evidence, f.Input.Group) {
					t.Fatalf("source/group evidence lost: %+v", finding)
				}
				if finding.Remediation == "" || len(finding.References) == 0 {
					t.Fatalf("missing explanation/evidence: %+v", finding)
				}
			}
			r := report.ScanReport{Findings: findings, Coverage: report.Coverage{Complete: true}}
			code, err := report.ExitCode(r, "high")
			if err != nil || code != 0 {
				t.Fatalf("M2 invents affected application role: exit=%d error=%v", code, err)
			}
		})
	}
}

func TestNoThresholdExtrapolation(t *testing.T) {
	for _, version := range []string{"16.3.2", "16.4.1", "17.3.1", "17.3.0-beta.1", "17.3.0+build", "v17.3.0", "latest", "^17.3.0", ""} {
		d := resolvedJS("17.3.0")
		d.Version = version
		observed, findings, warnings := Evaluate([]deps.Dependency{d})
		if len(findings) != 0 || len(warnings) != 1 || warnings[0].Code != "capability_unknown" || observed[0].Capabilities.CAP88 != unknown {
			t.Fatalf("extrapolated %q: %+v %+v", version, observed, findings)
		}
	}
}

func TestUnresolvedVersionNeverProducesFinding(t *testing.T) {
	base := resolvedJS("16.3.1")
	for _, tc := range []struct {
		name   string
		mutate func(*deps.Dependency)
	}{
		{"unresolved", func(d *deps.Dependency) { d.Effective = false }},
		{"replacement", func(d *deps.Dependency) {
			d.Effective = false
			d.Resolution = "replacement"
			d.Uncertainty = []string{"local replacement"}
		}},
		{"contradictory-parser-state", func(d *deps.Dependency) { d.Uncertainty = []string{"lock conflicts with manifest"} }},
		{"unsupported-resolution", func(d *deps.Dependency) { d.Resolution = "declared" }},
		{"missing-lock-evidence", func(d *deps.Dependency) { d.Source = "" }},
		{"missing-declaration-evidence", func(d *deps.Dependency) { d.Path = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := base
			tc.mutate(&d)
			_, findings, warnings := Evaluate([]deps.Dependency{d})
			if len(findings) != 0 || len(warnings) != 1 || warnings[0].Code != "dependency_unresolved" {
				t.Fatalf("unresolved dependency misclassified: %+v %+v", findings, warnings)
			}
		})
	}
}

func TestIdentityBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, ecosystem, alias string
		related                bool
	}{
		{"stellar-sdk", "npm", "", true},
		{"@stellar/stellar-base", "npm", "", true},
		{"stellar-base", "npm", "", true},
		{"github.com/stellar/go", "go", "", true},
		{"github.com/stellar/go-stellar-sdk/xdr", "go", "", true},
		{"github.com/acme/fork", "go", goSDK, true},
		{jsSDK, "npm", "sdk-alias", true},
		{"sdk-alias", "npm", jsSDK, true},
		{jsSDK, "go", "", true},
		{"left-pad", "npm", "", false},
		{"github.com/acme/library", "go", "", false},
	} {
		d := resolvedJS("16.3.1")
		d.Name = tc.name
		d.Ecosystem = tc.ecosystem
		d.Alias = tc.alias
		observed, findings, warnings := Evaluate([]deps.Dependency{d})
		if len(findings) != 0 || observed[0].Capabilities != nil {
			t.Fatalf("identity %s incorrectly classified: %+v %+v", tc.name, observed, findings)
		}
		if tc.related && (len(warnings) != 1 || warnings[0].Code != "unsupported_sdk") {
			t.Fatalf("missing unknown SDK warning: %+v", warnings)
		}
		if !tc.related && len(warnings) != 0 {
			t.Fatalf("unrelated dependency warned: %+v", warnings)
		}
	}
}

func TestInputImmutabilityAndRecomputedAnnotations(t *testing.T) {
	in := []deps.Dependency{resolvedJS("16.4.0"), resolvedJS("17.3.0")}
	in[0].Uncertainty = []string{"z", "a"}
	in[0].Capabilities = &deps.Capabilities{CAP88: "fake", References: []string{"z", "a"}}
	before, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	observed, _, _ := Evaluate(in)
	after, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("Evaluate mutated input")
	}
	if observed[0].Capabilities.CAP88 != "supported" {
		t.Fatal("trusted previous annotation")
	}
	observed[0].Uncertainty[0] = "changed"
	observed[0].Capabilities.References[0] = "changed"
	if in[0].Uncertainty[0] != "z" || in[0].Capabilities.References[0] != "z" {
		t.Fatal("returned state aliases input")
	}
}

func TestDeterministicOrderAndSeparateGroups(t *testing.T) {
	a, b := resolvedJS("16.4.0"), resolvedJS("16.4.0")
	a.Group = "dependencies"
	b.Group = "devDependencies"
	d1, f1, w1 := Evaluate([]deps.Dependency{a, b})
	d2, f2, w2 := Evaluate([]deps.Dependency{b, a})
	if !reflect.DeepEqual(d1, d2) || !reflect.DeepEqual(f1, f2) || !reflect.DeepEqual(w1, w2) {
		t.Fatal("permuted input changed output order")
	}
	if len(f1) != 2 || f1[0].Evidence == f1[1].Evidence {
		t.Fatalf("declaration group evidence lost: %+v", f1)
	}
}

func TestEmptyInputHasArrays(t *testing.T) {
	d, f, w := Evaluate(nil)
	if d == nil || f == nil || w == nil || len(d)+len(f)+len(w) != 0 {
		t.Fatal("expected empty arrays")
	}
}

func resolvedJS(version string) deps.Dependency {
	return deps.Dependency{
		Ecosystem: "npm", Group: "dependencies", Name: jsSDK, Scope: ".", Path: "package.json", Line: 5,
		Declared: version, Version: version, Resolution: "lockfile", Source: "package-lock.json", Effective: true, Uncertainty: []string{},
	}
}
