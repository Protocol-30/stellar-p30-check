package ledgers

import "github.com/stellar/go-stellar-sdk/xdr"

// Kind names the StellarValue arm of a header, including CAP-88 arms.
func Kind(v xdr.StellarValue) string {
	switch v.Ext.V {
	case xdr.StellarValueTypeStellarValueSigned, xdr.StellarValueTypeStellarValueSignedMs:
		return "signed"
	case xdr.StellarValueTypeStellarValueEmptyTxSet, xdr.StellarValueTypeStellarValueEmptyTxSetMs:
		return "empty"
	default:
		return "basic"
	}
}

// Signer uses the v0.8.0 helper that reads every arm.
func Signer(v xdr.StellarValue) (xdr.NodeId, bool) {
	sig, ok := v.Ext.LedgerCloseValueSignature()
	return sig.NodeId, ok
}

// SignerByArm reads each arm itself, like the SDK's own helper.
func SignerByArm(e xdr.StellarValueExt) (xdr.LedgerCloseValueSignature, bool) {
	if sig, ok := e.GetLcValueSignature(); ok {
		return sig, true
	}
	if v, ok := e.GetSignedMsValue(); ok {
		return v.LcValueSignature, true
	}
	return xdr.LedgerCloseValueSignature{}, false
}

// MsSigner reads the signature of a specific arm.
func MsSigner(e xdr.StellarValueExt) xdr.NodeId {
	return e.MustSignedMsValue().LcValueSignature.NodeId
}
