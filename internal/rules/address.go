package rules

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/Protocol-30/stellar-p30-check/internal/deps"
	"github.com/Protocol-30/stellar-p30-check/internal/report"
	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

const AddressRuleID = "P30-ADDR-001"

const (
	addressWOnly          = "w_only"
	addressSACDestination = "sac_destination"
	addressCAP84          = "https://github.com/stellar/stellar-protocol/blob/1ae39233055a0fb94bb737650a39ede8d77d924c/core/cap-0084.md"
	addressJSStrKey       = "https://github.com/stellar/js-stellar-sdk/blob/0f2fd40ba6db761390e3e5e4eb5c432234696a61/src/base/strkey.ts"
	addressJSBuilder      = "https://github.com/stellar/js-stellar-sdk/blob/0f2fd40ba6db761390e3e5e4eb5c432234696a61/src/base/transaction_builder.ts"
	addressGoStrKey       = "https://github.com/stellar/go-stellar-sdk/blob/921a5b16af9a72d973ab9ba666e96ad8bf2f6ce4/strkey/muxed_contract.go"
)

// addressCandidate is emitted only after a language analyzer proves a supported
// guard rejects W before the same parameter reaches a specific imported sink.
type addressCandidate struct {
	path      string
	line      int
	evidence  string
	parameter string
	api       string
	ecosystem string
	kind      string
}

// AnalyzeAddresses analyzes bounded source contents without executing targets.
// Limitations describe unsupported source syntax. A resource failure is an
// error, so callers cannot mistake a truncated analysis for normal completion.
func AnalyzeAddresses(files []walker.File, dependencies []deps.Dependency) ([]report.Finding, []string, error) {
	findings := []report.Finding{}
	limitations := []string{}
	ordered := slices.Clone(files)
	slices.SortFunc(ordered, func(a, b walker.File) int { return strings.Compare(a.Path, b.Path) })
	for _, file := range ordered {
		var candidates []addressCandidate
		var notes []string
		var err error
		switch file.Kind {
		case "go_source":
			candidates, notes, err = analyzeGoAddress(file)
		case "js_source", "ts_source":
			candidates, notes, err = analyzeJSAddress(file)
		default:
			continue
		}
		limitations = append(limitations, notes...)
		if err != nil {
			return findings, limitations, fmt.Errorf("address analysis %q: %w", file.Path, err)
		}
		for _, candidate := range candidates {
			finding := addressFinding(candidate, files, dependencies)
			findings = append(findings, finding)
		}
	}
	slices.Sort(limitations)
	limitations = slices.Compact(limitations)
	return findings, limitations, nil
}

func addressFinding(c addressCandidate, files []walker.File, dependencies []deps.Dependency) report.Finding {
	severity, confidence := report.SeverityMedium, report.ConfidenceMedium
	context := "SDK selection is unresolved or not verified for this source's nearest manifest boundary"
	scope, manifest := addressScope(c.path, c.ecosystem, files)
	if c.kind == addressWOnly && c.ecosystem == "npm" && manifest != "" {
		var matches []deps.Dependency
		for _, d := range dependencies {
			if d.Path == manifest && d.Scope == scope && d.Ecosystem == "npm" && d.Name == jsSDK {
				matches = append(matches, d)
			}
		}
		if len(matches) > 0 {
			verified := true
			for _, d := range matches {
				row, known := matrix[d.Ecosystem+":"+d.Name+":"+d.Version]
				if !known || row.wHelpers != supported || !d.Effective || d.Alias != "" || len(d.Uncertainty) != 0 || d.Resolution != "lockfile" || d.Source == "" {
					verified = false
				}
			}
			if verified {
				severity, confidence = report.SeverityHigh, report.ConfidenceHigh
				context = "the nearest package manifest has supported nominal lock evidence for the verified W-capable official SDK release"
			}
		}
	}
	message := fmt.Sprintf("A guard rejects W-prefixed %q immediately before %s consumes the same parameter; review the contradictory W-only input handling.", c.parameter, c.api)
	if c.kind == addressSACDestination {
		confidence = report.ConfidenceLow
		context = "SAC transfer destinations can accept W, but an account-only application policy may be intentional; no high-confidence application defect is asserted"
		message = fmt.Sprintf("A guard excludes W-prefixed %q before a SAC transfer destination; confirm whether this flow should allow muxed contract destinations.", c.parameter)
	}
	references := []string{addressCAP84}
	if c.ecosystem == "go" {
		references = append(references, addressGoStrKey)
	} else if c.kind == addressSACDestination {
		references = append(references, addressJSBuilder)
	} else {
		references = append(references, addressJSStrKey)
	}
	return report.Finding{
		RuleID: AddressRuleID, Severity: severity, Confidence: confidence, Path: c.path, Line: c.line,
		Message:     message,
		Evidence:    fmt.Sprintf("%s; parameter=%q; sink=%q; scope=%q; manifest=%q; context=%s", c.evidence, c.parameter, c.api, scope, manifest, context),
		Remediation: "For a W-only decoder, validate with the matching official muxed-contract helper rather than a C/G/M-only prefix rule. For SAC destination flows, confirm the intended address domain before changing policy. Keep account-only fields, C-only decoders, AddressObject inputs, and contract storage keys restricted as documented. Static metadata and local flow evidence do not verify installed code or all runtime behavior.",
		References:  references,
	}
}

func addressScope(source, ecosystem string, files []walker.File) (scope, manifest string) {
	name := "package.json"
	if ecosystem == "go" {
		name = "go.mod"
	}
	for dir := path.Dir(source); ; dir = path.Dir(dir) {
		wanted := path.Join(dir, name)
		for _, f := range files {
			if f.Path == wanted {
				return dir, wanted
			}
		}
		if dir == "." || dir == "/" || dir == "" {
			return "", ""
		}
	}
}
