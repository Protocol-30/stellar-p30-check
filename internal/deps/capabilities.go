package deps

// Ecosystem names a package registry.
type Ecosystem string

const (
	Go  Ecosystem = "go"
	NPM Ecosystem = "npm"
)

// Support is whether a package version has a Protocol 30 capability.
type Support int

const (
	Unknown Support = iota
	No
	Yes
)

// Assessment is the matrix verdict for one package version.
type Assessment struct {
	// LedgerXDR: the XDR definitions include the CAP-88 StellarValue arms,
	// so ledger headers closed after the upgrade decode.
	LedgerXDR Support
	// MuxedContract: CAP-84 W strkey encode/decode support.
	MuxedContract Support
	Basis         string // why, in one sentence
	Remediation   string
	References    []string
}

// Reference URLs shared by findings.
const (
	RefCAP84       = "https://github.com/stellar/stellar-protocol/blob/master/core/cap-0084.md"
	RefCAP88       = "https://github.com/stellar/stellar-protocol/blob/master/core/cap-0088.md"
	RefUpgrade     = "https://stellar.org/blog/developers/bell-protocol-30-upgrade-guide"
	RefGoChangelog = "https://github.com/stellar/go-stellar-sdk/blob/v0.8.0/CHANGELOG.md"
	RefJSSDK       = "https://www.npmjs.com/package/@stellar/stellar-sdk"
)

// matrix maps each tracked package to its verdict function. Every version
// boundary is recorded in docs/evidence.md (rows E10-E14).
var matrix = map[Ecosystem]map[string]func(Version) Assessment{
	Go: {
		"github.com/stellar/go-stellar-sdk": goSDK,
		"github.com/stellar/go":             goLegacy,
	},
	NPM: {
		"@stellar/stellar-sdk":  jsSDK,
		"@stellar/stellar-base": jsBase,
		"stellar-sdk":           unscoped("stellar-sdk", "13.3.0"),
		"stellar-base":          unscoped("stellar-base", "13.1.0"),
	},
}

// Tracked reports whether the matrix covers a package.
func Tracked(eco Ecosystem, name string) bool {
	_, ok := matrix[eco][name]
	return ok
}

// Assess returns the matrix verdict for an exact version of a tracked
// package. ok is false for packages the matrix does not cover.
func Assess(eco Ecosystem, name string, v Version) (a Assessment, ok bool) {
	assess, ok := matrix[eco][name]
	if !ok {
		return Assessment{}, false
	}
	return assess(v), true
}

var (
	goSDKFirstP30   = must("0.8.0")
	jsXDRFirst16    = must("16.4.0")
	jsLine17        = must("17.0.0")
	jsFirstFull     = must("17.3.0")
	jsBaseLastKnown = must("15.0.0")
)

const goFix = "Upgrade to github.com/stellar/go-stellar-sdk v0.8.0 or later (it requires Go 1.25.0 or newer)."

// goSDK: v0.8.0 is the first release with Protocol 30 XDR and the W strkey;
// v0.7.3 and earlier releases lack both (E10).
func goSDK(v Version) Assessment {
	refs := []string{RefGoChangelog, RefCAP88, RefCAP84}
	switch {
	case !v.Less(goSDKFirstP30):
		return Assessment{LedgerXDR: Yes, MuxedContract: Yes, References: refs,
			Basis: "v0.8.0 added Protocol 30 XDR and the W muxed-contract strkey"}
	case v.IsPrerelease():
		return Assessment{LedgerXDR: Unknown, MuxedContract: Unknown, References: refs,
			Basis:       "pre-release or pseudo-version older than v0.8.0; it may or may not include the Protocol 30 changes",
			Remediation: goFix}
	}
	return Assessment{LedgerXDR: No, MuxedContract: No, References: refs,
		Basis:       "releases before v0.8.0 lack the CAP-88 ledger close-value XDR arms and the CAP-84 W strkey, so they cannot decode ledger headers closed after the upgrade",
		Remediation: goFix}
}

// goLegacy: github.com/stellar/go is deprecated in favour of
// go-stellar-sdk; its latest version (2025-12-10) has no Protocol 30 support (E11).
func goLegacy(Version) Assessment {
	return Assessment{LedgerXDR: No, MuxedContract: No,
		References:  []string{RefGoChangelog, RefCAP88, RefCAP84},
		Basis:       "github.com/stellar/go is deprecated and its final version has no Protocol 30 XDR or W strkey support",
		Remediation: "Migrate to github.com/stellar/go-stellar-sdk v0.8.0 or later (it requires Go 1.25.0 or newer)."}
}

const jsFix = "Upgrade to @stellar/stellar-sdk 17.3.0 or later."

// jsSDK: 16.4.x has Protocol 30 XDR but no W strkey helpers; 17.0.0-17.2.1
// and everything before 16.4.0 lack both; 17.3.0+ has both (E12).
func jsSDK(v Version) Assessment {
	refs := []string{RefUpgrade, RefCAP88, RefCAP84, RefJSSDK}
	switch {
	case !v.Less(jsFirstFull):
		return Assessment{LedgerXDR: Yes, MuxedContract: Yes, References: refs,
			Basis: "17.3.0 added Protocol 30 XDR and W muxed-contract strkey support"}
	case v.IsPrerelease():
		return Assessment{LedgerXDR: Unknown, MuxedContract: Unknown, References: refs,
			Basis: "pre-release older than 17.3.0; its Protocol 30 support was not verified", Remediation: jsFix}
	case !v.Less(jsXDRFirst16) && v.Less(jsLine17):
		return Assessment{LedgerXDR: Yes, MuxedContract: No, References: refs,
			Basis:       "the 16.4.x maintenance line decodes Protocol 30 ledgers but has no W muxed-contract strkey support",
			Remediation: jsFix + " Stay on 16.4.x only if the app never parses, validates or displays contract addresses."}
	}
	return Assessment{LedgerXDR: No, MuxedContract: No, References: refs,
		Basis:       "this version lacks the CAP-88 ledger close-value XDR arms and the CAP-84 W strkey; only 16.4.x and 17.3.0+ decode Protocol 30 ledgers",
		Remediation: jsFix}
}

// jsBase: no @stellar/stellar-base release up to 15.0.0 has Protocol 30
// support; the SDK bundles its own copy since 16.0.0 (E13).
func jsBase(v Version) Assessment {
	refs := []string{RefCAP88, RefCAP84, RefJSSDK}
	fix := "Drop the direct @stellar/stellar-base dependency and use @stellar/stellar-sdk 17.3.0 or later, which bundles the base library."
	if jsBaseLastKnown.Less(v) {
		return Assessment{LedgerXDR: Unknown, MuxedContract: Unknown, References: refs,
			Basis: "newer than the last verified release (15.0.0); its Protocol 30 support was not verified", Remediation: fix}
	}
	return Assessment{LedgerXDR: No, MuxedContract: No, References: refs,
		Basis:       "no @stellar/stellar-base release up to 15.0.0 has Protocol 30 XDR or W strkey support",
		Remediation: fix}
}

// unscoped: the old unscoped npm names are deprecated; their final
// releases predate Protocol 30 (E14).
func unscoped(name, last string) func(Version) Assessment {
	lastV := must(last)
	return func(v Version) Assessment {
		refs := []string{RefCAP88, RefCAP84, RefJSSDK}
		fix := "Replace " + name + " with @stellar/stellar-sdk 17.3.0 or later."
		if lastV.Less(v) {
			return Assessment{LedgerXDR: Unknown, MuxedContract: Unknown, References: refs,
				Basis: "newer than the last known release (" + last + ") of this deprecated package", Remediation: fix}
		}
		return Assessment{LedgerXDR: No, MuxedContract: No, References: refs,
			Basis:       name + " is deprecated; its final release (" + last + ") predates Protocol 30",
			Remediation: fix}
	}
}
