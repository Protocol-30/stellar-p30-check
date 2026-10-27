package rules

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

const addrFixtures = "../../testdata/rules/addr-001"

func checkAddresses(t *testing.T, dir string) Output {
	t.Helper()
	res, err := walker.Walk(dir, walker.Options{})
	if err != nil {
		t.Fatal(err)
	}
	out, err := Addresses{}.Check(context.Background(), Input{
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
		if f.Evidence == "" || f.Remediation == "" || len(f.References) == 0 {
			t.Errorf("finding lacks evidence, remediation or references: %+v", f)
		}
	}
	return out
}

func TestAddressFixtures(t *testing.T) {
	cases := []struct {
		dir      string
		findings []string
		warnings []string
		message  string
	}{
		// Positives.
		{"js-regex", []string{"validate.ts:2 high/high"}, nil, "address regex accepts G, M, C but not W"},
		{"js-predicates", []string{"address.js:5 high/high"}, nil, "SDK address checks accept G, M, C but not W"},
		{"js-prefix-tests", []string{"Destination.jsx:2 high/medium"}, nil, "prefix checks accept G, M, C"},
		{"js-letter-list", []string{"prefix.ts:1 high/medium"}, nil, "prefix list accepts G, M, C"},
		{"js-account-contract", []string{"target.ts:2 info/low"}, nil, "accepts G, C but not W"},
		{"js-switch", []string{"decode.ts:4 medium/medium"}, nil, "W addresses reach the default branch"},
		{"go-regex", []string{"addr.go:5 high/high"}, nil, "address regex accepts G, M, C"},
		{"go-predicates", []string{"addr.go:7 high/high"}, nil, "SDK address checks accept G, M, C"},
		{"go-switch", []string{"scaddress.go:7 high/medium"}, nil, "silently skipped"},
		{"go-prefix", []string{"prefix.go:7 info/low"}, nil, "prefix checks accept G, C"},
		{"js-class-methods", []string{"wallet.ts:5 high/high"}, nil, "SDK address checks accept G, M, C"},
		// Negatives.
		{"js-modern", nil, nil, ""},
		{"js-account-only", nil, nil, ""},
		{"js-contract-only", nil, nil, ""},
		{"js-unrelated", nil, nil, ""},
		{"js-test-files", nil, nil, ""},
		{"go-modern", nil, nil, ""},
		{"go-lookalike", nil, nil, ""},
		{"js-w-same-function", nil, nil, ""},
		{"go-w-same-function", nil, nil, ""},
		{"go-broken", nil, []string{"source-unparsed:broken.go"}, ""},
	}
	for _, c := range cases {
		t.Run(c.dir, func(t *testing.T) {
			out := checkAddresses(t, filepath.Join(addrFixtures, c.dir))
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

func TestRegexAllowlist(t *testing.T) {
	cases := map[string]string{ // pattern -> letters, "" for not an address allowlist
		`^[GMC][A-Z2-7]{55}$`:       "G, M, C",
		`^[GC][A-Z0-9]{55}$`:        "G, C",
		`^(G|M|C)[A-Z2-7]+$`:        "G, M, C",
		`^(?:G|C)[A-Z\d]{55}$`:      "G, C",
		`^[G|M|C][A-Z2-7]{55}$`:     "G, M, C",
		`^[GMCW][A-Z2-7]{55}`:       "G, M, C, W",
		`^[A-Z][A-Z2-7]{55}$`:       "", // A is not a strkey prefix
		`^[A-C]\d+$`:                "",
		`[GMC][A-Z2-7]{55}`:         "", // not anchored
		`^[^GMC][A-Z2-7]{55}$`:      "",
		`^[GMC]\w+$`:                "", // body is not a strkey class
		`^[GMC]$`:                   "",
		`^(GA|GB)[A-Z2-7]{54}$`:     "",
		`^[gmc][a-z2-7]{55}$`:       "",
		`^[GMC][A-Z2-7]{55}$|^foo$`: "G, M, C",
		`^[GM-P][A-Z2-7]{55}$`:      "", // range covers N, O
		`^[L-M][A-Z2-7]{55}$`:       "M, L",
		`^(G|MM)[A-Z2-7]{55}$`:      "",
	}
	for pattern, want := range cases {
		k, ok := regexAllowlist(pattern)
		got := ""
		if ok {
			got = k.String()
		}
		if got != want {
			t.Errorf("regexAllowlist(%q) = %q, want %q", pattern, got, want)
		}
	}
}

func TestJSTokenize(t *testing.T) {
	src := "const a = b / c / d; // x /^[GMC]/\n" +
		"const re = /^[G/]x/gi; /* block\ncomment */ const s = 'it\\'s';\n" +
		"const t = `a ${ {x: '}'}.x } b`; return /y/;\n" +
		"<p>Don't</p>; const z = \"q\";"
	var got []string
	for _, tok := range jsTokenize([]byte(src)) {
		switch tok.Kind {
		case jsRegex:
			got = append(got, "re:"+tok.Text+"/"+tok.Flags+"@"+string(rune('0'+tok.Line)))
		case jsString:
			got = append(got, "str:"+tok.Text+"@"+string(rune('0'+tok.Line)))
		case jsTemplate:
			got = append(got, "tpl@"+string(rune('0'+tok.Line)))
		}
	}
	want := []string{"re:^[G/]x/gi@2", "str:it's@3", "tpl@4", "re:y/@4", "str:q@5"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tokens = %v, want %v", got, want)
	}
}

func TestIsTestFile(t *testing.T) {
	for p, want := range map[string]bool{
		"pkg/addr_test.go":         true,
		"src/addr.test.ts":         true,
		"src/addr.spec.jsx":        true,
		"src/__tests__/helpers.js": true,
		"__tests__/a.ts":           true,
		"src/attest.ts":            false,
		"pkg/addr.go":              false,
		"src/latest.js":            false,
	} {
		if got := isTestFile(p); got != want {
			t.Errorf("isTestFile(%q) = %v, want %v", p, got, want)
		}
	}
}

func TestAddressesUnreadableFileIsAWarning(t *testing.T) {
	out, err := Addresses{}.Check(context.Background(), Input{
		Files:    []walker.File{{Path: "a.ts", Kind: "source-ts", Category: walker.CategorySource}},
		ReadFile: func(string) ([]byte, error) { return nil, os.ErrPermission },
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := warningCodes(out.Warnings); !reflect.DeepEqual(got, []string{"source-unparsed:a.ts"}) {
		t.Errorf("warnings = %v", got)
	}
}

// FuzzJSAddressObservations checks that arbitrary input never panics the
// tokenizer or the JS/TS shape matcher.
func FuzzJSAddressObservations(f *testing.F) {
	for _, seed := range []string{
		"const r = /^[GMC][A-Z2-7]{55}$/;",
		"switch (a) { case 'scAddressTypeContract': default: }",
		"x.startsWith('G') || x[0] === 'C' || 'M' === x.charAt(0)",
		"`${`${'}'}`}` / / [ ' \" /* ",
		"StrKey.isValidContract(a) || StrKey.",
		"[ 'G', 'C'",
		"class { { } } } {",
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		for _, o := range jsAddressObservations(src) {
			if o.line < 1 {
				t.Fatalf("observation with line %d", o.line)
			}
		}
	})
}

// FuzzRegexAllowlist checks the pattern analyser on arbitrary text.
func FuzzRegexAllowlist(f *testing.F) {
	f.Add(`^[GMC][A-Z2-7]{55}$`)
	f.Add(`^(G|`)
	f.Add(`^[Z-A][`)
	f.Fuzz(func(t *testing.T, p string) { regexAllowlist(p) })
}

// FuzzGoAddressObservations checks the Go matcher on arbitrary source.
func FuzzGoAddressObservations(f *testing.F) {
	f.Add([]byte("package p\nimport \"strings\"\nvar _ = strings.HasPrefix(s, \"G\") || s[0] == 'C' || s[:1] == \"M\"\n"))
	f.Add([]byte("package p\nfunc f() { switch x { case xdr.ScAddressTypeScAddressTypeContract: default: } }\n"))
	f.Add([]byte("package p\nvar _ = map[byte]bool{'G': true, 'C': true}\nvar _ = []string{\"G\", `C`}\n"))
	f.Fuzz(func(t *testing.T, src []byte) {
		_, _ = goAddressObservations("f.go", src)
	})
}
