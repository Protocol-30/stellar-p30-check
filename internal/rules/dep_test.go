package rules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Protocol-30/stellar-p30-check/internal/report"
	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

const depFixtures = "../../testdata/rules/dep-001"

// checkDir runs the dependency rule over a fixture directory.
func checkDir(t *testing.T, dir string) Output {
	t.Helper()
	res, err := walker.Walk(dir, walker.Options{})
	if err != nil {
		t.Fatal(err)
	}
	out, err := Dependencies{}.Check(context.Background(), Input{
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

// summarize renders findings as "path:line severity/confidence".
func summarize(fs []report.Finding) []string {
	out := []string{}
	for _, f := range fs {
		out = append(out, fmt.Sprintf("%s:%d %s/%s", f.Path, f.Line, f.Severity, f.Confidence))
	}
	return out
}

func warningCodes(ws []report.Warning) []string {
	out := []string{}
	for _, w := range ws {
		out = append(out, w.Code+":"+w.Path)
	}
	return out
}

func TestDependenciesFixtures(t *testing.T) {
	cases := []struct {
		dir      string
		findings []string
		warnings []string
		message  string // substring of the first finding's message
	}{
		{"go-legacy", []string{"go.mod:5 high/high"}, nil, "github.com/stellar/go-stellar-sdk v0.7.3: no Protocol 30 support"},
		{"go-deprecated", []string{"go.mod:6 high/high"}, nil, "deprecated"},
		{"go-modern", nil, nil, ""},
		{"go-replaced", []string{"go.mod:5 info/low"}, nil, "replaced by a fork or local directory"},
		{"js-range-old", []string{"package.json:5 high/low"}, nil, `the range "^16.0.0" for @stellar/stellar-sdk allows 16.0.0`},
		{"js-locked-old", []string{"package-lock.json:12 high/high"}, nil, "@stellar/stellar-sdk 16.3.1: no Protocol 30 support"},
		{"js-xdr-only", []string{"package.json:5 medium/high"}, nil, "cannot handle W muxed-contract addresses"},
		{"js-modern", nil, nil, ""},
		{"js-yarn", []string{"package.json:5 high/low"}, []string{"lockfile-not-parsed:yarn.lock"}, "allows 17.2.0"},
		{"js-workspace", []string{"package-lock.json:13 high/high"}, nil, "@stellar/stellar-base 15.0.0"},
		{"js-unresolved", []string{"package.json:5 info/low", "package.json:6 info/low"}, nil, "could not be checked"},
		{"malformed", nil, []string{"manifest-unparsed:go.mod", "manifest-unparsed:package.json"}, ""},
	}
	for _, c := range cases {
		t.Run(c.dir, func(t *testing.T) {
			out := checkDir(t, filepath.Join(depFixtures, c.dir))
			if got, want := summarize(out.Findings), append([]string{}, c.findings...); !reflect.DeepEqual(got, want) {
				t.Errorf("findings = %v, want %v\n%+v", got, want, out.Findings)
			}
			want := append([]string{}, c.warnings...)
			if got := warningCodes(out.Warnings); !reflect.DeepEqual(got, want) {
				t.Errorf("warnings = %v, want %v", got, want)
			}
			if c.message != "" && len(out.Findings) > 0 && !strings.Contains(out.Findings[0].Message, c.message) {
				t.Errorf("message = %q, want it to contain %q", out.Findings[0].Message, c.message)
			}
		})
	}
}

func TestDependenciesEvidenceNamesTheSource(t *testing.T) {
	out := checkDir(t, filepath.Join(depFixtures, "js-yarn"))
	ev := out.Findings[0].Evidence
	for _, want := range []string{`"~17.2.0"`, "yarn.lock is present but not read"} {
		if !strings.Contains(ev, want) {
			t.Errorf("evidence %q lacks %q", ev, want)
		}
	}
	out = checkDir(t, filepath.Join(depFixtures, "js-locked-old"))
	if ev := out.Findings[0].Evidence; !strings.Contains(ev, "package-lock.json installs @stellar/stellar-sdk 16.3.1 at node_modules/@stellar/stellar-sdk") {
		t.Errorf("lockfile evidence = %q", ev)
	}
}

func TestDependenciesIgnoresUnrelatedManifests(t *testing.T) {
	out := checkDir(t, "../../testdata/fixtures/go-module")
	if len(out.Findings) != 0 || len(out.Warnings) != 0 {
		t.Errorf("unexpected output for a P30-ready module: %+v", out)
	}
}

func TestDependenciesUnreadableFileIsAWarning(t *testing.T) {
	out, err := Dependencies{}.Check(context.Background(), Input{
		Files:    []walker.File{{Path: "go.mod", Kind: "go-mod", Category: walker.CategoryManifest}},
		ReadFile: func(string) ([]byte, error) { return nil, os.ErrPermission },
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := warningCodes(out.Warnings); !reflect.DeepEqual(got, []string{"manifest-unparsed:go.mod"}) {
		t.Errorf("warnings = %v", got)
	}
}

func TestDependenciesHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Dependencies{}.Check(ctx, Input{
		Files:    []walker.File{{Path: "go.mod", Kind: "go-mod"}},
		ReadFile: func(string) ([]byte, error) { return nil, nil },
	})
	if err == nil {
		t.Error("cancelled context ignored")
	}
}
