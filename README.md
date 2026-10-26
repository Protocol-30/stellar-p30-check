# stellar-p30-check

An offline CLI for bounded repository discovery and conservative Stellar Protocol 30 signals. **M3 adds a narrowly scoped W-address rule, P30-ADDR-001**, alongside the existing dependency inventory/capability rule P30-DEP-001.

An empty findings list or exit 0 does **not** certify compatibility. Unsupported formats, unresolved selections, unknown versions, and missing runtime context remain explicit. Address analysis covers only supported local guard/helper patterns; it is not general dataflow analysis. Raw-XDR usage correlation and timestamp checks are not implemented.

## Build and run

Use Go **1.27.3**, **`CGO_ENABLED=1`**, and a C compiler for normal builds as well as race tests. JS/TS syntax analysis uses official Tree-sitter Go bindings `v0.24.0`, JavaScript grammar `v0.23.1`, and TypeScript grammar `v0.23.2`. The grammar pins use ABI 14 compatible with these bindings; the newer JS grammar v0.25 uses a different ABI and is not substituted. Existing manifest parsers remain `golang.org/x/mod v0.42.0` and `go.yaml.in/yaml/v3 v3.0.5`. Versions/checksums are pinned in `go.mod`/`go.sum`; no pure-Go feature-disabled fallback is provided. It has no Stellar SDK dependency. Initial development installation may download these modules; running the built scanner performs no network calls or target commands.

```sh
export CGO_ENABLED=1
go mod download
go build ./cmd/p30check
./p30check --help
./p30check scan .
# Intentionally incompatible fixture: this gated scan exits 1
./p30check scan testdata/m3-js-reject-w --format json --fail-on high
```

In the prepared cloud environment, activate the installed compiler first:

```sh
source /workspace/dev-env.sh
cd /workspace/stellar-p30-check
```

The portable repository helper is `scripts/dev-env.sh`; see its usage outside this cloud environment. No credentials, RPC endpoint, database, or background service is needed. Scan targets are read as data: their commands, package managers, build scripts, and source code are never executed.

## Dependency evidence

- Go: parse `go.mod` requirements/replacements and `go.work` statically. Declared and replacement versions are observations, not proven minimal-version selection; Go records remain `effective: false`. `go.sum` does not prove selected versions and cannot generate a compatibility finding.
- JS: inspect package manifests and supported npm v1/v2/v3, Yarn Classic v1, and partial pnpm v9 lock data. A nominal effective lock selection requires compatible declaration/version/identity evidence and matching expected official-registry tarball URL metadata. This does not verify actual registry routing, artifact origin, download, or installation. npm registry URLs can be redirected by installer configuration; the scanner does not read credential-bearing npmrc files. Unproven sources, unsupported ranges/workspace patterns, aliases, forks, patches, and unsupported constructs retain uncertainty. Ancestor locks need current supported workspace-membership evidence; stale lock importers alone are insufficient. Declared package-manager metadata must also match the supported lock family/version. A typical integrity-only pnpm record lacks the expected registry URL metadata.
- Capabilities: exact verified official JS SDK versions 16.3.1, 16.4.0, and 17.3.0, and Go SDK v0.8.0. XDR decoding and high-level W helpers are separate capabilities. Unknown/future versions are not extrapolated.

`P30-DEP-001` emits a **medium-severity, high-confidence** finding only when supported lock/declaration metadata nominally selects a canonical official SDK release with a verified missing capability and no unresolved uncertainty. It does not establish that the application uses that feature. Accordingly these dependency findings do **not** trip `--fail-on high`. See [rules](docs/rules.md) and [limitations](docs/limitations.md).

## W-address checks

P30-ADDR-001 requires a supported same-function guard that rejects W immediately before a specific imported W-only helper consumes the **same simple parameter**. An isolated prefix/regex match is insufficient.

| Local evidence | Severity / confidence |
|---|---|
| JS/TS W-only helper, unambiguous local flow/import, nearest current package has uncertainty-free nominal SDK 17.3.0 W-support selection | high / high |
| Supported W-only helper flow but SDK selection unresolved, including Go's unproven selected version | medium / medium |
| JS instance `TransactionBuilder.addSacTransferOperation(destination, asset, amount, ...)` destination flow | medium / low; account-only business policy may be intentional |

Account-only G/M fields, C-only decoders, AddressObject inputs, contract storage keys, and Go `MuxedContract.SetContractID` are not reasons to accept W indiscriminately. Supported W-accepting guards do not produce a rejection finding. See [exact rule scope](docs/rules.md) and [false negatives](docs/limitations.md).

## Scope and limits

Discovery includes Go/JS manifests and locks, plus `.go`, `.js`, `.jsx`, `.ts`, and `.tsx` sources. Go syntax uses the standard parser; JS/TS/JSX/TSX syntax uses Tree-sitter. Neither compiles nor executes target code.

| Flag | Default | Meaning |
|---|---:|---|
| `--max-file-bytes` | 2097152 (2 MiB) | Maximum bytes per selected file |
| `--max-total-bytes` | 67108864 (64 MiB) | Total bytes read, including subsequently excluded generated/binary candidates |
| `--max-files` | 10000 | Maximum relevant regular-file candidates |
| `--max-entries` | 100000 | Child entries encountered, including unrelated extensions |
| `--max-depth` | 64 | Directory nesting; the target root is depth 0 |

Ignored directory basenames are `.git`, `node_modules`, `vendor`, `dist`, `build`, `.next`, `coverage`, and `.cache`. Symlinks, special files, `*.min.js`, detected generated Go sources, and NUL-containing selected source files are excluded with recorded reasons. NUL-containing recognized dependency metadata instead produces an incomplete error; it cannot be silently skipped. The target must be a directory and may not be a symlink. Unexpected read/traversal errors, exceeded ceilings, and malformed or safety-rejected recognized manifests/locks produce an incomplete report and exit 2. Valid unsupported dependency formats produce warnings rather than invented analysis. Unsupported/invalid source syntax adds a per-file limitation without a finding; parser resource exhaustion instead makes the scan incomplete. JS parsing is bounded to 2 seconds per file and source AST traversal to 200000 nodes per file. Unmatched source patterns remain false negatives, not proof of compatibility.

## Reports and exit codes

Schema **`"2"`** adds `dependencies` and `dependency_warnings` to findings, inventory, limitations, and coverage. Records distinguish declarations, observed versions, resolution/source evidence, effective selection, and uncertainty. `coverage.complete` means processing completed within the chosen filesystem scope, not that every dependency was resolved or every compatibility condition was checked. `coverage.rules_implemented` is 2. See the [report contract](docs/architecture.md#report-contract).

| Exit | Meaning |
|---:|---|
| 0 | Completed scan without gated findings, or successful help |
| 1 | At least one high-severity **and** high-confidence finding, with `--fail-on high` |
| 2 | Invalid arguments, malformed recognized input, scan/report failure, or unexpected incomplete scan |

The default is `--fail-on none`. A proven high/high M3 JS rejection can now trigger actual exit 1 with `--fail-on high`; dependency, unresolved-selection, and SAC-policy hints cannot. Source-analysis notices use `limitations`, not `dependency_warnings`. Address analysis runs only after complete filesystem discovery and dependency-metadata processing. The repository includes deliberately incompatible fixtures, so a full-repository gated scan may legitimately exit 1. Schema v1 is retained as a historical document, not an output option.

## Try the fixtures

```sh
./p30check scan testdata/m2-js-legacy --format json
./p30check scan testdata/m2-js-modern --format json
./p30check scan testdata/m2-go-observed --format json
./p30check scan testdata/m3-js-allow-w --format json --fail-on high
./p30check scan testdata/m3-js-sac-policy --format json
./p30check scan testdata/m3-go-review --format json
```

The legacy JS fixture demonstrates a resolved missing-capability finding; the modern fixture demonstrates a known supported SDK observation without source-code certification. The Go dependency fixture demonstrates a known observed version whose effective build selection remains unproven. M3 fixtures distinguish W-accepting code, lower-confidence SAC policy, and unresolved Go W-only-helper review guidance. Fixtures are synthetic local examples, not captured ledgers or installed target environments; see [testdata](testdata/README.md).

## Development

```sh
export CGO_ENABLED=1
go mod verify
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/p30check
python3 scripts/smoke.py
```

The smoke script uses Python 3 standard library; the scanner binary itself has no Python dependency.

The workflow `.github/workflows/ci.yml` defines job `test`, displayed as **Go checks**. Local validation and a checked-in workflow do not establish a successful GitHub Actions run. Read the [M3 build report](docs/m3-build-report.md) for recorded validation status.

Read [architecture](docs/architecture.md), [limitations](docs/limitations.md), [evidence](docs/evidence.md), [rules and next increment](docs/rules.md), and [contributing](CONTRIBUTING.md). The [M0/M1 build report](docs/build-report.md), [M2 build report](docs/m2-build-report.md), and original planning documents are historical; later planned milestones are not shipped functionality.

The repository has no selected license yet. No release, distribution grant, program acceptance, or compatibility certification is claimed.
