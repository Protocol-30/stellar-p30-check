package stats

import "github.com/stellar/go-stellar-sdk/xdr"

// Expired compares one close time with a deadline, which CAP-88 does not change.
func Expired(h xdr.LedgerHeader, deadline xdr.TimePoint) bool {
	return h.ScpValue.CloseTime > deadline
}

// PerLedger divides by a ledger sequence difference.
func PerLedger(ops int, cur, prev xdr.LedgerHeader) int {
	return ops / int(cur.LedgerSeq-prev.LedgerSeq)
}

// Elapsed subtracts close times without dividing by the result.
func Elapsed(cur, prev xdr.LedgerHeader) xdr.TimePoint {
	return cur.ScpValue.CloseTime - prev.ScpValue.CloseTime
}

type msLedger struct{ CloseTimeMs uint64 }

// RateMs divides by millisecond close times, which strictly increase.
func RateMs(n uint64, cur, prev msLedger) uint64 {
	return n / (cur.CloseTimeMs - prev.CloseTimeMs)
}
