# Limitations

## What this build does

- Discovers Go and JavaScript/TypeScript manifests, lockfiles and sources
  under one directory and reports them, with every skipped entry and why.
- Produces a deterministic text or JSON report and a CI-friendly exit code.

## What this build does not do

- **It has no Protocol 30 rules yet.** It does not check SDK versions,
  `W` (CAP-84 muxed contract) address handling, or CAP-88 ledger-header
  decoding. Every report says so (`no-rules` and `check-not-implemented`
  warnings). An empty findings list from this build means nothing about
  compatibility.
- It never runs, builds or imports the scanned code, and makes no network or
  RPC calls. It cannot observe runtime behaviour.

## Limits that will remain once rules exist

- Static matches are indications. A high-confidence finding is strong
  evidence to review, not proof of breakage; no findings is not proof of
  compatibility.
- Only Go and JS/TS are in scope. Python, Rust, Java and other SDKs are not
  inspected.
- Code in ignored directories (`vendor`, `node_modules`, `dist`, ...),
  symlinked directories, and files over the size limit is not inspected.
  Vendored or bundled copies of an old SDK can therefore be missed.
- Generated files are recognised only by name (`*.min.js`, `*.d.ts`).
  Generated Go files with a `Code generated ... DO NOT EDIT.` header are
  still listed as sources.
- Protocol dates in reports (Testnet vote 2026-11-03, Mainnet vote
  2026-11-23, both 17:00 UTC) are proposed upgrade votes, not activation
  guarantees.

## Known false-positive / false-negative cases

To be filled in per rule as rules land, each with a fixture.
