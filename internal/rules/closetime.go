package rules

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/Protocol-30/stellar-p30-check/internal/deps"
	"github.com/Protocol-30/stellar-p30-check/internal/report"
)

// CloseTimesID is the rule ID of the close-time check.
const CloseTimesID = "P30-TS-001"

// CloseTimes (P30-TS-001) looks for code that assumes whole-second ledger
// close times differ between ledgers. Under CAP-88 the effective close time
// has millisecond resolution and consecutive ledgers may share the same
// whole-second closeTime, so a close-time difference can be 0.
//
// It reports two shapes: dividing by the difference of two close times
// (medium) and comparing two close times (info, a review hint).
type CloseTimes struct{}

// ID implements Rule.
func (CloseTimes) ID() string { return CloseTimesID }

// Whole-second close-time field names: XDR StellarValue.CloseTime, RPC
// LedgerCloseTime, Horizon ClosedAt (go-stellar-sdk v0.8.0; JS equivalents).
var (
	goCloseTimeFields = map[string]bool{"CloseTime": true, "LedgerCloseTime": true, "ClosedAt": true}
	jsCloseTimeFields = map[string]bool{"closeTime": true, "ledgerCloseTime": true, "closed_at": true, "closedAt": true}
	goNumericConv     = map[string]bool{
		"int": true, "int32": true, "int64": true, "uint": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true,
	}
	goTimeUnits = map[string]bool{"Unix": true, "UnixMilli": true, "UnixNano": true, "Seconds": true, "Milliseconds": true, "Minutes": true, "Hours": true}
)

// Check implements Rule.
func (r CloseTimes) Check(ctx context.Context, in Input) (Output, error) {
	return checkSource(ctx, in, r)
}

func (CloseTimes) prepare(context.Context, Input) (func(*sourceFile) Output, error) {
	return func(sf *sourceFile) Output {
		var out Output
		if !isCodeKind(sf.Kind) || isTestFile(sf.Path) {
			return out
		}
		if err := sf.codeProblem(); err != nil {
			out.Warnings = append(out.Warnings, sourceWarning(sf.Path, err))
			return out
		}
		var obs []closeTimeObs
		if sf.goAST != nil {
			obs = goCloseTimeObs(sf.fset, sf.goAST)
		} else {
			obs = jsCloseTimeObs(sf.js)
		}
		for _, o := range obs {
			out.Findings = append(out.Findings, o.finding(sf.Path, sf.lines))
		}
		return out
	}, nil
}

type closeTimeObs struct {
	line     int
	division bool // false: comparison
}

func (o closeTimeObs) finding(file string, lines []string) report.Finding {
	f := report.Finding{
		RuleID: CloseTimesID, Path: file, Line: o.line, Evidence: snippet(lines, o.line),
		References: []string{deps.RefCAP88},
	}
	if o.division {
		f.Severity, f.Confidence = report.SeverityMedium, report.ConfidenceMedium
		f.Message = "divides by the difference of two whole-second close times; after CAP-88 consecutive ledgers can close within the same second, so the divisor can be 0"
		f.Remediation = "Guard the divisor against 0, or use the millisecond close time (closeTimeMs in the STELLAR_VALUE_SIGNED_MS arm), which strictly increases."
		return f
	}
	f.Severity, f.Confidence = report.SeverityInfo, report.ConfidenceLow
	f.Message = "compares two whole-second close times; after CAP-88 consecutive ledgers can have the same closeTime, so check this does not assume it strictly increases"
	f.Remediation = "Allow equal close times between ledgers, or compare ledger sequence numbers or closeTimeMs instead."
	return f
}

func goCloseTimeObservations(filename string, src []byte) ([]closeTimeObs, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	return goCloseTimeObs(fset, file), nil
}

func goCloseTimeObs(fset *token.FileSet, file *ast.File) []closeTimeObs {
	var obs []closeTimeObs
	line := func(n ast.Node) int { return fset.Position(n.Pos()).Line }
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BinaryExpr:
			switch n.Op {
			case token.QUO, token.REM:
				if goCloseTimeDiff(n.Y) {
					obs = append(obs, closeTimeObs{line: line(n), division: true})
				}
			case token.LSS, token.GTR, token.LEQ, token.GEQ, token.EQL, token.NEQ:
				if goCloseTime(n.X) && goCloseTime(n.Y) {
					obs = append(obs, closeTimeObs{line: line(n)})
				}
			}
		case *ast.CallExpr:
			// a.ClosedAt.Before(b.ClosedAt), After, Equal
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok && len(n.Args) == 1 &&
				(sel.Sel.Name == "Before" || sel.Sel.Name == "After" || sel.Sel.Name == "Equal") &&
				goCloseTime(sel.X) && goCloseTime(n.Args[0]) {
				obs = append(obs, closeTimeObs{line: line(n)})
			}
		}
		return true
	})
	return obs
}

// goCloseTime matches x.CloseTime, int64(x.CloseTime), x.ClosedAt.Unix() and
// similar wrappers around a whole-second close-time field.
func goCloseTime(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.ParenExpr:
		return goCloseTime(e.X)
	case *ast.SelectorExpr:
		return goCloseTimeFields[e.Sel.Name]
	case *ast.CallExpr:
		if id, ok := e.Fun.(*ast.Ident); ok && goNumericConv[id.Name] && len(e.Args) == 1 {
			return goCloseTime(e.Args[0])
		}
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok && len(e.Args) == 0 && goTimeUnits[sel.Sel.Name] {
			return goCloseTime(sel.X)
		}
	}
	return false
}

// goCloseTimeDiff matches a - b and a.Sub(b) over close times, through
// conversions and unit methods such as .Seconds().
func goCloseTimeDiff(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.ParenExpr:
		return goCloseTimeDiff(e.X)
	case *ast.BinaryExpr:
		return e.Op == token.SUB && goCloseTime(e.X) && goCloseTime(e.Y)
	case *ast.CallExpr:
		if id, ok := e.Fun.(*ast.Ident); ok && goNumericConv[id.Name] && len(e.Args) == 1 {
			return goCloseTimeDiff(e.Args[0])
		}
		sel, ok := e.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		if sel.Sel.Name == "Sub" && len(e.Args) == 1 {
			return goCloseTime(sel.X) && goCloseTime(e.Args[0])
		}
		if goTimeUnits[sel.Sel.Name] && len(e.Args) == 0 {
			return goCloseTimeDiff(sel.X)
		}
	}
	return false
}

// jsCloseTimeObservations works on tokens. A close time is a member access
// (.closeTime, .ledgerCloseTime, .closed_at, .closedAt), optionally called.
func jsCloseTimeObservations(src []byte) []closeTimeObs {
	return jsCloseTimeObs(newJSFile(src))
}

func jsCloseTimeObs(f *jsFile) []closeTimeObs {
	toks := f.toks
	isField := func(i int) bool {
		return i > 0 && toks[i].Kind == jsIdent && jsCloseTimeFields[toks[i].Text] &&
			toks[i-1].Kind == jsPunct && (toks[i-1].Text == "." || toks[i-1].Text == "?.")
	}
	hasField := func(lo, hi int) bool { return f.count("close-time-field", isField, lo, hi) > 0 }
	var obs []closeTimeObs
	for i, t := range toks {
		if t.Kind != jsPunct {
			continue
		}
		switch t.Text {
		case "/", "%":
			// x / (a.closeTime() - b.closeTime())
			if !f.at(i+1, jsPunct, "(") {
				continue
			}
			end, minus := f.match[i+1], f.firstMinus[i+1]
			if end > 0 && minus > 0 && hasField(i+2, minus) && hasField(minus+1, end) {
				obs = append(obs, closeTimeObs{line: t.Line, division: true})
			}
		case "<", ">", "<=", ">=", "===", "!==", "==", "!=":
			lo := jsOperandStart(f, i)
			hi := jsOperandEnd(f, i)
			if hasField(lo, i) && hasField(i+1, hi) {
				obs = append(obs, closeTimeObs{line: t.Line})
			}
		}
	}
	return obs
}

var jsOperandStops = map[string]bool{
	",": true, ";": true, "&&": true, "||": true, "??": true, "?": true, ":": true, "{": true, "}": true,
	"=": true, "=>": true, "<": true, ">": true, "<=": true, ">=": true, "===": true, "!==": true, "==": true, "!=": true, "!": true,
}

// jsOperandStart walks left from the operator at i to the start of its
// left operand. Balanced groups are skipped in one step via the bracket
// index, so the walk is linear in the operand's top-level tokens.
func jsOperandStart(f *jsFile, i int) int {
	toks := f.toks
	for j := i - 1; j >= 0; j-- {
		t := toks[j]
		if t.Kind == jsIdent && (t.Text == "return" || t.Text == "if" || t.Text == "while") {
			return j + 1
		}
		if t.Kind != jsPunct {
			continue
		}
		switch {
		case t.Text == ")" || t.Text == "]":
			if f.match[j] < 0 {
				return j + 1
			}
			j = f.match[j] // continue left of the opening bracket
		case t.Text == "(" || t.Text == "[" || jsOperandStops[t.Text]:
			return j + 1
		}
	}
	return 0
}

// jsOperandEnd walks right from the operator at i to the end of its right
// operand (exclusive).
func jsOperandEnd(f *jsFile, i int) int {
	toks := f.toks
	for j := i + 1; j < len(toks); j++ {
		t := toks[j]
		if t.Kind != jsPunct {
			continue
		}
		switch {
		case t.Text == "(" || t.Text == "[":
			if f.match[j] < 0 {
				return j
			}
			j = f.match[j]
		case t.Text == ")" || t.Text == "]" || jsOperandStops[t.Text]:
			return j
		}
	}
	return len(toks)
}
