package addr

import "regexp"

var destinationRE = regexp.MustCompile(`^[GMC][A-Z2-7]{55}$`)

// ValidDestination reports whether s looks like a payment destination.
func ValidDestination(s string) bool { return destinationRE.MatchString(s) }
