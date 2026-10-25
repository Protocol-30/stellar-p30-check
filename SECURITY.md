# Security policy

This is an unreleased M0/M1 CLI foundation. There is no supported release series and no claim of security-audit or Protocol 30 certification.

## Intended behavior

The scanner reads a user-selected local directory. It does not execute scanned code, invoke target package managers, upload source text, call RPC, or require credentials. Discovery is rooted, symlinks are excluded, special files are skipped before reading, and reads/traversal have explicit ceilings. Unexpected incomplete scans return exit 2. Text reports escape target-controlled strings.

The selected target can still contain sensitive information. Keep reports local when needed; paths and future finding evidence may expose repository details. Do not put secrets in bug reports.

The scanner is not a stable filesystem snapshot and cannot promise consistent results when another process modifies a tree during a scan. Prefer an isolated, quiescent checkout. Filesystem and operating-system restrictions remain relevant; see [limitations](docs/limitations.md).

## Reporting a vulnerability

Do not publish exploit details or private source in a public issue. Use GitHub's private vulnerability reporting **if the repository owner has enabled it**. If no private channel is available, contact the owner through an established private channel to arrange disclosure; no security email or response SLA is configured by this project.

Include the affected commit/toolchain/OS, a minimal non-sensitive reproduction, expected behavior, observed behavior, and whether the issue involves file escape, special files, limits, output controls, or unintended execution/network access.
