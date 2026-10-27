package prefix

import "strings"

// IsAccountOrContract accepts G and C addresses.
func IsAccountOrContract(s string) bool {
	return len(s) == 56 && (strings.HasPrefix(s, "G") || s[0] == 'C')
}
