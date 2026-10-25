# Limitations

## What M1 establishes

M1 provides a working CLI command, bounded discovery of selected files, file inventory, deterministic report ordering, and tested exit-policy behavior. It does not parse dependencies, evaluate SDK capabilities, analyze address handling, decode ledger XDR, or detect timestamp assumptions. `rules_implemented` is 0. Empty findings and exit 0 are not Protocol 30 compatibility evidence.

`coverage.complete` describes only completion within the selected filesystem scope. It does not mean every file was included, every dependency was resolved, or a compatibility analysis was performed. Inspect `coverage.skipped` and `limitations`.

## File scope

Recognized manifests are `go.mod`, `go.sum`, `package.json`, `package-lock.json`, `npm-shrinkwrap.json`, `yarn.lock`, and `pnpm-lock.yaml`. Recognized sources have `.go`, `.js`, `.jsx`, `.ts`, or `.tsx` extensions. Contents are not executed or interpreted as dependency-resolution commands. `go.work` resolution is a future M2 requirement, not an M1 capability.

Directories with basename `.git`, `node_modules`, `vendor`, `dist`, `build`, `.next`, `coverage`, or `.cache` are excluded. Unrelated extensions are outside the file scope. Child symlinks and special files are skipped; the target root cannot itself be a symlink. There is no option to follow symlinks in this increment.

`*.min.js` files are excluded by name. A Go file with a standard `// Code generated ... DO NOT EDIT.` marker before the first non-comment, nonblank text is excluded after its bounded read. Markers inside the program body do not suppress analysis. Selected files containing NUL bytes are excluded as binary. Detection is intentionally limited: not every generated or binary file is recognizable. Generated/binary candidates consume read limits; an oversized generated Go file can cause an incomplete-scan error before it is classified.

## Resource and filesystem boundaries

Defaults are 2 MiB per selected file, 64 MiB bytes read, 10000 relevant regular-file candidates, 100000 encountered child entries, and directory nesting 64. Depth starts at 0 for the root. Entry counts include ignored directory entries and irrelevant extensions, but not the contents of excluded directories.

Unexpected limit exhaustion or filesystem errors yield a partial/incomplete report and exit 2. Intentional scope exclusions are recorded without declaring the selected scope incomplete. Raising a ceiling changes the scan's resource budget; it does not broaden compatibility coverage.

The scan is not an atomic snapshot. Use a quiescent checkout, especially for untrusted repositories that another process can modify. Rooted operations and regular-file checks protect confinement and special-file handling, but cannot establish a consistent point-in-time view. The prepared/tested platform is Linux amd64; do not infer that every OS/filesystem has been validated.

## Later-rule boundaries

- Rejecting W is correct in some account-only fields, AddressObject inputs, and contract storage keys. A future W rule must establish the accepted address domain before assigning high confidence.
- A `require` line in go.mod is not necessarily the effective Go build version. Replacements, minimal version selection, workspaces, and stale go.sum records require explicit handling or uncertainty.
- JS SDK 16.4.0 can decode muxed-contract XDR while higher-level W helpers reject it. One blanket "CAP-84 supported" boolean is insufficient.
- Static patterns are indications, not proof of runtime breakage. Synthetic source fixtures are not captured network ledgers. No live post-upgrade fixture claim is made.
- The planning package's vote dates could not be revalidated from the cited Stellar blog pages in this environment; see [evidence](evidence.md). Votes must never be presented as unconditional activation dates.
