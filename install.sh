#!/bin/sh
set -e

REPO="Kiloforge/kiloforge"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY="kf"

main() {
  detect_platform
  fetch_latest_version
  download_and_install
  echo ""
  echo "kf ${VERSION} installed to ${INSTALL_DIR}/${BINARY}"
  echo "Run 'kf init' to get started."
}

detect_platform() {
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  ARCH="$(uname -m)"

  case "${OS}" in
    darwin|linux) ;;
    mingw*|msys*|cygwin*)
      echo "Error: Windows is not supported by this installer." >&2
      echo "Download manually from https://github.com/${REPO}/releases" >&2
      exit 1
      ;;
    *)
      echo "Error: Unsupported operating system: ${OS}" >&2
      exit 1
      ;;
  esac

  case "${ARCH}" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)
      echo "Error: Unsupported architecture: ${ARCH}" >&2
      exit 1
      ;;
  esac

  echo "Detected platform: ${OS}/${ARCH}"
}

fetch_latest_version() {
  echo "Fetching latest release..."

  if ! command -v curl >/dev/null 2>&1; then
    echo "Error: curl is required but not installed." >&2
    exit 1
  fi

  # Use /releases endpoint (not /releases/latest) so pre-releases are included
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases" \
    | grep '"tag_name"' | head -1 \
    | sed -E 's/.*"tag_name":[[:space:]]*"([^"]+)".*/\1/')"

  if [ -z "${VERSION}" ]; then
    echo "Error: Failed to fetch latest version from GitHub." >&2
    echo "This may be due to API rate limiting. Try again later or set a GITHUB_TOKEN." >&2
    exit 1
  fi

  echo "Latest version: ${VERSION}"
}

download_and_install() {
  # Strip leading 'v' for archive naming (goreleaser uses version without v prefix)
  VERSION_NUM="${VERSION#v}"
  ARCHIVE="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
  URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

  TMPDIR="$(mktemp -d)"
  trap 'rm -rf "${TMPDIR}"' EXIT

  echo "Downloading ${URL}..."
  if ! curl -fsSL -o "${TMPDIR}/${ARCHIVE}" "${URL}"; then
    echo "Error: Failed to download ${URL}" >&2
    echo "Check that a release exists for your platform: ${OS}/${ARCH}" >&2
    exit 1
  fi

  echo "Extracting..."
  tar -xzf "${TMPDIR}/${ARCHIVE}" -C "${TMPDIR}" "${BINARY}"

  echo "Installing to ${INSTALL_DIR}..."
  mkdir -p "${INSTALL_DIR}"
  mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  chmod +x "${INSTALL_DIR}/${BINARY}"
}

main
