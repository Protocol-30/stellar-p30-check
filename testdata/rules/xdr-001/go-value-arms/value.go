package ledgers

import "github.com/stellar/go-stellar-sdk/xdr"

// Kind names the StellarValue arm of a header.
func Kind(v xdr.StellarValue) string {
	switch v.Ext.V {
	case xdr.StellarValueTypeStellarValueBasic:
		return "basic"
	case xdr.StellarValueTypeStellarValueSigned:
		return "signed"
	case xdr.StellarValueTypeStellarValueEmptyTxSet:
		return "empty"
	}
	return ""
}

// IsSigned reports whether the header carries a close value signature.
func IsSigned(v xdr.StellarValue) bool {
	return v.Ext.V == xdr.StellarValueTypeStellarValueSigned
}

// Signer returns the node that signed the close value.
func Signer(v xdr.StellarValue) (xdr.NodeId, bool) {
	sig, ok := v.Ext.GetLcValueSignature()
	return sig.NodeId, ok
}

// MustSigner panics when the close value is unsigned.
func MustSigner(v xdr.StellarValue) xdr.NodeId {
	return v.Ext.MustLcValueSignature().NodeId
}
