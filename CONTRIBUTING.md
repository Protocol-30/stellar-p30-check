# Contributing

The current implementation stops at M3: bounded discovery, static dependency inventory, exact-version capabilities, and narrow local W-address guard/helper checks. General interprocedural analysis and raw-XDR/timestamp rules belong to later milestones.

Use Go 1.27.3, CGO_ENABLED=1, and a C compiler for normal builds and race testing. JS/TS uses pinned official Tree-sitter bindings/grammars; no pure-Go disabled-feature fallback exists. Initial setup runs `go mod download` with the pinned `go.mod`/`go.sum`; the scanner itself stays offline. In the prepared cloud environment, source `/workspace/dev-env.sh`, then work in `/workspace/stellar-p30-check`.

Before submitting a focused change, run:

```sh
export CGO_ENABLED=1
go mod verify
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/p30check
python3 scripts/smoke.py
```

The smoke script needs Python 3 standard library; it is development tooling, not a scanner runtime dependency.

Add tests that can fail for the changed behavior. Walker changes need boundary/error cases, root confinement, special-file handling, and explicit incomplete-scan reporting. Parser changes need malformed recognized input, valid unsupported formats, scoped declarations, lock conflicts/source-URL metadata, and unknown selection cases. Report changes need schema-v2 validation, deterministic output, correct arrays/enums, and escaped target-controlled text. Address changes need AST-based same-parameter guard/helper evidence, current package-boundary selection, symbol/import/control-flow negatives, per-file syntax limitations, and parser/node-budget failure tests. Do not execute target regexes.

Never run target builds, `go list`, package-manager scripts, or network fetches to resolve scan targets. Declared minimums, local replacements, stale checksums, integrity-only metadata, and unresolved graphs must not be promoted to supported nominal lock selection. An expected registry URL is metadata agreement, not independently proven artifact origin; installer configuration can redirect npm registry URLs and credential-bearing npmrc files are not scanned. Keep `effective` and uncertainty honest even when an observed version has a known capability row.

For compatibility rules, provide at least one positive and one negative fixture, official URLs, checked dates, and an evidence/confidence policy. M2 dependency findings remain medium severity because application-role evidence is absent. W rules require negative fixtures for intentional account-only, AddressObject, and storage-key rejection. Label synthetic source/lock examples as examples; do not label invented XDR as captured network data. Only unambiguous JS W-only flows with the verified nominal W-capable SDK selection may become high/high. Go selection remains unproven (medium/medium); broader SAC destination policy is medium/low. Unknown or unsupported analysis is not a passing compatibility result. Source notices belong in limitations, not dependency_warnings.

Use focused conventional commits and stage only intended files. Do not include credentials, private scanned contents, binaries, or caches. Update relevant docs and CHANGELOG. Keep formatting limited to changed Go files. Pin intentional parser dependencies and preserve module/checksum verification; scanner builds must not silently disable TLS or package integrity checks.

The GitHub Actions job is `test` (**Go checks**) in workflow **CI**. A workflow file is not evidence that remote checks passed. Branch protection/release policy must refer to checks actually observed in GitHub. No license has been selected; resolve licensing with the owner before distributing or accepting contributions under assumed terms. Remote writes and publication require owner authorization.
