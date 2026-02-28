#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-dev}"
OUT_DIR="${2:-dist}"

mkdir -p "$(dirname "$OUT_DIR")"
OUT_DIR="$(cd "$(dirname "$OUT_DIR")" && pwd)/$(basename "$OUT_DIR")"
mkdir -p "$OUT_DIR"
rm -f "$OUT_DIR"/*

LDFLAGS="-s -w -X github.com/nayeemzen/hatch/internal/hatch.version=${VERSION}"

build_unix() {
  local os="$1"
  local arch="$2"
  local tmp
  tmp="$(mktemp -d)"
  GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "$LDFLAGS" -o "$tmp/hatch" .
  cp README.md LICENSE "$tmp/"
  tar -C "$tmp" -czf "$OUT_DIR/hatch_${VERSION}_${os}_${arch}.tar.gz" hatch README.md LICENSE
  rm -rf "$tmp"
}

build_windows() {
  local arch="$1"
  local tmp
  tmp="$(mktemp -d)"
  GOOS="windows" GOARCH="$arch" go build -trimpath -ldflags "$LDFLAGS" -o "$tmp/hatch.exe" .
  cp README.md LICENSE "$tmp/"
  (
    cd "$tmp"
    zip -q "$OUT_DIR/hatch_${VERSION}_windows_${arch}.zip" hatch.exe README.md LICENSE
  )
  rm -rf "$tmp"
}

build_unix linux amd64
build_unix linux arm64
build_unix darwin amd64
build_unix darwin arm64
build_windows amd64
build_windows arm64

echo "Built release artifacts in $OUT_DIR"
