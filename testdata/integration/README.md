# Integration fixtures

Two small invented projects, each with a Go indexer and a TypeScript web
app, scanned end to end through the CLI by `internal/cli/cli_test.go`.

- `legacy-app` uses go-stellar-sdk v0.7.3 and @stellar/stellar-sdk 16.3.1,
  an address check without `W`, `MustLcValueSignature`, and a division by a
  close-time difference. All four rules report; `--fail-on high` exits 1.
- `modern-app` is the same project after the Protocol 30 fixes. No rule
  reports anything and `--fail-on high` exits 0.

Full reports: `testdata/golden/integration-*.json`.
