# Architecture

M1 is one offline Go process. It has no third-party dependencies or running services.

```text
CLI arguments -> rooted discovery -> inventory -> report normalization -> text/JSON -> exit policy
```

`cmd/p30check` supplies process arguments and exit status. `internal/cli` owns argument handling and orchestration with an injectable clock for deterministic tests. `internal/walker` owns filesystem access and limits. `internal/report` owns the report model, enum validation, normalization, renderers, and exit policy. Compatibility analysis will be inserted between inventory and report normalization; no analysis engine or fake rules are included now.

## Discovery contract

The target is an explicit directory, not a symlink. Child paths are relative to an opened root and are never allowed to escape it. Symlinks and nonregular files are excluded. On Linux, safe file opening also protects against replacement by symlinks/FIFOs between inspection and opening. This is confinement, not a snapshot guarantee: concurrent tree changes may produce errors or inconsistent inventories.

Defaults are 2 MiB per file, 64 MiB total bytes read, 10000 selected-file candidates, 100000 encountered child entries, and directory depth 64 with root depth 0. Directory enumeration is bounded. Generated/binary candidates may consume the byte budget before classification excludes them. Exact exclusions are in [limitations](limitations.md).

Normal completion includes documented scope exclusions. Unexpected traversal/read errors or exceeded ceilings produce a partial report with `coverage.complete: false`, an error, and exit 2. A complete filesystem inventory still provides zero compatibility-rule coverage in M1.

## Report contract

The machine-readable contract is [report.schema.json](report.schema.json) (JSON Schema draft 2020-12). JSON root fields in schema version `"1"`:

| Field | Type | Meaning |
|---|---|---|
| `schema_version` | string | Report contract version (`"1"`) |
| `tool_version` | string | Build/tool version, distinct from schema version |
| `target` | string | Selected target path |
| `checked_at_utc` | string | UTC RFC3339 timestamp; tests inject the clock |
| `findings` | array | Findings; always empty in M1 |
| `summary` | object | `total`, `info`, `medium`, `high` finding counts |
| `limitations` | array of strings | Missing analysis and other interpretation limits |
| `inventory` | array | Selected-file records: `path`, `kind`, `size` |
| `coverage` | object | Completeness and discovery/rule-coverage counters |
| `error` | string, optional | Scan failure associated with a partial report |

Coverage contains `complete` (boolean), `rules_implemented`, `files_scanned`, `entries_visited`, `bytes_read` (integers), and `skipped` (array of `{ "path": string, "reason": string }`). Inspect exclusions as well as `complete` before interpreting a run. `files_scanned` counts included selected files, not all entries in the repository. The inventory is file metadata; it is not a resolved dependency graph.

Future findings use `rule_id`, `severity`, `confidence`, `path`, optional `line`, `message`, `evidence`, `remediation`, and `references`. Severity is `info|medium|high`; confidence is `low|medium|high`. Line numbers, when known, are positive and one-based. "Informational" is a severity description, not an allowed confidence value. Findings sort by path, line, rule, and stable tie-breakers. Inventory and exclusions are sorted. Findings, references, inventory, limitations, and exclusions use arrays rather than null collections.

The following is an **illustrative empty-directory report**, not a compatibility result:

```json
{
  "schema_version": "1",
  "tool_version": "example",
  "target": "/example/empty",
  "checked_at_utc": "2026-10-25T00:00:00Z",
  "findings": [],
  "summary": {"total": 0, "info": 0, "medium": 0, "high": 0},
  "limitations": ["No Protocol 30 compatibility rules are implemented."],
  "inventory": [],
  "coverage": {
    "complete": true,
    "rules_implemented": 0,
    "files_scanned": 0,
    "entries_visited": 0,
    "bytes_read": 0,
    "skipped": []
  }
}
```

JSON encoding escapes JSON control characters. Text rendering escapes target-controlled strings, including evidence, paths, and errors, so control sequences cannot forge additional output lines. The timestamp is expected to vary between ordinary runs; deterministic tests supply the same clock.

## Exit policy

`--fail-on none` is the default. A completed scan exits 0 unless the caller opts into `--fail-on high` and there is a finding with both high severity and high confidence. That combination exits 1. Invalid arguments, failures, and unexpected incomplete scans exit 2 regardless of the gate. Policy tests inject findings; the actual M1 scanner emits none.

## Differences from the planning tree

M1 deliberately omits `internal/deps`, `internal/rules`, CAP/XDR fixtures, an SDK dependency, and `go.sum`: none are needed by a standard-library-only discovery/reporting CLI. Product docs replace the handoff README. No project-local model or approval settings are installed. No license is assumed and no remote CI/release success is claimed.
