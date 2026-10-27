## What and why

<!-- What changes, and the issue it addresses. -->

## Checks

- [ ] `gofmt -l cmd internal` prints nothing; `go vet ./...`, `go test ./...` and `go test -race ./...` pass
- [ ] Rule changes: positive and negative fixtures, and any new fact recorded in `docs/evidence.md` with a source and date
- [ ] New matcher input shapes added to `TestAdversarialInputsStayLinear`
- [ ] Golden files regenerated and reviewed if report output changed on purpose
- [ ] `docs/rules.md`, `docs/limitations.md` and `CHANGELOG.md` (Unreleased) match the change
