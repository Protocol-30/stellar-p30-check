package rules

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

// memInput builds an Input over in-memory files and counts reads per path.
func memInput(files map[string]string) (Input, map[string]int) {
	reads := map[string]int{}
	var mu sync.Mutex
	var list []walker.File
	for p, body := range files {
		kind, cat, ok := walker.Classify(path.Base(p))
		if !ok {
			continue
		}
		list = append(list, walker.File{Path: p, Kind: kind, Category: cat, Size: int64(len(body))})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Path < list[j].Path })
	return Input{
		Files: list,
		ReadFile: func(p string) ([]byte, error) {
			mu.Lock()
			reads[p]++
			mu.Unlock()
			body, ok := files[p]
			if !ok {
				return nil, os.ErrNotExist
			}
			return []byte(body), nil
		},
	}, reads
}

// TestAdversarialInputsStayLinear pins the inputs that once made matchers
// quadratic (each took over 60 s at about 1 MB). Every case must finish well
// inside the budget; a quadratic regression at these sizes takes minutes.
func TestAdversarialInputsStayLinear(t *testing.T) {
	// The plain `go test` run is the regression gate: quadratic code took
	// over 20 s per case at these sizes. Race builds run several times
	// slower, so they get a looser budget.
	budget := 10 * time.Second
	if raceEnabled {
		budget = 60 * time.Second
	}
	n := 100000
	cases := map[string]string{
		"nested-division.js":   strings.Repeat("a / (", n) + "b" + strings.Repeat(")", n) + ";\n",
		"nested-switch.js":     strings.Repeat("switch (a) {", n/2) + strings.Repeat("}", n/2) + "\n",
		"nested-compare.js":    strings.Repeat("(", n) + "a.closeTime" + strings.Repeat("<b.closeTime)", n) + ";\n",
		"nested-class.js":      strings.Repeat("class A {", n) + `m(){ return a.startsWith("G")||a.startsWith("C"); }` + strings.Repeat("}", n) + "\n",
		"chained-compare.js":   "x" + strings.Repeat("<y", 4*n) + ";\n",
		"open-braces.js":       strings.Repeat("{", 3*n) + "\n",
		"unclosed-quotes.js":   strings.Repeat("'", 5*n) + "\n",
		"many-comparisons.go":  "package p\nimport \"github.com/stellar/go-stellar-sdk/xdr\"\nfunc f(v xdr.StellarValue) {\n" + strings.Repeat("\t_ = v.Ext.V == xdr.StellarValueTypeStellarValueSigned\n", n/5) + "}\n",
		"many-prefix-tests.go": "package p\nimport \"strings\"\nfunc f(s string) {\n" + strings.Repeat("\t_ = strings.HasPrefix(s, \"G\") || strings.HasPrefix(s, \"C\")\n", n/5) + "}\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			in, _ := memInput(map[string]string{name: body})
			start := time.Now()
			if _, err := CheckAll(context.Background(), in, Default()); err != nil {
				t.Fatal(err)
			}
			if d := time.Since(start); d > budget {
				t.Errorf("took %v, budget %v", d, budget)
			}
		})
	}
}

// TestCheckAllReadsEachSourceOnce checks the shared pass: three source rules,
// one read per source file. Manifests are read by P30-DEP-001 and by
// P30-XDR-001's version resolution.
func TestCheckAllReadsEachSourceOnce(t *testing.T) {
	in, reads := memInput(map[string]string{
		"go.mod":           "module x\n\nrequire github.com/stellar/go-stellar-sdk v0.7.3\n",
		"main.go":          "package main\nimport \"github.com/stellar/go-stellar-sdk/xdr\"\nvar _ xdr.LedgerHeader\n",
		"web/package.json": `{"dependencies": {"@stellar/stellar-sdk": "16.3.1"}}`,
		"web/a.ts":         "const r = /^[GMC][A-Z2-7]{55}$/;\n",
		"web/b.js":         "x / (a.closeTime - b.closeTime);\n",
		"defs/Stellar.x":   "enum StellarValueType\n{\n    STELLAR_VALUE_BASIC = 0\n};\n",
	})
	outs, err := CheckAll(context.Background(), in, Default())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"main.go", "web/a.ts", "web/b.js", "defs/Stellar.x"} {
		if reads[p] != 1 {
			t.Errorf("%s read %d times, want 1", p, reads[p])
		}
	}
	total := 0
	for _, o := range outs {
		total += len(o.Findings)
	}
	if total == 0 {
		t.Error("expected findings from the shared pass")
	}
}

// TestCheckAllMatchesSeparateChecks runs every fixture directory through the
// shared pass and through each rule's own Check; the results must agree.
func TestCheckAllMatchesSeparateChecks(t *testing.T) {
	dirs, err := filepath.Glob("../../testdata/rules/*/*")
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range dirs {
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			continue
		}
		res, err := walker.Walk(dir, walker.Options{})
		if err != nil {
			t.Fatal(err)
		}
		in := Input{Files: res.Files, ReadFile: func(rel string) ([]byte, error) {
			return os.ReadFile(filepath.Join(res.Root, filepath.FromSlash(rel)))
		}}
		rs := Default()
		shared, err := CheckAll(context.Background(), in, rs)
		if err != nil {
			t.Fatal(err)
		}
		for i, r := range rs {
			alone, err := r.Check(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(alone, shared[i]) {
				t.Errorf("%s / %s: CheckAll and Check differ\nshared: %+v\nalone:  %+v", dir, r.ID(), shared[i], alone)
			}
		}
	}
}

func TestCheckAllErrorsNameTheRule(t *testing.T) {
	in, _ := memInput(map[string]string{"a.go": "package a\n"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CheckAll(ctx, in, Default()); err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("cancelled scan: err = %v", err)
	}
	_, err := CheckAll(context.Background(), Input{
		Files:    []walker.File{{Path: "go.mod", Kind: "go-mod"}},
		ReadFile: func(string) ([]byte, error) { return nil, fmt.Errorf("unexpected") },
	}, []Rule{failing{}})
	if err == nil || !strings.Contains(err.Error(), "rule P30-FAIL") {
		t.Errorf("err = %v, want it to name the rule", err)
	}
}

type failing struct{}

func (failing) ID() string { return "P30-FAIL" }
func (failing) Check(context.Context, Input) (Output, error) {
	return Output{}, fmt.Errorf("boom")
}

// FuzzCheckAll feeds the same bytes to every rule as each kind of file at
// once (Go, TS, go.mod, package.json, npm lockfile, .x definitions). Any
// input must produce valid findings and never panic.
func FuzzCheckAll(f *testing.F) {
	f.Add([]byte("module x\nrequire github.com/stellar/go-stellar-sdk v0.7.3\n"))
	f.Add([]byte(`{"dependencies":{"@stellar/stellar-sdk":"^16"},"packages":{"node_modules/@stellar/stellar-sdk":{"version":"16.3.1"}}}`))
	f.Add([]byte("package p\nfunc f(){ switch x { case xdr.StellarValueTypeStellarValueSigned: } }\n"))
	f.Add([]byte("enum StellarValueType\n{\n#ifdef X\n STELLAR_VALUE_SIGNED_MS = 3,\n#endif\n};\n"))
	f.Add([]byte("\xef\xbb\xbfconst r = /^[GMC][A-Z2-7]{55}$/;\r\n\x00\xff"))
	f.Fuzz(func(t *testing.T, src []byte) {
		body := string(src)
		in, _ := memInput(map[string]string{
			"a.go": body, "a.ts": body, "go.mod": body, "package.json": body,
			"package-lock.json": body, "Stellar-ledger.x": body,
		})
		outs, err := CheckAll(context.Background(), in, Default())
		if err != nil {
			t.Fatal(err)
		}
		for _, o := range outs {
			for _, fd := range o.Findings {
				if err := fd.Validate(); err != nil {
					t.Fatalf("invalid finding %+v: %v", fd, err)
				}
			}
		}
	})
}
