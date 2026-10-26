# Rules and next increment

## Implemented: P30-DEP-001

M3 retains this conservative dependency-capability rule alongside P30-ADDR-001. It uses scoped static inventory and the evidence-linked exact-version matrix `2026-10-25.1`, checked 2026-10-25. Capability states are `supported|unsupported|unknown`.

| Canonical identity / exact version | CAP-88 XDR | Muxed-contract XDR | High-level W helpers |
|---|---|---|---|
| `@stellar/stellar-sdk` 16.3.1 | unsupported | unsupported | unsupported |
| `@stellar/stellar-sdk` 16.4.0 | supported | supported | unsupported |
| `@stellar/stellar-sdk` 17.3.0 | supported | supported | supported |
| `github.com/stellar/go-stellar-sdk` v0.8.0 | supported | supported | supported |

These are observed-version capabilities, not a target compatibility grade. No extrapolation is made to other releases, prereleases, metadata variants, forks, aliases, legacy package identities, or community SDKs. Unknown canonical versions receive unknown capabilities and warnings; unsupported identities remain unverified. Sources and verification limits are in [evidence.md](evidence.md).

### Finding contract

A record must have supported nominal lock/declaration agreement, a canonical official SDK name/version in metadata, no uncertainty, and at least one verified missing capability. This classifies the named official release conditionally on the metadata identity; it does not verify registry routing, artifact origin, or installation. One P30-DEP-001 finding lists the missing capabilities for that resolved declaration/group. Evidence includes its source path, group/scope, declaration, observed version, and source. An observed version's annotation alone is insufficient.

Severity is **medium** and confidence **high**. The missing capability is verified, but M2 does not prove the application depends on that feature. These findings do not trigger `--fail-on high`. A supported matrix row or no findings never proves that custom target code is compatible.

Examples of intended outcomes:

- Declaration-compatible JS 16.3.1 lock metadata with the expected registry URL: one medium/high finding naming the missing capabilities.
- Equivalently supported nominal JS 16.4.0: one medium/high finding for missing high-level W helpers, not missing XDR support.
- Equivalently supported nominal JS 17.3.0: no missing-capability finding; only the separate supported M3 local address patterns are analyzed.
- Go v0.8.0 in a requirement/replacement: observed-version annotation, but no finding or effective-selection claim from manifest text.
- Unknown version, alias, fork, local/workspace replacement, stale checksum, or lock with insufficient source-URL metadata: explicit uncertainty/warnings, not invented incompatibility or compatibility.

## Implemented: P30-ADDR-001

This rule detects a **supported local contradiction**: a guard rejects W immediately before an imported W-only helper consumes the same simple parameter. A prefix/regex match by itself is not a finding. The analyzer does not execute target code/regexes, compile/type-check the target, or attempt general dataflow analysis.

### Evidence and confidence

| Supported context | Severity | Confidence | Reason |
|---|---|---|---|
| Unambiguous JS/TS W-only helper flow plus uncertainty-free nominal SDK 17.3.0 W support from the nearest current package | high | high | Supported local guard contradicts the immediately following W-only helper |
| Supported JS W-only flow with unresolved/unverified nearest SDK selection | medium | medium | Local flow exists, selected W-capable SDK evidence is incomplete |
| Supported Go W-only flow | medium | medium | Go declared/replacement versions do not prove selected build version |
| Broader JS SAC instance destination flow | medium | low | API permits W, but account-only application policy may be intentional |

Nominal JS selection is the M2 metadata contract, not proof of installed artifact identity or all runtime behavior. An unrelated parent/sibling manifest cannot provide source-package certainty. Unsupported/control-flow-ambiguous contexts do not become high findings. A supported high/high finding exits 1 only when `--fail-on high` is chosen; the default gate still exits 0 on a completed scan.

### Supported JS/TS shapes

JS, TS, JSX, and TSX are parsed with pinned official Tree-sitter grammars. Supported imports are ESM named imports (including aliases) or namespace imports from `@stellar/stellar-sdk` or its `/base` subpath. Default/CommonJS and type-only imports are not analyzed as proof. Unrelated/fork, missing, shadowed, mutated, or otherwise ambiguous imports cannot prove a helper binding.

The intentionally small function subset is a top-level function declaration, block-bodied arrow, or function expression with simple identifier parameters (required annotated identifiers in TS; no defaults, rest, destructuring, or optional parameters), followed by a supported rejection guard and direct return of the same value into `StrKey.decodeMuxedContract` or `StrKey.isValidMuxedContract`. There is no interprocedural/alias tracing. For the broader SAC hint, the supported extra setup is a local `const` instance of the imported TransactionBuilder, or a directly constructed instance, followed by guard and return. The destination is the **first positional argument** of `builder.addSacTransferOperation(destination, asset, amount, ...)`, not an arbitrary property or another argument.

Guard proof recognizes supported literal startsWith, strict first-character `[0]`/`.charAt(0)` equality/inequality, fresh literal-array includes checks, and fresh anchored literal-regex tests combined through pure `!`, `&&`, and `||` expressions. A rejecting branch is a sole throw or supported false/null/unshadowed-undefined return. The entire guard must fit the supported pure grammar; an unknown side-effecting call does not become safe because another boolean arm rejects W. Shared anchored-regex reasoning requires a literal leading `^` and supports only its conservative literal subset (no flags or case-insensitive `i`). At the JS proof boundary, POSIX named classes, Go-style brace-hex/property escapes, and other semantically mismatched constructs are excluded. Dynamic/stateful/unsupported regex syntax cannot support proof; valid Go-only regex semantics are not silently reused for JS. Ordinary hex/four-digit Unicode string escapes are supported; legacy/braced/continuation forms are not guessed. Obvious SDK mutation/escape, relevant built-in prototype writes, and dynamic eval/Function constructs suppress proof. Arbitrary source text that resembles a regex/call is not executable evidence.

### Supported Go shapes

The Go subset is a named, nonmethod, nongeneric function with exactly one simple `string` parameter and exactly two body statements: a rejection `if`, then direct `return` of a W-only helper. The `if` has no initializer/else and a single rejecting action. Accepted rejecting actions are a literal panic, `return false` for the boolean helper, or `return nil` plus standard `errors.New`/`fmt.Errorf` for a decoder.

The exact imported strkey package is `github.com/stellar/go-stellar-sdk/strkey` (named import aliases are allowed; dot/fork imports are not proof). W-only helper forms are `DecodeMuxedContract(value)`, `IsValidMuxedContractAddress(value)`, and `Decode(strkey.VersionByteMuxedContract, value)`. Prefix predicates cover imported standard `strings.HasPrefix` with a literal, supported first-byte/one-character slice comparisons, and direct `regexp.MustCompile(literal).MatchString(value)`. Pure boolean combinations are supported; unknown calls/mutation do not support proof. Conservative declaration/assignment checks can suppress an entire binding on ambiguity.

### Required negatives and false negatives

No generic G/M/C whitelist is automatically wrong. Classic account/payment G/M fields, C-only decoders, Go `MuxedContract.SetContractID`, AddressObject inputs, and contract storage keys remain required negative contexts. Supported W-accepting modern guards remain negative. Comments/string literals, wrong parameters, type-only/shadowed imports, reassignment, nested closures, exceptions/bypasses, and unsupported flows cannot establish the local contradiction.

A broad SAC destination hint must not be presented as a definite need to accept W. Confirm the input domain before remediation. Never expand every validator or remove required account/storage restrictions.

Unsupported syntax adds a source-path limitation and no source finding. Parser/node-budget exhaustion makes the scan incomplete/exit 2. Successfully parsed but unsupported patterns are deliberate false negatives; the scanner does not enumerate every unrecognized flow. See [limitations](limitations.md) for bounds and coverage meaning.

## Next increment: M4 XDR and timestamp analysis

M4 implements P30-XDR-001 and P30-TS-001 within a deliberately bounded subset.

- **JS/TS XDR:** top-level named functions with a single direct return of a named ESM-imported `xdr.LedgerHeader.fromXDR(parameter, format?)` or `xdr.LedgerCloseMeta.fromXDR`. Aliases work; namespace/CommonJS, indirect helpers and variable setup are outside this subset. Import shadowing/mutation/escape and duplicate bindings suppress matches. The nearest package manifest scopes capability evidence.
- **Go XDR:** single-return named nonmethod/nongeneric functions calling the official `xdr.SafeUnmarshalBase64` with a pointer parameter typed as LedgerHeader or LedgerCloseMeta. Local-variable setup and general interface/dataflow resolution are not implemented.
- Exact modern observed capability rows suppress the decoder warning, without certifying effective runtime selection. Existing dependency warnings retain selection uncertainty. Known scoped nominal legacy selection yields medium/high-confidence XDR review guidance. Other direct supported uses yield medium/low-confidence guidance. Historical-only decoding may be intentional; no M4 finding is high severity.
- **Timing:** direct strict `<`/`>` comparisons between distinct typed Go LedgerHeader value parameters' `ScpValue.CloseTime` fields yield info/low-confidence P30-TS-001. Non-strict, unrelated clocks, millisecond fields and arbitrary word matches are excluded. No consecutive-ledger relationship or runtime bug is inferred.
- CAP-88 schema fixtures are pinned official source excerpts, not captured or serialized network ledgers. No target code, regex, decoder, or build command is executed. Source syntax notices and resource failures retain the M3 semantics and budgets. Schema remains2, four rules are available.

## Next increment: M5 verification and CI hardening

Broaden integration/adversarial coverage where evidence warrants it, assess distribution targets, and finish remaining verification. The vulnerability audit remains incomplete until database access works. Do not infer release readiness or program eligibility from local tests. SARIF, reusable actions, deployment and publication remain outside M4.

## Next increment: M3 W-address analysis

This anchor is retained for the historical [M2 report](m2-build-report.md). M3 is now the implemented increment above; it is no longer the next work item. [m3-build-report.md](m3-build-report.md) records validation and boundaries.

## Exact next increment: M2

This anchor is retained for the historical [M0/M1 report](build-report.md). M2 is retained in the current implementation; its [build report](m2-build-report.md) describes that earlier increment.
