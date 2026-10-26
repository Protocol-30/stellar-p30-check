package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Protocol-30/stellar-p30-check/internal/report"
	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

var fixedNow = time.Date(2026, 10, 25, 17, 0, 0, 0, time.UTC)

func TestFixtureReports(t *testing.T) {
	for _, tc := range []struct {
		name     string
		manifest string
		kind     string
	}{
		{"m1-go", "go.mod", "go_module"},
		{"m1-js", "package.json", "js_manifest"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := filepath.Join("..", "..", "testdata", tc.name)
			var first []byte
			for i := 0; i < 2; i++ {
				var stdout, stderr bytes.Buffer
				code := Run([]string{"scan", target, "--format", "json", "--fail-on", "high"}, &stdout, &stderr, Options{Now: func() time.Time { return fixedNow }})
				if code != 0 || stderr.Len() != 0 {
					t.Fatalf("exit=%d stderr=%s stdout=%s", code, &stderr, &stdout)
				}
				var got report.ScanReport
				if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if got.SchemaVersion != report.SchemaVersion || got.CheckedAtUTC != fixedNow.Format(time.RFC3339Nano) || !got.Coverage.Complete || got.Coverage.RulesImplemented != 2 {
					t.Fatalf("incorrect report: %+v", got)
				}
				if got.Findings == nil || len(got.Findings) != 0 || len(got.Inventory) != 2 || len(got.Limitations) == 0 {
					t.Fatalf("incorrect inventory/findings/limitations: %+v", got)
				}
				found := false
				for _, f := range got.Inventory {
					if f.Path == tc.manifest && f.Kind == tc.kind {
						found = true
					}
				}
				if !found {
					t.Fatalf("missing manifest %s: %+v", tc.manifest, got.Inventory)
				}
				if i == 0 {
					first = bytes.Clone(stdout.Bytes())
				} else if !bytes.Equal(first, stdout.Bytes()) {
					t.Fatal("identical scan with fixed clock produced different JSON")
				}
			}
		})
	}
}

func TestScanErrorsProduceStructuredIncompleteReports(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file.go")
	if err := os.WriteFile(file, []byte("package example"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{filepath.Join(t.TempDir(), "missing"), file, "invalid\x00path"} {
		t.Run(strconvName(target), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := Run([]string{"scan", target, "--format=json"}, &stdout, &stderr, Options{}); code != 2 {
				t.Fatalf("exit=%d", code)
			}
			var got report.ScanReport
			if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
				t.Fatalf("invalid JSON: %v; %s", err, &stdout)
			}
			if got.Coverage.Complete || got.Error == "" || stderr.Len() == 0 {
				t.Fatalf("failure not reported: %+v stderr=%s", got, &stderr)
			}
		})
	}
}

func strconvName(path string) string { return strings.ReplaceAll(filepath.Base(path), "\x00", "NUL") }

func TestUsageErrorsDoNotScan(t *testing.T) {
	tests := [][]string{
		{}, {"unknown"}, {"scan"}, {"scan", ""}, {"scan", ".", "extra"},
		{"scan", ".", "--format", "csv"}, {"scan", ".", "--fail-on", "medium"},
		{"scan", ".", "--unknown"}, {"scan", ".", "--format"},
		{"scan", ".", "--max-files", "0"}, {"scan", ".", "--max-entries", "-1"},
		{"scan", ".", "--max-depth=0"}, {"scan", ".", "--max-file-bytes=0"},
		{"scan", ".", "--max-total-bytes=-1"}, {"scan", ".", "--max-files=abc"},
		{"version", "extra"}, {"help", "extra"}, {"scan", ".", "--help=false"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			opts := Options{Scan: func(string, walker.Limits) (walker.Result, error) {
				t.Fatal("unexpected scan on invalid args")
				return walker.Result{}, nil
			}}
			if code := Run(args, &stdout, &stderr, opts); code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "Usage:") {
				t.Fatalf("exit=%d stdout=%s stderr=%s", code, &stdout, &stderr)
			}
		})
	}
}

func TestHelpAndVersion(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"help"}, {"scan", "--help"}, {"scan", "missing", "-h"}, {"version"}, {"--version"}} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr, Options{Version: "test"}); code != 0 || stdout.Len() == 0 || stderr.Len() != 0 {
			t.Fatalf("%v exit=%d stdout=%s stderr=%s", args, code, &stdout, &stderr)
		}
	}
}

func TestInterspersedFlagsAndHyphenPath(t *testing.T) {
	var captured walker.Limits
	var target string
	opts := Options{Scan: func(path string, limits walker.Limits) (walker.Result, error) {
		target, captured = path, limits
		return walker.Result{Complete: true}, nil
	}}
	var stdout, stderr bytes.Buffer
	args := []string{"scan", "--max-files=3", "--format", "json", "--max-file-bytes", "7", "--max-total-bytes=9", "--max-entries=4", "--max-depth=2", "--", "-target"}
	if code := Run(args, &stdout, &stderr, opts); code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, &stderr)
	}
	if filepath.Base(target) != "-target" || captured.MaxFiles != 3 || captured.MaxFileBytes != 7 || captured.MaxTotalBytes != 9 || captured.MaxEntries != 4 || captured.MaxDepth != 2 {
		t.Fatalf("incorrect args: target=%s limits=%+v", target, captured)
	}
}

func TestPartialScanAndControlCharacters(t *testing.T) {
	opts := Options{Scan: func(string, walker.Limits) (walker.Result, error) {
		return walker.Result{Files: []walker.File{{Path: "prior.go", Kind: "go_source", Size: 1}}, EntriesVisited: 2, BytesRead: 1}, errors.New("read evil\x1b[2J\nforged: failure")
	}}
	for _, format := range []string{"text", "json"} {
		var stdout, stderr bytes.Buffer
		if code := Run([]string{"scan", ".", "--format", format}, &stdout, &stderr, opts); code != 2 {
			t.Fatalf("exit=%d", code)
		}
		if bytes.Contains(stdout.Bytes(), []byte{0x1b}) || bytes.Contains(stderr.Bytes(), []byte{0x1b}) || strings.Contains(stderr.String(), "\nforged:") {
			t.Fatal("unescaped terminal control or forged error")
		}
		if format == "json" {
			var r report.ScanReport
			if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
				t.Fatal(err)
			}
			if r.Coverage.Complete || len(r.Inventory) != 1 || r.Coverage.FilesScanned != 1 || r.Error == "" {
				t.Fatalf("partial result lost: %+v", r)
			}
		}
	}
}

func TestLimitFailure(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte("package sample"), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"scan", dir, "--format=json", "--max-file-bytes=1"}, &stdout, &stderr, Options{}); code != 2 {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, &stdout, &stderr)
	}
	var r report.ScanReport
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Coverage.Complete || r.Error == "" {
		t.Fatalf("limit failure hidden: %+v", r)
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestOutputFailures(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"version"}, {"scan", "--help"}, {"scan", "."}} {
		var stderr bytes.Buffer
		opts := Options{Scan: func(string, walker.Limits) (walker.Result, error) { return walker.Result{Complete: true}, nil }}
		if code := Run(args, brokenWriter{}, &stderr, opts); code != 2 || !strings.Contains(stderr.String(), "output failed") {
			t.Fatalf("%v exit=%d stderr=%s", args, code, &stderr)
		}
	}
}

func TestIncompleteWithoutErrorStillFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	opts := Options{Scan: func(string, walker.Limits) (walker.Result, error) { return walker.Result{}, nil }}
	if code := Run([]string{"scan", ".", "--format=json"}, &stdout, &stderr, opts); code != 2 {
		t.Fatalf("exit=%d", code)
	}
	if !strings.Contains(stdout.String(), "incomplete scan") {
		t.Fatal("missing explicit failure")
	}
}
