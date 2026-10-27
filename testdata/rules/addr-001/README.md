# P30-ADDR-001 fixtures

Each directory is a small set of source files for the `W` address check.
The code is invented; SDK function names come from go-stellar-sdk v0.8.0
and @stellar/stellar-sdk 17.3.0 (`docs/evidence.md` E16), and the `W`
address in `js-modern` is the SEP-23 test vector (E15). Expected results are
in `internal/rules/address_test.go`.

| Directory | Case | Expected |
|-----------|------|----------|
| js-regex | `/^[GMC][A-Z2-7]{55}$/` | high, high |
| js-predicates | `StrKey.isValid*` for G, M, C | high, high |
| js-prefix-tests | `startsWith` G, M, C in JSX (with an apostrophe in JSX text) | high, medium |
| js-letter-list | `["G", "M", "C"].includes(addr[0])` | high, medium |
| js-account-contract | `RegExp("^[GC]...")` | info, low |
| js-switch | `ScAddress` switch, contract case, `default` | medium, medium |
| js-class-methods | G, M, C check in one method; W handled in another | high, high |
| go-regex | `regexp.MustCompile` with `^[GMC]` | high, high |
| go-predicates | aliased `strkey` import, G, M, C | high, high |
| go-switch | `ScAddress` switch, contract case, no `default` | high, medium |
| go-prefix | `HasPrefix(s, "G") \|\| s[0] == 'C'` | info, low |
| js-modern, go-modern | the same shapes with `W` | none |
| js-account-only | G and M only | none |
| js-contract-only | C only (contract to invoke) | none |
| js-unrelated | grade letters, `/^[A-C]\d+$/`, division | none |
| js-test-files | narrow regex inside test files | none (skipped) |
| js-w-same-function, go-w-same-function | G, M, C branch with W handled in the same function | none |
| go-lookalike | same names from non-Stellar packages | none |
| go-broken | syntax error | `source-unparsed` warning |
