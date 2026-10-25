package deps

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

const jsSDK = "@stellar/stellar-sdk"

func jsInputs(input map[string]string) []walker.File {
	var files []walker.File
	for name, data := range input {
		files = append(files, walker.File{Path: name, Data: []byte(data), Size: int64(len(data))})
	}
	return files
}

func jsManifestText(name, declared string) string {
	value, _ := json.Marshal(map[string]any{"dependencies": map[string]string{name: declared}})
	return string(value)
}

func npmLockText(version int, name, declared, selected, resolved string) string {
	var value map[string]any
	if version == 1 {
		value = map[string]any{"lockfileVersion": 1, "dependencies": map[string]any{name: map[string]any{"version": selected, "resolved": resolved}}}
	} else {
		value = map[string]any{"lockfileVersion": version, "packages": map[string]any{"": map[string]any{"dependencies": map[string]string{name: declared}}, "node_modules/" + name: map[string]any{"version": selected, "resolved": resolved}}}
	}
	data, _ := json.Marshal(value)
	return string(data)
}

func registryTarball(version string) string {
	return "https://registry.npmjs.org/@stellar/stellar-sdk/-/stellar-sdk-" + version + ".tgz"
}

func TestNPMDirectSelections(t *testing.T) {
	for _, version := range []int{1, 2, 3} {
		for _, selected := range []string{"16.4.0", "17.3.0"} {
			t.Run(fmt.Sprintf("v%d_%s", version, selected), func(t *testing.T) {
				declared := "^" + selected
				deps, warnings, err := inspectJS(jsInputs(map[string]string{"package.json": jsManifestText(jsSDK, declared), "package-lock.json": npmLockText(version, jsSDK, declared, selected, registryTarball(selected))}))
				if err != nil || len(deps) != 1 || len(warnings) != 0 {
					t.Fatalf("deps=%+v warnings=%+v error=%v", deps, warnings, err)
				}
				d := deps[0]
				if !d.Effective || len(d.Uncertainty) != 0 || d.Version != selected || d.Resolution != "lockfile" || d.Source != "package-lock.json" || d.Path != "package.json" || d.Scope != "." || d.Group != "dependencies" || d.Name != jsSDK {
					t.Fatalf("incorrect selection: %+v", d)
				}
			})
		}
	}
}

func TestJSUncertainSelections(t *testing.T) {
	modern := npmLockText(3, jsSDK, "17.3.0", "17.3.0", registryTarball("17.3.0"))
	for _, tc := range []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"missing-lock", map[string]string{"package.json": jsManifestText(jsSDK, "17.3.0")}, "no applicable lockfile"},
		{"unsupported-range", map[string]string{"package.json": jsManifestText(jsSDK, ">=16.4.0"), "package-lock.json": modern}, "unsupported version constraint"},
		{"lock-mismatch", map[string]string{"package.json": jsManifestText(jsSDK, "16.4.0"), "package-lock.json": modern}, "does not satisfy"},
		{"custom-registry", map[string]string{"package.json": jsManifestText(jsSDK, "17.3.0"), "package-lock.json": npmLockText(3, jsSDK, "17.3.0", "17.3.0", "https://packages.example.test/@stellar/stellar-sdk/-/stellar-sdk-17.3.0.tgz")}, "official npm/yarn"},
		{"wrong-tarball", map[string]string{"package.json": jsManifestText(jsSDK, "17.3.0"), "package-lock.json": npmLockText(3, jsSDK, "17.3.0", "17.3.0", "https://registry.npmjs.org/fork/-/fork-17.3.0.tgz")}, "official npm/yarn"},
		{"missing-origin", map[string]string{"package.json": jsManifestText(jsSDK, "17.3.0"), "package-lock.json": npmLockText(3, jsSDK, "17.3.0", "17.3.0", "")}, "official npm/yarn"},
		{"override", map[string]string{"package.json": `{"dependencies":{"@stellar/stellar-sdk":"17.3.0"},"overrides":{"@stellar/stellar-sdk":"16.4.0"}}`, "package-lock.json": modern}, "overrides or resolutions"},
		{"resolutions", map[string]string{"package.json": `{"dependencies":{"@stellar/stellar-sdk":"17.3.0"},"resolutions":{"@stellar/stellar-sdk":"16.4.0"}}`, "package-lock.json": modern}, "overrides or resolutions"},
		{"pnpm-override", map[string]string{"package.json": `{"dependencies":{"@stellar/stellar-sdk":"17.3.0"},"pnpm":{"overrides":{"@stellar/stellar-sdk":"16.4.0"}}}`, "package-lock.json": modern}, "overrides or resolutions"},
		{"alias", map[string]string{"package.json": jsManifestText("sdk", "npm:@stellar/stellar-sdk@17.3.0"), "package-lock.json": npmLockText(3, "sdk", "npm:@stellar/stellar-sdk@17.3.0", "17.3.0", registryTarball("17.3.0"))}, "npm aliases"},
		{"local", map[string]string{"package.json": jsManifestText(jsSDK, "file:./sdk"), "package-lock.json": modern}, "non-registry source"},
		{"workspace", map[string]string{"package.json": jsManifestText(jsSDK, "workspace:*"), "package-lock.json": modern}, "non-registry source"},
		{"multiple-locks", map[string]string{"package.json": jsManifestText(jsSDK, "17.3.0"), "package-lock.json": modern, "yarn.lock": "# yarn lockfile v1\n"}, "multiple same-scope"},
		{"unsupported-lock", map[string]string{"package.json": jsManifestText(jsSDK, "17.3.0"), "package-lock.json": `{"lockfileVersion":9}`}, "unsupported or malformed"},
		{"group-conflict", map[string]string{"package.json": `{"dependencies":{"@stellar/stellar-sdk":"17.3.0"},"devDependencies":{"@stellar/stellar-sdk":"16.4.0"}}`, "package-lock.json": modern}, "conflicting declarations"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deps, warnings, err := inspectJS(jsInputs(tc.files))
			if err != nil || len(deps) == 0 || len(warnings) == 0 {
				t.Fatalf("deps=%+v warnings=%+v error=%v", deps, warnings, err)
			}
			for _, d := range deps {
				if d.Effective || !strings.Contains(strings.Join(d.Uncertainty, ";"), tc.want) {
					t.Fatalf("uncertainty missing: %+v", d)
				}
			}
		})
	}
}

func TestNPMWorkspaceImporterScope(t *testing.T) {
	lock := `{"lockfileVersion":3,"packages":{"":{"name":"workspace"},"packages/app":{"dependencies":{"@stellar/stellar-sdk":"^17.3.0"}},"node_modules/@stellar/stellar-sdk":{"version":"17.3.0","resolved":"` + registryTarball("17.3.0") + `"},"packages/app/node_modules/@stellar/stellar-sdk":{"version":"16.4.0","resolved":"` + registryTarball("16.4.0") + `"}}}`
	deps, _, err := inspectJS(jsInputs(map[string]string{"package.json": `{"workspaces":["packages/*"]}`, "package-lock.json": lock, "packages/app/package.json": jsManifestText(jsSDK, "^17.3.0"), "packages/other/package.json": jsManifestText(jsSDK, "^17.3.0")}))
	if err != nil || len(deps) != 2 {
		t.Fatalf("deps=%+v error=%v", deps, err)
	}
	for _, d := range deps {
		if d.Effective {
			t.Fatalf("shadowed or absent importer accepted: %+v", d)
		}
		if d.Scope == "packages/app" && d.Version != "16.4.0" {
			t.Fatalf("failed nearest-node_modules resolution: %+v", d)
		}
	}
	lock = strings.ReplaceAll(lock, `,"packages/app/node_modules/@stellar/stellar-sdk":{"version":"16.4.0","resolved":"`+registryTarball("16.4.0")+`"}`, "")
	deps, _, err = inspectJS(jsInputs(map[string]string{"package.json": `{"workspaces":{"packages":["packages/app"]}}`, "package-lock.json": lock, "packages/app/package.json": jsManifestText(jsSDK, "^17.3.0")}))
	if err != nil || len(deps) != 1 || !deps[0].Effective {
		t.Fatalf("valid workspace importer not accepted: %+v %v", deps, err)
	}
}

func TestAncestorJSOverrideAndLockAmbiguity(t *testing.T) {
	files := map[string]string{
		"package.json":          `{"overrides":{"@stellar/stellar-sdk":"16.4.0"}}`,
		"app/package.json":      jsManifestText(jsSDK, "17.3.0"),
		"app/package-lock.json": npmLockText(3, jsSDK, "17.3.0", "17.3.0", registryTarball("17.3.0")),
	}
	deps, _, err := inspectJS(jsInputs(files))
	if err != nil || len(deps) != 1 || deps[0].Effective || !strings.Contains(strings.Join(deps[0].Uncertainty, " "), "overrides") {
		t.Fatalf("ancestor override ignored: %+v %v", deps, err)
	}
	delete(files, "package.json")
	files["package-lock.json"] = npmLockText(3, jsSDK, "17.3.0", "17.3.0", registryTarball("17.3.0"))
	deps, _, err = inspectJS(jsInputs(files))
	if err != nil || deps[0].Effective || !strings.Contains(strings.Join(deps[0].Uncertainty, " "), "multiple") {
		t.Fatalf("ancestor lock ambiguity ignored: %+v %v", deps, err)
	}
}

func TestJSManifestGroups(t *testing.T) {
	files := map[string]string{"package.json": `{"dependencies":{"a":"1.0.0"},"devDependencies":{"b":"1.0.0"},"optionalDependencies":{"c":"1.0.0"},"peerDependencies":{"d":"1.0.0"}}`}
	deps, _, err := inspectJS(jsInputs(files))
	if err != nil || len(deps) != 4 {
		t.Fatalf("deps=%+v error=%v", deps, err)
	}
	var groups []string
	for _, d := range deps {
		groups = append(groups, d.Group)
	}
	slices.Sort(groups)
	if !slices.Equal(groups, []string{"dependencies", "devDependencies", "optionalDependencies", "peerDependencies"}) {
		t.Fatalf("groups=%v", groups)
	}
}

func TestMalformedJSFilesPreserveOtherInventory(t *testing.T) {
	for _, bad := range []struct{ path, data string }{
		{"bad/package.json", `{"dependencies":`},
		{"bad/package.json", `{"dependencies":null}`},
		{"bad/package.json", `{"dependencies":{"a":3}}`},
		{"bad/package.json", `{"dependencies":{"a":"1.0.0","a":"2.0.0"}}`},
		{"bad/package.json", `{"dependencies":{}} {}`},
		{"bad/package-lock.json", `{"lockfileVersion":1,"dependencies":{"a":null}}`},
		{"bad/package-lock.json", `{"lockfileVersion":3,"packages":{"":null}}`},
		{"bad/package-lock.json", `{"lockfileVersion":3,"packages":{"":{},"node_modules/a":{"version":3}}}`},
	} {
		t.Run(bad.data, func(t *testing.T) {
			deps, warnings, err := inspectJS(jsInputs(map[string]string{bad.path: bad.data, "good/package.json": jsManifestText(jsSDK, "17.3.0")}))
			if err == nil || len(deps) != 1 || len(warnings) == 0 || deps[0].Path != "good/package.json" {
				t.Fatalf("partial inventory not preserved: %+v %+v %v", deps, warnings, err)
			}
		})
	}
	if _, _, err := inspectJS(jsInputs(map[string]string{"package-lock.json": `{"lockfileVersion":1}`})); err != nil {
		t.Fatalf("empty npm v1 lock should be valid: %v", err)
	}
}

func TestJSConstraintsAndOrigin(t *testing.T) {
	for _, tc := range []struct {
		version, declared string
		match, supported  bool
	}{
		{"17.3.0", "17.3.0", true, true}, {"17.3.0", "v17.3.0", true, true}, {"17.3.0", "^17.0.0", true, true}, {"18.0.0", "^17.0.0", false, true},
		{"0.8.0", "^0.7.0", false, true}, {"0.0.3", "^0.0.2", false, true}, {"17.3.2", "~17.3.0", true, true}, {"17.4.0", "~17.3.0", false, true},
		{"17.3.0", "latest", false, false}, {"17.3.0", "17.x", false, false}, {"17.3.0-beta.1", "^17.3.0", false, false}, {"17.3.0", "17.3", false, false},
	} {
		match, supported := jsSatisfies(tc.version, tc.declared)
		if match != tc.match || supported != tc.supported {
			t.Errorf("%s vs %s=%v,%v", tc.version, tc.declared, match, supported)
		}
	}
	for _, raw := range []string{"http://registry.npmjs.org/@stellar/stellar-sdk/-/stellar-sdk-17.3.0.tgz", "https://user:secret@registry.npmjs.org/@stellar/stellar-sdk/-/stellar-sdk-17.3.0.tgz", registryTarball("17.3.0") + "?foo=bar", registryTarball("16.4.0")} {
		if jsOfficialTarball(raw, jsSDK, "17.3.0") {
			t.Fatalf("untrusted origin accepted: %q", raw)
		}
	}
	if !jsOfficialTarball(strings.Replace(registryTarball("17.3.0"), "registry.npmjs.org", "registry.yarnpkg.com", 1)+"#checksum", jsSDK, "17.3.0") {
		t.Fatal("official Yarn mirror rejected")
	}
}

func TestJSDeclarationCredentialsRedacted(t *testing.T) {
	deps, _, err := inspectJS(jsInputs(map[string]string{"package.json": jsManifestText(jsSDK, "git+https://user:secret@example.test/fork.git")}))
	if err != nil || len(deps) != 1 {
		t.Fatalf("%+v %v", deps, err)
	}
	if strings.Contains(deps[0].Declared, "secret") || strings.Contains(deps[0].Declared, "user") {
		t.Fatal("URL userinfo was retained")
	}
}

func TestJSInventoryOrderingIndependentOfInputOrder(t *testing.T) {
	files := jsInputs(map[string]string{"b/package.json": jsManifestText(jsSDK, "17.3.0"), "a/package.json": jsManifestText(jsSDK, "16.4.0")})
	first, err := Inspect(files)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(files)
	second, err := Inspect(files)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) {
		t.Fatalf("nondeterministic inventory: %s vs %s", a, b)
	}
}

func TestNPMAncestorLockRequiresCurrentWorkspaceMembership(t *testing.T) {
	lock := `{"lockfileVersion":3,"packages":{"":{"name":"root"},"child":{"dependencies":{"@stellar/stellar-sdk":"16.4.0"}},"node_modules/@stellar/stellar-sdk":{"version":"16.4.0","resolved":"` + registryTarball("16.4.0") + `"}}}`
	for _, tc := range []struct {
		root      string
		effective bool
	}{
		{`{}`, false}, {`{"workspaces":["other"]}`, false}, {`{"workspaces":["child"]}`, true},
		{`{"workspaces":["*"]}`, true}, {`{"workspaces":["**"]}`, false}, {`{"workspaces":["child","!child"]}`, false},
	} {
		deps, _, err := inspectJS(jsInputs(map[string]string{"package.json": tc.root, "child/package.json": jsManifestText(jsSDK, "16.4.0"), "package-lock.json": lock}))
		if err != nil || len(deps) != 1 || deps[0].Effective != tc.effective {
			t.Fatalf("root=%s deps=%+v error=%v", tc.root, deps, err)
		}
	}
}

func pnpmWorkspaceLock() string {
	return "lockfileVersion: '9.0'\nimporters:\n  app:\n    dependencies:\n      '@stellar/stellar-sdk':\n        specifier: 17.3.0\n        version: 17.3.0\npackages:\n  '@stellar/stellar-sdk@17.3.0':\n    resolution:\n      tarball: " + registryTarball("17.3.0") + "\n"
}

func TestPNPMAncestorMembershipAndPatches(t *testing.T) {
	for _, tc := range []struct {
		name, config, manifestSuffix, lockSuffix string
		effective                                bool
	}{
		{"no-config", "", "", "", false},
		{"matching", "packages: [app]\n", "", "", true},
		{"matching-glob", "packages: ['a*']\n", "", "", true},
		{"mismatch", "packages: [other]\n", "", "", false},
		{"unsupported-glob", "packages: ['**']\n", "", "", false},
		{"exclusion", "packages: [app, '!app']\n", "", "", false},
		{"workspace-patch", "packages: [app]\npatchedDependencies:\n  '@stellar/stellar-sdk@17.3.0': patches/sdk.patch\n", "", "", false},
		{"manifest-patch", "packages: [app]\n", `,"pnpm":{"patchedDependencies":{"@stellar/stellar-sdk@17.3.0":"patches/sdk.patch"}}`, "", false},
		{"lock-patch", "packages: [app]\n", "", "patchedDependencies:\n  '@stellar/stellar-sdk@17.3.0':\n    hash: patchhash\n    path: patches/sdk.patch\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manifest := jsManifestText(jsSDK, "17.3.0")
			if tc.manifestSuffix != "" {
				manifest = strings.TrimSuffix(manifest, "}") + tc.manifestSuffix + "}"
			}
			files := map[string]string{"app/package.json": manifest, "pnpm-lock.yaml": pnpmWorkspaceLock() + tc.lockSuffix}
			if tc.config != "" {
				files["pnpm-workspace.yaml"] = tc.config
			}
			deps, _, err := inspectJS(jsInputs(files))
			if err != nil || len(deps) != 1 || deps[0].Effective != tc.effective {
				t.Fatalf("deps=%+v error=%v", deps, err)
			}
		})
	}
	for _, bad := range []string{"packages: bad\n", "packages: [app]\npackages: [other]\n", "packages: &list [app]\nother: *list\n", "packages: [app]\npatchedDependencies: true\n"} {
		_, _, err := inspectJS(jsInputs(map[string]string{"pnpm-workspace.yaml": bad}))
		if err == nil {
			t.Fatalf("invalid workspace accepted: %q", bad)
		}
	}
}

func TestJSPackageManagerMustMatchLockFamily(t *testing.T) {
	for _, manager := range []string{"pnpm@9.15.0", "yarn@1.22.22", "bun@1.0.0", "npm@latest", "nonsense"} {
		manifest := strings.TrimSuffix(jsManifestText(jsSDK, "16.4.0"), "}") + `,"packageManager":"` + manager + `"}`
		deps, _, err := inspectJS(jsInputs(map[string]string{"package.json": manifest, "package-lock.json": npmLockText(3, jsSDK, "16.4.0", "16.4.0", registryTarball("16.4.0"))}))
		if err != nil || len(deps) != 1 || deps[0].Effective || !strings.Contains(strings.Join(deps[0].Uncertainty, ";"), "packageManager") {
			t.Fatalf("manager=%s deps=%+v error=%v", manager, deps, err)
		}
	}
	manifest := strings.TrimSuffix(jsManifestText(jsSDK, "16.4.0"), "}") + `,"packageManager":"npm@10.9.0"}`
	deps, _, err := inspectJS(jsInputs(map[string]string{"package.json": manifest, "package-lock.json": npmLockText(3, jsSDK, "16.4.0", "16.4.0", registryTarball("16.4.0"))}))
	if err != nil || len(deps) != 1 || !deps[0].Effective {
		t.Fatalf("matching manager rejected: %+v %v", deps, err)
	}
	yarn := `# yarn lockfile v1
"@stellar/stellar-sdk@16.4.0":
  version "16.4.0"
  resolved "` + registryTarball("16.4.0") + `"
`
	for _, manager := range []string{"yarn@4.0.0", "yarn@1.22.22"} {
		manifest = strings.TrimSuffix(jsManifestText(jsSDK, "16.4.0"), "}") + `,"packageManager":"` + manager + `"}`
		deps, _, err := inspectJS(jsInputs(map[string]string{"package.json": manifest, "yarn.lock": yarn}))
		if err != nil || len(deps) != 1 || deps[0].Effective != (manager == "yarn@1.22.22") {
			t.Fatalf("manager=%s deps=%+v error=%v", manager, deps, err)
		}
	}
	files := map[string]string{"package.json": `{"packageManager":"pnpm@9.15.0","workspaces":["app"]}`, "app/package.json": jsManifestText(jsSDK, "16.4.0"), "package-lock.json": `{"lockfileVersion":3,"packages":{"":{},"app":{"dependencies":{"@stellar/stellar-sdk":"16.4.0"}},"node_modules/@stellar/stellar-sdk":{"version":"16.4.0","resolved":"` + registryTarball("16.4.0") + `"}}}`}
	deps, _, err = inspectJS(jsInputs(files))
	if err != nil || len(deps) != 1 || deps[0].Effective {
		t.Fatalf("inherited manager mismatch ignored: %+v %v", deps, err)
	}
}

func TestJSPackageManagerGenerationCompatibility(t *testing.T) {
	for _, tc := range []struct {
		manager    string
		generation int
		effective  bool
	}{
		{"npm@4.6.1", 1, false}, {"npm@6.14.18", 1, true},
		{"npm@6.14.18", 2, false}, {"npm@7.24.2", 2, true},
		{"npm@6.14.18", 3, false}, {"npm@7.24.2", 3, true}, {"npm@10.9.0", 3, true},
	} {
		t.Run(fmt.Sprintf("%s_lock%d", tc.manager, tc.generation), func(t *testing.T) {
			manifest := strings.TrimSuffix(jsManifestText(jsSDK, "16.4.0"), "}") + `,"packageManager":"` + tc.manager + `"}`
			deps, _, err := inspectJS(jsInputs(map[string]string{"package.json": manifest, "package-lock.json": npmLockText(tc.generation, jsSDK, "16.4.0", "16.4.0", registryTarball("16.4.0"))}))
			if err != nil || len(deps) != 1 || deps[0].Effective != tc.effective {
				t.Fatalf("deps=%+v error=%v", deps, err)
			}
		})
	}
	lock := `lockfileVersion: '9.0'
importers:
  .:
    dependencies:
      '@stellar/stellar-sdk':
        specifier: 16.4.0
        version: 16.4.0
packages:
  '@stellar/stellar-sdk@16.4.0':
    resolution:
      tarball: ` + registryTarball("16.4.0") + "\n"
	for _, manager := range []string{"pnpm@8.15.0", "pnpm@9.15.0"} {
		manifest := strings.TrimSuffix(jsManifestText(jsSDK, "16.4.0"), "}") + `,"packageManager":"` + manager + `"}`
		deps, _, err := inspectJS(jsInputs(map[string]string{"package.json": manifest, "pnpm-lock.yaml": lock}))
		if err != nil || len(deps) != 1 || deps[0].Effective != (manager == "pnpm@9.15.0") {
			t.Fatalf("manager=%s deps=%+v error=%v", manager, deps, err)
		}
	}
}
