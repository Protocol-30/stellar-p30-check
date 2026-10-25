# stellar-p30-check

An offline CLI foundation for inspecting Stellar application repositories. This increment implements **M0/M1: bounded file discovery, inventory, versioned reports, and exit policy**.

**No Protocol 30 compatibility rules are implemented yet.** An empty findings list means no rules ran, not that the target is compatible. The report states `coverage.rules_implemented: 0` and describes the missing checks.

## Build and run

Use Go **1.27.3** (the tested development/CI toolchain) and a C compiler for race tests. The module uses only the Go standard library. No Stellar SDK, credentials, RPC endpoint, database, or background service is required.

```sh
go build ./cmd/p30check
./p30check --help
./p30check scan .
./p30check scan . --format json --fail-on high
```

In the prepared cloud environment, activate the installed compiler first:

```sh
source /workspace/dev-env.sh
cd /workspace/stellar-p30-check
```

The portable repository helper is `scripts/dev-env.sh`; see its usage before running it outside this cloud environment. Scan targets are read as data: their commands, package managers, build scripts, and source code are never executed.

## Scope and limits

The CLI discovers Go/JS dependency manifests and lockfiles, and `.go`, `.js`, `.jsx`, `.ts`, and `.tsx` sources. M1 inventories these files without parsing dependency versions or applying compatibility rules. Selected files are read within the following bounds:

| Flag | Default | Meaning |
|---|---:|---|
| `--max-file-bytes` | 2097152 (2 MiB) | Maximum bytes per selected file |
| `--max-total-bytes` | 67108864 (64 MiB) | Total bytes read, including subsequently excluded generated/binary candidates |
| `--max-files` | 10000 | Maximum relevant regular-file candidates |
| `--max-entries` | 100000 | Child entries encountered, including unrelated extensions |
| `--max-depth` | 64 | Directory nesting; the target root is depth 0 |

Ignored directory basenames are `.git`, `node_modules`, `vendor`, `dist`, `build`, `.next`, `coverage`, and `.cache`. Symlinks, special files, `*.min.js`, detected generated Go sources, and NUL-containing selected files are excluded with recorded reasons. The target itself must be a directory and may not be a symlink. An unexpected read/traversal error or exceeded ceiling makes the scan incomplete and returns exit 2. See [limitations](docs/limitations.md) for the exact scope and threat model.

## Reports and exit codes

Text and JSON reports expose findings, inventory, limitations, and coverage. `coverage.complete` means discovery completed within the chosen file scope; it says nothing about Protocol 30 compatibility or omitted directories. The JSON schema version is `"1"`; see the [report contract](docs/architecture.md#report-contract).

| Exit | Meaning |
|---:|---|
| 0 | Completed scan without gated findings, or successful help |
| 1 | At least one high-severity **and** high-confidence finding, with `--fail-on high` |
| 2 | Invalid arguments, scan/report failure, or unexpected incomplete scan |

The default is `--fail-on none`. M1 cannot produce compatibility findings, so exit 1 is exercised by policy tests with injected findings rather than fabricated scanner results.

## Development

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/p30check
```

The workflow in `.github/workflows/ci.yml` defines job `test`, displayed as **Go checks**. Local validation and a checked-in workflow do not establish a successful GitHub Actions run.

Read [architecture](docs/architecture.md), [limitations](docs/limitations.md), [evidence](docs/evidence.md), [rule roadmap](docs/rules.md), and [contributing](CONTRIBUTING.md). `BUILD_PLAN.md`, `BACKLOG.md`, and `CODEX_TASK.md` retain the original planning context with review corrections. Later milestones are not shipped functionality.

The repository has no selected license yet. No release, distribution grant, program acceptance, or compatibility certification is claimed.
