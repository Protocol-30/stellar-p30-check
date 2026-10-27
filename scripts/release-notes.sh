#!/usr/bin/env bash
# Prints the CHANGELOG.md section for one version, for the release body.
#
#   scripts/release-notes.sh 0.1.0
#
# Fails if CHANGELOG.md has no "## v0.1.0" section, so a tag cannot be
# released without its changelog entry.
set -euo pipefail
version="${1:?usage: scripts/release-notes.sh <version>}"
version="${version#v}"
root="$(cd "$(dirname "$0")/.." && pwd)"
changelog="${CHANGELOG:-$root/CHANGELOG.md}"
notes="$(awk -v h="## v${version}" '
  index($0, h) == 1 && (length($0) == length(h) || substr($0, length(h) + 1, 1) == " ") { on = 1; next }
  on && /^## / { exit }
  on { print }
' "$changelog")"
if [ -z "$(printf '%s' "$notes" | tr -d '[:space:]')" ]; then
  echo "CHANGELOG.md has no section \"## v${version}\"" >&2
  exit 1
fi
printf '%s\n' "$notes"
