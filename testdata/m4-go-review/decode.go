package review

import "github.com/stellar/go-stellar-sdk/xdr"

func decode(encoded string, header *xdr.LedgerHeader) error {
	return xdr.SafeUnmarshalBase64(encoded, header)
}
func ordered(previous, current xdr.LedgerHeader) bool {
	return current.ScpValue.CloseTime > previous.ScpValue.CloseTime
}
