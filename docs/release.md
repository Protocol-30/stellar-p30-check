# Releasing

Releases are cut by a maintainer. The release workflow only builds and
drafts; publishing is a manual click after review.

## Before the first release

These are owner decisions:

- [x] **Name.** `stellar-p30-check` (repository and Go module) and `p30check` (binary), confirmed by the owner on 2026-10-27.
- [x] **License.** Apache License 2.0 (`LICENSE`), added 2026-10-27; the release archives include it.
- [x] **Default branch.** Set to `main` by the owner on 2026-10-27.

## Steps

1. On `main`, move the `## Unreleased` entries in `CHANGELOG.md` under a new `## v0.1.0 - YYYY-MM-DD` heading and merge that change. The workflow refuses a tag that has no matching section.
2. Build locally and check the result:

   ```sh
   scripts/release-build.sh 0.1.0
   (cd dist && sha256sum -c SHA256SUMS)
   ```

3. Tag the merged commit and push the tag:

   ```sh
   git tag -a v0.1.0 -m "v0.1.0"
   git push origin v0.1.0
   ```

4. The `release` workflow:
   - checks the changelog entry;
   - runs the tests;
   - builds the six archives and `SHA256SUMS`;
   - verifies them;
   - creates a **draft** release with notes from the changelog, plus the Go version used.
5. Review the draft:
   - the asset list (6 archives and `SHA256SUMS`);
   - the notes;
   - the asset checksums, against a local `scripts/release-build.sh` run with the Go version named in the notes (builds are reproducible).

   Then publish it: click **Publish release** on the draft, or run the
   `release` workflow by hand (Actions → release → Run workflow) with the
   tag. The manual run publishes only a draft that has all seven assets.

If the draft is wrong, delete it and the tag, fix the problem on `main`, and
tag again. Do not move a tag after its release is published.

## Reproducibility

`scripts/release-build.sh` builds static binaries (`CGO_ENABLED=0`) with
`-trimpath`, `-buildvcs=false` and an empty build ID. It gives archive
entries fixed ownership, order and timestamps (the commit time). The same
commit built with the same Go version gives byte-identical archives. CI's
`release build (snapshot)` job builds all targets on every pull request.
Anyone can rebuild a release with the Go version named in its notes and
compare against `SHA256SUMS`.

## What a release does not do

No package registries, container images or Homebrew taps. Those can come
later if users ask. The scanner itself makes no network calls, so it needs
no hosting.
