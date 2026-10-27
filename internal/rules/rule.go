// Package rules defines the contract between the scanner and individual
// Protocol 30 compatibility checks, and the checks themselves.
package rules

import (
	"context"

	"github.com/Protocol-30/stellar-p30-check/internal/report"
	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

// Input is what a rule sees: the discovered inventory and a bounded,
// root-confined reader. Rules must treat file contents as untrusted text and
// never execute or evaluate them.
type Input struct {
	Files    []walker.File
	ReadFile func(relPath string) ([]byte, error)
}

// Output is what a rule returns. Warnings describe coverage gaps such as a
// manifest that could not be parsed; they are not findings.
type Output struct {
	Findings []report.Finding
	Warnings []report.Warning
}

// Rule is one compatibility check. Every finding it returns must pass
// report.Finding.Validate and carry file evidence and references. An error
// means the rule itself failed; problems with individual scanned files are
// warnings, so one malformed file cannot abort the scan.
type Rule interface {
	ID() string
	Check(ctx context.Context, in Input) (Output, error)
}

// Planned lists every rule ID on the roadmap (docs/roadmap.md). The scanner
// warns about the ones a build does not include.
var Planned = []string{"P30-DEP-001", "P30-ADDR-001", "P30-XDR-001", "P30-TS-001"}

// Default returns the rules shipped in this build.
func Default() []Rule {
	return []Rule{Dependencies{}, Addresses{}, LedgerXDR{}, CloseTimes{}}
}
