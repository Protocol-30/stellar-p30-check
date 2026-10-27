package stats

import "github.com/stellar/go-stellar-sdk/xdr"

// OpsPerSecond divides by the time between two ledgers.
func OpsPerSecond(ops int, cur, prev xdr.LedgerHeader) float64 {
	return float64(ops) / float64(cur.ScpValue.CloseTime-prev.ScpValue.CloseTime)
}

type Ledger struct {
	ClosedAt        interface{ Sub(any) interface{ Seconds() float64 } }
	LedgerCloseTime int64
}

// Rate uses Horizon-style ClosedAt times.
func Rate(n float64, cur, prev Ledger) float64 {
	return n / cur.ClosedAt.Sub(prev.ClosedAt).Seconds()
}

// Ordered assumes close times strictly increase.
func Ordered(cur, prev Ledger) bool {
	return cur.LedgerCloseTime > prev.LedgerCloseTime
}
