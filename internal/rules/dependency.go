// Package rules applies evidence-backed static checks to dependency inventories.
// It does not execute targets or contact package registries during a scan.
package rules

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/Protocol-30/stellar-p30-check/internal/deps"
	"github.com/Protocol-30/stellar-p30-check/internal/report"
)

const (
	// MatrixVersion identifies a deliberately small set of exact, checked releases.
	MatrixVersion    = "2026-10-25.1"
	CheckedAt        = "2026-10-25"
	DependencyRuleID = "P30-DEP-001"
	supported        = "supported"
	unsupported      = "unsupported"
	unknown          = "unknown"
	jsSDK            = "@stellar/stellar-sdk"
	goSDK            = "github.com/stellar/go-stellar-sdk"
)

const (
	js164        = "https://github.com/stellar/js-stellar-sdk/blob/v16.4.0/CHANGELOG.md"
	js173        = "https://github.com/stellar/js-stellar-sdk/blob/v17.3.0/CHANGELOG.md"
	go080        = "https://github.com/stellar/go-stellar-sdk/blob/v0.8.0/CHANGELOG.md"
	js163XDR     = "https://github.com/stellar/js-stellar-sdk/blob/v16.3.1/src/base/generated/curr_generated.js"
	js163Address = "https://github.com/stellar/js-stellar-sdk/blob/v16.3.1/src/base/address.ts"
	js163Strkey  = "https://github.com/stellar/js-stellar-sdk/blob/v16.3.1/src/base/strkey.ts"
)

type matrixRow struct {
	cap88, muxedXDR, wHelpers string
	references                []string
}

// No semver threshold is inferred from these exact rows. A newer, older,
// prerelease, build-metadata, community, or legacy release remains unknown.
var matrix = map[string]matrixRow{
	"npm:" + jsSDK + ":16.3.1": {unsupported, unsupported, unsupported, []string{js163XDR, js163Address, js163Strkey}},
	"npm:" + jsSDK + ":16.4.0": {supported, supported, unsupported, []string{js164}},
	"npm:" + jsSDK + ":17.3.0": {supported, supported, supported, []string{js173}},
	"go:" + goSDK + ":v0.8.0":  {supported, supported, supported, []string{go080}},
}

// Evaluate annotates observed versions and checks known missing capabilities.
// Effective is a prerequisite for a finding, not proof of affected runtime use.
// M2 therefore reports missing capabilities as medium severity; it cannot
// justify a high-severity application-impact gate without later source context.
// Returned dependencies own their slices/pointers and never alias input state.
func Evaluate(input []deps.Dependency) (dependencies []deps.Dependency, findings []report.Finding, warnings []deps.Warning) {
	dependencies = make([]deps.Dependency, len(input))
	findings = []report.Finding{}
	warnings = []deps.Warning{}
	for n, original := range input {
		d := cloneDependency(original)
		// Capability annotations are recomputed from this matrix. Never trust a
		// caller-supplied previous annotation for a changed identity or version.
		d.Capabilities = nil
		dependencies[n] = d
		if !isOfficial(d) {
			if isRelatedSDK(d) {
				warnings = append(warnings, warning(d, "unsupported_sdk", "SDK identity, alias, or legacy family has no verified capability matrix; no compatibility classification was made."))
			}
			continue
		}
		row, known := matrix[d.Ecosystem+":"+d.Name+":"+d.Version]
		caps := &deps.Capabilities{
			MatrixVersion: MatrixVersion, CheckedAt: CheckedAt, Version: d.Version,
			CAP88: unknown, MuxedContractXDR: unknown, WAddressHelpers: unknown,
			References: []string{},
		}
		if known {
			caps.CAP88, caps.MuxedContractXDR, caps.WAddressHelpers = row.cap88, row.muxedXDR, row.wHelpers
			caps.References = slices.Clone(row.references)
		}
		dependencies[n].Capabilities = caps
		if !known {
			warnings = append(warnings, warning(d, "capability_unknown", fmt.Sprintf("Observed SDK version %q is not an exact release in capability matrix %s; no supported/unsupported threshold was inferred.", d.Version, MatrixVersion)))
		}
		// M2's effective selections are supported direct lock records. Keep
		// the source/selection contract explicit even for callers other than
		// the CLI, rather than relying on an isolated boolean.
		if !d.Effective || len(d.Uncertainty) != 0 || d.Resolution != "lockfile" || d.Source == "" || d.Path == "" {
			warnings = append(warnings, warning(d, "dependency_unresolved", "The observed SDK version is not a verified effective selection. Capability annotations describe only that observed version; no dependency finding was emitted."))
			continue
		}
		if !known {
			continue
		}
		missing := []string{}
		if row.cap88 == unsupported {
			missing = append(missing, "CAP-88 millisecond ledger XDR")
		}
		if row.muxedXDR == unsupported {
			missing = append(missing, "muxed-contract XDR")
		}
		if row.wHelpers == unsupported {
			missing = append(missing, "high-level W address helpers")
		}
		if len(missing) == 0 {
			continue
		}
		refs := slices.Clone(row.references)
		if d.Ecosystem == "npm" {
			refs = append(refs, js173)
		}
		refs = append(refs, "https://github.com/stellar/stellar-protocol/blob/master/core/cap-0084.md")
		slices.Sort(refs)
		refs = slices.Compact(refs)
		findings = append(findings, report.Finding{
			RuleID:     DependencyRuleID,
			Severity:   report.SeverityMedium,
			Confidence: report.ConfidenceHigh,
			Path:       d.Path, Line: d.Line,
			Message:     fmt.Sprintf("Locked metadata selects %s %s; that official release lacks %s. Review whether this application requires those capabilities.", d.Name, d.Version, strings.Join(missing, ", ")),
			Evidence:    fmt.Sprintf("scope=%q; group=%q; package=%q; declared=%q; observed=%q; resolution=%q; source=%q; matrix=%s", d.Scope, d.Group, d.Name, d.Declared, d.Version, d.Resolution, d.Source, MatrixVersion),
			Remediation: "For affected flows, use a release whose required capabilities are verified (the matrix verifies @stellar/stellar-sdk 17.3.0). Recheck lockfile selection and application-specific handling after upgrading. W helpers are required only in W-capable address flows; account-only fields, AddressObject inputs, and contract storage keys must retain their documented restrictions. A dependency capability gap does not establish runtime breakage.",
			References:  refs,
		})
	}
	inv := deps.Inventory{Dependencies: dependencies, Warnings: warnings}
	deps.Normalize(&inv)
	dependencies, warnings = inv.Dependencies, inv.Warnings
	slices.SortFunc(findings, func(a, b report.Finding) int {
		for _, c := range []int{cmp.Compare(a.Path, b.Path), cmp.Compare(a.Line, b.Line), cmp.Compare(a.RuleID, b.RuleID), cmp.Compare(a.Message, b.Message), cmp.Compare(a.Evidence, b.Evidence)} {
			if c != 0 {
				return c
			}
		}
		return slices.Compare(a.References, b.References)
	})
	return dependencies, findings, warnings
}

func cloneDependency(d deps.Dependency) deps.Dependency {
	d.Uncertainty = slices.Clone(d.Uncertainty)
	if d.Uncertainty == nil {
		d.Uncertainty = []string{}
	}
	if d.Capabilities != nil {
		c := *d.Capabilities
		c.References = slices.Clone(c.References)
		d.Capabilities = &c
	}
	return d
}

func isOfficial(d deps.Dependency) bool {
	if d.Alias != "" {
		return false
	}
	return (d.Ecosystem == "npm" && d.Name == jsSDK) || (d.Ecosystem == "go" && d.Name == goSDK)
}

func isRelatedSDK(d deps.Dependency) bool {
	for _, name := range []string{d.Name, d.Alias} {
		if name == jsSDK || name == goSDK || name == "stellar-sdk" || name == "@stellar/stellar-base" || name == "stellar-base" || name == "github.com/stellar/go" || strings.HasPrefix(name, goSDK+"/") || strings.HasPrefix(name, "github.com/stellar/go/") {
			return true
		}
	}
	return false
}

func warning(d deps.Dependency, code, message string) deps.Warning {
	return deps.Warning{Path: d.Path, Line: d.Line, Code: code, Message: fmt.Sprintf("%s (package %q, scope %q, group %q)", message, d.Name, d.Scope, d.Group)}
}
