# M3 build report

Validated October 26, 2026 (Africa/Lagos), against the current Linux amd64 source with Go 1.27.3, CGO_ENABLED=1, and a C compiler. Source validation passed as recorded below, and M3 installation/startup instructions were saved in the environment draft. No remote CI run, fresh-task restore, release, or publication is claimed.

## Delivered scope

- Add P30-ADDR-001 for supported same-function W-rejecting guards immediately before specific imported W-only helpers consume the same simple parameter.
- Use nearest current JS package evidence; high/high requires unambiguous local syntax/import/flow and uncertainty-free nominal SDK 17.3.0 W support. Unresolved JS and all supported Go selected-version contexts remain medium/medium.
- Add broader JS TransactionBuilder instance SAC destination-policy guidance as medium/low only. Account-only policy can be intentional; this does not justify blanket W acceptance.
- Preserve required negatives for account-only G/M, C-only helpers, Go SetContractID, AddressObject/storage restrictions, modern W acceptance, and unsupported/shadowed/mutated/bypassed flows.
- Parse Go source with its standard parser and JS/TS/JSX/TSX with official pinned Tree-sitter modules. Normal builds require CGO_ENABLED=1 and a C compiler; no pure-Go feature-disabled fallback is provided.
- Keep schema version 2 without new root fields, with rules_implemented 2. Source notices use limitations rather than dependency_warnings. Parser resource exhaustion yields incomplete/exit 2; invalid/unsupported source syntax yields a per-file limitation without a source finding.

## Validation performed

After the final JS ambiguity and cross-language regex fixes, the following checks completed successfully in the checkout. `-mod=readonly` prevented validation from changing module declarations/checksums:

```sh
source scripts/dev-env.sh
export CGO_ENABLED=1
go mod verify
go test -mod=readonly ./...
go test -race -mod=readonly ./...
go vet -mod=readonly ./...
go build -mod=readonly ./cmd/p30check
python3 scripts/smoke.py
```

All five internal package suites ran and passed. The thin command entry point has no standalone unit tests and was exercised through the compiled binary. Module checksum verification passed; the normal CGO build succeeded. The smoke script exercised existing dependency cases and all seven address fixtures, including actual high-gate exit 1, default-gate exit 0, lower-confidence Go/JS/SAC contexts, good/account/storage negatives, missing root, and an exhausted file limit.

The complete reusable `/workspace/setup-p30-environment.sh` was also executed successfully after the final fixes. It activates CGO for normal builds, refreshes pinned modules without changing repository declarations, and runs the unit/race/vet/build/smoke sequence. `scripts/dev-env.sh` now explicitly activates CGO in the prepared cloud checkout. Final source formatting and local Markdown-link checks passed.

Full draft-2020-12 JSON Schema validation with format checking passed on all 14 actual M1/M2/M3 fixture reports, including the gated high finding. Actual NUL-metadata exit-2, unsupported-source-syntax limitation/exit-0, and source-resource-error exit-2 reports also passed. Both current v2 and historical v1 schema documents passed schema self-validation. The unsupported-source case emitted a per-file limitation without a finding; the resource case was incomplete without source findings.

An independent review rebuilt the trusted scanner and rechecked nine concrete earlier false-high reproductions: Go-only regex anchoring, POSIX classes, global built-in mutation, three undefined-shadowing forms, Object.assign mutation, and SDK cast/container aliases. All nine now exited 0 with zero findings and valid schema-v2 reports. Separate positive bad-W, allow-W, account-only, and unresolved-Go checks retained the intended behavior and valid reports. New regressions also cover JS property/brace-hex escape semantics. No broad zero-false-positive claim is inferred from these cases.

Local Markdown-link and documentation whitespace checks passed. The checked-in CI workflow is not evidence of a remote green run. Source-syntax and resource behavior remain explicitly distinct; the report/schema version stays 2 while available rule count is 2.

## Toolchain and source evidence

Pinned Go 1.27.3 is retained. Source parsing adds official Tree-sitter Go bindings v0.24.0, JavaScript grammar v0.23.1, and TypeScript grammar v0.23.2, using compatible ABI 14. The newer JS v0.25 ABI is not silently substituted. Existing manifest parsers remain x/mod v0.42.0 and YAML v3.0.5, recorded with checksums. No Stellar SDK runtime dependency is introduced.

Source proof is a narrow syntax/local-flow contract, not compilation, type checking, interprocedural analysis, installed-artifact verification, or complete runtime assurance. Exact subset and references are in [rules](rules.md) and [evidence](evidence.md). Symbolic guard/regex reasoning does not execute target regexes or rely on fictional W payloads. Current limits are a 2-second JS parse timeout and 200000 source AST nodes per file.

## Remaining limitations

Whole-program address handling, ambiguous/dynamic input flows, raw-XDR decoder/use correlation, and timestamp checks are outside this increment. Nominal SDK lock metadata still does not prove actual routing, artifact identity, installation, or runtime use. Unsupported patterns may be false negatives. Complete discovery/zero findings/exit 0 never certify compatibility.

The repository intentionally includes incompatible source fixtures. Scanning the whole repository with --fail-on high may legitimately exit 1 on those fixtures; they are not silently excluded from discovery.

The last dependency audit in M2 did not complete because the vulnerability database request returned Forbidden. No successful M3 audit was established by this validation; no passed audit or vulnerability-free claim is made. Module checksum verification is separate from vulnerability-database analysis.

## Reusable environment

M3 `install_script` and `start_skill` updates were confirmed saved. They capture CGO and the C compiler requirement for normal builds, the compatible pinned modules, real smoke checks and gate behavior, and use of the existing isolated checkout without creating a worktree. Network and credential settings were preserved. Saving does not apply or publish configuration; review/save the environment settings and publish through the product to activate the draft. Current-instance execution was verified separately. Fresh-task restoration and the vulnerability audit remain unverified.

## M3 source additions

```text
internal/rules/address.go
internal/rules/address_predicate.go
internal/rules/address_predicate_test.go
internal/rules/address_go.go
internal/rules/address_go_test.go
internal/rules/address_javascript.go
internal/rules/address_javascript_test.go
internal/rules/address_test.go
internal/cli/addresses_test.go
docs/m3-build-report.md
testdata/m3-js-reject-w/
testdata/m3-js-allow-w/
testdata/m3-js-account-only/
testdata/m3-js-negative-contexts/
testdata/m3-js-scope-uncertain/
testdata/m3-js-sac-policy/
testdata/m3-go-review/
```

Existing CLI, smoke checks, module pins/checksums, evidence, and product docs were extended. `address_javascript.go` deliberately avoids the `_js.go` suffix, which Go would interpret as a GOOS-specific filename and exclude from the Linux build.

Earlier [M0/M1](build-report.md) and [M2](m2-build-report.md) reports remain historical. No source fixtures are represented as captured ledger data. No license selection, remote push/release, or external publication is claimed.

## Next increment

M4 should correlate raw-XDR decoder selection and actual use with official CAP-88 fixtures and conservative timing guidance, without broadening into unrelated services or release work. See [rules](rules.md#next-increment-m4-xdr-and-timestamp-analysis).
