# Changelog

## Unreleased

### Added
- `p30check scan <path>` with `--format text|json`, `--fail-on none|high`,
  `--max-file-bytes`, `--max-files`, `--follow-symlinks`; `version` and
  `help` commands; exit codes 0/1/2.
- Root-bound file discovery for Go and JS/TS manifests, lockfiles and
  sources, with ignored directories, size/count/depth ceilings and symlink
  handling.
- Versioned report (`schema_version` "1") with text and JSON renderers,
  stable sorting and an injectable clock.
- Evidence log for CAP-84/87/88 and the official Go and JS SDKs.

No Protocol 30 compatibility rules are included yet.
