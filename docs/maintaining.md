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

## Daily maintenance

A scheduled Claude Code session runs this procedure once a day. It starts
fresh each time, reads this section, and follows it step by step. The
owner set it up and approved its merging of PRs that pass CI. These
instructions are its whole mandate: anything not listed here goes into
the report as a question for the owner.

### Hard limits

The daily run never:
- pushes or moves a tag, creates or publishes a release, or changes
  repository or organization settings (rulesets, topics, security,
  members, Apps);
- force-pushes, rewrites history, or deletes branches other than its own
  merged maintenance branch;
- skips, disables or loosens a test, a CI check or a linear-time bound to
  get green;
- merges a PR unless it is its own or Dependabot's and all five required
  checks passed on its current head;
- works on the Wave issues listed in [wave-submission.md](wave-submission.md)
  (#6, #7, #8, #12, #13, #14, #15) or on any issue assigned to someone
  else, because those are for contributors;
- changes rule logic, thresholds or severities without a new
  [evidence.md](evidence.md) row, from a primary source read that day, plus
  a positive and a negative fixture;
- comments on issues or PRs opened by people. It reports them to the
  owner instead.

### Steps

1. **Start from `main`.** Fetch `origin/main` and create the branch
   `maintenance/daily-YYYY-MM-DD` (UTC date) from it.
2. **Check `main`'s CI.** Look at the latest `ci` workflow run on `main`.
   If it is red, find the root cause. If the cause is in this repository,
   fix it (step 7). If not, report it.
3. **Run the local checks:**
   ```sh
   test -z "$(gofmt -l cmd internal)"
   go vet ./...
   go test ./...
   go test -race ./...
   go build ./cmd/p30check
   go test ./internal/rules -run=NONE -fuzz=FuzzCheckAll -fuzztime=60s
   go test ./internal/rules -run=NONE -fuzz=FuzzJSAddressObservations -fuzztime=30s
   go test ./internal/rules -run=NONE -fuzz=FuzzGoAddressObservations -fuzztime=30s
   go test ./internal/rules -run=NONE -fuzz=FuzzLedgerObservations -fuzztime=30s
   ```
   A fuzz crasher is a real bug. Fix it and commit the crasher under
   `internal/rules/testdata/fuzz/` with the fix.
4. **Look for upstream changes.** Compare these against
   [evidence.md](evidence.md) (E4, E7, E9, E13):

   | What | Where | Recorded |
   |------|-------|----------|
   | `github.com/stellar/go-stellar-sdk` latest | https://proxy.golang.org/github.com/stellar/go-stellar-sdk/@latest | v0.8.0 |
   | `@stellar/stellar-sdk` latest | https://registry.npmjs.org/@stellar/stellar-sdk/latest | 17.4.0 |
   | `@stellar/stellar-base` latest | https://registry.npmjs.org/@stellar/stellar-base/latest | 15.0.0 |
   | Go toolchains | https://proxy.golang.org/golang.org/toolchain/@v/list | go1.27.3, go1.26.10 |
   | CAP-84, CAP-88 text | https://raw.githubusercontent.com/stellar/stellar-protocol/master/core/cap-0084.md, `.../cap-0088.md` | E1, E2, E18 |

   For each change:
   - **A new version above a rule threshold**, such as go-stellar-sdk
     v0.8.1 or `@stellar/stellar-sdk` 17.5.0, needs no rule change. Add the
     version to an evidence row with the date checked.
   - **A new `@stellar/stellar-base` release** (currently frozen at 15.0.0
     in `internal/deps/capabilities.go`) needs its tarball checked for the
     markers in E12. Then update the evidence row, `jsBaseLastKnown` and
     the fixtures, or report the release if the check cannot be done that
     day.
   - **A changed CAP text** gets reported with a diff summary. Change rules
     only under the hard limits above.
5. **Check the docs and links.** Every relative link in `README.md` and
   `docs/` must resolve. Fix any statement found to be out of date (for
   example a Go version, or a date that has passed).
6. **Triage open PRs and issues:**
   - Dependabot PRs: merge when all five checks pass and the diff only
     changes `uses:` versions in `.github/workflows/`. For a major version,
     read the action's release notes first and name any breaking change in
     the report.
   - The run's own PRs from earlier days: finish them (see step 7) or
     close them with a reason.
   - Issues and PRs opened by people since the last run: list them in the
     report with a one-line summary each. Do not answer them.
7. **Fix and merge.** If steps 2–6 changed anything:
   1. Commit each logical change separately, as a conventional commit
      (`fix(...)`, `docs(...)`, `test(...)`, `ci(...)`), staging explicit
      paths only. Add a `CHANGELOG.md` Unreleased line for anything a
      user would notice.
   2. Push the branch and open one PR titled
      `chore(maintenance): YYYY-MM-DD`. List each change and the evidence
      for it in the PR body.
   3. Wait for the five required checks. Merge with a merge commit only if
      all of them pass.
   4. If a check fails, fix the cause and push again. If it is still red
      by the end of the run, leave the PR open with a comment naming the
      failure, and report it.

   If nothing changed, open no PR.
8. **Report.** End the session with a short report:
   - `main` CI status;
   - local check results;
   - upstream versions found;
   - PRs merged or left open (with links);
   - new issues and PRs from people;
   - anything the owner needs to decide.
