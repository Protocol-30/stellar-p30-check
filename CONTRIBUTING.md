# Contributing

## Before you open a PR

```sh
gofmt -l cmd internal      # must print nothing (testdata holds scan targets)
go vet ./...
go test ./...
go test -race ./...
go build ./cmd/p30check
```

Code that reads scanned files must not panic on any input. After changing
the JS/TS tokenizer or the address matchers, run the fuzzers for a while:

```sh
go test ./internal/rules -run=NONE -fuzz=FuzzJSAddressObservations -fuzztime=60s
go test ./internal/rules -run=NONE -fuzz=FuzzGoAddressObservations -fuzztime=60s
go test ./internal/rules -run=NONE -fuzz=FuzzLedgerObservations -fuzztime=60s
go test ./internal/rules -run=NONE -fuzz=FuzzCheckAll -fuzztime=60s
```

Matchers must stay linear in the file size, whatever the nesting: use the
bracket index in `jsindex.go` instead of rescanning token ranges, and
`goMentions` instead of re-walking a function per finding. Add any new
input shape that could blow up to `TestAdversarialInputsStayLinear`.

Crashers land in `internal/rules/testdata/fuzz/`; commit them with the fix.

If you change report output on purpose, regenerate the golden files and
review the diff:

```sh
go test ./internal/cli -run TestGoldenReports -update
git diff testdata/golden
```

## Rules for rules

- Every rule fact (SDK version threshold, XDR arm, strkey prefix) needs a row
  in [docs/evidence.md](docs/evidence.md) with a primary source and date.
- Every rule needs at least one positive and one negative fixture under
  `testdata/`. Fixtures must not be presented as real ledger data unless they
  are, with network and ledger sequence cited.
- Set confidence from the evidence: `high` only when the issue is directly
  established (e.g. a resolved package version below a verified threshold);
  suspected patterns are `medium` or `low`.
- Document known false positives and negatives in
  [docs/limitations.md](docs/limitations.md).
- Prefer three well-tested rules over many shallow ones.

## Code

- Standard library first; justify any new dependency in the PR.
- Never execute, import or evaluate scanned code; read files only through
  `rules.Input.ReadFile`.
- Wrap errors with context. Keep output deterministic.

## Commits

Conventional commits (`feat(walker): ...`, `fix(report): ...`,
`docs(evidence): ...`), one logical change per commit.
