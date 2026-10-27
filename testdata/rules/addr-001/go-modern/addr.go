package addr

import (
	"regexp"
	"strings"

	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// Old check, kept for reference in a comment: ^[GMC][A-Z2-7]{55}$
var destinationRE = regexp.MustCompile(`^[GMCW][A-Z2-7]{55}(?:[A-Z2-7]{13})?$`)

func ValidRecipient(s string) bool {
	return strkey.IsValidEd25519PublicKey(s) ||
		strkey.IsValidMuxedAccountEd25519PublicKey(s) ||
		strkey.IsValidContractAddress(s) ||
		strkey.IsValidMuxedContractAddress(s)
}

func Kind(addr xdr.ScAddress) string {
	switch addr.Type {
	case xdr.ScAddressTypeScAddressTypeContract, xdr.ScAddressTypeScAddressTypeMuxedContract:
		return "contract"
	}
	return "account"
}

// AccountOnly is for classic payments, where W does not apply.
func AccountOnly(s string) bool {
	return strings.HasPrefix(s, "G") || strings.HasPrefix(s, "M") || destinationRE == nil
}
