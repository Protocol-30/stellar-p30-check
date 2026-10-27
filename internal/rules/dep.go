package rules

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/Protocol-30/stellar-p30-check/internal/deps"
	"github.com/Protocol-30/stellar-p30-check/internal/report"
	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

// DependenciesID is the rule ID of the SDK version check.
const DependenciesID = "P30-DEP-001"

// Dependencies (P30-DEP-001) compares the Stellar SDK versions a project
// declares or installs against the sourced capability matrix in
// internal/deps.
//
// Confidence follows how the version was learned: an exact version from
// go.mod, an npm lockfile or an exact package.json pin is high; the lowest
// version a package.json range admits is low, because the installed
// version depends on a lockfile the scanner did not read.
type Dependencies struct{}

// ID implements Rule.
func (Dependencies) ID() string { return DependenciesID }

// Check implements Rule.
func (Dependencies) Check(ctx context.Context, in Input) (Output, error) {
	var out Output
	warnUnparsed := func(p string, err error) {
		out.Warnings = append(out.Warnings, report.Warning{
			Code: "manifest-unparsed", Path: p,
			Message: fmt.Sprintf("%s was not checked by %s: %v", path.Base(p), DependenciesID, err),
		})
	}

	// Lockfiles first: they record what is actually installed, and they
	// decide which package.json entries still need range evaluation.
	locked := map[string]map[string]bool{} // lockfile dir -> tracked package names
	unparsedLocks := map[string]string{}   // dir -> yarn.lock / pnpm-lock.yaml
	for _, f := range in.Files {
		switch f.Kind {
		case "npm-lockfile":
			data, err := in.ReadFile(f.Path)
			if err != nil {
				warnUnparsed(f.Path, err)
				continue
			}
			pkgs, err := deps.ParseNPMLock(data)
			if err != nil {
				warnUnparsed(f.Path, err)
				continue
			}
			dir := path.Dir(f.Path)
			for _, p := range pkgs {
				if !deps.Tracked(deps.NPM, p.Name) {
					continue
				}
				if locked[dir] == nil {
					locked[dir] = map[string]bool{}
				}
				locked[dir][p.Name] = true
				evidence := fmt.Sprintf("%s installs %s %s at %s", path.Base(f.Path), p.Name, p.Version, p.Location)
				out.Findings = append(out.Findings, exactFinding(deps.NPM, p.Name, p.Version, f.Path, p.Line, evidence)...)
			}
		case "yarn-lockfile", "pnpm-lockfile":
			unparsedLocks[path.Dir(f.Path)] = path.Base(f.Path)
			out.Warnings = append(out.Warnings, report.Warning{
				Code: "lockfile-not-parsed", Path: f.Path,
				Message: fmt.Sprintf("%s does not read %s yet; Stellar SDK versions are judged from package.json ranges instead", DependenciesID, path.Base(f.Path)),
			})
		}
	}

	for _, f := range in.Files {
		if err := ctx.Err(); err != nil {
			return Output{}, err
		}
		switch f.Kind {
		case "go-mod":
			data, err := in.ReadFile(f.Path)
			if err != nil {
				warnUnparsed(f.Path, err)
				continue
			}
			mod, err := deps.ParseGoMod(data)
			if err != nil {
				warnUnparsed(f.Path, err)
				continue
			}
			out.Findings = append(out.Findings, goModFindings(f, mod)...)
		case "package-json":
			data, err := in.ReadFile(f.Path)
			if err != nil {
				warnUnparsed(f.Path, err)
				continue
			}
			declared, err := deps.ParsePackageJSON(data)
			if err != nil {
				warnUnparsed(f.Path, err)
				continue
			}
			for _, d := range declared {
				if !deps.Tracked(deps.NPM, d.Name) || coveredByLock(locked, path.Dir(f.Path), d.Name) {
					continue
				}
				out.Findings = append(out.Findings, specFindings(f.Path, d, unparsedLocks[path.Dir(f.Path)])...)
			}
		}
	}
	return out, nil
}

// coveredByLock reports whether an npm lockfile in dir or an ancestor
// records the package (npm workspaces keep one lockfile at the root).
func coveredByLock(locked map[string]map[string]bool, dir, name string) bool {
	for {
		if locked[dir][name] {
			return true
		}
		if dir == "." || dir == "/" || dir == "" {
			return false
		}
		dir = path.Dir(dir)
	}
}

func goModFindings(f walker.File, mod deps.GoMod) []report.Finding {
	var out []report.Finding
	for _, req := range mod.Requires {
		if !deps.Tracked(deps.Go, req.Path) {
			continue
		}
		evidence := fmt.Sprintf("go.mod requires %s %s", req.Path, req.Version)
		name, version := req.Path, req.Version
		if r, ok := mod.ReplacementFor(req.Path, req.Version); ok {
			if r.NewVersion == "" || !deps.Tracked(deps.Go, r.New) {
				out = append(out, unresolvedFinding(req.Path, f.Path, req.Line,
					fmt.Sprintf("%s, replaced by %s on line %d", evidence, strings.TrimSpace(r.New+" "+r.NewVersion), r.Line),
					"it is replaced by a fork or local directory, so the built code is not an official release"))
				continue
			}
			name, version = r.New, r.NewVersion
			evidence = fmt.Sprintf("%s, replaced by %s %s on line %d", evidence, r.New, r.NewVersion, r.Line)
		}
		out = append(out, exactFinding(deps.Go, name, version, f.Path, req.Line, evidence)...)
	}
	return out
}

func specFindings(file string, d deps.NPMDependency, unparsedLock string) []report.Finding {
	evidence := fmt.Sprintf("package.json %s declares %q: %q", d.Field, d.Key, d.Spec)
	if d.Key != d.Name {
		evidence += " (alias for " + d.Name + ")"
	}
	v, kind := deps.ResolveSpec(d.Spec)
	switch kind {
	case deps.SpecExact:
		return exactFinding(deps.NPM, d.Name, v.String(), file, d.Line, evidence)
	case deps.SpecRange:
		a, _ := deps.Assess(deps.NPM, d.Name, v)
		note := "no npm lockfile records it"
		if unparsedLock != "" {
			note = unparsedLock + " is present but not read by this version"
		}
		subject := fmt.Sprintf("the range %q for %s allows %s", d.Spec, d.Name, v)
		evidence += "; the installed version is unknown because " + note
		f, ok := fromAssessment(a, subject, report.ConfidenceLow, file, d.Line, evidence)
		if !ok {
			return nil
		}
		return []report.Finding{f}
	}
	return []report.Finding{unresolvedFinding(d.Name, file, d.Line, evidence,
		"the version spec is not a registry version this rule can interpret (tag, URL, git, file or complex range)")}
}

// exactFinding assesses a single known version.
func exactFinding(eco deps.Ecosystem, name, version, file string, line int, evidence string) []report.Finding {
	v, ok := deps.ParseVersion(version)
	if !ok {
		return []report.Finding{unresolvedFinding(name, file, line, evidence, fmt.Sprintf("%q is not a semantic version", version))}
	}
	a, _ := deps.Assess(eco, name, v)
	f, ok := fromAssessment(a, name+" "+version, report.ConfidenceHigh, file, line, evidence)
	if !ok {
		return nil
	}
	return []report.Finding{f}
}

// fromAssessment turns a matrix verdict into a finding. ok is false when
// the version supports both Protocol 30 capabilities.
func fromAssessment(a deps.Assessment, subject string, conf report.Confidence, file string, line int, evidence string) (report.Finding, bool) {
	f := report.Finding{
		RuleID: DependenciesID, Confidence: conf, Path: file, Line: line,
		Evidence: evidence, Remediation: a.Remediation, References: a.References,
	}
	switch {
	case a.LedgerXDR == deps.Yes && a.MuxedContract == deps.Yes:
		return report.Finding{}, false
	case a.LedgerXDR == deps.No:
		f.Severity = report.SeverityHigh
		f.Message = fmt.Sprintf("%s: no Protocol 30 support; %s", subject, a.Basis)
	case a.LedgerXDR == deps.Yes && a.MuxedContract == deps.No:
		f.Severity = report.SeverityMedium
		f.Message = fmt.Sprintf("%s: decodes Protocol 30 ledgers but cannot handle W muxed-contract addresses (CAP-84)", subject)
	default:
		f.Severity = report.SeverityInfo
		f.Confidence = report.ConfidenceLow
		f.Message = fmt.Sprintf("%s: Protocol 30 support could not be verified; %s", subject, a.Basis)
	}
	return f, true
}

func unresolvedFinding(name, file string, line int, evidence, why string) report.Finding {
	return report.Finding{
		RuleID: DependenciesID, Severity: report.SeverityInfo, Confidence: report.ConfidenceLow,
		Path: file, Line: line, Evidence: evidence,
		Message:     fmt.Sprintf("%s: Protocol 30 support could not be checked; %s", name, why),
		Remediation: "Confirm the resolved version supports Protocol 30: @stellar/stellar-sdk 17.3.0+ or github.com/stellar/go-stellar-sdk v0.8.0+.",
		References:  []string{deps.RefUpgrade, deps.RefCAP88, deps.RefCAP84},
	}
}
