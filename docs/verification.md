# Verification (Phase 8.5)

What was checked before closing Phase 8, how, and what it showed. Dates
are 2026-10-27; timings are from one Linux development container and are
indicative, not benchmarks.

## Results on real code

The rules were run on published SDK packages. The pre-Protocol 30 releases
are expected to trip the rules; the current releases are not.

| Package | P30-ADDR-001 | P30-XDR-001 | P30-TS-001 |
|---------|--------------|-------------|------------|
| go-stellar-sdk v0.7.3 | 10 (4 high/medium, 6 medium/medium) | 8 (2 high/high, 4 high/medium, 2 medium/medium) | 0 |
| go-stellar-sdk v0.8.0 | 0 | 0 | 0 |
| @stellar/stellar-sdk 16.3.1, 16.4.0 | 8 medium, 4 info | 0 | 0 |
| @stellar/stellar-sdk 17.3.0, 17.4.0 | 4 info (`fundAddress`, correctly G or C only) | 0 | 0 |

Every v0.7.3 finding is at code that v0.8.0 changed for Protocol 30. See
[evidence.md](evidence.md) E17 and E22.

## Untrusted input: time and crashes

Scanned repositories are untrusted, and a CI step must not hang on them.
While verifying, six inputs of 1 MB or less were found that made the
JavaScript/TypeScript matchers or the Go scope checks super-linear:

| Input | Before (on `main` at a375c58) | After |
|-------|-------------------------------|-------|
| nested `a / ( a / ( …` | over 60 s | 0.13 s |
| nested `switch (a) {` | over 60 s | 0.17 s |
| nested `((a.closeTime < b.closeTime) < …` | over 60 s | 0.58 s |
| nested `class A {` around an address check | over 60 s | 0.35 s |
| one Go function with 20,000 `StellarValueType` comparisons | over 60 s | 0.06 s |
| one Go function with 20,000 prefix checks | over 60 s | 0.08 s |

The "after" times are for the test-sized inputs in
`internal/rules/perf_test.go`; the same inputs time out after 20 s with the
old code.

The fixes:
- **JS/TS index:** each file is tokenized once into an index of matching brackets, enclosing groups and per-token counts, so every lookup the matchers make is O(1) (`internal/rules/jsindex.go`).
- **Go scope checks:** the "does this function also handle W / the `_MS` arms" check is memoised per function, and the function is found by binary search (`internal/rules/gomentions.go`).

`TestAdversarialInputsStayLinear` keeps all nine adversarial shapes under
10 s, or 60 s under the race detector.

Fuzzing found two crashes in Phase 8.3 (a file ending in `StrKey.`, a lone
backtick); both were fixed and kept as regression inputs. In 8.5 the whole
pipeline is fuzzed with `FuzzCheckAll`, which feeds the same bytes to every
rule as a Go file, a TS file, `go.mod`, `package.json`, an npm lockfile and
an `.x` file. Runs:

| Fuzzer | Executions | Crashes |
|--------|-----------|---------|
| `FuzzCheckAll` | about 720,000 in 90 s | 0 |
| `FuzzJSAddressObservations`, `FuzzGoAddressObservations`, `FuzzLedgerObservations`, `FuzzRegexAllowlist` | 30 s each after the refactor | 0 |

`TestMalformedInputs` covers specific hostile files through the full scan:
- binary data under `.go`, `.ts`, `.x`, `go.mod` and `package.json` names;
- invalid UTF-8;
- a byte order mark with CRLF lines;
- empty manifests;
- a lockfile nested past the JSON decoder's depth limit.

The scan completes and gives identical results twice. Unreadable manifests
and Go files become warnings, and the odd but valid files are still
checked, at the right lines.

## One read and parse per file

Source rules (P30-ADDR-001, P30-XDR-001, P30-TS-001) now run in one shared
pass (`rules.CheckAll`):
- each source file is read, parsed or tokenized once;
- the result is handed to every source rule, then dropped, so only one parsed file is held at a time;
- each rule's own `Check` still works on its own, using the same path.

Checks:
- `TestCheckAllReadsEachSourceOnce` counts reads.
- `TestCheckAllMatchesSeparateChecks` compares the shared pass with each rule's own `Check` on every fixture directory.
- Before and after the change, full JSON reports (timestamp excluded) were byte-identical on the six real SDK packages, this repository and the adversarial inputs the old code could finish.
- Scanning go-stellar-sdk v0.7.3 (657 Go files) went from about 0.85 s to 0.43 s.

## End to end

- **Integration fixtures** (`testdata/integration`):
  - `legacy-app` (Go indexer on v0.7.3, TS app on 16.3.1) produces 8 findings covering all four rules, and `--fail-on high` exits 1.
  - `modern-app`, the same project after the fixes, produces no findings or warnings and exits 0.
  - Both reports are pinned as golden files.
- **Self-scan:** `TestSelfScan` scans this repository and requires every finding and file warning to be under `testdata/`. The scanner's own code is clean.
- **Toolchains:** `go vet`, `go test`, `go test -race` and `go build` pass on Go 1.25.0, 1.26.10 and 1.27.3; CI runs Go oldstable and stable on Ubuntu and macOS.

## Not verified

- Behaviour against real repositories other than the SDKs themselves.
- Ledgers captured from Testnet after the Protocol 30 vote. They do not exist yet; this is Phase 13 work ([#10](https://github.com/Protocol-30/stellar-p30-check/issues/10)).
- Windows. CI covers Linux and macOS only ([#12](https://github.com/Protocol-30/stellar-p30-check/issues/12)).
