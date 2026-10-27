package scan

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/Protocol-30/stellar-p30-check/internal/rules"
)

// TestMalformedInputs scans files that are hostile or broken in ways real
// repositories contain: binary data under source extensions, NUL bytes,
// invalid UTF-8, a byte order mark with CRLF lines, empty manifests and JSON
// nested past the decoder's limit. The scan must complete, report each
// unreadable file as a warning, and give identical results twice.
func TestMalformedInputs(t *testing.T) {
	binary := string(bytes.Repeat([]byte{0x00, 0xff, 0xfe, 0x80, '`', '"', '/', '{'}, 4096))
	files := map[string]string{
		"bin/blob.go":        binary,
		"bin/blob.ts":        binary,
		"bin/blob.x":         binary,
		"bin/package.json":   binary,
		"bin/go.mod":         binary,
		"utf8/bad.js":        "const s = \"\xff\xfe\"; const r = /^[GMC][A-Z2-7]{55}$/;\n",
		"bom/addr.go":        "\xef\xbb\xbfpackage addr\r\n\r\nimport \"regexp\"\r\n\r\nvar re = regexp.MustCompile(`^[GMC][A-Z2-7]{55}$`)\r\n",
		"empty/package.json": "",
		"empty/go.mod":       "",
		"empty/a.ts":         "",
		"empty/Stellar.x":    "",
		"deep/package-lock.json": `{"packages":{"node_modules/@stellar/stellar-sdk":{"version":"16.3.1"}},"x":` +
			strings.Repeat("[", 20000) + strings.Repeat("]", 20000) + "}",
	}
	root := tree(t, files)
	cfg := Config{Target: root, Now: fixed, Rules: rules.Default()}
	rep, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rep, again) {
		t.Error("two scans of the same tree differ")
	}

	warned := map[string]string{}
	for _, w := range rep.Warnings {
		warned[w.Path] = w.Code
	}
	for p, code := range map[string]string{
		"bin/blob.go":            "source-unparsed",
		"bin/package.json":       "manifest-unparsed",
		"bin/go.mod":             "manifest-unparsed",
		"empty/package.json":     "manifest-unparsed",
		"deep/package-lock.json": "manifest-unparsed",
	} {
		if warned[p] != code {
			t.Errorf("%s: warning %q, want %q (all: %+v)", p, warned[p], code, rep.Warnings)
		}
	}

	// Files that are odd but valid are still checked, with correct lines.
	found := map[string]int{}
	for _, f := range rep.Findings {
		if f.RuleID == rules.AddressesID {
			found[f.Path] = f.Line
		}
	}
	if found["utf8/bad.js"] != 1 || found["bom/addr.go"] != 5 {
		t.Errorf("address findings by path = %v, want utf8/bad.js:1 and bom/addr.go:5", found)
	}
}
