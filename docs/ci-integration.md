# CI integration

The scanner workflow is `.github/workflows/ci.yml`, named **CI**, job ID `test` displayed as **Go checks**, on Ubuntu with Go 1.27.3, CGO_ENABLED=1, and a C compiler for normal builds/race checks. It runs module integrity checks, unit tests, race tests, vet, build, and actual CLI smoke requests. A checked-in workflow or local success does not establish a remote green GitHub Actions run.

M3 can gate a specific high-severity/high-confidence JS W-only contradiction under --fail-on high when the nearest package has supported nominal W-capable SDK selection. This is a narrow static gate, not runtime compatibility certification. P30-DEP-001 remains medium/high; unresolved Go/JS source guidance is medium/medium, and broader SAC destination-policy guidance is medium/low. Those cases do not cause exit 1.

After building the scanner from a trusted reviewed revision:

```sh
./p30check scan /path/to/checkout --format json --fail-on high > report.json
```

Check process status before consuming output. Exit 2 means invalid usage, malformed or safety-rejected recognized input, failure, or unexpected incompleteness. Exit 1 requires a high-severity/high-confidence gated finding; a supported M3 JS source fixture now exercises this real exit. With the default --fail-on none, exit 0 may even include high findings; with --fail-on high, nongated findings and warnings still permit exit 0; it never means compatible.

Consume schema version **2**. Read `coverage.rules_implemented` (2), `coverage.complete`, exclusions, limitations, dependencies' `effective`/`uncertainty`, and `dependency_warnings`. Do not treat an observed supported SDK row as proof of effective selection or safe target code. Even `effective: true` means supported nominal metadata agreement, not verified actual registry routing, artifact origin, or installation. M3 retains schema 2 without new root fields. Source-analysis notices are in limitations, not dependency_warnings; parser resource errors make a scan incomplete/exit 2. The v1 schema is retained as documentation only, not as a renderer option.

The checked-in testdata includes deliberately incompatible W-rejecting source. A whole-repository high-gated scan may legitimately fail on these fixtures; do not silently exclude them or claim that is a scanner defect. Use selected good/bad fixture expectations in smoke checks.

Keep the trusted scanner binary outside the target when practical. Do not invoke target builds or dependency installation to help the offline scan. The scanner's own pinned parser modules are installed during tool setup; that is separate from scanning. Keep private source/declarations/URLs and reports out of public artifacts. Preserve the underlying exit status when piping output.

A reusable GitHub Action, SARIF exporter, release binaries/checksums, automated PRs, and downstream release-pinning policy are later work. Mark no branch-protection check required until its real remote name/behavior has been observed.
