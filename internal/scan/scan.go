// Package scan runs discovery and rules over a directory and assembles the
// report: scan -> discover -> rules -> findings -> report.
package scan

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Protocol-30/stellar-p30-check/internal/report"
	"github.com/Protocol-30/stellar-p30-check/internal/rules"
	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

// Config describes one scan.
type Config struct {
	Target      string // path as given by the user; echoed in the report
	ToolVersion string
	Walk        walker.Options
	Rules       []rules.Rule
	Now         func() time.Time // injectable for deterministic reports
}

// Limitations are printed with every report. They describe what a scan of
// this build can and cannot tell the user.
var Limitations = []string{
	"Static, offline analysis only: scanned code is never executed, and no network or RPC calls are made.",
	"A finding is an indication to review, not proof of runtime breakage; no findings is not proof of Protocol 30 compatibility.",
	"Stellar's November 3, 2026 (Testnet) and November 23, 2026 (Mainnet) dates are upgrade votes, not guaranteed activation dates.",
	"Ignored directories, symlinks that are not followed, and files over the size limit are not inspected; see the skipped list.",
}

// Run performs the scan. Errors mean the scan could not complete and no
// report should be trusted; findings are never errors.
func Run(ctx context.Context, cfg Config) (report.ScanReport, error) {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}

	res, err := walker.Walk(cfg.Target, cfg.Walk)
	if err != nil {
		return report.ScanReport{}, err
	}

	maxBytes := cfg.Walk.MaxFileBytes
	if maxBytes <= 0 {
		maxBytes = walker.DefaultMaxFileBytes
	}
	in := rules.Input{
		Files:    res.Files,
		ReadFile: boundedReader(res.Root, maxBytes),
	}

	var findings []report.Finding
	ws := coverageWarnings(cfg.Rules, res.Files)
	seenWarning := map[report.Warning]bool{} // rules may report the same file problem
	for _, r := range cfg.Rules {
		if err := ctx.Err(); err != nil {
			return report.ScanReport{}, fmt.Errorf("scan cancelled: %w", err)
		}
		got, err := r.Check(ctx, in)
		if err != nil {
			return report.ScanReport{}, fmt.Errorf("rule %s: %w", r.ID(), err)
		}
		for _, f := range got.Findings {
			if err := f.Validate(); err != nil {
				return report.ScanReport{}, fmt.Errorf("rule %s produced an invalid finding: %w", r.ID(), err)
			}
		}
		findings = append(findings, got.Findings...)
		for _, w := range got.Warnings {
			if !seenWarning[w] {
				seenWarning[w] = true
				ws = append(ws, w)
			}
		}
	}

	inventory := make([]report.InventoryItem, 0, len(res.Files))
	for _, f := range res.Files {
		inventory = append(inventory, report.InventoryItem{Path: f.Path, Kind: f.Kind, Category: string(f.Category)})
	}
	skipped := make([]report.SkippedEntry, 0, len(res.Skipped))
	for _, s := range res.Skipped {
		skipped = append(skipped, report.SkippedEntry{Path: s.Path, Reason: s.Reason})
	}

	return report.New(cfg.ToolVersion, cfg.Target, now(), len(cfg.Rules),
		findings, inventory, skipped, ws, Limitations), nil
}

// coverageWarnings explains gaps in coverage so an empty findings list is
// never read as an all-clear.
func coverageWarnings(rs []rules.Rule, files []walker.File) []report.Warning {
	var ws []report.Warning
	if len(rs) == 0 {
		ws = append(ws, report.Warning{
			Code:    "no-rules",
			Message: "this build ships no Protocol 30 compatibility rules; the report is an inventory only and makes no compatibility claim",
		})
	} else {
		var missing []string
		for _, id := range rules.Planned {
			if !hasRule(rs, id) {
				missing = append(missing, id)
			}
		}
		if len(missing) > 0 {
			ws = append(ws, report.Warning{
				Code:    "rules-not-implemented",
				Message: "not checked by this build: " + strings.Join(missing, ", ") + "; see docs/roadmap.md",
			})
		}
	}
	if len(files) == 0 {
		ws = append(ws, report.Warning{
			Code:    "no-relevant-files",
			Message: "no Go or JavaScript/TypeScript manifests or sources were found under the target",
		})
	}
	if !hasRule(rs, "P30-DEP-001") {
		for _, f := range files {
			if f.Category == walker.CategoryManifest {
				ws = append(ws, report.Warning{
					Code:    "check-not-implemented",
					Path:    f.Path,
					Message: "dependency capability check P30-DEP-001 is not implemented yet; SDK versions in this file were not evaluated",
				})
			}
		}
	}
	return ws
}

func hasRule(rs []rules.Rule, id string) bool {
	for _, r := range rs {
		if r.ID() == id {
			return true
		}
	}
	return false
}

// boundedReader returns a reader for root-relative slash paths that refuses
// paths escaping the root, symlinks anywhere in the path, and files larger
// than maxBytes.
func boundedReader(root string, maxBytes int64) func(string) ([]byte, error) {
	return func(rel string) ([]byte, error) {
		if !fs.ValidPath(rel) || rel == "." {
			return nil, fmt.Errorf("read %q: path is not a clean root-relative path", rel)
		}
		abs := filepath.Join(root, filepath.FromSlash(rel))
		resolved, err := filepath.EvalSymlinks(abs)
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", rel, err)
		}
		if resolved != abs {
			return nil, fmt.Errorf("read %q: path traverses a symlink", rel)
		}
		f, err := os.Open(abs)
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", rel, err)
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", rel, err)
		}
		if int64(len(data)) > maxBytes {
			return nil, fmt.Errorf("read %q: %w", rel, ErrTooLarge)
		}
		return data, nil
	}
}

// ErrTooLarge is returned by Input.ReadFile for files over the size limit.
var ErrTooLarge = errors.New("file exceeds size limit")
