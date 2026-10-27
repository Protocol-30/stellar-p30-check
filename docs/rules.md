# Rules

| ID | Checks | Status |
|----|--------|--------|
| [P30-DEP-001](#p30-dep-001-stellar-sdk-version) | Stellar SDK versions in Go and npm manifests | Shipped (Phase 8.2) |
| [P30-ADDR-001](#p30-addr-001-w-address-handling) | Address handling that rejects CAP-84 `W` addresses | Shipped (Phase 8.3) |
| P30-XDR-001 | Ledger-header decoding with a pre-Protocol 30 XDR library | Planned (Phase 8.4) |
| P30-TS-001 | Assumptions of strictly increasing whole-second close times | Planned (Phase 8.4) |

## P30-DEP-001: Stellar SDK version

Compares the official Stellar SDK versions a project uses with a capability
table built from the published packages ([evidence.md](evidence.md), rows
E10–E14). Two capabilities matter for Protocol 30:

- **Ledger XDR** (CAP-88): after the upgrade every ledger header uses a new
  `StellarValue` arm. Libraries without it fail to decode those headers.
- **`W` addresses** (CAP-84): the new muxed-contract strkey.

### Packages and thresholds

| Package | No support | Ledger XDR only | Both |
|---------|-----------|-----------------|------|
| `github.com/stellar/go-stellar-sdk` | before v0.8.0 | — | v0.8.0+ |
| `github.com/stellar/go` (deprecated) | every version | — | — |
| `@stellar/stellar-sdk` | before 16.4.0; 17.0.0–17.2.x | 16.4.x | 17.3.0+ |
| `@stellar/stellar-base` | up to 15.0.0 (latest) | — | — |
| `stellar-sdk`, `stellar-base` (deprecated unscoped) | up to 13.3.0 / 13.1.0 (latest) | — | — |

### Where versions come from

| Source | Confidence |
|--------|-----------|
| `go.mod` `require` (after applying `replace`) | high |
| `package-lock.json` / `npm-shrinkwrap.json` (lockfileVersion 1–3, workspaces, `npm:` aliases) | high |
| `package.json` exact pin such as `"16.4.0"` | high |
| `package.json` range such as `^16.0.0`, `~17.2.0`, `>=16.0.0`, `17.x` — judged at the **lowest** version it allows | low |

A `package.json` entry is skipped when an npm lockfile in the same folder or
any parent folder records that package (the lockfile finding covers it).
`yarn.lock` and `pnpm-lock.yaml` are not read yet; the rule says so with a
`lockfile-not-parsed` warning and falls back to the range.

### Severity

| Situation | Severity |
|-----------|----------|
| No Protocol 30 support | high |
| Ledger XDR only (`@stellar/stellar-sdk` 16.4.x) | medium |
| Pre-release below the threshold, version newer than the last known release of a frozen package, fork/local `replace`, git/URL/tag spec | info, low confidence |

`--fail-on high` therefore fails a build only for an exact, resolved version
with no Protocol 30 support.

### Examples

Flagged (high, high):

```text
require github.com/stellar/go-stellar-sdk v0.7.3
```

Flagged (medium, high):

```json
"dependencies": { "@stellar/stellar-sdk": "16.4.0" }
```

Not flagged:

```json
"dependencies": { "@stellar/stellar-sdk": "^17.3.0" }
```

All cases: [testdata/rules/dep-001](../testdata/rules/dep-001/README.md).

### Fix

- Go: `go get github.com/stellar/go-stellar-sdk@v0.8.0` (needs Go 1.25.0+).
  Code still on `github.com/stellar/go` must move to the new module path.
- JS: `npm install @stellar/stellar-sdk@^17.3.0`. Drop direct
  `@stellar/stellar-base` dependencies; the SDK bundles it.
- Upgrading the SDK does not change your own address validation. Code that
  checks prefixes (`G`, `M`, `C`) needs `W` too; that is P30-ADDR-001's job.

### Known false positives and negatives

- **False positive:** a range such as `^16.0.0` is judged at 16.0.0 even if
  your yarn/pnpm lockfile installs 16.4.1. Confidence is low for that reason.
- **False positive:** an app that only builds and submits transactions and
  never decodes ledger headers or contract addresses may keep working on an
  old SDK. The rule cannot see how the SDK is used.
- **False negative:** SDK copies under `vendor/` or `node_modules/`, bundled
  builds, and git-sourced forks are not version-checked.
- **False negative:** in an npm workspace, one root lockfile entry for a
  package counts for every `package.json` below it.

## P30-ADDR-001: `W` address handling

CAP-84 adds muxed contract addresses. Their strkey starts with `W` and is 69
characters long (SEP-23, [evidence.md](evidence.md) E15). Upgrading the SDK
does not change an application's own validation, so this rule looks for
address checks in Go, JavaScript and TypeScript source that leave `W` out.

### What it recognises

| Shape | Go | JS / TS |
|-------|----|---------|
| Address regex: `^` + prefix class + strkey body class | `regexp.MustCompile` / `Compile` / `MatchString` with a literal | regex literals, `RegExp("...")` |
| SDK checks combined in one expression | `strkey.IsValid*` from go-stellar-sdk (or the old `github.com/stellar/go`) | `StrKey.isValid*` |
| Prefix checks on one value | `strings.HasPrefix(s, "G")`, `s[0] == 'G'`, `s[:1] == "G"` | `s.startsWith("G")`, `s[0] === "G"`, `s.charAt(0) === "G"` |
| Prefix lists | `[]string{"G","C"}`, `[]byte{...}`, map keys, `strings.ContainsRune("GMC", ...)` | `["G","M","C"]`, `"GMC".includes(...)` |
| Address type switches | `switch` with `xdr.ScAddressTypeScAddressTypeContract` or `strkey.VersionByteContract` cases | `switch` mentioning `scAddressTypeContract` |

Go is parsed with the standard `go/parser`, and packages are matched by
their real import path, so a look-alike package named `strkey` is ignored.
JS/TS is read with a tokenizer, not a full parser: strings, comments,
template literals and regex literals are recognised, but expressions are
grouped by statement (`;`, `{`, `}`), not by syntax tree. Nothing in the
scanned code is run, including its regexes.

### When it reports

An allowlist (any shape except switches) is reported only when it accepts
`C` together with `G` or `M`, does not accept `W`, and the enclosing
function does not handle `W` elsewhere. In Go the enclosing function comes
from the syntax tree. In JS/TS it is the outermost brace block around the
check (class bodies are skipped, so each method is its own scope), or the
whole file at top level.

| Situation | Severity | Confidence |
|-----------|----------|-----------|
| Accepts `M` and `C` but not `W`, from an address regex or SDK checks | high | high |
| Accepts `M` and `C` but not `W`, from prefix checks or a prefix list | high | medium |
| Accepts `G` and `C` but not `M` or `W` | info | low |
| Address type switch with a contract case, no muxed-contract case, no `default` | high | medium |
| Same, with a `default` branch | medium | medium |

Accepting `M` and `C` marks a destination field, which is exactly where `W`
applies, so that case is high. A check that accepts only `G` and `C` is often
correct: a contract that will be invoked, an account to fund, or a storage
key must not be `W` (CAP-84). That case is only a hint. `--fail-on high`
fails a build only for the high/high rows.

Not reported: account-only checks (`G`, `M`), contract-only checks (`C`),
anything that already accepts `W`, and test files (`*_test.go`,
`*.test.*`, `*.spec.*`, `__tests__/`).

### Examples

Flagged (high, high):

```ts
const DESTINATION_RE = /^[GMC][A-Z2-7]{55}$/;
```

```go
return strkey.IsValidEd25519PublicKey(s) ||
	strkey.IsValidMuxedAccountEd25519PublicKey(s) ||
	strkey.IsValidContractAddress(s)
```

Not flagged:

```ts
const DESTINATION_RE = /^[GMCW][A-Z2-7]{55}(?:[A-Z2-7]{13})?$/;
```

All cases: [testdata/rules/addr-001](../testdata/rules/addr-001/README.md).

### Fix

- Destinations (payments, SAC transfers): accept `W` with
  `StrKey.isValidMuxedContract` (@stellar/stellar-sdk 17.3.0+) or
  `strkey.IsValidMuxedContractAddress` (go-stellar-sdk v0.8.0+). Regexes need
  a `W` in the prefix class and must allow 69 characters, like `M`.
- Switches: add the muxed-contract case, or make `default` return an error.
- Contract IDs to invoke and storage keys: keep rejecting `W`.

### Known false positives and negatives

- **False positive:** an `M`+`C` check for a field that can never be a muxed
  contract. There is no ignore comment yet (planned).
- **False positive:** in JS, `W` handling in a helper called from the same
  function is not seen; only the function's own text is checked.
- **False negative:** validators built from variables, loops, `switch` on the
  first character, or a regex stored in a constant elsewhere and passed in.
- **False negative:** a regex that lists `W` but only allows 56 characters,
  which still rejects real `W` addresses.
- **False negative:** in JS, `W` handled anywhere in a top-level file
  suppresses top-level allowlists in that file.
- **False negative:** JS/TS checks spread over several statements, such as an
  `if` / `else if` chain.
