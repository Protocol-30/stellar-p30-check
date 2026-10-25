# Contributing

The current implementation stops at M0/M1. Propose changes against the current code and documented scope; address rules, SDK capability classification, and XDR checks belong to later milestones.

Use Go 1.27.3 and a C compiler for race testing. The project has no third-party dependencies. In the prepared cloud environment, run `source /workspace/dev-env.sh`, then work in `/workspace/stellar-p30-check`. Keep scanner execution offline and never execute code from a scan target.

Before submitting a focused change, run:

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/p30check
```

Add tests that can fail for the behavior you change. Walker changes need boundary/error cases and must preserve root confinement, special-file handling, and explicit incomplete-scan reporting. Report changes need deterministic output, valid schema fields, and escaped target-controlled text. Do not replace unsupported analysis with a passing "compatible" result.

For future compatibility rules, provide at least one positive and one negative fixture, official source URLs, checked dates, and a documented evidence/confidence policy. Synthetic source examples are acceptable when labeled as examples; do not label invented XDR as captured network data. Include account-only and storage-key address contexts among negative fixtures for W-related rules.

Use focused conventional commits and stage only intended files. Do not include credentials, scanned private repository content, compiled binaries, or environment caches. Update relevant docs and the changelog. Keep automatic formatting limited to changed Go files.

The GitHub Actions job is `test` (**Go checks**) in workflow **CI**. A workflow file is not evidence that remote checks passed. Branch protection and release policy must refer to checks actually observed in GitHub.

No license has been selected. Resolve licensing with the repository owner before distributing code or accepting contributions under assumed terms. Publishing, pushing, deploying, and creating remote resources require the owner's authorization.
