package deps

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/Protocol-30/stellar-p30-check/internal/walker"
	"go.yaml.in/yaml/v3"
	"golang.org/x/mod/semver"
)

// jsLocked is an observed direct lock selection. Origin and declaration checks
// are applied separately; a version string alone does not establish identity.
type jsLocked struct {
	Version   string
	Resolved  string
	Specifier string
	Reasons   []string
}

type jsLock struct {
	Path        string
	Family      string
	Generation  int
	Unsupported string
	Reasons     []string
	Resolve     func(scope, name, declared string) (jsLocked, bool)
}

type jsManifest struct {
	file       walker.File
	groups     map[string]map[string]string
	overrides  bool
	workspaces []string
	manager    string
}

type jsWorkspace struct {
	file     walker.File
	patterns []string
	modifies bool
}

var jsGroups = []string{"dependencies", "devDependencies", "optionalDependencies", "peerDependencies"}

func inspectJS(files []walker.File) ([]Dependency, []Warning, error) {
	var manifests []jsManifest
	var locks []*jsLock
	var workspaces []jsWorkspace
	var warnings []Warning
	var errs []error
	ordered := append([]walker.File(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	for _, file := range ordered {
		name := path.Base(file.Path)
		if name == "pnpm-workspace.yaml" {
			workspace, err := parseJSWorkspace(file)
			if err != nil {
				errs = append(errs, fmt.Errorf("parse pnpm workspace %q: %w", file.Path, err))
				warnings = append(warnings, Warning{Path: file.Path, Code: "malformed_manifest", Message: "The pnpm workspace configuration could not be parsed; current workspace membership is unavailable."})
			} else {
				workspaces = append(workspaces, workspace)
			}
			continue
		}
		if name == "package.json" {
			m, err := parseJSManifest(file)
			if err != nil {
				errs = append(errs, fmt.Errorf("parse JS manifest %q: %w", file.Path, err))
				warnings = append(warnings, Warning{Path: file.Path, Code: "malformed_manifest", Message: "The package manifest could not be parsed; dependency inventory is incomplete."})
				continue
			}
			manifests = append(manifests, m)
			continue
		}
		var lock *jsLock
		var err error
		switch name {
		case "package-lock.json", "npm-shrinkwrap.json":
			lock, err = parseNPMLock(file)
		case "yarn.lock":
			lock, err = parseYarnLock(file)
		case "pnpm-lock.yaml":
			lock, err = parsePNPMLock(file)
		default:
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("parse JS lockfile %q: %w", file.Path, err))
			warnings = append(warnings, Warning{Path: file.Path, Code: "malformed_lockfile", Message: "The lockfile could not be parsed; dependency selection is unavailable."})
			// A failed lock must still prevent fallback to another family.
			locks = append(locks, &jsLock{Path: file.Path, Unsupported: "malformed lockfile"})
			continue
		}
		locks = append(locks, lock)
		if lock.Unsupported != "" {
			warnings = append(warnings, Warning{Path: file.Path, Code: "unsupported_lockfile", Message: lock.Unsupported})
		}
	}
	var dependencies []Dependency
	for _, manifest := range manifests {
		scope := path.Dir(manifest.file.Path)
		var applicable []*jsLock
		for _, lock := range locks {
			if jsContains(path.Dir(lock.Path), scope) {
				applicable = append(applicable, lock)
			}
		}
		for _, group := range jsGroups {
			names := make([]string, 0, len(manifest.groups[group]))
			for name := range manifest.groups[group] {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, declaredName := range names {
				declared := manifest.groups[group][declaredName]
				name, alias, aliasVersion := jsAlias(declaredName, declared)
				d := Dependency{Ecosystem: "npm", Name: name, Alias: alias, Group: group, Scope: scope, Path: manifest.file.Path, Declared: redactJSDeclaration(declared), Source: manifest.file.Path, Resolution: "declared", Uncertainty: []string{}}
				if version, ok := jsExactVersion(declared); ok {
					d.Version = version
				} else if version, ok := jsExactVersion(aliasVersion); ok {
					d.Version = version
				}
				if alias != "" {
					d.Uncertainty = append(d.Uncertainty, "npm aliases are inventoried but not accepted as proven package identity")
				}
				for _, otherGroup := range jsGroups {
					if other, ok := manifest.groups[otherGroup][declaredName]; ok && other != declared {
						d.Uncertainty = append(d.Uncertainty, "the dependency has conflicting declarations in multiple manifest groups")
						break
					}
				}
				for _, ancestor := range manifests {
					if ancestor.overrides && jsContains(path.Dir(ancestor.file.Path), scope) {
						d.Uncertainty = append(d.Uncertainty, "manifest overrides or resolutions may change the selected dependency")
						break
					}
				}
				for _, workspace := range workspaces {
					if workspace.modifies && jsContains(path.Dir(workspace.file.Path), scope) {
						d.Uncertainty = append(d.Uncertainty, "pnpm workspace overrides, resolutions or patches may change the selected dependency")
					}
				}
				switch len(applicable) {
				case 0:
					d.Uncertainty = append(d.Uncertainty, "no applicable lockfile is available; declarations do not prove selected versions")
				case 1:
					lock := applicable[0]
					d.Uncertainty = append(d.Uncertainty, lock.Reasons...)
					for _, ancestor := range manifests {
						if ancestor.manager != "" && jsContains(path.Dir(ancestor.file.Path), scope) && !jsManagerMatches(ancestor.manager, lock) {
							d.Uncertainty = append(d.Uncertainty, "the declared packageManager does not match the supported lockfile family or generation")
						}
					}
					if path.Dir(lock.Path) != scope {
						member := false
						if relative, ok := jsRelative(path.Dir(lock.Path), scope); ok {
							switch lock.Family {
							case "npm":
								for _, root := range manifests {
									if path.Dir(root.file.Path) == path.Dir(lock.Path) && jsWorkspaceMember(root.workspaces, relative) {
										member = true
									}
								}
							case "pnpm":
								for _, root := range workspaces {
									if path.Dir(root.file.Path) == path.Dir(lock.Path) && jsWorkspaceMember(root.patterns, relative) {
										member = true
									}
								}
							}
						}
						if !member {
							d.Uncertainty = append(d.Uncertainty, "current supported workspace membership is not proven for the ancestor lockfile")
						}
					}
					if lock.Unsupported != "" {
						d.Uncertainty = append(d.Uncertainty, "the applicable lockfile format is unsupported or malformed")
					} else if locked, found := lock.Resolve(scope, declaredName, declared); !found {
						d.Uncertainty = append(d.Uncertainty, "the applicable lockfile has no supported direct selection for this manifest dependency")
					} else {
						d.Resolution, d.Source = "lockfile", lock.Path
						d.Uncertainty = append(d.Uncertainty, locked.Reasons...)
						version, valid := jsExactVersion(locked.Version)
						if valid {
							d.Version = version
						} else {
							d.Version = ""
							d.Uncertainty = append(d.Uncertainty, "the lock entry does not contain a supported exact semantic version")
						}
						matched, supported := jsSatisfies(locked.Version, declared)
						if !supported {
							d.Uncertainty = append(d.Uncertainty, "the declaration uses an unsupported version constraint or non-registry source")
						} else if !matched {
							d.Uncertainty = append(d.Uncertainty, "the lockfile version does not satisfy the manifest declaration")
						}
						if locked.Specifier != "" && locked.Specifier != declared {
							d.Uncertainty = append(d.Uncertainty, "the lockfile importer declaration differs from the manifest")
						}
						if !jsOfficialTarball(locked.Resolved, name, version) {
							d.Uncertainty = append(d.Uncertainty, "the lock entry does not prove an official npm/yarn registry tarball for this package and version")
						}
						d.Effective = valid && matched && supported && len(d.Uncertainty) == 0
					}
				default:
					d.Uncertainty = append(d.Uncertainty, "multiple same-scope or ancestor lockfiles are applicable; package-manager selection is ambiguous")
				}
				if !d.Effective {
					warnings = append(warnings, Warning{Path: d.Path, Code: "unresolved_dependency", Message: fmt.Sprintf("Dependency %q in %s has no proven effective offline selection.", d.Name, d.Group)})
				}
				dependencies = append(dependencies, d)
			}
		}
	}
	return dependencies, warnings, errors.Join(errs...)
}

func parseJSManifest(file walker.File) (jsManifest, error) {
	var raw map[string]json.RawMessage
	if err := jsJSON(file.Data, &raw); err != nil {
		return jsManifest{}, err
	}
	if raw == nil {
		return jsManifest{}, errors.New("package.json must be an object")
	}
	m := jsManifest{file: file, groups: map[string]map[string]string{}}
	for _, group := range jsGroups {
		if b, ok := raw[group]; ok {
			var values map[string]string
			if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
				return m, fmt.Errorf("%s must be an object", group)
			}
			if err := json.Unmarshal(b, &values); err != nil {
				return m, fmt.Errorf("%s must map names to string constraints", group)
			}
			m.groups[group] = values
		}
	}
	for _, key := range []string{"overrides", "resolutions"} {
		m.overrides = m.overrides || jsNonEmpty(raw[key])
	}
	if b := raw["pnpm"]; jsNonEmpty(b) {
		var pnpm map[string]json.RawMessage
		if err := json.Unmarshal(b, &pnpm); err != nil {
			return m, errors.New("pnpm manifest settings must be an object")
		}
		m.overrides = m.overrides || jsNonEmpty(pnpm["overrides"])
		m.overrides = m.overrides || jsNonEmpty(pnpm["patchedDependencies"])
	}
	if data, ok := raw["workspaces"]; ok {
		if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
			return m, errors.New("workspaces must be an array or packages object")
		}
		if bytes.HasPrefix(bytes.TrimSpace(data), []byte("[")) {
			if err := json.Unmarshal(data, &m.workspaces); err != nil {
				return m, errors.New("workspaces must contain string patterns")
			}
		} else {
			var config struct {
				Packages []string `json:"packages"`
			}
			if err := json.Unmarshal(data, &config); err != nil {
				return m, errors.New("workspaces must be a string array or packages object")
			}
			m.workspaces = config.Packages
		}
	}
	if data, ok := raw["packageManager"]; ok {
		if err := json.Unmarshal(data, &m.manager); err != nil || m.manager == "" {
			return m, errors.New("packageManager must be a nonempty string")
		}
	}
	return m, nil
}

func jsManagerMatches(manager string, lock *jsLock) bool {
	name, version, ok := strings.Cut(manager, "@")
	if !ok || name != lock.Family {
		return false
	}
	exact, ok := jsExactVersion(version)
	if !ok {
		return false
	}
	major, err := strconv.ParseInt(strings.TrimPrefix(semver.Major("v"+exact), "v"), 10, 64)
	if err != nil {
		return false
	}
	switch lock.Family {
	case "npm":
		// npm v2 is backward-compatible via its legacy dependencies section,
		// but this parser reads packages, whose semantics require npm >= 7.
		// v3 is explicitly backward-compatible to npm 7, not npm 6.
		if lock.Generation == 1 {
			return major >= 5
		}
		return (lock.Generation == 2 || lock.Generation == 3) && major >= 7
	case "pnpm":
		return lock.Generation == 9 && major >= 9
	case "yarn":
		return lock.Generation == 1 && major == 1
	}
	return false
}

func parseJSWorkspace(file walker.File) (jsWorkspace, error) {
	workspace := jsWorkspace{file: file}
	var document yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(file.Data))
	if err := decoder.Decode(&document); err != nil {
		return workspace, err
	}
	if err := validateLockYAML(&document, 0); err != nil {
		return workspace, err
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return workspace, errors.New("expected one pnpm workspace document")
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return workspace, errors.New("pnpm workspace must be a mapping")
	}
	root := document.Content[0]
	if patterns := yamlMappingValue(root, "packages"); patterns != nil {
		if patterns.Kind != yaml.SequenceNode {
			return workspace, errors.New("pnpm workspace packages must be a string sequence")
		}
		for _, pattern := range patterns.Content {
			if pattern.Kind != yaml.ScalarNode || pattern.Tag != "!!str" {
				return workspace, errors.New("pnpm workspace packages must contain string patterns")
			}
			workspace.patterns = append(workspace.patterns, pattern.Value)
		}
	}
	for _, key := range []string{"overrides", "resolutions", "patchedDependencies"} {
		if value := yamlMappingValue(root, key); value != nil {
			if value.Kind != yaml.MappingNode {
				return workspace, errors.New("pnpm override, resolution and patch settings must be mappings")
			}
			workspace.modifies = workspace.modifies || len(value.Content) > 0
		}
	}
	return workspace, nil
}

// Membership is deliberately narrower than package-manager glob engines:
// exact paths and ordinary '*' segments only. Exclusions, globstar, parent
// traversal and other syntax make membership unknown instead of overmatching.
func jsWorkspaceMember(patterns []string, relative string) bool {
	if len(patterns) == 0 || relative == "." || strings.HasPrefix(relative, "../") {
		return false
	}
	for _, pattern := range patterns {
		if pattern == "" || pattern == "." || path.IsAbs(pattern) || path.Clean(pattern) != pattern || strings.Contains(pattern, "**") || strings.ContainsAny(pattern, "!?[]{}\\") || strings.HasPrefix(pattern, "../") {
			return false
		}
	}
	for _, pattern := range patterns {
		if strings.Contains(pattern, "*") {
			for _, part := range strings.Split(relative, "/") {
				if strings.HasPrefix(part, ".") {
					return false
				}
			}
		}
		if matched, err := path.Match(pattern, relative); err == nil && matched {
			return true
		}
	}
	return false
}

func jsNonEmpty(raw json.RawMessage) bool {
	s := string(bytes.TrimSpace(raw))
	return s != "" && s != "null" && s != "{}" && s != "[]"
}

// jsJSON rejects duplicate object keys (encoding/json otherwise silently keeps
// the last value) and bounds nesting before decoding to a typed representation.
func jsJSON(data []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := jsJSONValue(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return json.Unmarshal(data, v)
}

func jsJSONValue(d *json.Decoder, depth int) error {
	if depth > 64 {
		return errors.New("JSON nesting exceeds 64 levels")
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	switch t {
	case json.Delim('{'):
		keys := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return err
			}
			s, ok := key.(string)
			if !ok {
				return errors.New("JSON object key must be a string")
			}
			if keys[s] {
				return errors.New("duplicate JSON object key")
			}
			keys[s] = true
			if err := jsJSONValue(d, depth+1); err != nil {
				return err
			}
		}
		_, err = d.Token()
		return err
	case json.Delim('['):
		for d.More() {
			if err := jsJSONValue(d, depth+1); err != nil {
				return err
			}
		}
		_, err = d.Token()
		return err
	case json.Delim('}'), json.Delim(']'):
		return errors.New("unexpected closing JSON delimiter")
	}
	return nil
}

type npmRecord struct {
	Version              string            `json:"version"`
	Resolved             string            `json:"resolved"`
	Name                 string            `json:"name"`
	Link                 bool              `json:"link"`
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
}

func parseNPMLock(file walker.File) (*jsLock, error) {
	var raw map[string]json.RawMessage
	if err := jsJSON(file.Data, &raw); err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, errors.New("npm lockfile must be an object")
	}
	var version int
	if err := json.Unmarshal(raw["lockfileVersion"], &version); err != nil {
		return nil, errors.New("npm lockfileVersion must be an integer")
	}
	lock := &jsLock{Path: file.Path, Family: "npm", Generation: version}
	if version < 1 || version > 3 {
		lock.Unsupported = "Only npm lockfileVersion 1, 2 and 3 are supported."
		return lock, nil
	}
	if version == 1 {
		// Version 1 is a dependency tree. Do not decode nested dependency maps
		// as modern declaration maps; only root direct selections are supported.
		var entries map[string]json.RawMessage
		if data, exists := raw["dependencies"]; exists {
			if err := json.Unmarshal(data, &entries); err != nil || entries == nil {
				return nil, errors.New("npm v1 dependencies must be an object")
			}
		}
		locked := map[string]jsLocked{}
		for name, data := range entries {
			var rec struct {
				Version  string `json:"version"`
				Resolved string `json:"resolved"`
			}
			if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
				return nil, errors.New("invalid npm v1 dependency record")
			}
			if err := json.Unmarshal(data, &rec); err != nil {
				return nil, errors.New("invalid npm v1 dependency record")
			}
			locked[name] = jsLocked{Version: rec.Version, Resolved: rec.Resolved}
		}
		lock.Resolve = func(scope, name, declared string) (jsLocked, bool) {
			if scope != path.Dir(file.Path) {
				return jsLocked{}, false
			}
			rec, ok := locked[name]
			return rec, ok
		}
		return lock, nil
	}
	var rawEntries map[string]json.RawMessage
	if err := json.Unmarshal(raw["packages"], &rawEntries); err != nil || rawEntries == nil {
		return nil, errors.New("npm v2/v3 packages must be an object")
	}
	entries := map[string]npmRecord{}
	for name, data := range rawEntries {
		var rec npmRecord
		if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
			return nil, errors.New("invalid npm v2/v3 package record")
		}
		if err := json.Unmarshal(data, &rec); err != nil {
			return nil, errors.New("invalid npm v2/v3 package record")
		}
		entries[name] = rec
	}
	lock.Resolve = func(scope, name, declared string) (jsLocked, bool) {
		rel, ok := jsRelative(path.Dir(file.Path), scope)
		if !ok {
			return jsLocked{}, false
		}
		manifest, hasImporter := entries[rel]
		if rel == "." {
			manifest, hasImporter = entries[""]
		}
		var importerSpec string
		for _, group := range []map[string]string{manifest.Dependencies, manifest.DevDependencies, manifest.OptionalDependencies, manifest.PeerDependencies} {
			if spec, ok := group[name]; ok {
				if importerSpec != "" && importerSpec != spec {
					return jsLocked{Reasons: []string{"the lockfile importer has conflicting dependency declarations"}}, true
				}
				importerSpec = spec
			}
		}
		for dir := rel; ; dir = path.Dir(dir) {
			key := path.Join(dir, "node_modules", name)
			if rec, exists := entries[key]; exists {
				locked := jsLocked{Version: rec.Version, Resolved: rec.Resolved, Specifier: importerSpec}
				if !hasImporter || importerSpec == "" {
					locked.Reasons = append(locked.Reasons, "the lockfile does not include a matching direct manifest importer")
				}
				if rec.Link {
					locked.Reasons = append(locked.Reasons, "the lock entry is a local or workspace link")
				}
				if rec.Name != "" && rec.Name != name {
					locked.Reasons = append(locked.Reasons, "the lock entry declares a different package identity")
				}
				return locked, true
			}
			if dir == "." || dir == "" {
				break
			}
		}
		return jsLocked{}, false
	}
	return lock, nil
}

func jsContains(parent, child string) bool {
	return parent == "." || parent == child || strings.HasPrefix(child, parent+"/")
}
func jsRelative(parent, child string) (string, bool) {
	if !jsContains(parent, child) {
		return "", false
	}
	if parent == child {
		return ".", true
	}
	if parent == "." {
		return child, true
	}
	return strings.TrimPrefix(child, parent+"/"), true
}

func jsAlias(name, declared string) (string, string, string) {
	if !strings.HasPrefix(declared, "npm:") {
		return name, "", ""
	}
	target := strings.TrimPrefix(declared, "npm:")
	index := strings.LastIndex(target, "@")
	if index <= 0 {
		return name, name, ""
	}
	return target[:index], name, target[index+1:]
}

func jsExactVersion(raw string) (string, bool) {
	s := strings.TrimPrefix(raw, "v")
	base := strings.SplitN(strings.SplitN(s, "-", 2)[0], "+", 2)[0]
	if len(strings.Split(base, ".")) != 3 || !semver.IsValid("v"+s) {
		return "", false
	}
	return s, true
}

// Exact, caret, and tilde full-version ranges are deliberately the only
// supported npm constraints. Tags, OR/comparison ranges, aliases and sources
// remain explicit uncertainty instead of invoking a package manager.
func jsSatisfies(version, declared string) (bool, bool) {
	v, ok := jsExactVersion(version)
	if !ok {
		return false, false
	}
	if exact, ok := jsExactVersion(declared); ok {
		return semver.Compare("v"+v, "v"+exact) == 0, true
	}
	if len(declared) < 2 || (declared[0] != '^' && declared[0] != '~') {
		return false, false
	}
	base, ok := jsExactVersion(declared[1:])
	if !ok || strings.ContainsAny(base+v, "-+") {
		return false, false
	}
	parts := strings.Split(base, ".")
	var numbers [3]int64
	for n, p := range parts {
		number, err := strconv.ParseInt(p, 10, 64)
		if err != nil || number == 1<<63-1 {
			return false, false
		}
		numbers[n] = number
	}
	upper := numbers
	switch {
	case declared[0] == '~':
		upper[1]++
		upper[2] = 0
	case numbers[0] > 0:
		upper[0]++
		upper[1] = 0
		upper[2] = 0
	case numbers[1] > 0:
		upper[1]++
		upper[2] = 0
	default:
		upper[2]++
	}
	bound := fmt.Sprintf("v%d.%d.%d", upper[0], upper[1], upper[2])
	return semver.Compare("v"+v, "v"+base) >= 0 && semver.Compare("v"+v, bound) < 0, true
}

func jsOfficialTarball(raw, name, version string) bool {
	if raw == "" || version == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || (u.Port() != "" && u.Port() != "443") {
		return false
	}
	if u.Hostname() != "registry.npmjs.org" && u.Hostname() != "registry.yarnpkg.com" {
		return false
	}
	return u.Path == "/"+name+"/-/"+path.Base(name)+"-"+version+".tgz"
}

func redactJSDeclaration(raw string) string {
	u, err := url.Parse(raw)
	if err == nil && u.User != nil {
		u.User = nil
		return u.String()
	}
	return raw
}
