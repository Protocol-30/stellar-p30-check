# P30-TS-001 fixtures

Invented code using the close-time field names from go-stellar-sdk v0.8.0
(`CloseTime`, `LedgerCloseTime`, `ClosedAt`) and their JS equivalents.
Expected results are in `internal/rules/closetime_test.go`.

| Directory | Case | Expected |
|-----------|------|----------|
| go-rate | divide by `CloseTime` difference; divide by `ClosedAt.Sub(...).Seconds()`; `>` between two `LedgerCloseTime` | medium/medium, medium/medium, info/low |
| go-safe | one close time vs a deadline; sequence difference; difference without division; `CloseTimeMs` | none |
| js-rate | `ops / (cur.closeTime - prev.closeTime)`; `<=` between two close times | medium/medium, info/low |
| js-safe | one close time; sequence difference; a local variable named `closeTime` | none |
