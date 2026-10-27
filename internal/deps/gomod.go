package deps

import (
	"fmt"
	"strconv"
	"strings"
)

// Requirement is one require directive in a go.mod file.
type Requirement struct {
	Path     string
	Version  string
	Line     int
	Indirect bool
}

// Replacement is one replace directive. OldVersion is empty when the
// directive replaces every version of the module.
type Replacement struct {
	Old, OldVersion string
	New, NewVersion string // NewVersion is empty for a local directory replacement
	Line            int
}

// GoMod holds the parts of a go.mod file the rules need.
type GoMod struct {
	Module   string
	Requires []Requirement
	Replaces []Replacement
}

// ReplacementFor returns the replace directive that applies to path at
// version, if any. A version-specific directive wins over a general one,
// matching the go command.
func (m GoMod) ReplacementFor(path, version string) (Replacement, bool) {
	var general *Replacement
	for i := range m.Replaces {
		r := &m.Replaces[i]
		if r.Old != path {
			continue
		}
		if r.OldVersion == version {
			return *r, true
		}
		if r.OldVersion == "" {
			general = r
		}
	}
	if general != nil {
		return *general, true
	}
	return Replacement{}, false
}

// ParseGoMod reads module, require and replace directives from go.mod
// text. Other directives are ignored. It follows the go.mod grammar closely
// enough for dependency inventory; it does not validate module paths.
func ParseGoMod(data []byte) (GoMod, error) {
	var m GoMod
	block := "" // directive name while inside "name ( ... )"
	for i, raw := range strings.Split(string(data), "\n") {
		lineNo := i + 1
		toks, comment, err := tokenize(raw)
		if err != nil {
			return GoMod{}, fmt.Errorf("go.mod line %d: %w", lineNo, err)
		}
		if len(toks) == 0 {
			continue
		}
		if block != "" {
			if len(toks) == 1 && toks[0] == ")" {
				block = ""
				continue
			}
			if err := m.directive(block, toks, comment, lineNo); err != nil {
				return GoMod{}, err
			}
			continue
		}
		if len(toks) == 2 && toks[1] == "(" {
			block = toks[0]
			continue
		}
		if err := m.directive(toks[0], toks[1:], comment, lineNo); err != nil {
			return GoMod{}, err
		}
	}
	if block != "" {
		return GoMod{}, fmt.Errorf("go.mod: unterminated %s block", block)
	}
	return m, nil
}

func (m *GoMod) directive(verb string, args []string, comment string, line int) error {
	switch verb {
	case "module":
		if len(args) != 1 {
			return fmt.Errorf("go.mod line %d: malformed module directive", line)
		}
		m.Module = args[0]
	case "require":
		if len(args) != 2 {
			return fmt.Errorf("go.mod line %d: malformed require directive", line)
		}
		m.Requires = append(m.Requires, Requirement{
			Path: args[0], Version: args[1], Line: line,
			Indirect: strings.Contains(comment, "indirect"),
		})
	case "replace":
		r := Replacement{Line: line}
		arrow := -1
		for i, a := range args {
			if a == "=>" {
				arrow = i
			}
		}
		left, right := args, []string(nil)
		if arrow >= 0 {
			left, right = args[:arrow], args[arrow+1:]
		}
		if arrow < 0 || len(left) < 1 || len(left) > 2 || len(right) < 1 || len(right) > 2 {
			return fmt.Errorf("go.mod line %d: malformed replace directive", line)
		}
		r.Old = left[0]
		if len(left) == 2 {
			r.OldVersion = left[1]
		}
		r.New = right[0]
		if len(right) == 2 {
			r.NewVersion = right[1]
		}
		m.Replaces = append(m.Replaces, r)
	}
	return nil
}

// tokenize splits a go.mod line into tokens and its trailing // comment.
// Tokens may be double-quoted or back-quoted strings.
func tokenize(line string) (toks []string, comment string, err error) {
	s := strings.TrimSpace(line)
	for s != "" {
		switch {
		case strings.HasPrefix(s, "//"):
			return toks, s[2:], nil
		case s[0] == '"':
			end := 1
			for end < len(s) && s[end] != '"' {
				if s[end] == '\\' {
					end++
				}
				end++
			}
			if end >= len(s) {
				return nil, "", fmt.Errorf("unterminated string")
			}
			u, err := strconv.Unquote(s[:end+1])
			if err != nil {
				return nil, "", fmt.Errorf("bad string %s", s[:end+1])
			}
			toks = append(toks, u)
			s = s[end+1:]
		case s[0] == '`':
			end := strings.IndexByte(s[1:], '`')
			if end < 0 {
				return nil, "", fmt.Errorf("unterminated string")
			}
			toks = append(toks, s[1:end+1])
			s = s[end+2:]
		case s[0] == '(' || s[0] == ')':
			toks = append(toks, s[:1])
			s = s[1:]
		default:
			end := strings.IndexAny(s, " \t\"`()")
			if i := strings.Index(s, "//"); i >= 0 && (end < 0 || i < end) {
				end = i
			}
			if end < 0 {
				end = len(s)
			}
			toks = append(toks, s[:end])
			s = s[end:]
		}
		s = strings.TrimLeft(s, " \t\r")
	}
	return toks, "", nil
}
