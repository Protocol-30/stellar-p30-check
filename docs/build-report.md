# M0/M1 build report

Validated October 25, 2026 (Africa/Lagos), on Linux amd64 with checksum-verified Go 1.27.3 and GCC. This report describes the current cloud instance; no fresh-task restore or remote CI run has been verified.

## Delivered

- Working `p30check scan <path> --format text|json --fail-on none|high` CLI, help/version commands, configurable resource ceilings, and flags before or after the target.
- Offline, root-bound file discovery and inventory. Targets are never executed. Symlinks and special files are excluded; Unix opening also guards against FIFO/symlink replacement. Selected contents remain bounded.
- Schema version `1`, stable ordering, UTC clock injection, explicit rule coverage and scope limitations, partial failure reports, sanitized text, and tested opt-in exit policy.
- No fake findings: no SDK/address/XDR/timestamp rules are implemented. Empty findings and exit 0 do not establish compatibility.
- Development/architecture/security/contribution docs, an evidence log, JSON Schema, CI workflow, and an exact M2 plan.

## Validation performed

All commands completed successfully in `/workspace/stellar-p30-check` with the prepared compiler activated:

```sh
source scripts/dev-env.sh
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/p30check
```

The three internal package suites executed. The thin `cmd/p30check` main package has no standalone unit tests and was exercised through the compiled binary.

Functional checks exercised both Go and JS discovery fixtures, successful JSON/text output, missing and invalid roots, exceeded file limits, and actual exit codes 0 and 2. Policy tests inject findings to verify exit 1 only for opt-in high-severity/high-confidence gating. Two fixture scans produce identical JSON with an injected fixed clock. Meaningful filesystem tests cover ignored entries, rooted/symlink behavior, FIFOs and replacement, malformed paths, controlled I/O failures, limit boundaries, and partial inventories.

The JSON Schema document passed `Draft202012Validator.check_schema`; actual reports for both discovery fixtures, a missing target, and an oversized file passed full schema validation with format checks using the environment's installed Python jsonschema tool. The repository itself has no Python or Go third-party runtime dependency.

Independent review found a generated-header false exclusion for a marker after `package\tmain`; it was corrected using lexical header detection and regression tests. The final unit/race/vet/build/smoke sequence passed after that fix. Archive integrity, local Markdown links, and CI YAML structure were also checked. A checked-in workflow does not establish a remote green run.

## Evidence and boundaries

The verified official SDK module/version evidence and Protocol 30 CAP references are in [evidence.md](evidence.md). SDK v0.8.0 declares a Go 1.25.0 minimum, while this project deliberately uses Go 1.27.3 and has no SDK dependency yet. JS 16.4.0's XDR support remains distinct from high-level W-helper support.

The cited Stellar blog pages returned proxy HTTP 403. Vote dates remain supplied planning claims rather than newly verified facts. No protocol fixture was invented or presented as captured network data.

All six review findings are addressed either by implemented M1 protections or corrected contracts for later rules: W-context requirements, incomplete-scan behavior, effective dependency selection, special-file exclusion, valid confidence enums, and safe evidence rendering.

## Reusable environment

The tested installation script and startup instructions were saved in the environment draft as `install_script` and `start_skill`. They activate the pinned compiler and writable caches and describe checks using the existing isolated checkout, without creating a worktree. No services or credentials are required. Saving did not publish the environment; review/save the settings and publish through the product to activate that draft. Restoration of these local-only commits has not been verified.

The binary `p30check` is generated and ignored. Model/client approval settings from the archive were not installed. No organization, repository, remote, release, deployment, push, or external message was created by this build. The preconfigured Git remote was preserved.

## Next increment

Implement M2 only when requested, following the dependency and capability steps in [rules.md](rules.md#exact-next-increment-m2). M1 deliberately omits the proposed `internal/deps`/`internal/rules` packages, SDK imports, CAP/XDR fixtures, and `go.sum`; none are needed for standard-library discovery/reporting. No license has been selected.

The planning import was committed as `dd20cfa` (`docs(plan): import reviewed M0 and M1 build brief`). The implementation commit is identifiable with `git log --oneline`; its exact SHA is reported with the delivered result.

## Actual file tree

See the source tree below (generated binary and Git metadata excluded).

```text
stellar-p30-check/
  .github/workflows/ci.yml
  .gitignore
  AGENTS.md
  BACKLOG.md
  BUILD_PLAN.md
  CHANGELOG.md
  CODEX_TASK.md
  CONTRIBUTING.md
  README.md
  SECURITY.md
  SOURCES.md
  cmd/p30check/main.go
  docs/architecture.md
  docs/build-report.md
  docs/ci-integration.md
  docs/evidence.md
  docs/limitations.md
  docs/report.schema.json
  docs/rules.md
  go.mod
  internal/cli/cli.go
  internal/cli/cli_test.go
  internal/report/model.go
  internal/report/render.go
  internal/report/report_test.go
  internal/report/schema_test.go
  internal/walker/open_flags_other.go
  internal/walker/open_flags_unix.go
  internal/walker/special_unix_test.go
  internal/walker/walker.go
  internal/walker/walker_test.go
  reference/dripwave.skill
  scripts/dev-env.sh
  testdata/m1-go/go.mod
  testdata/m1-go/main.go
  testdata/m1-js/index.ts
  testdata/m1-js/package.json
```
