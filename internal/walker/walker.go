// Package walker discovers relevant regular files without executing target code.
package walker

import (
	"bytes"
	"fmt"
	goscanner "go/scanner"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Limits bound discovery work and retained source contents. All values must be
// positive. MaxFiles counts selected regular-file candidates, including ones
// subsequently recognized as binary or generated. Root has directory depth 0.
type Limits struct {
	MaxFileBytes  int64
	MaxTotalBytes int64
	MaxFiles      int
	MaxEntries    int
	MaxDepth      int
}

func DefaultLimits() Limits {
	return Limits{
		MaxFileBytes: 2 * 1024 * 1024, MaxTotalBytes: 64 * 1024 * 1024,
		MaxFiles: 10000, MaxEntries: 100000, MaxDepth: 64,
	}
}

// File contains a root-relative slash path and safely read contents. Data is
// intended for future rule analysis, not inclusion in public reports.
type File struct {
	Path string
	Kind string
	Size int64
	Data []byte
}

type Skip struct {
	Path   string
	Reason string
}

// Complete describes discovery of the selected regular-file scope, not protocol
// compatibility or coverage of intentionally excluded directories and files.
type Result struct {
	Files          []File
	Skipped        []Skip
	EntriesVisited int
	BytesRead      int64
	Complete       bool
}

var ignoredDirectories = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true,
	"build": true, ".next": true, "coverage": true, ".cache": true,
}

// rootFS provides a small test seam for deterministic I/O errors. Production
// operations always use an os.Root; scanner targets are never executed.
type rootFS interface {
	Lstat(string) (fs.FileInfo, error)
	OpenFile(string, int, fs.FileMode) (*os.File, error)
}

type scanner struct {
	root           rootFS
	limits         Limits
	result         Result
	candidateFiles int
}

// Scan reads a directory tree under path. Static symlinks and special files are
// excluded. Unexpected I/O failures or exceeded limits return partial results
// with Complete=false. The underlying os.Root prevents symlink traversal outside
// the opened root; concurrent target mutation does not yield snapshot semantics.
func Scan(target string, limits Limits) (Result, error) {
	if err := validateLimits(limits); err != nil {
		return emptyResult(), err
	}
	if target == "" {
		return emptyResult(), fmt.Errorf("scan root must not be empty")
	}
	// A trailing slash or /. would otherwise make Lstat follow a final symlink.
	target = filepath.Clean(target)
	info, err := os.Lstat(target)
	if err != nil {
		return emptyResult(), fmt.Errorf("inspect scan root %q: %w", target, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return emptyResult(), fmt.Errorf("scan root %q is a symbolic link", target)
	}
	if !info.IsDir() {
		return emptyResult(), fmt.Errorf("scan root %q is not a directory", target)
	}
	root, err := os.OpenRoot(target)
	if err != nil {
		return emptyResult(), fmt.Errorf("open scan root %q: %w", target, err)
	}
	defer root.Close()
	opened, err := root.Lstat(".")
	if err != nil {
		return emptyResult(), fmt.Errorf("inspect opened scan root: %w", err)
	}
	if !os.SameFile(info, opened) {
		return emptyResult(), fmt.Errorf("scan root changed during discovery")
	}
	return scanRoot(root, limits)
}

func scanRoot(root rootFS, limits Limits) (Result, error) {
	s := scanner{root: root, limits: limits, result: emptyResult()}
	err := s.walk(".", 0)
	s.result.Complete = err == nil
	sort.Slice(s.result.Files, func(i, j int) bool { return s.result.Files[i].Path < s.result.Files[j].Path })
	sort.Slice(s.result.Skipped, func(i, j int) bool {
		if s.result.Skipped[i].Path != s.result.Skipped[j].Path {
			return s.result.Skipped[i].Path < s.result.Skipped[j].Path
		}
		return s.result.Skipped[i].Reason < s.result.Skipped[j].Reason
	})
	return s.result, err
}

func emptyResult() Result { return Result{Files: []File{}, Skipped: []Skip{}} }

func validateLimits(l Limits) error {
	if l.MaxFileBytes <= 0 || l.MaxTotalBytes <= 0 || l.MaxFiles <= 0 || l.MaxEntries <= 0 || l.MaxDepth <= 0 {
		return fmt.Errorf("all scan limits must be positive")
	}
	return nil
}

func (s *scanner) walk(dir string, depth int) error {
	if depth > s.limits.MaxDepth {
		return fmt.Errorf("directory %q exceeds maximum depth %d", dir, s.limits.MaxDepth)
	}
	f, err := s.root.OpenFile(dir, directoryOpenFlags(), 0)
	if err != nil {
		return fmt.Errorf("open directory %q: %w", dir, err)
	}
	info, err := f.Stat()
	if err != nil || !info.IsDir() {
		f.Close()
		if err != nil {
			return fmt.Errorf("inspect opened directory %q: %w", dir, err)
		}
		return fmt.Errorf("directory %q changed to a non-directory", dir)
	}
	// Read incrementally and stop after one entry beyond the global budget.
	// A directory with millions of entries cannot cause unbounded allocation.
	entries := []fs.DirEntry{}
	for {
		remaining := s.limits.MaxEntries - s.result.EntriesVisited
		n := 128
		if remaining < n {
			n = remaining + 1
		}
		batch, readErr := f.ReadDir(n)
		for _, entry := range batch {
			if s.result.EntriesVisited == s.limits.MaxEntries {
				f.Close()
				return fmt.Errorf("directory %q exceeds maximum visited entries %d", dir, s.limits.MaxEntries)
			}
			s.result.EntriesVisited++
			entries = append(entries, entry)
		}
		if readErr != nil {
			if readErr != io.EOF {
				f.Close()
				return fmt.Errorf("read directory %q: %w", dir, readErr)
			}
			break
		}
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close directory %q: %w", dir, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		name := path.Join(dir, entry.Name())
		info, err := s.root.Lstat(name)
		if err != nil {
			return fmt.Errorf("inspect entry %q: %w", name, err)
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			s.skip(name, "symlink")
		case info.IsDir():
			if ignoredDirectories[entry.Name()] {
				s.skip(name, "ignored_directory")
				continue
			}
			if err := s.walk(name, depth+1); err != nil {
				return err
			}
		case !info.Mode().IsRegular():
			s.skip(name, "non_regular")
		default:
			kind := fileKind(entry.Name())
			if kind == "" {
				continue
			}
			if strings.HasSuffix(entry.Name(), ".min.js") {
				s.skip(name, "generated")
				continue
			}
			if err := s.readFile(name, kind, info); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *scanner) readFile(name, kind string, before fs.FileInfo) error {
	if s.candidateFiles == s.limits.MaxFiles {
		return fmt.Errorf("file %q exceeds maximum selected files %d", name, s.limits.MaxFiles)
	}
	s.candidateFiles++
	if before.Size() > s.limits.MaxFileBytes {
		return fmt.Errorf("file %q exceeds maximum file bytes %d", name, s.limits.MaxFileBytes)
	}
	remaining := s.limits.MaxTotalBytes - s.result.BytesRead
	if before.Size() > remaining {
		return fmt.Errorf("file %q exceeds maximum total bytes %d", name, s.limits.MaxTotalBytes)
	}
	f, err := s.root.OpenFile(name, sourceOpenFlags(), 0)
	if err != nil {
		return fmt.Errorf("open file %q: %w", name, err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("inspect opened file %q: %w", name, err)
	}
	if !info.Mode().IsRegular() || !os.SameFile(before, info) {
		return fmt.Errorf("file %q changed during discovery", name)
	}
	limit := min(s.limits.MaxFileBytes, remaining)
	data, err := io.ReadAll(io.LimitReader(f, limit))
	s.result.BytesRead += int64(len(data))
	if err != nil {
		return fmt.Errorf("read file %q: %w", name, err)
	}
	// The extra byte detects growth since Stat without overflowing limit+1.
	var extra [1]byte
	n, err := f.Read(extra[:])
	s.result.BytesRead += int64(n)
	if n > 0 {
		return fmt.Errorf("file %q grew beyond file or total byte limit", name)
	}
	if err != nil && err != io.EOF {
		return fmt.Errorf("finish reading file %q: %w", name, err)
	}
	if bytes.IndexByte(data, 0) >= 0 {
		s.skip(name, "binary")
		return nil
	}
	if kind == "go_source" && generatedGo(data) {
		s.skip(name, "generated")
		return nil
	}
	s.result.Files = append(s.result.Files, File{Path: name, Kind: kind, Size: int64(len(data)), Data: data})
	return nil
}

func (s *scanner) skip(name, reason string) {
	s.result.Skipped = append(s.result.Skipped, Skip{Path: name, Reason: reason})
}

func fileKind(name string) string {
	switch name {
	case "go.mod":
		return "go_module"
	case "go.sum":
		return "go_checksum"
	case "package.json":
		return "js_manifest"
	case "package-lock.json", "npm-shrinkwrap.json":
		return "npm_lock"
	case "yarn.lock":
		return "yarn_lock"
	case "pnpm-lock.yaml":
		return "pnpm_lock"
	}
	switch path.Ext(name) {
	case ".go":
		return "go_source"
	case ".js", ".jsx":
		return "js_source"
	case ".ts", ".tsx":
		return "ts_source"
	}
	return ""
}

func generatedGo(data []byte) bool {
	// The generated-code convention requires an exact marker line before the
	// first non-comment, nonblank text (https://go.dev/s/generatedcode). Lexing
	// the header handles package tabs, block comments, and malformed leading
	// tokens without mistaking markers in the program body for generated headers.
	f := token.NewFileSet().AddFile("", -1, len(data))
	var lex goscanner.Scanner
	invalid := false
	lex.Init(f, data, func(token.Position, string) { invalid = true }, goscanner.ScanComments)
	for {
		pos, tok, text := lex.Scan()
		if invalid || tok != token.COMMENT {
			return false
		}
		firstLine := true
		for line := range strings.SplitSeq(text, "\n") {
			// A first-line comment token may start after indentation; the exact
			// marker regexp requires // at the beginning of the physical line.
			atLineStart := !firstLine || f.Position(pos).Column == 1
			firstLine = false
			if atLineStart {
				if rest, ok := strings.CutPrefix(line, "// Code generated "); ok {
					if _, ok := strings.CutSuffix(rest, " DO NOT EDIT."); ok {
						return true
					}
				}
			}
		}
	}
}
