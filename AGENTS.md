# AGENTS.md — Codex engineering instructions

Adapted from the user-provided DripWave skill. Treat these instructions as engineering constraints, not a license to skip evidence or approvals.

## Goal and scope
Build an **offline, read-only Go CLI** for repo-level Stellar Protocol 30 compatibility signals. Do not create or deploy Soroban smart contracts, a web frontend, API, PostgreSQL database, or live indexing service for v0.1. The MVP lives in one repository in its own GitHub organization once the owner creates/authorizes it.

## Non-negotiables
1. **Verify ecosystem facts live** using official Stellar documentation/releases and record URLs + checked dates. Do not invent SDK versions, package paths, XDR fixtures, or GitHub CI job names.
2. **Use actual repo state**, not assumed file structures. Read `docs/plan/BUILD_PLAN.md`, `docs/evidence.md`, the tree, `go.mod`, and existing tests before editing.
3. **Make correctness falsifiable.** Every rule needs at least one positive and one negative fixture. Differentiate identified mismatch, probable risk, and unverified suspicion.
4. **Ship complete working increments.** No fake implementations, silent stubs, placeholders in user-facing executable paths, or "all clear" based on unsupported inference.
5. **Respect permissions.** Never create an organization, repository, remote, deployment, publish release, or push without a configured user-authorized destination. Approval controls override any automatic-push convention.
6. **Use precise language**: no exaggerated security/compatibility claims. The tool performs static checks; it does not certify runtime safety.
7. **Avoid code execution of scan targets.** Paths and source text are untrusted. No uploading contents, no symlink escape, no dynamic evaluation.

## Implementation standards
- Go as the main implementation language. Verify Go toolchain and Stellar SDK package versions before pinning them.
- Prefer Go standard library and minimal dependencies; wrap errors with context; use dependency injection for clock/filesystem where testing requires it.
- API design: `scan -> discovery -> rules -> findings -> reporters -> exit policy`.
- All findings carry rule ID, filename/line evidence, severity, confidence, remediation and sources. Stable sorting and versioned JSON.
- CLI defaults are local/offline, text output, zero exit on warnings unless user opts into a high-confidence CI gate.
- Tests use `testdata/` with fixtures grounded in actual relevant CAP-84/88 XDR specifications and documented source patterns.
- Run targeted tests when editing rules, then `go test ./...`, `go test -race ./...`, `go vet ./...`, build; report executed commands and results honestly.
- Stage only intended paths (`git add <explicit paths>`) after scaffold, use conventional commits (`feat(scope): message`); one coherent logical unit per commit. Push each **approved** commit only when a permitted remote exists. Never force push without authorization.
- Keep README, rules, limitations, CI usage, CONTRIBUTING, SECURITY and CHANGELOG accurate to shipped functionality.

## DripWave mapping
- Phase 1–3: verify ecosystem landscape and critically assess need; already provisionally selected, continue technical recon without inventing uniqueness.
- Phase 4: finalize product boundaries and proposed naming; owner controls final name.
- Phase 5: CLI modules, rule model, fixture/test contracts replace Soroban contract architecture.
- Phases 6–7: `CODEX_TASK.md` and scoped follow-up tasks replace Claude-specific contract/app prompts.
- Phases 8–9: local binary build, CI packaging, distribution; no deployment unless a later use case requires it.
- Phases 10–11: repo quality, docs site only if it adds value after v0.1.
- Phases 12–13: confirm current Drips Wave eligibility before submission; keep issue tracker honest and maintain the project.

## Repository map
- Planning snapshot: `docs/plan/BUILD_PLAN.md`, `docs/plan/BACKLOG.md` (the package's `SOURCES.md` is superseded by `docs/evidence.md`).
- Architecture, report schema, exit codes: `docs/architecture.md`. Limitations: `docs/limitations.md`.
- The original DripWave skill is not committed here; ask the owner for it if a process question needs it.

## Where the project is
Progress is tracked by DripWave phase in `docs/roadmap.md`. Phases 1–8 are done: all four rules ship and Phase 8.5 verification is recorded in `docs/verification.md`. Phase 9 is done: v0.1.0 was published on 2026-10-27 through `.github/workflows/release.yml` (see `docs/release.md`). The name (`stellar-p30-check`, binary `p30check`), license (Apache-2.0) and default branch (`main`) are settled. Phase 10 (repo hygiene) is in progress: templates, Dependabot and the backlog issues are in the repo; the owner applies the settings in `docs/maintaining.md` (rulesets on the real CI job names, topics, private vulnerability reporting). Post-release work is tracked as GitHub issues. This session's credentials cannot push tags; the owner pushes release tags. Never push a tag or publish a release without the owner's explicit go-ahead. Any new rule logic must rest on facts recorded in `docs/evidence.md`, with positive and negative fixtures, and every matcher must stay linear on untrusted input (`TestAdversarialInputsStayLinear`).
