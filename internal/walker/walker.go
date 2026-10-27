// Package walker discovers files relevant to Protocol 30 checks inside a
// single root directory. It only stats entries; it never reads, executes or
// uploads file contents.
package walker

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Defaults for the traversal ceilings.
const (
	DefaultMaxFileBytes = 16 << 20 // 16 MiB; npm lockfiles in large repos exceed a few MiB
	DefaultMaxFiles     = 20000
	DefaultMaxEntries   = 500000
	DefaultMaxDepth     = 64
)

// DefaultIgnoredDirs are directory names never descended into. They hold
// VCS metadata, installed dependencies or build output rather than the
// project's own code.
var DefaultIgnoredDirs = []string{
	".git", ".hg", ".svn",
	"node_modules", "bower_components", "vendor",
	"dist", "build", "out", "coverage",
	".next", ".nuxt", ".turbo", ".cache",
}

// Category groups file kinds.
type Category string

const (
	CategoryManifest Category = "manifest"
	CategorySource   Category = "source"
)

// File is a relevant file found under the root.
type File struct {
	Path     string // slash-separated, relative to the root
	Kind     string
	Category Category
	Size     int64
}

// Skip records an entry the walker deliberately did not inspect.
type Skip struct {
	Path   string
	Reason string
}

// Result is the outcome of a walk.
type Result struct {
	Root    string // absolute, symlink-resolved root
	Files   []File
	Skipped []Skip
}

// Options bounds a walk. Zero values select the defaults.
type Options struct {
	MaxFileBytes   int64
	MaxFiles       int
	MaxEntries     int
	MaxDepth       int
	IgnoredDirs    []string // nil selects DefaultIgnoredDirs
	FollowSymlinks bool     // include symlinked files whose target stays inside the root

	// readDir and lstat are swapped out by tests to inject I/O errors.
	readDir func(string) ([]os.DirEntry, error)
	lstat   func(string) (os.FileInfo, error)
}

// ErrLimit is wrapped by errors returned when a traversal ceiling is hit.
// Partial results are not returned in that case, so a truncated walk can
// never be mistaken for a complete one.
var ErrLimit = errors.New("traversal limit exceeded")

var manifestKinds = map[string]string{
	"go.mod":              "go-mod",
	"go.sum":              "go-sum",
	"go.work":             "go-work",
	"package.json":        "package-json",
	"package-lock.json":   "npm-lockfile",
	"npm-shrinkwrap.json": "npm-lockfile",
	"yarn.lock":           "yarn-lockfile",
	"pnpm-lock.yaml":      "pnpm-lockfile",
}

var sourceKinds = map[string]string{
	".go":  "source-go",
	".js":  "source-js",
	".jsx": "source-jsx",
	".ts":  "source-ts",
	".tsx": "source-tsx",
	".x":   "xdr-definition", // XDR schema files such as a vendored Stellar-ledger.x
}

// Classify returns the kind and category of a file name, or ok=false when
// the file is not relevant. Minified bundles and TypeScript declaration
// files are treated as generated and not classified as sources.
func Classify(name string) (kind string, cat Category, ok bool) {
	if k, found := manifestKinds[name]; found {
		return k, CategoryManifest, true
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".min.js") || strings.HasSuffix(lower, ".d.ts") {
		return "", "", false
	}
	if k, found := sourceKinds[path.Ext(lower)]; found {
		return k, CategorySource, true
	}
	return "", "", false
}

// Walk discovers relevant files below root. Results are sorted by path.
func Walk(root string, opts Options) (Result, error) {
	opts = withDefaults(opts)

	if root == "" {
		return Result{}, errors.New("scan path is empty")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return Result{}, fmt.Errorf("resolve scan path %q: %w", root, err)
	}
	// The root itself may be a symlink the user chose deliberately; resolve
	// it once and bound everything below to the resolved directory.
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return Result{}, fmt.Errorf("resolve scan path %q: %w", root, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return Result{}, fmt.Errorf("stat scan path %q: %w", root, err)
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("scan path %q is not a directory", root)
	}

	w := &walk{opts: opts, root: resolved, ignored: map[string]bool{}}
	for _, d := range opts.IgnoredDirs {
		w.ignored[d] = true
	}
	if err := w.dir(resolved, "", 0); err != nil {
		return Result{}, err
	}
	// Depth-first order puts "a/b.go" before "a.go"; sort so callers get
	// plain path order.
	sort.Slice(w.files, func(i, j int) bool { return w.files[i].Path < w.files[j].Path })
	sort.Slice(w.skipped, func(i, j int) bool { return w.skipped[i].Path < w.skipped[j].Path })
	return Result{Root: resolved, Files: w.files, Skipped: w.skipped}, nil
}

func withDefaults(o Options) Options {
	if o.MaxFileBytes <= 0 {
		o.MaxFileBytes = DefaultMaxFileBytes
	}
	if o.MaxFiles <= 0 {
		o.MaxFiles = DefaultMaxFiles
	}
	if o.MaxEntries <= 0 {
		o.MaxEntries = DefaultMaxEntries
	}
	if o.MaxDepth <= 0 {
		o.MaxDepth = DefaultMaxDepth
	}
	if o.IgnoredDirs == nil {
		o.IgnoredDirs = DefaultIgnoredDirs
	}
	if o.readDir == nil {
		o.readDir = os.ReadDir
	}
	if o.lstat == nil {
		o.lstat = os.Lstat
	}
	return o
}

type walk struct {
	opts    Options
	root    string
	ignored map[string]bool
	entries int
	files   []File
	skipped []Skip
}

func (w *walk) skip(rel, reason string) {
	w.skipped = append(w.skipped, Skip{Path: rel, Reason: reason})
}

// dir visits the directory at abs, whose root-relative path is rel.
func (w *walk) dir(abs, rel string, depth int) error {
	if depth > w.opts.MaxDepth {
		return fmt.Errorf("%w: %q is deeper than %d directories", ErrLimit, rel, w.opts.MaxDepth)
	}
	entries, err := w.opts.readDir(abs)
	if err != nil {
		if rel == "" {
			return fmt.Errorf("read scan root: %w", err)
		}
		// An unreadable subdirectory is reported, not fatal: the rest of
		// the tree can still be scanned and the gap is visible in the report.
		w.skip(rel, "unreadable directory: "+errText(err))
		return nil
	}
	for _, e := range entries {
		w.entries++
		if w.entries > w.opts.MaxEntries {
			return fmt.Errorf("%w: more than %d filesystem entries", ErrLimit, w.opts.MaxEntries)
		}
		name := e.Name()
		childAbs := filepath.Join(abs, name)
		childRel := path.Join(rel, name)

		switch {
		case e.Type()&fs.ModeSymlink != 0:
			if err := w.symlink(childAbs, childRel); err != nil {
				return err
			}
		case e.IsDir():
			if w.ignored[name] {
				w.skip(childRel, "ignored directory")
				continue
			}
			if err := w.dir(childAbs, childRel, depth+1); err != nil {
				return err
			}
		case e.Type().IsRegular():
			info, err := w.opts.lstat(childAbs)
			if err != nil {
				w.skip(childRel, "unreadable file: "+errText(err))
				continue
			}
			if err := w.file(childRel, name, info.Size()); err != nil {
				return err
			}
		default:
			// Devices, sockets and pipes are never relevant and reading
			// them could block.
			if _, _, ok := Classify(name); ok {
				w.skip(childRel, "not a regular file")
			}
		}
	}
	return nil
}

func (w *walk) file(rel, name string, size int64) error {
	kind, cat, ok := Classify(name)
	if !ok {
		return nil
	}
	if size > w.opts.MaxFileBytes {
		w.skip(rel, fmt.Sprintf("larger than %d bytes", w.opts.MaxFileBytes))
		return nil
	}
	if len(w.files) >= w.opts.MaxFiles {
		return fmt.Errorf("%w: more than %d relevant files", ErrLimit, w.opts.MaxFiles)
	}
	w.files = append(w.files, File{Path: rel, Kind: kind, Category: cat, Size: size})
	return nil
}

// symlink handles a symbolic link. Links are never followed into
// directories, which rules out cycles; with FollowSymlinks a link to a
// regular file is included only when its target resolves inside the root.
func (w *walk) symlink(abs, rel string) error {
	if !w.opts.FollowSymlinks {
		w.skip(rel, "symlink not followed")
		return nil
	}
	target, err := filepath.EvalSymlinks(abs)
	if err != nil {
		w.skip(rel, "broken symlink")
		return nil
	}
	if !within(w.root, target) {
		w.skip(rel, "symlink target outside scan root")
		return nil
	}
	info, err := os.Stat(target)
	if err != nil {
		w.skip(rel, "unreadable symlink target: "+errText(err))
		return nil
	}
	if info.IsDir() {
		w.skip(rel, "symlinked directory not followed")
		return nil
	}
	if !info.Mode().IsRegular() {
		w.skip(rel, "symlink to non-regular file not followed")
		return nil
	}
	return w.file(rel, path.Base(rel), info.Size())
}

func within(root, p string) bool {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) && !filepath.IsAbs(r)
}

// errText strips the absolute path from *PathError values so reports only
// carry root-relative paths.
func errText(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error()
	}
	return err.Error()
}
