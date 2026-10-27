package rules

import (
	"context"
	"fmt"

	"github.com/Protocol-30/stellar-p30-check/internal/deps"
	"github.com/Protocol-30/stellar-p30-check/internal/report"
)

// DependenciesID is the rule ID of the SDK version check.
const DependenciesID = "P30-DEP-001"

// Dependencies (P30-DEP-001) compares the Stellar SDK versions a project
// declares or installs against the sourced capability matrix in
// internal/deps.
//
// Confidence follows how the version was learned: an exact version from
// go.mod, an npm lockfile or an exact package.json pin is high; the lowest
// version a package.json range admits is low, because the installed
// version depends on a lockfile the scanner did not read.
type Dependencies struct{}

// ID implements Rule.
func (Dependencies) ID() string { return DependenciesID }

// Check implements Rule.
func (Dependencies) Check(ctx context.Context, in Input) (Output, error) {
	uses, ws, err := collectSDKUses(ctx, in, DependenciesID)
	if err != nil {
		return Output{}, err
	}
	out := Output{Warnings: ws}
	for _, u := range uses {
		switch u.Kind {
		case useExact, useRange:
			conf := report.ConfidenceHigh
			if u.Kind == useRange {
				conf = report.ConfidenceLow
			}
			a, _ := deps.Assess(u.Eco, u.Name, u.Parsed)
			if f, ok := fromAssessment(a, u.Subject, conf, u.File, u.Line, u.Evidence); ok {
				out.Findings = append(out.Findings, f)
			}
		case useUnresolved:
			out.Findings = append(out.Findings, unresolvedFinding(u.Name, u.File, u.Line, u.Evidence, u.Why))
		}
	}
	return out, nil
}

// fromAssessment turns a matrix verdict into a finding. ok is false when
// the version supports both Protocol 30 capabilities.
func fromAssessment(a deps.Assessment, subject string, conf report.Confidence, file string, line int, evidence string) (report.Finding, bool) {
	f := report.Finding{
		RuleID: DependenciesID, Confidence: conf, Path: file, Line: line,
		Evidence: evidence, Remediation: a.Remediation, References: a.References,
	}
	switch {
	case a.LedgerXDR == deps.Yes && a.MuxedContract == deps.Yes:
		return report.Finding{}, false
	case a.LedgerXDR == deps.No:
		f.Severity = report.SeverityHigh
		f.Message = fmt.Sprintf("%s: no Protocol 30 support; %s", subject, a.Basis)
	case a.LedgerXDR == deps.Yes && a.MuxedContract == deps.No:
		f.Severity = report.SeverityMedium
		f.Message = fmt.Sprintf("%s: decodes Protocol 30 ledgers but cannot handle W muxed-contract addresses (CAP-84)", subject)
	default:
		f.Severity = report.SeverityInfo
		f.Confidence = report.ConfidenceLow
		f.Message = fmt.Sprintf("%s: Protocol 30 support could not be verified; %s", subject, a.Basis)
	}
	return f, true
}

func unresolvedFinding(name, file string, line int, evidence, why string) report.Finding {
	return report.Finding{
		RuleID: DependenciesID, Severity: report.SeverityInfo, Confidence: report.ConfidenceLow,
		Path: file, Line: line, Evidence: evidence,
		Message:     fmt.Sprintf("%s: Protocol 30 support could not be checked; %s", name, why),
		Remediation: "Confirm the resolved version supports Protocol 30: @stellar/stellar-sdk 17.3.0+ or github.com/stellar/go-stellar-sdk v0.8.0+.",
		References:  []string{deps.RefUpgrade, deps.RefCAP88, deps.RefCAP84},
	}
}
