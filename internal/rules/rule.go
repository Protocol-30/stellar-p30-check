// Package rules defines the contract between the scanner and individual
// Protocol 30 compatibility checks.
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

// Rule is one compatibility check. Every finding it returns must pass
// report.Finding.Validate and carry file evidence and references.
type Rule interface {
	ID() string
	Check(ctx context.Context, in Input) ([]report.Finding, error)
}

// Default returns the rules shipped in this build. It is empty until the
// first rules (P30-DEP-001, P30-ADDR-001, P30-XDR-001) land with positive
// and negative fixtures; the scanner reports that no rules ran rather than
// presenting an empty result as a clean bill of health.
func Default() []Rule {
	return nil
}
