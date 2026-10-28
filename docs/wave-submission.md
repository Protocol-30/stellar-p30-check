# Drips Stellar Wave submission brief (Phase 12)

Prepared 2026-10-28 for issue
[#11](https://github.com/Protocol-30/stellar-p30-check/issues/11). The
owner decides whether to apply; nothing here has been submitted.

## How these facts were checked

The official Drips sites (`docs.drips.network`, `www.drips.network`) could
not be opened from the session that wrote this brief (blocked by its
network policy). The rules below come from **search-engine excerpts of
those pages** on 2026-10-28, plus third-party reports where marked. Treat
them as a checklist to confirm in the Drips Wave app, not as verified
rules.

| Source | URL | How read |
|--------|-----|----------|
| Drips Wave docs | https://docs.drips.network/wave/ | search excerpts |
| Participating in a Wave (maintainers) | https://docs.drips.network/wave/maintainers/participating-in-a-wave/ | search excerpts |
| Terms and rules | https://docs.drips.network/wave/terms-and-rules/ | search excerpts |
| Points and rewards | https://docs.drips.network/wave/points-and-rewards/ | search excerpts |
| Stellar Wave program page | https://www.drips.network/wave/stellar | search excerpts |
| Stellar Wave repo list | https://www.drips.network/wave/stellar/repos | search excerpts |
| Creating meaningful issues | https://www.drips.network/blog/posts/creating-meaningful-issues | search excerpts |
| Wave 6 changelog (third party) | https://lumenloop.com/news/changelog-new-stellar-wave-6 | search excerpts |

## Eligibility checklist

| Requirement | Where it comes from | This repo |
|-------------|--------------------|-----------|
| Public GitHub repository | Drips maintainer docs (repos are synced from the Drips Wave GitHub App; only public repos are listed) | Yes |
| Drips Wave GitHub App installed on the org that hosts the repo | Drips maintainer docs | **Owner to do** on the `Protocol-30` org |
| Repo applied to a specific Wave Program and approved by its organizers | Drips maintainer docs | **Owner to do**: apply to the Stellar Wave |
| Relevant to the program's ecosystem | Drips maintainer docs ("only apply to Programs that are relevant to your ecosystem"); Wave 7 changelog ties appeals to Stellar relevance | Yes: the tool exists for Stellar's Protocol 30 upgrade (CAP-84, CAP-88) |
| Identity verification (KYC) to apply | Third-party reports (Wave 6 changelog summary, another project's research notes); official docs confirm KYC for contributors and for withdrawals | **Owner to check** in the app; complete KYC first if asked |
| At most 5 repo applications per wave, per user and per org | Third-party report (Wave 6 changelog summary) | One repo, so within the limit |
| Issues prepared for contributors, with a complexity set per issue | Drips maintainer docs | Ready: see "Issues for the Wave" below |
| No low-quality, untested or not-understood LLM-generated contributions (aimed at contributors) | Terms and rules | Applies to PRs you accept; see "Things to be ready for" |

A rejected repository cannot simply be re-applied. Appeals go only through
**Maintainers → Orgs and Repos → Appeal** in the app:
- the first appeal is allowed two weeks after the rejection;
- a declined appeal starts a one-month cooldown;
- each repo can be appealed at most three times;
- an appeal is reconsidered only after substantive change.

## Timing

- The Stellar Wave is described as monthly, with each Wave lasting
  about 7 days.
- Search results show Wave 10 listed as Oct 15–22, with no year shown,
  and "no active or upcoming Waves" at the time of the excerpt.
- The next Wave's dates were not found. Check the Stellar Wave page or the
  Drips Discord or newsletter.
- Stellar's Protocol 30 Testnet vote is proposed for 2026-11-03 and the
  Mainnet vote for 2026-11-23 (these are votes, not guaranteed
  activations). A November Wave would fall in the window where this tool
  is most useful.

## How to apply (owner steps)

1. Sign in to the Drips Wave app (from https://www.drips.network/wave/stellar)
   with the GitHub account that administers `Protocol-30`.
2. Complete identity verification under **Settings → Profile** if the app
   asks for it.
3. **Maintainers → Orgs and Repos**: install the **Drips Wave** GitHub App
   (https://github.com/apps/drips-wave) on the `Protocol-30` organization,
   granting access to `stellar-p30-check` only.
4. Sync `Protocol-30/stellar-p30-check` and apply it to the **Stellar
   Wave** program.
5. Use the description below where the form asks about the project.
6. Wait for the email or in-app notice. Approval status also shows on the
   Orgs and Repos dashboard.

### Project description (draft)

> **stellar-p30-check** (`p30check`) is an offline, read-only command-line
> scanner that flags Stellar Protocol 30 compatibility risks in Go and
> JavaScript/TypeScript repositories before the upgrade. Its four rules are:
>
> - outdated Stellar SDK versions;
> - address validation that rejects CAP-84 `W` muxed-contract addresses;
> - ledger decoding that fails on CAP-88 ledgers;
> - arithmetic that assumes whole-second close times always differ.
>
> Each finding comes with a severity, a confidence and a cited source. The
> `--fail-on high` flag fails CI only on high-confidence findings.
>
> v0.1.0 is released with reproducible binaries for Linux, macOS and
> Windows, plus `SHA256SUMS`. The thresholds were checked against the
> published SDK packages: the rules flag go-stellar-sdk v0.7.3 and none of
> its code in v0.8.0. CI runs tests, the race detector and fuzz-derived
> regression inputs on every PR. It is a static check: it does not certify
> that an application works on Protocol 30.
>
> Repository: https://github.com/Protocol-30/stellar-p30-check. Apache-2.0.
> Open, scoped issues cover Yarn/pnpm lockfiles, SARIF output, a GitHub
> Action, Windows CI and accuracy improvements to the rules.

### Demo

Two fixture projects in the repo show the before and after
(`testdata/integration`, reports pinned in `testdata/golden/`):

```sh
p30check scan testdata/integration/legacy-app --fail-on high   # 8 findings (7 high, 1 medium), exit 1
p30check scan testdata/integration/modern-app --fail-on high   # 0 findings, exit 0
```

`legacy-app` is a Go indexer on go-stellar-sdk v0.7.3 and a TypeScript app
on `@stellar/stellar-sdk` 16.3.1; `modern-app` is the same code after the
upgrade. Real-package results are in [verification.md](verification.md).

## Issues for the Wave

According to the docs, issues added from GitHub default to **Trivial**. Set
the complexity of each issue in the Drips app. The points per complexity
are:

| Complexity | Points | Docs' description |
|-----------|--------|-------------------|
| Trivial | 100 | Typos, minor copy changes, very small bug fixes |
| Medium | 150 | Standard feature work or involved bug fixes |
| High | 200 | Complex architecture, refactors, new integrations |

Suggested complexity, judged from each issue's "done when" list:

| Issue | Suggested | Notes |
|-------|-----------|-------|
| [#6](https://github.com/Protocol-30/stellar-p30-check/issues/6) yarn.lock and pnpm-lock.yaml | High | Two parsers, fuzzing, fixtures |
| [#7](https://github.com/Protocol-30/stellar-p30-check/issues/7) inline ignore comments | High | Report schema change, Go and JS/TS |
| [#8](https://github.com/Protocol-30/stellar-p30-check/issues/8) SARIF output | High | New output format, schema validation |
| [#12](https://github.com/Protocol-30/stellar-p30-check/issues/12) Windows CI | Medium | Path and symlink tests on Windows |
| [#13](https://github.com/Protocol-30/stellar-p30-check/issues/13) W regexes capped at 56 | Medium | Linear-time length analysis |
| [#14](https://github.com/Protocol-30/stellar-p30-check/issues/14) if / else-if chains | Medium | Uses the existing bracket index |
| [#15](https://github.com/Protocol-30/stellar-p30-check/issues/15) closeTime false positives | Medium | SDK-use gate for JS/TS |

Keep these out of the Wave:
- [#9](https://github.com/Protocol-30/stellar-p30-check/issues/9) (GitHub
  Action) depends on #8.
- [#10](https://github.com/Protocol-30/stellar-p30-check/issues/10) is
  blocked until Protocol 30 is active on Testnet.
- [#11](https://github.com/Protocol-30/stellar-p30-check/issues/11) is the
  owner's own task.

There are no Trivial issues yet. The docs encourage matching complexity
to real scope, so add small ones (doc fixes, extra fixtures) only when
they are real.

## Things to be ready for during a Wave

From the maintainer docs, as excerpted:
- **Assign quickly.** Assign contributors from the dashboard or on GitHub.
- **Resolve before the Wave ends.** Points count only for issues marked
  resolved before the end. A good PR that cannot merge for reasons outside
  the contributor's control can still be marked resolved. Unresolved
  issues stay in the program for the next Wave.
- **Review within 14 days.** Two-way reviews are due within 14 days of an
  issue closing.
- **Compliment window.** Payouts follow the Wave end and a 7-day
  maintainer compliment window.

Two questions for the owner:
- **Reviewing PRs.** Every PR goes through the `main` ruleset (five CI
  checks), and rule changes need evidence rows and fixtures
  (CONTRIBUTING.md). Make sure you can review a burst of PRs in a 7-day
  window.
- **AI-assisted contributions.** The terms prohibit low-quality, untested
  or not-understood LLM-generated contributions. This repository was built
  with an AI coding assistant under the owner's direction. Be ready to
  explain and maintain the code yourself, and decide whether to say so in
  the application.

## Similar tools

A search on 2026-10-28 for Protocol 30 / CAP-84 / CAP-88 compatibility
scanners found no other tool that scans application repositories. The
results were:
- this repository;
- the SDK teams' own "SDK Updates for Protocol 30 Compatibility" tracking
  issues (Flutter, Java, .NET, iOS, PHP, Rust and KMP SDKs);
- the go-stellar-sdk Protocol 30 release train.

That is not proof that nothing similar exists, so do not claim the tool is
unique. This scanner covers Go and JS/TS only.

## Status

- [x] Rules and limits gathered, with sources (search excerpts; confirm in the app)
- [x] Description, demo and issue list ready
- [ ] Owner: install the GitHub App, complete KYC if asked, and apply
- [ ] Owner: set complexity on the Wave issues once approved
