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
