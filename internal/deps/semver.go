// Package deps reads Go and npm dependency manifests and holds the sourced
// Protocol 30 capability matrix for official Stellar SDK packages. It parses
// text only and never runs a package manager.
package deps

import (
	"strconv"
	"strings"
)

// Version is a parsed semantic version. Build metadata is dropped because
// it does not affect ordering.
type Version struct {
	Major, Minor, Patch int
	Pre                 []string // dot-separated pre-release identifiers
}

// ParseVersion parses "1.2.3", "v1.2.3", "1.2.3-rc.1" and Go pseudo-versions
// such as "v0.8.1-0.20261021000000-abcdef123456". It rejects partial
// versions like "17" or "17.3".
func ParseVersion(s string) (Version, bool) {
	s = strings.TrimPrefix(s, "v")
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	core, pre, hasPre := strings.Cut(s, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return Version{}, false
	}
	var nums [3]int
	for i, p := range parts {
		n, ok := numeric(p)
		if !ok {
			return Version{}, false
		}
		nums[i] = n
	}
	v := Version{Major: nums[0], Minor: nums[1], Patch: nums[2]}
	if hasPre {
		if pre == "" {
			return Version{}, false
		}
		v.Pre = strings.Split(pre, ".")
		for _, id := range v.Pre {
			if id == "" {
				return Version{}, false
			}
		}
	}
	return v, true
}

// numeric parses a non-negative decimal without leading zeros.
func numeric(s string) (int, bool) {
	if s == "" || len(s) > 1 && s[0] == '0' {
		return 0, false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

// IsPrerelease reports whether v has pre-release identifiers. Go
// pseudo-versions are pre-releases.
func (v Version) IsPrerelease() bool { return len(v.Pre) > 0 }

// Compare returns -1, 0 or 1 following semver 2.0.0 precedence.
func (v Version) Compare(o Version) int {
	for _, d := range [][2]int{{v.Major, o.Major}, {v.Minor, o.Minor}, {v.Patch, o.Patch}} {
		if d[0] != d[1] {
			if d[0] < d[1] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(v.Pre) == 0 && len(o.Pre) == 0:
		return 0
	case len(v.Pre) == 0:
		return 1
	case len(o.Pre) == 0:
		return -1
	}
	for i := 0; i < len(v.Pre) && i < len(o.Pre); i++ {
		if c := compareIdent(v.Pre[i], o.Pre[i]); c != 0 {
			return c
		}
	}
	switch {
	case len(v.Pre) < len(o.Pre):
		return -1
	case len(v.Pre) > len(o.Pre):
		return 1
	}
	return 0
}

func compareIdent(a, b string) int {
	an, aNum := numeric(a)
	bn, bNum := numeric(b)
	switch {
	case aNum && bNum:
		switch {
		case an < bn:
			return -1
		case an > bn:
			return 1
		}
		return 0
	case aNum:
		return -1 // numeric identifiers sort before alphanumeric ones
	case bNum:
		return 1
	}
	return strings.Compare(a, b)
}

// Less reports whether v sorts before o.
func (v Version) Less(o Version) bool { return v.Compare(o) < 0 }

func (v Version) String() string {
	s := strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor) + "." + strconv.Itoa(v.Patch)
	if len(v.Pre) > 0 {
		s += "-" + strings.Join(v.Pre, ".")
	}
	return s
}

// must parses a version literal from the capability tables.
func must(s string) Version {
	v, ok := ParseVersion(s)
	if !ok {
		panic("deps: bad version literal " + s)
	}
	return v
}
