#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
command -v go >/dev/null || { echo 'Go is required for Linux captures' >&2; exit 1; }
out=${1:-snapshots/linux}
mkdir -p "$out"
out=$(cd "$out" && pwd)
for page in editor vhosts; do for variant in Light Dark; do rm -f -- "$out/${variant}_${page}.png"; done; done
GOAMPP_SNAPSHOT="$out" go test ./internal/linuxapp -run '^TestLinuxPageSnapshots$' -count=1
for page in editor vhosts; do for variant in Light Dark; do
  [[ -s "$out/${variant}_${page}.png" ]] || { echo "Missing screenshot: ${variant}_${page}.png" >&2; exit 1; }
done; done
printf 'Linux screenshots written to %s\n' "$out"
