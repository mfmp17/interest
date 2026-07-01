#!/usr/bin/env bash
#
# Fred "interest" installer
#   curl -fsSL https://get.fred.cash | bash
#
# Detects macOS arch, downloads the matching binary from GitHub Releases,
# installs to a directory on PATH, and prints next steps.

set -euo pipefail

REPO="mfmp17/interest"
BINARY="interest"

# --- pretty output -----------------------------------------------------------
c_green='\033[32m'; c_cyan='\033[36m'; c_red='\033[31m'; c_dim='\033[2m'; c_bold='\033[1m'; c_reset='\033[0m'
say()  { printf "%b%s%b\n" "$c_dim" "$1" "$c_reset"; }
ok()   { printf "%b● %b%b\n" "$c_green" "$1" "$c_reset"; }
err()  { printf "%b✗ %s%b\n" "$c_red" "$1" "$c_reset" >&2; }

# --- platform detection ------------------------------------------------------
OS="$(uname -s)"
if [ "$OS" != "Darwin" ]; then
  err "Fred currently supports macOS only (found: $OS)."
  exit 1
fi

ARCH="$(uname -m)"
case "$ARCH" in
  arm64)  ASSET_ARCH="arm64" ;;
  x86_64) ASSET_ARCH="amd64" ;;
  *) err "Unsupported architecture: $ARCH"; exit 1 ;;
esac

ASSET="${BINARY}_darwin_${ASSET_ARCH}"

# --- resolve latest release --------------------------------------------------
say "Finding latest release..."
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

# --- choose install dir ------------------------------------------------------
# Prefer /usr/local/bin if writable, else ~/.local/bin (added to PATH note).
if [ -w "/usr/local/bin" ] 2>/dev/null; then
  INSTALL_DIR="/usr/local/bin"
elif [ -w "/opt/homebrew/bin" ] 2>/dev/null; then
  INSTALL_DIR="/opt/homebrew/bin"
else
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
fi

TMP="$(mktemp)"
say "Downloading ${BINARY} (${ASSET_ARCH})..."
if ! curl -fsSL "$URL" -o "$TMP"; then
  err "Download failed from $URL"
  err "Has a release been published yet? See https://github.com/${REPO}/releases"
  rm -f "$TMP"
  exit 1
fi

chmod +x "$TMP"
mv "$TMP" "${INSTALL_DIR}/${BINARY}"
ok "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"

# --- PATH check --------------------------------------------------------------
case ":$PATH:" in
  *":$INSTALL_DIR:"*) : ;;
  *)
    say ""
    say "Add ${INSTALL_DIR} to your PATH:"
    printf "  %becho 'export PATH=\"%s:\$PATH\"' >> ~/.zshrc && source ~/.zshrc%b\n" "$c_cyan" "$INSTALL_DIR" "$c_reset"
    ;;
esac

printf "\n"
ok "Done. Run: ${c_cyan}${c_bold}${BINARY}${c_reset}"
