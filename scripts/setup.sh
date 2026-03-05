#!/usr/bin/env zsh
#
# Setup script for ios-app-manager skill.
# Registers skill globally, builds CLI, symlinks binary to PATH.
#

set -euo pipefail

SKILL_NAME="ios-app-manager"
SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"
GO_DIR="$SKILL_DIR/tuist-starter"
BINARY_NAME="ios-app-manager"
BIN_DIR="$HOME/.local/bin"

AGENTS_SKILLS="$HOME/.agents/skills"
CLAUDE_SKILLS="$HOME/.claude/skills"
CODEX_SKILLS="$HOME/.codex/skills"

# --- Colors ---
red()   { print -P "%F{red}$1%f" }
green() { print -P "%F{green}$1%f" }
yellow(){ print -P "%F{yellow}$1%f" }

# --- 1. Check / install Go ---
check_go() {
  if command -v go &>/dev/null; then
    green "Go: $(go version)"
    return
  fi

  yellow "Go not found. Installing via Homebrew..."
  if ! command -v brew &>/dev/null; then
    red "Homebrew not found. Install it first: https://brew.sh"
    exit 1
  fi

  brew install go
  green "Go installed: $(go version)"
}

# --- 2. Check / install Tuist ---
check_tuist() {
  if command -v tuist &>/dev/null; then
    green "Tuist: $(tuist version)"
    return
  fi

  yellow "Tuist not found. Installing via Homebrew..."
  if ! command -v brew &>/dev/null; then
    red "Homebrew not found. Install it first: https://brew.sh"
    exit 1
  fi

  brew install tuist
  green "Tuist installed: $(tuist version)"
}

# --- 3. Build CLI ---
build_cli() {
  green "Building $BINARY_NAME..."
  cd "$GO_DIR"
  make build
  green "Built: $GO_DIR/$BINARY_NAME"
}

# --- 3. Register skill globally ---
register_skill() {
  mkdir -p "$AGENTS_SKILLS" "$CLAUDE_SKILLS" "$CODEX_SKILLS"

  # ~/.agents/skills/ios-app-manager -> repo root
  _symlink "$AGENTS_SKILLS/$SKILL_NAME" "$SKILL_DIR"

  # ~/.claude/skills/ and ~/.codex/skills/ -> ~/.agents/skills/
  _symlink "$CLAUDE_SKILLS/$SKILL_NAME" "$AGENTS_SKILLS/$SKILL_NAME"
  _symlink "$CODEX_SKILLS/$SKILL_NAME" "$AGENTS_SKILLS/$SKILL_NAME"
}

# --- 4. Symlink binary to PATH ---
install_binary() {
  local target="$GO_DIR/$BINARY_NAME"
  _symlink "$BIN_DIR/$BINARY_NAME" "$target"
}

# --- 5. Verify PATH ---
check_path() {
  if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    yellow "WARNING: $BIN_DIR is not in PATH."
    yellow "Add to ~/.zshrc:  export PATH=\"\$HOME/.local/bin:\$PATH\""
  fi
}

# --- 6. Verify ---
verify() {
  if command -v $BINARY_NAME &>/dev/null; then
    green "Verified: $($BINARY_NAME --version 2>&1 || echo "$BINARY_NAME available")"
  else
    yellow "$BINARY_NAME not found in PATH after setup. Check PATH settings."
  fi

  if [[ -f "$CLAUDE_SKILLS/$SKILL_NAME/SKILL.md" ]]; then
    green "Skill registered: $CLAUDE_SKILLS/$SKILL_NAME/SKILL.md"
  else
    red "Skill registration broken — SKILL.md not found via symlink chain"
  fi
}

# --- Helper: create/update symlink ---
_symlink() {
  local link="$1"
  local target="$2"

  mkdir -p "$(dirname "$link")"

  if [[ -L "$link" ]]; then
    local existing
    existing="$(readlink "$link")"
    if [[ "$existing" == "$target" ]]; then
      green "  OK: $link -> $target"
      return
    fi
    yellow "  Updating: $link (was: $existing)"
    rm "$link"
  elif [[ -e "$link" ]]; then
    red "  $link exists and is not a symlink. Skipping."
    return
  fi

  ln -s "$target" "$link"
  green "  Linked: $link -> $target"
}

# --- Run ---
print ""
green "=== $SKILL_NAME skill setup ==="
print ""
check_go
check_tuist
build_cli
print ""
green "Registering skill globally..."
register_skill
print ""
green "Installing binary..."
install_binary
check_path
print ""
verify
print ""
green "=== Done ==="
