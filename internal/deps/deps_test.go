package deps

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestParseVersion(t *testing.T) {
	good := map[string]string{
		"1.2.3":                                "1.2.3",
		"v0.8.0":                               "0.8.0",
		"17.3.0-rc.1":                          "17.3.0-rc.1",
		"1.0.0+build.5":                        "1.0.0",
		"v0.8.1-0.20261021000000-abcdef123456": "0.8.1-0.20261021000000-abcdef123456",
	}
	for in, want := range good {
		v, ok := ParseVersion(in)
		if !ok || v.String() != want {
			t.Errorf("ParseVersion(%q) = %v, %v; want %s", in, v, ok, want)
		}
	}
	for _, bad := range []string{"", "17", "17.3", "1.2.3.4", "01.2.3", "1.2.x", "1.2.3-", "1.2.3-a..b", "latest", "^1.2.3"} {
		if v, ok := ParseVersion(bad); ok {
			t.Errorf("ParseVersion(%q) = %v, want failure", bad, v)
		}
	}
}

func TestVersionOrdering(t *testing.T) {
	// Ascending, per semver 2.0.0 section 11.
	order := []string{
		"0.7.3", "0.8.0-rc.1", "0.8.0", "0.8.1-0.20261021000000-abcdef123456", "0.8.1",
		"1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta", "1.0.0-beta.2",
		"1.0.0-beta.11", "1.0.0-rc.1", "1.0.0", "16.4.0", "16.10.0", "17.0.0",
	}
	vs := make([]Version, len(order))
	for i, s := range order {
		vs[i] = must(s)
	}
	shuffled := append([]Version{}, vs...)
	for i, j := 0, len(shuffled)-1; i < j; i, j = i+1, j-1 {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	sort.Slice(shuffled, func(i, j int) bool { return shuffled[i].Less(shuffled[j]) })
	if !reflect.DeepEqual(shuffled, vs) {
		t.Errorf("sorted = %v\nwant %v", shuffled, vs)
	}
	if must("1.2.3").Compare(must("v1.2.3")) != 0 {
		t.Error("v prefix changed comparison")
	}
}

// TestCapabilityMatrix pins every boundary recorded in docs/evidence.md.
func TestCapabilityMatrix(t *testing.T) {
	type want struct{ xdr, w Support }
	var (
		none    = want{No, No}
		xdrOnly = want{Yes, No}
		full    = want{Yes, Yes}
		unknown = want{Unknown, Unknown}
	)
	cases := []struct {
		eco     Ecosystem
		pkg     string
		version string
		want    want
	}{
		// E10: go-stellar-sdk v0.8.0 is the first P30 release.
		{Go, "github.com/stellar/go-stellar-sdk", "v0.1.0", none},
		{Go, "github.com/stellar/go-stellar-sdk", "v0.7.3", none},
		{Go, "github.com/stellar/go-stellar-sdk", "v0.8.0", full},
		{Go, "github.com/stellar/go-stellar-sdk", "v0.9.0", full},
		{Go, "github.com/stellar/go-stellar-sdk", "v0.8.0-rc.1", unknown},
		{Go, "github.com/stellar/go-stellar-sdk", "v0.7.4-0.20260901000000-abcdef123456", unknown},
		{Go, "github.com/stellar/go-stellar-sdk", "v0.8.1-0.20261101000000-abcdef123456", full},
		// E11: the deprecated module never gained P30 support.
		{Go, "github.com/stellar/go", "v0.0.0-20251210100531-aab2ea4aca88", none},
		// E12: @stellar/stellar-sdk.
		{NPM, "@stellar/stellar-sdk", "14.6.1", none},
		{NPM, "@stellar/stellar-sdk", "16.3.1", none},
		{NPM, "@stellar/stellar-sdk", "16.4.0", xdrOnly},
		{NPM, "@stellar/stellar-sdk", "16.4.1", xdrOnly},
		{NPM, "@stellar/stellar-sdk", "17.0.0", none},
		{NPM, "@stellar/stellar-sdk", "17.2.1", none},
		{NPM, "@stellar/stellar-sdk", "17.3.0", full},
		{NPM, "@stellar/stellar-sdk", "17.4.0", full},
		{NPM, "@stellar/stellar-sdk", "17.0.0-rc.2", unknown},
		{NPM, "@stellar/stellar-sdk", "17.3.0-rc.1", unknown},
		// E13, E14.
		{NPM, "@stellar/stellar-base", "15.0.0", none},
		{NPM, "@stellar/stellar-base", "15.1.0", unknown},
		{NPM, "stellar-sdk", "13.3.0", none},
		{NPM, "stellar-sdk", "14.0.0", unknown},
		{NPM, "stellar-base", "13.1.0", none},
	}
	for _, c := range cases {
		a, ok := Assess(c.eco, c.pkg, must(c.version))
		if !ok {
			t.Fatalf("%s not tracked", c.pkg)
		}
		if got := (want{a.LedgerXDR, a.MuxedContract}); got != c.want {
			t.Errorf("%s %s: xdr=%v w=%v, want xdr=%v w=%v", c.pkg, c.version, got.xdr, got.w, c.want.xdr, c.want.w)
		}
		if a.Basis == "" || len(a.References) == 0 {
			t.Errorf("%s %s: assessment lacks basis or references", c.pkg, c.version)
		}
		if c.want != full && a.Remediation == "" {
			t.Errorf("%s %s: no remediation", c.pkg, c.version)
		}
	}
	for _, untracked := range []string{"@stellar/freighter-api", "soroban-client", "github.com/stellar/go-xdr"} {
		if Tracked(NPM, untracked) || Tracked(Go, untracked) {
			t.Errorf("%s should not be tracked", untracked)
		}
	}
}

func TestParseGoMod(t *testing.T) {
	src := `// comment line
module example.com/app // trailing

go 1.25.0

require github.com/stellar/go-stellar-sdk v0.7.3

require (
	github.com/pkg/errors v0.9.1 // indirect
	"github.com/stellar/go" v0.0.0-20251210100531-aab2ea4aca88
)

replace github.com/stellar/go-stellar-sdk v0.7.3 => ../fork

replace (
	github.com/a/b => github.com/c/d v1.0.0
)

exclude github.com/x/y v1.0.0
`
	m, err := ParseGoMod([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if m.Module != "example.com/app" {
		t.Errorf("module = %q", m.Module)
	}
	wantReq := []Requirement{
		{Path: "github.com/stellar/go-stellar-sdk", Version: "v0.7.3", Line: 6},
		{Path: "github.com/pkg/errors", Version: "v0.9.1", Line: 9, Indirect: true},
		{Path: "github.com/stellar/go", Version: "v0.0.0-20251210100531-aab2ea4aca88", Line: 10},
	}
	if !reflect.DeepEqual(m.Requires, wantReq) {
		t.Errorf("requires = %+v\nwant %+v", m.Requires, wantReq)
	}
	r, ok := m.ReplacementFor("github.com/stellar/go-stellar-sdk", "v0.7.3")
	if !ok || r.New != "../fork" || r.NewVersion != "" || r.Line != 13 {
		t.Errorf("replacement = %+v, %v", r, ok)
	}
	if _, ok := m.ReplacementFor("github.com/stellar/go-stellar-sdk", "v0.8.0"); ok {
		t.Error("version-specific replace applied to another version")
	}
	if r, ok := m.ReplacementFor("github.com/a/b", "v9.9.9"); !ok || r.New != "github.com/c/d" || r.NewVersion != "v1.0.0" {
		t.Errorf("general replacement = %+v, %v", r, ok)
	}
}

func TestParseGoModErrors(t *testing.T) {
	for name, src := range map[string]string{
		"unterminated block":  "require (\n\tgithub.com/a/b v1.0.0\n",
		"short require":       "require github.com/a/b\n",
		"replace without =>":  "replace github.com/a/b github.com/c/d v1.0.0\n",
		"unterminated string": "require \"github.com/a/b v1.0.0\n",
	} {
		if _, err := ParseGoMod([]byte(src)); err == nil {
			t.Errorf("%s: parsed without error", name)
		} else if !strings.Contains(err.Error(), "go.mod") {
			t.Errorf("%s: error lacks context: %v", name, err)
		}
	}
}

func TestParsePackageJSON(t *testing.T) {
	src := `{
  "name": "app",
  "dependencies": {
    "react": "^18.0.0",
    "@stellar/stellar-sdk": "^16.0.0"
  },
  "devDependencies": {
    "stellar": "npm:@stellar/stellar-sdk@17.3.0"
  },
  "peerDependencies": null
}`
	got, err := ParsePackageJSON([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	want := []NPMDependency{
		{Name: "@stellar/stellar-sdk", Key: "@stellar/stellar-sdk", Spec: "^16.0.0", Field: "dependencies", Line: 5},
		{Name: "react", Key: "react", Spec: "^18.0.0", Field: "dependencies", Line: 4},
		{Name: "@stellar/stellar-sdk", Key: "stellar", Spec: "17.3.0", Field: "devDependencies", Line: 8},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("deps = %+v\nwant %+v", got, want)
	}
	for _, bad := range []string{`{`, `[]`, `{"dependencies": ["x"]}`, `{"dependencies": {"x": 1}}`} {
		if _, err := ParsePackageJSON([]byte(bad)); err == nil {
			t.Errorf("ParsePackageJSON(%s) accepted", bad)
		}
	}
}

func TestParseNPMLockV3(t *testing.T) {
	src := `{
  "name": "app",
  "lockfileVersion": 3,
  "packages": {
    "": {"name": "app", "dependencies": {"@stellar/stellar-sdk": "^16.0.0"}},
    "node_modules/@stellar/stellar-sdk": {"version": "16.3.1"},
    "packages/web/node_modules/@stellar/stellar-sdk": {"version": "17.3.0"},
    "node_modules/stellar": {"name": "@stellar/stellar-sdk", "version": "16.4.0"},
    "node_modules/web": {"resolved": "packages/web", "link": true},
    "packages/web": {"version": "1.0.0"}
  }
}`
	got, err := ParseNPMLock([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	want := []LockedPackage{
		{Name: "@stellar/stellar-sdk", Version: "16.3.1", Location: "node_modules/@stellar/stellar-sdk", Line: 6},
		{Name: "@stellar/stellar-sdk", Version: "16.4.0", Location: "node_modules/stellar", Line: 8},
		{Name: "@stellar/stellar-sdk", Version: "17.3.0", Location: "packages/web/node_modules/@stellar/stellar-sdk", Line: 7},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("locked = %+v\nwant %+v", got, want)
	}
}

func TestParseNPMLockV1(t *testing.T) {
	src := `{
  "lockfileVersion": 1,
  "dependencies": {
    "@stellar/stellar-sdk": {
      "version": "16.2.0",
      "dependencies": {"stellar-base": {"version": "13.1.0"}}
    },
    "stellar": {"version": "npm:@stellar/stellar-sdk@17.3.0"}
  }
}`
	got, err := ParseNPMLock([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	want := []LockedPackage{
		{Name: "@stellar/stellar-sdk", Version: "16.2.0", Location: "@stellar/stellar-sdk", Line: 4},
		{Name: "stellar-base", Version: "13.1.0", Location: "@stellar/stellar-sdk > stellar-base", Line: 6},
		{Name: "@stellar/stellar-sdk", Version: "17.3.0", Location: "stellar", Line: 8},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("locked = %+v\nwant %+v", got, want)
	}
	if _, err := ParseNPMLock([]byte(`{"packages": 5}`)); err == nil {
		t.Error("malformed lockfile accepted")
	}
}

func TestResolveSpec(t *testing.T) {
	cases := []struct {
		spec string
		want string
		kind SpecKind
	}{
		{"17.3.0", "17.3.0", SpecExact},
		{"=16.4.0", "16.4.0", SpecExact},
		{"v17.3.0", "17.3.0", SpecExact},
		{"^16.0.0", "16.0.0", SpecRange},
		{"~17.2.1", "17.2.1", SpecRange},
		{">=17.3.0", "17.3.0", SpecRange},
		{"^ 17.3.0", "17.3.0", SpecRange},
		{"17.x", "17.0.0", SpecRange},
		{"17.3", "17.3.0", SpecRange},
		{"^17", "17.0.0", SpecRange},
		{"16.4.*", "16.4.0", SpecRange},
		{"*", "", SpecUnresolved},
		{"x", "", SpecUnresolved},
		{"", "", SpecUnresolved},
		{"latest", "", SpecUnresolved},
		{">=16.0.0 <17.0.0", "", SpecUnresolved},
		{"^16.0.0 || ^17.0.0", "", SpecUnresolved},
		{"16.0.0 - 17.0.0", "", SpecUnresolved},
		{"github:stellar/js-stellar-sdk", "", SpecUnresolved},
		{"file:../sdk", "", SpecUnresolved},
		{"workspace:*", "", SpecUnresolved},
		{"17.x.3", "", SpecUnresolved},
	}
	for _, c := range cases {
		v, kind := ResolveSpec(c.spec)
		if kind != c.kind || (kind != SpecUnresolved && v.String() != c.want) {
			t.Errorf("ResolveSpec(%q) = %v, %v; want %s, %v", c.spec, v, kind, c.want, c.kind)
		}
	}
}
