package rules

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/Protocol-30/stellar-p30-check/internal/deps"
	"github.com/Protocol-30/stellar-p30-check/internal/report"
)

// AddressesID is the rule ID of the W address check.
const AddressesID = "P30-ADDR-001"

// Addresses (P30-ADDR-001) looks for address handling that accepts contract
// (C) addresses together with account (G) or muxed-account (M) addresses
// but not CAP-84 muxed-contract (W) addresses, and for SCAddress or strkey
// version-byte switches with a contract case but no muxed-contract case.
//
// Go files are parsed with go/parser. JavaScript and TypeScript are read
// with a small tokenizer, not a full parser, so only the literal shapes
// listed in docs/rules.md are recognised. Nothing is executed: regular
// expressions found in source are analysed as text.
type Addresses struct{}

// ID implements Rule.
func (Addresses) ID() string { return AddressesID }

// Check implements Rule.
func (a Addresses) Check(ctx context.Context, in Input) (Output, error) {
	return checkSource(ctx, in, a)
}

func (Addresses) prepare(context.Context, Input) (func(*sourceFile) Output, error) {
	return func(sf *sourceFile) Output {
		var out Output
		if !isCodeKind(sf.Kind) || isTestFile(sf.Path) {
			return out
		}
		if err := sf.codeProblem(); err != nil {
			out.Warnings = append(out.Warnings, sourceWarning(sf.Path, err))
			return out
		}
		var obs []observation
		if sf.goAST != nil {
			obs = goAddressObs(sf.fset, sf.goAST)
		} else {
			obs = jsAddressObs(sf.js)
		}
		for _, o := range obs {
			if fd, ok := o.finding(sf.Path, sf.lines); ok {
				out.Findings = append(out.Findings, fd)
			}
		}
		return out
	}, nil
}

// sourceWarning reports a source file the source rules could not read or
// parse. The message does not name the rule, so the scanner can merge the
// identical warnings every source rule produces for the same file.
func sourceWarning(p string, err error) report.Warning {
	return report.Warning{
		Code: "source-unparsed", Path: p,
		Message: fmt.Sprintf("%s was not checked by the source rules: %v", path.Base(p), err),
	}
}

// isCodeKind reports whether an inventory kind is Go or JS/TS source.
func isCodeKind(kind string) bool {
	switch kind {
	case "source-go", "source-js", "source-jsx", "source-ts", "source-tsx":
		return true
	}
	return false
}

// isTestFile reports whether a source path is test code, which is skipped:
// tests routinely contain deliberately narrow validators and fixtures.
func isTestFile(p string) bool {
	base := path.Base(p)
	if strings.HasSuffix(base, "_test.go") {
		return true
	}
	for _, marker := range []string{".test.", ".spec."} {
		if strings.Contains(base, marker) {
			return true
		}
	}
	return strings.Contains("/"+p, "/__tests__/")
}

// strkeyLetters are the SEP-23 strkey prefixes. W is the CAP-84 muxed
// contract address.
const strkeyLetters = "GMCWSTXPLB"

// kindSet is a set of strkey prefix letters.
type kindSet uint16

func (k kindSet) has(letter byte) bool {
	i := strings.IndexByte(strkeyLetters, letter)
	return i >= 0 && k&(1<<i) != 0
}

func (k *kindSet) add(letter byte) bool {
	i := strings.IndexByte(strkeyLetters, letter)
	if i < 0 {
		return false
	}
	*k |= 1 << i
	return true
}

func (k kindSet) size() int {
	n := 0
	for i := range strkeyLetters {
		if k&(1<<i) != 0 {
			n++
		}
	}
	return n
}

// String lists the letters in a fixed, readable order.
func (k kindSet) String() string {
	var parts []string
	for i := range strkeyLetters {
		if k&(1<<i) != 0 {
			parts = append(parts, strkeyLetters[i:i+1])
		}
	}
	return strings.Join(parts, ", ")
}

// lettersOf builds a set from a string of letters. ok is false when any
// character is not a strkey prefix.
func lettersOf(s string) (kindSet, bool) {
	var k kindSet
	for i := 0; i < len(s); i++ {
		if !k.add(s[i]) {
			return 0, false
		}
	}
	return k, s != ""
}

// construct is the kind of source shape an observation came from. It
// decides how strong the evidence is.
type construct int

const (
	cRegex      construct = iota // anchored address regex: prefix class + strkey body
	cPredicates                  // SDK StrKey / strkey validity predicates combined in one expression
	cPrefixTest                  // startsWith / HasPrefix / [0] comparisons on one value
	cLetterList                  // a literal list or string of prefix letters
	cTypeSwitch                  // switch over SCAddress type or strkey version byte
)

// constructSubjects start finding messages ("<subject> G, M, C but not W").
var constructSubjects = map[construct]string{
	cRegex:      "address regex accepts",
	cPredicates: "SDK address checks accept",
	cPrefixTest: "prefix checks accept",
	cLetterList: "prefix list accepts",
}

// observation is one recognised shape in a source file.
type observation struct {
	line      int
	construct construct
	kinds     kindSet // accepted prefixes (allowlists)
	// wNearby: the enclosing function (or file, at top level) also handles
	// W addresses, so an allowlist without W is likely one branch of a
	// complete check. Allowlist findings are suppressed then.
	wNearby bool
	pos     int // language-specific position used to compute wNearby
	// For type switches: whether a contract and a muxed-contract case are
	// present, and whether there is a default branch.
	contractCase, muxedContractCase, hasDefault bool
}

const (
	refSEP23 = "https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0023.md"
)

// finding applies the P30-ADDR-001 policy (docs/rules.md) to an
// observation.
func (o observation) finding(file string, lines []string) (report.Finding, bool) {
	f := report.Finding{
		RuleID: AddressesID, Path: file, Line: o.line,
		Evidence:   snippet(lines, o.line),
		References: []string{deps.RefCAP84, refSEP23, deps.RefUpgrade},
	}
	if o.construct == cTypeSwitch {
		if !o.contractCase || o.muxedContractCase {
			return report.Finding{}, false
		}
		f.Confidence = report.ConfidenceMedium
		if o.hasDefault {
			f.Severity = report.SeverityMedium
			f.Message = "address type switch handles contract addresses but has no muxed-contract (W, CAP-84) case; W addresses reach the default branch"
		} else {
			f.Severity = report.SeverityHigh
			f.Message = "address type switch handles contract addresses but has no muxed-contract (W, CAP-84) case and no default, so W addresses are silently skipped"
		}
		f.Remediation = "Add a case for the muxed-contract type (Go: xdr.ScAddressTypeScAddressTypeMuxedContract or strkey.VersionByteMuxedContract; JS: scAddressTypeMuxedContract), or return an explicit error for unsupported types."
		return f, true
	}

	k := o.kinds
	if o.wNearby || !k.has('C') || k.has('W') || !(k.has('G') || k.has('M')) || k.size() < 2 {
		return report.Finding{}, false
	}
	strong := o.construct == cRegex || o.construct == cPredicates
	subject := constructSubjects[o.construct]
	if k.has('M') {
		f.Severity = report.SeverityHigh
		f.Confidence = report.ConfidenceMedium
		if strong {
			f.Confidence = report.ConfidenceHigh
		}
		f.Message = fmt.Sprintf("%s %s but not W: muxed-contract addresses (CAP-84) are rejected where muxed and contract addresses are both allowed", subject, k)
		f.Remediation = "If this value can be a payment or transfer destination, accept W too: StrKey.isValidMuxedContract (@stellar/stellar-sdk 17.3.0+) or strkey.IsValidMuxedContractAddress (go-stellar-sdk v0.8.0+). W addresses are 69 characters long, like M addresses."
		return f, true
	}
	f.Severity = report.SeverityInfo
	f.Confidence = report.ConfidenceLow
	f.Message = fmt.Sprintf("%s %s but not W; review whether this value can be a muxed-contract (CAP-84) transfer destination", subject, k)
	f.Remediation = "Accept W only if this value can be a transfer destination. Contract IDs that are invoked and contract storage keys must not be W (CAP-84), so rejecting W is correct there."
	return f, true
}

// snippet returns the trimmed source line, shortened for reports.
func snippet(lines []string, line int) string {
	if line < 1 || line > len(lines) {
		return ""
	}
	s := strings.TrimSpace(lines[line-1])
	if len(s) > 160 {
		s = s[:157] + "..."
	}
	return s
}

// regexAllowlist analyses an address-validation regular expression as
// text. It recognises an anchored pattern whose first element is a set of
// strkey prefix letters ([GMC], (G|M|C), (?:G|C)) and whose next element is
// a strkey body class such as [A-Z2-7] or [A-Z0-9]. ok is false for any
// other pattern.
func regexAllowlist(pattern string) (kindSet, bool) {
	p, found := strings.CutPrefix(pattern, "^")
	if !found {
		return 0, false
	}
	var set kindSet
	var rest string
	switch {
	case strings.HasPrefix(p, "["):
		end := strings.IndexByte(p, ']')
		if end < 0 {
			return 0, false
		}
		class := p[1:end]
		if strings.HasPrefix(class, "^") {
			return 0, false
		}
		s, ok := expandClass(class)
		if !ok {
			return 0, false
		}
		set, rest = s, p[end+1:]
	case strings.HasPrefix(p, "("):
		end := strings.IndexByte(p, ')')
		if end < 0 {
			return 0, false
		}
		group := strings.TrimPrefix(p[1:end], "?:")
		for _, alt := range strings.Split(group, "|") {
			if len(alt) != 1 || !set.add(alt[0]) {
				return 0, false
			}
		}
		rest = p[end+1:]
	default:
		return 0, false
	}
	if !strkeyBody(rest) {
		return 0, false
	}
	return set, true
}

// expandClass turns a character class body into a strkey letter set. A
// class containing anything other than uppercase strkey letters, or a range
// that covers a non-strkey letter, is not an address prefix class.
func expandClass(class string) (kindSet, bool) {
	var set kindSet
	for i := 0; i < len(class); i++ {
		c := class[i]
		if i+2 < len(class) && class[i+1] == '-' {
			hi := class[i+2]
			if c < 'A' || hi > 'Z' || hi < c {
				return 0, false
			}
			for l := c; l <= hi; l++ {
				if !set.add(l) {
					return 0, false
				}
			}
			i += 2
			continue
		}
		if c == '|' || c == ',' || c == ' ' {
			continue // common mistakes like [G|M|C] still list letters
		}
		if !set.add(c) {
			return 0, false
		}
	}
	return set, set != 0
}

// strkeyBody reports whether the pattern after the prefix starts with a
// base32-style body: a class covering A-Z plus digits.
func strkeyBody(rest string) bool {
	if !strings.HasPrefix(rest, "[") {
		return false
	}
	end := strings.IndexByte(rest, ']')
	if end < 0 {
		return false
	}
	class := rest[1:end]
	hasLetters := strings.Contains(class, "A-Z")
	hasDigits := strings.ContainsAny(class, "0123456789") || strings.Contains(class, `\d`)
	return hasLetters && hasDigits
}
