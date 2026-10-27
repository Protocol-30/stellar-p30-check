# Architecture

`p30check` is a single offline Go binary. One scan is one pipeline:

```
CLI flags ──> walker ──> rules ──> report.New ──> text | JSON ──> exit policy
               │           ▲
               └─ files ───┘ (plus a bounded, root-confined ReadFile)
```

| Package | Responsibility |
|---------|----------------|
| `cmd/p30check` | Process entry point; wires stdout/stderr and Ctrl-C cancellation. |
| `internal/cli` | Argument parsing (flags before or after the path), usage errors, exit-code policy (`Gated`, `ExitCode`). |
| `internal/scan` | Orchestration: walk, run rules, validate findings, add coverage warnings, build the report. Owns `Limitations`. |
| `internal/walker` | Root-bound discovery. Stats entries only; never reads contents. |
| `internal/rules` | `Rule` interface, the shipped rule set (`Default()`) and the rules themselves, including a small JS/TS tokenizer (`jslex.go`) and the SDK version resolution shared by P30-DEP-001 and P30-XDR-001 (`sdkuse.go`); Go source is parsed with `go/parser`. |
| `internal/deps` | `go.mod`, `package.json` and npm lockfile parsers, semver, and the sourced SDK capability table. |
| `internal/report` | Versioned report model, sorting, summary, text and JSON renderers. |

## Walker

- The root path is resolved once (it may itself be a symlink the user chose).
  Everything below is bounded to that resolved directory.
- Entries are visited in name order, so results are deterministic.
- Ignored directories (by name): `.git .hg .svn node_modules bower_components
  vendor dist build out coverage .next .nuxt .turbo .cache`.
- Relevant files: `go.mod`, `go.sum`, `go.work`, `package.json`,
  `package-lock.json`, `npm-shrinkwrap.json`, `yarn.lock`, `pnpm-lock.yaml`,
  `.go .js .jsx .ts .tsx` sources, and `.x` XDR definition files.
  `*.min.js` and `*.d.ts` are treated as generated and not listed.
- Symlinks are not followed by default and appear in `skipped`. With
  `--follow-symlinks`, a link to a regular file is included only if its
  resolved target is inside the root; symlinked directories are never
  followed, which rules out cycles.
- Ceilings: file size (`--max-file-bytes`, default 2 MiB; larger files are
  skipped and listed; default 16 MiB, since npm lockfiles in large repos run to
  several MiB), relevant files (`--max-files`, default 20000), total
  entries (500000) and depth (64). Hitting a count or depth ceiling is an
  error (exit 2) with no partial report, so a truncated scan cannot look
  complete.
- An unreadable subdirectory or file is listed in `skipped` and the scan
  continues; an unreadable root is an error.

## Rules contract

```go
type Rule interface {
    ID() string
    Check(ctx context.Context, in rules.Input) (rules.Output, error) // Output{Findings, Warnings}
}
```

`rules.Input` carries the inventory and `ReadFile(rel)`, which accepts only
clean root-relative paths (`io/fs.ValidPath`), refuses any path that
traverses a symlink, and refuses files over the size limit. Every finding is
checked with `Finding.Validate`; an invalid finding or a rule error fails the
scan (exit 2) rather than producing a misleading report. A problem with one
scanned file (unreadable, malformed JSON, broken `go.mod`) is a warning, so
it cannot abort the scan.

Shipped rules are listed in [rules.md](rules.md). Coverage gaps are always
reported:

| Warning code | When |
|--------------|------|
| `rules-not-implemented` | A planned rule (`rules.Planned`) is not in this build (none at present) |
| `no-rules` | The rule set is empty (only with injected rule sets in tests) |
| `check-not-implemented` | Per manifest, when `P30-DEP-001` is absent |
| `no-relevant-files` | Nothing relevant was found under the target |
| `manifest-unparsed` | A manifest or lockfile could not be read or parsed |
| `source-unparsed` | A Go file has a syntax error or a source file could not be read |
| `lockfile-not-parsed` | A `yarn.lock` or `pnpm-lock.yaml` exists; it is not read yet |

## Report schema (`schema_version` "1")

Root object, always all ten keys; collections are arrays, never `null`:

| Field | Type | Notes |
|-------|------|-------|
| `schema_version` | string | `"1"`. Additive optional fields keep it; renames/removals bump it. |
| `tool_version` | string | Build version. |
| `target` | string | The path exactly as passed on the command line. |
| `checked_at_utc` | string | RFC 3339, UTC, second precision. The only field that differs between identical scans. |
| `findings` | array of Finding | Sorted by path, line, rule ID, severity (desc), confidence (desc), message. |
| `summary` | object | `manifest_files`, `source_files`, `skipped_entries`, `rules_executed`, `findings`, `findings_by_severity` (`info`, `medium`, `high`, always present), `warnings`. |
| `inventory` | array | `{path, kind, category}`; `category` is `manifest` or `source`. Sorted by path. |
| `skipped` | array | `{path, reason}`. Sorted by path. |
| `warnings` | array | `{code, path?, message}`. Sorted by code then path. |
| `limitations` | array of string | Fixed text describing what a scan cannot tell you. |

Finding:

| Field | Type | Notes |
|-------|------|-------|
| `rule_id` | string | e.g. `P30-ADDR-001`. |
| `severity` | `info` \| `medium` \| `high` | Impact if the issue is real. |
| `confidence` | `low` \| `medium` \| `high` | Strength of the evidence. |
| `path` | string | Root-relative, `/`-separated. |
| `line` | integer | 1-based; omitted when unknown. |
| `message`, `evidence`, `remediation` | string | |
| `references` | array of string | Source URLs; `[]` when none. |

All paths are root-relative; absolute paths of the scanning machine are not
written to reports. The text renderer quotes any value containing
non-printable characters, so file names cannot inject terminal escapes.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Scan completed; no finding matched `--fail-on` (always the case with the default `--fail-on none`). |
| 1 | Scan completed; `--fail-on high` and at least one finding is **both** high severity and high confidence. |
| 2 | Usage error, unreadable/invalid target, traversal ceiling hit, rule error, or output failure. |

## Deviations from the planning tree

- `internal/scan` was added to keep orchestration out of the CLI package.
- `internal/report` uses `model.go` + `render.go` instead of separate
  `text.go`/`json.go`.
- Rule files are created in the Phase 8 step that implements them; no
  empty placeholders.
- Golden reports live in `testdata/golden/`; discovery fixtures in
  `testdata/fixtures/`.
