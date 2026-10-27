#!/usr/bin/env bash
# Builds release archives and SHA256SUMS into dist/.
#
#   scripts/release-build.sh 0.1.0
#
# Builds are reproducible: static binaries (CGO off), -trimpath, no VCS
# stamping, and archive entries with fixed owners, order and timestamps
# taken from the last commit. The same commit gives byte-identical output.
set -euo pipefail

version="${1:?usage: scripts/release-build.sh <version, e.g. 0.1.0>}"
version="${version#v}"
case "$version" in
  *[!0-9A-Za-z.+-]*|"") echo "invalid version: $version" >&2; exit 2 ;;
esac

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"
out="$root/dist"
rm -rf "$out"
mkdir -p "$out"

epoch="${SOURCE_DATE_EPOCH:-$(git log -1 --format=%ct)}"
targets="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64"
ldflags="-s -w -buildid= -X github.com/Protocol-30/stellar-p30-check/internal/cli.Version=${version}"

for target in $targets; do
  goos="${target%/*}"
  goarch="${target#*/}"
  name="p30check_${version}_${goos}_${goarch}"
  stage="$out/stage/$name"
  mkdir -p "$stage"
  bin="p30check"
  [ "$goos" = "windows" ] && bin="p30check.exe"

  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -buildvcs=false -ldflags "$ldflags" -o "$stage/$bin" ./cmd/p30check

  cp README.md CHANGELOG.md "$stage/"
  [ -f LICENSE ] && cp LICENSE "$stage/"
  find "$stage" -exec touch -h -d "@${epoch}" {} +

  if [ "$goos" = "windows" ]; then
    (cd "$out/stage" && find "$name" | LC_ALL=C sort | zip -q -X -@ "$out/$name.zip")
  else
    tar -C "$out/stage" --sort=name --mtime="@${epoch}" --owner=0 --group=0 --numeric-owner \
      -cf - "$name" | gzip -n -9 > "$out/$name.tar.gz"
  fi
done

rm -rf "$out/stage"
(cd "$out" && sha256sum ./*.tar.gz ./*.zip | sed 's| \./| |' > SHA256SUMS)
echo "built $(ls "$out" | grep -c -v SHA256SUMS) archives in dist/"
cat "$out/SHA256SUMS"
