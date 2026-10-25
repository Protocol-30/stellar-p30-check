# Rules and next increment

## Implemented: P30-DEP-001

M2 implements **one** rule: conservative dependency-capability signals. It uses scoped static inventory and the evidence-linked exact-version matrix `2026-10-25.1`, checked 2026-10-25. Capability states are `supported|unsupported|unknown`.

| Canonical identity / exact version | CAP-88 XDR | Muxed-contract XDR | High-level W helpers |
|---|---|---|---|
| `@stellar/stellar-sdk` 16.3.1 | unsupported | unsupported | unsupported |
| `@stellar/stellar-sdk` 16.4.0 | supported | supported | unsupported |
| `@stellar/stellar-sdk` 17.3.0 | supported | supported | supported |
| `github.com/stellar/go-stellar-sdk` v0.8.0 | supported | supported | supported |

These are observed-version capabilities, not a target compatibility grade. No extrapolation is made to other releases, prereleases, metadata variants, forks, aliases, legacy package identities, or community SDKs. Unknown canonical versions receive unknown capabilities and warnings; unsupported identities remain unverified. Sources and verification limits are in [evidence.md](evidence.md).

### Finding contract

A record must have supported nominal lock/declaration agreement, a canonical official SDK name/version in metadata, no uncertainty, and at least one verified missing capability. This classifies the named official release conditionally on the metadata identity; it does not verify registry routing, artifact origin, or installation. One P30-DEP-001 finding lists the missing capabilities for that resolved declaration/group. Evidence includes its source path, group/scope, declaration, observed version, and source. An observed version's annotation alone is insufficient.

Severity is **medium** and confidence **high**. The missing capability is verified, but M2 does not prove the application depends on that feature. These findings do not trigger `--fail-on high`. A supported matrix row or no findings never proves that custom target code is compatible.

Examples of intended outcomes:

- Declaration-compatible JS 16.3.1 lock metadata with the expected registry URL: one medium/high finding naming the missing capabilities.
- Equivalently supported nominal JS 16.4.0: one medium/high finding for missing high-level W helpers, not missing XDR support.
- Equivalently supported nominal JS 17.3.0: no missing-capability finding; source analysis remains absent.
- Go v0.8.0 in a requirement/replacement: observed-version annotation, but no finding or effective-selection claim from manifest text.
- Unknown version, alias, fork, local/workspace replacement, stale checksum, or lock with insufficient source-URL metadata: explicit uncertainty/warnings, not invented incompatibility or compatibility.

## Next increment: M3 W-address analysis

Implement M3 only after the M2 behavior is validated and accepted:

1. Define a small set of source patterns tied to an address domain and scoped dependency/application context. Ordinary rejection of W is not enough.
2. Require proven W-capable fields/APIs before high-confidence classification; leave ambiguous patterns as lower-confidence review hints. Account-only G/M fields, AddressObject inputs, and storage keys are required negative cases.
3. Add paired positive/negative fixtures grounded in CAP-84 and actual supported SDK/API semantics. Label examples accurately and cite official source URLs/checked dates. Do not generate fictional chain fixtures.
4. Emit P30-ADDR-001 with precise file/line evidence, remediation that respects the input domain, and documented false-positive/false-negative limits. Do not blanket-expand every prefix validator.
5. Preserve offline execution, limits, scoped dependency uncertainty, deterministic schema-v2 reports, and high-gate policy. Add targeted rule tests and rerun unit/race/vet/build plus real CLI/schema validation.
6. Stop before M4 raw-XDR/timestamp analysis, SARIF, reusable actions, releases, or external writes.

## Later proposals

| Proposed ID | Planned evidence | Boundary |
|---|---|---|
| P30-ADDR-001 | Rejection in a proven W-capable input domain | M3; not implemented |
| P30-XDR-001 | Legacy effective decoder plus relevant raw-ledger use | M4; correlate within one affected scope |
| P30-TS-001 | Whole-second monotonicity assumptions | M4; isolated patterns default low confidence |

The earlier M2 planning section remains reachable as [exact-next-increment-m2](#exact-next-increment-m2) for historical links.

## Exact next increment: M2

This anchor is retained for the historical [M0/M1 report](build-report.md). M2 is now the implemented increment described above; it is no longer the next work item. [m2-build-report.md](m2-build-report.md) records validation and boundaries.
