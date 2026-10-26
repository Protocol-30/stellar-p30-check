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

## M3 source-address evidence

Checked **October 26, 2026 (Africa/Lagos)** by reading the official SDK and protocol sources below. The SDK release tags were resolved with read-only Git operations: JS SDK `v17.3.0` is commit `0f2fd40ba6db761390e3e5e4eb5c432234696a61`; Go SDK `v0.8.0` is commit `921a5b16af9a72d973ab9ba666e96ad8bf2f6ce4`. The checked Stellar protocol snapshot is commit `1ae39233055a0fb94bb737650a39ede8d77d924c`; the Protocol 30 XDR tag `v30.0` resolves to `4f524bbac80c781c06e4fb5fc93a2d0d331d6705`. These immutable links record the evidence actually inspected, rather than assuming that a mutable branch remains unchanged.

| Verified source fact | Immutable evidence | M3 interpretation boundary |
|---|---|---|
| JS `StrKey.decodeMuxedContract(address: string): Uint8Array` decodes a muxed-contract W address; `StrKey.isValidMuxedContract(address: string): boolean` validates one | [StrKey methods](https://github.com/stellar/js-stellar-sdk/blob/0f2fd40ba6db761390e3e5e4eb5c432234696a61/src/base/strkey.ts#L399) | A restrictive guard on the same input before a verified W-only use is a candidate contradiction. The name appearing in an unrelated string, comment, or shadowed binding is not proof of that use. |
| JS `TransactionBuilder.addSacTransferOperation` is an **instance** method whose **first positional argument** is the destination string; its signature is `(destination: string, asset: Asset, amount: bigint \| string, sorobanFees?: SorobanFees): TransactionBuilder` | [Builder declaration and W validation](https://github.com/stellar/js-stellar-sdk/blob/0f2fd40ba6db761390e3e5e4eb5c432234696a61/src/base/transaction_builder.ts#L771) | The destination is G/M/C/W-capable. A static call on the class or an object property named `destination` does not match this API. Identifying a builder instance and its first argument requires separate source evidence. |
| That builder method passes the destination to `nativeToScVal(destination, { type: "address" })` as the SAC `transfer` destination | [Transfer construction](https://github.com/stellar/js-stellar-sdk/blob/0f2fd40ba6db761390e3e5e4eb5c432234696a61/src/base/transaction_builder.ts#L836) | The specific transfer flow establishes the supported destination domain; generic address conversion alone does not establish the caller's domain. |
| SDK root exports the base module, which exports `StrKey` and `TransactionBuilder` | [Root exports](https://github.com/stellar/js-stellar-sdk/blob/0f2fd40ba6db761390e3e5e4eb5c432234696a61/src/index.ts), [base exports](https://github.com/stellar/js-stellar-sdk/blob/0f2fd40ba6db761390e3e5e4eb5c432234696a61/src/base/index.ts) | Named imports, aliases of named imports, and namespace imports are legitimate source forms. Their lexical bindings still need to be distinguished from local replacements or shadowing. |
| Go `strkey.DecodeMuxedContract(address string) (*MuxedContract, error)` decodes W; `IsValidMuxedContractAddress(string)` checks W; `Decode(VersionByteMuxedContract, input)` explicitly selects its version byte | [Muxed-contract decoder](https://github.com/stellar/go-stellar-sdk/blob/921a5b16af9a72d973ab9ba666e96ad8bf2f6ce4/strkey/muxed_contract.go#L54), [version/decoder/validator](https://github.com/stellar/go-stellar-sdk/blob/921a5b16af9a72d973ab9ba666e96ad8bf2f6ce4/strkey/main.go) | These identify W-specific source contexts in that official release. A source import does not resolve the target's actual module graph or transfer capabilities to a fork or local replacement. |
| Go `(*MuxedContract).SetContractID(address)` expects the underlying **C-address**, using `Decode(VersionByteContract, address)` | [SetContractID implementation](https://github.com/stellar/go-stellar-sdk/blob/921a5b16af9a72d973ab9ba666e96ad8bf2f6ce4/strkey/muxed_contract.go#L19) | A receiver or variable name containing `MuxedContract` is insufficient W-domain evidence. A C-only guard on this API is a valid negative case. |
| Classic payment destinations and transaction sources use `MuxedAccount`, with ED25519 and muxed-ED25519 arms | [MuxedAccount](https://github.com/stellar/stellar-xdr/blob/4f524bbac80c781c06e4fb5fc93a2d0d331d6705/Stellar-transaction.x#L21), [PaymentOp](https://github.com/stellar/stellar-xdr/blob/4f524bbac80c781c06e4fb5fc93a2d0d331d6705/Stellar-transaction.x#L92), [transaction source](https://github.com/stellar/stellar-xdr/blob/4f524bbac80c781c06e4fb5fc93a2d0d331d6705/Stellar-transaction.x#L939) | G/M-only validation remains appropriate for those account fields. W omission is not a universal address-validation defect. |
| `AddressObject` inputs reject muxed-contract values, and W is prohibited in contract storage keys, including nested containers and instance storage | [CAP-84 semantics](https://github.com/stellar/stellar-protocol/blob/1ae39233055a0fb94bb737650a39ede8d77d924c/core/cap-0084.md#semantics) | These are required negative domains. Generic `Address.fromString` or `nativeToScVal` usage alone must not override the field's documented restrictions. |

The checked CAP-84 enables muxed-contract addresses as the **destination of SAC `transfer`**, not the source. Its `mint` event discussion covers the issuer-as-source transfer case; it does not enable arbitrary direct `mint` or `transfer_from` destinations. Those extensions are explicitly deferred in [Destination-only support and Limiting scope to transfer](https://github.com/stellar/stellar-protocol/blob/1ae39233055a0fb94bb737650a39ede8d77d924c/core/cap-0084.md#destination-only-support). A changelog mentioning a W-bearing Horizon mint event is therefore not sufficient evidence for expanding the direct-call API domain.

### Normative address-vector provenance

[SEP-23 at the checked protocol commit](https://github.com/stellar/stellar-protocol/blob/1ae39233055a0fb94bb737650a39ede8d77d924c/ecosystem/sep-0023.md#L251) provides this valid muxed-contract vector with id `0`:

```text
WA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUAAAAAAAAAAAAAWWC
```

Its underlying contract address is `CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA`. The same specification assigns `STRKEY_MUXED_CONTRACT` the value `22 << 3` and prefix `W`; the checked SDK accepts the 69-character strkey form. This is a published specification vector, not an address captured from a network ledger, and its use does not establish activation or network behavior.

### Parser provenance and compatibility

M3 selects the official Go Tree-sitter binding/runtime `github.com/tree-sitter/go-tree-sitter v0.24.0`, JavaScript grammar `github.com/tree-sitter/tree-sitter-javascript v0.23.1`, and TypeScript/TSX grammars `github.com/tree-sitter/tree-sitter-typescript v0.23.2`. The selected [runtime API header](https://github.com/tree-sitter/go-tree-sitter/blob/v0.24.0/include/tree_sitter/api.h#L29) accepts language ABI versions 13–14. The checked generated [JavaScript parser](https://github.com/tree-sitter/tree-sitter-javascript/blob/v0.23.1/src/parser.c#L7), [TypeScript parser](https://github.com/tree-sitter/tree-sitter-typescript/blob/v0.23.2/typescript/src/parser.c#L7), and [TSX parser](https://github.com/tree-sitter/tree-sitter-typescript/blob/v0.23.2/tsx/src/parser.c#L7) declare ABI 14.

The newer [JavaScript grammar v0.25.0](https://github.com/tree-sitter/tree-sitter-javascript/blob/v0.25.0/src/parser.c#L9) declares ABI 15 and was not selected because it exceeds the selected runtime's maximum. A newer grammar tag alone does not establish runtime compatibility. The chosen parser bindings use Cgo/native C code; the prepared compiler and CI need an available C compiler for the scanner build, in addition to race tests. Module versions/checksums and generated artifacts must remain verified and unchanged.

### Facts versus detection heuristics

The sources above establish SDK API behavior and restricted address domains. They do not by themselves prove target import resolution, binding identity, control flow, reachability, mutation behavior, or which inputs an application actually receives. M3's chosen source patterns and confidence rules are scanner design decisions, not statements made by Stellar. Each implemented pattern must be described and tested with paired positive/negative fixtures; unsupported or ambiguous source forms must not become fabricated high-confidence findings. Ordinary scans use local parser libraries and embedded evidence references offline, without executing target code or fetching SDK/protocol sources.
