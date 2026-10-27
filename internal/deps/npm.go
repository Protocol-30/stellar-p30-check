package deps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// DependencyFields are the package.json sections read, in report order.
var DependencyFields = []string{"dependencies", "devDependencies", "optionalDependencies", "peerDependencies"}

// NPMDependency is one entry from a package.json dependency section.
type NPMDependency struct {
	Name  string // the real package name, after resolving npm: aliases
	Key   string // the key as written in package.json
	Spec  string // version spec, without any npm:<name>@ alias prefix
	Field string
	Line  int // 0 when the key could not be located
}

// ParsePackageJSON returns the dependencies declared in package.json text,
// sorted by field order then key.
func ParsePackageJSON(data []byte) ([]NPMDependency, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}
	var out []NPMDependency
	for _, field := range DependencyFields {
		raw, ok := doc[field]
		if !ok || string(raw) == "null" {
			continue
		}
		var deps map[string]string
		if err := json.Unmarshal(raw, &deps); err != nil {
			return nil, fmt.Errorf("parse package.json %s: %w", field, err)
		}
		keys := make([]string, 0, len(deps))
		for k := range deps {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			name, spec := k, deps[k]
			if alias, aliasSpec, ok := parseAlias(spec); ok {
				name, spec = alias, aliasSpec
			}
			out = append(out, NPMDependency{
				Name: name, Key: k, Spec: spec, Field: field,
				Line: keyLine(data, k),
			})
		}
	}
	return out, nil
}

// parseAlias splits "npm:@scope/name@1.2.3" into its package and spec.
func parseAlias(spec string) (name, version string, ok bool) {
	rest, found := strings.CutPrefix(spec, "npm:")
	if !found {
		return "", "", false
	}
	at := strings.LastIndexByte(rest, '@')
	if at <= 0 { // no version, or only a scope "@"
		return rest, "", rest != ""
	}
	return rest[:at], rest[at+1:], true
}

// LockedPackage is one installed package recorded in an npm lockfile.
type LockedPackage struct {
	Name     string
	Version  string
	Location string // "node_modules/..." path (v2/v3) or dependency chain (v1)
	Line     int
}

// packageEntry is a lockfileVersion 2/3 "packages" value.
type packageEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Link    bool   `json:"link"`
}

// legacyEntry is a lockfileVersion 1 "dependencies" value, which nests.
type legacyEntry struct {
	Version      string                 `json:"version"`
	Dependencies map[string]legacyEntry `json:"dependencies"`
}

// ParseNPMLock reads package-lock.json or npm-shrinkwrap.json text
// (lockfileVersion 1, 2 or 3) and returns every installed package, sorted
// by location.
func ParseNPMLock(data []byte) ([]LockedPackage, error) {
	var doc struct {
		Packages     map[string]packageEntry `json:"packages"`
		Dependencies map[string]legacyEntry  `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse npm lockfile: %w", err)
	}
	var out []LockedPackage
	if doc.Packages != nil {
		for loc, e := range doc.Packages {
			i := strings.LastIndex(loc, "node_modules/")
			if loc == "" || i < 0 || e.Link || e.Version == "" {
				continue // root project, workspace folder or link
			}
			name := loc[i+len("node_modules/"):]
			if e.Name != "" {
				name = e.Name
			}
			out = append(out, LockedPackage{Name: name, Version: e.Version, Location: loc, Line: keyLine(data, loc)})
		}
	} else {
		var walk func(prefix string, deps map[string]legacyEntry)
		walk = func(prefix string, deps map[string]legacyEntry) {
			for key, e := range deps {
				loc := prefix + key
				name, version := key, e.Version
				if alias, aliasVersion, ok := parseAlias(version); ok {
					name, version = alias, aliasVersion
				}
				if version != "" {
					out = append(out, LockedPackage{Name: name, Version: version, Location: loc, Line: keyLine(data, key)})
				}
				walk(loc+" > ", e.Dependencies)
			}
		}
		walk("", doc.Dependencies)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Location < out[j].Location })
	return out, nil
}

// keyLine returns the 1-based line of the first JSON object key equal to
// key, or 0. It is a text search, so for a key repeated at several nesting
// levels it reports the first occurrence.
func keyLine(data []byte, key string) int {
	quoted, err := json.Marshal(key)
	if err != nil {
		return 0
	}
	re, err := regexp.Compile(regexp.QuoteMeta(string(quoted)) + `\s*:`)
	if err != nil {
		return 0
	}
	loc := re.FindIndex(data)
	if loc == nil {
		return 0
	}
	return bytes.Count(data[:loc[0]], []byte("\n")) + 1
}

// SpecKind classifies a package.json version spec.
type SpecKind int

const (
	SpecUnresolved SpecKind = iota // tag, URL, git, file:, workspace:, complex range
	SpecExact                      // a single version: npm installs exactly this
	SpecRange                      // a range with a known lowest version
)

// ResolveSpec interprets the simple, common forms of an npm version spec:
// exact versions, ^ and ~ ranges, >= bounds and x-ranges such as 17.x.
// For a range it returns the lowest version the range admits. Anything else
// (unions with ||, hyphen ranges, upper bounds, tags, URLs) is unresolved.
func ResolveSpec(spec string) (Version, SpecKind) {
	s := strings.TrimSpace(spec)
	prefix := ""
	for _, p := range []string{">=", "^", "~", "="} {
		if strings.HasPrefix(s, p) {
			prefix, s = p, strings.TrimSpace(s[len(p):])
			break
		}
	}
	if s == "" || strings.ContainsAny(s, " |<>") {
		return Version{}, SpecUnresolved
	}
	if v, ok := ParseVersion(s); ok {
		if prefix == "" || prefix == "=" {
			return v, SpecExact
		}
		return v, SpecRange
	}
	// x-range or partial version: 17, 17.3, 17.x, 17.3.*
	parts := strings.Split(strings.TrimPrefix(s, "v"), ".")
	if len(parts) > 3 {
		return Version{}, SpecUnresolved
	}
	var nums [3]int
	for i, p := range parts {
		if p == "x" || p == "X" || p == "*" {
			for _, rest := range parts[i:] {
				if rest != "x" && rest != "X" && rest != "*" {
					return Version{}, SpecUnresolved
				}
			}
			if i == 0 {
				return Version{}, SpecUnresolved // "*" admits every version
			}
			break
		}
		n, ok := numeric(p)
		if !ok {
			return Version{}, SpecUnresolved
		}
		nums[i] = n
	}
	return Version{Major: nums[0], Minor: nums[1], Patch: nums[2]}, SpecRange
}
