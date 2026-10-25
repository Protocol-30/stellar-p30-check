package deps

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/Protocol-30/stellar-p30-check/internal/walker"
	"golang.org/x/mod/modfile"
)

type goModuleInput struct {
	file walker.File
	mod  *modfile.File
}

type goWorkspaceInput struct {
	file walker.File
	work *modfile.WorkFile
	uses map[string]bool
}

// inspectGo observes declared minima and replacement provenance; it never
// claims to reconstruct a selected build list from an incomplete offline graph.
func inspectGo(files []walker.File) ([]Dependency, []Warning, error) {
	files = slices.Clone(files)
	slices.SortFunc(files, func(a, b walker.File) int { return strings.Compare(a.Path, b.Path) })
	modules := map[string]*goModuleInput{}
	workspaces := []*goWorkspaceInput{}
	warnings := []Warning{}
	var parseErrors []error
	for _, file := range files {
		switch path.Base(file.Path) {
		case "go.mod":
			mod, err := modfile.Parse(file.Path, file.Data, nil)
			if err == nil && (mod.Module == nil || mod.Module.Mod.Path == "") {
				err = fmt.Errorf("module directive is missing")
			}
			if err != nil {
				parseErrors = append(parseErrors, fmt.Errorf("parse Go module %q: %w", file.Path, err))
				warnings = append(warnings, Warning{Path: file.Path, Code: "malformed_manifest", Message: "Cannot inspect Go dependencies: " + err.Error()})
				continue
			}
			modules[path.Dir(file.Path)] = &goModuleInput{file: file, mod: mod}
		case "go.work":
			work, err := modfile.ParseWork(file.Path, file.Data, nil)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Errorf("parse Go workspace %q: %w", file.Path, err))
				warnings = append(warnings, Warning{Path: file.Path, Code: "malformed_manifest", Message: "Cannot inspect Go workspace: " + err.Error()})
				continue
			}
			workspaces = append(workspaces, &goWorkspaceInput{file: file, work: work, uses: map[string]bool{}})
		case "go.sum", "go.work.sum":
			warnings = append(warnings, Warning{Path: file.Path, Code: "go_checksum_not_selection", Message: "Checksum entries are not selected dependency versions; this file is not used for SDK classification."})
		}
	}

	// Resolve only path membership against supplied inventory. No filesystem
	// lookup is performed, even for an absolute or escaping workspace use.
	for _, workspace := range workspaces {
		for _, use := range workspace.work.Use {
			scope, ok := goLocalScope(path.Dir(workspace.file.Path), use.Path)
			if !ok {
				warnings = append(warnings, Warning{Path: workspace.file.Path, Line: use.Syntax.Start.Line, Code: "go_external_path", Message: fmt.Sprintf("Workspace use %q is outside or not a portable scan-root-relative path; it was not read.", use.Path)})
				continue
			}
			workspace.uses[scope] = true
			if modules[scope] == nil {
				warnings = append(warnings, Warning{Path: workspace.file.Path, Line: use.Syntax.Start.Line, Code: "go_workspace_module_unavailable", Message: fmt.Sprintf("Workspace module %q has no parsed go.mod in the selected inventory; the build graph is incomplete.", scope)})
			}
		}
	}

	scopes := make([]string, 0, len(modules))
	for scope := range modules {
		scopes = append(scopes, scope)
	}
	slices.Sort(scopes)
	dependencies := []Dependency{}
	for _, scope := range scopes {
		module := modules[scope]
		var memberships []*goWorkspaceInput
		for _, workspace := range workspaces {
			if workspace.uses[scope] {
				memberships = append(memberships, workspace)
			}
		}
		var workspace *goWorkspaceInput
		if len(memberships) == 1 {
			workspace = memberships[0]
		} else if len(memberships) > 1 {
			warnings = append(warnings, Warning{Path: module.file.Path, Code: "go_workspace_ambiguous", Message: "Multiple supplied workspaces use this module; no workspace override is assumed because GOWORK/build invocation is unknown."})
		}
		if len(module.mod.Require) != 0 {
			warnings = append(warnings, Warning{Path: module.file.Path, Code: "go_selection_unknown", Message: "go.mod require versions are declared minima, not an effective build list. MVS, unavailable transitive manifests, GOWORK, and runtime overrides are not resolved offline."})
		}
		for _, require := range module.mod.Require {
			dependency := Dependency{
				Ecosystem: "go", Name: require.Mod.Path, Scope: scope, Path: module.file.Path, Group: "require",
				Line: require.Syntax.Start.Line, Declared: require.Mod.Version, Version: require.Mod.Version,
				Resolution: "go_minimum", Source: module.file.Path, Effective: false,
				Uncertainty: []string{"Declared minimum only; the effective selected version cannot be established without the complete module graph and build context."},
			}
			if len(memberships) > 1 {
				dependency.Version = ""
				dependency.Resolution = "unknown"
				dependency.Uncertainty = append(dependency.Uncertainty, "Multiple supplied workspaces use this module; workspace replacements are ambiguous.")
			}
			for _, exclude := range module.mod.Exclude {
				if exclude.Mod == require.Mod {
					dependency.Version = ""
					dependency.Resolution = "unknown"
					dependency.Uncertainty = append(dependency.Uncertainty, "The declared minimum is excluded in this module; no replacement selected version is inferred.")
				}
			}
			if workspace == nil && len(memberships) == 0 && goHasAncestorWorkspace(scope, workspaces) {
				dependency.Uncertainty = append(dependency.Uncertainty, "An ancestor workspace does not use this module; workspace-enabled builds may reject it, while GOWORK=off may differ.")
			}
			// Workspace-local module identity supersedes a released dependency
			// version. Local content carries no trustworthy release version.
			if workspace != nil && goWorkspaceProvides(workspace, require.Mod.Path, modules) {
				dependency.Version = ""
				dependency.Resolution = "workspace"
				dependency.Source = workspace.file.Path
				dependency.Uncertainty = append(dependency.Uncertainty, "The supplied workspace uses a local module with this identity; its contents cannot be assigned the declared release version.")
			} else if len(memberships) <= 1 {
				replacement, source, ambiguous := goReplacement(module, workspace, require.Mod.Path, require.Mod.Version, modules)
				if ambiguous {
					dependency.Version = ""
					dependency.Resolution = "unknown"
					dependency.Uncertainty = append(dependency.Uncertainty, "Applicable replacement directives conflict; the effective replacement is unknown.")
				} else if replacement != nil {
					dependency.Source = source
					dependency.Resolution = "replacement"
					dependency.Uncertainty = append(dependency.Uncertainty, "A replacement is observed for the declared version; MVS or a different build/workspace context may change its applicability.")
					if replacement.New.Version == "" {
						dependency.Version = ""
						dependency.Uncertainty = append(dependency.Uncertainty, fmt.Sprintf("Local replacement %q has unversioned contents; no SDK release identity is assumed.", replacement.New.Path))
						if _, ok := goLocalScope(path.Dir(source), replacement.New.Path); !ok {
							warnings = append(warnings, Warning{Path: source, Line: replacement.Syntax.Start.Line, Code: "go_external_path", Message: fmt.Sprintf("Local replacement %q is outside or not a portable scan-root-relative path; it was not read.", replacement.New.Path)})
						}
					} else {
						dependency.Version = replacement.New.Version
						if replacement.New.Path != dependency.Name {
							dependency.Alias = dependency.Name
							dependency.Name = replacement.New.Path
							dependency.Uncertainty = append(dependency.Uncertainty, "Replacement module identity differs from the declared module; official SDK capabilities cannot be transferred to a fork.")
						}
					}
				}
			}
			dependencies = append(dependencies, dependency)
		}
	}
	return dependencies, warnings, errors.Join(parseErrors...)
}

func goLocalScope(base, local string) (string, bool) {
	// Backslashes/drive paths have host-dependent meaning. Treat them as
	// unavailable rather than accidentally normalizing them to an in-root path.
	if local == "" || path.IsAbs(local) || strings.ContainsAny(local, "\\:") {
		return "", false
	}
	scope := path.Clean(path.Join(base, local))
	return scope, scope != ".." && !strings.HasPrefix(scope, "../")
}

func goHasAncestorWorkspace(scope string, workspaces []*goWorkspaceInput) bool {
	for _, workspace := range workspaces {
		base := path.Dir(workspace.file.Path)
		if base == "." || scope == base || strings.HasPrefix(scope, base+"/") {
			return true
		}
	}
	return false
}

func goWorkspaceProvides(workspace *goWorkspaceInput, identity string, modules map[string]*goModuleInput) bool {
	for scope := range workspace.uses {
		if module := modules[scope]; module != nil && module.mod.Module.Mod.Path == identity {
			return true
		}
	}
	return false
}

func goReplacement(module *goModuleInput, workspace *goWorkspaceInput, identity, version string, modules map[string]*goModuleInput) (*modfile.Replace, string, bool) {
	if workspace != nil {
		if replacement, ambiguous := goFindReplacement(workspace.work.Replace, identity, version); replacement != nil || ambiguous {
			return replacement, workspace.file.Path, ambiguous
		}
		// In workspace mode replacement directives from all main modules may
		// affect selection. A go.work replacement, handled above, overrides them.
		var found *modfile.Replace
		var source string
		for _, scope := range goSortedUseScopes(workspace) {
			other := modules[scope]
			if other == nil {
				continue
			}
			replacement, ambiguous := goFindReplacement(other.mod.Replace, identity, version)
			if ambiguous {
				return nil, "", true
			}
			if replacement != nil {
				if found != nil && (found.New != replacement.New || (replacement.New.Version == "" && path.Dir(source) != scope)) {
					return nil, "", true
				}
				found, source = replacement, other.file.Path
			}
		}
		return found, source, false
	}
	replacement, ambiguous := goFindReplacement(module.mod.Replace, identity, version)
	return replacement, module.file.Path, ambiguous
}

func goSortedUseScopes(workspace *goWorkspaceInput) []string {
	scopes := make([]string, 0, len(workspace.uses))
	for scope := range workspace.uses {
		scopes = append(scopes, scope)
	}
	slices.Sort(scopes)
	return scopes
}

func goFindReplacement(replacements []*modfile.Replace, identity, version string) (*modfile.Replace, bool) {
	var exact, wildcard *modfile.Replace
	for _, replacement := range replacements {
		if replacement.Old.Path != identity || (replacement.Old.Version != "" && replacement.Old.Version != version) {
			continue
		}
		candidate := &wildcard
		if replacement.Old.Version != "" {
			candidate = &exact
		}
		if *candidate != nil && (*candidate).New != replacement.New {
			return nil, true
		}
		*candidate = replacement
	}
	if exact != nil {
		return exact, false
	}
	return wildcard, false
}
