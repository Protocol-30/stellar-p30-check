package rules

import (
	"testing"

	"github.com/Protocol-30/stellar-p30-check/internal/deps"
	"github.com/Protocol-30/stellar-p30-check/internal/report"
	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

func TestAddressFindingNeedsNearestVerifiedSelection(t *testing.T) {
	candidate := addressCandidate{path: "src/decoder.ts", line: 4, parameter: "address", api: "StrKey.decodeMuxedContract", ecosystem: "npm", kind: addressWOnly, evidence: "literal guard before W-only decoder"}
	verified := deps.Dependency{Ecosystem: "npm", Group: "dependencies", Name: jsSDK, Scope: ".", Path: "package.json", Declared: "17.3.0", Version: "17.3.0", Resolution: "lockfile", Source: "package-lock.json", Effective: true}
	for _, tc := range []struct {
		name           string
		mutate         func(*addressCandidate, *deps.Dependency)
		files          []walker.File
		wantSeverity   report.Severity
		wantConfidence report.Confidence
	}{
		{"verified", func(*addressCandidate, *deps.Dependency) {}, []walker.File{{Path: "package.json"}}, report.SeverityHigh, report.ConfidenceHigh},
		{"no-manifest", func(*addressCandidate, *deps.Dependency) {}, nil, report.SeverityMedium, report.ConfidenceMedium},
		{"nested-boundary", func(c *addressCandidate, d *deps.Dependency) { c.path = "child/src/decoder.ts" }, []walker.File{{Path: "package.json"}, {Path: "child/package.json"}}, report.SeverityMedium, report.ConfidenceMedium},
		{"unresolved", func(c *addressCandidate, d *deps.Dependency) { d.Effective = false }, []walker.File{{Path: "package.json"}}, report.SeverityMedium, report.ConfidenceMedium},
		{"uncertain", func(c *addressCandidate, d *deps.Dependency) { d.Uncertainty = []string{"replacement"} }, []walker.File{{Path: "package.json"}}, report.SeverityMedium, report.ConfidenceMedium},
		{"alias", func(c *addressCandidate, d *deps.Dependency) { d.Alias = "sdk" }, []walker.File{{Path: "package.json"}}, report.SeverityMedium, report.ConfidenceMedium},
		{"fork", func(c *addressCandidate, d *deps.Dependency) { d.Alias = jsSDK; d.Name = "example-fork" }, []walker.File{{Path: "package.json"}}, report.SeverityMedium, report.ConfidenceMedium},
		{"unknown-release", func(c *addressCandidate, d *deps.Dependency) { d.Version = "99.0.0" }, []walker.File{{Path: "package.json"}}, report.SeverityMedium, report.ConfidenceMedium},
		{"missing-helper", func(c *addressCandidate, d *deps.Dependency) {
			d.Version = "16.4.0"
			d.Capabilities = &deps.Capabilities{WAddressHelpers: "supported"}
		}, []walker.File{{Path: "package.json"}}, report.SeverityMedium, report.ConfidenceMedium},
		{"no-lock-source", func(c *addressCandidate, d *deps.Dependency) { d.Source = "" }, []walker.File{{Path: "package.json"}}, report.SeverityMedium, report.ConfidenceMedium},
		{"declaration-only", func(c *addressCandidate, d *deps.Dependency) { d.Resolution = "declared" }, []walker.File{{Path: "package.json"}}, report.SeverityMedium, report.ConfidenceMedium},
		{"go-unresolved", func(c *addressCandidate, d *deps.Dependency) { c.ecosystem = "go"; c.path = "src/main.go" }, []walker.File{{Path: "go.mod"}}, report.SeverityMedium, report.ConfidenceMedium},
		{"broader-SAC-policy", func(c *addressCandidate, d *deps.Dependency) { c.kind = addressSACDestination }, []walker.File{{Path: "package.json"}}, report.SeverityMedium, report.ConfidenceLow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, d := candidate, verified
			tc.mutate(&c, &d)
			f := addressFinding(c, tc.files, []deps.Dependency{d})
			if f.Severity != tc.wantSeverity || f.Confidence != tc.wantConfidence || f.RuleID != AddressRuleID || f.Path != c.path || f.Line != 4 || f.Evidence == "" || len(f.References) == 0 {
				t.Fatalf("incorrect finding: %+v", f)
			}
		})
	}
}

func TestAddressFindingConflictingGroupsCannotGateHigh(t *testing.T) {
	c := addressCandidate{path: "decoder.ts", ecosystem: "npm", kind: addressWOnly, api: "StrKey.decodeMuxedContract"}
	d := deps.Dependency{Ecosystem: "npm", Name: jsSDK, Scope: ".", Path: "package.json", Version: "17.3.0", Resolution: "lockfile", Source: "package-lock.json", Effective: true}
	other := d
	other.Group = "peerDependencies"
	other.Version = "16.4.0"
	f := addressFinding(c, []walker.File{{Path: "package.json"}}, []deps.Dependency{d, other})
	if f.Severity == report.SeverityHigh || f.Confidence == report.ConfidenceHigh {
		t.Fatalf("conflicting groups promoted: %+v", f)
	}
}
