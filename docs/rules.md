# Rules

| ID | Checks | Status |
|----|--------|--------|
| [P30-DEP-001](#p30-dep-001-stellar-sdk-version) | Stellar SDK versions in Go and npm manifests | Shipped (Phase 8.2) |
| P30-ADDR-001 | Address validation that rejects CAP-84 `W` addresses | Planned (Phase 8.3) |
| P30-XDR-001 | Ledger-header decoding with a pre-Protocol 30 XDR library | Planned (Phase 8.4) |
| P30-TS-001 | Assumptions of strictly increasing whole-second close times | Planned (Phase 8.4) |

## P30-DEP-001: Stellar SDK version

Compares the official Stellar SDK versions a project uses with a capability
table built from the published packages ([evidence.md](evidence.md), rows
E10–E14). Two capabilities matter for Protocol 30:

- **Ledger XDR** (CAP-88): after the upgrade every ledger header uses a new
  `StellarValue` arm. Libraries without it fail to decode those headers.
- **`W` addresses** (CAP-84): the new muxed-contract strkey.

### Packages and thresholds

| Package | No support | Ledger XDR only | Both |
|---------|-----------|-----------------|------|
| `github.com/stellar/go-stellar-sdk` | before v0.8.0 | — | v0.8.0+ |
| `github.com/stellar/go` (deprecated) | every version | — | — |
| `@stellar/stellar-sdk` | before 16.4.0; 17.0.0–17.2.x | 16.4.x | 17.3.0+ |
| `@stellar/stellar-base` | up to 15.0.0 (latest) | — | — |
| `stellar-sdk`, `stellar-base` (deprecated unscoped) | up to 13.3.0 / 13.1.0 (latest) | — | — |

### Where versions come from

| Source | Confidence |
|--------|-----------|
| `go.mod` `require` (after applying `replace`) | high |
| `package-lock.json` / `npm-shrinkwrap.json` (lockfileVersion 1–3, workspaces, `npm:` aliases) | high |
| `package.json` exact pin such as `"16.4.0"` | high |
| `package.json` range such as `^16.0.0`, `~17.2.0`, `>=16.0.0`, `17.x` — judged at the **lowest** version it allows | low |

A `package.json` entry is skipped when an npm lockfile in the same folder or
any parent folder records that package (the lockfile finding covers it).
`yarn.lock` and `pnpm-lock.yaml` are not read yet; the rule says so with a
`lockfile-not-parsed` warning and falls back to the range.

### Severity

| Situation | Severity |
|-----------|----------|
| No Protocol 30 support | high |
| Ledger XDR only (`@stellar/stellar-sdk` 16.4.x) | medium |
| Pre-release below the threshold, version newer than the last known release of a frozen package, fork/local `replace`, git/URL/tag spec | info, low confidence |

`--fail-on high` therefore fails a build only for an exact, resolved version
with no Protocol 30 support.

### Examples

Flagged (high, high):

```text
require github.com/stellar/go-stellar-sdk v0.7.3
```

Flagged (medium, high):

```json
"dependencies": { "@stellar/stellar-sdk": "16.4.0" }
```

Not flagged:

```json
"dependencies": { "@stellar/stellar-sdk": "^17.3.0" }
```

All cases: [testdata/rules/dep-001](../testdata/rules/dep-001/README.md).

### Fix

- Go: `go get github.com/stellar/go-stellar-sdk@v0.8.0` (needs Go 1.25.0+).
  Code still on `github.com/stellar/go` must move to the new module path.
- JS: `npm install @stellar/stellar-sdk@^17.3.0`. Drop direct
  `@stellar/stellar-base` dependencies; the SDK bundles it.
- Upgrading the SDK does not change your own address validation. Code that
  checks prefixes (`G`, `M`, `C`) needs `W` too; that is P30-ADDR-001's job.

### Known false positives and negatives

- **False positive:** a range such as `^16.0.0` is judged at 16.0.0 even if
  your yarn/pnpm lockfile installs 16.4.1. Confidence is low for that reason.
- **False positive:** an app that only builds and submits transactions and
  never decodes ledger headers or contract addresses may keep working on an
  old SDK. The rule cannot see how the SDK is used.
- **False negative:** SDK copies under `vendor/` or `node_modules/`, bundled
  builds, and git-sourced forks are not version-checked.
- **False negative:** in an npm workspace, one root lockfile entry for a
  package counts for every `package.json` below it.
