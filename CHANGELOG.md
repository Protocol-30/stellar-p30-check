# Changelog

## Unreleased

### Added
- P30-ADDR-001: finds address checks in Go and JS/TS that accept `C` with
  `G` or `M` but not CAP-84 `W` addresses (regexes, `StrKey`/`strkey`
  predicates, prefix checks and lists), and `SCAddress` / strkey version
  switches with no muxed-contract case. Go uses `go/parser`; JS/TS uses a
  tokenizer. New `source-unparsed` warning.
- P30-DEP-001: flags official Stellar SDK versions without Protocol 30
  support, read from `go.mod` (with `replace`), npm lockfiles (v1-v3,
  workspaces, aliases) and `package.json`. Capability table sourced from the
  published packages (docs/evidence.md E10-E14).
- Rules can return warnings; malformed manifests produce `manifest-unparsed`
  instead of failing the scan. New `rules-not-implemented` and
  `lockfile-not-parsed` warnings.
- docs/roadmap.md tracks progress by DripWave phase (replaces M0-M8 labels);
  docs/rules.md documents each rule.
- `p30check scan <path>` with `--format text|json`, `--fail-on none|high`,
  `--max-file-bytes`, `--max-files`, `--follow-symlinks`; `version` and
  `help` commands; exit codes 0/1/2.
- Root-bound file discovery for Go and JS/TS manifests, lockfiles and
  sources, with ignored directories, size/count/depth ceilings and symlink
  handling.
- Versioned report (`schema_version` "1") with text and JSON renderers,
  stable sorting and an injectable clock.
- Evidence log for CAP-84/87/88 and the official Go and JS SDKs.

### Changed
- Default `--max-file-bytes` raised from 2 MiB to 16 MiB so typical npm
  lockfiles are read.
