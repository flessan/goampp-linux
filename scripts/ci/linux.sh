#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
mode=${1:-all}
need() { command -v "$1" >/dev/null 2>&1 || { echo "Required command not found: $1" >&2; exit 1; }; }
need go
export CGO_ENABLED=1
format() {
  need git; need gofmt
  mapfile -d '' files < <(git ls-files -z -- '*.go')
  ((${#files[@]})) || { echo 'No tracked Go files found' >&2; exit 1; }
  local unformatted
  unformatted=$(gofmt -l "${files[@]}")
  if [[ -n "$unformatted" ]]; then printf 'Go files needing formatting:\n%s\n' "$unformatted" >&2; return 1; fi
}
case "$mode" in
  format) format ;;
  test) go vet ./...; go test ./... -count=1 ;;
  race) go test -race ./internal/linuxapp -count=1 ;;
  build) mkdir -p dist/linux; go build -trimpath -o dist/linux/goampp-linux . ;;
  package)
    need bash; need nfpm; need convert; need curl
    [[ -x dist/linux/goampp-linux ]] || { echo 'Run linux.sh build before package' >&2; exit 1; }
    bash packaging/linux/package.sh ;;
  all) format; go vet ./...; go test ./... -count=1; go test -race ./internal/linuxapp -count=1; mkdir -p dist/linux; go build -trimpath -o dist/linux/goampp-linux . ;;
  *) echo "Usage: $0 {format|test|race|build|package|all}" >&2; exit 2 ;;
esac
