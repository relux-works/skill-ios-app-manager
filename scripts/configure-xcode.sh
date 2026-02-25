#!/bin/bash
# Configure Xcode editor preferences for the project.
# Run once per machine — settings are global (per-user).
#
# What it does:
#   - Spaces instead of tabs (4-space indent)
#   - Indent switch cases (Swift style, not C style)
#   - Show invisible characters (spaces, newlines)
#   - Disable word wrap (lines go off-screen)

set -euo pipefail

DOMAIN="com.apple.dt.Xcode"

echo "Configuring Xcode editor preferences..."

# Spaces instead of tabs
defaults write "$DOMAIN" DVTTextIndentUsingTabs -bool false
defaults write "$DOMAIN" DVTTextIndentWidth -int 4
defaults write "$DOMAIN" DVTTextTabWidth -int 4

# Indent switch case bodies (Swift style)
defaults write "$DOMAIN" DVTTextIndentCase -bool true

# Show invisible characters
defaults write "$DOMAIN" DVTTextShowInvisibleCharacters -bool true

# Disable word wrap
defaults write "$DOMAIN" DVTTextEditorWrapsLines -bool false

echo "Done. Restart Xcode to apply."
echo ""
echo "Settings applied:"
echo "  Indent with spaces: 4"
echo "  Indent switch case: yes"
echo "  Show invisibles:    yes"
echo "  Word wrap:          off"
