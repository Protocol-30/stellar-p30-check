# Limitations

## What this build does

- Discovers Go and JavaScript/TypeScript manifests, lockfiles and sources
  under one directory and reports them, with every skipped entry and why.
- Produces a deterministic text or JSON report and a CI-friendly exit code.
- Flags official Stellar SDK versions without Protocol 30 support
  (P30-DEP-001; see [rules.md](rules.md)).

## What this build does not do

- **Only the SDK version check (P30-DEP-001) exists.** It does not yet check
  your own `W` (CAP-84 muxed contract) address handling or CAP-88
  ledger-header decoding code. Every report says so
  (`rules-not-implemented` warning). An empty findings list from this build
  means only that no outdated official SDK version was found.
- `yarn.lock` and `pnpm-lock.yaml` are not read; ranges in `package.json`
  are judged at their lowest allowed version, with low confidence.
- It never runs, builds or imports the scanned code, and makes no network or
  RPC calls. It cannot observe runtime behaviour.

## Limits that will remain once rules exist

- Static matches are indications. A high-confidence finding is strong
  evidence to review, not proof of breakage; no findings is not proof of
  compatibility.
- Only Go and JS/TS are in scope. Python, Rust, Java and other SDKs are not
  inspected.
- Code in ignored directories (`vendor`, `node_modules`, `dist`, ...),
  symlinked directories, and files over the size limit (16 MiB by default)
  is not inspected. Vendored or bundled copies of an old SDK can therefore be missed.
- Generated files are recognised only by name (`*.min.js`, `*.d.ts`).
  Generated Go files with a `Code generated ... DO NOT EDIT.` header are
  still listed as sources.
- Protocol dates in reports (Testnet vote 2026-11-03, Mainnet vote
  2026-11-23, both 17:00 UTC) are proposed upgrade votes, not activation
  guarantees.

## Known false-positive / false-negative cases

Listed per rule in [rules.md](rules.md).
