# P30-XDR-001 fixtures

Expected results are in `internal/rules/ledger_test.go`.

`xdr-defs-pre-cap88/` and `xdr-defs-curr/` hold `Stellar-ledger.x` and
`Stellar-contract.x` copied unmodified from
[stellar/stellar-xdr](https://github.com/stellar/stellar-xdr) (Apache License
2.0):

- `xdr-defs-pre-cap88/`: commit `bd7b53f57b6ccd23ca942bbb97c724208e3db90e`,
  the base that the CAP-88 XDR diff is written against. `StellarValueType`
  has no `_MS` arms; `SC_ADDRESS_TYPE_MUXED_CONTRACT` is behind
  `#ifdef CAP_0084_MUXED_CONTRACT`.
- `xdr-defs-curr/`: commit `579a90b2654e3811e8b672e128a5e094c7325b27` (`curr`,
  2026-10-27), where both are present without `#ifdef`.

The other projects are invented; SDK versions are real releases
(`docs/evidence.md` E10, E12).

| Directory | Case | Expected |
|-----------|------|----------|
| go-legacy-decoder | decodes `xdr.LedgerCloseMeta`, go-stellar-sdk v0.7.3 | high, high |
| go-modern-decoder | same code, v0.8.0 | none |
| go-no-sdk | same code, no go.mod | none (no SDK to judge) |
| js-legacy-decoder | `LedgerHeaderHistoryEntry.fromXDR`, exact 16.3.1 | high, high |
| js-range-decoder | `server.getLedgers()`, `^16.0.0`, no lockfile | high, low |
| js-xdr-only-decoder | exact 16.4.0 (decodes P30 ledgers) | none |
| js-workspace-decoder | root lock: 17.3.0 at the root, 16.2.0 for `packages/web` | high, high in `packages/web` only |
| go-value-arms | `StellarValueType` switch, `==` comparison, `GetLcValueSignature`, `MustLcValueSignature` | high/medium, medium/medium, medium/high, high/high |
| go-value-modern | `_MS` cases, `LedgerCloseValueSignature()`, per-arm reads | none |
| js-value-switch | switch on `stellarValueSigned` with `default` | medium, medium |
| xdr-defs-pre-cap88 | old `.x` files | high/high (ledger), medium/low (contract, `#ifdef`) |
| xdr-defs-curr | current `.x` files | none |
