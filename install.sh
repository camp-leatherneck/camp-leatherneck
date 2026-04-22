#!/usr/bin/env bash
#
# Camp Leatherneck installer — fetches the latest `lt` release from GitHub,
# verifies the sha256, and installs the binary to /usr/local/bin (or
# $HOME/.local/bin with --user).
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/camp-leatherneck/camp-leatherneck/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/camp-leatherneck/camp-leatherneck/main/install.sh | bash -s -- --user
#
# Options:
#   --user            Install to $HOME/.local/bin instead of /usr/local/bin (no sudo)
#   --version VER     Install a specific version (default: latest)
#   --help            Show this help
#
# Idempotent: re-running overwrites the existing binary.

set -euo pipefail

REPO="camp-leatherneck/camp-leatherneck"
BINARY="lt"
DEFAULT_SYSTEM_DIR="/usr/local/bin"
DEFAULT_USER_DIR="${HOME}/.local/bin"

# ----- output helpers --------------------------------------------------------

_bold() { printf '\033[1m%s\033[0m\n' "$*"; }
_info() { printf '\033[36m==>\033[0m %s\n' "$*"; }
_ok()   { printf '\033[32m OK\033[0m %s\n' "$*"; }
_warn() { printf '\033[33m !!\033[0m %s\n' "$*" >&2; }
_err()  { printf '\033[31mERR\033[0m %s\n' "$*" >&2; }
_die()  { _err "$*"; exit 1; }

# ----- arg parsing -----------------------------------------------------------

USER_MODE=0
VERSION=""

while [ $# -gt 0 ]; do
  case "$1" in
    --user)    USER_MODE=1; shift ;;
    --version) VERSION="${2:-}"; shift 2 ;;
    --help|-h)
      sed -n '3,17p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) _die "unknown argument: $1 (try --help)" ;;
  esac
done

# ----- preflight -------------------------------------------------------------

for dep in curl tar uname mktemp; do
  command -v "$dep" >/dev/null 2>&1 || _die "required tool not found: $dep"
done

SHA_TOOL=""
if command -v sha256sum >/dev/null 2>&1; then
  SHA_TOOL="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
  SHA_TOOL="shasum -a 256"
else
  _die "need sha256sum or shasum on PATH"
fi

# ----- OS / arch detection ---------------------------------------------------

OS_RAW="$(uname -s)"
ARCH_RAW="$(uname -m)"

case "$OS_RAW" in
  Darwin) OS="darwin" ;;
  Linux)  OS="linux" ;;
  *) _die "unsupported OS: $OS_RAW (supported: Darwin, Linux)" ;;
esac

case "$ARCH_RAW" in
  arm64|aarch64) ARCH="arm64" ;;
  x86_64|amd64)  ARCH="amd64" ;;
  *) _die "unsupported arch: $ARCH_RAW (supported: arm64, amd64/x86_64)" ;;
esac

_info "detected platform: ${OS}/${ARCH}"

# ----- resolve version -------------------------------------------------------

if [ -z "$VERSION" ]; then
  _info "resolving latest release from GitHub API"
  # Use GitHub's "latest" redirect — no auth required, works rate-limit-friendly.
  LATEST_URL="$(curl -fsSLI -o /dev/null -w '%{url_effective}' \
    "https://github.com/${REPO}/releases/latest")" \
    || _die "failed to query GitHub for latest release"
  VERSION="${LATEST_URL##*/}"
  if [ -z "$VERSION" ] || [ "$VERSION" = "latest" ] || [ "$VERSION" = "releases" ]; then
    _die "could not determine latest version (no releases published yet?)"
  fi
fi

# Normalize: strip leading 'v' for archive filename, keep tag form for URL.
TAG="$VERSION"
case "$TAG" in
  v*) VER_NUM="${TAG#v}" ;;
  *)  VER_NUM="$TAG"; TAG="v${TAG}" ;;
esac

_info "installing version ${TAG}"

# ----- target dir ------------------------------------------------------------

if [ "$USER_MODE" -eq 1 ]; then
  TARGET_DIR="$DEFAULT_USER_DIR"
  SUDO=""
else
  TARGET_DIR="$DEFAULT_SYSTEM_DIR"
  if [ -w "$TARGET_DIR" ] || [ ! -d "$TARGET_DIR" ] && [ -w "$(dirname "$TARGET_DIR")" ]; then
    SUDO=""
  else
    SUDO="sudo"
    _info "${TARGET_DIR} requires elevated permissions — using sudo"
  fi
fi

mkdir -p "$TARGET_DIR" 2>/dev/null || ${SUDO} mkdir -p "$TARGET_DIR"

# ----- download + verify -----------------------------------------------------

ARCHIVE="camp-leatherneck_${VER_NUM}_${OS}_${ARCH}.tar.gz"
DL_BASE="https://github.com/${REPO}/releases/download/${TAG}"
ARCHIVE_URL="${DL_BASE}/${ARCHIVE}"
CHECKSUMS_URL="${DL_BASE}/checksums.txt"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

_info "downloading ${ARCHIVE}"
curl -fsSL --retry 3 --retry-delay 2 -o "${TMP_DIR}/${ARCHIVE}" "$ARCHIVE_URL" \
  || _die "failed to download ${ARCHIVE_URL}"

_info "downloading checksums.txt"
curl -fsSL --retry 3 --retry-delay 2 -o "${TMP_DIR}/checksums.txt" "$CHECKSUMS_URL" \
  || _die "failed to download ${CHECKSUMS_URL}"

_info "verifying sha256"
EXPECTED="$(awk -v f="${ARCHIVE}" '$2 == f {print $1}' "${TMP_DIR}/checksums.txt")"
if [ -z "$EXPECTED" ]; then
  _die "archive ${ARCHIVE} not listed in checksums.txt — release may be incomplete"
fi
ACTUAL="$(${SHA_TOOL} "${TMP_DIR}/${ARCHIVE}" | awk '{print $1}')"
if [ "$ACTUAL" != "$EXPECTED" ]; then
  _die "sha256 mismatch — expected ${EXPECTED}, got ${ACTUAL}"
fi
_ok "sha256 verified"

# ----- extract + install -----------------------------------------------------

_info "extracting"
tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "${TMP_DIR}"

if [ ! -f "${TMP_DIR}/${BINARY}" ]; then
  _die "extracted archive does not contain ${BINARY} binary"
fi

chmod +x "${TMP_DIR}/${BINARY}"

TARGET_PATH="${TARGET_DIR}/${BINARY}"
_info "installing to ${TARGET_PATH}"
${SUDO} mv "${TMP_DIR}/${BINARY}" "${TARGET_PATH}"

_ok "installed ${BINARY} ${TAG} to ${TARGET_PATH}"

# ----- post-install hints ----------------------------------------------------

if ! command -v "${BINARY}" >/dev/null 2>&1; then
  _warn "${TARGET_DIR} is not on your PATH"
  _warn "add this to your shell rc:  export PATH=\"${TARGET_DIR}:\$PATH\""
fi

INSTALLED_VERSION="$("${TARGET_PATH}" --version 2>/dev/null || echo 'unknown')"
_ok "lt reports: ${INSTALLED_VERSION}"

echo
_bold "Next step: bootstrap your Camp Leatherneck workspace"
echo "  ${BINARY} install"
echo
echo "Docs: https://github.com/${REPO}"
