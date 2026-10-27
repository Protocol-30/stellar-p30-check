package rules

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Protocol-30/stellar-p30-check/internal/deps"
	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

// runRule walks a fixture directory and runs one rule over it.
func runRule(t *testing.T, r Rule, dir string) Output {
	t.Helper()
	res, err := walker.Walk(dir, walker.Options{})
	if err != nil {
		t.Fatal(err)
	}
	out, err := r.Check(context.Background(), Input{
		Files:    res.Files,
		ReadFile: func(rel string) ([]byte, error) { return os.ReadFile(filepath.Join(res.Root, filepath.FromSlash(rel))) },
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range out.Findings {
		if err := f.Validate(); err != nil {
			t.Errorf("invalid finding: %v", err)
		}
		if f.RuleID != r.ID() {
			t.Errorf("finding has rule %s, want %s", f.RuleID, r.ID())
		}
		if f.Evidence == "" || f.Remediation == "" || len(f.References) == 0 {
			t.Errorf("finding lacks evidence, remediation or references: %+v", f)
		}
	}
	return out
}

type ruleCase struct {
	dir      string
	findings []string
	warnings []string
	message  string // substring of the first finding's message
}

func runCases(t *testing.T, r Rule, base string, cases []ruleCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.dir, func(t *testing.T) {
			out := runRule(t, r, filepath.Join(base, c.dir))
			if got, want := summarize(out.Findings), append([]string{}, c.findings...); !reflect.DeepEqual(got, want) {
				t.Errorf("findings = %v, want %v\n%+v", got, want, out.Findings)
			}
			if got, want := warningCodes(out.Warnings), append([]string{}, c.warnings...); !reflect.DeepEqual(got, want) {
				t.Errorf("warnings = %v, want %v", got, want)
			}
			if c.message != "" && len(out.Findings) > 0 && !strings.Contains(out.Findings[0].Message, c.message) {
				t.Errorf("message = %q, want it to contain %q", out.Findings[0].Message, c.message)
			}
		})
	}
}

func TestLedgerXDRFixtures(t *testing.T) {
	runCases(t, LedgerXDR{}, "../../testdata/rules/xdr-001", []ruleCase{
		{"go-legacy-decoder", []string{"ingest/meta.go:6 high/high"}, nil, "decodes xdr.LedgerCloseMeta with github.com/stellar/go-stellar-sdk v0.7.3"},
		{"go-modern-decoder", nil, nil, ""},
		{"go-no-sdk", nil, nil, ""},
		{"js-legacy-decoder", []string{"src/headers.ts:4 high/high"}, nil, "decodes LedgerHeaderHistoryEntry.fromXDR with @stellar/stellar-sdk 16.3.1"},
		{"js-range-decoder", []string{"src/feed.js:6 high/low"}, nil, `the range "^16.0.0"`},
		{"js-xdr-only-decoder", nil, nil, ""},
		{"js-workspace-decoder", []string{"packages/web/src/headers.ts:4 high/high"}, nil, "@stellar/stellar-sdk 16.2.0"},
		{"go-value-arms", []string{
			"value.go:7 high/medium",
			"value.go:20 medium/medium",
			"value.go:25 medium/high",
			"value.go:31 high/high",
		}, nil, "no case for the CAP-88 _MS arms and no default"},
		{"go-value-modern", nil, nil, ""},
		{"js-value-switch", []string{"arm.ts:4 medium/medium"}, nil, "reach the default branch"},
		{"xdr-defs-pre-cap88", []string{"Stellar-contract.x:188 medium/low", "Stellar-ledger.x:13 high/high"}, nil, "only inside an #ifdef block"},
		{"xdr-defs-curr", nil, nil, ""},
	})
}

func TestLockUseDir(t *testing.T) {
	for _, c := range []struct{ lockDir, location, want string }{
		{".", "node_modules/@stellar/stellar-sdk", "."},
		{".", "packages/web/node_modules/@stellar/stellar-sdk", "packages/web"},
		{"app", "node_modules/@stellar/stellar-sdk", "app"},
		{".", "node_modules/foo/node_modules/@stellar/stellar-sdk", ""},
		{".", "@stellar/stellar-sdk > stellar-base", ""},
		{".", "@stellar/stellar-sdk", "."},
	} {
		if got := lockUseDir(c.lockDir, c.location); got != c.want {
			t.Errorf("lockUseDir(%q, %q) = %q, want %q", c.lockDir, c.location, got, c.want)
		}
	}
}

func TestNearestUses(t *testing.T) {
	uses := []sdkUse{
		{Eco: deps.NPM, Name: "root", Dir: "."},
		{Eco: deps.NPM, Name: "web", Dir: "packages/web"},
		{Eco: deps.Go, Name: "go", Dir: "packages/web"},
	}
	names := func(us []sdkUse) []string {
		var out []string
		for _, u := range us {
			out = append(out, u.Name)
		}
		return out
	}
	if got := names(nearestUses(uses, deps.NPM, "packages/web/src/deep")); !reflect.DeepEqual(got, []string{"web"}) {
		t.Errorf("web file got %v", got)
	}
	if got := names(nearestUses(uses, deps.NPM, "packages/api")); !reflect.DeepEqual(got, []string{"root"}) {
		t.Errorf("api file got %v", got)
	}
	if got := nearestUses(uses, deps.Go, "cmd"); got != nil {
		t.Errorf("go file outside module got %v", got)
	}
}

// FuzzLedgerObservations checks the ledger and close-time matchers on
// arbitrary input.
func FuzzLedgerObservations(f *testing.F) {
	f.Add([]byte("switch (v.ext().switch().name) { case 'stellarValueSigned': default: }"))
	f.Add([]byte("x = ops / (a.closeTime - b.closeTime); if (a.closeTime <= b.closeTime) {}"))
	f.Add([]byte("enum StellarValueType\n{\n#ifdef X\n STELLAR_VALUE_SIGNED_MS = 3\n"))
	f.Add([]byte("/ ( - ) < > .closeTime"))
	f.Fuzz(func(t *testing.T, src []byte) {
		jsLedgerObservations(src)
		jsCloseTimeObservations(src)
		xdrDefinitionFindings("f.x", src)
		_, _ = goLedgerObservations("f.go", src)
		_, _ = goCloseTimeObservations("f.go", src)
	})
}
