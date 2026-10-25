# stellar-p30-check

An offline, read-only command-line scanner that will flag Stellar
**Protocol 30** compatibility risks in Go and JavaScript/TypeScript
repositories: outdated SDKs, address validation that rejects CAP-84 `W`
muxed-contract addresses, and ledger-header decoding that predates CAP-88.

> **Status: v0.1.0-dev, milestone M1.** The CLI, safe file discovery and
> versioned reports work. **No compatibility rules are implemented yet**, so
> a scan today is an inventory of relevant files and makes no compatibility
> claim. See [docs/limitations.md](docs/limitations.md).

`stellar-p30-check` is a working name.

## Quick start

Requires Go 1.25 or newer (CI tests on Go 1.26 and 1.27).

```sh
go build -o p30check ./cmd/p30check
./p30check scan path/to/your/repo
./p30check scan path/to/your/repo --format json
```

Example (from this repository's fixtures):

```text
$ ./p30check scan testdata/fixtures/js-app
p30check 0.1.0-dev - Stellar Protocol 30 compatibility scan (report schema 1)
Target:     testdata/fixtures/js-app
...
Rules run:  0

Findings (0): none reported

Warnings (2):
  [check-not-implemented] package.json: dependency capability check P30-DEP-001 is not implemented yet; SDK versions in this file were not evaluated
  [no-rules] this build ships no Protocol 30 compatibility rules; the report is an inventory only and makes no compatibility claim
...
```

Full outputs: [testdata/golden/](testdata/golden/).

## Usage

```text
p30check scan <path> [flags]
  --format text|json        output format (default text)
  --fail-on none|high       exit 1 when a high-severity, high-confidence finding is reported (default none)
  --max-file-bytes N        skip files larger than N bytes (default 2097152)
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

- Reads files only. Never executes, builds or imports scanned code; no network.
- Stays inside `<path>`: symlinks are not followed by default, and never
  outside the root.
- Bounded by file-size, file-count, entry-count and depth ceilings.
- Reports use root-relative paths; text output escapes control characters.

## Planned rules

Proposed, not implemented — see [docs/plan/BUILD_PLAN.md](docs/plan/BUILD_PLAN.md):

| ID | Checks |
|----|--------|
| P30-DEP-001 | Go/JS Stellar SDK versions against a sourced capability matrix |
| P30-ADDR-001 | Address validation that omits the `W` prefix |
| P30-XDR-001 | Raw ledger-header decoding with a pre-P30 XDR library |
| P30-TS-001 | Assumptions of strictly increasing whole-second close times |

## Documentation

- [docs/architecture.md](docs/architecture.md) — pipeline, report schema, exit codes
- [docs/limitations.md](docs/limitations.md) — what a scan can and cannot tell you
- [docs/evidence.md](docs/evidence.md) — verified Protocol 30 / SDK facts with sources and dates
- [docs/plan/](docs/plan/) — build plan and backlog
- [CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md), [CHANGELOG.md](CHANGELOG.md)

## License

Not yet chosen; the repository owner will add a `LICENSE` file.
