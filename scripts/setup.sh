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
FULL_XCODE_DEVELOPER_DIR="/Applications/Xcode.app/Contents/Developer"
SELECT_XCODE=0

AGENTS_SKILLS="$HOME/.agents/skills"
CLAUDE_SKILLS="$HOME/.claude/skills"
CODEX_SKILLS="$HOME/.codex/skills"
INSTALLED_SKILL_DIR="$AGENTS_SKILLS/$SKILL_NAME"

RUNTIME_EXCLUDES=(
  --exclude=.git
  --exclude=.gitignore
  --exclude=.DS_Store
  --exclude=.cache
  --exclude=.github
  --exclude=.planning
  --exclude=.research
  --exclude=.task-board
  --exclude=.temp
  --exclude=task-board.config.json
)

# --- Colors ---
red()   { print -P "%F{red}$1%f" }
green() { print -P "%F{green}$1%f" }
yellow(){ print -P "%F{yellow}$1%f" }

scrub_git_metadata() {
  local target_dir="$1"
  rm -rf "$target_dir/.git"
  rm -f "$target_dir/.gitignore" "$target_dir/.gitattributes" "$target_dir/.gitmodules"
}

usage() {
  cat <<EOF
Usage: ./scripts/setup.sh [options]

Options:
  --select-xcode                 Select /Applications/Xcode.app as the active developer directory.
  --print-privileged-commands    Print the exact sudo commands used by --select-xcode and exit.
  -h, --help                     Show this help.

Default setup does not run sudo. Use --select-xcode when iOS simulator/build/profile
workflows need full Xcode instead of CommandLineTools.
EOF
}

parse_args() {
  while (($# > 0)); do
    case "$1" in
      --select-xcode)
        SELECT_XCODE=1
        ;;
      --print-privileged-commands)
        print_privileged_commands
        exit 0
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        red "Unknown argument: $1"
        usage
        exit 1
        ;;
    esac
    shift
  done
}

print_privileged_commands() {
  print "Privileged commands for --select-xcode:"
  print "  sudo -v"
  print "  sudo xcode-select -s $FULL_XCODE_DEVELOPER_DIR"
}

select_full_xcode() {
  if [[ ! -d "$FULL_XCODE_DEVELOPER_DIR" ]]; then
    red "Full Xcode developer directory not found: $FULL_XCODE_DEVELOPER_DIR"
    red "Install Xcode.app first or select it manually with xcode-select."
    exit 1
  fi

  local current_dir=""
  current_dir="$(xcode-select -p 2>/dev/null || true)"
  if [[ "$current_dir" == "$FULL_XCODE_DEVELOPER_DIR" ]]; then
    green "Full Xcode already selected: $FULL_XCODE_DEVELOPER_DIR"
    return
  fi

  yellow "Full Xcode selection requested."
  print_privileged_commands
  sudo -v
  sudo xcode-select -s "$FULL_XCODE_DEVELOPER_DIR"
  green "Selected full Xcode: $FULL_XCODE_DEVELOPER_DIR"
}

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

# --- 3. Check Xcode / profile diagnostics tools ---
check_xcode_tools() {
  if ! command -v xcode-select &>/dev/null; then
    yellow "WARNING: xcode-select not found. iOS build/profile diagnostics require Xcode tools."
    return
  fi

  local developer_dir
  if ! developer_dir="$(xcode-select -p 2>/dev/null)"; then
    yellow "WARNING: Xcode developer directory is not selected. Run: xcode-select --install"
    return
  fi
  green "Xcode: $developer_dir"

  if ! command -v xcodebuild &>/dev/null; then
    yellow "WARNING: xcodebuild not found. Build profiling requires Xcode."
  else
    local xcodebuild_version
    if xcodebuild_version="$(xcodebuild -version 2>/dev/null | head -n 1)"; then
      green "xcodebuild: $xcodebuild_version"
    else
      yellow "WARNING: xcodebuild is present but not usable with active developer directory."
      yellow "         Select full Xcode for iOS workflows: sudo xcode-select -s /Applications/Xcode.app/Contents/Developer"
    fi
  fi

  if ! command -v xcrun &>/dev/null; then
    yellow "WARNING: xcrun not found. Simulator/runtime diagnostics require Xcode tools."
  elif xcrun --find simctl &>/dev/null; then
    green "xcrun/simctl: $(xcrun --find simctl)"
  else
    yellow "WARNING: simctl not found via xcrun. Simulator runtime/layout diagnostics require full Xcode."
  fi

  if ! command -v log &>/dev/null; then
    yellow "WARNING: macOS unified log tool not found; runtime error diagnostics require /usr/bin/log."
  else
    green "log: $(command -v log)"
  fi

  if command -v xcrun &>/dev/null && xcrun --find xctrace &>/dev/null; then
    green "xctrace: $(xcrun --find xctrace)"
  else
    yellow "WARNING: xctrace not found. Optional Instruments trace workflows will be unavailable."
  fi
}

# --- 4. Build CLI ---
build_cli() {
  green "Building $BINARY_NAME..."
  cd "$GO_DIR"
  rm -f "$BINARY_NAME"
  make build
  green "Built: $GO_DIR/$BINARY_NAME"
}

# --- 5. Sync installed skill copy + symlinks ---
register_skill() {
  mkdir -p "$AGENTS_SKILLS" "$CLAUDE_SKILLS" "$CODEX_SKILLS"

  if [[ -L "$INSTALLED_SKILL_DIR" ]] || [[ -f "$INSTALLED_SKILL_DIR" ]]; then
    rm -rf "$INSTALLED_SKILL_DIR"
  fi
  mkdir -p "$INSTALLED_SKILL_DIR"
  rsync -a --delete "${RUNTIME_EXCLUDES[@]}" "$SKILL_DIR/" "$INSTALLED_SKILL_DIR/"
  scrub_git_metadata "$INSTALLED_SKILL_DIR"
  green "  Installed copy: $INSTALLED_SKILL_DIR"

  # ~/.claude/skills/ and ~/.codex/skills/ -> ~/.agents/skills/
  _symlink "$CLAUDE_SKILLS/$SKILL_NAME" "$INSTALLED_SKILL_DIR"
  _symlink "$CODEX_SKILLS/$SKILL_NAME" "$INSTALLED_SKILL_DIR"
}

# --- 6. Symlink binary to PATH ---
install_binary() {
  local target="$GO_DIR/$BINARY_NAME"
  _symlink "$BIN_DIR/$BINARY_NAME" "$target"
}

# --- 7. Verify PATH ---
check_path() {
  if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    yellow "WARNING: $BIN_DIR is not in PATH."
    yellow "Add to ~/.zshrc:  export PATH=\"\$HOME/.local/bin:\$PATH\""
  fi
}

# --- 8. Verify ---
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
parse_args "$@"

print ""
green "=== $SKILL_NAME skill setup ==="
print ""
if (( SELECT_XCODE )); then
  select_full_xcode
fi
check_go
check_tuist
check_xcode_tools
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
