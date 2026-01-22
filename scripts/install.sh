#!/usr/bin/env bash
set -euo pipefail

REPO="oodaris/autocodex"
VERSION="${VERSION:-}"
DEST="${DEST:-/usr/local/bin}"
PLUGIN_DEST="${PLUGIN_DEST:-}"

if [[ -z "${VERSION}" ]]; then
  VERSION="$(
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
      | grep -m1 '"tag_name"' \
      | sed -E 's/.*"v?([^"]+)".*/\1/'
  )"
fi

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "${OS}" in
  darwin|linux) ;;
  *) echo "Unsupported OS: ${OS}" >&2; exit 1 ;;
esac

case "${ARCH}" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported arch: ${ARCH}" >&2; exit 1 ;;
esac

TARBALL="autocodex_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${TARBALL}"
TMPDIR="$(mktemp -d)"

cleanup() {
  rm -rf "${TMPDIR}"
}
trap cleanup EXIT

echo "Downloading ${URL}"
curl -fsSL -o "${TMPDIR}/${TARBALL}" "${URL}"
tar -xzf "${TMPDIR}/${TARBALL}" -C "${TMPDIR}"

if [[ ! -w "${DEST}" ]]; then
  echo "Installing to ${DEST} (sudo required)"
  sudo install -d "${DEST}"
  sudo install -m 0755 "${TMPDIR}/autocodex" "${DEST}/autocodex"
else
  install -d "${DEST}"
  install -m 0755 "${TMPDIR}/autocodex" "${DEST}/autocodex"
fi

PREFIX="$(cd "${DEST}/.." && pwd)"
if [[ -z "${PLUGIN_DEST}" ]]; then
  PLUGIN_DEST="${PREFIX}/share/autocodex/plugins"
fi

if [[ -d "${TMPDIR}/plugins" ]]; then
  if [[ ! -w "${PLUGIN_DEST}" ]]; then
    echo "Installing plugins to ${PLUGIN_DEST} (sudo required)"
    sudo install -d "${PLUGIN_DEST}"
    sudo cp -R "${TMPDIR}/plugins/." "${PLUGIN_DEST}/"
  else
    install -d "${PLUGIN_DEST}"
    cp -R "${TMPDIR}/plugins/." "${PLUGIN_DEST}/"
  fi
  echo "Installed plugins to ${PLUGIN_DEST}"
else
  echo "No plugins directory found in release archive"
fi

echo "Installed autocodex to ${DEST}/autocodex"
autocodex --version || true
