package addr

import "github.com/stellar/go-stellar-sdk/strkey"

// Base strips the muxing id; W is handled in the same function.
func Base(s string) (string, bool) {
	if strkey.IsValidEd25519PublicKey(s) || strkey.IsValidMuxedAccountEd25519PublicKey(s) || strkey.IsValidContractAddress(s) {
		return s, true
	}
	if strkey.IsValidMuxedContractAddress(s) {
		return s[:1], true
	}
	return "", false
}
