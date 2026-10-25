# Security policy

This is an unreleased M2 CLI. There is no supported release series and no security-audit or Protocol 30 certification claim.

## Intended behavior

The scanner reads a user-selected local directory. It does not execute target code, invoke target package managers, upload source text, call RPC, or require credentials. Discovery is rooted; symlinks and special files are excluded; file reads/traversal have explicit ceilings. Unexpected incomplete scans and malformed or safety-rejected recognized manifests/locks return exit 2. NUL-containing recognized dependency metadata is an incomplete error, not a silent binary exclusion. Text reports escape target-controlled strings.

Go and YAML parsing dependencies are pinned in `go.mod`/`go.sum`; obtaining them during tool development is distinct from scanning. Scan-time dependency resolution uses only already bounded local contents. Replacement paths and lockfile URLs are evidence strings, not instructions to open external files, execute commands, or fetch artifacts. A record's `effective` flag describes supported nominal static lock/declaration agreement, not actual registry routing, artifact origin, runtime safety, or proof that a target used that dependency. npm registry URLs may be redirected by installer configuration; credential-bearing npmrc files are not read.

Targets can contain sensitive information. Keep reports local when needed: paths, dependency source URLs/declarations, and findings can reveal repository details. Do not include credentials or private source in bug reports or public artifacts.

The scanner is not a stable filesystem snapshot. Prefer an isolated, quiescent checkout; concurrent tree changes can produce errors or inconsistent inventories. Filesystem and OS restrictions remain relevant. Unknown formats/versions and unproven sources produce uncertainty rather than security or compatibility assurances. See [limitations](docs/limitations.md).

## Reporting a vulnerability

Do not publish exploit details or private source in a public issue. Use GitHub's private vulnerability reporting **if the owner has enabled it**. Otherwise contact the owner through an established private channel to arrange disclosure; no security email or response SLA is configured.

Include the affected commit/toolchain/OS, a minimal non-sensitive reproduction, expected/observed behavior, and whether it involves file escape, special files, limits, parser behavior, output controls, unintended execution/network access, or incorrect promotion of uncertain dependency evidence.
