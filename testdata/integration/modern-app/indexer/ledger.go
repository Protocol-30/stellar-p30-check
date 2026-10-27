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

// Signer returns the validator that signed the close value, from any arm.
func Signer(h xdr.LedgerHeader) (xdr.NodeId, bool) {
	sig, ok := h.ScpValue.Ext.LedgerCloseValueSignature()
	return sig.NodeId, ok
}

// TxPerLedger divides by the ledger sequence difference, which is never 0
// for distinct ledgers.
func TxPerLedger(txs int, cur, prev xdr.LedgerHeader) float64 {
	return float64(txs) / float64(cur.LedgerSeq-prev.LedgerSeq)
}

// ValidRecipient accepts accounts, muxed accounts, contracts and muxed contracts.
func ValidRecipient(s string) bool {
	return strkey.IsValidEd25519PublicKey(s) ||
		strkey.IsValidMuxedAccountEd25519PublicKey(s) ||
		strkey.IsValidContractAddress(s) ||
		strkey.IsValidMuxedContractAddress(s)
}
