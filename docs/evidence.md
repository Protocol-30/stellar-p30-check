# Evidence and capability boundaries

Checked **October 25, 2026 (Africa/Lagos)**. This is an implementation evidence log, distinct from the supplied planning snapshot and original DripWave reference. External source verification is a development activity; ordinary scanner runs are offline.

| Subject | Evidence and result | Consequence |
|---|---|---|
| Go SDK identity/minimum | Tagged [v0.8.0 go.mod](https://github.com/stellar/go-stellar-sdk/blob/v0.8.0/go.mod) declares `github.com/stellar/go-stellar-sdk` and Go `1.25.0` | Verified SDK reference; the SDK is not a dependency of M1 |
| Go SDK P30 support | Tagged [v0.8.0 changelog](https://github.com/stellar/go-stellar-sdk/blob/v0.8.0/CHANGELOG.md) records CAP-84/87/88 and W support | Candidate capability evidence for M2, not a shipped rule |
| JS SDK 17.3.0 | Tagged [changelog](https://github.com/stellar/js-stellar-sdk/blob/v17.3.0/CHANGELOG.md) records P30 XDR and W helpers | Candidate supported capability combination |
| JS SDK 16.4.0 | Tagged [changelog](https://github.com/stellar/js-stellar-sdk/blob/v16.4.0/CHANGELOG.md) provides P30 XDR maintenance support while Address/scValToNative reject muxed contract addresses | Decode capability and high-level W support must be separate fields |
| CAP-84 scope | Official [CAP-84](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0084.md) defines muxed contract addresses, destination support and restrictions; tagged [transaction XDR](https://github.com/stellar/stellar-xdr/blob/v30.0/Stellar-transaction.x) retains G/M MuxedAccount fields | W rejection alone is not a compatibility defect |
| CAP-87 | Official [CAP-87](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0087.md) defines ML-DSA host functions | No broad existing-contract redeployment rule is justified here |
| CAP-88 | Official [CAP-88](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0088.md) defines millisecond close-time variants | Future raw-XDR checks must use real specification definitions |
| Development compiler | Official [Go download metadata](https://go.dev/dl/?mode=json) supplied Go 1.27.3 linux/amd64 and its archive SHA-256 | Pin 1.27.3 for prepared development and CI; no third-party module requirements in M1 |
| Effective Go dependencies | Official [Go modules reference](https://go.dev/ref/mod) documents minimal version selection, replacements, workspaces, and stale/multiple go.sum versions | Declared dependency text alone cannot establish an effective build version |

The official Go 1.27.3 linux/amd64 archive checksum from the fetched metadata is `65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40`. The environment setup verifies this digest before extracting the compiler. Consult the [Go release policy](https://go.dev/doc/devel/release) when refreshing the pinned toolchain rather than silently downloading an unpinned latest version.

## Verification limits

The supplied package cites the [Bell upgrade guide](https://stellar.org/blog/developers/bell-protocol-30-upgrade-guide) and [overview](https://stellar.org/blog/developers/introducing-bell-protocol-30-on-stellar), with a November 3 Testnet vote and November 23 Mainnet vote, both 17:00 UTC, and an October 23 guide update. Those blog requests returned proxy HTTP 403 during the review. Those dates remain supplied planning claims, not independently revalidated facts in this environment. They are votes, not guaranteed activations. Recheck the official source before schedule-dependent decisions.

No Drips approved-repository count, project uniqueness, program eligibility, model-account access, remote GitHub Actions result, or activation status was established by this build. No protocol fixtures have been captured from a live network.

## Evidence-change process

For a new SDK family/version or changed protocol claim, record the exact official URL (prefer immutable tag/commit), checked date, observed capability, and limits. Keep source facts separate from scanner heuristics. Add positive and negative fixture tests before extending the capability matrix; do not extrapolate community SDK thresholds or infer effective dependency versions from checksum history. When a source is blocked, record the operation and response and leave the claim unverified.

## M2 exact-release capability matrix

Rechecked **October 25, 2026 (Africa/Lagos)** against the official tagged artifacts below. Matrix version **`2026-10-25.1`** is embedded in each capability annotation. This is an exact-release table, not a minimum-version threshold. Other releases, prereleases, build-metadata variants, legacy package families, and forks remain unknown until independently checked.

| Ecosystem and canonical identity | Exact observed version | CAP-88 ledger XDR | Muxed-contract XDR | High-level W helpers | Tagged evidence |
|---|---|---|---|---|---|
| npm `@stellar/stellar-sdk` | `16.3.1` | Unsupported | Unsupported | Unsupported | [Generated XDR](https://github.com/stellar/js-stellar-sdk/blob/v16.3.1/src/base/generated/curr_generated.js), [Address](https://github.com/stellar/js-stellar-sdk/blob/v16.3.1/src/base/address.ts), [StrKey](https://github.com/stellar/js-stellar-sdk/blob/v16.3.1/src/base/strkey.ts), [package identity](https://github.com/stellar/js-stellar-sdk/blob/v16.3.1/package.json) |
| npm `@stellar/stellar-sdk` | `16.4.0` | Supported | Supported | Unsupported | [Changelog](https://github.com/stellar/js-stellar-sdk/blob/v16.4.0/CHANGELOG.md), [package identity](https://github.com/stellar/js-stellar-sdk/blob/v16.4.0/package.json) |
| npm `@stellar/stellar-sdk` | `17.3.0` | Supported | Supported | Supported | [Changelog](https://github.com/stellar/js-stellar-sdk/blob/v17.3.0/CHANGELOG.md), [package identity](https://github.com/stellar/js-stellar-sdk/blob/v17.3.0/package.json) |
| Go `github.com/stellar/go-stellar-sdk` | `v0.8.0` | Supported | Supported | Supported (strkey helpers) | [Changelog](https://github.com/stellar/go-stellar-sdk/blob/v0.8.0/CHANGELOG.md), [module identity](https://github.com/stellar/go-stellar-sdk/blob/v0.8.0/go.mod) |

The exact JS 16.3.1 generated schema has no CAP-88 millisecond `StellarValue` arms and no muxed-contract `ScAddress` arm. Its tagged Address/StrKey implementations have no W-address handling. This direct source evidence supports that exact negative row; it is not an assertion about every 16.3.x release. The 16.4.0 changelog explicitly distinguishes successful muxed-contract XDR decoding from higher-level helpers that throw on those addresses. The 17.3.0 and Go v0.8.0 changelogs explicitly add the relevant generated variants and W helpers.

Capability annotations describe the **observed version**, while `effective` and `uncertainty` describe whether that version is sufficiently resolved for the dependency rule. M2 does not reconstruct a complete Go MVS graph; Go minimums, replacements, and workspaces therefore cannot create a dependency finding. npm aliases/forks, unsupported lock formats, contradictory locks, unsupported constraints, and missing effective-version evidence likewise cannot create one.

`P30-DEP-001` emits one finding per adequately resolved dependency declaration/group with a verified missing capability. It combines missing capabilities into that finding and cites the declaration plus lock source. Findings are **medium severity / high confidence** in the dependency-capability evidence. M2 does not establish an affected runtime role, so it cannot justify high-severity application-impact gating. In particular, a 16.4.0 W-helper gap is guidance to review W-capable flows, not a claim that account-only validators should accept W or that all applications using that SDK are broken. Source-context checks remain later milestones.

The rule's positive/negative dependency observations are stored in `internal/rules/testdata/capabilities.json`; they are static fixture declarations, not fabricated network/XDR payloads. Ordinary scans use the embedded table offline and never fetch these sources.
