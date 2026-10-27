# Using p30check in CI

`p30check` is a single static binary. It reads the checked-out repository
and nothing else: no network access, no tokens, and it never runs your
code. A CI job therefore needs only read access to the checkout.

## Install

**Release binary.** Download the archive for
the runner and verify it against `SHA256SUMS` before running it:

```sh
VERSION=0.1.0
BASE=https://github.com/Protocol-30/stellar-p30-check/releases/download/v${VERSION}
curl -fsSLO "${BASE}/p30check_${VERSION}_linux_amd64.tar.gz"
curl -fsSLO "${BASE}/SHA256SUMS"
grep " p30check_${VERSION}_linux_amd64.tar.gz\$" SHA256SUMS | sha256sum -c -
tar -xzf "p30check_${VERSION}_linux_amd64.tar.gz"
./p30check_${VERSION}_linux_amd64/p30check version
```

Archives exist for `linux`, `darwin` and `windows` on `amd64` and `arm64`
(Windows archives are `.zip`).

**From source** with Go 1.25 or newer:

```sh
go install github.com/Protocol-30/stellar-p30-check/cmd/p30check@v0.1.0
```

Pin an exact version in CI rather than `@latest`, so a new release cannot
change your build's result without a commit on your side.

## Gate on what is certain, record everything

- `--fail-on high` exits 1 only for findings that are **both** high severity and high confidence: an exact SDK version with no Protocol 30 support, an address regex or SDK check that accepts `M` and `C` but not `W`, ledger decoding in a project whose exact SDK version cannot decode Protocol 30 ledgers, a vendored `.x` file without the new arms, and `MustLcValueSignature()` on the `StellarValueExt` union. Everything else is reported but never fails the job.
- The default, `--fail-on none`, always exits 0 when the scan completes.
- Exit code 2 means the scan itself failed: a bad path, a file-count ceiling, or a usage error. Treat it as a broken job, not a pass.
- Keep the full JSON report as an artifact so reviewers can see everything else.

## GitHub Actions

```yaml
name: protocol-30
on: [pull_request]
permissions:
  contents: read
jobs:
  p30check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: install p30check
        env:
          VERSION: 0.1.0
        run: |
          BASE=https://github.com/Protocol-30/stellar-p30-check/releases/download/v${VERSION}
          curl -fsSLO "${BASE}/p30check_${VERSION}_linux_amd64.tar.gz"
          curl -fsSLO "${BASE}/SHA256SUMS"
          grep " p30check_${VERSION}_linux_amd64.tar.gz\$" SHA256SUMS | sha256sum -c -
          tar -xzf "p30check_${VERSION}_linux_amd64.tar.gz"
          mv "p30check_${VERSION}_linux_amd64/p30check" "$RUNNER_TEMP/"
      - name: scan (text, for the log)
        run: '"$RUNNER_TEMP/p30check" scan .'
      - name: scan (JSON report, gate on high/high)
        run: '"$RUNNER_TEMP/p30check" scan . --format json --fail-on high > p30check.json'
      - uses: actions/upload-artifact@v4
        if: always()
        with:
          name: p30check-report
          path: p30check.json
```

## GitLab CI

```yaml
p30check:
  image: golang:1.27
  script:
    - go install github.com/Protocol-30/stellar-p30-check/cmd/p30check@v0.1.0
    - p30check scan .
    - p30check scan . --format json --fail-on high > p30check.json
  artifacts:
    when: always
    paths: [p30check.json]
```

## Reading the JSON report

What the findings, warnings and skipped entries mean, and what to do
about each: [reading-reports.md](reading-reports.md). The schema is in
[architecture.md](architecture.md):
- `findings[]` carries `rule_id`, `severity`, `confidence`, `path`, `line`, `evidence`, `remediation` and `references`;
- `warnings[]` lists files the scanner could not read or rules that do not apply;
- `summary` holds counts.

`schema_version` changes only for incompatible changes. With `jq`, for example:

```sh
jq -r '.findings[] | "\(.severity)/\(.confidence) \(.rule_id) \(.path):\(.line) \(.message)"' p30check.json
```

## Limits to keep in mind

A clean scan means none of the documented shapes were found, not that the
project is Protocol 30 compatible. See [limitations.md](limitations.md) and
the false-positive and false-negative notes in [rules.md](rules.md).
