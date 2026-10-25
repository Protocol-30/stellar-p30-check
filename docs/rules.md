# Rule roadmap

**Implemented compatibility rules: zero.** M1 returns inventory and limitations, not synthetic findings. IDs below are reserved design proposals.

| Proposed ID | Planned evidence | Requirements before a high-confidence finding |
|---|---|---|
| P30-DEP-001 | Effective SDK identity/version and affected role | Concrete effective version and verified capability; declared minimums or stale checksum entries alone are insufficient |
| P30-ADDR-001 | Local rejection in a W-capable address flow | Establish that the field/API accepts muxed contract addresses; preserve account-only, AddressObject, and storage-key negative cases |
| P30-XDR-001 | Legacy decoder plus relevant raw-ledger use | Correlate decoder selection and use within the same module/application scope |
| P30-TS-001 | Whole-second close-time monotonicity assumptions | Pattern-based evidence remains low confidence unless contextual analysis supports more; use valid confidence enums |

## Exact next increment: M2

1. Define dependency inventory records distinguishing declared constraints, lockfile-resolved versions, effective versions, module/workspace identity, source location, and uncertainty.
2. Parse Go manifests statically. Handle `replace`, version replacements, local replacements, and workspace overrides where the available data is sufficient. Treat unresolved minimal-version selection and unavailable graph information as unknown. Never run `go list`, target builds, package-manager scripts, or network fetches during a scan. Do not treat go.sum as a selected-version list.
3. Parse JS package manifests and explicitly supported npm/yarn/pnpm lock formats. Keep ranges, aliases, forks, malformed files, conflicts, and missing locks distinct from resolved official versions. Unsupported lock formats produce an explicit limitation.
4. Add a versioned, evidence-linked capability matrix for verified official SDK families. Model CAP-88 XDR, muxed-contract XDR, and high-level W helpers separately. Fixtures must distinguish JS 16.4.0, JS 17.3.0, and Go SDK 0.8.0.
5. Emit P30-DEP-001 only from adequate scoped evidence. Unknown versions/replacements become uncertainty, not fabricated compatibility or incompatibility. Add positive and negative tests for every classification, including transitive selection uncertainty, replacements/workspaces, and stale checksums.
6. Extend the report contract deliberately and run the full existing discovery/report/CLI suite plus targeted dependency tests, race tests, vet, and build. Update evidence checked dates. Stop before M3 address heuristics.

M3/M4 require separately reviewed source-context and fixture-provenance contracts. No captured post-upgrade ledger fixtures are available in this increment.
