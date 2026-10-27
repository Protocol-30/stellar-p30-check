package indexer

import (
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// Decode reads one LedgerCloseMeta from RPC getLedgers.
func Decode(b64 string) (xdr.LedgerCloseMeta, error) {
	var meta xdr.LedgerCloseMeta
	err := xdr.SafeUnmarshalBase64(b64, &meta)
	return meta, err
}

// Signer returns the validator that signed the close value.
func Signer(h xdr.LedgerHeader) xdr.NodeId {
	return h.ScpValue.Ext.MustLcValueSignature().NodeId
}

// TxRate is transactions per second between two ledgers.
func TxRate(txs int, cur, prev xdr.LedgerHeader) float64 {
	return float64(txs) / float64(cur.ScpValue.CloseTime-prev.ScpValue.CloseTime)
}

// ValidRecipient accepts accounts, muxed accounts and contracts.
func ValidRecipient(s string) bool {
	return strkey.IsValidEd25519PublicKey(s) ||
		strkey.IsValidMuxedAccountEd25519PublicKey(s) ||
		strkey.IsValidContractAddress(s)
}
