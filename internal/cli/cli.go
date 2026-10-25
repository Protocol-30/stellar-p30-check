// Package cli implements the offline command interface. It never executes targets.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Protocol-30/stellar-p30-check/internal/deps"
	"github.com/Protocol-30/stellar-p30-check/internal/report"
	"github.com/Protocol-30/stellar-p30-check/internal/rules"
	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

const usage = `Usage:
  p30check scan <path> [--format text|json] [--fail-on none|high]
  p30check version
  p30check --help

M2 checks supported SDK dependency capabilities using static manifest/lock data.
Unresolved versions are warnings; address and raw-XDR source checks are not implemented.
An empty report does not establish compatibility.

Scan options (flags may precede or follow the path):
  --format text|json       Report format (default text)
  --fail-on none|high      Gate high severity AND high confidence (default none)
  --max-file-bytes N       Maximum bytes per relevant file (default 2097152)
  --max-total-bytes N      Maximum total bytes read (default 67108864)
  --max-files N            Maximum relevant files read (default 10000)
  --max-entries N          Maximum discovered entries (default 100000)
  --max-depth N            Maximum directory depth (default 64)
  -h, --help              Show help

Use -- to separate paths beginning with a hyphen.
Exit codes: 0 completed scan without gated findings; 1 gated findings;
2 invalid arguments, scan failure, or unexpected incomplete scan.
`

// Options supplies deterministic clocks and discovery for tests.
type Options struct {
	Version string
	Now     func() time.Time
	Scan    func(string, walker.Limits) (walker.Result, error)
}

type scanArgs struct {
	path   string
	format string
	failOn string
	limits walker.Limits
}

// Run returns the process exit code; it never terminates the caller.
func Run(args []string, stdout, stderr io.Writer, opts Options) int {
	if opts.Version == "" {
		opts.Version = "0.1.0-dev"
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Scan == nil {
		opts.Scan = walker.Scan
	}
	if len(args) == 0 {
		return usageError(stderr, errors.New("a command is required"))
	}
	switch args[0] {
	case "--help", "-h", "help":
		if len(args) != 1 {
			return usageError(stderr, errors.New("help does not take arguments"))
		}
		if _, err := io.WriteString(stdout, usage); err != nil {
			return outputError(stderr, err)
		}
		return 0
	case "version", "--version":
		if len(args) != 1 {
			return usageError(stderr, errors.New("version does not take arguments"))
		}
		if _, err := fmt.Fprintf(stdout, "p30check %s\n", strconv.QuoteToASCII(opts.Version)); err != nil {
			return outputError(stderr, err)
		}
		return 0
	case "scan":
		cfg, err := parseScan(args[1:])
		if errors.Is(err, flag.ErrHelp) {
			if _, err := io.WriteString(stdout, usage); err != nil {
				return outputError(stderr, err)
			}
			return 0
		}
		if err != nil {
			return usageError(stderr, err)
		}
		return scan(cfg, stdout, stderr, opts)
	default:
		return usageError(stderr, fmt.Errorf("unknown command %q", args[0]))
	}
}

func parseScan(args []string) (scanArgs, error) {
	cfg := scanArgs{limits: walker.DefaultLimits()}
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.format, "format", "text", "report format")
	fs.StringVar(&cfg.failOn, "fail-on", "none", "exit gate")
	fs.Int64Var(&cfg.limits.MaxFileBytes, "max-file-bytes", cfg.limits.MaxFileBytes, "bytes per file")
	fs.Int64Var(&cfg.limits.MaxTotalBytes, "max-total-bytes", cfg.limits.MaxTotalBytes, "total bytes")
	fs.IntVar(&cfg.limits.MaxFiles, "max-files", cfg.limits.MaxFiles, "files")
	fs.IntVar(&cfg.limits.MaxEntries, "max-entries", cfg.limits.MaxEntries, "entries")
	fs.IntVar(&cfg.limits.MaxDepth, "max-depth", cfg.limits.MaxDepth, "directory depth")

	// flag.FlagSet normally stops at the first positional argument. Rearrange
	// known options so the documented `scan path --format json` also works.
	var flags, paths []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			paths = append(paths, args[i+1:]...)
			break
		}
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			paths = append(paths, arg)
			continue
		}
		name, _, hasValue := strings.Cut(strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-"), "=")
		if name == "h" || name == "help" {
			if hasValue {
				return cfg, errors.New("help does not take a value")
			}
			return cfg, flag.ErrHelp
		}
		if fs.Lookup(name) == nil {
			return cfg, fmt.Errorf("unknown flag %q", arg)
		}
		flags = append(flags, arg)
		if !hasValue {
			if i+1 == len(args) {
				return cfg, fmt.Errorf("flag %q needs a value", arg)
			}
			i++
			flags = append(flags, args[i])
		}
	}
	flags = append(flags, "--")
	flags = append(flags, paths...)
	if err := fs.Parse(flags); err != nil {
		return cfg, err
	}
	if fs.NArg() != 1 || fs.Arg(0) == "" {
		return cfg, errors.New("scan requires exactly one nonempty directory path")
	}
	cfg.path = fs.Arg(0)
	if cfg.format != "text" && cfg.format != "json" {
		return cfg, fmt.Errorf("unsupported format %q; use text or json", cfg.format)
	}
	if cfg.failOn != "none" && cfg.failOn != "high" {
		return cfg, fmt.Errorf("unsupported fail-on policy %q; use none or high", cfg.failOn)
	}
	if cfg.limits.MaxFileBytes <= 0 || cfg.limits.MaxTotalBytes <= 0 || cfg.limits.MaxFiles <= 0 || cfg.limits.MaxEntries <= 0 || cfg.limits.MaxDepth <= 0 {
		return cfg, errors.New("all scan limits must be positive")
	}
	return cfg, nil
}

func scan(cfg scanArgs, stdout, stderr io.Writer, opts Options) int {
	target, err := filepath.Abs(cfg.path)
	if err != nil {
		return usageError(stderr, fmt.Errorf("resolve target directory: %w", err))
	}
	r := report.New(target, opts.Version, opts.Now())
	r.Limitations = []string{
		"M2 implements P30-DEP-001 SDK dependency capability checks only; address logic, raw-XDR usage, and timestamp assumptions are not analyzed.",
		"Capability evidence covers exact verified releases only. Manifest constraints, unavailable graphs, replacements, unsupported locks, and uncertain registry provenance do not prove an effective SDK version.",
		"Effective dependency records mean supported nominal lock/declaration agreement, not installed-artifact verification. npm registry URLs can use a configured registry; credential-bearing registry configuration is not read.",
		"Missing SDK capabilities are review guidance, not proven application breakage; M2 cannot establish affected runtime roles and emits no high-severity findings.",
		"An empty findings list is not evidence of Protocol 30 compatibility.",
		"Completeness describes only the selected filesystem scope; ignored, generated, symlink, and nonregular entries are excluded. Concurrent file changes are not a stable snapshot.",
	}
	discovered, scanErr := opts.Scan(target, cfg.limits)
	r.Coverage = report.Coverage{
		Complete: discovered.Complete, RulesImplemented: 1,
		FilesScanned: len(discovered.Files), EntriesVisited: discovered.EntriesVisited,
		BytesRead: discovered.BytesRead,
	}
	for _, f := range discovered.Files {
		r.Inventory = append(r.Inventory, report.InventoryItem{Path: f.Path, Kind: f.Kind, Size: f.Size})
	}
	for _, skipped := range discovered.Skipped {
		r.Coverage.Skipped = append(r.Coverage.Skipped, report.Skipped{Path: skipped.Path, Reason: skipped.Reason})
	}
	inventory, dependencyErr := deps.Inspect(discovered.Files)
	r.Dependencies = inventory.Dependencies
	r.DependencyWarnings = inventory.Warnings
	if discovered.Complete && scanErr == nil && dependencyErr == nil {
		var warnings []deps.Warning
		r.Dependencies, r.Findings, warnings = rules.Evaluate(r.Dependencies)
		r.DependencyWarnings = append(r.DependencyWarnings, warnings...)
	} else {
		// A partial tree may omit an override or conflicting lockfile. Never
		// promote observations from that tree to resolved capability findings.
		for i := range r.Dependencies {
			r.Dependencies[i].Effective = false
			r.Dependencies[i].Uncertainty = append(r.Dependencies[i].Uncertainty, "discovery or dependency parsing is incomplete")
		}
		r.Limitations = append(r.Limitations, "SDK capability evaluation was withheld because discovery or dependency parsing is incomplete.")
	}
	scanErr = errors.Join(scanErr, dependencyErr)
	if scanErr != nil {
		r.Error = scanErr.Error()
		r.Coverage.Complete = false
	} else if !r.Coverage.Complete {
		r.Error = "discovery returned an incomplete scan"
	}
	if err := report.Render(stdout, cfg.format, r); err != nil {
		return outputError(stderr, err)
	}
	code, err := report.ExitCode(r, cfg.failOn)
	if err != nil {
		return outputError(stderr, err)
	}
	if r.Error != "" {
		fmt.Fprintf(stderr, "p30check: %s\n", strconv.QuoteToASCII(r.Error))
	}
	return code
}

func usageError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "p30check: %s\n\n%s", strconv.QuoteToASCII(err.Error()), usage)
	return 2
}

func outputError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "p30check: output failed: %s\n", strconv.QuoteToASCII(err.Error()))
	return 2
}
