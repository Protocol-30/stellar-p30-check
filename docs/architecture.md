# Architecture

M3 is one offline Go process. It uses pinned Go/YAML and source-syntax parsers, not a Stellar SDK runtime dependency, and requires no running services. Normal builds require CGO_ENABLED=1 and a C compiler.

```text
CLI arguments -> rooted walker -> dependency inventory/capabilities -> dependency + local address rules -> report -> exit policy
```

`cmd/p30check` supplies arguments and process status. `internal/cli` orchestrates with an injectable clock. `internal/walker` owns bounded rooted filesystem reads. `internal/deps` parses only the already discovered contents and separates observed declarations from effective selection. `internal/rules` annotates exact known SDK capabilities, emits the dependency rule, and analyzes supported local W-rejecting guard/helper flows. `internal/report` owns normalization, schema, sanitized text/JSON, and exit policy. Scan-time parsing/evaluation never invokes target tools or the network.

## Discovery and parser boundaries

The explicit target is a directory, not a symlink. Child operations remain rooted; symlinks and nonregular files are excluded. Unix safe opening also guards FIFO/symlink replacement. This provides confinement, not a point-in-time snapshot.

Defaults remain 2 MiB per selected file, 64 MiB bytes read, 10000 selected regular-file candidates, 100000 child entries, and directory depth 64 with root at 0. Directory enumeration is bounded. Generated/binary candidates can consume byte limits before exclusion. Exact scope is in [limitations](limitations.md).

The parser receives bounded in-memory file records, including `go.work` and `pnpm-workspace.yaml` in M2. Go module/workspace parsing uses `golang.org/x/mod v0.42.0`; YAML lock parsing uses `go.yaml.in/yaml/v3 v3.0.5`. Source analysis uses the standard Go parser and official Tree-sitter Go bindings v0.24.0, JS grammar v0.23.1, and TS grammar v0.23.2. Grammar ABI 14 is pinned for binding compatibility rather than substituting the newer JS ABI-15 grammar. All modules are pinned/checksummed. No parser dependency is a Stellar runtime dependency; no pure-Go feature-disabled fallback is provided.

A valid but unsupported format or unresolved selection yields structured warnings/uncertainty. A malformed or safety-rejected recognized manifest/lock, NUL-containing recognized dependency metadata, unexpected traversal/read error, or exhausted ceiling yields a partial report, `coverage.complete: false`, and exit 2. Complete describes completion in the selected scope, including deliberate exclusions; it does not mean all dependency selections or protocol behavior are known.

## Dependency evidence

Dependency records contain ecosystem, group, canonical/observed name, scope, source path/optional line, declaration, observed version, resolution/source descriptions, `effective`, uncertainty, and optional capability annotation. Optional alias preserves a declaration's alternate name. Records remain scoped to their source module/package so unrelated directories are not silently treated as one application.

An observed version is not inherently an effective build version. Go requirements/replacements/workspaces remain `effective: false` because local manifest text alone cannot prove complete MVS selection. Go checksum history cannot prove selection. For supported JS cases, effective selection requires current workspace membership where an ancestor lock is used, consistent explicit package-manager metadata when present, compatible declaration/lock identity/version and matching expected HTTPS official-registry tarball URL metadata, with no unresolved uncertainty. No artifact is downloaded or verified. npm registry URLs are installation-time magic values that can use a configured registry; credential-bearing npmrc files are not read. Effective therefore means supported **nominal static lock/declaration agreement**, not proof of actual routing, official artifact origin, installation, or target runtime use.

Capability annotations describe the **observed version only**, even on an ineffective/uncertain record. Matrix version `2026-10-25.1` records a checked date and source references. Each of `cap88`, `muxed_contract_xdr`, and `w_address_helpers` is `supported|unsupported|unknown`. Unknown/future releases, prereleases, and unverified families are not extrapolated.

## Local address analysis

Address analysis runs only after complete filesystem discovery and dependency-metadata processing. The source analyzers parse bounded contents as syntax; they neither compile/type-check the target nor execute its code or regexes. Comments and string literals cannot masquerade as calls. Predicate proof uses a limited pure AST grammar and symbolic prefix/anchored-regex reasoning; an unsupported or side-effecting condition does not support a finding.

The narrow W-only pattern requires a rejecting guard immediately before a specific imported helper consumes the same simple parameter, without intervening mutation, shadowing, nested/alternate flow, or ambiguous binding. Package evidence comes from the nearest current package/module boundary, not a convenient unrelated manifest elsewhere in the tree. Supported JS W-only flows become high/high only with uncertainty-free nominal official SDK 17.3.0 W support; unresolved JS or Go selection remains medium/medium.

The broader JS instance SAC transfer destination is medium/low even with known SDK support: accepting W is an API capability, not proof that an application's account-only policy is wrong. The first positional destination argument is the relevant value; unrelated arguments/object properties do not establish that flow. Account-only/C-only helpers, AddressObject/storage-key restrictions, and Go SetContractID are not blanket W-acceptance targets. Exact language shapes and exclusions are in [rules](rules.md).

JS/TS/JSX/TSX parsing has a 2-second per-file timeout; syntax-tree traversal has a 200000-node per-file ceiling. Go AST traversal also has a 200000-node ceiling. Resource failures make analysis incomplete/exit 2. Invalid/unsupported source syntax instead adds a path-specific `limitations` entry and no source finding. Ordinary unmatched patterns are deliberate false negatives, not automatically enumerated findings. Source notices do not use `dependency_warnings`.

## Report contract

The emitted contract is [report.schema.json](report.schema.json), JSON Schema draft 2020-12, schema version **`"2"`**. [report-v1.schema.json](report-v1.schema.json) preserves the old document for historical consumers; there is no v1 output renderer/flag.

| Field | Type | Meaning |
|---|---|---|
| `schema_version` | string | `"2"` |
| `tool_version` | string | Build version, distinct from schema version |
| `target` | string | Selected target path |
| `checked_at_utc` | string | UTC RFC3339 timestamp; tests inject the clock |
| `findings` | array | Dependency and supported local address findings; absence is not compatibility proof |
| `summary` | object | `total`, `info`, `medium`, `high` finding counts |
| `limitations` | array of strings | Interpretation, missing-analysis limits, and source-syntax notices |
| `inventory` | array | Selected file metadata: `path`, `kind`, `size` |
| `dependencies` | array | Dependency declarations, observations, resolution evidence, uncertainty, optional capabilities |
| `dependency_warnings` | array | `path`, optional positive `line`, `code`, `message` |
| `coverage` | object | Discovery completion/counters and available rule count |
| `error` | string, optional | Failure associated with a partial report |

Coverage fields remain `complete`, `rules_implemented`, `files_scanned`, `entries_visited`, `bytes_read`, and `skipped` (`path`/`reason` records). `rules_implemented` is **2**, whether or not either rule applies to the target. It does not count findings or prove rule applicability. Warnings and skipped records must be considered alongside completeness.

Finding fields remain `rule_id`, `severity`, `confidence`, `path`, optional positive one-based `line`, `message`, `evidence`, `remediation`, and `references`. Severity is `info|medium|high`; confidence is `low|medium|high`. Arrays are nonnull, output is deterministically sorted, and repeated uncertainty is deduplicated. The clock is the documented ordinary source of variability.

Text includes dependency and warning sections and escapes all target-controlled strings. JSON uses proper encoding. Dependency rows distinguish declared/observed version, source/resolution, effective selection, uncertainty, and observed-version capabilities; no output field should be read as a runtime certificate.

## Rule and exit policy

P30-DEP-001 can emit one finding when uncertainty-free supported metadata nominally selects a canonical official SDK release with verified missing capabilities. It lists missing capabilities in one scoped finding. Severity is **medium**, confidence **high**: missing SDK capability is known, application reliance is not.

`--fail-on none` is default. Exit 1 requires both high severity and high confidence with `--fail-on high`; the supported JS W-only contradiction can now trigger this through actual source analysis. Medium dependency/unresolved-source findings and low-confidence SAC policy hints do not gate. Errors/incomplete scans exit 2 regardless of gate. Completed nongated scans exit 0, even with findings. The rule count and schema version are separate: M3 preserves schema 2 and adds no root fields.

## Scope relative to the original plan

M3 adds local address-analysis helpers/tests and source fixtures to the M2 packages. M4 adds the narrow decoder-use and timestamp stage below. Full-program analysis, captured ledger payloads, SDK runtime dependencies, reusable actions and releases remain outside this increment. The original plan and M0/M1/M2 reports are historical; [m3-build-report.md](m3-build-report.md) records this increment's validation. Deliberate incompatible fixtures remain in normal discovery scope, so a high-gated scan of this whole repository may correctly exit 1.

## M4 ledger stage

After address analysis succeeds, `AnalyzeLedgers` parses bounded source, correlates direct uses with the nearest dependency scope, and emits conservative decoder/timing findings. Any ledger resource failure withholds source findings and sets incomplete/exit2. It reuses syntax helpers, not target execution. See [rules](rules.md) for supported shapes and [M4 report](m4-build-report.md).
