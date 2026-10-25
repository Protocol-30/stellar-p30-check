# M2 build report

Validated October 25, 2026 (Africa/Lagos), in the current Linux amd64 cloud instance with checksum-verified Go 1.27.3 and a C compiler for race tests. This report does not claim a fresh-task restore, remote CI run, release, or publication.

## Delivered scope

- Add static Go module/workspace inventory using pinned `golang.org/x/mod v0.42.0`, preserving declared/replaced observations separately from unproven effective MVS selection. `go.sum`/`go.work.sum` are not selected-version proof; Go dependency records remain ineffective.
- Add JS manifests, supported npm v1/v2/v3 and Yarn Classic v1 locks, and a partial pnpm v9 subset using pinned `go.yaml.in/yaml/v3 v3.0.5` where YAML parsing is needed. Keep identity, constraints, expected source-URL metadata, aliases, workspaces, manager generation, patches, and unresolved selections explicit.
- Add the exact verified SDK capability matrix and P30-DEP-001. Findings require uncertainty-free supported metadata nominally selecting a canonical official release with a known missing capability. They are medium severity/high confidence because application-role proof is absent.
- Add schema version 2 dependency/warning arrays; retain the v1 schema as historical documentation. Capability annotations describe observed versions, not necessarily effective target runtime versions.
- Preserve bounded rooted discovery, sanitized deterministic reports, offline target handling, and incomplete-scan errors. Malformed/safety-rejected recognized inputs and NUL-containing dependency metadata make a scan incomplete; unsupported valid lock formats remain warnings.

## Validation performed

The exact reusable installation script `/workspace/setup-p30-environment.sh` completed successfully after the final source fixes. With the prepared compiler activated, it executed:

```sh
go mod download
go mod verify
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/p30check
python3 scripts/smoke.py
```

All five internal package suites ran and passed. The thin `cmd/p30check` entry point has no standalone unit tests and was exercised through the compiled binary. Module checksum verification passed. The smoke script uses Python standard library as development tooling; Python is not a scanner runtime dependency.

The real CLI smoke exercised three known-capability JS fixtures, two uncertain-selection fixtures, a missing root, and a resource-limit failure. Known legacy/maintenance releases produced the intended medium findings, modern observed support did not certify target code, and unresolved Go/JS selections remained uncertain. Failure cases returned exit 2. The existing high-gate policy is tested with injected high-severity/high-confidence findings; actual M2 dependency findings do not trip that gate.

Full JSON Schema draft 2020-12 validation with format checking passed for actual reports from all seven checked-in discovery/dependency fixtures and three malformed/NUL/duplicate-key failure cases. Both the historical v1 and current v2 schema documents passed schema self-validation. Local Markdown links and the CI YAML structure, including eight steps in job `test` / **Go checks**, passed inspection. A local pass does not establish a remote GitHub Actions result.

Review-driven corrections include rejecting stale ancestor-lock importers without current workspace membership, preserving uncertainty for patches and incompatible explicit package-manager generations, and making NUL-containing dependency metadata an incomplete error rather than a silent binary exclusion. The final full suite includes regressions for these corrections.

## Dependency-audit limitation

`govulncheck` v1.9.0 was installed and an audit was attempted. The audit **did not complete**: fetching `https://vuln.go.dev/index/modules.json.gz` returned Forbidden in this environment. This is not a passed audit or a vulnerability-free result. Retry after the destination becomes accessible; module checksum verification is a separate integrity check and cannot replace vulnerability-database analysis.

## Evidence and interpretation

Go 1.27.3 remains the prepared compiler; parser modules and checksums are pinned. No Stellar SDK runtime dependency is introduced. Checked official capability sources, matrix version `2026-10-25.1`, and blog-access limits are documented in [evidence.md](evidence.md).

For JS, `effective: true` means supported **nominal static lock/declaration agreement**, not independently verified actual registry routing, official artifact origin, installation, or runtime selection. npm registry URLs can be redirected by installer configuration; the scanner does not read credential-bearing npmrc files or fetch artifacts. Findings describe capabilities of the named official release conditionally on metadata identity.

M2 has one available dependency rule, not source-address analysis, raw-XDR use correlation, or timestamp checks. Go requirements/replacements remain ineffective; unknown/future versions, unsupported identities/formats, and unresolved JS metadata do not become fabricated certainty. Medium dependency findings do not trip `--fail-on high`. Zero findings and exit 0 never certify compatibility.

The [M0/M1 report](build-report.md) remains historical: its standard-library-only and zero-rule statements describe that earlier increment. No new captured network ledger fixture, remote CI result, release artifact, license selection, push, or publication is claimed here.

## Reusable environment

The environment draft was successfully saved with updated `install_script` and `start_skill` instructions for M2. The custom allowed destination `vuln.go.dev` was added while preserving the package-manager presets. Saving these fields does not execute the script, apply the network change to the running instance, or publish the environment. The current machine was prepared and validated separately.

Review/save the changes in environment settings, then publish through the product to activate the saved environment configuration. The vulnerability audit remains uncompleted until database access works and the audit is rerun. Fresh-task restoration, including local-only commits, has not been verified.

## M2 source additions

```text
internal/deps/
  model.go
  go.go
  go_test.go
  js.go
  js_test.go
  js_locks.go
  js_locks_test.go
internal/rules/
  dependency.go
  dependency_test.go
  testdata/capabilities.json
internal/cli/dependencies_test.go
docs/report-v1.schema.json
docs/m2-build-report.md
go.sum
scripts/smoke.py
testdata/README.md
testdata/m2-go-observed/go.mod
testdata/m2-js-legacy/{package.json,package-lock.json}
testdata/m2-js-maintenance/{package.json,package-lock.json}
testdata/m2-js-modern/{package.json,package-lock.json}
testdata/m2-js-unresolved/package.json
```

Existing CLI, walker, report/schema, CI, module manifest, and product docs were extended for M2; the M0/M1 implementation was retained. Fixtures are synthetic local examples, not installed target environments or captured network ledgers.

## Next increment

M3 introduces a narrowly scoped W-address rule only with proven input-domain context and paired positive/negative fixtures. Account-only G/M fields, AddressObject inputs, and contract storage-key validators must remain negative cases. See [rules.md](rules.md#next-increment-m3-w-address-analysis). Stop before M4 raw-XDR/timestamp checks and external publication work.
