# CI integration

The scanner workflow is `.github/workflows/ci.yml`, named **CI**, job ID `test` displayed as **Go checks**, on Ubuntu with Go 1.27.3. It runs module integrity checks, unit tests, race tests, vet, build, and actual CLI smoke requests. A checked-in workflow or local success does not establish a remote green GitHub Actions run.

M2 can provide dependency warnings and inventory downstream. It is not yet a runtime compatibility gate: P30-DEP-001 is medium severity/high confidence because application feature use is unproven. With the existing `--fail-on high` policy, these findings do not cause exit 1.

After building the scanner from a trusted reviewed revision:

```sh
./p30check scan /path/to/checkout --format json --fail-on high > report.json
```

Check process status before consuming output. Exit 2 means invalid usage, malformed or safety-rejected recognized input, failure, or unexpected incompleteness. Exit 1 requires a high-severity/high-confidence gated finding, which the M2 dependency rule does not produce. Exit 0 may include medium findings and dependency warnings; it never means compatible.

Consume schema version **2**. Read `coverage.rules_implemented`, `coverage.complete`, exclusions, limitations, dependencies' `effective`/`uncertainty`, and `dependency_warnings`. Do not treat an observed supported SDK row as proof of effective selection or safe target code. Even `effective: true` means supported nominal metadata agreement, not verified actual registry routing, artifact origin, or installation. The v1 schema is retained as documentation only, not as a renderer option.

Keep the trusted scanner binary outside the target when practical. Do not invoke target builds or dependency installation to help the offline scan. The scanner's own pinned parser modules are installed during tool setup; that is separate from scanning. Keep private source/declarations/URLs and reports out of public artifacts. Preserve the underlying exit status when piping output.

A reusable GitHub Action, SARIF exporter, release binaries/checksums, automated PRs, and downstream release-pinning policy are later work. Mark no branch-protection check required until its real remote name/behavior has been observed.
