# Offline fixtures

These are synthetic application manifests and minimal lock records, not installed
applications or captured protocol payloads. Scanner tests read them as data and
never run package scripts or install their dependencies.

The `m2-js-*` locked fixtures use exact official SDK versions. Their `resolved`
and `integrity` fields were copied from the official npm registry's version
metadata on October 25, 2026:

- https://registry.npmjs.org/@stellar%2fstellar-sdk/16.3.1
- https://registry.npmjs.org/@stellar%2fstellar-sdk/16.4.0
- https://registry.npmjs.org/@stellar%2fstellar-sdk/17.3.0

The lock records intentionally omit unrelated transitive packages. They establish
only the direct static selection exercised by M2, not a fully installable graph.
No archive integrity assertion is made by the scanner without downloading the
archive; scans remain offline. Capability evidence comes from the tagged official
sources in [docs/evidence.md](../docs/evidence.md).

`m2-go-observed` is a declared minimum with no proven selected Go build list.
`m2-js-unresolved` has a range and no lockfile. Both must remain uncertain.
The older `m1-*` fixtures exercise discovery and deterministic report behavior.

The `m3-*` fixtures are deliberately small local source-analysis examples. The
bad JS guard rejects every W-prefixed string immediately before a W-only
`StrKey.decodeMuxedContract` call; the paired modern guard accepts the W prefix.
Account-only/C-only helpers are negative cases. The SAC destination fixture
models a potentially intentional account-only application policy and yields
review guidance, never a high gate. Ambient TypeScript declarations model the
builder inputs without pretending to initialize an actual transaction.

SDK lock metadata is reused from the verified 17.3.0 fixture above. Missing-lock
and Go-minimum fixtures cannot prove selected SDK versions and remain medium
confidence. Scope and control-flow adversarial cases live in the rule tests.
No fixture application is installed, type-checked, or executed by the scanner.
Published SDK/CAP source provenance is in [M3 evidence](../docs/evidence.md).

M4 paired JS fixtures use existing exact legacy/modern metadata and direct raw decoding calls. The Go fixture uses unresolved selection and a typed whole-second comparison. cap88 contains exact official schema source/excerpts; no binary ledger payload or captured network data is claimed.
