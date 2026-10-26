# Changelog

## Unreleased — M4 ledger analysis

- Scoped direct raw-ledger decoding review and low-confidence typed whole-second timing clues.
- Pinned CAP-88 schema excerpts, paired legacy/modern source fixtures and integration checks.
- Schema2 retained; four available rules. No runtime compatibility certification.

## Unreleased — M3 narrow W-address analysis

- Add P30-ADDR-001 for supported same-function W-rejecting guards before specific imported W-only helpers, with scoped SDK evidence and conservative confidence.
- Add high/high JS findings only with verified nominal W-capable selection; keep unresolved Go/JS contexts medium/medium and broader SAC destination-policy hints medium/low.
- Parse Go/JS/TS/JSX/TSX syntax without executing targets; preserve account-only/C-only/storage-key negatives, shadowing/mutation/bypass exclusions, and per-file source limitations.
- Add parser timeout/node budgets, paired source fixtures, real gated exit-1 checks, and schema-v2 integration tests.
- Pin official Tree-sitter bindings/compatible grammars. CGO_ENABLED=1 and a C compiler are now required for normal builds, not just race tests; no pure-Go fallback is provided.

## M2 dependencies and capability signals

- Add static Go requirement/replacement/workspace inventory without claiming effective MVS selection; checksum history is not selected-version proof.
- Add JS manifests and supported npm, Yarn Classic, and partial pnpm lock parsing, with explicit provenance/constraint uncertainty.
- Add the exact-version official SDK capability matrix and conservative P30-DEP-001 medium-severity findings.
- Add schema version 2 with dependency inventory and warnings; retain the v1 schema as historical documentation.
- Pin Go module/YAML parsers and checksums, extend tests/CI, and document unsupported inputs and M3 scope.

## M0/M1 foundation

- Add the offline `p30check scan` command, bounded rooted discovery, and file inventory.
- Add versioned text/JSON reports, explicit analysis limitations, deterministic sorting, and policy-controlled exit codes.
- Add discovery/reporting tests, a Go checks workflow, and official evidence/verification limits.

No published release artifacts, reusable action, compatibility certification, or network activation claims are included.
