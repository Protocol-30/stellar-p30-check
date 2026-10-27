# Maintaining

Repository settings that only the owner can change, and how the project
keeps its issue tracker. Session credentials used for automated work cannot
change settings or push tags.

## Branch protection on `main`

**Settings → Rules → Rulesets → New ruleset → New branch ruleset**:

1. Name: `main`. Enforcement status: **Active**.
2. Target branches: **Add target → Include default branch**.
3. Turn on:
   - **Restrict deletions**
   - **Block force pushes**
   - **Require a pull request before merging** (required approvals: 0 while
     there is one maintainer; raise it when there are more)
   - **Require status checks to pass**, with **Require branches to be up to
     date before merging**. Add these checks (type the name, then pick the
     one from GitHub Actions):
     - `test (go oldstable, ubuntu-latest)`
     - `test (go stable, ubuntu-latest)`
     - `test (go oldstable, macos-latest)`
     - `test (go stable, macos-latest)`
     - `release build (snapshot)`
4. **Create**.

These are the job names in `.github/workflows/ci.yml`. If a job or matrix
entry is renamed, update the ruleset in the same PR, or merges block on a
check that never reports.

Do not require the `release` or `publish` jobs from `release.yml`; they run
only on tags and manual dispatch.

## Tag protection

**Settings → Rules → Rulesets → New ruleset → New tag ruleset**: target
`v*`, turn on **Restrict deletions**, **Block force pushes** and
**Restrict updates**, so a published version's tag cannot move.

## Security

**Settings → Advanced Security** (on some accounts **Security & analysis**):

- **Private vulnerability reporting**: enable. `SECURITY.md` and the issue
  form point reporters to it.
- **Dependabot alerts**: enable. `.github/dependabot.yml` opens weekly PRs
  for workflow actions; the Go module has no third-party dependencies.
- **Secret scanning** and **Push protection**: enable if offered.

## About box and topics

On the repository front page, the gear next to **About**:

- Description: `Offline, read-only scanner for Stellar Protocol 30 compatibility risks in Go and JS/TS repositories`
- Topics: `stellar`, `stellar-protocol`, `protocol-30`, `soroban`,
  `static-analysis`, `compatibility`, `cli`, `golang`, `xdr`, `ci`
- Tick **Releases**; untick **Packages** and **Deployments** (none exist).

## Issues

Work after v0.1.0 is tracked as GitHub issues, one per backlog item or
known limitation that has a concrete fix. Each issue states what done looks
like and, for rule changes, which evidence it rests on. The labels are
GitHub's defaults: `bug` (crashes, wrong exit codes, false findings),
`enhancement`, `documentation`, `good first issue`.

Issue forms live in `.github/ISSUE_TEMPLATE/`. Security reports go through
private vulnerability reporting, never a public issue.

## Releases

See [release.md](release.md). The owner pushes release tags.
