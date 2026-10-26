package cli

import (
	"bytes"
	"encoding/json"
	"github.com/Protocol-30/stellar-p30-check/internal/report"
	"path/filepath"
	"testing"
	"time"
)

func TestLedgerFixtureReports(t *testing.T) {
	for _, tc := range []struct {
		name        string
		xdr, timing int
	}{{"m4-js-legacy", 1, 0}, {"m4-js-modern", 0, 0}, {"m4-go-review", 1, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			var previous []byte
			for i := 0; i < 2; i++ {
				var out, errOut bytes.Buffer
				exit := Run([]string{"scan", filepath.Join("..", "..", "testdata", tc.name), "--format=json", "--fail-on=high"}, &out, &errOut, Options{Now: func() time.Time { return fixedNow }})
				if exit != 0 || errOut.Len() != 0 {
					t.Fatalf("exit=%d error=%s", exit, &errOut)
				}
				var r report.ScanReport
				if err := json.Unmarshal(out.Bytes(), &r); err != nil {
					t.Fatal(err)
				}
				counts := map[string]int{}
				for _, f := range r.Findings {
					counts[f.RuleID]++
					if f.RuleID == "P30-TS-001" && (f.Severity != report.SeverityInfo || f.Confidence != report.ConfidenceLow) {
						t.Fatalf("timing overstated: %+v", f)
					}
				}
				if !r.Coverage.Complete || r.Coverage.RulesImplemented != 4 || counts["P30-XDR-001"] != tc.xdr || counts["P30-TS-001"] != tc.timing {
					t.Fatalf("bad report: %+v", r)
				}
				if i == 0 {
					previous = bytes.Clone(out.Bytes())
				} else if !bytes.Equal(previous, out.Bytes()) {
					t.Fatal("nondeterministic ledger findings")
				}
			}
		})
	}
}
