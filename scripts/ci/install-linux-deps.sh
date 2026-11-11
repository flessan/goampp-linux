#!/usr/bin/env bash
# For Debian/Ubuntu runners. Installs system dependencies; prompts for sudo locally.
set -euo pipefail
if [[ $# -gt 1 || ( $# -eq 1 && $1 != package ) ]]; then
  echo "Usage: $0 [package]" >&2; exit 2
fi
if [[ $(id -u) -eq 0 ]]; then sudo_cmd=(); else sudo_cmd=(sudo); fi
command -v apt-get >/dev/null || { echo 'apt-get is required; install the dependencies manually on other distributions' >&2; exit 1; }
"${sudo_cmd[@]}" apt-get update
"${sudo_cmd[@]}" apt-get install -y build-essential libgl1-mesa-dev xorg-dev libwayland-dev wayland-protocols libxkbcommon-dev libxkbcommon-x11-dev libdecor-0-dev php-cli nodejs npm python3
if [[ ${1:-} == package ]]; then
  "${sudo_cmd[@]}" apt-get install -y file desktop-file-utils patchelf imagemagick curl
  command -v go >/dev/null || { echo 'Go is required to install nfpm' >&2; exit 1; }
  go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.41.3
  echo "nfpm installed in $(go env GOPATH)/bin; add it to PATH"
fi
