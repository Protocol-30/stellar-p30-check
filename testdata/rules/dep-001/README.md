# P30-DEP-001 fixtures

Each directory is a minimal project for the SDK version check. Package
names and versions are real published releases (see `docs/evidence.md`
rows E10-E14); the projects themselves are invented and contain no ledger
data. Expected results are in `internal/rules/dep_test.go`.

| Directory | Case | Expected |
|-----------|------|----------|
| go-legacy | go-stellar-sdk v0.7.3 | high severity, high confidence |
| go-deprecated | deprecated github.com/stellar/go | high, high |
| go-modern | go-stellar-sdk v0.8.0 | no finding |
| go-replaced | v0.7.3 replaced by a local fork | info, low (cannot verify) |
| js-range-old | `^16.0.0`, no lockfile | high, low (range only) |
| js-locked-old | `^16.0.0`, lockfile installs 16.3.1 | high, high on the lockfile |
| js-xdr-only | exact 16.4.0 | medium, high (no `W` support) |
| js-modern | `^17.3.0`, lockfile installs 17.4.0 | no finding |
| js-yarn | `~17.2.0` with yarn.lock (not parsed) | high, low + lockfile-not-parsed warning |
| js-workspace | npm workspace; root lock installs 17.3.0 for `web`, stellar-base 15.0.0 at root | high, high for stellar-base only |
| js-unresolved | git URL and `npm:...@latest` alias | info, low for each |
| malformed | truncated package.json and go.mod | manifest-unparsed warnings, no findings |
