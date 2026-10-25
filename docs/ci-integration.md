# CI integration

The scanner's workflow is `.github/workflows/ci.yml`, named **CI**. It defines job ID `test`, displayed as **Go checks**, on Ubuntu with Go 1.27.3. It runs the unit suite, race suite, vet, build, and a real JSON CLI smoke request. Remote execution has not been established merely by adding this file.

M1 is not yet a useful compatibility gate for downstream repositories: it implements no detection rules. If you run it in CI, use its inventory/coverage as a discovery check and retain the limitations in any report consumers.

After building the scanner from a trusted reviewed revision:

```sh
./p30check scan /path/to/checkout --format json --fail-on high > report.json
```

Check the process status before consuming findings. Exit 2 means invalid usage, a failure, or an unexpected incomplete scan. Exit 1 is reserved for high-severity and high-confidence findings under the opted-in gate; M1 has no rules that can produce such findings. Exit 0 is not a compatibility certificate. Also check `coverage.rules_implemented`, `coverage.complete`, exclusions, and limitations.

Keep the scanner binary outside the target when practical. Do not invoke target builds or dependency installation to "help" the offline scan. Avoid placing private source or sensitive reports in public job artifacts. Preserve the scanner's exit status when piping output; shell pipelines otherwise can hide failures.

A reusable GitHub Action, SARIF exporter, release binaries/checksums, automated PRs, and downstream version-pinning instructions are later work. No branch-protection check should be marked required until its actual name and behavior have been observed in GitHub.
