package addr

import (
	"example.com/fake/regexp"
	"example.com/fake/strkey"
)

// Same function names, different packages: not the Stellar SDK or the
// standard library, so nothing here is evidence.
var re = regexp.MustCompile(`^[GMC][A-Z2-7]{55}$`)

func Valid(s string) bool {
	return strkey.IsValidEd25519PublicKey(s) || strkey.IsValidContractAddress(s) || re != nil
}
