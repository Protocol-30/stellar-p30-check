# Changelog

## Unreleased

## v0.1.0 - 2026-10-27

First release. `p30check` is an offline, read-only scanner that flags Stellar
Protocol 30 compatibility risks in Go and JavaScript/TypeScript
repositories, with four rules: outdated SDK versions (P30-DEP-001), address
checks that reject CAP-84 `W` addresses (P30-ADDR-001), ledger decoding that
fails on CAP-88 ledgers (P30-XDR-001) and close-time assumptions
(P30-TS-001). Findings are static indications, not proof of breakage or of
compatibility; see
[limitations](https://github.com/Protocol-30/stellar-p30-check/blob/v0.1.0/docs/limitations.md)
and [rules](https://github.com/Protocol-30/stellar-p30-check/blob/v0.1.0/docs/rules.md).
Verify downloads against `SHA256SUMS`; see the
[CI guide](https://github.com/Protocol-30/stellar-p30-check/blob/v0.1.0/docs/ci-integration.md).

### Added
- Apache License 2.0 (`LICENSE`).
- Release tooling: `scripts/release-build.sh` builds reproducible archives
  for linux/darwin/windows on amd64/arm64 plus `SHA256SUMS`;
  `.github/workflows/release.yml` drafts a GitHub release on a `v*` tag; CI
  builds a snapshot of every target on each pull request.
- `p30check version` reports the module version for `go install` and VCS
  builds when no release version was linked in.
- docs/ci-integration.md and docs/release.md.
- Integration fixtures (`testdata/integration`), a malformed-input scan
  test, a whole-pipeline fuzzer (`FuzzCheckAll`), a self-scan test and
  docs/verification.md.
- P30-XDR-001: ledger decoding in projects whose resolved SDK has no
  Protocol 30 XDR, `StellarValueType` switches and comparisons without the
  CAP-88 `_MS` arms, single-arm `GetLcValueSignature` /
  `MustLcValueSignature`, and vendored `.x` files that predate CAP-88 or
  CAP-84. `.x` files are now discovered.
- P30-TS-001: division by, and comparison of, whole-second close times that
  CAP-88 allows to repeat between ledgers.
- P30-ADDR-001: finds address checks in Go and JS/TS that accept `C` with
  `G` or `M` but not CAP-84 `W` addresses (regexes, `StrKey`/`strkey`
  predicates, prefix checks and lists), and `SCAddress` / strkey version
  switches with no muxed-contract case. Go uses `go/parser`; JS/TS uses a
  tokenizer. New `source-unparsed` warning.
- P30-DEP-001: flags official Stellar SDK versions without Protocol 30
  support, read from `go.mod` (with `replace`), npm lockfiles (v1-v3,
  workspaces, aliases) and `package.json`. Capability table sourced from the
  published packages (docs/evidence.md E10-E14).
- Rules can return warnings; malformed manifests produce `manifest-unparsed`
  instead of failing the scan. New `rules-not-implemented` and
  `lockfile-not-parsed` warnings.
- docs/roadmap.md tracks progress by DripWave phase (replaces M0-M8 labels);
  docs/rules.md documents each rule.
- `p30check scan <path>` with `--format text|json`, `--fail-on none|high`,
  `--max-file-bytes`, `--max-files`, `--follow-symlinks`; `version` and
  `help` commands; exit codes 0/1/2.
- Root-bound file discovery for Go and JS/TS manifests, lockfiles and
  sources, with ignored directories, size/count/depth ceilings and symlink
  handling.
- Versioned report (`schema_version` "1") with text and JSON renderers,
  stable sorting and an injectable clock.
- Evidence log for CAP-84/87/88 and the official Go and JS SDKs.

### Fixed
- Six crafted inputs of 1 MB or less (deeply nested JS division, switches,
  comparisons and classes; Go functions with tens of thousands of
  comparisons or prefix checks) took over a minute to scan. The JS/TS
  matchers now use a precomputed bracket index and the Go scope checks are
  memoised; all six finish in under a second. Regression tests pin them.

### Changed
- Source rules run in one shared pass: each file is read and parsed once
  for all of them (about 2x faster on a 650-file Go tree), with identical
  results.
- Default `--max-file-bytes` raised from 2 MiB to 16 MiB so typical npm
  lockfiles are read.
