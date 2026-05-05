#!/usr/bin/env bash
set -euo pipefail

BINARY="sshselector"
INSTALL_DIR="/usr/local/bin"
MODULE="github.com/romain/sshselector"

# ── Colours ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

info()    { echo -e "${CYAN}${BOLD}==>${RESET} $*"; }
success() { echo -e "${GREEN}${BOLD}  ✓${RESET} $*"; }
die()     { echo -e "${RED}${BOLD}error:${RESET} $*" >&2; exit 1; }

# ── Sanity checks ──────────────────────────────────────────────────────────────
[[ -f go.mod ]] || die "Run this script from the SSHSelector repository root."

grep -q "^module ${MODULE}" go.mod || \
  die "go.mod module name does not match '${MODULE}'. Are you in the right directory?"

command -v go &>/dev/null || die "Go is not installed or not on \$PATH."

# ── Build ──────────────────────────────────────────────────────────────────────
info "Building ${BINARY}…"

VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo "dev")"
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

go build \
  -ldflags "-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
  -o "${BINARY}" \
  .

success "Build complete  →  ./${BINARY}  (${VERSION})"

# ── Install ────────────────────────────────────────────────────────────────────
info "Installing to ${INSTALL_DIR}/${BINARY}…"

if [[ ! -w "${INSTALL_DIR}" ]]; then
  # Not writable — re-invoke with sudo.
  echo -e "  ${BOLD}${INSTALL_DIR} requires elevated privileges — running sudo mv${RESET}"
  sudo mv "${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  mv "${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

success "Installed  →  ${INSTALL_DIR}/${BINARY}"

# ── Verify ─────────────────────────────────────────────────────────────────────
if command -v "${BINARY}" &>/dev/null; then
  success "'${BINARY}' is on your \$PATH and ready to use."
else
  echo -e "\n  ${RED}Warning:${RESET} ${INSTALL_DIR} does not appear to be on your \$PATH."
  echo    "  Add the following to your shell profile and restart your terminal:"
  echo    ""
  echo    "      export PATH=\"\$PATH:${INSTALL_DIR}\""
fi
