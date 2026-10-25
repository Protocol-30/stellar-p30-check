package deps

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/Protocol-30/stellar-p30-check/internal/walker"
	"go.yaml.in/yaml/v3"
)

// parseYarnLock supports only the classic v1 format. Modern Yarn formats have
// different selection semantics and are explicitly not treated as v1 records.
func parseYarnLock(file walker.File) (*jsLock, error) {
	lock := &jsLock{Path: file.Path, Family: "yarn", Generation: 1}
	header := false
	for line := range bytes.SplitSeq(file.Data, []byte("\n")) {
		if bytes.Equal(bytes.TrimSuffix(line, []byte("\r")), []byte("# yarn lockfile v1")) {
			header = true
			break
		}
	}
	if !header {
		lock.Unsupported = "only Yarn classic lockfile v1 is supported"
		return lock, nil
	}
	entries := map[string]jsLocked{}
	var selectors []string
	var current jsLocked
	fields := map[string]bool{}
	finish := func() error {
		for _, selector := range selectors {
			if _, exists := entries[selector]; exists {
				return fmt.Errorf("duplicate Yarn selector")
			}
			value := current
			if value.Version == "" {
				value.Reasons = append(value.Reasons, "Yarn record has no version")
			}
			entries[selector] = value
		}
		return nil
	}
	scanner := bufio.NewScanner(bytes.NewReader(file.Data))
	// The walker already bounds total file bytes; do not impose Scanner's much
	// smaller implicit line limit on legitimate multi-selector lock entries.
	scanner.Buffer(make([]byte, 4096), max(len(file.Data)+1, 4096))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSuffix(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			if strings.HasPrefix(trimmed, "__metadata:") {
				lock.Unsupported = "modern Yarn metadata is not a classic v1 lockfile"
				return lock, nil
			}
			if err := finish(); err != nil {
				return nil, fmt.Errorf("parse %q line %d: %w", file.Path, lineNumber, err)
			}
			if !strings.HasSuffix(trimmed, ":") {
				return nil, fmt.Errorf("parse %q line %d: invalid Yarn selector record", file.Path, lineNumber)
			}
			var err error
			selectors, err = yarnSelectors(strings.TrimSuffix(trimmed, ":"))
			if err != nil {
				return nil, fmt.Errorf("parse %q line %d: %w", file.Path, lineNumber, err)
			}
			current = jsLocked{}
			fields = map[string]bool{}
			continue
		}
		if len(selectors) == 0 {
			return nil, fmt.Errorf("parse %q line %d: Yarn field without selector", file.Path, lineNumber)
		}
		// Classic records indent direct fields by two spaces. More-indented
		// dependency records are not direct dependency selections.
		if !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "   ") {
			continue
		}
		key, raw, ok := strings.Cut(trimmed, " ")
		if !ok || (key != "version" && key != "resolved") {
			continue
		}
		if fields[key] {
			return nil, fmt.Errorf("parse %q line %d: duplicate Yarn field %q", file.Path, lineNumber, key)
		}
		fields[key] = true
		value, err := yarnScalar(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("parse %q line %d: %w", file.Path, lineNumber, err)
		}
		if key == "version" {
			current.Version = value
		} else {
			current.Resolved = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse %q: %w", file.Path, err)
	}
	if err := finish(); err != nil {
		return nil, fmt.Errorf("parse %q: %w", file.Path, err)
	}
	lock.Resolve = func(scope, name, declared string) (jsLocked, bool) {
		// Classic Yarn selectors have no importer map. A descendant manifest
		// cannot be tied to this lock without separately verified workspace
		// membership, which this intentionally narrow parser does not infer.
		if path.Clean(scope) != path.Dir(file.Path) {
			return jsLocked{}, false
		}
		selected, ok := entries[name+"@"+declared]
		selected.Specifier = declared
		return selected, ok
	}
	return lock, nil
}

func yarnSelectors(text string) ([]string, error) {
	var result []string
	var start int
	quoted, escaped := false, false
	for i, c := range text {
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && quoted {
			escaped = true
			continue
		}
		if c == '"' {
			quoted = !quoted
		}
		if c == ',' && !quoted {
			value, err := yarnScalar(strings.TrimSpace(text[start:i]))
			if err != nil || value == "" {
				return nil, fmt.Errorf("invalid Yarn selector list")
			}
			result = append(result, value)
			start = i + 1
		}
	}
	if quoted || escaped {
		return nil, fmt.Errorf("unterminated quoted Yarn selector")
	}
	value, err := yarnScalar(strings.TrimSpace(text[start:]))
	if err != nil || value == "" {
		return nil, fmt.Errorf("invalid Yarn selector")
	}
	return append(result, value), nil
}

func yarnScalar(text string) (string, error) {
	if strings.HasPrefix(text, "\"") {
		value, err := strconv.Unquote(text)
		if err != nil {
			return "", fmt.Errorf("invalid quoted Yarn scalar: %w", err)
		}
		return value, nil
	}
	if strings.ContainsAny(text, " \t\"'") {
		return "", fmt.Errorf("unsupported Yarn scalar syntax")
	}
	return text, nil
}

// parsePNPMLock deliberately supports only v9 importers. Integrity-only package
// resolutions retain their observed version but no registry-origin assertion.
func parsePNPMLock(file walker.File) (*jsLock, error) {
	lock := &jsLock{Path: file.Path, Family: "pnpm", Generation: 9}
	var document yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(file.Data))
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("parse %q: %w", file.Path, err)
	}
	if err := validateLockYAML(&document, 0); err != nil {
		return nil, fmt.Errorf("parse %q: %w", file.Path, err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("parse %q: expected a single YAML document", file.Path)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("parse %q: expected a pnpm mapping", file.Path)
	}
	root := document.Content[0]
	if patched := yamlMappingValue(root, "patchedDependencies"); patched != nil {
		if patched.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("parse %q: pnpm patchedDependencies must be a mapping", file.Path)
		}
		if len(patched.Content) > 0 {
			lock.Reasons = append(lock.Reasons, "pnpm patched dependencies may differ from the official package release")
		}
	}
	version := yamlMappingValue(root, "lockfileVersion")
	if version == nil || version.Kind != yaml.ScalarNode || version.Value != "9.0" {
		lock.Unsupported = "only pnpm lockfileVersion 9.0 is supported"
		return lock, nil
	}
	importers := yamlMappingValue(root, "importers")
	packages := yamlMappingValue(root, "packages")
	if importers == nil || importers.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("parse %q: pnpm v9 importers must be a mapping", file.Path)
	}
	if packages != nil && packages.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("parse %q: pnpm v9 packages must be a mapping", file.Path)
	}
	if packages != nil {
		for i := 1; i < len(packages.Content); i += 2 {
			pkg := packages.Content[i]
			if pkg.Kind != yaml.MappingNode {
				return nil, fmt.Errorf("parse %q: pnpm package entries must be mappings", file.Path)
			}
			if resolution := yamlMappingValue(pkg, "resolution"); resolution != nil {
				if resolution.Kind != yaml.MappingNode {
					return nil, fmt.Errorf("parse %q: pnpm package resolution must be a mapping", file.Path)
				}
				if tarball := yamlMappingValue(resolution, "tarball"); tarball != nil && (tarball.Kind != yaml.ScalarNode || tarball.Tag != "!!str") {
					return nil, fmt.Errorf("parse %q: pnpm package tarball must be a string", file.Path)
				}
			}
		}
	}
	// Validate importer dependency shapes now, even if a target manifest does not
	// happen to reference the malformed record.
	for i := 0; i < len(importers.Content); i += 2 {
		importer := importers.Content[i+1]
		if importer.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("parse %q: pnpm importer at line %d must be a mapping", file.Path, importers.Content[i].Line)
		}
		for _, group := range []string{"dependencies", "devDependencies", "optionalDependencies"} {
			dependencies := yamlMappingValue(importer, group)
			if dependencies == nil {
				continue
			}
			if dependencies.Kind != yaml.MappingNode {
				return nil, fmt.Errorf("parse %q: pnpm %s must be a mapping", file.Path, group)
			}
			for j := 1; j < len(dependencies.Content); j += 2 {
				record := dependencies.Content[j]
				if record.Kind != yaml.MappingNode || !yamlStringField(record, "specifier") || !yamlStringField(record, "version") {
					return nil, fmt.Errorf("parse %q: pnpm dependency entries need scalar specifier/version", file.Path)
				}
			}
		}
	}
	lock.Resolve = func(scope, name, declared string) (jsLocked, bool) {
		if !lockScopeContains(file.Path, scope) {
			return jsLocked{}, false
		}
		dir := path.Dir(file.Path)
		importerName := scope
		if dir != "." {
			if scope == dir {
				importerName = "."
			} else {
				importerName = strings.TrimPrefix(scope, dir+"/")
			}
		}
		importer := yamlMappingValue(importers, importerName)
		if importer == nil {
			return jsLocked{}, false
		}
		var selected *yaml.Node
		duplicate := false
		for _, group := range []string{"dependencies", "devDependencies", "optionalDependencies"} {
			if record := yamlMappingValue(yamlMappingValue(importer, group), name); record != nil {
				if selected != nil {
					duplicate = true
				}
				selected = record
			}
		}
		if selected == nil {
			return jsLocked{}, false
		}
		version := yamlMappingValue(selected, "version").Value
		value := jsLocked{Version: version, Specifier: yamlMappingValue(selected, "specifier").Value}
		if duplicate {
			value.Reasons = append(value.Reasons, "pnpm dependency occurs in multiple importer groups")
		}
		if value.Specifier != declared {
			value.Reasons = append(value.Reasons, "pnpm importer specifier differs from package declaration")
		}
		if base, _, hasPeers := strings.Cut(version, "("); hasPeers {
			value.Version = base
			value.Reasons = append(value.Reasons, "pnpm peer-qualified resolution is not fully verified")
		}
		pkg := yamlMappingValue(packages, name+"@"+value.Version)
		if pkg == nil {
			value.Reasons = append(value.Reasons, "pnpm package resolution record is absent")
			return value, true
		}
		resolution := yamlMappingValue(pkg, "resolution")
		if resolution == nil || resolution.Kind != yaml.MappingNode {
			value.Reasons = append(value.Reasons, "pnpm package resolution is not a supported mapping")
			return value, true
		}
		if tarball := yamlMappingValue(resolution, "tarball"); tarball != nil && tarball.Kind == yaml.ScalarNode {
			value.Resolved = tarball.Value
		}
		if value.Resolved == "" {
			value.Reasons = append(value.Reasons, "pnpm integrity-only resolution does not establish registry origin")
		}
		return value, true
	}
	return lock, nil
}

func validateLockYAML(node *yaml.Node, depth int) error {
	if depth > 64 {
		return fmt.Errorf("YAML nesting exceeds supported depth 64")
	}
	if node.Kind == yaml.AliasNode {
		return fmt.Errorf("YAML aliases are not supported")
	}
	if node.Kind == yaml.MappingNode {
		seen := map[string]bool{}
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind != yaml.ScalarNode {
				return fmt.Errorf("YAML mapping keys must be scalar")
			}
			if seen[key.Value] {
				return fmt.Errorf("duplicate YAML key %q", key.Value)
			}
			seen[key.Value] = true
		}
	}
	for _, child := range node.Content {
		if err := validateLockYAML(child, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func yamlMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func yamlStringField(mapping *yaml.Node, key string) bool {
	value := yamlMappingValue(mapping, key)
	return value != nil && value.Kind == yaml.ScalarNode && value.Tag == "!!str"
}

func lockScopeContains(lockPath, scope string) bool {
	dir := path.Dir(lockPath)
	scope = path.Clean(scope)
	return scope != ".." && !strings.HasPrefix(scope, "../") && !path.IsAbs(scope) && (dir == "." || scope == dir || strings.HasPrefix(scope, dir+"/"))
}
