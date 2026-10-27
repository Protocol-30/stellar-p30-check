package ingest

import "github.com/stellar/go-stellar-sdk/xdr"

// DecodeMeta decodes a base64 LedgerCloseMeta from RPC getLedgers.
func DecodeMeta(b64 string) (xdr.LedgerCloseMeta, error) {
	var meta xdr.LedgerCloseMeta
	err := xdr.SafeUnmarshalBase64(b64, &meta)
	return meta, err
}
