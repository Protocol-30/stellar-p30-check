# stellar-p30-check

An offline, read-only command-line scanner that flags Stellar
**Protocol 30** compatibility risks in Go and JavaScript/TypeScript
repositories: outdated SDKs, address validation that rejects CAP-84 `W`
muxed-contract addresses, and ledger-header decoding that predates CAP-88.

> **Status: v0.1.0-dev, DripWave Phase 8 (build and test) done.** All four
> planned rules ship:
> [P30-DEP-001](docs/rules.md#p30-dep-001-stellar-sdk-version) (SDK versions),
> [P30-ADDR-001](docs/rules.md#p30-addr-001-w-address-handling) (address
> checks that reject `W`), [P30-XDR-001](docs/rules.md#p30-xdr-001-ledger-xdr)
> (ledger decoding that fails on CAP-88 ledgers) and
> [P30-TS-001](docs/rules.md#p30-ts-001-close-times) (close-time
> assumptions). Verification results: [docs/verification.md](docs/verification.md).
> Latest release: [v0.1.0](https://github.com/Protocol-30/stellar-p30-check/releases/tag/v0.1.0)
> (binaries for Linux, macOS and Windows, with `SHA256SUMS`). See [docs/roadmap.md](docs/roadmap.md) and
> [docs/limitations.md](docs/limitations.md).

## Quick start

Download a binary from the
[v0.1.0 release](https://github.com/Protocol-30/stellar-p30-check/releases/tag/v0.1.0)
and check it against `SHA256SUMS` (steps in
[docs/ci-integration.md](docs/ci-integration.md)), or install with Go 1.25 or
newer:

```sh
go install github.com/Protocol-30/stellar-p30-check/cmd/p30check@v0.1.0
```

To build from source (CI tests on Go 1.26 and 1.27):

```sh
go build -o p30check ./cmd/p30check
./p30check scan path/to/your/repo
./p30check scan path/to/your/repo --format json
```

Example, on a fixture that pins `@stellar/stellar-sdk` 16.4.0:

```text
$ ./p30check scan testdata/fixtures/js-app
p30check 0.1.0-dev - Stellar Protocol 30 compatibility scan (report schema 1)
Target:     testdata/fixtures/js-app
...
Rules run:  4

Findings (1): 0 high, 1 medium, 0 info
  [medium severity, high confidence] P30-DEP-001 package.json:5
      @stellar/stellar-sdk 16.4.0: decodes Protocol 30 ledgers but cannot handle W muxed-contract addresses (CAP-84)
      evidence: package.json dependencies declares "@stellar/stellar-sdk": "16.4.0"
      fix:      Upgrade to @stellar/stellar-sdk 17.3.0 or later. Stay on 16.4.x only if the app never parses, validates or displays contract addresses.
...
```

Full outputs: [testdata/golden/](testdata/golden/).

## What it checks today

| SDK | Flagged |
|-----|---------|
| `github.com/stellar/go-stellar-sdk` | before v0.8.0 |
| `github.com/stellar/go` | always (deprecated, no Protocol 30 support) |
| `@stellar/stellar-sdk` | before 17.3.0 (16.4.x is medium: ledgers decode, `W` addresses do not) |
| `@stellar/stellar-base`, `stellar-sdk`, `stellar-base` | published versions have no Protocol 30 support |

Versions are read from `go.mod`, `package-lock.json` / `npm-shrinkwrap.json`
and `package.json`.

It also reads Go and JS/TS source for address checks that accept contract
(`C`) and muxed (`M`) addresses but not `W`: regexes such as
`/^[GMC][A-Z2-7]{55}$/`, combined `StrKey.isValid*` calls, prefix checks,
and `SCAddress` type switches with no muxed-contract case.

It flags ledger decoding (`xdr.LedgerCloseMeta`, `LedgerHeader.fromXDR`,
`getLedgers()`) in projects whose SDK cannot decode Protocol 30 ledgers,
`StellarValue` switches without the CAP-88 `_MS` arms, single-arm signature
accessors, vendored `.x` files that predate Protocol 30, and division by a
difference of whole-second close times.

Details and known false positives: [docs/rules.md](docs/rules.md).

Using it in CI (GitHub Actions, GitLab): [docs/ci-integration.md](docs/ci-integration.md).

## Usage

```text
p30check scan <path> [flags]
  --format text|json        output format (default text)
  --fail-on none|high       exit 1 when a high-severity, high-confidence finding is reported (default none)
  --max-file-bytes N        skip files larger than N bytes (default 16777216)
  --max-files N             fail if more than N relevant files are found (default 20000)
  --follow-symlinks         include symlinked files whose target stays inside <path>
p30check version
p30check help
```

Flags may come before or after the path.

| Exit code | Meaning |
|-----------|---------|
| 0 | Scan completed, nothing matched `--fail-on` |
| 1 | Scan completed, `--fail-on high` and a finding is high severity **and** high confidence |
| 2 | Usage error or the scan could not complete (no partial report is printed) |

## Safety

- Reads files only. Never executes, builds or imports scanned code, and
  never runs a package manager; no network.
- Stays inside `<path>`: symlinks are not followed by default, and never
  outside the root.
- Bounded by file-size, file-count, entry-count and depth ceilings, and
  every matcher runs in linear time, so crafted files cannot stall a CI job
  (see [docs/verification.md](docs/verification.md)).
- A malformed manifest produces a warning, not a crash.
- Reports use root-relative paths; text output escapes control characters.

## Documentation

- [docs/roadmap.md](docs/roadmap.md) — where the project is, by DripWave phase
- [docs/rules.md](docs/rules.md) — rule catalog
- [docs/architecture.md](docs/architecture.md) — pipeline, report schema, exit codes
- [docs/limitations.md](docs/limitations.md) — what a scan can and cannot tell you
- [docs/evidence.md](docs/evidence.md) — verified Protocol 30 / SDK facts with sources and dates
- [docs/verification.md](docs/verification.md) — what was tested, on what, and the results
- [docs/ci-integration.md](docs/ci-integration.md) — installing and gating in CI
- [docs/release.md](docs/release.md) — how releases are built and published
- [docs/plan/](docs/plan/) — original build plan and backlog
- [CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md), [CHANGELOG.md](CHANGELOG.md)

## License

Apache License 2.0; see [LICENSE](LICENSE). This is the license the
official Stellar Go and JS SDKs and stellar-xdr use.
