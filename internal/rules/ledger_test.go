package rules

import (
	"os"
	"strings"
	"testing"

	"github.com/Protocol-30/stellar-p30-check/internal/deps"
	"github.com/Protocol-30/stellar-p30-check/internal/report"
	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

func TestLedgerJSScopedDecoderUse(t *testing.T) {
	source := `import { xdr } from "@stellar/stellar-sdk";
 export function decode(data: string) { return xdr.LedgerHeader.fromXDR(data, "base64"); }`
	d := deps.Dependency{Ecosystem: "npm", Name: jsSDK, Scope: ".", Path: "package.json", Version: "16.3.1", Effective: true, Resolution: "lockfile", Source: "package-lock.json"}
	for _, tc := range []struct {
		name, source, version string
		child                 bool
		count                 int
		confidence            report.Confidence
	}{
		{"legacy", source, "16.3.1", false, 1, report.ConfidenceHigh},
		{"modern", source, "17.3.0", false, 0, ""},
		{"maintenance", source, "16.4.0", false, 0, ""},
		{"unknown", source, "99.0.0", false, 1, report.ConfidenceLow},
		{"nested", source, "17.3.0", true, 1, report.ConfidenceLow},
		{"meta", strings.ReplaceAll(source, "LedgerHeader", "LedgerCloseMeta"), "16.3.1", false, 1, report.ConfidenceHigh},
		{"unrelated-type", strings.ReplaceAll(source, "LedgerHeader", "TransactionEnvelope"), "16.3.1", false, 0, ""},
		{"shadow", strings.Replace(source, "data: string", "data: string, xdr: any", 1), "16.3.1", false, 0, ""},
		{"type-only", strings.Replace(source, "import {", "import type {", 1), "16.3.1", false, 0, ""},
		{"mutation", source + `;xdr.LedgerHeader.fromXDR = fake;`, "16.3.1", false, 0, ""},
		{"comment", `// ` + strings.ReplaceAll(source, "\n", "\n// "), "16.3.1", false, 0, ""},
		{"string", `const code = 'xdr.LedgerHeader.fromXDR(data)';`, "16.3.1", false, 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := walker.File{Path: "src/decode.ts", Kind: "ts_source", Data: []byte(tc.source)}
			files := []walker.File{{Path: "package.json"}, f}
			if tc.child {
				files = append(files, walker.File{Path: "src/package.json"})
			}
			selected := d
			selected.Version = tc.version
			got, _, err := AnalyzeLedgers(files, []deps.Dependency{selected})
			if err != nil || len(got) != tc.count {
				t.Fatalf("findings=%+v err=%v", got, err)
			}
			for _, finding := range got {
				if finding.RuleID != "P30-XDR-001" || finding.Severity != report.SeverityMedium || finding.Confidence != tc.confidence || finding.Line < 1 || finding.Evidence == "" || len(finding.References) == 0 {
					t.Fatalf("invalid evidence: %+v", finding)
				}
			}
		})
	}
}

func TestLedgerGoTypedUseAndTiming(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		count        int
		id           string
	}{
		{"decode", `package p; import "github.com/stellar/go-stellar-sdk/xdr"; func decode(data string, header *xdr.LedgerHeader) error {return xdr.SafeUnmarshalBase64(data,header)}`, 1, "P30-XDR-001"},
		{"timing", `package p; import "github.com/stellar/go-stellar-sdk/xdr"; func ordered(a,b xdr.LedgerHeader)bool{return a.ScpValue.CloseTime>b.ScpValue.CloseTime}`, 1, "P30-TS-001"},
		{"non-strict", `package p; import "github.com/stellar/go-stellar-sdk/xdr"; func ordered(a,b xdr.LedgerHeader)bool{return a.ScpValue.CloseTime>=b.ScpValue.CloseTime}`, 0, ""},
		{"untyped", `package p; func ordered(a,b Header)bool{return a.ScpValue.CloseTime>b.ScpValue.CloseTime}`, 0, ""},
		{"millis", `package p; import "github.com/stellar/go-stellar-sdk/xdr"; func ordered(a,b xdr.LedgerHeader)bool{return a.ScpValue.CloseTimeMs>b.ScpValue.CloseTimeMs}`, 0, ""},
		{"shadow", `package p; import "github.com/stellar/go-stellar-sdk/xdr"; func decode(data string, xdr any, header *xdr.LedgerHeader)error{return xdr.SafeUnmarshalBase64(data,header)}`, 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, _, err := AnalyzeLedgers([]walker.File{{Path: "decode.go", Kind: "go_source", Data: []byte(tc.source)}}, nil)
			if err != nil || len(got) != tc.count {
				t.Fatalf("%+v %v", got, err)
			}
			for _, f := range got {
				if f.RuleID != tc.id || f.Confidence != report.ConfidenceLow {
					t.Fatalf("%+v", f)
				}
			}
		})
	}
}

func TestCAP88SpecificationArmsAndRounding(t *testing.T) {
	data, err := os.ReadFile("../../testdata/cap88/Stellar-ledger.x")
	if err != nil {
		t.Fatal(err)
	}
	for _, arm := range []string{"STELLAR_VALUE_SIGNED_MS", "STELLAR_VALUE_EMPTY_TX_SET_MS"} {
		start := strings.Index(string(data), "case "+arm+":")
		if start < 0 {
			t.Fatal("missing official union arm", arm)
		}
		block := string(data)[start:]
		end := strings.Index(block, ";") + 1
		if end <= 0 || !strings.Contains(block[:end], "TimePointMs closeTimeMs") {
			t.Fatal("missing millisecond field", arm)
		}
	}
	// Synthetic arithmetic examples, not serialized or captured ledgers.
	previous, current := int64(1000001), int64(1000002)
	if current <= previous || current/1000 != previous/1000 {
		t.Fatal("invalid same-second example")
	}
}

func TestCAP88ExtractsMatchPinnedSchema(t *testing.T) {
	full, err := os.ReadFile("../../testdata/cap88/Stellar-ledger.x")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"signed-ms.x", "empty-tx-set-ms.x"} {
		excerpt, err := os.ReadFile("../../testdata/cap88/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(full), strings.TrimSuffix(string(excerpt), "\n")) {
			t.Fatal("excerpt differs from pinned normative schema", name)
		}
	}
}
