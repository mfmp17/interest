#!/usr/bin/env bash
#
# Fred CLI installer
#   curl -fsSL https://get.fred.cash | bash
#
# Installs the `fred.cash` command and keeps `interest` as a legacy alias.

set -euo pipefail

REPO="mfmp17/interest"
BINARY="fred.cash"
LEGACY_BINARY="interest"

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
INSTALL_DIR=""
USE_SUDO=""

if [ -d "/opt/homebrew/bin" ] && [ -w "/opt/homebrew/bin" ]; then
  INSTALL_DIR="/opt/homebrew/bin"
elif [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
  INSTALL_DIR="/usr/local/bin"
elif command -v sudo >/dev/null 2>&1; then
  INSTALL_DIR="/usr/local/bin"
  USE_SUDO="sudo"
else
  INSTALL_DIR="$HOME/.local/bin"
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

install_legacy_alias() {
  if [ -n "$USE_SUDO" ]; then
    sudo ln -sf "${INSTALL_DIR}/${BINARY}" "${INSTALL_DIR}/${LEGACY_BINARY}" 2>/dev/null || true
  else
    ln -sf "${INSTALL_DIR}/${BINARY}" "${INSTALL_DIR}/${LEGACY_BINARY}" 2>/dev/null || true
  fi
}

if [ -n "$USE_SUDO" ]; then
  say "Installing to ${INSTALL_DIR} (may ask for your Mac password)..."
  if ! sudo mkdir -p "$INSTALL_DIR" || ! sudo install -m 0755 "$TMP" "${INSTALL_DIR}/${BINARY}"; then
    say "sudo install failed/cancelled; falling back to ~/.local/bin"
    INSTALL_DIR="$HOME/.local/bin"
    USE_SUDO=""
    mkdir -p "$INSTALL_DIR"
    mv "$TMP" "${INSTALL_DIR}/${BINARY}"
  else
    rm -f "$TMP"
  fi
else
  mkdir -p "$INSTALL_DIR"
  mv "$TMP" "${INSTALL_DIR}/${BINARY}"
fi

install_legacy_alias
ok "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"
ok "Legacy alias available as ${LEGACY_BINARY}"

# --- verify ------------------------------------------------------------------
if ! "${INSTALL_DIR}/${BINARY}" version >/dev/null 2>&1; then
  err "Installed file did not run correctly."
  exit 1
fi

# --- PATH check --------------------------------------------------------------
case ":$PATH:" in
  *":$INSTALL_DIR:"*) : ;;
  *)
    say ""
    say "${INSTALL_DIR} is not currently on your PATH. Add it with:"
    printf "  %becho 'export PATH=\"%s:\$PATH\"' >> ~/.zshrc && source ~/.zshrc%b\n" "$c_cyan" "$INSTALL_DIR" "$c_reset"
    say "Or run it directly once: ${INSTALL_DIR}/${BINARY}"
    ;;
esac

printf "\n"
ok "Done. Run: ${c_cyan}${c_bold}${BINARY} deposit${c_reset}"
