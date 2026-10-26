package rules

import (
	"strings"
	"testing"

	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

const goAddressSDKImport = `"github.com/stellar/go-stellar-sdk/strkey"`

func goAddressFixture(imports, parameter, result, body, extra string) string {
	return "package fixture\nimport (\n" + imports + "\n)\n" + extra + "\nfunc decode(" + parameter + ") " + result + " {\n" + body + "\n}\n"
}

func TestGoAddressSupportedLocalProofs(t *testing.T) {
	pair := "(*strkey.MuxedContract, error)"
	for _, tc := range []struct {
		name, imports, parameter, result, body, api string
	}{
		{
			"has-prefix", goAddressSDKImport + "\n\"strings\"\n\"errors\"", "address string", pair,
			`if !strings.HasPrefix(address, "C") { return nil, errors.New("unsupported") }
return strkey.DecodeMuxedContract(address)`, "strkey.DecodeMuxedContract",
		},
		{
			"combined-prefixes", goAddressSDKImport + "\n\"strings\"\n\"errors\"", "address string", pair,
			`if !strings.HasPrefix(address, "G") && !strings.HasPrefix(address, "M") && !strings.HasPrefix(address, "C") { return nil, errors.New("unsupported") }
return strkey.DecodeMuxedContract(address)`, "strkey.DecodeMuxedContract",
		},
		{
			"index-comparison", goAddressSDKImport + "\n\"errors\"", "address string", pair,
			`if address[0] != 'C' { return nil, errors.New("unsupported") }
return strkey.DecodeMuxedContract(address)`, "strkey.DecodeMuxedContract",
		},
		{
			"reversed-byte-comparison", goAddressSDKImport + "\n\"errors\"", "address string", pair,
			`if 'C' != address[0] { return nil, errors.New("unsupported") }
return strkey.DecodeMuxedContract(address)`, "strkey.DecodeMuxedContract",
		},
		{
			"slice-prefix", goAddressSDKImport + "\n\"errors\"", "address string", pair,
			`if address[:1] != "C" { return nil, errors.New("unsupported") }
return strkey.DecodeMuxedContract(address)`, "strkey.DecodeMuxedContract",
		},
		{
			"explicit-slice-prefix", goAddressSDKImport + "\n\"fmt\"", "address string", pair,
			`if address[0:1] != "C" { return nil, fmt.Errorf("unsupported") }
return strkey.DecodeMuxedContract(address)`, "strkey.DecodeMuxedContract",
		},
		{
			"panic", goAddressSDKImport + "\n\"strings\"", "address string", "bool",
			`if !strings.HasPrefix(address, "C") { panic("unsupported") }
return strkey.IsValidMuxedContractAddress(address)`, "strkey.IsValidMuxedContractAddress",
		},
		{
			"boolean-rejection", goAddressSDKImport + "\n\"strings\"", "address string", "bool",
			`if !strings.HasPrefix(address, "C") { return false }
return strkey.IsValidMuxedContractAddress(address)`, "strkey.IsValidMuxedContractAddress",
		},
		{
			"version-byte", goAddressSDKImport + "\n\"strings\"\n\"errors\"", "address string", "([]byte, error)",
			`if !strings.HasPrefix(address, "C") { return nil, errors.New("unsupported") }
return strkey.Decode(strkey.VersionByteMuxedContract, address)`, "strkey.Decode(VersionByteMuxedContract)",
		},
		{
			"import-aliases", `keys "github.com/stellar/go-stellar-sdk/strkey"` + "\ntext \"strings\"\nerrs \"errors\"", "value string", "(*keys.MuxedContract, error)",
			`if !text.HasPrefix(value, "C") { return nil, errs.New("unsupported") }
return keys.DecodeMuxedContract(value)`, "strkey.DecodeMuxedContract",
		},
		{
			"regexp", goAddressSDKImport + "\n\"regexp\"\n\"errors\"", "address string", pair,
			`if !regexp.MustCompile("^[GMC][A-Z2-7]+$").MatchString(address) { return nil, errors.New("unsupported") }
return strkey.DecodeMuxedContract(address)`, "strkey.DecodeMuxedContract",
		},
		{
			"raw-regexp", goAddressSDKImport + "\n\"regexp\"\n\"errors\"", "address string", pair,
			"if !regexp.MustCompile(`^(G|M|C)[A-Z2-7]+$`).MatchString(address) { return nil, errors.New(\"unsupported\") }\nreturn strkey.DecodeMuxedContract(address)", "strkey.DecodeMuxedContract",
		},
		{
			"escaped-byte-and-parens", goAddressSDKImport + "\n\"errors\"", "address string", pair,
			`if ((address)[0]) != '\x43' { return nil, errors.New("unsupported") }
return (strkey.DecodeMuxedContract)((address))`, "strkey.DecodeMuxedContract",
		},
		{
			"direct-w-rejection", goAddressSDKImport + "\n\"strings\"\n\"errors\"", "address string", pair,
			`if strings.HasPrefix(address, "W") { return nil, errors.New("unsupported") }
return strkey.DecodeMuxedContract(address)`, "strkey.DecodeMuxedContract",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := goAddressFixture(tc.imports, tc.parameter, tc.result, tc.body, "")
			file := walker.File{Path: "src/decode.go", Kind: "go_source", Data: []byte(source)}
			for i := 0; i < 2; i++ {
				got, notes, err := analyzeGoAddress(file)
				if err != nil || len(notes) != 0 || len(got) != 1 {
					t.Fatalf("candidates=%+v notes=%v error=%v", got, notes, err)
				}
				candidate := got[0]
				if candidate.path != file.Path || candidate.api != tc.api || candidate.ecosystem != "go" || candidate.kind != addressWOnly {
					t.Fatalf("incorrect candidate: %+v", candidate)
				}
				if candidate.line != strings.Count(source[:strings.Index(source, "if ")], "\n")+1 || !strings.Contains(candidate.evidence, "if ") || !strings.Contains(candidate.evidence, "return ") {
					t.Fatalf("missing guard/sink source evidence: %+v", candidate)
				}
			}
		})
	}
}

func TestGoAddressRequiredNegatives(t *testing.T) {
	imports := goAddressSDKImport + "\n\"strings\"\n\"errors\""
	pair := "(*strkey.MuxedContract, error)"
	baseBody := `if !strings.HasPrefix(address, "C") { return nil, errors.New("unsupported") }
return strkey.DecodeMuxedContract(address)`
	for _, tc := range []struct {
		name, imports, parameter, result, body, extra string
	}{
		{"allows-w", imports, "address string", pair, strings.Replace(baseBody, `"C"`, `"W"`, 1), ""},
		{"unknown-long-prefix", imports, "address string", pair, strings.Replace(baseBody, `"C"`, `"WA"`, 1), ""},
		{"unrelated-guard-variable", imports, "address string", pair, strings.Replace(baseBody, "HasPrefix(address,", "HasPrefix(other,", 1), "var other string\n"},
		{"unrelated-sink-variable", imports, "address string", pair, strings.Replace(baseBody, "DecodeMuxedContract(address)", "DecodeMuxedContract(other)", 1), "var other string\n"},
		{"reassigned-parameter", imports, "address string", pair, "address = \"C\"\n" + baseBody, ""},
		{"intervening-call", imports, "address string", pair, strings.Replace(baseBody, "\nreturn", "\nprintln(address)\nreturn", 1), ""},
		{"guard-else", imports, "address string", pair, strings.Replace(baseBody, " }\nreturn", " } else { println(address) }\nreturn", 1), ""},
		{"guard-init", imports, "address string", pair, strings.Replace(baseBody, "if !", "if _ = address; !", 1), ""},
		{"guard-extra-statement", imports, "address string", pair, strings.Replace(baseBody, "{ return nil,", "{ println(address); return nil,", 1), ""},
		{"not-rejection", imports, "address string", pair, strings.Replace(baseBody, "errors.New(\"unsupported\")", "nil", 1), ""},
		{"unknown-error-factory", imports, "address string", pair, strings.Replace(baseBody, "errors.New(\"unsupported\")", "maybeError()", 1), "func maybeError() error { return nil }\n"},
		{"side-effectful-unknown-operand", imports, "address string", pair, strings.Replace(baseBody, "if !strings", "if rewrite(&address) || !strings", 1), "func rewrite(value *string) bool { *value = \"C\"; return false }\n"},
		{"unsupported-logical-operand", imports, "address string", pair, strings.Replace(baseBody, "if !strings", "if unrelated() || !strings", 1), "func unrelated() bool { return false }\n"},
		{"shadowed-sdk-import", imports, "address string", pair, baseBody, "func other() { strkey := 1; _ = strkey }\n"},
		{"shadowed-stdlib-import", imports, "address string", pair, baseBody, "func other() { strings := 1; _ = strings }\n"},
		{"fork-import", strings.Replace(imports, "github.com/stellar/go-stellar-sdk", "github.com/acme/fork", 1), "address string", pair, baseBody, ""},
		{"account-only-decoder", imports, "address string", "([]byte, error)", strings.Replace(baseBody, "strkey.DecodeMuxedContract(address)", "strkey.Decode(strkey.VersionByteAccountID, address)", 1), ""},
		{"contract-only-decoder", imports, "address string", "([]byte, error)", strings.Replace(baseBody, "strkey.DecodeMuxedContract(address)", "strkey.Decode(strkey.VersionByteContract, address)", 1), ""},
		{"muxed-contract-set-contract-id", imports, "address string", "error", strings.Replace(strings.Replace(baseBody, "return nil, errors.New", "return errors.New", 1), "strkey.DecodeMuxedContract(address)", "contract.SetContractID(address)", 1), "var contract strkey.MuxedContract\n"},
		{"multiple-parameters", imports, "address, other string", pair, baseBody, ""},
		{"typed-alias-parameter", imports, "address Address", pair, baseBody, "type Address string\n"},
		{"comment-and-string-lookalike", imports, "address string", pair, "// " + strings.ReplaceAll(baseBody, "\n", "\n// ") + "\nreturn strkey.DecodeMuxedContract(address)", "const source = `" + baseBody + "`\n"},
		{"closure-sink", imports, "address string", pair, strings.Replace(baseBody, "return strkey.DecodeMuxedContract(address)", "return func() (*strkey.MuxedContract,error) { return strkey.DecodeMuxedContract(address) }()", 1), ""},
		{"loop", imports, "address string", pair, "for {\n" + baseBody + "\n}", ""},
		{"shadowed-panic", goAddressSDKImport + "\n\"strings\"", "address string", "bool", "if !strings.HasPrefix(address, \"C\") { panic(\"unsupported\") }\nreturn strkey.IsValidMuxedContractAddress(address)", "func panic(any) {}\n"},
		{"true-is-not-rejection", goAddressSDKImport + "\n\"strings\"", "address string", "bool", "if !strings.HasPrefix(address, \"C\") { return true }\nreturn strkey.IsValidMuxedContractAddress(address)", ""},
		{"regex-allows-w", goAddressSDKImport + "\n\"regexp\"\n\"errors\"", "address string", pair, "if !regexp.MustCompile(`^[GMCW][A-Z2-7]+$`).MatchString(address) { return nil, errors.New(\"unsupported\") }\nreturn strkey.DecodeMuxedContract(address)", ""},
		{"unanchored-regex", goAddressSDKImport + "\n\"regexp\"\n\"errors\"", "address string", pair, "if !regexp.MustCompile(`[GMC]`).MatchString(address) { return nil, errors.New(\"unsupported\") }\nreturn strkey.DecodeMuxedContract(address)", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := goAddressFixture(tc.imports, tc.parameter, tc.result, tc.body, tc.extra)
			got, notes, err := analyzeGoAddress(walker.File{Path: "negative.go", Kind: "go_source", Data: []byte(source)})
			if err != nil || len(notes) != 0 || len(got) != 0 {
				t.Fatalf("negative source misclassified: candidates=%+v notes=%v error=%v\n%s", got, notes, err, source)
			}
		})
	}
}

func TestGoAddressNoGuardOrOfficialImport(t *testing.T) {
	for _, source := range []string{
		`package fixture
func valid(address string) bool { return true }`,
		`package fixture
import . "github.com/stellar/go-stellar-sdk/strkey"
func valid(address string) bool { if address[0] != 'C' { return false }; return IsValidMuxedContractAddress(address) }`,
		`package fixture
import "github.com/stellar/go-stellar-sdk/strkey"
func valid(address string) bool { return strkey.IsValidMuxedContractAddress(address) }`,
		`package fixture
import "github.com/stellar/go-stellar-sdk/strkey"
type Decoder struct{}
func (d Decoder) valid(address string) bool { if address[0] != 'C' { return false }; return strkey.IsValidMuxedContractAddress(address) }`,
	} {
		got, notes, err := analyzeGoAddress(walker.File{Path: "negative.go", Kind: "go_source", Data: []byte(source)})
		if err != nil || len(notes) != 0 || len(got) != 0 {
			t.Fatalf("negative source misclassified: candidates=%+v notes=%v error=%v", got, notes, err)
		}
	}
}

func TestGoAddressSyntaxFailureIsLimitation(t *testing.T) {
	got, notes, err := analyzeGoAddress(walker.File{Path: "invalid.go", Kind: "go_source", Data: []byte("package fixture\nfunc broken(")})
	if err != nil || len(got) != 0 || len(notes) != 1 || !strings.Contains(notes[0], "invalid.go") {
		t.Fatalf("syntax failure not explicit: candidates=%+v notes=%v error=%v", got, notes, err)
	}
}

func TestGoAddressNodeBudgetIsError(t *testing.T) {
	source := "package fixture\nfunc large() {\n" + strings.Repeat("_ = 0\n", 80000) + "}\n"
	got, _, err := analyzeGoAddress(walker.File{Path: "large.go", Kind: "go_source", Data: []byte(source)})
	if err == nil || len(got) != 0 || !strings.Contains(err.Error(), "nodes") {
		t.Fatalf("budget exhaustion hidden: candidates=%+v error=%v", got, err)
	}
}
