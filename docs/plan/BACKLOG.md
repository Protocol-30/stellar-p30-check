# Proposed GitHub Backlog (not yet created)

These are draft issues for a future `stellar-p30-check` repository. They are ordered by dependencies; after the remote repo exists, convert them to actual issues with the supplied acceptance criteria. Proposed labels are not Drips program labels.

## Phases 1–5: Evidence and architecture

### I-01 `docs(p30): pin verified protocol 30 compatibility sources`
**Why:** Prevent invented version gates. **Depends:** none. **Type/complexity:** docs / medium.
- [ ] Record official CAP-84, CAP-88, and upgrade guide references with checked date.
- [ ] Verify official Go SDK module path, minimum Go toolchain, JS package name and release caps.
- [ ] Document JS v16.4.0 XDR-only distinction and why CAP-87 isn't a breaking-change rule.
- [ ] Create an evidence-change process for newer SDK versions.

### I-02 `docs(architecture): define finding schema and rule confidence`
**Why:** Avoid untrustworthy alerts. **Depends:** I-01. **Type/complexity:** design / medium.
- [ ] Document schema_version, `Finding` fields, severity/confidence and JSON sample.
- [ ] Explain no false all-clear and CI opt-in behavior.
- [ ] Define fixture provenance requirements and limitation language.

## Phase 8, step 8.1: Scanner core

### I-03 `feat(cli): scaffold scanner command, flags and exit policy`
**Depends:** I-02. **Type/complexity:** feature / medium.
- [ ] `p30check scan <path>` runs without network/target code execution.
- [ ] `--format text|json`, `--fail-on none|high`, `--help` work.
- [ ] Exit codes 0,1,2 documented and tested.

### I-04 `feat(walker): bound and filter repository traversal`
**Depends:** I-03. **Type/complexity:** feature / medium.
- [ ] Ignore large/generated/dependency directories and symlink escapes.
- [ ] Handle invalid paths and unreadable files gracefully.
- [ ] Deterministic file list and unit tests.

### I-05 `feat(report): emit stable human and JSON reports`
**Depends:** I-02,I-03. **Type/complexity:** feature / medium.
- [ ] Stable sorted findings with line/path evidence.
- [ ] Versioned JSON and text renderers, testable injected clock.
- [ ] Clear error for unsupported formats; no fabricated "ready" grade.

## Phase 8, step 8.2: SDK version check

### I-06 `feat(deps): parse Go and JS dependency inventories`
**Depends:** I-04. **Type/complexity:** feature / high.
- [ ] Read `go.mod`, `package.json`, useful lockfiles where available.
- [ ] Handle version ranges and unresolved package versions as unknown.
- [ ] Fixture tests for known versions, aliases, forks, malformed manifests.

### I-07 `feat(rules): implement version/capability matrix`
**Depends:** I-01,I-05,I-06. **Type/complexity:** feature / high.
- [ ] Version-specific CAP-84 and CAP-88 capability fields; source citations.
- [ ] Test JS 16.4.0 (XDR yes/W no), 17.3.0 (both), official Go 0.8.0.
- [ ] Unknown versions produce uncertainty, not invented certainty.

## Phase 8, step 8.3: `W` address check

### I-08 `feat(rules): flag narrow address validation patterns omitting W`
**Depends:** I-04,I-05,I-07. **Type/complexity:** feature / high.
- [ ] Known bad regex/prefix and enum samples produce actionable findings.
- [ ] Known good `W` handling produces no high-confidence finding.
- [ ] Rule documentation includes fix and false-positive cases.

## Phase 8, step 8.4: Ledger XDR and close-time checks

### I-09 `feat(rules): correlate legacy raw-XDR decoding with CAP-88 risk`
**Depends:** I-06,I-07. **Type/complexity:** feature / high.
- [ ] Correlate decoder version with `LedgerHeader`/`LedgerCloseMeta` use.
- [ ] Use sourced CAP-88 union-arm fixtures, not fictional ledger samples.
- [ ] Test modern and legacy decoder examples; uncertain cases downgraded.

### I-10 `feat(rules): identify strict close-time monotonicity assumptions`
**Depends:** I-04,I-05. **Type/complexity:** feature / medium.
- [ ] Only report supported patterns; default to informational confidence.
- [ ] Document why consecutive ledgers can have same whole-second time.

## Phase 8, step 8.5 and Phases 9–11: Verification, release and docs

### I-11 `ci(test): add build, race, vet and fixture matrix`
**Depends:** I-03–I-10. **Type/complexity:** ci / medium.
- [ ] CI uses actual commands/Go version verified against SDK.
- [ ] Tests cover true negatives, false positives, malformed inputs and deterministic reports.
- [ ] Branch protection criteria refer only to actual CI job names.

### I-12 `docs(release): ship quickstart, limitations, demo and tagged release`
**Depends:** I-11. **Type/complexity:** docs/release / medium.
- [ ] README + CONTRIBUTING + SECURITY + rules and CI integration guide.
- [ ] Demo runs on one known-bad and one known-good fixture.
- [ ] Release binaries, checksums, changelog only when tested and approved.

## After MVP

### I-13 `feat(report): SARIF exporter` — depends I-05,I-11; validate SARIF schema and GitHub Code Scanning support.
### I-14 `feat(action): reusable GitHub Action` — depends I-12,I-13; version pinning and least privilege.
### I-15 `test(protocol): post-upgrade Testnet fixture verification` — depends on confirmed Protocol 30 activation; cite ledger sequence/network and source.
### I-16 `research(wave): verify submission eligibility and competition` — no approved status presumed; prepare demo and real issue record before applying.
