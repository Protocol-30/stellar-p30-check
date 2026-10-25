# Supplied source snapshot and implementation evidence

The original package marked its source snapshot checked **October 25, 2026**. The list below is retained as planning provenance, not proof that every item was rechecked in this build. See [docs/evidence.md](docs/evidence.md) for the implementation's checked official tags, compiler provenance, and explicit access limits. These sources are separate from the user-provided `reference/dripwave.skill`.

## Original planning-source claims (see revalidation status above)

1. **Stellar Development Foundation, Bell/Protocol 30 Upgrade Guide (last updated October 23, 2026):** https://stellar.org/blog/developers/bell-protocol-30-upgrade-guide
   - Proposed Testnet vote Nov 3, Mainnet vote Nov 23, both 17:00 UTC; these are votes, not guaranteed activation.
   - CAP-84 adds `SC_ADDRESS_TYPE_MUXED_CONTRACT`, `W...` addresses. App-specific validators in W-capable contexts need appropriate support; account-only, AddressObject, and contract storage-key contexts must not indiscriminately accept W.
   - CAP-88 adds `STELLAR_VALUE_SIGNED_MS` and `STELLAR_VALUE_EMPTY_TX_SET_MS`, requiring updated XDR for raw ledger-header decoding.
   - CAP-87 adds opt-in ML-DSA host functions; not a broad breaking change requiring existing contract redeployment.
   - Official JS SDK v17.3.0 provides P30 and W support; maintenance v16.4.0 can decode P30/muxed-contract XDR, while higher-level Address/scValToNative helpers reject W. Go SDK v0.8.0 listed.
2. **Stellar Protocol 30 overview:** https://stellar.org/blog/developers/introducing-bell-protocol-30-on-stellar
   - Explains three CAPs and tentative vote timeline.
3. **Official Stellar Go SDK repository:** https://github.com/stellar/go-stellar-sdk
   - Official module repository; docs show `go get github.com/stellar/go-stellar-sdk@latest`. Verify actual import path and SDK modules at build time.
4. **Stellar Core releases:** https://github.com/stellar/stellar-core/releases
   - v30.0.0 is listed, corresponding to protocol changes.
5. **Drips Stellar Wave approved repositories:** https://www.drips.network/wave/stellar/repos
   - Currently shows 850 approved repos. This is a snapshot and not evidence of the scanner's uniqueness or approval.
6. **Drips maintainer onboarding:** https://docs.drips.network/wave/maintainers/participating-in-a-wave/
   - Organization/repo must be connected, public repos explicitly submitted, accepted by program organizers before program issues can be added.
7. **OpenAI Codex configuration reference:** https://learn.chatgpt.com/docs/config-file/config-reference
   - `model_reasoning_effort` supports model/client-dependent values such as `high`, `xhigh`, `max`, or `ultra`. Model and account compatibility must be checked in Codex.
8. **OpenAI developer settings:** https://learn.chatgpt.com/docs/developer-settings
   - Use `/model` or `codex --model ...` when access allows. `--config model_reasoning_effort` can override for a run.
9. **OpenAI model docs:** https://developers.openai.com/api/docs/models
   - `gpt-6.1-sol` exists; supported API reasoning efforts include `xhigh` and `max`. API support does not guarantee it is in this user's Codex plan.

## Source distinction

The **original supplied DripWave skill** is in `reference/dripwave.skill` and establishes process (verification, Phases 1–13, repo hygiene, agent tasks, release/submission gates). It does **not** contain confirmed Protocol 30 versions, XDR fixtures, or SDK package guarantees.

Every later rule choice, proposed module layout, and milestone target in the original package remains a proposed design until implemented and tested. README.md and docs/architecture.md describe the current CLI/report contract. The cited vote dates remain unverified planning claims here and must be rechecked before schedule decisions.
