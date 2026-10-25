package deps

import (
	"strings"
	"testing"

	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

const goSDK = "github.com/stellar/go-stellar-sdk"

func goInputs(input map[string]string) []walker.File {
	files := []walker.File{}
	for name, data := range input {
		files = append(files, walker.File{Path: name, Data: []byte(data), Size: int64(len(data))})
	}
	return files
}

func TestGoDeclaredVersionIsNotSelected(t *testing.T) {
	files := goInputs(map[string]string{
		"go.mod": "module example.test/app\n\ngo 1.25.0\nrequire github.com/stellar/go-stellar-sdk v0.7.3\nrequire example.test/indirect v1.2.0 // indirect\n",
		"go.sum": goSDK + " v0.8.0 h1:not-a-selected-version\n" + goSDK + " v0.6.0 h1:stale\n",
	})
	deps, warnings, err := inspectGo(files)
	if err != nil || len(deps) != 2 {
		t.Fatalf("deps=%+v warnings=%+v error=%v", deps, warnings, err)
	}
	for _, dependency := range deps {
		if dependency.Effective || dependency.Resolution != "go_minimum" || len(dependency.Uncertainty) == 0 || dependency.Group != "require" || dependency.Scope != "." || dependency.Path != "go.mod" {
			t.Fatalf("declared version treated as selected: %+v", dependency)
		}
	}
	if deps[0].Version != "v0.7.3" || deps[0].Declared != "v0.7.3" || deps[0].Line != 4 || deps[0].Source != "go.mod" {
		t.Fatalf("wrong declaration/source: %+v", deps[0])
	}
	if !goWarning(warnings, "go_checksum_not_selection") || !goWarning(warnings, "go_selection_unknown") {
		t.Fatalf("missing uncertainty warnings: %+v", warnings)
	}
}

func TestGoReplacements(t *testing.T) {
	for _, tc := range []struct {
		name        string
		replace     string
		wantName    string
		wantAlias   string
		wantVersion string
		wantSource  string
		external    bool
	}{
		{"same-module-wildcard", goSDK + " => " + goSDK + " v0.8.0", goSDK, "", "v0.8.0", "go.mod", false},
		{"same-module-version", goSDK + " v0.7.3 => " + goSDK + " v0.8.0", goSDK, "", "v0.8.0", "go.mod", false},
		{"fork", goSDK + " => example.test/fork v0.8.0", "example.test/fork", goSDK, "v0.8.0", "go.mod", false},
		{"local", goSDK + " => ./fork", goSDK, "", "", "go.mod", false},
		{"escaping-local", goSDK + " => ../outside", goSDK, "", "", "go.mod", true},
		{"absolute-local", goSDK + " => /tmp/outside", goSDK, "", "", "go.mod", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deps, warnings, err := inspectGo(goInputs(map[string]string{
				"go.mod": "module example.test/app\nrequire " + goSDK + " v0.7.3\nreplace " + tc.replace + "\n",
			}))
			if err != nil || len(deps) != 1 {
				t.Fatalf("deps=%+v warnings=%+v error=%v", deps, warnings, err)
			}
			d := deps[0]
			if d.Name != tc.wantName || d.Alias != tc.wantAlias || d.Version != tc.wantVersion || d.Source != tc.wantSource || d.Resolution != "replacement" || d.Effective || len(d.Uncertainty) < 2 {
				t.Fatalf("wrong replacement observation: %+v", d)
			}
			if goWarning(warnings, "go_external_path") != tc.external {
				t.Fatalf("external warning mismatch: %+v", warnings)
			}
		})
	}
}

func TestGoDifferentScopeMinimaDoNotSelectOneVersionGlobally(t *testing.T) {
	// The additional manifest can represent a transitive dependency in some
	// build contexts, but its relationship to the selected graph is not proved
	// by its presence in a scanned source tree. Preserve both observed minima.
	deps, _, err := inspectGo(goInputs(map[string]string{
		"go.mod":        "module example.test/app\nrequire (\n" + goSDK + " v0.7.3\nexample.test/higher v1.0.0\n)\n",
		"higher/go.mod": "module example.test/higher\nrequire " + goSDK + " v0.8.0\n",
	}))
	if err != nil || len(deps) != 3 {
		t.Fatalf("deps=%+v error=%v", deps, err)
	}
	versions := map[string]string{}
	for _, dependency := range deps {
		if dependency.Effective {
			t.Fatalf("offline minimum was promoted to effective: %+v", dependency)
		}
		if dependency.Name == goSDK {
			versions[dependency.Scope] = dependency.Version
		}
	}
	if versions["."] != "v0.7.3" || versions["higher"] != "v0.8.0" {
		t.Fatalf("cross-scope minima incorrectly merged: %+v", versions)
	}
}

func TestGoSpecificReplacementPrecedenceAndVersionMatch(t *testing.T) {
	deps, _, err := inspectGo(goInputs(map[string]string{
		"go.mod": "module example.test/app\nrequire " + goSDK + " v0.7.3\nreplace (\n" + goSDK + " => " + goSDK + " v0.6.0\n" + goSDK + " v0.7.3 => " + goSDK + " v0.8.0\n" + goSDK + " v0.7.2 => " + goSDK + " v0.5.0\n)\n",
	}))
	if err != nil || len(deps) != 1 || deps[0].Version != "v0.8.0" || deps[0].Effective {
		t.Fatalf("wrong matching replacement: %+v error=%v", deps, err)
	}
}

func TestGoWorkspaceOverride(t *testing.T) {
	deps, warnings, err := inspectGo(goInputs(map[string]string{
		"go.work":    "go 1.27.0\nuse ./app\nreplace " + goSDK + " => " + goSDK + " v0.8.0\n",
		"app/go.mod": "module example.test/app\nrequire " + goSDK + " v0.7.3\nreplace " + goSDK + " => example.test/fork v0.7.3\n",
	}))
	if err != nil || len(deps) != 1 {
		t.Fatalf("deps=%+v warnings=%+v error=%v", deps, warnings, err)
	}
	d := deps[0]
	if d.Name != goSDK || d.Alias != "" || d.Version != "v0.8.0" || d.Source != "go.work" || d.Scope != "app" || d.Effective {
		t.Fatalf("workspace override did not take precedence: %+v", d)
	}
}

func TestGoWorkspaceLocalModuleOverridesRelease(t *testing.T) {
	deps, warnings, err := inspectGo(goInputs(map[string]string{
		"go.work":    "go 1.27.0\nuse (\n./app\n./sdk\n)\n",
		"app/go.mod": "module example.test/app\nrequire " + goSDK + " v0.8.0\n",
		"sdk/go.mod": "module " + goSDK + "\n",
	}))
	if err != nil || len(deps) != 1 || deps[0].Version != "" || deps[0].Resolution != "workspace" || deps[0].Source != "go.work" || deps[0].Effective {
		t.Fatalf("local workspace module assigned a release: %+v warnings=%+v error=%v", deps, warnings, err)
	}
}

func TestGoWorkspaceOtherMainModuleReplacement(t *testing.T) {
	files := goInputs(map[string]string{
		"go.work":      "go 1.27.0\nuse (\n./app\n./other\n)\n",
		"app/go.mod":   "module example.test/app\nrequire " + goSDK + " v0.7.3\n",
		"other/go.mod": "module example.test/other\nreplace " + goSDK + " => " + goSDK + " v0.8.0\n",
	})
	deps, _, err := inspectGo(files)
	if err != nil || len(deps) != 1 || deps[0].Version != "v0.8.0" || deps[0].Source != "other/go.mod" || deps[0].Effective {
		t.Fatalf("other main-module replacement not observed: %+v error=%v", deps, err)
	}
}

func TestGoWorkspaceReplacementConflict(t *testing.T) {
	deps, _, err := inspectGo(goInputs(map[string]string{
		"go.work":      "go 1.27.0\nuse (\n./app\n./other\n)\n",
		"app/go.mod":   "module example.test/app\nrequire " + goSDK + " v0.7.3\nreplace " + goSDK + " => " + goSDK + " v0.8.0\n",
		"other/go.mod": "module example.test/other\nreplace " + goSDK + " => " + goSDK + " v0.6.0\n",
	}))
	if err != nil || len(deps) != 1 || deps[0].Version != "" || deps[0].Resolution != "unknown" || deps[0].Effective || !strings.Contains(strings.Join(deps[0].Uncertainty, " "), "conflict") {
		t.Fatalf("conflicting workspace replacement treated as known: %+v error=%v", deps, err)
	}
}

func TestGoAmbiguousWorkspaces(t *testing.T) {
	deps, warnings, err := inspectGo(goInputs(map[string]string{
		"go.work":     "go 1.27.0\nuse ./app\nreplace " + goSDK + " => " + goSDK + " v0.8.0\n",
		"app/go.work": "go 1.27.0\nuse .\nreplace " + goSDK + " => example.test/fork v0.7.3\n",
		"app/go.mod":  "module example.test/app\nrequire " + goSDK + " v0.7.3\n",
	}))
	if err != nil || len(deps) != 1 || deps[0].Version != "" || deps[0].Resolution != "unknown" || !goWarning(warnings, "go_workspace_ambiguous") {
		t.Fatalf("ambiguous workspace assumed selected: %+v warnings=%+v error=%v", deps, warnings, err)
	}
}

func TestGoUnavailableAndExternalWorkspaceInputs(t *testing.T) {
	deps, warnings, err := inspectGo(goInputs(map[string]string{
		"group/go.work": "go 1.27.0\nuse (\n../app\n./missing\n../../escape\n/tmp/outside\n)\n",
		"app/go.mod":    "module example.test/app\nrequire " + goSDK + " v0.8.0\n",
	}))
	if err != nil || len(deps) != 1 || deps[0].Effective || !goWarning(warnings, "go_external_path") || !goWarning(warnings, "go_workspace_module_unavailable") {
		t.Fatalf("missing path uncertainty not surfaced: %+v warnings=%+v error=%v", deps, warnings, err)
	}
}

func TestGoNestedModuleIsNotImplicitWorkspaceMember(t *testing.T) {
	deps, warnings, err := inspectGo(goInputs(map[string]string{
		"go.work":           "go 1.27.0\nuse ./app\nreplace " + goSDK + " => " + goSDK + " v0.8.0\n",
		"app/go.mod":        "module example.test/app\n",
		"app/nested/go.mod": "module example.test/nested\nrequire " + goSDK + " v0.7.3\n",
	}))
	if err != nil || len(deps) != 1 || deps[0].Version != "v0.7.3" || deps[0].Scope != "app/nested" || deps[0].Source != "app/nested/go.mod" || !strings.Contains(strings.Join(deps[0].Uncertainty, " "), "does not use") {
		t.Fatalf("nested module inherited unrelated workspace override: %+v warnings=%+v error=%v", deps, warnings, err)
	}
}

func TestGoExcludeAndRetractionScope(t *testing.T) {
	deps, _, err := inspectGo(goInputs(map[string]string{
		"go.mod": "module example.test/app\nrequire (\n" + goSDK + " v0.8.0\nexample.test/unaffected v1.0.0\n)\nexclude " + goSDK + " v0.8.0\nretract v1.0.0\n",
	}))
	if err != nil || len(deps) != 2 || deps[0].Version != "" || deps[0].Resolution != "unknown" || deps[1].Version != "v1.0.0" || deps[1].Effective {
		t.Fatalf("exclude or local retract misapplied: %+v error=%v", deps, err)
	}
}

func TestGoMalformedManifestPreservesOtherScopes(t *testing.T) {
	for _, broken := range []struct{ name, data string }{
		{"go.mod", "module example.test/broken\nrequire " + goSDK + " invalid-version\n"},
		{"go.mod", "require " + goSDK + " v0.8.0\n"},
		{"go.work", "go 1.27.0\nuse (\n"},
	} {
		deps, warnings, err := inspectGo(goInputs(map[string]string{
			broken.name:    broken.data,
			"valid/go.mod": "module example.test/valid\nrequire " + goSDK + " v0.8.0\n",
		}))
		if err == nil || len(deps) != 1 || deps[0].Scope != "valid" || !goWarning(warnings, "malformed_manifest") || !strings.Contains(err.Error(), broken.name) {
			t.Fatalf("malformed file not contextual/partial: %+v warnings=%+v error=%v", deps, warnings, err)
		}
	}
}

func TestGoChecksumOnlyDoesNotInventDependency(t *testing.T) {
	deps, warnings, err := inspectGo(goInputs(map[string]string{
		"go.sum":      goSDK + " v0.7.3 h1:old\n" + goSDK + " v0.8.0 h1:new\n",
		"go.work.sum": goSDK + " v0.7.3 h1:old\n",
	}))
	if err != nil || len(deps) != 0 || len(warnings) != 2 || !goWarning(warnings, "go_checksum_not_selection") {
		t.Fatalf("checksum created a selected dependency: %+v warnings=%+v error=%v", deps, warnings, err)
	}
}

func goWarning(warnings []Warning, code string) bool {
	for _, warning := range warnings {
		if warning.Code == code {
			return true
		}
	}
	return false
}
