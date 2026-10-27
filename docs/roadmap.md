# Roadmap (DripWave phases)

This project follows the DripWave playbook's 13 phases. DripWave was written
for Soroban contract projects, so a few phases change meaning for an offline
CLI: "contract architecture" becomes scanner architecture, "deployment"
becomes building and testing the binary, and "hosting" becomes publishing
release binaries (there is no server).

The coding work is all in **Phase 8**, split into steps 8.1–8.5 so each one
ships something that works and is tested.

| Phase | What it means here | Status |
|-------|--------------------|--------|
| 1–3 Recon, ideas, critical review | Protocol 30 problem verified; this scanner chosen as the project | Done (planning package) |
| 4 Naming and scoping | Working name `stellar-p30-check`; one repository; scope fence in [plan/BUILD_PLAN.md](plan/BUILD_PLAN.md) | Final name is the owner's call |
| 5 Architecture | Pipeline, report schema, exit codes: [architecture.md](architecture.md); sourced facts: [evidence.md](evidence.md) | Done |
| 6–7 Coding-agent prompts | [AGENTS.md](../AGENTS.md) | Done |
| **8 Build and test locally** | Steps below | **In progress** |
| 9 Distribution | Tagged release with binaries and checksums; CI usage guide. No hosting. | Not started |
| 10 Repo hygiene | LICENSE, branch protection on real CI job names, topics, issues from the backlog, release tag | Partly done (CONTRIBUTING, SECURITY, CI workflow) |
| 11 Documentation | Rule catalog and CI guide; a docs site only if the README stops being enough | Not started |
| 12 Wave submission | Re-check Drips Wave eligibility and the approved-repo list, then submit | Not started |
| 13 Iteration | Testnet feedback from the Nov 3 vote, Mainnet hardening before the Nov 23 vote, SARIF / GitHub Action | Not started |

## Phase 8 steps

| Step | Delivers | Done when | Status |
|------|----------|-----------|--------|
| 8.1 Scanner core | `p30check scan`, safe file discovery, text/JSON reports, exit codes | Builds; tests pass; two fixture reports are byte-for-byte repeatable | Done |
| 8.2 SDK version check (`P30-DEP-001`) | Reads `go.mod`, `package.json`, npm lockfiles; flags Stellar SDK versions without Protocol 30 support | Tells JS 16.4.x (ledger XDR yes, `W` no), JS 17.3.0+ (both) and Go v0.8.0 apart, with fixtures | Done |
| 8.3 `W` address check (`P30-ADDR-001`) | Flags address validation that rejects CAP-84 `W` addresses | Catches rejecting samples; safe samples are not flagged high | Done |
| 8.4 Ledger XDR and close-time checks (`P30-XDR-001`, `P30-TS-001`) | Links old XDR libraries to code that decodes ledger headers; low-confidence close-time hints | At least two CAP-88 fixtures from the spec; no blanket claims | Next |
| 8.5 Verification | Integration tests, malformed-input tests, scanner run on its own repo | Green CI; no high-confidence findings on clean fixtures | Not started |

Dates are planning targets, not promises. Stellar's November 3 (Testnet) and
November 23, 2026 (Mainnet) dates are upgrade **votes**, not guaranteed
activation dates.

## Old labels

Earlier documents used milestone numbers. They map as follows:
M0 = Phases 1–5, M1 = step 8.1, M2 = 8.2, M3 = 8.3, M4 = 8.4, M5 = 8.5,
M6 = Phases 9–11, M7–M8 = Phase 13.
