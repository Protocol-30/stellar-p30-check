package events

import "github.com/stellar/go-stellar-sdk/xdr"

// Recipient returns a printable recipient for a transfer event.
func Recipient(addr xdr.ScAddress) (out string) {
	switch addr.Type {
	case xdr.ScAddressTypeScAddressTypeAccount:
		out = addr.AccountId.Address()
	case xdr.ScAddressTypeScAddressTypeContract:
		out = "contract"
	}
	return out
}
