package addr

import sk "github.com/stellar/go-stellar-sdk/strkey"

// ValidRecipient accepts accounts, muxed accounts and contracts.
func ValidRecipient(s string) bool {
	return sk.IsValidEd25519PublicKey(s) ||
		sk.IsValidMuxedAccountEd25519PublicKey(s) ||
		sk.IsValidContractAddress(s)
}
