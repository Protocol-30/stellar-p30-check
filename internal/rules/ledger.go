package rules

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strconv"
	"strings"

	"github.com/Protocol-30/stellar-p30-check/internal/deps"
	"github.com/Protocol-30/stellar-p30-check/internal/report"
)

// LedgerXDRID is the rule ID of the ledger XDR check.
const LedgerXDRID = "P30-XDR-001"

// LedgerXDR (P30-XDR-001) looks for code that will mis-handle ledger
// headers closed after the Protocol 30 upgrade, which all use the CAP-88
// STELLAR_VALUE_SIGNED_MS or STELLAR_VALUE_EMPTY_TX_SET_MS arms:
//
//   - code that decodes ledger data while its project resolves a Stellar SDK
//     without Protocol 30 XDR (shared resolution with P30-DEP-001);
//   - StellarValueType switches and comparisons without the _MS arms;
//   - the single-arm GetLcValueSignature / MustLcValueSignature accessors;
//   - vendored XDR definition files (.x) that predate CAP-88 or CAP-84.
type LedgerXDR struct{}

// ID implements Rule.
func (LedgerXDR) ID() string { return LedgerXDRID }

const refStellarXDR = "https://github.com/stellar/stellar-xdr/blob/curr/Stellar-ledger.x"

// Check implements Rule.
func (r LedgerXDR) Check(ctx context.Context, in Input) (Output, error) {
	return checkSource(ctx, in, r)
}

func (LedgerXDR) prepare(ctx context.Context, in Input) (func(*sourceFile) Output, error) {
	uses, _, err := collectSDKUses(ctx, in, LedgerXDRID) // manifest warnings are P30-DEP-001's to report
	if err != nil {
		return nil, err
	}
	return func(sf *sourceFile) Output {
		var out Output
		if sf.Kind == "xdr-definition" {
			if sf.readErr != nil {
				out.Warnings = append(out.Warnings, sourceWarning(sf.Path, sf.readErr))
				return out
			}
			out.Findings = xdrDefinitionFindings(sf.Path, sf.data)
			return out
		}
		if !isCodeKind(sf.Kind) || isTestFile(sf.Path) {
			return out
		}
		if err := sf.codeProblem(); err != nil {
			out.Warnings = append(out.Warnings, sourceWarning(sf.Path, err))
			return out
		}
		var lo ledgerObservations
		eco := deps.NPM
		if sf.goAST != nil {
			eco = deps.Go
			lo = goLedgerObs(sf.fset, sf.goAST)
		} else {
			lo = jsLedgerObs(sf.js)
		}
		if lo.decodeLine > 0 {
			if fd, ok := decodeFinding(sf.Path, sf.lines, lo, nearestUses(uses, eco, path.Dir(sf.Path))); ok {
				out.Findings = append(out.Findings, fd)
			}
		}
		for _, o := range lo.other {
			o.RuleID, o.Path = LedgerXDRID, sf.Path
			o.Evidence = snippet(sf.lines, o.Line)
			o.References = []string{deps.RefCAP88, deps.RefGoChangelog, refStellarXDR}
			out.Findings = append(out.Findings, o)
		}
		return out
	}, nil
}

// ledgerObservations are the P30-XDR-001 shapes found in one source file.
type ledgerObservations struct {
	decodeLine int    // first line that decodes ledger data, 0 if none
	decodeWhat string // what is decoded, for the message
	other      []report.Finding
}

// decodeFinding ties a ledger-decoding file to the SDK its project uses.
func decodeFinding(file string, lines []string, lo ledgerObservations, uses []sdkUse) (report.Finding, bool) {
	var pick *sdkUse
	for i := range uses {
		u := &uses[i]
		if u.Kind == useUnresolved {
			continue
		}
		a, _ := deps.Assess(u.Eco, u.Name, u.Parsed)
		if a.LedgerXDR != deps.No {
			continue
		}
		if pick == nil || (pick.Kind == useRange && u.Kind == useExact) {
			pick = u
		}
	}
	if pick == nil {
		return report.Finding{}, false
	}
	conf := report.ConfidenceHigh
	if pick.Kind == useRange {
		conf = report.ConfidenceLow
	}
	fix := "Upgrade to @stellar/stellar-sdk 17.3.0 or later (16.4.x also decodes Protocol 30 ledgers)."
	if pick.Eco == deps.Go {
		fix = "Upgrade to github.com/stellar/go-stellar-sdk v0.8.0 or later (it requires Go 1.25.0 or newer)."
	}
	return report.Finding{
		RuleID: LedgerXDRID, Severity: report.SeverityHigh, Confidence: conf,
		Path: file, Line: lo.decodeLine,
		Message: fmt.Sprintf("decodes %s with %s, whose XDR lacks the CAP-88 StellarValue arms; every ledger closed after the Protocol 30 upgrade will fail to decode",
			lo.decodeWhat, pick.Subject),
		Evidence:    fmt.Sprintf("%s; SDK from %s:%d: %s", snippet(lines, lo.decodeLine), pick.File, pick.Line, pick.Evidence),
		Remediation: fix,
		References:  []string{deps.RefCAP88, deps.RefUpgrade, deps.RefGoChangelog},
	}, true
}

// Go names (go-stellar-sdk v0.8.0, docs/evidence.md E6 and E16).
var (
	goLedgerTypes = map[string]bool{
		"LedgerHeader": true, "LedgerHeaderHistoryEntry": true, "LedgerCloseMeta": true,
		"LedgerCloseMetaBatch": true, "StellarValue": true,
	}
	goLedgerPackages = []string{"ingest", "ingest/ledgerbackend", "historyarchive"}
	goXDRPaths       = []string{"github.com/stellar/go-stellar-sdk/xdr", "github.com/stellar/go/xdr"}
	goOldValueTypes  = map[string]bool{"StellarValueTypeStellarValueSigned": true, "StellarValueTypeStellarValueEmptyTxSet": true}
	goMsValueTypes   = map[string]bool{"StellarValueTypeStellarValueSignedMs": true, "StellarValueTypeStellarValueEmptyTxSetMs": true}
	// goMsMarkers show that a function already handles the CAP-88 arms.
	goMsMarkers = map[string]bool{
		"StellarValueTypeStellarValueSignedMs": true, "StellarValueTypeStellarValueEmptyTxSetMs": true,
		"GetSignedMsValue": true, "MustSignedMsValue": true, "SignedMsValue": true,
		"GetProposedMsValue": true, "MustProposedMsValue": true, "ProposedMsValue": true,
		"LedgerCloseValueSignature": true,
	}
	goArmAccessors = map[string]bool{
		"MustSignedMsValue": true, "GetSignedMsValue": true, "MustProposedValue": true,
		"GetProposedValue": true, "MustProposedMsValue": true, "GetProposedMsValue": true,
	}
)

func goLedgerObservations(filename string, src []byte) (ledgerObservations, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.SkipObjectResolution)
	if err != nil {
		return ledgerObservations{}, err
	}
	return goLedgerObs(fset, file), nil
}

func goLedgerObs(fset *token.FileSet, file *ast.File) ledgerObservations {
	var lo ledgerObservations
	line := func(n ast.Node) int { return fset.Position(n.Pos()).Line }
	noteDecode := func(n ast.Node, what string) {
		if lo.decodeLine == 0 || line(n) < lo.decodeLine {
			lo.decodeLine, lo.decodeWhat = line(n), what
		}
	}

	handlesMs := newGoMentions(file, identIn(goMsMarkers))
	xdrNames := map[string]bool{}
	for _, imp := range file.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		name := path0(p)
		if imp.Name != nil {
			name = imp.Name.Name
		}
		for _, x := range goXDRPaths {
			if p == x {
				xdrNames[name] = true
			}
		}
		for _, root := range []string{"github.com/stellar/go-stellar-sdk/", "github.com/stellar/go/"} {
			for _, pkg := range goLedgerPackages {
				if p == root+pkg {
					noteDecode(imp, "ledger data (via "+p+")")
				}
			}
		}
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			if id, ok := n.X.(*ast.Ident); ok && xdrNames[id.Name] && goLedgerTypes[n.Sel.Name] {
				noteDecode(n, "xdr."+n.Sel.Name)
			}
		case *ast.SwitchStmt:
			hasOld, hasMs, hasDefault := false, false, false
			for _, stmt := range n.Body.List {
				cc := stmt.(*ast.CaseClause)
				if cc.List == nil {
					hasDefault = true
				}
				for _, e := range cc.List {
					hasOld = hasOld || goOldValueTypes[selectorName(e)]
					hasMs = hasMs || goMsValueTypes[selectorName(e)]
				}
			}
			if hasOld && !hasMs {
				lo.other = append(lo.other, valueTypeSwitchFinding(line(n), hasDefault))
			}
		case *ast.BinaryExpr:
			if n.Op != token.EQL && n.Op != token.NEQ {
				return true
			}
			name := selectorName(n.X)
			if !goOldValueTypes[name] {
				name = selectorName(n.Y)
			}
			if goOldValueTypes[name] && !handlesMs.at(n.Pos()) {
				lo.other = append(lo.other, report.Finding{
					Severity: report.SeverityMedium, Confidence: report.ConfidenceMedium, Line: line(n),
					Message:     fmt.Sprintf("compares a StellarValue type with %s only; after the Protocol 30 upgrade ledgers use the _MS arms (CAP-88), so this branch changes behaviour", name),
					Remediation: "Also handle StellarValueTypeStellarValueSignedMs and StellarValueTypeStellarValueEmptyTxSetMs.",
				})
			}
		case *ast.CallExpr:
			sel, ok := n.Fun.(*ast.SelectorExpr)
			if !ok || len(n.Args) != 0 || (sel.Sel.Name != "GetLcValueSignature" && sel.Sel.Name != "MustLcValueSignature") {
				return true
			}
			if recv, ok := sel.X.(*ast.CallExpr); ok && goArmAccessors[selectorName(recv.Fun)] {
				return true // reading the signature of a specific arm, which is fine
			}
			if handlesMs.at(n.Pos()) {
				return true
			}
			f := report.Finding{
				Line:        line(n),
				Remediation: "Use StellarValueExt.LedgerCloseValueSignature() (go-stellar-sdk v0.8.0+), which reads the signature from every arm.",
			}
			if sel.Sel.Name == "MustLcValueSignature" {
				f.Severity, f.Confidence = report.SeverityHigh, report.ConfidenceHigh
				f.Message = "MustLcValueSignature reads only the STELLAR_VALUE_SIGNED arm and panics on ledgers closed after the Protocol 30 upgrade, which use STELLAR_VALUE_SIGNED_MS (CAP-88)"
			} else {
				f.Severity, f.Confidence = report.SeverityMedium, report.ConfidenceHigh
				f.Message = "GetLcValueSignature reads only the STELLAR_VALUE_SIGNED arm, so it finds no signature on ledgers closed after the Protocol 30 upgrade (CAP-88)"
			}
			lo.other = append(lo.other, f)
		}
		return true
	})
	return lo
}

func valueTypeSwitchFinding(line int, hasDefault bool) report.Finding {
	f := report.Finding{
		Line: line, Confidence: report.ConfidenceMedium,
		Remediation: "Add cases for STELLAR_VALUE_SIGNED_MS and STELLAR_VALUE_EMPTY_TX_SET_MS (CAP-88); they carry the same fields plus closeTimeMs.",
	}
	if hasDefault {
		f.Severity = report.SeverityMedium
		f.Message = "StellarValue type switch has no case for the CAP-88 _MS arms; ledgers closed after the Protocol 30 upgrade reach the default branch"
	} else {
		f.Severity = report.SeverityHigh
		f.Message = "StellarValue type switch has no case for the CAP-88 _MS arms and no default, so ledgers closed after the Protocol 30 upgrade are silently skipped"
	}
	return f
}

// JS names: XDR types decoded with fromXDR, and the RPC call that decodes
// ledger headers inside the SDK (docs/evidence.md E18).
var (
	jsLedgerTypes   = map[string]bool{"LedgerHeader": true, "LedgerHeaderHistoryEntry": true, "LedgerCloseMeta": true, "StellarValue": true}
	jsOldValueTypes = map[string]bool{"stellarValueSigned": true, "stellarValueEmptyTxSet": true}
	jsMsValueTypes  = map[string]bool{"stellarValueSignedMs": true, "stellarValueEmptyTxSetMs": true}
)

func jsLedgerObservations(src []byte) ledgerObservations {
	return jsLedgerObs(newJSFile(src))
}

func jsLedgerObs(f *jsFile) ledgerObservations {
	toks, at := f.toks, f.at
	isOld := func(j int) bool {
		return (toks[j].Kind == jsIdent || toks[j].Kind == jsString) && jsOldValueTypes[toks[j].Text]
	}
	isMs := func(j int) bool {
		return (toks[j].Kind == jsIdent || toks[j].Kind == jsString) && jsMsValueTypes[toks[j].Text]
	}
	var lo ledgerObservations
	for i, t := range toks {
		switch {
		case lo.decodeLine == 0 && t.Kind == jsIdent && jsLedgerTypes[t.Text] && at(i+1, jsPunct, ".") && at(i+2, jsIdent, "fromXDR"):
			lo.decodeLine, lo.decodeWhat = t.Line, t.Text+".fromXDR"
		case lo.decodeLine == 0 && at(i, jsPunct, ".") && at(i+1, jsIdent, "getLedgers") && at(i+2, jsPunct, "("):
			lo.decodeLine, lo.decodeWhat = t.Line, "ledgers from rpc getLedgers()"
		default:
			sw, ok := f.switchAt(i)
			if !ok {
				continue
			}
			body0, body1 := sw.bodyOpen+1, sw.bodyClose
			if f.count("old-value-type", isOld, body0, body1) > 0 && f.count("ms-value-type", isMs, body0, body1) == 0 {
				lo.other = append(lo.other, valueTypeSwitchFinding(t.Line, sw.hasDefault))
			}
		}
	}
	return lo
}

// xdrDefinitionFindings checks a vendored .x file for the Protocol 30 enum
// members. Their text is taken from stellar-xdr (docs/evidence.md E19).
func xdrDefinitionFindings(file string, src []byte) []report.Finding {
	checks := []struct {
		enum, member, cap, ref string
	}{
		{"StellarValueType", "STELLAR_VALUE_SIGNED_MS", "CAP-88", deps.RefCAP88},
		{"SCAddressType", "SC_ADDRESS_TYPE_MUXED_CONTRACT", "CAP-84", deps.RefCAP84},
	}
	lines := strings.Split(string(src), "\n")
	var out []report.Finding
	for _, c := range checks {
		start, end := xdrEnumBlock(lines, c.enum)
		if start < 0 {
			continue
		}
		f := report.Finding{
			RuleID: LedgerXDRID, Path: file, Line: start + 1,
			Evidence:   strings.TrimSpace(lines[start]),
			References: []string{c.ref, refStellarXDR},
		}
		member, gated := -1, false
		depth := 0
		for i := start; i <= end; i++ {
			l := strings.TrimSpace(lines[i])
			switch {
			case strings.HasPrefix(l, "#if"):
				depth++
			case strings.HasPrefix(l, "#endif"):
				depth--
			case strings.Contains(l, c.member):
				member, gated = i, depth > 0
			}
		}
		switch {
		case member < 0:
			f.Severity, f.Confidence = report.SeverityHigh, report.ConfidenceHigh
			f.Message = fmt.Sprintf("XDR definition of %s has no %s (%s); code generated from it cannot decode Protocol 30 data", c.enum, c.member, c.cap)
			f.Remediation = "Update the vendored .x files from stellar-xdr (the curr branch includes the Protocol 30 changes) and regenerate."
		case gated:
			f.Severity, f.Confidence = report.SeverityMedium, report.ConfidenceLow
			f.Line = member + 1
			f.Evidence = strings.TrimSpace(lines[member])
			f.Message = fmt.Sprintf("%s is defined only inside an #ifdef block; code generated without that feature flag cannot decode Protocol 30 data (%s)", c.member, c.cap)
			f.Remediation = "Enable the feature flag in your XDR generator, or update to the stellar-xdr curr branch, where the Protocol 30 changes are no longer behind #ifdef."
		default:
			continue
		}
		out = append(out, f)
	}
	return out
}

// xdrEnumBlock returns the line range of "enum Name { ... };", or -1.
func xdrEnumBlock(lines []string, name string) (start, end int) {
	for i, l := range lines {
		fields := strings.Fields(l)
		if len(fields) >= 2 && fields[0] == "enum" && strings.TrimSuffix(fields[1], "{") == name {
			for j := i; j < len(lines); j++ {
				if strings.HasPrefix(strings.TrimSpace(lines[j]), "}") {
					return i, j
				}
			}
			return i, len(lines) - 1
		}
	}
	return -1, -1
}
