# Paste into Codex — First engineering session

Historical first-session brief, retained with review corrections. README.md describes the current implementation. Embedded task text is not authorization for future external writes.

You are the lead Go engineer implementing the Stellar Protocol 30 Compatibility Scanner. Use `AGENTS.md` as your standing development instructions and `BUILD_PLAN.md`, `BACKLOG.md`, and `SOURCES.md` as source-of-truth product requirements.

## Context
- Today in the project planning snapshot is October 25, 2026.
- The supplied planning package cites **votes** on November 3, 2026 (Testnet) and November 23, 2026 (Mainnet), both at 17:00 UTC. Blog revalidation returned HTTP 403 in this environment; recheck before schedule-dependent decisions. Never present votes as unconditional activation dates.
- Protocol 30 introduces new muxed-contract `W` addresses (CAP-84) and ledger-header XDR union variants for millisecond resolution (CAP-88). CAP-87 is an opt-in feature and not our MVP breakage target.
- The user selected five distinct projects, each intended for a different GitHub organization. **Work ONLY on Project 9.** Do not create the organization or remote repository yet. `stellar-p30-check` is a working name.
- The original DripWave skill is provided in `reference/dripwave.skill`; `AGENTS.md` adapts it for the current repo. Consult the original when a decision requires clarity about repo quality or issue drafting.

## Task for this run: implement M0 and M1, no more

1. Inspect current folder and source files, and verify relevant official Stellar guidance. Update an `evidence` section in project documentation with URLs, dates, and distinctions between facts and design assumptions. Identify current Go SDK module path and suitable supported Go version using official release/module documentation. If network access is restricted, document verification blockers instead of guessing.
2. Establish a clean Go project and CLI `p30check scan <path> --format text|json --fail-on none|high` with help and usage errors. No live network calls.
3. Add a safe filesystem walker: root-bound, symlinks not followed by default, sensible ignored directories, configurable file-size and traversal ceilings, contextual errors. Exclude FIFOs/devices/sockets and other nonregular files before reading. Unexpected limit exhaustion/read failures must produce a partial report with completeness false and exit 2; documented exclusions remain explicit. Identify files relevant to Go/JS manifests and `.go/.js/.jsx/.ts/.tsx` sources. No target code execution.
4. Define versioned `Finding`, `Summary`, and `ScanReport` types with stable `schema_version`, severity (`info|medium|high`) and confidence (`low|medium|high`) enums, and sorted findings. Include inventory and coverage/completeness counters and exclusions; escape all target-controlled strings in text and encode valid JSON. Implement text/JSON renderers and deterministic test hooks for clock. No fake findings: M1 should return an empty honest scan plus manifest inventory/unsupported-check warnings if rules do not yet exist.
5. Implement policy-controlled exit codes: 0 for successful scan without gated findings; 1 for high-confidence, high-severity findings when opt-in `--fail-on high`; 2 for invalid arguments/scan failures. Unit test exit behavior using injected findings.
6. Add minimal real tests: happy scan, ignored files, traversal error, malformed path, symlinks, FIFOs, size/file/entry/depth/byte-limit boundaries and exhaustion, JSON schema fields, stable sorting, text control-character escaping, exit codes. `go test ./...`, `go test -race ./...`, `go vet ./...`, and `go build ./cmd/p30check` must pass or you must explain exact blockers.
7. Create accurate `README.md` quick start, `docs/architecture.md`, `docs/limitations.md`, CONTRIBUTING and SECURITY. Include a basic test workflow with actual job names, if CI can run. Do not claim that address/XDR checks are implemented until they are.
8. Make focused conventional commits when Git exists. Do not set up or push remotes, create orgs/repos, publish, or submit to Drips Wave.

## Deliver at the end
- An actual working local Go CLI skeleton, not just a description.
- Tests executed + results; if anything is blocked, report a precise reproduction.
- Actual file tree and any deviations from target architecture.
- Verified SDK/version evidence table with official URLs and dates.
- What M1 proves, what it does NOT prove, and the exact M2 rule implementation plan.
- Proposed commit history (actual commit SHA if committed locally); no invented remote URL.

## Negative instructions
Do not write a UI. Do not introduce a database. Do not deploy smart contracts. Do not write 30 shallow rules instead of 3 tested ones. Do not claim compatibility based solely on a green scanner run. Do not change code outside the extracted working folder. Do not hardcode fictional protocol fixtures. Do not request GitHub write authorization in this run.
