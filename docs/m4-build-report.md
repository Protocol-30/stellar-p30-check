# M4 build report

Validated October 26, 2026 (Africa/Lagos) on the prepared Linux amd64 environment with Go 1.27.3, CGO and a C compiler.

## Delivered

P30-XDR-001 correlates supported direct raw-ledger decoder calls with the nearest package/module capability observations. Named ESM JS/TS imports and direct return calls to LedgerHeader/LedgerCloseMeta.fromXDR are supported. Go supports direct return calls to the official SafeUnmarshalBase64 with a pointer parameter typed as either ledger type. Modern observed rows suppress the warning; uncertain selections remain visible in dependency warnings and limitations. A scoped nominal legacy selection gives medium/high-confidence review guidance; other supported unresolved uses are medium/low. Historical-only inputs may be intentional, so M4 emits no high-severity finding.

P30-TS-001 recognizes only strict comparisons between distinct typed Go LedgerHeader value parameters' whole-second ScpValue.CloseTime fields. It emits info/low-confidence guidance. It does not establish adjacency, input provenance, or runtime failure. Non-strict comparisons, millisecond fields and unrelated clocks are negative cases.

The official pinned ledger schema and two exact CAP-88 arm excerpts are in [testdata/cap88](../testdata/cap88/PROVENANCE.md). They are specification fixtures, not captured or serialized network payloads. Checked facts and source URLs are in [evidence](evidence.md).

## Validation

Passed unit tests, whole-suite race tests, vet, module checksum verification, normal CGO build, and the real CLI smoke script. All five internal package suites ran; the thin command entry point was exercised through the binary.

```sh
source scripts/dev-env.sh
go test -mod=readonly ./...
go test -race -mod=readonly ./...
go vet -mod=readonly ./...
go mod verify
go build -mod=readonly ./cmd/p30check
python3 scripts/smoke.py
```

The paired legacy/modern JS and unresolved Go/timing fixtures passed actual CLI checks, including low-confidence limits and no new high gate. Fixed-clock integration tests confirm deterministic reports. The existing M3 real high-gate case still passes. Draft-2020-12 JSON Schema validation with format checks passed for all 17 actual M1–M4 fixture reports. Schema remains version2; four rules are available. Unit negatives cover unrelated decoder types, comments/strings, shadowed/type-only/mutated imports, nested manifest scope, modern observations, non-strict comparisons, and millisecond fields.

## Limits and next work

No JS/TS timing analysis, namespace/CommonJS decoder import support, general local-variable decoder/dataflow analysis, restrictive union-switch analysis, target compilation, or runtime XDR decoding is provided. There are no new module dependencies. Syntax and resource handling retain the explicit M3 distinctions; source failures withhold source findings and exit2.

The optional vulnerability audit remains incomplete after the earlier database access failure. Local checks do not establish remote CI, publication, installation fidelity, compatibility certification, or a fresh-task restore. No release, push, license choice, or external submission was performed.

Updated `start_skill` instructions were confirmed saved in the environment draft; the existing installation script and network/credential settings were preserved. Review/save and publish environment settings to activate the draft. Saving does not apply runtime configuration or publish a snapshot.

M5 verification and CI hardening is next. Historical [M3](m3-build-report.md), [M2](m2-build-report.md), and [M0/M1](build-report.md) reports describe their earlier increments.
