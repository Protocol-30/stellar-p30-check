# Security policy

## Reporting a vulnerability

Please report vulnerabilities privately through GitHub's **Report a
vulnerability** button on this repository's Security tab (private
vulnerability reporting). Do not open a public issue for a security problem.
If that button is not available yet, open an issue asking for a private
contact without including details.

Include the version (`p30check version`), the command you ran, and a minimal
directory tree that reproduces the problem.

## Scope

In scope: anything that makes `p30check` read outside the scan root, follow
a symlink it should not, execute or evaluate scanned content, hang or exhaust
memory on crafted input, or write unescaped control sequences to a terminal.

Out of scope: incorrect or missing compatibility findings. The scanner gives
guidance, not a security audit or compatibility certification; report those
as normal issues.

## Supported versions

Fixes go into the latest release (currently v0.1.x) and the default branch.
Older releases are not patched; upgrade to the latest release.
