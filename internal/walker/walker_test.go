package walker

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// mkTree creates files (path -> contents) under a fresh temp dir.
func mkTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for p, body := range files {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func paths(fs []File) []string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.Path)
	}
	return out
}

func skips(ss []Skip) map[string]string {
	out := map[string]string{}
	for _, s := range ss {
		out[s.Path] = s.Reason
	}
	return out
}

func symlinkOrSkip(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		kind string
		cat  Category
		ok   bool
	}{
		{"go.mod", "go-mod", CategoryManifest, true},
		{"go.sum", "go-sum", CategoryManifest, true},
		{"package.json", "package-json", CategoryManifest, true},
		{"package-lock.json", "npm-lockfile", CategoryManifest, true},
		{"yarn.lock", "yarn-lockfile", CategoryManifest, true},
		{"pnpm-lock.yaml", "pnpm-lockfile", CategoryManifest, true},
		{"main.go", "source-go", CategorySource, true},
		{"App.TSX", "source-tsx", CategorySource, true},
		{"index.js", "source-js", CategorySource, true},
		{"view.jsx", "source-jsx", CategorySource, true},
		{"lib.ts", "source-ts", CategorySource, true},
		{"bundle.min.js", "", "", false},
		{"types.d.ts", "", "", false},
		{"README.md", "", "", false},
		{"Package.json", "", "", false}, // manifests are matched exactly
	}
	for _, c := range cases {
		kind, cat, ok := Classify(c.name)
		if kind != c.kind || cat != c.cat || ok != c.ok {
			t.Errorf("Classify(%q) = %q, %q, %v; want %q, %q, %v", c.name, kind, cat, ok, c.kind, c.cat, c.ok)
		}
	}
}

func TestWalkHappyPathSortedAndFiltered(t *testing.T) {
	root := mkTree(t, map[string]string{
		"go.mod":                    "module x\n",
		"cmd/app/main.go":           "package main\n",
		"web/package.json":          "{}",
		"web/src/b.ts":              "",
		"web/src/a.tsx":             "",
		"web/node_modules/m/i.js":   "",
		"web/dist/out.js":           "",
		".git/config":               "",
		"vendor/x/y.go":             "",
		"docs/notes.md":             "",
		"web/public/lib.min.js":     "",
		"internal/zz/types.d.ts":    "",
		"internal/aa/handler_x.go":  "",
		"internal/aa/handler_x.txt": "",
		"cmd.go":                    "",
	})
	res, err := Walk(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"cmd.go",
		"cmd/app/main.go",
		"go.mod",
		"internal/aa/handler_x.go",
		"web/package.json",
		"web/src/a.tsx",
		"web/src/b.ts",
	}
	if got := paths(res.Files); !reflect.DeepEqual(got, want) {
		t.Errorf("files = %v\nwant %v", got, want)
	}
	wantSkips := map[string]string{
		".git":             "ignored directory",
		"vendor":           "ignored directory",
		"web/dist":         "ignored directory",
		"web/node_modules": "ignored directory",
	}
	if got := skips(res.Skipped); !reflect.DeepEqual(got, wantSkips) {
		t.Errorf("skipped = %v\nwant %v", got, wantSkips)
	}

	// Identical input must give identical output.
	again, err := Walk(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res, again) {
		t.Error("two walks of the same tree differ")
	}
}

func TestWalkCustomIgnoredDirs(t *testing.T) {
	root := mkTree(t, map[string]string{
		"node_modules/a/index.js": "",
		"gen/x.go":                "",
	})
	res, err := Walk(root, Options{IgnoredDirs: []string{"gen"}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := paths(res.Files), []string{"node_modules/a/index.js"}; !reflect.DeepEqual(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
}

func TestWalkInvalidRoots(t *testing.T) {
	file := filepath.Join(mkTree(t, map[string]string{"go.mod": ""}), "go.mod")
	cases := map[string]string{
		"empty":     "",
		"missing":   filepath.Join(t.TempDir(), "does-not-exist"),
		"not a dir": file,
		"nul byte":  "bad\x00path",
	}
	for name, root := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Walk(root, Options{}); err == nil {
				t.Fatalf("Walk(%q) succeeded, want error", root)
			}
		})
	}
}

func TestWalkFileSizeLimit(t *testing.T) {
	root := mkTree(t, map[string]string{
		"small.go": "package x\n",
		"big.go":   strings.Repeat("x", 100),
	})
	res, err := Walk(root, Options{MaxFileBytes: 50})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := paths(res.Files), []string{"small.go"}; !reflect.DeepEqual(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
	if r := skips(res.Skipped)["big.go"]; r != "larger than 50 bytes" {
		t.Errorf("big.go skip reason = %q", r)
	}
}

func TestWalkLimitsAreErrors(t *testing.T) {
	files := map[string]string{}
	for i := 0; i < 5; i++ {
		files[fmt.Sprintf("f%d.go", i)] = ""
	}
	root := mkTree(t, files)

	if _, err := Walk(root, Options{MaxFiles: 4}); !errors.Is(err, ErrLimit) {
		t.Errorf("MaxFiles: err = %v, want ErrLimit", err)
	}
	if _, err := Walk(root, Options{MaxFiles: 5}); err != nil {
		t.Errorf("MaxFiles at limit: %v", err)
	}
	if _, err := Walk(root, Options{MaxEntries: 3}); !errors.Is(err, ErrLimit) {
		t.Errorf("MaxEntries: err = %v, want ErrLimit", err)
	}

	deep := mkTree(t, map[string]string{"a/b/c/d/x.go": ""})
	if _, err := Walk(deep, Options{MaxDepth: 2}); !errors.Is(err, ErrLimit) {
		t.Errorf("MaxDepth: err = %v, want ErrLimit", err)
	}
	if _, err := Walk(deep, Options{MaxDepth: 4}); err != nil {
		t.Errorf("MaxDepth at limit: %v", err)
	}
}

func TestWalkSymlinksNotFollowedByDefault(t *testing.T) {
	outside := mkTree(t, map[string]string{"secret.go": "package s\n", "pkg/x.go": ""})
	root := mkTree(t, map[string]string{"real.go": "package r\n"})
	symlinkOrSkip(t, filepath.Join(outside, "secret.go"), filepath.Join(root, "escape.go"))
	symlinkOrSkip(t, filepath.Join(outside, "pkg"), filepath.Join(root, "escape-dir"))
	symlinkOrSkip(t, filepath.Join(root, "real.go"), filepath.Join(root, "alias.go"))

	res, err := Walk(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := paths(res.Files), []string{"real.go"}; !reflect.DeepEqual(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
	wantSkips := map[string]string{
		"alias.go":   "symlink not followed",
		"escape-dir": "symlink not followed",
		"escape.go":  "symlink not followed",
	}
	if got := skips(res.Skipped); !reflect.DeepEqual(got, wantSkips) {
		t.Errorf("skipped = %v\nwant %v", got, wantSkips)
	}
}

func TestWalkFollowSymlinksStaysInsideRoot(t *testing.T) {
	outside := mkTree(t, map[string]string{"secret.go": "package s\n"})
	root := mkTree(t, map[string]string{"real.go": "package r\n", "sub/inner.go": ""})
	symlinkOrSkip(t, filepath.Join(outside, "secret.go"), filepath.Join(root, "escape.go"))
	symlinkOrSkip(t, filepath.Join(root, "real.go"), filepath.Join(root, "alias.go"))
	symlinkOrSkip(t, filepath.Join(root, "sub"), filepath.Join(root, "loop"))
	symlinkOrSkip(t, filepath.Join(root, "missing.go"), filepath.Join(root, "broken.go"))

	res, err := Walk(root, Options{FollowSymlinks: true})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := paths(res.Files), []string{"alias.go", "real.go", "sub/inner.go"}; !reflect.DeepEqual(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
	wantSkips := map[string]string{
		"broken.go": "broken symlink",
		"escape.go": "symlink target outside scan root",
		"loop":      "symlinked directory not followed",
	}
	if got := skips(res.Skipped); !reflect.DeepEqual(got, wantSkips) {
		t.Errorf("skipped = %v\nwant %v", got, wantSkips)
	}
}

func TestWalkSymlinkedRootIsResolved(t *testing.T) {
	real := mkTree(t, map[string]string{"main.go": ""})
	link := filepath.Join(t.TempDir(), "link")
	symlinkOrSkip(t, real, link)
	res, err := Walk(link, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := paths(res.Files), []string{"main.go"}; !reflect.DeepEqual(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
}

func TestWalkUnreadableEntriesAreReportedNotFatal(t *testing.T) {
	root := mkTree(t, map[string]string{
		"ok.go":        "",
		"locked/a.go":  "",
		"vanishing.go": "",
	})
	denied := &fs.PathError{Op: "open", Path: filepath.Join(root, "locked"), Err: fs.ErrPermission}
	opts := Options{
		readDir: func(dir string) ([]os.DirEntry, error) {
			if filepath.Base(dir) == "locked" {
				return nil, denied
			}
			return os.ReadDir(dir)
		},
		lstat: func(p string) (os.FileInfo, error) {
			if filepath.Base(p) == "vanishing.go" {
				return nil, &fs.PathError{Op: "lstat", Path: p, Err: fs.ErrNotExist}
			}
			return os.Lstat(p)
		},
	}
	res, err := Walk(root, opts)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := paths(res.Files), []string{"ok.go"}; !reflect.DeepEqual(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
	wantSkips := map[string]string{
		"locked":       "unreadable directory: permission denied",
		"vanishing.go": "unreadable file: file does not exist",
	}
	if got := skips(res.Skipped); !reflect.DeepEqual(got, wantSkips) {
		t.Errorf("skipped = %v\nwant %v", got, wantSkips)
	}
	for _, s := range res.Skipped {
		if strings.Contains(s.Reason, root) {
			t.Errorf("skip reason leaks absolute path: %q", s.Reason)
		}
	}
}

func TestWalkUnreadableRootIsFatal(t *testing.T) {
	root := mkTree(t, map[string]string{"a.go": ""})
	opts := Options{readDir: func(string) ([]os.DirEntry, error) { return nil, fs.ErrPermission }}
	if _, err := Walk(root, opts); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("err = %v, want permission error", err)
	}
}
