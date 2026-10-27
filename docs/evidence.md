# Evidence log

This file separates **verified facts** (with where and when they were
checked) from **design assumptions**. Rules may only encode verified facts.
When a fact changes, follow the update process at the bottom.

Last checked: **2026-10-25** (UTC), from a sandbox whose network proxy allowed
`raw.githubusercontent.com`, `proxy.golang.org` and `registry.npmjs.org`, but
refused `stellar.org`, `github.com` web pages and `go.dev`.

## Verified for Phases 1–5 (2026-10-25)

| # | Fact | Source (as fetched) | How it was checked |
|---|------|---------------------|--------------------|
| E1 | CAP-84 "Muxed Contract Addresses": status Implemented, protocol version 30. Adds `SC_ADDRESS_TYPE_MUXED_CONTRACT = 5` to `SCAddressType` (contract id + `uint64` id). Strkey version byte `22 << 3` = `0xb0`, prefix letter `W`, 40-byte payload; normative definition deferred to SEP-23. "Downstream consumers that parse `SCAddress` will need to handle the new variant and the new strkey type." | https://raw.githubusercontent.com/stellar/stellar-protocol/master/core/cap-0084.md | Read CAP text |
| E2 | CAP-88 "Millisecond-Resolution Close Times": status Implemented, protocol version 30. Adds `StellarValueType` arms `STELLAR_VALUE_SIGNED_MS = 3` and `STELLAR_VALUE_EMPTY_TX_SET_MS = 4`; after the upgrade the whole-second types are invalid. | https://raw.githubusercontent.com/stellar/stellar-protocol/master/core/cap-0088.md | Read CAP text |
| E3 | CAP-87 "Host functions for ML-DSA signature verification": status Implemented, protocol version 30. Adds host functions; no MVP rule targets it. | https://raw.githubusercontent.com/stellar/stellar-protocol/master/core/cap-0087.md | Read CAP preamble |
| E4 | Official Go SDK module path is `github.com/stellar/go-stellar-sdk`; latest version `v0.8.0`, published 2026-10-20T21:30:21Z, tag commit `921a5b16af9a72d973ab9ba666e96ad8bf2f6ce4`. | https://proxy.golang.org/github.com/stellar/go-stellar-sdk/@latest | Module proxy query |
| E5 | `go-stellar-sdk` v0.8.0's `go.mod` declares `go 1.25.0`. | https://proxy.golang.org/github.com/stellar/go-stellar-sdk/@v/v0.8.0.mod | Module proxy |
| E6 | `go-stellar-sdk` v0.8.0 CHANGELOG: "xdr: Protocol 30 support (CAP-0084, CAP-0087, CAP-0088)"; adds `ScAddressTypeScAddressTypeMuxedContract`, `StellarValueTypeStellarValueSignedMs`, `StellarValueTypeStellarValueEmptyTxSetMs`, and `strkey.VersionByteMuxedContract` (`W...`). It states every ledger after the P30 upgrade uses an `_MS` arm, "so earlier SDK versions cannot decode them", and that the generated `GetLcValueSignature()` only reads the `SIGNED` arm. | v0.8.0 module zip from proxy.golang.org (`CHANGELOG.md`, `strkey/main.go`) | Downloaded and grepped the module zip |
| E7 | npm `@stellar/stellar-sdk` dist-tags: `latest` = 17.4.0 (2026-10-25T11:30:46Z), `lts-16` = 16.4.1. 16.4.0 published 2026-10-21T20:44:03Z; 17.3.0 published 2026-10-21T19:14:25Z. `engines.node`: 17.3.0 `>=22.12.0`, 16.4.0 `>=22.0.0`. | https://registry.npmjs.org/@stellar/stellar-sdk | Registry metadata |
| E8 | In the published 16.4.0 tarball, the generated XDR includes `STELLAR_VALUE_SIGNED_MS` and `scAddressTypeMuxedContract`, but no `muxedContractFromStrkey` / `muxedContractToStrkey` helpers were found. In 17.3.0 all of these are present. | npm tarballs for 16.4.0 and 17.3.0 | Text search of `lib/` and `types/` (an observation, not a full capability audit) |
| E9 | Latest Go toolchains on the module proxy: go1.27.3 and go1.26.10, so the supported Go releases are 1.27 and 1.26. | https://proxy.golang.org/golang.org/toolchain/@v/list | Module proxy |

## Verified for Phase 8.2, the SDK version check (2026-10-27)

These rows back every version boundary in `internal/deps/capabilities.go`.
Markers searched for: Go — `StellarValueTypeStellarValueSignedMs` in
`xdr/xdr_generated.go` (CAP-88 XDR) and `VersionByteMuxedContract` in
`strkey/main.go` (`W` strkey); JS — `stellarValueSignedMs` (CAP-88 XDR),
`scAddressTypeMuxedContract` (CAP-84 XDR arm) and `encodeMuxedContract`
(`W` strkey) anywhere in the published package.

| # | Fact | Source (as fetched) | How it was checked |
|---|------|---------------------|--------------------|
| E10 | `go-stellar-sdk` releases: v0.1.0 (2025-12-11) through v0.7.3 (2026-08-20), then v0.8.0 (2026-10-20). v0.1.0, v0.5.0, v0.6.1, v0.7.0 and v0.7.3 contain neither marker; v0.8.0 contains both. Every release before v0.7.3 was published before CAP-88 was created (2026-08-21). | https://proxy.golang.org/github.com/stellar/go-stellar-sdk/@v/list and each `.info`/`.zip` | Downloaded module zips and searched |
| E11 | `github.com/stellar/go` README: "REPOSITORY DEPRECATED … moved to github.com/stellar/go-stellar-sdk". Its latest version `v0.0.0-20251210100531-aab2ea4aca88` (2025-12-10) contains neither marker. | https://proxy.golang.org/github.com/stellar/go/@latest and its zip | Downloaded and searched |
| E12 | `@stellar/stellar-sdk`: every release 15.0.0–16.3.1 and 17.0.0–17.2.1 has no marker; 16.4.0 and 16.4.1 have both XDR markers but not `encodeMuxedContract`; 17.3.0 and 17.4.0 have all three. All releases before 15.0.0 were published by 2026-03-04 (14.6.1), before CAP-84 (2026-06-24) and CAP-88 (2026-08-21) were created. Pre-releases: 16.0.0-rc.1/2, 17.0.0-rc.1/2 (not inspected). | https://registry.npmjs.org/@stellar/stellar-sdk and the tarballs of all 18 releases 15.0.0–17.4.0 | Downloaded and searched |
| E13 | `@stellar/stellar-base` latest is 15.0.0 (2026-03-30) and has no marker. `@stellar/stellar-sdk` 16.0.0 and 17.0.0 do not depend on `@stellar/stellar-base` (they bundle it). | https://registry.npmjs.org/@stellar/stellar-base, SDK `package.json` files | Registry metadata and tarball |
| E14 | Unscoped npm `stellar-sdk` (latest 13.3.0, 2025-04-21) and `stellar-base` (latest 13.1.0, 2025-03-28) are deprecated: "This package has moved to @stellar/…". | https://registry.npmjs.org/stellar-sdk, https://registry.npmjs.org/stellar-base | Registry metadata |

## Reported by the planning package, not re-verified here

`stellar.org` was unreachable from this session, so these come from the
planning package's `SOURCES.md` (checked 2026-10-25 by its author) and are
**not** re-verified:

- Bell/Protocol 30 Upgrade Guide (last updated 2026-10-23):
  https://stellar.org/blog/developers/bell-protocol-30-upgrade-guide —
  proposed Testnet vote 2026-11-03 17:00 UTC and Mainnet vote 2026-11-23
  17:00 UTC (votes, not guaranteed activation); JS SDK 17.3.0 supports P30
  and `W`; maintenance 16.4.0 decodes P30 ledgers but not `W` addresses; Go
  SDK v0.8.0 listed; app-specific validators must accept `W` even after an
  SDK update.
- Protocol 30 overview:
  https://stellar.org/blog/developers/introducing-bell-protocol-30-on-stellar
- Stellar Core v30.0.0 release: https://github.com/stellar/stellar-core/releases

E6 and E8 are consistent with the guide's SDK claims. The vote dates are only
used as text in report limitations, never in rule logic.

## Not yet verified (blocks the named rule)

| Open question | Blocks |
|---------------|--------|
| Final SEP-23 text for the muxed-contract strkey | P30-ADDR-001 remediation text |
| Protocol 30 support of npm pre-releases (16.0.0-rc.x, 17.0.0-rc.x) | Reported as "could not be verified" until checked |
| Community SDKs (Python, Rust, Java, Flutter, ...) | Out of scope for P30-DEP-001 |

## Design assumptions (not facts)

- Rule IDs, severities, confidence policy, CLI flags, exit codes, JSON field
  names and the ignored-directory list are this project's design choices.
- **Later releases keep a capability.** A version newer than the first one
  with a feature is assumed to have it (e.g. 17.5.0, v0.9.0, 16.4.2). Removing
  Protocol 30 support would be a breaking change; re-check when new majors
  ship. Exception: versions above the last known release of the deprecated
  or frozen packages (E13, E14) are reported as unverified.
- `go 1.25.0` in `go.mod` matches the Go SDK floor (E5) so the SDK can be
  imported later without raising it; CI tests on the supported releases (E9).
  The scanner currently imports nothing outside the standard library.

## Updating evidence

1. Fetch the primary source (CAP text, registry/proxy metadata, release
   notes) and record URL, date and method in the table above.
2. Change rule data only in the same commit as the evidence row it relies on.
3. Add or update a fixture that fails without the change.
