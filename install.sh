#!/bin/sh
set -eu

REPO="${REPO:-nayeemzen/hatch}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${VERSION:-}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command not found: $1" >&2
    exit 1
  fi
}

detect_os() {
  os="$(uname -s 2>/dev/null | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    linux|darwin) printf '%s' "$os" ;;
    *)
      echo "error: unsupported operating system: $os" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  arch="$(uname -m 2>/dev/null)"
  case "$arch" in
    x86_64|amd64) printf 'amd64' ;;
    aarch64|arm64) printf 'arm64' ;;
    *)
      echo "error: unsupported architecture: $arch" >&2
      exit 1
      ;;
  esac
}

resolve_version() {
  if [ -n "$VERSION" ]; then
    printf '%s' "$VERSION"
    return
  fi

  latest_api="https://api.github.com/repos/${REPO}/releases/latest"
  latest_version="$(curl -fsSL "$latest_api" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
  if [ -z "$latest_version" ]; then
    echo "error: could not resolve the latest version from ${latest_api}" >&2
    exit 1
  fi
  printf '%s' "$latest_version"
}

require_cmd curl
require_cmd tar
require_cmd uname
require_cmd sed
require_cmd head
require_cmd mkdir
require_cmd cp
require_cmd chmod

OS="$(detect_os)"
ARCH="$(detect_arch)"
VERSION="$(resolve_version)"
ARCHIVE="hatch_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/hatch-install.XXXXXX")"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

echo "Installing hatch ${VERSION} for ${OS}/${ARCH}"
echo "Download: ${URL}"

curl -fsSL "$URL" -o "${TMP_DIR}/${ARCHIVE}"
tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "$TMP_DIR"

if [ ! -f "${TMP_DIR}/hatch" ]; then
  echo "error: extracted archive does not contain hatch binary" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
cp "${TMP_DIR}/hatch" "${INSTALL_DIR}/hatch"
chmod 0755 "${INSTALL_DIR}/hatch"

echo "Installed to ${INSTALL_DIR}/hatch"
"${INSTALL_DIR}/hatch" --version || true

case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo
    echo "Add ${INSTALL_DIR} to PATH if needed:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac
