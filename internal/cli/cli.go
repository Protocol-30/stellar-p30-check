// Package cli implements the p30check command line.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Protocol-30/stellar-p30-check/internal/report"
	"github.com/Protocol-30/stellar-p30-check/internal/rules"
	"github.com/Protocol-30/stellar-p30-check/internal/scan"
	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

// Exit codes.
const (
	ExitOK    = 0 // scan completed; no finding matched the --fail-on gate
	ExitGated = 1 // scan completed; at least one finding matched the gate
	ExitError = 2 // invalid usage or the scan could not complete
)

// Version is set by release builds with
// -ldflags "-X github.com/Protocol-30/stellar-p30-check/internal/cli.Version=0.1.0".
// When empty, ToolVersion falls back to the module version Go records for
// "go install ...@v0.1.0" and VCS builds, then to devVersion.
var Version = ""

const devVersion = "0.1.0-dev"

// ToolVersion returns the version printed by "p30check version" and
// written to reports.
func ToolVersion() string {
	if Version != "" {
		return Version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return strings.TrimPrefix(bi.Main.Version, "v")
	}
	return devVersion
}

// Env carries the process environment so tests can inject output streams,
// a fixed clock and rules.
type Env struct {
	Stdout io.Writer
	Stderr io.Writer
	Now    func() time.Time
	Rules  []rules.Rule // nil selects rules.Default()
}

// FailOn is the CI gate policy.
type FailOn string

const (
	FailOnNone FailOn = "none"
	FailOnHigh FailOn = "high"
)

// ParseFailOn validates a --fail-on value.
func ParseFailOn(s string) (FailOn, error) {
	switch FailOn(s) {
	case FailOnNone, FailOnHigh:
		return FailOn(s), nil
	}
	return "", fmt.Errorf("unsupported --fail-on value %q (want none or high)", s)
}

// Gated returns the findings that trip the gate. With FailOnHigh only
// findings that are both high severity and high confidence count, so
// suspected or weakly evidenced issues never fail a build.
func Gated(findings []report.Finding, policy FailOn) []report.Finding {
	if policy != FailOnHigh {
		return nil
	}
	var out []report.Finding
	for _, f := range findings {
		if f.Severity == report.SeverityHigh && f.Confidence == report.ConfidenceHigh {
			out = append(out, f)
		}
	}
	return out
}

// ExitCode maps a completed scan's findings to a process exit code.
func ExitCode(findings []report.Finding, policy FailOn) int {
	if len(Gated(findings, policy)) > 0 {
		return ExitGated
	}
	return ExitOK
}

const usage = `p30check - offline Stellar Protocol 30 compatibility scanner

Usage:
  p30check scan <path> [flags]   scan a local directory
  p30check version               print the version
  p30check help                  show this help

Scan flags:
  --format text|json        output format (default text)
  --fail-on none|high       exit 1 when a high-severity, high-confidence
                            finding is reported (default none)
  --max-file-bytes N        skip files larger than N bytes (default %d)
  --max-files N             fail if more than N relevant files are found (default %d)
  --follow-symlinks         include symlinked files whose target stays inside <path>
                            (symlinked directories are never followed)

Exit codes:
  0  scan completed and no finding matched --fail-on
  1  scan completed and at least one finding matched --fail-on
  2  invalid usage, or the scan could not complete

The scanner reads files only; it never executes scanned code or uses the network.
`

func printUsage(w io.Writer) {
	fmt.Fprintf(w, usage, walker.DefaultMaxFileBytes, walker.DefaultMaxFiles)
}

// Run executes the command line in args (without the program name) and
// returns the process exit code.
func Run(ctx context.Context, args []string, env Env) int {
	if len(args) == 0 {
		printUsage(env.Stderr)
		return ExitError
	}
	switch args[0] {
	case "help", "-h", "-help", "--help":
		printUsage(env.Stdout)
		return ExitOK
	case "version", "--version":
		fmt.Fprintf(env.Stdout, "p30check %s (report schema %s)\n", ToolVersion(), report.SchemaVersion)
		return ExitOK
	case "scan":
		return runScan(ctx, args[1:], env)
	}
	fmt.Fprintf(env.Stderr, "p30check: unknown command %q\n\n", args[0])
	printUsage(env.Stderr)
	return ExitError
}

func runScan(ctx context.Context, args []string, env Env) int {
	fset := flag.NewFlagSet("scan", flag.ContinueOnError)
	fset.SetOutput(io.Discard)
	format := fset.String("format", string(report.FormatText), "")
	failOn := fset.String("fail-on", string(FailOnNone), "")
	maxFileBytes := fset.Int64("max-file-bytes", walker.DefaultMaxFileBytes, "")
	maxFiles := fset.Int("max-files", walker.DefaultMaxFiles, "")
	follow := fset.Bool("follow-symlinks", false, "")

	paths, err := parseInterspersed(fset, args)
	if errors.Is(err, flag.ErrHelp) {
		printUsage(env.Stdout)
		return ExitOK
	}
	if err != nil {
		return usageError(env, err.Error())
	}
	if len(paths) != 1 {
		return usageError(env, fmt.Sprintf("scan takes exactly one path, got %d", len(paths)))
	}
	f, err := report.ParseFormat(*format)
	if err != nil {
		return usageError(env, err.Error())
	}
	policy, err := ParseFailOn(*failOn)
	if err != nil {
		return usageError(env, err.Error())
	}
	if *maxFileBytes <= 0 {
		return usageError(env, "--max-file-bytes must be positive")
	}
	if *maxFiles <= 0 {
		return usageError(env, "--max-files must be positive")
	}

	rs := env.Rules
	if rs == nil {
		rs = rules.Default()
	}
	rep, err := scan.Run(ctx, scan.Config{
		Target:      paths[0],
		ToolVersion: ToolVersion(),
		Walk: walker.Options{
			MaxFileBytes:   *maxFileBytes,
			MaxFiles:       *maxFiles,
			FollowSymlinks: *follow,
		},
		Rules: rs,
		Now:   env.Now,
	})
	if err != nil {
		fmt.Fprintf(env.Stderr, "p30check: scan failed: %v\n", err)
		if errors.Is(err, walker.ErrLimit) {
			fmt.Fprintln(env.Stderr, "p30check: no partial report is produced; narrow <path> or raise the limit")
		}
		return ExitError
	}
	if err := report.Render(env.Stdout, rep, f); err != nil {
		fmt.Fprintf(env.Stderr, "p30check: %v\n", err)
		return ExitError
	}
	return ExitCode(rep.Findings, policy)
}

// parseInterspersed lets flags appear before or after the path, as in
// "scan ./repo --format json". A literal "--" ends flag parsing.
func parseInterspersed(fset *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fset.Parse(args); err != nil {
			return nil, err
		}
		rest := fset.Args()
		if len(rest) == 0 {
			return positional, nil
		}
		// flag.Parse stops at the first non-flag or after "--". Distinguish
		// the two by checking whether "--" was consumed just before rest.
		consumed := len(args) - len(rest)
		if consumed > 0 && args[consumed-1] == "--" {
			return append(positional, rest...), nil
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}

func usageError(env Env, msg string) int {
	fmt.Fprintf(env.Stderr, "p30check: %s\n\nRun 'p30check help' for usage.\n", msg)
	return ExitError
}
