// Package deps inventories dependency declarations without running target code
// or consulting the network. Observed versions are not necessarily effective.
package deps

import (
	"cmp"
	"errors"
	"slices"

	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

// Dependency separates observed declarations from supported static selection.
// Effective means nominal lock/declaration agreement under the documented
// registry assumptions, not installed-artifact or actual-registry verification.
// Capabilities describe the named official Version, never a target runtime.
type Dependency struct {
	Ecosystem    string        `json:"ecosystem"`
	Group        string        `json:"group"`
	Name         string        `json:"name"`
	Alias        string        `json:"alias,omitempty"`
	Scope        string        `json:"scope"`
	Path         string        `json:"path"`
	Line         int           `json:"line,omitempty"`
	Declared     string        `json:"declared"`
	Version      string        `json:"version"`
	Resolution   string        `json:"resolution"`
	Source       string        `json:"source"`
	Effective    bool          `json:"effective"`
	Uncertainty  []string      `json:"uncertainty"`
	Capabilities *Capabilities `json:"capabilities,omitempty"`
}

type Capabilities struct {
	MatrixVersion    string   `json:"matrix_version"`
	CheckedAt        string   `json:"checked_at"`
	Version          string   `json:"version"`
	CAP88            string   `json:"cap88"`
	MuxedContractXDR string   `json:"muxed_contract_xdr"`
	WAddressHelpers  string   `json:"w_address_helpers"`
	References       []string `json:"references"`
}

type Warning struct {
	Path    string `json:"path"`
	Line    int    `json:"line,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Inventory struct {
	Dependencies []Dependency
	Warnings     []Warning
}

// Inspect uses only the already-bounded discovery contents. Errors preserve
// partial inventory and make the caller's scan incomplete.
func Inspect(files []walker.File) (Inventory, error) {
	goDeps, goWarnings, goErr := inspectGo(files)
	jsDeps, jsWarnings, jsErr := inspectJS(files)
	i := Inventory{
		Dependencies: append(goDeps, jsDeps...),
		Warnings:     append(goWarnings, jsWarnings...),
	}
	Normalize(&i)
	return i, errors.Join(goErr, jsErr)
}

// Normalize keeps machine-readable inventory deterministic, including arrays.
func Normalize(i *Inventory) {
	if i.Dependencies == nil {
		i.Dependencies = []Dependency{}
	}
	if i.Warnings == nil {
		i.Warnings = []Warning{}
	}
	for n := range i.Dependencies {
		d := &i.Dependencies[n]
		if d.Uncertainty == nil {
			d.Uncertainty = []string{}
		}
		slices.Sort(d.Uncertainty)
		d.Uncertainty = slices.Compact(d.Uncertainty)
		if d.Capabilities != nil {
			if d.Capabilities.References == nil {
				d.Capabilities.References = []string{}
			}
			slices.Sort(d.Capabilities.References)
		}
	}
	slices.SortFunc(i.Dependencies, func(a, b Dependency) int {
		for _, c := range []int{cmp.Compare(a.Path, b.Path), cmp.Compare(a.Line, b.Line), cmp.Compare(a.Name, b.Name), cmp.Compare(a.Group, b.Group), cmp.Compare(a.Alias, b.Alias), cmp.Compare(a.Declared, b.Declared), cmp.Compare(a.Source, b.Source), cmp.Compare(a.Version, b.Version), cmp.Compare(a.Resolution, b.Resolution)} {
			if c != 0 {
				return c
			}
		}
		return slices.Compare(a.Uncertainty, b.Uncertainty)
	})
	slices.SortFunc(i.Warnings, func(a, b Warning) int {
		for _, c := range []int{cmp.Compare(a.Path, b.Path), cmp.Compare(a.Line, b.Line), cmp.Compare(a.Code, b.Code), cmp.Compare(a.Message, b.Message)} {
			if c != 0 {
				return c
			}
		}
		return 0
	})
}
