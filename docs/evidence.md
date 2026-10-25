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
