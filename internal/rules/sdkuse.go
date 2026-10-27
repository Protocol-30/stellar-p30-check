package rules

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/Protocol-30/stellar-p30-check/internal/deps"
	"github.com/Protocol-30/stellar-p30-check/internal/report"
)

// useKind says how a Stellar SDK version was learned.
type useKind int

const (
	useExact      useKind = iota // go.mod require, npm lockfile entry, exact package.json pin
	useRange                     // package.json range; Parsed is the lowest version it allows
	useUnresolved                // fork, local replace, tag, URL or unparsable version
)

// sdkUse is one Stellar SDK package version declared or installed by a
// manifest. It is shared by P30-DEP-001, which reports outdated versions,
// and P30-XDR-001, which ties them to code that decodes ledgers.
type sdkUse struct {
	Eco      deps.Ecosystem
	Name     string
	Kind     useKind
	Parsed   deps.Version // valid for useExact and useRange
	Subject  string       // "name version" or `the range "^16" for name allows 16.0.0`
	Why      string       // for useUnresolved
	File     string
	Line     int
	Dir      string // project directory the use applies to; "" for nested dependency copies
	Evidence string
}

// collectSDKUses reads go.mod, npm lockfiles and package.json files once.
// Problems with individual files become warnings attributed to ruleID.
func collectSDKUses(ctx context.Context, in Input, ruleID string) ([]sdkUse, []report.Warning, error) {
	var uses []sdkUse
	var ws []report.Warning
	warnUnparsed := func(p string, err error) {
		ws = append(ws, report.Warning{
			Code: "manifest-unparsed", Path: p,
			Message: fmt.Sprintf("%s was not checked by %s: %v", path.Base(p), ruleID, err),
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
				uses = append(uses, exactUse(deps.NPM, p.Name, p.Version, f.Path, p.Line, lockUseDir(dir, p.Location), evidence))
			}
		case "yarn-lockfile", "pnpm-lockfile":
			unparsedLocks[path.Dir(f.Path)] = path.Base(f.Path)
			ws = append(ws, report.Warning{
				Code: "lockfile-not-parsed", Path: f.Path,
				Message: fmt.Sprintf("%s does not read %s yet; Stellar SDK versions are judged from package.json ranges instead", ruleID, path.Base(f.Path)),
			})
		}
	}

	for _, f := range in.Files {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
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
			uses = append(uses, goModUses(f.Path, mod)...)
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
			dir := path.Dir(f.Path)
			for _, d := range declared {
				if !deps.Tracked(deps.NPM, d.Name) || coveredByLock(locked, dir, d.Name) {
					continue
				}
				uses = append(uses, specUse(f.Path, d, unparsedLocks[dir]))
			}
		}
	}
	return uses, ws, nil
}

// lockUseDir maps a lockfile location to the project directory whose code
// resolves that copy: "packages/web/node_modules/x" belongs to
// packages/web, "node_modules/x" to the lockfile's directory. Copies nested
// inside other dependencies belong to no project directory.
func lockUseDir(lockDir, location string) string {
	if strings.Contains(location, " > ") {
		return "" // lockfileVersion 1 nested dependency
	}
	i := strings.LastIndex(location, "node_modules/")
	if i < 0 {
		return lockDir
	}
	prefix := strings.TrimSuffix(location[:i], "/")
	if strings.Contains(prefix, "node_modules") {
		return ""
	}
	if prefix == "" {
		return lockDir
	}
	return path.Join(lockDir, prefix)
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

func goModUses(file string, mod deps.GoMod) []sdkUse {
	var out []sdkUse
	dir := path.Dir(file)
	for _, req := range mod.Requires {
		if !deps.Tracked(deps.Go, req.Path) {
			continue
		}
		evidence := fmt.Sprintf("go.mod requires %s %s", req.Path, req.Version)
		name, version := req.Path, req.Version
		if r, ok := mod.ReplacementFor(req.Path, req.Version); ok {
			if r.NewVersion == "" || !deps.Tracked(deps.Go, r.New) {
				out = append(out, sdkUse{
					Eco: deps.Go, Name: req.Path, Kind: useUnresolved, File: file, Line: req.Line, Dir: dir,
					Evidence: fmt.Sprintf("%s, replaced by %s on line %d", evidence, strings.TrimSpace(r.New+" "+r.NewVersion), r.Line),
					Why:      "it is replaced by a fork or local directory, so the built code is not an official release",
				})
				continue
			}
			name, version = r.New, r.NewVersion
			evidence = fmt.Sprintf("%s, replaced by %s %s on line %d", evidence, r.New, r.NewVersion, r.Line)
		}
		out = append(out, exactUse(deps.Go, name, version, file, req.Line, dir, evidence))
	}
	return out
}

func specUse(file string, d deps.NPMDependency, unparsedLock string) sdkUse {
	u := sdkUse{Eco: deps.NPM, Name: d.Name, File: file, Line: d.Line, Dir: path.Dir(file)}
	u.Evidence = fmt.Sprintf("package.json %s declares %q: %q", d.Field, d.Key, d.Spec)
	if d.Key != d.Name {
		u.Evidence += " (alias for " + d.Name + ")"
	}
	v, kind := deps.ResolveSpec(d.Spec)
	switch kind {
	case deps.SpecExact:
		return exactUse(deps.NPM, d.Name, v.String(), file, d.Line, u.Dir, u.Evidence)
	case deps.SpecRange:
		note := "no npm lockfile records it"
		if unparsedLock != "" {
			note = unparsedLock + " is present but not read by this version"
		}
		u.Kind, u.Parsed = useRange, v
		u.Subject = fmt.Sprintf("the range %q for %s allows %s", d.Spec, d.Name, v)
		u.Evidence += "; the installed version is unknown because " + note
		return u
	}
	u.Kind = useUnresolved
	u.Why = "the version spec is not a registry version this rule can interpret (tag, URL, git, file or complex range)"
	return u
}

func exactUse(eco deps.Ecosystem, name, version, file string, line int, dir, evidence string) sdkUse {
	u := sdkUse{Eco: eco, Name: name, File: file, Line: line, Dir: dir, Evidence: evidence}
	v, ok := deps.ParseVersion(version)
	if !ok {
		u.Kind = useUnresolved
		u.Why = fmt.Sprintf("%q is not a semantic version", version)
		return u
	}
	u.Kind, u.Parsed, u.Subject = useExact, v, name+" "+version
	return u
}

// nearestUses returns the SDK uses of one ecosystem that apply to code in
// dir: those of the closest enclosing directory that has any.
func nearestUses(uses []sdkUse, eco deps.Ecosystem, dir string) []sdkUse {
	for {
		var found []sdkUse
		for _, u := range uses {
			if u.Eco == eco && u.Dir == dir {
				found = append(found, u)
			}
		}
		if len(found) > 0 {
			return found
		}
		if dir == "." || dir == "/" || dir == "" {
			return nil
		}
		dir = path.Dir(dir)
	}
}
